package credits

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
	"ytpd/utils"
)

const discogsBaseURL = "https://api.discogs.com"

type discogsSearchResponse struct {
	Results []discogsSearchResult `json:"results"`
}

type discogsSearchResult struct {
	ID int `json:"id"`
}

type discogsRelease struct {
	ID           int               `json:"id"`
	Title        string            `json:"title"`
	Year         int               `json:"year"`
	Country      string            `json:"country"`
	Genres       []string          `json:"genres"`
	Styles       []string          `json:"styles"`
	Labels       []discogsLabel    `json:"labels"`
	Artists      []discogsCredit   `json:"artists"`
	ExtraArtists []discogsCredit   `json:"extraartists"`
	Tracklist    []discogsTrack    `json:"tracklist"`
	Identifiers  []discogsIdent    `json:"identifiers"`
	Notes        string            `json:"notes"`
}

type discogsLabel struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Catno string `json:"catno"`
}

type discogsTrack struct {
	Position     string          `json:"position"`
	Title        string          `json:"title"`
	Duration     string          `json:"duration"`
	ExtraArtists []discogsCredit `json:"extraartists"`
}

type discogsCredit struct {
	Name string `json:"name"`
	Anv  string `json:"anv"`
	Role string `json:"role"`
}

type discogsIdent struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

func discogsHeaders() map[string]string {
	token := os.Getenv("DISCOGS_TOKEN")
	headers := map[string]string{
		"User-Agent": "ytpd/1.0",
	}
	if token != "" {
		headers["Authorization"] = "Discogs token=" + token
	}
	return headers
}

func fetchFromDiscogs(artist, album string, tracks []TrackInfo) AlbumCredits {
	result := AlbumCredits{
		Tracks: make([]TrackMetadata, len(tracks)),
	}
	for i, t := range tracks {
		result.Tracks[i] = TrackMetadata{TrackNumber: t.Number, Title: t.Title}
	}

	releaseID, err := searchDiscogsRelease(artist, album)
	if err != nil || releaseID == 0 {
		log.Printf("Discogs: no release found for %s - %s", artist, album)
		return result
	}

	time.Sleep(time.Second)

	release, err := getDiscogsRelease(releaseID)
	if err != nil {
		log.Printf("Discogs: error fetching release %d: %v", releaseID, err)
		return result
	}

	// Album-level metadata
	if len(release.Genres) > 0 {
		result.Album.Genre = strings.Join(release.Genres, ", ")
	}
	result.Album.Styles = release.Styles
	if release.Year > 0 {
		result.Album.Year = fmt.Sprintf("%d", release.Year)
	}
	result.Album.Country = release.Country

	if len(release.Labels) > 0 {
		result.Album.Label = release.Labels[0].Name
		result.Album.CatalogNumber = release.Labels[0].Catno
	}

	// Barcode from identifiers
	for _, ident := range release.Identifiers {
		if ident.Type == "Barcode" && result.Album.Barcode == "" {
			result.Album.Barcode = ident.Value
		}
	}

	// Release-level credits (apply to all tracks if no track-level credits)
	releaseCredits := extractAllRoles(release.ExtraArtists)

	// Per-track credits
	for i, track := range tracks {
		trackCredits := findTrackCredits(release.Tracklist, track)
		tm := &result.Tracks[i]

		if trackCredits != nil {
			applyRoles(tm, trackCredits)
		} else {
			applyRoles(tm, &releaseCredits)
		}

		// ISRC from identifiers (track-level ISRCs in Discogs are rare but possible)
		for _, ident := range release.Identifiers {
			if ident.Type == "ISRC" && tm.ISRC == "" {
				// Discogs doesn't cleanly map ISRCs to tracks, skip unless description matches
				if strings.Contains(strings.ToLower(ident.Description), strings.ToLower(track.Title)) {
					tm.ISRC = ident.Value
				}
			}
		}

		if len(tm.Composers) > 0 {
			log.Printf("Discogs: %s → composers: %s", track.Title, strings.Join(tm.Composers, ", "))
		}
	}

	log.Printf("Discogs release: %s | label=%s year=%s country=%s genres=%s",
		release.Title, result.Album.Label, result.Album.Year, result.Album.Country, result.Album.Genre)

	return result
}

