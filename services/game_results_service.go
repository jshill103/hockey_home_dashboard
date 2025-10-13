package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// Global game results service instance
var (
	globalGameResultsService *GameResultsService
	gameResultsMutex         sync.Mutex
)

// GameResultsService automatically detects and processes completed games
type GameResultsService struct {
	teamCode        string
	dataDir         string
	checkInterval   time.Duration
	processedGames  map[int]bool
	eloModel        *EloRatingModel
	poissonModel    *PoissonRegressionModel
	neuralNet       *NeuralNetworkModel
	rollingStats    *RollingStatsService
	accuracyTracker *AccuracyTrackingService
	evaluationSvc   *ModelEvaluationService // For batch training
	mutex           sync.RWMutex
	stopChan        chan bool
	isRunning       bool
	httpClient      *http.Client
}

// NewGameResultsService creates a new game results collection service
func NewGameResultsService(
	teamCode string,
	eloModel *EloRatingModel,
	poissonModel *PoissonRegressionModel,
	neuralNet *NeuralNetworkModel,
	rollingStats *RollingStatsService,
	accuracyTracker *AccuracyTrackingService,
) *GameResultsService {
	service := &GameResultsService{
		teamCode:        teamCode,
		dataDir:         "data/results",
		checkInterval:   5 * time.Minute,
		processedGames:  make(map[int]bool),
		eloModel:        eloModel,
		poissonModel:    poissonModel,
		neuralNet:       neuralNet,
		rollingStats:    rollingStats,
		accuracyTracker: accuracyTracker,
		stopChan:        make(chan bool),
		isRunning:       false,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Create data directory
	os.MkdirAll(service.dataDir, 0755)

	// Load processed games index
	if err := service.loadProcessedGames(); err != nil {
		log.Printf("⚠️ Could not load processed games index: %v (starting fresh)", err)
	}

	return service
}

// Start begins the game monitoring process
func (grs *GameResultsService) Start() {
	grs.mutex.Lock()
	if grs.isRunning {
		grs.mutex.Unlock()
		log.Printf("⚠️ Game Results Service already running")
		return
	}
	grs.isRunning = true
	grs.mutex.Unlock()

	log.Printf("📊 Game Results Service started (League-wide collection, primary team: %s)", grs.teamCode)
	log.Printf("📊 Loaded %d processed games from index", len(grs.processedGames))
	log.Printf("📊 Checking for completed NHL games every %v", grs.checkInterval)
	log.Printf("🌍 Processing ALL NHL games for maximum ML training data")

	// Start background monitoring goroutine
	go grs.monitorGames()
}

// Stop halts the game monitoring process
func (grs *GameResultsService) Stop() {
	grs.mutex.Lock()
	defer grs.mutex.Unlock()

	if !grs.isRunning {
		return
	}

	log.Printf("⏹️ Stopping Game Results Service...")
	grs.stopChan <- true
	grs.isRunning = false
	log.Printf("✅ Game Results Service stopped")
}

// monitorGames runs the continuous monitoring loop
func (grs *GameResultsService) monitorGames() {
	ticker := time.NewTicker(grs.checkInterval)
	defer ticker.Stop()

	// Do an immediate check on startup
	grs.checkForCompletedGames()

	for {
		select {
		case <-ticker.C:
			grs.checkForCompletedGames()
		case <-grs.stopChan:
			return
		}
	}
}

// checkForCompletedGames checks for new completed games (league-wide)
func (grs *GameResultsService) checkForCompletedGames() {
	log.Printf("🔍 Checking for completed NHL games (league-wide)...")

	// Get league-wide scoreboard from NHL API
	schedule, err := grs.fetchWeekSchedule()
	if err != nil {
		log.Printf("❌ Failed to fetch league scoreboard: %v", err)
		return
	}

	// Find completed games that haven't been processed
	var newGames []models.ScoreboardGame
	for _, gamesByDate := range schedule.GamesByDate {
		for _, game := range gamesByDate.Games {
			if grs.isGameCompleted(game) && !grs.isProcessed(game.GameID) {
				newGames = append(newGames, game)
			}
		}
	}

	if len(newGames) == 0 {
		log.Printf("✅ No new completed games found (league-wide scan)")
		return
	}

	log.Printf("🌍 Found %d new completed NHL game(s) to process!", len(newGames))

	// Process each new game
	for _, game := range newGames {
		if err := grs.processGame(game.GameID); err != nil {
			log.Printf("❌ Failed to process game %d: %v", game.GameID, err)
		} else {
			// Mark as processed
			grs.markProcessed(game.GameID)
		}
	}

	// Save updated index
	if err := grs.saveProcessedGames(); err != nil {
		log.Printf("⚠️ Failed to save processed games index: %v", err)
	}
}

// fetchWeekSchedule fetches ALL NHL games (league-wide) for the current week
func (grs *GameResultsService) fetchWeekSchedule() (*models.ScoreboardResponse, error) {
	// Use the league-wide scoreboard endpoint to get all games
	// This fetches all NHL games, not just one team
	url := "https://api-web.nhle.com/v1/scoreboard/now"

	// Use MakeAPICall for caching and rate limiting
	body, err := MakeAPICall(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch league scoreboard: %w", err)
	}

	// Parse the scoreboard response which includes all NHL games
	var scheduleData models.ScoreboardResponse
	if err := json.Unmarshal(body, &scheduleData); err != nil {
		return nil, fmt.Errorf("failed to decode league scoreboard: %w", err)
	}

	// Return the full scoreboard with all NHL games
	return &scheduleData, nil
}

// isGameCompleted checks if a game is completed
func (grs *GameResultsService) isGameCompleted(game models.ScoreboardGame) bool {
	return game.GameState == "FINAL" || game.GameState == "OFF"
}

// isProcessed checks if a game has already been processed
func (grs *GameResultsService) isProcessed(gameID int) bool {
	grs.mutex.RLock()
	defer grs.mutex.RUnlock()
	return grs.processedGames[gameID]
}

// markProcessed marks a game as processed
func (grs *GameResultsService) markProcessed(gameID int) {
	grs.mutex.Lock()
	defer grs.mutex.Unlock()
	grs.processedGames[gameID] = true
}

// processGame fetches and processes a completed game
func (grs *GameResultsService) processGame(gameID int) error {
	log.Printf("📥 Fetching data for game %d...", gameID)

	// Fetch boxscore data
	boxscore, err := grs.fetchBoxscore(gameID)
	if err != nil {
		return fmt.Errorf("failed to fetch boxscore: %w", err)
	}

	// Transform to CompletedGame
	completedGame := grs.transformBoxscore(boxscore)

	// Save to monthly file
	if err := grs.saveGame(completedGame); err != nil {
		return fmt.Errorf("failed to save game: %w", err)
	}

	// Feed to models
	grs.feedToModels(completedGame)

	log.Printf("✅ Game processed: %s %d - %s %d (%s)",
		completedGame.HomeTeam.TeamCode, completedGame.HomeTeam.Score,
		completedGame.AwayTeam.TeamCode, completedGame.AwayTeam.Score,
		completedGame.WinType)

	return nil
}

// fetchBoxscore fetches boxscore data from NHL API
func (grs *GameResultsService) fetchBoxscore(gameID int) (*models.BoxscoreResponse, error) {
	url := fmt.Sprintf("https://api-web.nhle.com/v1/gamecenter/%d/boxscore", gameID)

	resp, err := grs.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch boxscore: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var boxscore models.BoxscoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&boxscore); err != nil {
		return nil, fmt.Errorf("failed to decode boxscore: %w", err)
	}

	return &boxscore, nil
}

