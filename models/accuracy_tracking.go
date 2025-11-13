package models

import (
	"time"
)

// DataQualityScore represents the quality assessment of prediction data
type DataQualityScore struct {
	OverallScore      float64                  `json:"overallScore"`      // 0-100 overall data quality
	FreshnessScore    float64                  `json:"freshnessScore"`    // How recent the data is
	CompletenessScore float64                  `json:"completenessScore"` // How complete the data is
	AccuracyScore     float64                  `json:"accuracyScore"`     // How accurate the data appears
	ConsistencyScore  float64                  `json:"consistencyScore"`  // How consistent across sources
	ReliabilityScore  float64                  `json:"reliabilityScore"`  // Source reliability
	CategoryScores    map[string]float64       `json:"categoryScores"`    // Scores by data category
	QualityFactors    []DataQualityFactor      `json:"qualityFactors"`    // Detailed quality factors
	ConfidenceImpact  float64                  `json:"confidenceImpact"`  // Impact on prediction confidence
	LastAssessment    time.Time                `json:"lastAssessment"`    // When assessment was made
	DataSources       map[string]SourceQuality `json:"dataSources"`       // Quality by data source
	IssuesDetected    []DataQualityIssue       `json:"issuesDetected"`    // Any data quality issues found
	Recommendations   []string                 `json:"recommendations"`   // Suggestions for improvement
}

// DataQualityFactor represents a specific factor affecting data quality
type DataQualityFactor struct {
	FactorName  string    `json:"factorName"`  // e.g., "API Response Time"
	Category    string    `json:"category"`    // e.g., "Freshness", "Completeness"
	Score       float64   `json:"score"`       // 0-100 score for this factor
	Weight      float64   `json:"weight"`      // Importance weight
	Description string    `json:"description"` // Human-readable description
	IsCritical  bool      `json:"isCritical"`  // Whether this is a critical factor
	Trend       string    `json:"trend"`       // "improving", "stable", "declining"
	LastUpdated time.Time `json:"lastUpdated"` // When this factor was last updated
}

// SourceQuality represents quality metrics for a specific data source
type SourceQuality struct {
	SourceName       string    `json:"sourceName"`       // e.g., "NHL API", "ESPN API"
	ReliabilityScore float64   `json:"reliabilityScore"` // Historical reliability 0-100
	ResponseTime     float64   `json:"responseTime"`     // Average response time (ms)
	UptimeScore      float64   `json:"uptimeScore"`      // Uptime percentage
	DataFreshness    float64   `json:"dataFreshness"`    // How fresh the data is (hours)
	ErrorRate        float64   `json:"errorRate"`        // Percentage of failed requests
	LastSuccess      time.Time `json:"lastSuccess"`      // Last successful data fetch
	LastError        time.Time `json:"lastError"`        // Last error encountered
	IsAvailable      bool      `json:"isAvailable"`      // Currently available
	QualityTrend     string    `json:"qualityTrend"`     // Recent quality trend
}

// DataQualityIssue represents a detected data quality problem
type DataQualityIssue struct {
	IssueType    string    `json:"issueType"`    // e.g., "stale_data", "missing_field"
	Severity     string    `json:"severity"`     // "low", "medium", "high", "critical"
	Description  string    `json:"description"`  // Human-readable description
	AffectedData []string  `json:"affectedData"` // Which data fields are affected
	DetectedAt   time.Time `json:"detectedAt"`   // When issue was detected
	Impact       float64   `json:"impact"`       // Impact on prediction quality (0-1)
	Suggestion   string    `json:"suggestion"`   // How to resolve the issue
	IsResolved   bool      `json:"isResolved"`   // Whether issue has been resolved
}

