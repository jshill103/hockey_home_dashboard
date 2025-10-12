package models

import "time"

// TrainingMetrics tracks training frequency and effectiveness for a model
type TrainingMetrics struct {
	ModelName           string    `json:"modelName"`
	TrainingType        string    `json:"trainingType"` // "real-time", "batch", "manual"
	TotalTrainings      int       `json:"totalTrainings"`
	LastTraining        time.Time `json:"lastTraining"`
	GamesProcessed      int       `json:"gamesProcessed"`
	AvgBatchSize        float64   `json:"avgBatchSize"`
	AccuracyBefore      float64   `json:"accuracyBefore"`
	AccuracyAfter       float64   `json:"accuracyAfter"`
	AccuracyImprovement float64   `json:"accuracyImprovement"`
	TotalTrainingTime   float64   `json:"totalTrainingTime"` // seconds
	AvgTrainingTime     float64   `json:"avgTrainingTime"`   // seconds per training
}

// TrainingEvent represents a single training occurrence
type TrainingEvent struct {
	ModelName     string    `json:"modelName"`
	EventType     string    `json:"eventType"` // "batch", "game", "auto", "manual"
	Timestamp     time.Time `json:"timestamp"`
	GamesInBatch  int       `json:"gamesInBatch"`
	TrainingTime  float64   `json:"trainingTime"`  // seconds
	AccuracyDelta float64   `json:"accuracyDelta"` // change in accuracy
}

// SystemTrainingMetrics provides system-wide training statistics
type SystemTrainingMetrics struct {
	TotalGamesProcessed int                         `json:"totalGamesProcessed"`
	TotalTrainingEvents int                         `json:"totalTrainingEvents"`
	ModelMetrics        map[string]*TrainingMetrics `json:"modelMetrics"`
	LastUpdated         time.Time                   `json:"lastUpdated"`
	CurrentBatchSizes   map[string]int              `json:"currentBatchSizes"`   // Model -> current batch size
	TrainingFrequencies map[string]string           `json:"trainingFrequencies"` // Model -> "every X games"
}
