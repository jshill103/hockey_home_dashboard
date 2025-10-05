package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// PlayerImpactService tracks simplified player impact for predictions
type PlayerImpactService struct {
	index   *models.TeamPlayerIndex
	dataDir string
	mutex   sync.RWMutex
}

// NewPlayerImpactService creates a new player impact service
func NewPlayerImpactService() *PlayerImpactService {
	dataDir := "data/player_impact"
	os.MkdirAll(dataDir, 0755)

	pis := &PlayerImpactService{
		index: &models.TeamPlayerIndex{
			Teams: make(map[string]*models.PlayerImpact),
			LeagueAverages: models.LeagueAverages{
				AvgTop3PPG:       2.5,   // Typical top 3 combined
				AvgSecondaryPPG:  0.5,   // Typical 4th-10th scorer
				AvgGoalieSavePct: 0.905, // League average save %
				AvgTopScorerPPG:  1.1,   // Top scorer average
				AvgDepthScore:    0.5,   // Mid-range depth
			},
		},
		dataDir: dataDir,
	}

	// Load existing data
	pis.loadPlayerIndex()

	return pis
}

// GetPlayerImpact retrieves player impact for a team
func (pis *PlayerImpactService) GetPlayerImpact(teamCode string) *models.PlayerImpact {
	pis.mutex.RLock()
	defer pis.mutex.RUnlock()

	if impact, exists := pis.index.Teams[teamCode]; exists {
		return impact
	}

	// Return empty impact if none exists
	return &models.PlayerImpact{
		TeamCode:    teamCode,
		LastUpdated: time.Now(),
		TopScorers:  []models.TopScorer{},
	}
}

// NeedsUpdate checks if player data for a team needs updating (older than 1 hour)
func (pis *PlayerImpactService) NeedsUpdate(teamCode string) bool {
	pis.mutex.RLock()
	defer pis.mutex.RUnlock()

	impact, exists := pis.index.Teams[teamCode]
	if !exists || impact == nil {
		return true // No data exists, needs update
	}

	// Check if data is older than 1 hour
	return time.Since(impact.LastUpdated) > time.Hour
}

