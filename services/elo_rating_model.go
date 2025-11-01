package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// EloRatingModel implements dynamic team strength tracking using Elo rating system
type EloRatingModel struct {
	teamRatings       map[string]float64        // Current Elo ratings for each team
	ratingHistory     map[string][]RatingRecord // Historical rating changes
	initialRating     float64                   // Starting rating for new teams
	kFactor           float64                   // Learning rate
	homeAdvantage     float64                   // Home ice advantage adjustment
	weight            float64                   // Model weight in ensemble
	lastUpdated       time.Time
	mutex             sync.RWMutex       // Thread safety for concurrent updates
	updateStats       ModelUpdateStats   // Statistics about updates
	seasonDecayRate   float64            // How much ratings decay over time
	confidenceFactors map[string]float64 // Confidence in each team's rating
	dataDir           string             // Directory for persistent storage
}

// EloModelData represents the serializable state of the Elo model
type EloModelData struct {
	TeamRatings       map[string]float64        `json:"teamRatings"`
	RatingHistory     map[string][]RatingRecord `json:"ratingHistory"`
	ConfidenceFactors map[string]float64        `json:"confidenceFactors"`
	LastUpdated       time.Time                 `json:"lastUpdated"`
	Version           string                    `json:"version"`
}

// RatingRecord tracks historical rating changes
type RatingRecord struct {
	Date       time.Time `json:"date"`
	OldRating  float64   `json:"oldRating"`
	NewRating  float64   `json:"newRating"`
	Change     float64   `json:"change"`
	Opponent   string    `json:"opponent"`
	GameResult string    `json:"gameResult"` // "W", "L", "OTL", "SOL"
	GameScore  string    `json:"gameScore"`
	KFactor    float64   `json:"kFactor"`
	WasHome    bool      `json:"wasHome"`
	Confidence float64   `json:"confidence"`
}

// NewEloRatingModel creates a new Elo rating prediction model
func NewEloRatingModel() *EloRatingModel {
	model := &EloRatingModel{
		teamRatings:       make(map[string]float64),
		ratingHistory:     make(map[string][]RatingRecord),
		confidenceFactors: make(map[string]float64),
		initialRating:     1500.0, // Standard Elo starting rating
		kFactor:           32.0,   // Standard K-factor (higher = more volatile)
		homeAdvantage:     100.0,  // Home team gets +100 Elo equivalent
		weight:            0.20,   // 20% weight in ensemble
		seasonDecayRate:   0.95,   // 5% decay per season
		lastUpdated:       time.Now(),
		dataDir:           "data/models",
		updateStats: ModelUpdateStats{
			UpdateFrequency: 1 * time.Hour,
		},
	}

	// Create data directory if it doesn't exist
	os.MkdirAll(model.dataDir, 0755)

	// Load existing ratings if available
	if err := model.loadRatings(); err != nil {
		log.Printf("⚠️ Could not load Elo ratings: %v (starting fresh)", err)
	}

	return model
}

// Predict implements the PredictionModel interface for Elo ratings
func (elo *EloRatingModel) Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error) {
	startTime := time.Now()

	log.Printf("🏆 Running Elo Rating prediction for %s vs %s...", homeFactors.TeamCode, awayFactors.TeamCode)

	// Get current Elo ratings for both teams
	homeRating := elo.getTeamRating(homeFactors.TeamCode)
	awayRating := elo.getTeamRating(awayFactors.TeamCode)

	// Apply home ice advantage
	adjustedHomeRating := homeRating + elo.homeAdvantage

	// Calculate expected win probability using Elo formula
	expectedHomeWin := elo.calculateWinProbability(adjustedHomeRating, awayRating)

	// Adjust for situational factors
	situationalAdjustment := elo.calculateSituationalAdjustment(homeFactors, awayFactors)
	adjustedWinProb := elo.applySituationalAdjustment(expectedHomeWin, situationalAdjustment)

	// Predict score based on Elo difference and historical scoring patterns
	homeScore, awayScore := elo.predictScore(adjustedHomeRating, awayRating, homeFactors, awayFactors)

	// Calculate confidence based on rating difference and situational factors
	confidence := elo.calculateConfidence(homeRating, awayRating, situationalAdjustment)

	processingTime := time.Since(startTime).Milliseconds()

	log.Printf("🏆 Elo Ratings - %s: %.0f, %s: %.0f (Home Adj: +%.0f)",
		homeFactors.TeamCode, homeRating, awayFactors.TeamCode, awayRating, elo.homeAdvantage)
	log.Printf("📊 Elo Rating: %.1f%% confidence, %d-%d prediction", confidence*100, homeScore, awayScore)

	return &models.ModelResult{
		ModelName:      "Elo Rating",
		WinProbability: adjustedWinProb,
		Confidence:     confidence,
		PredictedScore: fmt.Sprintf("%d-%d", homeScore, awayScore),
		Weight:         0.20, // Will be dynamically adjusted by ensemble
		ProcessingTime: processingTime,
	}, nil
}

