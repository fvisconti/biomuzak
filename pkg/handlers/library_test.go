package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/router"
	"go-postgres-example/pkg/subsonic"

	"github.com/stretchr/testify/assert"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
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
	authHandler := handlers.NewAuthHandler(db, cfg)
	uploadHandler := handlers.NewUploadHandler(db, cfg)
	libraryHandler := handlers.NewLibraryHandler(db, cfg)
	playlistHandler := handlers.NewPlaylistHandler(db, cfg)
	songHandler := handlers.NewSongHandler(db, cfg)
	subsonicHandler := subsonic.NewHandler(db, cfg)
	r := router.New(authHandler, uploadHandler, libraryHandler, playlistHandler, songHandler, subsonicHandler)

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
	authHandler := handlers.NewAuthHandler(db, cfg)
	uploadHandler := handlers.NewUploadHandler(db, cfg)
	libraryHandler := handlers.NewLibraryHandler(db, cfg)
	playlistHandler := handlers.NewPlaylistHandler(db, cfg)
	songHandler := handlers.NewSongHandler(db, cfg)
	subsonicHandler := subsonic.NewHandler(db, cfg)
	r := router.New(authHandler, uploadHandler, libraryHandler, playlistHandler, songHandler, subsonicHandler)

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
