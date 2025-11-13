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
	timeWeightedInstance *TimeWeightedStatsService
	timeWeightedOnce     sync.Once
)

// TimeWeightedStats stores time-weighted statistics for a team
type TimeWeightedStats struct {
	TeamCode string    `json:"teamCode"`
	LastUpdate time.Time `json:"lastUpdate"`

	// Exponentially weighted stats (recent games matter more)
	WeightedWinPct         float64 `json:"weightedWinPct"`
	WeightedGoalsFor       float64 `json:"weightedGoalsFor"`
	WeightedGoalsAgainst   float64 `json:"weightedGoalsAgainst"`
	WeightedSavePct        float64 `json:"weightedSavePct"`
	WeightedPowerPlayPct   float64 `json:"weightedPowerPlayPct"`
	WeightedPenaltyKillPct float64 `json:"weightedPenaltyKillPct"`
	WeightedCorsiFor       float64 `json:"weightedCorsiFor"`
	WeightedXGFor          float64 `json:"weightedXGFor"`
	WeightedXGAgainst      float64 `json:"weightedXGAgainst"`

	// Sliding window averages
	Last5GamesWinPct     float64 `json:"last5GamesWinPct"`
	Last5GamesGoalsFor   float64 `json:"last5GamesGoalsFor"`
	Last5GamesGoalsAgainst float64 `json:"last5GamesGoalsAgainst"`
	Last5GamesXGFor      float64 `json:"last5GamesXGFor"`
	Last5GamesXGAgainst  float64 `json:"last5GamesXGAgainst"`

	Last10GamesWinPct      float64 `json:"last10GamesWinPct"`
	Last10GamesGoalsFor    float64 `json:"last10GamesGoalsFor"`
	Last10GamesGoalsAgainst float64 `json:"last10GamesGoalsAgainst"`
	Last10GamesXGFor       float64 `json:"last10GamesXGFor"`
	Last10GamesXGAgainst   float64 `json:"last10GamesXGAgainst"`

	Last20GamesWinPct      float64 `json:"last20GamesWinPct"`
	Last20GamesGoalsFor    float64 `json:"last20GamesGoalsFor"`
	Last20GamesGoalsAgainst float64 `json:"last20GamesGoalsAgainst"`
	Last20GamesXGFor       float64 `json:"last20GamesXGFor"`
	Last20GamesXGAgainst   float64 `json:"last20GamesXGAgainst"`

	// Streak detection
	IsOnHotStreak  bool    `json:"isOnHotStreak"`  // 4+ wins in last 6 games
	IsOnColdStreak bool    `json:"isOnColdStreak"` // 4+ losses in last 6 games
	StreakLength   int     `json:"streakLength"`   // Current win/loss streak
	StreakType     string  `json:"streakType"`     // "win", "loss", "none"
	HotStreakScore float64 `json:"hotStreakScore"` // 0-1, how "hot" is the team
	Momentum       float64 `json:"momentum"`       // Overall momentum score (-1 to +1)

	// Regression indicators
	IsOverperforming bool    `json:"isOverperforming"` // Win% > xG would suggest
	IsUnderperforming bool    `json:"isUnderperforming"` // Win% < xG would suggest
	RegressionRisk    float64 `json:"regressionRisk"`    // 0-1, likelihood of regression
	PDORegression     float64 `json:"pdoRegression"`     // Expected PDO regression

	// Recent performance trends
	Last5vs10Trend    float64 `json:"last5vs10Trend"`    // How last 5 compares to last 10
	Last10vs20Trend   float64 `json:"last10vs20Trend"`   // How last 10 compares to last 20
	PerformanceTrend  string  `json:"performanceTrend"`  // "improving", "declining", "stable"
	TrendStrength     float64 `json:"trendStrength"`     // 0-1, how strong is the trend
}

