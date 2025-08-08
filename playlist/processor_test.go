package playlist

import (
	"testing"
	"ytpd/utils"
)

func TestFormatDirName(t *testing.T) {
	dir := utils.FormatDirName("prefix", "My Playlist:Test?")
	expected := "prefixMyPlaylistTest"

	if dir != expected {
		t.Errorf("expected '%s', got '%s'", expected, dir)
	}
}
