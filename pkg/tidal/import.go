package tidal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/storage"
)

// Importer performs Tidal track imports into the biomuzak library.
type Importer struct {
	Client  *Client
	DB      *sql.DB
	Storage storage.StorageService
	HTTP    *http.Client
}

// NewImporter builds an Importer.
func NewImporter(client *Client, db *sql.DB, s storage.StorageService) *Importer {
	return &Importer{
		Client:  client,
		DB:      db,
		Storage: s,
		HTTP:    &http.Client{Timeout: 5 * time.Minute}, // large FLAC downloads
	}
}

// TrackMeta is the (possibly user-edited) metadata for a track to import.
type TrackMeta struct {
	TrackID int    `json:"track_id"`
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Album   string `json:"album"`
	Year    int    `json:"year"`
}

// ImportResult reports the outcome of a single track import.
type ImportResult struct {
	SongID int    `json:"song_id"`
	Dedup  bool   `json:"dedup"` // true if the file already existed
	Error  string `json:"error,omitempty"`
}

// ImportTrack fetches, decrypts, stores, and registers a single Tidal track.
func (im *Importer) ImportTrack(ctx context.Context, userID int, meta TrackMeta, cfg TidalConfig) (*ImportResult, error) {
	qualityID := cfg.qualityToAudioQualityID()

	// 1. Get the signed stream URL.
	stream, err := im.Client.GetStreamURL(ctx, userID, meta.TrackID, qualityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream url: %w", err)
	}

	// 2. Download the encrypted stream.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, stream.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build stream request: %w", err)
	}
	resp, err := im.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download stream: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stream download failed (HTTP %d)", resp.StatusCode)
	}
	encrypted, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream: %w", err)
	}

	// 3. Decrypt.
	decrypted, err := DecryptStream(encrypted, meta.TrackID)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt stream: %w", err)
	}

	// 4. Hash + dedup.
	sum := sha256.Sum256(decrypted)
	hash := hex.EncodeToString(sum[:])

	exists, existingID, err := songExists(im.DB, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing song: %w", err)
	}

	var songID int
	if exists {
		songID = existingID
	} else {
		// 5. Upload to storage.
		ext := cfg.fileExt()
		objectName := hash + ext
		if err := im.Storage.UploadFile(ctx, objectName, bytes.NewReader(decrypted), int64(len(decrypted)), contentTypeForExt(ext)); err != nil {
			return nil, fmt.Errorf("failed to upload to storage: %w", err)
		}

		// 6. Insert song row with the provided (possibly edited) metadata.
		songID, err = saveTidalSong(ctx, im.DB, hash, objectName, meta, int64(len(decrypted)))
		if err != nil {
			return nil, fmt.Errorf("failed to save song: %w", err)
		}
	}

	// 7. Link to user.
	if err := db.AddUserSong(im.DB, userID, songID); err != nil {
		log.Printf("Failed to link song %d to user %d: %v", songID, userID, err)
	}

	return &ImportResult{SongID: songID, Dedup: exists}, nil
}

// songExists checks whether a song with the given fingerprint hash exists.
func songExists(db *sql.DB, hash string) (bool, int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM songs WHERE fingerprint_hash = $1", hash).Scan(&id)
	if err == sql.ErrNoRows {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, id, nil
}

// saveTidalSong inserts a song row using the provided metadata.
func saveTidalSong(ctx context.Context, db *sql.DB, hash, objectName string, meta TrackMeta, fileSize int64) (int, error) {
	query := `
		INSERT INTO songs (
			fingerprint_hash, file_path, title, artist, album, year,
			duration, bitrate, file_size, last_modified
		) VALUES ($1, $2, $3, $4, $5, $6, 0, 0, $7, NOW())
		RETURNING id
	`
	var songID int
	err := db.QueryRowContext(ctx, query,
		hash, objectName, meta.Title, meta.Artist, meta.Album, meta.Year, fileSize,
	).Scan(&songID)
	if err != nil {
		return 0, err
	}
	return songID, nil
}

// contentTypeForExt returns a MIME type for a given audio extension.
func contentTypeForExt(ext string) string {
	switch ext {
	case ".flac":
		return "audio/flac"
	case ".m4a":
		return "audio/mp4"
	case ".mp3":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}

// AddToPlaylist adds a song to a playlist by name (creating it if needed).
func (im *Importer) AddToPlaylist(ctx context.Context, userID, songID int, playlistName string) error {
	if playlistName == "" {
		return nil
	}
	pl, err := db.GetOrCreatePlaylist(im.DB, userID, playlistName)
	if err != nil {
		return fmt.Errorf("failed to get or create playlist: %w", err)
	}
	return db.AddSongToPlaylist(im.DB, pl.ID, songID, 0)
}