// GetName implements the PredictionModel interface
func (elo *EloRatingModel) GetName() string {
	return "Elo Rating"
}

// GetWeight implements the PredictionModel interface
func (elo *EloRatingModel) GetWeight() float64 {
	return elo.weight
}

// getTeamRating returns the current Elo rating for a team
func (elo *EloRatingModel) getTeamRating(teamCode string) float64 {
	if rating, exists := elo.teamRatings[teamCode]; exists {
		return rating
	}

	// Initialize new team with default rating, adjusted for NHL competitiveness
	initialRating := elo.calculateInitialRating(teamCode)
	elo.teamRatings[teamCode] = initialRating

	log.Printf("🆕 Initialized Elo rating for %s: %.0f", teamCode, initialRating)
	return initialRating
}

// calculateInitialRating determines starting Elo rating based on team historical performance
func (elo *EloRatingModel) calculateInitialRating(teamCode string) float64 {
	// Base rating for all NHL teams
	baseRating := elo.initialRating

	// Adjust based on team's recent historical performance (simplified)
	// In a real implementation, this would analyze last season's performance
	teamAdjustments := map[string]float64{
		"UTA": -50,  // New franchise, slightly below average
		"SJS": -100, // Rebuilding team
		"COL": +150, // Strong team
		"VGK": +100, // Competitive team
		"EDM": +120, // McDavid effect
		"TOR": +80,  // Generally competitive
		"BOS": +90,  // Consistently strong
		"FLA": +110, // Recent Cup winners
		"NYR": +70,  // Generally competitive
		"CAR": +60,  // Solid team
	}

	if adjustment, exists := teamAdjustments[teamCode]; exists {
		return baseRating + adjustment
	}

	return baseRating
}

// calculateWinProbability calculates expected win probability using Elo formula
func (elo *EloRatingModel) calculateWinProbability(rating1, rating2 float64) float64 {
	// Standard Elo probability formula: 1 / (1 + 10^((rating2 - rating1) / 400))
	ratingDiff := rating2 - rating1
	return 1.0 / (1.0 + math.Pow(10, ratingDiff/400.0))
}

// calculateSituationalAdjustment applies situational factors to Elo predictions
func (elo *EloRatingModel) calculateSituationalAdjustment(homeFactors, awayFactors *models.PredictionFactors) float64 {
	adjustment := 0.0

	// Travel fatigue impact
	if awayFactors.TravelFatigue.FatigueScore > 0.3 {
		adjustment += 0.05 // Favor home team
	}

	// Injury impact
	homeHealthPenalty := homeFactors.InjuryImpact.ImpactScore * 0.001
	awayHealthPenalty := awayFactors.InjuryImpact.ImpactScore * 0.001
	adjustment += awayHealthPenalty - homeHealthPenalty

	// Momentum factors
	homeMomentum := homeFactors.MomentumFactors.MomentumScore
	awayMomentum := awayFactors.MomentumFactors.MomentumScore
	momentumDiff := (homeMomentum - awayMomentum) * 0.02
	adjustment += momentumDiff

	// Recent form impact
	formDiff := (homeFactors.RecentForm - awayFactors.RecentForm) * 0.01
	adjustment += formDiff

	// Advanced analytics impact
	if homeFactors.AdvancedStats.OverallRating > 0 && awayFactors.AdvancedStats.OverallRating > 0 {
		ratingDiff := (homeFactors.AdvancedStats.OverallRating - awayFactors.AdvancedStats.OverallRating) * 0.001
		adjustment += ratingDiff
	}

	// Cap adjustment to prevent extreme swings
	if adjustment > 0.15 {
		adjustment = 0.15
	} else if adjustment < -0.15 {
		adjustment = -0.15
	}

	return adjustment
}

