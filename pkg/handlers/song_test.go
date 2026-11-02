package handlers_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// embedding38 is the pgvector text form of a valid 38-dim embedding (the
// repo-wide contract enforced by db.SaveSongEmbedding / db.FindSimilarSongs).
var embedding38 = "[" + strings.TrimSuffix(strings.Repeat("0.1,", 38), ",") + "]"

// TestGetSimilarSongsHandler_UsesConfiguredLimit asserts the similar-songs
// endpoint returns the number of tracks configured via SIMILAR_SONGS_LIMIT
// (default 3), not a hardcoded value. The LIMIT is the third query arg.
func TestGetSimilarSongsHandler_UsesConfiguredLimit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// 1. Embedding lookup.
	mock.ExpectQuery(`SELECT embedding::text FROM song_embeddings WHERE song_id = \$1`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"embedding"}).AddRow(embedding38))

	// 2. Similar songs: the LIMIT ($3) must be the configured value (3).
	mock.ExpectQuery(`ORDER BY se.embedding <=> \$1`).
		WithArgs(embedding38, 7, 3).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
			"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "similarity",
		}).AddRow(8, "h", "p", "T", "A", "Al", 2020, sql.NullInt64{}, "", 100, 320, int64(1), time.Now(), 0.9))

	cfg := &config.Config{JWTSecret: "default-secret", SimilarSongsLimit: 3}
	h := handlers.NewSongHandler(db, cfg, nil)

	req, _ := http.NewRequest("GET", "/api/songs/7/similar", nil)
	req = withSongID(req, "7")
	token, _ := auth.GenerateJWT(1, "default-secret")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.GetSimilarSongsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
