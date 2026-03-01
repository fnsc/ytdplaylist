package playlist

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"ytpd/utils"
)

type Result struct {
	URL       string
	Directory string
	Data      PlaylistData
	Err       error
}

func ProcessAll(urls []string, artist string) []Result {
	return utils.Map(urls, func(url string) Result {
		return ProcessOne(url, artist)
	})
}

func ProcessOne(url, artist string) Result {
	data, err := ExtractData(url)
	if err != nil {
		return Result{URL: url, Err: err}
	}

	playlistDir := utils.FormatDirName("", data.Title)
	dir := filepath.Join(artist, playlistDir)
	data.Directory = dir

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return Result{URL: url, Err: err}
	}

	coverPath := filepath.Join(dir, "cover.jpg")
	if err := utils.SaveImage(data.ThumbURL, coverPath); err != nil {
		return Result{URL: url, Err: err}
	}

	cmd := exec.Command(
		"yt-dlp",
		"-f", "bestaudio",
		"--extract-audio",
		"--audio-format", "m4a",
		"--audio-quality", "0",
		"--embed-metadata",
		"--parse-metadata", "%(playlist)s:%(album)s",
		"--replace-in-metadata", "album", "^Album - ", "",
		"--parse-metadata", "%(playlist_index)s:%(track_number)s",
		"--parse-metadata", "%(release_date)s:%(meta_date)s",
		"-o", "%(playlist_index)s - %(title)s.%(ext)s",
		"--cookies", "~/.yt-dlp-config/yt-cookies.txt",
		"-P", dir,
		"--yes-playlist",
		url,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return Result{URL: url, Err: err}
	}

	if err := embedCoverArt(dir, coverPath); err != nil {
		return Result{URL: url, Err: err}
	}

	return Result{URL: url, Directory: dir, Data: data}
}

func embedCoverArt(dir, coverPath string) error {
	entries, err := filepath.Glob(filepath.Join(dir, "*.m4a"))
	if err != nil {
		return fmt.Errorf("error listing m4a files in %s: %w", dir, err)
	}

	for _, m4a := range entries {
		tmp := m4a + ".tmp.m4a"
		cmd := exec.Command(
			"ffmpeg", "-y",
			"-i", m4a,
			"-i", coverPath,
			"-map", "0:a",
			"-map", "1:v",
			"-c", "copy",
			"-disposition:v:0", "attached_pic",
			tmp,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("error embedding cover in %s: %w", m4a, err)
		}

		if err := os.Rename(tmp, m4a); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("error replacing %s: %w", m4a, err)
		}

		log.Printf("cover embedded: %s", m4a)
	}

	return nil
}
