package services

import (
	"sync"
	"testing"
	"time"
)

// resetWarmupForTest puts the package-level state back to a known point.
func resetWarmupForTest() {
	predictionWarmup.mu.Lock()
	defer predictionWarmup.mu.Unlock()
	predictionWarmup.ready = false
	predictionWarmup.degraded = false
	predictionWarmup.reason = ""
	predictionWarmup.startedAt = time.Now()
	predictionWarmup.completedAt = time.Time{}
}

func TestPredictionsAreWithheldUntilWarm(t *testing.T) {
	resetWarmupForTest()

	if PredictionDataReady() {
		t.Fatal("predictions should be withheld before warm-up completes")
	}

	CompletePredictionWarmup(false, "")

	if !PredictionDataReady() {
		t.Fatal("predictions should be served once warm-up completes")
	}
	if GetPredictionWarmupStatus().Degraded {
		t.Fatal("a clean warm-up should not be reported as degraded")
	}
}

// TestFailedBackfillStillReleasesPredictions covers the deliberate trade-off:
// if the backfill fails we serve anyway rather than refusing forever, but we
// record that the data is degraded.
func TestFailedBackfillStillReleasesPredictions(t *testing.T) {
	resetWarmupForTest()

	CompletePredictionWarmup(true, "backfill failed: upstream 500")

	if !PredictionDataReady() {
		t.Fatal("a failed backfill must not block predictions indefinitely")
	}

	status := GetPredictionWarmupStatus()
	if !status.Degraded {
		t.Fatal("a failed backfill should be reported as degraded")
	}
	if status.Reason == "" {
		t.Fatal("a degraded warm-up should explain itself")
	}
}

// TestWarmupCompletionIsIdempotent guards against a later call downgrading an
// already-good state, since the safety valve and the backfill can both fire.
func TestWarmupCompletionIsIdempotent(t *testing.T) {
	resetWarmupForTest()

	CompletePredictionWarmup(false, "")
	CompletePredictionWarmup(true, "late safety valve")

	if GetPredictionWarmupStatus().Degraded {
		t.Fatal("a clean warm-up was downgraded by a later call")
	}
}

func TestWarmupStateIsRaceFree(t *testing.T) {
	resetWarmupForTest()

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = PredictionDataReady()
				_ = GetPredictionWarmupStatus()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		CompletePredictionWarmup(false, "")
	}()
	wg.Wait()

	if !PredictionDataReady() {
		t.Fatal("warm-up should have completed")
	}
}
