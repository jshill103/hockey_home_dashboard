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
	streakDetectionInstance *StreakDetectionService
	streakDetectionOnce     sync.Once
)

// StreakDetectionService identifies and analyzes team streaks
type StreakDetectionService struct {
	mu            sync.RWMutex
	teamStreaks   map[string]*models.StreakAnalysis // teamCode -> analysis
	streakHistory map[string]*models.StreakHistory  // teamCode -> history
	dataDir       string
}

// InitializeStreakDetection initializes the singleton StreakDetectionService
func InitializeStreakDetection() error {
	var err error
	streakDetectionOnce.Do(func() {
		dataDir := filepath.Join("data", "streaks")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create streaks directory: %w", err)
			return
		}

		streakDetectionInstance = &StreakDetectionService{
			teamStreaks:   make(map[string]*models.StreakAnalysis),
			streakHistory: make(map[string]*models.StreakHistory),
			dataDir:       dataDir,
		}

		// Load existing streak data
		if loadErr := streakDetectionInstance.loadStreakData(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing streak data: %v\n", loadErr)
		}

		fmt.Println("✅ Streak Detection Service initialized")
	})
	return err
}

// GetStreakDetectionService returns the singleton instance
func GetStreakDetectionService() *StreakDetectionService {
	return streakDetectionInstance
}

// GetCurrentStreak returns the current streak for a team
func (sds *StreakDetectionService) GetCurrentStreak(teamCode string) *models.StreakPattern {
	sds.mu.RLock()
	defer sds.mu.RUnlock()

	analysis, exists := sds.teamStreaks[teamCode]
	if !exists || len(analysis.ActiveStreaks) == 0 {
		// Return neutral streak
		return &models.StreakPattern{
			TeamCode:         teamCode,
			Type:             "Neutral",
			Length:           0,
			ImpactFactor:     0.0,
			BreakProbability: 0.5,
			Confidence:       0.5,
		}
	}

	// Return the primary streak (win or loss)
	for i := range analysis.ActiveStreaks {
		if analysis.ActiveStreaks[i].Type == "Win" || analysis.ActiveStreaks[i].Type == "Loss" {
			return &analysis.ActiveStreaks[i]
		}
	}

	// Return first active streak
	if len(analysis.ActiveStreaks) > 0 {
		return &analysis.ActiveStreaks[0]
	}

	// Fallback to neutral streak
	return &models.StreakPattern{
		TeamCode:         teamCode,
		Type:             "Neutral",
		Length:           0,
		ImpactFactor:     0.0,
		BreakProbability: 0.5,
		Confidence:       0.5,
	}
}

// CalculateStreakImpact calculates the prediction adjustment based on streaks
func (sds *StreakDetectionService) CalculateStreakImpact(homeStreak, awayStreak *models.StreakPattern) float64 {
	if homeStreak == nil || awayStreak == nil {
		return 0.0
	}

	homeImpact := homeStreak.ImpactFactor
	awayImpact := awayStreak.ImpactFactor

	// Net impact (positive favors home, negative favors away)
	netImpact := homeImpact - awayImpact

	// Apply confidence scaling
	avgConfidence := (homeStreak.Confidence + awayStreak.Confidence) / 2.0
	scaledImpact := netImpact * avgConfidence

	// Clamp to reasonable bounds
	return math.Max(-0.15, math.Min(0.15, scaledImpact))
}

// DetectStreaks analyzes a team's streak patterns
func (sds *StreakDetectionService) DetectStreaks(teamCode string, recentGames []models.GameResult) *models.StreakAnalysis {
	sds.mu.Lock()
	defer sds.mu.Unlock()

	analysis := &models.StreakAnalysis{
		TeamCode:      teamCode,
		Season:        getCurrentSeason(),
		ActiveStreaks: []models.StreakPattern{},
		LastUpdated:   time.Now(),
	}

	if len(recentGames) == 0 {
		sds.teamStreaks[teamCode] = analysis
		return analysis
	}

	// Calculate win/loss streak
	currentStreak := 0
	streakType := ""
	
	for i := len(recentGames) - 1; i >= 0; i-- {
		game := recentGames[i]
		isWin := game.DidTeamWin(teamCode)
		
		if currentStreak == 0 {
			// Start new streak
			if isWin {
				currentStreak = 1
				streakType = "Win"
			} else {
				currentStreak = -1
				streakType = "Loss"
			}
		} else if (isWin && currentStreak > 0) || (!isWin && currentStreak < 0) {
			// Continue streak
			if isWin {
				currentStreak++
			} else {
				currentStreak--
			}
		} else {
			// Streak broken
			break
		}
	}

	// Calculate streak impact
	impact := sds.calculateStreakImpactValue(currentStreak)
	breakProb := sds.calculateBreakProbability(currentStreak)

	streak := models.StreakPattern{
		TeamCode:         teamCode,
		Type:             streakType,
		Length:           int(math.Abs(float64(currentStreak))),
		StartDate:        time.Now().AddDate(0, 0, -int(math.Abs(float64(currentStreak)))+1),
		Confidence:       0.8, // Base confidence
		Historical:       sds.getHistoricalContinuationRate(teamCode, currentStreak),
		BreakProbability: breakProb,
		ImpactFactor:     impact,
		IsHot:            currentStreak >= 3,
		IsCold:           currentStreak <= -3,
		IsDominant:       currentStreak >= 5,
		IsInCrisis:       currentStreak <= -5,
	}

	analysis.ActiveStreaks = append(analysis.ActiveStreaks, streak)
	
	if currentStreak > 0 {
		analysis.CurrentWinStreak = currentStreak
	} else if currentStreak < 0 {
		analysis.CurrentLossStreak = -currentStreak
	}

	sds.teamStreaks[teamCode] = analysis
	sds.saveStreakData()

	return analysis
}