// applySituationalAdjustment applies the calculated adjustment to win probability
func (elo *EloRatingModel) applySituationalAdjustment(baseProbability, adjustment float64) float64 {
	adjusted := baseProbability + adjustment

	// Ensure probability stays within valid range
	if adjusted > 0.95 {
		adjusted = 0.95
	} else if adjusted < 0.05 {
		adjusted = 0.05
	}

	return adjusted
}

// predictScore predicts the final score based on Elo ratings and team factors
func (elo *EloRatingModel) predictScore(homeRating, awayRating float64, homeFactors, awayFactors *models.PredictionFactors) (int, int) {
	// Base expected goals based on Elo rating difference
	ratingDiff := homeRating - awayRating

	// Convert rating difference to expected goal differential
	// Typical NHL games average ~6 total goals, with rating differences affecting distribution
	baseHomeGoals := 3.0 + (ratingDiff / 200.0) // Rating diff of 200 = +1 goal expectation
	baseAwayGoals := 3.0 - (ratingDiff / 200.0)

	// Adjust for team-specific factors
	homeGoalsAdjustment := elo.calculateGoalsAdjustment(homeFactors, true)
	awayGoalsAdjustment := elo.calculateGoalsAdjustment(awayFactors, false)

	adjustedHomeGoals := baseHomeGoals + homeGoalsAdjustment
	adjustedAwayGoals := baseAwayGoals + awayGoalsAdjustment

	// Ensure minimum of 0 goals
	if adjustedHomeGoals < 0 {
		adjustedHomeGoals = 0
	}
	if adjustedAwayGoals < 0 {
		adjustedAwayGoals = 0
	}

	// Round to nearest integer for final score
	homeScore := int(math.Round(adjustedHomeGoals))
	awayScore := int(math.Round(adjustedAwayGoals))

	// Ensure at least one goal total (NHL games rarely end 0-0)
	if homeScore == 0 && awayScore == 0 {
		homeScore = 1
	}

	return homeScore, awayScore
}

// calculateGoalsAdjustment calculates goal expectation adjustments based on team factors
func (elo *EloRatingModel) calculateGoalsAdjustment(factors *models.PredictionFactors, isHome bool) float64 {
	adjustment := 0.0

	// Offensive/defensive strength from basic factors
	offensiveStrength := factors.GoalsFor / 82.0 // Goals per game approximation
	defensiveStrength := factors.GoalsAgainst / 82.0

	// Adjust based on league average (~3 goals per game per team)
	adjustment += (offensiveStrength - 3.0) * 0.3
	adjustment -= (defensiveStrength - 3.0) * 0.3

	// Recent form impact
	formImpact := (factors.RecentForm - 0.5) * 0.5 // Recent form as win rate
	adjustment += formImpact

	// Advanced analytics impact
	if factors.AdvancedStats.OverallRating > 0 {
		// Higher rating = better offensive/defensive performance
		ratingImpact := (factors.AdvancedStats.OverallRating - 50.0) / 50.0 * 0.3
		adjustment += ratingImpact
	}

	// Injury impact (reduces offensive capability)
	injuryPenalty := factors.InjuryImpact.ImpactScore * 0.01
	adjustment -= injuryPenalty

	// Home ice advantage for offensive production
	if isHome {
		adjustment += 0.1 // Small home scoring boost
	}

	return adjustment
}

