package metadata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/musicbrainz"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/models"

	"go-postgres-example/pkg/storage"

	"github.com/dhowden/tag"
)

// ProcessorAPI defines the interface for a metadata processor.
// ProcessorAPI defines the interface for a metadata processor.
type ProcessorAPI interface {
	ProcessFile(filePath string, userID int, playlistID int) error
}

// Processor handles the metadata processing logic.
type Processor struct {
	DB       *sql.DB
	Cfg      *config.Config
	MBClient musicbrainz.Clienter
	Storage  storage.StorageService
}

// NewProcessor creates a new Processor.
func NewProcessor(db *sql.DB, cfg *config.Config, storage storage.StorageService) *Processor {
	mbClient, err := musicbrainz.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create MusicBrainz client: %v", err)
	}
	return &Processor{DB: db, Cfg: cfg, MBClient: mbClient, Storage: storage}
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

func (p *Processor) ProcessFile(filePath string, userID int, playlistID int) error {
	// 1. Generate file hash
	hash, err := generateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to generate hash for %s: %w", filePath, err)
	}

	// 2. Check for duplicates
	// 2. Check for duplicates
	exists, existingID, err := p.songExists(hash)
	if err != nil {
		return fmt.Errorf("failed to check for song existence: %w", err)
	}

	var songID int
	if exists {
		log.Printf("Song with hash %s already exists. Using existing ID %d.", hash, existingID)
		songID = existingID
		// Even if it exists, we must link it to the user and playlist!
	} else {
		// New Song Logic

		// 3. Extract metadata from file
		song, err := extractMetadata(filePath)
		if err != nil {
			return fmt.Errorf("failed to extract metadata from %s: %w", filePath, err)
		}
		song.FingerprintHash = hash

		// 4. Move file to permanent storage location
		permanentPath, err := p.moveToUploadDir(filePath, hash)
		if err != nil {
			return fmt.Errorf("failed to move file to upload directory: %w", err)
		}
		song.FilePath = permanentPath

		// 5. Enrich metadata with MusicBrainz
		if err := p.MBClient.EnrichMetadata(song); err != nil {
			// Log the error but continue, as MusicBrainz is an enhancement, not a requirement.
			log.Printf("Failed to enrich metadata for %s: %v", filePath, err)
		}

		// 6. Get genre using the new logic
		genreName, err := p.getGenre(song, filePath)
		if err != nil {
			return fmt.Errorf("failed to get genre: %w", err)
		}

		// 7. Find or create genre
		if genreName != "" {
			genreID, err := p.findOrCreateGenre(genreName)
			if err != nil {
				return fmt.Errorf("failed to find or create genre: %w", err)
			}
			song.GenreID = sql.NullInt64{Int64: int64(genreID), Valid: true}
		}

		// 8. Save song to database
		songID, err = p.saveSong(song)
		if err != nil {
			return fmt.Errorf("failed to save song %s: %w", song.Title, err)
		}
		song.ID = songID

		// 9. Get and save song embedding (using the original temp file path)
		embedding, err := p.getEmbeddingsFromService(filePath)
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
		log.Printf("Successfully processed and saved NEW song: %s - %s (ID: %d)", song.Artist, song.Title, song.ID)
	}

	// ALWAYS perform these steps (for both new and existing songs)

	// 10. Link to User
	if err := db.AddUserSong(p.DB, userID, songID); err != nil {
		log.Printf("Failed to link song %d to user %d: %v", songID, userID, err)
	} else {
		log.Printf("Linked song %d to user %d", songID, userID)
	}

	// 11. Add to Playlist if requested
	if playlistID > 0 {
		// Default position is 0 (append)
		if err := db.AddSongToPlaylist(p.DB, playlistID, songID, 0); err != nil {
			log.Printf("Failed to add song %d to playlist %d: %v", songID, playlistID, err)
		} else {
			log.Printf("Added song %d to playlist %d", songID, playlistID)
		}
	}

	return nil
}

// getEmbeddingsFromService calls the audio processor microservice to generate embeddings.
func (p *Processor) getEmbeddingsFromService(filePath string) ([]float64, error) {
	// Open the audio file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create a multipart form body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create the file part
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file content to the form
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close the multipart writer
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the HTTP request
	url := p.Cfg.AudioProcessorURL + "/process-audio/"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	log.Printf("Calling audio processor at %s for file %s", url, filePath)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to audio processor: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("audio processor returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the JSON response
	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("Successfully received embedding of dimension %d from audio processor", len(result.Embedding))
	return result.Embedding, nil
}

// moveToUploadDir uploads a file from temporary location to MinIO.
// Files are organized by hash to prevent collisions.
func (p *Processor) moveToUploadDir(tempPath string, hash string) (string, error) {
	// Use the file extension from the original file
	ext := filepath.Ext(tempPath)

	// Create the permanent filename (object key) based on hash
	objectName := hash + ext

	// Open the source file
	sourceFile, err := os.Open(tempPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// get file info for size
	i, err := sourceFile.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	// Detect content type
	header := make([]byte, 512)
	if _, err := sourceFile.Read(header); err != nil {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}
	contentType := http.DetectContentType(header)

	// Seek back to start
	if _, err := sourceFile.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	// Upload to MinIO
	ctx := context.Background() // Use appropriate context if available
	err = p.Storage.UploadFile(ctx, objectName, sourceFile, i.Size(), contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to storage: %w", err)
	}

	log.Printf("Uploaded file %s to storage as %s", tempPath, objectName)
	return objectName, nil
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
		Bitrate:      0, // TODO: Extract from MP3 frame headers - library doesn't expose easily
		FileSize:     fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
	}

	return song, nil
}

func (p *Processor) songExists(hash string) (bool, int, error) {
	var songID int
	err := p.DB.QueryRow("SELECT id FROM songs WHERE fingerprint_hash = $1", hash).Scan(&songID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, 0, nil
		}
		return false, 0, err
	}
	return true, songID, nil
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