// GameResult stores a single game result for time-weighting
type GameResult struct {
	GameID      int       `json:"gameId"`
	Date        time.Time `json:"date"`
	IsWin       bool      `json:"isWin"`
	GoalsFor    int       `json:"goalsFor"`
	GoalsAgainst int       `json:"goalsAgainst"`
	IsHome      bool      `json:"isHome"`
	Opponent    string    `json:"opponent"`
	XGFor       float64   `json:"xgFor"`
	XGAgainst   float64   `json:"xgAgainst"`
	CorsiFor    float64   `json:"corsiFor"`
	SavePct     float64   `json:"savePct"`
	PowerPlayGoals int    `json:"powerPlayGoals"`
	PowerPlayOpps  int    `json:"powerPlayOpps"`
	PenaltyKillSuccesses int `json:"penaltyKillSuccesses"`
	PenaltyKillOpps      int `json:"penaltyKillOpps"`
}

// TimeWeightedStatsService manages time-weighted statistics
type TimeWeightedStatsService struct {
	dataDir    string
	stats      map[string]*TimeWeightedStats // team code -> stats
	gameHistory map[string][]*GameResult     // team code -> recent games
	mu         sync.RWMutex
	decayRate  float64 // Exponential decay rate (default 0.96)
}

// InitializeTimeWeightedStats creates the time-weighted stats service
func InitializeTimeWeightedStats() error {
	var initErr error
	timeWeightedOnce.Do(func() {
		dataDir := "data/time_weighted_stats"
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create time-weighted stats directory: %w", err)
			return
		}

		timeWeightedInstance = &TimeWeightedStatsService{
			dataDir:     dataDir,
			stats:       make(map[string]*TimeWeightedStats),
			gameHistory: make(map[string][]*GameResult),
			decayRate:   0.96, // Each game back is worth 96% of the previous
		}

		// Load existing stats
		if err := timeWeightedInstance.loadAllStats(); err != nil {
			fmt.Printf("⚠️ Warning: Could not load time-weighted stats: %v\n", err)
		}

		fmt.Println("✅ Time-Weighted Stats Service initialized")
	})

	return initErr
}

// GetTimeWeightedStatsService returns the singleton instance
func GetTimeWeightedStatsService() *TimeWeightedStatsService {
	return timeWeightedInstance
}

// AddGameResult adds a game result and recalculates time-weighted stats
func (tws *TimeWeightedStatsService) AddGameResult(teamCode string, result *GameResult) error {
	tws.mu.Lock()
	defer tws.mu.Unlock()

	// Initialize if needed
	if tws.gameHistory[teamCode] == nil {
		tws.gameHistory[teamCode] = make([]*GameResult, 0)
	}
	if tws.stats[teamCode] == nil {
		tws.stats[teamCode] = &TimeWeightedStats{
			TeamCode: teamCode,
		}
	}

	// Add game to history
	tws.gameHistory[teamCode] = append(tws.gameHistory[teamCode], result)

	// Keep only last 30 games in memory
	if len(tws.gameHistory[teamCode]) > 30 {
		tws.gameHistory[teamCode] = tws.gameHistory[teamCode][len(tws.gameHistory[teamCode])-30:]
	}

	// Recalculate stats
	tws.recalculateStats(teamCode)

	// Save stats
	return tws.saveStats(teamCode)
}

// recalculateStats recalculates all time-weighted statistics for a team
func (tws *TimeWeightedStatsService) recalculateStats(teamCode string) {
	history := tws.gameHistory[teamCode]
	if len(history) == 0 {
		return
	}

	stats := tws.stats[teamCode]
	stats.LastUpdate = time.Now()

	// Calculate exponentially weighted stats (most recent games weighted highest)
	tws.calculateExponentialWeights(stats, history)

	// Calculate sliding window averages
	tws.calculateSlidingWindows(stats, history)

	// Detect streaks
	tws.detectStreaks(stats, history)

	// Calculate momentum
	tws.calculateMomentum(stats, history)

	// Detect regression risk
	tws.detectRegressionRisk(stats, history)

	// Analyze trends
	tws.analyzeTrends(stats)
}

