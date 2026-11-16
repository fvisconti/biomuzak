package db

import (
	"database/sql"
)

// SetPlaylistGenreIfEmpty sets the playlist's genre only when it is currently
// unset, so an explicit genre is never overwritten by name-based matching.
func SetPlaylistGenreIfEmpty(db *sql.DB, playlistID int, genre string) error {
	if genre == "" {
		return nil
	}
	_, err := db.Exec(
		`UPDATE playlists SET genre = $1, updated_at = NOW() WHERE id = $2 AND (genre IS NULL OR genre = '')`,
		genre, playlistID,
	)
	return err
}