// UpdatePlayerImpact updates player impact after fetching NHL API data
func (pis *PlayerImpactService) UpdatePlayerImpact(teamCode string, season int) error {
	pis.mutex.Lock()
	defer pis.mutex.Unlock()

	// Fetch player stats from NHL API
	playerStats, err := pis.fetchPlayerStats(teamCode, season)
	if err != nil {
		return fmt.Errorf("failed to fetch player stats: %w", err)
	}

	// Validate players against current roster
	rosterService := GetRosterValidationService()
	if rosterService != nil {
		roster, err := rosterService.FetchRoster(teamCode, season)
		if err != nil {
			log.Printf("⚠️ Could not fetch roster for validation: %v", err)
		} else {
			// Filter out players not on current roster
			validStats := []models.TopScorer{}
			for _, player := range playerStats {
				if roster.IsOnRoster(player.PlayerID) {
					validStats = append(validStats, player)
				} else {
					log.Printf("🏒 Filtered out %s (ID %d) - not on current %s roster",
						player.Name, player.PlayerID, teamCode)
				}
			}
			playerStats = validStats
			log.Printf("🏒 Roster validation complete: %d/%d players on current roster",
				len(validStats), len(playerStats))
		}
	}

	// Create or update team impact
	impact := &models.PlayerImpact{
		TeamCode:    teamCode,
		Season:      season,
		LastUpdated: time.Now(),
		TopScorers:  []models.TopScorer{},
	}

	// Calculate top 10 scorers and fetch their game logs for comprehensive depth analysis
	numTopScorers := 10
	if len(playerStats) < numTopScorers {
		numTopScorers = len(playerStats)
	}

	if numTopScorers >= 3 {
		impact.TopScorers = playerStats[:numTopScorers]

		log.Printf("📊 Fetching game logs for top %d scorers...", numTopScorers)

		// Fetch game logs for all top scorers to calculate recent form
		for i := range impact.TopScorers {
			gameLog, err := pis.fetchPlayerGameLog(impact.TopScorers[i].PlayerID, season, 10)
			if err != nil {
				log.Printf("⚠️ Could not fetch game log for player %d (%s): %v",
					impact.TopScorers[i].PlayerID, impact.TopScorers[i].Name, err)
				// Continue with default form values
				continue
			}

			// Calculate recent form (last 10 games)
			last10Goals, last10Assists, last10Points, last10PPG, formRating := pis.calculateRecentForm(gameLog)

			impact.TopScorers[i].Last10Goals = last10Goals
			impact.TopScorers[i].Last10Assists = last10Assists
			impact.TopScorers[i].Last10Points = last10Points
			impact.TopScorers[i].Last10PPG = last10PPG
			impact.TopScorers[i].FormRating = formRating

			// Detect hot/cold streaks
			isHot := pis.detectHotStreak(gameLog)
			isCold := pis.detectColdStreak(gameLog)

			// Log detailed info for top 3, summary for depth (4-10)
			if i < 3 {
				if isHot {
					log.Printf("🔥 #%d %s is HOT! %d points in last 10 games (%.2f PPG)",
						i+1, impact.TopScorers[i].Name, last10Points, last10PPG)
				} else if isCold {
					log.Printf("🧊 #%d %s is COLD: %d points in last 10 games (%.2f PPG)",
						i+1, impact.TopScorers[i].Name, last10Points, last10PPG)
				}
				log.Printf("📊 #%d %s recent form: %d G, %d A, %d PTS in last 10 games (%.2f PPG, form rating: %.1f/10)",
					i+1, impact.TopScorers[i].Name, last10Goals, last10Assists, last10Points, last10PPG, formRating)
			} else {
				// Summarize depth scorers (4-10)
				log.Printf("   #%d %s: %d PTS in last 10 (%.2f PPG, form: %.1f/10)",
					i+1, impact.TopScorers[i].Name, last10Points, last10PPG, formRating)
			}
		}

		// Calculate top 3 PPG
		for i := 0; i < 3 && i < len(impact.TopScorers); i++ {
			impact.Top3PPG += impact.TopScorers[i].PointsPerGame
		}

		// Star power (0-1 scale based on top scorer)
		topPPG := impact.TopScorers[0].PointsPerGame
		impact.StarPower = pis.calculateStarPower(topPPG)

		// Top scorer form (top 3)
		impact.TopScorerForm = pis.calculateTopScorerForm(impact.TopScorers[:min(3, len(impact.TopScorers))])
	}

	// Calculate depth scoring (4th-10th scorers) using actual form data
	if len(impact.TopScorers) >= 10 {
		var secondaryPoints float64
		var secondaryGames int
		var depthFormSum float64
		depthCount := 0

		for i := 3; i < 10 && i < len(impact.TopScorers); i++ {
			secondaryPoints += float64(impact.TopScorers[i].Points)
			secondaryGames += impact.TopScorers[i].GamesPlayed

			// Use actual form ratings from game logs
			if impact.TopScorers[i].FormRating > 0 {
				depthFormSum += impact.TopScorers[i].FormRating
				depthCount++
			}
		}

		if secondaryGames > 0 {
			impact.Secondary4to10 = secondaryPoints / float64(secondaryGames)
		}

		// Depth score (0-1 scale)
		impact.DepthScore = pis.calculateDepthScore(impact.Secondary4to10)

		// Balance rating (how evenly distributed is scoring across top 10)
		impact.BalanceRating = pis.calculateBalanceRating(impact.TopScorers)

		// Depth form (average form of 4th-10th scorers)
		if depthCount > 0 {
			impact.DepthForm = depthFormSum / float64(depthCount)
			log.Printf("📊 Depth scorers (4-10) average form: %.1f/10 (%d players)", impact.DepthForm, depthCount)
		} else {
			// Fallback if no form data
			impact.DepthForm = pis.calculateDepthForm(impact.TopScorers[3:min(10, len(impact.TopScorers))])
		}
	}

	// Calculate differentials vs league average
	impact.StarPowerDiff = impact.StarPower - 0.5 // 0.5 is league average
	impact.DepthDiff = impact.DepthScore - pis.index.LeagueAverages.AvgDepthScore

	// Store in index
	pis.index.Teams[teamCode] = impact

	// Save to disk
	pis.savePlayerIndex()

	log.Printf("⭐ Updated player impact for %s: StarPower=%.2f, Depth=%.2f",
		teamCode, impact.StarPower, impact.DepthScore)

	return nil
}

