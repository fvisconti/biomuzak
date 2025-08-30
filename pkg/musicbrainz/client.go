package musicbrainz

import (
	"fmt"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/models"
	"log"

	"github.com/michiwend/gomusicbrainz"
)

// Clienter defines the interface for a MusicBrainz client.
type Clienter interface {
	EnrichMetadata(song *models.Song) error
	GetArtistGenres(artistName string) ([]string, error)
}

// Client handles the communication with the MusicBrainz API.
type Client struct {
	client *gomusicbrainz.WS2Client
	cfg    *config.Config
}

// NewClient creates a new MusicBrainz client.
func NewClient(cfg *config.Config) (*Client, error) {
	client, err := gomusicbrainz.NewWS2Client("https://musicbrainz.org/ws/2", "go-music-app", "0.1", cfg.MusicBrainzEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to create musicbrainz client: %w", err)
	}
	return &Client{client: client, cfg: cfg}, nil
}

// EnrichMetadata enriches the song metadata with data from MusicBrainz.
func (c *Client) EnrichMetadata(song *models.Song) error {
	if song.Artist == "" || song.Title == "" {
		return nil // Not enough info for a lookup
	}

	// Search for releases, as they contain the album and date information.
	query := fmt.Sprintf("release:\"%s\" AND artist:\"%s\"", song.Title, song.Artist)
	resp, err := c.client.SearchRelease(query, 1, 0)
	if err != nil {
		return fmt.Errorf("musicbrainz search failed: %w", err)
	}

	if len(resp.Releases) > 0 {
		release := resp.Releases[0]

		// The release title is often the album title. The original song title is likely correct.
		song.Album = release.Title

		if len(release.ArtistCredit.NameCredits) > 0 {
			song.Artist = release.ArtistCredit.NameCredits[0].Artist.Name
		}

		if !release.Date.IsZero() {
			song.Year = release.Date.Year()
		}
	}

	return nil
}

// GetArtistGenres retrieves the genres for a given artist.
func (c *Client) GetArtistGenres(artistName string) ([]string, error) {
	if artistName == "" {
		return nil, nil // Not enough info for a lookup
	}

	// Search for the artist
	query := fmt.Sprintf("artist:\"%s\"", artistName)
	resp, err := c.client.SearchArtist(query, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz artist search failed: %w", err)
	}

	if len(resp.Artists) == 0 {
		return nil, nil // No artist found
	}

	// Get the first artist's tags (genres)
	artist := resp.Artists[0]
	var genres []string
	for _, tag := range artist.Tags {
		genres = append(genres, tag.Name)
	}

	log.Printf("Found genres for artist %s: %v", artistName, genres)

	return genres, nil
}