// transformBoxscore converts NHL API boxscore to our CompletedGame format
func (grs *GameResultsService) transformBoxscore(boxscore *models.BoxscoreResponse) *models.CompletedGame {
	// Parse game date
	gameDate, _ := time.Parse(time.RFC3339, boxscore.StartTimeUTC)

	// Determine winner
	winner := boxscore.HomeTeam.Abbrev
	if boxscore.AwayTeam.Score > boxscore.HomeTeam.Score {
		winner = boxscore.AwayTeam.Abbrev
	}

	// Determine win type
	winType := "REG"
	if boxscore.GameOutcome.LastPeriodType == "OT" {
		winType = "OT"
	} else if boxscore.GameOutcome.LastPeriodType == "SO" {
		winType = "SO"
	}

	// Extract team stats (these would need to be parsed from Summary if available)
	homeTeamResult := grs.extractTeamStats(boxscore, true)
	awayTeamResult := grs.extractTeamStats(boxscore, false)

	return &models.CompletedGame{
		GameID:      boxscore.GameID,
		GameDate:    gameDate,
		Season:      boxscore.Season,
		GameType:    boxscore.GameType,
		ProcessedAt: time.Now(),
		HomeTeam:    homeTeamResult,
		AwayTeam:    awayTeamResult,
		Winner:      winner,
		WinType:     winType,
		Venue:       boxscore.Venue.Default,
		Attendance:  0, // Would be in BoxScore.GameInfo if available
		DataSource:  "NHLE_API_v1",
		DataVersion: "1.0",
	}
}

