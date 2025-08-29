package middleware

import (
	"context"
	"go-postgres-example/pkg/auth"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey = contextKey("userID")

// Authenticator is a middleware to protect routes
func Authenticator(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Could not find bearer token in Authorization header", http.StatusUnauthorized)
				return
			}

			token, err := auth.ValidateJWT(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if claims, ok := token.Claims.(*auth.Claims); ok && token.Valid {
				ctx := context.WithValue(r.Context(), userContextKey, claims.UserID)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
			}
		})
	}
}

// GetUserIDFromContext retrieves the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userContextKey).(int)
	return userID, ok
}
