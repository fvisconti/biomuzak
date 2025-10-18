package middleware

import (
	"context"
	"go-postgres-example/pkg/auth"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthenticator(t *testing.T) {
	jwtSecret := "test-secret"
	userID := 123

	// Create a test handler that will be protected
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user ID is in context
		id, ok := GetUserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, userID, id)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with authenticator middleware
	protectedHandler := Authenticator(jwtSecret)(testHandler)

	t.Run("valid token", func(t *testing.T) {
		token, err := auth.GenerateJWT(userID, jwtSecret)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "success", rr.Body.String())
	})

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "Authorization header is required")
	})

	t.Run("missing Bearer prefix", func(t *testing.T) {
		token, err := auth.GenerateJWT(userID, jwtSecret)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", token) // Missing "Bearer " prefix
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "Could not find bearer token")
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid token")
	})

	t.Run("token with wrong secret", func(t *testing.T) {
		token, err := auth.GenerateJWT(userID, "wrong-secret")
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestGetUserIDFromContext(t *testing.T) {
	t.Run("user ID exists in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userContextKey, 123)
		userID, ok := GetUserIDFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, 123, userID)
	})

	t.Run("user ID does not exist in context", func(t *testing.T) {
		ctx := context.Background()
		_, ok := GetUserIDFromContext(ctx)
		assert.False(t, ok)
	})

	t.Run("wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userContextKey, "not-an-int")
		_, ok := GetUserIDFromContext(ctx)
		assert.False(t, ok)
	})
}
