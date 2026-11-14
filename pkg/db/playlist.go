package db

import (
	"database/sql"
	"errors"
	"fmt"

	"go-postgres-example/pkg/models"
)

// playlistSortColumns maps the allowed playlist sort keys to their ORDER BY
// expressions. Only these literal values ever reach query text (whitelist —
// the same pattern as GetSongsByUserID's sortBy).
var playlistSortColumns = map[string]string{
	"added_at": `ps.added_at DESC, ps.position ASC`,
	"title":    `s.title COLLATE "C" ASC, ps.position ASC`,
	"artist":   `s.artist COLLATE "C" ASC, s.title COLLATE "C" ASC, ps.position ASC`,
	"position": `ps.position ASC`,
}

// playlistSortKeyError is returned for sort keys outside the whitelist.
var playlistSortKeyError = errors.New("invalid playlist sort key")

// IsInvalidPlaylistSortKey reports whether err marks an unknown sort key.
func IsInvalidPlaylistSortKey(err error) bool { return errors.Is(err, playlistSortKeyError) }

// CreatePlaylist inserts a new playlist for a user into the database.
func CreatePlaylist(db *sql.DB, userID int, name string) (*models.Playlist, error) {
	query := `
		INSERT INTO playlists (user_id, name)
		VALUES ($1, $2)
		RETURNING id, user_id, name, created_at, updated_at, sort_key
	`
	playlist := &models.Playlist{}
	err := db.QueryRow(query, userID, name).Scan(
		&playlist.ID,
		&playlist.UserID,
		&playlist.Name,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
		&playlist.SortKey,
	)
	if err != nil {
		return nil, err
	}
	return playlist, nil
}

// GetUserPlaylists retrieves all playlists owned by a specific user.
func GetUserPlaylists(db *sql.DB, userID int) ([]models.Playlist, error) {
	query := `
		SELECT p.id, p.user_id, p.name, p.created_at, p.updated_at, p.sort_key, COUNT(ps.song_id) as song_count
		FROM playlists p
		LEFT JOIN playlist_songs ps ON p.id = ps.playlist_id
		WHERE p.user_id = $1
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []models.Playlist
	for rows.Next() {
		var p models.Playlist
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.CreatedAt, &p.UpdatedAt, &p.SortKey, &p.SongCount); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, nil
}

// GetPlaylistByID retrieves a single playlist by its ID, checking for user ownership.
func GetPlaylistByID(db *sql.DB, userID int, playlistID int) (*models.Playlist, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at, sort_key
		FROM playlists
		WHERE id = $1 AND user_id = $2
	`
	playlist := &models.Playlist{}
	err := db.QueryRow(query, playlistID, userID).Scan(
		&playlist.ID,
		&playlist.UserID,
		&playlist.Name,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
		&playlist.SortKey,
	)
	if err != nil {
		// Return the error, the handler can check for sql.ErrNoRows
		return nil, err
	}
	return playlist, nil
}

