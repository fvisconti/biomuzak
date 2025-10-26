package config

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application's configuration
type Config struct {
	// Computed DSN for the database connection
	DatabaseURL string

	// Optional raw components (useful for debugging and future features)
	DatabaseHost     string
	DatabasePort     string
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseSSLMode  string

	JWTSecret        string
	Port             string
	MusicBrainzEmail string

	// Upload & Audio Processing
	UploadDir          string
	AudioProcessorURL  string

	// Registration & bootstrap
	AllowRegistration bool
	AdminUsername     string
	AdminPassword     string
	AdminEmail        string
}

// New creates a new Config struct
func New() *Config {
	// Although this is not a hard requirement, it's a good practice to use a .env file for local development
	if os.Getenv("ENV") != "production" {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found")
		}
	}

	cfg := &Config{
		JWTSecret:        getEnv("JWT_SECRET", "default-secret"),
		Port:             getEnv("PORT", "8080"),
		MusicBrainzEmail: getEnv("MUSICBRAINZ_EMAIL", "youremail@example.com"),

		UploadDir:         getEnv("UPLOAD_DIR", "./uploads"),
		AudioProcessorURL: getEnv("AUDIO_PROCESSOR_URL", "http://localhost:8000"),

		AllowRegistration: getEnv("ALLOW_REGISTRATION", "false") == "true",
		AdminUsername:     getEnv("ADMIN_USERNAME", ""),
		AdminPassword:     getEnv("ADMIN_PASSWORD", ""),
		AdminEmail:        getEnv("ADMIN_EMAIL", "admin@local"),
	}

	// Prefer explicit DATABASE_URL if provided
	if dsn := getEnv("DATABASE_URL", ""); dsn != "" {
		cfg.DatabaseURL = dsn
		return cfg
	}

	// Otherwise, construct DSN from individual env vars
	cfg.DatabaseHost = getEnv("DB_HOST", getEnv("POSTGRES_HOST", "localhost"))
	cfg.DatabasePort = getEnv("DB_PORT", getEnv("POSTGRES_PORT", "5432"))
	cfg.DatabaseName = getEnv("POSTGRES_DB", "musicdb")
	cfg.DatabaseUser = getEnv("POSTGRES_USER", "musicuser")
	cfg.DatabasePassword = getEnv("POSTGRES_PASSWORD", "musicpass")
	cfg.DatabaseSSLMode = getEnv("DB_SSLMODE", "disable")

	// Build a properly URL-encoded DSN
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DatabaseUser, cfg.DatabasePassword),
		Host:   fmt.Sprintf("%s:%s", cfg.DatabaseHost, cfg.DatabasePort),
		Path:   "/" + cfg.DatabaseName,
	}
	q := u.Query()
	if cfg.DatabaseSSLMode != "" {
		q.Set("sslmode", cfg.DatabaseSSLMode)
	}
	u.RawQuery = q.Encode()
	cfg.DatabaseURL = u.String()

	return cfg
}

// getEnv is a helper function to read an environment variable or return a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
