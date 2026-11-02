package main

import (
	"context"
	"database/sql"
	"fmt"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/router"
	"go-postgres-example/pkg/subsonic"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

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

	// Refuse to start with the insecure default JWT secret: anyone could forge
	// valid tokens and impersonate any user.
	if cfg.JWTSecret == "" || cfg.JWTSecret == "default-secret" {
		log.Fatal("JWT_SECRET must be set to a strong, unique value (the default is not allowed)")
	}

	// Object-storage credentials are secrets: there is no meaningful default
	// (directive 3). Missing MINIO_ACCESS_KEY / MINIO_SECRET_KEY fail fast at
	// boot instead of silently trying well-known credentials.
	if cfg.MinIOAccessKey == "" || cfg.MinIOSecretKey == "" {
		log.Fatal("MINIO_ACCESS_KEY and MINIO_SECRET_KEY are required")
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
	authHandler := handlers.NewAuthHandler(conn, cfg, minioStorage)
	uploadHandler := handlers.NewUploadHandler(conn, cfg, minioStorage)
	libraryHandler := handlers.NewLibraryHandler(conn, cfg)
	playlistHandler := handlers.NewPlaylistHandler(conn, cfg, minioStorage)
	songHandler := handlers.NewSongHandler(conn, cfg, minioStorage)
	streamHandler := handlers.NewStreamHandler(conn, cfg, minioStorage)
	tidalHandler := handlers.NewTidalHandler(conn, cfg, minioStorage)
	exportHandler := handlers.NewExportHandler(conn, cfg, minioStorage, cfg.ExportConcurrency)
	backfillHandler := handlers.NewBackfillHandler(conn, cfg, minioStorage)

	// Initialize the Subsonic handler
	subsonicHandler := subsonic.NewHandler(conn, cfg, minioStorage)

	// Initialize the router
	r := router.New(authHandler, uploadHandler, libraryHandler, playlistHandler, songHandler, streamHandler, subsonicHandler, tidalHandler, exportHandler, backfillHandler, cfg.CORSAllowedOrigins)

	// Start server with explicit timeouts to mitigate Slowloris and
	// connection-exhaustion attacks.
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      10 * time.Minute, // generous: large downloads/streams
		IdleTimeout:       2 * time.Minute,
	}
	log.Printf("Server starting on port %s", cfg.Port)

	// Graceful shutdown: on SIGINT/SIGTERM stop accepting connections, cancel
	// any running backfill job (its detached goroutine needs a clear exit
	// path), and let in-flight requests drain within a bounded timeout.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	case <-ctx.Done():
		log.Printf("Shutdown signal received; stopping server and cancelling backfill")
		handlers.CancelBackfill()
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelShutdown()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		<-errCh // ListenAndServe returns http.ErrServerClosed
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
