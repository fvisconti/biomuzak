package handlers_test

import (
	"bytes"
	"database/sql"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUploadRequest builds an authenticated multipart upload request with one
// fake audio file and the given extra form fields.
func newUploadRequest(t *testing.T, userID int, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	part, err := mw.CreateFormFile("files", "song.mp3")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake audio bytes"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, err := http.NewRequest("POST", "/api/upload", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	token, err := auth.GenerateJWT(userID, "default-secret")
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// serveUpload wraps the handler in the auth middleware, like the router does.
func serveUpload(h *handlers.UploadHandler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.Upload)).ServeHTTP(rr, req)
	return rr
}

func newUploadHandler(db *sql.DB) *handlers.UploadHandler {
	cfg := &config.Config{JWTSecret: "default-secret"}
	return handlers.NewUploadHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})
}

// TestUpload_PlaylistID_ForeignPlaylistRejected asserts that uploading with a
// playlist_id owned by another user fails with 404 and never starts
// processing. Prime directive 1: playlist scope comes from the verified JWT,
// never from a request body field.
//
// Objective: an explicit playlist_id is ownership-checked against the JWT user.
// Given-When-Then: Given user 1 uploads with playlist_id=99, When the owner
// lookup (WHERE id = $1 AND user_id = $2) finds no row, Then the response is
// 404 and the ownership query was executed.
// Edge cases: a DB error yields a sanitized 500; a non-numeric id is rejected
// with 400 before any DB call.
func TestUpload_PlaylistID_ForeignPlaylistRejected(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`WHERE id = \$1 AND user_id = \$2`).
		WithArgs(99, 1).
		WillReturnError(sql.ErrNoRows)

	h := newUploadHandler(db)
	req := newUploadRequest(t, 1, map[string]string{"playlist_id": "99"})
	rr := serveUpload(h, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "Playlist not found", strings.TrimSpace(rr.Body.String()))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpload_PlaylistID_DBErrorSanitized asserts a failed ownership check
// returns a generic 500 without leaking the internal error (prime directive 5).
func TestUpload_PlaylistID_DBErrorSanitized(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`WHERE id = \$1 AND user_id = \$2`).
		WithArgs(99, 1).
		WillReturnError(errors.New("internal connection reset"))

	h := newUploadHandler(db)
	req := newUploadRequest(t, 1, map[string]string{"playlist_id": "99"})
	rr := serveUpload(h, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "Failed to verify playlist", strings.TrimSpace(rr.Body.String()))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpload_PlaylistID_InvalidRejectedBeforeDB asserts a non-numeric
// playlist_id is rejected with 400 without touching the database. The old
// fmt.Sscanf silently swallowed parse errors and treated "abc" as "no
// playlist", which masked client bugs and (combined with the missing
// ownership check) enabled the IDOR.
func TestUpload_PlaylistID_InvalidRejectedBeforeDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := newUploadHandler(db)
	req := newUploadRequest(t, 1, map[string]string{"playlist_id": "not-a-number"})
	rr := serveUpload(h, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "Invalid playlist ID", strings.TrimSpace(rr.Body.String()))
	require.NoError(t, mock.ExpectationsWereMet()) // no query expected
}
