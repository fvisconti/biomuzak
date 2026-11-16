package subsonic

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/config"
)

type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware is a middleware that handles Subsonic authentication.
// It supports password auth (p) and token auth (t): the token is derived
// server-side from the username under the JWT secret (see token.go), so it
// can be verified without knowing the plaintext password.
func AuthMiddleware(db *sql.DB, cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			params := r.URL.Query()
			username := params.Get("u")
			password := params.Get("p")
			token := params.Get("t")

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

			if token != "" {
				// Token-based authentication (Subsonic spec: t). The token is
				// derived from the username under the JWT secret, so bcrypt is
				// not needed. Constant-time compare to avoid timing oracles.
				expected := DeriveSubsonicToken(cfg.JWTSecret, username)
				if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
					respondWithXML(w, &Response{
						Status: "failed",
						Error: &Error{
							Code:    40,
							Message: "Wrong username or token",
						},
					})
					return
				}
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