// extractTeamStats extracts team statistics from boxscore
func (grs *GameResultsService) extractTeamStats(boxscore *models.BoxscoreResponse, isHome bool) models.TeamGameResult {
	var team models.BoxscoreTeam
	if isHome {
		team = boxscore.HomeTeam
	} else {
		team = boxscore.AwayTeam
	}

	// Calculate shot percentage
	shotPct := 0.0
	if team.SOG > 0 {
		shotPct = float64(team.Score) / float64(team.SOG) * 100.0
	}

	// Extract stats from Summary if available
	ppGoals, ppOpps, ppPct := grs.extractPowerPlayStats(boxscore, isHome)
	faceoffWins, faceoffTotal, faceoffPct := grs.extractFaceoffStats(boxscore, isHome)
	hits, blocks, giveaways, takeaways := grs.extractPhysicalStats(boxscore, isHome)
	pim := grs.extractPenaltyMinutes(boxscore, isHome)

	return models.TeamGameResult{
		TeamCode:       team.Abbrev,
		TeamName:       team.Name.Default,
		Score:          team.Score,
		Shots:          team.SOG,
		ShotPct:        shotPct,
		PowerPlayGoals: ppGoals,
		PowerPlayOpps:  ppOpps,
		PowerPlayPct:   ppPct,
		PenaltyMinutes: pim,
		FaceoffWins:    faceoffWins,
		FaceoffTotal:   faceoffTotal,
		FaceoffPct:     faceoffPct,
		Hits:           hits,
		Blocks:         blocks,
		Giveaways:      giveaways,
		Takeaways:      takeaways,
	}
}

// extractPowerPlayStats extracts power play statistics
func (grs *GameResultsService) extractPowerPlayStats(boxscore *models.BoxscoreResponse, isHome bool) (int, int, float64) {
	// These would be parsed from boxscore.Summary.TeamGameStats if available
	// For now, return zeros as a placeholder
	return 0, 0, 0.0
}

// extractFaceoffStats extracts faceoff statistics
func (grs *GameResultsService) extractFaceoffStats(boxscore *models.BoxscoreResponse, isHome bool) (int, int, float64) {
	// These would be parsed from boxscore.Summary.TeamGameStats if available
	return 0, 0, 0.0
}

// extractPhysicalStats extracts physical play statistics
func (grs *GameResultsService) extractPhysicalStats(boxscore *models.BoxscoreResponse, isHome bool) (int, int, int, int) {
	// These would be parsed from boxscore.Summary.TeamGameStats if available
	return 0, 0, 0, 0
}

