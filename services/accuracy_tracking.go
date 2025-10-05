package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// AccuracyTrackingService handles historical accuracy tracking and confidence boosting
type AccuracyTrackingService struct {
	dataDir              string
	accuracyData         []models.PredictionAccuracy
	modelStats           map[string]*models.ModelAccuracyStats
	ensembleStats        *models.EnsembleAccuracyStats
	lastUpdate           time.Time
	confidenceBoostCache map[string]*models.ConfidenceBoostFactors
}

// NewAccuracyTrackingService creates a new accuracy tracking service
func NewAccuracyTrackingService() *AccuracyTrackingService {
	service := &AccuracyTrackingService{
		dataDir:              "data/accuracy",
		modelStats:           make(map[string]*models.ModelAccuracyStats),
		confidenceBoostCache: make(map[string]*models.ConfidenceBoostFactors),
	}

	// Create data directory if it doesn't exist
	os.MkdirAll(service.dataDir, 0755)

	// Load existing accuracy data
	service.loadAccuracyData()
	service.calculateModelStats()

	return service
}

// RecordPrediction records a new prediction for future accuracy tracking
func (ats *AccuracyTrackingService) RecordPrediction(prediction *models.GamePrediction, factors *models.PredictionFactors) error {
	log.Printf("📝 Recording prediction: %s vs %s", prediction.HomeTeam.Code, prediction.AwayTeam.Code)

	// Record prediction for each model
	for _, modelResult := range prediction.Prediction.ModelResults {
		accuracy := models.PredictionAccuracy{
			ModelName:         modelResult.ModelName,
			PredictionDate:    time.Now(),
			GameDate:          prediction.GameDate,
			HomeTeam:          prediction.HomeTeam.Code,
			AwayTeam:          prediction.AwayTeam.Code,
			PredictedWinner:   prediction.Prediction.Winner,
			PredictedScore:    modelResult.PredictedScore,
			WinProbability:    modelResult.WinProbability,
			Confidence:        modelResult.Confidence,
			GameType:          "regular", // Could be determined from game data
			PredictionFactors: *factors,
		}

		ats.accuracyData = append(ats.accuracyData, accuracy)
	}

	// Save to disk
	return ats.saveAccuracyData()
}

// UpdateGameResult updates prediction accuracy with actual game results
func (ats *AccuracyTrackingService) UpdateGameResult(homeTeam, awayTeam string, gameDate time.Time, actualResult *models.ActualGameFactors) error {
	log.Printf("🎯 Updating game result: %s vs %s", homeTeam, awayTeam)

	updated := false

	// Find and update matching predictions
	for i := range ats.accuracyData {
		prediction := &ats.accuracyData[i]

		// Match by teams and date (within 24 hours)
		if prediction.HomeTeam == homeTeam &&
			prediction.AwayTeam == awayTeam &&
			math.Abs(prediction.GameDate.Sub(gameDate).Hours()) < 24 {

			// Determine actual winner
			if actualResult.HomeGoals > actualResult.AwayGoals {
				prediction.ActualWinner = homeTeam
			} else {
				prediction.ActualWinner = awayTeam
			}

			// Check if prediction was correct
			prediction.IsCorrect = (prediction.PredictedWinner == prediction.ActualWinner)

			// Calculate score accuracy
			prediction.ScoreAccuracy = ats.calculateScoreAccuracy(prediction.PredictedScore, actualResult)

			// Calculate probability error
			actualProbability := 1.0 // Winner gets 1.0, loser gets 0.0
			if prediction.PredictedWinner != prediction.ActualWinner {
				actualProbability = 0.0
			}
			prediction.ProbabilityError = math.Abs(prediction.WinProbability - actualProbability)

			// Calculate confidence calibration
			prediction.ConfidenceCalibration = ats.calculateConfidenceCalibration(prediction.Confidence, prediction.IsCorrect)

			// Store actual factors
			prediction.ActualFactors = *actualResult
			prediction.ActualScore = fmt.Sprintf("%d-%d", actualResult.HomeGoals, actualResult.AwayGoals)

			updated = true
		}
	}

	if updated {
		// Recalculate model stats
		ats.calculateModelStats()

		// Save updated data
		return ats.saveAccuracyData()
	}

	return nil
}

