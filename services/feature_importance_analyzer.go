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
	featureImportanceInstance *FeatureImportanceAnalyzer
	featureImportanceOnce     sync.Once
)

// FeatureImportanceAnalyzer tracks which features are most important for predictions
type FeatureImportanceAnalyzer struct {
	dataDir            string
	featureImportance  map[string]*models.FeatureImportance
	featureValues      map[string][]float64 // Track all values for each feature
	correctPredictions map[string][]float64 // Feature values when prediction was correct
	wrongPredictions   map[string][]float64 // Feature values when prediction was wrong
	mu                 sync.RWMutex
}

// InitializeFeatureImportanceAnalyzer creates the feature importance analyzer
func InitializeFeatureImportanceAnalyzer() error {
	var initErr error
	featureImportanceOnce.Do(func() {
		dataDir := "data/feature_importance"
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create feature importance directory: %w", err)
			return
		}

		featureImportanceInstance = &FeatureImportanceAnalyzer{
			dataDir:            dataDir,
			featureImportance:  make(map[string]*models.FeatureImportance),
			featureValues:      make(map[string][]float64),
			correctPredictions: make(map[string][]float64),
			wrongPredictions:   make(map[string][]float64),
		}

		// Load existing importance data
		if err := featureImportanceInstance.loadImportanceData(); err != nil {
			fmt.Printf("⚠️ Warning: Could not load existing feature importance data: %v\n", err)
		}

		// Analyze Neural Network weights
		if err := featureImportanceInstance.analyzeNeuralNetworkWeights(); err != nil {
			fmt.Printf("⚠️ Warning: Could not analyze Neural Network weights: %v\n", err)
		}

		fmt.Println("✅ Feature Importance Analyzer initialized")
	})

	return initErr
}

// GetFeatureImportanceAnalyzer returns the singleton instance
func GetFeatureImportanceAnalyzer() *FeatureImportanceAnalyzer {
	return featureImportanceInstance
}

// TrackFeatureUsage records feature values for a prediction
func (fia *FeatureImportanceAnalyzer) TrackFeatureUsage(
	homeFactors, awayFactors *models.PredictionFactors,
	isCorrect bool,
) error {
	fia.mu.Lock()
	defer fia.mu.Unlock()

	// Extract all features into a map
	features := fia.extractFeatureMap(homeFactors, awayFactors)

	// Track each feature
	for featureName, value := range features {
		// Initialize if needed
		if fia.featureImportance[featureName] == nil {
			fia.featureImportance[featureName] = &models.FeatureImportance{
				FeatureName:  featureName,
				LastUpdated:  time.Now(),
			}
		}

		// Track value
		fia.featureValues[featureName] = append(fia.featureValues[featureName], value)

		// Track correct vs wrong predictions
		if isCorrect {
			fia.correctPredictions[featureName] = append(fia.correctPredictions[featureName], value)
		} else {
			fia.wrongPredictions[featureName] = append(fia.wrongPredictions[featureName], value)
		}
	}

	return nil
}

// RecalculateImportance recalculates feature importance scores
func (fia *FeatureImportanceAnalyzer) RecalculateImportance() error {
	fia.mu.Lock()
	defer fia.mu.Unlock()

	for featureName, importance := range fia.featureImportance {
		values := fia.featureValues[featureName]
		correctVals := fia.correctPredictions[featureName]
		wrongVals := fia.wrongPredictions[featureName]

		if len(values) == 0 {
			continue
		}

		// Calculate statistics
		importance.AverageValue = fia.calculateMean(values)
		importance.ValueRange = fia.calculateRange(values)
		importance.UsageFrequency = fia.calculateUsageFrequency(values)

		// Calculate correlation with correct predictions
		importance.CorrelationWithWin = fia.calculateCorrelation(correctVals, wrongVals)

		// Calculate importance score (composite metric)
		importance.ImportanceScore = fia.calculateImportanceScore(importance)

		// Check if feature helps with upsets
		importance.ContributesToUpsets = fia.detectUpsetContribution(featureName, correctVals, wrongVals)

		importance.LastUpdated = time.Now()
	}

	// Save importance data
	return fia.saveImportanceData()
}

