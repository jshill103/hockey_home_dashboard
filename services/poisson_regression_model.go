package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// PoissonRegressionModel implements goal prediction using Poisson distribution
// Hockey goals follow a Poisson distribution, making this model highly effective
type PoissonRegressionModel struct {
	leagueAvgGoalsPerGame float64                 // League average goals per team per game
	teamOffensiveRates    map[string]float64      // Offensive rate parameters (λ)
	teamDefensiveRates    map[string]float64      // Defensive rate parameters (μ)
	rateHistory           map[string][]RateRecord // Historical rate changes
	homeAdvantage         float64                 // Home ice advantage multiplier
	weight                float64                 // Model weight in ensemble
	lastUpdated           time.Time
	mutex                 sync.RWMutex       // Thread safety
	updateStats           ModelUpdateStats   // Update statistics
	learningRate          float64            // Adaptive learning rate
	confidenceTracking    map[string]float64 // Confidence in rate estimates
	seasonDecayRate       float64            // Rate decay over time
	rand                  *rand.Rand         // For Poisson sampling
	dataDir               string             // Directory for persistent storage
}

// PoissonModelData represents the serializable state of the Poisson model
type PoissonModelData struct {
	TeamOffensiveRates map[string]float64      `json:"teamOffensiveRates"`
	TeamDefensiveRates map[string]float64      `json:"teamDefensiveRates"`
	RateHistory        map[string][]RateRecord `json:"rateHistory"`
	ConfidenceTracking map[string]float64      `json:"confidenceTracking"`
	LastUpdated        time.Time               `json:"lastUpdated"`
	Version            string                  `json:"version"`
}

// RateRecord tracks historical rate changes
type RateRecord struct {
	Date            time.Time `json:"date"`
	OldOffensive    float64   `json:"oldOffensive"`
	NewOffensive    float64   `json:"newOffensive"`
	OldDefensive    float64   `json:"oldDefensive"`
	NewDefensive    float64   `json:"newDefensive"`
	OffensiveChange float64   `json:"offensiveChange"`
	DefensiveChange float64   `json:"defensiveChange"`
	Opponent        string    `json:"opponent"`
	GameScore       string    `json:"gameScore"`
	LearningRate    float64   `json:"learningRate"`
	WasHome         bool      `json:"wasHome"`
	GoalsExpected   float64   `json:"goalsExpected"`
	GoalsActual     int       `json:"goalsActual"`
	Confidence      float64   `json:"confidence"`
}

// NewPoissonRegressionModel creates a new Poisson regression prediction model
func NewPoissonRegressionModel() *PoissonRegressionModel {
	model := &PoissonRegressionModel{
		leagueAvgGoalsPerGame: 3.1, // Approximate NHL average goals per team per game
		teamOffensiveRates:    make(map[string]float64),
		teamDefensiveRates:    make(map[string]float64),
		rateHistory:           make(map[string][]RateRecord),
		confidenceTracking:    make(map[string]float64),
		homeAdvantage:         1.08, // Home teams score ~8% more goals
		weight:                0.15, // 15% weight in ensemble
		learningRate:          0.1,  // Adaptive learning rate
		seasonDecayRate:       0.98, // 2% decay per season
		lastUpdated:           time.Now(),
		rand:                  rand.New(rand.NewSource(time.Now().UnixNano())),
		dataDir:               "data/models",
		updateStats: ModelUpdateStats{
			UpdateFrequency: 1 * time.Hour,
		},
	}

	// Create data directory if it doesn't exist
	os.MkdirAll(model.dataDir, 0755)

	// Load existing rates if available
	if err := model.loadRates(); err != nil {
		log.Printf("⚠️ Could not load Poisson rates: %v (starting fresh)", err)
	}

	return model
}

// Predict implements the PredictionModel interface for Poisson regression
func (pr *PoissonRegressionModel) Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error) {
	startTime := time.Now()

	log.Printf("🎯 Running Poisson Regression prediction for %s vs %s...", homeFactors.TeamCode, awayFactors.TeamCode)

	// Calculate expected goals for each team using Poisson regression
	homeExpectedGoals := pr.calculateExpectedGoals(homeFactors, awayFactors, true)
	awayExpectedGoals := pr.calculateExpectedGoals(awayFactors, homeFactors, false)

	// Calculate win probability using Poisson distributions
	winProbability := pr.calculateWinProbability(homeExpectedGoals, awayExpectedGoals)

	// Predict most likely score
	homeScore, awayScore := pr.predictMostLikelyScore(homeExpectedGoals, awayExpectedGoals)

	// Calculate confidence based on goal expectation certainty
	confidence := pr.calculateConfidence(homeExpectedGoals, awayExpectedGoals, homeFactors, awayFactors)

	processingTime := time.Since(startTime).Milliseconds()

	log.Printf("🎯 Expected Goals - %s: %.2f, %s: %.2f",
		homeFactors.TeamCode, homeExpectedGoals, awayFactors.TeamCode, awayExpectedGoals)
	log.Printf("📊 Poisson Regression: %.1f%% confidence, %d-%d prediction", confidence*100, homeScore, awayScore)

	return &models.ModelResult{
		ModelName:      "Poisson Regression",
		WinProbability: winProbability,
		Confidence:     confidence,
		PredictedScore: fmt.Sprintf("%d-%d", homeScore, awayScore),
		Weight:         0.20, // Will be dynamically adjusted by ensemble
		ProcessingTime: processingTime,
	}, nil
}

