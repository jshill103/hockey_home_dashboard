package services

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// SystemStatsService tracks system-wide statistics
type SystemStatsService struct {
	stats           models.SystemStats
	predictions     map[int]*models.PredictionRecord // keyed by gameID
	mu              sync.RWMutex
	statsFile       string
	predictionsFile string
	startTime       time.Time
}

var (
	systemStatsService *SystemStatsService
	systemStatsOnce    sync.Once
)

// GetSystemStatsService returns the global singleton instance
func GetSystemStatsService() *SystemStatsService {
	return systemStatsService
}

// NewSystemStatsService creates a new system statistics service
func NewSystemStatsService() *SystemStatsService {
	systemStatsOnce.Do(func() {
		systemStatsService = &SystemStatsService{
			stats: models.SystemStats{
				BackfillStats: models.BackfillStats{},
				PredictionStats: models.PredictionStats{
					ModelAccuracy: make(map[string]*models.ModelAccuracy),
				},
				LastUpdated: time.Now(),
			},
			predictions:     make(map[int]*models.PredictionRecord),
			statsFile:       "data/metrics/system_stats.json",
			predictionsFile: "data/metrics/prediction_records.json",
			startTime:       time.Now(),
		}

		// Load existing data
		systemStatsService.loadStats()
		systemStatsService.loadPredictions()

		// Initialize model accuracy tracking for all models
		systemStatsService.initializeModelTracking()

		fmt.Println("📊 System Stats Service initialized")
	})
	return systemStatsService
}

// initializeModelTracking ensures all models are tracked
func (s *SystemStatsService) initializeModelTracking() {
	modelNames := []string{
		"Enhanced Statistical",
		"Bayesian Inference",
		"Monte Carlo Simulation",
		"Elo Rating",
		"Poisson Regression",
		"Neural Network",
		"Gradient Boosting",
		"LSTM",
		"Random Forest",
	}

	for _, modelName := range modelNames {
		if _, exists := s.stats.PredictionStats.ModelAccuracy[modelName]; !exists {
			s.stats.PredictionStats.ModelAccuracy[modelName] = &models.ModelAccuracy{
				ModelName:   modelName,
				LastUpdated: time.Now(),
			}
		}
	}
}

// RecordBackfillGame records a successfully backfilled game
func (s *SystemStatsService) RecordBackfillGame(gameType string, eventsProcessed int, processingTime time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.BackfillStats.TotalGamesProcessed++
	s.stats.BackfillStats.TotalEventsProcessed += eventsProcessed
	s.stats.BackfillStats.LastBackfillTime = time.Now()

	// Update processing time average
	if s.stats.BackfillStats.ProcessingTimeAvg == 0 {
		s.stats.BackfillStats.ProcessingTimeAvg = float64(processingTime.Milliseconds())
	} else {
		s.stats.BackfillStats.ProcessingTimeAvg = (s.stats.BackfillStats.ProcessingTimeAvg + float64(processingTime.Milliseconds())) / 2
	}

	// Track by type
	switch gameType {
	case "play-by-play":
		s.stats.BackfillStats.PlayByPlayGames++
	case "shifts":
		s.stats.BackfillStats.ShiftDataGames++
	case "summary":
		s.stats.BackfillStats.GameSummaryGames++
	}

	s.stats.LastUpdated = time.Now()
	s.saveStats()

	fmt.Printf("📊 Backfill recorded: %s game, %d events, %.2fms\n", gameType, eventsProcessed, processingTime.Seconds()*1000)
}

// RecordBackfillFailure records a failed backfill attempt
func (s *SystemStatsService) RecordBackfillFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.BackfillStats.FailedGames++
	s.stats.LastUpdated = time.Now()
	s.saveStats()
}

// RecordPrediction stores a prediction for future verification
func (s *SystemStatsService) RecordPrediction(
	gameID int,
	gameDate time.Time,
	homeTeam, awayTeam string,
	predictedWinner, predictedScore string,
	confidence float64,
	modelPredictions map[string]models.ModelPredictionRecord,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := &models.PredictionRecord{
		GameID:           gameID,
		GameDate:         gameDate,
		HomeTeam:         homeTeam,
		AwayTeam:         awayTeam,
		PredictedWinner:  predictedWinner,
		PredictedScore:   predictedScore,
		ConfidenceLevel:  confidence,
		ModelPredictions: modelPredictions,
		Verified:         false,
	}

	s.predictions[gameID] = record
	s.stats.PredictionStats.TotalPredictions++
	s.stats.PredictionStats.LastPredictionTime = time.Now()
	s.stats.LastUpdated = time.Now()

	s.savePredictions()
	s.saveStats()

	fmt.Printf("🎯 Prediction recorded for game %d: %s vs %s (Winner: %s)\n", gameID, awayTeam, homeTeam, predictedWinner)
}