// ComparePlayerImpact compares two teams' player impact
func (pis *PlayerImpactService) ComparePlayerImpact(homeTeam, awayTeam string) *models.PlayerImpactComparison {
	pis.mutex.RLock()
	defer pis.mutex.RUnlock()

	homeImpact := pis.GetPlayerImpact(homeTeam)
	awayImpact := pis.GetPlayerImpact(awayTeam)

	comparison := &models.PlayerImpactComparison{
		HomeTeam:   homeTeam,
		AwayTeam:   awayTeam,
		KeyFactors: []string{},
	}

	// Star Power Advantage
	starPowerDiff := homeImpact.StarPower - awayImpact.StarPower
	comparison.StarPowerAdvantage = starPowerDiff * 0.10 // Scale to -0.10 to +0.10

	if len(homeImpact.TopScorers) > 0 && len(awayImpact.TopScorers) > 0 {
		comparison.TopScorerDifferential = homeImpact.TopScorers[0].PointsPerGame -
			awayImpact.TopScorers[0].PointsPerGame

		if comparison.TopScorerDifferential > 0.3 {
			comparison.ElitePlayerEdge = homeTeam
			comparison.KeyFactors = append(comparison.KeyFactors,
				fmt.Sprintf("%s has elite scoring advantage (%.2f PPG vs %.2f PPG)",
					homeTeam, homeImpact.TopScorers[0].PointsPerGame, awayImpact.TopScorers[0].PointsPerGame))
		} else if comparison.TopScorerDifferential < -0.3 {
			comparison.ElitePlayerEdge = awayTeam
			comparison.KeyFactors = append(comparison.KeyFactors,
				fmt.Sprintf("%s has elite scoring advantage", awayTeam))
		}
	}

	// Depth Advantage
	depthDiff := homeImpact.DepthScore - awayImpact.DepthScore
	comparison.DepthAdvantage = depthDiff * 0.05 // Scale to -0.05 to +0.05

	if math.Abs(depthDiff) > 0.15 {
		if depthDiff > 0 {
			comparison.KeyFactors = append(comparison.KeyFactors,
				fmt.Sprintf("%s has superior depth scoring", homeTeam))
		} else {
			comparison.KeyFactors = append(comparison.KeyFactors,
				fmt.Sprintf("%s has superior depth scoring", awayTeam))
		}
	}

	// Form Comparison
	comparison.TopScorerFormDiff = homeImpact.TopScorerForm - awayImpact.TopScorerForm
	comparison.DepthFormDiff = homeImpact.DepthForm - awayImpact.DepthForm

	formImpact := (comparison.TopScorerFormDiff + comparison.DepthFormDiff) * 0.01 // Small impact

	// Total Impact
	comparison.TotalPlayerImpact = comparison.StarPowerAdvantage +
		comparison.DepthAdvantage +
		formImpact

	// Cap at reasonable bounds
	if comparison.TotalPlayerImpact > 0.15 {
		comparison.TotalPlayerImpact = 0.15
	} else if comparison.TotalPlayerImpact < -0.15 {
		comparison.TotalPlayerImpact = -0.15
	}

	// Confidence based on data quality
	dataQuality := 0.0
	if len(homeImpact.TopScorers) >= 3 && len(awayImpact.TopScorers) >= 3 {
		dataQuality += 0.5
	}
	if homeImpact.DepthScore > 0 && awayImpact.DepthScore > 0 {
		dataQuality += 0.5
	}
	comparison.ConfidenceLevel = dataQuality

	// Build reasoning
	if len(comparison.KeyFactors) > 0 {
		comparison.Reasoning = fmt.Sprintf("Player advantage: %s", comparison.KeyFactors[0])
	} else {
		comparison.Reasoning = "Teams evenly matched in player talent"
	}

	return comparison
}