// PredictionAccuracy tracks the historical accuracy of our prediction models
type PredictionAccuracy struct {
	ModelName             string            `json:"modelName"`             // "Enhanced Statistical", "Bayesian", etc.
	PredictionDate        time.Time         `json:"predictionDate"`        // When prediction was made
	GameDate              time.Time         `json:"gameDate"`              // When game was played
	HomeTeam              string            `json:"homeTeam"`              // Home team code
	AwayTeam              string            `json:"awayTeam"`              // Away team code
	PredictedWinner       string            `json:"predictedWinner"`       // Who we predicted to win
	ActualWinner          string            `json:"actualWinner"`          // Who actually won
	PredictedScore        string            `json:"predictedScore"`        // Our predicted score
	ActualScore           string            `json:"actualScore"`           // Actual final score
	WinProbability        float64           `json:"winProbability"`        // Our predicted win probability
	Confidence            float64           `json:"confidence"`            // Our confidence level
	IsCorrect             bool              `json:"isCorrect"`             // Did we predict the winner correctly?
	ScoreAccuracy         float64           `json:"scoreAccuracy"`         // How close was our score prediction?
	ProbabilityError      float64           `json:"probabilityError"`      // Difference between predicted and actual probability
	ConfidenceCalibration float64           `json:"confidenceCalibration"` // How well-calibrated was our confidence?
	GameType              string            `json:"gameType"`              // "regular", "playoff", "preseason"
	PredictionFactors     PredictionFactors `json:"predictionFactors"`     // Factors used in prediction
	ActualFactors         ActualGameFactors `json:"actualFactors"`         // What actually happened
}

// ActualGameFactors represents what actually happened in the game
type ActualGameFactors struct {
	HomeGoals        int     `json:"homeGoals"`        // Actual home team goals
	AwayGoals        int     `json:"awayGoals"`        // Actual away team goals
	HomeShotsOnGoal  int     `json:"homeShotsOnGoal"`  // Actual home team shots
	AwayShotsOnGoal  int     `json:"awayShotsOnGoal"`  // Actual away team shots
	HomePowerPlays   int     `json:"homePowerPlays"`   // Home team power play opportunities
	AwayPowerPlays   int     `json:"awayPowerPlays"`   // Away team power play opportunities
	HomePPGoals      int     `json:"homePPGoals"`      // Home team power play goals
	AwayPPGoals      int     `json:"awayPPGoals"`      // Away team power play goals
	HomePenalties    int     `json:"homePenalties"`    // Home team penalties
	AwayPenalties    int     `json:"awayPenalties"`    // Away team penalties
	HomeHits         int     `json:"homeHits"`         // Home team hits
	AwayHits         int     `json:"awayHits"`         // Away team hits
	HomeFaceoffWins  int     `json:"homeFaceoffWins"`  // Home team faceoff wins
	AwayFaceoffWins  int     `json:"awayFaceoffWins"`  // Away team faceoff wins
	HomeBlocks       int     `json:"homeBlocks"`       // Home team blocked shots
	AwayBlocks       int     `json:"awayBlocks"`       // Away team blocked shots
	GameDuration     string  `json:"gameDuration"`     // "regulation", "overtime", "shootout"
	AttendanceImpact float64 `json:"attendanceImpact"` // How crowd affected the game
	InjuryImpact     float64 `json:"injuryImpact"`     // Impact of injuries during game
	WeatherImpact    float64 `json:"weatherImpact"`    // Weather conditions impact
}