// analyzeNeuralNetworkWeights extracts feature importance from NN weights
func (fia *FeatureImportanceAnalyzer) analyzeNeuralNetworkWeights() error {
	// TODO: Integrate with ensemble service to analyze Neural Network weights
	// This requires accessing the ensemble prediction service and extracting weight matrices
	fmt.Println("📊 Neural Network weight analysis will be available after first training")
	return nil
}

// extractFeatureMap converts PredictionFactors to a feature map
func (fia *FeatureImportanceAnalyzer) extractFeatureMap(
	homeFactors, awayFactors *models.PredictionFactors,
) map[string]float64 {
	features := make(map[string]float64)

	// Basic stats (differential) - using only known fields
	features["WinPercentageDiff"] = homeFactors.WinPercentage - awayFactors.WinPercentage
	features["GoalsForDiff"] = homeFactors.GoalsFor - awayFactors.GoalsFor
	features["GoalsAgainstDiff"] = homeFactors.GoalsAgainst - awayFactors.GoalsAgainst
	features["PowerPlayPctDiff"] = homeFactors.PowerPlayPct - awayFactors.PowerPlayPct
	features["PenaltyKillPctDiff"] = homeFactors.PenaltyKillPct - awayFactors.PenaltyKillPct

	// Recent form
	features["HomeRecentForm"] = homeFactors.RecentForm
	features["AwayRecentForm"] = awayFactors.RecentForm
	features["RecentFormDiff"] = homeFactors.RecentForm - awayFactors.RecentForm

	// Expected goals
	features["HomeExpectedGoals"] = homeFactors.ExpectedGoalsFor
	features["AwayExpectedGoals"] = awayFactors.ExpectedGoalsFor
	features["xGDiff"] = homeFactors.ExpectedGoalsFor - awayFactors.ExpectedGoalsFor

	// Travel fatigue
	features["HomeTravelFatigue"] = homeFactors.TravelFatigue.FatigueScore
	features["AwayTravelFatigue"] = awayFactors.TravelFatigue.FatigueScore
	features["FatigueDiff"] = awayFactors.TravelFatigue.FatigueScore - homeFactors.TravelFatigue.FatigueScore

	// Injury impact
	features["HomeInjuryImpact"] = homeFactors.InjuryImpact.ImpactScore
	features["AwayInjuryImpact"] = awayFactors.InjuryImpact.ImpactScore
	features["InjuryDiff"] = awayFactors.InjuryImpact.ImpactScore - homeFactors.InjuryImpact.ImpactScore

	return features
}

// getFeatureNames returns ordered list of feature names matching NN input
func (fia *FeatureImportanceAnalyzer) getFeatureNames() []string {
	return []string{
		"WinPercentage_Home", "WinPercentage_Away",
		"GoalsFor_Home", "GoalsAgainst_Home", "GoalsFor_Away", "GoalsAgainst_Away",
		"PowerPlayPct_Home", "PowerPlayPct_Away",
		"PenaltyKillPct_Home", "PenaltyKillPct_Away",
		"GoalieSavePct_Home", "GoalieSavePct_Away",
		"CorsiFor_Home", "CorsiFor_Away",
		"ExpectedGoals_Home", "ExpectedGoals_Away",
		"RecentForm_Home", "RecentForm_Away",
		"HomeAdvantage",
		"ShootingPct_Home", "ShootingPct_Away",
		"PDO_Home", "PDO_Away",
		"WinStreak_Home", "WinStreak_Away",
		// ... (abbreviated for brevity - would include all 168 features)
	}
}

// calculateMean calculates the mean of a slice of values
func (fia *FeatureImportanceAnalyzer) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateRange calculates max - min
func (fia *FeatureImportanceAnalyzer) calculateRange(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return max - min
}

// calculateUsageFrequency calculates % of times feature is non-zero
func (fia *FeatureImportanceAnalyzer) calculateUsageFrequency(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	nonZero := 0
	for _, v := range values {
		if math.Abs(v) > 0.001 {
			nonZero++
		}
	}
	return float64(nonZero) / float64(len(values))
}

