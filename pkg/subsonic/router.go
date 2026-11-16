package subsonic

import (
	"go-postgres-example/pkg/handlers"

	"github.com/go-chi/chi/v5"
)

// NewRouter creates a new chi router and sets up the Subsonic routes
func NewRouter(authHandler *handlers.AuthHandler, subsonicHandler *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(AuthMiddleware(authHandler.DB, subsonicHandler.Cfg))

	r.Get("/ping.view", subsonicHandler.Ping)
	r.Get("/getMusicFolders.view", subsonicHandler.GetMusicFolders)
	r.Get("/getIndexes.view", subsonicHandler.GetIndexes)
	r.Get("/search3.view", subsonicHandler.Search3)
	r.Get("/stream.view", subsonicHandler.Stream)
	r.Get("/getSimilarSongs.view", subsonicHandler.GetSimilarSongs)
	r.Get("/getAlbums.view", subsonicHandler.GetAlbums)
	r.Get("/getAlbum.view", subsonicHandler.GetAlbum)
	r.Get("/getArtists.view", subsonicHandler.GetArtists)
	r.Get("/getArtist.view", subsonicHandler.GetArtist)
	r.Get("/getChildren.view", subsonicHandler.GetChildren)
	r.Get("/getArtwork.view", subsonicHandler.GetArtwork)
	r.Get("/getRandomSongs.view", subsonicHandler.GetRandomSongs)
	r.Get("/getPlaylists.view", subsonicHandler.GetPlaylists)
	r.Get("/getPlaylist.view", subsonicHandler.GetPlaylist)
	r.Get("/createPlaylist.view", subsonicHandler.CreatePlaylist)
	r.Get("/updatePlaylist.view", subsonicHandler.UpdatePlaylist)
	r.Get("/deletePlaylist.view", subsonicHandler.DeletePlaylist)

	return r
}
