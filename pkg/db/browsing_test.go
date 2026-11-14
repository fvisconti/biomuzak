package db

import (
	"context"
	"time"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullSongRows(ids ...int) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"id", "title", "artist", "album", "year", "genre", "duration", "bitrate", "file_size", "last_modified"})
	for i, id := range ids {
		rows.AddRow(id, "Song "+string(rune('A'+i)), "Artist X", "Album Y", 2020, "House", 200, 320, 1234, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	}
	return rows
}

func TestGetSongFull(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT s\.id, s\.title, s\.artist, s\.album, s\.year,`+`.*WHERE s\.id = \$1`).
		WithArgs(5).
		WillReturnRows(fullSongRows(5))

	song, err := GetSongFull(context.Background(), db, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, song.ID)
	assert.Equal(t, "House", song.Genre)
	assert.Equal(t, 320, song.Bitrate)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllAlbums(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT DISTINCT artist, album FROM songs`).
		WillReturnRows(sqlmock.NewRows([]string{"artist", "album"}).
			AddRow("A", "One").AddRow("A", "Two"))

	albums, err := GetAllAlbums(context.Background(), db)
	require.NoError(t, err)
	require.Len(t, albums, 2)
	assert.Equal(t, "One", albums[0].Name)
	assert.Equal(t, "A", albums[0].Artist)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSongsByAlbum(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`WHERE s\.artist = \$1 AND s\.album = \$2 ORDER BY s\.id`).
		WithArgs("A", "One").
		WillReturnRows(fullSongRows(1, 2))

	songs, err := GetSongsByAlbum(context.Background(), db, "A", "One")
	require.NoError(t, err)
	require.Len(t, songs, 2)
	assert.Equal(t, 1, songs[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSongsByArtist(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`WHERE s\.artist = \$1 ORDER BY s\.album, s\.id`).
		WithArgs("A").
		WillReturnRows(fullSongRows(1))

	songs, err := GetSongsByArtist(context.Background(), db, "A")
	require.NoError(t, err)
	require.Len(t, songs, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlbumsByArtist(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT DISTINCT album FROM songs WHERE artist = \$1`).
		WithArgs("A").
		WillReturnRows(sqlmock.NewRows([]string{"album"}).AddRow("One"))

	albums, err := GetAlbumsByArtist(context.Background(), db, "A")
	require.NoError(t, err)
	require.Len(t, albums, 1)
	assert.Equal(t, "One", albums[0].Name)
	assert.Equal(t, "A", albums[0].Artist)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetArtistList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT artist, COUNT\(DISTINCT album\)`).
		WillReturnRows(sqlmock.NewRows([]string{"artist", "album_count"}).
			AddRow("A", 3).AddRow("B", 1))

	artists, err := GetArtistList(context.Background(), db)
	require.NoError(t, err)
	require.Len(t, artists, 2)
	assert.Equal(t, 3, artists[0].AlbumCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRandomSongs_BindsLimit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`ORDER BY random\(\) LIMIT \$1`).
		WithArgs(20).
		WillReturnRows(fullSongRows(9))

	songs, err := GetRandomSongs(context.Background(), db, 20)
	require.NoError(t, err)
	require.Len(t, songs, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRandomSongs_CapsAt100(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`ORDER BY random\(\) LIMIT \$1`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err = GetRandomSongs(context.Background(), db, 5000)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
