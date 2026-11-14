package db

// Tests for the genre catalogue repository (E3c).
//
// Objective: admins can list (with song counts), create, and delete genres.
//
// Given/When/Then:
//   - ListGenres: given a catalogue, returns every genre with the number of
//     songs referencing it (genres with no songs show 0).
//   - CreateGenre: given a new name, the genre is inserted and returned; a
//     duplicate name yields sql.ErrNoRows (from the ON CONFLICT DO NOTHING
//     upsert), which the handler maps to 409.
//   - DeleteGenre: given a known id, the row is deleted (songs keep their
//     data via the FK ON DELETE SET NULL); an unknown id yields sql.ErrNoRows.
//
// Edge cases: empty catalogue, duplicate create, unknown delete.

import (
	"database/sql"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestListGenres_WithSongCounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name", "song_count"}).
		AddRow(1, "Deep House", 12).
		AddRow(2, "Folk", 0)

	mock.ExpectQuery(regexp.QuoteMeta(
		"LEFT JOIN songs s ON s.genre_id = g.id")).
		WillReturnRows(rows)

	genres, err := ListGenres(db)
	assert.NoError(t, err)
	assert.Len(t, genres, 2)
	assert.Equal(t, "Deep House", genres[0].Name)
	assert.Equal(t, 12, genres[0].SongCount)
	assert.Equal(t, 0, genres[1].SongCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateGenre_InsertsAndReturns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		"INSERT INTO genres (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id, name")).
		WithArgs("Chillwave").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(7, "Chillwave"))

	g, err := CreateGenre(db, "Chillwave")
	assert.NoError(t, err)
	assert.Equal(t, 7, g.ID)
	assert.Equal(t, "Chillwave", g.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateGenre_DuplicateReturnsNoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		"INSERT INTO genres (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id, name")).
		WithArgs("House").
		WillReturnError(sql.ErrNoRows)

	_, err = CreateGenre(db, "House")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteGenre_RemovesRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM genres WHERE id = $1")).
		WithArgs(3).
		WillReturnResult(sqlmock.NewResult(1, 1))

	assert.NoError(t, DeleteGenre(db, 3))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteGenre_UnknownIDReturnsNoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM genres WHERE id = $1")).
		WithArgs(99).
		WillReturnResult(sqlmock.NewResult(1, 0))

	assert.ErrorIs(t, DeleteGenre(db, 99), sql.ErrNoRows)
	assert.NoError(t, mock.ExpectationsWereMet())
}
