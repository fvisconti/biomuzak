package subsonic

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

// DeriveSubsonicToken derives a user's stable Subsonic API token
// (HMAC-SHA256 under the JWT secret, bound to the username). Deriving it —
// rather than storing it in a column — avoids a schema migration; rotating
// JWT_SECRET revokes every token.
func DeriveSubsonicToken(secret, username string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("biomuzak-subsonic-token:" + username))
	return hex.EncodeToString(mac.Sum(nil))
}

// Artist/album ID encoding. The schema has no artists/albums tables, so IDs
// are synthesized as "prefix.base64url(payload)" — stable, reversible, and
// safe to echo back as a query parameter.
const (
	artistIDPrefix = "a."
	albumIDPrefix  = "al."
)

func encodeID(prefix, payload string) string {
	return prefix + base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeID(id, prefix string) (string, error) {
	if !strings.HasPrefix(id, prefix) {
		return "", errors.New("bad id prefix")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(id, prefix))
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func makeArtistID(artistName string) string {
	return encodeID(artistIDPrefix, artistName)
}

func makeAlbumID(artist, album string) string {
	return encodeID(albumIDPrefix, artist+"\x00"+album)
}

func parseArtistID(id string) (string, error) {
	return decodeID(id, artistIDPrefix)
}

func parseAlbumID(id string) (artist, album string, err error) {
	payload, err := decodeID(id, albumIDPrefix)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(payload, "\x00", 2)
	if len(parts) != 2 {
		return "", "", errors.New("bad album id payload")
	}
	return parts[0], parts[1], nil
}