// calculateConfidence determines prediction confidence based on various factors
func (elo *EloRatingModel) calculateConfidence(homeRating, awayRating, situationalAdjustment float64) float64 {
	// Base confidence from rating difference
	ratingDiff := math.Abs(homeRating - awayRating)
	baseConfidence := 0.5 + (ratingDiff / 800.0) // Larger rating gaps = higher confidence

	// Situational factors can increase or decrease confidence
	situationalConfidence := math.Abs(situationalAdjustment) * 2.0

	totalConfidence := baseConfidence + situationalConfidence

	// Cap confidence between 0.6 and 0.95
	if totalConfidence > 0.95 {
		totalConfidence = 0.95
	} else if totalConfidence < 0.6 {
		totalConfidence = 0.6
	}

	return totalConfidence
}

// UpdateRatings updates Elo ratings after a game result (for future implementation)
func (elo *EloRatingModel) UpdateRatings(homeTeam, awayTeam string, homeScore, awayScore int, wasOvertime bool) {
	homeRating := elo.getTeamRating(homeTeam)
	awayRating := elo.getTeamRating(awayTeam)

	// Determine actual result
	var homeResult float64
	if homeScore > awayScore {
		homeResult = 1.0 // Home win
	} else if homeScore < awayScore {
		homeResult = 0.0 // Away win
	} else {
		homeResult = 0.5 // Tie (shouldn't happen in modern NHL, but just in case)
	}

	// Adjust K-factor for overtime/shootout games (less impact)
	kFactor := elo.kFactor
	if wasOvertime {
		kFactor *= 0.8 // Reduce impact of OT/SO games
	}

	// Calculate expected results
	adjustedHomeRating := homeRating + elo.homeAdvantage
	expectedHomeResult := elo.calculateWinProbability(adjustedHomeRating, awayRating)

	// Update ratings
	homeChange := kFactor * (homeResult - expectedHomeResult)
	awayChange := kFactor * ((1.0 - homeResult) - (1.0 - expectedHomeResult))

	elo.teamRatings[homeTeam] = homeRating + homeChange
	elo.teamRatings[awayTeam] = awayRating + awayChange
	elo.lastUpdated = time.Now()

	log.Printf("🏆 Elo Update: %s %.0f→%.0f (%.0f), %s %.0f→%.0f (%.0f)",
		homeTeam, homeRating, elo.teamRatings[homeTeam], homeChange,
		awayTeam, awayRating, elo.teamRatings[awayTeam], awayChange)
}

// GetTeamRating returns the current Elo rating for a team (public method)
func (elo *EloRatingModel) GetTeamRating(teamCode string) float64 {
	return elo.getTeamRating(teamCode)
}

// GetAllRatings returns all current team ratings
func (elo *EloRatingModel) GetAllRatings() map[string]float64 {
	elo.mutex.RLock()
	defer elo.mutex.RUnlock()

	ratings := make(map[string]float64)
	for team, rating := range elo.teamRatings {
		ratings[team] = rating
	}
	return ratings
}

// ========== UpdatableModel Interface Implementation ==========

