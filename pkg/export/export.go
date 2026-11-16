package export

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/storage"
	"go-postgres-example/pkg/tidal"
)

// resolved is a song resolved from the DB, ready for export.
type resolved struct {
	songID   int
	title    string
	artist   string
	album    string
	year     int
	genre    string
	filePath string
}

// Exporter writes a copy of library songs to a local folder on the server.
type Exporter struct {
	DB          *sql.DB
	Storage     storage.StorageService
	Concurrency int // max parallel export workers (EXPORT_CONCURRENCY, default 4)
}

// NewExporter builds an Exporter.
func NewExporter(db *sql.DB, s storage.StorageService, concurrency int) *Exporter {
	if concurrency < 1 {
		concurrency = 4
	}
	return &Exporter{DB: db, Storage: s, Concurrency: concurrency}
}

// ItemStatus is the per-song status within an export job.
type ItemStatus struct {
	SongID int    `json:"song_id"`
	Title  string `json:"title"`
	Status string `json:"status"` // pending | writing | done | skipped | failed
	Path   string `json:"path,omitempty"`
	Error  string `json:"error,omitempty"`
}

// exportRetentionTTL is how long a finished export job stays queryable before
// the store evicts it. Jobs are in-memory status snapshots; without eviction
// the map grows forever (one leaked entry per export).
const exportRetentionTTL = 1 * time.Hour

// ExportJob tracks a batch export.
type ExportJob struct {
	ID string `json:"id"`
	// UserID owns the job for status authorization (never serialized).
	UserID  int          `json:"-"`
	Total   int          `json:"total"`
	Done    int          `json:"done"`
	Skipped int          `json:"skipped"`
	Failed  int          `json:"failed"`
	Status  string       `json:"status"` // running | done
	Items   []ItemStatus `json:"items"`

	// finishedAt is set (under mu) when the job stops running; the store
	// sweeps entries older than exportRetentionTTL.
	finishedAt time.Time

	mu sync.Mutex
}

// expired reports whether the job finished long enough ago to be evicted.
func (j *ExportJob) expired(now time.Time) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.Status != "running" && !j.finishedAt.IsZero() &&
		now.Sub(j.finishedAt) > exportRetentionTTL
}

type exportJobStore struct {
	mu   sync.Mutex
	jobs map[string]*ExportJob
}

var exportJobs = &exportJobStore{jobs: make(map[string]*ExportJob)}

// sweepLocked evicts finished jobs older than the retention TTL. Callers
// must hold s.mu. Sweeping on add/get bounds the store size without a
// background ticker (and its shutdown path).
func (s *exportJobStore) sweepLocked(now time.Time) {
	for id, j := range s.jobs {
		if j.expired(now) {
			delete(s.jobs, id)
		}
	}
}

// add stores a job after sweeping expired entries.
func (s *exportJobStore) add(job *ExportJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(time.Now())
	s.jobs[job.ID] = job
}

// GetExportJob returns a job by ID; a job past its retention TTL is gone.
func GetExportJob(id string) (*ExportJob, bool) {
	exportJobs.mu.Lock()
	defer exportJobs.mu.Unlock()
	now := time.Now()
	exportJobs.sweepLocked(now)
	j, ok := exportJobs.jobs[id]
	return j, ok && !j.expired(now)
}

