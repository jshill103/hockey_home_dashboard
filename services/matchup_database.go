package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// MatchupDatabaseService manages head-to-head matchup history
type MatchupDatabaseService struct {
	index     *models.MatchupIndex
	rivalries []models.RivalryDefinition
	divisions models.DivisionInfo
	dataDir   string
	mutex     sync.RWMutex
}

// NewMatchupDatabaseService creates a new matchup database service
func NewMatchupDatabaseService() *MatchupDatabaseService {
	dataDir := "data/matchups"
	os.MkdirAll(dataDir, 0755)

	mds := &MatchupDatabaseService{
		index: &models.MatchupIndex{
			Matchups: make(map[string]*models.MatchupHistory),
		},
		dataDir:   dataDir,
		rivalries: initializeRivalries(),
		divisions: initializeDivisions(),
	}

	// Load existing data
	mds.loadMatchupIndex()

	return mds
}

// GetMatchupHistory retrieves head-to-head history between two teams
func (mds *MatchupDatabaseService) GetMatchupHistory(team1, team2 string) *models.MatchupHistory {
	mds.mutex.RLock()
	defer mds.mutex.RUnlock()

	key := makeMatchupKey(team1, team2)

	if history, exists := mds.index.Matchups[key]; exists {
		return history
	}

	// Return empty history if none exists
	return &models.MatchupHistory{
		TeamA:       team1,
		TeamB:       team2,
		LastUpdated: time.Now(),
	}
}