// GetName implements the PredictionModel interface
func (pr *PoissonRegressionModel) GetName() string {
	return "Poisson Regression"
}

// GetWeight implements the PredictionModel interface
func (pr *PoissonRegressionModel) GetWeight() float64 {
	return pr.weight
}

// calculateExpectedGoals calculates expected goals using Poisson regression approach
func (pr *PoissonRegressionModel) calculateExpectedGoals(teamFactors, opponentFactors *models.PredictionFactors, isHome bool) float64 {
	// Get team's offensive and opponent's defensive rates
	offensiveRate := pr.getOffensiveRate(teamFactors.TeamCode)
	defensiveRate := pr.getDefensiveRate(opponentFactors.TeamCode)

	// Base expected goals = (Team Offensive Rate) × (Opponent Defensive Rate) × League Average
	baseExpectedGoals := offensiveRate * defensiveRate * pr.leagueAvgGoalsPerGame

	// Apply home ice advantage
	if isHome {
		baseExpectedGoals *= pr.homeAdvantage
	}

	// Apply situational adjustments
	situationalMultiplier := pr.calculateSituationalMultiplier(teamFactors, opponentFactors, isHome)
	adjustedExpectedGoals := baseExpectedGoals * situationalMultiplier

	// Apply advanced analytics adjustments
	analyticsMultiplier := pr.calculateAnalyticsMultiplier(teamFactors)
	finalExpectedGoals := adjustedExpectedGoals * analyticsMultiplier

	// Ensure reasonable bounds (NHL games typically see 0-8 goals per team)
	if finalExpectedGoals < 0.5 {
		finalExpectedGoals = 0.5
	} else if finalExpectedGoals > 7.0 {
		finalExpectedGoals = 7.0
	}

	return finalExpectedGoals
}

// getOffensiveRate calculates or retrieves team's offensive rate parameter
func (pr *PoissonRegressionModel) getOffensiveRate(teamCode string) float64 {
	if rate, exists := pr.teamOffensiveRates[teamCode]; exists {
		return rate
	}

	// Initialize based on team's historical performance
	rate := pr.calculateInitialOffensiveRate(teamCode)
	pr.teamOffensiveRates[teamCode] = rate

	log.Printf("🆕 Initialized offensive rate for %s: %.3f", teamCode, rate)
	return rate
}

// getDefensiveRate calculates or retrieves team's defensive rate parameter
func (pr *PoissonRegressionModel) getDefensiveRate(teamCode string) float64 {
	if rate, exists := pr.teamDefensiveRates[teamCode]; exists {
		return rate
	}

	// Initialize based on team's historical performance
	rate := pr.calculateInitialDefensiveRate(teamCode)
	pr.teamDefensiveRates[teamCode] = rate

	log.Printf("🆕 Initialized defensive rate for %s: %.3f", teamCode, rate)
	return rate
}

// calculateInitialOffensiveRate determines initial offensive rate based on team performance
func (pr *PoissonRegressionModel) calculateInitialOffensiveRate(teamCode string) float64 {
	// Base rate of 1.0 represents league average offensive capability
	baseRate := 1.0

	// Adjust based on team's known offensive capabilities
	offensiveAdjustments := map[string]float64{
		"EDM": 1.25, // McDavid, Draisaitl - elite offense
		"COL": 1.18, // MacKinnon, Rantanen - strong offense
		"TOR": 1.15, // Matthews, Marner - high-powered offense
		"FLA": 1.12, // Balanced, Cup-winning offense
		"VGK": 1.08, // Solid offensive team
		"NYR": 1.05, // Good offensive depth
		"BOS": 1.03, // Consistent offense
		"CAR": 1.00, // League average
		"UTA": 0.95, // New franchise, unknown but likely average
		"SJS": 0.85, // Rebuilding, weaker offense
	}

	if adjustment, exists := offensiveAdjustments[teamCode]; exists {
		return adjustment
	}

	return baseRate
}

// calculateInitialDefensiveRate determines initial defensive rate based on team performance
func (pr *PoissonRegressionModel) calculateInitialDefensiveRate(teamCode string) float64 {
	// Base rate of 1.0 represents league average defensive capability
	// Lower values = better defense (allow fewer goals)
	baseRate := 1.0

	// Adjust based on team's known defensive capabilities
	defensiveAdjustments := map[string]float64{
		"BOS": 0.88, // Elite defensive system
		"FLA": 0.90, // Strong defensive team
		"CAR": 0.92, // Good defensive structure
		"VGK": 0.95, // Solid defensive team
		"COL": 0.98, // Decent defense
		"NYR": 1.00, // Average defense
		"TOR": 1.05, // Weaker defensive team
		"EDM": 1.08, // Offense-first, weaker defense
		"UTA": 1.00, // Unknown, assume average
		"SJS": 1.12, // Rebuilding, weaker defense
	}

	if adjustment, exists := defensiveAdjustments[teamCode]; exists {
		return adjustment
	}

	return baseRate
}

