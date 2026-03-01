package credits

import (
	"log"
	"strings"
)

type TrackInfo struct {
	Number int
	Title  string
}

// AlbumMetadata holds release-level metadata shared across all tracks.
type AlbumMetadata struct {
	Genre         string
	Styles        []string
	Year          string
	Label         string
	CatalogNumber string
	Barcode       string
	Country       string
}

// TrackMetadata holds per-track metadata including all credit roles.
type TrackMetadata struct {
	TrackNumber int
	Title       string
	Composers   []string
	Lyricists   []string
	Arrangers   []string
	Producers   []string
	Engineers   []string
	Mixers      []string
	ISRC        string
	MBRecordingID string
	MBReleaseID   string
	MBArtistID    string
}

// AlbumCredits is the full result returned by FetchAlbumCredits.
type AlbumCredits struct {
	Album  AlbumMetadata
	Tracks []TrackMetadata
}

func FetchAlbumCredits(artist, album string, tracks []TrackInfo) AlbumCredits {
	result := fetchFromMusicBrainz(artist, album, tracks)

	found := 0
	for _, t := range result.Tracks {
		if len(t.Composers) > 0 {
			found++
		}
	}

	if found == 0 {
		log.Printf("MusicBrainz: no composers found, trying Discogs...")
		result = fetchFromDiscogs(artist, album, tracks)
	} else {
		log.Printf("MusicBrainz: found composers for %d/%d tracks", found, len(tracks))
	}

	return result
}

// JoinField joins a string slice with ", " for embedding in metadata.
func JoinField(values []string) string {
	return strings.Join(values, ", ")
}