// calculateCorrelation calculates correlation between feature and correct predictions
func (fia *FeatureImportanceAnalyzer) calculateCorrelation(correctVals, wrongVals []float64) float64 {
	if len(correctVals) == 0 || len(wrongVals) == 0 {
		return 0.0
	}

	// Simple approach: compare mean values
	correctMean := fia.calculateMean(correctVals)
	wrongMean := fia.calculateMean(wrongVals)

	// Calculate difference (positive = higher values correlate with correct predictions)
	diff := correctMean - wrongMean

	// Normalize to [-1, 1] range based on value range
	allVals := append(correctVals, wrongVals...)
	valueRange := fia.calculateRange(allVals)
	if valueRange > 0 {
		return math.Max(-1.0, math.Min(1.0, diff/valueRange))
	}

	return 0.0
}

// calculateImportanceScore computes composite importance score
func (fia *FeatureImportanceAnalyzer) calculateImportanceScore(importance *models.FeatureImportance) float64 {
	// Weighted combination of factors
	score := 0.0

	// Neural network weight (0-40% of score)
	score += importance.NeuralNetWeight * 0.4

	// Correlation with correct predictions (0-30% of score)
	score += math.Abs(importance.CorrelationWithWin) * 0.3

	// Usage frequency (0-20% of score)
	score += importance.UsageFrequency * 0.2

	// Value range (0-10% of score) - higher range = more potential impact
	normalizedRange := math.Min(1.0, importance.ValueRange/10.0)
	score += normalizedRange * 0.1

	return math.Min(1.0, score)
}

// detectUpsetContribution checks if feature helps predict upsets
func (fia *FeatureImportanceAnalyzer) detectUpsetContribution(
	featureName string,
	correctVals, wrongVals []float64,
) bool {
	// This is a placeholder - would need more sophisticated logic
	// Look for features that have different distributions in upset games
	return false
}

// GetTopFeatures returns the N most important features
func (fia *FeatureImportanceAnalyzer) GetTopFeatures(n int) []*models.FeatureImportance {
	fia.mu.RLock()
	defer fia.mu.RUnlock()

	features := make([]*models.FeatureImportance, 0, len(fia.featureImportance))
	for _, importance := range fia.featureImportance {
		features = append(features, importance)
	}

	// Sort by importance score
	sort.Slice(features, func(i, j int) bool {
		return features[i].ImportanceScore > features[j].ImportanceScore
	})

	if n > len(features) {
		n = len(features)
	}

	return features[:n]
}

// GetAllFeatures returns all feature importance data
func (fia *FeatureImportanceAnalyzer) GetAllFeatures() []*models.FeatureImportance {
	fia.mu.RLock()
	defer fia.mu.RUnlock()

	features := make([]*models.FeatureImportance, 0, len(fia.featureImportance))
	for _, importance := range fia.featureImportance {
		features = append(features, importance)
	}

	// Sort by importance score
	sort.Slice(features, func(i, j int) bool {
		return features[i].ImportanceScore > features[j].ImportanceScore
	})

	return features
}

// GetLowValueFeatures returns features with low importance scores
func (fia *FeatureImportanceAnalyzer) GetLowValueFeatures(threshold float64) []string {
	fia.mu.RLock()
	defer fia.mu.RUnlock()

	lowValue := make([]string, 0)
	for featureName, importance := range fia.featureImportance {
		if importance.ImportanceScore < threshold {
			lowValue = append(lowValue, featureName)
		}
	}

	return lowValue
}

// File operations

func (fia *FeatureImportanceAnalyzer) saveImportanceData() error {
	filename := filepath.Join(fia.dataDir, "feature_importance.json")
	data, err := json.MarshalIndent(fia.featureImportance, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (fia *FeatureImportanceAnalyzer) loadImportanceData() error {
	filename := filepath.Join(fia.dataDir, "feature_importance.json")
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data yet, that's okay
		}
		return err
	}

	return json.Unmarshal(data, &fia.featureImportance)
}

