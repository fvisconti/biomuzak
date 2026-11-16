package subsonic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/middleware"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSubsonicTokenHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := NewTokenHandler(db, &config.Config{JWTSecret: "test-secret"})

	mock.ExpectQuery(`SELECT username FROM users WHERE id = \$1`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("alice"))

	req := httptest.NewRequest("GET", "/api/subsonic/token", nil)
	req = req.WithContext(middleware.WithUserIDForTest(req.Context(), 7))
	rr := httptest.NewRecorder()
	h.GetSubsonicTokenHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	expected := DeriveSubsonicToken("test-secret", "alice")
	assert.Contains(t, rr.Body.String(), expected)
	assert.NoError(t, mock.ExpectationsWereMet())
}
