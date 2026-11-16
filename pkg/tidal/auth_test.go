package tidal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newMockTidalServer returns a test server emulating the Tidal OAuth endpoints.
func newMockTidalServer(t *testing.T, tokenResp *TokenResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/oauth2/device_authorization":
			json.NewEncoder(w).Encode(DeviceAuthResponse{
				DeviceCode:              "dev-code",
				UserCode:                "ABCD-1234",
				VerificationURI:         "link.tidal.com",
				VerificationURIComplete: "link.tidal.com/ABCD-1234",
				ExpiresIn:               300,
				Interval:                2,
			})
		case "/v1/oauth2/token":
			// Distinguish device-code poll vs refresh by grant_type.
			body := r.FormValue("grant_type")
			if body == "urn:ietf:params:oauth:grant-type:device_code" {
				// First poll: pending. We can't easily track state here, so
				// return the token directly for the success path.
				json.NewEncoder(w).Encode(tokenResp)
				return
			}
			// refresh_token
			json.NewEncoder(w).Encode(tokenResp)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestStartDeviceAuth(t *testing.T) {
	srv := newMockTidalServer(t, &TokenResponse{})
	defer srv.Close()

	a := &AuthClient{
		ClientID:   "cid",
		ClientSecret: "csecret",
		BaseURL:    srv.URL + "/v1/oauth2",
		HTTPClient: srv.Client(),
	}

	dar, err := a.StartDeviceAuth(t.Context())
	if err != nil {
		t.Fatalf("StartDeviceAuth error: %v", err)
	}
	if dar.DeviceCode != "dev-code" || dar.UserCode != "ABCD-1234" {
		t.Fatalf("unexpected device auth response: %+v", dar)
	}
}

func TestPollDeviceAuth_Success(t *testing.T) {
	tr := &TokenResponse{
		AccessToken:  "at",
		RefreshToken: "rt",
		ExpiresIn:    604800,
	}
	tr.User.UserID = 42
	tr.User.Username = "alice"

	srv := newMockTidalServer(t, tr)
	defer srv.Close()

	a := &AuthClient{
		ClientID:     "cid",
		ClientSecret: "csecret",
		BaseURL:      srv.URL + "/v1/oauth2",
		HTTPClient:   srv.Client(),
	}

	got, err := a.PollDeviceAuth(t.Context(), "dev-code")
	if err != nil {
		t.Fatalf("PollDeviceAuth error: %v", err)
	}
	if got.AccessToken != "at" || got.RefreshToken != "rt" {
		t.Fatalf("unexpected token response: %+v", got)
	}
	if got.User.Username != "alice" {
		t.Fatalf("unexpected user: %+v", got.User)
	}
}

func TestPollDeviceAuth_Pending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TokenError{Error: "authorization_pending", SubStatus: 1002})
	}))
	defer srv.Close()

	a := &AuthClient{
		ClientID:     "cid",
		ClientSecret: "csecret",
		BaseURL:      srv.URL + "/v1/oauth2",
		HTTPClient:   srv.Client(),
	}

	_, err := a.PollDeviceAuth(t.Context(), "dev-code")
	if err != ErrAuthorizationPending {
		t.Fatalf("expected ErrAuthorizationPending, got %v", err)
	}
}

func TestPollDeviceAuth_Expired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TokenError{Error: "expired_token"})
	}))
	defer srv.Close()

	a := &AuthClient{
		ClientID:     "cid",
		ClientSecret: "csecret",
		BaseURL:      srv.URL + "/v1/oauth2",
		HTTPClient:   srv.Client(),
	}

	_, err := a.PollDeviceAuth(t.Context(), "dev-code")
	if err != ErrAuthorizationExpired {
		t.Fatalf("expected ErrAuthorizationExpired, got %v", err)
	}
}

func TestRefreshAccessToken(t *testing.T) {
	tr := &TokenResponse{AccessToken: "new-at", RefreshToken: "new-rt", ExpiresIn: 604800}
	srv := newMockTidalServer(t, tr)
	defer srv.Close()

	a := &AuthClient{
		ClientID:     "cid",
		ClientSecret: "csecret",
		BaseURL:      srv.URL + "/v1/oauth2",
		HTTPClient:   srv.Client(),
	}

	got, err := a.RefreshAccessToken(t.Context(), "old-rt")
	if err != nil {
		t.Fatalf("RefreshAccessToken error: %v", err)
	}
	if got.AccessToken != "new-at" {
		t.Fatalf("unexpected access token: %q", got.AccessToken)
	}
}
