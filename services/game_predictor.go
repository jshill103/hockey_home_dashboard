package services

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// GamePredictor interface for different prediction strategies
type GamePredictor interface {
	PredictWinProbability(homeTeam, awayTeam string, context *PredictionContext) (float64, error)
	Name() string
}

// PredictionContext provides additional context for playoff odds predictions
type PredictionContext struct {
	Date           time.Time
	HomeRecord     *models.TeamStanding
	AwayRecord     *models.TeamStanding
	IsPlayoffs     bool
	IsDivisionGame bool
	IsRivalryGame  bool
	RestDaysHome   int
	RestDaysAway   int
	
	// Phase 4: Referee Information
	GameID            int
	RefereeAssignment *models.RefereeGameAssignment
	RefereeImpact     *models.RefereeImpactAnalysis
}

// ================================================================================================
// SIMPLE PREDICTOR - Point Percentage + Home Ice (baseline)
// ================================================================================================

// SimplePredictor uses basic point percentage and home ice advantage
type SimplePredictor struct{}

func NewSimplePredictor() *SimplePredictor {
	return &SimplePredictor{}
}

func (sp *SimplePredictor) Name() string {
	return "Simple"
}

func (sp *SimplePredictor) PredictWinProbability(homeTeam, awayTeam string, context *PredictionContext) (float64, error) {
	// Check for nil context first, then nil records
	if context == nil || context.HomeRecord == nil || context.AwayRecord == nil {
		return 0.55, nil // Default with slight home advantage
	}

	homeWinPct := context.HomeRecord.PointPctg
	awayWinPct := context.AwayRecord.PointPctg
	homeAdvantage := 0.06 // ~6% home ice advantage

	if homeWinPct+awayWinPct > 0 {
		baseProbability := homeWinPct / (homeWinPct + awayWinPct)
		winProbability := baseProbability + homeAdvantage

		// Clamp to reasonable range
		winProbability = math.Max(0.30, math.Min(0.80, winProbability))
		return winProbability, nil
	}

	return 0.55, nil
}

// ================================================================================================
// ELO-BASED PREDICTOR - ELO Rating System
// ================================================================================================

// EloPredictor uses ELO rating system for predictions
type EloPredictor struct {
	eloRatings      map[string]float64
	predictionCache map[string]float64 // Cache for predictions
	mu              sync.RWMutex
	cacheMu         sync.RWMutex
	kFactor         float64
	cacheEnabled    bool
}

func NewEloPredictor() *EloPredictor {
	return &EloPredictor{
		eloRatings:      make(map[string]float64),
		predictionCache: make(map[string]float64),
		kFactor:         32.0, // Standard NHL K-factor
		cacheEnabled:    true,  // Safe for ELO (doesn't depend on full team records)
	}
}

func (ep *EloPredictor) Name() string {
	return "ELO"
}

func (ep *EloPredictor) PredictWinProbability(homeTeam, awayTeam string, context *PredictionContext) (float64, error) {
	// Check cache first (ELO predictions are based on ratings, not full team records)
	if ep.cacheEnabled && context != nil {
		cacheKey := fmt.Sprintf("%s_%s_%s", homeTeam, awayTeam, context.Date.Format("2006-01-02"))
		ep.cacheMu.RLock()
		if prob, exists := ep.predictionCache[cacheKey]; exists {
			ep.cacheMu.RUnlock()
			return prob, nil
		}
		ep.cacheMu.RUnlock()
	}
	
	ep.mu.RLock()
	homeElo := ep.eloRatings[homeTeam]
	awayElo := ep.eloRatings[awayTeam]
	ep.mu.RUnlock()

	// Handle nil context - return prediction with just home advantage
	if context == nil {
		if homeElo == 0 {
			homeElo = 1500
		}
		if awayElo == 0 {
			awayElo = 1500
		}
		homeElo += 35 // Home advantage
		expectedHome := 1.0 / (1.0 + math.Pow(10, (awayElo-homeElo)/400.0))
		return math.Max(0.25, math.Min(0.85, expectedHome)), nil
	}

	// If no ELO ratings exist, calculate from current record
	if homeElo == 0 && context.HomeRecord != nil {
		homeElo = ep.calculateEloFromRecord(context.HomeRecord)
	}
	if awayElo == 0 && context.AwayRecord != nil {
		awayElo = ep.calculateEloFromRecord(context.AwayRecord)
	}

	// ELO home advantage (roughly 30-40 points)
	homeEloWithAdvantage := homeElo + 35

	// Calculate expected probability using ELO formula
	expectedHome := 1.0 / (1.0 + math.Pow(10, (awayElo-homeEloWithAdvantage)/400.0))

	// Rest adjustment
	if context.RestDaysHome > context.RestDaysAway+1 {
		expectedHome += 0.03 // Well-rested bonus
	} else if context.RestDaysAway > context.RestDaysHome+1 {
		expectedHome -= 0.03 // Fatigue penalty
	}

	// Division game intensity
	if context.IsDivisionGame {
		// Division games tend to be closer
		expectedHome = 0.5 + (expectedHome-0.5)*0.9
	}

	// Clamp to reasonable range
	expectedHome = math.Max(0.25, math.Min(0.85, expectedHome))

	// Cache the result
	if ep.cacheEnabled && context != nil {
		cacheKey := fmt.Sprintf("%s_%s_%s", homeTeam, awayTeam, context.Date.Format("2006-01-02"))
		ep.cacheMu.Lock()
		ep.predictionCache[cacheKey] = expectedHome
		ep.cacheMu.Unlock()
	}

	return expectedHome, nil
}