// CalculateMatchupAdvantage calculates the advantage based on matchup history
func (mds *MatchupDatabaseService) CalculateMatchupAdvantage(homeTeam, awayTeam string) *models.MatchupAdvantage {
	mds.mutex.RLock()
	defer mds.mutex.RUnlock()

	history := mds.GetMatchupHistory(homeTeam, awayTeam)

	advantage := &models.MatchupAdvantage{
		HomeTeam:   homeTeam,
		AwayTeam:   awayTeam,
		KeyFactors: []string{},
	}

	// Calculate historical advantage
	if history.TotalGames > 0 {
		// Determine which team is A and which is B
		key := makeMatchupKey(homeTeam, awayTeam)
		parts := strings.Split(key, "-")
		teamAIsHome := parts[0] == homeTeam

		var homeWins, awayWins int
		if teamAIsHome {
			homeWins = history.TeamAWins
			awayWins = history.TeamBWins
		} else {
			homeWins = history.TeamBWins
			awayWins = history.TeamAWins
		}

		winDiff := homeWins - awayWins
		totalGames := history.TotalGames

		// Scale by sample size (more games = more confident)
		confidenceFactor := float64(totalGames) / (float64(totalGames) + 20.0)

		// Historical advantage: -0.15 to +0.15
		advantage.HistoricalAdvantage = (float64(winDiff) / float64(totalGames)) * 0.15 * confidenceFactor
		advantage.ConfidenceLevel = confidenceFactor

		if winDiff > 2 {
			advantage.KeyFactors = append(advantage.KeyFactors,
				fmt.Sprintf("%s leads series %d-%d", homeTeam, homeWins, awayWins))
		} else if winDiff < -2 {
			advantage.KeyFactors = append(advantage.KeyFactors,
				fmt.Sprintf("%s trails series %d-%d", homeTeam, homeWins, awayWins))
		}
	}

	// Calculate recent advantage (last 10 games)
	if history.Recent10Games >= 3 {
		key := makeMatchupKey(homeTeam, awayTeam)
		parts := strings.Split(key, "-")
		teamAIsHome := parts[0] == homeTeam

		var recentHomeWins, recentAwayWins int
		if teamAIsHome {
			recentHomeWins = history.Recent10TeamAWins
			recentAwayWins = history.Recent10TeamBWins
		} else {
			recentHomeWins = history.Recent10TeamBWins
			recentAwayWins = history.Recent10TeamAWins
		}

		recentDiff := recentHomeWins - recentAwayWins
		advantage.RecentAdvantage = (float64(recentDiff) / float64(history.Recent10Games)) * 0.10

		if recentHomeWins >= recentAwayWins+2 {
			advantage.KeyFactors = append(advantage.KeyFactors,
				fmt.Sprintf("%s won %d of last %d", homeTeam, recentHomeWins, history.Recent10Games))
		}
	}

	// Calculate venue advantage
	key := makeMatchupKey(homeTeam, awayTeam)
	parts := strings.Split(key, "-")

	var venueRecord models.VenueRecord
	if parts[0] == homeTeam {
		// Home team is Team A, check B's record at A
		venueRecord = history.TeamBAtTeamA
	} else {
		// Home team is Team B, check A's record at B
		venueRecord = history.TeamAAtTeamB
	}

	if venueRecord.Games >= 3 {
		// Home team advantage in this venue
		homeWinPct := float64(venueRecord.Wins) / float64(venueRecord.Games)
		advantage.VenueAdvantage = (homeWinPct - 0.5) * 0.05 // -0.025 to +0.025

		if homeWinPct > 0.65 {
			advantage.KeyFactors = append(advantage.KeyFactors,
				fmt.Sprintf("%s strong at home vs %s (%.0f%%)", homeTeam, awayTeam, homeWinPct*100))
		}
	}

	// Check rivalry
	if mds.isRivalry(homeTeam, awayTeam) {
		rivalry := mds.getRivalry(homeTeam, awayTeam)
		advantage.RivalryBoost = rivalry.Intensity * 0.05 // 0 to +0.05
		advantage.KeyFactors = append(advantage.KeyFactors,
			fmt.Sprintf("Rivalry game: %s", rivalry.RivalryName))
	}

	// Check division game
	if mds.isDivisionGame(homeTeam, awayTeam) {
		advantage.DivisionGameBoost = 0.02
		advantage.KeyFactors = append(advantage.KeyFactors, "Division matchup")
	}

	// Check playoff rematch
	if history.PlayoffHistory > 0 && history.LastPlayoffYear >= time.Now().Year()-2 {
		advantage.PlayoffRematchBoost = 0.03
		advantage.KeyFactors = append(advantage.KeyFactors,
			fmt.Sprintf("Playoff rematch (%d)", history.LastPlayoffYear))
	}

	// Calculate total advantage
	advantage.TotalAdvantage = advantage.HistoricalAdvantage +
		advantage.RecentAdvantage +
		advantage.VenueAdvantage +
		advantage.RivalryBoost +
		advantage.DivisionGameBoost +
		advantage.PlayoffRematchBoost

	// Cap at reasonable bounds
	if advantage.TotalAdvantage > 0.20 {
		advantage.TotalAdvantage = 0.20
	} else if advantage.TotalAdvantage < -0.20 {
		advantage.TotalAdvantage = -0.20
	}

	// Build reasoning
	if len(advantage.KeyFactors) > 0 {
		advantage.Reasoning = strings.Join(advantage.KeyFactors, "; ")
	} else {
		advantage.Reasoning = "No significant matchup history"
	}

	return advantage
}

