package tidal

import (
	"fmt"
	"strings"
)

// TidalConfig holds per-user Tidal import/export settings, stored as JSONB in
// tidal_credentials.config. Defaults are applied by DefaultConfig().
type TidalConfig struct {
	// Quality: "low" | "normal" | "high" | "max"
	// Maps to Tidal audioQualityId: LOW, NORMAL, LOSSLESS, HIRES.
	Quality string `json:"quality"`

	// Concurrency: number of parallel downloads (1-8).
	Concurrency int `json:"concurrency"`

	// DelayMinSec / DelayMaxSec: random pause range (seconds) between downloads
	// to mimic human behaviour and avoid rate-limiting.
	DelayMinSec int `json:"delay_min_sec"`
	DelayMaxSec int `json:"delay_max_sec"`

	// SinglesFilter: "none" | "only" | "include" (artist import).
	SinglesFilter string `json:"singles_filter"`

	// VideosFilter: "none" | "allow" (biomuzak is audio-only).
	VideosFilter string `json:"videos_filter"`

	// AtmosFilter: "none" | "only" | "allow" (Dolby Atmos).
	AtmosFilter string `json:"atmos_filter"`

	// Export settings (manual "Export to local folder" feature).
	ExportPath       string `json:"export_path"`
	ExportTemplate   string `json:"export_template"`
	ExportEmbedTags  bool   `json:"export_embed_tags"`
}

// DefaultConfig returns a TidalConfig with sensible defaults.
func DefaultConfig() TidalConfig {
	return TidalConfig{
		Quality:         "high",
		Concurrency:     4,
		DelayMinSec:     2,
		DelayMaxSec:     8,
		SinglesFilter:   "include",
		VideosFilter:    "none",
		AtmosFilter:     "none",
		ExportPath:      "",
		ExportTemplate:  "{artist}/{album}/{title}",
		ExportEmbedTags: true,
	}
}

// qualityToAudioQualityID maps the user-facing quality to Tidal's audioQualityId.
func (c TidalConfig) qualityToAudioQualityID() string {
	switch c.Quality {
	case "low":
		return "LOW"
	case "normal":
		return "NORMAL"
	case "max":
		return "HIRES"
	case "high":
		fallthrough
	default:
		return "LOSSLESS"
	}
}

// fileExt returns the expected file extension for the configured quality.
func (c TidalConfig) fileExt() string {
	switch c.Quality {
	case "low", "normal":
		return ".m4a"
	default:
		return ".flac"
	}
}

// Validate checks the config and returns a sanitized copy with clamped values.
func (c TidalConfig) Validate() (TidalConfig, error) {
	// Quality
	switch c.Quality {
	case "low", "normal", "high", "max":
	default:
		return c, fmt.Errorf("invalid quality %q (must be low, normal, high, or max)", c.Quality)
	}

	// Concurrency: clamp 1-8
	if c.Concurrency < 1 {
		c.Concurrency = 1
	}
	if c.Concurrency > 8 {
		c.Concurrency = 8
	}

	// Delay: ensure min <= max, both >= 0
	if c.DelayMinSec < 0 {
		c.DelayMinSec = 0
	}
	if c.DelayMaxSec < c.DelayMinSec {
		c.DelayMaxSec = c.DelayMinSec
	}

	// Singles filter
	switch c.SinglesFilter {
	case "none", "only", "include":
	default:
		return c, fmt.Errorf("invalid singles_filter %q", c.SinglesFilter)
	}

	// Videos filter
	switch c.VideosFilter {
	case "none", "allow":
	default:
		return c, fmt.Errorf("invalid videos_filter %q", c.VideosFilter)
	}

	// Atmos filter
	switch c.AtmosFilter {
	case "none", "only", "allow":
	default:
		return c, fmt.Errorf("invalid atmos_filter %q", c.AtmosFilter)
	}

	// Export path: if set, must be absolute
	if c.ExportPath != "" && !strings.HasPrefix(c.ExportPath, "/") {
		return c, fmt.Errorf("export_path must be an absolute path (start with /)")
	}

	// Export template: if set, must be non-empty
	if c.ExportTemplate != "" && strings.TrimSpace(c.ExportTemplate) == "" {
		return c, fmt.Errorf("export_template must not be empty")
	}

	return c, nil
}
