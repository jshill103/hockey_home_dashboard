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

// GoalieIntelligenceService tracks and analyzes goalie performance
type GoalieIntelligenceService struct {
	goalies     map[int]*models.GoalieInfo       // playerID -> GoalieInfo
	teamDepth   map[string]*models.GoalieDepth   // teamCode -> GoalieDepth
	matchups    map[string]*models.GoalieMatchup // "goalieID_teamCode" -> GoalieMatchup
	dataDir     string
	mutex       sync.RWMutex
	lastUpdated time.Time
}

// NewGoalieIntelligenceService creates a new goalie intelligence service
func NewGoalieIntelligenceService() *GoalieIntelligenceService {
	service := &GoalieIntelligenceService{
		goalies:   make(map[int]*models.GoalieInfo),
		teamDepth: make(map[string]*models.GoalieDepth),
		matchups:  make(map[string]*models.GoalieMatchup),
		dataDir:   "data/goalies",
	}

	// Create data directory
	os.MkdirAll(service.dataDir, 0755)

	// Load existing goalie data
	if err := service.loadGoalieData(); err != nil {
		log.Printf("⚠️ Could not load goalie data: %v (starting fresh)", err)
	} else {
		log.Printf("🥅 Loaded goalie data for %d goalies", len(service.goalies))
	}

	return service
}

// GetGoalieComparison analyzes goalie matchup for a game
func (gis *GoalieIntelligenceService) GetGoalieComparison(homeTeam, awayTeam string, gameDate time.Time) (*models.GoalieComparison, error) {
	gis.mutex.RLock()
	defer gis.mutex.RUnlock()

	homeDepth, homeOk := gis.teamDepth[homeTeam]
	awayDepth, awayOk := gis.teamDepth[awayTeam]

	if !homeOk || !awayOk {
		return nil, fmt.Errorf("goalie depth not available for teams")
	}

	// Determine starting goalies
	homeGoalie := gis.determineStarter(homeDepth, gameDate)
	awayGoalie := gis.determineStarter(awayDepth, gameDate)

	if homeGoalie == nil || awayGoalie == nil {
		return nil, fmt.Errorf("could not determine starting goalies")
	}

	// Calculate comparison
	comparison := &models.GoalieComparison{
		HomeGoalie: homeGoalie,
		AwayGoalie: awayGoalie,
	}

	// Compare season performance
	comparison.SeasonPerformance = gis.compareSeasonPerformance(homeGoalie, awayGoalie)

	// Compare recent form
	comparison.RecentForm = gis.compareRecentForm(homeGoalie, awayGoalie)

	// Compare workload/fatigue
	comparison.WorkloadFatigue = gis.compareWorkload(homeGoalie, awayGoalie)

	// Compare matchup history
	comparison.MatchupHistory = gis.compareMatchupHistory(homeGoalie, awayGoalie, homeTeam, awayTeam)

	// Compare home/away performance
	comparison.HomeAwayFactor = gis.compareHomeAway(homeGoalie, awayGoalie)

	// Calculate overall advantage
	comparison.AdvantageScore = gis.calculateOverallAdvantage(comparison)

	if comparison.AdvantageScore > 0.1 {
		comparison.OverallAdvantage = "home"
	} else if comparison.AdvantageScore < -0.1 {
		comparison.OverallAdvantage = "away"
	} else {
		comparison.OverallAdvantage = "even"
	}

	// Calculate impact on win probability
	comparison.WinProbabilityImpact = comparison.AdvantageScore * 0.15 // Goalie worth up to 15% swing
	comparison.Confidence = gis.calculateConfidence(homeGoalie, awayGoalie)

	return comparison, nil
}

// determineStarter determines which goalie is starting
func (gis *GoalieIntelligenceService) determineStarter(depth *models.GoalieDepth, gameDate time.Time) *models.GoalieInfo {
	// If we have confirmed starter for tonight, use it
	if depth.StartingTonight != nil && depth.StartingTonight.LastUpdated.After(gameDate.Add(-6*time.Hour)) {
		return depth.StartingTonight
	}

	// Otherwise, use the designated starter
	if depth.Starter != nil {
		return depth.Starter
	}

	return nil
}