// calculateSituationalMultiplier applies situational factors to expected goals
func (pr *PoissonRegressionModel) calculateSituationalMultiplier(teamFactors, opponentFactors *models.PredictionFactors, isHome bool) float64 {
	multiplier := 1.0

	// Recent form impact (hot/cold streaks affect scoring)
	formImpact := (teamFactors.RecentForm - 0.5) * 0.2 // ±10% based on recent form
	multiplier += formImpact

	// Injury impact (key players missing reduces scoring)
	injuryPenalty := teamFactors.InjuryImpact.ImpactScore * 0.005 // Up to -25% for severe injuries
	multiplier -= injuryPenalty

	// Momentum factors (teams on winning streaks score more)
	momentumBoost := (teamFactors.MomentumFactors.MomentumScore - 0.5) * 0.15
	multiplier += momentumBoost

	// Travel fatigue (tired teams score less)
	if !isHome {
		fatigueImpact := teamFactors.TravelFatigue.FatigueScore * 0.1
		multiplier -= fatigueImpact
	}

	// Rest advantage/disadvantage
	if teamFactors.RestDays > opponentFactors.RestDays {
		restAdvantage := math.Min(float64(teamFactors.RestDays-opponentFactors.RestDays)*0.02, 0.08)
		multiplier += restAdvantage
	} else if teamFactors.RestDays < opponentFactors.RestDays {
		restDisadvantage := math.Min(float64(opponentFactors.RestDays-teamFactors.RestDays)*0.02, 0.08)
		multiplier -= restDisadvantage
	}

	// Back-to-back penalty
	multiplier -= teamFactors.BackToBackPenalty * 0.15

	// Ensure multiplier stays within reasonable bounds
	if multiplier < 0.7 {
		multiplier = 0.7
	} else if multiplier > 1.4 {
		multiplier = 1.4
	}

	return multiplier
}

// calculateAnalyticsMultiplier applies advanced analytics to expected goals
func (pr *PoissonRegressionModel) calculateAnalyticsMultiplier(teamFactors *models.PredictionFactors) float64 {
	if teamFactors.AdvancedStats.OverallRating == 0 {
		return 1.0 // No advanced stats available
	}

	multiplier := 1.0

	// Expected goals differential impact
	xgDiff := teamFactors.AdvancedStats.XGDifferential
	xgImpact := xgDiff * 0.05 // Each +1.0 xG diff = +5% scoring
	multiplier += xgImpact

	// Shooting talent (goals above/below expected)
	shootingTalent := teamFactors.AdvancedStats.ShootingTalent
	multiplier += shootingTalent * 0.03

	// Possession quality impact
	possessionImpact := (teamFactors.AdvancedStats.PossessionQuality - 50.0) / 50.0 * 0.08
	multiplier += possessionImpact

	// High danger chances impact
	hdcImpact := (teamFactors.AdvancedStats.HighDangerPct - 50.0) / 50.0 * 0.06
	multiplier += hdcImpact

	// Special teams impact (power play effectiveness)
	specialTeamsImpact := (teamFactors.AdvancedStats.SpecialTeamsEdge - 0.5) * 0.04
	multiplier += specialTeamsImpact

	// Cap the analytics impact
	if multiplier < 0.8 {
		multiplier = 0.8
	} else if multiplier > 1.25 {
		multiplier = 1.25
	}

	return multiplier
}

// calculateWinProbability calculates win probability using Poisson distributions
func (pr *PoissonRegressionModel) calculateWinProbability(homeExpected, awayExpected float64) float64 {
	// Monte Carlo simulation using Poisson distributions
	simulations := 10000
	homeWins := 0

	for i := 0; i < simulations; i++ {
		homeGoals := pr.samplePoisson(homeExpected)
		awayGoals := pr.samplePoisson(awayExpected)

		if homeGoals > awayGoals {
			homeWins++
		}
	}

	return float64(homeWins) / float64(simulations)
}

// predictMostLikelyScore predicts the most probable score outcome
func (pr *PoissonRegressionModel) predictMostLikelyScore(homeExpected, awayExpected float64) (int, int) {
	// For Poisson distribution, the mode (most likely value) is floor(λ) when λ is not an integer
	// and both floor(λ) and floor(λ)+1 when λ is an integer

	homeMode := int(math.Floor(homeExpected))
	awayMode := int(math.Floor(awayExpected))

	// Adjust for common hockey scoring patterns
	homeScore := pr.adjustScoreForHockeyReality(homeMode, homeExpected)
	awayScore := pr.adjustScoreForHockeyReality(awayMode, awayExpected)

	return homeScore, awayScore
}

// adjustScoreForHockeyReality adjusts predicted scores based on hockey-specific patterns
func (pr *PoissonRegressionModel) adjustScoreForHockeyReality(mode int, expected float64) int {
	// If expected is very close to the next integer, consider rounding up
	if expected-float64(mode) > 0.7 {
		return mode + 1
	}

	// Ensure minimum of 1 goal if expected > 0.8 (NHL games rarely end 0-0)
	if mode == 0 && expected > 0.8 {
		return 1
	}

	return mode
}

