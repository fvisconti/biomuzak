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
			s.genre_id, COALESCE(g.name, '') as genre, s.duration, s.bitrate, s.file_size, s.last_modified,
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

// AddUserSong links a song to a user in the library
func AddUserSong(db *sql.DB, userID int, songID int) error {
	// Insert with NULL rating if not exists
	query := `
		INSERT INTO user_songs (user_id, song_id, rating)
		VALUES ($1, $2, NULL)
		ON CONFLICT (user_id, song_id) DO NOTHING
	`
	_, err := db.Exec(query, userID, songID)
	return err
}

// DeleteUserSong removes a song from the user's library
func DeleteUserSong(db *sql.DB, userID int, songID int) error {
	query := `DELETE FROM user_songs WHERE user_id = $1 AND song_id = $2`
	_, err := db.Exec(query, userID, songID)
	return err
}

// UpdateSongGenre updates the genre of a song
func UpdateSongGenre(db *sql.DB, userID int, songID int, genreName string) error {
	// First, check if user owns this song
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM user_songs WHERE user_id = $1 AND song_id = $2)`, userID, songID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("song not found in user's library")
	}

	// Find or create genre
	var genreID int
	err = db.QueryRow(`SELECT id FROM genres WHERE name = $1`, genreName).Scan(&genreID)
	if err == sql.ErrNoRows {
		// Create new genre
		err = db.QueryRow(`INSERT INTO genres (name) VALUES ($1) RETURNING id`, genreName).Scan(&genreID)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Update song's genre
	query := `UPDATE songs SET genre_id = $1 WHERE id = $2`
	_, err = db.Exec(query, genreID, songID)
	return err
}

// GetAllArtists retrieves all artists from the database
func GetAllArtists(db *sql.DB) ([]*models.Artist, error) {
	rows, err := db.Query("SELECT id, name FROM artists ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []*models.Artist
	for rows.Next() {
		var artist models.Artist
		if err := rows.Scan(&artist.ID, &artist.Name); err != nil {
			return nil, err
		}
		artists = append(artists, &artist)
	}

	return artists, nil
}

// Search performs a search for artists, albums, and songs
func Search(db *sql.DB, query string) ([]*models.Artist, []*models.Album, []*models.Song, error) {
	query = "%" + query + "%"

	// Search artists
	artistRows, err := db.Query("SELECT id, name FROM artists WHERE name ILIKE $1", query)
	if err != nil {
		return nil, nil, nil, err
	}
	defer artistRows.Close()

	var artists []*models.Artist
	for artistRows.Next() {
		var artist models.Artist
		if err := artistRows.Scan(&artist.ID, &artist.Name); err != nil {
			return nil, nil, nil, err
		}
		artists = append(artists, &artist)
	}

	// Search albums
	albumRows, err := db.Query("SELECT id, name, artist FROM albums WHERE name ILIKE $1", query)
	if err != nil {
		return nil, nil, nil, err
	}
	defer albumRows.Close()

	var albums []*models.Album
	for albumRows.Next() {
		var album models.Album
		if err := albumRows.Scan(&album.ID, &album.Name, &album.Artist); err != nil {
			return nil, nil, nil, err
		}
		albums = append(albums, &album)
	}

	// Search songs
	songRows, err := db.Query("SELECT id, title, artist, album FROM songs WHERE title ILIKE $1", query)
	if err != nil {
		return nil, nil, nil, err
	}
	defer songRows.Close()

	var songs []*models.Song
	for songRows.Next() {
		var song models.Song
		if err := songRows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album); err != nil {
			return nil, nil, nil, err
		}
		songs = append(songs, &song)
	}

	return artists, albums, songs, nil
}

// GetSongFilePath retrieves the file path for a song by its ID
func GetSongFilePath(db *sql.DB, songID int) (string, error) {
	var filePath string
	err := db.QueryRow("SELECT file_path FROM songs WHERE id = $1", songID).Scan(&filePath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("song with ID %d not found", songID)
		}
		return "", err
	}
	return filePath, nil
}

// FindGenreByTrigramSearch finds the closest matching genre using trigram similarity.
func FindGenreByTrigramSearch(db *sql.DB, genreName string) (string, error) {
	var bestMatch string
	// ... (existing code)
	// This query calculates the similarity between the input genreName and all existing genre names.
	// It returns the name of the genre with the highest similarity, but only if the similarity is above a certain threshold (e.g., 0.3).
	// The threshold helps to avoid bad matches for very dissimilar names.
	query := `
		SELECT name FROM genres WHERE similarity(name, $1) > 0.3
		ORDER BY similarity(name, $1) DESC
		LIMIT 1
	`
	err := db.QueryRow(query, genreName).Scan(&bestMatch)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No good match found
		}
		return "", err
	}
	return bestMatch, nil
}

// GetAlbums retrieves a list of unique albums from the user's library
func GetAlbums(db *sql.DB, userID int) ([]models.VirtualAlbum, error) {
	query := `
		SELECT s.album, s.artist, COUNT(*) as song_count, MAX(s.year) as year
		FROM songs s
		JOIN user_songs us ON s.id = us.song_id
		WHERE us.user_id = $1 AND s.album IS NOT NULL AND s.album != ''
		GROUP BY s.album, s.artist
		ORDER BY s.album
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []models.VirtualAlbum
	for rows.Next() {
		var album models.VirtualAlbum
		var year sql.NullInt64
		if err := rows.Scan(&album.Name, &album.Artist, &album.SongCount, &year); err != nil {
			return nil, err
		}
		if year.Valid {
			album.Year = int(year.Int64)
		}
		albums = append(albums, album)
	}
	return albums, nil
}

// GetAlbumSongs retrieves songs for a specific album and artist
func GetAlbumSongs(db *sql.DB, userID int, albumName string, artistName string) ([]models.LibrarySong, error) {
	// Re-use GetSongsByUserID but we might need exact match for album/artist?
	// GetSongsByUserID uses ILIKE which is fuzzy.
	// For album details, we want exact matches usually, but ILIKE is probably fine for now.
	// However, if we want exact grouping, we should probably write a specific query or update filters to support exact match.
	// Let's use a specific query for precision.
	query := `
		SELECT
			s.id, s.fingerprint_hash, s.file_path, s.title, s.artist, s.album, s.year,
			s.genre_id, COALESCE(g.name, '') as genre, s.duration, s.bitrate, s.file_size, s.last_modified,
			us.rating
		FROM songs s
		LEFT JOIN genres g ON s.genre_id = g.id
		JOIN user_songs us ON s.id = us.song_id
		WHERE us.user_id = $1 AND s.album = $2 AND ($3 = '' OR s.artist = $3)
		ORDER BY s.title
	`
	// Note: Handling empty artist if songs have album but no artist?
	// Usually grouping is by (Album, Artist). If Artist is empty in DB, we pass empty string.

	rows, err := db.Query(query, userID, albumName, artistName)
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