// GetConfidenceBoost calculates confidence boost based on historical accuracy
func (ats *AccuracyTrackingService) GetConfidenceBoost(modelName string, factors *models.PredictionFactors) *models.ConfidenceBoostFactors {
	cacheKey := fmt.Sprintf("%s_%s_%s", modelName, factors.TeamCode, time.Now().Format("2006-01-02"))

	// Check cache first
	if cached, exists := ats.confidenceBoostCache[cacheKey]; exists {
		return cached
	}

	boost := &models.ConfidenceBoostFactors{}

	// Model consensus (how much models typically agree)
	boost.ModelConsensus = ats.calculateModelConsensus(factors)

	// Historical accuracy for this model
	if stats, exists := ats.modelStats[modelName]; exists {
		boost.HistoricalAccuracy = stats.RecentAccuracy
	} else {
		boost.HistoricalAccuracy = 0.75 // Default for new models
	}

	// Data quality assessment
	boost.DataQuality = ats.assessDataQuality(factors)

	// Situational clarity (how clear-cut the situation is)
	boost.SituationalClarity = ats.assessSituationalClarity(factors)

	// Factor strength (how strong the predictive factors are)
	boost.FactorStrength = ats.assessFactorStrength(factors)

	// Market consensus (would be implemented with betting data)
	boost.MarketConsensus = 0.0 // No market data available

	// Injury clarity
	boost.InjuryClarity = ats.assessInjuryClarity(factors)

	// Weather stability (would be implemented with weather data)
	boost.WeatherStability = 0.0 // No weather data available

	// Venue advantage clarity
	boost.VenueAdvantage = ats.assessVenueAdvantage(factors)

	// Motivational factors
	boost.MotivationalFactors = ats.assessMotivationalFactors(factors)

	// Calculate overall confidence boost
	boost.OverallConfidenceBoost = ats.calculateOverallConfidenceBoost(boost)

	// Cache the result
	ats.confidenceBoostCache[cacheKey] = boost

	log.Printf("🚀 Confidence boost for %s: %.2fx (Historical: %.1f%%, Data Quality: %.1f%%)",
		modelName, boost.OverallConfidenceBoost, boost.HistoricalAccuracy*100, boost.DataQuality*100)

	return boost
}

// GetModelAccuracyStats returns accuracy statistics for a specific model
func (ats *AccuracyTrackingService) GetModelAccuracyStats(modelName string) *models.ModelAccuracyStats {
	if stats, exists := ats.modelStats[modelName]; exists {
		return stats
	}
	return nil
}

// GetEnsembleAccuracyStats returns overall ensemble accuracy statistics
func (ats *AccuracyTrackingService) GetEnsembleAccuracyStats() *models.EnsembleAccuracyStats {
	return ats.ensembleStats
}

// loadAccuracyData loads historical accuracy data from disk
func (ats *AccuracyTrackingService) loadAccuracyData() error {
	filePath := filepath.Join(ats.dataDir, "accuracy_data.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("📊 No existing accuracy data found, starting fresh")
		return nil
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading accuracy data: %v", err)
	}

	err = json.Unmarshal(data, &ats.accuracyData)
	if err != nil {
		return fmt.Errorf("error unmarshaling accuracy data: %v", err)
	}

	log.Printf("📊 Loaded %d historical predictions", len(ats.accuracyData))
	return nil
}

// saveAccuracyData saves accuracy data to disk
func (ats *AccuracyTrackingService) saveAccuracyData() error {
	filePath := filepath.Join(ats.dataDir, "accuracy_data.json")

	data, err := json.MarshalIndent(ats.accuracyData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling accuracy data: %v", err)
	}

	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing accuracy data: %v", err)
	}

	return nil
}

// calculateModelStats calculates accuracy statistics for each model
func (ats *AccuracyTrackingService) calculateModelStats() {
	modelData := make(map[string][]models.PredictionAccuracy)

	// Group predictions by model
	for _, prediction := range ats.accuracyData {
		if prediction.ActualWinner != "" { // Only include completed games
			modelData[prediction.ModelName] = append(modelData[prediction.ModelName], prediction)
		}
	}

	// Calculate stats for each model
	for modelName, predictions := range modelData {
		stats := ats.calculateStatsForModel(modelName, predictions)
		ats.modelStats[modelName] = stats
	}

	// Calculate ensemble stats
	ats.calculateEnsembleStats()
}

