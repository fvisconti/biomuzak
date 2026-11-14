package db

import (
	"context"
	"database/sql"

	"go-postgres-example/pkg/models"
)

// fullSongSelect is the shared column list for subsonic browsing queries.
// Genre is COALESCEd so songs without a genre scan cleanly into a string.
const fullSongSelect = `
	SELECT s.id, s.title, s.artist, s.album, s.year,
	       COALESCE(g.name, ''), s.duration, s.bitrate, s.file_size, s.last_modified
	FROM songs s
	LEFT JOIN genres g ON s.genre_id = g.id`

// scanFullSong scans one full-song row.
func scanFullSong(row interface{ Scan(...any) error }) (*models.Song, error) {
	var song models.Song
	err := row.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Year,
		&song.Genre, &song.Duration, &song.Bitrate, &song.FileSize, &song.LastModified)
	if err != nil {
		return nil, err
	}
	return &song, nil
}

// GetSongFull retrieves a song with all fields needed for the Subsonic Song
// model (id, year, genre, duration, bitrate, size, last_modified).
func GetSongFull(ctx context.Context, db *sql.DB, songID int) (*models.Song, error) {
	row := db.QueryRowContext(ctx, fullSongSelect+" WHERE s.id = $1", songID)
	return scanFullSong(row)
}

// GetAllAlbums lists distinct (artist, album) pairs, ordered for stable
// Subsonic getAlbums responses.
func GetAllAlbums(ctx context.Context, db *sql.DB) ([]*models.Album, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT DISTINCT artist, album FROM songs WHERE album IS NOT NULL AND album != '' ORDER BY artist, album")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []*models.Album
	for rows.Next() {
		var album models.Album
		if err := rows.Scan(&album.Artist, &album.Name); err != nil {
			return nil, err
		}
		albums = append(albums, &album)
	}
	return albums, rows.Err()
}

// GetSongsByAlbum lists the songs of one album, ordered by id for stable
// playback order.
func GetSongsByAlbum(ctx context.Context, db *sql.DB, artist, album string) ([]*models.Song, error) {
	rows, err := db.QueryContext(ctx, fullSongSelect+" WHERE s.artist = $1 AND s.album = $2 ORDER BY s.id", artist, album)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []*models.Song
	for rows.Next() {
		song, err := scanFullSong(rows)
		if err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}

// GetSongsByArtist lists all songs by one artist.
func GetSongsByArtist(ctx context.Context, db *sql.DB, artist string) ([]*models.Song, error) {
	rows, err := db.QueryContext(ctx, fullSongSelect+" WHERE s.artist = $1 ORDER BY s.album, s.id", artist)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []*models.Song
	for rows.Next() {
		song, err := scanFullSong(rows)
		if err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}

// GetAlbumsByArtist lists the distinct albums of one artist.
func GetAlbumsByArtist(ctx context.Context, db *sql.DB, artist string) ([]*models.Album, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT DISTINCT album FROM songs WHERE artist = $1 AND album IS NOT NULL AND album != '' ORDER BY album", artist)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []*models.Album
	for rows.Next() {
		var album models.Album
		if err := rows.Scan(&album.Name); err != nil {
			return nil, err
		}
		album.Artist = artist
		albums = append(albums, &album)
	}
	return albums, rows.Err()
}

// GetArtistList lists distinct artists with their album count (Subsonic
// getArtists/getIndexes enrichment).
func GetArtistList(ctx context.Context, db *sql.DB) ([]*models.Artist, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT artist, COUNT(DISTINCT album)
		   FROM songs
		  WHERE artist IS NOT NULL AND artist != ''
		  GROUP BY artist
		  ORDER BY artist`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []*models.Artist
	for rows.Next() {
		var a models.Artist
		var albumCount int
		if err := rows.Scan(&a.Name, &albumCount); err != nil {
			return nil, err
		}
		a.AlbumCount = albumCount
		artists = append(artists, &a)
	}
	return artists, rows.Err()
}

// GetRandomSongs picks up to limit random songs (Subsonic getRandomSongs).
func GetRandomSongs(ctx context.Context, db *sql.DB, limit int) ([]*models.Song, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := db.QueryContext(ctx, fullSongSelect+" ORDER BY random() LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []*models.Song
	for rows.Next() {
		song, err := scanFullSong(rows)
		if err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}