// calculateConfidence determines prediction confidence based on various factors
func (pr *PoissonRegressionModel) calculateConfidence(homeExpected, awayExpected float64, homeFactors, awayFactors *models.PredictionFactors) float64 {
	// Base confidence from goal expectation difference
	goalDiff := math.Abs(homeExpected - awayExpected)
	baseConfidence := 0.6 + (goalDiff * 0.08) // Larger differences = higher confidence

	// Adjust for data quality
	dataQualityBonus := 0.0
	if homeFactors.AdvancedStats.OverallRating > 0 && awayFactors.AdvancedStats.OverallRating > 0 {
		dataQualityBonus += 0.05 // Bonus for having advanced stats
	}

	// Adjust for situational uncertainty
	injuryUncertainty := (homeFactors.InjuryImpact.ImpactScore + awayFactors.InjuryImpact.ImpactScore) * 0.002
	travelUncertainty := awayFactors.TravelFatigue.FatigueScore * 0.03

	totalConfidence := baseConfidence + dataQualityBonus - injuryUncertainty - travelUncertainty

	// Cap confidence between 0.65 and 0.92
	if totalConfidence > 0.92 {
		totalConfidence = 0.92
	} else if totalConfidence < 0.65 {
		totalConfidence = 0.65
	}

	return totalConfidence
}

// samplePoisson generates a random sample from a Poisson distribution with parameter λ
func (pr *PoissonRegressionModel) samplePoisson(lambda float64) int {
	// Use Knuth's algorithm for small lambda values
	if lambda < 30 {
		return pr.poissonKnuth(lambda)
	}

	// Use normal approximation for large lambda values
	return pr.poissonNormal(lambda)
}

// poissonKnuth implements Knuth's algorithm for Poisson sampling
func (pr *PoissonRegressionModel) poissonKnuth(lambda float64) int {
	L := math.Exp(-lambda)
	p := 1.0
	k := 0

	for {
		k++
		p *= pr.rand.Float64()
		if p <= L {
			break
		}
	}

	return k - 1
}

// poissonNormal uses normal approximation for large lambda Poisson sampling
func (pr *PoissonRegressionModel) poissonNormal(lambda float64) int {
	// Normal approximation: N(λ, λ)
	normal := pr.rand.NormFloat64()*math.Sqrt(lambda) + lambda
	result := int(math.Round(normal))

	// Ensure non-negative result
	if result < 0 {
		result = 0
	}

	return result
}

// UpdateRates updates offensive and defensive rates based on actual game results (for future implementation)
func (pr *PoissonRegressionModel) UpdateRates(homeTeam, awayTeam string, homeScore, awayScore int) {
	// Get current rates
	homeOffensive := pr.getOffensiveRate(homeTeam)
	homeDefensive := pr.getDefensiveRate(homeTeam)
	awayOffensive := pr.getOffensiveRate(awayTeam)
	awayDefensive := pr.getDefensiveRate(awayTeam)

	// Calculate expected goals with current rates
	homeExpected := homeOffensive * awayDefensive * pr.leagueAvgGoalsPerGame * pr.homeAdvantage
	awayExpected := awayOffensive * homeDefensive * pr.leagueAvgGoalsPerGame

	// Update rates based on actual vs expected (simplified approach)
	learningRate := 0.1

	// Update offensive rates
	homeOffensiveError := float64(homeScore) - homeExpected
	awayOffensiveError := float64(awayScore) - awayExpected

	pr.teamOffensiveRates[homeTeam] += learningRate * homeOffensiveError / homeExpected
	pr.teamOffensiveRates[awayTeam] += learningRate * awayOffensiveError / awayExpected

	// Update defensive rates (inverse relationship)
	homeDefensiveError := awayExpected - float64(awayScore)
	awayDefensiveError := homeExpected - float64(homeScore)

	pr.teamDefensiveRates[homeTeam] += learningRate * homeDefensiveError / awayExpected
	pr.teamDefensiveRates[awayTeam] += learningRate * awayDefensiveError / homeExpected

	// Ensure rates stay within reasonable bounds
	pr.boundRate(homeTeam, true)
	pr.boundRate(homeTeam, false)
	pr.boundRate(awayTeam, true)
	pr.boundRate(awayTeam, false)

	pr.lastUpdated = time.Now()

	log.Printf("🎯 Poisson Update: %s Off: %.3f, Def: %.3f | %s Off: %.3f, Def: %.3f",
		homeTeam, pr.teamOffensiveRates[homeTeam], pr.teamDefensiveRates[homeTeam],
		awayTeam, pr.teamOffensiveRates[awayTeam], pr.teamDefensiveRates[awayTeam])
}

// boundRate ensures rates stay within reasonable bounds
func (pr *PoissonRegressionModel) boundRate(teamCode string, isOffensive bool) {
	if isOffensive {
		if rate := pr.teamOffensiveRates[teamCode]; rate < 0.5 {
			pr.teamOffensiveRates[teamCode] = 0.5
		} else if rate > 1.8 {
			pr.teamOffensiveRates[teamCode] = 1.8
		}
	} else {
		if rate := pr.teamDefensiveRates[teamCode]; rate < 0.6 {
			pr.teamDefensiveRates[teamCode] = 0.6
		} else if rate > 1.5 {
			pr.teamDefensiveRates[teamCode] = 1.5
		}
	}
}

