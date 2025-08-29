package db

import (
	"database/sql"
	"fmt"
	"go-postgres-example/pkg/models"
)

// GetSongsByUserID retrieves a user's songs from the database with optional filtering and sorting
func GetSongsByUserID(db *sql.DB, userID int, filters map[string]string, sortBy string) ([]models.LibrarySong, error) {
	// Base query
	query := `
		SELECT
			s.id, s.fingerprint_hash, s.file_path, s.title, s.artist, s.album, s.year,
			s.genre_id, g.name as genre, s.duration, s.bitrate, s.file_size, s.last_modified,
			us.rating
		FROM songs s
		LEFT JOIN genres g ON s.genre_id = g.id
		JOIN user_songs us ON s.id = us.song_id
		WHERE us.user_id = $1
	`

	args := []interface{}{userID}
	argID := 2

	// Add filters to the query
	for key, value := range filters {
		if value != "" {
			if key == "year" {
				query += fmt.Sprintf(" AND s.%s = $%d", key, argID)
			} else {
				query += fmt.Sprintf(" AND s.%s ILIKE $%d", key, argID)
				value = "%" + value + "%"
			}
			args = append(args, value)
			argID++
		}
	}

	// Add sorting to the query
	if sortBy != "" {
		// Whitelist the sortable columns to prevent SQL injection
		allowedSortBy := []string{"title", "artist", "album", "year", "duration", "rating", "last_modified"}
		for _, allowed := range allowedSortBy {
			if sortBy == allowed {
				query += fmt.Sprintf(" ORDER BY %s", sortBy)
				break
			}
		}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []models.LibrarySong
	for rows.Next() {
		var song models.LibrarySong
		if err := rows.Scan(
			&song.ID, &song.FingerprintHash, &song.FilePath, &song.Title, &song.Artist, &song.Album, &song.Year,
			&song.GenreID, &song.Genre, &song.Duration, &song.Bitrate, &song.FileSize, &song.LastModified,
			&song.Rating,
		); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}

	return songs, nil
}

// RateSong inserts or updates a user's rating for a song
func RateSong(db *sql.DB, userID int, songID int, rating int) error {
	query := `
		INSERT INTO user_songs (user_id, song_id, rating)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, song_id)
		DO UPDATE SET rating = $3
	`
	_, err := db.Exec(query, userID, songID, rating)
	return err
}
