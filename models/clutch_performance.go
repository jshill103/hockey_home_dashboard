package models

import "time"

// ClutchPerformance represents a team's performance in critical situations
type ClutchPerformance struct {
	TeamCode              string    `json:"teamCode"`
	Season                string    `json:"season"`
	CloseGameRecord       Record    `json:"closeGameRecord"`       // 1-goal games
	ThirdPeriodComebacks  int       `json:"thirdPeriodComebacks"`  // Comebacks in 3rd period
	OvertimeRecord        Record    `json:"overtimeRecord"`        // OT/SO performance
	LateGameCollapses     int       `json:"lateGameCollapses"`     // Blown leads
	PressurePerformance   float64   `json:"pressurePerformance"`   // Performance under pressure (0-1)
	ClutchFactor          float64   `json:"clutchFactor"`          // Overall clutch rating (-0.10 to +0.10)
	ThirdPeriodGoalDiff   float64   `json:"thirdPeriodGoalDiff"`   // 3rd period goal differential
	ComebackWinRate       float64   `json:"comebackWinRate"`       // % of comebacks successful
	ProtectLeadRate       float64   `json:"protectLeadRate"`       // % of leads protected
	Confidence            float64   `json:"confidence"`            // Statistical confidence (0-1)
	LastUpdated           time.Time `json:"lastUpdated"`
}

// Record represents a win-loss record
type Record struct {
	Wins           int     `json:"wins"`
	Losses         int     `json:"losses"`
	OvertimeLosses int     `json:"overtimeLosses"` // Optional
	Total          int     `json:"total"`
	WinPercentage  float64 `json:"winPercentage"`
}

// ClutchSituation represents a specific clutch scenario
type ClutchSituation struct {
	GameID           int       `json:"gameId"`
	TeamCode         string    `json:"teamCode"`
	Date             time.Time `json:"date"`
	SituationType    string    `json:"situationType"`    // "CloseGame", "Comeback", "ProtectLead", "Overtime"
	Period           int       `json:"period"`           // Period when situation occurred
	TimeRemaining    string    `json:"timeRemaining"`    // Time remaining in game
	ScoreDeficit     int       `json:"scoreDeficit"`     // Goals behind (negative if ahead)
	Outcome          string    `json:"outcome"`          // "Success", "Failure"
	GameImportance   float64   `json:"gameImportance"`   // How important was the game (0-1)
	PerformanceRating float64  `json:"performanceRating"` // How well team performed (0-1)
}

// ClutchMetrics provides detailed clutch metrics
type ClutchMetrics struct {
	TeamCode                string    `json:"teamCode"`
	Season                  string    `json:"season"`
	
	// Close Game Metrics
	OneGoalGames            int     `json:"oneGoalGames"`
	OneGoalWins             int     `json:"oneGoalWins"`
	OneGoalWinRate          float64 `json:"oneGoalWinRate"`
	
	// Third Period Metrics
	ThirdPeriodGoalsFor     float64 `json:"thirdPeriodGoalsFor"`     // Per game
	ThirdPeriodGoalsAgainst float64 `json:"thirdPeriodGoalsAgainst"` // Per game
	ThirdPeriodDifferential float64 `json:"thirdPeriodDifferential"`
	TrailingAfter2Wins      int     `json:"trailingAfter2Wins"`      // Wins when trailing after 2
	TrailingAfter2Games     int     `json:"trailingAfter2Games"`     // Games when trailing after 2
	ComebackRate            float64 `json:"comebackRate"`
	
	// Lead Protection
	LeadingAfter2Wins       int     `json:"leadingAfter2Wins"`
	LeadingAfter2Games      int     `json:"leadingAfter2Games"`
	LeadProtectionRate      float64 `json:"leadProtectionRate"`
	BlownLeads              int     `json:"blownLeads"`
	
	// Overtime/Shootout
	OvertimeGames           int     `json:"overtimeGames"`
	OvertimeWins            int     `json:"overtimeWins"`
	ShootoutGames           int     `json:"shootoutGames"`
	ShootoutWins            int     `json:"shootoutWins"`
	OTSOWinRate             float64 `json:"otsoWinRate"`
	
	// Pressure Situations
	PlayoffRaceGames        int     `json:"playoffRaceGames"`    // Games with playoff implications
	PlayoffRaceWins         int     `json:"playoffRaceWins"`
	PlayoffRaceWinRate      float64 `json:"playoffRaceWinRate"`
	MustWinGames            int     `json:"mustWinGames"`        // Critical late-season games
	MustWinSuccesses        int     `json:"mustWinSuccesses"`
	
	LastUpdated             time.Time `json:"lastUpdated"`
}

