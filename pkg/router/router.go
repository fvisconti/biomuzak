package router

import (
	"encoding/json"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// New creates a new chi router and sets up the routes
func New(authHandler *handlers.AuthHandler, uploadHandler *handlers.UploadHandler, libraryHandler *handlers.LibraryHandler) *chi.Mux {
	r := chi.NewRouter()

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

		// Library routes
		r.Get("/api/library", libraryHandler.GetLibraryHandler)
		r.Post("/api/songs/{songID}/rate", libraryHandler.RateSongHandler)
	})

	return r
}
