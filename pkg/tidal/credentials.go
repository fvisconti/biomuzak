package tidal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Credentials holds a user's Tidal OAuth tokens + config.
type Credentials struct {
	UserID                int
	TidalUserID           int64
	TidalUsername         string
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	Config                TidalConfig
}

// UpsertCredentials inserts or updates a user's Tidal credentials.
func UpsertCredentials(ctx context.Context, db *sql.DB, c *Credentials) error {
	// Serialize config to JSONB
	cfgJSON, err := json.Marshal(c.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal tidal config: %w", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO tidal_credentials (
			user_id, tidal_user_id, tidal_username,
			access_token, refresh_token, access_token_expires_at, config
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			tidal_user_id = EXCLUDED.tidal_user_id,
			tidal_username = EXCLUDED.tidal_username,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			access_token_expires_at = EXCLUDED.access_token_expires_at,
			config = EXCLUDED.config,
			updated_at = NOW()
	`, c.UserID, c.TidalUserID, c.TidalUsername,
		c.AccessToken, c.RefreshToken, c.AccessTokenExpiresAt, cfgJSON)
	if err != nil {
		return fmt.Errorf("failed to upsert tidal credentials: %w", err)
	}
	return nil
}

// GetCredentials retrieves a user's Tidal credentials.
// Returns sql.ErrNoRows if the user has no Tidal connection.
func GetCredentials(ctx context.Context, db *sql.DB, userID int) (*Credentials, error) {
	var c Credentials
	var cfgJSON []byte
	err := db.QueryRowContext(ctx, `
		SELECT user_id, COALESCE(tidal_user_id, 0), COALESCE(tidal_username, ''),
			access_token, refresh_token, access_token_expires_at, config
		FROM tidal_credentials WHERE user_id = $1
	`, userID).Scan(&c.UserID, &c.TidalUserID, &c.TidalUsername,
		&c.AccessToken, &c.RefreshToken, &c.AccessTokenExpiresAt, &cfgJSON)
	if err != nil {
		return nil, err
	}

	// Parse config; fall back to defaults if empty/invalid
	if len(cfgJSON) > 0 {
		if err := json.Unmarshal(cfgJSON, &c.Config); err != nil {
			c.Config = DefaultConfig()
		}
	} else {
		c.Config = DefaultConfig()
	}
	return &c, nil
}

// DeleteCredentials removes a user's Tidal connection.
func DeleteCredentials(ctx context.Context, db *sql.DB, userID int) error {
	_, err := db.ExecContext(ctx, `DELETE FROM tidal_credentials WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete tidal credentials: %w", err)
	}
	return nil
}

// SaveConfig persists a user's TidalConfig (JSONB).
func SaveConfig(ctx context.Context, db *sql.DB, userID int, cfg TidalConfig) error {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal tidal config: %w", err)
	}
	_, err = db.ExecContext(ctx, `
		UPDATE tidal_credentials SET config = $1, updated_at = NOW() WHERE user_id = $2
	`, cfgJSON, userID)
	if err != nil {
		return fmt.Errorf("failed to save tidal config: %w", err)
	}
	return nil
}

// GetConfig retrieves a user's TidalConfig. Returns defaults if no row exists.
func GetConfig(ctx context.Context, db *sql.DB, userID int) (TidalConfig, error) {
	var cfgJSON []byte
	err := db.QueryRowContext(ctx, `
		SELECT config FROM tidal_credentials WHERE user_id = $1
	`, userID).Scan(&cfgJSON)
	if err == sql.ErrNoRows {
		return DefaultConfig(), nil
	}
	if err != nil {
		return DefaultConfig(), fmt.Errorf("failed to get tidal config: %w", err)
	}
	var cfg TidalConfig
	if len(cfgJSON) > 0 {
		if err := json.Unmarshal(cfgJSON, &cfg); err != nil {
			return DefaultConfig(), nil
		}
	}
	return cfg, nil
}
