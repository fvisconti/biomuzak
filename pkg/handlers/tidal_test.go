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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
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

// trigramFilterPattern matches the trigram genre-match query (same shape as
// the one in pkg/tidal's tests).
var trigramFilterPattern = `name\s+%`

// ---- E5: link-paste import ----

// expectTidalCredentials mocks a valid (unexpired) credentials row.
func expectTidalCredentials(mock sqlmock.Sqlmock, userID int) {
	mock.ExpectQuery(`tidal_credentials WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "tidal_user_id", "tidal_username",
			"access_token", "refresh_token", "access_token_expires_at", "config"}).
			AddRow(userID, 7, "user", "tok", "ignored", time.Now().Add(2*time.Hour),
				`{"quality":"normal","concurrency":1,"delay_min_sec":0,"delay_max_sec":0}`))
}

// newTidalLinkHandler builds a TidalHandler whose Client points at a fake
// Tidal API served by ts.
func newTidalLinkHandler(db *sql.DB, ts *httptest.Server) *handlers.TidalHandler {
	h := handlers.NewTidalHandler(db, &config.Config{JWTSecret: "default-secret"}, fakeImportStorage{})
	h.Client.BaseURL = ts.URL
	return h
}

func linkImportRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	token, _ := auth.GenerateJWT(1, "default-secret")
	req, err := http.NewRequest("POST", "/api/tidal/import/link", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestLinkImportHandler_InvalidURL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	h := handlers.NewTidalHandler(db, &config.Config{JWTSecret: "default-secret"}, fakeImportStorage{})
	rr := httptest.NewRecorder()

	// Rejected before any Tidal or DB lookup.
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.LinkImportHandler)).
		ServeHTTP(rr, linkImportRequest(t, `{"url":"https://example.com/track/1"}`))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkImportHandler_NoCredentials(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`tidal_credentials WHERE user_id = \$1`).
		WithArgs(1).
		WillReturnError(sql.ErrNoRows)

	h := handlers.NewTidalHandler(db, &config.Config{JWTSecret: "default-secret"}, fakeImportStorage{})
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.LinkImportHandler)).
		ServeHTTP(rr, linkImportRequest(t, `{"url":"https://tidal.com/album/42"}`))

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code) // not a bad upstream; user isn't connected
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkImportHandler_AlbumURL_StartsJob(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/albums/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":42,"title":"Test Album","artistName":"X","year":2020}`))
	})
	// Two tracks, one a video: the video must be skipped.
	mux.HandleFunc("/albums/42/tracks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"offset":0,"limit":500,"totalNumItems":2,"items":[
			{"id":1,"title":"T1","artistName":"A","albumName":"Test Album","year":2020},
			{"id":2,"title":"V1","artistName":"A","albumName":"Test Album","year":2020,"isVideo":true}
		]}`))
	})
	mux.HandleFunc("/tracks/1/audio", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound) // job fails here, harmlessly
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Resolve: token for album fetch, token for tracks fetch. The job then
	// loads config, then applies the E3b genre match (no match here).
	expectTidalCredentials(mock, 1)
	expectTidalCredentials(mock, 1)
	mock.ExpectQuery(`SELECT config FROM tidal_credentials WHERE user_id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).
			AddRow(`{"quality":"normal","concurrency":1,"delay_min_sec":0,"delay_max_sec":0}`))
	mock.ExpectQuery(trigramFilterPattern).
		WithArgs("Test Album").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))
	// Job worker: token, then the stream URL fails.
	expectTidalCredentials(mock, 1)

	h := newTidalLinkHandler(db, ts)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.LinkImportHandler)).
		ServeHTTP(rr, linkImportRequest(t, `{"url":"https://tidal.com/album/42"}`))

	assert.Equal(t, http.StatusAccepted, rr.Code)

	var resp map[string]string
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	if jobID, ok := resp["jobId"]; ok {
		job := waitForJobDone(t, jobID, 10*time.Second).Snapshot()
		assert.Equal(t, "done", job.Status)
		assert.Equal(t, 1, job.Total) // video skipped
		assert.Equal(t, 1, job.Failed)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkImportHandler_TrackURL_StartsJob(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/tracks/7", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":7,"title":"Solo","artistName":"B","albumName":"Al","year":2021}`))
	})
	mux.HandleFunc("/tracks/7/audio", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Resolve: token for the track fetch. No genre match (no source name).
	expectTidalCredentials(mock, 1)
	mock.ExpectQuery(`SELECT config FROM tidal_credentials WHERE user_id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).
			AddRow(`{"quality":"normal","concurrency":1,"delay_min_sec":0,"delay_max_sec":0}`))
	// Job worker.
	expectTidalCredentials(mock, 1)

	h := newTidalLinkHandler(db, ts)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.LinkImportHandler)).
		ServeHTTP(rr, linkImportRequest(t, `{"url":"https://tidal.com/track/7"}`))

	assert.Equal(t, http.StatusAccepted, rr.Code)
	var resp map[string]string
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	if jobID, ok := resp["jobId"]; ok {
		job := waitForJobDone(t, jobID, 10*time.Second).Snapshot()
		assert.Equal(t, "done", job.Status)
		assert.Equal(t, 1, job.Total)
		assert.Equal(t, 1, job.Failed)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

// F1 regression: a DB/config failure must not leak wrapped driver details
// (connection strings) into the response body.
func TestImportHandler_ConfigErrorBodySanitized(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT config FROM tidal_credentials WHERE user_id = \$1`).
		WithArgs(1).
		WillReturnError(errors.New(`pgx: conn closed (host=db.internal:5432 user=biomuzak)`))

	h := handlers.NewTidalHandler(db, &config.Config{JWTSecret: "default-secret"}, fakeImportStorage{})
	rr := httptest.NewRecorder()
	body := `{"tracks":[{"track_id":1,"title":"T","artist":"A"}]}`
	token, _ := auth.GenerateJWT(1, "default-secret")
	req, _ := http.NewRequest("POST", "/api/tidal/import", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+token)

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.ImportHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "Failed to start import", strings.TrimSpace(rr.Body.String()))
	assert.NotContains(t, rr.Body.String(), "db.internal")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkImportHandler_PlaylistURL_StartsJob(t *testing.T) {
	const plID = "pldeadbeefcafebabe12345678"
	mux := http.NewServeMux()
	mux.HandleFunc("/playlists/"+plID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":0,"title":"Deep House Mix"}`))
	})
	mux.HandleFunc("/playlists/"+plID+"/tracks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"offset":0,"limit":500,"totalNumItems":2,"items":[
			{"id":11,"title":"P1","artistName":"C","year":2022},
			{"id":12,"title":"V2","artistName":"C","year":2022,"isVideo":true}
		]}`))
	})
	mux.HandleFunc("/tracks/11/audio", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Resolve via the string-ID playlist lookups, then config + E3b genre
	// match against the playlist title.
	expectTidalCredentials(mock, 1)
	expectTidalCredentials(mock, 1)
	mock.ExpectQuery(`SELECT config FROM tidal_credentials WHERE user_id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).
			AddRow(`{"quality":"normal","concurrency":1,"delay_min_sec":0,"delay_max_sec":0}`))
	mock.ExpectQuery(trigramFilterPattern).
		WithArgs("Deep House Mix").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))
	// Job worker.
	expectTidalCredentials(mock, 1)

	h := newTidalLinkHandler(db, ts)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.LinkImportHandler)).
		ServeHTTP(rr, linkImportRequest(t, `{"url":"https://tidal.com/playlist/`+plID+`"}`))

	assert.Equal(t, http.StatusAccepted, rr.Code)
	var resp map[string]string
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	if jobID, ok := resp["jobId"]; ok {
		job := waitForJobDone(t, jobID, 10*time.Second).Snapshot()
		assert.Equal(t, "done", job.Status)
		assert.Equal(t, 1, job.Total) // video skipped
		assert.Equal(t, 1, job.Failed)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}