// calculateStatsForModel calculates detailed statistics for a single model
func (ats *AccuracyTrackingService) calculateStatsForModel(modelName string, predictions []models.PredictionAccuracy) *models.ModelAccuracyStats {
	if len(predictions) == 0 {
		return &models.ModelAccuracyStats{
			ModelName:   modelName,
			LastUpdated: time.Now(),
		}
	}

	stats := &models.ModelAccuracyStats{
		ModelName:        modelName,
		TotalPredictions: len(predictions),
		LastUpdated:      time.Now(),
	}

	// Sort by date for streak and recent accuracy calculations
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].GameDate.Before(predictions[j].GameDate)
	})

	// Calculate basic accuracy
	correctCount := 0
	totalScoreAccuracy := 0.0
	totalProbError := 0.0
	totalConfidenceCalibration := 0.0

	for _, p := range predictions {
		if p.IsCorrect {
			correctCount++
		}
		totalScoreAccuracy += p.ScoreAccuracy
		totalProbError += p.ProbabilityError
		totalConfidenceCalibration += p.ConfidenceCalibration
	}

	stats.CorrectPredictions = correctCount
	stats.WinnerAccuracy = float64(correctCount) / float64(len(predictions))
	stats.AverageScoreAccuracy = totalScoreAccuracy / float64(len(predictions))
	stats.AverageProbabilityError = totalProbError / float64(len(predictions))
	stats.ConfidenceCalibration = totalConfidenceCalibration / float64(len(predictions))

	// Calculate recent accuracy (last 10 games)
	recentCount := int(math.Min(10, float64(len(predictions))))
	recentCorrect := 0
	for i := len(predictions) - recentCount; i < len(predictions); i++ {
		if predictions[i].IsCorrect {
			recentCorrect++
		}
	}
	stats.RecentAccuracy = float64(recentCorrect) / float64(recentCount)

	// Calculate streaks
	stats.StreakLength, stats.BestStreak, stats.WorstStreak = ats.calculateStreaks(predictions)

	// Calculate performance by game type
	stats.RegularSeasonAccuracy = ats.calculateAccuracyByGameType(predictions, "regular")
	stats.PlayoffAccuracy = ats.calculateAccuracyByGameType(predictions, "playoff")
	stats.PreseasonAccuracy = ats.calculateAccuracyByGameType(predictions, "preseason")

	// Calculate performance by confidence level
	stats.HighConfidenceAccuracy = ats.calculateAccuracyByConfidence(predictions, 0.8, 1.0)
	stats.MediumConfidenceAccuracy = ats.calculateAccuracyByConfidence(predictions, 0.6, 0.8)
	stats.LowConfidenceAccuracy = ats.calculateAccuracyByConfidence(predictions, 0.0, 0.6)

	// Calculate trend analysis
	stats.AccuracyTrend, stats.TrendStrength = ats.calculateAccuracyTrend(predictions)

	return stats
}

// calculateScoreAccuracy calculates how accurate a score prediction was
func (ats *AccuracyTrackingService) calculateScoreAccuracy(predictedScore string, actualResult *models.ActualGameFactors) float64 {
	// Parse predicted score
	parts := strings.Split(predictedScore, "-")
	if len(parts) != 2 {
		return 0.0
	}

	predHome, err1 := strconv.Atoi(parts[0])
	predAway, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0.0
	}

	// Calculate accuracy based on goal difference
	homeDiff := math.Abs(float64(predHome - actualResult.HomeGoals))
	awayDiff := math.Abs(float64(predAway - actualResult.AwayGoals))
	totalDiff := homeDiff + awayDiff

	// Perfect score = 1.0, each goal off reduces accuracy
	accuracy := math.Max(0.0, 1.0-totalDiff*0.2)
	return accuracy
}

// calculateConfidenceCalibration measures how well-calibrated confidence was
func (ats *AccuracyTrackingService) calculateConfidenceCalibration(confidence float64, wasCorrect bool) float64 {
	// Perfect calibration: confidence matches actual probability
	actualProb := 0.0
	if wasCorrect {
		actualProb = 1.0
	}

	// Calibration error (lower is better)
	calibrationError := math.Abs(confidence - actualProb)

	// Convert to calibration score (higher is better)
	return 1.0 - calibrationError
}

