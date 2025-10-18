package subsonic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"go-postgres-example/pkg/config"
)

func TestPing(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{}
	handler := NewHandler(db, cfg)

	req := httptest.NewRequest("GET", "/rest/ping.view", nil)
	rr := httptest.NewRecorder()

	handler.Ping(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "subsonic-response")
	assert.Contains(t, rr.Body.String(), `status="ok"`)
}

func TestGetMusicFolders(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{}
	handler := NewHandler(db, cfg)

	req := httptest.NewRequest("GET", "/rest/getMusicFolders.view", nil)
	rr := httptest.NewRecorder()

	handler.GetMusicFolders(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "musicFolders")
	assert.Contains(t, rr.Body.String(), "Music")
}

func TestNewOkResponse(t *testing.T) {
	response := NewOkResponse()
	
	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "1.16.1", response.Version)
	assert.Equal(t, "http://subsonic.org/restapi", response.XMLNS)
}

func TestRespondWithXML(t *testing.T) {
	rr := httptest.NewRecorder()
	response := NewOkResponse()
	
	respondWithXML(rr, response)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/xml; charset=UTF-8", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), `subsonic-response`)
	assert.Contains(t, rr.Body.String(), `status="ok"`)
}
