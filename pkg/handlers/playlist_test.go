package handlers_test

// Tests for playlist sorting in the handlers (E6).
//
// Objective: GET /api/playlists/{id} honours a ?sort_by= query param
// (whitelisted server-side) and exposes the playlist's stored sort_key;
// PUT /api/playlists/{id} persists sort_key (whitelisted, 400 when invalid)
// and still supports name-only updates.
//
// Given/When/Then: Given a JWT for the owning user, When the playlist is
// fetched with sort_by=title, Then the song query orders by title; When the
// playlist is fetched plain, Then the response carries sort_key; When the
// sort key is updated with an invalid value, Then 400.
//
// Edge cases: invalid sort key, name-only update regression.

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func playlistDetailRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	token, _ := auth.GenerateJWT(1, "default-secret")
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, target, bytes.NewReader([]byte(body)))
	} else {
		req, err = http.NewRequest(method, target, nil)
	}
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func playlistIDRequest(req *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("playlistID", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// expectPlaylistOwnership mocks the GetPlaylistByID ownership lookup
// (including the new sort_key column).
func expectPlaylistOwnership(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, user_id, name, created_at, updated_at, sort_key FROM playlists WHERE id = $1 AND user_id = $2")).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "created_at", "updated_at", "sort_key"}).
			AddRow(2, 1, "Mix", time.Now(), time.Now(), "added_at"))
}

func TestGetPlaylist_PassesSortByParam(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	expectPlaylistOwnership(mock)
	mock.ExpectQuery(`ORDER BY s\.title COLLATE "C"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
			"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "position", "added_at"}))

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "GET", "/api/playlists/2?sort_by=title", ""), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.GetPlaylistHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPlaylist_UnknownSortFallsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	expectPlaylistOwnership(mock)
	mock.ExpectQuery(`ORDER BY ps\.added_at DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
			"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "position", "added_at"}))

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "GET", "/api/playlists/2?sort_by=bogus", ""), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.GetPlaylistHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPlaylist_RespondsWithSortKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	expectPlaylistOwnership(mock)
	mock.ExpectQuery(`ORDER BY ps\.added_at DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
			"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "position", "added_at"}))

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "GET", "/api/playlists/2", ""), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.GetPlaylistHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"sort_key":"added_at"`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPlaylist_UsesPersistedSortKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// No ?sort_by= param: the playlist's persisted sort_key (artist) decides.
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, user_id, name, created_at, updated_at, sort_key FROM playlists WHERE id = $1 AND user_id = $2")).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "created_at", "updated_at", "sort_key"}).
			AddRow(2, 1, "Mix", time.Now(), time.Now(), "artist"))
	mock.ExpectQuery(`ORDER BY s\.artist COLLATE "C"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
			"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "position", "added_at"}))

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "GET", "/api/playlists/2", ""), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.GetPlaylistHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePlaylist_SortKeyOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE playlists SET name = COALESCE($3, name), sort_key = COALESCE($4, sort_key), updated_at = NOW() WHERE id = $1 AND user_id = $2")).
		WithArgs(2, 1, nil, "title").
		WillReturnResult(sqlmock.NewResult(1, 1))

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "PUT", "/api/playlists/2", `{"sort_key":"title"}`), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.UpdatePlaylistHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePlaylist_SortKeyInvalid(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// No DB expectations: the whitelist check rejects before any query.

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "PUT", "/api/playlists/2", `{"sort_key":"bogus"}`), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.UpdatePlaylistHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdatePlaylist_NameOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE playlists SET name = COALESCE($3, name), sort_key = COALESCE($4, sort_key), updated_at = NOW() WHERE id = $1 AND user_id = $2")).
		WithArgs(2, 1, "New Name", nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "PUT", "/api/playlists/2", `{"name":"New Name"}`), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.UpdatePlaylistHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReorderPlaylistOrder_OK(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	expectPlaylistOwnership(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1")).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	for pos, songID := range []int{9, 7} {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE playlist_songs SET position = $1 WHERE playlist_id = $2 AND song_id = $3")).
			WithArgs(pos+1, 2, songID).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "PUT", "/api/playlists/2/order", `{"song_ids":[9,7]}`), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.ReorderPlaylistOrderHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReorderPlaylistOrder_Mismatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	expectPlaylistOwnership(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1")).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectRollback()

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "PUT", "/api/playlists/2/order", `{"song_ids":[9,7]}`), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.ReorderPlaylistOrderHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReorderPlaylistOrder_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Ownership lookup finds nothing: 404 before any reorder work.
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, user_id, name, created_at, updated_at, sort_key FROM playlists WHERE id = $1 AND user_id = $2")).
		WithArgs(2, 1).
		WillReturnError(sql.ErrNoRows)

	h := handlers.NewPlaylistHandler(db, &config.Config{JWTSecret: "default-secret"}, nil)
	req := playlistIDRequest(playlistDetailRequest(t, "PUT", "/api/playlists/2/order", `{"song_ids":[9,7]}`), "2")
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.ReorderPlaylistOrderHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
