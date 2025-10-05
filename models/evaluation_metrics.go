package models

import "time"

// ModelEvaluationMetrics tracks performance metrics for a prediction model
type ModelEvaluationMetrics struct {
	ModelName string `json:"modelName"`

	// Overall Performance
	TotalPredictions   int     `json:"totalPredictions"`
	CorrectPredictions int     `json:"correctPredictions"`
	Accuracy           float64 `json:"accuracy"` // Correct / Total

	// Classification Metrics
	TruePositives  int `json:"truePositives"`  // Predicted win, actual win
	TrueNegatives  int `json:"trueNegatives"`  // Predicted loss, actual loss
	FalsePositives int `json:"falsePositives"` // Predicted win, actual loss
	FalseNegatives int `json:"falseNegatives"` // Predicted loss, actual win

	Precision float64 `json:"precision"` // TP / (TP + FP)
	Recall    float64 `json:"recall"`    // TP / (TP + FN)
	F1Score   float64 `json:"f1Score"`   // 2 * (P * R) / (P + R)

	// Probability Calibration
	BrierScore float64 `json:"brierScore"` // Mean squared error of probabilities
	LogLoss    float64 `json:"logLoss"`    // Cross-entropy loss

	// Score Prediction
	MAE  float64 `json:"mae"`  // Mean Absolute Error (goals)
	RMSE float64 `json:"rmse"` // Root Mean Squared Error (goals)

	// Confidence Analysis
	AvgConfidence    float64 `json:"avgConfidence"`    // Average prediction confidence
	CalibrationError float64 `json:"calibrationError"` // Difference between confidence and accuracy

	// Context-Specific Performance
	HomeAccuracy   float64 `json:"homeAccuracy"`   // Accuracy for home predictions
	AwayAccuracy   float64 `json:"awayAccuracy"`   // Accuracy for away predictions
	UpsetDetection float64 `json:"upsetDetection"` // Accuracy predicting upsets

	// Time-Based Performance
	Last10Accuracy float64 `json:"last10Accuracy"` // Accuracy on last 10 predictions
	Last30Accuracy float64 `json:"last30Accuracy"` // Accuracy on last 30 predictions

	// Training Progress
	TrainingGames   int       `json:"trainingGames"`   // Games used for training
	ValidationGames int       `json:"validationGames"` // Games used for validation
	TestGames       int       `json:"testGames"`       // Games used for testing
	LastEvaluated   time.Time `json:"lastEvaluated"`
}

// PredictionOutcome represents the result of a prediction
type PredictionOutcome struct {
	GameID          int       `json:"gameId"`
	PredictedWinner string    `json:"predictedWinner"`
	ActualWinner    string    `json:"actualWinner"`
	PredictedScore  string    `json:"predictedScore"`
	ActualScore     string    `json:"actualScore"`
	WinProbability  float64   `json:"winProbability"`
	Confidence      float64   `json:"confidence"`
	IsCorrect       bool      `json:"isCorrect"`
	IsUpset         bool      `json:"isUpset"`        // Underdog won
	PredictedUpset  bool      `json:"predictedUpset"` // Predicted underdog to win
	HomeTeam        string    `json:"homeTeam"`
	AwayTeam        string    `json:"awayTeam"`
	PredictionTime  time.Time `json:"predictionTime"`
	GameTime        time.Time `json:"gameTime"`
}

// EnsembleMetrics tracks overall ensemble performance
type EnsembleMetrics struct {
	Timestamp           time.Time                          `json:"timestamp"`
	ModelMetrics        map[string]*ModelEvaluationMetrics `json:"modelMetrics"`
	EnsembleAccuracy    float64                            `json:"ensembleAccuracy"`
	EnsembleBrierScore  float64                            `json:"ensembleBrierScore"`
	TotalGamesEvaluated int                                `json:"totalGamesEvaluated"`

	// Model Comparison
	BestModel     string  `json:"bestModel"`
	BestAccuracy  float64 `json:"bestAccuracy"`
	WorstModel    string  `json:"worstModel"`
	WorstAccuracy float64 `json:"worstAccuracy"`

	// Confidence vs Accuracy
	HighConfidenceAccuracy float64 `json:"highConfidenceAccuracy"` // Accuracy when confidence > 75%
	LowConfidenceAccuracy  float64 `json:"lowConfidenceAccuracy"`  // Accuracy when confidence < 60%

	Version string `json:"version"`
}

// TrainTestSplit represents data split for validation
type TrainTestSplit struct {
	TrainingSet   []CompletedGame `json:"trainingSet"`
	ValidationSet []CompletedGame `json:"validationSet"`
	TestSet       []CompletedGame `json:"testSet"`

	TrainSize      int `json:"trainSize"`
	ValidationSize int `json:"validationSize"`
	TestSize       int `json:"testSize"`

	SplitRatio  []float64 `json:"splitRatio"`  // e.g., [0.7, 0.15, 0.15]
	SplitMethod string    `json:"splitMethod"` // "temporal", "random", "stratified"
	CreatedAt   time.Time `json:"createdAt"`
}

// ConfusionMatrix represents prediction outcomes
type ConfusionMatrix struct {
	TruePositives  int `json:"truePositives"`
	TrueNegatives  int `json:"trueNegatives"`
	FalsePositives int `json:"falsePositives"`
	FalseNegatives int `json:"falseNegatives"`
}

// CalculateMetrics computes derived metrics from confusion matrix
func (cm *ConfusionMatrix) CalculateMetrics() (precision, recall, f1, accuracy float64) {
	total := float64(cm.TruePositives + cm.TrueNegatives + cm.FalsePositives + cm.FalseNegatives)

	if total == 0 {
		return 0, 0, 0, 0
	}

	// Accuracy = (TP + TN) / Total
	accuracy = float64(cm.TruePositives+cm.TrueNegatives) / total

	// Precision = TP / (TP + FP)
	if cm.TruePositives+cm.FalsePositives > 0 {
		precision = float64(cm.TruePositives) / float64(cm.TruePositives+cm.FalsePositives)
	}

	// Recall = TP / (TP + FN)
	if cm.TruePositives+cm.FalseNegatives > 0 {
		recall = float64(cm.TruePositives) / float64(cm.TruePositives+cm.FalseNegatives)
	}

	// F1 = 2 * (Precision * Recall) / (Precision + Recall)
	if precision+recall > 0 {
		f1 = 2 * (precision * recall) / (precision + recall)
	}

	return precision, recall, f1, accuracy
}