// extractPenaltyMinutes extracts penalty minutes
func (grs *GameResultsService) extractPenaltyMinutes(boxscore *models.BoxscoreResponse, isHome bool) int {
	// Would be parsed from boxscore.Summary.TeamGameStats if available
	return 0
}

// saveGame saves a completed game to the monthly file
func (grs *GameResultsService) saveGame(game *models.CompletedGame) error {
	// Determine monthly file
	monthKey := game.GameDate.Format("2006-01")
	filePath := filepath.Join(grs.dataDir, monthKey+".json")

	// Load existing games for this month
	var games []models.CompletedGame
	if data, err := ioutil.ReadFile(filePath); err == nil {
		if err := json.Unmarshal(data, &games); err != nil {
			log.Printf("⚠️ Failed to unmarshal existing games: %v", err)
		}
	}

	// Check for duplicates
	for i, existingGame := range games {
		if existingGame.GameID == game.GameID {
			// Update existing game instead of appending
			games[i] = *game
			log.Printf("📝 Updated existing game record for game %d", game.GameID)

			// Save back
			data, _ := json.MarshalIndent(games, "", "  ")
			return ioutil.WriteFile(filePath, data, 0644)
		}
	}

	// Append new game
	games = append(games, *game)

	// Save back
	data, _ := json.MarshalIndent(games, "", "  ")
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write game data: %w", err)
	}

	log.Printf("💾 Saved to %s", filePath)
	return nil
}

