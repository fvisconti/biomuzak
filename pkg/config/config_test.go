package config

import (
	"os"
	"testing"
)

// TestSimilarSongsLimit covers how SIMILAR_SONGS_LIMIT is resolved by New().
//
// Objective: the default comes from DefaultSimilarSongsLimit; a valid env
// value is honoured; an invalid or empty value falls back to the default.
//
// Given-When-Then: Given SIMILAR_SONGS_LIMIT is set to the test-case value,
// When New() is called, Then SimilarSongsLimit equals the expected result.
//
// Edge cases: an empty string is treated as absent (Sscanf fails inside
// getEnvInt, so the default is returned); a non-numeric string likewise
// falls back. We always set the variable with t.Setenv to avoid polluting
// the process environment across test runs.
func TestSimilarSongsLimit(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string // value passed to t.Setenv
		expected int
	}{
		{
			name:     "empty string falls back to default",
			envVal:   "",
			expected: DefaultSimilarSongsLimit,
		},
		{
			name:     "valid integer from env",
			envVal:   "7",
			expected: 7,
		},
		{
			name:     "non-numeric falls back to default",
			envVal:   "not-a-number",
			expected: DefaultSimilarSongsLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SIMILAR_SONGS_LIMIT", tt.envVal)
			cfg := New()
			if cfg.SimilarSongsLimit != tt.expected {
				t.Errorf(
					"SimilarSongsLimit = %d, want %d (env=%q)",
					cfg.SimilarSongsLimit, tt.expected, tt.envVal,
				)
			}
		})
	}
}

// TestMinIOCredentialsHaveNoDefaults covers how MINIO_ACCESS_KEY and
// MINIO_SECRET_KEY are resolved by New().
//
// Objective: object-storage credentials are secrets (directive 3) and must
// have no insecure default — an absent variable yields an empty string, which
// cmd/server/main.go rejects at boot (fail fast). The log.Fatal gate itself
// runs before any test could observe it, so this pins the config half.
//
// Given-When-Then: Given the credential env vars, When New() is called, Then
// the values pass through unchanged and unset variables resolve to "".
//
// Edge cases: an explicitly empty value stays empty; the well-known default
// "minioadmin" is never injected.
func TestMinIOCredentialsHaveNoDefaults(t *testing.T) {
	tests := []struct {
		name       string
		unset      bool
		accessEnv  string
		secretEnv  string
		wantAccess string
		wantSecret string
	}{
		{
			name:       "unset yields empty (main.go fails fast)",
			unset:      true,
			wantAccess: "",
			wantSecret: "",
		},
		{
			name:       "values pass through",
			accessEnv:  "ak",
			secretEnv:  "sk",
			wantAccess: "ak",
			wantSecret: "sk",
		},
		{
			name:       "empty is not upgraded to minioadmin",
			accessEnv:  "",
			secretEnv:  "sk",
			wantAccess: "",
			wantSecret: "sk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unset {
				// t.Setenv registers a cleanup that restores the prior value;
				// os.Unsetenv then removes it for the test body so getEnv
				// exercises its default (the real regression path).
				t.Setenv("MINIO_ACCESS_KEY", "")
				t.Setenv("MINIO_SECRET_KEY", "")
				os.Unsetenv("MINIO_ACCESS_KEY")
				os.Unsetenv("MINIO_SECRET_KEY")
			} else {
				t.Setenv("MINIO_ACCESS_KEY", tt.accessEnv)
				t.Setenv("MINIO_SECRET_KEY", tt.secretEnv)
			}
			cfg := New()
			if cfg.MinIOAccessKey != tt.wantAccess {
				t.Errorf("MinIOAccessKey = %q, want %q", cfg.MinIOAccessKey, tt.wantAccess)
			}
			if cfg.MinIOSecretKey != tt.wantSecret {
				t.Errorf("MinIOSecretKey = %q, want %q", cfg.MinIOSecretKey, tt.wantSecret)
			}
		})
	}
}
