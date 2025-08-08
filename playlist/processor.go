package playlist

import (
	"os"
	"os/exec"
	"path/filepath"
	"ytpd/utils"
)

type Result struct {
	URL       string
	Directory string
	Err       error
}

func ProcessAll(urls []string, prefix string) []Result {
	return utils.Map(urls, func(url string) Result {
		return ProcessOne(url, prefix)
	})
}

func ProcessOne(url, prefix string) Result {
	data, err := ExtractData(url)
	if err != nil {
		return Result{URL: url, Err: err}
	}

	dir := utils.FormatDirName(prefix, data.Title)
	data.Directory = dir

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return Result{URL: url, Err: err}
	}

	if err := utils.SaveImage(data.ThumbURL, filepath.Join(dir, "cover.jpg")); err != nil {
		return Result{URL: url, Err: err}
	}

	cmd := exec.Command(
		"yt-dlp",
		"-f", "bestaudio",
		"--extract-audio",
		"--audio-format", "m4a",
		"--audio-quality", "0",
		"-o", "%(playlist_index)s - %(title)s.%(ext)s",
		"-P", dir,
		"--yes-playlist",
		url,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return Result{URL: url, Err: err}
	}

	return Result{URL: url, Directory: dir}
}
