package export

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"go-postgres-example/pkg/storage"
	"go-postgres-example/pkg/tidal"
)

// mockStorage implements storage.StorageService for tests.
type mockStorage struct {
	files map[string][]byte
}

func (m *mockStorage) UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	return nil
}
func (m *mockStorage) GetFileStream(ctx context.Context, objectName string) (storage.ReadSeekCloser, error) {
	b, ok := m.files[objectName]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &bytesReadSeekCloser{r: bytes.NewReader(b)}, nil
}
func (m *mockStorage) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	return nil, nil
}
func (m *mockStorage) DeleteObject(ctx context.Context, objectName string) error { return nil }

// bytesReadSeekCloser wraps a *bytes.Reader to satisfy storage.ReadSeekCloser.
type bytesReadSeekCloser struct{ r *bytes.Reader }
func (b *bytesReadSeekCloser) Read(p []byte) (int, error)  { return b.r.Read(p) }
func (b *bytesReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	return b.r.Seek(offset, whence)
}
func (b *bytesReadSeekCloser) Close() error { return nil }

// buildFLAC creates a minimal valid FLAC file: magic + STREAMINFO block (last) + dummy audio.
func buildFLAC() []byte {
	// "fLaC" magic
	data := []byte{'f', 'L', 'a', 'C'}
	// STREAMINFO block: header 0x80 (last, type 0), length 34, 34 zero bytes
	data = append(data, 0x80, 0x00, 0x00, 0x22)
	data = append(data, make([]byte, 34)...) // 34 bytes of STREAMINFO
	// Dummy audio frame
	data = append(data, 0xFF, 0xF8, 0x00, 0x00)
	return data
}

func TestExportOne_WritesFile(t *testing.T) {
	exportPath := t.TempDir()
	flacData := buildFLAC()

	ms := &mockStorage{files: map[string][]byte{"hash1.flac": flacData}}
	e := &Exporter{Storage: ms}

	cfg := tidal.DefaultConfig()
	cfg.ExportPath = exportPath
	cfg.ExportTemplate = "{artist}/{album}/{title}"
	cfg.ExportEmbedTags = true

	s := resolved{
		songID:   1,
		title:    "Song One",
		artist:   "Artist A",
		album:    "Album X",
		year:     2024,
		genre:    "Rock",
		filePath: "hash1.flac",
	}

	dest, err := e.exportOne(context.Background(), s, cfg)
	if err != nil {
		t.Fatalf("exportOne error: %v", err)
	}

	expected := filepath.Join(exportPath, "Artist A", "Album X", "Song One.flac")
	if dest != expected {
		t.Fatalf("dest = %q, want %q", dest, expected)
	}

	// Verify file exists.
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("expected file at %s: %v", dest, err)
	}

	// Verify it's still a valid FLAC (starts with fLaC).
	content, _ := os.ReadFile(dest)
	if len(content) < 4 || string(content[:4]) != "fLaC" {
		t.Fatalf("exported file is not a valid FLAC")
	}
}

func TestExportOne_SkipsExisting(t *testing.T) {
	exportPath := t.TempDir()
	flacData := buildFLAC()

	ms := &mockStorage{files: map[string][]byte{"hash1.flac": flacData}}
	e := &Exporter{Storage: ms}

	cfg := tidal.DefaultConfig()
	cfg.ExportPath = exportPath
	cfg.ExportTemplate = "{title}"
	cfg.ExportEmbedTags = false

	s := resolved{songID: 1, title: "Existing", filePath: "hash1.flac"}

	// First export.
	_, err := e.exportOne(context.Background(), s, cfg)
	if err != nil {
		t.Fatalf("first export error: %v", err)
	}

	// Second export should skip.
	_, err = e.exportOne(context.Background(), s, cfg)
	if err != errSkipped {
		t.Fatalf("expected errSkipped, got %v", err)
	}
}

func TestExportOne_PathTraversalContained(t *testing.T) {
	exportPath := t.TempDir()
	flacData := buildFLAC()

	ms := &mockStorage{files: map[string][]byte{"hash1.flac": flacData}}
	e := &Exporter{Storage: ms}

	cfg := tidal.DefaultConfig()
	cfg.ExportPath = exportPath
	cfg.ExportTemplate = "{title}"
	cfg.ExportEmbedTags = false

	// A title that tries to escape the export path.
	s := resolved{songID: 1, title: "../../etc/passwd", filePath: "hash1.flac"}

	dest, err := e.exportOne(context.Background(), s, cfg)
	if err != nil {
		t.Fatalf("exportOne error: %v", err)
	}
	// The destination must remain under the export path.
	absBase, _ := filepath.Abs(exportPath)
	absDest, _ := filepath.Abs(dest)
	if !strings.HasPrefix(absDest, absBase+string(os.PathSeparator)) {
		t.Fatalf("destination %q escaped export path %q", absDest, absBase)
	}
	// The rendered name must not contain a parent-directory reference.
	if strings.Contains(filepath.Base(dest), "..") {
		t.Fatalf("destination base %q contains '..'", filepath.Base(dest))
	}
}

func TestExportSongs_EmptyPath(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	e := &Exporter{DB: db, Storage: &mockStorage{}}
	cfg := tidal.DefaultConfig()
	cfg.ExportPath = "" // not set

	_, err = e.ExportSongs(context.Background(), "job1", 1, []int{1}, cfg)
	if err == nil {
		t.Fatalf("expected error for empty export path")
	}
}

func TestSanitizeSegment(t *testing.T) {
	if got := sanitizeSegment("../../etc"); strings.Contains(got, "..") {
		t.Fatalf("sanitizeSegment did not strip dots: %q", got)
	}
	if got := sanitizeSegment("a/b\\c"); strings.ContainsAny(got, "/\\") {
		t.Fatalf("sanitizeSegment did not strip separators: %q", got)
	}
	if got := sanitizeSegment(""); got != "unknown" {
		t.Fatalf("sanitizeSegment empty = %q, want 'unknown'", got)
	}
}

func TestRenderTemplate(t *testing.T) {
	s := resolved{artist: "A", album: "B", title: "C", year: 2024, genre: "G"}
	got := renderTemplate("{artist}/{album}/{title}", s)
	if got != "A/B/C" {
		t.Fatalf("renderTemplate = %q, want A/B/C", got)
	}
}
