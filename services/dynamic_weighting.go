package services

import (
	"fmt"
	"log"
	"math"
	"time"
)

// ModelPerformanceTracker tracks detailed performance metrics for dynamic weighting
type ModelPerformanceTracker struct {
	modelName         string
	accuracyHistory   []AccuracyRecord
	contextualMetrics map[string]*ContextualPerformance
	recentPerformance *RecentPerformanceWindow
	lastUpdate        time.Time
	performanceCache  map[string]float64
}

// AccuracyRecord represents a single prediction outcome with context
type AccuracyRecord struct {
	PredictionID     string      `json:"predictionId"`
	GameDate         time.Time   `json:"gameDate"`
	HomeTeam         string      `json:"homeTeam"`
	AwayTeam         string      `json:"awayTeam"`
	PredictedWinner  string      `json:"predictedWinner"`
	ActualWinner     string      `json:"actualWinner"`
	WinProbability   float64     `json:"winProbability"`
	Confidence       float64     `json:"confidence"`
	IsCorrect        bool        `json:"isCorrect"`
	ProbabilityError float64     `json:"probabilityError"`
	GameContext      GameContext `json:"gameContext"`
	ProcessingTime   int64       `json:"processingTime"`
	RecordedAt       time.Time   `json:"recordedAt"`
	Weight           float64     `json:"weight"` // Recency weight for this record
}

// GameContext provides situational context for the prediction
type GameContext struct {
	IsHomeGame         bool    `json:"isHomeGame"`
	IsPlayoffGame      bool    `json:"isPlayoffGame"`
	TeamStrengthGap    float64 `json:"teamStrengthGap"` // Win% difference
	IsUpsetPrediction  bool    `json:"isUpsetPrediction"`
	IsBackToBack       bool    `json:"isBackToBack"`
	RestDaysAdvantage  int     `json:"restDaysAdvantage"`
	TravelDistance     float64 `json:"travelDistance"`
	AltitudeDifference float64 `json:"altitudeDifference"`
	GameImportance     string  `json:"gameImportance"` // "low", "medium", "high", "critical"
	OpponentType       string  `json:"opponentType"`   // "weak", "average", "strong", "elite"
}

// ContextualPerformance tracks performance in specific contexts
type ContextualPerformance struct {
	Context             string    `json:"context"` // e.g., "home_games", "upset_predictions"
	TotalPredictions    int       `json:"totalPredictions"`
	CorrectPredictions  int       `json:"correctPredictions"`
	Accuracy            float64   `json:"accuracy"`
	AvgConfidence       float64   `json:"avgConfidence"`
	AvgProbabilityError float64   `json:"avgProbabilityError"`
	LastUpdated         time.Time `json:"lastUpdated"`
	Trend               string    `json:"trend"`      // "improving", "stable", "declining"
	SampleSize          int       `json:"sampleSize"` // For statistical significance
}

// RecentPerformanceWindow tracks performance in sliding time windows
type RecentPerformanceWindow struct {
	Window5Games        *WindowStats `json:"window5Games"`        // Last 5 predictions
	Window10Games       *WindowStats `json:"window10Games"`       // Last 10 predictions
	Window20Games       *WindowStats `json:"window20Games"`       // Last 20 predictions
	Window7Days         *WindowStats `json:"window7Days"`         // Last 7 days
	Window30Days        *WindowStats `json:"window30Days"`        // Last 30 days
	CurrentStreak       int          `json:"currentStreak"`       // Current correct/incorrect streak
	LongestStreak       int          `json:"longestStreak"`       // Longest correct streak
	WeightedAccuracy    float64      `json:"weightedAccuracy"`    // Recency-weighted accuracy
	PerformanceVelocity float64      `json:"performanceVelocity"` // Rate of improvement/decline
}

