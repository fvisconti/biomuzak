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
// Genre is stamped by the server only — ApplySourceGenre matches the import
// source name and sets it (E3b); empty when the name matches nothing. It has
// no JSON tag on purpose: a client-supplied "genre" in the import body must
// never reach the shared genre catalogue (genre editing is admin-only).
type TrackMeta struct {
	TrackID int    `json:"track_id"`
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Album   string `json:"album"`
	Year    int    `json:"year"`
	Genre   string `json:"-"`
}

// ImportResult reports the outcome of a single track import.
type ImportResult struct {
	SongID int    `json:"song_id"`
	Dedup  bool   `json:"dedup"` // true if the file already existed
	Error  string `json:"error,omitempty"`
}

// ApplySourceGenre matches the import source name (playlist/album name) against
// the genre catalogue and stamps the result on every track, so imported songs
// inherit the matched genre (E3b). It returns the matched genre ("" when there
// is no match or the name is empty). A missing match is not an error.
func (im *Importer) ApplySourceGenre(ctx context.Context, tracks []TrackMeta, sourceName string) (string, error) {
	if sourceName == "" {
		return "", nil
	}
	genre, err := db.FindGenreByTrigramSearch(im.DB, sourceName)
	if err != nil {
		return "", fmt.Errorf("genre match failed: %w", err)
	}
	for i := range tracks {
		tracks[i].Genre = genre
	}
	return genre, nil
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
		// E4: informational object tags (genre/artist); best-effort.
		im.tagObject(ctx, objectName, &meta)

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

// tagObject applies the E4 informational object tags (genre/artist) to a
// stored object. Best-effort: tagging failures are logged, never returned,
// so a MinIO hiccup cannot fail an otherwise successful import.
func (im *Importer) tagObject(ctx context.Context, objectName string, meta *TrackMeta) {
	tagMap := map[string]string{}
	if meta.Genre != "" {
		tagMap["genre"] = meta.Genre
	}
	if meta.Artist != "" {
		tagMap["artist"] = meta.Artist
	}
	if err := im.Storage.TagObject(ctx, objectName, tagMap); err != nil {
		log.Printf("failed to tag object %s: %v", objectName, err)
	}
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

// saveTidalSong inserts a song row using the provided metadata. A non-empty
// meta.Genre is resolved to a genre_id by lookup only (no insert — a deleted
// genre must stay deleted); a vanished genre degrades to a NULL genre.
func saveTidalSong(ctx context.Context, db *sql.DB, hash, objectName string, meta TrackMeta, fileSize int64) (int, error) {
	var genreID sql.NullInt64
	if meta.Genre != "" {
		var id int
		// Lookup only — never re-insert. The genre was already matched by
		// ApplySourceGenre; if the row vanished in the meantime (admin
		// delete), the song keeps its data with a NULL genre instead of
		// resurrecting the deleted row.
		err := db.QueryRowContext(ctx, `SELECT id FROM genres WHERE name = $1`, meta.Genre).Scan(&id)
		switch {
		case err == sql.ErrNoRows:
			log.Printf("tidal import: genre %q not found, saving song without genre", meta.Genre)
		case err != nil:
			return 0, err
		default:
			genreID = sql.NullInt64{Int64: int64(id), Valid: true}
		}
	}

	query := `
		INSERT INTO songs (
			fingerprint_hash, file_path, title, artist, album, year, genre_id,
			duration, bitrate, file_size, last_modified
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 0, 0, $8, NOW())
		RETURNING id
	`
	var songID int
	err := db.QueryRowContext(ctx, query,
		hash, objectName, meta.Title, meta.Artist, meta.Album, meta.Year, genreID, fileSize,
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
	// E3b: give the playlist a genre matched from its name, without overwriting
	// one set explicitly. A match failure is non-fatal for the import.
	if genre, err := db.FindGenreByTrigramSearch(im.DB, playlistName); err == nil && genre != "" {
		if err := db.SetPlaylistGenreIfEmpty(im.DB, pl.ID, genre); err != nil {
			log.Printf("Failed to set genre %q on playlist %d: %v", genre, pl.ID, err)
		}
	}
	return db.AddSongToPlaylist(im.DB, pl.ID, songID, 0)
}
