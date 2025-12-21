package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
// MigrateAll applies all SQL migrations in the given directory, in filename order, using explicit version tracking.
func MigrateAll(db *sql.DB, migrationsDir string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	appliedVersions, err := fetchAppliedVersions(db)
	if err != nil {
		return fmt.Errorf("failed to load applied migrations: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}

	sort.Strings(files)

	// If the database already has the base tables but no migration history, seed the first migration as applied.
	if len(appliedVersions) == 0 {
		exists, terr := tableExists(db, "users")
		if terr != nil {
			return fmt.Errorf("failed to check existing tables: %w", terr)
		}
		if exists {
			if err := recordVersion(db, strings.TrimSuffix(files[0], ".sql")); err != nil {
				return fmt.Errorf("failed to seed base migration: %w", err)
			}
			appliedVersions[strings.TrimSuffix(files[0], ".sql")] = struct{}{}
			log.Printf("Seeded existing schema as applied for %s", files[0])
		}
	}

	for _, file := range files {
		version := strings.TrimSuffix(file, ".sql")
		if _, already := appliedVersions[version]; already {
			continue
		}

		path := filepath.Join(migrationsDir, file)
		content, rerr := os.ReadFile(path)
		if rerr != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, rerr)
		}

		tx, terr := db.Begin()
		if terr != nil {
			return fmt.Errorf("failed to start transaction for %s: %w", file, terr)
		}

		if _, execErr := tx.Exec(string(content)); execErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", file, execErr)
		}

		if err := recordVersionTx(tx, version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", file, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", file, err)
		}

		log.Printf("Applied migration %s", file)
	}

	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func fetchAppliedVersions(db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = struct{}{}
	}
	return applied, nil
}

func recordVersion(db *sql.DB, version string) error {
	_, err := db.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)", version, time.Now().UTC())
	return err
}

func recordVersionTx(tx *sql.Tx, version string) error {
	_, err := tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)", version, time.Now().UTC())
	return err
}

func tableExists(db *sql.DB, table string) (bool, error) {
	var exists bool
	err := db.QueryRow(`SELECT EXISTS (
		SELECT FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_name = $1
	)`, table).Scan(&exists)
	return exists, err
}
