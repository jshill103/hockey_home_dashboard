package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// DailyPredictionService generates predictions for all upcoming NHL games
type DailyPredictionService struct {
	predictionStorage *PredictionStorageService
	ensemble          *EnsemblePredictionService
	ticker            *time.Ticker
	stopChan          chan bool
	mutex             sync.Mutex
	running           bool
}

var (
	dailyPredictionService     *DailyPredictionService
	dailyPredictionServiceOnce sync.Once
)

// InitDailyPredictionService initializes the singleton
func InitDailyPredictionService(predictionStorage *PredictionStorageService, ensemble *EnsemblePredictionService) *DailyPredictionService {
	dailyPredictionServiceOnce.Do(func() {
		dailyPredictionService = &DailyPredictionService{
			predictionStorage: predictionStorage,
			ensemble:          ensemble,
			stopChan:          make(chan bool),
		}

		log.Println("🎯 Daily Prediction Service initialized")
	})
	return dailyPredictionService
}

// GetDailyPredictionService returns the singleton instance
func GetDailyPredictionService() *DailyPredictionService {
	return dailyPredictionService
}

// Start begins the daily prediction generation
func (dps *DailyPredictionService) Start() {
	dps.mutex.Lock()
	if dps.running {
		dps.mutex.Unlock()
		return
	}
	dps.running = true
	dps.mutex.Unlock()

	log.Println("🚀 Starting Daily Prediction Service...")

	// Run immediately on startup
	go func() {
		log.Println("⏳ Waiting 10 seconds for services to initialize before first prediction run...")
		time.Sleep(10 * time.Second) // Wait for services to initialize
		log.Println("🎯 Triggering initial prediction generation...")
		dps.generateDailyPredictions()
	}()

	// Schedule daily predictions at 6 AM
	go dps.scheduleDailyPredictions()

	log.Println("✅ Daily Prediction Service started (runs daily at 6:00 AM)")
}

// Stop stops the daily prediction service
func (dps *DailyPredictionService) Stop() {
	dps.mutex.Lock()
	defer dps.mutex.Unlock()

	if !dps.running {
		return
	}

	log.Println("⏹️ Stopping Daily Prediction Service...")
	dps.stopChan <- true
	if dps.ticker != nil {
		dps.ticker.Stop()
	}
	dps.running = false
	log.Println("✅ Daily Prediction Service stopped")
}

// scheduleDailyPredictions runs predictions once per day at 6 AM
func (dps *DailyPredictionService) scheduleDailyPredictions() {
	// Calculate time until next 6 AM
	now := time.Now()
	next6AM := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
	if now.After(next6AM) {
		next6AM = next6AM.Add(24 * time.Hour)
	}

	durationUntil6AM := next6AM.Sub(now)
	log.Printf("⏰ Next daily prediction run scheduled for: %s (in %s)",
		next6AM.Format("2006-01-02 15:04:05"), durationUntil6AM.Round(time.Minute))

	// Wait until 6 AM
	timer := time.NewTimer(durationUntil6AM)
	<-timer.C

	// Run predictions
	dps.generateDailyPredictions()

	// Set up daily ticker (every 24 hours)
	dps.ticker = time.NewTicker(24 * time.Hour)

	for {
		select {
		case <-dps.ticker.C:
			dps.generateDailyPredictions()
		case <-dps.stopChan:
			return
		}
	}
}

// TriggerNow manually triggers prediction generation (for testing/manual refresh)
func (dps *DailyPredictionService) TriggerNow() {
	log.Println("🔄 Manually triggering daily prediction generation...")
	dps.generateDailyPredictions()
}

// generateDailyPredictions generates predictions for all upcoming games
func (dps *DailyPredictionService) generateDailyPredictions() {
	log.Println("🎯 Generating predictions for upcoming NHL games...")

	startTime := time.Now()

	// Fetch upcoming games (next 7 days)
	games, err := dps.fetchUpcomingGames()
	if err != nil {
		log.Printf("❌ Failed to fetch upcoming games: %v", err)
		return
	}

	if len(games) == 0 {
		log.Println("ℹ️ No upcoming games found")
		return
	}

	log.Printf("📊 Found %d upcoming games to predict", len(games))

	// Generate predictions for each game
	successCount := 0
	errorCount := 0

	for _, game := range games {
		// Check if we already have a prediction for this game
		existing, _ := dps.predictionStorage.LoadPrediction(game.GameID)
		if existing != nil {
			log.Printf("⏭️ Skipping game %d (prediction already exists)", game.GameID)
			continue
		}

		// Generate prediction
		prediction, err := dps.generatePredictionForGame(game)
		if err != nil {
			log.Printf("❌ Failed to predict game %d (%s @ %s): %v",
				game.GameID, game.AwayTeam, game.HomeTeam, err)
			errorCount++
			continue
		}

		// Store prediction
		err = dps.predictionStorage.StorePrediction(
			game.GameID,
			game.GameDate,
			game.HomeTeam,
			game.AwayTeam,
			prediction,
		)
		if err != nil {
			log.Printf("❌ Failed to store prediction for game %d: %v", game.GameID, err)
			errorCount++
			continue
		}

		successCount++
	}

	duration := time.Since(startTime)
	log.Printf("✅ Daily predictions complete: %d successful, %d errors (took %s)",
		successCount, errorCount, duration.Round(time.Millisecond))

	// Log next run time
	nextRun := time.Now().Add(24 * time.Hour)
	nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), 6, 0, 0, 0, nextRun.Location())
	log.Printf("⏰ Next prediction run: %s", nextRun.Format("2006-01-02 15:04:05"))
}

