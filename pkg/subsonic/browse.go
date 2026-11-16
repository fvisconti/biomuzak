package subsonic

import (
	"net/http"
	"strconv"
	"time"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/models"
)

// toSong converts a stored song into the Subsonic Song model, deriving the
// synthesized album id and an absolute streamUrl from the request.
func toSong(s *models.Song, r *http.Request) Song {
	albumID := makeAlbumID(s.Artist, s.Album)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return Song{
		ID:        strconv.Itoa(s.ID),
		Parent:    albumID,
		IsDir:     false,
		Title:     s.Title,
		Album:     s.Album,
		Artist:    s.Artist,
		AlbumID:   albumID,
		Year:      s.Year,
		Genre:     s.Genre,
		Duration:  s.Duration,
		Bitrate:   s.Bitrate,
		Size:      s.FileSize,
		Created:   s.LastModified.UTC().Format(time.RFC3339),
		StreamURL: scheme + "://" + r.Host + "/rest/stream.view?id=" + strconv.Itoa(s.ID),
		IsVideo:   false,
	}
}

// albumFromSongs builds the album metadata from its songs.
func albumFromSongs(artist, name string, songs []*models.Song) *Album {
	var duration int
	for _, s := range songs {
		duration += s.Duration
	}
	return &Album{
		ID:        makeAlbumID(artist, name),
		Name:      name,
		Artist:    artist,
		ArtistID:  makeArtistID(artist),
		SongCount: len(songs),
		Duration:  duration,
	}
}

// failWith responds with a Subsonic error.
func failWith(w http.ResponseWriter, code int, message string) {
	respondWithXML(w, &Response{
		Status:  "failed",
		Version: "1.16.1",
		XMLNS:   "http://subsonic.org/restapi",
		Error:   &Error{Code: code, Message: message},
	})
}

// GetSimilarSongs is the /rest/getSimilarSongs.view handler (reuses the E2
// vector-similarity pipeline).
func (h *Handler) GetSimilarSongs(w http.ResponseWriter, r *http.Request) {
	songID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		failWith(w, 10, "Required parameter 'id' is missing or invalid")
		return
	}
	limit := h.Cfg.SimilarSongsLimit
	if limit < 1 {
		limit = config.DefaultSimilarSongsLimit
	}

	embedding, err := db.GetSongEmbedding(h.DB, songID)
	if err != nil {
		failWith(w, 70, "Song not found")
		return
	}
	songs, err := db.FindSimilarSongs(h.DB, songID, embedding, limit)
	if err != nil {
		failWith(w, 0, "Failed to find similar songs")
		return
	}

	result := &SimilarSongsResult{}
	for _, s := range songs {
		result.Songs = append(result.Songs, toSong(&s.Song, r))
	}
	response := NewOkResponse()
	response.SimilarSongs = result
	respondWithXML(w, response)
}

// GetAlbums is the /rest/getAlbums.view handler.
func (h *Handler) GetAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := db.GetAllAlbums(r.Context(), h.DB)
	if err != nil {
		failWith(w, 0, "Failed to get albums")
		return
	}
	list := &AlbumList{}
	for _, a := range albums {
		list.Albums = append(list.Albums, Album{
			ID:       makeAlbumID(a.Artist, a.Name),
			Name:     a.Name,
			Artist:   a.Artist,
			ArtistID: makeArtistID(a.Artist),
		})
	}
	response := NewOkResponse()
	response.AlbumList = list
	respondWithXML(w, response)
}

// GetAlbum is the /rest/getAlbum.view handler.
func (h *Handler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	artist, albumName, err := parseAlbumID(r.URL.Query().Get("id"))
	if err != nil {
		failWith(w, 70, "Album not found")
		return
	}
	songs, err := db.GetSongsByAlbum(r.Context(), h.DB, artist, albumName)
	if err != nil || len(songs) == 0 {
		failWith(w, 70, "Album not found")
		return
	}
	album := albumFromSongs(artist, albumName, songs)
	for _, s := range songs {
		album.Songs = append(album.Songs, toSong(s, r))
	}
	response := NewOkResponse()
	response.Album = album
	respondWithXML(w, response)
}

// GetArtists is the /rest/getArtists.view handler.
func (h *Handler) GetArtists(w http.ResponseWriter, r *http.Request) {
	artists, err := db.GetArtistList(r.Context(), h.DB)
	if err != nil {
		failWith(w, 0, "Failed to get artists")
		return
	}
	list := &ArtistList{}
	for _, a := range artists {
		list.Artists = append(list.Artists, Artist{
			ID:         makeArtistID(a.Name),
			Name:       a.Name,
			AlbumCount: a.AlbumCount,
		})
	}
	response := NewOkResponse()
	response.ArtistList = list
	respondWithXML(w, response)
}

