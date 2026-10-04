package services

import (
	"math"
	"math/rand"
	"testing"
)

func newTestCalibrator(t *testing.T) *ProbabilityCalibrationService {
	t.Helper()
	return newProbabilityCalibrationService(t.TempDir())
}

// feed records n outcomes where the true home-win probability is derived from
// the stated one by distort, letting a specific miscalibration be planted.
func feed(t *testing.T, pcs *ProbabilityCalibrationService, n int, seed int64, distort func(float64) float64) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	for i := 0; i < n; i++ {
		stated := 0.05 + 0.90*rng.Float64()
		trueProb := distort(stated)
		pcs.samples = append(pcs.samples, calibrationSample{
			RawProbability: stated,
			HomeWon:        rng.Float64() < trueProb,
		})
	}
}

func TestCalibrationIsIdentityBeforeEnoughData(t *testing.T) {
	pcs := newTestCalibrator(t)
	feed(t, pcs, minCalibrationSamples-1, 1, func(p float64) float64 { return p })

	if err := pcs.Fit(); err == nil {
		t.Fatal("expected Fit to refuse below the sample minimum")
	}
	for _, p := range []float64{0.1, 0.35, 0.5, 0.8, 0.97} {
		if got := pcs.Calibrate(p); got != p {
			t.Errorf("Calibrate(%.2f) = %.4f, want the identity until fitted", p, got)
		}
	}
}

// An overconfident model states probabilities further from 0.5 than reality
// warrants. Calibration must pull them back toward the middle.
func TestCalibrationCorrectsOverconfidence(t *testing.T) {
	pcs := newTestCalibrator(t)

	// True probability is a shrunk version of the stated one.
	overconfident := func(stated float64) float64 { return 0.5 + (stated-0.5)*0.55 }
	feed(t, pcs, 1500, 42, overconfident)

	if err := pcs.Fit(); err != nil {
		t.Fatalf("Fit failed: %v", err)
	}

	pcs.mutex.RLock()
	slope := pcs.slope
	pcs.mutex.RUnlock()
	if slope >= 1.0 {
		t.Errorf("slope = %.3f, want < 1 to damp an overconfident model", slope)
	}

	// Every confident statement should move toward 0.5, and never past it.
	for _, stated := range []float64{0.80, 0.90, 0.95} {
		got := pcs.Calibrate(stated)
		want := overconfident(stated)
		if got >= stated {
			t.Errorf("Calibrate(%.2f) = %.4f, expected it to be pulled below the stated value", stated, got)
		}
		if math.Abs(got-want) > 0.06 {
			t.Errorf("Calibrate(%.2f) = %.4f, want near the true %.4f", stated, got, want)
		}
	}
	for _, stated := range []float64{0.05, 0.10, 0.20} {
		got := pcs.Calibrate(stated)
		if got <= stated {
			t.Errorf("Calibrate(%.2f) = %.4f, expected it to be pulled above the stated value", stated, got)
		}
	}
}

// A systematically biased model is shifted, not just damped.
func TestCalibrationCorrectsBias(t *testing.T) {
	pcs := newTestCalibrator(t)

	biased := func(stated float64) float64 {
		return clampProbability(calibrationSigmoid(logit(stated) - 0.8))
	}
	feed(t, pcs, 2000, 7, biased)

	if err := pcs.Fit(); err != nil {
		t.Fatalf("Fit failed: %v", err)
	}

	pcs.mutex.RLock()
	intercept := pcs.intercept
	pcs.mutex.RUnlock()
	if intercept >= 0 {
		t.Errorf("intercept = %+.3f, want negative to offset a home-biased model", intercept)
	}

	for _, stated := range []float64{0.3, 0.5, 0.7} {
		got := pcs.Calibrate(stated)
		want := biased(stated)
		if math.Abs(got-want) > 0.05 {
			t.Errorf("Calibrate(%.2f) = %.4f, want near the true %.4f", stated, got, want)
		}
	}
}

// Calibrating an already-calibrated model must not damage it.
func TestCalibrationLeavesGoodModelAlone(t *testing.T) {
	pcs := newTestCalibrator(t)
	feed(t, pcs, 2000, 99, func(p float64) float64 { return p })

	before := pcs.Report()

	// Fit may legitimately refuse here: identity is already optimal, and the
	// guard rejects a mapping that fails to beat it.
	_ = pcs.Fit()

	after := pcs.Report()

	if after.ExpectedCalibrationError > before.ExpectedCalibrationError+0.01 {
		t.Errorf("calibration made a well-calibrated model worse: ECE %.4f -> %.4f",
			before.ExpectedCalibrationError, after.ExpectedCalibrationError)
	}
	for _, p := range []float64{0.2, 0.5, 0.85} {
		if math.Abs(pcs.Calibrate(p)-p) > 0.05 {
			t.Errorf("Calibrate(%.2f) = %.4f, want it left near the stated value", p, pcs.Calibrate(p))
		}
	}
}

// The headline claim: calibration reduces expected calibration error and Brier
// score on a miscalibrated model.
func TestCalibrationImprovesECEAndBrier(t *testing.T) {
	pcs := newTestCalibrator(t)
	overconfident := func(stated float64) float64 { return 0.5 + (stated-0.5)*0.55 }
	feed(t, pcs, 2000, 2026, overconfident)

	before := pcs.Report()
	if err := pcs.Fit(); err != nil {
		t.Fatalf("Fit failed: %v", err)
	}
	after := pcs.Report()

	t.Logf("ECE   %.4f -> %.4f", before.ExpectedCalibrationError, after.ExpectedCalibrationError)
	t.Logf("Brier %.4f -> %.4f", before.BrierScore, after.BrierScore)

	if after.ExpectedCalibrationError >= before.ExpectedCalibrationError {
		t.Errorf("ECE did not improve: %.4f -> %.4f", before.ExpectedCalibrationError, after.ExpectedCalibrationError)
	}
	if after.BrierScore >= before.BrierScore {
		t.Errorf("Brier did not improve: %.4f -> %.4f", before.BrierScore, after.BrierScore)
	}
}

func TestCalibrationRejectsDegenerateData(t *testing.T) {
	pcs := newTestCalibrator(t)
	// Every game a home win: nothing to calibrate against.
	for i := 0; i < minCalibrationSamples+50; i++ {
		pcs.samples = append(pcs.samples, calibrationSample{RawProbability: 0.6, HomeWon: true})
	}
	if err := pcs.Fit(); err == nil {
		t.Fatal("expected Fit to refuse when only one outcome is present")
	}
	if got := pcs.Calibrate(0.6); got != 0.6 {
		t.Errorf("Calibrate(0.6) = %.4f, want the identity after a refused fit", got)
	}
}

func TestCalibrateHandlesExtremeInputs(t *testing.T) {
	pcs := newTestCalibrator(t)
	feed(t, pcs, 1500, 5, func(p float64) float64 { return 0.5 + (p-0.5)*0.6 })
	if err := pcs.Fit(); err != nil {
		t.Fatalf("Fit failed: %v", err)
	}

	for _, p := range []float64{0, 1, -0.5, 1.5, math.NaN()} {
		got := pcs.Calibrate(p)
		if math.IsNaN(got) || got <= 0 || got >= 1 {
			t.Errorf("Calibrate(%v) = %v, want a finite probability strictly inside (0,1)", p, got)
		}
	}
}