// fetchUpcomingGames retrieves all NHL games for the next 7 days
func (dps *DailyPredictionService) fetchUpcomingGames() ([]UpcomingGame, error) {
	// Use the NHL scoreboard to get upcoming games
	today := time.Now().Format("2006-01-02")

	// Fetch league-wide schedule
	url := fmt.Sprintf("https://api-web.nhle.com/v1/schedule/%s", today)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schedule: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var scheduleData struct {
		GameWeek []struct {
			Date  string `json:"date"`
			Games []struct {
				ID       int    `json:"id"`
				Season   int    `json:"season"`
				GameType int    `json:"gameType"`
				GameDate string `json:"gameDate"`
				Venue    struct {
					Default string `json:"default"`
				} `json:"venue"`
				StartTimeUTC string `json:"startTimeUTC"`
				HomeTeam     struct {
					Abbrev string `json:"abbrev"`
					ID     int    `json:"id"`
				} `json:"homeTeam"`
				AwayTeam struct {
					Abbrev string `json:"abbrev"`
					ID     int    `json:"id"`
				} `json:"awayTeam"`
				GameState string `json:"gameState"`
			} `json:"games"`
		} `json:"gameWeek"`
	}

	err = json.Unmarshal(body, &scheduleData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schedule: %w", err)
	}

	games := []UpcomingGame{}
	for _, week := range scheduleData.GameWeek {
		for _, game := range week.Games {
			// Only predict games that haven't started yet
			if game.GameState != "FUT" && game.GameState != "PRE" {
				continue
			}

			// Only regular season and playoff games
			if game.GameType != 2 && game.GameType != 3 {
				continue
			}

			gameDate, _ := time.Parse(time.RFC3339, game.StartTimeUTC)

			// Only games in next 7 days
			if gameDate.After(time.Now().Add(7 * 24 * time.Hour)) {
				continue
			}

			games = append(games, UpcomingGame{
				GameID:   game.ID,
				GameDate: gameDate,
				HomeTeam: game.HomeTeam.Abbrev,
				AwayTeam: game.AwayTeam.Abbrev,
				Venue:    game.Venue.Default,
			})
		}
	}

	return games, nil
}

// UpcomingGame represents a game that needs a prediction
type UpcomingGame struct {
	GameID   int
	GameDate time.Time
	HomeTeam string
	AwayTeam string
	Venue    string
}

// generatePredictionForGame creates a prediction for a specific game
func (dps *DailyPredictionService) generatePredictionForGame(game UpcomingGame) (*models.GamePrediction, error) {
	log.Printf("🎲 Predicting: %s @ %s (Game %d)", game.AwayTeam, game.HomeTeam, game.GameID)

	// Build basic prediction factors
	// TODO: In future, fetch actual team stats for more accurate predictions
	homeFactors := &models.PredictionFactors{
		WinPercentage: 0.5,
		GoalsFor:      3.0,
		GoalsAgainst:  3.0,
	}

	awayFactors := &models.PredictionFactors{
		WinPercentage: 0.5,
		GoalsFor:      3.0,
		GoalsAgainst:  3.0,
	}

	// Use ensemble prediction service
	result, err := dps.ensemble.PredictGame(homeFactors, awayFactors)
	if err != nil {
		return nil, fmt.Errorf("ensemble prediction failed: %w", err)
	}

	// Convert PredictionResult to GamePrediction
	// Calculate home and away win probabilities based on prediction
	homeWinProb := result.WinProbability
	awayWinProb := 1.0 - result.WinProbability
	if result.Winner == game.AwayTeam {
		// If away team is predicted to win, swap probabilities
		homeWinProb = 1.0 - result.WinProbability
		awayWinProb = result.WinProbability
	}

	gamePrediction := &models.GamePrediction{
		GameID: game.GameID,
		HomeTeam: models.PredictionTeam{
			Name:           game.HomeTeam,
			Code:           game.HomeTeam,
			WinProbability: homeWinProb,
		},
		AwayTeam: models.PredictionTeam{
			Name:           game.AwayTeam,
			Code:           game.AwayTeam,
			WinProbability: awayWinProb,
		},
		GameDate:    game.GameDate,
		Confidence:  result.Confidence,
		Prediction:  *result,
		KeyFactors:  []string{},
		GeneratedAt: time.Now(),
	}

	return gamePrediction, nil
}

// GetStats returns statistics about daily predictions
func (dps *DailyPredictionService) GetStats() map[string]interface{} {
	// Get predictions from storage
	predictions, err := dps.predictionStorage.GetAllPredictions()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Count predictions by status
	totalPredictions := len(predictions)
	withResults := 0
	pending := 0

	for _, pred := range predictions {
		if pred.ActualResult != nil {
			withResults++
		} else {
			pending++
		}
	}

	return map[string]interface{}{
		"totalPredictions":       totalPredictions,
		"predictionsWithResults": withResults,
		"pendingPredictions":     pending,
		"running":                dps.running,
	}
}
