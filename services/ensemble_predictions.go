package services

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// EnsemblePredictionService combines multiple prediction models with cross-validation
type EnsemblePredictionService struct {
	models          []PredictionModel
	teamCode        string
	accuracyTracker *AccuracyTrackingService
	dataQuality     *DataQualityService
	dynamicWeights  *DynamicWeightingService
	crossValidation *CrossValidationService
}

// NewEnsemblePredictionService creates a new ensemble service with cross-validation
func NewEnsemblePredictionService(teamCode string) *EnsemblePredictionService {
	return &EnsemblePredictionService{
		teamCode:        teamCode,
		accuracyTracker: NewAccuracyTrackingService(),
		dataQuality:     NewDataQualityService(teamCode),
		dynamicWeights:  NewDynamicWeightingService(),
		crossValidation: NewCrossValidationService(),
		models: []PredictionModel{
			NewStatisticalModel(),       // 35% (reduced from 40%)
			NewBayesianModel(),          // 15% (reduced from 20%)
			NewMonteCarloModel(),        // 10% (reduced from 15%)
			NewEloRatingModel(),         // 20%
			NewPoissonRegressionModel(), // 15% (reduced from 20%)
			NewNeuralNetworkModel(),     // 5% (new, starting conservative)
		},
	}
}

// PredictGame runs all models and combines their predictions with dynamic weighting
func (eps *EnsemblePredictionService) PredictGame(homeFactors, awayFactors *models.PredictionFactors) (*models.PredictionResult, error) {
	start := time.Now()
	fmt.Printf("🤖 Running ensemble prediction with %d models...\n", len(eps.models))

	// ============================================================================
	// PHASE 4: ENHANCED DATA ENRICHMENT
	// ============================================================================

	// 1. Goalie Intelligence
	goalieService := GetGoalieService()
	if goalieService != nil {
		goalieImpact := goalieService.GetGoalieImpact(homeFactors.TeamCode, awayFactors.TeamCode, time.Now())
		homeFactors.GoalieAdvantage = goalieImpact
		awayFactors.GoalieAdvantage = -goalieImpact

		if goalieImpact != 0 {
			if goalieImpact > 0 {
				fmt.Printf("🥅 Goalie Impact: Home advantage (%.1f%% swing)\n", goalieImpact*100)
			} else {
				fmt.Printf("🥅 Goalie Impact: Away advantage (%.1f%% swing)\n", math.Abs(goalieImpact)*100)
			}
		}
	}

	// 2. Betting Market Intelligence
	marketService := GetBettingMarketService()
	if marketService != nil && marketService.isEnabled {
		marketAdj, err := marketService.GetMarketAdjustment(homeFactors.TeamCode, awayFactors.TeamCode, time.Now())
		if err == nil && marketAdj != nil {
			homeFactors.MarketConsensus = marketAdj.MarketPrediction
			awayFactors.MarketConsensus = 1.0 - marketAdj.MarketPrediction
			homeFactors.MarketConfidenceVal = marketAdj.MarketEfficiency

			if marketAdj.MarketPrediction > 0 {
				fmt.Printf("💰 Market Consensus: %.1f%% home win (confidence: %.1f%%)\n",
					marketAdj.MarketPrediction*100,
					marketAdj.MarketEfficiency*100)
			}
		}
	}

	// 3. Schedule Context Analysis
	scheduleService := GetScheduleContextService()
	if scheduleService != nil {
		scheduleComp, err := scheduleService.GetScheduleComparison(homeFactors.TeamCode, awayFactors.TeamCode, time.Now())
		if err == nil && scheduleComp != nil {
			homeCtx := scheduleComp.HomeContext
			awayCtx := scheduleComp.AwayContext

			// Apply schedule factors
			homeFactors.TravelDistance = homeCtx.TravelDistance
			awayFactors.TravelDistance = awayCtx.TravelDistance

			homeFactors.BackToBackIndicator = 0.0
			if homeCtx.IsBackToBack {
				homeFactors.BackToBackIndicator = 1.0
			}
			awayFactors.BackToBackIndicator = 0.0
			if awayCtx.IsBackToBack {
				awayFactors.BackToBackIndicator = 1.0
			}

			homeFactors.ScheduleDensity = float64(homeCtx.GamesInLast7Days)
			awayFactors.ScheduleDensity = float64(awayCtx.GamesInLast7Days)

			homeFactors.TrapGameFactor = homeCtx.TrapGameScore
			awayFactors.TrapGameFactor = awayCtx.TrapGameScore

			homeFactors.PlayoffImportance = homeCtx.PlayoffImportance
			awayFactors.PlayoffImportance = awayCtx.PlayoffImportance

			homeFactors.RestAdvantage = float64(homeCtx.RestAdvantage)
			awayFactors.RestAdvantage = float64(-homeCtx.RestAdvantage)

			if scheduleComp.TotalImpact != 0 {
				fmt.Printf("📅 Schedule Impact: %s advantage (%.1f%% swing)\n",
					scheduleComp.OverallAdvantage,
					math.Abs(scheduleComp.TotalImpact)*100)
			}
		}
	}

	// ============================================================================
	// PHASE 6: FEATURE ENGINEERING ENRICHMENT
	// ============================================================================

	// 1. Matchup Database
	matchupService := GetMatchupService()
	if matchupService != nil {
		advantage := matchupService.CalculateMatchupAdvantage(homeFactors.TeamCode, awayFactors.TeamCode)

		homeFactors.HeadToHeadAdvantage = advantage.TotalAdvantage
		awayFactors.HeadToHeadAdvantage = -advantage.TotalAdvantage

		homeFactors.RecentMatchupTrend = advantage.RecentAdvantage
		awayFactors.RecentMatchupTrend = -advantage.RecentAdvantage

		homeFactors.VenueSpecificRecord = advantage.VenueAdvantage
		awayFactors.VenueSpecificRecord = -advantage.VenueAdvantage

		homeFactors.IsRivalryGame = advantage.RivalryBoost > 0
		awayFactors.IsRivalryGame = advantage.RivalryBoost > 0

		if advantage.RivalryBoost > 0 {
			homeFactors.RivalryIntensity = advantage.RivalryBoost / 0.05 // Normalize to 0-1
			awayFactors.RivalryIntensity = advantage.RivalryBoost / 0.05
		}

		homeFactors.IsDivisionGame = advantage.DivisionGameBoost > 0
		awayFactors.IsDivisionGame = advantage.DivisionGameBoost > 0

		homeFactors.IsPlayoffRematch = advantage.PlayoffRematchBoost > 0
		awayFactors.IsPlayoffRematch = advantage.PlayoffRematchBoost > 0

		// Get matchup history for additional details
		history := matchupService.GetMatchupHistory(homeFactors.TeamCode, awayFactors.TeamCode)
		if history != nil {
			homeFactors.GamesInSeries = history.TotalGames
			awayFactors.GamesInSeries = history.TotalGames

			if !history.LastGameDate.IsZero() {
				daysSince := int(time.Since(history.LastGameDate).Hours() / 24)
				homeFactors.DaysSinceLastMeeting = daysSince
				awayFactors.DaysSinceLastMeeting = daysSince
			}

			homeFactors.AverageGoalDiff = history.AvgGoalsTeamA - history.AvgGoalsTeamB
			awayFactors.AverageGoalDiff = history.AvgGoalsTeamB - history.AvgGoalsTeamA
		}

		if advantage.TotalAdvantage != 0 {
			fmt.Printf("📊 Matchup Advantage: %s (+%.1f%% from H2H)\n",
				homeFactors.TeamCode, advantage.TotalAdvantage*100)
		}
	}

	// 2. Advanced Rolling Statistics
	rollingService := GetRollingStatsService()
	if rollingService != nil {
		homeStats, _ := rollingService.GetTeamStats(homeFactors.TeamCode)
		awayStats, _ := rollingService.GetTeamStats(awayFactors.TeamCode)

		if homeStats != nil {
			homeFactors.FormRating = homeStats.FormRating
			homeFactors.MomentumScore = homeStats.MomentumScore
			homeFactors.IsHot = homeStats.IsHot
			homeFactors.IsCold = homeStats.IsCold
			homeFactors.IsStreaking = homeStats.IsStreaking
			homeFactors.WeightedWinPct = homeStats.WeightedWinPct
			homeFactors.WeightedGoalsFor = homeStats.WeightedGoalsFor
			homeFactors.WeightedGoalsAgainst = homeStats.WeightedGoalsAgainst
			homeFactors.QualityOfWins = homeStats.QualityOfWins
			homeFactors.QualityOfLosses = homeStats.QualityOfLosses
			homeFactors.VsPlayoffTeamsPct = homeStats.VsPlayoffTeamsPct
			homeFactors.VsTop10TeamsPct = homeStats.VsTop10TeamsPct
			homeFactors.ClutchPerformance = homeStats.ClutchPerformance
			homeFactors.Last3GamesPoints = homeStats.Last3GamesPoints
			homeFactors.Last5GamesPoints = homeStats.Last5GamesPoints
			homeFactors.GoalDifferential3 = homeStats.GoalDifferential3
			homeFactors.GoalDifferential5 = homeStats.GoalDifferential5
			homeFactors.ScoringTrend = homeStats.ScoringTrend
			homeFactors.DefensiveTrend = homeStats.DefensiveTrend
			homeFactors.StrengthOfSchedule = homeStats.StrengthOfSchedule
			homeFactors.AdjustedWinPct = homeStats.AdjustedWinPct
			homeFactors.PointsTrendDirection = homeStats.PointsTrendDirection

			if homeStats.IsHot {
				fmt.Printf("🔥 %s is HOT! (Form: %.1f/10, Momentum: %.2f)\n",
					homeFactors.TeamCode, homeStats.FormRating, homeStats.MomentumScore)
			} else if homeStats.IsCold {
				fmt.Printf("🧊 %s is COLD (Form: %.1f/10, Momentum: %.2f)\n",
					homeFactors.TeamCode, homeStats.FormRating, homeStats.MomentumScore)
			}
		}

		if awayStats != nil {
			awayFactors.FormRating = awayStats.FormRating
			awayFactors.MomentumScore = awayStats.MomentumScore
			awayFactors.IsHot = awayStats.IsHot
			awayFactors.IsCold = awayStats.IsCold
			awayFactors.IsStreaking = awayStats.IsStreaking
			awayFactors.WeightedWinPct = awayStats.WeightedWinPct
			awayFactors.WeightedGoalsFor = awayStats.WeightedGoalsFor
			awayFactors.WeightedGoalsAgainst = awayStats.WeightedGoalsAgainst
			awayFactors.QualityOfWins = awayStats.QualityOfWins
			awayFactors.QualityOfLosses = awayStats.QualityOfLosses
			awayFactors.VsPlayoffTeamsPct = awayStats.VsPlayoffTeamsPct
			awayFactors.VsTop10TeamsPct = awayStats.VsTop10TeamsPct
			awayFactors.ClutchPerformance = awayStats.ClutchPerformance
			awayFactors.Last3GamesPoints = awayStats.Last3GamesPoints
			awayFactors.Last5GamesPoints = awayStats.Last5GamesPoints
			awayFactors.GoalDifferential3 = awayStats.GoalDifferential3
			awayFactors.GoalDifferential5 = awayStats.GoalDifferential5
			awayFactors.ScoringTrend = awayStats.ScoringTrend
			awayFactors.DefensiveTrend = awayStats.DefensiveTrend
			awayFactors.StrengthOfSchedule = awayStats.StrengthOfSchedule
			awayFactors.AdjustedWinPct = awayStats.AdjustedWinPct
			awayFactors.PointsTrendDirection = awayStats.PointsTrendDirection

			if awayStats.IsHot {
				fmt.Printf("🔥 %s is HOT! (Form: %.1f/10, Momentum: %.2f)\n",
					awayFactors.TeamCode, awayStats.FormRating, awayStats.MomentumScore)
			} else if awayStats.IsCold {
				fmt.Printf("🧊 %s is COLD (Form: %.1f/10, Momentum: %.2f)\n",
					awayFactors.TeamCode, awayStats.FormRating, awayStats.MomentumScore)
			}
		}
	}

	// 3. Player Impact
	playerService := GetPlayerImpactService()
	if playerService != nil {
		comparison := playerService.ComparePlayerImpact(homeFactors.TeamCode, awayFactors.TeamCode)

		homeFactors.StarPowerEdge = comparison.StarPowerAdvantage / 0.10 // Normalize to -1 to +1
		awayFactors.StarPowerEdge = -comparison.StarPowerAdvantage / 0.10

		homeFactors.DepthEdge = comparison.DepthAdvantage / 0.05 // Normalize to -1 to +1
		awayFactors.DepthEdge = -comparison.DepthAdvantage / 0.05

		// Get individual team impacts
		homeImpact := playerService.GetPlayerImpact(homeFactors.TeamCode)
		awayImpact := playerService.GetPlayerImpact(awayFactors.TeamCode)

		if homeImpact != nil {
			homeFactors.StarPowerRating = homeImpact.StarPower
			homeFactors.Top3CombinedPPG = homeImpact.Top3PPG
			homeFactors.DepthScoring = homeImpact.DepthScore
			homeFactors.SecondaryPPG = homeImpact.Secondary4to10
			homeFactors.ScoringBalance = homeImpact.BalanceRating
			homeFactors.TopScorerForm = homeImpact.TopScorerForm
			homeFactors.DepthForm = homeImpact.DepthForm

			if len(homeImpact.TopScorers) > 0 {
				homeFactors.TopScorerPPG = homeImpact.TopScorers[0].PointsPerGame
			}
		}

		if awayImpact != nil {
			awayFactors.StarPowerRating = awayImpact.StarPower
			awayFactors.Top3CombinedPPG = awayImpact.Top3PPG
			awayFactors.DepthScoring = awayImpact.DepthScore
			awayFactors.SecondaryPPG = awayImpact.Secondary4to10
			awayFactors.ScoringBalance = awayImpact.BalanceRating
			awayFactors.TopScorerForm = awayImpact.TopScorerForm
			awayFactors.DepthForm = awayImpact.DepthForm

			if len(awayImpact.TopScorers) > 0 {
				awayFactors.TopScorerPPG = awayImpact.TopScorers[0].PointsPerGame
			}
		}

		if comparison.TotalPlayerImpact != 0 {
			fmt.Printf("⭐ Player Advantage: %s (+%.1f%% from talent)\n",
				homeFactors.TeamCode, comparison.TotalPlayerImpact*100)
		}
	}

	var modelResults []models.ModelResult
	var totalWeight float64

	// 🚀 NEW: Get current dynamic weights
	currentWeights := eps.dynamicWeights.GetCurrentWeights()
	fmt.Printf("⚖️ Current model weights: Statistical=%.1f%%, Bayesian=%.1f%%, Monte Carlo=%.1f%%, Elo=%.1f%%, Poisson=%.1f%%, Neural Net=%.1f%%\n",
		currentWeights["Enhanced Statistical"]*100,
		currentWeights["Bayesian Inference"]*100,
		currentWeights["Monte Carlo Simulation"]*100,
		currentWeights["Elo Rating"]*100,
		currentWeights["Poisson Regression"]*100,
		currentWeights["Neural Network"]*100)

	// Run all models with dynamic weights
	for _, model := range eps.models {
		result, err := model.Predict(homeFactors, awayFactors)
		if err != nil {
			fmt.Printf("⚠️ Model %s failed: %v\n", model.GetName(), err)
			continue
		}

		// 🔄 Update model weight with current dynamic weight
		if dynamicWeight, exists := currentWeights[model.GetName()]; exists {
			result.Weight = dynamicWeight
		} else {
			// Fallback to original weight if dynamic weight not found
			result.Weight = model.GetWeight()
		}

		modelResults = append(modelResults, *result)
		totalWeight += result.Weight

		fmt.Printf("📊 %s: %.1f%% confidence, %s prediction (Weight: %.1f%%)\n",
			model.GetName(), result.Confidence*100, result.PredictedScore, result.Weight*100)
	}

	if len(modelResults) == 0 {
		return nil, fmt.Errorf("all prediction models failed")
	}

	// Combine predictions using weighted ensemble with dynamic weights
	combinedResult := eps.combineWeightedPredictions(modelResults, totalWeight, homeFactors, awayFactors)
	combinedResult.ModelResults = modelResults
	combinedResult.EnsembleMethod = "Weighted Average with Dynamic Weighting"

	fmt.Printf("🎯 Ensemble Result: %s wins with %.1f%% probability (Score: %s, Confidence: %.1f%%)\n",
		combinedResult.Winner, combinedResult.WinProbability*100,
		combinedResult.PredictedScore, combinedResult.Confidence*100)

	fmt.Printf("⏱️ Total processing time: %dms\n", time.Since(start).Milliseconds())

	return combinedResult, nil
}

