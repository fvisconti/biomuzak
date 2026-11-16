package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"go-postgres-example/pkg/config"
	"go-postgres-example/pkg/db"
	"go-postgres-example/pkg/metadata"
	"go-postgres-example/pkg/middleware"
	"go-postgres-example/pkg/storage"
)

// backfillRetentionTTL is how long a finished backfill job stays queryable
// before the store evicts it. Jobs are in-memory and only useful while an
// admin is polling status; without eviction the map grows forever (every
// backfill leaks one entry for the process lifetime).
const backfillRetentionTTL = 1 * time.Hour

// BackfillJob tracks an in-flight embedding backfill.
type BackfillJob struct {
	ID     string `json:"id"`
	Total  int    `json:"total"`
	Done   int    `json:"done"`
	Failed int    `json:"failed"`
	Status string `json:"status"` // running | done | cancelled

	// finishedAt is set (under mu) when the job stops running; the store
	// sweeps entries whose finishedAt is older than backfillRetentionTTL.
	finishedAt time.Time
	// cancel stops the worker goroutine; set before the store sees the job.
	cancel context.CancelFunc

	mu sync.Mutex
}

func (j *BackfillJob) incDone() {
	j.mu.Lock()
	j.Done++
	j.mu.Unlock()
}

func (j *BackfillJob) incFailed() {
	j.mu.Lock()
	j.Failed++
	j.mu.Unlock()
}

// finish marks the job stopped (done or cancelled) and records finishedAt so
// the retention sweep can evict it after the TTL.
func (j *BackfillJob) finish(status string) {
	j.mu.Lock()
	j.Status = status
	j.finishedAt = time.Now()
	j.mu.Unlock()
}

func (j *BackfillJob) isRunning() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.Status == "running"
}

// expired reports whether the job finished long enough ago to be evicted.
func (j *BackfillJob) expired(now time.Time) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.Status != "running" && !j.finishedAt.IsZero() &&
		now.Sub(j.finishedAt) > backfillRetentionTTL
}

// Snapshot returns a copy of the job for safe serialization.
func (j *BackfillJob) Snapshot() BackfillJob {
	j.mu.Lock()
	defer j.mu.Unlock()
	return BackfillJob{ID: j.ID, Total: j.Total, Done: j.Done, Failed: j.Failed, Status: j.Status}
}

// backfillStore holds in-flight backfill jobs (single-instance, in-memory).
type backfillStore struct {
	mu   sync.Mutex
	jobs map[string]*BackfillJob
}

var backfillJobs = &backfillStore{jobs: make(map[string]*BackfillJob)}

// sweepLocked evicts finished jobs older than the retention TTL. Callers
// must hold s.mu. Sweeping on every add/get bounds the store size without
// needing a background ticker (and its shutdown path).
func (s *backfillStore) sweepLocked(now time.Time) {
	for id, j := range s.jobs {
		if j.expired(now) {
			delete(s.jobs, id)
		}
	}
}

// add inserts a job unless another job is still running (single-flight), so
// two admins POSTing at once cannot double-process the library. The boolean
// result reports whether the job was admitted.
func (s *backfillStore) add(job *BackfillJob) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(time.Now())
	for _, j := range s.jobs {
		if j.isRunning() {
			return false
		}
	}
	s.jobs[job.ID] = job
	return true
}

// get returns a job by ID; a job past its retention TTL is treated as gone.
func (s *backfillStore) get(id string) (*BackfillJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.sweepLocked(now)
	j, ok := s.jobs[id]
	return j, ok && !j.expired(now)
}

// hasRunning reports whether a backfill job is currently running. Used by
// the start handler to fail fast (409) without touching the DB; the store's
// add() still enforces single-flight under the same lock.
func (s *backfillStore) hasRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		if j.isRunning() {
			return true
		}
	}
	return false
}

// cancelRunning stops the running backfill job, if any (server shutdown).
func (s *backfillStore) cancelRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		if j.isRunning() && j.cancel != nil {
			j.cancel()
		}
	}
}

// BackfillHandler handles the admin embedding backfill job.
type BackfillHandler struct {
	DB      *sql.DB
	Cfg     *config.Config
	Storage storage.StorageService
}

// NewBackfillHandler builds a BackfillHandler.
func NewBackfillHandler(db *sql.DB, cfg *config.Config, s storage.StorageService) *BackfillHandler {
	return &BackfillHandler{DB: db, Cfg: cfg, Storage: s}
}

// StartBackfillHandler kicks off a background job that generates embeddings
// for every song that lacks one. Admin-only, single-flight: a second start
// while a job is running returns 409. Returns a job ID immediately.
func (h *BackfillHandler) StartBackfillHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.GetUserIDFromContext(r.Context()); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Single-flight, checked before any DB work: a second POST while a job is
	// running must return 409 without re-querying the library.
	if backfillJobs.hasRunning() {
		http.Error(w, "A backfill job is already running", http.StatusConflict)
		return
	}

	// Build the processor before reserving a job slot: NewProcessor can fail
	// and must never reach log.Fatalf from a request path (directive: check
	// every error, return a sanitized message).
	processor, err := metadata.NewProcessor(h.DB, h.Cfg, h.Storage)
	if err != nil {
		log.Printf("backfill: failed to create metadata processor: %v", err)
		http.Error(w, "Failed to start backfill", http.StatusInternalServerError)
		return
	}

	ids, err := db.GetSongsMissingEmbeddings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "Failed to find songs missing embeddings", http.StatusInternalServerError)
		return
	}

	// Cancellable context so the job has a clear exit path (server shutdown)
	// instead of running detached with context.Background() forever.
	ctx, cancel := context.WithCancel(context.Background())
	job := &BackfillJob{ID: newBackfillJobID(), Total: len(ids), Status: "running", cancel: cancel}
	if !backfillJobs.add(job) {
		cancel()
		http.Error(w, "A backfill job is already running", http.StatusConflict)
		return
	}

	// Detached context so the job outlives the HTTP request.
	go func() {
		defer cancel()
		for _, id := range ids {
			if ctx.Err() != nil {
				break
			}
			if err := processor.GenerateEmbeddingForSong(ctx, id); err != nil {
				job.incFailed()
				continue
			}
			job.incDone()
		}
		if ctx.Err() != nil {
			job.finish("cancelled")
			return
		}
		job.finish("done")
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"jobId": job.ID})
}

// BackfillStatusHandler returns the status of a backfill job. Admin-only.
func (h *BackfillHandler) BackfillStatusHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.GetUserIDFromContext(r.Context()); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	jobID := r.URL.Query().Get("jobId")
	job, exists := backfillJobs.get(jobID)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, job.Snapshot())
}

// CancelBackfill stops the running backfill job, if any. Called on server
// shutdown so the worker goroutine gets a clear exit path.
func CancelBackfill() { backfillJobs.cancelRunning() }

// ResetBackfillJobs clears the in-memory job store. Intended for tests to
// isolate the package-level store between runs.
func ResetBackfillJobs() {
	backfillJobs.mu.Lock()
	defer backfillJobs.mu.Unlock()
	backfillJobs.jobs = make(map[string]*BackfillJob)
}

var backfillJobSeq atomic.Int64

func newBackfillJobID() string {
	return fmt.Sprintf("backfill-%d-%d", time.Now().UnixNano(), backfillJobSeq.Add(1))
}
