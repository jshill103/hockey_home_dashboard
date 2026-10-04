package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Calibration of the ensemble's final win probability.
//
// Being accurate and being calibrated are different things. A model that says
// 70% can be right more often than one that says 90% while still being badly
// calibrated: if the games it calls at 70% are actually won 85% of the time,
// every downstream consumer that treats the number as a probability -- the
// confidence display, the playoff simulation, any staking decision -- is
// working from a distorted figure. Nothing in the ensemble corrected for this
// previously; confidence was calibrated, but the probability itself was not.
//
// This applies Platt scaling: a one-feature logistic regression on the log-odds
// of the raw probability, fitted against observed outcomes.
//
//	calibrated = sigmoid(slope * logit(raw) + intercept)
//
// Platt scaling is used rather than isotonic regression because isotonic needs
// considerably more data to avoid overfitting its own step function, and an
// NHL season supplies well under two thousand games. Identity (slope 1,
// intercept 0) is a point in the family, so a well-calibrated model is left
// essentially untouched.

const (
	// minCalibrationSamples is the number of settled games required before the
	// mapping is applied at all. Below this, fitted parameters say more about
	// the sample than the model.
	minCalibrationSamples = 100

	// maxCalibrationSamples caps the rolling history. Old games reflect a model
	// and a roster that no longer exist.
	maxCalibrationSamples = 2000

	// gamesBetweenCalibrationFits paces refitting.
	gamesBetweenCalibrationFits = 10

	calibrationBinCount    = 10
	calibrationMaxNewton   = 50
	calibrationConvergence = 1e-8
)

// calibrationSample is one settled prediction: what the ensemble said the home
// team's chances were, and whether the home team then won.
type calibrationSample struct {
	RawProbability float64   `json:"rawProbability"`
	HomeWon        bool      `json:"homeWon"`
	RecordedAt     time.Time `json:"recordedAt"`
}

// ProbabilityCalibrationService maps raw ensemble probabilities onto observed
// frequencies.
type ProbabilityCalibrationService struct {
	mutex   sync.RWMutex
	samples []calibrationSample

	slope     float64
	intercept float64
	fitted    bool
	fittedAt  time.Time

	sinceLastFit int
	dataDir      string
}

type calibrationModelData struct {
	Samples   []calibrationSample `json:"samples"`
	Slope     float64             `json:"slope"`
	Intercept float64             `json:"intercept"`
	Fitted    bool                `json:"fitted"`
	FittedAt  time.Time           `json:"fittedAt"`
}

var (
	probabilityCalibrationInstance *ProbabilityCalibrationService
	probabilityCalibrationOnce     sync.Once
)

// GetProbabilityCalibrationService returns the process-wide calibrator.
func GetProbabilityCalibrationService() *ProbabilityCalibrationService {
	probabilityCalibrationOnce.Do(func() {
		probabilityCalibrationInstance = newProbabilityCalibrationService("data/models")
		if err := probabilityCalibrationInstance.load(); err != nil {
			log.Printf("📏 No existing probability calibration found: %v", err)
		}
	})
	return probabilityCalibrationInstance
}

func newProbabilityCalibrationService(dataDir string) *ProbabilityCalibrationService {
	os.MkdirAll(dataDir, 0755)
	return &ProbabilityCalibrationService{
		slope:     1.0, // identity until fitted
		intercept: 0.0,
		dataDir:   dataDir,
	}
}

// Calibrate maps a raw home-win probability onto the calibrated one. It is the
// identity until enough outcomes have been seen to fit the mapping.
func (pcs *ProbabilityCalibrationService) Calibrate(raw float64) float64 {
	pcs.mutex.RLock()
	fitted, slope, intercept := pcs.fitted, pcs.slope, pcs.intercept
	pcs.mutex.RUnlock()

	if !fitted {
		return raw
	}
	return clampProbability(sigmoid(slope*logit(raw) + intercept))
}

// RecordOutcome records a settled prediction and refits periodically.
func (pcs *ProbabilityCalibrationService) RecordOutcome(rawProbability float64, homeWon bool) error {
	if math.IsNaN(rawProbability) || math.IsInf(rawProbability, 0) {
		return fmt.Errorf("refusing to record non-finite probability %v", rawProbability)
	}

	pcs.mutex.Lock()
	pcs.samples = append(pcs.samples, calibrationSample{
		RawProbability: clampProbability(rawProbability),
		HomeWon:        homeWon,
		RecordedAt:     time.Now(),
	})
	if len(pcs.samples) > maxCalibrationSamples {
		pcs.samples = pcs.samples[len(pcs.samples)-maxCalibrationSamples:]
	}
	pcs.sinceLastFit++
	shouldFit := pcs.sinceLastFit >= gamesBetweenCalibrationFits && len(pcs.samples) >= minCalibrationSamples
	if shouldFit {
		pcs.sinceLastFit = 0
	}
	pcs.mutex.Unlock()

	if shouldFit {
		if err := pcs.Fit(); err != nil {
			log.Printf("📏 Probability calibration fit skipped: %v", err)
		}
	}
	return pcs.save()
}

