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

// UpdatePlayerImpact updates player impact after fetching NHL API data
func (pis *PlayerImpactService) UpdatePlayerImpact(teamCode string, season int) error {
	pis.mutex.Lock()
	defer pis.mutex.Unlock()

	// Fetch player stats from NHL API
	playerStats, err := pis.fetchPlayerStats(teamCode, season)
	if err != nil {
		return fmt.Errorf("failed to fetch player stats: %w", err)
	}

	// Create or update team impact
	impact := &models.PlayerImpact{
		TeamCode:    teamCode,
		Season:      season,
		LastUpdated: time.Now(),
		TopScorers:  []models.TopScorer{},
	}

	// Calculate top scorers (top 3)
	if len(playerStats) >= 3 {
		impact.TopScorers = playerStats[:3]

		// Calculate top 3 PPG
		for _, scorer := range impact.TopScorers {
			impact.Top3PPG += scorer.PointsPerGame
		}

		// Star power (0-1 scale based on top scorer)
		topPPG := impact.TopScorers[0].PointsPerGame
		impact.StarPower = pis.calculateStarPower(topPPG)

		// Top scorer form
		impact.TopScorerForm = pis.calculateTopScorerForm(impact.TopScorers)
	}

	// Calculate depth scoring (4th-10th scorers)
	if len(playerStats) >= 10 {
		var secondaryPoints float64
		var secondaryGames int

		for i := 3; i < 10 && i < len(playerStats); i++ {
			secondaryPoints += float64(playerStats[i].Points)
			secondaryGames += playerStats[i].GamesPlayed
		}

		if secondaryGames > 0 {
			impact.Secondary4to10 = secondaryPoints / float64(secondaryGames)
		}

		// Depth score (0-1 scale)
		impact.DepthScore = pis.calculateDepthScore(impact.Secondary4to10)

		// Balance rating (how evenly distributed is scoring)
		impact.BalanceRating = pis.calculateBalanceRating(playerStats)

		// Depth form
		impact.DepthForm = pis.calculateDepthForm(playerStats[3:10])
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

func (pis *PlayerImpactService) fetchPlayerStats(teamCode string, season int) ([]models.TopScorer, error) {
	// Fetch from NHL API stats leaders endpoint
	// For now, return empty list - will be populated by NHL API integration
	// This is a placeholder for the actual API call

	log.Printf("📊 Fetching player stats for %s (season %d)", teamCode, season)

	// TODO: Implement actual NHL API call to get player stats
	// Endpoint: https://api-web.nhle.com/v1/club-stats/{teamCode}/now

	return []models.TopScorer{}, nil
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
