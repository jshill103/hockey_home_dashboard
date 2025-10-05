package models

import "time"

// TeamRecentPerformance tracks rolling statistics for a team
type TeamRecentPerformance struct {
	TeamCode    string    `json:"teamCode"`
	Season      int       `json:"season"`
	LastUpdated time.Time `json:"lastUpdated"`

	// Recent Games History
	Last5Games  []GameSummary `json:"last5Games"`
	Last10Games []GameSummary `json:"last10Games"`

	// Rolling Averages (Last 10 Games)
	RecentGoalsFor       float64 `json:"recentGoalsFor"`       // Average goals scored
	RecentGoalsAgainst   float64 `json:"recentGoalsAgainst"`   // Average goals allowed
	RecentWinPct         float64 `json:"recentWinPct"`         // Win percentage
	RecentShotPct        float64 `json:"recentShotPct"`        // Shooting percentage
	RecentSavesPct       float64 `json:"recentSavesPct"`       // Save percentage
	RecentPowerPlayPct   float64 `json:"recentPowerPlayPct"`   // PP success rate
	RecentPenaltyKillPct float64 `json:"recentPenaltyKillPct"` // PK success rate
	RecentFaceoffPct     float64 `json:"recentFaceoffPct"`     // Faceoff win %

	// Momentum Indicators
	Momentum          float64 `json:"momentum"`          // Trend score (-1 to +1)
	MomentumDirection string  `json:"momentumDirection"` // "improving", "declining", "stable"
	PointsPerGame     float64 `json:"pointsPerGame"`     // Recent points per game

	// Streak Information
	CurrentStreak     int    `json:"currentStreak"`     // +5 = 5 wins, -3 = 3 losses
	StreakType        string `json:"streakType"`        // "win", "loss", "none"
	LongestWinStreak  int    `json:"longestWinStreak"`  // Season high
	LongestLossStreak int    `json:"longestLossStreak"` // Season low

	// Schedule Context
	GamesSinceLastHome int     `json:"gamesSinceLastHome"` // Games since last home game
	GamesSinceLastAway int     `json:"gamesSinceLastAway"` // Games since last away game
	BackToBackGames    int     `json:"backToBackGames"`    // B2B games in last 10
	RestDaysAverage    float64 `json:"restDaysAverage"`    // Average rest between games

	// Performance Splits
	HomeRecord       Record  `json:"homeRecord"`       // Home W-L-OTL
	AwayRecord       Record  `json:"awayRecord"`       // Away W-L-OTL
	RecentHomeWinPct float64 `json:"recentHomeWinPct"` // Last 5 home games
	RecentAwayWinPct float64 `json:"recentAwayWinPct"` // Last 5 away games

	// Advanced Metrics
	CorsiFor        float64 `json:"corsiFor"`        // Shot attempts for
	CorsiAgainst    float64 `json:"corsiAgainst"`    // Shot attempts against
	CorsiPercentage float64 `json:"corsiPercentage"` // CF / (CF + CA)
	PdoScore        float64 `json:"pdoScore"`        // Shooting % + Save % (luck indicator)

	// Consistency Metrics
	GoalsVariance        float64 `json:"goalsVariance"`        // Consistency of scoring
	DefenseVariance      float64 `json:"defenseVariance"`      // Consistency of defense
	PerformanceStability float64 `json:"performanceStability"` // Overall consistency (0-1)

	// PHASE 6: Quality-Weighted Performance
	QualityOfWins     float64 `json:"qualityOfWins"`     // Strength of opponents beaten
	QualityOfLosses   float64 `json:"qualityOfLosses"`   // Strength of opponents lost to
	VsPlayoffTeamsPct float64 `json:"vsPlayoffTeamsPct"` // Win % vs playoff teams
	VsTop10TeamsPct   float64 `json:"vsTop10TeamsPct"`   // Win % vs top 10 teams
	ClutchPerformance float64 `json:"clutchPerformance"` // Performance in close games
	BlowoutWinPct     float64 `json:"blowoutWinPct"`     // % of wins by 3+ goals
	CloseGameRecord   Record  `json:"closeGameRecord"`   // Record in 1-goal games

	// PHASE 6: Time-Weighted Metrics (exponential decay)
	WeightedGoalsFor     float64 `json:"weightedGoalsFor"`     // Recent games weighted more
	WeightedGoalsAgainst float64 `json:"weightedGoalsAgainst"` // Recent games weighted more
	WeightedWinPct       float64 `json:"weightedWinPct"`       // Recent wins matter more
	MomentumScore        float64 `json:"momentumScore"`        // -1 to +1 (weighted trend)
	FormRating           float64 `json:"formRating"`           // 0-10 current form rating

	// PHASE 6: Momentum Indicators
	Last3GamesPoints     int    `json:"last3GamesPoints"`     // Points in last 3 games
	Last5GamesPoints     int    `json:"last5GamesPoints"`     // Points in last 5 games
	Last10GamesPoints    int    `json:"last10GamesPoints"`    // Points in last 10 games
	PointsTrendDirection string `json:"pointsTrendDirection"` // "accelerating", "stable", "declining"
	GoalDifferential3    int    `json:"goalDifferential3"`    // +/- last 3 games
	GoalDifferential5    int    `json:"goalDifferential5"`    // +/- last 5 games
	GoalDifferential10   int    `json:"goalDifferential10"`   // +/- last 10 games

	// PHASE 6: Hot/Cold Indicators
	IsHot             bool `json:"isHot"`             // 4+ wins in last 5
	IsCold            bool `json:"isCold"`            // 4+ losses in last 5
	IsStreaking       bool `json:"isStreaking"`       // 3+ game streak
	DaysSinceLastWin  int  `json:"daysSinceLastWin"`  // Drought tracking
	DaysSinceLastLoss int  `json:"daysSinceLastLoss"` // Dominance tracking

	// PHASE 6: Advanced Scoring Trends
	ScoringTrend     float64 `json:"scoringTrend"`     // Goals/game trend (+/-)
	DefensiveTrend   float64 `json:"defensiveTrend"`   // Goals against trend (+/-)
	PowerPlayTrend   float64 `json:"powerPlayTrend"`   // PP% trend
	PenaltyKillTrend float64 `json:"penaltyKillTrend"` // PK% trend
	GoalieTrend      float64 `json:"goalieTrend"`      // Save% trend

	// PHASE 6: Opponent-Adjusted Metrics
	StrengthOfSchedule   float64 `json:"strengthOfSchedule"`   // Average opponent quality
	AdjustedGoalsFor     float64 `json:"adjustedGoalsFor"`     // Quality-adjusted scoring
	AdjustedGoalsAgainst float64 `json:"adjustedGoalsAgainst"` // Quality-adjusted defense
	AdjustedWinPct       float64 `json:"adjustedWinPct"`       // Opponent-adjusted wins
}

