package models

import "time"

// MomentumScore represents a team's overall momentum
type MomentumScore struct {
	TeamCode       string    `json:"teamCode"`
	Season         string    `json:"season"`
	Overall        float64   `json:"overall"`        // -1.0 to +1.0 (composite momentum)
	Trend          float64   `json:"trend"`          // Recent trend direction (-1 to +1)
	Acceleration   float64   `json:"acceleration"`   // Is momentum increasing? (-1 to +1)
	Quality        float64   `json:"quality"`        // Quality of recent results (0-1)
	Consistency    float64   `json:"consistency"`    // How consistent is the momentum? (0-1)
	Confidence     float64   `json:"confidence"`     // Statistical confidence (0-1)
	ImpactFactor   float64   `json:"impactFactor"`   // Prediction adjustment (-0.12 to +0.12)
	LastUpdated    time.Time `json:"lastUpdated"`
	
	// Component Scores
	PerformanceMomentum float64 `json:"performanceMomentum"` // Based on wins/losses
	StatisticalMomentum float64 `json:"statisticalMomentum"` // Based on underlying stats
	ScoringMomentum     float64 `json:"scoringMomentum"`     // Based on goal differential
	QualityMomentum     float64 `json:"qualityMomentum"`     // Based on opponent quality
}

// MomentumTrend tracks momentum over time
type MomentumTrend struct {
	TeamCode        string          `json:"teamCode"`
	RecentGames     []GameMomentum  `json:"recentGames"`     // Last 10 games with momentum data
	TrendLine       []float64       `json:"trendLine"`       // Smoothed trend
	TrendDirection  string          `json:"trendDirection"`  // "Rising", "Falling", "Stable"
	TrendStrength   float64         `json:"trendStrength"`   // How strong is the trend (0-1)
	PeakMomentum    float64         `json:"peakMomentum"`    // Highest recent momentum
	LowestMomentum  float64         `json:"lowestMomentum"`  // Lowest recent momentum
	Volatility      float64         `json:"volatility"`      // How much momentum varies
	PredictedNext   float64         `json:"predictedNext"`   // Predicted next game momentum
	LastUpdated     time.Time       `json:"lastUpdated"`
}

// GameMomentum represents momentum data for a single game
type GameMomentum struct {
	GameID            int       `json:"gameId"`
	GameDate          time.Time `json:"gameDate"`
	Opponent          string    `json:"opponent"`
	Result            string    `json:"result"`            // "W", "L", "OTL"
	Score             string    `json:"score"`             // e.g., "4-2"
	GoalDifferential  int       `json:"goalDifferential"`
	ShotDifferential  int       `json:"shotDifferential"`
	XGDifferential    float64   `json:"xgDifferential"`    // Expected goals differential
	OpponentQuality   float64   `json:"opponentQuality"`   // Opponent's strength (0-1)
	MomentumContribution float64 `json:"momentumContribution"` // This game's momentum value
	Weight            float64   `json:"weight"`            // Exponential decay weight
}

// MomentumComparison compares momentum between two teams
type MomentumComparison struct {
	HomeTeam          string        `json:"homeTeam"`
	AwayTeam          string        `json:"awayTeam"`
	HomeMomentum      MomentumScore `json:"homeMomentum"`
	AwayMomentum      MomentumScore `json:"awayMomentum"`
	Advantage         string        `json:"advantage"`         // "Home", "Away", "Neutral"
	MomentumGap       float64       `json:"momentumGap"`       // Difference in momentum
	ImpactFactor      float64       `json:"impactFactor"`      // Prediction adjustment
	Confidence        float64       `json:"confidence"`
	TrendAnalysis     string        `json:"trendAnalysis"`     // Human-readable trend comparison
	RecommendedWeight float64       `json:"recommendedWeight"` // How much to weight momentum
}

// MomentumHistory stores historical momentum data
type MomentumHistory struct {
	TeamCode        string            `json:"teamCode"`
	Season          string            `json:"season"`
	Games           []GameMomentum    `json:"games"`
	MomentumByMonth map[string]float64 `json:"momentumByMonth"` // Average momentum per month
	PeakPeriods     []PeakPeriod      `json:"peakPeriods"`     // Periods of high momentum
	LowPeriods      []LowPeriod       `json:"lowPeriods"`      // Periods of low momentum
	LastUpdated     time.Time         `json:"lastUpdated"`
}

// PeakPeriod represents a period of sustained high momentum
type PeakPeriod struct {
	StartDate   time.Time `json:"startDate"`
	EndDate     time.Time `json:"endDate"`
	Duration    int       `json:"duration"`    // Games
	AvgMomentum float64   `json:"avgMomentum"`
	Record      string    `json:"record"`      // W-L record during period
}

// LowPeriod represents a period of sustained low momentum
type LowPeriod struct {
	StartDate   time.Time `json:"startDate"`
	EndDate     time.Time `json:"endDate"`
	Duration    int       `json:"duration"`    // Games
	AvgMomentum float64   `json:"avgMomentum"`
	Record      string    `json:"record"`      // W-L record during period
}

// MomentumShift detects significant momentum changes
type MomentumShift struct {
	TeamCode      string    `json:"teamCode"`
	ShiftDate     time.Time `json:"shiftDate"`
	ShiftType     string    `json:"shiftType"`     // "Surge", "Collapse", "Reversal"
	BeforeMomentum float64  `json:"beforeMomentum"`
	AfterMomentum  float64  `json:"afterMomentum"`
	Magnitude     float64   `json:"magnitude"`     // Size of shift
	Trigger       string    `json:"trigger"`       // What caused it (if known)
	Impact        float64   `json:"impact"`        // Effect on predictions
}

// MomentumIndicators provides leading indicators of momentum
type MomentumIndicators struct {
	TeamCode              string  `json:"teamCode"`
	RecentXGTrend         float64 `json:"recentXGTrend"`         // Expected goals trend
	RecentShotDifferential float64 `json:"recentShotDifferential"` // Shot differential trend
	RecentCorsiTrend      float64 `json:"recentCorsiTrend"`      // Possession trend
	ScoringFirst          float64 `json:"scoringFirst"`          // Rate of scoring first
	ComebackRate          float64 `json:"comebackRate"`          // Rate of comebacks
	BlowoutRate           float64 `json:"blowoutRate"`           // Rate of dominant wins
	LeadingIndicator      float64 `json:"leadingIndicator"`      // Predictive momentum
}

// MomentumContext provides context for current momentum
type MomentumContext struct {
	TeamCode          string  `json:"teamCode"`
	CurrentMomentum   float64 `json:"currentMomentum"`
	IsHistoricallyHigh bool   `json:"isHistoricallyHigh"` // In top 20% for this team
	IsHistoricallyLow  bool   `json:"isHistoricallyLow"`  // In bottom 20% for this team
	DaysOfMomentum    int     `json:"daysOfMomentum"`     // Days since momentum shift
	Sustainability    float64 `json:"sustainability"`     // Likelihood of continuing (0-1)
	RegressionRisk    float64 `json:"regressionRisk"`     // Risk of mean reversion (0-1)
}