// GetArtist is the /rest/getArtist.view handler.
func (h *Handler) GetArtist(w http.ResponseWriter, r *http.Request) {
	artistName, err := parseArtistID(r.URL.Query().Get("id"))
	if err != nil {
		failWith(w, 70, "Artist not found")
		return
	}
	albums, err := db.GetAlbumsByArtist(r.Context(), h.DB, artistName)
	if err != nil || len(albums) == 0 {
		failWith(w, 70, "Artist not found")
		return
	}
	artist := &Artist{ID: makeArtistID(artistName), Name: artistName, AlbumCount: len(albums)}
	for _, a := range albums {
		artist.Albums = append(artist.Albums, Album{
			ID:       makeAlbumID(a.Artist, a.Name),
			Name:     a.Name,
			Artist:   a.Artist,
			ArtistID: makeArtistID(a.Artist),
		})
	}
	response := NewOkResponse()
	response.Artist = artist
	respondWithXML(w, response)
}

// GetChildren is the /rest/getChildren.view handler. Depending on the id
// kind it returns: a song's album (with sibling songs), an artist's albums,
// or an album's songs.
func (h *Handler) GetChildren(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	response := NewOkResponse()

	switch {
	case id == "":
		failWith(w, 10, "Required parameter 'id' is missing")
		return
	case id == "1":
		// The single root music folder (advertised by getMusicFolders):
		// all albums. Checked before the digit case because "1" is a digit
		// too — the root folder wins the ambiguity on purpose.
		h.GetAlbums(w, r)
		return
	case id[0] >= '0' && id[0] <= '9':
		// A song id: return the album containing it.
		songID, err := strconv.Atoi(id)
		if err != nil {
			failWith(w, 70, "Not found")
			return
		}
		song, err := db.GetSongFull(r.Context(), h.DB, songID)
		if err != nil {
			failWith(w, 70, "Not found")
			return
		}
		songs, err := db.GetSongsByAlbum(r.Context(), h.DB, song.Artist, song.Album)
		if err != nil {
			failWith(w, 0, "Failed to load album")
			return
		}
		album := albumFromSongs(song.Artist, song.Album, songs)
		for _, s := range songs {
			album.Songs = append(album.Songs, toSong(s, r))
		}
		response.Album = album
	default:
		if artist, albumName, err := parseAlbumID(id); err == nil {
			songs, err := db.GetSongsByAlbum(r.Context(), h.DB, artist, albumName)
			if err != nil || len(songs) == 0 {
				failWith(w, 70, "Not found")
				return
			}
			album := albumFromSongs(artist, albumName, songs)
			for _, s := range songs {
				album.Songs = append(album.Songs, toSong(s, r))
			}
			response.Album = album
			respondWithXML(w, response)
			return
		}
		if artistName, err := parseArtistID(id); err == nil {
			albums, err := db.GetAlbumsByArtist(r.Context(), h.DB, artistName)
			if err != nil || len(albums) == 0 {
				failWith(w, 70, "Not found")
				return
			}
			artist := &Artist{ID: makeArtistID(artistName), Name: artistName, AlbumCount: len(albums)}
			for _, a := range albums {
				artist.Albums = append(artist.Albums, Album{
					ID:       makeAlbumID(a.Artist, a.Name),
					Name:     a.Name,
					Artist:   a.Artist,
					ArtistID: makeArtistID(a.Artist),
				})
			}
			response.Artist = artist
			respondWithXML(w, response)
			return
		}
		failWith(w, 70, "Not found")
	}
	respondWithXML(w, response)
}

// GetArtwork is the /rest/getArtwork.view handler. biomuzak stores no cover
// art, so clients get a clean "not found" (Subsonic error 70) and fall back
// to their defaults.
func (h *Handler) GetArtwork(w http.ResponseWriter, r *http.Request) {
	failWith(w, 70, "Cover art not found")
}

// GetRandomSongs is the /rest/getRandomSongs.view handler.
func (h *Handler) GetRandomSongs(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	songs, err := db.GetRandomSongs(r.Context(), h.DB, limit)
	if err != nil {
		failWith(w, 0, "Failed to get random songs")
		return
	}
	result := &RandomSongsResult{}
	for _, s := range songs {
		result.Songs = append(result.Songs, toSong(s, r))
	}
	response := NewOkResponse()
	response.RandomSongs = result
	respondWithXML(w, response)
}
