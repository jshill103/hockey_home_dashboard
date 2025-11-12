package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleForceTraining forces training on all existing completed games
// This is useful when games were collected but not trained (e.g., after fixing initialization order bug)
func HandleForceTraining(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get services
	grs := services.GetGameResultsService()
	evalSvc := services.GetEvaluationService()

	if grs == nil {
		http.Error(w, `{"error": "Game Results Service not available"}`, http.StatusServiceUnavailable)
		return
	}

	if evalSvc == nil {
		http.Error(w, `{"error": "Evaluation Service not available - batch training not enabled"}`, http.StatusServiceUnavailable)
		return
	}

	// Return immediately with status
	response := map[string]interface{}{
		"status":  "processing",
		"message": "Force training started. Check server logs for progress.",
	}
	json.NewEncoder(w).Encode(response)

	// Run training in background
	go forceTrainOnExistingGames(grs, evalSvc)
}

// forceTrainOnExistingGames loads all stored games and feeds them to batch training
func forceTrainOnExistingGames(grs *services.GameResultsService, evalSvc *services.ModelEvaluationService) {
	log.Printf("🔄 Starting force training on existing completed games...")
	start := time.Now()

	// Get all stored games (October and November 2025)
	octoberGames, err := grs.GetMonthlyGames(2025, 10)
	if err != nil {
		log.Printf("⚠️ Failed to load October games: %v", err)
		octoberGames = []models.CompletedGame{}
	}

	novemberGames, err := grs.GetMonthlyGames(2025, 11)
	if err != nil {
		log.Printf("⚠️ Failed to load November games: %v", err)
		novemberGames = []models.CompletedGame{}
	}

	allGames := append(octoberGames, novemberGames...)
	log.Printf("📊 Found %d stored completed games to process for training", len(allGames))

	if len(allGames) == 0 {
		log.Printf("⚠️ No completed games found to train on")
		return
	}

	// Feed each game to the batch training queue
	successCount := 0
	failCount := 0

	for i, game := range allGames {
		log.Printf("🏒 [%d/%d] Adding game %d to training batches: %s %d - %s %d",
			i+1, len(allGames), game.GameID,
			game.HomeTeam.TeamCode, game.HomeTeam.Score,
			game.AwayTeam.TeamCode, game.AwayTeam.Score)

		// Add game to batch training queue
		if err := evalSvc.AddGameToBatch(game); err != nil {
			log.Printf("⚠️ Failed to add game %d to batch: %v", game.GameID, err)
			failCount++
		} else {
			successCount++
		}

		// Brief delay to avoid overwhelming the system
		if i > 0 && i%10 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	duration := time.Since(start)
	log.Printf("✅ Force training complete!")
	log.Printf("   Games processed: %d", successCount)
	log.Printf("   Failed: %d", failCount)
	log.Printf("   Duration: %.2fs", duration.Seconds())
	log.Printf("   Check /api/training-metrics to see training results")
}

