package models

import (
	"time"
)

// Playlist represents a playlist in the database
type Playlist struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PlaylistSong represents a song in a playlist, with its position.
// It embeds the Song model to include all song details.
type PlaylistSong struct {
	Song
	Position int `json:"position"`
}

// FullPlaylist represents a playlist along with its songs.
// This is likely what we'll return from the GetPlaylist handler.
type FullPlaylist struct {
	ID        int            `json:"id"`
	UserID    int            `json:"user_id"`
	Name      string         `json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Songs     []PlaylistSong `json:"songs"`
}
