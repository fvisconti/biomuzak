package db

import (
	"os"
	"path/filepath"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// writeMigrationFile creates a migration file inside dir for MigrateAll tests.
func writeMigrationFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

// TestMigrateAll_RenumbersLegacyAppliedVersions guards against re-running a
// migration whose file was renumbered.
//
// Objective: databases that recorded the OLD filename in schema_migrations
// must not execute the renamed file a second time.
// Given-When-Then: Given an applied history containing
// 0004_fix_embedding_dimensions while the repo now ships
// 0005_fix_embedding_dimensions, When MigrateAll runs, Then the history row is
// renumbered in place and the 0005 body (DROP TABLE song_embeddings) is NOT
// executed, while the genuinely new 0002_add_playlist_genre still applies.
// Edge cases: a history that already uses the new names performs no UPDATE at
// all (covered by TestMigrateAll_NoRenameOnCurrentNames); sqlmock is strict,
// so any extra statement fails the test.
func TestMigrateAll_RenumbersLegacyAppliedVersions(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "0001_initial_schema.sql", "CREATE TABLE users (id INTEGER);")
	writeMigrationFile(t, dir, "0002_add_playlist_genre.sql", "ALTER TABLE playlists ADD COLUMN IF NOT EXISTS genre TEXT;")
	writeMigrationFile(t, dir, "0003_enable_pg_trgm.sql", "CREATE EXTENSION IF NOT EXISTS pg_trgm;")
	writeMigrationFile(t, dir, "0004_add_is_admin.sql", "ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN;")
	writeMigrationFile(t, dir, "0005_fix_embedding_dimensions.sql", "DROP TABLE IF EXISTS song_embeddings CASCADE;")

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	// A pre-E2 database: it applied the old 0004_fix_embedding_dimensions, which
	// is now numbered 0005. Re-running it would drop every embedding.
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).
			AddRow("0001_initial_schema").
			AddRow("0002_enable_pg_trgm").
			AddRow("0003_add_is_admin").
			AddRow("0004_fix_embedding_dimensions"))

	// The three renumbered files must be rewritten in the history table.
	mock.ExpectExec(`UPDATE schema_migrations SET version = \$1 WHERE version = \$2`).
		WithArgs("0003_enable_pg_trgm", "0002_enable_pg_trgm").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE schema_migrations SET version = \$1 WHERE version = \$2`).
		WithArgs("0004_add_is_admin", "0003_add_is_admin").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE schema_migrations SET version = \$1 WHERE version = \$2`).
		WithArgs("0005_fix_embedding_dimensions", "0004_fix_embedding_dimensions").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Only the genuinely new migration runs. NOTE: no DROP TABLE expectation —
	// if the rename map is missing, sqlmock reports the unexpected statement.
	mock.ExpectBegin()
	mock.ExpectExec(`ALTER TABLE playlists ADD COLUMN IF NOT EXISTS genre`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO schema_migrations`).
		WithArgs("0002_add_playlist_genre", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, MigrateAll(db, dir))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestMigrateAll_NoRenameOnCurrentNames covers the healthy path: a database
// that already records the current filenames must not be rewritten or re-run.
//
// Objective: renaming is a one-time repair, not a per-boot migration.
// Given-When-Then: Given a history already using 0005_fix_embedding_dimensions
// and 0006_add_tidal_credentials, When MigrateAll runs, Then no UPDATE and no
// migration body executes.
func TestMigrateAll_NoRenameOnCurrentNames(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "0001_initial_schema.sql", "CREATE TABLE users (id INTEGER);")
	writeMigrationFile(t, dir, "0005_fix_embedding_dimensions.sql", "DROP TABLE IF EXISTS song_embeddings CASCADE;")
	writeMigrationFile(t, dir, "0006_add_tidal_credentials.sql", "CREATE TABLE IF NOT EXISTS tidal_credentials (user_id INTEGER);")

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT version FROM schema_migrations`).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).
			AddRow("0001_initial_schema").
			AddRow("0002_add_playlist_genre").
			AddRow("0003_enable_pg_trgm").
			AddRow("0004_add_is_admin").
			AddRow("0005_fix_embedding_dimensions").
			AddRow("0006_add_tidal_credentials"))

	// Strict mock: no further statements are allowed.
	require.NoError(t, MigrateAll(db, dir))
	require.NoError(t, mock.ExpectationsWereMet())
}
