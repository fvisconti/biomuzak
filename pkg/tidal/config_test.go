package tidal

import "testing"

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.Quality != "high" {
		t.Errorf("default quality = %q, want high", c.Quality)
	}
	if c.Concurrency != 4 {
		t.Errorf("default concurrency = %d, want 4", c.Concurrency)
	}
	if c.DelayMinSec != 2 || c.DelayMaxSec != 8 {
		t.Errorf("default delay = %d/%d, want 2/8", c.DelayMinSec, c.DelayMaxSec)
	}
	if c.ExportTemplate != "{artist}/{album}/{title}" {
		t.Errorf("default export template = %q", c.ExportTemplate)
	}
}

func TestValidate_ClampsConcurrency(t *testing.T) {
	c := DefaultConfig()
	c.Concurrency = 100
	v, err := c.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Concurrency != 8 {
		t.Errorf("concurrency = %d, want clamped to 8", v.Concurrency)
	}

	c.Concurrency = 0
	v, err = c.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Concurrency != 1 {
		t.Errorf("concurrency = %d, want clamped to 1", v.Concurrency)
	}
}

func TestValidate_DelayOrdering(t *testing.T) {
	c := DefaultConfig()
	c.DelayMinSec = 10
	c.DelayMaxSec = 2
	v, err := c.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.DelayMaxSec < v.DelayMinSec {
		t.Errorf("delay max %d < min %d after validation", v.DelayMaxSec, v.DelayMinSec)
	}
}

func TestValidate_InvalidQuality(t *testing.T) {
	c := DefaultConfig()
	c.Quality = "ultra"
	if _, err := c.Validate(); err == nil {
		t.Fatalf("expected error for invalid quality")
	}
}

func TestValidate_ExportPathMustBeAbsolute(t *testing.T) {
	c := DefaultConfig()
	c.ExportPath = "relative/path"
	if _, err := c.Validate(); err == nil {
		t.Fatalf("expected error for relative export path")
	}
	c.ExportPath = "/mnt/external/Music"
	if _, err := c.Validate(); err != nil {
		t.Fatalf("unexpected error for absolute export path: %v", err)
	}
}

func TestQualityToAudioQualityID(t *testing.T) {
	cases := map[string]string{
		"low":    "LOW",
		"normal": "NORMAL",
		"high":   "LOSSLESS",
		"max":    "HIRES",
	}
	for q, want := range cases {
		c := DefaultConfig()
		c.Quality = q
		if got := c.qualityToAudioQualityID(); got != want {
			t.Errorf("quality %q -> %q, want %q", q, got, want)
		}
	}
}

func TestFileExt(t *testing.T) {
	c := DefaultConfig()
	c.Quality = "high"
	if got := c.fileExt(); got != ".flac" {
		t.Errorf("high ext = %q, want .flac", got)
	}
	c.Quality = "normal"
	if got := c.fileExt(); got != ".m4a" {
		t.Errorf("normal ext = %q, want .m4a", got)
	}
}
