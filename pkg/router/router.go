package router

import (
	"encoding/json"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/subsonic"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

// New creates a new chi router and sets up the routes
func New(authHandler *handlers.AuthHandler, uploadHandler *handlers.UploadHandler, libraryHandler *handlers.LibraryHandler, playlistHandler *handlers.PlaylistHandler, songHandler *handlers.SongHandler, subsonicHandler *subsonic.Handler) *chi.Mux {
	r := chi.NewRouter()

	// Mount the Subsonic router
	r.Mount("/rest", subsonic.NewRouter(authHandler, subsonicHandler))

	// API routes
	// Public routes
	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticator(authHandler.Cfg.JWTSecret))

		r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
			userID, ok := middleware.GetUserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"message": "This is a protected route", "user_id": userID})
		})

		r.Post("/api/upload", uploadHandler.Upload)

		// Library and Song routes
		r.Get("/api/library", libraryHandler.GetLibraryHandler)
		r.Post("/api/songs/{songID}/rate", libraryHandler.RateSongHandler)
		r.Get("/api/songs/{songID}/similar", songHandler.GetSimilarSongsHandler)

		// Playlist routes
		r.Route("/api/playlists", func(r chi.Router) {
			r.Post("/", playlistHandler.CreatePlaylistHandler)
			r.Get("/", playlistHandler.GetUserPlaylistsHandler)

			r.Route("/{playlistID}", func(r chi.Router) {
				r.Get("/", playlistHandler.GetPlaylistHandler)
				r.Put("/", playlistHandler.UpdatePlaylistHandler)
				r.Delete("/", playlistHandler.DeletePlaylistHandler)

				// Playlist songs routes
				r.Post("/songs", playlistHandler.AddSongToPlaylistHandler)
				r.Delete("/songs/{songID}", playlistHandler.RemoveSongFromPlaylistHandler)
			})
		})
	})

	// Serve static files from frontend/build directory
	staticDir := "./frontend/build"
	if _, err := os.Stat(staticDir); err == nil {
		// Serve static files (CSS, JS, images, etc.)
		fileServer := http.FileServer(http.Dir(staticDir))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// Check if the requested file exists
			path := filepath.Join(staticDir, r.URL.Path)
			
			// If the file doesn't exist or is a directory, serve index.html for SPA routing
			if info, err := os.Stat(path); err != nil || info.IsDir() {
				http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
				return
			}
			
			// Otherwise serve the file
			fileServer.ServeHTTP(w, r)
		})
	}

	return r
}
