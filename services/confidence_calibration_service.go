package services

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

var (
	calibrationInstance *ConfidenceCalibrationService
	calibrationOnce     sync.Once
)

// ConfidenceCalibrationService adjusts confidence to match actual accuracy
type ConfidenceCalibrationService struct {
	mu              sync.RWMutex
	calibrationCurve *models.CalibrationCurve
	dataDir         string
	predictionLog   []predictionRecord // For building calibration curve
}

// predictionRecord tracks a prediction for calibration
type predictionRecord struct {
	PredictedConfidence float64
	ActualCorrect       bool
	Timestamp           time.Time
}

// InitializeConfidenceCalibration initializes the singleton ConfidenceCalibrationService
func InitializeConfidenceCalibration() error {
	var err error
	calibrationOnce.Do(func() {
		dataDir := filepath.Join("data", "calibration")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create calibration directory: %w", err)
			return
		}

		calibrationInstance = &ConfidenceCalibrationService{
			calibrationCurve: &models.CalibrationCurve{
				Bins: initializeCalibrationBins(),
			},
			dataDir:       dataDir,
			predictionLog: []predictionRecord{},
		}

		// Load existing calibration data
		if loadErr := calibrationInstance.loadCalibrationData(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing calibration data: %v\n", loadErr)
		}

		fmt.Println("✅ Confidence Calibration Service initialized")
	})
	return err
}

// GetConfidenceCalibrationService returns the singleton instance
func GetConfidenceCalibrationService() *ConfidenceCalibrationService {
	return calibrationInstance
}

// CalibrateConfidence adjusts a predicted confidence to match historical accuracy
func (ccs *ConfidenceCalibrationService) CalibrateConfidence(predictedConf float64) float64 {
	ccs.mu.RLock()
	defer ccs.mu.RUnlock()

	// If we don't have enough data yet, use a conservative adjustment
	if ccs.calibrationCurve.TotalSamples < 30 {
		// Be more conservative early on (reduce overconfidence)
		return predictedConf * 0.95
	}

	// Find the appropriate bin
	var bin *models.ConfidenceBin
	for i := range ccs.calibrationCurve.Bins {
		b := &ccs.calibrationCurve.Bins[i]
		if predictedConf >= b.MinConfidence && predictedConf < b.MaxConfidence {
			bin = b
			break
		}
	}

	// If no bin found (shouldn't happen), use last bin
	if bin == nil {
		bin = &ccs.calibrationCurve.Bins[len(ccs.calibrationCurve.Bins)-1]
	}

	// If bin has insufficient samples, return original
	if bin.SampleSize < 5 {
		return predictedConf
	}

	// Use the actual accuracy from this bin as the calibrated confidence
	calibratedConf := bin.ActualAccuracy

	// Smooth the adjustment to avoid dramatic changes
	smoothingFactor := 0.7
	finalConf := smoothingFactor*calibratedConf + (1-smoothingFactor)*predictedConf

	// Ensure reasonable bounds
	return math.Max(0.50, math.Min(0.95, finalConf))
}

// RecordPredictionOutcome records a prediction outcome for calibration curve updates
func (ccs *ConfidenceCalibrationService) RecordPredictionOutcome(predictedConf float64, correct bool) error {
	ccs.mu.Lock()
	defer ccs.mu.Unlock()

	// Add to prediction log
	ccs.predictionLog = append(ccs.predictionLog, predictionRecord{
		PredictedConfidence: predictedConf,
		ActualCorrect:       correct,
		Timestamp:           time.Now(),
	})

	// Keep only last 200 predictions
	if len(ccs.predictionLog) > 200 {
		ccs.predictionLog = ccs.predictionLog[len(ccs.predictionLog)-200:]
	}

	// Update the bin
	if err := ccs.updateBin(predictedConf, correct); err != nil {
		return err
	}

	// Recalculate overall metrics every 10 predictions
	if len(ccs.predictionLog)%10 == 0 {
		ccs.updateOverallMetrics()
	}

	return ccs.saveCalibrationData()
}

// UpdateCalibrationCurve rebuilds the calibration curve from scratch
func (ccs *ConfidenceCalibrationService) UpdateCalibrationCurve() error {
	ccs.mu.Lock()
	defer ccs.mu.Unlock()

	// Reset bins
	ccs.calibrationCurve.Bins = initializeCalibrationBins()

	// Reprocess all predictions
	for _, record := range ccs.predictionLog {
		if err := ccs.updateBin(record.PredictedConfidence, record.ActualCorrect); err != nil {
			return err
		}
	}

	ccs.updateOverallMetrics()
	ccs.calibrationCurve.LastUpdated = time.Now()

	fmt.Println("📊 Calibration curve updated")
	return ccs.saveCalibrationData()
}

