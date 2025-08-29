package db

import (
	"context"
	"database/sql"
	"io/ioutil"
	"log"

	_ "github.com/jackc/pgx/v4/stdlib" // PostgreSQL driver
)

// NewConnection creates a new database connection pool
func NewConnection(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	// Ping the database to verify the connection
	err = db.PingContext(context.Background())
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to the database")
	return db, nil
}

// Migrate runs the database migrations
func Migrate(db *sql.DB, filePath string) error {
	// Read the migration file
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Execute the migration
	_, err = db.Exec(string(content))
	if err != nil {
		return err
	}

	log.Println("Successfully ran migrations")
	return nil
}