// GetTeamRates returns current offensive and defensive rates for a team
func (pr *PoissonRegressionModel) GetTeamRates(teamCode string) (offensive, defensive float64) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return pr.getOffensiveRate(teamCode), pr.getDefensiveRate(teamCode)
}

// ========== UpdatableModel Interface Implementation ==========

// Update implements the UpdatableModel interface for live data updates
func (pr *PoissonRegressionModel) Update(data *LiveGameData) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	log.Printf("🎯 Updating Poisson regression rates with new data...")

	updateCount := 0
	errorCount := 0

	// Process game results to update rates
	for _, gameResult := range data.GameResults {
		if err := pr.processGameResult(gameResult); err != nil {
			log.Printf("❌ Failed to process game result %d: %v", gameResult.GameID, err)
			errorCount++
		} else {
			updateCount++
		}
	}

	// Update rates based on standings if no game results
	if len(data.GameResults) == 0 && data.Standings != nil {
		if err := pr.updateFromStandings(data.Standings); err != nil {
			log.Printf("⚠️ Failed to update from standings: %v", err)
			errorCount++
		} else {
			updateCount++
		}
	}

	// Apply seasonal decay if enough time has passed
	pr.applySeasonalDecay()

	// Update learning rate based on recent performance
	pr.updateAdaptiveLearningRate()

	// Update statistics
	pr.updateStats.TotalUpdates++
	if errorCount == 0 {
		pr.updateStats.SuccessfulUpdates++
	} else {
		pr.updateStats.FailedUpdates++
	}
	pr.updateStats.LastUpdateTime = time.Now()
	pr.lastUpdated = time.Now()

	log.Printf("✅ Poisson regression update completed: %d processed, %d errors", updateCount, errorCount)

	// Save rates to disk after update
	// Note: saveRates acquires its own lock, so we need to release ours first
	pr.mutex.Unlock()
	if err := pr.saveRates(); err != nil {
		log.Printf("⚠️ Failed to save Poisson rates: %v", err)
	}
	pr.mutex.Lock() // Re-acquire before defer unlock

	return nil
}

// GetLastUpdate implements the UpdatableModel interface
func (pr *PoissonRegressionModel) GetLastUpdate() time.Time {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return pr.lastUpdated
}

// GetUpdateStats implements the UpdatableModel interface
func (pr *PoissonRegressionModel) GetUpdateStats() ModelUpdateStats {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	// Calculate data freshness
	pr.updateStats.DataFreshness = time.Since(pr.lastUpdated)

	// Calculate average update time (simplified)
	if pr.updateStats.TotalUpdates > 0 {
		pr.updateStats.AverageUpdateTime = time.Since(pr.lastUpdated) / time.Duration(pr.updateStats.TotalUpdates)
	}

	return pr.updateStats
}

// GetModelName implements the UpdatableModel interface
func (pr *PoissonRegressionModel) GetModelName() string {
	return "Poisson Regression"
}

// RequiresUpdate implements the UpdatableModel interface
func (pr *PoissonRegressionModel) RequiresUpdate(data *LiveGameData) bool {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	// Always update if we have new game results
	if len(data.GameResults) > 0 {
		return true
	}

	// Update if standings data is newer than our last update
	if data.Standings != nil && data.UpdateTime.After(pr.lastUpdated) {
		return true
	}

	// Update if it's been more than the update frequency
	if time.Since(pr.lastUpdated) > pr.updateStats.UpdateFrequency {
		return true
	}

	return false
}

// ========== Live Update Helper Methods ==========

