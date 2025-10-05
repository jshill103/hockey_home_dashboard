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

// RollingStatsService tracks and calculates rolling statistics for all teams
type RollingStatsService struct {
	teamStats map[string]*models.TeamRecentPerformance
	dataDir   string
	mutex     sync.RWMutex
}

// NewRollingStatsService creates a new rolling statistics service
func NewRollingStatsService() *RollingStatsService {
	service := &RollingStatsService{
		teamStats: make(map[string]*models.TeamRecentPerformance),
		dataDir:   "data/rolling_stats",
	}

	// Create data directory
	os.MkdirAll(service.dataDir, 0755)

	// Load existing stats
	if err := service.loadStats(); err != nil {
		log.Printf("⚠️ Could not load rolling stats: %v (starting fresh)", err)
	} else {
		log.Printf("📊 Loaded rolling statistics for %d teams", len(service.teamStats))
	}

	return service
}

// UpdateTeamStats updates a team's rolling statistics after a game
func (rss *RollingStatsService) UpdateTeamStats(game *models.CompletedGame) error {
	rss.mutex.Lock()
	defer rss.mutex.Unlock()

	// Update both teams
	if err := rss.updateSingleTeam(game, true); err != nil {
		return fmt.Errorf("failed to update home team: %w", err)
	}

	if err := rss.updateSingleTeam(game, false); err != nil {
		return fmt.Errorf("failed to update away team: %w", err)
	}

	// PHASE 6: Calculate advanced rolling statistics for both teams
	advancedCalc := NewAdvancedRollingStatsCalculator()
	homeTeamCode := game.HomeTeam.TeamCode
	awayTeamCode := game.AwayTeam.TeamCode

	if homeStats, exists := rss.teamStats[homeTeamCode]; exists {
		advancedCalc.CalculateAdvancedMetrics(homeStats, rss.teamStats)
	}

	if awayStats, exists := rss.teamStats[awayTeamCode]; exists {
		advancedCalc.CalculateAdvancedMetrics(awayStats, rss.teamStats)
	}

	// Save updated stats
	if err := rss.saveStats(); err != nil {
		log.Printf("⚠️ Failed to save rolling stats: %v", err)
	}

	return nil
}

// updateSingleTeam updates one team's statistics
func (rss *RollingStatsService) updateSingleTeam(game *models.CompletedGame, isHome bool) error {
	var team, opponent models.TeamGameResult
	var teamCode string

	if isHome {
		team = game.HomeTeam
		opponent = game.AwayTeam
		teamCode = game.HomeTeam.TeamCode
	} else {
		team = game.AwayTeam
		opponent = game.HomeTeam
		teamCode = game.AwayTeam.TeamCode
	}

	// Get or create team stats
	stats, exists := rss.teamStats[teamCode]
	if !exists {
		stats = &models.TeamRecentPerformance{
			TeamCode:    teamCode,
			Season:      game.Season,
			Last5Games:  []models.GameSummary{},
			Last10Games: []models.GameSummary{},
			HomeRecord:  models.Record{},
			AwayRecord:  models.Record{},
		}
		rss.teamStats[teamCode] = stats
	}

	// Create game summary
	result := "L"
	points := 0
	if team.Score > opponent.Score {
		result = "W"
		points = 2
	} else if game.WinType == "OT" || game.WinType == "SO" {
		result = "OTL"
		points = 1
	}

	gameSummary := models.GameSummary{
		GameID:         game.GameID,
		Date:           game.GameDate,
		Opponent:       opponent.TeamCode,
		IsHome:         isHome,
		GoalsFor:       team.Score,
		GoalsAgainst:   opponent.Score,
		Result:         result,
		Points:         points,
		Shots:          team.Shots,
		ShotsAgainst:   opponent.Shots,
		PowerPlayGoals: team.PowerPlayGoals,
		PowerPlayOpps:  team.PowerPlayOpps,
	}

	// Add to game history
	stats.Last10Games = append([]models.GameSummary{gameSummary}, stats.Last10Games...)
	if len(stats.Last10Games) > 10 {
		stats.Last10Games = stats.Last10Games[:10]
	}

	stats.Last5Games = append([]models.GameSummary{gameSummary}, stats.Last5Games...)
	if len(stats.Last5Games) > 5 {
		stats.Last5Games = stats.Last5Games[:5]
	}

	// Update records
	if isHome {
		if result == "W" {
			stats.HomeRecord.Wins++
		} else if result == "OTL" {
			stats.HomeRecord.OTLosses++
		} else {
			stats.HomeRecord.Losses++
		}
		stats.HomeRecord.Points += points
	} else {
		if result == "W" {
			stats.AwayRecord.Wins++
		} else if result == "OTL" {
			stats.AwayRecord.OTLosses++
		} else {
			stats.AwayRecord.Losses++
		}
		stats.AwayRecord.Points += points
	}

	// Recalculate all rolling statistics
	rss.calculateRollingStats(stats)

	stats.LastUpdated = time.Now()

	return nil
}

