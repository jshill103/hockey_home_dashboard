package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// TrainingMetricsService tracks training frequency and effectiveness
type TrainingMetricsService struct {
	dataDir         string
	modelMetrics    map[string]*models.TrainingMetrics
	trainingHistory []models.TrainingEvent
	mutex           sync.RWMutex
}

var (
	trainingMetricsService     *TrainingMetricsService
	trainingMetricsServiceOnce sync.Once
)

// InitTrainingMetricsService initializes the singleton
func InitTrainingMetricsService() *TrainingMetricsService {
	trainingMetricsServiceOnce.Do(func() {
		dataDir := "data/metrics"
		os.MkdirAll(dataDir, 0755)

		trainingMetricsService = &TrainingMetricsService{
			dataDir:         dataDir,
			modelMetrics:    make(map[string]*models.TrainingMetrics),
			trainingHistory: []models.TrainingEvent{},
		}

		// Load existing metrics
		if err := trainingMetricsService.loadMetrics(); err != nil {
			log.Printf("⚠️ Could not load training metrics: %v (starting fresh)", err)
		} else {
			log.Printf("📊 Training Metrics Service initialized with %d models", len(trainingMetricsService.modelMetrics))
		}
	})
	return trainingMetricsService
}

// GetTrainingMetricsService returns the singleton instance
func GetTrainingMetricsService() *TrainingMetricsService {
	return trainingMetricsService
}

// RecordTraining records a training event
func (tms *TrainingMetricsService) RecordTraining(modelName, eventType string, gamesInBatch int, trainingTime float64, accuracyDelta float64) {
	tms.mutex.Lock()
	defer tms.mutex.Unlock()

	// Get or create model metrics
	if tms.modelMetrics[modelName] == nil {
		tms.modelMetrics[modelName] = &models.TrainingMetrics{
			ModelName:      modelName,
			TrainingType:   tms.getTrainingType(modelName),
			TotalTrainings: 0,
			LastTraining:   time.Time{},
			GamesProcessed: 0,
		}
	}

	metrics := tms.modelMetrics[modelName]

	// Update metrics
	metrics.TotalTrainings++
	metrics.LastTraining = time.Now()
	metrics.GamesProcessed += gamesInBatch
	metrics.TotalTrainingTime += trainingTime
	metrics.AvgTrainingTime = metrics.TotalTrainingTime / float64(metrics.TotalTrainings)

	// Update batch size average
	if eventType == "batch" {
		totalBatches := float64(metrics.TotalTrainings)
		metrics.AvgBatchSize = (metrics.AvgBatchSize*(totalBatches-1) + float64(gamesInBatch)) / totalBatches
	}

	// Update accuracy
	if accuracyDelta != 0 {
		metrics.AccuracyBefore = metrics.AccuracyAfter
		metrics.AccuracyAfter = metrics.AccuracyBefore + accuracyDelta
		metrics.AccuracyImprovement += accuracyDelta
	}

	// Record event in history
	event := models.TrainingEvent{
		ModelName:     modelName,
		EventType:     eventType,
		Timestamp:     time.Now(),
		GamesInBatch:  gamesInBatch,
		TrainingTime:  trainingTime,
		AccuracyDelta: accuracyDelta,
	}
	tms.trainingHistory = append(tms.trainingHistory, event)

	// Keep only last 1000 events
	if len(tms.trainingHistory) > 1000 {
		tms.trainingHistory = tms.trainingHistory[len(tms.trainingHistory)-1000:]
	}

	// Save metrics
	go tms.saveMetrics()

	log.Printf("📊 Training recorded: %s (%s) - %d games in %.2fs",
		modelName, eventType, gamesInBatch, trainingTime)
}

// GetSystemMetrics returns system-wide training metrics
func (tms *TrainingMetricsService) GetSystemMetrics() *models.SystemTrainingMetrics {
	tms.mutex.RLock()
	defer tms.mutex.RUnlock()

	totalGames := 0
	totalEvents := 0
	for _, metrics := range tms.modelMetrics {
		totalGames += metrics.GamesProcessed
		totalEvents += metrics.TotalTrainings
	}

	// Get current batch sizes
	batchSizes := make(map[string]int)
	frequencies := make(map[string]string)

	for modelName, metrics := range tms.modelMetrics {
		switch metrics.TrainingType {
		case "real-time":
			batchSizes[modelName] = 1
			frequencies[modelName] = "after every game"
		case "batch":
			batchSizes[modelName] = int(metrics.AvgBatchSize)
			frequencies[modelName] = fmt.Sprintf("every %d games", int(metrics.AvgBatchSize))
		case "manual":
			batchSizes[modelName] = 0
			frequencies[modelName] = "manual trigger"
		}
	}

	return &models.SystemTrainingMetrics{
		TotalGamesProcessed: totalGames,
		TotalTrainingEvents: totalEvents,
		ModelMetrics:        tms.modelMetrics,
		LastUpdated:         time.Now(),
		CurrentBatchSizes:   batchSizes,
		TrainingFrequencies: frequencies,
	}
}

// GetModelMetrics returns metrics for a specific model
func (tms *TrainingMetricsService) GetModelMetrics(modelName string) *models.TrainingMetrics {
	tms.mutex.RLock()
	defer tms.mutex.RUnlock()

	return tms.modelMetrics[modelName]
}

// GetRecentTrainingEvents returns recent training events
func (tms *TrainingMetricsService) GetRecentTrainingEvents(limit int) []models.TrainingEvent {
	tms.mutex.RLock()
	defer tms.mutex.RUnlock()

	if limit > len(tms.trainingHistory) {
		limit = len(tms.trainingHistory)
	}

	// Return last N events
	start := len(tms.trainingHistory) - limit
	if start < 0 {
		start = 0
	}

	events := make([]models.TrainingEvent, limit)
	copy(events, tms.trainingHistory[start:])

	return events
}

// getTrainingType determines the training type based on model name
func (tms *TrainingMetricsService) getTrainingType(modelName string) string {
	switch modelName {
	case "Elo Rating", "Poisson Regression", "Rolling Stats", "Matchup History":
		return "real-time"
	case "Neural Network", "Gradient Boosting", "LSTM", "Random Forest":
		return "batch"
	case "Meta-Learner":
		return "auto"
	default:
		return "unknown"
	}
}

// SaveMetrics persists training metrics to disk (public for graceful shutdown)
func (tms *TrainingMetricsService) SaveMetrics() error {
	return tms.saveMetrics()
}

// saveMetrics persists training metrics to disk (internal implementation)
func (tms *TrainingMetricsService) saveMetrics() error {
	tms.mutex.RLock()
	defer tms.mutex.RUnlock()

	data, err := json.MarshalIndent(tms.modelMetrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal training metrics: %w", err)
	}

	filename := filepath.Join(tms.dataDir, "training_metrics.json")
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write training metrics: %w", err)
	}

	return nil
}

// loadMetrics loads training metrics from disk
func (tms *TrainingMetricsService) loadMetrics() error {
	filename := filepath.Join(tms.dataDir, "training_metrics.json")

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No metrics file yet
		}
		return fmt.Errorf("failed to read training metrics: %w", err)
	}

	err = json.Unmarshal(data, &tms.modelMetrics)
	if err != nil {
		return fmt.Errorf("failed to unmarshal training metrics: %w", err)
	}

	return nil
}
