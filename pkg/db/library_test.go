package db

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// TestGetSongsByUserID_GlobalCatalog asserts that the library listing is a
// GLOBAL catalog: the query must LEFT JOIN user_songs (so a user's rating shows
// when present) rather than hard-filtering on user_songs.user_id. This is the
// E1 "no track is owned" change.
func TestGetSongsByUserID_GlobalCatalog(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
		"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "rating",
	}).
		AddRow(1, "hash1", "path1", "title1", "artist1", "album1", 2021, 1, "genre1", 180, 320, 12345, time.Now(), 5).
		AddRow(2, "hash2", "path2", "title2", "artist2", "album2", 2020, 2, "genre2", 200, 320, 22345, time.Now(), nil)

	mock.ExpectQuery(regexp.QuoteMeta("LEFT JOIN user_songs us ON s.id = us.song_id AND us.user_id = $1")).
		WithArgs(1).
		WillReturnRows(rows)

	songs, err := GetSongsByUserID(db, 1, map[string]string{}, "")
	assert.NoError(t, err)
	assert.Len(t, songs, 2)
	assert.Equal(t, 1, songs[0].ID)
	assert.True(t, songs[0].Rating.Valid)
	assert.Equal(t, 5, int(songs[0].Rating.Int64))
	// A song the user has not rated still appears (global catalog), rating NULL.
	assert.False(t, songs[1].Rating.Valid)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSongGenre_NoOwnershipCheck asserts that updating a song's genre no
// longer requires the song to be in the requesting user's library. The old code
// ran a SELECT EXISTS(...) against user_songs first; the new code must not.
func TestUpdateSongGenre_NoOwnershipCheck(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// No expectation for a SELECT EXISTS(...) ownership check. If the code still
	// performs it, sqlmock will fail with "call to Query was not expected".
	// The INSERT ... RETURNING id runs via QueryRow (a Query), not Exec.
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO genres (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id")).
		WithArgs("House").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE songs SET genre_id = $1 WHERE id = $2")).
		WithArgs(1, 42).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = UpdateSongGenre(db, 42, "House")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAlbums_Global asserts albums are aggregated from the global catalog
// (no user filter).
func TestGetAlbums_Global(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"album", "artist", "song_count", "year"}).
		AddRow("Album1", "Artist1", 3, 2021)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT s.album, s.artist, COUNT(*) as song_count, MAX(s.year) as year")).
		WillReturnRows(rows)

	albums, err := GetAlbums(db)
	assert.NoError(t, err)
	assert.Len(t, albums, 1)
	assert.Equal(t, "Album1", albums[0].Name)
	assert.Equal(t, 3, albums[0].SongCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSongsByUserID_RatingIsolation asserts that the LEFT JOIN is scoped to
// the requesting user: the query must bind the requesting userID as the first
// argument so that only that user's rating is joined. A different user's rating
// for the same song must NOT appear.
func TestGetSongsByUserID_RatingIsolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Simulate: user 42 requests the catalog. The mock returns a song where
	// user 42 has NOT rated it (rating NULL). If the join were unscoped or
	// bound a different user, the rating column would be non-NULL.
	rows := sqlmock.NewRows([]string{
		"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
		"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "rating",
	}).
		AddRow(10, "hash10", "path10", "Shared Song", "Artist", "Album", 2023, 0, "", 200, 320, 99999, time.Now(), nil)

	// The join predicate must carry the requesting user's ID (42) as $1.
	mock.ExpectQuery(regexp.QuoteMeta("LEFT JOIN user_songs us ON s.id = us.song_id AND us.user_id = $1")).
		WithArgs(42).
		WillReturnRows(rows)

	songs, err := GetSongsByUserID(db, 42, map[string]string{}, "")
	assert.NoError(t, err)
	assert.Len(t, songs, 1)
	// User 42 has not rated this song → rating must be NULL, not another user's.
	assert.False(t, songs[0].Rating.Valid, "rating must be NULL for a song the requesting user has not rated")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAlbumSongs_Global asserts album songs come from the global catalog with
// a LEFT JOIN for the requesting user's rating (no hard user filter).
func TestGetAlbumSongs_Global(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
		"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "rating",
	}).
		AddRow(1, "hash1", "path1", "title1", "artist1", "album1", 2021, 1, "genre1", 180, 320, 12345, time.Now(), nil)

	mock.ExpectQuery(regexp.QuoteMeta("LEFT JOIN user_songs us ON s.id = us.song_id AND us.user_id = $1")).
		WithArgs(1, "album1", "artist1").
		WillReturnRows(rows)

	songs, err := GetAlbumSongs(db, 1, "album1", "artist1")
	assert.NoError(t, err)
	assert.Len(t, songs, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}
