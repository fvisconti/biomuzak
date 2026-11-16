package tidal

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ItemStatus is the per-track status within an import job.
type ItemStatus struct {
	TrackID int    `json:"track_id"`
	Title   string `json:"title"`
	Status  string `json:"status"` // pending | downloading | done | failed
	Error   string `json:"error,omitempty"`
}

// ImportJob tracks a batch import.
type ImportJob struct {
	ID     string       `json:"id"`
	UserID int          `json:"-"`
	Total  int          `json:"total"`
	Done   int          `json:"done"`
	Failed int          `json:"failed"`
	Status string       `json:"status"` // running | done
	Items  []ItemStatus `json:"items"`

	// finishedAt is set (under mu) when the job stops running; the store
	// sweeps entries older than importRetentionTTL.
	finishedAt time.Time

	mu sync.Mutex
}

// importRetentionTTL is how long a finished import job stays queryable before
// the store evicts it. Jobs are in-memory status snapshots; without eviction
// the map grows forever (one leaked entry per import).
const importRetentionTTL = 1 * time.Hour

// expired reports whether the job finished long enough ago to be evicted.
func (j *ImportJob) expired(now time.Time) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.Status != "running" && !j.finishedAt.IsZero() &&
		now.Sub(j.finishedAt) > importRetentionTTL
}

// jobStore holds in-flight import jobs (single-instance, in-memory).
type jobStore struct {
	mu   sync.Mutex
	jobs map[string]*ImportJob
}

var importJobs = &jobStore{jobs: make(map[string]*ImportJob)}

// sweepLocked evicts finished jobs older than the retention TTL. Callers
// must hold s.mu. Sweeping on add/get bounds the store size without a
// background ticker (and its shutdown path).
func (s *jobStore) sweepLocked(now time.Time) {
	for id, j := range s.jobs {
		if j.expired(now) {
			delete(s.jobs, id)
		}
	}
}

func (s *jobStore) add(job *ImportJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(time.Now())
	s.jobs[job.ID] = job
}

func (s *jobStore) get(id string) (*ImportJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.sweepLocked(now)
	j, ok := s.jobs[id]
	return j, ok && !j.expired(now)
}

// GetImportJob returns a job by ID.
func GetImportJob(id string) (*ImportJob, bool) {
	return importJobs.get(id)
}

// randomDelay returns a random duration in [min, max] seconds.
func randomDelay(minSec, maxSec int) time.Duration {
	if maxSec < minSec {
		maxSec = minSec
	}
	if maxSec <= 0 {
		return 0
	}
	span := maxSec - minSec
	var extra int
	if span > 0 {
		extra = rand.Intn(span + 1)
	}
	return time.Duration(minSec+extra) * time.Second
}

// RunBatch runs a batch import with a bounded worker pool and human-like pacing.
// It returns immediately after starting the workers; progress is tracked on the job.
func RunBatch(ctx context.Context, jobID string, userID int, tracks []TrackMeta, cfg TidalConfig, importTrack func(ctx context.Context, userID int, meta TrackMeta, cfg TidalConfig) (*ImportResult, error)) *ImportJob {
	job := &ImportJob{
		ID:     jobID,
		UserID: userID,
		Total:  len(tracks),
		Status: "running",
		Items:  make([]ItemStatus, len(tracks)),
	}
	for i, t := range tracks {
		job.Items[i] = ItemStatus{TrackID: t.TrackID, Title: t.Title, Status: "pending"}
	}
	importJobs.add(job)

	concurrency := cfg.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	// Buffered to the full track count so the feeder never blocks on send
	// while workers are still draining; this avoids a send/receive deadlock.
	work := make(chan int, len(tracks))
	var wg sync.WaitGroup

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range work {
				// Human-like pacing: random pause before each download.
				if d := randomDelay(cfg.DelayMinSec, cfg.DelayMaxSec); d > 0 {
					select {
					case <-time.After(d):
					case <-ctx.Done():
						return
					}
				}

				job.setItemStatus(idx, "downloading", "")
				_, err := importTrack(ctx, userID, tracks[idx], cfg)
				if err != nil {
					job.setItemStatus(idx, "failed", err.Error())
					job.incFailed()
				} else {
					job.setItemStatus(idx, "done", "")
					job.incDone()
				}
			}
		}()
	}

	go func() {
		for i := range tracks {
			select {
			case work <- i:
			case <-ctx.Done():
				return
			}
		}
		// Close before waiting so workers can drain and exit their range
		// loops; otherwise wg.Wait() would deadlock on the workers.
		close(work)
		wg.Wait()
		job.finish()
	}()

	return job
}

func (j *ImportJob) setItemStatus(idx int, status, errMsg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Items[idx].Status = status
	j.Items[idx].Error = errMsg
}

func (j *ImportJob) incDone() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Done++
}

func (j *ImportJob) incFailed() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Failed++
}

func (j *ImportJob) finish() {
	j.mu.Lock()
	j.Status = "done"
	j.finishedAt = time.Now()
	j.mu.Unlock()
}

// Snapshot returns a copy of the job for safe serialization.
func (j *ImportJob) Snapshot() ImportJob {
	j.mu.Lock()
	defer j.mu.Unlock()
	items := make([]ItemStatus, len(j.Items))
	copy(items, j.Items)
	return ImportJob{
		ID:     j.ID,
		Total:  j.Total,
		Done:   j.Done,
		Failed: j.Failed,
		Status: j.Status,
		Items:  items,
	}
}

// NewJobID generates a simple unique job id.
func NewJobID() string {
	return fmt.Sprintf("job-%d-%d", time.Now().UnixNano(), rand.Intn(1<<30))
}