// compareSeasonPerformance compares season-long performance
func (gis *GoalieIntelligenceService) compareSeasonPerformance(home, away *models.GoalieInfo) float64 {
	homeSavePct := home.SeasonSavePercentage
	awaySavePct := away.SeasonSavePercentage

	if homeSavePct == 0 || awaySavePct == 0 {
		return 0.0
	}

	// Difference in save percentage (normalize to -1 to 1 range)
	diff := (homeSavePct - awaySavePct) * 10.0 // 0.01 save % = 0.1 advantage
	return math.Max(-1.0, math.Min(1.0, diff))
}

// compareRecentForm compares last 5 starts
func (gis *GoalieIntelligenceService) compareRecentForm(home, away *models.GoalieInfo) float64 {
	homeRecent := home.RecentSavePct
	awayRecent := away.RecentSavePct

	if homeRecent == 0 || awayRecent == 0 {
		return 0.0
	}

	// Recent form matters more than season stats
	diff := (homeRecent - awayRecent) * 12.0 // Amplify recent performance
	return math.Max(-1.0, math.Min(1.0, diff))
}

// compareWorkload compares fatigue/workload
func (gis *GoalieIntelligenceService) compareWorkload(home, away *models.GoalieInfo) float64 {
	// Lower fatigue is better
	homeFatigue := home.WorkloadFatigueScore
	awayFatigue := away.WorkloadFatigueScore

	// Invert (low fatigue = advantage)
	diff := awayFatigue - homeFatigue
	return math.Max(-1.0, math.Min(1.0, diff))
}

// compareMatchupHistory compares goalie vs opponent history
func (gis *GoalieIntelligenceService) compareMatchupHistory(home, away *models.GoalieInfo, homeTeam, awayTeam string) float64 {
	homeVsAway := gis.getMatchupRecord(home.PlayerID, awayTeam)
	awayVsHome := gis.getMatchupRecord(away.PlayerID, homeTeam)

	if homeVsAway == nil || awayVsHome == nil {
		return 0.0
	}

	homeSavePct := homeVsAway.AverageSavePct
	awaySavePct := awayVsHome.AverageSavePct

	if homeSavePct == 0 || awaySavePct == 0 {
		return 0.0
	}

	diff := (homeSavePct - awaySavePct) * 8.0
	return math.Max(-1.0, math.Min(1.0, diff))
}

// compareHomeAway compares home/away splits
func (gis *GoalieIntelligenceService) compareHomeAway(home, away *models.GoalieInfo) float64 {
	homeSplit := home.HomeRecord.SavePct - home.AwayRecord.SavePct
	awaySplit := away.HomeRecord.SavePct - away.AwayRecord.SavePct

	// Home goalie gets home advantage, away goalie gets away disadvantage
	homeAdvantage := homeSplit * 5.0
	awayDisadvantage := awaySplit * 5.0

	return math.Max(-1.0, math.Min(1.0, homeAdvantage-awayDisadvantage))
}

// calculateOverallAdvantage calculates weighted advantage
func (gis *GoalieIntelligenceService) calculateOverallAdvantage(comp *models.GoalieComparison) float64 {
	// Weighted combination of factors
	weights := map[string]float64{
		"season":   0.20, // Season stats (baseline)
		"recent":   0.35, // Recent form (most important)
		"workload": 0.15, // Fatigue
		"matchup":  0.20, // Historical matchup
		"homeAway": 0.10, // Home/away splits
	}

	advantage := 0.0
	advantage += comp.SeasonPerformance * weights["season"]
	advantage += comp.RecentForm * weights["recent"]
	advantage += comp.WorkloadFatigue * weights["workload"]
	advantage += comp.MatchupHistory * weights["matchup"]
	advantage += comp.HomeAwayFactor * weights["homeAway"]

	return advantage
}

// calculateConfidence calculates confidence in goalie comparison
func (gis *GoalieIntelligenceService) calculateConfidence(home, away *models.GoalieInfo) float64 {
	confidence := 0.5 // Base confidence

	// More games = more confidence
	homeGames := float64(home.SeasonGamesPlayed)
	awayGames := float64(away.SeasonGamesPlayed)
	gamesConfidence := math.Min(1.0, (homeGames+awayGames)/60.0) * 0.3
	confidence += gamesConfidence

	// Recent data = more confidence
	if len(home.RecentStarts) >= 3 && len(away.RecentStarts) >= 3 {
		confidence += 0.2
	}

	return math.Min(1.0, confidence)
}