// Fit refits the Platt mapping over the recorded samples.
func (pcs *ProbabilityCalibrationService) Fit() error {
	pcs.mutex.Lock()
	samples := make([]calibrationSample, len(pcs.samples))
	copy(samples, pcs.samples)
	pcs.mutex.Unlock()

	if len(samples) < minCalibrationSamples {
		return fmt.Errorf("need %d samples to calibrate, have %d", minCalibrationSamples, len(samples))
	}

	var positives, negatives int
	for _, s := range samples {
		if s.HomeWon {
			positives++
		} else {
			negatives++
		}
	}
	if positives == 0 || negatives == 0 {
		return fmt.Errorf("calibration needs both outcomes, have %d home wins and %d losses", positives, negatives)
	}

	// Platt's target smoothing keeps the fit off the asymptotes when a bucket
	// happens to be unanimous.
	hiTarget := (float64(positives) + 1) / (float64(positives) + 2)
	loTarget := 1 / (float64(negatives) + 2)

	z := make([]float64, len(samples))
	y := make([]float64, len(samples))
	for i, s := range samples {
		z[i] = logit(s.RawProbability)
		if s.HomeWon {
			y[i] = hiTarget
		} else {
			y[i] = loTarget
		}
	}

	slope, intercept, err := fitPlattScaling(z, y)
	if err != nil {
		return err
	}

	// Refuse a mapping that does not actually improve the fit on the data it
	// was fitted to. Combined with identity being inside the family, this makes
	// enabling calibration a safe operation rather than a gamble.
	identityLoss := plattLogLoss(z, samples, 1, 0)
	fittedLoss := plattLogLoss(z, samples, slope, intercept)
	if !(fittedLoss < identityLoss) {
		return fmt.Errorf("fitted mapping does not improve log loss (%.5f vs identity %.5f)", fittedLoss, identityLoss)
	}

	pcs.mutex.Lock()
	pcs.slope = slope
	pcs.intercept = intercept
	pcs.fitted = true
	pcs.fittedAt = time.Now()
	pcs.mutex.Unlock()

	log.Printf("📏 Probability calibration fitted over %d games: slope %.3f, intercept %+.3f (log loss %.4f -> %.4f)",
		len(samples), slope, intercept, identityLoss, fittedLoss)
	return nil
}

// fitPlattScaling solves the one-feature logistic regression by Newton's
// method. The problem is two-dimensional and convex, so the 2x2 Hessian can be
// inverted directly and convergence takes a handful of steps.
func fitPlattScaling(z, y []float64) (slope, intercept float64, err error) {
	slope, intercept = 1.0, 0.0

	for iter := 0; iter < calibrationMaxNewton; iter++ {
		var gradSlope, gradIntercept float64
		var h11, h12, h22 float64

		for i := range z {
			p := sigmoid(slope*z[i] + intercept)
			residual := p - y[i]
			gradSlope += residual * z[i]
			gradIntercept += residual

			w := p * (1 - p)
			h11 += w * z[i] * z[i]
			h12 += w * z[i]
			h22 += w
		}

		// Ridge term keeps the Hessian invertible when the probabilities are
		// saturated and every weight collapses toward zero.
		const ridge = 1e-9
		h11 += ridge
		h22 += ridge

		det := h11*h22 - h12*h12
		if math.Abs(det) < 1e-18 {
			return 1, 0, fmt.Errorf("calibration Hessian is singular")
		}

		stepSlope := (h22*gradSlope - h12*gradIntercept) / det
		stepIntercept := (h11*gradIntercept - h12*gradSlope) / det

		slope -= stepSlope
		intercept -= stepIntercept

		if math.IsNaN(slope) || math.IsNaN(intercept) ||
			math.IsInf(slope, 0) || math.IsInf(intercept, 0) {
			return 1, 0, fmt.Errorf("calibration diverged")
		}
		if math.Abs(stepSlope) < calibrationConvergence && math.Abs(stepIntercept) < calibrationConvergence {
			break
		}
	}

	// A negative slope would invert the model's ordering, which is never a
	// calibration result worth trusting.
	if slope <= 0 {
		return 1, 0, fmt.Errorf("calibration produced a non-positive slope (%.4f)", slope)
	}
	return slope, intercept, nil
}

func plattLogLoss(z []float64, samples []calibrationSample, slope, intercept float64) float64 {
	var total float64
	for i, s := range samples {
		p := clampProbability(sigmoid(slope*z[i] + intercept))
		if s.HomeWon {
			total -= math.Log(p)
		} else {
			total -= math.Log(1 - p)
		}
	}
	return total / float64(len(samples))
}