// combineWeightedPredictions combines model results using weighted averaging
func (eps *EnsemblePredictionService) combineWeightedPredictions(results []models.ModelResult, totalWeight float64, homeFactors, awayFactors *models.PredictionFactors) *models.PredictionResult {
	var weightedHomeProb, weightedAwayProb float64
	var weightedConfidence float64
	var homeGoalsSum, awayGoalsSum float64
	var validScores int

	// Calculate weighted averages
	for _, result := range results {
		normalizedWeight := result.Weight / totalWeight

		// Weight probabilities by model confidence as well
		confidenceBoost := 1.0 + (result.Confidence-0.5)*0.4 // Boost high-confidence models
		adjustedWeight := normalizedWeight * confidenceBoost

		if result.WinProbability > 0.5 {
			// Model predicts home team wins
			weightedHomeProb += result.WinProbability * adjustedWeight
		} else {
			// Model predicts away team wins
			weightedAwayProb += (1.0 - result.WinProbability) * adjustedWeight
		}

		weightedConfidence += result.Confidence * normalizedWeight

		// Parse and combine scores
		homeGoals, awayGoals := eps.parseScore(result.PredictedScore)
		if homeGoals >= 0 && awayGoals >= 0 {
			homeGoalsSum += float64(homeGoals) * normalizedWeight
			awayGoalsSum += float64(awayGoals) * normalizedWeight
			validScores++
		}
	}

	// Determine final winner and probability
	var winner string
	var finalProb float64

	if weightedHomeProb > weightedAwayProb {
		winner = homeFactors.TeamCode
		finalProb = weightedHomeProb / (weightedHomeProb + weightedAwayProb)
	} else {
		winner = awayFactors.TeamCode
		finalProb = weightedAwayProb / (weightedHomeProb + weightedAwayProb)
	}

	// Create final score prediction
	predictedScore := "3-2" // Default fallback
	if validScores > 0 {
		avgHomeGoals := int(math.Round(homeGoalsSum))
		avgAwayGoals := int(math.Round(awayGoalsSum))
		predictedScore = fmt.Sprintf("%d-%d", avgHomeGoals, avgAwayGoals)
	}

	// Determine game type based on probability margin
	gameType := eps.determineGameType(finalProb, homeGoalsSum, awayGoalsSum)

	// Check if it's an upset (lower-ranked team predicted to win)
	isUpset := eps.isUpsetPrediction(homeFactors, awayFactors, winner)

	// Ensemble confidence considers model agreement, historical accuracy, and cross-validation
	ensembleConfidence := eps.calculateEnsembleConfidence(results, weightedConfidence, homeFactors, awayFactors)

	// 🚀 NEW: Apply cross-validation calibration to final confidence
	if eps.crossValidation.IsCalibrated() {
		calibratedConfidence := eps.crossValidation.GetCalibratedConfidence(ensembleConfidence, results)
		fmt.Printf("🎯 Cross-validation calibration: %.1f%% → %.1f%%\n",
			ensembleConfidence*100, calibratedConfidence*100)
		ensembleConfidence = calibratedConfidence
	}

	return &models.PredictionResult{
		Winner:         winner,
		WinProbability: finalProb,
		PredictedScore: predictedScore,
		IsUpset:        isUpset,
		GameType:       gameType,
		Confidence:     ensembleConfidence,
		ModelResults:   results,
		EnsembleMethod: "Weighted Average with Cross-Validation",
	}
}