// WindowStats represents performance statistics for a time/game window
type WindowStats struct {
	WindowSize         int       `json:"windowSize"`
	Predictions        int       `json:"predictions"`
	Correct            int       `json:"correct"`
	Accuracy           float64   `json:"accuracy"`
	AvgConfidence      float64   `json:"avgConfidence"`
	AvgProbError       float64   `json:"avgProbError"`
	ConfidenceAccuracy float64   `json:"confidenceAccuracy"` // How well-calibrated confidence was
	LastUpdated        time.Time `json:"lastUpdated"`
}

// ModelWeightCalculator handles dynamic weight calculation based on performance
type ModelWeightCalculator struct {
	trackers          map[string]*ModelPerformanceTracker
	baseWeights       map[string]float64 // Original fixed weights
	currentWeights    map[string]float64 // Current dynamic weights
	weightConstraints WeightConstraints
	smoothingFactor   float64            // How much to smooth weight changes
	recencyFactor     float64            // How much recent performance matters
	contextWeights    map[string]float64 // Weights for different contexts
	lastWeightUpdate  time.Time
	weightHistory     []WeightSnapshot
}

// WeightConstraints define limits for weight adjustments
type WeightConstraints struct {
	MinWeight         float64 `json:"minWeight"`         // Minimum weight for any model (e.g., 0.15)
	MaxWeight         float64 `json:"maxWeight"`         // Maximum weight for any model (e.g., 0.6)
	MaxShiftPerUpdate float64 `json:"maxShiftPerUpdate"` // Max weight change per update (e.g., 0.05)
	MinSampleSize     int     `json:"minSampleSize"`     // Min predictions before adjusting (e.g., 10)
}

// WeightSnapshot captures weight changes over time
type WeightSnapshot struct {
	Timestamp       time.Time          `json:"timestamp"`
	Weights         map[string]float64 `json:"weights"`
	TriggerReason   string             `json:"triggerReason"`
	PerformanceData map[string]float64 `json:"performanceData"` // Accuracy that triggered change
}

// DynamicWeightingService orchestrates the entire dynamic weighting system
type DynamicWeightingService struct {
	calculator          *ModelWeightCalculator
	performanceTrackers map[string]*ModelPerformanceTracker
	updateInterval      time.Duration
	lastFullUpdate      time.Time
	isEnabled           bool
	settings            DynamicWeightingSettings
}

// DynamicWeightingSettings configures the dynamic weighting behavior
type DynamicWeightingSettings struct {
	EnableDynamicWeights  bool          `json:"enableDynamicWeights"`
	RecencyDecayRate      float64       `json:"recencyDecayRate"`      // How quickly old data loses importance
	PerformanceThreshold  float64       `json:"performanceThreshold"`  // Min accuracy difference to trigger weight change
	SmoothingStrength     float64       `json:"smoothingStrength"`     // How gradual weight transitions are
	ContextualWeighting   bool          `json:"contextualWeighting"`   // Whether to use context-specific weights
	AdaptationSpeed       string        `json:"adaptationSpeed"`       // "conservative", "moderate", "aggressive"
	MinEvaluationPeriod   time.Duration `json:"minEvaluationPeriod"`   // Min time before first weight adjustment
	WeightUpdateFrequency time.Duration `json:"weightUpdateFrequency"` // How often to recalculate weights
}