// calculateRollingStats recalculates all rolling statistics
func (rss *RollingStatsService) calculateRollingStats(stats *models.TeamRecentPerformance) {
	if len(stats.Last10Games) == 0 {
		return
	}

	// Calculate rolling averages (last 10 games)
	var totalGoalsFor, totalGoalsAgainst, totalPoints float64
	var totalShots, totalShotsAgainst int
	var totalPPGoals, totalPPOpps int
	var wins, otWins int

	for _, game := range stats.Last10Games {
		totalGoalsFor += float64(game.GoalsFor)
		totalGoalsAgainst += float64(game.GoalsAgainst)
		totalPoints += float64(game.Points)
		totalShots += game.Shots
		totalShotsAgainst += game.ShotsAgainst
		totalPPGoals += game.PowerPlayGoals
		totalPPOpps += game.PowerPlayOpps

		if game.Result == "W" {
			wins++
		} else if game.Result == "OTL" || game.Result == "SOL" {
			otWins++
		}
	}

	gameCount := float64(len(stats.Last10Games))
	stats.RecentGoalsFor = totalGoalsFor / gameCount
	stats.RecentGoalsAgainst = totalGoalsAgainst / gameCount
	stats.RecentWinPct = float64(wins) / gameCount
	stats.PointsPerGame = totalPoints / gameCount

	// Shooting percentage
	if totalShots > 0 {
		stats.RecentShotPct = (totalGoalsFor / float64(totalShots)) * 100
	}

	// Save percentage (goals against / shots against)
	if totalShotsAgainst > 0 {
		stats.RecentSavesPct = (1.0 - (totalGoalsAgainst / float64(totalShotsAgainst))) * 100
	}

	// Power Play percentage
	if totalPPOpps > 0 {
		stats.RecentPowerPlayPct = (float64(totalPPGoals) / float64(totalPPOpps)) * 100
	}

	// PDO (shooting % + save % - luck indicator, should trend to 100)
	stats.PdoScore = stats.RecentShotPct + stats.RecentSavesPct

	// Calculate momentum
	stats.Momentum = rss.calculateMomentum(stats.Last10Games)
	if stats.Momentum > 0.2 {
		stats.MomentumDirection = "improving"
	} else if stats.Momentum < -0.2 {
		stats.MomentumDirection = "declining"
	} else {
		stats.MomentumDirection = "stable"
	}

	// Calculate streak
	rss.calculateStreak(stats)

	// Calculate home/away splits
	rss.calculateHomeSplits(stats)

	// Calculate variance (consistency)
	rss.calculateVariance(stats)

	// Calculate Corsi approximation (using shots as proxy)
	if totalShots+totalShotsAgainst > 0 {
		stats.CorsiFor = float64(totalShots)
		stats.CorsiAgainst = float64(totalShotsAgainst)
		stats.CorsiPercentage = (float64(totalShots) / float64(totalShots+totalShotsAgainst)) * 100
	}
}

// calculateMomentum calculates team momentum based on recent results
func (rss *RollingStatsService) calculateMomentum(games []models.GameSummary) float64 {
	if len(games) == 0 {
		return 0.0
	}

	// Weight recent games more heavily
	var momentum float64
	for i, game := range games {
		// Weight: recent games count more (exponential decay)
		weight := math.Exp(-float64(i) * 0.2)

		var gameValue float64
		switch game.Result {
		case "W":
			gameValue = 1.0
		case "OTL", "SOL":
			gameValue = 0.3
		case "L":
			gameValue = -1.0
		}

		// Consider goal differential
		goalDiff := float64(game.GoalsFor - game.GoalsAgainst)
		goalDiffFactor := 1.0 + (goalDiff * 0.1) // Adjust momentum based on margin

		momentum += gameValue * weight * goalDiffFactor
	}

	// Normalize to -1 to +1 range
	maxMomentum := float64(len(games))
	return momentum / maxMomentum
}

// calculateStreak calculates current win/loss streak
func (rss *RollingStatsService) calculateStreak(stats *models.TeamRecentPerformance) {
	if len(stats.Last10Games) == 0 {
		stats.CurrentStreak = 0
		stats.StreakType = "none"
		return
	}

	firstGame := stats.Last10Games[0]
	isWinStreak := firstGame.Result == "W"
	isLossStreak := firstGame.Result == "L"

	stats.CurrentStreak = 1
	stats.StreakType = "none"

	if isWinStreak {
		stats.StreakType = "win"
		for i := 1; i < len(stats.Last10Games); i++ {
			if stats.Last10Games[i].Result == "W" {
				stats.CurrentStreak++
			} else {
				break
			}
		}
	} else if isLossStreak {
		stats.StreakType = "loss"
		stats.CurrentStreak = -1
		for i := 1; i < len(stats.Last10Games); i++ {
			if stats.Last10Games[i].Result == "L" {
				stats.CurrentStreak--
			} else {
				break
			}
		}
	}

	// Update longest streaks
	if stats.CurrentStreak > stats.LongestWinStreak {
		stats.LongestWinStreak = stats.CurrentStreak
	}
	if stats.CurrentStreak < stats.LongestLossStreak {
		stats.LongestLossStreak = stats.CurrentStreak
	}
}

