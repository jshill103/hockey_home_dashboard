package services

import (
	"log"
	"sync"
	"time"
)

// Prediction data warm-up.
//
// On boot the process kicks off a league-wide play-by-play backfill in a
// background goroutine so the HTTP server can start immediately. Until that
// finishes the models are working from whatever partial history happens to
// have loaded, and predictions made in that window are not merely less
// precise, they can be wrong in ways the finished data flatly contradicts.
//
// A real example: for the same NYR/UTA matchup, a prediction served at
// 17:10:40 had Poisson expected goals of NYR 4.08 to UTA 1.69 at 83.8%
// confidence. The backfill completed at 17:12:43. Eighteen seconds after
// that first run the same model, on the same matchup, reported NYR 3.08 to
// UTA 4.07 -- the opposite result. The early prediction was then cached and
// displayed as if it were sound.
//
// Serving "not ready yet" is better than serving a confident wrong answer.

// warmupSafetyValve bounds how long predictions can be withheld. If the
// backfill wedges or an upstream API is down we would otherwise refuse to
// predict forever, which is a worse failure than predicting from partial
// data. Comfortably exceeds the 10-15 minutes the backfill normally takes.
const warmupSafetyValve = 25 * time.Minute

type warmupState struct {
	mu          sync.RWMutex
	ready       bool
	degraded    bool
	reason      string
	startedAt   time.Time
	completedAt time.Time
}

var predictionWarmup = &warmupState{startedAt: time.Now()}

// BeginPredictionWarmup records the start of the warm-up window and arms the
// safety valve.
func BeginPredictionWarmup() {
	predictionWarmup.mu.Lock()
	predictionWarmup.startedAt = time.Now()
	predictionWarmup.ready = false
	predictionWarmup.mu.Unlock()

	go func() {
		time.Sleep(warmupSafetyValve)
		predictionWarmup.mu.Lock()
		defer predictionWarmup.mu.Unlock()
		if !predictionWarmup.ready {
			predictionWarmup.ready = true
			predictionWarmup.degraded = true
			predictionWarmup.reason = "warm-up exceeded the safety valve; serving from partial history"
			log.Printf("⚠️ Prediction warm-up timed out after %s; serving predictions from partial data",
				warmupSafetyValve)
		}
	}()
}

// CompletePredictionWarmup marks prediction data usable. degraded reports
// whether the backfill finished cleanly; predictions are released either way,
// because refusing to predict indefinitely is the worse failure.
func CompletePredictionWarmup(degraded bool, reason string) {
	predictionWarmup.mu.Lock()
	defer predictionWarmup.mu.Unlock()

	if predictionWarmup.ready {
		return
	}

	predictionWarmup.ready = true
	predictionWarmup.degraded = degraded
	predictionWarmup.reason = reason
	predictionWarmup.completedAt = time.Now()

	elapsed := predictionWarmup.completedAt.Sub(predictionWarmup.startedAt).Round(time.Second)
	if degraded {
		log.Printf("⚠️ Prediction warm-up finished in %s but is degraded: %s", elapsed, reason)
		return
	}
	log.Printf("✅ Prediction data warm after %s; predictions are now served", elapsed)
}

// PredictionDataReady reports whether predictions should be served.
func PredictionDataReady() bool {
	predictionWarmup.mu.RLock()
	defer predictionWarmup.mu.RUnlock()
	return predictionWarmup.ready
}

// PredictionWarmupStatus describes the current warm-up state for callers that
// need to explain themselves to a client.
type PredictionWarmupStatus struct {
	Ready     bool    `json:"ready"`
	Degraded  bool    `json:"degraded,omitempty"`
	Reason    string  `json:"reason,omitempty"`
	ElapsedS  float64 `json:"elapsedSeconds"`
	ExpectedS float64 `json:"expectedSeconds,omitempty"`
}

// GetPredictionWarmupStatus returns a snapshot of warm-up progress.
func GetPredictionWarmupStatus() PredictionWarmupStatus {
	predictionWarmup.mu.RLock()
	defer predictionWarmup.mu.RUnlock()

	status := PredictionWarmupStatus{
		Ready:    predictionWarmup.ready,
		Degraded: predictionWarmup.degraded,
		Reason:   predictionWarmup.reason,
		ElapsedS: time.Since(predictionWarmup.startedAt).Seconds(),
	}
	if !predictionWarmup.ready {
		status.ExpectedS = warmupSafetyValve.Seconds()
	}
	return status
}
