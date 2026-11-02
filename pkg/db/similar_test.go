package db

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// EmbeddingDim is the repo-wide 38-dim contract (13+13+6+6, see
// vchord.instructions.md). Helpers below build valid and invalid vectors.
func embedding38(val float64) []float64 {
	v := make([]float64, EmbeddingDim)
	for i := range v {
		v[i] = val
	}
	return v
}

// embedding38String is the pgvector text form of embedding38(val) — the exact
// string vectorToString produces, so sqlmock WithArgs match it.
func embedding38String(val float64) string {
	s := strconv.FormatFloat(val, 'f', -1, 64)
	return "[" + strings.TrimSuffix(strings.Repeat(s+",", EmbeddingDim), ",") + "]"
}

func TestVectorToString(t *testing.T) {
	tests := []struct {
		name     string
		input    []float64
		expected string
	}{
		{
			name:     "empty vector",
			input:    []float64{},
			expected: "[]",
		},
		{
			name:     "single element",
			input:    []float64{1.5},
			expected: "[1.5]",
		},
		{
			name:     "multiple elements",
			input:    []float64{1.0, 2.5, 3.7},
			expected: "[1,2.5,3.7]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vectorToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringToVector(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []float64
		expectErr bool
	}{
		{
			name:      "empty vector",
			input:     "[]",
			expected:  []float64{},
			expectErr: false,
		},
		{
			name:      "single element",
			input:     "[1.5]",
			expected:  []float64{1.5},
			expectErr: false,
		},
		{
			name:      "multiple elements",
			input:     "[1.0,2.5,3.7]",
			expected:  []float64{1.0, 2.5, 3.7},
			expectErr: false,
		},
		{
			name:      "invalid format",
			input:     "[1.0,abc]",
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stringToVector(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetSongEmbedding(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	t.Run("successful retrieval", func(t *testing.T) {
		expectedEmbedding := []float64{1.0, 2.0, 3.0}
		embeddingStr := "[1,2,3]"

		mock.ExpectQuery("SELECT embedding::text FROM song_embeddings WHERE song_id = \\$1").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"embedding"}).AddRow(embeddingStr))

		result, err := GetSongEmbedding(db, 1)
		assert.NoError(t, err)
		assert.Equal(t, expectedEmbedding, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("song not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT embedding::text FROM song_embeddings WHERE song_id = \\$1").
			WithArgs(999).
			WillReturnError(sqlmock.ErrCancelled)

		_, err := GetSongEmbedding(db, 999)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSaveSongEmbedding(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	t.Run("successful save", func(t *testing.T) {
		embedding := embedding38(1.0)

		mock.ExpectExec("INSERT INTO song_embeddings").
			WithArgs(1, embedding38String(1.0)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := SaveSongEmbedding(db, 1, embedding)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestEmbeddingDimensionGuard verifies that vectors whose length differs from
// the 38-dim contract are rejected in Go, before any SQL text is built
// (prime directive: the pgvector column is vector(38)).
//
// Objective: SaveSongEmbedding and FindSimilarSongs refuse non-38-dim input.
// Given-When-Then: Given a wrong-width vector, When the function is called,
// Then a dimension error is returned and no SQL statement is executed.
// Edge cases: empty and nil vectors are rejected too; a valid 38-dim vector
// passes the guard and reaches SQL.
func TestEmbeddingDimensionGuard(t *testing.T) {
	tests := []struct {
		name string
		vec  []float64
	}{
		{"too short", []float64{1, 2, 3}},
		{"too long", make([]float64, 39)},
		{"empty", []float64{}},
		{"nil", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/save", func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock: %v", err)
			}
			defer db.Close()

			err = SaveSongEmbedding(db, 1, tt.vec)
			assert.ErrorContains(t, err, "dimension")
			// No SQL must have been executed: no expectations were set and
			// sqlmock would have failed the call if one had been issued.
			assert.NoError(t, mock.ExpectationsWereMet())
		})
		t.Run(tt.name+"/find", func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock: %v", err)
			}
			defer db.Close()

			_, err = FindSimilarSongs(db, 1, tt.vec, 5)
			assert.ErrorContains(t, err, "dimension")
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("valid 38-dim passes the guard", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create mock: %v", err)
		}
		defer db.Close()

		mock.ExpectExec("INSERT INTO song_embeddings").
			WithArgs(1, embedding38String(0.5)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		assert.NoError(t, SaveSongEmbedding(db, 1, embedding38(0.5)))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindSimilarSongs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	queryEmbedding := embedding38(1.0)
	embeddingStr := embedding38String(1.0)

	// Use proper time.Time values
	time1, _ := time.Parse("2006-01-02", "2020-01-01")
	time2, _ := time.Parse("2006-01-02", "2021-01-01")

	rows := sqlmock.NewRows([]string{
		"id", "fingerprint_hash", "file_path", "title", "artist", "album", "year",
		"genre_id", "genre", "duration", "bitrate", "file_size", "last_modified", "similarity",
	}).
		AddRow(2, "hash2", "/path/2", "Song 2", "Artist 2", "Album 2", 2020, 1, "Rock", 200, 320, 5000000, time1, 0.95).
		AddRow(3, "hash3", "/path/3", "Song 3", "Artist 3", "Album 3", 2021, 1, "Rock", 180, 320, 4500000, time2, 0.90)

	mock.ExpectQuery("SELECT (.+) FROM song_embeddings se").
		WithArgs(embeddingStr, 1, 5).
		WillReturnRows(rows)

	result, err := FindSimilarSongs(db, 1, queryEmbedding, 5)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	if len(result) > 0 {
		assert.Equal(t, 2, result[0].ID)
		assert.Equal(t, 0.95, result[0].Similarity)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSongsMissingEmbeddings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	t.Run("returns songs without embeddings", func(t *testing.T) {
		mock.ExpectQuery(`SELECT s.id FROM songs s`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).
				AddRow(4).AddRow(9))

		ids, err := GetSongsMissingEmbeddings(context.Background(), db)
		assert.NoError(t, err)
		assert.Equal(t, []int{4, 9}, ids)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty when all songs have embeddings", func(t *testing.T) {
		mock.ExpectQuery(`SELECT s.id FROM songs s`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		ids, err := GetSongsMissingEmbeddings(context.Background(), db)
		assert.NoError(t, err)
		assert.Empty(t, ids)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
