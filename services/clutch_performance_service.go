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
	clutchInstance *ClutchPerformanceService
	clutchOnce     sync.Once
)

// ClutchPerformanceService analyzes team performance in clutch situations
type ClutchPerformanceService struct {
	mu                sync.RWMutex
	teamClutch        map[string]*models.ClutchPerformance // teamCode -> clutch performance
	clutchSituations  map[string][]models.ClutchSituation  // teamCode -> situations
	dataDir           string
}

// InitializeClutchPerformance initializes the singleton ClutchPerformanceService
func InitializeClutchPerformance() error {
	var err error
	clutchOnce.Do(func() {
		dataDir := filepath.Join("data", "clutch")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create clutch directory: %w", err)
			return
		}

		clutchInstance = &ClutchPerformanceService{
			teamClutch:       make(map[string]*models.ClutchPerformance),
			clutchSituations: make(map[string][]models.ClutchSituation),
			dataDir:          dataDir,
		}

		// Load existing clutch data
		if loadErr := clutchInstance.loadClutchData(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing clutch data: %v\n", loadErr)
		}

		fmt.Println("✅ Clutch Performance Service initialized")
	})
	return err
}

// GetClutchPerformanceService returns the singleton instance
func GetClutchPerformanceService() *ClutchPerformanceService {
	return clutchInstance
}

// GetClutchFactor returns the clutch factor for a team
func (cps *ClutchPerformanceService) GetClutchFactor(teamCode string) float64 {
	cps.mu.RLock()
	defer cps.mu.RUnlock()

	if clutch, exists := cps.teamClutch[teamCode]; exists {
		return clutch.ClutchFactor
	}

	return 0.0 // Neutral clutch factor
}

// AnalyzeClutchPerformance analyzes a team's clutch performance
func (cps *ClutchPerformanceService) AnalyzeClutchPerformance(teamCode string, games []models.GameResult) *models.ClutchPerformance {
	cps.mu.Lock()
	defer cps.mu.Unlock()

	clutch := &models.ClutchPerformance{
		TeamCode:    teamCode,
		Season:      getCurrentSeason(),
		LastUpdated: time.Now(),
	}

	if len(games) == 0 {
		clutch.ClutchFactor = 0.0
		clutch.Confidence = 0.3
		cps.teamClutch[teamCode] = clutch
		return clutch
	}

	// Analyze close games (1-goal games)
	closeGameWins := 0
	closeGameTotal := 0
	overtimeWins := 0
	overtimeTotal := 0
	thirdPeriodComebacks := 0
	lateGameCollapses := 0
	totalGoalDiff3rd := 0.0

	for _, game := range games {
		// Close games
		if math.Abs(float64(game.GoalDifferential)) == 1 {
			closeGameTotal++
			if game.Result == "W" {
				closeGameWins++
			}
		}

		// Overtime games
		if game.WentToOvertime {
			overtimeTotal++
			if game.Result == "W" {
				overtimeWins++
			}
		}

		// Third period comebacks (simplified - would need period-by-period data)
		if game.Result == "W" && game.GoalDifferential > 0 {
			// Assume some wins were comebacks
			if game.GoalDifferential <= 2 {
				thirdPeriodComebacks++
			}
		}

		// Blown leads (simplified)
		if game.Result == "L" && game.GoalDifferential < 0 {
			if game.GoalDifferential >= -1 {
				lateGameCollapses++
			}
		}

		// Track 3rd period performance (simplified estimate)
		totalGoalDiff3rd += float64(game.GoalDifferential) * 0.3 // Rough estimate
	}

	// Calculate close game record
	clutch.CloseGameRecord = models.Record{
		Wins:          closeGameWins,
		Losses:        closeGameTotal - closeGameWins,
		Total:         closeGameTotal,
		WinPercentage: calculateWinPercentage(closeGameWins, closeGameTotal),
	}

	// Calculate overtime record
	clutch.OvertimeRecord = models.Record{
		Wins:          overtimeWins,
		Losses:        overtimeTotal - overtimeWins,
		Total:         overtimeTotal,
		WinPercentage: calculateWinPercentage(overtimeWins, overtimeTotal),
	}

	clutch.ThirdPeriodComebacks = thirdPeriodComebacks
	clutch.LateGameCollapses = lateGameCollapses
	clutch.ThirdPeriodGoalDiff = totalGoalDiff3rd / float64(len(games))

	// Calculate comeback/protect rates
	if thirdPeriodComebacks+lateGameCollapses > 0 {
		clutch.ComebackWinRate = float64(thirdPeriodComebacks) / float64(thirdPeriodComebacks+lateGameCollapses)
	}
	clutch.ProtectLeadRate = 1.0 - (float64(lateGameCollapses) / float64(len(games)))

	// Calculate overall clutch factor
	clutch.ClutchFactor = cps.calculateClutchFactor(clutch)

	// Calculate pressure performance
	clutch.PressurePerformance = cps.calculatePressurePerformance(clutch)

	// Calculate confidence based on sample size
	clutch.Confidence = math.Min(1.0, float64(closeGameTotal+overtimeTotal)/15.0)

	cps.teamClutch[teamCode] = clutch
	cps.saveClutchData()

	return clutch
}

