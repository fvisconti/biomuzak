package subsonic

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/models"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// vectorText768 builds a 768-dim vector literal for sqlmock rows.
func vectorText768() string {
	parts := make([]string, 768)
	for i := range parts {
		parts[i] = "0.1"
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func songRows(ids ...int) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"id", "title", "artist", "album", "year", "genre", "duration", "bitrate", "file_size", "last_modified"})
	for _, id := range ids {
		rows.AddRow(id, "Song "+strconv.Itoa(id), "Artist X", "Album Y", 2021, "House", 210, 320, 9999,
			time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC))
	}
	return rows
}

// withUserID injects the authenticated user id the way AuthMiddleware does.
func withUserID(req *http.Request, id int) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), userIDKey, id))
}

func TestAuthMiddleware_TokenAuth(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{JWTSecret: "test-secret-123"}
	token := DeriveSubsonicToken(cfg.JWTSecret, "alice")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("authenticated"))
	})
	wrapped := AuthMiddleware(db, cfg)(handler)

	// --- valid token ---
	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE username = \$1`).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(7, "$2a$10$hash"))
	req := httptest.NewRequest("GET", "/rest/ping.view?u=alice&t="+token, nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	assert.Equal(t, "authenticated", rr.Body.String())

	// --- wrong token ---
	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE username = \$1`).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(7, "$2a$10$hash"))
	req = httptest.NewRequest("GET", "/rest/ping.view?u=alice&t=deadbeef", nil)
	rr = httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	assert.Contains(t, rr.Body.String(), `code="40"`)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestToSong_EnrichedFields(t *testing.T) {
	s := modelsSong(3)
	req := httptest.NewRequest("GET", "http://music.local/rest/stream.view", nil)
	got := toSong(s, req)

	assert.Equal(t, "3", got.ID)
	assert.Equal(t, "Stream Song", got.Title)
	assert.Equal(t, 201, got.Duration)
	assert.Equal(t, 320, got.Bitrate)
	assert.Equal(t, int64(12345), got.Size)
	assert.False(t, got.IsVideo)
	assert.False(t, got.IsDir)
	assert.Contains(t, got.StreamURL, "http://music.local/rest/stream.view?id=3")
	assert.Equal(t, got.AlbumID, got.Parent)
	// album id round-trips to the right artist+album
	artist, album, err := parseAlbumID(got.AlbumID)
	require.NoError(t, err)
	assert.Equal(t, "Daft Punk", artist)
	assert.Equal(t, "Discovery", album)
}