// calculateExponentialWeights calculates exponentially weighted averages
func (tws *TimeWeightedStatsService) calculateExponentialWeights(
	stats *TimeWeightedStats,
	history []*GameResult,
) {
	if len(history) == 0 {
		return
	}

	var weightedWins, weightedGF, weightedGA, weightedXGF, weightedXGA float64
	var weightedCorsi, weightedSavePct, weightedPP, weightedPK float64
	var totalWeight float64
	var ppGames, pkGames int

	// Start from most recent game (highest weight) and go backwards
	for i := len(history) - 1; i >= 0; i-- {
		game := history[i]
		gamesBack := len(history) - 1 - i
		weight := math.Pow(tws.decayRate, float64(gamesBack))

		if game.IsWin {
			weightedWins += weight
		}
		weightedGF += float64(game.GoalsFor) * weight
		weightedGA += float64(game.GoalsAgainst) * weight
		weightedXGF += game.XGFor * weight
		weightedXGA += game.XGAgainst * weight
		weightedCorsi += game.CorsiFor * weight
		weightedSavePct += game.SavePct * weight

		// Special teams (only count games with opportunities)
		if game.PowerPlayOpps > 0 {
			ppPct := float64(game.PowerPlayGoals) / float64(game.PowerPlayOpps)
			weightedPP += ppPct * weight
			ppGames++
		}
		if game.PenaltyKillOpps > 0 {
			pkPct := float64(game.PenaltyKillSuccesses) / float64(game.PenaltyKillOpps)
			weightedPK += pkPct * weight
			pkGames++
		}

		totalWeight += weight
	}

	// Normalize by total weight
	if totalWeight > 0 {
		stats.WeightedWinPct = weightedWins / totalWeight
		stats.WeightedGoalsFor = weightedGF / totalWeight
		stats.WeightedGoalsAgainst = weightedGA / totalWeight
		stats.WeightedXGFor = weightedXGF / totalWeight
		stats.WeightedXGAgainst = weightedXGA / totalWeight
		stats.WeightedCorsiFor = weightedCorsi / totalWeight
		stats.WeightedSavePct = weightedSavePct / totalWeight
		if ppGames > 0 {
			stats.WeightedPowerPlayPct = weightedPP / totalWeight
		}
		if pkGames > 0 {
			stats.WeightedPenaltyKillPct = weightedPK / totalWeight
		}
	}
}

// calculateSlidingWindows calculates sliding window averages
func (tws *TimeWeightedStatsService) calculateSlidingWindows(
	stats *TimeWeightedStats,
	history []*GameResult,
) {
	// Last 5 games
	tws.calculateWindow(stats, history, 5, &stats.Last5GamesWinPct, &stats.Last5GamesGoalsFor,
		&stats.Last5GamesGoalsAgainst, &stats.Last5GamesXGFor, &stats.Last5GamesXGAgainst)

	// Last 10 games
	tws.calculateWindow(stats, history, 10, &stats.Last10GamesWinPct, &stats.Last10GamesGoalsFor,
		&stats.Last10GamesGoalsAgainst, &stats.Last10GamesXGFor, &stats.Last10GamesXGAgainst)

	// Last 20 games
	tws.calculateWindow(stats, history, 20, &stats.Last20GamesWinPct, &stats.Last20GamesGoalsFor,
		&stats.Last20GamesGoalsAgainst, &stats.Last20GamesXGFor, &stats.Last20GamesXGAgainst)
}

// calculateWindow calculates stats for a specific window
func (tws *TimeWeightedStatsService) calculateWindow(
	stats *TimeWeightedStats,
	history []*GameResult,
	window int,
	winPct, gf, ga, xgf, xga *float64,
) {
	start := len(history) - window
	if start < 0 {
		start = 0
	}

	games := history[start:]
	if len(games) == 0 {
		return
	}

	var wins, goalsFor, goalsAgainst int
	var expectedGoalsFor, expectedGoalsAgainst float64

	for _, game := range games {
		if game.IsWin {
			wins++
		}
		goalsFor += game.GoalsFor
		goalsAgainst += game.GoalsAgainst
		expectedGoalsFor += game.XGFor
		expectedGoalsAgainst += game.XGAgainst
	}

	*winPct = float64(wins) / float64(len(games))
	*gf = float64(goalsFor) / float64(len(games))
	*ga = float64(goalsAgainst) / float64(len(games))
	*xgf = expectedGoalsFor / float64(len(games))
	*xga = expectedGoalsAgainst / float64(len(games))
}

