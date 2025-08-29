package models

import (
	"database/sql"
	"time"
)

// LibrarySong represents a song in the user's library, including the user's rating
type LibrarySong struct {
	ID              int           `json:"id"`
	FingerprintHash string        `json:"fingerprint_hash"`
	FilePath        string        `json:"file_path"`
	Title           string        `json:"title"`
	Artist          string        `json:"artist"`
	Album           string        `json:"album"`
	Year            int           `json:"year"`
	GenreID         sql.NullInt64 `json:"-"`
	Genre           string        `json:"genre"`
	Duration        int           `json:"duration"`
	Bitrate         int           `json:"bitrate"`
	FileSize        int64         `json:"file_size"`
	LastModified    time.Time     `json:"last_modified"`
	Rating          sql.NullInt64 `json:"rating"`
}
