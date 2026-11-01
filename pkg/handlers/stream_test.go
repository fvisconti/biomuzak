package handlers_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/storage"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// withSongID sets the chi route context so chi.URLParam(r, "songID") resolves.
func withSongID(r *http.Request, songID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("songID", songID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// fakeStreamStorage is a minimal storage.StorageService that returns a fixed
// byte stream for any object name.
type fakeStreamStorage struct{ data []byte }

func (f *fakeStreamStorage) UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	return nil
}
func (f *fakeStreamStorage) GetFileStream(ctx context.Context, objectName string) (storage.ReadSeekCloser, error) {
	return &bytesReadSeekCloser{r: bytes.NewReader(f.data)}, nil
}
func (f *fakeStreamStorage) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	return nil, nil
}
func (f *fakeStreamStorage) DeleteObject(ctx context.Context, objectName string) error { return nil }

type bytesReadSeekCloser struct{ r *bytes.Reader }

func (b *bytesReadSeekCloser) Read(p []byte) (int, error)         { return b.r.Read(p) }
func (b *bytesReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	return b.r.Seek(offset, whence)
}
func (b *bytesReadSeekCloser) Close() error { return nil }

// TestStreamSongHandler_GlobalCatalog asserts that streaming a song is NOT
// scoped to the requesting user's library. The query must be the unscoped
// "SELECT file_path FROM songs WHERE id = $1" (no user_songs join). This is the
// E1 "no track is owned" change: any authenticated user can stream any track.
func TestStreamSongHandler_GlobalCatalog(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Unscoped lookup: no user_songs join, single arg (song id).
	mock.ExpectQuery(`SELECT file_path FROM songs WHERE id = \$1`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).AddRow("abc123.mp3"))

	cfg := &config.Config{JWTSecret: "default-secret"}
	h := handlers.NewStreamHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})

	token, _ := auth.GenerateJWT(99, "default-secret")
	req, _ := http.NewRequest("GET", "/api/songs/7/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req = withSongID(req, "7")
	rr := httptest.NewRecorder()

	// Run through the auth middleware so the user ID is set in the context.
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.StreamSongHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDownloadSongHandler_GlobalCatalog asserts that downloading a song is NOT
// scoped to the requesting user's library. The query must be the unscoped
// "SELECT id, title, artist, album, file_path FROM songs WHERE id = $1".
func TestDownloadSongHandler_GlobalCatalog(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, title, artist, album, file_path FROM songs WHERE id = \$1`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "artist", "album", "file_path"}).
			AddRow(7, "Title", "Artist", "Album", "abc123.mp3"))

	cfg := &config.Config{JWTSecret: "default-secret"}
	h := handlers.NewStreamHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})

	token, _ := auth.GenerateJWT(99, "default-secret")
	req, _ := http.NewRequest("GET", "/api/songs/7/download", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req = withSongID(req, "7")
	rr := httptest.NewRecorder()

	// Run through the auth middleware so the user ID is set in the context.
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.DownloadSongHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
