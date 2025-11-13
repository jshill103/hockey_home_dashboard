package models

import "time"

// GameContext represents the contextual situation of a game for model selection
type GameContext struct {
	HomeTeam       string    `json:"homeTeam"`
	AwayTeam       string    `json:"awayTeam"`
	GameDate       time.Time `json:"gameDate"`
	
	// Context Flags
	IsPlayoffGame     bool `json:"isPlayoffGame"`
	IsRivalryGame     bool `json:"isRivalryGame"`
	IsDivisionGame    bool `json:"isDivisionGame"`
	IsEliminationGame bool `json:"isEliminationGame"`
	IsBackToBack      bool `json:"isBackToBack"`      // Either team on B2B
	IsSeasonOpener    bool `json:"isSeasonOpener"`
	IsSeasonFinale    bool `json:"isSeasonFinale"`
	
	// Contextual Factors
	StandingsGap      int     `json:"standingsGap"`      // Points between teams
	PlayoffRaceGap    int     `json:"playoffRaceGap"`    // Points from playoff line
	SeasonProgress    float64 `json:"seasonProgress"`    // 0.0 to 1.0 (% of season complete)
	GamesSinceMeeting int     `json:"gamesSinceMeeting"` // Games since last H2H
	
	// Importance Scoring
	ImpactLevel       string  `json:"impactLevel"`       // "Critical", "High", "Medium", "Low"
	ImportanceScore   float64 `json:"importanceScore"`   // 0.0 to 1.0
	PlayoffImpact     float64 `json:"playoffImpact"`     // How much this affects playoff odds
	
	// Difficulty Assessment
	PredictionDifficulty string  `json:"predictionDifficulty"` // "Very Hard", "Hard", "Medium", "Easy"
	DifficultyScore      float64 `json:"difficultyScore"`      // 0.0 to 1.0 (higher = harder)
	
	// Recommended Strategy
	RecommendedStrategy string `json:"recommendedStrategy"` // "Conservative", "Balanced", "Aggressive", "Data-Driven"
	StrategyReasoning   string `json:"strategyReasoning"`   // Why this strategy
}

// ContextualWeights represents model weight adjustments based on context
type ContextualWeights struct {
	Context      GameContext        `json:"context"`
	BaseWeights  map[string]float64 `json:"baseWeights"`  // Original model weights
	Adjustments  map[string]float64 `json:"adjustments"`  // Adjustment factors (multipliers)
	FinalWeights map[string]float64 `json:"finalWeights"` // Adjusted weights
	Reasoning    map[string]string  `json:"reasoning"`    // Why each adjustment
	Confidence   float64            `json:"confidence"`   // Confidence in these weights (0-1)
}

// SituationalFactor represents a factor affecting prediction difficulty
type SituationalFactor struct {
	Name        string  `json:"name"`        // e.g., "Close Standings", "Injury Uncertainty"
	Impact      float64 `json:"impact"`      // -1.0 to +1.0 (negative = harder prediction)
	Description string  `json:"description"` // Human-readable explanation
	Weight      float64 `json:"weight"`      // Importance of this factor (0-1)
}

// ContextPerformance tracks how well predictions work in different contexts
type ContextPerformance struct {
	ContextType    string  `json:"contextType"`    // e.g., "Playoff", "Rivalry", "B2B"
	TotalPredictions int   `json:"totalPredictions"`
	CorrectPredictions int `json:"correctPredictions"`
	Accuracy       float64 `json:"accuracy"`
	AvgConfidence  float64 `json:"avgConfidence"`
	CalibrationGap float64 `json:"calibrationGap"` // Difference between confidence and accuracy
	LastUpdated    time.Time `json:"lastUpdated"`
}

// ContextualModelPreference stores which models perform best in each context
type ContextualModelPreference struct {
	ContextType string             `json:"contextType"`
	ModelRanking []ModelRanking    `json:"modelRanking"` // Models ranked by performance
	OptimalWeights map[string]float64 `json:"optimalWeights"`
	SampleSize   int                `json:"sampleSize"`
	LastUpdated  time.Time          `json:"lastUpdated"`
}

// ModelRanking represents a model's performance rank in a context
type ModelRanking struct {
	ModelName string  `json:"modelName"`
	Rank      int     `json:"rank"`      // 1 = best
	Accuracy  float64 `json:"accuracy"`  // Historical accuracy in this context
	Weight    float64 `json:"weight"`    // Recommended weight
	Confidence float64 `json:"confidence"` // Confidence in this ranking
}