func (ep *EloPredictor) calculateEloFromRecord(team *models.TeamStanding) float64 {
	// Start at 1500 baseline
	baseElo := 1500.0

	// Adjust based on point percentage
	// Point percentage of 0.5 = 1500 ELO
	// Point percentage of 0.7 = 1700 ELO
	// Point percentage of 0.3 = 1300 ELO
	winPctDiff := (team.PointPctg - 0.5) * 400

	return baseElo + winPctDiff
}

func (ep *EloPredictor) UpdateElo(winner, loser string, margin int) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	winnerElo := ep.eloRatings[winner]
	loserElo := ep.eloRatings[loser]

	if winnerElo == 0 {
		winnerElo = 1500
	}
	if loserElo == 0 {
		loserElo = 1500
	}

	// Calculate expected outcome
	expectedWinner := 1.0 / (1.0 + math.Pow(10, (loserElo-winnerElo)/400.0))

	// Update ratings
	ep.eloRatings[winner] = winnerElo + ep.kFactor*(1.0-expectedWinner)
	ep.eloRatings[loser] = loserElo + ep.kFactor*(0.0-(1.0-expectedWinner))
}

func (ep *EloPredictor) GetElo(teamCode string) float64 {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.eloRatings[teamCode]
}

func (ep *EloPredictor) SetElo(teamCode string, elo float64) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.eloRatings[teamCode] = elo
}

// ================================================================================================
// ML-BASED PREDICTOR - Full Ensemble Model
// ================================================================================================

// MLPredictor uses the full 156-feature ensemble prediction service
type MLPredictor struct {
	ensemble        *EnsemblePredictionService
	cacheEnabled    bool
	predictionCache map[string]float64
	cacheMu         sync.RWMutex
}

func NewMLPredictor(ensemble *EnsemblePredictionService) *MLPredictor {
	return &MLPredictor{
		ensemble:        ensemble,
		cacheEnabled:    false, // Disabled for simulations - cache doesn't account for evolving team records
		predictionCache: make(map[string]float64),
	}
}

// NewMLPredictorWithCache creates an ML predictor with caching enabled (for single predictions)
func NewMLPredictorWithCache(ensemble *EnsemblePredictionService) *MLPredictor {
	return &MLPredictor{
		ensemble:        ensemble,
		cacheEnabled:    true, // Safe for single predictions
		predictionCache: make(map[string]float64),
	}
}

func (mlp *MLPredictor) Name() string {
	return "ML-Ensemble"
}