// UpdateMatchupHistory updates the history after a completed game
func (mds *MatchupDatabaseService) UpdateMatchupHistory(game models.CompletedGame) error {
	mds.mutex.Lock()
	defer mds.mutex.Unlock()

	homeTeam := game.HomeTeam.TeamCode
	awayTeam := game.AwayTeam.TeamCode

	key := makeMatchupKey(homeTeam, awayTeam)

	// Get or create history
	history, exists := mds.index.Matchups[key]
	if !exists {
		history = &models.MatchupHistory{
			TeamA:       strings.Split(key, "-")[0],
			TeamB:       strings.Split(key, "-")[1],
			RecentGames: []models.MatchupGame{},
		}
		mds.index.Matchups[key] = history
	}

	// Determine winner
	var winner string
	if game.HomeTeam.Score > game.AwayTeam.Score {
		winner = homeTeam
	} else {
		winner = awayTeam
	}

	// Create game record
	// Assume OT if score difference is 1 (rough heuristic since we don't have period data)
	scoreDiff := game.HomeTeam.Score - game.AwayTeam.Score
	isOT := scoreDiff == 1 || scoreDiff == -1

	matchupGame := models.MatchupGame{
		GameID:       fmt.Sprintf("%d", game.GameID),
		Date:         game.GameDate,
		HomeTeam:     homeTeam,
		AwayTeam:     awayTeam,
		HomeScore:    game.HomeTeam.Score,
		AwayScore:    game.AwayTeam.Score,
		Winner:       winner,
		OvertimeGame: isOT,
		Season:       fmt.Sprintf("%d-%d", game.Season, game.Season+1),
	}

	// Add to recent games (keep last 10)
	history.RecentGames = append([]models.MatchupGame{matchupGame}, history.RecentGames...)
	if len(history.RecentGames) > 10 {
		history.RecentGames = history.RecentGames[:10]
	}

	// Update overall stats
	history.TotalGames++

	if winner == history.TeamA {
		history.TeamAWins++
	} else {
		history.TeamBWins++
	}

	if matchupGame.OvertimeGame {
		history.OTGames++
	}

	// Update recent 10 stats
	history.Recent10Games = len(history.RecentGames)
	history.Recent10TeamAWins = 0
	history.Recent10TeamBWins = 0

	for _, g := range history.RecentGames {
		if g.Winner == history.TeamA {
			history.Recent10TeamAWins++
		} else {
			history.Recent10TeamBWins++
		}
	}

	// Update scoring trends
	totalGoalsA := 0
	totalGoalsB := 0
	totalGoals := 0
	highScoring := 0
	lowScoring := 0

	for _, g := range history.RecentGames {
		total := g.HomeScore + g.AwayScore
		totalGoals += total

		if g.HomeTeam == history.TeamA {
			totalGoalsA += g.HomeScore
			totalGoalsB += g.AwayScore
		} else {
			totalGoalsA += g.AwayScore
			totalGoalsB += g.HomeScore
		}

		if total >= 6 {
			highScoring++
		} else if total < 4 {
			lowScoring++
		}
	}

	if history.Recent10Games > 0 {
		history.AvgGoalsTeamA = float64(totalGoalsA) / float64(history.Recent10Games)
		history.AvgGoalsTeamB = float64(totalGoalsB) / float64(history.Recent10Games)
		history.AvgTotalGoals = float64(totalGoals) / float64(history.Recent10Games)
	}

	history.HighScoringGames = highScoring
	history.LowScoringGames = lowScoring

	// Update home/away splits
	if winner == homeTeam {
		history.HomeTeamWins++
	} else {
		history.AwayTeamWins++
	}

	if history.TotalGames > 0 {
		history.HomeAdvantage = float64(history.HomeTeamWins) / float64(history.TotalGames)
	}

	// Update venue-specific records
	if homeTeam == history.TeamA {
		// Team B is visiting Team A
		history.TeamBAtTeamA.Games++
		if winner == awayTeam {
			history.TeamBAtTeamA.Wins++
		} else {
			history.TeamBAtTeamA.Losses++
		}
		if history.TeamBAtTeamA.Games > 0 {
			history.TeamBAtTeamA.WinPct = float64(history.TeamBAtTeamA.Wins) / float64(history.TeamBAtTeamA.Games)
		}
	} else {
		// Team A is visiting Team B
		history.TeamAAtTeamB.Games++
		if winner == awayTeam {
			history.TeamAAtTeamB.Wins++
		} else {
			history.TeamAAtTeamB.Losses++
		}
		if history.TeamAAtTeamB.Games > 0 {
			history.TeamAAtTeamB.WinPct = float64(history.TeamAAtTeamB.Wins) / float64(history.TeamAAtTeamB.Games)
		}
	}

	// Update recency
	history.LastGameDate = game.GameDate
	history.LastGameScore = fmt.Sprintf("%d-%d", game.HomeTeam.Score, game.AwayTeam.Score)
	history.LastGameWinner = winner

	// Update rivalry/division flags
	history.IsRivalry = mds.isRivalry(history.TeamA, history.TeamB)
	history.IsDivisionGame = mds.isDivisionGame(history.TeamA, history.TeamB)

	// Update recent trend
	if history.Recent10TeamAWins > history.Recent10TeamBWins+1 {
		history.RecentTrend = "A_winning"
	} else if history.Recent10TeamBWins > history.Recent10TeamAWins+1 {
		history.RecentTrend = "B_winning"
	} else {
		history.RecentTrend = "even"
	}

	history.LastUpdated = time.Now()

	// Save to disk
	mds.index.LastUpdated = time.Now()
	mds.index.TotalMatchups = len(mds.index.Matchups)
	mds.saveMatchupIndex()

	log.Printf("📊 Updated matchup history: %s vs %s (Total: %d games)",
		history.TeamA, history.TeamB, history.TotalGames)

	return nil
}