// Update implements the UpdatableModel interface for live data updates
func (elo *EloRatingModel) Update(data *LiveGameData) error {
	elo.mutex.Lock()

	log.Printf("🏆 Updating Elo ratings with new data...")

	updateCount := 0
	errorCount := 0

	// Process game results to update ratings
	for _, gameResult := range data.GameResults {
		if err := elo.processGameResult(gameResult); err != nil {
			log.Printf("❌ Failed to process game result %d: %v", gameResult.GameID, err)
			errorCount++
		} else {
			updateCount++
		}
	}

	// Update team ratings based on standings if no game results
	if len(data.GameResults) == 0 && data.Standings != nil {
		if err := elo.updateFromStandings(data.Standings); err != nil {
			log.Printf("⚠️ Failed to update from standings: %v", err)
			errorCount++
		} else {
			updateCount++
		}
	}

	// Apply seasonal decay if enough time has passed
	elo.applySeasonalDecay()

	// Update statistics
	elo.updateStats.TotalUpdates++
	if errorCount == 0 {
		elo.updateStats.SuccessfulUpdates++
	} else {
		elo.updateStats.FailedUpdates++
	}
	elo.updateStats.LastUpdateTime = time.Now()
	elo.lastUpdated = time.Now()

	log.Printf("✅ Elo rating update completed: %d processed, %d errors", updateCount, errorCount)

	// Save ratings to disk after update
	// Copy data we need while holding the lock, then unlock and save
	saveData := EloModelData{
		TeamRatings:       make(map[string]float64),
		RatingHistory:     make(map[string][]RatingRecord),
		ConfidenceFactors: make(map[string]float64),
		LastUpdated:       time.Now(),
		Version:           "1.0",
	}
	// Deep copy the maps
	for k, v := range elo.teamRatings {
		saveData.TeamRatings[k] = v
	}
	for k, v := range elo.ratingHistory {
		historyCopy := make([]RatingRecord, len(v))
		copy(historyCopy, v)
		saveData.RatingHistory[k] = historyCopy
	}
	for k, v := range elo.confidenceFactors {
		saveData.ConfidenceFactors[k] = v
	}

	// Release lock before I/O operation to avoid blocking other goroutines
	// The save function operates on a copy (saveData), so it doesn't need the lock
	elo.mutex.Unlock()
	if err := elo.saveRatingsWithData(saveData); err != nil {
		log.Printf("⚠️ Failed to save Elo ratings: %v", err)
	}

	return nil
}

// GetLastUpdate implements the UpdatableModel interface
func (elo *EloRatingModel) GetLastUpdate() time.Time {
	elo.mutex.RLock()
	defer elo.mutex.RUnlock()
	return elo.lastUpdated
}

// GetUpdateStats implements the UpdatableModel interface
func (elo *EloRatingModel) GetUpdateStats() ModelUpdateStats {
	elo.mutex.RLock()
	defer elo.mutex.RUnlock()

	// Calculate data freshness
	elo.updateStats.DataFreshness = time.Since(elo.lastUpdated)

	// Calculate average update time (simplified)
	if elo.updateStats.TotalUpdates > 0 {
		elo.updateStats.AverageUpdateTime = time.Since(elo.lastUpdated) / time.Duration(elo.updateStats.TotalUpdates)
	}

	return elo.updateStats
}

// GetModelName implements the UpdatableModel interface
func (elo *EloRatingModel) GetModelName() string {
	return "Elo Rating"
}

// RequiresUpdate implements the UpdatableModel interface
func (elo *EloRatingModel) RequiresUpdate(data *LiveGameData) bool {
	elo.mutex.RLock()
	defer elo.mutex.RUnlock()

	// Always update if we have new game results
	if len(data.GameResults) > 0 {
		return true
	}

	// Update if standings data is newer than our last update
	if data.Standings != nil && data.UpdateTime.After(elo.lastUpdated) {
		return true
	}

	// Update if it's been more than the update frequency
	if time.Since(elo.lastUpdated) > elo.updateStats.UpdateFrequency {
		return true
	}

	return false
}

// ========== Live Update Helper Methods ==========

