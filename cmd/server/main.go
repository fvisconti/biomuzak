package main

import (
	"log"

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
	err = db.Migrate(conn, "migrations/0001_initial_schema.up.sql")
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Application started successfully")
}