// RecordPredictionOutcome records the actual game outcome for dynamic weight adjustment
func (eps *EnsemblePredictionService) RecordPredictionOutcome(homeTeam, awayTeam string, gameDate time.Time, actualWinner string, actualScore string) error {
	if eps.dynamicWeights == nil || !eps.dynamicWeights.IsEnabled() {
		return nil // Dynamic weighting not enabled
	}

	// For each model, create an accuracy record
	for _, model := range eps.models {
		// This would need to be enhanced to store the original prediction
		// For now, we'll create a basic record structure
		record := AccuracyRecord{
			PredictionID: fmt.Sprintf("%s_vs_%s_%s_%s", homeTeam, awayTeam, gameDate.Format("2006-01-02"), model.GetName()),
			GameDate:     gameDate,
			HomeTeam:     homeTeam,
			AwayTeam:     awayTeam,
			ActualWinner: actualWinner,
			RecordedAt:   time.Now(),
			GameContext: GameContext{
				IsHomeGame:        true,      // Would need to determine this
				IsPlayoffGame:     false,     // Would need to determine this
				TeamStrengthGap:   0.1,       // Would calculate from team stats
				IsUpsetPrediction: false,     // Would determine from prediction
				OpponentType:      "average", // Would classify opponent
				GameImportance:    "medium",  // Would assess importance
			},
		}

		// Record the outcome for this model
		err := eps.dynamicWeights.RecordPredictionOutcome(model.GetName(), record)
		if err != nil {
			fmt.Printf("⚠️ Error recording outcome for %s: %v\n", model.GetName(), err)
		}
	}

	return nil
}