func (mlp *MLPredictor) PredictWinProbability(homeTeam, awayTeam string, context *PredictionContext) (float64, error) {
	if mlp.ensemble == nil {
		return 0, fmt.Errorf("ensemble service not available")
	}
	
	// Validate context
	if context == nil || context.HomeRecord == nil || context.AwayRecord == nil {
		return 0, fmt.Errorf("invalid prediction context")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s_%s_%s", homeTeam, awayTeam, context.Date.Format("2006-01-02"))

	if mlp.cacheEnabled {
		mlp.cacheMu.RLock()
		if prob, exists := mlp.predictionCache[cacheKey]; exists {
			mlp.cacheMu.RUnlock()
			return prob, nil
		}
		mlp.cacheMu.RUnlock()
	}

	// Build prediction factors from context (Phase 3)
	homeFactors := mlp.buildFactorsFromContext(homeTeam, context.HomeRecord, context, true)
	awayFactors := mlp.buildFactorsFromContext(awayTeam, context.AwayRecord, context, false)

	// Use ensemble to predict
	result, err := mlp.ensemble.PredictGame(homeFactors, awayFactors)
	if err != nil {
		return 0, fmt.Errorf("ensemble prediction failed: %v", err)
	}

	// Cache the result
	if mlp.cacheEnabled {
		mlp.cacheMu.Lock()
		mlp.predictionCache[cacheKey] = result.WinProbability
		mlp.cacheMu.Unlock()
	}

	return result.WinProbability, nil
}

// buildFactorsFromContext builds PredictionFactors from TeamStanding and context
func (mlp *MLPredictor) buildFactorsFromContext(
	teamCode string,
	record *models.TeamStanding,
	context *PredictionContext,
	isHome bool,
) *models.PredictionFactors {
	factors := &models.PredictionFactors{
		TeamCode:      teamCode,
		WinPercentage: record.PointPctg,
		RestDays:      context.RestDaysHome,
	}
	
	if !isHome {
		factors.RestDays = context.RestDaysAway
	}
	
	// Home ice advantage
	if isHome {
		factors.HomeAdvantage = 0.06 // Standard 6% home advantage
	}
	
	// Goals per game (approximate from total stats)
	if record.GamesPlayed > 0 {
		factors.GoalsFor = float64(record.GoalFor) / float64(record.GamesPlayed)
		factors.GoalsAgainst = float64(record.GoalAgainst) / float64(record.GamesPlayed)
	}
	
	// Back-to-back penalty
	if factors.RestDays == 0 {
		factors.BackToBackPenalty = 0.08 // 8% penalty for back-to-back
	} else if factors.RestDays == 1 {
		factors.BackToBackPenalty = 0.03 // 3% penalty for 1 day rest
	}
	
	// Recent form (last 10 games win percentage)
	// We don't have detailed recent game data, so approximate from overall record
	// A more accurate implementation would track last 10 games
	factors.RecentForm = record.PointPctg
	
	// Power play and penalty kill (use league averages if not available)
	// In a full implementation, we'd fetch these from team stats
	factors.PowerPlayPct = 0.20   // League average ~20%
	factors.PenaltyKillPct = 0.80 // League average ~80%
	
	// Head-to-head: neutral if unknown
	factors.HeadToHead = 0.5
	
	// Initialize situational factors with defaults
	factors.TravelFatigue = models.TravelFatigue{
		MilesTraveled:    0,
		TimeZonesCrossed: 0,
		DaysOnRoad:       0,
		FatigueScore:     0,
	}
	
	factors.AltitudeAdjust = models.AltitudeAdjust{
		VenueAltitude:    0,
		TeamHomeAltitude: 0,
		AltitudeDiff:     0,
		AdjustmentFactor: 0,
	}
	
	factors.ScheduleStrength = models.ScheduleStrength{
		GamesInLast7Days: 0,
		OpponentStrength: 0.5,
		RestAdvantage:    0,
		ScheduleDensity:  1.0,
	}
	
	factors.InjuryImpact = models.InjuryImpact{
		KeyPlayersOut:    0,
		InjuryScore:      0,
		HealthPercentage: 100,
	}
	
	factors.MomentumFactors = models.MomentumFactors{
		WinStreak:      0,
		LastGameMargin: 0,
		MomentumScore:  record.PointPctg,
	}
	
	// Advanced stats - use defaults/estimates
	factors.AdvancedStats = models.AdvancedAnalytics{
		CorsiForPct:        50.0,
		FenwickForPct:      50.0,
		XGForPerGame:       factors.GoalsFor,
		XGAgainstPerGame:   factors.GoalsAgainst,
		PossessionQuality:  0.5,
	}
	
	// Weather analysis - neutral (empty struct is fine, defaults will be used)
	factors.WeatherAnalysis = models.WeatherAnalysis{}
	
	// Market data - neutral
	factors.MarketData = models.MarketAdjustment{
		MarketConfidence: 0.5,
		VolumeConfidence: 0.5,
	}
	
	// Goalie factors - defaults
	factors.GoalieAdvantage = 0
	factors.GoalieSavePctDiff = 0
	factors.GoalieRecentFormDiff = 0
	factors.GoalieFatigueDiff = 0
	
	// Market consensus - derive from win percentage
	factors.MarketConsensus = record.PointPctg
	factors.MarketLineMovement = 0
	factors.SharpMoneyIndicator = 0.5
	factors.MarketConfidenceVal = 0.5
	
	return factors
}

func (mlp *MLPredictor) ClearCache() {
	mlp.cacheMu.Lock()
	defer mlp.cacheMu.Unlock()
	mlp.predictionCache = make(map[string]float64)
}

func (mlp *MLPredictor) GetCacheSize() int {
	mlp.cacheMu.RLock()
	defer mlp.cacheMu.RUnlock()
	return len(mlp.predictionCache)
}

// ================================================================================================
// HYBRID PREDICTOR - ML for Important Games, ELO for Others
// ================================================================================================

// HybridPredictor uses ML for important games and ELO for routine simulations
// This provides the best balance of accuracy and performance
type HybridPredictor struct {
	mlPredictor  *MLPredictor
	eloPredictor *EloPredictor
	mlThreshold  float64 // Importance threshold for using ML
	mlUsageCount int
	eloUsageCount int
	mu           sync.Mutex
}

func NewHybridPredictor(ml *MLPredictor, elo *EloPredictor) *HybridPredictor {
	return &HybridPredictor{
		mlPredictor:  ml,
		eloPredictor: elo,
		mlThreshold:  0.65, // Use ML for top 65% most important games
	}
}

func (hp *HybridPredictor) Name() string {
	return "Hybrid"
}

func (hp *HybridPredictor) PredictWinProbability(homeTeam, awayTeam string, context *PredictionContext) (float64, error) {
	importance := hp.calculateGameImportance(context)

	// Use ML for important games
	if importance >= hp.mlThreshold {
		prob, err := hp.mlPredictor.PredictWinProbability(homeTeam, awayTeam, context)
		if err == nil {
			hp.mu.Lock()
			hp.mlUsageCount++
			hp.mu.Unlock()
			return prob, nil
		}
		// Fall back to ELO if ML fails
	}

	// Use ELO for less important games or as fallback
	prob, err := hp.eloPredictor.PredictWinProbability(homeTeam, awayTeam, context)
	if err == nil {
		hp.mu.Lock()
		hp.eloUsageCount++
		hp.mu.Unlock()
	}
	return prob, err
}

func (hp *HybridPredictor) calculateGameImportance(context *PredictionContext) float64 {
	if context == nil || context.HomeRecord == nil || context.AwayRecord == nil {
		return 0.5 // Medium importance if no context
	}

	importance := 0.5 // Base importance

	// Division games are more important
	if context.IsDivisionGame {
		importance += 0.15
	}

	// Rivalry games
	if context.IsRivalryGame {
		importance += 0.10
	}

	// Playoff games are always maximum importance
	if context.IsPlayoffs {
		return 1.0
	}

	// Games between teams close in standings
	pointsDiff := math.Abs(float64(context.HomeRecord.Points - context.AwayRecord.Points))
	if pointsDiff <= 5 {
		importance += 0.15 // Very close in standings
	} else if pointsDiff <= 10 {
		importance += 0.05 // Somewhat close
	}

	// Games involving playoff bubble teams (80-100 points range)
	homeInBubble := context.HomeRecord.Points >= 80 && context.HomeRecord.Points <= 100
	awayInBubble := context.AwayRecord.Points >= 80 && context.AwayRecord.Points <= 100
	if homeInBubble || awayInBubble {
		importance += 0.10
	}

	return math.Min(1.0, importance)
}

func (hp *HybridPredictor) SetMLThreshold(threshold float64) {
	hp.mlThreshold = math.Max(0.0, math.Min(1.0, threshold))
}

func (hp *HybridPredictor) GetStats() (mlUsage, eloUsage int) {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	return hp.mlUsageCount, hp.eloUsageCount
}

func (hp *HybridPredictor) ResetStats() {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	hp.mlUsageCount = 0
	hp.eloUsageCount = 0
}

// ================================================================================================
// FACTORY FUNCTIONS
// ================================================================================================

// GetDefaultPredictor returns the recommended default predictor
func GetDefaultPredictor(ensemble *EnsemblePredictionService) GamePredictor {
	if ensemble != nil {
		ml := NewMLPredictor(ensemble)
		elo := NewEloPredictor()
		return NewHybridPredictor(ml, elo)
	}
	// Fallback to ELO if ensemble not available
	return NewEloPredictor()
}