// calculateClutchFactor determines overall clutch rating
func (cps *ClutchPerformanceService) calculateClutchFactor(clutch *models.ClutchPerformance) float64 {
	var factor float64

	// Close game performance (40% weight)
	if clutch.CloseGameRecord.Total > 0 {
		closeGameImpact := (clutch.CloseGameRecord.WinPercentage - 0.50) * 0.40
		factor += closeGameImpact
	}

	// Overtime performance (25% weight)
	if clutch.OvertimeRecord.Total > 0 {
		otImpact := (clutch.OvertimeRecord.WinPercentage - 0.50) * 0.25
		factor += otImpact
	}

	// Comeback ability (20% weight)
	comebackImpact := (clutch.ComebackWinRate - 0.50) * 0.20
	factor += comebackImpact

	// Lead protection (15% weight)
	protectImpact := (clutch.ProtectLeadRate - 0.75) * 0.15
	factor += protectImpact

	// Clamp to -0.10 to +0.10
	return math.Max(-0.10, math.Min(0.10, factor))
}

// calculatePressurePerformance measures performance under pressure
func (cps *ClutchPerformanceService) calculatePressurePerformance(clutch *models.ClutchPerformance) float64 {
	// Combine all pressure metrics
	var pressure float64

	pressure += clutch.CloseGameRecord.WinPercentage * 0.4
	pressure += clutch.OvertimeRecord.WinPercentage * 0.3
	pressure += clutch.ComebackWinRate * 0.2
	pressure += clutch.ProtectLeadRate * 0.1

	return math.Max(0.0, math.Min(1.0, pressure))
}

// PredictClutchAdvantage predicts clutch advantage in a matchup
func (cps *ClutchPerformanceService) PredictClutchAdvantage(homeTeam, awayTeam string, gameImportance float64) float64 {
	homeClutch := cps.GetClutchFactor(homeTeam)
	awayClutch := cps.GetClutchFactor(awayTeam)

	// Base advantage
	advantage := homeClutch - awayClutch

	// Scale by game importance (clutch matters more in important games)
	scaledAdvantage := advantage * (0.5 + gameImportance*0.5)

	return scaledAdvantage
}

// CompareClutch compares clutch performance between two teams
func (cps *ClutchPerformanceService) CompareClutch(homeTeam, awayTeam string, gameImportance float64) *models.ClutchComparison {
	cps.mu.RLock()
	defer cps.mu.RUnlock()

	homeClutch := cps.getClutchPerformance(homeTeam)
	awayClutch := cps.getClutchPerformance(awayTeam)

	impact := cps.PredictClutchAdvantage(homeTeam, awayTeam, gameImportance)

	advantage := "Neutral"
	if impact > 0.03 {
		advantage = "Home"
	} else if impact < -0.03 {
		advantage = "Away"
	}

	analysis := fmt.Sprintf("Home clutch: %.2f (close games: %.0f%%), Away clutch: %.2f (close games: %.0f%%)",
		homeClutch.ClutchFactor, homeClutch.CloseGameRecord.WinPercentage*100,
		awayClutch.ClutchFactor, awayClutch.CloseGameRecord.WinPercentage*100)

	return &models.ClutchComparison{
		HomeTeam:          homeTeam,
		AwayTeam:          awayTeam,
		HomeClutch:        *homeClutch,
		AwayClutch:        *awayClutch,
		GameImportance:    gameImportance,
		ClutchAdvantage:   advantage,
		ImpactFactor:      impact,
		Confidence:        (homeClutch.Confidence + awayClutch.Confidence) / 2.0,
		Analysis:          analysis,
		RecommendedWeight: gameImportance * 0.8,
	}
}

func (cps *ClutchPerformanceService) getClutchPerformance(teamCode string) *models.ClutchPerformance {
	if clutch, exists := cps.teamClutch[teamCode]; exists {
		return clutch
	}

	// Return neutral clutch if not found
	return &models.ClutchPerformance{
		TeamCode:       teamCode,
		ClutchFactor:   0.0,
		Confidence:     0.3,
		CloseGameRecord: models.Record{WinPercentage: 0.50},
		OvertimeRecord:  models.Record{WinPercentage: 0.50},
	}
}

func calculateWinPercentage(wins, total int) float64 {
	if total == 0 {
		return 0.50
	}
	return float64(wins) / float64(total)
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (cps *ClutchPerformanceService) getClutchDataPath() string {
	return filepath.Join(cps.dataDir, "clutch_performance.json")
}

func (cps *ClutchPerformanceService) saveClutchData() error {
	data, err := json.MarshalIndent(cps.teamClutch, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal clutch data: %w", err)
	}
	if err := os.WriteFile(cps.getClutchDataPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write clutch data: %w", err)
	}
	return nil
}

func (cps *ClutchPerformanceService) loadClutchData() error {
	filePath := cps.getClutchDataPath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data yet
		}
		return fmt.Errorf("failed to read clutch data: %w", err)
	}

	if err := json.Unmarshal(data, &cps.teamClutch); err != nil {
		return fmt.Errorf("failed to unmarshal clutch data: %w", err)
	}

	fmt.Printf("🎯 Loaded clutch data for %d teams\n", len(cps.teamClutch))
	return nil
}

