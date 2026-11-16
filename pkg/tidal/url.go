package tidal

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// LinkType identifies the kind of Tidal resource a pasted URL points to.
type LinkType string

const (
	LinkTrack    LinkType = "track"
	LinkAlbum    LinkType = "album"
	LinkPlaylist LinkType = "playlist"
)

// TidalLink is a parsed Tidal web URL (E5). ID is numeric for tracks and
// albums; for playlists it is the URL's playlist ID string (e.g. "pl…"),
// which the Tidal API also accepts in path form.
type TidalLink struct {
	Type LinkType `json:"type"`
	ID   string   `json:"id"`
}

var (
	tidalHosts    = map[string]bool{"tidal.com": true, "www.tidal.com": true, "play.tidal.h5": true, "www.play.tidal.h5": true}
	numericID     = regexp.MustCompile(`^[0-9]{1,12}$`)
	playlistIDHex = regexp.MustCompile(`^pl[0-9a-fA-F]{16,32}$`)
)

// ParseTidalURL recognizes Tidal track/album/playlist web URLs and reduces
// them to a typed resource ID. Any other URL is rejected.
func ParseTidalURL(raw string) (*TidalLink, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("empty url")
	}

	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("not a valid url")
	}
	if !tidalHosts[strings.ToLower(u.Hostname())] {
		return nil, fmt.Errorf("not a tidal url")
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[1] == "" {
		return nil, fmt.Errorf("not a track, album, or playlist link")
	}

	switch parts[0] {
	case "track":
		if !numericID.MatchString(parts[1]) {
			return nil, fmt.Errorf("invalid track id %q", parts[1])
		}
		return &TidalLink{Type: LinkTrack, ID: parts[1]}, nil
	case "album":
		if !numericID.MatchString(parts[1]) {
			return nil, fmt.Errorf("invalid album id %q", parts[1])
		}
		return &TidalLink{Type: LinkAlbum, ID: parts[1]}, nil
	case "playlist":
		if !playlistIDHex.MatchString(parts[1]) {
			return nil, fmt.Errorf("invalid playlist id %q", parts[1])
		}
		return &TidalLink{Type: LinkPlaylist, ID: parts[1]}, nil
	default:
		return nil, fmt.Errorf("not a track, album, or playlist link")
	}
}