// Helper functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (pis *PlayerImpactService) fetchPlayerStats(teamCode string, season int) ([]models.TopScorer, error) {
	log.Printf("📊 Fetching player stats for %s (season %d) from NHL API...", teamCode, season)

	// Try current season first (using /now endpoint)
	url := fmt.Sprintf("https://api-web.nhle.com/v1/club-stats/%s/now", teamCode)

	body, err := MakeAPICall(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch club stats from NHL API: %w", err)
	}

	var clubStats models.ClubStatsResponse
	if err := json.Unmarshal(body, &clubStats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal club stats JSON: %w", err)
	}

	// Check if we have data (current season has started)
	if len(clubStats.Skaters) == 0 {
		log.Printf("⚠️ No current season data for %s, trying previous season...", teamCode)

		// Try previous season for seed data
		previousSeason := season - 10001 // e.g., 20252026 -> 20242025 (or use utils.GetPreviousSeason())
		prevSeasonStr := fmt.Sprintf("%d%d", previousSeason/10000, previousSeason%10000)
		url = fmt.Sprintf("https://api-web.nhle.com/v1/club-stats/%s/%s/2", teamCode, prevSeasonStr)

		body, err = MakeAPICall(url)
		if err == nil {
			if err := json.Unmarshal(body, &clubStats); err == nil && len(clubStats.Skaters) > 0 {
				log.Printf("✅ Using previous season (%d) data as seed for %s", previousSeason, teamCode)
			}
		}
	}

	// Convert skaters to TopScorer format
	topScorers := make([]models.TopScorer, 0, len(clubStats.Skaters))

	for _, skater := range clubStats.Skaters {
		// Calculate points per game
		ppg := 0.0
		if skater.GamesPlayed > 0 {
			ppg = float64(skater.Points) / float64(skater.GamesPlayed)
		}

		// Build player name
		firstName := skater.FirstName.Default
		lastName := skater.LastName.Default
		fullName := fmt.Sprintf("%s %s", firstName, lastName)

		topScorer := models.TopScorer{
			PlayerID:      skater.PlayerID,
			Name:          fullName,
			Position:      skater.Position,
			Number:        skater.SweaterNumber,
			Goals:         skater.Goals,
			Assists:       skater.Assists,
			Points:        skater.Points,
			GamesPlayed:   skater.GamesPlayed,
			PointsPerGame: ppg,
			PlusMinus:     skater.PlusMinus,
			IsPlaying:     skater.GamesPlayed > 0, // Has played this season

			// TODO Phase 2: Fetch game logs for recent form
			Last10Goals:   0,
			Last10Assists: 0,
			Last10Points:  0,
			Last10PPG:     0.0,
			FormRating:    5.0, // Default neutral form
		}

		topScorers = append(topScorers, topScorer)
	}

	// Sort by points descending to get top scorers first
	sort.Slice(topScorers, func(i, j int) bool {
		if topScorers[i].Points != topScorers[j].Points {
			return topScorers[i].Points > topScorers[j].Points
		}
		// Tie-breaker: PPG (for players with different games played)
		return topScorers[i].PointsPerGame > topScorers[j].PointsPerGame
	})

	if len(topScorers) > 0 {
		log.Printf("✅ Successfully fetched %d player stats for %s (Top scorer: %s with %d points in %d games, %.2f PPG)",
			len(topScorers), teamCode,
			topScorers[0].Name, topScorers[0].Points, topScorers[0].GamesPlayed, topScorers[0].PointsPerGame)
	} else {
		log.Printf("⚠️ No player stats found for %s", teamCode)
	}

	return topScorers, nil
}

