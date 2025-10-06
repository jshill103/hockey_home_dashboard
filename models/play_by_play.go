package models

import "time"

// PlayByPlayResponse represents the NHL API play-by-play response
type PlayByPlayResponse struct {
	ID           int          `json:"id"`
	Season       int          `json:"season"`
	GameType     int          `json:"gameType"`
	GameDate     string       `json:"gameDate"`
	Venue        VenueInfo    `json:"venue"`
	StartTimeUTC string       `json:"startTimeUTC"`
	Plays        []PlayEvent  `json:"plays"`
	RosterSpots  []RosterSpot `json:"rosterSpots"`
	HomeTeam     BoxscoreTeam `json:"homeTeam"`
	AwayTeam     BoxscoreTeam `json:"awayTeam"`
}

// PlayEvent represents a single event in the game
type PlayEvent struct {
	EventID           int              `json:"eventId"`
	PeriodDescriptor  PeriodDescriptor `json:"periodDescriptor"`
	TimeInPeriod      string           `json:"timeInPeriod"`
	TimeRemaining     string           `json:"timeRemaining"`
	SituationCode     string           `json:"situationCode"` // e.g., "1551" (5v5)
	HomeTeamDefending string           `json:"homeTeamDefendingSide"`
	TypeDescKey       string           `json:"typeDescKey"` // "shot-on-goal", "goal", "hit", etc.
	TypeCode          int              `json:"typeCode"`
	Details           PlayDetails      `json:"details"`
}

// PlayDetails contains detailed information about the play
type PlayDetails struct {
	XCoord              int    `json:"xCoord,omitempty"`   // Shot location X
	YCoord              int    `json:"yCoord,omitempty"`   // Shot location Y
	ZoneCode            string `json:"zoneCode,omitempty"` // "O", "D", "N" (offensive, defensive, neutral)
	ShotType            string `json:"shotType,omitempty"` // "Wrist", "Slap", "Snap", "Backhand", "Tip-In", "Deflected"
	EventOwnerTeamID    int    `json:"eventOwnerTeamId,omitempty"`
	ScoringPlayerID     int    `json:"scoringPlayerId,omitempty"`
	Assist1PlayerID     int    `json:"assist1PlayerId,omitempty"`
	Assist2PlayerID     int    `json:"assist2PlayerId,omitempty"`
	ShootingPlayerID    int    `json:"shootingPlayerId,omitempty"`
	GoalieInNetID       int    `json:"goalieInNetId,omitempty"`
	HittingPlayerID     int    `json:"hittingPlayerId,omitempty"`
	HitteePlayerID      int    `json:"hitteePlayerId,omitempty"`
	BlockingPlayerID    int    `json:"blockingPlayerId,omitempty"`
	Reason              string `json:"reason,omitempty"` // For stoppages, penalties
	SecondaryReason     string `json:"secondaryReason,omitempty"`
	Duration            int    `json:"duration,omitempty"`            // Penalty duration
	CommittedByPlayerID int    `json:"committedByPlayerId,omitempty"` // Penalty
	WinningPlayerID     int    `json:"winningPlayerId,omitempty"`     // Faceoff winner
	LosingPlayerID      int    `json:"losingPlayerId,omitempty"`      // Faceoff loser
}

// ============================================================================
// AGGREGATED PLAY-BY-PLAY ANALYTICS
// ============================================================================

// PlayByPlayAnalytics contains aggregated analytics from play-by-play data
type PlayByPlayAnalytics struct {
	GameID        int               `json:"gameId"`
	Season        int               `json:"season"`
	GameDate      time.Time         `json:"gameDate"`
	HomeTeam      string            `json:"homeTeam"`
	AwayTeam      string            `json:"awayTeam"`
	HomeAnalytics TeamPlayAnalytics `json:"homeAnalytics"`
	AwayAnalytics TeamPlayAnalytics `json:"awayAnalytics"`
	ProcessedAt   time.Time         `json:"processedAt"`
	DataSource    string            `json:"dataSource"`
}