// NewDynamicWeightingService creates a new dynamic weighting service
func NewDynamicWeightingService() *DynamicWeightingService {
	// Default settings for conservative but effective weighting
	settings := DynamicWeightingSettings{
		EnableDynamicWeights:  true,
		RecencyDecayRate:      0.95, // 5% decay per prediction
		PerformanceThreshold:  0.05, // 5% accuracy difference triggers change
		SmoothingStrength:     0.3,  // Moderate smoothing
		ContextualWeighting:   true,
		AdaptationSpeed:       "moderate",
		MinEvaluationPeriod:   24 * time.Hour, // Wait 1 day before adjusting
		WeightUpdateFrequency: time.Hour,      // Check hourly
	}

	constraints := WeightConstraints{
		MinWeight:         0.15, // No model below 15%
		MaxWeight:         0.6,  // No model above 60%
		MaxShiftPerUpdate: 0.03, // Max 3% change per update
		MinSampleSize:     15,   // Need 15 predictions minimum
	}

	calculator := &ModelWeightCalculator{
		trackers: make(map[string]*ModelPerformanceTracker),
		baseWeights: map[string]float64{
			"Enhanced Statistical":   0.35,
			"Bayesian Inference":     0.15,
			"Monte Carlo Simulation": 0.10,
			"Elo Rating":             0.20,
			"Poisson Regression":     0.15,
			"Neural Network":         0.05, // New model, starting conservative
		},
		currentWeights: map[string]float64{
			"Enhanced Statistical":   0.35,
			"Bayesian Inference":     0.15,
			"Monte Carlo Simulation": 0.10,
			"Elo Rating":             0.20,
			"Poisson Regression":     0.15,
			"Neural Network":         0.05, // New model, starting conservative
		},
		weightConstraints: constraints,
		smoothingFactor:   settings.SmoothingStrength,
		recencyFactor:     settings.RecencyDecayRate,
		contextWeights:    make(map[string]float64),
		weightHistory:     make([]WeightSnapshot, 0),
	}

	return &DynamicWeightingService{
		calculator:          calculator,
		performanceTrackers: make(map[string]*ModelPerformanceTracker),
		updateInterval:      settings.WeightUpdateFrequency,
		isEnabled:           settings.EnableDynamicWeights,
		settings:            settings,
	}
}

// RecordPredictionOutcome records the result of a prediction for weight calculation
func (dws *DynamicWeightingService) RecordPredictionOutcome(modelName string, record AccuracyRecord) error {
	if !dws.isEnabled {
		return nil // Dynamic weighting disabled
	}

	tracker, exists := dws.performanceTrackers[modelName]
	if !exists {
		tracker = &ModelPerformanceTracker{
			modelName:         modelName,
			accuracyHistory:   make([]AccuracyRecord, 0),
			contextualMetrics: make(map[string]*ContextualPerformance),
			performanceCache:  make(map[string]float64),
			recentPerformance: &RecentPerformanceWindow{},
		}
		dws.performanceTrackers[modelName] = tracker
	}

	// Calculate recency weight for this record
	record.Weight = dws.calculateRecencyWeight(len(tracker.accuracyHistory))

	// Add to history
	tracker.accuracyHistory = append(tracker.accuracyHistory, record)

	// Maintain rolling window (keep last 100 predictions)
	if len(tracker.accuracyHistory) > 100 {
		tracker.accuracyHistory = tracker.accuracyHistory[1:]
	}

	// Update performance metrics
	dws.updatePerformanceMetrics(tracker)

	// Update contextual performance
	dws.updateContextualPerformance(tracker, record)

	tracker.lastUpdate = time.Now()

	log.Printf("📊 Recorded %s prediction: %s vs %s, Correct: %v, Confidence: %.1f%%",
		modelName, record.HomeTeam, record.AwayTeam, record.IsCorrect, record.Confidence*100)

	return nil
}

// GetCurrentWeights returns the current dynamic weights for all models
func (dws *DynamicWeightingService) GetCurrentWeights() map[string]float64 {
	if !dws.isEnabled {
		return dws.calculator.baseWeights
	}

	// Update weights if enough time has passed
	if time.Since(dws.lastFullUpdate) > dws.updateInterval {
		dws.updateWeights()
	}

	return dws.calculator.currentWeights
}