// GetPlaylistSongs retrieves all songs for a given playlist. Without a sort
// key (or with one outside the whitelist) they are ordered most-recent first
// (E6 default); otherwise by the whitelisted column.
func GetPlaylistSongs(db *sql.DB, playlistID int, sortBy string) ([]models.PlaylistSong, error) {
	order, ok := playlistSortColumns[sortBy]
	if !ok {
		order = playlistSortColumns["added_at"]
	}
	query := `
		SELECT
			s.id, s.fingerprint_hash, s.file_path, s.title, s.artist, s.album, s.year,
			s.genre_id, COALESCE(g.name, '') as genre, s.duration, s.bitrate, s.file_size, s.last_modified,
			ps.position, ps.added_at
		FROM songs s
		LEFT JOIN genres g ON s.genre_id = g.id
		JOIN playlist_songs ps ON s.id = ps.song_id
		WHERE ps.playlist_id = $1
		ORDER BY ` + order
	rows, err := db.Query(query, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []models.PlaylistSong
	for rows.Next() {
		var song models.PlaylistSong
		// Scan into the embedded Song struct's fields directly
		if err := rows.Scan(
			&song.Song.ID, &song.Song.FingerprintHash, &song.Song.FilePath, &song.Song.Title, &song.Song.Artist, &song.Song.Album, &song.Song.Year,
			&song.Song.GenreID, &song.Song.Genre, &song.Song.Duration, &song.Song.Bitrate, &song.Song.FileSize, &song.Song.LastModified,
			&song.Position, &song.AddedAt,
		); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, nil
}

// ErrPlaylistOrderMismatch marks a bulk order whose song set doesn't match
// the playlist's contents (wrong count or a duplicated song ID).
var ErrPlaylistOrderMismatch = errors.New("playlist order mismatch")

// UpdatePlaylistFields updates a playlist's name and/or sort key in a single
// statement, checking user ownership. NULL fields leave the existing value
// untouched (COALESCE). A sort key outside the whitelist is rejected before
// any DB access.
func UpdatePlaylistFields(db *sql.DB, userID int, playlistID int, name, sortKey sql.NullString) error {
	if sortKey.Valid {
		if _, ok := playlistSortColumns[sortKey.String]; !ok {
			return playlistSortKeyError
		}
	}
	query := `
		UPDATE playlists
		SET name = COALESCE($3, name), sort_key = COALESCE($4, sort_key), updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`
	res, err := db.Exec(query, playlistID, userID, name, sortKey)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows // Not found or not authorized
	}
	return nil
}

// ReorderPlaylistSongs rewrites the positions of a playlist's songs to match
// the given full order (songIDs[0] becomes position 1, and so on). The list
// must contain exactly the playlist's songs, otherwise
// ErrPlaylistOrderMismatch is returned. Runs in one transaction.
func ReorderPlaylistSongs(db *sql.DB, playlistID int, songIDs []int) error {
	if len(songIDs) == 0 {
		return nil // empty playlist: nothing to order
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1", playlistID).Scan(&count)
	if err != nil {
		return err
	}
	if count != len(songIDs) {
		return fmt.Errorf("%w: playlist has %d songs, got %d", ErrPlaylistOrderMismatch, count, len(songIDs))
	}
	seen := make(map[int]bool, len(songIDs))
	for _, songID := range songIDs {
		if seen[songID] {
			return fmt.Errorf("%w: duplicate song id %d", ErrPlaylistOrderMismatch, songID)
		}
		seen[songID] = true
	}

	for i, songID := range songIDs {
		res, err := tx.Exec("UPDATE playlist_songs SET position = $1 WHERE playlist_id = $2 AND song_id = $3", i+1, playlistID, songID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf("%w: song %d not in playlist %d", ErrPlaylistOrderMismatch, songID, playlistID)
		}
	}

	return tx.Commit()
}

// DeletePlaylist deletes a playlist, checking for user ownership.
func DeletePlaylist(db *sql.DB, userID int, playlistID int) error {
	query := `
		DELETE FROM playlists
		WHERE id = $1 AND user_id = $2
	`
	res, err := db.Exec(query, playlistID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows // Not found or not authorized
	}
	return nil
}

// AddSongToPlaylist adds a song to a playlist.
func AddSongToPlaylist(db *sql.DB, playlistID int, songID int, position int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove the song if it's already in the playlist to handle re-ordering.
	_, err = tx.Exec("DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2", playlistID, songID)
	if err != nil {
		return err
	}

	// If position is not specified (<= 0), append to the end.
	if position <= 0 {
		err = tx.QueryRow("SELECT COALESCE(MAX(position), 0) + 1 FROM playlist_songs WHERE playlist_id = $1", playlistID).Scan(&position)
		if err != nil {
			return err
		}
	} else {
		// Shift existing songs to make space.
		_, err = tx.Exec("UPDATE playlist_songs SET position = position + 1 WHERE playlist_id = $1 AND position >= $2", playlistID, position)
		if err != nil {
			return err
		}
	}

	// Insert the song at the desired position (added_at marks the add time).
	_, err = tx.Exec("INSERT INTO playlist_songs (playlist_id, song_id, position, added_at) VALUES ($1, $2, $3, NOW())", playlistID, songID, position)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveSongFromPlaylist removes a song from a playlist and re-orders the remaining songs.
func RemoveSongFromPlaylist(db *sql.DB, playlistID int, songID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var removedPosition int
	err = tx.QueryRow("SELECT position FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2", playlistID, songID).Scan(&removedPosition)
	if err != nil {
		if err == sql.ErrNoRows {
			// Song isn't in the playlist, so there's nothing to do.
			return nil
		}
		return err
	}

	_, err = tx.Exec("DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2", playlistID, songID)
	if err != nil {
		return err
	}

	// Shift subsequent songs to close the gap.
	_, err = tx.Exec("UPDATE playlist_songs SET position = position - 1 WHERE playlist_id = $1 AND position > $2", playlistID, removedPosition)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetOrCreatePlaylist retrieves a playlist by name for a user, or creates it if it doesn't exist.
func GetOrCreatePlaylist(db *sql.DB, userID int, name string) (*models.Playlist, error) {
	// Try to find existing playlist
	query := `SELECT id, user_id, name, created_at, updated_at, sort_key FROM playlists WHERE user_id = $1 AND name = $2`
	playlist := &models.Playlist{}
	err := db.QueryRow(query, userID, name).Scan(
		&playlist.ID,
		&playlist.UserID,
		&playlist.Name,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
		&playlist.SortKey,
	)
	if err == nil {
		return playlist, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new playlist if not found
	return CreatePlaylist(db, userID, name)
}
