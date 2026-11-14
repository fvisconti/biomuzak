package tidal

// Tests for ParseTidalURL (E5).
//
// Objective: recognize Tidal track/album/playlist web URLs and reduce them to
// a typed resource ID the existing Client lookups can consume.
//
// Given/When/Then: Given a tidal.com (or play.tidal.h5) URL, When parsed,
// Then the link type and ID come back; When the URL is anything else, Then an
// error and no link.
//
// Edge cases: http vs https, www, trailing slashes, query strings, wrong
// hosts, unknown kinds, non-numeric track/album IDs, non-hex playlist IDs.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTidalURL_ValidLinks(t *testing.T) {
	cases := []struct {
		raw string
		typ LinkType
		id  string
	}{
		{"https://tidal.com/track/12345678", LinkTrack, "12345678"},
		{"http://tidal.com/track/1", LinkTrack, "1"},
		{"https://play.tidal.h5/track/12345678", LinkTrack, "12345678"},
		{"https://tidal.com/album/87654321", LinkAlbum, "87654321"},
		{"https://play.tidal.h5/album/42", LinkAlbum, "42"},
		{"https://tidal.com/playlist/pl1e309238b9d04b54a0b3f1c2d3e4f5a6", LinkPlaylist, "pl1e309238b9d04b54a0b3f1c2d3e4f5a6"},
		{"https://tidal.com/playlist/pldeadbeefcafebabe12345678", LinkPlaylist, "pldeadbeefcafebabe12345678"},
		{"https://play.tidal.h5/playlist/pl0123456789abcdef01234567", LinkPlaylist, "pl0123456789abcdef01234567"},
		{"https://www.tidal.com/track/12345678", LinkTrack, "12345678"},
		{"https://www.tidal.com/album/42", LinkAlbum, "42"},
		{" https://tidal.com/track/9 ", LinkTrack, "9"},
		{"https://tidal.com/track/12345678/", LinkTrack, "12345678"},
		{"https://tidal.com/album/42?ref=homepage", LinkAlbum, "42"},
	}
	for _, c := range cases {
		link, err := ParseTidalURL(c.raw)
		assert.NoError(t, err, "url %q", c.raw)
		if assert.NotNil(t, link, "url %q", c.raw) {
			assert.Equal(t, c.typ, link.Type, "url %q", c.raw)
			assert.Equal(t, c.id, link.ID, "url %q", c.raw)
		}
	}
}

func TestParseTidalURL_Rejects(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"not a url",
		"https://example.com/track/123",
		"https://tidal.com/",
		"https://tidal.com/artist/12345678",
		"https://tidal.com/track/",
		"https://tidal.com/track/abc",
		"https://tidal.com/album/12e345",
		"https://tidal.com/track/123456789012345678901", // over 12 digits: not a real ID
		"https://tidal.com/playlist/notahex",
		"https://tidal.com/playlist/plXYZnotahex", // non-hex after pl
		"https://tidal.com/playlist/pl123",        // too short
		"https://tidal.com/track/123/extra",       // too deep
	}
	for _, raw := range cases {
		link, err := ParseTidalURL(raw)
		assert.Error(t, err, "url %q", raw)
		assert.Nil(t, link, "url %q", raw)
	}
}

func TestParseTidalURL_TrackQueryFragmentTolerated(t *testing.T) {
	link, err := ParseTidalURL("https://tidal.com/track/12345678?x=1#frag")
	assert.NoError(t, err)
	if assert.NotNil(t, link) {
		assert.Equal(t, LinkTrack, link.Type)
		assert.Equal(t, "12345678", link.ID)
	}
}