// updateWeights recalculates and updates model weights based on recent performance
func (dws *DynamicWeightingService) updateWeights() {
	log.Printf("🔄 Updating dynamic model weights...")

	newWeights := make(map[string]float64)
	performanceScores := make(map[string]float64)

	// Calculate performance scores for each model
	totalScore := 0.0
	for modelName, tracker := range dws.performanceTrackers {
		score := dws.calculatePerformanceScore(tracker)
		performanceScores[modelName] = score
		totalScore += score

		log.Printf("📈 %s performance score: %.3f", modelName, score)
	}

	// Convert scores to weights
	if totalScore > 0 {
		for modelName := range dws.calculator.baseWeights {
			if score, exists := performanceScores[modelName]; exists {
				newWeights[modelName] = score / totalScore
			} else {
				// Model has no performance data, use base weight
				newWeights[modelName] = dws.calculator.baseWeights[modelName]
			}
		}
	} else {
		// No performance data, use base weights
		newWeights = dws.calculator.baseWeights
	}

	// Apply constraints and smoothing
	dws.applyWeightConstraints(newWeights)
	dws.smoothWeightTransitions(newWeights)

	// Ensure weights sum to 1.0
	dws.normalizeWeights(newWeights)

	// Record weight change
	if dws.hasSignificantWeightChange(newWeights) {
		snapshot := WeightSnapshot{
			Timestamp:       time.Now(),
			Weights:         make(map[string]float64),
			TriggerReason:   "Performance-based update",
			PerformanceData: performanceScores,
		}

		for k, v := range newWeights {
			snapshot.Weights[k] = v
		}

		dws.calculator.weightHistory = append(dws.calculator.weightHistory, snapshot)

		// Log significant changes
		for modelName, newWeight := range newWeights {
			oldWeight := dws.calculator.currentWeights[modelName]
			change := newWeight - oldWeight
			if math.Abs(change) > 0.01 { // 1% change threshold
				log.Printf("⚖️ %s weight: %.3f → %.3f (Δ%.3f)",
					modelName, oldWeight, newWeight, change)
			}
		}
	}

	// Update current weights
	dws.calculator.currentWeights = newWeights
	dws.lastFullUpdate = time.Now()
}

// calculateRecencyWeight assigns higher weights to more recent predictions
func (dws *DynamicWeightingService) calculateRecencyWeight(position int) float64 {
	// More recent predictions get higher weights
	// Position 0 (most recent) gets weight 1.0
	// Each older position gets multiplied by decay rate
	return math.Pow(dws.settings.RecencyDecayRate, float64(position))
}

// calculatePerformanceScore calculates a comprehensive performance score for a model
func (dws *DynamicWeightingService) calculatePerformanceScore(tracker *ModelPerformanceTracker) float64 {
	if len(tracker.accuracyHistory) < dws.calculator.weightConstraints.MinSampleSize {
		// Not enough data, return base score
		return 1.0
	}

	score := 0.0
	weightSum := 0.0

	// Calculate weighted accuracy
	for _, record := range tracker.accuracyHistory {
		recordScore := 0.0
		if record.IsCorrect {
			recordScore = 1.0
		}

		// Adjust for confidence calibration
		confidenceError := math.Abs(record.Confidence - recordScore)
		calibrationBonus := 1.0 - confidenceError
		recordScore *= calibrationBonus

		score += recordScore * record.Weight
		weightSum += record.Weight
	}

	if weightSum == 0 {
		return 1.0
	}

	weightedAccuracy := score / weightSum

	// Apply performance velocity (improvement/decline trend)
	velocity := dws.calculatePerformanceVelocity(tracker)
	velocityAdjustment := 1.0 + (velocity * 0.1) // Max 10% boost/penalty

	// Apply contextual performance adjustments
	contextAdjustment := dws.calculateContextualAdjustment(tracker)

	finalScore := weightedAccuracy * velocityAdjustment * contextAdjustment

	log.Printf("📊 %s score breakdown: Accuracy=%.3f, Velocity=%.3f, Context=%.3f → Final=%.3f",
		tracker.modelName, weightedAccuracy, velocityAdjustment, contextAdjustment, finalScore)

	return math.Max(0.1, finalScore) // Minimum score to prevent zero weights
}

