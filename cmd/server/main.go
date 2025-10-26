package main

import (
	"database/sql"
	"fmt"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/router"
	"go-postgres-example/pkg/subsonic"
	"log"
	"net/http"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
)

func main() {
	// Initialize configuration
	cfg := config.New()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Connect to the database
	conn, err := db.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer conn.Close()

	// Run migrations
	exists, terr := tableExists(conn, "users")
	if terr != nil {
		log.Fatalf("Failed checking tables: %v", terr)
	}
	if !exists {
		if err = db.Migrate(conn, "db/migrations/0001_initial_schema.sql"); err != nil {
			log.Fatalf("Failed to run migration 0001: %v", err)
		}
	} else {
		log.Println("Skipping 0001_initial_schema.sql (users table exists)")
	}
	if err = db.Migrate(conn, "db/migrations/0002_enable_pg_trgm.sql"); err != nil {
		log.Fatalf("Failed to run migration 0002: %v", err)
	}
	if err = db.Migrate(conn, "db/migrations/0003_add_is_admin.sql"); err != nil {
		log.Fatalf("Failed to run migration 0003: %v", err)
	}

	// Ensure an admin user exists on first deployment
	if err := ensureAdminUser(conn, cfg); err != nil {
		log.Fatalf("Admin bootstrap failed: %v", err)
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(conn, cfg)
	uploadHandler := handlers.NewUploadHandler(conn, cfg)
	libraryHandler := handlers.NewLibraryHandler(conn, cfg)
	playlistHandler := handlers.NewPlaylistHandler(conn, cfg)
	songHandler := handlers.NewSongHandler(conn, cfg)
	subsonicHandler := subsonic.NewHandler(conn, cfg)

	// Initialize router
	r := router.New(authHandler, uploadHandler, libraryHandler, playlistHandler, songHandler, subsonicHandler)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// ensureAdminUser creates the initial admin user if the users table is empty
func ensureAdminUser(conn *sql.DB, cfg *config.Config) error {
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return fmt.Errorf("failed counting users: %w", err)
	}
	if count > 0 {
		return nil
	}

	if cfg.AdminUsername == "" || cfg.AdminPassword == "" {
		return fmt.Errorf("no users exist; please set ADMIN_USERNAME and ADMIN_PASSWORD in environment to bootstrap the initial admin user")
	}

	// Hash password
	hashed, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// Insert admin user
	_, err = conn.Exec("INSERT INTO users (username, email, password_hash, is_admin) VALUES ($1, $2, $3, TRUE)", cfg.AdminUsername, cfg.AdminEmail, hashed)
	if err != nil {
		return fmt.Errorf("failed inserting admin user: %w", err)
	}
	log.Printf("Bootstrap admin user '%s' created", cfg.AdminUsername)
	return nil
}

// tableExists checks whether a table exists in the public schema
func tableExists(conn *sql.DB, table string) (bool, error) {
	var exists bool
	err := conn.QueryRow(`SELECT EXISTS (
		SELECT FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_name = $1
	)`, table).Scan(&exists)
	return exists, err
}