// RecordHistoricalPrediction records a prediction for cross-validation analysis
func (eps *EnsemblePredictionService) RecordHistoricalPrediction(homeTeam, awayTeam string, gameDate time.Time,
	prediction *models.PredictionResult, actualWinner string, actualScore string) error {

	if eps.crossValidation == nil {
		return nil // Cross-validation not enabled
	}

	// Create historical prediction record
	historicalPred := HistoricalPrediction{
		PredictionID:    fmt.Sprintf("%s_vs_%s_%s", homeTeam, awayTeam, gameDate.Format("2006-01-02")),
		GameDate:        gameDate,
		HomeTeam:        homeTeam,
		AwayTeam:        awayTeam,
		PredictedWinner: prediction.Winner,
		PredictedScore:  prediction.PredictedScore,
		WinProbability:  prediction.WinProbability,
		RawConfidence:   prediction.Confidence,
		ModelResults:    prediction.ModelResults,
		ActualWinner:    actualWinner,
		ActualScore:     actualScore,
		GameCompleted:   actualWinner != "",
		GameType:        "regular", // Could be enhanced to detect game type
		RecordedAt:      time.Now(),
	}

	// Add to cross-validation service
	eps.crossValidation.AddHistoricalPrediction(historicalPred)

	// Check if we should run cross-validation
	if eps.crossValidation.ShouldRevalidate() {
		go func() {
			log.Printf("🔄 Running cross-validation update...")
			if err := eps.crossValidation.RunCrossValidation(); err != nil {
				log.Printf("⚠️ Cross-validation failed: %v", err)
			} else {
				log.Printf("✅ Cross-validation completed successfully")
			}
		}()
	}

	return nil
}