// calculatePerformanceVelocity determines if model performance is improving or declining
func (dws *DynamicWeightingService) calculatePerformanceVelocity(tracker *ModelPerformanceTracker) float64 {
	history := tracker.accuracyHistory
	if len(history) < 10 {
		return 0.0 // Not enough data for trend analysis
	}

	// Compare recent half vs older half
	midpoint := len(history) / 2
	recentHalf := history[midpoint:]
	olderHalf := history[:midpoint]

	recentAccuracy := dws.calculateWindowAccuracy(recentHalf)
	olderAccuracy := dws.calculateWindowAccuracy(olderHalf)

	// Velocity = (recent - older) / time_span
	velocity := recentAccuracy - olderAccuracy

	return math.Max(-0.5, math.Min(0.5, velocity)) // Cap at ±50%
}

// calculateWindowAccuracy calculates accuracy for a subset of predictions
func (dws *DynamicWeightingService) calculateWindowAccuracy(records []AccuracyRecord) float64 {
	if len(records) == 0 {
		return 0.0
	}

	correct := 0.0
	for _, record := range records {
		if record.IsCorrect {
			correct += 1.0
		}
	}

	return correct / float64(len(records))
}

// calculateContextualAdjustment adjusts score based on performance in different contexts
func (dws *DynamicWeightingService) calculateContextualAdjustment(tracker *ModelPerformanceTracker) float64 {
	if !dws.settings.ContextualWeighting {
		return 1.0
	}

	adjustment := 1.0

	// Check performance in key contexts
	for _, performance := range tracker.contextualMetrics {
		if performance.SampleSize < 5 {
			continue // Not enough data for this context
		}

		contextWeight := 0.1 // Each context contributes up to 10%
		if performance.Accuracy > 0.6 {
			adjustment += contextWeight * (performance.Accuracy - 0.5) // Bonus for good performance
		} else if performance.Accuracy < 0.4 {
			adjustment -= contextWeight * (0.5 - performance.Accuracy) // Penalty for poor performance
		}
	}

	return math.Max(0.5, math.Min(1.5, adjustment)) // Cap adjustment at ±50%
}

// updatePerformanceMetrics updates rolling window statistics
func (dws *DynamicWeightingService) updatePerformanceMetrics(tracker *ModelPerformanceTracker) {
	history := tracker.accuracyHistory
	if len(history) == 0 {
		return
	}

	// Update recent performance windows
	rp := tracker.recentPerformance
	if rp.Window5Games == nil {
		rp.Window5Games = &WindowStats{}
		rp.Window10Games = &WindowStats{}
		rp.Window20Games = &WindowStats{}
		rp.Window7Days = &WindowStats{}
		rp.Window30Days = &WindowStats{}
	}

	// Update game-based windows
	rp.Window5Games = dws.calculateWindowStats(history, 5, false)
	rp.Window10Games = dws.calculateWindowStats(history, 10, false)
	rp.Window20Games = dws.calculateWindowStats(history, 20, false)

	// Update time-based windows
	rp.Window7Days = dws.calculateTimeWindowStats(history, 7*24*time.Hour)
	rp.Window30Days = dws.calculateTimeWindowStats(history, 30*24*time.Hour)

	// Update streaks
	rp.CurrentStreak, rp.LongestStreak = dws.calculateStreaks(history)

	// Update weighted accuracy
	rp.WeightedAccuracy = dws.calculateWeightedAccuracy(history)

	// Update performance velocity
	rp.PerformanceVelocity = dws.calculatePerformanceVelocity(tracker)
}