// detectStreaks identifies hot and cold streaks
func (tws *TimeWeightedStatsService) detectStreaks(
	stats *TimeWeightedStats,
	history []*GameResult,
) {
	if len(history) == 0 {
		return
	}

	// Current streak
	streakLength := 0
	streakType := "none"
	mostRecent := history[len(history)-1]
	currentStreakWin := mostRecent.IsWin

	for i := len(history) - 1; i >= 0; i-- {
		if history[i].IsWin == currentStreakWin {
			streakLength++
		} else {
			break
		}
	}

	if currentStreakWin {
		streakType = "win"
	} else {
		streakType = "loss"
	}

	stats.StreakLength = streakLength
	stats.StreakType = streakType

	// Hot streak: 4+ wins in last 6 games
	if len(history) >= 6 {
		last6 := history[len(history)-6:]
		wins := 0
		for _, game := range last6 {
			if game.IsWin {
				wins++
			}
		}
		stats.IsOnHotStreak = (wins >= 4)
		stats.IsOnColdStreak = (wins <= 2)
	}

	// Hot streak score (0-1)
	// Based on recent win%, weighted by streak length
	recentWins := 0
	checkWindow := 10
	if len(history) < checkWindow {
		checkWindow = len(history)
	}
	for i := len(history) - checkWindow; i < len(history); i++ {
		if history[i].IsWin {
			recentWins++
		}
	}
	baseScore := float64(recentWins) / float64(checkWindow)
	
	// Boost by streak length
	streakBonus := math.Min(0.3, float64(streakLength)*0.05)
	if streakType == "win" {
		stats.HotStreakScore = math.Min(1.0, baseScore+streakBonus)
	} else {
		stats.HotStreakScore = math.Max(0.0, baseScore-streakBonus)
	}
}

// calculateMomentum calculates overall team momentum
func (tws *TimeWeightedStatsService) calculateMomentum(
	stats *TimeWeightedStats,
	history []*GameResult,
) {
	// Momentum considers:
	// - Recent win rate vs season average
	// - Goal differential trend
	// - Expected goals trend
	// - Current streak

	if len(history) < 10 {
		stats.Momentum = 0.0
		return
	}

	// Compare last 5 to last 10
	last5WinRate := stats.Last5GamesWinPct
	last10WinRate := stats.Last10GamesWinPct

	// Compare goal differentials
	last5GD := stats.Last5GamesGoalsFor - stats.Last5GamesGoalsAgainst
	last10GD := stats.Last10GamesGoalsFor - stats.Last10GamesGoalsAgainst

	// Compare xG differentials
	last5XGD := stats.Last5GamesXGFor - stats.Last5GamesXGAgainst
	last10XGD := stats.Last10GamesXGFor - stats.Last10GamesXGAgainst

	// Calculate momentum components
	winRateMomentum := (last5WinRate - last10WinRate) * 2.0   // -2 to +2
	gdMomentum := (last5GD - last10GD) / 5.0                   // Normalized
	xgMomentum := (last5XGD - last10XGD) / 5.0                 // Normalized

	// Streak bonus
	streakMomentum := 0.0
	if stats.StreakType == "win" && stats.StreakLength >= 3 {
		streakMomentum = math.Min(0.5, float64(stats.StreakLength)*0.1)
	} else if stats.StreakType == "loss" && stats.StreakLength >= 3 {
		streakMomentum = -math.Min(0.5, float64(stats.StreakLength)*0.1)
	}

	// Weighted average
	momentum := (winRateMomentum*0.4 + gdMomentum*0.2 + xgMomentum*0.2 + streakMomentum*0.2)
	
	// Clamp to [-1, 1]
	stats.Momentum = math.Max(-1.0, math.Min(1.0, momentum))
}

