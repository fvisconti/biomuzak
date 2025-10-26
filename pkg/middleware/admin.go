package middleware

import (
	"database/sql"
	"net/http"
)

// AdminOnly ensures the authenticated user is an admin
func AdminOnly(db *sql.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
				return
			}
			var isAdmin bool
			err := db.QueryRow("SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = $1", userID).Scan(&isAdmin)
			if err != nil {
				http.Error(w, "Failed to verify admin", http.StatusForbidden)
				return
			}
			if !isAdmin {
				http.Error(w, "Admin access required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
