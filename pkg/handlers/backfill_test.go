package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// resetBackfillStore clears the package-level job store so each test starts
// with no running or leftover jobs (the store is single-flight by design).
func resetBackfillStore(t *testing.T) {
	t.Helper()
	handlers.ResetBackfillJobs()
	t.Cleanup(handlers.ResetBackfillJobs)
}

// TestStartBackfillHandler_NoMissingSongs asserts the admin backfill endpoint
// returns 202 with a jobId when there are no songs missing embeddings.
func TestStartBackfillHandler_NoMissingSongs(t *testing.T) {
	resetBackfillStore(t)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT s.id FROM songs s`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	cfg := &config.Config{JWTSecret: "default-secret"}
	h := handlers.NewBackfillHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})

	token, _ := auth.GenerateJWT(1, "default-secret")
	req, _ := http.NewRequest("POST", "/api/admin/embeddings/backfill", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.StartBackfillHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)

	var resp map[string]string
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["jobId"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestStartBackfillHandler_Unauthorized asserts the backfill endpoint rejects
// requests without a valid token.
func TestStartBackfillHandler_Unauthorized(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{JWTSecret: "default-secret"}
	h := handlers.NewBackfillHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})

	req, _ := http.NewRequest("POST", "/api/admin/embeddings/backfill", nil)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.StartBackfillHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// TestBackfillStatusHandler_NotFound asserts the status endpoint returns 404
// for an unknown job id.
func TestBackfillStatusHandler_NotFound(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{JWTSecret: "default-secret"}
	h := handlers.NewBackfillHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})

	token, _ := auth.GenerateJWT(1, "default-secret")
	req, _ := http.NewRequest("GET", "/api/admin/embeddings/backfill/status?jobId=nope", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.BackfillStatusHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestBackfill_AdminGate verifies the backfill routes enforce the admin check
// (the router wraps them in middleware.AdminOnly). A non-admin gets 403; an
// admin reaches the handler.
//
// Objective: the admin gate protects the backfill endpoints.
// Given-When-Then: Given a JWT for user 1 and an is_admin row, When a
// non-admin POSTs the backfill route, Then 403 without any embedding query;
// an admin proceeds to 202.
// Edge cases: the gate is the middleware itself (same wiring as router.go).
func TestBackfill_AdminGate(t *testing.T) {
	tests := []struct {
		name     string
		isAdmin  bool
		wantCode int
	}{
		{"non-admin forbidden", false, http.StatusForbidden},
		{"admin allowed", true, http.StatusAccepted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBackfillStore(t)
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock: %v", err)
			}
			defer db.Close()

			mock.ExpectQuery(`SELECT COALESCE\(is_admin, FALSE\) FROM users WHERE id = \$1`).
				WithArgs(1).
				WillReturnRows(sqlmock.NewRows([]string{"is_admin"}).AddRow(tt.isAdmin))
			if tt.isAdmin {
				mock.ExpectQuery(`SELECT s.id FROM songs s`).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
			}

			cfg := &config.Config{JWTSecret: "default-secret"}
			h := handlers.NewBackfillHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})

			token, _ := auth.GenerateJWT(1, "default-secret")
			req, _ := http.NewRequest("POST", "/api/admin/embeddings/backfill", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rr := httptest.NewRecorder()

			handler := middleware.AdminOnly(db)(http.HandlerFunc(h.StartBackfillHandler))
			middleware.Authenticator("default-secret")(handler).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantCode, rr.Code)
			if !tt.isAdmin {
				assert.Contains(t, rr.Body.String(), "Admin access required")
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestBackfill_JobAccounting drives a real job over two songs and asserts the
// per-song Done/Failed counters and the terminal status — the worker goroutine
// in StartBackfillHandler was previously untested.
//
// Objective: one processed song increments Done, one failure increments Failed,
// and the job ends "done" with Total == 2.
// Given-When-Then: Given an audio processor that 500s on song 2's file, When
// the job runs, Then the status endpoint reports Done=1, Failed=1, Status=done.
// Edge cases: the polling loop has a bounded deadline; a stuck job fails the
// test rather than hanging it.
func TestBackfill_JobAccounting(t *testing.T) {
	resetBackfillStore(t)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT s.id FROM songs s`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))
	// Each song looks up its file path; the streaming read uses fakeStreamStorage.
	mock.ExpectQuery(`SELECT file_path FROM songs WHERE id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).AddRow("songs/1.mp3"))
	// Song 1's valid embedding is saved.
	mock.ExpectExec(`INSERT INTO song_embeddings`).
		WithArgs(1, "["+strings.TrimSuffix(strings.Repeat("0.5,", 38), ",")+"]").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT file_path FROM songs WHERE id = \$1`).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).AddRow("songs/2.mp3"))

	// Audio processor: the backfill worker processes songs sequentially (id 1
	// then id 2), so fail the second call to exercise Done/Failed accounting.
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := r.FormFile("file"); err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		if calls.Add(1) == 2 {
			http.Error(w, "essentia exploded", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"embedding": [`+strings.TrimSuffix(strings.Repeat("0.5,", 38), ",")+`]}`)
	}))
	defer server.Close()

	cfg := &config.Config{JWTSecret: "default-secret", AudioProcessorURL: server.URL}
	h := handlers.NewBackfillHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})

	token, _ := auth.GenerateJWT(1, "default-secret")
	req, _ := http.NewRequest("POST", "/api/admin/embeddings/backfill", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.StartBackfillHandler)).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusAccepted, rr.Code)

	var start map[string]string
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &start))
	jobID := start["jobId"]
	assert.NotEmpty(t, jobID)

	// Poll the status endpoint until the worker finishes (bounded).
	deadline := time.Now().Add(5 * time.Second)
	var snap handlers.BackfillJob
	for time.Now().Before(deadline) {
		req, _ = http.NewRequest("GET", "/api/admin/embeddings/backfill/status?jobId="+jobID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr = httptest.NewRecorder()
		middleware.Authenticator("default-secret")(http.HandlerFunc(h.BackfillStatusHandler)).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &snap))
		if snap.Status != "running" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Song 1 succeeds, song 2 fails at the processor.
	assert.Equal(t, "done", snap.Status)
	assert.Equal(t, 2, snap.Total)
	assert.Equal(t, 1, snap.Done)
	assert.Equal(t, 1, snap.Failed)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestBackfill_StatusHappyPath asserts the status endpoint returns the job
// snapshot (200 JSON) for a known job id — previously only the 404 path was
// tested.
func TestBackfill_StatusHappyPath(t *testing.T) {
	resetBackfillStore(t)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// No missing songs → the job finishes immediately with Total 0.
	mock.ExpectQuery(`SELECT s.id FROM songs s`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	cfg := &config.Config{JWTSecret: "default-secret"}
	h := handlers.NewBackfillHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})

	token, _ := auth.GenerateJWT(1, "default-secret")
	req, _ := http.NewRequest("POST", "/api/admin/embeddings/backfill", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.StartBackfillHandler)).ServeHTTP(rr, req)
	require202 := assert.Equal(t, http.StatusAccepted, rr.Code)
	if !require202 {
		t.FailNow()
	}

	var start map[string]string
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &start))

	deadline := time.Now().Add(5 * time.Second)
	var snap handlers.BackfillJob
	for time.Now().Before(deadline) {
		req, _ = http.NewRequest("GET", "/api/admin/embeddings/backfill/status?jobId="+start["jobId"], nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr = httptest.NewRecorder()
		middleware.Authenticator("default-secret")(http.HandlerFunc(h.BackfillStatusHandler)).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &snap))
		if snap.Status != "running" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	assert.Equal(t, "done", snap.Status)
	assert.Equal(t, 0, snap.Total)
	assert.Equal(t, start["jobId"], snap.ID)
}

// TestBackfill_SingleFlight asserts a second POST while a job is running is
// rejected with 409 — two concurrent backfills would double-process the
// library.
func TestBackfill_SingleFlight(t *testing.T) {
	resetBackfillStore(t)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Three songs missing embeddings, so the worker has real work while the
	// second request arrives.
	mock.ExpectQuery(`SELECT s.id FROM songs s`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2).AddRow(3))
	for i := 1; i <= 3; i++ {
		mock.ExpectQuery(`SELECT file_path FROM songs WHERE id = \$1`).
			WithArgs(i).
			WillReturnRows(sqlmock.NewRows([]string{"file_path"}).AddRow(fmt.Sprintf("songs/%d.mp3", i)))
	}

	// A processor that blocks on release keeps the first job deterministically
	// "running" while the second POST arrives.
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		http.Error(w, "done-signal", http.StatusInternalServerError)
	}))
	defer server.Close()
	defer close(release)
	defer handlers.CancelBackfill()

	cfg := &config.Config{JWTSecret: "default-secret", AudioProcessorURL: server.URL}
	h := handlers.NewBackfillHandler(db, cfg, &fakeStreamStorage{data: []byte("ID3fakeaudio")})

	token, _ := auth.GenerateJWT(1, "default-secret")
	first, _ := http.NewRequest("POST", "/api/admin/embeddings/backfill", nil)
	first.Header.Set("Authorization", "Bearer "+token)
	rrFirst := httptest.NewRecorder()
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.StartBackfillHandler)).ServeHTTP(rrFirst, first)
	assert.Equal(t, http.StatusAccepted, rrFirst.Code)

	var start map[string]string
	assert.NoError(t, json.Unmarshal(rrFirst.Body.Bytes(), &start))
	assert.NotEmpty(t, start["jobId"])

	second, _ := http.NewRequest("POST", "/api/admin/embeddings/backfill", nil)
	second.Header.Set("Authorization", "Bearer "+token)
	rrSecond := httptest.NewRecorder()
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.StartBackfillHandler)).ServeHTTP(rrSecond, second)

	assert.Equal(t, http.StatusConflict, rrSecond.Code)
	assert.Contains(t, rrSecond.Body.String(), "already running")
}
