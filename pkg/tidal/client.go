package tidal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultAPIBaseURL = "https://api.tidal.com/v1"

// Client is a Tidal REST API client. It obtains a valid access token per call
// via the embedded AuthClient.
type Client struct {
	Auth     *AuthClient
	BaseURL  string
	HTTP     *http.Client
}

// NewClient builds a Tidal REST client.
func NewClient(auth *AuthClient) *Client {
	return &Client{
		Auth:    auth,
		BaseURL: defaultAPIBaseURL,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

// ---- Generic response envelopes ----

// Resource is a Tidal resource (track, album, playlist, artist).
type Resource struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ArtistName  string `json:"artistName"`
	AlbumName   string `json:"albumName"`
	Year        int    `json:"year"`
	Duration    int    `json:"duration"`
	Explicit    bool   `json:"explicit"`
	IsVideo     bool   `json:"isVideo"`
	IsAtmos     bool   `json:"isAtmos"`
	IsSingle    bool   `json:"isSingle"`
	Copyright   string `json:"copyright"`
}

// Page is a paginated Tidal list response.
type Page struct {
	Offset      int        `json:"offset"`
	Limit       int        `json:"limit"`
	TotalNumItems int      `json:"totalNumItems"`
	Items       []Resource `json:"items"`
}

// do performs an authenticated GET against the Tidal API and decodes JSON.
func (c *Client) do(ctx context.Context, userID int, path string, query url.Values, out interface{}) error {
	token, err := c.Auth.GetValidAccessToken(ctx, userID)
	if err != nil {
		return err
	}

	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("tidal request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read tidal response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tidal API error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("failed to parse tidal response: %w", err)
		}
	}
	return nil
}

// StreamURL is the response from the track audio endpoint.
type StreamURL struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
}

// GetLibraryTracks returns the user's Tidal library tracks.
func (c *Client) GetLibraryTracks(ctx context.Context, userID, offset, limit int) (*Page, error) {
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	var p Page
	if err := c.do(ctx, userID, "/albums/0/tracks", q, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetLibraryAlbums returns the user's Tidal library albums.
func (c *Client) GetLibraryAlbums(ctx context.Context, userID, offset, limit int) (*Page, error) {
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	var p Page
	if err := c.do(ctx, userID, "/albums", q, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetLibraryPlaylists returns the user's Tidal playlists.
func (c *Client) GetLibraryPlaylists(ctx context.Context, userID, offset, limit int) (*Page, error) {
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	var p Page
	if err := c.do(ctx, userID, "/playlists", q, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetAlbum returns a single album.
func (c *Client) GetAlbum(ctx context.Context, userID, albumID int) (*Resource, error) {
	var r Resource
	if err := c.do(ctx, userID, fmt.Sprintf("/albums/%d", albumID), nil, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// GetAlbumTracks returns the tracks in an album.
func (c *Client) GetAlbumTracks(ctx context.Context, userID, albumID, offset, limit int) (*Page, error) {
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	var p Page
	if err := c.do(ctx, userID, fmt.Sprintf("/albums/%d/tracks", albumID), q, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPlaylist returns a single playlist.
func (c *Client) GetPlaylist(ctx context.Context, userID, playlistID int) (*Resource, error) {
	var r Resource
	if err := c.do(ctx, userID, fmt.Sprintf("/playlists/%d", playlistID), nil, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// GetPlaylistTracks returns the tracks in a playlist.
func (c *Client) GetPlaylistTracks(ctx context.Context, userID, playlistID, offset, limit int) (*Page, error) {
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	var p Page
	if err := c.do(ctx, userID, fmt.Sprintf("/playlists/%d/tracks", playlistID), q, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetStreamURL returns the signed (encrypted) stream URL for a track at the
// given audioQualityId (e.g. "LOSSLESS").
func (c *Client) GetStreamURL(ctx context.Context, userID, trackID int, audioQualityID string) (*StreamURL, error) {
	q := url.Values{}
	q.Set("assetPresentation", "STREAM")
	q.Set("audioQualityId", audioQualityID)
	var s StreamURL
	if err := c.do(ctx, userID, fmt.Sprintf("/tracks/%d/audio", trackID), q, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
