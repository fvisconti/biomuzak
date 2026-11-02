package handlers_test

import (
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/export"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// passthroughConverter lets sqlmock accept the []int bound by GetSongsByIDs
// (a pgx array argument) as a driver value.
type exportArgConverter struct{}

func (exportArgConverter) ConvertValue(v any) (driver.Value, error) { return v, nil }

// serveExportStatus runs the status handler behind the auth middleware as
// the given user id.
func serveExportStatus(t *testing.T, h *handlers.ExportHandler, userID int, jobID string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("GET", "/api/export/status?jobId="+jobID, nil)
	require.NoError(t, err)
	token, err := auth.GenerateJWT(userID, "default-secret")
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.StatusHandler)).ServeHTTP(rr, req)
	return rr
}

// TestExportStatus_OwnerScoped asserts export job status is scoped to the
// job's owner, mirroring the Tidal import-status route (prime directive 1:
// per-user boundary — job state belongs to the user who started it).
//
// Objective: a job started by user 1 is unreachable as user 2, even with a
// guessed job id.
// Given-When-Then: Given user 1 starts a one-song export, When user 2 polls
// the returned jobId, Then 403 "Forbidden" with no job data; when user 1
// polls, Then 200 with a snapshot that contains no owner field.
// Edge cases: unknown job ids remain 404 for every user
// (TestExportStatus_UnknownJob).
func TestExportStatus_OwnerScoped(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(exportArgConverter{}))
	require.NoError(t, err)
	defer db.Close()

	exportPath := t.TempDir()
	mock.ExpectQuery(`SELECT config FROM tidal_credentials`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).
			AddRow([]byte(`{"export_path":"` + exportPath + `"}`)))
	// The storage object is missing: the export worker records a failure and
	// the job finishes deterministically (race-free).
	mock.ExpectQuery(`FROM songs s`).
		WithArgs([]int{1}).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "artist", "album", "year", "genre", "file_path"}).
			AddRow(1, "Song", "Artist", "Album", 2020, "Rock", "missing.flac"))

	cfg := &config.Config{JWTSecret: "default-secret"}
	h := handlers.NewExportHandler(db, cfg, &fakeStreamStorage{data: []byte("audio")}, 1)

	// 1) User 1 starts an export and receives the job id.
	startReq, err := http.NewRequest("POST", "/api/export", strings.NewReader(`{"song_ids":[1]}`))
	require.NoError(t, err)
	token1, err := auth.GenerateJWT(1, "default-secret")
	require.NoError(t, err)
	startReq.Header.Set("Authorization", "Bearer "+token1)
	startRR := httptest.NewRecorder()
	middleware.Authenticator("default-secret")(http.HandlerFunc(h.StartHandler)).ServeHTTP(startRR, startReq)
	require.Equal(t, http.StatusAccepted, startRR.Code, "start should be accepted: %s", startRR.Body.String())

	var resp map[string]string
	require.NoError(t, json.Unmarshal(startRR.Body.Bytes(), &resp))
	jobID := resp["jobId"]
	require.NotEmpty(t, jobID)

	// Wait for the bounded worker pool to finish so the job is in a stable
	// state before polling status (avoids a data race on the live job).
	require.Eventually(t, func() bool {
		j, ok := export.GetExportJob(jobID)
		return ok && j.Snapshot().Status == "done"
	}, 5*time.Second, 20*time.Millisecond, "export job did not finish")

	// The store keeps the owner on the job for authorization.
	stored, ok := export.GetExportJob(jobID)
	require.True(t, ok)
	assert.Equal(t, 1, stored.Snapshot().UserID)

	// 2) User 2 guesses the job id and must NOT see user 1's job.
	rr := serveExportStatus(t, h, 2, jobID)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Equal(t, "Forbidden", strings.TrimSpace(rr.Body.String()))

	// 3) The owner still gets the snapshot.
	rr = serveExportStatus(t, h, 1, jobID)
	assert.Equal(t, http.StatusOK, rr.Code)
	var snap map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &snap))
	assert.Equal(t, jobID, snap["id"])
	assert.NotContains(t, snap, "user_id", "owner must never leak into responses")
	assert.NotContains(t, snap, "UserID")

	require.NoError(t, mock.ExpectationsWereMet())
}

// TestExportStatus_UnknownJob asserts an unknown job id is 404 for any user —
// the owner check must not change the not-found path.
func TestExportStatus_UnknownJob(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{JWTSecret: "default-secret"}
	h := handlers.NewExportHandler(db, cfg, &fakeStreamStorage{data: []byte("audio")}, 1)

	rr := serveExportStatus(t, h, 1, "does-not-exist")
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