// GetCalibrationBins returns the current calibration bins
func (ccs *ConfidenceCalibrationService) GetCalibrationBins() []models.ConfidenceBin {
	ccs.mu.RLock()
	defer ccs.mu.RUnlock()

	// Return a copy
	bins := make([]models.ConfidenceBin, len(ccs.calibrationCurve.Bins))
	copy(bins, ccs.calibrationCurve.Bins)
	return bins
}

// GetCalibrationCurve returns the full calibration curve
func (ccs *ConfidenceCalibrationService) GetCalibrationCurve() *models.CalibrationCurve {
	ccs.mu.RLock()
	defer ccs.mu.RUnlock()

	// Return a copy
	curveCopy := *ccs.calibrationCurve
	curveCopy.Bins = make([]models.ConfidenceBin, len(ccs.calibrationCurve.Bins))
	copy(curveCopy.Bins, ccs.calibrationCurve.Bins)
	return &curveCopy
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func initializeCalibrationBins() []models.ConfidenceBin {
	bins := []models.ConfidenceBin{
		{Range: "50-60%", MinConfidence: 0.50, MaxConfidence: 0.60, PredictedConf: 0.55, ExpectedAccuracy: 0.55, PredictionCount: 0, SampleSize: 0},
		{Range: "60-70%", MinConfidence: 0.60, MaxConfidence: 0.70, PredictedConf: 0.65, ExpectedAccuracy: 0.65, PredictionCount: 0, SampleSize: 0},
		{Range: "70-80%", MinConfidence: 0.70, MaxConfidence: 0.80, PredictedConf: 0.75, ExpectedAccuracy: 0.75, PredictionCount: 0, SampleSize: 0},
		{Range: "80-90%", MinConfidence: 0.80, MaxConfidence: 0.90, PredictedConf: 0.85, ExpectedAccuracy: 0.85, PredictionCount: 0, SampleSize: 0},
		{Range: "90-100%", MinConfidence: 0.90, MaxConfidence: 1.00, PredictedConf: 0.95, ExpectedAccuracy: 0.95, PredictionCount: 0, SampleSize: 0},
	}
	return bins
}

func (ccs *ConfidenceCalibrationService) updateBin(predictedConf float64, correct bool) error {
	// Find the appropriate bin
	for i := range ccs.calibrationCurve.Bins {
		bin := &ccs.calibrationCurve.Bins[i]
		if predictedConf >= bin.MinConfidence && predictedConf < bin.MaxConfidence {
			// Update bin statistics
			oldSampleSize := float64(bin.SampleSize)
			bin.SampleSize++
			bin.PredictionCount = bin.SampleSize // Keep both in sync
			newSampleSize := float64(bin.SampleSize)

			// Update average predicted confidence (running average)
			bin.PredictedConf = (bin.PredictedConf*oldSampleSize + predictedConf) / newSampleSize
			bin.ExpectedAccuracy = bin.PredictedConf // Keep both in sync

			// Update actual accuracy (running average)
			accuracy := 0.0
			if correct {
				accuracy = 1.0
			}
			bin.ActualAccuracy = (bin.ActualAccuracy*oldSampleSize + accuracy) / newSampleSize

			// Update calibration adjustment
			bin.CalibrationAdj = bin.ActualAccuracy - bin.PredictedConf
			bin.CalibrationError = bin.CalibrationAdj // Keep both in sync

			bin.LastUpdated = time.Now()
			return nil
		}
	}

	// If we get here, prediction confidence was out of range - use last bin
	bin := &ccs.calibrationCurve.Bins[len(ccs.calibrationCurve.Bins)-1]
	oldSampleSize := float64(bin.SampleSize)
	bin.SampleSize++
	bin.PredictionCount = bin.SampleSize
	newSampleSize := float64(bin.SampleSize)

	accuracy := 0.0
	if correct {
		accuracy = 1.0
	}
	bin.ActualAccuracy = (bin.ActualAccuracy*oldSampleSize + accuracy) / newSampleSize
	bin.CalibrationAdj = bin.ActualAccuracy - bin.PredictedConf
	bin.CalibrationError = bin.CalibrationAdj
	bin.LastUpdated = time.Now()

	return nil
}

func (ccs *ConfidenceCalibrationService) updateOverallMetrics() {
	// Calculate overall bias (are we systematically over/under-confident?)
	totalBias := 0.0
	totalWeight := 0
	for _, bin := range ccs.calibrationCurve.Bins {
		if bin.SampleSize > 0 {
			totalBias += bin.CalibrationAdj * float64(bin.SampleSize)
			totalWeight += bin.SampleSize
		}
	}

	if totalWeight > 0 {
		ccs.calibrationCurve.OverallBias = totalBias / float64(totalWeight)
	}

	// Calculate reliability (how well-calibrated we are)
	// Perfect calibration = reliability 1.0, poor calibration = reliability 0.0
	totalError := 0.0
	for _, bin := range ccs.calibrationCurve.Bins {
		if bin.SampleSize > 0 {
			error := math.Abs(bin.PredictedConf - bin.ActualAccuracy)
			totalError += error * float64(bin.SampleSize)
		}
	}

	if totalWeight > 0 {
		avgError := totalError / float64(totalWeight)
		// Convert error to reliability score (0-1)
		ccs.calibrationCurve.Reliability = math.Max(0.0, 1.0-avgError*2.0)
	}

	ccs.calibrationCurve.TotalSamples = totalWeight
	ccs.calibrationCurve.LastUpdated = time.Now()
}

// GetCalibrationReport generates a human-readable calibration report
func (ccs *ConfidenceCalibrationService) GetCalibrationReport() string {
	ccs.mu.RLock()
	defer ccs.mu.RUnlock()

	report := "📊 CONFIDENCE CALIBRATION REPORT\n\n"
	report += fmt.Sprintf("Total Samples: %d\n", ccs.calibrationCurve.TotalSamples)
	report += fmt.Sprintf("Overall Bias: %.2f%% ", ccs.calibrationCurve.OverallBias*100)

	if ccs.calibrationCurve.OverallBias > 0.05 {
		report += "(underconfident)\n"
	} else if ccs.calibrationCurve.OverallBias < -0.05 {
		report += "(overconfident)\n"
	} else {
		report += "(well-calibrated)\n"
	}

	report += fmt.Sprintf("Reliability Score: %.1f%%\n\n", ccs.calibrationCurve.Reliability*100)

	report += "Calibration by Confidence Range:\n"
	for _, bin := range ccs.calibrationCurve.Bins {
		if bin.SampleSize > 0 {
			report += fmt.Sprintf("  %s: Predicted %.1f%%, Actual %.1f%% (%d samples)\n",
				bin.Range, bin.PredictedConf*100, bin.ActualAccuracy*100, bin.SampleSize)
		}
	}

	return report
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (ccs *ConfidenceCalibrationService) getCalibrationDataPath() string {
	return filepath.Join(ccs.dataDir, "calibration_curve.json")
}

func (ccs *ConfidenceCalibrationService) getPredictionLogPath() string {
	return filepath.Join(ccs.dataDir, "prediction_log.json")
}

func (ccs *ConfidenceCalibrationService) saveCalibrationData() error {
	// Save calibration curve
	curveData, err := json.MarshalIndent(ccs.calibrationCurve, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal calibration curve: %w", err)
	}
	if err := os.WriteFile(ccs.getCalibrationDataPath(), curveData, 0644); err != nil {
		return fmt.Errorf("failed to write calibration curve: %w", err)
	}

	// Save prediction log
	logData, err := json.MarshalIndent(ccs.predictionLog, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prediction log: %w", err)
	}
	if err := os.WriteFile(ccs.getPredictionLogPath(), logData, 0644); err != nil {
		return fmt.Errorf("failed to write prediction log: %w", err)
	}

	return nil
}

func (ccs *ConfidenceCalibrationService) loadCalibrationData() error {
	// Load calibration curve
	curvePath := ccs.getCalibrationDataPath()
	if data, err := os.ReadFile(curvePath); err == nil {
		if err := json.Unmarshal(data, &ccs.calibrationCurve); err != nil {
			return fmt.Errorf("failed to unmarshal calibration curve: %w", err)
		}
		fmt.Printf("📊 Loaded calibration curve with %d samples\n", ccs.calibrationCurve.TotalSamples)
	}

	// Load prediction log
	logPath := ccs.getPredictionLogPath()
	if data, err := os.ReadFile(logPath); err == nil {
		if err := json.Unmarshal(data, &ccs.predictionLog); err != nil {
			return fmt.Errorf("failed to unmarshal prediction log: %w", err)
		}

		// Sort by timestamp
		sort.Slice(ccs.predictionLog, func(i, j int) bool {
			return ccs.predictionLog[i].Timestamp.Before(ccs.predictionLog[j].Timestamp)
		})

		fmt.Printf("📊 Loaded prediction log with %d entries\n", len(ccs.predictionLog))
	}

	return nil
}

