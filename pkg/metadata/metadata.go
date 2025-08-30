package metadata

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/musicbrainz"
	"io"
	"log"
	"os"
	"path/filepath"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/models"

	"github.com/dhowden/tag"
)

// ProcessorAPI defines the interface for a metadata processor.
type ProcessorAPI interface {
	ProcessFile(filePath string) error
}

// Processor handles the metadata processing logic.
type Processor struct {
	DB         *sql.DB
	Cfg        *config.Config
	MBClient   musicbrainz.Clienter
}

// NewProcessor creates a new Processor.
func NewProcessor(db *sql.DB, cfg *config.Config) *Processor {
	mbClient, err := musicbrainz.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create MusicBrainz client: %v", err)
	}
	return &Processor{DB: db, Cfg: cfg, MBClient: mbClient}
}

// ProcessFile orchestrates the entire process for a single file.
func (p *Processor) getGenre(song *models.Song, filePath string) (string, error) {
	// 1. Try to get genre from MusicBrainz
	if song.Artist != "" {
		genres, err := p.MBClient.GetArtistGenres(song.Artist)
		if err != nil {
			log.Printf("Failed to get genres from MusicBrainz for artist %s: %v", song.Artist, err)
		}
		if len(genres) > 0 {
			// For simplicity, we'll just take the first genre.
			// A more sophisticated approach might involve checking all genres against our internal list.
			return genres[0], nil
		}
	}

	// 2. Fallback to trigram search on the parent folder name
	parentDir := filepath.Base(filepath.Dir(filePath))
	if parentDir != "" && parentDir != "." {
		genre, err := db.FindGenreByTrigramSearch(p.DB, parentDir)
		if err != nil {
			return "", fmt.Errorf("failed to find genre by trigram search: %w", err)
		}
		if genre != "" {
			return genre, nil
		}
	}

	// 3. Fallback to the genre from the file's metadata
	return song.Genre, nil
}

func (p *Processor) ProcessFile(filePath string) error {
	// 1. Generate file hash
	hash, err := generateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to generate hash for %s: %w", filePath, err)
	}

	// 2. Check for duplicates
	exists, err := p.songExists(hash)
	if err != nil {
		return fmt.Errorf("failed to check for song existence: %w", err)
	}
	if exists {
		log.Printf("Song with hash %s already exists. Skipping.", hash)
		return nil
	}

	// 3. Extract metadata from file
	song, err := extractMetadata(filePath)
	if err != nil {
		return fmt.Errorf("failed to extract metadata from %s: %w", filePath, err)
	}
	song.FingerprintHash = hash
	song.FilePath = filePath // This might be a temporary path. The final path should be set after moving the file.

	// 4. Enrich metadata with MusicBrainz
	if err := p.MBClient.EnrichMetadata(song); err != nil {
		// Log the error but continue, as MusicBrainz is an enhancement, not a requirement.
		log.Printf("Failed to enrich metadata for %s: %v", filePath, err)
	}

	// 5. Get genre using the new logic
	genreName, err := p.getGenre(song, filePath)
	if err != nil {
		return fmt.Errorf("failed to get genre: %w", err)
	}

	// 6. Find or create genre
	if genreName != "" {
		genreID, err := p.findOrCreateGenre(genreName)
		if err != nil {
			return fmt.Errorf("failed to find or create genre: %w", err)
		}
		song.GenreID = sql.NullInt64{Int64: int64(genreID), Valid: true}
	}

	// 7. Save song to database
	songID, err := p.saveSong(song)
	if err != nil {
		return fmt.Errorf("failed to save song %s: %w", song.Title, err)
	}
	song.ID = songID

	// 7. Get and save song embedding
	embedding, err := getEmbeddingsFromService(filePath)
	if err != nil {
		// Log the error but don't fail the whole process, as embedding is an enhancement.
		log.Printf("Failed to get embedding for song ID %d: %v", songID, err)
	} else {
		if err := db.SaveSongEmbedding(p.DB, songID, embedding); err != nil {
			log.Printf("Failed to save embedding for song ID %d: %v", songID, err)
		} else {
			log.Printf("Successfully saved embedding for song ID %d", songID)
		}
	}

	log.Printf("Successfully processed and saved song: %s - %s (ID: %d)", song.Artist, song.Title, song.ID)
	return nil
}

// getEmbeddingsFromService is a placeholder for calling an external embedding service.
func getEmbeddingsFromService(filePath string) ([]float64, error) {
	// TODO: Implement the actual call to the Essentia microservice.
	// This will likely involve making an HTTP request to the service with the
	// audio file and receiving the embedding vector in response.
	log.Printf("TODO: Call external service to generate embedding for %s", filePath)

	// For now, return a dummy vector of the correct dimension (512).
	dummyEmbedding := make([]float64, 512)
	return dummyEmbedding, nil
}

func generateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func extractMetadata(filePath string) (*models.Song, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	meta, err := tag.ReadFrom(file)
	if err != nil {
		return nil, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	song := &models.Song{
		Title:        meta.Title(),
		Artist:       meta.Artist(),
		Album:        meta.Album(),
		Year:         meta.Year(),
		Genre:        meta.Genre(),
		Duration:     0, // dhowden/tag doesn't seem to provide duration directly
		Bitrate:      0, // dhowden/tag doesn't seem to provide bitrate directly
		FileSize:     fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
	}

	return song, nil
}

func (p *Processor) songExists(hash string) (bool, error) {
	var exists bool
	err := p.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM songs WHERE fingerprint_hash = $1)", hash).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}

func (p *Processor) findOrCreateGenre(name string) (int, error) {
	var genreID int
	err := p.DB.QueryRow("SELECT id FROM genres WHERE name = $1", name).Scan(&genreID)
	if err == sql.ErrNoRows {
		err = p.DB.QueryRow("INSERT INTO genres (name) VALUES ($1) RETURNING id", name).Scan(&genreID)
		if err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	return genreID, nil
}

func (p *Processor) saveSong(song *models.Song) (int, error) {
	query := `
		INSERT INTO songs (
			fingerprint_hash, file_path, title, artist, album, year, genre_id,
			duration, bitrate, file_size, last_modified
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`
	var songID int
	err := p.DB.QueryRow(
		query,
		song.FingerprintHash,
		song.FilePath,
		song.Title,
		song.Artist,
		song.Album,
		song.Year,
		song.GenreID,
		song.Duration,
		song.Bitrate,
		song.FileSize,
		song.LastModified,
	).Scan(&songID)

	if err != nil {
		return 0, err
	}
	return songID, nil
}