// processGameResult processes a single game result to update Poisson rates
func (pr *PoissonRegressionModel) processGameResult(gameResult *models.GameResult) error {
	if gameResult.GameState != "FINAL" && gameResult.GameState != "OFF" {
		return fmt.Errorf("game %d not finished yet", gameResult.GameID)
	}

	homeTeam := gameResult.HomeTeam
	awayTeam := gameResult.AwayTeam
	homeScore := gameResult.HomeScore
	awayScore := gameResult.AwayScore

	// Get current rates
	homeOffensive := pr.getOffensiveRate(homeTeam)
	homeDefensive := pr.getDefensiveRate(homeTeam)
	awayOffensive := pr.getOffensiveRate(awayTeam)
	awayDefensive := pr.getDefensiveRate(awayTeam)

	// Calculate expected goals with current rates
	homeExpected := homeOffensive * awayDefensive * pr.leagueAvgGoalsPerGame * pr.homeAdvantage
	awayExpected := awayOffensive * homeDefensive * pr.leagueAvgGoalsPerGame

	// Calculate adaptive learning rates based on game context
	homeLearningRate := pr.calculateAdaptiveLearningRate(gameResult, homeTeam, true)
	awayLearningRate := pr.calculateAdaptiveLearningRate(gameResult, awayTeam, false)

	// Update offensive rates based on actual vs expected goals
	homeOffensiveError := float64(homeScore) - homeExpected
	awayOffensiveError := float64(awayScore) - awayExpected

	homeOffensiveChange := homeLearningRate * homeOffensiveError / math.Max(homeExpected, 0.1)
	awayOffensiveChange := awayLearningRate * awayOffensiveError / math.Max(awayExpected, 0.1)

	newHomeOffensive := homeOffensive + homeOffensiveChange
	newAwayOffensive := awayOffensive + awayOffensiveChange

	// Update defensive rates (inverse relationship)
	homeDefensiveError := awayExpected - float64(awayScore)
	awayDefensiveError := homeExpected - float64(homeScore)

	homeDefensiveChange := homeLearningRate * homeDefensiveError / math.Max(awayExpected, 0.1)
	awayDefensiveChange := awayLearningRate * awayDefensiveError / math.Max(homeExpected, 0.1)

	newHomeDefensive := homeDefensive + homeDefensiveChange
	newAwayDefensive := awayDefensive + awayDefensiveChange

	// Apply bounds and update rates
	pr.teamOffensiveRates[homeTeam] = pr.boundOffensiveRate(newHomeOffensive)
	pr.teamDefensiveRates[homeTeam] = pr.boundDefensiveRate(newHomeDefensive)
	pr.teamOffensiveRates[awayTeam] = pr.boundOffensiveRate(newAwayOffensive)
	pr.teamDefensiveRates[awayTeam] = pr.boundDefensiveRate(newAwayDefensive)

	// Update confidence tracking
	pr.updateConfidenceTracking(homeTeam, homeOffensiveError, homeDefensiveError)
	pr.updateConfidenceTracking(awayTeam, awayOffensiveError, awayDefensiveError)

	// Record rate history
	pr.recordRateChange(homeTeam, homeOffensive, newHomeOffensive, homeDefensive, newHomeDefensive,
		homeOffensiveChange, homeDefensiveChange, awayTeam,
		fmt.Sprintf("%d-%d", homeScore, awayScore), homeLearningRate, true,
		homeExpected, homeScore, gameResult.GameDate)

	pr.recordRateChange(awayTeam, awayOffensive, newAwayOffensive, awayDefensive, newAwayDefensive,
		awayOffensiveChange, awayDefensiveChange, homeTeam,
		fmt.Sprintf("%d-%d", awayScore, homeScore), awayLearningRate, false,
		awayExpected, awayScore, gameResult.GameDate)

	log.Printf("🎯 Poisson Update: %s Off: %.3f→%.3f (Δ%+.3f), Def: %.3f→%.3f (Δ%+.3f)",
		homeTeam, homeOffensive, newHomeOffensive, homeOffensiveChange,
		homeDefensive, newHomeDefensive, homeDefensiveChange)
	log.Printf("🎯 Poisson Update: %s Off: %.3f→%.3f (Δ%+.3f), Def: %.3f→%.3f (Δ%+.3f)",
		awayTeam, awayOffensive, newAwayOffensive, awayOffensiveChange,
		awayDefensive, newAwayDefensive, awayDefensiveChange)

	return nil
}

// calculateAdaptiveLearningRate calculates context-aware learning rate
func (pr *PoissonRegressionModel) calculateAdaptiveLearningRate(gameResult *models.GameResult, teamCode string, isHome bool) float64 {
	baseLearningRate := pr.learningRate

	// PHASE 2: Calculate expected vs actual for surprise-based learning
	var teamScore, opponentScore int
	var expected float64

	if isHome {
		teamScore = gameResult.HomeScore
		opponentScore = gameResult.AwayScore
		// Calculate expected based on current rates
		homeOffensive := pr.getOffensiveRate(gameResult.HomeTeam)
		awayDefensive := pr.getDefensiveRate(gameResult.AwayTeam)
		expected = homeOffensive * awayDefensive * pr.leagueAvgGoalsPerGame * pr.homeAdvantage
	} else {
		teamScore = gameResult.AwayScore
		opponentScore = gameResult.HomeScore
		// Calculate expected based on current rates
		awayOffensive := pr.getOffensiveRate(gameResult.AwayTeam)
		homeDefensive := pr.getDefensiveRate(gameResult.HomeTeam)
		expected = awayOffensive * homeDefensive * pr.leagueAvgGoalsPerGame
	}

	// PHASE 2: Surprise-based learning boost
	// Learn more from unexpected results
	actual := float64(teamScore)
	surprise := math.Abs(expected - actual)
	surpriseBoost := 1.0 + math.Min(surprise/3.0, 1.0) // Up to 2x for 3+ goal surprise
	baseLearningRate *= surpriseBoost

	scoreDiff := int(math.Abs(float64(teamScore - opponentScore)))

	// Higher learning rate for more decisive games
	if scoreDiff >= 4 {
		baseLearningRate *= 1.3 // Big blowouts are very informative
	} else if scoreDiff >= 3 {
		baseLearningRate *= 1.2
	} else if scoreDiff <= 1 {
		baseLearningRate *= 0.9 // Close games are less informative about true rates
	}

	// Adjust for overtime/shootout (less predictive)
	if gameResult.IsOvertime || gameResult.IsShootout {
		baseLearningRate *= 0.8
	}

	// Adjust for shutouts (very informative about defensive capability)
	if teamScore == 0 || opponentScore == 0 {
		baseLearningRate *= 1.4
	}

	// PHASE 2: Learning rate decay based on games played
	gamesPlayed := len(pr.rateHistory[teamCode])
	decayFactor := 1.0 / (1.0 + float64(gamesPlayed)/40.0) // Decay over ~40 games
	baseLearningRate *= (0.6 + 0.4*decayFactor)            // Never go below 60% of base rate

	// Adjust based on team's rate confidence
	confidence := pr.confidenceTracking[teamCode]
	if confidence == 0 {
		confidence = 0.5 // Default medium confidence
	}

	// Lower confidence = higher learning rate (reduced weight for Phase 2)
	confidenceAdjustment := 1.5 - (0.5 * confidence)
	baseLearningRate *= confidenceAdjustment

	// Bound the learning rate
	if baseLearningRate > 0.4 {
		baseLearningRate = 0.4
	} else if baseLearningRate < 0.03 {
		baseLearningRate = 0.03
	}

	return baseLearningRate
}