// CalibrationReport summarises how closely stated probabilities have matched
// observed frequencies.
type CalibrationReport struct {
	Samples                  int              `json:"samples"`
	Fitted                   bool             `json:"fitted"`
	Slope                    float64          `json:"slope"`
	Intercept                float64          `json:"intercept"`
	ExpectedCalibrationError float64          `json:"expectedCalibrationError"`
	BrierScore               float64          `json:"brierScore"`
	Bins                     []CalibrationBin `json:"bins"`
	FittedAt                 time.Time        `json:"fittedAt"`
}

// CalibrationBin is one bucket of the reliability curve.
type CalibrationBin struct {
	LowerBound        float64 `json:"lowerBound"`
	UpperBound        float64 `json:"upperBound"`
	Count             int     `json:"count"`
	MeanPredicted     float64 `json:"meanPredicted"`
	ObservedFrequency float64 `json:"observedFrequency"`
}

// Report computes the reliability curve and expected calibration error over the
// recorded samples, measured on the probabilities as they are actually served
// (that is, after any fitted mapping is applied).
func (pcs *ProbabilityCalibrationService) Report() CalibrationReport {
	pcs.mutex.RLock()
	samples := make([]calibrationSample, len(pcs.samples))
	copy(samples, pcs.samples)
	report := CalibrationReport{
		Samples:   len(samples),
		Fitted:    pcs.fitted,
		Slope:     pcs.slope,
		Intercept: pcs.intercept,
		FittedAt:  pcs.fittedAt,
	}
	pcs.mutex.RUnlock()

	if len(samples) == 0 {
		return report
	}

	type bucket struct {
		count          int
		predictedTotal float64
		positives      int
	}
	buckets := make([]bucket, calibrationBinCount)

	var brierSum float64
	for _, s := range samples {
		p := pcs.Calibrate(s.RawProbability)

		outcome := 0.0
		if s.HomeWon {
			outcome = 1.0
		}
		brierSum += (p - outcome) * (p - outcome)

		idx := int(p * calibrationBinCount)
		if idx >= calibrationBinCount {
			idx = calibrationBinCount - 1
		}
		if idx < 0 {
			idx = 0
		}
		buckets[idx].count++
		buckets[idx].predictedTotal += p
		if s.HomeWon {
			buckets[idx].positives++
		}
	}

	report.BrierScore = brierSum / float64(len(samples))

	var ece float64
	for i, b := range buckets {
		lower := float64(i) / calibrationBinCount
		upper := float64(i+1) / calibrationBinCount
		bin := CalibrationBin{LowerBound: lower, UpperBound: upper, Count: b.count}
		if b.count > 0 {
			bin.MeanPredicted = b.predictedTotal / float64(b.count)
			bin.ObservedFrequency = float64(b.positives) / float64(b.count)
			ece += float64(b.count) / float64(len(samples)) *
				math.Abs(bin.ObservedFrequency-bin.MeanPredicted)
		}
		report.Bins = append(report.Bins, bin)
	}
	report.ExpectedCalibrationError = ece

	return report
}

func (pcs *ProbabilityCalibrationService) save() error {
	pcs.mutex.RLock()
	data := calibrationModelData{
		Samples:   pcs.samples,
		Slope:     pcs.slope,
		Intercept: pcs.intercept,
		Fitted:    pcs.fitted,
		FittedAt:  pcs.fittedAt,
	}
	path := filepath.Join(pcs.dataDir, "probability_calibration.json")
	pcs.mutex.RUnlock()

	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling probability calibration: %v", err)
	}
	return os.WriteFile(path, encoded, 0644)
}

func (pcs *ProbabilityCalibrationService) load() error {
	path := filepath.Join(pcs.dataDir, "probability_calibration.json")
	encoded, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var data calibrationModelData
	if err := json.Unmarshal(encoded, &data); err != nil {
		return fmt.Errorf("error unmarshaling probability calibration: %v", err)
	}

	pcs.mutex.Lock()
	defer pcs.mutex.Unlock()
	pcs.samples = data.Samples
	if data.Fitted && data.Slope > 0 {
		pcs.slope = data.Slope
		pcs.intercept = data.Intercept
		pcs.fitted = true
		pcs.fittedAt = data.FittedAt
	}
	log.Printf("📏 Loaded probability calibration: %d samples, fitted=%v", len(pcs.samples), pcs.fitted)
	return nil
}

// calibrationSigmoid is the numerically stable form. The package already has a
// plain sigmoid for the LSTM activations; this one keeps full precision in the
// tails, where Platt scaling spends most of its time.
func calibrationSigmoid(x float64) float64 {
	if x >= 0 {
		return 1 / (1 + math.Exp(-x))
	}
	e := math.Exp(x)
	return e / (1 + e)
}

func logit(p float64) float64 {
	p = clampProbability(p)
	return math.Log(p / (1 - p))
}

// clampProbability keeps probabilities strictly inside (0,1) so that logit and
// log loss stay finite.
func clampProbability(p float64) float64 {
	const eps = 1e-6
	if math.IsNaN(p) {
		return 0.5
	}
	if p < eps {
		return eps
	}
	if p > 1-eps {
		return 1 - eps
	}
	return p
}
