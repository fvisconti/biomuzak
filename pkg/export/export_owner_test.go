package export

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-postgres-example/pkg/tidal"
)

// passthroughConverter lets sqlmock accept the []int bound by
// GetSongsByIDs (a pgx array) as a driver value.
type passthroughConverter struct{}

func (passthroughConverter) ConvertValue(v any) (driver.Value, error) { return v, nil }

// TestExportSongs_JobRecordsOwner verifies the job created for a user carries
// that user's ID for status authorization, and never serializes it.
//
// Objective: export jobs are owner-scoped (prime directive 1).
// Given-When-Then: Given user 7 starts a one-song export, When the job is
// registered, Then the stored job reports UserID 7, its snapshot carries the
// owner, and marshalled JSON contains no user field.
// Edge cases: the worker fails on the missing storage object — the job still
// finishes deterministically, so the test is race-free.
func TestExportSongs_JobRecordsOwner(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(passthroughConverter{}))
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`FROM songs s`).
		WithArgs([]int{1}).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "artist", "album", "year", "genre", "file_path"}).
			AddRow(1, "Song", "Artist", "Album", 2020, "Rock", "missing.flac"))

	e := NewExporter(db, &mockStorage{files: map[string][]byte{}}, 1)
	cfg := tidal.TidalConfig{ExportPath: t.TempDir()}

	job, err := e.ExportSongs(context.Background(), "job-owner", 7, []int{1}, cfg)
	require.NoError(t, err)
	// UserID is set at construction and never mutated, so reading the live
	// job is safe; Status must always be read through Snapshot (worker-owned).
	assert.Equal(t, 7, job.UserID)

	require.Eventually(t, func() bool {
		snap, ok := GetExportJob("job-owner")
		return ok && snap.Snapshot().Status == "done"
	}, 5*time.Second, 10*time.Millisecond)

	snap, ok := GetExportJob("job-owner")
	require.True(t, ok)
	assert.Equal(t, 7, snap.UserID, "snapshot must carry the owner for status checks")

	b, err := json.Marshal(snap)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "user_id")
	assert.NotContains(t, string(b), "UserID")

	require.NoError(t, mock.ExpectationsWereMet())
}
