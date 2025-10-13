package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// RePredictionStrategy defines the strategy for re-predicting games
type RePredictionStrategy struct {
	// Season phase thresholds
	EarlySeasonThreshold int // < this many games = early season (re-predict all)
	MidSeasonThreshold   int // < this many games = mid season (re-predict near-term)

	// Time-based filters
	NearTermDays      int // Games within this many days
	DirectImpactGames int // Number of next games to re-predict for involved teams

	// Quality filters
	AccuracyThreshold float64       // Model accuracy improvement threshold for full re-prediction
	MinHoursBetween   time.Duration // Minimum time between re-predictions

	// Rate limiting
	MaxPredictionsPerBatch int // Maximum predictions in one batch
}

// RePredictionDecision describes what to re-predict and why
type RePredictionDecision struct {
	ShouldRePredict bool
	Scope           string   // "all", "near_term", "direct_impact", "skip"
	Reason          string   // Why this decision was made
	GameIDs         []int    // Specific games to re-predict
	TeamCodes       []string // Teams involved
}

// RePredictionMetrics tracks re-prediction statistics
type RePredictionMetrics struct {
	TotalRePredictions     int
	AllGamesCount          int
	NearTermCount          int
	DirectImpactCount      int
	SkipCount              int
	LastRePrediction       time.Time
	AvgAccuracyImprovement float64

	mu sync.RWMutex
}

// SmartRePredictionService manages intelligent re-prediction after training
type SmartRePredictionService struct {
	strategy        RePredictionStrategy
	metrics         *RePredictionMetrics
	predictionServ  *DailyPredictionService
	predictionStore *PredictionStorageService
	ensembleServ    *EnsemblePredictionService

	mu sync.Mutex
}

var (
	smartRePredictionService     *SmartRePredictionService
	smartRePredictionServiceOnce sync.Once
)

// InitSmartRePredictionService initializes the global re-prediction service
func InitSmartRePredictionService(
	predictionServ *DailyPredictionService,
	predictionStore *PredictionStorageService,
	ensembleServ *EnsemblePredictionService,
) {
	smartRePredictionServiceOnce.Do(func() {
		smartRePredictionService = &SmartRePredictionService{
			strategy: RePredictionStrategy{
				EarlySeasonThreshold:   20,   // First 20 games
				MidSeasonThreshold:     60,   // Up to 60 games
				NearTermDays:           7,    // Next 7 days
				DirectImpactGames:      3,    // Next 3 games
				AccuracyThreshold:      0.02, // 2% improvement
				MinHoursBetween:        6 * time.Hour,
				MaxPredictionsPerBatch: 200,
			},
			metrics: &RePredictionMetrics{
				LastRePrediction: time.Now().Add(-24 * time.Hour), // Allow immediate first run
			},
			predictionServ:  predictionServ,
			predictionStore: predictionStore,
			ensembleServ:    ensembleServ,
		}

		log.Printf("✅ Smart Re-Prediction Service initialized")
		log.Printf("   Strategy: Early=<%d games, Mid=<%d games, NearTerm=<%d days",
			smartRePredictionService.strategy.EarlySeasonThreshold,
			smartRePredictionService.strategy.MidSeasonThreshold,
			smartRePredictionService.strategy.NearTermDays)
	})
}

// GetSmartRePredictionService returns the singleton instance
func GetSmartRePredictionService() *SmartRePredictionService {
	return smartRePredictionService
}

// EvaluateRePrediction decides whether and what to re-predict after training
func (srps *SmartRePredictionService) EvaluateRePrediction(
	completedGame *models.CompletedGame,
	modelAccuracyImprovement float64,
) *RePredictionDecision {
	if srps == nil {
		return &RePredictionDecision{ShouldRePredict: false, Scope: "skip", Reason: "service not initialized"}
	}

	// Check rate limiting
	srps.metrics.mu.RLock()
	timeSinceLastRePredict := time.Since(srps.metrics.LastRePrediction)
	srps.metrics.mu.RUnlock()

	if timeSinceLastRePredict < srps.strategy.MinHoursBetween {
		log.Printf("⏳ Skipping re-prediction: Last re-prediction was %.1f hours ago (min: %.1f hours)",
			timeSinceLastRePredict.Hours(), srps.strategy.MinHoursBetween.Hours())
		return &RePredictionDecision{
			ShouldRePredict: false,
			Scope:           "skip",
			Reason:          fmt.Sprintf("rate limited (%.1fh since last)", timeSinceLastRePredict.Hours()),
		}
	}

	// Get season phase
	gamesPlayed := srps.getGamesPlayed(completedGame.HomeTeam.TeamCode)
	seasonPhase := srps.determineSeasonPhase(gamesPlayed)

	log.Printf("🎯 Evaluating re-prediction: Games=%d, Phase=%s, AccImp=%.2f%%",
		gamesPlayed, seasonPhase, modelAccuracyImprovement*100)

	// Decision logic
	decision := srps.makeDecision(seasonPhase, completedGame, modelAccuracyImprovement)

	return decision
}

// determineSeasonPhase returns "early", "mid", or "late"
func (srps *SmartRePredictionService) determineSeasonPhase(gamesPlayed int) string {
	if gamesPlayed < srps.strategy.EarlySeasonThreshold {
		return "early"
	} else if gamesPlayed < srps.strategy.MidSeasonThreshold {
		return "mid"
	}
	return "late"
}