// calculateWindowStats calculates statistics for a specific window
func (dws *DynamicWeightingService) calculateWindowStats(history []AccuracyRecord, windowSize int, isTimeWindow bool) *WindowStats {
	if len(history) == 0 {
		return &WindowStats{WindowSize: windowSize}
	}

	// Get relevant records
	var records []AccuracyRecord
	if len(history) <= windowSize || windowSize == 0 {
		records = history
	} else {
		records = history[len(history)-windowSize:]
	}

	stats := &WindowStats{
		WindowSize:  windowSize,
		Predictions: len(records),
		LastUpdated: time.Now(),
	}

	if len(records) == 0 {
		return stats
	}

	// Calculate metrics
	correct := 0
	totalConfidence := 0.0
	totalProbError := 0.0
	calibrationSum := 0.0

	for _, record := range records {
		if record.IsCorrect {
			correct++
		}
		totalConfidence += record.Confidence
		totalProbError += record.ProbabilityError

		// Confidence calibration: how close was confidence to actual outcome
		actualOutcome := 0.0
		if record.IsCorrect {
			actualOutcome = 1.0
		}
		calibrationError := math.Abs(record.Confidence - actualOutcome)
		calibrationSum += (1.0 - calibrationError)
	}

	stats.Correct = correct
	stats.Accuracy = float64(correct) / float64(len(records))
	stats.AvgConfidence = totalConfidence / float64(len(records))
	stats.AvgProbError = totalProbError / float64(len(records))
	stats.ConfidenceAccuracy = calibrationSum / float64(len(records))

	return stats
}

// calculateTimeWindowStats calculates statistics for a time-based window
func (dws *DynamicWeightingService) calculateTimeWindowStats(history []AccuracyRecord, duration time.Duration) *WindowStats {
	cutoff := time.Now().Add(-duration)

	var recentRecords []AccuracyRecord
	for _, record := range history {
		if record.RecordedAt.After(cutoff) {
			recentRecords = append(recentRecords, record)
		}
	}

	return dws.calculateWindowStats(recentRecords, 0, true)
}

// calculateStreaks calculates current and longest correct prediction streaks
func (dws *DynamicWeightingService) calculateStreaks(history []AccuracyRecord) (int, int) {
	if len(history) == 0 {
		return 0, 0
	}

	currentStreak := 0
	longestStreak := 0
	tempStreak := 0

	// Calculate current streak (from most recent)
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].IsCorrect {
			currentStreak++
		} else {
			break
		}
	}

	// Calculate longest streak
	for _, record := range history {
		if record.IsCorrect {
			tempStreak++
			if tempStreak > longestStreak {
				longestStreak = tempStreak
			}
		} else {
			tempStreak = 0
		}
	}

	return currentStreak, longestStreak
}