// getMatchupRecord retrieves goalie vs team history
func (gis *GoalieIntelligenceService) getMatchupRecord(goalieID int, opponentTeam string) *models.GoalieMatchup {
	key := fmt.Sprintf("%d_%s", goalieID, opponentTeam)
	return gis.matchups[key]
}

// UpdateGoalieAfterGame updates goalie stats after a game
func (gis *GoalieIntelligenceService) UpdateGoalieAfterGame(game *models.CompletedGame) error {
	gis.mutex.Lock()
	defer gis.mutex.Unlock()

	// This would be called by GameResultsService after each game
	// to update goalie stats, recent starts, matchup history, etc.

	// For now, stub implementation
	log.Printf("🥅 Goalie stats updated for game %d", game.GameID)

	return nil
}

// GetGoalieImpact returns simplified goalie impact for predictions
func (gis *GoalieIntelligenceService) GetGoalieImpact(homeTeam, awayTeam string, gameDate time.Time) float64 {
	comparison, err := gis.GetGoalieComparison(homeTeam, awayTeam, gameDate)
	if err != nil {
		log.Printf("⚠️ Could not get goalie comparison: %v", err)
		return 0.0 // Neutral impact
	}

	// Return win probability impact (-0.15 to +0.15)
	return comparison.WinProbabilityImpact
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// loadGoalieData loads goalie data from disk
func (gis *GoalieIntelligenceService) loadGoalieData() error {
	filePath := filepath.Join(gis.dataDir, "goalies.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("goalies file not found")
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading goalies file: %w", err)
	}

	var data struct {
		Goalies     map[int]*models.GoalieInfo       `json:"goalies"`
		TeamDepth   map[string]*models.GoalieDepth   `json:"teamDepth"`
		Matchups    map[string]*models.GoalieMatchup `json:"matchups"`
		LastUpdated time.Time                        `json:"lastUpdated"`
	}

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return fmt.Errorf("error unmarshaling goalies data: %w", err)
	}

	gis.goalies = data.Goalies
	gis.teamDepth = data.TeamDepth
	gis.matchups = data.Matchups
	gis.lastUpdated = data.LastUpdated

	return nil
}

// saveGoalieData saves goalie data to disk
func (gis *GoalieIntelligenceService) saveGoalieData() error {
	filePath := filepath.Join(gis.dataDir, "goalies.json")

	data := struct {
		Goalies     map[int]*models.GoalieInfo       `json:"goalies"`
		TeamDepth   map[string]*models.GoalieDepth   `json:"teamDepth"`
		Matchups    map[string]*models.GoalieMatchup `json:"matchups"`
		LastUpdated time.Time                        `json:"lastUpdated"`
		Version     string                           `json:"version"`
	}{
		Goalies:     gis.goalies,
		TeamDepth:   gis.teamDepth,
		Matchups:    gis.matchups,
		LastUpdated: time.Now(),
		Version:     "1.0",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling goalies data: %w", err)
	}

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing goalies file: %w", err)
	}

	return nil
}

// ============================================================================
// GLOBAL SERVICE
// ============================================================================

var (
	globalGoalieService *GoalieIntelligenceService
	goalieMutex         sync.Mutex
)

// InitializeGoalieService initializes the global goalie intelligence service
func InitializeGoalieService() error {
	goalieMutex.Lock()
	defer goalieMutex.Unlock()

	if globalGoalieService != nil {
		return fmt.Errorf("goalie service already initialized")
	}

	globalGoalieService = NewGoalieIntelligenceService()
	log.Printf("🥅 Goalie Intelligence Service initialized")

	return nil
}

// GetGoalieService returns the global goalie intelligence service
func GetGoalieService() *GoalieIntelligenceService {
	goalieMutex.Lock()
	defer goalieMutex.Unlock()
	return globalGoalieService
}
