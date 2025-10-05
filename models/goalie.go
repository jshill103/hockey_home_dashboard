package models

import "time"

// GoalieInfo represents comprehensive goalie data
type GoalieInfo struct {
	PlayerID    int       `json:"playerId"`
	Name        string    `json:"name"`
	TeamCode    string    `json:"teamCode"`
	IsStarting  bool      `json:"isStarting"`
	LastUpdated time.Time `json:"lastUpdated"`

	// Career Stats
	CareerWins           int     `json:"careerWins"`
	CareerLosses         int     `json:"careerLosses"`
	CareerOTLosses       int     `json:"careerOTLosses"`
	CareerSavePercentage float64 `json:"careerSavePercentage"`
	CareerGAA            float64 `json:"careerGAA"`
	CareerShutouts       int     `json:"careerShutouts"`

	// Season Stats
	SeasonGamesPlayed    int     `json:"seasonGamesPlayed"`
	SeasonWins           int     `json:"seasonWins"`
	SeasonLosses         int     `json:"seasonLosses"`
	SeasonOTLosses       int     `json:"seasonOTLosses"`
	SeasonSavePercentage float64 `json:"seasonSavePercentage"`
	SeasonGAA            float64 `json:"seasonGAA"`
	SeasonShutouts       int     `json:"seasonShutouts"`

	// Recent Performance (Last 5 Starts)
	RecentStarts  []GoalieStart `json:"recentStarts"`
	RecentWinPct  float64       `json:"recentWinPct"`
	RecentSavePct float64       `json:"recentSavePct"`
	RecentGAA     float64       `json:"recentGAA"`

	// Advanced Metrics
	GoalsSavedAboveExpected float64 `json:"goalsSavedAboveExpected"` // GSAx
	HighDangerSavePct       float64 `json:"highDangerSavePct"`
	ReboundControlRate      float64 `json:"reboundControlRate"`
	QualityStartPct         float64 `json:"qualityStartPct"`   // QS%
	ReallyBadStartPct       float64 `json:"reallyBadStartPct"` // RBS%

	// Workload
	GamesInLast7Days     int     `json:"gamesInLast7Days"`
	ShotsFacedPerGame    float64 `json:"shotsFacedPerGame"`
	WorkloadFatigueScore float64 `json:"workloadFatigueScore"` // 0.0 (rested) to 1.0 (exhausted)

	// Situational Performance
	HomeRecord          GoalieRecord         `json:"homeRecord"`
	AwayRecord          GoalieRecord         `json:"awayRecord"`
	BackToBackRecord    GoalieRecord         `json:"backToBackRecord"`
	RestDaysPerformance map[int]GoalieRecord `json:"restDaysPerformance"` // rest days -> record
}

// GoalieStart represents a single start
type GoalieStart struct {
	GameID         int       `json:"gameId"`
	GameDate       time.Time `json:"gameDate"`
	Opponent       string    `json:"opponent"`
	IsHome         bool      `json:"isHome"`
	Result         string    `json:"result"` // "W", "L", "OTL", "SOL"
	ShotsAgainst   int       `json:"shotsAgainst"`
	Saves          int       `json:"saves"`
	GoalsAgainst   int       `json:"goalsAgainst"`
	SavePct        float64   `json:"savePct"`
	IsQualityStart bool      `json:"isQualityStart"` // QS: Save% > .913 or <3 GA in 60+ min
	MinutesPlayed  int       `json:"minutesPlayed"`
}

// GoalieRecord represents win-loss record
type GoalieRecord struct {
	Wins        int     `json:"wins"`
	Losses      int     `json:"losses"`
	OTLosses    int     `json:"otLosses"`
	WinPct      float64 `json:"winPct"`
	SavePct     float64 `json:"savePct"`
	GAA         float64 `json:"gaa"`
	GamesPlayed int     `json:"gamesPlayed"`
}

// GoalieMatchup represents goalie vs team history
type GoalieMatchup struct {
	GoalieID       int           `json:"goalieId"`
	GoalieName     string        `json:"goalieName"`
	OpponentTeam   string        `json:"opponentTeam"`
	Record         GoalieRecord  `json:"record"`
	RecentGames    []GoalieStart `json:"recentGames"` // Last 5 vs this team
	AverageSavePct float64       `json:"averageSavePct"`
	LastFaced      time.Time     `json:"lastFaced"`
}

// GoalieComparison represents matchup comparison
type GoalieComparison struct {
	HomeGoalie *GoalieInfo `json:"homeGoalie"`
	AwayGoalie *GoalieInfo `json:"awayGoalie"`

	// Advantage Calculations
	OverallAdvantage string  `json:"overallAdvantage"` // "home", "away", "even"
	AdvantageScore   float64 `json:"advantageScore"`   // -1.0 (away) to +1.0 (home)

	// Factor Breakdown
	SeasonPerformance float64 `json:"seasonPerformance"` // Season save % comparison
	RecentForm        float64 `json:"recentForm"`        // Last 5 starts comparison
	WorkloadFatigue   float64 `json:"workloadFatigue"`   // Workload comparison
	MatchupHistory    float64 `json:"matchupHistory"`    // vs opponent history
	HomeAwayFactor    float64 `json:"homeAwayFactor"`    // Home/away splits

	// Impact on Game
	WinProbabilityImpact float64 `json:"winProbabilityImpact"` // Adjust win % by this much
	Confidence           float64 `json:"confidence"`           // 0.0-1.0
}

// GoalieDepth represents team's goalie situation
type GoalieDepth struct {
	TeamCode        string      `json:"teamCode"`
	Starter         *GoalieInfo `json:"starter"`
	Backup          *GoalieInfo `json:"backup"`
	StartingTonight *GoalieInfo `json:"startingTonight"`

	// Depth Quality
	StarterQuality float64 `json:"starterQuality"` // 0.0-1.0
	BackupQuality  float64 `json:"backupQuality"`  // 0.0-1.0
	DepthScore     float64 `json:"depthScore"`     // Overall goalie depth

	// Status
	StarterHealthy  bool      `json:"starterHealthy"`
	BackupHealthy   bool      `json:"backupHealthy"`
	EmergencyBackup bool      `json:"emergencyBackup"` // Using EBUG?
	LastUpdated     time.Time `json:"lastUpdated"`
}

// GoalieTrendAnalysis represents goalie momentum
type GoalieTrendAnalysis struct {
	GoalieID   int    `json:"goalieId"`
	GoalieName string `json:"goalieName"`

	// Trend Direction
	Trend         string  `json:"trend"`         // "Hot", "Cold", "Stable"
	TrendStrength float64 `json:"trendStrength"` // 0.0-1.0

	// Performance Trajectory
	Last3SavePct     float64 `json:"last3SavePct"`
	Last5SavePct     float64 `json:"last5SavePct"`
	Last10SavePct    float64 `json:"last10SavePct"`
	PerformanceDelta float64 `json:"performanceDelta"` // Change from avg

	// Momentum Indicators
	CurrentStreak       int `json:"currentStreak"` // Positive = wins, negative = losses
	RecentShutouts      int `json:"recentShutouts"`
	ConsecutiveQLStarts int `json:"consecutiveQLStarts"` // Quality starts in a row

	// Confidence Level
	MomentumScore float64   `json:"momentumScore"` // -1.0 (bad) to +1.0 (hot)
	LastUpdated   time.Time `json:"lastUpdated"`
}