// processGameResult processes a single game result to update Elo ratings
func (elo *EloRatingModel) processGameResult(gameResult *models.GameResult) error {
	if gameResult.GameState != "FINAL" && gameResult.GameState != "OFF" {
		return fmt.Errorf("game %d not finished yet", gameResult.GameID)
	}

	homeTeam := gameResult.HomeTeam
	awayTeam := gameResult.AwayTeam
	homeScore := gameResult.HomeScore
	awayScore := gameResult.AwayScore
	wasOvertime := gameResult.IsOvertime || gameResult.IsShootout

	// Get current ratings
	homeRating := elo.getTeamRating(homeTeam)
	awayRating := elo.getTeamRating(awayTeam)

	// Determine actual result
	var homeResult float64
	var gameResultStr string

	if homeScore > awayScore {
		homeResult = 1.0 // Home win
		gameResultStr = "W"
	} else if homeScore < awayScore {
		if wasOvertime {
			homeResult = 0.5 // Home OT/SO loss (gets some credit)
			gameResultStr = "OTL"
		} else {
			homeResult = 0.0 // Home regulation loss
			gameResultStr = "L"
		}
	} else {
		homeResult = 0.5 // Tie (shouldn't happen in modern NHL)
		gameResultStr = "T"
	}

	// Calculate dynamic K-factor based on various factors
	kFactor := elo.calculateDynamicKFactor(gameResult, homeRating, awayRating)

	// Calculate expected results
	adjustedHomeRating := homeRating + elo.homeAdvantage
	expectedHomeResult := elo.calculateWinProbability(adjustedHomeRating, awayRating)

	// Update ratings
	homeChange := kFactor * (homeResult - expectedHomeResult)
	awayChange := kFactor * ((1.0 - homeResult) - (1.0 - expectedHomeResult))

	newHomeRating := homeRating + homeChange
	newAwayRating := awayRating + awayChange

	// Update team ratings
	elo.teamRatings[homeTeam] = newHomeRating
	elo.teamRatings[awayTeam] = newAwayRating

	// Update confidence factors based on rating stability
	elo.updateConfidenceFactors(homeTeam, homeChange)
	elo.updateConfidenceFactors(awayTeam, awayChange)

	// Record rating history
	elo.recordRatingChange(homeTeam, homeRating, newHomeRating, homeChange, awayTeam, gameResultStr,
		fmt.Sprintf("%d-%d", homeScore, awayScore), kFactor, true, gameResult.GameDate)

	awayResultStr := "L"
	if gameResultStr == "W" {
		awayResultStr = "L"
	} else if gameResultStr == "L" {
		awayResultStr = "W"
	} else if gameResultStr == "OTL" {
		awayResultStr = "W"
	}

	elo.recordRatingChange(awayTeam, awayRating, newAwayRating, awayChange, homeTeam, awayResultStr,
		fmt.Sprintf("%d-%d", awayScore, homeScore), kFactor, false, gameResult.GameDate)

	log.Printf("🏆 Elo Update: %s %.0f→%.0f (%+.0f), %s %.0f→%.0f (%+.0f)",
		homeTeam, homeRating, newHomeRating, homeChange,
		awayTeam, awayRating, newAwayRating, awayChange)

	return nil
}

// calculateDynamicKFactor calculates an adaptive K-factor based on game context
func (elo *EloRatingModel) calculateDynamicKFactor(gameResult *models.GameResult, homeRating, awayRating float64) float64 {
	baseK := elo.kFactor

	// PHASE 2: Learning rate decay based on games played
	homeGames := len(elo.ratingHistory[gameResult.HomeTeam])
	awayGames := len(elo.ratingHistory[gameResult.AwayTeam])
	avgGamesPlayed := float64(homeGames+awayGames) / 2.0

	// Exponential decay: higher learning early season, lower as teams stabilize
	decayFactor := 1.0 / (1.0 + (avgGamesPlayed / 30.0)) // Decay over ~30 games
	baseK *= (0.7 + 0.3*decayFactor)                     // Never go below 70% of base K

	// Reduce K-factor for overtime/shootout games (less predictive)
	if gameResult.IsOvertime || gameResult.IsShootout {
		baseK *= 0.8
	}

	// Increase K-factor for blowout games (more decisive)
	scoreDiff := int(math.Abs(float64(gameResult.HomeScore - gameResult.AwayScore)))
	if scoreDiff >= 4 {
		baseK *= 1.2
	} else if scoreDiff >= 3 {
		baseK *= 1.1
	}

	// Adjust K-factor based on rating difference (upset protection)
	ratingDiff := math.Abs(homeRating - awayRating)
	if ratingDiff > 200 {
		// Big upset or expected result - adjust accordingly
		if (homeRating > awayRating && gameResult.HomeScore > gameResult.AwayScore) ||
			(awayRating > homeRating && gameResult.AwayScore > gameResult.HomeScore) {
			// Expected result - reduce K-factor
			baseK *= 0.9
		} else {
			// Upset - increase K-factor
			baseK *= 1.3
		}
	}

	// PHASE 2: Game importance multiplier
	importanceMultiplier := 1.0

	// Check if it's a playoff game (game number > 1230 typically indicates playoffs)
	if gameResult.GameID > 2025021230 {
		importanceMultiplier = 1.5 // Playoffs are 50% more important
	}

	// Check for divisional games (same first letter of team code often indicates division)
	// This is a simple heuristic - in production would use actual division data
	if len(gameResult.HomeTeam) > 0 && len(gameResult.AwayTeam) > 0 {
		// Divisional games matter more (rivals, playoff implications)
		// Note: This is simplified - real implementation would check actual divisions
		importanceMultiplier *= 1.1
	}

	baseK *= importanceMultiplier

	// Season-based adjustments (early season = higher K, late season = lower K)
	// Already handled by decay factor above

	return baseK
}

