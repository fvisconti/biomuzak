package tidal

// Client request-path tests: the link-import path builds URLs by ID, so the
// exact paths the methods issue matter.
import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// validCredentialsRow returns a mock row with a token valid well beyond the
// 1-hour refresh skew.
func validCredentialsRow(userID int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"user_id", "tidal_user_id", "tidal_username",
		"access_token", "refresh_token", "access_token_expires_at", "config"}).
		AddRow(userID, 7, "user", "tok", "ignored", time.Now().Add(2*time.Hour),
			`{"quality":"normal","concurrency":1}`)
}

func TestClient_PlaylistURLIDPaths(t *testing.T) {
	const plID = "pldeadbeefcafebabe12345678"
	var seen []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/tracks") {
			w.Write([]byte(`{"offset":0,"limit":500,"totalNumItems":0,"items":[]}`))
		} else {
			w.Write([]byte(`{"id":0,"title":"PL"}`))
		}
	}))
	defer ts.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// One credentials lookup per do() call.
	mock.ExpectQuery(`tidal_credentials WHERE user_id = \$1`).
		WithArgs(1).WillReturnRows(validCredentialsRow(1))
	mock.ExpectQuery(`tidal_credentials WHERE user_id = \$1`).
		WithArgs(1).WillReturnRows(validCredentialsRow(1))
	mock.ExpectQuery(`tidal_credentials WHERE user_id = \$1`).
		WithArgs(1).WillReturnRows(validCredentialsRow(1))

	client := NewClient(NewAuthClient("id", "secret", db))
	client.BaseURL = ts.URL

	ctx := context.Background()

	pl, err := client.GetPlaylistByURLID(ctx, 1, plID)
	assert.NoError(t, err)
	assert.Equal(t, "PL", pl.Title)

	_, err = client.GetPlaylistTracksByURLID(ctx, 1, plID, 0, 500)
	assert.NoError(t, err)

	_, err = client.GetTrack(ctx, 1, 7)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Equal(t, []string{
		"/playlists/" + plID,
		"/playlists/" + plID + "/tracks",
		"/tracks/7",
	}, seen)
}