// TeamPlayAnalytics contains play-by-play analytics for a single team
type TeamPlayAnalytics struct {
	TeamCode string `json:"teamCode"`

	// Shot Metrics
	TotalShots   int `json:"totalShots"`
	ShotsOnGoal  int `json:"shotsOnGoal"`
	MissedShots  int `json:"missedShots"`
	BlockedShots int `json:"blockedShots"` // Shots blocked by opponent
	ShotAttempts int `json:"shotAttempts"` // Corsi (all shot attempts)

	// Expected Goals (xG)
	ExpectedGoals        float64 `json:"expectedGoals"`        // Total xG
	ExpectedGoalsAgainst float64 `json:"expectedGoalsAgainst"` // xG allowed
	XGDifferential       float64 `json:"xgDifferential"`       // xG - xGA
	ActualGoals          int     `json:"actualGoals"`
	GoalsVsExpected      float64 `json:"goalsVsExpected"` // Actual - xG (luck/skill)

	// Shot Quality
	DangerousShots    int     `json:"dangerousShots"` // High-danger chances
	HighDangerXG      float64 `json:"highDangerXG"`   // xG from high-danger
	MediumDangerShots int     `json:"mediumDangerShots"`
	LowDangerShots    int     `json:"lowDangerShots"`
	AvgShotDistance   float64 `json:"avgShotDistance"` // Feet from net
	AvgShotAngle      float64 `json:"avgShotAngle"`    // Degrees from center

	// Shot Types
	WristShots     int `json:"wristShots"`
	SlapShots      int `json:"slapShots"`
	SnapShots      int `json:"snapShots"`
	BackhandShots  int `json:"backhandShots"`
	TipInShots     int `json:"tipInShots"`
	DeflectedShots int `json:"deflectedShots"`

	// Zone Control
	OffensiveZoneTime  float64 `json:"offensiveZoneTime"` // % of game
	DefensiveZoneTime  float64 `json:"defensiveZoneTime"` // % of game
	NeutralZoneTime    float64 `json:"neutralZoneTime"`   // % of game
	OffensiveZoneShots int     `json:"offensiveZoneShots"`

	// Faceoffs
	FaceoffsWon         int     `json:"faceoffsWon"`
	FaceoffsLost        int     `json:"faceoffsLost"`
	FaceoffWinPct       float64 `json:"faceoffWinPct"`
	OffensiveZoneFOWins int     `json:"offensiveZoneFOWins"`
	DefensiveZoneFOWins int     `json:"defensiveZoneFOWins"`
	NeutralZoneFOWins   int     `json:"neutralZoneFOWins"`

	// Physical Play
	Hits            int     `json:"hits"`
	BlockedShotsFor int     `json:"blockedShotsFor"` // Blocks by this team
	Giveaways       int     `json:"giveaways"`
	Takeaways       int     `json:"takeaways"`
	PossessionRatio float64 `json:"possessionRatio"` // Takeaways / (Takeaways + Giveaways)

	// Penalties
	PenaltiesTaken   int `json:"penaltiesTaken"`
	PenaltyMinutes   int `json:"penaltyMinutes"`
	PowerPlayShots   int `json:"powerPlayShots"`
	ShortHandedShots int `json:"shortHandedShots"`

	// Advanced Metrics
	CorsiFor         int     `json:"corsiFor"` // All shot attempts
	CorsiAgainst     int     `json:"corsiAgainst"`
	CorsiForPct      float64 `json:"corsiForPct"` // CF / (CF + CA)
	FenwickFor       int     `json:"fenwickFor"`  // Unblocked shot attempts
	FenwickAgainst   int     `json:"fenwickAgainst"`
	FenwickForPct    float64 `json:"fenwickForPct"`
	ShotQualityIndex float64 `json:"shotQualityIndex"` // Avg xG per shot
	ReboundShots     int     `json:"reboundShots"`     // Shots off rebounds
}