// GameSummary represents a simplified game result for rolling stats
type GameSummary struct {
	GameID         int       `json:"gameId"`
	Date           time.Time `json:"date"`
	Opponent       string    `json:"opponent"`
	IsHome         bool      `json:"isHome"`
	GoalsFor       int       `json:"goalsFor"`
	GoalsAgainst   int       `json:"goalsAgainst"`
	Result         string    `json:"result"` // "W", "L", "OTL", "SOL"
	Points         int       `json:"points"` // 2 for W, 1 for OTL/SOL, 0 for L
	Shots          int       `json:"shots"`
	ShotsAgainst   int       `json:"shotsAgainst"`
	PowerPlayGoals int       `json:"powerPlayGoals"`
	PowerPlayOpps  int       `json:"powerPlayOpps"`
	RestDays       int       `json:"restDays"`

	// PHASE 6: Quality Metrics
	OpponentRank     int     `json:"opponentRank"`     // Opponent's league rank (1-32)
	OpponentWinPct   float64 `json:"opponentWinPct"`   // Opponent's win % at time of game
	OpponentStrength float64 `json:"opponentStrength"` // 0-1, opponent quality
	WasCloseGame     bool    `json:"wasCloseGame"`     // 1-goal game
	WasBlowout       bool    `json:"wasBlowout"`       // 3+ goal differential
	GameImportance   float64 `json:"gameImportance"`   // 0-1, how important
}

// Record represents a team's win-loss record
type Record struct {
	Wins     int `json:"wins"`
	Losses   int `json:"losses"`
	OTLosses int `json:"otLosses"`
	Points   int `json:"points"`
}

// CalculateWinPercentage calculates win percentage for a record
func (r *Record) CalculateWinPercentage() float64 {
	totalGames := r.Wins + r.Losses + r.OTLosses
	if totalGames == 0 {
		return 0.0
	}
	return float64(r.Wins) / float64(totalGames)
}

// CalculatePointsPercentage calculates points percentage (pts / max pts)
func (r *Record) CalculatePointsPercentage() float64 {
	totalGames := r.Wins + r.Losses + r.OTLosses
	if totalGames == 0 {
		return 0.0
	}
	maxPoints := totalGames * 2
	return float64(r.Points) / float64(maxPoints)
}
