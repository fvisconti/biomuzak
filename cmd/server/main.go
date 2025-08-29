package main

import (
	"fmt"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/router"
	"log"
	"net/http"

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
	err = db.Migrate(conn, "db/migrations/0001_initial_schema.sql")
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(conn, cfg)
	uploadHandler := handlers.NewUploadHandler(conn, cfg)
	libraryHandler := handlers.NewLibraryHandler(conn, cfg)

	// Initialize router
	r := router.New(authHandler, uploadHandler, libraryHandler)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
