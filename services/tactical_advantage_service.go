package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

var (
	tacticalAdvantageInstance *TacticalAdvantageService
	tacticalAdvantageOnce     sync.Once
)

// TacticalAdvantageService analyzes tactical matchups and advantages
type TacticalAdvantageService struct {
	mu             sync.RWMutex
	teamTendencies map[string]*models.TeamTendencies // teamCode -> tendencies
	dataDir        string
}

// InitializeTacticalAdvantage initializes the singleton TacticalAdvantageService
func InitializeTacticalAdvantage() error {
	var err error
	tacticalAdvantageOnce.Do(func() {
		dataDir := filepath.Join("data", "tactical")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create tactical directory: %w", err)
			return
		}

		tacticalAdvantageInstance = &TacticalAdvantageService{
			teamTendencies: make(map[string]*models.TeamTendencies),
			dataDir:        dataDir,
		}

		// Load existing tactical data
		if loadErr := tacticalAdvantageInstance.loadTacticalData(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing tactical data: %v\n", loadErr)
		}

		fmt.Println("✅ Tactical Advantage Service initialized")
	})
	return err
}

// GetTacticalAdvantageService returns the singleton instance
func GetTacticalAdvantageService() *TacticalAdvantageService {
	return tacticalAdvantageInstance
}

// AnalyzeTacticalAdvantages identifies tactical edges between two teams
func (tas *TacticalAdvantageService) AnalyzeTacticalAdvantages(homeTeam, awayTeam string, homeStats, awayStats models.TeamStats) []models.TacticalAdvantage {
	advantages := []models.TacticalAdvantage{}

	// 1. Special Teams Advantage
	stAdvantage := tas.analyzeSpecialTeamsAdvantage(homeTeam, awayTeam, homeStats, awayStats)
	advantages = append(advantages, stAdvantage)

	// 2. Possession Advantage
	possessionAdvantage := tas.analyzePossessionAdvantage(homeTeam, awayTeam, homeStats, awayStats)
	advantages = append(advantages, possessionAdvantage)

	// 3. Scoring Efficiency
	scoringAdvantage := tas.analyzeScoringEfficiency(homeTeam, awayTeam, homeStats, awayStats)
	advantages = append(advantages, scoringAdvantage)

	return advantages
}

func (tas *TacticalAdvantageService) analyzeSpecialTeamsAdvantage(homeTeam, awayTeam string, homeStats, awayStats models.TeamStats) models.TacticalAdvantage {
	// Home PP vs Away PK
	homePPAdv := homeStats.PowerPlayPct - (1.0 - awayStats.PenaltyKillPct)
	
	// Home PK vs Away PP
	homePKAdv := (1.0 - awayStats.PowerPlayPct) - homeStats.PenaltyKillPct
	
	// Net advantage
	netAdvantage := (homePPAdv + homePKAdv) / 2.0

	return models.TacticalAdvantage{
		Category:    "Special Teams",
		HomeRating:  (homeStats.PowerPlayPct + homeStats.PenaltyKillPct) / 2.0,
		AwayRating:  (awayStats.PowerPlayPct + awayStats.PenaltyKillPct) / 2.0,
		Advantage:   netAdvantage,
		Impact:      netAdvantage * 0.05, // Up to ±5% impact
		Confidence:  0.8,
		Explanation: fmt.Sprintf("Home PP: %.1f%%, Away PK: %.1f%% | Home PK: %.1f%%, Away PP: %.1f%%",
			homeStats.PowerPlayPct*100, awayStats.PenaltyKillPct*100,
			homeStats.PenaltyKillPct*100, awayStats.PowerPlayPct*100),
		KeyMetrics: map[string]float64{
			"homePP": homeStats.PowerPlayPct,
			"awayPK": awayStats.PenaltyKillPct,
			"homePK": homeStats.PenaltyKillPct,
			"awayPP": awayStats.PowerPlayPct,
		},
		LastUpdated: time.Now(),
	}
}

func (tas *TacticalAdvantageService) analyzePossessionAdvantage(homeTeam, awayTeam string, homeStats, awayStats models.TeamStats) models.TacticalAdvantage {
	// Faceoff wins as possession proxy
	advantage := homeStats.FaceoffWinPct - awayStats.FaceoffWinPct

	return models.TacticalAdvantage{
		Category:    "Possession",
		HomeRating:  homeStats.FaceoffWinPct,
		AwayRating:  awayStats.FaceoffWinPct,
		Advantage:   advantage,
		Impact:      advantage * 0.06, // Up to ±6% impact
		Confidence:  0.75,
		Explanation: fmt.Sprintf("Home FO%%: %.1f%%, Away FO%%: %.1f%%",
			homeStats.FaceoffWinPct*100, awayStats.FaceoffWinPct*100),
		KeyMetrics: map[string]float64{
			"homeFO": homeStats.FaceoffWinPct,
			"awayFO": awayStats.FaceoffWinPct,
		},
		LastUpdated: time.Now(),
	}
}

