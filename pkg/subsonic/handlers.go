package subsonic

import (
	"database/sql"
	"encoding/xml"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
)

// Handler holds the dependencies for the subsonic handlers
type Handler struct {
	DB  *sql.DB
	Cfg *config.Config
}

// NewHandler creates a new Handler
func NewHandler(db *sql.DB, cfg *config.Config) *Handler {
	return &Handler{DB: db, Cfg: cfg}
}

// Ping is a handler for the /rest/ping.view endpoint
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	respondWithXML(w, NewOkResponse())
}

// GetMusicFolders is a handler for the /rest/getMusicFolders.view endpoint
func (h *Handler) GetMusicFolders(w http.ResponseWriter, r *http.Request) {
	response := NewOkResponse()
	response.MusicFolders = &MusicFolders{
		MusicFolders: []MusicFolder{
			{ID: 1, Name: "Music"},
		},
	}
	respondWithXML(w, response)
}

// GetIndexes is a handler for the /rest/getIndexes.view endpoint
func (h *Handler) GetIndexes(w http.ResponseWriter, r *http.Request) {
	artists, err := db.GetAllArtists(h.DB)
	if err != nil {
		respondWithXML(w, &Response{
			Status: "failed",
			Error: &Error{
				Code:    0,
				Message: "Failed to get artists",
			},
		})
		return
	}

	indexMap := make(map[string][]Artist)
	for _, artist := range artists {
		firstLetter := strings.ToUpper(string(artist.Name[0]))
		indexMap[firstLetter] = append(indexMap[firstLetter], Artist{
			ID:   strconv.Itoa(artist.ID),
			Name: artist.Name,
		})
	}

	var indexes []Index
	for name, artists := range indexMap {
		indexes = append(indexes, Index{
			Name:    name,
			Artists: artists,
		})
	}

	// Sort indexes by name
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].Name < indexes[j].Name
	})

	response := NewOkResponse()
	response.Indexes = &Indexes{
		Indexes: indexes,
	}

	respondWithXML(w, response)
}

// Search3 is a handler for the /rest/search3.view endpoint
func (h *Handler) Search3(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		respondWithXML(w, &Response{
			Status: "failed",
			Error: &Error{
				Code:    10,
				Message: "Required parameter 'query' is missing",
			},
		})
		return
	}

	artists, albums, songs, err := db.Search(h.DB, query)
	if err != nil {
		respondWithXML(w, &Response{
			Status: "failed",
			Error: &Error{
				Code:    0,
				Message: "Search failed",
			},
		})
		return
	}

	response := NewOkResponse()
	response.SearchResult3 = &SearchResult3{}

	for _, artist := range artists {
		response.SearchResult3.Artists = append(response.SearchResult3.Artists, Artist{
			ID:   strconv.Itoa(artist.ID),
			Name: artist.Name,
		})
	}

	for _, album := range albums {
		response.SearchResult3.Albums = append(response.SearchResult3.Albums, Album{
			ID:     strconv.Itoa(album.ID),
			Name:   album.Name,
			Artist: album.Artist,
		})
	}

	for _, song := range songs {
		response.SearchResult3.Songs = append(response.SearchResult3.Songs, Song{
			ID:     strconv.Itoa(song.ID),
			Title:  song.Title,
			Artist: song.Artist,
			Album:  song.Album,
		})
	}

	respondWithXML(w, response)
}

// Stream is a handler for the /rest/stream.view endpoint
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		respondWithXML(w, &Response{
			Status: "failed",
			Error: &Error{
				Code:    10,
				Message: "Required parameter 'id' is missing",
			},
		})
		return
	}

	songID, err := strconv.Atoi(id)
	if err != nil {
		respondWithXML(w, &Response{
			Status: "failed",
			Error: &Error{
				Code:    70,
				Message: "Song not found",
			},
		})
		return
	}

	filePath, err := db.GetSongFilePath(h.DB, songID)
	if err != nil {
		respondWithXML(w, &Response{
			Status: "failed",
			Error: &Error{
				Code:    70,
				Message: "Song not found",
			},
		})
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	http.ServeContent(w, r, file.Name(), time.Time{}, file)
}

func respondWithXML(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	err := xml.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, "Failed to encode XML", http.StatusInternalServerError)
	}
}

// NewOkResponse creates a new SubsonicResponse with status "ok"
func NewOkResponse() *Response {
	return &Response{
		Status:  "ok",
		Version: "1.16.1",
		XMLNS:   "http://subsonic.org/restapi",
	}
}