// calculateModelConsensus estimates how much models typically agree
func (ats *AccuracyTrackingService) calculateModelConsensus(factors *models.PredictionFactors) float64 {
	// Real implementation would analyze historical model agreement
	// Return low consensus when no real data available
	return 0.5
}

// assessDataQuality evaluates the quality of input data
func (ats *AccuracyTrackingService) assessDataQuality(factors *models.PredictionFactors) float64 {
	quality := 1.0

	// Penalize for insufficient rest
	if factors.RestDays < 1 {
		quality -= 0.15
	}

	// Penalize for back-to-back games
	if factors.BackToBackPenalty > 0.3 {
		quality -= 0.1
	}

	// Boost for recent form data
	if factors.RecentForm > 75 {
		quality += 0.05
	}

	return math.Max(0.3, math.Min(1.0, quality))
}

// assessSituationalClarity evaluates how clear-cut the situation is
func (ats *AccuracyTrackingService) assessSituationalClarity(factors *models.PredictionFactors) float64 {
	clarity := 0.7 // Base clarity

	// Clear talent gap increases clarity
	if factors.WinPercentage > 0.65 || factors.WinPercentage < 0.35 {
		clarity += 0.2
	}

	// Strong home advantage increases clarity
	if factors.HomeAdvantage > 0.6 {
		clarity += 0.1
	}

	// Clear recent form difference increases clarity
	if factors.RecentForm > 80 || factors.RecentForm < 60 {
		clarity += 0.1
	}

	return math.Max(0.4, math.Min(1.0, clarity))
}

// assessFactorStrength evaluates the strength of predictive factors
func (ats *AccuracyTrackingService) assessFactorStrength(factors *models.PredictionFactors) float64 {
	strength := 0.6 // Base strength

	// Strong win percentage indicates reliable factor
	if factors.WinPercentage > 0.6 || factors.WinPercentage < 0.4 {
		strength += 0.2
	}

	// Good goal differential indicates strong factor
	goalDiff := factors.GoalsFor - factors.GoalsAgainst
	if math.Abs(goalDiff) > 0.5 {
		strength += 0.15
	}

	// Strong special teams indicates predictive factor
	if factors.PowerPlayPct > 0.25 || factors.PenaltyKillPct > 0.85 {
		strength += 0.1
	}

	return math.Max(0.3, math.Min(1.0, strength))
}

// assessInjuryClarity evaluates clarity of injury situation
func (ats *AccuracyTrackingService) assessInjuryClarity(factors *models.PredictionFactors) float64 {
	// Real implementation would analyze actual injury reports
	// Return low clarity when no real data available
	return 0.5
}

// assessVenueAdvantage evaluates clarity of venue advantage
func (ats *AccuracyTrackingService) assessVenueAdvantage(factors *models.PredictionFactors) float64 {
	venueAdvantage := factors.HomeAdvantage

	// Clear venue advantage increases confidence
	if venueAdvantage > 0.7 {
		return 0.9
	} else if venueAdvantage < 0.4 {
		return 0.5 // Away team has advantage
	}

	return 0.7 // Moderate venue advantage
}

// assessMotivationalFactors evaluates motivational clarity
func (ats *AccuracyTrackingService) assessMotivationalFactors(factors *models.PredictionFactors) float64 {
	motivation := 0.7 // Base motivation clarity

	// Strong momentum increases motivational clarity
	if factors.MomentumFactors.WinStreak > 3 {
		motivation += 0.2
	} else if factors.MomentumFactors.WinStreak < -3 { // Losing streak
		motivation += 0.1 // Clear desperation
	}

	// Recent blowouts affect motivation
	if factors.MomentumFactors.RecentBlowouts > 2 {
		motivation -= 0.1 // Confidence issues
	}

	return math.Max(0.4, math.Min(1.0, motivation))
}

