package subsonic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveSubsonicToken_StableAndBoundToUser(t *testing.T) {
	a1 := DeriveSubsonicToken("secret", "alice")
	a2 := DeriveSubsonicToken("secret", "alice")
	b := DeriveSubsonicToken("secret", "bob")
	other := DeriveSubsonicToken("other-secret", "alice")

	assert.Equal(t, a1, a2, "token must be stable per user")
	assert.NotEqual(t, a1, b, "tokens must differ per user")
	assert.NotEqual(t, a1, other, "tokens must depend on the secret")
}

func TestArtistIDRoundTrip(t *testing.T) {
	id := makeArtistID("Daft Punk")
	name, err := parseArtistID(id)
	require.NoError(t, err)
	assert.Equal(t, "Daft Punk", name)
}

func TestAlbumIDRoundTrip(t *testing.T) {
	id := makeAlbumID("Daft Punk", "Random Access Memories")
	artist, album, err := parseAlbumID(id)
	require.NoError(t, err)
	assert.Equal(t, "Daft Punk", artist)
	assert.Equal(t, "Random Access Memories", album)
}

func TestParseIDs_RejectWrongPrefix(t *testing.T) {
	_, err := parseArtistID(makeAlbumID("A", "B"))
	assert.Error(t, err)
	_, _, err = parseAlbumID(makeArtistID("A"))
	assert.Error(t, err)
	_, err = parseArtistID("garbage")
	assert.Error(t, err)
}