// fetchPlayerGameLog fetches the last N games for a specific player
// Falls back to previous season if current season has no data
func (pis *PlayerImpactService) fetchPlayerGameLog(playerID int, season int, numGames int) ([]models.PlayerGameLogEntry, error) {
	log.Printf("📈 Fetching game log for player %d (season %d, last %d games)...", playerID, season, numGames)

	// Try current season first
	gameLog, err := pis.fetchPlayerGameLogForSeason(playerID, season, numGames)
	if err == nil && len(gameLog) > 0 {
		log.Printf("✅ Fetched %d games for player %d (current season)", len(gameLog), playerID)
		return gameLog, nil
	}

	// If no data for current season, try previous season
	previousSeason := season - 10001 // e.g., 20252026 -> 20242025
	log.Printf("⚠️ No current season data, trying previous season %d...", previousSeason)

	gameLog, err = pis.fetchPlayerGameLogForSeason(playerID, previousSeason, numGames)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game log from current or previous season: %w", err)
	}

	if len(gameLog) > 0 {
		log.Printf("✅ Fetched %d games for player %d (previous season - seed data)", len(gameLog), playerID)
	} else {
		log.Printf("⚠️ No game log data found for player %d", playerID)
	}

	return gameLog, nil
}

// fetchPlayerGameLogForSeason fetches game log for a specific season
func (pis *PlayerImpactService) fetchPlayerGameLogForSeason(playerID int, season int, numGames int) ([]models.PlayerGameLogEntry, error) {
	// NHL API endpoint for player game logs
	url := fmt.Sprintf("https://api-web.nhle.com/v1/player/%d/game-log/%d/2", playerID, season)

	body, err := MakeAPICall(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game log from NHL API: %w", err)
	}

	var gameLogResp models.PlayerGameLogResponse
	if err := json.Unmarshal(body, &gameLogResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game log JSON: %w", err)
	}

	// Take only the last N games (most recent)
	gameLog := gameLogResp.GameLog
	if len(gameLog) > numGames {
		gameLog = gameLog[len(gameLog)-numGames:] // Get last N games
	}

	return gameLog, nil
}

// calculateRecentForm calculates a player's recent form from their game log
func (pis *PlayerImpactService) calculateRecentForm(gameLog []models.PlayerGameLogEntry) (goals, assists, points int, ppg, formRating float64) {
	if len(gameLog) == 0 {
		return 0, 0, 0, 0.0, 5.0 // Default neutral form
	}

	totalGoals := 0
	totalAssists := 0
	totalPoints := 0

	for _, game := range gameLog {
		totalGoals += game.Goals
		totalAssists += game.Assists
		totalPoints += game.Points
	}

	gamesPlayed := len(gameLog)
	ppg = float64(totalPoints) / float64(gamesPlayed)

	// Form rating (0-10 scale)
	// Based on PPG in recent games vs expected
	// 1.5+ PPG = 10.0 (elite form)
	// 1.0 PPG = 8.0 (great form)
	// 0.7 PPG = 6.0 (good form)
	// 0.5 PPG = 5.0 (average form)
	// 0.3 PPG = 3.0 (cold)
	// 0.0 PPG = 1.0 (ice cold)

	if ppg >= 1.5 {
		formRating = 10.0
	} else if ppg >= 1.0 {
		formRating = 8.0 + (ppg-1.0)*4.0 // 8.0-10.0
	} else if ppg >= 0.7 {
		formRating = 6.0 + (ppg-0.7)*6.67 // 6.0-8.0
	} else if ppg >= 0.5 {
		formRating = 5.0 + (ppg-0.5)*5.0 // 5.0-6.0
	} else if ppg >= 0.3 {
		formRating = 3.0 + (ppg-0.3)*10.0 // 3.0-5.0
	} else if ppg > 0 {
		formRating = 1.0 + (ppg)*6.67 // 1.0-3.0
	} else {
		formRating = 1.0 // Ice cold
	}

	return totalGoals, totalAssists, totalPoints, ppg, formRating
}