// ExportSongs writes the given songs to cfg.ExportPath using cfg.ExportTemplate.
// It returns the job immediately; progress is tracked on the job.
// ExportSongs exports the given songs to disk. The userID parameter is retained
// for API stability (callers pass it) but is no longer used for scoping: the
// catalog is shared, so any song can be exported by any authenticated user.
func (e *Exporter) ExportSongs(ctx context.Context, jobID string, userID int, songIDs []int, cfg tidal.TidalConfig) (*ExportJob, error) {
	if cfg.ExportPath == "" {
		return nil, fmt.Errorf("export_path is not set; configure it in Tidal settings")
	}
	if !filepath.IsAbs(cfg.ExportPath) {
		return nil, fmt.Errorf("export_path must be an absolute path")
	}
	if cfg.ExportTemplate == "" {
		cfg.ExportTemplate = "{artist}/{album}/{title}"
	}

	// Resolve songs up front in a single batched query (avoids N+1).
	// The catalog is shared, so any song can be exported.
	songMap, err := db.GetSongsByIDs(e.DB, songIDs)
	if err != nil {
		return nil, fmt.Errorf("resolve songs: %w", err)
	}
	resolvedSongs := make([]resolved, 0, len(songIDs))
	for _, sid := range songIDs {
		song, ok := songMap[sid]
		if !ok {
			return nil, fmt.Errorf("song %d not found", sid)
		}
		resolvedSongs = append(resolvedSongs, resolved{
			songID:   song.ID,
			title:    song.Title,
			artist:   song.Artist,
			album:    song.Album,
			year:     song.Year,
			genre:    song.Genre,
			filePath: song.FilePath,
		})
	}

	job := &ExportJob{
		ID:     jobID,
		UserID: userID,
		Total:  len(resolvedSongs),
		Status: "running",
		Items:  make([]ItemStatus, len(resolvedSongs)),
	}
	for i, s := range resolvedSongs {
		job.Items[i] = ItemStatus{SongID: s.songID, Title: s.title, Status: "pending"}
	}
	exportJobs.add(job)

	// Bounded worker pool: songs are independent, so fan out to N workers.
	// Concurrency is controlled by EXPORT_CONCURRENCY (default 4).
	concurrency := e.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(resolvedSongs) {
		concurrency = len(resolvedSongs)
	}

	indices := make(chan int, len(resolvedSongs))
	for i := range resolvedSongs {
		indices <- i
	}
	close(indices)

	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range indices {
				if ctx.Err() != nil {
					return
				}
				s := resolvedSongs[i]
				job.setItem(i, "writing", "", "")
				dest, err := e.exportOne(ctx, s, cfg)
				switch {
				case err == errSkipped:
					job.setItem(i, "skipped", dest, "")
					job.incSkipped()
				case err != nil:
					job.setItem(i, "failed", "", err.Error())
					job.incFailed()
				default:
					job.setItem(i, "done", dest, "")
					job.incDone()
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		job.finish()
	}()

	return job, nil
}

var errSkipped = fmt.Errorf("skipped: file already exists")

// exportOne writes a single song to the export path.
func (e *Exporter) exportOne(ctx context.Context, s resolved, cfg tidal.TidalConfig) (string, error) {
	// Build the destination path from the template, appending the source
	// file's extension so the exported file keeps its format.
	rel := renderTemplate(cfg.ExportTemplate, s)
	if ext := filepath.Ext(s.filePath); ext != "" {
		rel += ext
	}
	dest := filepath.Join(cfg.ExportPath, rel)

	// Security: ensure the final path stays under ExportPath.
	cleanBase, err := filepath.Abs(cfg.ExportPath)
	if err != nil {
		return "", fmt.Errorf("invalid export path: %w", err)
	}
	cleanDest, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("invalid destination: %w", err)
	}
	if !strings.HasPrefix(cleanDest, cleanBase+string(os.PathSeparator)) {
		return "", fmt.Errorf("destination escapes export path (rejected)")
	}

	// Skip if it already exists.
	if _, err := os.Stat(cleanDest); err == nil {
		return cleanDest, errSkipped
	}

	// Read from storage.
	stream, err := e.Storage.GetFileStream(ctx, s.filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read from storage: %w", err)
	}
	defer stream.Close()

	// Ensure parent dir.
	if err := os.MkdirAll(filepath.Dir(cleanDest), 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(cleanDest)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, stream); err != nil {
		out.Close()
		os.Remove(cleanDest)
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	out.Close()

	// Embed tags if requested.
	if cfg.ExportEmbedTags {
		if err := embedTags(cleanDest, s); err != nil {
			// Non-fatal: the file is still a valid copy.
			_ = err
		}
	}

	return cleanDest, nil
}

// embedTags writes title/artist/album/year/genre into the file.
// Only FLAC files are supported (the lossless import path); other formats
// are skipped (the file is still a valid copy).
func embedTags(path string, s resolved) error {
	if !strings.EqualFold(filepath.Ext(path), ".flac") {
		return nil // no tag writer for this format; skip
	}
	return embedFLACTags(path, s.title, s.artist, s.album, s.year, s.genre)
}

// renderTemplate substitutes {artist}, {album}, {title}, {year}, {genre}.
func renderTemplate(tpl string, s resolved) string {
	repl := strings.NewReplacer(
		"{artist}", sanitizeSegment(s.artist),
		"{album}", sanitizeSegment(s.album),
		"{title}", sanitizeSegment(s.title),
		"{year}", fmt.Sprintf("%d", s.year),
		"{genre}", sanitizeSegment(s.genre),
	)
	return repl.Replace(tpl)
}

// sanitizeSegment strips path separators and control characters from a
// template segment to prevent path traversal. Any '/' or '\' is removed
// entirely so a segment can never introduce a new directory level.
func sanitizeSegment(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '/' || r == '\\' || r == 0x00:
			// drop path separators / NUL
		case r < 0x20 || r == 0x7f:
			// drop control characters
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	// Collapse any ".." that could remain (e.g. from "a..b" is fine, but
	// "...." -> "..") to be safe.
	for strings.Contains(out, "..") {
		out = strings.ReplaceAll(out, "..", ".")
	}
	if out == "" || out == "." {
		return "unknown"
	}
	return out
}

func (j *ExportJob) setItem(idx int, status, path, errMsg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Items[idx].Status = status
	j.Items[idx].Path = path
	j.Items[idx].Error = errMsg
}

func (j *ExportJob) incDone() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Done++
}

func (j *ExportJob) incSkipped() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Skipped++
}

func (j *ExportJob) incFailed() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Failed++
}

func (j *ExportJob) finish() {
	j.mu.Lock()
	j.Status = "done"
	j.finishedAt = time.Now()
	j.mu.Unlock()
}

// Snapshot returns a copy of the job for safe serialization.
func (j *ExportJob) Snapshot() ExportJob {
	j.mu.Lock()
	defer j.mu.Unlock()
	items := make([]ItemStatus, len(j.Items))
	copy(items, j.Items)
	return ExportJob{
		ID:      j.ID,
		UserID:  j.UserID,
		Total:   j.Total,
		Done:    j.Done,
		Skipped: j.Skipped,
		Failed:  j.Failed,
		Status:  j.Status,
		Items:   items,
	}
}

// newExportJobID generates a simple unique job id.
func newExportJobID() string {
	return fmt.Sprintf("export-%d-%d", time.Now().UnixNano(), len(exportJobs.jobs))
}