// detectRegressionRisk identifies teams likely to regress
func (tws *TimeWeightedStatsService) detectRegressionRisk(
	stats *TimeWeightedStats,
	history []*GameResult,
) {
	if len(history) < 10 {
		return
	}

	// Compare actual results to expected goals
	actualWinPct := stats.Last10GamesWinPct
	
	// Expected win% based on xG (Pythagorean expectation)
	xgFor := stats.Last10GamesXGFor
	xgAgainst := stats.Last10GamesXGAgainst
	exponent := 2.0 // NHL Pythagorean exponent
	expectedWinPct := math.Pow(xgFor, exponent) / (math.Pow(xgFor, exponent) + math.Pow(xgAgainst, exponent))

	// Calculate difference
	diff := actualWinPct - expectedWinPct

	stats.IsOverperforming = (diff > 0.15) // Winning 15% more than expected
	stats.IsUnderperforming = (diff < -0.15)
	stats.RegressionRisk = math.Abs(diff)

	// PDO regression (shooting% + save% - 1.0)
	// Teams with PDO > 1.02 or < 0.98 tend to regress
	// We'll use recent goals vs xG as a proxy
	actualGoals := stats.Last10GamesGoalsFor
	expectedGoals := stats.Last10GamesXGFor
	pdo := actualGoals / math.Max(0.1, expectedGoals)
	
	stats.PDORegression = math.Abs(pdo - 1.0)
}

// analyzeTrends identifies performance trends
func (tws *TimeWeightedStatsService) analyzeTrends(stats *TimeWeightedStats) {
	// Compare recent windows to identify trends
	stats.Last5vs10Trend = stats.Last5GamesWinPct - stats.Last10GamesWinPct
	stats.Last10vs20Trend = stats.Last10GamesWinPct - stats.Last20GamesWinPct

	// Determine trend direction
	avgTrend := (stats.Last5vs10Trend + stats.Last10vs20Trend) / 2.0
	stats.TrendStrength = math.Abs(avgTrend)

	if avgTrend > 0.10 {
		stats.PerformanceTrend = "improving"
	} else if avgTrend < -0.10 {
		stats.PerformanceTrend = "declining"
	} else {
		stats.PerformanceTrend = "stable"
	}
}

// GetStats returns time-weighted stats for a team
func (tws *TimeWeightedStatsService) GetStats(teamCode string) *TimeWeightedStats {
	tws.mu.RLock()
	defer tws.mu.RUnlock()
	return tws.stats[teamCode]
}

// EnrichPredictionFactors adds time-weighted stats to prediction factors
func (tws *TimeWeightedStatsService) EnrichPredictionFactors(
	teamCode string,
	factors *models.PredictionFactors,
) {
	tws.mu.RLock()
	defer tws.mu.RUnlock()

	stats := tws.stats[teamCode]
	if stats == nil {
		return
	}

	// Override/supplement existing factors with time-weighted versions
	factors.RecentForm = stats.WeightedWinPct * 100.0
	
	// TODO: Add HotStreakFactor to PredictionFactors struct
	// factors.HotStreakFactor = stats.HotStreakScore
	
	// Add new time-weighted features
	// These could be added to PredictionFactors struct for use in models
	// For now, we'll store them in a way that can be accessed
}

// File operations

func (tws *TimeWeightedStatsService) saveStats(teamCode string) error {
	stats := tws.stats[teamCode]
	if stats == nil {
		return nil
	}

	filename := filepath.Join(tws.dataDir, fmt.Sprintf("%s.json", teamCode))
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (tws *TimeWeightedStatsService) loadAllStats() error {
	files, err := filepath.Glob(filepath.Join(tws.dataDir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var stats TimeWeightedStats
		if err := json.Unmarshal(data, &stats); err != nil {
			continue
		}

		tws.stats[stats.TeamCode] = &stats
	}

	fmt.Printf("📊 Loaded time-weighted stats for %d teams\n", len(tws.stats))
	return nil
}