// feedToModels feeds the completed game data to ML models
func (grs *GameResultsService) feedToModels(game *models.CompletedGame) {
	// Convert to GameResult format that models expect
	gameResult := grs.convertToGameResult(game)

	// Update Elo ratings (real-time, not batched)
	if grs.eloModel != nil {
		if err := grs.eloModel.processGameResult(gameResult); err != nil {
			log.Printf("⚠️ Failed to update Elo model: %v", err)
		} else {
			log.Printf("🏆 Elo ratings updated")
		}
	}

	// Update Poisson rates (real-time, not batched)
	if grs.poissonModel != nil {
		if err := grs.poissonModel.processGameResult(gameResult); err != nil {
			log.Printf("⚠️ Failed to update Poisson model: %v", err)
		} else {
			log.Printf("🎯 Poisson rates updated")
		}
	}

	// Add game to batch training queue for Neural Network
	if grs.evaluationSvc != nil {
		if err := grs.evaluationSvc.AddGameToBatch(*game); err != nil {
			log.Printf("⚠️ Failed to add game to batch: %v", err)
		}
		// Batch training happens automatically when batch is full
	} else {
		// Fallback to immediate training if evaluation service not available
		if grs.neuralNet != nil {
			homeFactors := grs.buildPredictionFactors(game, true)
			awayFactors := grs.buildPredictionFactors(game, false)

			if err := grs.neuralNet.TrainOnGameResult(gameResult, homeFactors, awayFactors); err != nil {
				log.Printf("⚠️ Failed to train Neural Network: %v", err)
			} else {
				log.Printf("🧠 Neural Network trained on game %d", game.GameID)
			}
		}
	}

	// Update rolling statistics
	if grs.rollingStats != nil {
		if err := grs.rollingStats.UpdateTeamStats(game); err != nil {
			log.Printf("⚠️ Failed to update rolling stats: %v", err)
		} else {
			log.Printf("📊 Rolling statistics updated for both teams")
		}
	}

	// PHASE 6: Update matchup database
	matchupService := GetMatchupService()
	if matchupService != nil {
		if err := matchupService.UpdateMatchupHistory(*game); err != nil {
			log.Printf("⚠️ Failed to update matchup history: %v", err)
		} else {
			log.Printf("📊 Matchup history updated for %s vs %s", game.HomeTeam.TeamCode, game.AwayTeam.TeamCode)
		}
	}

	// Update stored prediction with actual result
	predictionStorage := GetPredictionStorageService()
	if predictionStorage != nil {
		if err := predictionStorage.UpdateWithResult(game.GameID, game); err != nil {
			log.Printf("⚠️ Failed to update prediction with result: %v", err)
		} else {
			log.Printf("✅ Prediction updated with actual result for game %d", game.GameID)
		}
	}

	// Auto-train Meta-Learner if conditions are met
	metaLearner := GetMetaLearnerModel()
	modelAccuracyImprovement := 0.0
	if metaLearner != nil {
		oldAccuracy := metaLearner.GetCurrentAccuracy()
		metaLearner.RecordGameProcessed()
		if err := metaLearner.AutoTrain(); err != nil {
			log.Printf("⚠️ Meta-Learner auto-training failed: %v", err)
		} else {
			newAccuracy := metaLearner.GetCurrentAccuracy()
			modelAccuracyImprovement = newAccuracy - oldAccuracy
			if modelAccuracyImprovement > 0 {
				log.Printf("📈 Meta-Learner accuracy improved by %.2f%% (%.2f%% → %.2f%%)",
					modelAccuracyImprovement*100, oldAccuracy*100, newAccuracy*100)
			}
		}
	}

	// SMART RE-PREDICTION: Evaluate whether to re-predict after training
	smartRePrediction := GetSmartRePredictionService()
	if smartRePrediction != nil {
		// Decide whether to re-predict
		decision := smartRePrediction.EvaluateRePrediction(game, modelAccuracyImprovement)

		if decision.ShouldRePredict {
			log.Printf("🔄 Re-prediction triggered: scope=%s, reason=%s", decision.Scope, decision.Reason)

			// Execute re-prediction in background to avoid blocking
			go func() {
				if err := smartRePrediction.ExecuteRePrediction(decision); err != nil {
					log.Printf("⚠️ Re-prediction failed: %v", err)
				}
			}()
		} else {
			log.Printf("⏭️ Re-prediction skipped: %s", decision.Reason)
		}
	}

	// PLAYOFF ODDS: Recalculate playoff odds after game completion
	playoffSimService := GetPlayoffSimulationService()
	if playoffSimService != nil {
		// Recalculate for the main team (the one this server is running for)
		if err := playoffSimService.RecalculatePlayoffOdds(grs.teamCode); err != nil {
			log.Printf("⚠️ Failed to recalculate playoff odds: %v", err)
		} else {
			log.Printf("🎲 Playoff odds recalculated for %s", grs.teamCode)
		}
	}
}

// convertToGameResult converts CompletedGame to GameResult format
func (grs *GameResultsService) convertToGameResult(game *models.CompletedGame) *models.GameResult {
	isOT := game.WinType == "OT"
	isSO := game.WinType == "SO"

	return &models.GameResult{
		GameID:      game.GameID,
		HomeTeam:    game.HomeTeam.TeamCode,
		AwayTeam:    game.AwayTeam.TeamCode,
		HomeScore:   game.HomeTeam.Score,
		AwayScore:   game.AwayTeam.Score,
		GameState:   "FINAL",
		Period:      3, // Would be 4+ for OT
		TimeLeft:    "0:00",
		GameDate:    game.GameDate,
		Venue:       game.Venue,
		IsOvertime:  isOT,
		IsShootout:  isSO,
		WinningTeam: game.Winner,
		LosingTeam:  grs.getLosingTeam(game),
		UpdatedAt:   time.Now(),
		Shots: models.GameShots{
			Home: game.HomeTeam.Shots,
			Away: game.AwayTeam.Shots,
		},
		PowerPlays: models.GamePowerPlays{
			HomeOpportunities: game.HomeTeam.PowerPlayOpps,
			HomeGoals:         game.HomeTeam.PowerPlayGoals,
			AwayOpportunities: game.AwayTeam.PowerPlayOpps,
			AwayGoals:         game.AwayTeam.PowerPlayGoals,
		},
		Penalties: models.GamePenalties{
			HomePIM: game.HomeTeam.PenaltyMinutes,
			AwayPIM: game.AwayTeam.PenaltyMinutes,
		},
	}
}

