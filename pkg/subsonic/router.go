package subsonic

import (
	"go-postgres-example/pkg/handlers"

	"github.com/go-chi/chi/v5"
)

// NewRouter creates a new chi router and sets up the Subsonic routes
func NewRouter(authHandler *handlers.AuthHandler, subsonicHandler *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(AuthMiddleware(authHandler.DB))

	r.Get("/ping.view", subsonicHandler.Ping)
	r.Get("/getMusicFolders.view", subsonicHandler.GetMusicFolders)
	r.Get("/getIndexes.view", subsonicHandler.GetIndexes)
	r.Get("/search3.view", subsonicHandler.Search3)
	r.Get("/stream.view", subsonicHandler.Stream)

	return r
}