// GetValidationSummary returns cross-validation results
func (eps *EnsemblePredictionService) GetValidationSummary() ValidationSummary {
	if eps.crossValidation == nil {
		return ValidationSummary{
			IsValidated: false,
			Message:     "Cross-validation not enabled",
		}
	}

	return eps.crossValidation.GetValidationSummary()
}

// parseScore extracts home and away goals from score string
func (eps *EnsemblePredictionService) parseScore(scoreStr string) (int, int) {
	parts := strings.Split(scoreStr, "-")
	if len(parts) != 2 {
		return -1, -1 // Invalid score format
	}

	homeGoals, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	awayGoals, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)

	if err1 != nil || err2 != nil {
		return -1, -1
	}

	return int(homeGoals), int(awayGoals)
}

// determineGameType categorizes the predicted game outcome
func (eps *EnsemblePredictionService) determineGameType(probability, homeGoals, awayGoals float64) string {
	scoreDiff := math.Abs(homeGoals - awayGoals)
	probMargin := math.Abs(probability - 0.5)

	if probMargin < 0.1 || scoreDiff <= 1 {
		return "toss-up"
	} else if probMargin > 0.25 || scoreDiff > 2.5 {
		return "blowout"
	}
	return "close"
}

// isUpsetPrediction determines if prediction is an upset
func (eps *EnsemblePredictionService) isUpsetPrediction(homeFactors, awayFactors *models.PredictionFactors, winner string) bool {
	// Simple upset detection based on win percentage
	if winner == homeFactors.TeamCode {
		return homeFactors.WinPercentage < awayFactors.WinPercentage-0.1
	}
	return awayFactors.WinPercentage < homeFactors.WinPercentage-0.1
}