func searchDiscogsRelease(artist, album string) (int, error) {
	searchURL := fmt.Sprintf("%s/database/search?artist=%s&release_title=%s&type=release",
		discogsBaseURL,
		url.QueryEscape(artist),
		url.QueryEscape(album),
	)

	var resp discogsSearchResponse
	if err := utils.FetchJSON(searchURL, discogsHeaders(), &resp); err != nil {
		return 0, err
	}

	if len(resp.Results) == 0 {
		return 0, nil
	}

	return resp.Results[0].ID, nil
}

func getDiscogsRelease(id int) (*discogsRelease, error) {
	releaseURL := fmt.Sprintf("%s/releases/%d", discogsBaseURL, id)

	var release discogsRelease
	if err := utils.FetchJSON(releaseURL, discogsHeaders(), &release); err != nil {
		return nil, err
	}

	return &release, nil
}

// creditRoles holds extracted credits by role category.
type creditRoles struct {
	Composers []string
	Lyricists []string
	Arrangers []string
	Producers []string
	Engineers []string
	Mixers    []string
}

func extractAllRoles(artists []discogsCredit) creditRoles {
	var cr creditRoles
	for _, a := range artists {
		name := cleanDiscogsName(creditName(a))
		role := strings.ToLower(a.Role)

		switch {
		case isComposerRole(role):
			cr.Composers = append(cr.Composers, name)
		case containsAny(role, "lyrics by", "lyricist"):
			cr.Lyricists = append(cr.Lyricists, name)
		case containsAny(role, "arranged by", "arranger"):
			cr.Arrangers = append(cr.Arrangers, name)
		case containsAny(role, "producer", "produced by"):
			cr.Producers = append(cr.Producers, name)
		case containsAny(role, "engineer", "recorded by"):
			cr.Engineers = append(cr.Engineers, name)
		case containsAny(role, "mixed by", "mix"):
			cr.Mixers = append(cr.Mixers, name)
		}
	}
	return cr
}

func applyRoles(tm *TrackMetadata, cr *creditRoles) {
	if len(cr.Composers) > 0 && len(tm.Composers) == 0 {
		tm.Composers = cr.Composers
	}
	if len(cr.Lyricists) > 0 && len(tm.Lyricists) == 0 {
		tm.Lyricists = cr.Lyricists
	}
	if len(cr.Arrangers) > 0 && len(tm.Arrangers) == 0 {
		tm.Arrangers = cr.Arrangers
	}
	if len(cr.Producers) > 0 && len(tm.Producers) == 0 {
		tm.Producers = cr.Producers
	}
	if len(cr.Engineers) > 0 && len(tm.Engineers) == 0 {
		tm.Engineers = cr.Engineers
	}
	if len(cr.Mixers) > 0 && len(tm.Mixers) == 0 {
		tm.Mixers = cr.Mixers
	}
}

func findTrackCredits(tracklist []discogsTrack, track TrackInfo) *creditRoles {
	normalizedTitle := strings.ToLower(strings.TrimSpace(track.Title))

	for _, dt := range tracklist {
		if strings.ToLower(strings.TrimSpace(dt.Title)) == normalizedTitle {
			if len(dt.ExtraArtists) > 0 {
				cr := extractAllRoles(dt.ExtraArtists)
				return &cr
			}
			return nil
		}
	}
	return nil
}

func creditName(c discogsCredit) string {
	if c.Anv != "" {
		return c.Anv
	}
	return c.Name
}

func isComposerRole(role string) bool {
	return containsAny(role,
		"written-by", "written by",
		"composed by", "composer",
		"music by", "songwriter",
	)
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// cleanDiscogsName removes the trailing disambiguation number Discogs adds (e.g. "Artist (2)")
func cleanDiscogsName(name string) string {
	if idx := strings.LastIndex(name, " ("); idx != -1 {
		candidate := name[idx:]
		if strings.HasSuffix(candidate, ")") {
			trimmed := strings.TrimSuffix(strings.TrimPrefix(candidate, " ("), ")")
			allDigits := true
			for _, c := range trimmed {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				return name[:idx]
			}
		}
	}
	return name
}