// calculateWeightedAccuracy calculates recency-weighted accuracy
func (dws *DynamicWeightingService) calculateWeightedAccuracy(history []AccuracyRecord) float64 {
	if len(history) == 0 {
		return 0.0
	}

	weightedSum := 0.0
	totalWeight := 0.0

	for _, record := range history {
		if record.IsCorrect {
			weightedSum += record.Weight
		}
		totalWeight += record.Weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSum / totalWeight
}

// updateContextualPerformance updates performance metrics for specific contexts
func (dws *DynamicWeightingService) updateContextualPerformance(tracker *ModelPerformanceTracker, record AccuracyRecord) {
	contexts := dws.extractContexts(record)

	for _, context := range contexts {
		perf, exists := tracker.contextualMetrics[context]
		if !exists {
			perf = &ContextualPerformance{
				Context: context,
			}
			tracker.contextualMetrics[context] = perf
		}

		perf.TotalPredictions++
		if record.IsCorrect {
			perf.CorrectPredictions++
		}

		perf.Accuracy = float64(perf.CorrectPredictions) / float64(perf.TotalPredictions)
		perf.LastUpdated = time.Now()
		perf.SampleSize = perf.TotalPredictions
	}
}

// extractContexts identifies relevant contexts for a prediction
func (dws *DynamicWeightingService) extractContexts(record AccuracyRecord) []string {
	var contexts []string

	ctx := record.GameContext

	// Home/Away context
	if ctx.IsHomeGame {
		contexts = append(contexts, "home_games")
	} else {
		contexts = append(contexts, "away_games")
	}

	// Playoff context
	if ctx.IsPlayoffGame {
		contexts = append(contexts, "playoff_games")
	} else {
		contexts = append(contexts, "regular_season")
	}

	// Upset prediction context
	if ctx.IsUpsetPrediction {
		contexts = append(contexts, "upset_predictions")
	} else {
		contexts = append(contexts, "favorite_predictions")
	}

	// Opponent strength context
	contexts = append(contexts, fmt.Sprintf("vs_%s_teams", ctx.OpponentType))

	// Back-to-back context
	if ctx.IsBackToBack {
		contexts = append(contexts, "back_to_back")
	}

	// Team strength gap context
	if ctx.TeamStrengthGap > 0.2 {
		contexts = append(contexts, "large_strength_gap")
	} else if ctx.TeamStrengthGap < 0.05 {
		contexts = append(contexts, "even_matchup")
	}

	return contexts
}

// applyWeightConstraints ensures weights stay within defined limits
func (dws *DynamicWeightingService) applyWeightConstraints(weights map[string]float64) {
	constraints := dws.calculator.weightConstraints

	for modelName, weight := range weights {
		// Apply min/max constraints
		if weight < constraints.MinWeight {
			weights[modelName] = constraints.MinWeight
		} else if weight > constraints.MaxWeight {
			weights[modelName] = constraints.MaxWeight
		}

		// Apply max shift constraint
		currentWeight := dws.calculator.currentWeights[modelName]
		maxChange := constraints.MaxShiftPerUpdate

		if weight > currentWeight+maxChange {
			weights[modelName] = currentWeight + maxChange
		} else if weight < currentWeight-maxChange {
			weights[modelName] = currentWeight - maxChange
		}
	}
}

// smoothWeightTransitions applies smoothing to prevent abrupt weight changes
func (dws *DynamicWeightingService) smoothWeightTransitions(newWeights map[string]float64) {
	smoothing := dws.calculator.smoothingFactor

	for modelName, newWeight := range newWeights {
		currentWeight := dws.calculator.currentWeights[modelName]

		// Apply exponential smoothing
		smoothedWeight := (smoothing * newWeight) + ((1.0 - smoothing) * currentWeight)
		newWeights[modelName] = smoothedWeight
	}
}

// normalizeWeights ensures all weights sum to 1.0
func (dws *DynamicWeightingService) normalizeWeights(weights map[string]float64) {
	sum := 0.0
	for _, weight := range weights {
		sum += weight
	}

	if sum > 0 {
		for modelName := range weights {
			weights[modelName] /= sum
		}
	}
}

// hasSignificantWeightChange determines if weight changes are worth logging
func (dws *DynamicWeightingService) hasSignificantWeightChange(newWeights map[string]float64) bool {
	threshold := 0.01 // 1% change threshold

	for modelName, newWeight := range newWeights {
		currentWeight := dws.calculator.currentWeights[modelName]
		if math.Abs(newWeight-currentWeight) > threshold {
			return true
		}
	}

	return false
}

// GetPerformanceMetrics returns detailed performance metrics for analysis
func (dws *DynamicWeightingService) GetPerformanceMetrics() map[string]*ModelPerformanceTracker {
	return dws.performanceTrackers
}

// GetWeightHistory returns the history of weight changes
func (dws *DynamicWeightingService) GetWeightHistory() []WeightSnapshot {
	return dws.calculator.weightHistory
}

// IsEnabled returns whether dynamic weighting is currently enabled
func (dws *DynamicWeightingService) IsEnabled() bool {
	return dws.isEnabled
}

// SetEnabled enables or disables dynamic weighting
func (dws *DynamicWeightingService) SetEnabled(enabled bool) {
	dws.isEnabled = enabled
	if !enabled {
		// Reset to base weights
		dws.calculator.currentWeights = dws.calculator.baseWeights
		log.Printf("🔄 Dynamic weighting disabled - reverted to base weights")
	} else {
		log.Printf("🔄 Dynamic weighting enabled")
	}
}
