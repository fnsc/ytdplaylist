package playlist

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

type PlaylistData struct {
	URL       string
	Title     string
	ThumbURL  string
	Directory string
}

func ExtractData(url string) (PlaylistData, error) {
	resp, err := http.Get(url)
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

	title := doc.Find("meta[name=title]").AttrOr("content", "")
	thumb := doc.Find("meta[property='og:image']").AttrOr("content", "")

	if title == "" || thumb == "" {
		return PlaylistData{}, errors.New("falha ao extrair título ou thumbnail")
	}

	return PlaylistData{
		URL:      url,
		Title:    title,
		ThumbURL: thumb,
	}, nil
}
