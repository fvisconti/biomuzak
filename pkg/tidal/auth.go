package tidal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultAuthBaseURL = "https://auth.tidal.com/v1/oauth2"
	tidalScope         = "r_usr+w_usr+w_sub"
	// refresh a little before the 7-day expiry
	refreshSkew = time.Hour
)

// AuthClient performs Tidal OAuth (Device Authorization flow) and token refresh.
// BaseURL and HTTPClient are injectable for testing.
type AuthClient struct {
	ClientID     string
	ClientSecret string
	BaseURL      string // e.g. https://auth.tidal.com/v1/oauth2
	HTTPClient   *http.Client
	DB           *sql.DB
}

// NewAuthClient builds an AuthClient with production defaults.
func NewAuthClient(clientID, clientSecret string, db *sql.DB) *AuthClient {
	return &AuthClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		BaseURL:      defaultAuthBaseURL,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		DB:           db,
	}
}

// DeviceAuthResponse is the response from the device_authorization endpoint.
type DeviceAuthResponse struct {
	DeviceCode              string `json:"deviceCode"`
	UserCode                string `json:"userCode"`
	VerificationURI         string `json:"verificationUri"`
	VerificationURIComplete string `json:"verificationUriComplete"`
	ExpiresIn               int    `json:"expiresIn"`
	Interval                int    `json:"interval"`
}

// TokenResponse is the response from the token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	User         struct {
		UserID   int64  `json:"userId"`
		Username string `json:"username"`
	} `json:"user"`
}

// TokenError represents an OAuth error response.
type TokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	SubStatus        int    `json:"sub_status"`
}

// StartDeviceAuth initiates the Device Authorization flow.
func (a *AuthClient) StartDeviceAuth(ctx context.Context) (*DeviceAuthResponse, error) {
	form := url.Values{}
	form.Set("client_id", a.ClientID)
	form.Set("scope", tidalScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.BaseURL+"/device_authorization", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build device auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device authorization request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read device auth response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device authorization failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var dar DeviceAuthResponse
	if err := json.Unmarshal(body, &dar); err != nil {
		return nil, fmt.Errorf("failed to parse device auth response: %w", err)
	}
	return &dar, nil
}

// PollDeviceAuth polls the token endpoint for the device code.
// Returns (tokens, nil) on success, (nil, ErrAuthorizationPending) while the user
// has not yet confirmed, and (nil, ErrAuthorizationExpired) if the code expired.
func (a *AuthClient) PollDeviceAuth(ctx context.Context, deviceCode string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", a.ClientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	form.Set("scope", tidalScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.BaseURL+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build token poll request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(a.ClientID, a.ClientSecret)

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token poll request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	// 200 = success
	if resp.StatusCode == http.StatusOK {
		var tr TokenResponse
		if err := json.Unmarshal(body, &tr); err != nil {
			return nil, fmt.Errorf("failed to parse token response: %w", err)
		}
		return &tr, nil
	}

	// 400 = error (pending / expired / slow_down)
	var te TokenError
	if err := json.Unmarshal(body, &te); err != nil {
		return nil, fmt.Errorf("token poll failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	switch te.Error {
	case "authorization_pending":
		return nil, ErrAuthorizationPending
	case "expired_token":
		return nil, ErrAuthorizationExpired
	case "slow_down":
		return nil, ErrSlowDown
	default:
		return nil, fmt.Errorf("token poll error: %s (%s)", te.Error, te.ErrorDescription)
	}
}

// RefreshAccessToken exchanges a refresh token for a new access token.
func (a *AuthClient) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", a.ClientID)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")
	form.Set("scope", tidalScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.BaseURL+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(a.ClientID, a.ClientSecret)

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}
	return &tr, nil
}

// GetValidAccessToken returns a valid access token for the user, transparently
// refreshing it if it is within refreshSkew of expiry.
func (a *AuthClient) GetValidAccessToken(ctx context.Context, userID int) (string, error) {
	creds, err := GetCredentials(ctx, a.DB, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotConnected
		}
		return "", fmt.Errorf("failed to load tidal credentials: %w", err)
	}

	// If still valid, return as-is.
	if time.Now().Add(refreshSkew).Before(creds.AccessTokenExpiresAt) {
		return creds.AccessToken, nil
	}

	// Refresh.
	tr, err := a.RefreshAccessToken(ctx, creds.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to refresh tidal access token: %w", err)
	}

	// Persist the new tokens.
	creds.AccessToken = tr.AccessToken
	if tr.RefreshToken != "" {
		creds.RefreshToken = tr.RefreshToken
	}
	creds.AccessTokenExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	if err := UpsertCredentials(ctx, a.DB, creds); err != nil {
		return "", fmt.Errorf("failed to persist refreshed tidal token: %w", err)
	}
	return tr.AccessToken, nil
}
