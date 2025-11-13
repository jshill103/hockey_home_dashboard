package models

import (
	"time"
)

// PredictionAccuracyRecord tracks the accuracy of a single prediction
type PredictionAccuracyRecord struct {
	GameID          int       `json:"gameId"`
	GameDate        time.Time `json:"gameDate"`
	HomeTeam        string    `json:"homeTeam"`
	AwayTeam        string    `json:"awayTeam"`
	PredictedWinner string    `json:"predictedWinner"`
	ActualWinner    string    `json:"actualWinner"`
	IsCorrect       bool      `json:"isCorrect"`
	Confidence      float64   `json:"confidence"` // How confident was the prediction (0-1)

	// Predicted probabilities
	PredictedHomeWinProb float64 `json:"predictedHomeWinProb"`
	PredictedAwayWinProb float64 `json:"predictedAwayWinProb"`
	PredictedTieProb     float64 `json:"predictedTieProb"`

	// Actual outcome
	HomeScore int    `json:"homeScore"`
	AwayScore int    `json:"awayScore"`
	GameType  string `json:"gameType"` // "regulation", "overtime", "shootout"

	// Game characteristics
	IsUpset          bool    `json:"isUpset"`          // Underdog won
	IsBlowout        bool    `json:"isBlowout"`        // Margin > 3 goals
	IsCloseGame      bool    `json:"isCloseGame"`      // Margin = 1 goal
	WinningMargin    int     `json:"winningMargin"`    // Goal differential
	TotalGoals       int     `json:"totalGoals"`       // Total goals scored
	IsHighScoring    bool    `json:"isHighScoring"`    // Total goals > 7
	IsLowScoring     bool    `json:"isLowScoring"`     // Total goals < 4
	IsPlayoffGame    bool    `json:"isPlayoffGame"`    // Regular season or playoff
	IsRivalryGame    bool    `json:"isRivalryGame"`    // Rivalry matchup
	IsDivisionGame   bool    `json:"isDivisionGame"`   // Division game
	HomeBackToBack   bool    `json:"homeBackToBack"`   // Home team on B2B
	AwayBackToBack   bool    `json:"awayBackToBack"`   // Away team on B2B
	PredictionError  float64 `json:"predictionError"`  // Absolute probability error
	CalibrationScore float64 `json:"calibrationScore"` // How well-calibrated the prediction was

	// Model-specific predictions
	ModelPredictions map[string]*ModelPredictionResult `json:"modelPredictions"` // Individual model results

	// Metadata
	PredictionTime time.Time `json:"predictionTime"`
	Season         string    `json:"season"` // e.g., "2025-26"
}

// ModelPredictionResult stores individual model prediction and accuracy
type ModelPredictionResult struct {
	ModelName        string  `json:"modelName"`
	PredictedWinner  string  `json:"predictedWinner"`
	HomeWinProb      float64 `json:"homeWinProb"`
	AwayWinProb      float64 `json:"awayWinProb"`
	TieProb          float64 `json:"tieProb"`
	IsCorrect        bool    `json:"isCorrect"`
	Confidence       float64 `json:"confidence"`
	PredictionError  float64 `json:"predictionError"`
	ModelWeight      float64 `json:"modelWeight"` // Weight used in ensemble
	ContributionSign int     `json:"contributionSign"` // +1 if helped, -1 if hurt, 0 if neutral
}

