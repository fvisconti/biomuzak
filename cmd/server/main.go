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
	"go-postgres-example/pkg/storage"
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

	// Initialize storage
	minioStorage, err := storage.NewMinIOStorage(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Run migrations in explicit version order
	if err = db.MigrateAll(conn, "db/migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Ensure an admin user exists on first deployment
	if err := ensureAdminUser(conn, cfg); err != nil {
		log.Fatalf("Admin bootstrap failed: %v", err)
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(conn, cfg)
	uploadHandler := handlers.NewUploadHandler(conn, cfg, minioStorage)
	libraryHandler := handlers.NewLibraryHandler(conn, cfg)
	playlistHandler := handlers.NewPlaylistHandler(conn, cfg, minioStorage)
	songHandler := handlers.NewSongHandler(conn, cfg)
	streamHandler := handlers.NewStreamHandler(conn, cfg, minioStorage)

	// Initialize the Subsonic handler
	subsonicHandler := subsonic.NewHandler(conn, cfg, minioStorage)

	// Initialize the router
	r := router.New(authHandler, uploadHandler, libraryHandler, playlistHandler, songHandler, streamHandler, subsonicHandler)

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
		log.Printf("No admin credentials provided; skipping admin bootstrap")
		return nil
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
