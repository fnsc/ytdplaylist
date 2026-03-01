package credits

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
	"ytpd/utils"
)

const (
	mbBaseURL   = "https://musicbrainz.org/ws/2"
	mbUserAgent = "ytpd/1.0 (https://github.com/fnsc/ytpd)"
)

var mbHeaders = map[string]string{
	"User-Agent": mbUserAgent,
	"Accept":     "application/json",
}

// --- Search types ---

type mbSearchResponse struct {
	Recordings []mbSearchRecording `json:"recordings"`
}

type mbSearchRecording struct {
	ID       string      `json:"id"`
	Title    string      `json:"title"`
	Releases []mbRelease `json:"releases"`
}

// --- Recording lookup types ---

type mbRecordingLookup struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	ISRCs     []string     `json:"isrcs"`
	Genres    []mbGenre    `json:"genres"`
	Tags      []mbTag      `json:"tags"`
	Relations []mbRelation `json:"relations"`
	Releases  []mbRelease  `json:"releases"`
}

type mbGenre struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type mbTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type mbRelation struct {
	Type       string          `json:"type"`
	TargetType string          `json:"target-type"`
	Attributes []string        `json:"attributes"`
	Artist     *mbArtist       `json:"artist,omitempty"`
	Work       *mbWork         `json:"work,omitempty"`
}

type mbArtist struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	SortName string `json:"sort-name"`
}

type mbWork struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	Relations []mbRelation `json:"relations"`
}

// --- Release types ---

type mbRelease struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Date          string          `json:"date"`
	Country       string          `json:"country"`
	Status        string          `json:"status"`
	Barcode       string          `json:"barcode"`
	ArtistCredit  []mbArtistCred  `json:"artist-credit"`
	LabelInfo     []mbLabelInfo   `json:"label-info"`
	Genres        []mbGenre       `json:"genres"`
	Media         []mbMedia       `json:"media"`
}

type mbArtistCred struct {
	Artist mbArtist `json:"artist"`
}

type mbLabelInfo struct {
	CatalogNumber string   `json:"catalog-number"`
	Label         *mbLabel `json:"label"`
}

type mbLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type mbMedia struct {
	Position   int       `json:"position"`
	Format     string    `json:"format"`
	TrackCount int       `json:"track-count"`
	Tracks     []mbTrack `json:"tracks"`
}

type mbTrack struct {
	Number   string `json:"number"`
	Title    string `json:"title"`
	Position int    `json:"position"`
}

// --- Fetching ---

func fetchFromMusicBrainz(artist, album string, tracks []TrackInfo) AlbumCredits {
	result := AlbumCredits{
		Tracks: make([]TrackMetadata, len(tracks)),
	}
	for i, t := range tracks {
		result.Tracks[i] = TrackMetadata{TrackNumber: t.Number, Title: t.Title}
	}

	var releaseID string

	for i, track := range tracks {
		recording, rid := searchAndLookupRecording(artist, album, track.Title)
		if recording == nil {
			time.Sleep(time.Second)
			continue
		}

		if releaseID == "" && rid != "" {
			releaseID = rid
		}

		tm := &result.Tracks[i]
		tm.MBRecordingID = recording.ID

		// ISRC
		if len(recording.ISRCs) > 0 {
			tm.ISRC = recording.ISRCs[0]
		}

		// Genre from recording
		if result.Album.Genre == "" && len(recording.Genres) > 0 {
			result.Album.Genre = bestGenre(recording.Genres)
		}

		// Tags as styles
		if len(result.Album.Styles) == 0 && len(recording.Tags) > 0 {
			result.Album.Styles = topTags(recording.Tags, 3)
		}

		// Extract credits from relations
		extractRecordingCredits(recording, tm)

		if len(tm.Composers) > 0 {
			log.Printf("MusicBrainz: %s → composers: %s", track.Title, strings.Join(tm.Composers, ", "))
		}

		time.Sleep(time.Second)
	}

	// Fetch release-level metadata (label, barcode, catalog#, date, country)
	if releaseID != "" {
		fetchReleaseMetadata(releaseID, &result)
	}

	return result
}