// AccuracySummary provides aggregate accuracy statistics
type AccuracySummary struct {
	TotalPredictions      int     `json:"totalPredictions"`
	CorrectPredictions    int     `json:"correctPredictions"`
	IncorrectPredictions  int     `json:"incorrectPredictions"`
	OverallAccuracy       float64 `json:"overallAccuracy"`       // Percentage correct
	AverageConfidence     float64 `json:"averageConfidence"`     // Average confidence of predictions
	AveragePredictionError float64 `json:"averagePredictionError"` // Average absolute error
	BrierScore            float64 `json:"brierScore"`            // Probabilistic accuracy metric
	LogLoss               float64 `json:"logLoss"`               // Log loss score

	// Breakdown by game type
	RegulationAccuracy float64 `json:"regulationAccuracy"`
	OvertimeAccuracy   float64 `json:"overtimeAccuracy"`
	ShootoutAccuracy   float64 `json:"shootoutAccuracy"`

	// Breakdown by game characteristics
	UpsetPredictionRate     float64 `json:"upsetPredictionRate"`     // % of upsets predicted
	UpsetCatchRate          float64 `json:"upsetCatchRate"`          // % of actual upsets caught
	BlowoutAccuracy         float64 `json:"blowoutAccuracy"`         // Accuracy in blowouts
	CloseGameAccuracy       float64 `json:"closeGameAccuracy"`       // Accuracy in 1-goal games
	HighScoringAccuracy     float64 `json:"highScoringAccuracy"`     // Accuracy in high-scoring games
	LowScoringAccuracy      float64 `json:"lowScoringAccuracy"`      // Accuracy in low-scoring games
	RivalryGameAccuracy     float64 `json:"rivalryGameAccuracy"`     // Accuracy in rivalry games
	DivisionGameAccuracy    float64 `json:"divisionGameAccuracy"`    // Accuracy in division games
	BackToBackAccuracy      float64 `json:"backToBackAccuracy"`      // Accuracy when team on B2B
	PlayoffGameAccuracy     float64 `json:"playoffGameAccuracy"`     // Accuracy in playoff games

	// Calibration metrics
	HighConfidenceAccuracy   float64 `json:"highConfidenceAccuracy"`   // Accuracy when confidence > 70%
	MediumConfidenceAccuracy float64 `json:"mediumConfidenceAccuracy"` // Accuracy when confidence 50-70%
	LowConfidenceAccuracy    float64 `json:"lowConfidenceAccuracy"`    // Accuracy when confidence < 50%

	// Time-based metrics
	Last10GamesAccuracy float64 `json:"last10GamesAccuracy"`
	Last30GamesAccuracy float64 `json:"last30GamesAccuracy"`
	Last50GamesAccuracy float64 `json:"last50GamesAccuracy"`

	// Model-specific accuracy
	ModelAccuracies map[string]float64 `json:"modelAccuracies"` // Accuracy per model

	// Metadata
	StartDate  time.Time `json:"startDate"`
	EndDate    time.Time `json:"endDate"`
	Season     string    `json:"season"`
	LastUpdate time.Time `json:"lastUpdate"`
}

// FeatureImportance tracks which features contribute most to predictions
type FeatureImportance struct {
	FeatureName         string  `json:"featureName"`
	ImportanceScore     float64 `json:"importanceScore"`     // 0-1, higher = more important
	CorrelationWithWin  float64 `json:"correlationWithWin"`  // Correlation with correct predictions
	UsageFrequency      float64 `json:"usageFrequency"`      // How often this feature is non-zero
	AverageValue        float64 `json:"averageValue"`        // Average value across all games
	ValueRange          float64 `json:"valueRange"`          // Max - Min
	NeuralNetWeight     float64 `json:"neuralNetWeight"`     // Average absolute weight in NN
	GradientBoostGain   float64 `json:"gradientBoostGain"`   // Total gain in GB trees
	RandomForestGini    float64 `json:"randomForestGini"`    // Gini importance in RF
	ContributesToUpsets bool    `json:"contributesToUpsets"` // Does this feature help predict upsets?
	LastUpdated         time.Time `json:"lastUpdated"`
}

// ErrorPattern identifies common prediction failure patterns
type ErrorPattern struct {
	PatternType         string  `json:"patternType"` // "upset", "blowout_miss", "overtime_wrong", etc.
	Frequency           int     `json:"frequency"`   // How often this pattern occurs
	PercentageOfErrors  float64 `json:"percentageOfErrors"` // % of total errors
	CommonCharacteristics []string `json:"commonCharacteristics"` // What these games have in common
	AffectedModels      []string `json:"affectedModels"` // Which models struggle with this pattern
	PotentialFix        string   `json:"potentialFix"` // Suggested improvement
	ExampleGameIDs      []int    `json:"exampleGameIds"` // Sample games exhibiting this pattern
	Severity            string   `json:"severity"` // "low", "medium", "high"
	LastSeen            time.Time `json:"lastSeen"`
}

// DailyAccuracyReport provides a daily summary of prediction performance
type DailyAccuracyReport struct {
	Date                 time.Time          `json:"date"`
	TotalGames           int                `json:"totalGames"`
	CorrectPredictions   int                `json:"correctPredictions"`
	DailyAccuracy        float64            `json:"dailyAccuracy"`
	AverageConfidence    float64            `json:"averageConfidence"`
	BestModel            string             `json:"bestModel"`
	WorstModel           string             `json:"worstModel"`
	NotableUpsets        []string           `json:"notableUpsets"` // Teams that upset
	PredictionErrors     []ErrorPattern     `json:"predictionErrors"`
	TopPerformingFeatures []string          `json:"topPerformingFeatures"`
	Summary              string             `json:"summary"` // Natural language summary
	ModelAccuracies      map[string]float64 `json:"modelAccuracies"`
}

