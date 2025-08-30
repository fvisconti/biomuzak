package db

import (
	"database/sql"
	"fmt"
	"go-postgres-example/pkg/models"
	"strconv"
	"strings"
)

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
func FindSimilarSongs(db *sql.DB, songIDToExclude int, queryEmbedding []float64, topN int) ([]SimilarSong, error) {
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
		ORDER BY similarity DESC
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

// SaveSongEmbedding saves the embedding for a given song ID.
func SaveSongEmbedding(db *sql.DB, songID int, embedding []float64) error {
	embeddingStr := vectorToString(embedding)
	query := "INSERT INTO song_embeddings (song_id, embedding) VALUES ($1, $2::vector)"
	_, err := db.Exec(query, songID, embeddingStr)
	return err
}
