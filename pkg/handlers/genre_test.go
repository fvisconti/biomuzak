package handlers_test

// Tests for the admin genre catalogue handlers (E3c).
//
// Objective: the genre catalogue is admin-managed: list (with song counts),
// create, and delete are reachable only by admins; duplicates are 409;
// unknown deletes are 404; empty names are 400.
//
// Given/When/Then: Given a JWT for user 1 and an is_admin row, When a
// non-admin calls any genre route, Then 403; an admin gets the underlying
// behaviour (200 list, 201 create, 409 duplicate, 200/404 delete).
//
// Edge cases: the gate is middleware.AdminOnly (same wiring as router.go);
// the delete route param is bound via the chi route context.

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"go-postgres-example/pkg/auth"
	"go-postgres-example/pkg/handlers"
	"go-postgres-example/pkg/middleware"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// withGenreID sets the chi route context so chi.URLParam(r, "genreID") resolves.
func withGenreID(r *http.Request, genreID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("genreID", genreID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func adminGenreRequest(t *testing.T, method, target, body string) (*http.Request, string) {
	t.Helper()
	token, _ := auth.GenerateJWT(1, "default-secret")
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, target, bytes.NewReader([]byte(body)))
	} else {
		req, err = http.NewRequest(method, target, nil)
	}
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req, token
}

func TestGenreHandlers_AdminGate(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		target   string
		isAdmin  bool
		wantCode int
	}{
		{"non-admin list forbidden", "GET", "/api/admin/genres", false, http.StatusForbidden},
		{"non-admin create forbidden", "POST", "/api/admin/genres", false, http.StatusForbidden},
		{"non-admin delete forbidden", "DELETE", "/api/admin/genres/3", false, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock: %v", err)
			}
			defer db.Close()

			mock.ExpectQuery(`SELECT COALESCE\(is_admin, FALSE\) FROM users WHERE id = \$1`).
				WithArgs(1).
				WillReturnRows(sqlmock.NewRows([]string{"is_admin"}).AddRow(tt.isAdmin))

			h := handlers.NewGenreHandler(db)
			req, _ := adminGenreRequest(t, tt.method, tt.target, "")
			rr := httptest.NewRecorder()

			var target http.HandlerFunc
			switch tt.method {
			case "GET":
				target = h.ListHandler
			case "POST":
				target = h.CreateHandler
			case "DELETE":
				target = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					h.DeleteHandler(w, withGenreID(r, "3"))
				})
			}
			handler := middleware.AdminOnly(db)(target)
			middleware.Authenticator("default-secret")(handler).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantCode, rr.Code)
			assert.Contains(t, rr.Body.String(), "Admin access required")
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGenreHandler_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT COALESCE\(is_admin, FALSE\) FROM users WHERE id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"is_admin"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta("LEFT JOIN songs s ON s.genre_id = g.id")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "song_count"}).
			AddRow(1, "House", 12))

	h := handlers.NewGenreHandler(db)
	req, _ := adminGenreRequest(t, "GET", "/api/admin/genres", "")
	rr := httptest.NewRecorder()

	handler := middleware.AdminOnly(db)(http.HandlerFunc(h.ListHandler))
	middleware.Authenticator("default-secret")(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"name":"House"`)
	assert.Contains(t, rr.Body.String(), `"song_count":12`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenreHandler_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT COALESCE\(is_admin, FALSE\) FROM users WHERE id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"is_admin"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO genres (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id, name")).
		WithArgs("Chillwave").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(7, "Chillwave"))

	h := handlers.NewGenreHandler(db)
	req, _ := adminGenreRequest(t, "POST", "/api/admin/genres", `{"name":"Chillwave"}`)
	rr := httptest.NewRecorder()

	handler := middleware.AdminOnly(db)(http.HandlerFunc(h.CreateHandler))
	middleware.Authenticator("default-secret")(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":7`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenreHandler_CreateDuplicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT COALESCE\(is_admin, FALSE\) FROM users WHERE id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"is_admin"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO genres (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id, name")).
		WithArgs("House").
		WillReturnError(sql.ErrNoRows)

	h := handlers.NewGenreHandler(db)
	req, _ := adminGenreRequest(t, "POST", "/api/admin/genres", `{"name":"House"}`)
	rr := httptest.NewRecorder()

	handler := middleware.AdminOnly(db)(http.HandlerFunc(h.CreateHandler))
	middleware.Authenticator("default-secret")(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGenreHandler_CreateNameTooLong rejects a name that cannot fit in
// genres.name (VARCHAR(255)) with 400, before any DB round trip.
func TestGenreHandler_CreateNameTooLong(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// No DB expectations: an oversized name is rejected before any query.

	h := handlers.NewGenreHandler(db)
	name := strings.Repeat("a", 256)
	req, _ := adminGenreRequest(t, "POST", "/api/admin/genres", `{"name":"`+name+`"}`)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.CreateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGenreHandler_CreateEmptyName(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// No DB expectations: an empty name is rejected before any query.

	h := handlers.NewGenreHandler(db)
	req, _ := adminGenreRequest(t, "POST", "/api/admin/genres", `{"name":"  "}`)
	rr := httptest.NewRecorder()

	middleware.Authenticator("default-secret")(http.HandlerFunc(h.CreateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGenreHandler_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT COALESCE\(is_admin, FALSE\) FROM users WHERE id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"is_admin"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM genres WHERE id = $1")).
		WithArgs(3).
		WillReturnResult(sqlmock.NewResult(1, 1))

	h := handlers.NewGenreHandler(db)
	req, _ := adminGenreRequest(t, "DELETE", "/api/admin/genres/3", "")
	rr := httptest.NewRecorder()

	handler := middleware.AdminOnly(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.DeleteHandler(w, withGenreID(r, "3"))
	}))
	middleware.Authenticator("default-secret")(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenreHandler_DeleteUnknown(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT COALESCE\(is_admin, FALSE\) FROM users WHERE id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"is_admin"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM genres WHERE id = $1")).
		WithArgs(99).
		WillReturnResult(sqlmock.NewResult(1, 0))

	h := handlers.NewGenreHandler(db)
	req, _ := adminGenreRequest(t, "DELETE", "/api/admin/genres/99", "")
	rr := httptest.NewRecorder()

	handler := middleware.AdminOnly(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.DeleteHandler(w, withGenreID(r, "99"))
	}))
	middleware.Authenticator("default-secret")(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
