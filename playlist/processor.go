package playlist

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"ytpd/credits"
	"ytpd/utils"
)

type Result struct {
	URL       string
	Directory string
	Data      PlaylistData
	Err       error
}

var trackFileRegex = regexp.MustCompile(`^(\d+) - (.+)\.m4a$`)

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
		"--cookies-from-browser", "firefox",
		"-P", dir,
		"--yes-playlist",
		url,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return Result{URL: url, Err: err}
	}

	tracks := parseTracksFromDir(dir)
	albumName := utils.Sanitize(data.Title)

	var albumCredits credits.AlbumCredits
	if len(tracks) > 0 {
		albumCredits = credits.FetchAlbumCredits(artist, albumName, tracks)
	}

	if err := embedMetadata(dir, coverPath, albumCredits); err != nil {
		return Result{URL: url, Err: err}
	}

	return Result{URL: url, Directory: dir, Data: data}
}

func parseTracksFromDir(dir string) []credits.TrackInfo {
	entries, err := filepath.Glob(filepath.Join(dir, "*.m4a"))
	if err != nil {
		return nil
	}

	var tracks []credits.TrackInfo
	for _, path := range entries {
		filename := filepath.Base(path)
		matches := trackFileRegex.FindStringSubmatch(filename)
		if matches == nil {
			continue
		}

		num, _ := strconv.Atoi(matches[1])
		tracks = append(tracks, credits.TrackInfo{
			Number: num,
			Title:  matches[2],
		})
	}

	return tracks
}

func embedMetadata(dir, coverPath string, ac credits.AlbumCredits) error {
	entries, err := filepath.Glob(filepath.Join(dir, "*.m4a"))
	if err != nil {
		return fmt.Errorf("error listing m4a files in %s: %w", dir, err)
	}

	// Build a lookup map by track number
	trackMeta := map[int]credits.TrackMetadata{}
	for _, tm := range ac.Tracks {
		trackMeta[tm.TrackNumber] = tm
	}

	for _, m4a := range entries {
		tmp := m4a + ".tmp.m4a"

		args := []string{
			"-y",
			"-i", m4a,
			"-i", coverPath,
			"-map", "0:a",
			"-map", "1:v",
			"-c", "copy",
			"-disposition:v:0", "attached_pic",
		}

		// Album-level metadata
		addMetadata(&args, "genre", ac.Album.Genre)
		addMetadata(&args, "date", ac.Album.Year)
		addFreeform(&args, "LABEL", ac.Album.Label)
		addFreeform(&args, "CATALOGNUMBER", ac.Album.CatalogNumber)
		addFreeform(&args, "BARCODE", ac.Album.Barcode)
		addFreeform(&args, "RELEASECOUNTRY", ac.Album.Country)
		if len(ac.Album.Styles) > 0 {
			addFreeform(&args, "STYLE", strings.Join(ac.Album.Styles, "; "))
		}

		// Track-level metadata
		filename := filepath.Base(m4a)
		matches := trackFileRegex.FindStringSubmatch(filename)
		if matches != nil {
			trackNum, _ := strconv.Atoi(matches[1])
			if tm, ok := trackMeta[trackNum]; ok {
				addMetadata(&args, "composer", credits.JoinField(tm.Composers))
				addFreeform(&args, "LYRICIST", credits.JoinField(tm.Lyricists))
				addFreeform(&args, "ARRANGER", credits.JoinField(tm.Arrangers))
				addFreeform(&args, "PRODUCER", credits.JoinField(tm.Producers))
				addFreeform(&args, "ENGINEER", credits.JoinField(tm.Engineers))
				addFreeform(&args, "MIXER", credits.JoinField(tm.Mixers))
				addFreeform(&args, "ISRC", tm.ISRC)
				addFreeform(&args, "MUSICBRAINZ_TRACKID", tm.MBRecordingID)
				addFreeform(&args, "MUSICBRAINZ_ALBUMID", tm.MBReleaseID)
				addFreeform(&args, "MUSICBRAINZ_ARTISTID", tm.MBArtistID)
			}
		}

		args = append(args, tmp)

		cmd := exec.Command("ffmpeg", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("error embedding metadata in %s: %w", m4a, err)
		}

		if err := os.Rename(tmp, m4a); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("error replacing %s: %w", m4a, err)
		}

		log.Printf("metadata embedded: %s", m4a)
	}

	return nil
}

// addMetadata appends a standard `-metadata key=value` if value is non-empty.
func addMetadata(args *[]string, key, value string) {
	if value != "" {
		*args = append(*args, "-metadata", fmt.Sprintf("%s=%s", key, value))
	}
}

// addFreeform appends a freeform iTunes tag `-metadata ----:com.apple.iTunes:KEY=value` if value is non-empty.
func addFreeform(args *[]string, key, value string) {
	if value != "" {
		*args = append(*args, "-metadata", fmt.Sprintf("----:com.apple.iTunes:%s=%s", key, value))
	}
}