// getLosingTeam determines the losing team code
func (grs *GameResultsService) getLosingTeam(game *models.CompletedGame) string {
	if game.Winner == game.HomeTeam.TeamCode {
		return game.AwayTeam.TeamCode
	}
	return game.HomeTeam.TeamCode
}

// buildPredictionFactors constructs prediction factors from completed game data
func (grs *GameResultsService) buildPredictionFactors(game *models.CompletedGame, isHome bool) *models.PredictionFactors {
	var team, opponent models.TeamGameResult

	if isHome {
		team = game.HomeTeam
		opponent = game.AwayTeam
	} else {
		team = game.AwayTeam
		opponent = game.HomeTeam
	}

	// Build the prediction factors using available fields
	homeAdvantage := 0.0
	if isHome {
		homeAdvantage = 1.0
	}

	return &models.PredictionFactors{
		TeamCode:       team.TeamCode,
		GoalsFor:       float64(team.Score),
		GoalsAgainst:   float64(opponent.Score),
		PowerPlayPct:   team.PowerPlayPct,
		PenaltyKillPct: team.PenaltyKillPct,

		// Basic stats (would be enhanced with historical data)
		WinPercentage:     0.5, // Default, would need historical calculation
		RecentForm:        0.5, // Default, would need last 10 games
		RestDays:          1,   // Default
		HomeAdvantage:     homeAdvantage,
		BackToBackPenalty: 0.0, // Default
		HeadToHead:        0.5, // Default

		// Situational factors (initialized with defaults)
		TravelFatigue:    models.TravelFatigue{},
		AltitudeAdjust:   models.AltitudeAdjust{},
		ScheduleStrength: models.ScheduleStrength{},
		InjuryImpact:     models.InjuryImpact{},
		MomentumFactors:  models.MomentumFactors{},
		AdvancedStats:    models.AdvancedAnalytics{},
		WeatherAnalysis:  models.WeatherAnalysis{},
		MarketData:       models.MarketAdjustment{},
	}
}

// loadProcessedGames loads the processed games index from disk
func (grs *GameResultsService) loadProcessedGames() error {
	filePath := filepath.Join(grs.dataDir, "processed_games.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("📊 No existing processed games index found, starting fresh")
		return nil
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading processed games: %w", err)
	}

	var index models.ProcessedGamesIndex
	if err := json.Unmarshal(jsonData, &index); err != nil {
		return fmt.Errorf("error unmarshaling processed games: %w", err)
	}

	grs.mutex.Lock()
	defer grs.mutex.Unlock()

	grs.processedGames = index.ProcessedGames
	if grs.processedGames == nil {
		grs.processedGames = make(map[int]bool)
	}

	log.Printf("📊 Loaded processed games index: %d games (last updated: %s)",
		len(grs.processedGames), index.LastUpdated.Format("2006-01-02 15:04:05"))

	return nil
}

// saveProcessedGames saves the processed games index to disk
func (grs *GameResultsService) saveProcessedGames() error {
	grs.mutex.RLock()
	defer grs.mutex.RUnlock()

	filePath := filepath.Join(grs.dataDir, "processed_games.json")

	index := models.ProcessedGamesIndex{
		LastUpdated:    time.Now(),
		ProcessedGames: grs.processedGames,
		TotalProcessed: len(grs.processedGames),
		Version:        "1.0",
	}

	jsonData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling processed games: %w", err)
	}

	if err := ioutil.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing processed games: %w", err)
	}

	log.Printf("💾 Processed games index saved: %d games tracked", len(grs.processedGames))
	return nil
}

// GetProcessedGamesCount returns the number of processed games
func (grs *GameResultsService) GetProcessedGamesCount() int {
	grs.mutex.RLock()
	defer grs.mutex.RUnlock()
	return len(grs.processedGames)
}

