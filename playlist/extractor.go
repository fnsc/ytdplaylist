package playlist

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type PlaylistData struct {
	URL       string
	Title     string
	ThumbURL  string
	Directory string
}

func ExtractData(url string) (PlaylistData, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return PlaylistData{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return PlaylistData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return PlaylistData{}, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return PlaylistData{}, err
	}

	title := doc.Find("meta[property='og:title']").AttrOr("content", "")
	thumb := doc.Find("meta[property='og:image']").AttrOr("content", "")

	if title == "" || thumb == "" {
		return PlaylistData{}, errors.New("missing required metadata: title or thumb")
	}

	title = cleanTitle(title)

	return PlaylistData{
		URL:      url,
		Title:    title,
		ThumbURL: thumb,
	}, nil
}

// cleanTitle removes the " – Album by ..." suffix that YouTube Music appends.
func cleanTitle(title string) string {
	markers := []string{" – Album by ", " - Album by "}
	for _, marker := range markers {
		if idx := strings.Index(title, marker); idx != -1 {
			return strings.TrimSpace(title[:idx])
		}
	}
	return title
}
