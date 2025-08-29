package metadata

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/models"

	"github.com/dhowden/tag"
	"github.com/michiwend/gomusicbrainz"
)

// ProcessorAPI defines the interface for a metadata processor.
type ProcessorAPI interface {
	ProcessFile(filePath string) error
}

// Processor handles the metadata processing logic.
type Processor struct {
	DB  *sql.DB
	Cfg *config.Config
}

// NewProcessor creates a new Processor.
func NewProcessor(db *sql.DB, cfg *config.Config) *Processor {
	return &Processor{DB: db, Cfg: cfg}
}

// ProcessFile orchestrates the entire process for a single file.
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
	if err := enrichMetadata(song, p.Cfg); err != nil {
		// Log the error but continue, as MusicBrainz is an enhancement, not a requirement.
		log.Printf("Failed to enrich metadata for %s: %v", filePath, err)
	}

	// 5. Find or create genre
	if song.Genre != "" {
		genreID, err := p.findOrCreateGenre(song.Genre)
		if err != nil {
			return fmt.Errorf("failed to find or create genre: %w", err)
		}
		song.GenreID = sql.NullInt64{Int64: int64(genreID), Valid: true}
	}

	// 6. Save song to database
	if err := p.saveSong(song); err != nil {
		return fmt.Errorf("failed to save song %s: %w", song.Title, err)
	}

	log.Printf("Successfully processed and saved song: %s - %s", song.Artist, song.Title)
	return nil
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

func enrichMetadata(song *models.Song, cfg *config.Config) error {
	if song.Artist == "" || song.Title == "" {
		return nil // Not enough info for a lookup
	}

	client, err := gomusicbrainz.NewWS2Client("https://musicbrainz.org/ws/2", "go-music-app", "0.1", cfg.MusicBrainzEmail)
	if err != nil {
		return fmt.Errorf("failed to create musicbrainz client: %w", err)
	}

	// Search for releases, as they contain the album and date information.
	query := fmt.Sprintf("release:\"%s\" AND artist:\"%s\"", song.Title, song.Artist)
	resp, err := client.SearchRelease(query, 1, 0)
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

func (p *Processor) saveSong(song *models.Song) error {
	query := `
		INSERT INTO songs (
			fingerprint_hash, file_path, title, artist, album, year, genre_id,
			duration, bitrate, file_size, last_modified
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := p.DB.Exec(
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
	)
	return err
}