// ClutchComparison compares clutch performance between two teams
type ClutchComparison struct {
	HomeTeam          string            `json:"homeTeam"`
	AwayTeam          string            `json:"awayTeam"`
	HomeClutch        ClutchPerformance `json:"homeClutch"`
	AwayClutch        ClutchPerformance `json:"awayClutch"`
	GameImportance    float64           `json:"gameImportance"`    // How important is this game (0-1)
	ClutchAdvantage   string            `json:"clutchAdvantage"`   // "Home", "Away", "Neutral"
	ImpactFactor      float64           `json:"impactFactor"`      // Prediction adjustment
	Confidence        float64           `json:"confidence"`
	Analysis          string            `json:"analysis"`          // Human-readable analysis
	RecommendedWeight float64           `json:"recommendedWeight"` // How much to weight clutch
}

// ClutchProfile categorizes a team's clutch tendencies
type ClutchProfile struct {
	TeamCode          string  `json:"teamCode"`
	ProfileType       string  `json:"profileType"`       // "Clutch", "Choke", "Neutral", "Inconsistent"
	ClutchRating      float64 `json:"clutchRating"`      // -1.0 to +1.0
	Consistency       float64 `json:"consistency"`       // How consistent in clutch (0-1)
	BestClutchType    string  `json:"bestClutchType"`    // Where they excel
	WorstClutchType   string  `json:"worstClutchType"`   // Where they struggle
	ImprovementTrend  float64 `json:"improvementTrend"`  // Getting better/worse
	Description       string  `json:"description"`       // Human-readable profile
}

// ClutchHistory tracks historical clutch performances
type ClutchHistory struct {
	TeamCode       string            `json:"teamCode"`
	Season         string            `json:"season"`
	Situations     []ClutchSituation `json:"situations"`
	ByMonth        map[string]float64 `json:"byMonth"`        // Clutch factor by month
	ByOpponent     map[string]float64 `json:"byOpponent"`     // Clutch vs specific teams
	MostClutchGame ClutchSituation   `json:"mostClutchGame"` // Most clutch performance
	LastUpdated    time.Time         `json:"lastUpdated"`
}

// ClutchPrediction provides clutch-adjusted prediction
type ClutchPrediction struct {
	HomeTeam         string  `json:"homeTeam"`
	AwayTeam         string  `json:"awayTeam"`
	GameImportance   float64 `json:"gameImportance"`   // 0-1
	ExpectedMargin   float64 `json:"expectedMargin"`   // Expected goal margin
	IsCloseGame      bool    `json:"isCloseGame"`      // Likely to be close?
	ClutchImpact     float64 `json:"clutchImpact"`     // Adjustment from clutch factors
	AdjustedMargin   float64 `json:"adjustedMargin"`   // Margin after clutch adjustment
	Confidence       float64 `json:"confidence"`
	ClutchScenario   string  `json:"clutchScenario"`   // Most likely clutch scenario
}

// ClutchFactor represents individual clutch components
type ClutchFactor struct {
	Name        string  `json:"name"`        // e.g., "Close Games", "Comebacks"
	Value       float64 `json:"value"`       // Numeric value
	Impact      float64 `json:"impact"`      // Impact on clutch rating
	Weight      float64 `json:"weight"`      // Importance weight (0-1)
	Trend       string  `json:"trend"`       // "Improving", "Declining", "Stable"
	Confidence  float64 `json:"confidence"`  // Confidence in this factor (0-1)
}

// GameImportanceCalculation determines how important a game is
type GameImportanceCalculation struct {
	HomeTeam             string  `json:"homeTeam"`
	AwayTeam             string  `json:"awayTeam"`
	GameDate             time.Time `json:"gameDate"`
	HomePlayoffPosition  int     `json:"homePlayoffPosition"`  // Current playoff standing
	AwayPlayoffPosition  int     `json:"awayPlayoffPosition"`
	HomePlayoffGap       int     `json:"homePlayoffGap"`       // Points from playoff line
	AwayPlayoffGap       int     `json:"awayPlayoffGap"`
	IsDivisionGame       bool    `json:"isDivisionGame"`
	IsRivalryGame        bool    `json:"isRivalryGame"`
	DaysUntilPlayoffs    int     `json:"daysUntilPlayoffs"`
	ImportanceScore      float64 `json:"importanceScore"`      // 0-1 (1 = most important)
	ImpactOnStandings    float64 `json:"impactOnStandings"`    // How much game affects standings
	IsCritical           bool    `json:"isCritical"`           // Top 20% importance
}