// updateFromStandings updates rates based on current standings when no game results available
func (pr *PoissonRegressionModel) updateFromStandings(standings *models.StandingsResponse) error {
	log.Printf("📊 Updating Poisson rates from standings data...")

	totalTeams := len(standings.Standings)
	if totalTeams == 0 {
		return fmt.Errorf("no teams in standings")
	}

	for _, team := range standings.Standings {
		teamCode := team.TeamAbbrev.Default
		currentOffensive := pr.getOffensiveRate(teamCode)
		currentDefensive := pr.getDefensiveRate(teamCode)

		// Use goals for/against from standings
		gamesPlayed := float64(team.GamesPlayed)
		if gamesPlayed == 0 {
			continue // Skip teams that haven't played
		}

		actualGF := float64(team.GoalFor) / gamesPlayed
		actualGA := float64(team.GoalAgainst) / gamesPlayed

		// Calculate expected rates based on actual performance
		expectedOffensive := actualGF / pr.leagueAvgGoalsPerGame
		expectedDefensive := actualGA / pr.leagueAvgGoalsPerGame

		// Make small adjustments toward expected rates
		offensiveAdjustment := (expectedOffensive - currentOffensive) * 0.03 // 3% adjustment
		defensiveAdjustment := (expectedDefensive - currentDefensive) * 0.03

		newOffensive := pr.boundOffensiveRate(currentOffensive + offensiveAdjustment)
		newDefensive := pr.boundDefensiveRate(currentDefensive + defensiveAdjustment)

		pr.teamOffensiveRates[teamCode] = newOffensive
		pr.teamDefensiveRates[teamCode] = newDefensive

		if math.Abs(offensiveAdjustment) > 0.02 || math.Abs(defensiveAdjustment) > 0.02 {
			log.Printf("📊 Standings adjustment: %s Off: %.3f→%.3f (%+.3f), Def: %.3f→%.3f (%+.3f)",
				teamCode, currentOffensive, newOffensive, offensiveAdjustment,
				currentDefensive, newDefensive, defensiveAdjustment)
		}
	}

	return nil
}

// applySeasonalDecay applies gradual rate decay toward league average
func (pr *PoissonRegressionModel) applySeasonalDecay() {
	// Apply decay if it's been more than a week since last update
	if time.Since(pr.lastUpdated) < 7*24*time.Hour {
		return
	}

	decayApplied := false
	for teamCode, rate := range pr.teamOffensiveRates {
		// Decay offensive rates toward 1.0 (league average)
		decayAmount := (rate - 1.0) * (1.0 - pr.seasonDecayRate)
		newRate := rate - decayAmount

		if math.Abs(decayAmount) > 0.005 { // Only apply significant decay
			pr.teamOffensiveRates[teamCode] = newRate
			decayApplied = true
		}
	}

	for teamCode, rate := range pr.teamDefensiveRates {
		// Decay defensive rates toward 1.0 (league average)
		decayAmount := (rate - 1.0) * (1.0 - pr.seasonDecayRate)
		newRate := rate - decayAmount

		if math.Abs(decayAmount) > 0.005 { // Only apply significant decay
			pr.teamDefensiveRates[teamCode] = newRate
			decayApplied = true
		}
	}

	if decayApplied {
		log.Printf("📉 Applied seasonal decay to Poisson rates")
	}
}

// updateAdaptiveLearningRate adjusts the base learning rate based on recent performance
func (pr *PoissonRegressionModel) updateAdaptiveLearningRate() {
	// This could analyze recent prediction accuracy and adjust learning rate accordingly
	// For now, we'll keep it simple and just ensure it's within bounds
	if pr.learningRate > 0.2 {
		pr.learningRate = 0.2
	} else if pr.learningRate < 0.05 {
		pr.learningRate = 0.05
	}
}

// updateConfidenceTracking updates confidence in team rate estimates
func (pr *PoissonRegressionModel) updateConfidenceTracking(teamCode string, offensiveError, defensiveError float64) {
	currentConfidence := pr.confidenceTracking[teamCode]
	if currentConfidence == 0 {
		currentConfidence = 0.5 // Start with medium confidence
	}

	// Calculate prediction error magnitude
	totalError := math.Abs(offensiveError) + math.Abs(defensiveError)

	// Large errors reduce confidence, small errors increase it
	if totalError > 1.5 {
		// Large prediction error - reduce confidence
		currentConfidence *= 0.95
	} else if totalError < 0.5 {
		// Small prediction error - increase confidence
		currentConfidence = math.Min(1.0, currentConfidence*1.02)
	}

	// Ensure confidence stays within bounds
	if currentConfidence < 0.1 {
		currentConfidence = 0.1
	} else if currentConfidence > 0.95 {
		currentConfidence = 0.95
	}

	pr.confidenceTracking[teamCode] = currentConfidence
}