func (tas *TacticalAdvantageService) analyzeScoringEfficiency(homeTeam, awayTeam string, homeStats, awayStats models.TeamStats) models.TacticalAdvantage {
	// Shooting percentage and save percentage
	if homeStats.GamesPlayed == 0 || awayStats.GamesPlayed == 0 {
		return models.TacticalAdvantage{
			Category:   "Scoring Efficiency",
			Advantage:  0.0,
			Impact:     0.0,
			Confidence: 0.3,
		}
	}

	homeShPct := homeStats.GoalsFor / homeStats.ShotsFor
	awayShPct := awayStats.GoalsFor / awayStats.ShotsFor

	advantage := (homeShPct - awayShPct) * 10.0 // Scale up

	return models.TacticalAdvantage{
		Category:    "Scoring Efficiency",
		HomeRating:  homeShPct,
		AwayRating:  awayShPct,
		Advantage:   advantage,
		Impact:      advantage * 0.04, // Up to ±4% impact
		Confidence:  0.7,
		Explanation: fmt.Sprintf("Home Sh%%: %.1f%%, Away Sh%%: %.1f%%",
			homeShPct*100, awayShPct*100),
		KeyMetrics: map[string]float64{
			"homeShPct": homeShPct,
			"awayShPct": awayShPct,
		},
		LastUpdated: time.Now(),
	}
}

// CalculateTotalTacticalImpact sums tactical advantages
func (tas *TacticalAdvantageService) CalculateTotalTacticalImpact(advantages []models.TacticalAdvantage) float64 {
	var totalImpact float64
	var totalWeight float64

	for _, adv := range advantages {
		weight := adv.Confidence
		totalImpact += adv.Impact * weight
		totalWeight += weight
	}

	if totalWeight > 0 {
		return totalImpact / totalWeight
	}
	return 0.0
}

// GetSpecialTeamsAdvantage specifically gets ST advantage
func (tas *TacticalAdvantageService) GetSpecialTeamsAdvantage(homeTeam, awayTeam string, homeStats, awayStats models.TeamStats) float64 {
	stAdv := tas.analyzeSpecialTeamsAdvantage(homeTeam, awayTeam, homeStats, awayStats)
	return stAdv.Impact
}

// AnalyzeTeamTendencies analyzes a team's tactical tendencies
func (tas *TacticalAdvantageService) AnalyzeTeamTendencies(teamCode string, stats models.TeamStats) *models.TeamTendencies {
	tas.mu.Lock()
	defer tas.mu.Unlock()

	tendencies := &models.TeamTendencies{
		TeamCode:    teamCode,
		Season:      getCurrentSeason(),
		LastUpdated: time.Now(),
	}

	// Simplified tendency analysis based on available stats
	// Estimate period performance from overall stats
	avgGoals := stats.GoalsFor / float64(stats.GamesPlayed)
	avgGoalsAgainst := stats.GoalsAgainst / float64(stats.GamesPlayed)
	
	// Estimate period goal differential (evenly distributed)
	tendencies.Period1GoalDiff = (avgGoals - avgGoalsAgainst) / 3.0
	tendencies.Period2GoalDiff = (avgGoals - avgGoalsAgainst) / 3.0
	tendencies.Period3GoalDiff = (avgGoals - avgGoalsAgainst) / 3.0
	
	// Set default strongest/weakest (would need period data)
	tendencies.StrongestPeriod = 2
	tendencies.WeakestPeriod = 1

	// Determine pace based on goals/game
	if avgGoals > 3.5 {
		tendencies.PreferredPace = "Fast"
	} else if avgGoals < 2.5 {
		tendencies.PreferredPace = "Slow"
	} else {
		tendencies.PreferredPace = "Variable"
	}

	// Set default behaviors
	tendencies.LeadBehavior = "Balanced"
	tendencies.TrailingBehavior = "Measured"
	
	// Estimate rates from overall record
	winPct := float64(stats.Wins) / float64(stats.GamesPlayed)
	tendencies.LeadProtectionRate = 0.70 // Default
	tendencies.ComebackFrequency = 0.30  // Default
	tendencies.FirstGoalImportance = 0.55 + (winPct * 0.20) // Better teams benefit more from scoring first

	// Systems (simplified)
	tendencies.ForeCheckSystem = "Neutral"
	tendencies.DefensiveSystem = "Hybrid"
	tendencies.PowerPlayStyle = "Movement"
	tendencies.PenaltyKillStyle = "Box"

	tas.teamTendencies[teamCode] = tendencies
	tas.saveTacticalData()

	return tendencies
}

// GetTeamTendencies retrieves tendencies for a team
func (tas *TacticalAdvantageService) GetTeamTendencies(teamCode string) *models.TeamTendencies {
	tas.mu.RLock()
	defer tas.mu.RUnlock()

	if tendencies, exists := tas.teamTendencies[teamCode]; exists {
		return tendencies
	}

	// Return default tendencies
	return &models.TeamTendencies{
		TeamCode:              teamCode,
		StrongestPeriod:       2,
		WeakestPeriod:         1,
		PreferredPace:         "Variable",
		LeadBehavior:          "Balanced",
		TrailingBehavior:      "Measured",
		FirstGoalImportance:   0.60,
		LineMatchupImportance: 0.5,
	}
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (tas *TacticalAdvantageService) getTacticalDataPath() string {
	return filepath.Join(tas.dataDir, "team_tendencies.json")
}

func (tas *TacticalAdvantageService) saveTacticalData() error {
	data, err := json.MarshalIndent(tas.teamTendencies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tactical data: %w", err)
	}
	if err := os.WriteFile(tas.getTacticalDataPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write tactical data: %w", err)
	}
	return nil
}

func (tas *TacticalAdvantageService) loadTacticalData() error {
	filePath := tas.getTacticalDataPath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data yet
		}
		return fmt.Errorf("failed to read tactical data: %w", err)
	}

	if err := json.Unmarshal(data, &tas.teamTendencies); err != nil {
		return fmt.Errorf("failed to unmarshal tactical data: %w", err)
	}

	fmt.Printf("⚔️ Loaded tactical data for %d teams\n", len(tas.teamTendencies))
	return nil
}