// Helper functions

func makeMatchupKey(team1, team2 string) string {
	// Always alphabetical order for consistency
	teams := []string{team1, team2}
	sort.Strings(teams)
	return fmt.Sprintf("%s-%s", teams[0], teams[1])
}

func (mds *MatchupDatabaseService) isRivalry(team1, team2 string) bool {
	for _, rivalry := range mds.rivalries {
		if (rivalry.Team1 == team1 && rivalry.Team2 == team2) ||
			(rivalry.Team1 == team2 && rivalry.Team2 == team1) {
			return true
		}
	}
	return false
}

func (mds *MatchupDatabaseService) getRivalry(team1, team2 string) models.RivalryDefinition {
	for _, rivalry := range mds.rivalries {
		if (rivalry.Team1 == team1 && rivalry.Team2 == team2) ||
			(rivalry.Team1 == team2 && rivalry.Team2 == team1) {
			return rivalry
		}
	}
	return models.RivalryDefinition{}
}

func (mds *MatchupDatabaseService) isDivisionGame(team1, team2 string) bool {
	// Check if both teams are in the same division
	divisions := [][]string{
		mds.divisions.Atlantic,
		mds.divisions.Metropolitan,
		mds.divisions.Central,
		mds.divisions.Pacific,
	}

	for _, division := range divisions {
		found1 := false
		found2 := false
		for _, team := range division {
			if team == team1 {
				found1 = true
			}
			if team == team2 {
				found2 = true
			}
		}
		if found1 && found2 {
			return true
		}
	}
	return false
}

// Persistence

func (mds *MatchupDatabaseService) saveMatchupIndex() error {
	filePath := filepath.Join(mds.dataDir, "matchup_index.json")

	data, err := json.MarshalIndent(mds.index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal matchup index: %v", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write matchup index: %v", err)
	}

	return nil
}

func (mds *MatchupDatabaseService) loadMatchupIndex() error {
	filePath := filepath.Join(mds.dataDir, "matchup_index.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("📊 No existing matchup database found, starting fresh")
			return nil
		}
		return fmt.Errorf("failed to read matchup index: %v", err)
	}

	err = json.Unmarshal(data, mds.index)
	if err != nil {
		return fmt.Errorf("failed to unmarshal matchup index: %v", err)
	}

	log.Printf("📊 Loaded matchup database: %d matchups", mds.index.TotalMatchups)
	return nil
}