// ============================================================================
// HISTORICAL AGGREGATION
// ============================================================================

// TeamPlayByPlayStats contains rolling averages of play-by-play metrics
type TeamPlayByPlayStats struct {
	TeamCode      string    `json:"teamCode"`
	Season        int       `json:"season"`
	GamesAnalyzed int       `json:"gamesAnalyzed"`
	LastUpdated   time.Time `json:"lastUpdated"`

	// Averages per game (Last 10 games)
	AvgExpectedGoals  float64 `json:"avgExpectedGoals"`
	AvgXGAgainst      float64 `json:"avgXGAgainst"`
	AvgXGDifferential float64 `json:"avgXGDifferential"`
	AvgShotQuality    float64 `json:"avgShotQuality"` // xG per shot
	AvgDangerousShots float64 `json:"avgDangerousShots"`

	AvgCorsiForPct       float64 `json:"avgCorsiForPct"`
	AvgFenwickForPct     float64 `json:"avgFenwickForPct"`
	AvgFaceoffWinPct     float64 `json:"avgFaceoffWinPct"`
	AvgOffensiveZoneTime float64 `json:"avgOffensiveZoneTime"`

	AvgHits            float64 `json:"avgHits"`
	AvgBlockedShots    float64 `json:"avgBlockedShots"`
	AvgPossessionRatio float64 `json:"avgPossessionRatio"`

	// Luck/Skill Indicators
	TotalGoalsVsExpected float64 `json:"totalGoalsVsExpected"` // Running total
	ShootingTalent       float64 `json:"shootingTalent"`       // Sustained over/underperformance
	GoaltendingTalent    float64 `json:"goaltendingTalent"`    // Goals allowed vs xGA

	// Trends
	XGTrend          string `json:"xgTrend"` // "improving", "declining", "stable"
	ShotQualityTrend string `json:"shotQualityTrend"`
}

// ============================================================================
// EXPECTED GOALS (xG) MODEL
// ============================================================================

// ShotLocation represents a shot location for xG calculation
type ShotLocation struct {
	X            int     `json:"x"`            // X coordinate
	Y            int     `json:"y"`            // Y coordinate
	Distance     float64 `json:"distance"`     // Distance from net (feet)
	Angle        float64 `json:"angle"`        // Angle from center (degrees)
	IsDangerZone bool    `json:"isDangerZone"` // High-danger area
	ZoneCode     string  `json:"zoneCode"`     // "O", "D", "N"
}

// ShotContext represents the context of a shot for xG
type ShotContext struct {
	Location      ShotLocation `json:"location"`
	ShotType      string       `json:"shotType"`      // "Wrist", "Slap", etc.
	IsRebound     bool         `json:"isRebound"`     // Shot off rebound
	IsRush        bool         `json:"isRush"`        // Rush chance
	IsPowerPlay   bool         `json:"isPowerPlay"`   // On power play
	IsShortHanded bool         `json:"isShortHanded"` // Shorthanded
	IsEmptyNet    bool         `json:"isEmptyNet"`    // Empty net
}

// ExpectedGoalResult represents the xG calculation result
type ExpectedGoalResult struct {
	XG              float64 `json:"xg"`          // Expected goal value (0.0-1.0)
	DangerLevel     string  `json:"dangerLevel"` // "high", "medium", "low"
	IsGoal          bool    `json:"isGoal"`      // Actual result
	Distance        float64 `json:"distance"`
	Angle           float64 `json:"angle"`
	ShotType        string  `json:"shotType"`
	ConfidenceLevel float64 `json:"confidenceLevel"` // Model confidence
}

// ============================================================================
// CACHING
// ============================================================================

// PlayByPlayCache represents cached play-by-play analytics
type PlayByPlayCache struct {
	GameID    int                  `json:"gameId"`
	Analytics *PlayByPlayAnalytics `json:"analytics"`
	CachedAt  time.Time            `json:"cachedAt"`
	TTL       time.Duration        `json:"ttl"`
}