// calculateEnsembleConfidence determines overall confidence with accuracy tracking boost
func (eps *EnsemblePredictionService) calculateEnsembleConfidence(results []models.ModelResult, avgConfidence float64, homeFactors, awayFactors *models.PredictionFactors) float64 {
	if len(results) <= 1 {
		return avgConfidence
	}

	// Calculate agreement between models
	var probabilities []float64
	for _, result := range results {
		probabilities = append(probabilities, result.WinProbability)
	}

	// Sort to find median and range
	sort.Float64s(probabilities)

	// Agreement bonus based on how close models are
	span := probabilities[len(probabilities)-1] - probabilities[0]
	agreementBonus := math.Max(0, (0.3-span)/0.3*0.2) // Up to 20% bonus for agreement

	// Processing time penalty (longer = less confident in timeliness)
	var avgProcessingTime float64
	for _, result := range results {
		avgProcessingTime += float64(result.ProcessingTime)
	}
	avgProcessingTime /= float64(len(results))

	timeConfidence := math.Max(0.8, 1.0-avgProcessingTime/5000.0) // Penalty after 5 seconds

	// 🚀 NEW: Historical Accuracy Boost
	var accuracyBoost float64 = 1.0
	if eps.accuracyTracker != nil {
		// Get confidence boost for the winning team's factors
		var winningFactors *models.PredictionFactors
		if homeFactors.WinPercentage > awayFactors.WinPercentage {
			winningFactors = homeFactors
		} else {
			winningFactors = awayFactors
		}

		// Calculate weighted accuracy boost from all models
		totalBoost := 0.0
		totalWeight := 0.0

		for _, result := range results {
			boostFactors := eps.accuracyTracker.GetConfidenceBoost(result.ModelName, winningFactors)
			totalBoost += boostFactors.OverallConfidenceBoost * result.Weight
			totalWeight += result.Weight
		}

		if totalWeight > 0 {
			accuracyBoost = totalBoost / totalWeight
		}

		fmt.Printf("🚀 Historical accuracy boost: %.2fx\n", accuracyBoost)
	}

	// 🚀 NEW: Data Quality Boost
	var dataQualityBoost float64 = 1.0
	if eps.dataQuality != nil {
		// Assess data quality for both teams
		homeQuality := eps.dataQuality.AssessDataQuality(homeFactors)
		awayQuality := eps.dataQuality.AssessDataQuality(awayFactors)

		// Use average data quality impact
		avgQualityImpact := (homeQuality.ConfidenceImpact + awayQuality.ConfidenceImpact) / 2.0
		dataQualityBoost = avgQualityImpact

		fmt.Printf("📊 Data quality boost: %.2fx (Home: %.1f/100, Away: %.1f/100)\n",
			dataQualityBoost, homeQuality.OverallScore, awayQuality.OverallScore)
	}

	// Combine all confidence factors
	baseConfidence := (avgConfidence + agreementBonus) * timeConfidence
	finalConfidence := baseConfidence * accuracyBoost * dataQualityBoost

	return math.Min(1.0, math.Max(0.1, finalConfidence))
}

// GetModelNames returns names of all models in the ensemble
func (eps *EnsemblePredictionService) GetModelNames() []string {
	var names []string
	for _, model := range eps.models {
		names = append(names, model.GetName())
	}
	return names
}

// GetModelWeights returns weights of all models in the ensemble
func (eps *EnsemblePredictionService) GetModelWeights() map[string]float64 {
	weights := make(map[string]float64)
	for _, model := range eps.models {
		weights[model.GetName()] = model.GetWeight()
	}
	return weights
}

// GetAccuracyTracker returns the accuracy tracking service
func (eps *EnsemblePredictionService) GetAccuracyTracker() *AccuracyTrackingService {
	return eps.accuracyTracker
}
