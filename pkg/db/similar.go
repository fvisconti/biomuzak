package db

import (
	"context"
	"database/sql"
	"fmt"
	"go-postgres-example/pkg/models"
	"strconv"
	"strings"
)

// EmbeddingDim is the fixed width of every song embedding in this system:
// 38 dimensions (13 MFCC mean + 13 MFCC cov + 6 spectral mean + 6 spectral
// cov), matching the vector(38) column in song_embeddings. Enforcing it here
// — before any pgvector string is built — keeps a malformed vector from
// reaching SQL (see vchord.instructions.md).
const EmbeddingDim = 38

// checkEmbeddingDimension returns an error unless v has exactly EmbeddingDim
// elements. Callers must use it before building any query text.
func checkEmbeddingDimension(v []float64) error {
	if len(v) != EmbeddingDim {
		return fmt.Errorf("invalid embedding dimension: got %d, want %d", len(v), EmbeddingDim)
	}
	return nil
}

// vectorToString converts a slice of floats to a pgvector-compatible string.
func vectorToString(v []float64) string {
	vals := make([]string, len(v))
	for i, val := range v {
		vals[i] = strconv.FormatFloat(val, 'f', -1, 64)
	}
	return fmt.Sprintf("[%s]", strings.Join(vals, ","))
}

// stringToVector converts a pgvector string representation to a slice of floats.
func stringToVector(s string) ([]float64, error) {
	s = strings.Trim(s, "[]")
	if s == "" {
		return []float64{}, nil
	}
	parts := strings.Split(s, ",")
	vec := make([]float64, len(parts))
	for i, p := range parts {
		val, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, err
		}
		vec[i] = val
	}
	return vec, nil
}

// GetSongEmbedding retrieves the embedding for a given song ID.
func GetSongEmbedding(db *sql.DB, songID int) ([]float64, error) {
	var embeddingStr string
	// Casting to text to handle the vector type
	query := "SELECT embedding::text FROM song_embeddings WHERE song_id = $1"
	err := db.QueryRow(query, songID).Scan(&embeddingStr)
	if err != nil {
		return nil, err
	}
	return stringToVector(embeddingStr)
}

// SimilarSong represents a song with its similarity score.
type SimilarSong struct {
	models.Song
	Similarity float64 `json:"similarity"`
}

// FindSimilarSongs finds the top N most similar songs to a given embedding.
// The query embedding must be exactly EmbeddingDim-dimensional; the check
// happens in Go, before the k-NN SQL is built.
func FindSimilarSongs(db *sql.DB, songIDToExclude int, queryEmbedding []float64, topN int) ([]SimilarSong, error) {
	if err := checkEmbeddingDimension(queryEmbedding); err != nil {
		return nil, err
	}
	embeddingStr := vectorToString(queryEmbedding)

	query := `
		SELECT
			s.id, s.fingerprint_hash, s.file_path, s.title, s.artist, s.album, s.year,
			s.genre_id, g.name as genre, s.duration, s.bitrate, s.file_size, s.last_modified,
			1 - (se.embedding <=> $1) AS similarity
		FROM song_embeddings se
		JOIN songs s ON se.song_id = s.id
		LEFT JOIN genres g ON s.genre_id = g.id
		WHERE se.song_id != $2
		ORDER BY se.embedding <=> $1
		LIMIT $3
	`

	rows, err := db.Query(query, embeddingStr, songIDToExclude, topN)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []SimilarSong
	for rows.Next() {
		var song SimilarSong
		if err := rows.Scan(
			&song.ID, &song.FingerprintHash, &song.FilePath, &song.Title, &song.Artist, &song.Album, &song.Year,
			&song.GenreID, &song.Genre, &song.Duration, &song.Bitrate, &song.FileSize, &song.LastModified,
			&song.Similarity,
		); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, nil
}

// GetSongsMissingEmbeddings returns the IDs of all songs that have no row in
// song_embeddings. Used by the cold-start backfill to generate embeddings for
// pre-existing library songs, Tidal imports, and past processor failures.
func GetSongsMissingEmbeddings(ctx context.Context, db *sql.DB) ([]int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT s.id
		FROM songs s
		LEFT JOIN song_embeddings se ON se.song_id = s.id
		WHERE se.song_id IS NULL
		ORDER BY s.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SaveSongEmbedding saves the embedding for a given song ID. The embedding
// must be exactly EmbeddingDim-dimensional; the check happens in Go, before
// the INSERT is built.
func SaveSongEmbedding(db *sql.DB, songID int, embedding []float64) error {
	if err := checkEmbeddingDimension(embedding); err != nil {
		return err
	}
	embeddingStr := vectorToString(embedding)
	query := `
		INSERT INTO song_embeddings (song_id, embedding)
		VALUES ($1, $2::vector)
		ON CONFLICT (song_id) DO UPDATE SET embedding = EXCLUDED.embedding
	`
	_, err := db.Exec(query, songID, embeddingStr)
	return err
}
