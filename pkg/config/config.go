package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application's configuration
type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
}

// New creates a new Config struct
func New() *Config {
	// Although this is not a hard requirement, it's a good practice to use a .env file for local development
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Println("No .env file found")
		}
	}

	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", ""),
		JWTSecret:   getEnv("JWT_SECRET", "default-secret"),
		Port:        getEnv("PORT", "8080"),
	}
}

// getEnv is a helper function to read an environment variable or return a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