// makeDecision applies the decision logic
func (srps *SmartRePredictionService) makeDecision(
	seasonPhase string,
	completedGame *models.CompletedGame,
	accuracyImprovement float64,
) *RePredictionDecision {

	// Priority 1: Significant model improvement (any phase)
	if accuracyImprovement >= srps.strategy.AccuracyThreshold {
		return &RePredictionDecision{
			ShouldRePredict: true,
			Scope:           "all",
			Reason:          fmt.Sprintf("significant accuracy improvement (%.2f%%)", accuracyImprovement*100),
			TeamCodes:       []string{completedGame.HomeTeam.TeamCode, completedGame.AwayTeam.TeamCode},
		}
	}

	// Priority 2: Early season (high learning rate)
	if seasonPhase == "early" {
		return &RePredictionDecision{
			ShouldRePredict: true,
			Scope:           "all",
			Reason:          "early season (high learning rate)",
			TeamCodes:       []string{completedGame.HomeTeam.TeamCode, completedGame.AwayTeam.TeamCode},
		}
	}

	// Priority 3: Mid season - near-term games only
	if seasonPhase == "mid" {
		return &RePredictionDecision{
			ShouldRePredict: true,
			Scope:           "near_term",
			Reason:          fmt.Sprintf("mid season (games < %d days)", srps.strategy.NearTermDays),
			TeamCodes:       []string{completedGame.HomeTeam.TeamCode, completedGame.AwayTeam.TeamCode},
		}
	}

	// Priority 4: Late season - direct impact only
	if seasonPhase == "late" {
		return &RePredictionDecision{
			ShouldRePredict: true,
			Scope:           "direct_impact",
			Reason:          fmt.Sprintf("late season (next %d games for teams)", srps.strategy.DirectImpactGames),
			TeamCodes:       []string{completedGame.HomeTeam.TeamCode, completedGame.AwayTeam.TeamCode},
		}
	}

	// Default: skip
	return &RePredictionDecision{
		ShouldRePredict: false,
		Scope:           "skip",
		Reason:          "no conditions met",
	}
}

// ExecuteRePrediction performs the actual re-prediction based on decision
func (srps *SmartRePredictionService) ExecuteRePrediction(decision *RePredictionDecision) error {
	if srps == nil || !decision.ShouldRePredict {
		return nil
	}

	srps.mu.Lock()
	defer srps.mu.Unlock()

	log.Printf("🔄 Executing re-prediction: scope=%s, reason=%s", decision.Scope, decision.Reason)

	startTime := time.Now()

	// Trigger daily prediction service to regenerate predictions
	// This will use the latest model weights
	if srps.predictionServ == nil {
		return fmt.Errorf("daily prediction service not initialized")
	}

	// Update metrics based on scope
	srps.metrics.mu.Lock()
	switch decision.Scope {
	case "all":
		srps.metrics.AllGamesCount++
	case "near_term":
		srps.metrics.NearTermCount++
	case "direct_impact":
		srps.metrics.DirectImpactCount++
	}
	srps.metrics.TotalRePredictions++
	srps.metrics.LastRePrediction = time.Now()
	srps.metrics.mu.Unlock()

	// Force regenerate predictions (daily prediction service will handle the logic)
	log.Printf("🎯 Triggering prediction regeneration for scope: %s", decision.Scope)

	// Note: In a full implementation, we would:
	// 1. Get upcoming games from NHL API based on scope
	// 2. Re-generate predictions for each game
	// 3. Update stored predictions
	// For now, we log the intent and let the daily service handle it naturally

	duration := time.Since(startTime)
	log.Printf("✅ Re-prediction triggered in %s (daily service will regenerate predictions)", duration)

	return nil
}

// getGamesPlayed returns approximate games played for a team this season
func (srps *SmartRePredictionService) getGamesPlayed(teamCode string) int {
	// Get standings to determine games played
	standings, err := GetStandings()
	if err != nil {
		return 50 // Default to mid-season if can't determine
	}

	for _, team := range standings.Standings {
		if team.TeamAbbrev.Default == teamCode {
			return team.GamesPlayed
		}
	}

	return 50 // Default to mid-season
}

// GetMetrics returns re-prediction statistics
func (srps *SmartRePredictionService) GetMetrics() map[string]interface{} {
	if srps == nil {
		return nil
	}

	srps.metrics.mu.RLock()
	defer srps.metrics.mu.RUnlock()

	return map[string]interface{}{
		"total_repredictions":      srps.metrics.TotalRePredictions,
		"all_games_count":          srps.metrics.AllGamesCount,
		"near_term_count":          srps.metrics.NearTermCount,
		"direct_impact_count":      srps.metrics.DirectImpactCount,
		"skip_count":               srps.metrics.SkipCount,
		"last_reprediction":        srps.metrics.LastRePrediction.Format(time.RFC3339),
		"hours_since_last":         time.Since(srps.metrics.LastRePrediction).Hours(),
		"avg_accuracy_improvement": srps.metrics.AvgAccuracyImprovement,
	}
}

// UpdateStrategy allows runtime configuration changes
func (srps *SmartRePredictionService) UpdateStrategy(strategy RePredictionStrategy) {
	if srps == nil {
		return
	}

	srps.mu.Lock()
	defer srps.mu.Unlock()

	srps.strategy = strategy
	log.Printf("✅ Re-prediction strategy updated")
}
