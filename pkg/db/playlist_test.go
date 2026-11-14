package db

// Tests for playlist sorting (E6).
//
// Objective: playlist songs default to most-recent-first (added_at DESC) and
// support a server-side sort key; the sort key itself is stored per playlist.
//
// Given/When/Then:
//   - GetPlaylistSongs: given no sort key, songs come back ordered by
//     added_at DESC (position as tiebreak); given "title"/"artist"/"position"
//     they come back ordered by that column; an unknown key falls back to the
//     default instead of erroring.
//   - UpdatePlaylistSortKey: given a whitelisted key, it is persisted for the
//     owning user only; an unknown playlist returns sql.ErrNoRows; a key
//     outside the whitelist is rejected before any DB access.
//   - AddSongToPlaylist: given an add, the row is inserted with added_at.
//
// Edge cases: unknown sort key, non-owner update, invalid sort key.

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

var playlistSongCols = []string{
	"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
	"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified",
	"position", "added_at",
}

func playlistSongRow(n int) *sqlmock.Rows {
	return sqlmock.NewRows(playlistSongCols).AddRow(
		n, "hash", "path", "Title", "Artist", "Album", 2020,
		sql.NullInt64{}, "Genre", 300, 320, int64(1000), time.Now(),
		n, time.Now(),
	)
}

func TestGetPlaylistSongs_DefaultOrderByAddedAt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`ORDER BY ps\.added_at DESC`).
		WillReturnRows(playlistSongRow(1))

	songs, err := GetPlaylistSongs(db, 2, "")
	assert.NoError(t, err)
	assert.Len(t, songs, 1)
	assert.NotZero(t, songs[0].AddedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPlaylistSongs_SortByTitle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`ORDER BY s\.title COLLATE "C"`).
		WillReturnRows(playlistSongRow(1))

	_, err = GetPlaylistSongs(db, 2, "title")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPlaylistSongs_SortByArtist(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`ORDER BY s\.artist COLLATE "C"`).
		WillReturnRows(playlistSongRow(1))

	_, err = GetPlaylistSongs(db, 2, "artist")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPlaylistSongs_SortByPosition(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`ORDER BY ps\.position ASC`).
		WillReturnRows(playlistSongRow(1))

	_, err = GetPlaylistSongs(db, 2, "position")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPlaylistSongs_UnknownSortFallsBackToDefault(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// An unknown key must not error and must not be interpolated — it gets
	// the default added_at ordering.
	mock.ExpectQuery(`ORDER BY ps\.added_at DESC`).
		WillReturnRows(playlistSongRow(1))

	_, err = GetPlaylistSongs(db, 2, "bogus; DROP TABLE songs")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePlaylistFields_Both(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE playlists SET name = COALESCE($3, name), sort_key = COALESCE($4, sort_key), updated_at = NOW() WHERE id = $1 AND user_id = $2")).
		WithArgs(2, 1, "Renamed", "title").
		WillReturnResult(sqlmock.NewResult(1, 1))

	assert.NoError(t, UpdatePlaylistFields(db, 1, 2,
		sql.NullString{String: "Renamed", Valid: true},
		sql.NullString{String: "title", Valid: true}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePlaylistFields_SortKeyOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// A NULL name leaves the existing name untouched (COALESCE).
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE playlists SET name = COALESCE($3, name), sort_key = COALESCE($4, sort_key), updated_at = NOW() WHERE id = $1 AND user_id = $2")).
		WithArgs(2, 1, nil, "title").
		WillReturnResult(sqlmock.NewResult(1, 1))

	assert.NoError(t, UpdatePlaylistFields(db, 1, 2, sql.NullString{},
		sql.NullString{String: "title", Valid: true}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePlaylistFields_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE playlists SET name = COALESCE($3, name), sort_key = COALESCE($4, sort_key), updated_at = NOW() WHERE id = $1 AND user_id = $2")).
		WithArgs(99, 1, nil, "title").
		WillReturnResult(sqlmock.NewResult(1, 0))

	assert.ErrorIs(t, UpdatePlaylistFields(db, 1, 99, sql.NullString{},
		sql.NullString{String: "title", Valid: true}), sql.ErrNoRows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePlaylistFields_InvalidSortKey(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Whitelisted before any DB access: no expectations at all.
	err = UpdatePlaylistFields(db, 1, 2, sql.NullString{},
		sql.NullString{String: "bogus", Valid: true})
	assert.Error(t, err)
	assert.True(t, IsInvalidPlaylistSortKey(err))
}

func TestReorderPlaylistSongs_RewritesPositions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1")).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	for pos, songID := range []int{7, 3, 9} {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE playlist_songs SET position = $1 WHERE playlist_id = $2 AND song_id = $3")).
			WithArgs(pos+1, 2, songID).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	assert.NoError(t, ReorderPlaylistSongs(db, 2, []int{7, 3, 9}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReorderPlaylistSongs_MismatchCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1")).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectRollback()

	// Two IDs for a three-song playlist: rejected, nothing rewritten.
	err = ReorderPlaylistSongs(db, 2, []int{7, 3})
	assert.ErrorIs(t, err, ErrPlaylistOrderMismatch)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReorderPlaylistSongs_DuplicateID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1")).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectRollback()

	err = ReorderPlaylistSongs(db, 2, []int{7, 7})
	assert.ErrorIs(t, err, ErrPlaylistOrderMismatch)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddSongToPlaylist_SetsAddedAt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2")).
		WithArgs(2, 5).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(MAX(position), 0) + 1 FROM playlist_songs WHERE playlist_id = $1")).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"position"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO playlist_songs (playlist_id, song_id, position, added_at)")).
		WithArgs(2, 5, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	assert.NoError(t, AddSongToPlaylist(db, 2, 5, 0))
	assert.NoError(t, mock.ExpectationsWereMet())
}