func searchAndLookupRecording(artist, album, title string) (*mbRecordingLookup, string) {
	query := fmt.Sprintf(`recording:"%s" AND artist:"%s" AND release:"%s"`,
		title, artist, album,
	)
	searchURL := fmt.Sprintf("%s/recording?query=%s&fmt=json&limit=3",
		mbBaseURL, url.QueryEscape(query),
	)

	var resp mbSearchResponse
	if err := utils.FetchJSON(searchURL, mbHeaders, &resp); err != nil {
		log.Printf("MusicBrainz search error for %q: %v", title, err)
		return nil, ""
	}

	if len(resp.Recordings) == 0 {
		return nil, ""
	}

	// Get release ID from search results
	var releaseID string
	if len(resp.Recordings[0].Releases) > 0 {
		releaseID = resp.Recordings[0].Releases[0].ID
	}

	time.Sleep(time.Second)

	// Lookup with all available incs
	lookupURL := fmt.Sprintf(
		"%s/recording/%s?inc=work-rels+work-level-rels+artist-rels+isrcs+genres+tags+releases&fmt=json",
		mbBaseURL, resp.Recordings[0].ID,
	)

	var recording mbRecordingLookup
	if err := utils.FetchJSON(lookupURL, mbHeaders, &recording); err != nil {
		log.Printf("MusicBrainz lookup error for %s: %v", resp.Recordings[0].ID, err)
		return nil, releaseID
	}

	// Try to get release ID from lookup if we don't have one
	if releaseID == "" && len(recording.Releases) > 0 {
		releaseID = recording.Releases[0].ID
	}

	return &recording, releaseID
}

func extractRecordingCredits(rec *mbRecordingLookup, tm *TrackMetadata) {
	composers := map[string]bool{}
	lyricists := map[string]bool{}
	arrangers := map[string]bool{}
	producers := map[string]bool{}
	engineers := map[string]bool{}
	mixers := map[string]bool{}

	for _, rel := range rec.Relations {
		// Direct artist relations on recording (producer, engineer, mix, etc.)
		if rel.TargetType == "artist" && rel.Artist != nil {
			switch rel.Type {
			case "producer":
				producers[rel.Artist.Name] = true
			case "engineer":
				engineers[rel.Artist.Name] = true
			case "mix":
				mixers[rel.Artist.Name] = true
			case "arranger":
				arrangers[rel.Artist.Name] = true
			}

			// Set MB artist ID from first artist relation
			if tm.MBArtistID == "" {
				tm.MBArtistID = rel.Artist.ID
			}
		}

		// Work relations (composer, lyricist, writer)
		if rel.TargetType == "work" && rel.Work != nil {
			for _, wrel := range rel.Work.Relations {
				if wrel.Artist == nil {
					continue
				}
				switch wrel.Type {
				case "composer", "writer":
					composers[wrel.Artist.Name] = true
				case "lyricist":
					lyricists[wrel.Artist.Name] = true
				case "arranger":
					arrangers[wrel.Artist.Name] = true
				}
			}
		}
	}

	tm.Composers = mapKeys(composers)
	tm.Lyricists = mapKeys(lyricists)
	tm.Arrangers = mapKeys(arrangers)
	tm.Producers = mapKeys(producers)
	tm.Engineers = mapKeys(engineers)
	tm.Mixers = mapKeys(mixers)
}

func fetchReleaseMetadata(releaseID string, result *AlbumCredits) {
	time.Sleep(time.Second)

	releaseURL := fmt.Sprintf("%s/release/%s?inc=labels+genres&fmt=json", mbBaseURL, releaseID)

	var release mbRelease
	if err := utils.FetchJSON(releaseURL, mbHeaders, &release); err != nil {
		log.Printf("MusicBrainz release lookup error for %s: %v", releaseID, err)
		return
	}

	if result.Album.Year == "" && release.Date != "" {
		result.Album.Year = release.Date
	}
	if result.Album.Country == "" && release.Country != "" {
		result.Album.Country = release.Country
	}
	if release.Barcode != "" {
		result.Album.Barcode = release.Barcode
	}

	if len(release.LabelInfo) > 0 {
		li := release.LabelInfo[0]
		if li.Label != nil && result.Album.Label == "" {
			result.Album.Label = li.Label.Name
		}
		if li.CatalogNumber != "" && result.Album.CatalogNumber == "" {
			result.Album.CatalogNumber = li.CatalogNumber
		}
	}

	if result.Album.Genre == "" && len(release.Genres) > 0 {
		result.Album.Genre = bestGenre(release.Genres)
	}

	// Set MB release ID on all tracks
	for i := range result.Tracks {
		result.Tracks[i].MBReleaseID = releaseID
	}

	log.Printf("MusicBrainz release: %s | label=%s date=%s country=%s",
		release.Title, result.Album.Label, result.Album.Year, result.Album.Country)
}

// --- Helpers ---

func bestGenre(genres []mbGenre) string {
	best := ""
	bestCount := 0
	for _, g := range genres {
		if g.Count > bestCount {
			best = g.Name
			bestCount = g.Count
		}
	}
	return best
}

func topTags(tags []mbTag, max int) []string {
	// Simple: take tags sorted by count descending, up to max
	type tagScore struct {
		name  string
		count int
	}
	scored := make([]tagScore, len(tags))
	for i, t := range tags {
		scored[i] = tagScore{t.Name, t.Count}
	}
	// Bubble sort by count desc (small N)
	for i := range scored {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].count > scored[i].count {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	result := make([]string, 0, max)
	for i, s := range scored {
		if i >= max {
			break
		}
		result = append(result, s.name)
	}
	return result
}

func mapKeys(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