func TestGetAlbums(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT DISTINCT artist, album FROM songs`).
		WillReturnRows(sqlmock.NewRows([]string{"artist", "album"}).AddRow("A", "One"))

	h := NewHandler(db, &config.Config{}, nil)
	req := httptest.NewRequest("GET", "/rest/getAlbums.view", nil)
	rr := httptest.NewRecorder()
	h.GetAlbums(rr, req)

	body := rr.Body.String()
	assert.Contains(t, body, `<albumList>`)
	assert.Contains(t, body, `name="One"`)
	assert.Contains(t, body, `artist="A"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlbum(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`WHERE s\.artist = \$1 AND s\.album = \$2 ORDER BY s\.id`).
		WithArgs("A", "One").
		WillReturnRows(songRows(1, 2))

	h := NewHandler(db, &config.Config{}, nil)
	q := url.Values{}
	q.Set("id", makeAlbumID("A", "One"))
	req := httptest.NewRequest("GET", "/rest/getAlbum.view?"+q.Encode(), nil)
	rr := httptest.NewRecorder()
	h.GetAlbum(rr, req)

	body := rr.Body.String()
	assert.Contains(t, body, `status="ok"`)
	assert.Contains(t, body, `songCount="2"`)
	assert.Contains(t, body, `<song id="1"`)
	assert.Contains(t, body, `streamUrl=`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlbum_BadID(t *testing.T) {
	h := NewHandler(nil, &config.Config{}, nil)
	req := httptest.NewRequest("GET", "/rest/getAlbum.view?id=garbage", nil)
	rr := httptest.NewRecorder()
	h.GetAlbum(rr, req)
	assert.Contains(t, rr.Body.String(), `code="70"`)
}

func TestGetArtists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT artist, COUNT\(DISTINCT album\)`).
		WillReturnRows(sqlmock.NewRows([]string{"artist", "n"}).AddRow("A", 4))

	h := NewHandler(db, &config.Config{}, nil)
	req := httptest.NewRequest("GET", "/rest/getArtists.view", nil)
	rr := httptest.NewRecorder()
	h.GetArtists(rr, req)

	assert.Contains(t, rr.Body.String(), `albumCount="4"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetArtist(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT DISTINCT album FROM songs WHERE artist = \$1`).
		WithArgs("A").
		WillReturnRows(sqlmock.NewRows([]string{"album"}).AddRow("One").AddRow("Two"))

	h := NewHandler(db, &config.Config{}, nil)
	q := url.Values{}
	q.Set("id", makeArtistID("A"))
	req := httptest.NewRequest("GET", "/rest/getArtist.view?"+q.Encode(), nil)
	rr := httptest.NewRecorder()
	h.GetArtist(rr, req)

	body := rr.Body.String()
	assert.Contains(t, body, `albumCount="2"`)
	assert.Contains(t, body, `name="One"`)
	assert.Contains(t, body, `name="Two"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChildren_AlbumID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`WHERE s\.artist = \$1 AND s\.album = \$2 ORDER BY s\.id`).
		WithArgs("A", "One").
		WillReturnRows(songRows(5))

	h := NewHandler(db, &config.Config{}, nil)
	q := url.Values{}
	q.Set("id", makeAlbumID("A", "One"))
	req := httptest.NewRequest("GET", "/rest/getChildren.view?"+q.Encode(), nil)
	rr := httptest.NewRecorder()
	h.GetChildren(rr, req)

	assert.Contains(t, rr.Body.String(), `<song id="5"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChildren_SongID_ReturnsAlbum(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`WHERE s\.id = \$1`).WithArgs(9).WillReturnRows(songRows(9))
	mock.ExpectQuery(`WHERE s\.artist = \$1 AND s\.album = \$2 ORDER BY s\.id`).
		WithArgs("Artist X", "Album Y").WillReturnRows(songRows(9, 10))

	h := NewHandler(db, &config.Config{}, nil)
	req := httptest.NewRequest("GET", "/rest/getChildren.view?id=9", nil)
	rr := httptest.NewRecorder()
	h.GetChildren(rr, req)

	assert.Contains(t, rr.Body.String(), `<song id="10"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSimilarSongs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT embedding::text FROM song_embeddings WHERE song_id = \$1`).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"embedding"}).AddRow(vectorText768()))
	mock.ExpectQuery(`ORDER BY se\.embedding <=> \$1`).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year", "genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "similarity"}).
			AddRow(8, "h", "f.mp3", "Song 8", "Artist X", "Album Y", 2021, 7, "House", 210, 320, 9999,
				time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), 0.91))

	h := NewHandler(db, &config.Config{SimilarSongsLimit: 3}, nil)
	req := httptest.NewRequest("GET", "/rest/getSimilarSongs.view?id=3", nil)
	rr := httptest.NewRecorder()
	h.GetSimilarSongs(rr, req)

	assert.Contains(t, rr.Body.String(), `<similarSongs>`)
	assert.Contains(t, rr.Body.String(), `<song id="8"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRandomSongs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`ORDER BY random\(\) LIMIT \$1`).
		WithArgs(5).
		WillReturnRows(songRows(1))

	h := NewHandler(db, &config.Config{}, nil)
	req := httptest.NewRequest("GET", "/rest/getRandomSongs.view?size=5", nil)
	rr := httptest.NewRecorder()
	h.GetRandomSongs(rr, req)

	assert.Contains(t, rr.Body.String(), `<randomSongs>`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChildren_RootFolderReturnsAllAlbums(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT DISTINCT artist, album FROM songs`).
		WillReturnRows(sqlmock.NewRows([]string{"artist", "album"}).AddRow("A", "One"))

	h := NewHandler(db, &config.Config{}, nil)
	req := httptest.NewRequest("GET", "/rest/getChildren.view?id=1", nil)
	rr := httptest.NewRecorder()
	h.GetChildren(rr, req)

	// The root music folder lists all albums — not song 1's album.
	assert.Contains(t, rr.Body.String(), `<albumList>`)
	assert.Contains(t, rr.Body.String(), `name="One"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetArtwork_ReturnsCleanNotFound(t *testing.T) {
	h := NewHandler(nil, &config.Config{}, nil)
	req := httptest.NewRequest("GET", "/rest/getArtwork.view?id=al.x&coverArt=y", nil)
	rr := httptest.NewRecorder()
	h.GetArtwork(rr, req)
	assert.Contains(t, rr.Body.String(), `code="70"`)
}

func TestGetPlaylists_ScopedToUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT p\.id, p\.user_id, p\.name`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "created_at", "updated_at", "sort_key", "song_count"}).
			AddRow(11, 7, "Chill", time.Now(), time.Now(), "added_at", 3))

	h := NewHandler(db, &config.Config{}, nil)
	req := withUserID(httptest.NewRequest("GET", "/rest/getPlaylists.view", nil), 7)
	rr := httptest.NewRecorder()
	h.GetPlaylists(rr, req)

	body := rr.Body.String()
	assert.Contains(t, body, `<playlists>`)
	assert.Contains(t, body, `name="Chill"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPlaylist_ScopedToUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`WHERE id = \$1 AND user_id = \$2`).
		WithArgs(11, 7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "created_at", "updated_at", "sort_key"}).
			AddRow(11, 7, "Chill", time.Now(), time.Now(), "added_at"))
	mock.ExpectQuery(`WHERE ps\.playlist_id = \$1`).
		WithArgs(11).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year", "genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "position", "added_at"}).
			AddRow(1, "h", "f.mp3", "Track 1", "A", "B", 2020, nil, "", 200, 320, 1000, time.Now(), 1, time.Now()))

	h := NewHandler(db, &config.Config{}, nil)
	req := withUserID(httptest.NewRequest("GET", "/rest/getPlaylist.view?id=11", nil), 7)
	rr := httptest.NewRecorder()
	h.GetPlaylist(rr, req)

	body := rr.Body.String()
	assert.Contains(t, body, `name="Chill"`)
	assert.Contains(t, body, `<song id="1"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePlaylist_WithSongs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO playlists`).
		WithArgs(7, "New Mix").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "created_at", "updated_at", "sort_key"}).
			AddRow(20, 7, "New Mix", time.Now(), time.Now(), "added_at"))
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM playlist_songs`).WithArgs(20, 1).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`UPDATE playlist_songs SET position`).WithArgs(20, 1).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO playlist_songs`).WithArgs(20, 1, 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM playlist_songs`).WithArgs(20, 2).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`UPDATE playlist_songs SET position`).WithArgs(20, 2).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO playlist_songs`).WithArgs(20, 2, 2).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	h := NewHandler(db, &config.Config{}, nil)
	req := withUserID(httptest.NewRequest("POST", "/rest/createPlaylist.view?name=New%20Mix&songId=1%2C2", nil), 7)
	rr := httptest.NewRecorder()
	h.CreatePlaylist(rr, req)

	assert.Contains(t, rr.Body.String(), `id="20"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePlaylist_Renames(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`WHERE id = \$1 AND user_id = \$2`).
		WithArgs(11, 7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "created_at", "updated_at", "sort_key"}).
			AddRow(11, 7, "Old", time.Now(), time.Now(), "added_at"))
	mock.ExpectExec(`UPDATE playlists`).
		WithArgs(11, 7, "New Name", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	h := NewHandler(db, &config.Config{}, nil)
	req := withUserID(httptest.NewRequest("GET", "/rest/updatePlaylist.view?id=11&name=New%20Name", nil), 7)
	rr := httptest.NewRecorder()
	h.UpdatePlaylist(rr, req)

	assert.Contains(t, rr.Body.String(), `status="ok"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeletePlaylist_OwnershipChecked(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Ownership check finds nothing (playlist belongs to someone else).
	mock.ExpectQuery(`WHERE id = \$1 AND user_id = \$2`).
		WithArgs(11, 7).
		WillReturnError(sql.ErrNoRows)

	h := NewHandler(db, &config.Config{}, nil)
	req := withUserID(httptest.NewRequest("GET", "/rest/deletePlaylist.view?id=11", nil), 7)
	rr := httptest.NewRecorder()
	h.DeletePlaylist(rr, req)

	assert.Contains(t, rr.Body.String(), `code="70"`)
	// No DELETE may be issued.
	assert.NoError(t, mock.ExpectationsWereMet())
}

// modelsSong is a tiny constructor to keep test setup terse.
func modelsSong(id int) *models.Song {
	return &models.Song{
		ID:           id,
		Title:        "Stream Song",
		Artist:       "Daft Punk",
		Album:        "Discovery",
		Year:         2001,
		Genre:        "Electronic",
		Duration:     201,
		Bitrate:      320,
		FileSize:     12345,
		LastModified: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
	}
}
