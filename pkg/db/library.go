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
			if key == "q" {
				query += fmt.Sprintf(" AND (s.title ILIKE $%d OR s.artist ILIKE $%d OR s.album ILIKE $%d)", argID, argID, argID)
				value = "%" + value + "%"
			} else if key == "year" {
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

// GetOrphanedSongsForUser returns the IDs and file paths of songs that are
// referenced ONLY by the given user (no other user has them in their library).
// These become orphaned when the user is deleted and can be garbage-collected.
func GetOrphanedSongsForUser(db *sql.DB, userID int) ([]models.Song, error) {
	query := `
		SELECT s.id, s.file_path
		FROM songs s
		JOIN user_songs us ON us.song_id = s.id
		WHERE us.user_id = $1
		  AND NOT EXISTS (
			SELECT 1 FROM user_songs us2
			WHERE us2.song_id = s.id AND us2.user_id <> $1
		  )
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []models.Song
	for rows.Next() {
		var song models.Song
		if err := rows.Scan(&song.ID, &song.FilePath); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, nil
}

// DeleteSongs removes songs by ID (and cascades their embeddings).
func DeleteSongs(db *sql.DB, songIDs []int) error {
	if len(songIDs) == 0 {
		return nil
	}
	_, err := db.Exec("DELETE FROM songs WHERE id = ANY($1)", songIDs)
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

	// Find or create genre atomically to avoid a check-then-insert race.
	var genreID int
	err = db.QueryRow(`INSERT INTO genres (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id`, genreName).Scan(&genreID)
	if err == sql.ErrNoRows {
		// A concurrent insert won the race; fetch the existing row.
		err = db.QueryRow(`SELECT id FROM genres WHERE name = $1`, genreName).Scan(&genreID)
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

// GetAllArtists retrieves all distinct artists from the songs table.
// The dedicated artists/albums tables are not part of the schema, so we
// aggregate from songs to keep search functional.
func GetAllArtists(db *sql.DB) ([]*models.Artist, error) {
	rows, err := db.Query("SELECT DISTINCT artist FROM songs WHERE artist IS NOT NULL AND artist != '' ORDER BY artist")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []*models.Artist
	for rows.Next() {
		var artist models.Artist
		if err := rows.Scan(&artist.Name); err != nil {
			return nil, err
		}
		artists = append(artists, &artist)
	}

	return artists, nil
}

// Search performs a search for artists, albums, and songs by aggregating
// from the songs table (the dedicated artists/albums tables do not exist).
func Search(db *sql.DB, query string) ([]*models.Artist, []*models.Album, []*models.Song, error) {
	pattern := "%" + query + "%"

	// Search artists (distinct artist names)
	artistRows, err := db.Query("SELECT DISTINCT artist FROM songs WHERE artist IS NOT NULL AND artist != '' AND artist ILIKE $1 ORDER BY artist", pattern)
	if err != nil {
		return nil, nil, nil, err
	}
	defer artistRows.Close()

	var artists []*models.Artist
	for artistRows.Next() {
		var artist models.Artist
		if err := artistRows.Scan(&artist.Name); err != nil {
			return nil, nil, nil, err
		}
		artists = append(artists, &artist)
	}

	// Search albums (distinct album/artist pairs)
	albumRows, err := db.Query("SELECT DISTINCT album, artist FROM songs WHERE album IS NOT NULL AND album != '' AND album ILIKE $1 ORDER BY album", pattern)
	if err != nil {
		return nil, nil, nil, err
	}
	defer albumRows.Close()

	var albums []*models.Album
	for albumRows.Next() {
		var album models.Album
		if err := albumRows.Scan(&album.Name, &album.Artist); err != nil {
			return nil, nil, nil, err
		}
		albums = append(albums, &album)
	}

	// Search songs
	songRows, err := db.Query("SELECT id, title, artist, album FROM songs WHERE title ILIKE $1 ORDER BY title LIMIT 100", pattern)
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

// GetSongFilePathForUser retrieves the file path for a song, but only if the
// given user has the song in their library. This prevents one user from
// streaming/downloading another user's private uploads (IDOR).
func GetSongFilePathForUser(db *sql.DB, userID, songID int) (string, error) {
	var filePath string
	err := db.QueryRow(`
		SELECT s.file_path
		FROM songs s
		JOIN user_songs us ON us.song_id = s.id
		WHERE s.id = $1 AND us.user_id = $2`, songID, userID).Scan(&filePath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("song with ID %d not found", songID)
		}
		return "", err
	}
	return filePath, nil
}

// GetSongByID retrieves song details by its ID
func GetSongByID(db *sql.DB, songID int) (*models.Song, error) {
	var song models.Song
	query := `SELECT id, title, artist, album, file_path FROM songs WHERE id = $1`
	err := db.QueryRow(query, songID).Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.FilePath)
	if err != nil {
		return nil, err
	}
	return &song, nil
}

// GetSongByIDForUser retrieves song details by ID, but only if the given user
// has the song in their library (IDOR protection).
func GetSongByIDForUser(db *sql.DB, userID, songID int) (*models.Song, error) {
	var song models.Song
	query := `
		SELECT s.id, s.title, s.artist, s.album, s.file_path
		FROM songs s
		JOIN user_songs us ON us.song_id = s.id
		WHERE s.id = $1 AND us.user_id = $2`
	err := db.QueryRow(query, songID, userID).Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.FilePath)
	if err != nil {
		return nil, err
	}
	return &song, nil
}

// FindGenreByTrigramSearch finds the closest matching genre using trigram similarity.
func FindGenreByTrigramSearch(db *sql.DB, genreName string) (string, error) {
	var bestMatch string
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
