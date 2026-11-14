package tidal

// Tests for E3b genre wiring in the Tidal import pipeline.
//
// Objective: imported songs and the import playlist inherit a genre matched
// from the source (playlist/album) name via the trigram genre catalogue.
//
// Given/When/Then:
//   - ApplySourceGenre: given a source name and tracks, when the catalogue has
//     a match the tracks are stamped with it; when nothing matches (or the
//     name is empty) nothing is stamped and no error is returned.
//   - saveTidalSong: given meta with a genre, the genres row is upserted and
//     the songs INSERT binds the resolved genre_id; without a genre, no
//     genres upsert happens and a NULL genre_id is bound.
//   - AddToPlaylist: when the playlist is created, its genre is set from the
//     name match only if currently empty.
//
// Edge cases: empty source name (no query), no catalogue match ("" genre),
// existing playlist with an empty genre.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"regexp"
	"testing"
	"time"

	"go-postgres-example/pkg/storage"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

var trigramFilterPattern = `name\s+%`

func newTestImporter(db *sql.DB) *Importer {
	return &Importer{DB: db}
}

func TestApplySourceGenre_StampsTracks(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(trigramFilterPattern).
		WithArgs("Deep House Mix").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Deep House"))

	im := newTestImporter(db)
	tracks := []TrackMeta{
		{TrackID: 1, Title: "a"},
		{TrackID: 2, Title: "b"},
	}

	got, err := im.ApplySourceGenre(context.Background(), tracks, "Deep House Mix")
	assert.NoError(t, err)
	assert.Equal(t, "Deep House", got)
	assert.Equal(t, "Deep House", tracks[0].Genre)
	assert.Equal(t, "Deep House", tracks[1].Genre)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestApplySourceGenre_NoMatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(trigramFilterPattern).
		WithArgs("Zzzyzx").
		WillReturnError(sql.ErrNoRows)

	im := newTestImporter(db)
	tracks := []TrackMeta{{TrackID: 1, Title: "a"}}

	got, err := im.ApplySourceGenre(context.Background(), tracks, "Zzzyzx")
	assert.NoError(t, err) // no match is not an error
	assert.Equal(t, "", got)
	assert.Empty(t, tracks[0].Genre)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveTidalSong_GenreVanishedFallsBackToNull(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM genres WHERE name = $1")).
		WithArgs("Deep House").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO songs`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(56))

	meta := TrackMeta{Title: "t", Genre: "Deep House"}
	id, err := saveTidalSong(context.Background(), db, "abc", "abc.flac", meta, 1000)
	assert.NoError(t, err)
	assert.Equal(t, 56, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestTrackMeta_GenreNotClientInjectable asserts the import body cannot set a
// track's genre: TrackMeta.Genre carries no JSON tag (server-authoritative,
// stamped only by ApplySourceGenre from the matched source name). A client
// sending "genre" in the body must not reach the shared genre catalogue.
func TestTrackMeta_GenreNotClientInjectable(t *testing.T) {
	var meta TrackMeta
	err := json.Unmarshal([]byte(`{"track_id":1,"title":"t","genre":"Evil"}`), &meta)
	assert.NoError(t, err)
	assert.Empty(t, meta.Genre)
}

func TestApplySourceGenre_EmptyName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// No expectations at all: an empty source name must not query the DB.

	im := newTestImporter(db)
	tracks := []TrackMeta{{TrackID: 1, Title: "a"}}

	got, err := im.ApplySourceGenre(context.Background(), tracks, "")
	assert.NoError(t, err)
	assert.Equal(t, "", got)
	assert.Empty(t, tracks[0].Genre)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveTidalSong_WithGenre(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Lookup only (no re-insert): the genre was matched upstream.
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM genres WHERE name = $1")).
		WithArgs("House").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))

	mock.ExpectQuery(`fingerprint_hash, file_path, title, artist, album, year, genre_id, duration, bitrate, file_size, last_modified`).
		WithArgs("hash", "hash.flac", "Title", "Artist", "Album", 2020, 5, int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

	meta := TrackMeta{Title: "Title", Artist: "Artist", Album: "Album", Year: 2020, Genre: "House"}
	id, err := saveTidalSong(context.Background(), db, "hash", "hash.flac", meta, 123)
	assert.NoError(t, err)
	assert.Equal(t, 7, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveTidalSong_WithoutGenre(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// No expectation for a genres upsert: with an empty genre the code must
	// not create a genres row. The songs INSERT binds a NULL genre_id.
	mock.ExpectQuery(`fingerprint_hash, file_path, title, artist, album, year, genre_id, duration, bitrate, file_size, last_modified`).
		WithArgs("hash", "hash.flac", "Title", "Artist", "Album", 2020, sql.NullInt64{}, int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

	meta := TrackMeta{Title: "Title", Artist: "Artist", Album: "Album", Year: 2020}
	id, err := saveTidalSong(context.Background(), db, "hash", "hash.flac", meta, 123)
	assert.NoError(t, err)
	assert.Equal(t, 7, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddToPlaylist_SetsGenreOnNewPlaylist(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	now := time.Now()

	// GetOrCreatePlaylist: not found yet.
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, user_id, name, created_at, updated_at, sort_key FROM playlists WHERE user_id = $1 AND name = $2")).
		WithArgs(1, "Deep House Mix").
		WillReturnError(sql.ErrNoRows)

	// CreatePlaylist.
	mock.ExpectQuery(regexp.QuoteMeta(
		"INSERT INTO playlists (user_id, name) VALUES ($1, $2)")).
		WithArgs(1, "Deep House Mix").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "created_at", "updated_at", "sort_key"}).
			AddRow(9, 1, "Deep House Mix", now, now, "added_at"))

	// Name → genre catalogue match.
	mock.ExpectQuery(trigramFilterPattern).
		WithArgs("Deep House Mix").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Deep House"))

	// Set the playlist genre only if currently empty.
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE playlists SET genre = $1, updated_at = NOW() WHERE id = $2 AND (genre IS NULL OR genre = '')")).
		WithArgs("Deep House", 9).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// AddSongToPlaylist transaction.
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2")).
		WithArgs(9, 42).
		WillReturnResult(sqlmock.NewResult(1, 0))
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT COALESCE(MAX(position), 0) + 1 FROM playlist_songs WHERE playlist_id = $1")).
		WithArgs(9).
		WillReturnRows(sqlmock.NewRows([]string{"next"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO playlist_songs (playlist_id, song_id, position, added_at) VALUES ($1, $2, $3, NOW())")).
		WithArgs(9, 42, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	im := newTestImporter(db)
	err = im.AddToPlaylist(context.Background(), 1, 42, "Deep House Mix")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---- E4: object tags ----

type tagCaptureStorage struct {
	tagged map[string]map[string]string
	tagErr error
}

func (s *tagCaptureStorage) UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	return nil
}
func (s *tagCaptureStorage) GetFileStream(ctx context.Context, objectName string) (storage.ReadSeekCloser, error) {
	return nil, nil
}
func (s *tagCaptureStorage) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	return nil, nil
}
func (s *tagCaptureStorage) DeleteObject(ctx context.Context, objectName string) error { return nil }
func (s *tagCaptureStorage) TagObject(ctx context.Context, objectName string, tags map[string]string) error {
	if s.tagged == nil {
		s.tagged = map[string]map[string]string{}
	}
	s.tagged[objectName] = tags
	return s.tagErr
}
func (s *tagCaptureStorage) GetObjectTags(ctx context.Context, objectName string) (map[string]string, error) {
	return s.tagged[objectName], nil
}

func TestImporter_TagObject_RecordsGenreAndArtist(t *testing.T) {
	st := &tagCaptureStorage{}
	im := &Importer{Storage: st}

	im.tagObject(context.Background(), "hash.mp3", &TrackMeta{Title: "T", Artist: "Artist X", Genre: "Deep House"})

	assert.Equal(t,
		map[string]string{"genre": "Deep House", "artist": "Artist X"},
		st.tagged["hash.mp3"])
}

func TestImporter_TagObject_FailureNotFatal(t *testing.T) {
	st := &tagCaptureStorage{tagErr: errors.New("minio down")}
	im := &Importer{Storage: st}

	// Must log and return, not panic or propagate.
	im.tagObject(context.Background(), "hash.mp3", &TrackMeta{Artist: "A"})
}

func TestImporter_TagObject_EmptyValuesOmitted(t *testing.T) {
	st := &tagCaptureStorage{}
	im := &Importer{Storage: st}

	im.tagObject(context.Background(), "hash.mp3", &TrackMeta{Artist: "Only Artist"})

	assert.Equal(t, map[string]string{"artist": "Only Artist"}, st.tagged["hash.mp3"])
}
