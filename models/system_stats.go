package models

import "time"

// SystemStats tracks overall system statistics
type SystemStats struct {
	BackfillStats    BackfillStats   `json:"backfillStats"`
	PredictionStats  PredictionStats `json:"predictionStats"`
	LastUpdated      time.Time       `json:"lastUpdated"`
	SystemUptime     time.Duration   `json:"systemUptime"`
	TotalAPIRequests int             `json:"totalApiRequests"`
}

// BackfillStats tracks historical game processing
type BackfillStats struct {
	TotalGamesProcessed  int       `json:"totalGamesProcessed"`
	PlayByPlayGames      int       `json:"playByPlayGames"`
	ShiftDataGames       int       `json:"shiftDataGames"`
	GameSummaryGames     int       `json:"gameSummaryGames"`
	FailedGames          int       `json:"failedGames"`
	LastBackfillTime     time.Time `json:"lastBackfillTime"`
	ProcessingTimeAvg    float64   `json:"processingTimeAvgMs"`
	TotalEventsProcessed int       `json:"totalEventsProcessed"`
}

// PredictionStats tracks model accuracy
type PredictionStats struct {
	TotalPredictions     int                       `json:"totalPredictions"`
	CorrectPredictions   int                       `json:"correctPredictions"`
	OverallAccuracy      float64                   `json:"overallAccuracy"`
	ModelAccuracy        map[string]*ModelAccuracy `json:"modelAccuracy"`
	LastPredictionTime   time.Time                 `json:"lastPredictionTime"`
	BestPerformingModel  string                    `json:"bestPerformingModel"`
	WorstPerformingModel string                    `json:"worstPerformingModel"`
}

// ModelAccuracy tracks individual model performance
type ModelAccuracy struct {
	ModelName          string    `json:"modelName"`
	TotalPredictions   int       `json:"totalPredictions"`
	CorrectPredictions int       `json:"correctPredictions"`
	Accuracy           float64   `json:"accuracy"`
	AvgConfidence      float64   `json:"avgConfidence"`
	LastUpdated        time.Time `json:"lastUpdated"`
}

// PredictionRecord stores a single prediction for verification
type PredictionRecord struct {
	GameID           int                              `json:"gameId"`
	GameDate         time.Time                        `json:"gameDate"`
	HomeTeam         string                           `json:"homeTeam"`
	AwayTeam         string                           `json:"awayTeam"`
	PredictedWinner  string                           `json:"predictedWinner"`
	PredictedScore   string                           `json:"predictedScore"`
	ActualWinner     string                           `json:"actualWinner,omitempty"`
	ActualScore      string                           `json:"actualScore,omitempty"`
	WasCorrect       *bool                            `json:"wasCorrect,omitempty"`
	ConfidenceLevel  float64                          `json:"confidenceLevel"`
	ModelPredictions map[string]ModelPredictionRecord `json:"modelPredictions"`
	Verified         bool                             `json:"verified"`
	VerifiedAt       time.Time                        `json:"verifiedAt,omitempty"`
}

// ModelPredictionRecord stores individual model predictions
type ModelPredictionRecord struct {
	Winner     string  `json:"winner"`
	Score      string  `json:"score"`
	Confidence float64 `json:"confidence"`
	WasCorrect *bool   `json:"wasCorrect,omitempty"`
}