// ModelAccuracyStats represents aggregated accuracy statistics for a model
type ModelAccuracyStats struct {
	ModelName               string    `json:"modelName"`
	TotalPredictions        int       `json:"totalPredictions"`        // Total predictions made
	CorrectPredictions      int       `json:"correctPredictions"`      // Correct winner predictions
	WinnerAccuracy          float64   `json:"winnerAccuracy"`          // % of correct winner predictions
	AverageScoreAccuracy    float64   `json:"averageScoreAccuracy"`    // Average score prediction accuracy
	AverageProbabilityError float64   `json:"averageProbabilityError"` // Average probability error
	ConfidenceCalibration   float64   `json:"confidenceCalibration"`   // How well-calibrated confidence is
	RecentAccuracy          float64   `json:"recentAccuracy"`          // Accuracy over last 10 games
	StreakLength            int       `json:"streakLength"`            // Current correct prediction streak
	BestStreak              int       `json:"bestStreak"`              // Longest correct prediction streak
	WorstStreak             int       `json:"worstStreak"`             // Longest incorrect prediction streak
	LastUpdated             time.Time `json:"lastUpdated"`             // When stats were last updated

	// Performance by game type
	RegularSeasonAccuracy float64 `json:"regularSeasonAccuracy"`
	PlayoffAccuracy       float64 `json:"playoffAccuracy"`
	PreseasonAccuracy     float64 `json:"preseasonAccuracy"`

	// Performance by confidence level
	HighConfidenceAccuracy   float64 `json:"highConfidenceAccuracy"`   // Accuracy when confidence > 80%
	MediumConfidenceAccuracy float64 `json:"mediumConfidenceAccuracy"` // Accuracy when confidence 60-80%
	LowConfidenceAccuracy    float64 `json:"lowConfidenceAccuracy"`    // Accuracy when confidence < 60%

	// Performance by team strength
	FavoriteAccuracy float64 `json:"favoriteAccuracy"` // Accuracy when predicting favorites
	UnderdogAccuracy float64 `json:"underdogAccuracy"` // Accuracy when predicting underdogs
	UpsetDetection   float64 `json:"upsetDetection"`   // How well we detect upsets

	// Trend analysis
	AccuracyTrend     string  `json:"accuracyTrend"`     // "improving", "declining", "stable"
	TrendStrength     float64 `json:"trendStrength"`     // How strong the trend is
	SeasonalVariation float64 `json:"seasonalVariation"` // Variation in accuracy by season
}

// EnsembleAccuracyStats represents accuracy stats for the entire ensemble
type EnsembleAccuracyStats struct {
	TotalPredictions          int                            `json:"totalPredictions"`
	OverallAccuracy           float64                        `json:"overallAccuracy"`
	IndividualModelStats      map[string]*ModelAccuracyStats `json:"individualModelStats"`
	ModelConsensusAccuracy    float64                        `json:"modelConsensusAccuracy"`    // Accuracy when all models agree
	ModelDisagreementAccuracy float64                        `json:"modelDisagreementAccuracy"` // Accuracy when models disagree
	OptimalWeights            map[string]float64             `json:"optimalWeights"`            // Optimal weights based on performance
	ConfidenceCalibration     CalibrationCurve               `json:"confidenceCalibration"`     // Overall confidence calibration
	LastUpdated               time.Time                      `json:"lastUpdated"`
}

// CalibrationCurve represents how well-calibrated our confidence levels are
type CalibrationCurve struct {
	ConfidenceBins    []ConfidenceBin `json:"confidenceBins"`    // Bins for different confidence levels
	OverallBrierScore float64         `json:"overallBrierScore"` // Brier score for probability accuracy
	Reliability       float64         `json:"reliability"`       // How reliable our probabilities are
	Resolution        float64         `json:"resolution"`        // How much our probabilities vary
	Uncertainty       float64         `json:"uncertainty"`       // Base rate uncertainty
	
	// Phase 3 additions
	Bins         []ConfidenceBin `json:"bins"`         // Alias for ConfidenceBins (Phase 3 compatibility)
	OverallBias  float64         `json:"overallBias"`  // Systematic over/under-confidence
	TotalSamples int             `json:"totalSamples"` // Total number of predictions
	LastUpdated  time.Time       `json:"lastUpdated"`  // Last update timestamp
}

