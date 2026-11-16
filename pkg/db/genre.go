package db

import (
	"database/sql"
	"fmt"

	"go-postgres-example/pkg/models"
)

// ListGenres returns every genre with the number of songs referencing it.
// Genres with no songs show a count of 0.
func ListGenres(db *sql.DB) ([]models.GenreWithCount, error) {
	query := `
		SELECT g.id, g.name, COUNT(s.id) AS song_count
		FROM genres g
		LEFT JOIN songs s ON s.genre_id = g.id
		GROUP BY g.id, g.name
		ORDER BY g.name
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var genres []models.GenreWithCount
	for rows.Next() {
		var g models.GenreWithCount
		if err := rows.Scan(&g.ID, &g.Name, &g.SongCount); err != nil {
			return nil, fmt.Errorf("scan genre: %w", err)
		}
		genres = append(genres, g)
	}
	return genres, rows.Err()
}

// CreateGenre inserts a new genre and returns it. A duplicate name yields
// sql.ErrNoRows (the ON CONFLICT DO NOTHING upsert returns no row), which the
// handler maps to 409.
func CreateGenre(db *sql.DB, name string) (*models.Genre, error) {
	var g models.Genre
	err := db.QueryRow(
		`INSERT INTO genres (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id, name`,
		name,
	).Scan(&g.ID, &g.Name)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// DeleteGenre removes a genre. Songs referencing it keep their data with a
// NULL genre (FK ON DELETE SET NULL). Returns sql.ErrNoRows when no row is
// affected.
func DeleteGenre(db *sql.DB, genreID int) error {
	res, err := db.Exec(`DELETE FROM genres WHERE id = $1`, genreID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