// calculateHomeSplits calculates home/away performance splits
func (rss *RollingStatsService) calculateHomeSplits(stats *models.TeamRecentPerformance) {
	var homeWins, awayWins, homeGames, awayGames int

	for _, game := range stats.Last5Games {
		if game.IsHome {
			homeGames++
			if game.Result == "W" {
				homeWins++
			}
		} else {
			awayGames++
			if game.Result == "W" {
				awayWins++
			}
		}
	}

	if homeGames > 0 {
		stats.RecentHomeWinPct = float64(homeWins) / float64(homeGames)
	}

	if awayGames > 0 {
		stats.RecentAwayWinPct = float64(awayWins) / float64(awayGames)
	}
}

// calculateVariance calculates performance consistency
func (rss *RollingStatsService) calculateVariance(stats *models.TeamRecentPerformance) {
	if len(stats.Last10Games) < 2 {
		return
	}

	// Calculate variance in goals scored
	var goalsMean, defMean float64
	for _, game := range stats.Last10Games {
		goalsMean += float64(game.GoalsFor)
		defMean += float64(game.GoalsAgainst)
	}
	goalsMean /= float64(len(stats.Last10Games))
	defMean /= float64(len(stats.Last10Games))

	var goalsVar, defVar float64
	for _, game := range stats.Last10Games {
		goalsVar += math.Pow(float64(game.GoalsFor)-goalsMean, 2)
		defVar += math.Pow(float64(game.GoalsAgainst)-defMean, 2)
	}

	stats.GoalsVariance = goalsVar / float64(len(stats.Last10Games))
	stats.DefenseVariance = defVar / float64(len(stats.Last10Games))

	// Performance stability (lower variance = more stable, scale 0-1)
	// Lower variance is better, so invert and normalize
	totalVariance := stats.GoalsVariance + stats.DefenseVariance
	stats.PerformanceStability = 1.0 / (1.0 + totalVariance)
}

// GetTeamStats retrieves rolling statistics for a team
func (rss *RollingStatsService) GetTeamStats(teamCode string) (*models.TeamRecentPerformance, error) {
	rss.mutex.RLock()
	defer rss.mutex.RUnlock()

	stats, exists := rss.teamStats[teamCode]
	if !exists {
		return nil, fmt.Errorf("no statistics found for team %s", teamCode)
	}

	return stats, nil
}

// GetAllStats returns all team statistics
func (rss *RollingStatsService) GetAllStats() map[string]*models.TeamRecentPerformance {
	rss.mutex.RLock()
	defer rss.mutex.RUnlock()

	// Return a copy to avoid external modifications
	statsCopy := make(map[string]*models.TeamRecentPerformance)
	for k, v := range rss.teamStats {
		statsCopy[k] = v
	}

	return statsCopy
}

// saveStats saves rolling statistics to disk
func (rss *RollingStatsService) saveStats() error {
	filePath := filepath.Join(rss.dataDir, "rolling_stats.json")

	// Create a list of all team stats
	type StatsData struct {
		LastUpdated time.Time                                `json:"lastUpdated"`
		Teams       map[string]*models.TeamRecentPerformance `json:"teams"`
		Version     string                                   `json:"version"`
	}

	data := StatsData{
		LastUpdated: time.Now(),
		Teams:       rss.teamStats,
		Version:     "1.0",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling rolling stats: %w", err)
	}

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing rolling stats file: %w", err)
	}

	return nil
}

// loadStats loads rolling statistics from disk
func (rss *RollingStatsService) loadStats() error {
	filePath := filepath.Join(rss.dataDir, "rolling_stats.json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("rolling stats file not found")
	}

	// Read file
	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading rolling stats file: %w", err)
	}

	// Unmarshal data
	type StatsData struct {
		LastUpdated time.Time                                `json:"lastUpdated"`
		Teams       map[string]*models.TeamRecentPerformance `json:"teams"`
		Version     string                                   `json:"version"`
	}

	var data StatsData
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return fmt.Errorf("error unmarshaling rolling stats: %w", err)
	}

	rss.teamStats = data.Teams

	return nil
}

// Global rolling stats service instance
var (
	globalRollingStatsService *RollingStatsService
	rollingStatsMutex         sync.Mutex
)

// InitializeRollingStatsService initializes the global rolling stats service
func InitializeRollingStatsService() error {
	rollingStatsMutex.Lock()
	defer rollingStatsMutex.Unlock()

	if globalRollingStatsService != nil {
		return fmt.Errorf("rolling stats service already initialized")
	}

	globalRollingStatsService = NewRollingStatsService()

	return nil
}

// GetRollingStatsService returns the global rolling stats service
func GetRollingStatsService() *RollingStatsService {
	rollingStatsMutex.Lock()
	defer rollingStatsMutex.Unlock()
	return globalRollingStatsService
}