// Initialize known NHL rivalries
func initializeRivalries() []models.RivalryDefinition {
	return []models.RivalryDefinition{
		// Historic Rivalries
		{Team1: "BOS", Team2: "MTL", RivalryName: "Bruins-Canadiens", Intensity: 1.0, Historical: true,
			Reasons: []string{"Oldest rivalry in NHL", "24 playoff series", "Original Six"}},
		{Team1: "NYR", Team2: "NYI", RivalryName: "Battle of New York", Intensity: 0.9, Historical: true,
			Reasons: []string{"Same market", "1980s dynasty battles"}},
		{Team1: "TOR", Team2: "MTL", RivalryName: "Leafs-Habs", Intensity: 0.95, Historical: true,
			Reasons: []string{"Original Six", "Canadian rivalry", "Historic playoffs"}},

		// Regional Rivalries
		{Team1: "EDM", Team2: "CGY", RivalryName: "Battle of Alberta", Intensity: 0.95, Historical: true,
			Reasons: []string{"Provincial rivalry", "1980s dynasty battles", "Passionate fanbases"}},
		{Team1: "PIT", Team2: "PHI", RivalryName: "Pennsylvania War", Intensity: 0.9, Historical: true,
			Reasons: []string{"State rivals", "Physical games", "Playoff history"}},
		{Team1: "VAN", Team2: "CGY", RivalryName: "Northwest Division", Intensity: 0.75,
			Reasons: []string{"Division rivals", "Proximity"}},

		// Modern Rivalries
		{Team1: "PIT", Team2: "WSH", RivalryName: "Crosby vs Ovechkin", Intensity: 0.85,
			Reasons: []string{"Crosby-Ovechkin battles", "Multiple playoff series"}},
		{Team1: "CHI", Team2: "STL", RivalryName: "I-55 Rivalry", Intensity: 0.75,
			Reasons: []string{"Proximity", "Division rivals"}},
		{Team1: "BOS", Team2: "TOR", RivalryName: "Bruins-Leafs", Intensity: 0.80,
			Reasons: []string{"Division rivals", "Playoff history"}},

		// Geographic Rivalries
		{Team1: "LAK", Team2: "ANA", RivalryName: "Freeway Face-off", Intensity: 0.75,
			Reasons: []string{"Southern California", "Same market"}},
		{Team1: "NYR", Team2: "NJD", RivalryName: "Hudson River Rivalry", Intensity: 0.70,
			Reasons: []string{"Proximity", "Division rivals"}},
		{Team1: "DET", Team2: "CHI", RivalryName: "Original Six Rivalry", Intensity: 0.75, Historical: true,
			Reasons: []string{"Original Six", "Historic battles"}},
	}
}

// Initialize NHL divisions
func initializeDivisions() models.DivisionInfo {
	return models.DivisionInfo{
		Atlantic:     []string{"BOS", "BUF", "DET", "FLA", "MTL", "OTT", "TBL", "TOR"},
		Metropolitan: []string{"CAR", "CBJ", "NJD", "NYI", "NYR", "PHI", "PIT", "WSH"},
		Central:      []string{"ARI", "CHI", "COL", "DAL", "MIN", "NSH", "STL", "UTA", "WPG"},
		Pacific:      []string{"ANA", "CGY", "EDM", "LAK", "SJS", "SEA", "VAN", "VGK"},
	}
}

// Global service instance
var (
	globalMatchupService *MatchupDatabaseService
	matchupMutex         sync.Mutex
)

// InitializeMatchupService initializes the global matchup database service
func InitializeMatchupService() error {
	matchupMutex.Lock()
	defer matchupMutex.Unlock()

	if globalMatchupService != nil {
		return fmt.Errorf("matchup service already initialized")
	}

	globalMatchupService = NewMatchupDatabaseService()
	log.Printf("📊 Matchup Database Service initialized")

	return nil
}

// GetMatchupService returns the global matchup database service
func GetMatchupService() *MatchupDatabaseService {
	matchupMutex.Lock()
	defer matchupMutex.Unlock()
	return globalMatchupService
}
