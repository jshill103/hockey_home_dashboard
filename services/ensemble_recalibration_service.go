package services

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

var (
	recalibrationInstance *EnsembleRecalibrationService
	recalibrationOnce     sync.Once
)

// EnsembleRecalibrationService dynamically adjusts model weights based on performance
type EnsembleRecalibrationService struct {
	mu                sync.RWMutex
	modelPerformance  map[string]*models.ModelPerformanceMetrics
	recalibrationHistory *models.RecalibrationHistory
	config            models.RecalibrationConfig
	dataDir           string
	predictionsSinceLastRecal int
}

// InitializeRecalibration initializes the singleton EnsembleRecalibrationService
func InitializeRecalibration() error {
	var err error
	recalibrationOnce.Do(func() {
		dataDir := filepath.Join("data", "recalibration")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create recalibration directory: %w", err)
			return
		}

		// Default configuration
		config := models.RecalibrationConfig{
			Enabled:           true,
			UpdateFrequency:   10, // Recalibrate every 10 predictions
			AccuracyThreshold: 0.05, // 5% accuracy drop triggers recalibration
			LearningRate:      0.03, // 3% max adjustment per update
			Constraints: models.WeightConstraints{
				MinWeight:         0.05, // No model below 5%
				MaxWeight:         0.40, // No model above 40%
				MaxShiftPerUpdate: 0.03, // Max 3% change per update
				MinSampleSize:     15,   // Need 15 predictions minimum
				SmoothingFactor:   0.3,  // EMA smoothing
			},
			AutoRecalibrate:       true,
			RequireManualApproval: false,
		}

		recalibrationInstance = &EnsembleRecalibrationService{
			modelPerformance:  make(map[string]*models.ModelPerformanceMetrics),
			recalibrationHistory: &models.RecalibrationHistory{
				Events: []models.RecalibrationEvent{},
			},
			config:  config,
			dataDir: dataDir,
		}

		// Load existing data
		if loadErr := recalibrationInstance.loadRecalibrationData(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing recalibration data: %v\n", loadErr)
		}

		// Initialize model performance trackers
		recalibrationInstance.initializeModelTrackers()

		fmt.Println("✅ Ensemble Recalibration Service initialized")
	})
	return err
}

// GetRecalibrationService returns the singleton instance
func GetRecalibrationService() *EnsembleRecalibrationService {
	return recalibrationInstance
}

// RecordPredictionOutcome records the outcome of a prediction for all models
func (ers *EnsembleRecalibrationService) RecordPredictionOutcome(
	modelResults []models.ModelResult, 
	actualWinner string, 
	predictedWinner string,
	contextType string) error {
	
	ers.mu.Lock()
	defer ers.mu.Unlock()

	ers.predictionsSinceLastRecal++

	// Update each model's performance
	for _, result := range modelResults {
		perf := ers.getOrCreatePerformanceMetrics(result.ModelName)
		
		// Determine if this model was correct
		modelPredictedWinner := ""
		if result.WinProbability > 0.5 {
			modelPredictedWinner = actualWinner // Simplified - would need home/away team
		}
		modelCorrect := modelPredictedWinner == actualWinner
		
		// Update totals
		perf.TotalPredictions++
		if modelCorrect {
			perf.CorrectPredictions++
		}
		
		// Update recent performance (last 20)
		if perf.RecentPredictions < 20 {
			perf.RecentPredictions++
			if modelCorrect {
				perf.RecentCorrect++
			}
		} else {
			// Shift window - remove oldest, add newest
			perf.RecentCorrect = int(float64(perf.RecentCorrect) * 0.95)
			if modelCorrect {
				perf.RecentCorrect++
			}
		}
		
		// Recalculate accuracies
		if perf.TotalPredictions > 0 {
			perf.OverallAccuracy = float64(perf.CorrectPredictions) / float64(perf.TotalPredictions)
		}
		if perf.RecentPredictions > 0 {
			perf.RecentAccuracy = float64(perf.RecentCorrect) / float64(perf.RecentPredictions)
		}
		
		// Update confidence metrics
		alpha := 0.1
		perf.AvgConfidence = alpha*result.Confidence + (1-alpha)*perf.AvgConfidence
		perf.ConfidenceAccuracyGap = perf.AvgConfidence - perf.OverallAccuracy
		
		// Update context performance
		if perf.PerformanceByContext == nil {
			perf.PerformanceByContext = make(map[string]float64)
		}
		if contextType != "" {
			// Simple rolling average for context performance
			oldPerf := perf.PerformanceByContext[contextType]
			accuracy := 0.0
			if modelCorrect {
				accuracy = 1.0
			}
			perf.PerformanceByContext[contextType] = alpha*accuracy + (1-alpha)*oldPerf
		}
		
		// Determine trend
		if perf.TotalPredictions > 20 {
			perf.ImprovingTrend = perf.RecentAccuracy > perf.OverallAccuracy
			perf.AccuracyTrend = perf.RecentAccuracy - perf.OverallAccuracy
		}
		
		perf.LastUpdated = time.Now()
	}

	// Check if recalibration should be triggered
	if ers.config.AutoRecalibrate && ers.predictionsSinceLastRecal >= ers.config.UpdateFrequency {
		if err := ers.TriggerRecalibration("scheduled"); err != nil {
			fmt.Printf("⚠️ Warning: Recalibration failed: %v\n", err)
		}
	}

	return ers.saveRecalibrationData()
}

