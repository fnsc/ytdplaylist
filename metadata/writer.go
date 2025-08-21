package metadata

import (
	"fmt"
	"io/fs"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"ytpd/playlist"
)

type Metadata struct {
	Title  string
	Artist string
	Album  string
	Track  string
	Cover  string
}

func WriteAllMetadata(result playlist.Result, artist string) {
	absDir, err := filepath.Abs(result.Directory)
	if err != nil {
		log.Printf("error getting absolute path: %v", err)
		return
	}

	filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".m4a") {
			return nil
		}

		base := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		parts := strings.SplitN(base, " - ", 2)
		track := ""
		title := base
		if len(parts) == 2 {
			track = parts[0]
			title = parts[1]
		}

		coverPath := filepath.Join(absDir, "cover.jpg")

		err = WriteMetadata(path, Metadata{
			Title:  title,
			Artist: artist,
			Album:  result.Data.Title,
			Track:  track,
			Cover:  coverPath,
		})
		if err != nil {
			log.Printf("error wrinting metadata %s: %v", path, err)
		} else {
			log.Printf("metadata written successfully %s", path)
		}

		return nil
	})
}

func WriteMetadata(filePath string, metadata Metadata) error {
	tempFile := filepath.Join(filepath.Dir(filePath), "temp_"+filepath.Base(filePath))

	args := []string{
		"-i", filePath,
		"-c", "copy",
		"-metadata", fmt.Sprintf("title=%s", metadata.Title),
		"-metadata", fmt.Sprintf("artist=%s", metadata.Artist),
		"-metadata", fmt.Sprintf("album=%s", metadata.Album),
		"-metadata", fmt.Sprintf("track=%s", metadata.Track),
		tempFile,
	}
	cmd := exec.Command("ffmpeg", args...)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		log.Printf("ffmpeg error: %s", stderr.String())
		return err
	}
	return exec.Command("mv", tempFile, filePath).Run()
}

func ExtractTitleAndTrack(filename string) (title, track string) {
	re := regexp.MustCompile(`^(\d+)\s+(.+)\.m4a$`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) == 3 {
		return matches[2], matches[1]
	}

	return strings.TrimSuffix(filename, ".m4a"), ""
}