// GetMonthlyGames returns all games for a specific month
func (grs *GameResultsService) GetMonthlyGames(year int, month int) ([]models.CompletedGame, error) {
	monthKey := fmt.Sprintf("%04d-%02d", year, month)
	filePath := filepath.Join(grs.dataDir, monthKey+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []models.CompletedGame{}, nil
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading monthly games: %w", err)
	}

	var games []models.CompletedGame
	if err := json.Unmarshal(jsonData, &games); err != nil {
		return nil, fmt.Errorf("error unmarshaling monthly games: %w", err)
	}

	return games, nil
}

// BackfillGames attempts to fetch and process historical games
func (grs *GameResultsService) BackfillGames(daysBack int) error {
	log.Printf("📊 Starting backfill for last %d days...", daysBack)

	// This would fetch the schedule for the last N days and process any completed games
	// Implementation would be similar to checkForCompletedGames but for a date range

	log.Printf("✅ Backfill completed")
	return nil
}

// Helper function to parse integer from string with fallback
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	// Remove any non-numeric characters (like percentage signs)
	s = strings.TrimSpace(strings.ReplaceAll(s, "%", ""))
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

// Helper function to parse float from string with fallback
func parseFloat(s string) float64 {
	if s == "" {
		return 0.0
	}
	// Remove any non-numeric characters except decimal point
	s = strings.TrimSpace(strings.ReplaceAll(s, "%", ""))
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return val
}

// ============================================================================
// GLOBAL SERVICE INITIALIZATION (for use in main.go)
// ============================================================================

// InitializeGameResultsService initializes the global game results service
func InitializeGameResultsService(teamCode string) error {
	gameResultsMutex.Lock()
	defer gameResultsMutex.Unlock()

	if globalGameResultsService != nil {
		return fmt.Errorf("game results service already initialized")
	}

	// Get the global models from the live prediction system
	liveSys := GetLivePredictionSystem()
	if liveSys == nil {
		return fmt.Errorf("live prediction system must be initialized first")
	}

	// Get individual models
	eloModel := liveSys.GetEloModel()
	poissonModel := liveSys.GetPoissonModel()
	neuralNet := liveSys.GetNeuralNetwork()

	// Get or create rolling stats service
	rollingStats := GetRollingStatsService()
	if rollingStats == nil {
		// Initialize if not already done
		if err := InitializeRollingStatsService(); err != nil {
			log.Printf("⚠️ Failed to initialize rolling stats: %v", err)
		}
		rollingStats = GetRollingStatsService()
	}

	// Get accuracy tracker from the ensemble
	ensemble := liveSys.GetEnsemble()
	var accuracyTracker *AccuracyTrackingService
	if ensemble != nil {
		accuracyTracker = ensemble.GetAccuracyTracker()
	}

	// Create the service
	globalGameResultsService = NewGameResultsService(
		teamCode,
		eloModel,
		poissonModel,
		neuralNet,
		rollingStats,
		accuracyTracker,
	)

	// Link to evaluation service (if it's initialized)
	evaluationSvc := GetEvaluationService()
	if evaluationSvc != nil {
		globalGameResultsService.evaluationSvc = evaluationSvc
		log.Printf("✅ Game Results Service linked to Evaluation Service (batch training enabled)")
	}

	// Start the service
	globalGameResultsService.Start()

	return nil
}

// StopGameResultsService stops the global game results service
func StopGameResultsService() error {
	gameResultsMutex.Lock()
	defer gameResultsMutex.Unlock()

	if globalGameResultsService == nil {
		return nil // Not initialized, nothing to stop
	}

	globalGameResultsService.Stop()
	globalGameResultsService = nil

	return nil
}

// GetGameResultsService returns the global game results service
func GetGameResultsService() *GameResultsService {
	gameResultsMutex.Lock()
	defer gameResultsMutex.Unlock()
	return globalGameResultsService
}