// detectHotStreak detects if a player is on a hot streak (3+ points in last 3 games)
func (pis *PlayerImpactService) detectHotStreak(gameLog []models.PlayerGameLogEntry) bool {
	if len(gameLog) < 3 {
		return false
	}

	// Check last 3 games
	last3 := gameLog[len(gameLog)-3:]
	totalPoints := 0
	for _, game := range last3 {
		totalPoints += game.Points
	}

	return totalPoints >= 3 // 3+ points in last 3 games = hot
}

// detectColdStreak detects if a player is in a slump (0 points in last 5 games)
func (pis *PlayerImpactService) detectColdStreak(gameLog []models.PlayerGameLogEntry) bool {
	if len(gameLog) < 5 {
		return false
	}

	// Check last 5 games
	last5 := gameLog[len(gameLog)-5:]
	totalPoints := 0
	for _, game := range last5 {
		totalPoints += game.Points
	}

	return totalPoints == 0 // 0 points in last 5 games = cold
}

func (pis *PlayerImpactService) calculateStarPower(topPPG float64) float64 {
	// Star power scale:
	// 2.0+ PPG = 1.0 (elite, McDavid-level)
	// 1.5 PPG = 0.9 (superstar)
	// 1.2 PPG = 0.8 (star)
	// 1.0 PPG = 0.7 (very good)
	// 0.8 PPG = 0.6 (good)
	// 0.5 PPG = 0.5 (average)
	// 0.3 PPG = 0.3 (below average)

	if topPPG >= 2.0 {
		return 1.0
	} else if topPPG >= 1.5 {
		return 0.9
	} else if topPPG >= 1.2 {
		return 0.8
	} else if topPPG >= 1.0 {
		return 0.7
	} else if topPPG >= 0.8 {
		return 0.6
	} else {
		return math.Max(0.3, topPPG/1.5) // Scale below average
	}
}

func (pis *PlayerImpactService) calculateDepthScore(secondaryPPG float64) float64 {
	// Depth score scale:
	// 0.7+ PPG = 1.0 (elite depth)
	// 0.6 PPG = 0.8 (strong depth)
	// 0.5 PPG = 0.5 (average depth)
	// 0.4 PPG = 0.3 (weak depth)
	// 0.3 PPG = 0.2 (very weak depth)

	if secondaryPPG >= 0.7 {
		return 1.0
	} else if secondaryPPG >= 0.6 {
		return 0.8
	} else if secondaryPPG >= 0.5 {
		return 0.5
	} else {
		return math.Max(0.2, secondaryPPG/1.0)
	}
}

func (pis *PlayerImpactService) calculateBalanceRating(players []models.TopScorer) float64 {
	if len(players) < 5 {
		return 0.5
	}

	// Calculate how evenly distributed scoring is
	// More balanced = better (less reliance on top line)

	top3Total := 0.0
	next7Total := 0.0

	for i := 0; i < len(players) && i < 3; i++ {
		top3Total += players[i].PointsPerGame
	}

	for i := 3; i < len(players) && i < 10; i++ {
		next7Total += players[i].PointsPerGame
	}

	if top3Total == 0 {
		return 0.5
	}

	// Balance ratio: secondary / top3
	// 0.5+ = excellent balance
	// 0.3-0.5 = good balance
	// 0.2-0.3 = fair balance
	// <0.2 = top-heavy

	ratio := next7Total / top3Total

	if ratio >= 0.5 {
		return 1.0
	} else if ratio >= 0.3 {
		return 0.7
	} else if ratio >= 0.2 {
		return 0.5
	} else {
		return 0.3
	}
}

func (pis *PlayerImpactService) calculateTopScorerForm(topScorers []models.TopScorer) float64 {
	if len(topScorers) == 0 {
		return 5.0 // Neutral
	}

	totalForm := 0.0
	count := 0

	for _, scorer := range topScorers {
		if scorer.Last10PPG > 0 {
			// Compare recent to season average
			formRatio := scorer.Last10PPG / scorer.PointsPerGame

			// Convert to 0-10 scale
			// 1.5+ = 10 (on fire)
			// 1.2 = 8 (hot)
			// 1.0 = 5 (normal)
			// 0.8 = 3 (cold)
			// 0.5 = 0 (ice cold)

			form := 5.0 + (formRatio-1.0)*5.0
			if form > 10 {
				form = 10
			} else if form < 0 {
				form = 0
			}

			totalForm += form
			count++
		}
	}

	if count > 0 {
		return totalForm / float64(count)
	}

	return 5.0
}

