package models

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSongHidesInternalFields covers that server-internal fields are never
// serialized into API responses.
//
// Objective: file paths and content fingerprints must not be returned to any
// client — they leak the on-disk storage layout and the dedup key of the
// MinIO object, which no UI needs.
//
// Given-When-Then: Given a fully populated Song, When it is marshalled, Then
// neither "file_path" nor "fingerprint_hash" appears in the JSON; public
// fields (id, title) still do.
//
// Edge cases: LibrarySong carries the same sensitive fields and must hide
// them identically.
func TestSongHidesInternalFields(t *testing.T) {
	s := Song{ID: 1, Title: "T", Artist: "A", FilePath: "/data/x.flac", FingerprintHash: "deadbeef"}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal Song: %v", err)
	}
	out := string(b)
	for _, leak := range []string{"file_path", "/data/x.flac", "fingerprint_hash", "deadbeef"} {
		if strings.Contains(out, leak) {
			t.Errorf("Song JSON leaks %q: %s", leak, out)
		}
	}
	if !strings.Contains(out, `"title"`) {
		t.Errorf("Song JSON must keep public fields: %s", out)
	}

	ls := LibrarySong{ID: 1, Title: "T", FilePath: "/data/y.flac", FingerprintHash: "cafe"}
	b2, err := json.Marshal(ls)
	if err != nil {
		t.Fatalf("marshal LibrarySong: %v", err)
	}
	out2 := string(b2)
	for _, leak := range []string{"file_path", "/data/y.flac", "fingerprint_hash", "cafe"} {
		if strings.Contains(out2, leak) {
			t.Errorf("LibrarySong JSON leaks %q: %s", leak, out2)
		}
	}
	if !strings.Contains(out2, `"title"`) {
		t.Errorf("LibrarySong JSON must keep public fields: %s", out2)
	}
}