// updateFromStandings updates ratings based on current standings when no game results available
func (elo *EloRatingModel) updateFromStandings(standings *models.StandingsResponse) error {
	log.Printf("📊 Updating Elo ratings from standings data...")

	// This is a fallback method when we don't have individual game results
	// We'll make small adjustments based on current standings position

	totalTeams := len(standings.Standings)
	if totalTeams == 0 {
		return fmt.Errorf("no teams in standings")
	}

	for i, team := range standings.Standings {
		teamCode := team.TeamAbbrev.Default
		currentRating := elo.getTeamRating(teamCode)

		// Calculate expected rating based on standings position
		standingsPosition := float64(i + 1) // 1-based position
		expectedRating := elo.calculateExpectedRatingFromStandings(standingsPosition, float64(totalTeams))

		// Make small adjustment toward expected rating
		adjustment := (expectedRating - currentRating) * 0.05 // 5% adjustment
		newRating := currentRating + adjustment

		elo.teamRatings[teamCode] = newRating

		if math.Abs(adjustment) > 5 { // Only log significant changes
			log.Printf("📊 Standings adjustment: %s %.0f→%.0f (%+.1f)",
				teamCode, currentRating, newRating, adjustment)
		}
	}

	return nil
}

// calculateExpectedRatingFromStandings calculates expected Elo rating based on standings position
func (elo *EloRatingModel) calculateExpectedRatingFromStandings(position, totalTeams float64) float64 {
	// Convert standings position to expected rating
	// Position 1 (best) should be above average, position totalTeams (worst) should be below

	// Normalize position to 0-1 scale (0 = best, 1 = worst)
	normalizedPosition := (position - 1) / (totalTeams - 1)

	// Convert to rating scale (1700 for best team, 1300 for worst team)
	expectedRating := 1700 - (normalizedPosition * 400)

	return expectedRating
}

// applySeasonalDecay applies gradual rating decay over time
func (elo *EloRatingModel) applySeasonalDecay() {
	// Apply decay if it's been more than a week since last update
	if time.Since(elo.lastUpdated) < 7*24*time.Hour {
		return
	}

	decayApplied := false
	for teamCode, rating := range elo.teamRatings {
		// Decay ratings toward the mean (1500)
		decayAmount := (rating - elo.initialRating) * (1.0 - elo.seasonDecayRate)
		newRating := rating - decayAmount

		if math.Abs(decayAmount) > 1.0 { // Only apply significant decay
			elo.teamRatings[teamCode] = newRating
			decayApplied = true
		}
	}

	if decayApplied {
		log.Printf("📉 Applied seasonal decay to Elo ratings")
	}
}

// updateConfidenceFactors updates confidence in team ratings based on rating stability
func (elo *EloRatingModel) updateConfidenceFactors(teamCode string, ratingChange float64) {
	currentConfidence := elo.confidenceFactors[teamCode]
	if currentConfidence == 0 {
		currentConfidence = 0.5 // Start with medium confidence
	}

	// Large rating changes reduce confidence, small changes increase it
	changeImpact := math.Abs(ratingChange) / 50.0 // Normalize to 0-1 scale
	if changeImpact > 1.0 {
		changeImpact = 1.0
	}

	// Adjust confidence
	if changeImpact > 0.2 {
		// Large change - reduce confidence
		currentConfidence *= (1.0 - changeImpact*0.1)
	} else {
		// Small change - increase confidence slightly
		currentConfidence = math.Min(1.0, currentConfidence+0.01)
	}

	elo.confidenceFactors[teamCode] = currentConfidence
}