// VerifyPrediction verifies a prediction against actual game results
func (s *SystemStatsService) VerifyPrediction(gameID int, actualWinner, actualScore string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.predictions[gameID]
	if !exists || record.Verified {
		return
	}

	// Update record with actual results
	record.ActualWinner = actualWinner
	record.ActualScore = actualScore
	record.Verified = true
	record.VerifiedAt = time.Now()

	// Check if prediction was correct
	wasCorrect := record.PredictedWinner == actualWinner
	record.WasCorrect = &wasCorrect

	if wasCorrect {
		s.stats.PredictionStats.CorrectPredictions++
	}

	// Calculate overall accuracy
	if s.stats.PredictionStats.TotalPredictions > 0 {
		s.stats.PredictionStats.OverallAccuracy = float64(s.stats.PredictionStats.CorrectPredictions) / float64(s.stats.PredictionStats.TotalPredictions) * 100
	}

	// Update individual model accuracy
	for modelName, modelPred := range record.ModelPredictions {
		modelCorrect := modelPred.Winner == actualWinner
		modelPred.WasCorrect = &modelCorrect
		record.ModelPredictions[modelName] = modelPred

		// Update model stats
		if modelStats, exists := s.stats.PredictionStats.ModelAccuracy[modelName]; exists {
			modelStats.TotalPredictions++
			if modelCorrect {
				modelStats.CorrectPredictions++
			}
			if modelStats.TotalPredictions > 0 {
				modelStats.Accuracy = float64(modelStats.CorrectPredictions) / float64(modelStats.TotalPredictions) * 100
			}
			// Update average confidence
			if modelStats.AvgConfidence == 0 {
				modelStats.AvgConfidence = modelPred.Confidence
			} else {
				modelStats.AvgConfidence = (modelStats.AvgConfidence + modelPred.Confidence) / 2
			}
			modelStats.LastUpdated = time.Now()
		}
	}

	// Determine best and worst models
	s.updateBestWorstModels()

	s.stats.LastUpdated = time.Now()
	s.savePredictions()
	s.saveStats()

	fmt.Printf("✅ Prediction verified for game %d: %s (Correct: %t)\n", gameID, actualWinner, wasCorrect)
}

// updateBestWorstModels updates the best and worst performing models
func (s *SystemStatsService) updateBestWorstModels() {
	var bestModel, worstModel string
	var bestAccuracy, worstAccuracy float64 = 0, 100

	for modelName, stats := range s.stats.PredictionStats.ModelAccuracy {
		if stats.TotalPredictions < 3 {
			continue // Skip models with insufficient data
		}

		if stats.Accuracy > bestAccuracy {
			bestAccuracy = stats.Accuracy
			bestModel = modelName
		}

		if stats.Accuracy < worstAccuracy {
			worstAccuracy = stats.Accuracy
			worstModel = modelName
		}
	}

	s.stats.PredictionStats.BestPerformingModel = bestModel
	s.stats.PredictionStats.WorstPerformingModel = worstModel
}

// IncrementAPIRequest increments the API request counter
func (s *SystemStatsService) IncrementAPIRequest() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.TotalAPIRequests++
	s.stats.LastUpdated = time.Now()

	// Save stats every 100 requests to avoid too frequent writes
	if s.stats.TotalAPIRequests%100 == 0 {
		s.saveStats()
	}
}

// GetStats returns current system statistics
func (s *SystemStatsService) GetStats() models.SystemStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := s.stats
	stats.SystemUptime = time.Since(s.startTime)
	return stats
}

// GetPredictionRecords returns all prediction records
func (s *SystemStatsService) GetPredictionRecords() []*models.PredictionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*models.PredictionRecord, 0, len(s.predictions))
	for _, record := range s.predictions {
		records = append(records, record)
	}

	return records
}

// saveStats saves statistics to disk
func (s *SystemStatsService) saveStats() {
	data, err := json.MarshalIndent(s.stats, "", "  ")
	if err != nil {
		fmt.Printf("❌ Error marshaling system stats: %v\n", err)
		return
	}

	if err := os.WriteFile(s.statsFile, data, 0644); err != nil {
		fmt.Printf("❌ Error saving system stats: %v\n", err)
		return
	}
}

// loadStats loads statistics from disk
func (s *SystemStatsService) loadStats() {
	data, err := os.ReadFile(s.statsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("⚠️ Error loading system stats: %v\n", err)
		}
		return
	}

	if err := json.Unmarshal(data, &s.stats); err != nil {
		fmt.Printf("❌ Error unmarshaling system stats: %v\n", err)
		return
	}

	// Ensure map is initialized
	if s.stats.PredictionStats.ModelAccuracy == nil {
		s.stats.PredictionStats.ModelAccuracy = make(map[string]*models.ModelAccuracy)
	}

	fmt.Printf("📊 Loaded system stats: %d games backfilled, %d predictions made\n",
		s.stats.BackfillStats.TotalGamesProcessed,
		s.stats.PredictionStats.TotalPredictions)
}

// savePredictions saves prediction records to disk
func (s *SystemStatsService) savePredictions() {
	data, err := json.MarshalIndent(s.predictions, "", "  ")
	if err != nil {
		fmt.Printf("❌ Error marshaling prediction records: %v\n", err)
		return
	}

	if err := os.WriteFile(s.predictionsFile, data, 0644); err != nil {
		fmt.Printf("❌ Error saving prediction records: %v\n", err)
		return
	}
}

// loadPredictions loads prediction records from disk
func (s *SystemStatsService) loadPredictions() {
	data, err := os.ReadFile(s.predictionsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("⚠️ Error loading prediction records: %v\n", err)
		}
		return
	}

	if err := json.Unmarshal(data, &s.predictions); err != nil {
		fmt.Printf("❌ Error unmarshaling prediction records: %v\n", err)
		return
	}

	fmt.Printf("🎯 Loaded %d prediction records\n", len(s.predictions))
}