// calculateOverallConfidenceBoost combines all factors into final boost
func (ats *AccuracyTrackingService) calculateOverallConfidenceBoost(boost *models.ConfidenceBoostFactors) float64 {
	// Weighted combination of all factors
	weights := map[string]float64{
		"consensus":      0.25,
		"accuracy":       0.30,
		"dataQuality":    0.15,
		"clarity":        0.10,
		"factorStrength": 0.10,
		"market":         0.05,
		"venue":          0.05,
	}

	totalBoost := 0.0
	totalBoost += boost.ModelConsensus * weights["consensus"]
	totalBoost += boost.HistoricalAccuracy * weights["accuracy"]
	totalBoost += boost.DataQuality * weights["dataQuality"]
	totalBoost += boost.SituationalClarity * weights["clarity"]
	totalBoost += boost.FactorStrength * weights["factorStrength"]
	totalBoost += boost.MarketConsensus * weights["market"]
	totalBoost += boost.VenueAdvantage * weights["venue"]

	// Convert to multiplier (0.8x to 1.3x)
	multiplier := 0.8 + (totalBoost * 0.5)

	return math.Max(0.8, math.Min(1.3, multiplier))
}

// Helper functions for statistical calculations
func (ats *AccuracyTrackingService) calculateStreaks(predictions []models.PredictionAccuracy) (int, int, int) {
	if len(predictions) == 0 {
		return 0, 0, 0
	}

	currentStreak := 0
	bestStreak := 0
	worstStreak := 0
	lastResult := predictions[len(predictions)-1].IsCorrect

	// Calculate current streak (from the end)
	for i := len(predictions) - 1; i >= 0; i-- {
		if predictions[i].IsCorrect == lastResult {
			currentStreak++
		} else {
			break
		}
	}

	// Calculate best and worst streaks
	currentCorrectStreak := 0
	currentIncorrectStreak := 0

	for _, p := range predictions {
		if p.IsCorrect {
			currentCorrectStreak++
			currentIncorrectStreak = 0
			if currentCorrectStreak > bestStreak {
				bestStreak = currentCorrectStreak
			}
		} else {
			currentIncorrectStreak++
			currentCorrectStreak = 0
			if currentIncorrectStreak > worstStreak {
				worstStreak = currentIncorrectStreak
			}
		}
	}

	// Current streak is negative if we're on a losing streak
	if !lastResult {
		currentStreak = -currentStreak
	}

	return currentStreak, bestStreak, worstStreak
}

func (ats *AccuracyTrackingService) calculateAccuracyByGameType(predictions []models.PredictionAccuracy, gameType string) float64 {
	filtered := make([]models.PredictionAccuracy, 0)
	for _, p := range predictions {
		if p.GameType == gameType {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		return 0.0
	}

	correct := 0
	for _, p := range filtered {
		if p.IsCorrect {
			correct++
		}
	}

	return float64(correct) / float64(len(filtered))
}

func (ats *AccuracyTrackingService) calculateAccuracyByConfidence(predictions []models.PredictionAccuracy, minConf, maxConf float64) float64 {
	filtered := make([]models.PredictionAccuracy, 0)
	for _, p := range predictions {
		if p.Confidence >= minConf && p.Confidence < maxConf {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		return 0.0
	}

	correct := 0
	for _, p := range filtered {
		if p.IsCorrect {
			correct++
		}
	}

	return float64(correct) / float64(len(filtered))
}

func (ats *AccuracyTrackingService) calculateAccuracyTrend(predictions []models.PredictionAccuracy) (string, float64) {
	if len(predictions) < 10 {
		return "insufficient_data", 0.0
	}

	// Compare first half vs second half accuracy
	midpoint := len(predictions) / 2
	firstHalf := predictions[:midpoint]
	secondHalf := predictions[midpoint:]

	firstAccuracy := ats.calculateAccuracyForSlice(firstHalf)
	secondAccuracy := ats.calculateAccuracyForSlice(secondHalf)

	diff := secondAccuracy - firstAccuracy

	if math.Abs(diff) < 0.05 {
		return "stable", math.Abs(diff)
	} else if diff > 0 {
		return "improving", diff
	} else {
		return "declining", math.Abs(diff)
	}
}

func (ats *AccuracyTrackingService) calculateAccuracyForSlice(predictions []models.PredictionAccuracy) float64 {
	if len(predictions) == 0 {
		return 0.0
	}

	correct := 0
	for _, p := range predictions {
		if p.IsCorrect {
			correct++
		}
	}

	return float64(correct) / float64(len(predictions))
}

func (ats *AccuracyTrackingService) calculateEnsembleStats() {
	// Implementation for ensemble statistics would go here
	// For now, create a basic ensemble stats structure
	ats.ensembleStats = &models.EnsembleAccuracyStats{
		IndividualModelStats: ats.modelStats,
		LastUpdated:          time.Now(),
	}
}
