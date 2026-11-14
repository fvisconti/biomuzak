package subsonic

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/models"
)

// parseSongIDs parses a comma-separated Subsonic songId list.
func parseSongIDs(raw string) ([]int, error) {
	var ids []int
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		ids = append(ids, n)
	}
	return ids, nil
}

// GetPlaylists is the /rest/getPlaylists.view handler. Playlists are
// per-user: the user id comes from the authenticated Subsonic context, never
// from a parameter.
func (h *Handler) GetPlaylists(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		failWith(w, 40, "Unauthorized")
		return
	}
	playlists, err := db.GetUserPlaylists(h.DB, userID)
	if err != nil {
		failWith(w, 0, "Failed to get playlists")
		return
	}
	list := &PlaylistList{}
	for _, p := range playlists {
		list.Playlists = append(list.Playlists, playlistToSubsonic(&p))
	}
	response := NewOkResponse()
	response.PlaylistList = list
	respondWithXML(w, response)
}

// playlistToSubsonic converts a stored playlist to the Subsonic model.
func playlistToSubsonic(p *models.Playlist) SubsonicPlaylist {
	return SubsonicPlaylist{
		ID:        strconv.Itoa(p.ID),
		Name:      p.Name,
		Public:    false,
		SongCount: p.SongCount,
		Created:   p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// GetPlaylist is the /rest/getPlaylist.view handler.
func (h *Handler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		failWith(w, 40, "Unauthorized")
		return
	}
	playlistID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		failWith(w, 10, "Required parameter 'id' is missing or invalid")
		return
	}
	playlist, err := db.GetPlaylistByID(h.DB, userID, playlistID)
	if err != nil {
		failWith(w, 70, "Playlist not found")
		return
	}
	songs, err := db.GetPlaylistSongs(h.DB, playlistID, "added_at")
	if err != nil {
		failWith(w, 0, "Failed to get playlist songs")
		return
	}
	sp := playlistToSubsonic(playlist)
	for _, ps := range songs {
		sp.Songs = append(sp.Songs, toSong(&ps.Song, r))
	}
	response := NewOkResponse()
	response.SubsonicPlaylist = &sp
	respondWithXML(w, response)
}

// CreatePlaylist is the /rest/createPlaylist.view handler.
func (h *Handler) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		failWith(w, 40, "Unauthorized")
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		failWith(w, 10, "Required parameter 'name' is missing")
		return
	}
	playlist, err := db.CreatePlaylist(h.DB, userID, name)
	if err != nil {
		failWith(w, 0, "Failed to create playlist")
		return
	}
	if raw := r.URL.Query().Get("songId"); raw != "" {
		ids, err := parseSongIDs(raw)
		if err != nil {
			failWith(w, 10, "Invalid songId list")
			return
		}
		for i, id := range ids {
			if err := db.AddSongToPlaylist(h.DB, playlist.ID, id, i+1); err != nil {
				failWith(w, 0, "Failed to add song to playlist")
				return
			}
		}
	}
	response := NewOkResponse()
	response.SubsonicPlaylist = &SubsonicPlaylist{
		ID:   strconv.Itoa(playlist.ID),
		Name: playlist.Name,
	}
	respondWithXML(w, response)
}

// UpdatePlaylist is the /rest/updatePlaylist.view handler: rename and/or
// replace the song list.
func (h *Handler) UpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		failWith(w, 40, "Unauthorized")
		return
	}
	playlistID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		failWith(w, 10, "Required parameter 'id' is missing or invalid")
		return
	}
	// Ownership check up front.
	if _, err := db.GetPlaylistByID(h.DB, userID, playlistID); err != nil {
		failWith(w, 70, "Playlist not found")
		return
	}

	if name := r.URL.Query().Get("name"); name != "" {
		err := db.UpdatePlaylistFields(h.DB, userID, playlistID,
			sql.NullString{String: name, Valid: true}, sql.NullString{})
		if err != nil {
			failWith(w, 0, "Failed to update playlist")
			return
		}
	}
	// Subsonic semantics: a songId list REPLACES the playlist contents.
	if raw := r.URL.Query().Get("songId"); raw != "" {
		ids, err := parseSongIDs(raw)
		if err != nil {
			failWith(w, 10, "Invalid songId list")
			return
		}
		current, err := db.GetPlaylistSongs(h.DB, playlistID, "position")
		if err != nil {
			failWith(w, 0, "Failed to update playlist songs")
			return
		}
		for _, ps := range current {
			if err := db.RemoveSongFromPlaylist(h.DB, playlistID, ps.Song.ID); err != nil {
				failWith(w, 0, "Failed to update playlist songs")
				return
			}
		}
		for i, id := range ids {
			if err := db.AddSongToPlaylist(h.DB, playlistID, id, i+1); err != nil {
				failWith(w, 0, "Failed to update playlist songs")
				return
			}
		}
	}
	response := NewOkResponse()
	respondWithXML(w, response)
}

// DeletePlaylist is the /rest/deletePlaylist.view handler.
func (h *Handler) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		failWith(w, 40, "Unauthorized")
		return
	}
	playlistID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		failWith(w, 10, "Required parameter 'id' is missing or invalid")
		return
	}
	if _, err := db.GetPlaylistByID(h.DB, userID, playlistID); err != nil {
		failWith(w, 70, "Playlist not found")
		return
	}
	if err := db.DeletePlaylist(h.DB, userID, playlistID); err != nil {
		failWith(w, 0, "Failed to delete playlist")
		return
	}
	response := NewOkResponse()
	respondWithXML(w, response)
}
