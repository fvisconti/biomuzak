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
	"go-postgres-example/pkg/router"
	"go-postgres-example/pkg/storage"
	"go-postgres-example/pkg/subsonic"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetLibraryHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Mock the database query
	rows := sqlmock.NewRows([]string{"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year", "genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "rating"}).
		AddRow(1, "hash1", "path1", "title1", "artist1", "album1", 2021, 1, "genre1", 180, 320, 12345, time.Now(), 5)

	mock.ExpectQuery("^SELECT (.+) FROM songs s").
		WithArgs(1).
		WillReturnRows(rows)

	// Create a new router
	cfg := &config.Config{JWTSecret: "default-secret"}
	authHandler := handlers.NewAuthHandler(db, cfg, nil)
	uploadHandler := handlers.NewUploadHandler(db, cfg, nil)
	libraryHandler := handlers.NewLibraryHandler(db, cfg, nil)
	playlistHandler := handlers.NewPlaylistHandler(db, cfg, nil)
	songHandler := handlers.NewSongHandler(db, cfg, nil)
	subsonicHandler := subsonic.NewHandler(db, cfg, nil)
	streamHandler := handlers.NewStreamHandler(db, cfg, nil)
	r := router.New(authHandler, uploadHandler, libraryHandler, playlistHandler, songHandler, streamHandler, subsonicHandler, nil, nil, nil, nil, "")

	// Create a new request
	token, _ := auth.GenerateJWT(1, "default-secret")
	req, _ := http.NewRequest("GET", "/api/library", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateSongHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Mock the database query
	mock.ExpectExec("^INSERT INTO user_songs").
		WithArgs(1, 1, 5).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Create a new router
	cfg := &config.Config{JWTSecret: "default-secret"}
	authHandler := handlers.NewAuthHandler(db, cfg, nil)
	uploadHandler := handlers.NewUploadHandler(db, cfg, nil)
	libraryHandler := handlers.NewLibraryHandler(db, cfg, nil)
	playlistHandler := handlers.NewPlaylistHandler(db, cfg, nil)
	songHandler := handlers.NewSongHandler(db, cfg, nil)
	subsonicHandler := subsonic.NewHandler(db, cfg, nil)
	streamHandler := handlers.NewStreamHandler(db, cfg, nil)
	r := router.New(authHandler, uploadHandler, libraryHandler, playlistHandler, songHandler, streamHandler, subsonicHandler, nil, nil, nil, nil, "")

	// Create a new request
	token, _ := auth.GenerateJWT(1, "default-secret")
	body := `{"rating": 5}`
	req, _ := http.NewRequest("POST", "/api/songs/1/rate", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(rr, req)

	// Check the response
	assert.Equal(t, http.StatusOK, rr.Code)
}

// ---- E4: genre change keeps the MinIO object tag in sync ----

type fakeGenreTagStorage struct {
	tagged   map[string]map[string]string
	existing map[string]map[string]string
}

func (s *fakeGenreTagStorage) UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	return nil
}
func (s *fakeGenreTagStorage) GetFileStream(ctx context.Context, objectName string) (storage.ReadSeekCloser, error) {
	return nil, nil
}
func (s *fakeGenreTagStorage) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	return nil, nil
}
func (s *fakeGenreTagStorage) DeleteObject(ctx context.Context, objectName string) error { return nil }
func (s *fakeGenreTagStorage) TagObject(ctx context.Context, objectName string, tags map[string]string) error {
	if s.tagged == nil {
		s.tagged = map[string]map[string]string{}
	}
	s.tagged[objectName] = tags
	return nil
}
func (s *fakeGenreTagStorage) GetObjectTags(ctx context.Context, objectName string) (map[string]string, error) {
	if s.existing == nil {
		return nil, nil
	}
	return s.existing[objectName], nil
}

func TestUpdateSongGenreHandler_RetagsObject(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO genres \(name\) VALUES \(\$1\) ON CONFLICT \(name\) DO NOTHING RETURNING id`).
		WithArgs("Funky").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	mock.ExpectExec(`UPDATE songs SET genre_id = \$1 WHERE id = \$2`).
		WithArgs(9, 5).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT file_path FROM songs WHERE id = \$1`).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).AddRow("hash5.flac"))

	st := &fakeGenreTagStorage{existing: map[string]map[string]string{"hash5.flac": {"artist": "X"}}}
	cfg := &config.Config{JWTSecret: "default-secret"}
	lh := handlers.NewLibraryHandler(db, cfg, st)

	// Route through a chi mux so the {songID} URL param is populated.
	m := chi.NewMux()
	m.Put("/api/songs/{songID}/genre", lh.UpdateSongGenreHandler)

	body := `{"genre":"Funky"}`
	token, _ := auth.GenerateJWT(1, "default-secret")
	req, _ := http.NewRequest("PUT", "/api/songs/5/genre", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(m).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// The pre-existing artist tag is preserved; genre is replaced.
	assert.Equal(t, map[string]string{"artist": "X", "genre": "Funky"}, st.tagged["hash5.flac"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