func (pis *PlayerImpactService) calculateDepthForm(depthPlayers []models.TopScorer) float64 {
	if len(depthPlayers) == 0 {
		return 5.0
	}

	totalForm := 0.0
	count := 0

	for _, player := range depthPlayers {
		if player.Last10PPG > 0 && player.PointsPerGame > 0 {
			formRatio := player.Last10PPG / player.PointsPerGame
			form := 5.0 + (formRatio-1.0)*5.0

			if form > 10 {
				form = 10
			} else if form < 0 {
				form = 0
			}

			totalForm += form
			count++
		}
	}

	if count > 0 {
		return totalForm / float64(count)
	}

	return 5.0
}

// UpdateLeagueAverages recalculates league-wide averages
func (pis *PlayerImpactService) UpdateLeagueAverages() {
	pis.mutex.Lock()
	defer pis.mutex.Unlock()

	if len(pis.index.Teams) < 5 {
		return // Need at least 5 teams
	}

	var totalTop3PPG, totalSecondary, totalDepth float64
	count := 0

	for _, impact := range pis.index.Teams {
		if impact.Top3PPG > 0 {
			totalTop3PPG += impact.Top3PPG
			totalSecondary += impact.Secondary4to10
			totalDepth += impact.DepthScore
			count++
		}
	}

	if count > 0 {
		pis.index.LeagueAverages.AvgTop3PPG = totalTop3PPG / float64(count)
		pis.index.LeagueAverages.AvgSecondaryPPG = totalSecondary / float64(count)
		pis.index.LeagueAverages.AvgDepthScore = totalDepth / float64(count)

		log.Printf("📊 Updated league averages: Top3=%.2f, Secondary=%.2f",
			pis.index.LeagueAverages.AvgTop3PPG, pis.index.LeagueAverages.AvgSecondaryPPG)
	}
}

// Persistence

func (pis *PlayerImpactService) savePlayerIndex() error {
	filePath := filepath.Join(pis.dataDir, "player_impact_index.json")

	data, err := json.MarshalIndent(pis.index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal player index: %v", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write player index: %v", err)
	}

	return nil
}

func (pis *PlayerImpactService) loadPlayerIndex() error {
	filePath := filepath.Join(pis.dataDir, "player_impact_index.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("⭐ No existing player impact database found, starting fresh")
			return nil
		}
		return fmt.Errorf("failed to read player index: %v", err)
	}

	err = json.Unmarshal(data, pis.index)
	if err != nil {
		return fmt.Errorf("failed to unmarshal player index: %v", err)
	}

	log.Printf("⭐ Loaded player impact database: %d teams", len(pis.index.Teams))
	return nil
}

// Global service instance
var (
	globalPlayerImpactService *PlayerImpactService
	playerImpactMutex         sync.Mutex
)

// InitializePlayerImpactService initializes the global player impact service
func InitializePlayerImpactService() error {
	playerImpactMutex.Lock()
	defer playerImpactMutex.Unlock()

	if globalPlayerImpactService != nil {
		return fmt.Errorf("player impact service already initialized")
	}

	globalPlayerImpactService = NewPlayerImpactService()
	log.Printf("⭐ Player Impact Service initialized")

	return nil
}

// GetPlayerImpactService returns the global player impact service
func GetPlayerImpactService() *PlayerImpactService {
	playerImpactMutex.Lock()
	defer playerImpactMutex.Unlock()
	return globalPlayerImpactService
}

// Helper to sort players by points
type byPoints []models.TopScorer

func (a byPoints) Len() int           { return len(a) }
func (a byPoints) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPoints) Less(i, j int) bool { return a[i].Points > a[j].Points }

func sortPlayersByPoints(players []models.TopScorer) {
	sort.Sort(byPoints(players))
}
