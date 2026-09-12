package tidal

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunBatch_ConcurrencyAndCompletion(t *testing.T) {
	const total = 20
	const concurrency = 3

	var (
		mu           sync.Mutex
		current      int
		maxConcurrent int
	)

	importTrack := func(ctx context.Context, userID int, meta TrackMeta, cfg TidalConfig) (*ImportResult, error) {
		mu.Lock()
		current++
		if current > maxConcurrent {
			maxConcurrent = current
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond) // simulate work

		mu.Lock()
		current--
		mu.Unlock()

		return &ImportResult{SongID: meta.TrackID}, nil
	}

	tracks := make([]TrackMeta, total)
	for i := range tracks {
		tracks[i] = TrackMeta{TrackID: i + 1, Title: "t"}
	}

	cfg := DefaultConfig()
	cfg.Concurrency = concurrency
	cfg.DelayMinSec = 0
	cfg.DelayMaxSec = 0 // no pacing for this test

	job := RunBatch(context.Background(), "test-job", 1, tracks, cfg, importTrack)

	// Wait for completion.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		snap := job.Snapshot()
		if snap.Status == "done" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	snap := job.Snapshot()
	if snap.Status != "done" {
		t.Fatalf("job did not finish; status=%q", snap.Status)
	}
	if snap.Done+snap.Failed != total {
		t.Fatalf("done(%d)+failed(%d) != total(%d)", snap.Done, snap.Failed, total)
	}
	if snap.Done != total {
		t.Fatalf("expected all %d done, got %d (failed=%d)", total, snap.Done, snap.Failed)
	}
	if maxConcurrent > concurrency {
		t.Fatalf("max concurrent %d exceeded configured concurrency %d", maxConcurrent, concurrency)
	}
	if maxConcurrent < 1 {
		t.Fatalf("no concurrency observed")
	}
}

func TestRunBatch_PacingInsertsDelay(t *testing.T) {
	// With a 1s min delay and 2 tracks at concurrency 1, the job should take
	// at least ~2s (one pause per item).
	var calls int32
	importTrack := func(ctx context.Context, userID int, meta TrackMeta, cfg TidalConfig) (*ImportResult, error) {
		atomic.AddInt32(&calls, 1)
		return &ImportResult{SongID: meta.TrackID}, nil
	}

	tracks := []TrackMeta{{TrackID: 1}, {TrackID: 2}}
	cfg := DefaultConfig()
	cfg.Concurrency = 1
	cfg.DelayMinSec = 1
	cfg.DelayMaxSec = 1

	start := time.Now()
	job := RunBatch(context.Background(), "test-job-pacing", 1, tracks, cfg, importTrack)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if job.Snapshot().Status == "done" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	elapsed := time.Since(start)
	if job.Snapshot().Status != "done" {
		t.Fatalf("job did not finish")
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 import calls, got %d", calls)
	}
	if elapsed < 1800*time.Millisecond {
		t.Fatalf("expected pacing to add ~2s, but job finished in %v", elapsed)
	}
}

func TestRunBatch_FailureCounted(t *testing.T) {
	importTrack := func(ctx context.Context, userID int, meta TrackMeta, cfg TidalConfig) (*ImportResult, error) {
		if meta.TrackID == 2 {
			return nil, context.DeadlineExceeded
		}
		return &ImportResult{SongID: meta.TrackID}, nil
	}

	tracks := []TrackMeta{{TrackID: 1}, {TrackID: 2}}
	cfg := DefaultConfig()
	cfg.Concurrency = 2
	cfg.DelayMinSec = 0
	cfg.DelayMaxSec = 0

	job := RunBatch(context.Background(), "test-job-fail", 1, tracks, cfg, importTrack)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if job.Snapshot().Status == "done" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	snap := job.Snapshot()
	if snap.Done != 1 || snap.Failed != 1 {
		t.Fatalf("expected 1 done / 1 failed, got done=%d failed=%d", snap.Done, snap.Failed)
	}
}
