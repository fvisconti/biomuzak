package subsonic

import (
	"context"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"

	"go-postgres-example/pkg/auth"
)

type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware is a middleware that handles Subsonic authentication
func AuthMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			params := r.URL.Query()
			username := params.Get("u")
			password := params.Get("p")
			token := params.Get("t")
			salt := params.Get("s")

			if username == "" {
				respondWithXML(w, &Response{
					Status: "failed",
					Error: &Error{
						Code:    10,
						Message: "Required parameter 'u' is missing",
					},
				})
				return
			}

			var userID int
			var passwordHash string
			err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = $1", username).Scan(&userID, &passwordHash)
			if err != nil {
				if err == sql.ErrNoRows {
					respondWithXML(w, &Response{
						Status: "failed",
						Error: &Error{
							Code:    40,
							Message: "Wrong username or password",
						},
					})
					return
				}
				http.Error(w, "Failed to query user", http.StatusInternalServerError)
				return
			}

			if token != "" && salt != "" {
				// Token-based authentication (Subsonic spec: t = md5(plaintext_password + salt)).
				// We store passwords as one-way bcrypt hashes, so we cannot recompute
				// md5(plaintext + salt) server-side. Rather than fall back to an insecure
				// check, we fail closed: token auth is not supported by this backend.
				// Clients should authenticate using the password parameter (p) instead.
				respondWithXML(w, &Response{
					Status: "failed",
					Error: &Error{
						Code:    40,
						Message: "Token authentication is not supported; use the password parameter",
					},
				})
				return
			} else if password != "" {
				// Password-based authentication.
				if strings.HasPrefix(password, "enc:") {
					hexPassword, err := hex.DecodeString(strings.TrimPrefix(password, "enc:"))
					if err != nil {
						respondWithXML(w, &Response{
							Status: "failed",
							Error: &Error{
								Code:    40,
								Message: "Wrong username or password",
							},
						})
						return
					}
					password = string(hexPassword)
				}
				// Verify the supplied password against the stored bcrypt hash.
				if !auth.CheckPasswordHash(password, passwordHash) {
					respondWithXML(w, &Response{
						Status: "failed",
						Error: &Error{
							Code:    40,
							Message: "Wrong username or password",
						},
					})
					return
				}
			} else {
				respondWithXML(w, &Response{
					Status: "failed",
					Error: &Error{
						Code:    10,
						Message: "Required authentication parameters are missing",
					},
				})
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext returns the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userIDKey).(int)
	return userID, ok
}