// boundOffensiveRate ensures offensive rates stay within reasonable bounds
func (pr *PoissonRegressionModel) boundOffensiveRate(rate float64) float64 {
	if rate < 0.5 {
		return 0.5
	} else if rate > 1.8 {
		return 1.8
	}
	return rate
}

// boundDefensiveRate ensures defensive rates stay within reasonable bounds
func (pr *PoissonRegressionModel) boundDefensiveRate(rate float64) float64 {
	if rate < 0.6 {
		return 0.6
	} else if rate > 1.5 {
		return 1.5
	}
	return rate
}

// recordRateChange records a rate change in the team's history
func (pr *PoissonRegressionModel) recordRateChange(teamCode string, oldOffensive, newOffensive, oldDefensive, newDefensive,
	offensiveChange, defensiveChange float64, opponent, score string, learningRate float64, wasHome bool,
	expectedGoals float64, actualGoals int, gameDate time.Time) {

	record := RateRecord{
		Date:            gameDate,
		OldOffensive:    oldOffensive,
		NewOffensive:    newOffensive,
		OldDefensive:    oldDefensive,
		NewDefensive:    newDefensive,
		OffensiveChange: offensiveChange,
		DefensiveChange: defensiveChange,
		Opponent:        opponent,
		GameScore:       score,
		LearningRate:    learningRate,
		WasHome:         wasHome,
		GoalsExpected:   expectedGoals,
		GoalsActual:     actualGoals,
		Confidence:      pr.confidenceTracking[teamCode],
	}

	if pr.rateHistory[teamCode] == nil {
		pr.rateHistory[teamCode] = make([]RateRecord, 0)
	}

	pr.rateHistory[teamCode] = append(pr.rateHistory[teamCode], record)

	// Limit history size to last 100 games per team
	if len(pr.rateHistory[teamCode]) > 100 {
		pr.rateHistory[teamCode] = pr.rateHistory[teamCode][1:]
	}
}

// GetRateHistory returns the rate change history for a team
func (pr *PoissonRegressionModel) GetRateHistory(teamCode string) []RateRecord {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	if history, exists := pr.rateHistory[teamCode]; exists {
		// Return a copy to prevent external modification
		historyCopy := make([]RateRecord, len(history))
		copy(historyCopy, history)
		return historyCopy
	}

	return make([]RateRecord, 0)
}

// GetTeamConfidence returns the confidence factor for a team's rates
func (pr *PoissonRegressionModel) GetTeamConfidence(teamCode string) float64 {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	if confidence, exists := pr.confidenceTracking[teamCode]; exists {
		return confidence
	}

	return 0.5 // Default medium confidence
}

// GetAllRates returns all current team rates
func (pr *PoissonRegressionModel) GetAllRates() map[string]map[string]float64 {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	allRates := make(map[string]map[string]float64)

	for teamCode := range pr.teamOffensiveRates {
		allRates[teamCode] = map[string]float64{
			"offensive": pr.teamOffensiveRates[teamCode],
			"defensive": pr.teamDefensiveRates[teamCode],
		}
	}

	return allRates
}

// saveRates persists the current Poisson rates to disk
func (pr *PoissonRegressionModel) saveRates() error {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	filePath := filepath.Join(pr.dataDir, "poisson_rates.json")

	data := PoissonModelData{
		TeamOffensiveRates: pr.teamOffensiveRates,
		TeamDefensiveRates: pr.teamDefensiveRates,
		RateHistory:        pr.rateHistory,
		ConfidenceTracking: pr.confidenceTracking,
		LastUpdated:        time.Now(),
		Version:            "1.0",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling Poisson rates: %v", err)
	}

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing Poisson rates: %v", err)
	}

	log.Printf("💾 Poisson rates saved: %d teams tracked", len(pr.teamOffensiveRates))
	return nil
}

// loadRates loads persisted Poisson rates from disk
func (pr *PoissonRegressionModel) loadRates() error {
	filePath := filepath.Join(pr.dataDir, "poisson_rates.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("📊 No existing Poisson rates found, starting fresh")
		return nil
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading Poisson rates: %v", err)
	}

	var data PoissonModelData
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return fmt.Errorf("error unmarshaling Poisson rates: %v", err)
	}

	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	pr.teamOffensiveRates = data.TeamOffensiveRates
	pr.teamDefensiveRates = data.TeamDefensiveRates
	pr.rateHistory = data.RateHistory
	pr.confidenceTracking = data.ConfidenceTracking
	pr.lastUpdated = data.LastUpdated

	log.Printf("📊 Loaded Poisson rates: %d teams tracked (last updated: %s)",
		len(pr.teamOffensiveRates), data.LastUpdated.Format("2006-01-02 15:04:05"))

	return nil
}