// ConfidenceBin represents accuracy within a confidence range
type ConfidenceBin struct {
	MinConfidence    float64 `json:"minConfidence"`    // e.g., 0.8
	MaxConfidence    float64 `json:"maxConfidence"`    // e.g., 0.9
	PredictionCount  int     `json:"predictionCount"`  // Number of predictions in this bin
	ActualAccuracy   float64 `json:"actualAccuracy"`   // Actual accuracy in this bin
	ExpectedAccuracy float64 `json:"expectedAccuracy"` // Expected accuracy (average confidence)
	CalibrationError float64 `json:"calibrationError"` // |actual - expected|
	IsWellCalibrated bool    `json:"isWellCalibrated"` // Whether this bin is well-calibrated
	
	// Phase 3 additions
	Range          string    `json:"range"`          // e.g., "90-100%"
	PredictedConf  float64   `json:"predictedConf"`  // Alias for ExpectedAccuracy
	CalibrationAdj float64   `json:"calibrationAdj"` // Alias for CalibrationError
	SampleSize     int       `json:"sampleSize"`     // Alias for PredictionCount
	LastUpdated    time.Time `json:"lastUpdated"`    // Last update timestamp
}

// PredictionPerformanceMetrics represents detailed performance analysis
type PredictionPerformanceMetrics struct {
	ModelName              string              `json:"modelName"`
	TimeWindow             string              `json:"timeWindow"`             // "last_week", "last_month", "season"
	AccuracyByOpponent     map[string]float64  `json:"accuracyByOpponent"`     // Accuracy vs specific opponents
	AccuracyByVenue        map[string]float64  `json:"accuracyByVenue"`        // Accuracy at specific venues
	AccuracyByDayOfWeek    map[string]float64  `json:"accuracyByDayOfWeek"`    // Accuracy by day of week
	AccuracyByMonth        map[string]float64  `json:"accuracyByMonth"`        // Seasonal accuracy patterns
	FactorImportance       map[string]float64  `json:"factorImportance"`       // Which factors are most predictive
	CommonMistakes         []PredictionMistake `json:"commonMistakes"`         // Patterns in incorrect predictions
	StrengthAreas          []string            `json:"strengthAreas"`          // What we predict well
	WeaknessAreas          []string            `json:"weaknessAreas"`          // What we struggle with
	ImprovementSuggestions []string            `json:"improvementSuggestions"` // Suggestions for improvement
}

// PredictionMistake represents a pattern in incorrect predictions
type PredictionMistake struct {
	MistakeType  string   `json:"mistakeType"`  // "overconfident_favorite", "missed_upset", etc.
	Frequency    int      `json:"frequency"`    // How often this mistake occurs
	AverageError float64  `json:"averageError"` // Average error when this mistake occurs
	Description  string   `json:"description"`  // Human-readable description
	Suggestions  []string `json:"suggestions"`  // How to fix this mistake
}

// ConfidenceBoostFactors represents factors that should increase confidence
type ConfidenceBoostFactors struct {
	ModelConsensus         float64 `json:"modelConsensus"`         // How much models agree (0-1)
	HistoricalAccuracy     float64 `json:"historicalAccuracy"`     // Recent model accuracy (0-1)
	DataQuality            float64 `json:"dataQuality"`            // Quality of input data (0-1)
	SituationalClarity     float64 `json:"situationalClarity"`     // How clear the situation is (0-1)
	FactorStrength         float64 `json:"factorStrength"`         // Strength of predictive factors (0-1)
	MarketConsensus        float64 `json:"marketConsensus"`        // Agreement with betting markets (0-1)
	InjuryClarity          float64 `json:"injuryClarity"`          // Clarity of injury situation (0-1)
	WeatherStability       float64 `json:"weatherStability"`       // Weather predictability (0-1)
	VenueAdvantage         float64 `json:"venueAdvantage"`         // Clear venue advantage (0-1)
	MotivationalFactors    float64 `json:"motivationalFactors"`    // Clear motivational advantages (0-1)
	OverallConfidenceBoost float64 `json:"overallConfidenceBoost"` // Final confidence multiplier
}
