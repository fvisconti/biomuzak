package metadata

import (
	"database/sql"
	"errors"
	"go-postgres-example/pkg/models"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// MockMusicBrainzClient is a mock implementation of the musicbrainz.Clienter interface.
type MockMusicBrainzClient struct {
	GetArtistGenresFunc func(artistName string) ([]string, error)
}

func (m *MockMusicBrainzClient) EnrichMetadata(song *models.Song) error {
	// For this test, we don't need to implement this method.
	return nil
}

func (m *MockMusicBrainzClient) GetArtistGenres(artistName string) ([]string, error) {
	if m.GetArtistGenresFunc != nil {
		return m.GetArtistGenresFunc(artistName)
	}
	return nil, errors.New("GetArtistGenresFunc not implemented")
}

func TestGetGenre(t *testing.T) {
	// Test case 1: MusicBrainz returns a genre
	t.Run("MusicBrainz success", func(t *testing.T) {
		mockMBClient := &MockMusicBrainzClient{
			GetArtistGenresFunc: func(artistName string) ([]string, error) {
				return []string{"Progressive Rock"}, nil
			},
		}
		processor := &Processor{MBClient: mockMBClient}
		song := &models.Song{Artist: "Opeth"}
		genre, err := processor.getGenre(song, "/path/to/file")
		assert.NoError(t, err)
		assert.Equal(t, "Progressive Rock", genre)
	})

	// Test case 2: MusicBrainz fails, but trigram search succeeds
	t.Run("Trigram search success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"name"}).AddRow("Electronic")
		mock.ExpectQuery("SELECT name FROM genres").WithArgs("electronica").WillReturnRows(rows)

		mockMBClient := &MockMusicBrainzClient{
			GetArtistGenresFunc: func(artistName string) ([]string, error) {
				return nil, errors.New("MusicBrainz error")
			},
		}

		processor := &Processor{DB: db, MBClient: mockMBClient}
		song := &models.Song{Artist: "Aphex Twin"}
		genre, err := processor.getGenre(song, "/music/electronica/aphex_twin.mp3")
		assert.NoError(t, err)
		assert.Equal(t, "Electronic", genre)
	})

	// Test case 3: MusicBrainz and trigram search fail, fallback to file metadata
	t.Run("Fallback to file metadata", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery("SELECT name FROM genres").WithArgs("rock").WillReturnError(sql.ErrNoRows)

		mockMBClient := &MockMusicBrainzClient{
			GetArtistGenresFunc: func(artistName string) ([]string, error) {
				return nil, nil // No error, but no genres found
			},
		}

		processor := &Processor{DB: db, MBClient: mockMBClient}
		song := &models.Song{Artist: "Nirvana", Genre: "Alternative"}
		genre, err := processor.getGenre(song, "/music/rock/nirvana.mp3")
		assert.NoError(t, err)
		assert.Equal(t, "Alternative", genre)
	})
}