// TriggerRecalibration performs a recalibration of model weights
func (ers *EnsembleRecalibrationService) TriggerRecalibration(trigger string) error {
	ers.mu.Lock()
	defer ers.mu.Unlock()

	// Get current weights (from dynamic weighting service if available)
	oldWeights := ers.getCurrentWeights()
	
	// Calculate new weights based on recent performance
	newWeights := ers.calculateNewWeights(oldWeights)
	
	// Calculate weight changes
	weightChanges := make(map[string]float64)
	for model, newWeight := range newWeights {
		weightChanges[model] = newWeight - oldWeights[model]
	}
	
	// Create performance snapshot
	performanceSnapshot := make(map[string]float64)
	for model, perf := range ers.modelPerformance {
		performanceSnapshot[model] = perf.RecentAccuracy
	}
	
	// Record the recalibration event
	event := models.RecalibrationEvent{
		Timestamp:           time.Now(),
		Trigger:             trigger,
		OldWeights:          oldWeights,
		NewWeights:          newWeights,
		WeightChanges:       weightChanges,
		PerformanceSnapshot: performanceSnapshot,
		Reasoning:           ers.generateRecalibrationReasoning(oldWeights, newWeights, performanceSnapshot),
		PredictionCount:     ers.getTotalPredictions(),
	}
	
	ers.recalibrationHistory.Events = append(ers.recalibrationHistory.Events, event)
	ers.recalibrationHistory.LastRecalibration = time.Now()
	ers.recalibrationHistory.TotalRecalibrations++
	
	// Update model performance with new weights
	for model, newWeight := range newWeights {
		if perf, exists := ers.modelPerformance[model]; exists {
			perf.CurrentWeight = newWeight
			perf.WeightAdjustment = newWeight - perf.BaseWeight
		}
	}
	
	// Reset prediction counter
	ers.predictionsSinceLastRecal = 0
	
	fmt.Printf("⚖️ Recalibration complete: %d weight adjustments\n", len(weightChanges))
	
	return ers.saveRecalibrationData()
}

// GetCalibratedWeights returns the current calibrated model weights
func (ers *EnsembleRecalibrationService) GetCalibratedWeights() map[string]float64 {
	ers.mu.RLock()
	defer ers.mu.RUnlock()

	weights := make(map[string]float64)
	for model, perf := range ers.modelPerformance {
		weights[model] = perf.CurrentWeight
	}
	
	return weights
}