// calculateStreakImpactValue determines the prediction adjustment for a streak
func (sds *StreakDetectionService) calculateStreakImpactValue(streak int) float64 {
	absStreak := math.Abs(float64(streak))
	
	// Progressive impact: +5% per game for first 3, +3% for next 2, +2% after
	var impact float64
	if absStreak <= 3 {
		impact = absStreak * 0.05
	} else if absStreak <= 5 {
		impact = 0.15 + (absStreak-3)*0.03
	} else {
		impact = 0.21 + (absStreak-5)*0.02
	}

	// Cap at 15%
	impact = math.Min(impact, 0.15)

	// Negative for loss streaks
	if streak < 0 {
		impact = -impact
	}

	return impact
}

// calculateBreakProbability estimates probability of streak ending
func (sds *StreakDetectionService) calculateBreakProbability(streak int) float64 {
	absStreak := float64(math.Abs(float64(streak)))
	
	// Base probability increases with streak length
	// 3-game: 40%, 5-game: 50%, 7-game: 60%, 10-game: 70%
	baseProb := 0.30 + (absStreak * 0.04)
	
	return math.Min(baseProb, 0.80)
}

// getHistoricalContinuationRate gets historical rate of streak continuation
func (sds *StreakDetectionService) getHistoricalContinuationRate(teamCode string, streak int) float64 {
	// Simplified: Return baseline based on league averages
	absStreak := int(math.Abs(float64(streak)))
	
	// Historical NHL data shows continuation rates
	continuationRates := map[int]float64{
		1: 0.55,
		2: 0.52,
		3: 0.48,
		4: 0.45,
		5: 0.40,
	}
	
	if rate, exists := continuationRates[absStreak]; exists {
		return rate
	}
	
	// For longer streaks, continuation becomes less likely
	return math.Max(0.30, 0.55-(float64(absStreak)*0.03))
}

// CompareStreaks compares streaks between two teams
func (sds *StreakDetectionService) CompareStreaks(homeTeam, awayTeam string) *models.StreakComparison {
	homeStreak := sds.GetCurrentStreak(homeTeam)
	awayStreak := sds.GetCurrentStreak(awayTeam)

	impact := sds.CalculateStreakImpact(homeStreak, awayStreak)
	
	advantage := "Neutral"
	if impact > 0.03 {
		advantage = "Home"
	} else if impact < -0.03 {
		advantage = "Away"
	}

	analysis := fmt.Sprintf("%s on %d-game %s streak vs %s on %d-game %s streak",
		homeTeam, homeStreak.Length, homeStreak.Type,
		awayTeam, awayStreak.Length, awayStreak.Type)

	return &models.StreakComparison{
		HomeTeam:           homeTeam,
		AwayTeam:           awayTeam,
		HomeStreak:         *homeStreak,
		AwayStreak:         *awayStreak,
		Advantage:          advantage,
		ImpactDifferential: impact,
		Confidence:         (homeStreak.Confidence + awayStreak.Confidence) / 2.0,
		Analysis:           analysis,
	}
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (sds *StreakDetectionService) getStreakDataPath() string {
	return filepath.Join(sds.dataDir, "team_streaks.json")
}

func (sds *StreakDetectionService) saveStreakData() error {
	data, err := json.MarshalIndent(sds.teamStreaks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal streak data: %w", err)
	}
	if err := os.WriteFile(sds.getStreakDataPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write streak data: %w", err)
	}
	return nil
}

func (sds *StreakDetectionService) loadStreakData() error {
	filePath := sds.getStreakDataPath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data yet
		}
		return fmt.Errorf("failed to read streak data: %w", err)
	}

	if err := json.Unmarshal(data, &sds.teamStreaks); err != nil {
		return fmt.Errorf("failed to unmarshal streak data: %w", err)
	}

	fmt.Printf("🔥 Loaded streak data for %d teams\n", len(sds.teamStreaks))
	return nil
}

func getCurrentSeason() string {
	now := time.Now()
	year := now.Year()
	
	// NHL season spans two calendar years
	if now.Month() >= 10 { // October or later
		return fmt.Sprintf("%d-%d", year, year+1)
	}
	return fmt.Sprintf("%d-%d", year-1, year)
}

