package handlers_test

// Tests for the Tidal import handler (E3b).
//
// Objective: a batch import resolves the genre catalogue match for the source
// (playlist/album) name synchronously, before the background job starts, so
// every imported song inherits the matched genre.
//
// Given/When/Then: given an import request with a playlist name, when the
// handler starts the job, the trigram genre query runs for the source name
// (and not per track); the job itself proceeds (and here fails fast on the
// missing Tidal credentials — irrelevant to this invariant).
//
// Edge cases: no playlist name → no genre query at all.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/storage"
	"go-postgres-example/pkg/tidal"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// waitForJobDone polls the in-memory job store until the job finishes or the
// deadline passes. Returns the job (stable once finished).
func waitForJobDone(t *testing.T, jobID string, deadline time.Duration) *tidal.ImportJob {
	t.Helper()
	deadlineAt := time.Now().Add(deadline)
	for {
		snap, ok := tidal.GetImportJob(jobID)
		if !ok {
			t.Fatalf("job %s not found", jobID)
		}
		if snap.Snapshot().Status != "running" {
			return snap
		}
		if time.Now().After(deadlineAt) {
			t.Fatalf("job %s did not finish within %s", jobID, deadline)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// fakeImportStorage satisfies storage.StorageService; the import in this test
// never reaches storage (the worker fails on the credentials lookup first).
type fakeImportStorage struct{}

func (fakeImportStorage) UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	return nil
}
func (fakeImportStorage) GetFileStream(ctx context.Context, objectName string) (storage.ReadSeekCloser, error) {
	return nil, nil
}
func (fakeImportStorage) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	return nil, nil
}
func (fakeImportStorage) DeleteObject(ctx context.Context, objectName string) error { return nil }

func TestImportHandler_ResolvesGenreFromSourceName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// The handler loads the user's Tidal config first (no row → defaults).
	mock.ExpectQuery(`SELECT config FROM tidal_credentials WHERE user_id = \$1`).
		WithArgs(1).
		WillReturnError(sql.ErrNoRows)

	// Synchronous, before the job starts: one trigram match for the source name.
	mock.ExpectQuery(regexp.QuoteMeta("WHERE name % $1")).
		WithArgs("Deep House Mix").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Deep House"))
	// The background worker then fails fast: user has no Tidal credentials.
	mock.ExpectQuery(`SELECT user_id, COALESCE\(tidal_user_id, 0\)`).
		WithArgs(1).
		WillReturnError(sql.ErrNoRows)

	h := handlers.NewTidalHandler(db, &config.Config{JWTSecret: "default-secret"}, &fakeImportStorage{})

	cfg := tidal.DefaultConfig()
	cfg.Concurrency = 1
	cfg.DelayMinSec = 0
	cfg.DelayMaxSec = 0

	body, _ := json.Marshal(handlers.ImportRequest{
		Tracks:       []tidal.TrackMeta{{TrackID: 11, Title: "Track"}},
		PlaylistName: "Deep House Mix",
		// No human-like download delay in the test.
		Config: &cfg,
	})
	token, _ := auth.GenerateJWT(1, "default-secret")
	req, _ := http.NewRequest("POST", "/api/tidal/import", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.ImportHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)

	var resp map[string]string
	assert.NoError(t, json.NewDecoder(rr.Result().Body).Decode(&resp))
	job := waitForJobDone(t, resp["jobId"], 5*time.Second)
	assert.Equal(t, 1, job.Failed)

	// The genre query ran exactly once, with the source name.
	assert.NoError(t, mock.ExpectationsWereMet())
}
