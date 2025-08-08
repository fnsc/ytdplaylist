package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func SaveImage(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func Sanitize(s string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	return replacer.Replace(strings.TrimSpace(s))
}

func toCamelCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	for i := range words {
		words[i] = strings.ToLower(words[i])
		if i > 0 && len(words[i]) > 0 {
			words[i] = strings.ToUpper(string(words[i][0])) + words[i][1:]
		}
	}
	return strings.Join(words, "")
}

func FormatDirName(prefix, title string) string {
	sanitized := Sanitize(title)
	combined := fmt.Sprintf("%s_%s", prefix, sanitized)
	return toCamelCase(combined)
}
