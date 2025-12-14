package models

import (
	"database/sql"
	"time"
)

// Song represents a song in the database
type Song struct {
	ID              int           `json:"id"`
	FingerprintHash string        `json:"fingerprint_hash"`
	FilePath        string        `json:"file_path"`
	Title           string        `json:"title"`
	Artist          string        `json:"artist"`
	Album           string        `json:"album"`
	Year            int           `json:"year"`
	GenreID         sql.NullInt64 `json:"-"`     // Foreign key to genres table
	Genre           string        `json:"genre"` // Genre name for JSON response
	Duration        int           `json:"duration"`
	Bitrate         int           `json:"bitrate"`
	FileSize        int64         `json:"file_size"`
	LastModified    time.Time     `json:"last_modified"`
}

// Genre represents a genre in the database
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Album represents an album in the database
type Album struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
}

// Artist represents an artist in the database
type Artist struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