// GetModelPerformance returns performance metrics for a specific model
func (ers *EnsembleRecalibrationService) GetModelPerformance(modelName string) *models.ModelPerformanceMetrics {
	ers.mu.RLock()
	defer ers.mu.RUnlock()

	return ers.modelPerformance[modelName]
}

// GetAllModelPerformance returns performance metrics for all models
func (ers *EnsembleRecalibrationService) GetAllModelPerformance() map[string]*models.ModelPerformanceMetrics {
	ers.mu.RLock()
	defer ers.mu.RUnlock()

	// Return a copy to prevent external modification
	performance := make(map[string]*models.ModelPerformanceMetrics)
	for model, perf := range ers.modelPerformance {
		perfCopy := *perf
		performance[model] = &perfCopy
	}
	
	return performance
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (ers *EnsembleRecalibrationService) initializeModelTrackers() {
	models := []string{
		"Enhanced Statistical",
		"Bayesian Inference",
		"Monte Carlo Simulation",
		"Elo Rating",
		"Poisson Regression",
		"Neural Network",
		"Gradient Boosting",
		"LSTM",
		"Random Forest",
	}

	baseWeights := map[string]float64{
		"Enhanced Statistical":   0.30,
		"Bayesian Inference":     0.12,
		"Monte Carlo Simulation": 0.09,
		"Elo Rating":             0.17,
		"Poisson Regression":     0.12,
		"Neural Network":         0.06,
		"Gradient Boosting":      0.07,
		"LSTM":                   0.07,
		"Random Forest":          0.07,
	}

	for _, model := range models {
		if _, exists := ers.modelPerformance[model]; !exists {
			ers.modelPerformance[model] = &models.ModelPerformanceMetrics{
				ModelName:            model,
				BaseWeight:           baseWeights[model],
				CurrentWeight:        baseWeights[model],
				PerformanceByContext: make(map[string]float64),
				LastUpdated:          time.Now(),
			}
		}
	}
}

func (ers *EnsembleRecalibrationService) getOrCreatePerformanceMetrics(modelName string) *models.ModelPerformanceMetrics {
	if perf, exists := ers.modelPerformance[modelName]; exists {
		return perf
	}
	
	perf := &models.ModelPerformanceMetrics{
		ModelName:            modelName,
		BaseWeight:           0.10, // Default
		CurrentWeight:        0.10,
		PerformanceByContext: make(map[string]float64),
		LastUpdated:          time.Now(),
	}
	ers.modelPerformance[modelName] = perf
	return perf
}

func (ers *EnsembleRecalibrationService) getCurrentWeights() map[string]float64 {
	weights := make(map[string]float64)
	for model, perf := range ers.modelPerformance {
		weights[model] = perf.CurrentWeight
	}
	return weights
}

func (ers *EnsembleRecalibrationService) calculateNewWeights(oldWeights map[string]float64) map[string]float64 {
	newWeights := make(map[string]float64)
	
	// Calculate weights based on recent performance
	for model, perf := range ers.modelPerformance {
		if perf.RecentPredictions < ers.config.Constraints.MinSampleSize {
			// Not enough data, keep current weight
			newWeights[model] = perf.CurrentWeight
			continue
		}
		
		// Calculate adjustment based on recent vs overall performance
		performanceDelta := perf.RecentAccuracy - 0.55 // Compare to baseline
		adjustment := performanceDelta * ers.config.LearningRate
		
		// Apply constraints
		maxShift := ers.config.Constraints.MaxShiftPerUpdate
		adjustment = math.Max(-maxShift, math.Min(maxShift, adjustment))
		
		// Calculate new weight with smoothing
		rawNewWeight := perf.CurrentWeight * (1 + adjustment)
		smoothedWeight := ers.config.Constraints.SmoothingFactor*rawNewWeight + 
			(1-ers.config.Constraints.SmoothingFactor)*perf.CurrentWeight
		
		// Apply min/max constraints
		smoothedWeight = math.Max(ers.config.Constraints.MinWeight, 
			math.Min(ers.config.Constraints.MaxWeight, smoothedWeight))
		
		newWeights[model] = smoothedWeight
	}
	
	// Normalize to sum to 1.0
	totalWeight := 0.0
	for _, weight := range newWeights {
		totalWeight += weight
	}
	for model := range newWeights {
		newWeights[model] /= totalWeight
	}
	
	return newWeights
}

func (ers *EnsembleRecalibrationService) generateRecalibrationReasoning(
	oldWeights, newWeights, performance map[string]float64) string {
	
	// Find biggest changes
	maxIncrease := ""
	maxIncreaseAmt := 0.0
	maxDecrease := ""
	maxDecreaseAmt := 0.0
	
	for model := range newWeights {
		change := newWeights[model] - oldWeights[model]
		if change > maxIncreaseAmt {
			maxIncreaseAmt = change
			maxIncrease = model
		}
		if change < maxDecreaseAmt {
			maxDecreaseAmt = change
			maxDecrease = model
		}
	}
	
	reasoning := fmt.Sprintf("Recalibration based on recent performance. ")
	if maxIncrease != "" && maxIncreaseAmt > 0.01 {
		reasoning += fmt.Sprintf("%s increased %.1f%% (accuracy: %.1f%%). ", 
			maxIncrease, maxIncreaseAmt*100, performance[maxIncrease]*100)
	}
	if maxDecrease != "" && maxDecreaseAmt < -0.01 {
		reasoning += fmt.Sprintf("%s decreased %.1f%% (accuracy: %.1f%%). ", 
			maxDecrease, -maxDecreaseAmt*100, performance[maxDecrease]*100)
	}
	
	return reasoning
}

func (ers *EnsembleRecalibrationService) getTotalPredictions() int {
	total := 0
	for _, perf := range ers.modelPerformance {
		if perf.TotalPredictions > total {
			total = perf.TotalPredictions
		}
	}
	return total
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (ers *EnsembleRecalibrationService) getPerformanceDataPath() string {
	return filepath.Join(ers.dataDir, "model_performance.json")
}

func (ers *EnsembleRecalibrationService) getHistoryDataPath() string {
	return filepath.Join(ers.dataDir, "recalibration_history.json")
}

func (ers *EnsembleRecalibrationService) saveRecalibrationData() error {
	// Save performance data
	perfData, err := json.MarshalIndent(ers.modelPerformance, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal performance data: %w", err)
	}
	if err := os.WriteFile(ers.getPerformanceDataPath(), perfData, 0644); err != nil {
		return fmt.Errorf("failed to write performance data: %w", err)
	}
	
	// Save history data
	histData, err := json.MarshalIndent(ers.recalibrationHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history data: %w", err)
	}
	if err := os.WriteFile(ers.getHistoryDataPath(), histData, 0644); err != nil {
		return fmt.Errorf("failed to write history data: %w", err)
	}
	
	return nil
}

func (ers *EnsembleRecalibrationService) loadRecalibrationData() error {
	// Load performance data
	perfPath := ers.getPerformanceDataPath()
	if data, err := os.ReadFile(perfPath); err == nil {
		if err := json.Unmarshal(data, &ers.modelPerformance); err != nil {
			return fmt.Errorf("failed to unmarshal performance data: %w", err)
		}
		fmt.Printf("⚖️ Loaded performance data for %d models\n", len(ers.modelPerformance))
	}
	
	// Load history data
	histPath := ers.getHistoryDataPath()
	if data, err := os.ReadFile(histPath); err == nil {
		if err := json.Unmarshal(data, &ers.recalibrationHistory); err != nil {
			return fmt.Errorf("failed to unmarshal history data: %w", err)
		}
		fmt.Printf("⚖️ Loaded %d recalibration events\n", len(ers.recalibrationHistory.Events))
	}
	
	return nil
}