// recordRatingChange records a rating change in the team's history
func (elo *EloRatingModel) recordRatingChange(teamCode string, oldRating, newRating, change float64,
	opponent, result, score string, kFactor float64, wasHome bool, gameDate time.Time) {

	record := RatingRecord{
		Date:       gameDate,
		OldRating:  oldRating,
		NewRating:  newRating,
		Change:     change,
		Opponent:   opponent,
		GameResult: result,
		GameScore:  score,
		KFactor:    kFactor,
		WasHome:    wasHome,
		Confidence: elo.confidenceFactors[teamCode],
	}

	if elo.ratingHistory[teamCode] == nil {
		elo.ratingHistory[teamCode] = make([]RatingRecord, 0)
	}

	elo.ratingHistory[teamCode] = append(elo.ratingHistory[teamCode], record)

	// Limit history size to last 100 games per team
	if len(elo.ratingHistory[teamCode]) > 100 {
		elo.ratingHistory[teamCode] = elo.ratingHistory[teamCode][1:]
	}
}

// GetRatingHistory returns the rating history for a team
func (elo *EloRatingModel) GetRatingHistory(teamCode string) []RatingRecord {
	elo.mutex.RLock()
	defer elo.mutex.RUnlock()

	if history, exists := elo.ratingHistory[teamCode]; exists {
		// Return a copy to prevent external modification
		historyCopy := make([]RatingRecord, len(history))
		copy(historyCopy, history)
		return historyCopy
	}

	return make([]RatingRecord, 0)
}

// GetTeamConfidence returns the confidence factor for a team's rating
func (elo *EloRatingModel) GetTeamConfidence(teamCode string) float64 {
	elo.mutex.RLock()
	defer elo.mutex.RUnlock()

	if confidence, exists := elo.confidenceFactors[teamCode]; exists {
		return confidence
	}

	return 0.5 // Default medium confidence
}

// saveRatings persists the current Elo ratings to disk
func (elo *EloRatingModel) saveRatings() error {
	elo.mutex.RLock()
	defer elo.mutex.RUnlock()

	data := EloModelData{
		TeamRatings:       elo.teamRatings,
		RatingHistory:     elo.ratingHistory,
		ConfidenceFactors: elo.confidenceFactors,
		LastUpdated:       time.Now(),
		Version:           "1.0",
	}

	return elo.saveRatingsWithData(data)
}

// saveRatingsWithData saves the provided data to disk without acquiring locks
func (elo *EloRatingModel) saveRatingsWithData(data EloModelData) error {
	filePath := filepath.Join(elo.dataDir, "elo_ratings.json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling Elo ratings: %v", err)
	}

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing Elo ratings: %v", err)
	}

	log.Printf("💾 Elo ratings saved: %d teams tracked", len(data.TeamRatings))
	return nil
}

// loadRatings loads persisted Elo ratings from disk
func (elo *EloRatingModel) loadRatings() error {
	filePath := filepath.Join(elo.dataDir, "elo_ratings.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("📊 No existing Elo ratings found, starting fresh")
		return nil
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading Elo ratings: %v", err)
	}

	var data EloModelData
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return fmt.Errorf("error unmarshaling Elo ratings: %v", err)
	}

	elo.mutex.Lock()
	defer elo.mutex.Unlock()

	elo.teamRatings = data.TeamRatings
	elo.ratingHistory = data.RatingHistory
	elo.confidenceFactors = data.ConfidenceFactors
	elo.lastUpdated = data.LastUpdated

	log.Printf("📊 Loaded Elo ratings: %d teams tracked (last updated: %s)",
		len(elo.teamRatings), data.LastUpdated.Format("2006-01-02 15:04:05"))

	return nil
}
