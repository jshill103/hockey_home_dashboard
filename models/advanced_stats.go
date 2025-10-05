package models

import "time"

// AdvancedTeamStats represents comprehensive advanced analytics for a team
type AdvancedTeamStats struct {
	TeamCode    string    `json:"teamCode"`
	LastUpdated time.Time `json:"lastUpdated"`

	// Expected Goals Metrics
	ExpectedGoals ExpectedGoalsStats `json:"expectedGoals"`

	// Possession Metrics
	PossessionStats PossessionMetrics `json:"possession"`

	// Zone Play Analytics
	ZonePlay ZonePlayStats `json:"zonePlay"`

	// Special Teams Advanced
	SpecialTeams SpecialTeamsAdvanced `json:"specialTeams"`

	// Goalie Advanced Metrics
	GoalieAdvanced GoalieAdvancedStats `json:"goalieAdvanced"`

	// Shot Quality & Location
	ShotQuality ShotQualityStats `json:"shotQuality"`

	// Game State Analytics
	GameSituation GameSituationStats `json:"gameSituation"`
}

// ExpectedGoalsStats - Professional xG analytics
type ExpectedGoalsStats struct {
	ExpectedGoalsFor     float64 `json:"xGF"`     // Expected goals for
	ExpectedGoalsAgainst float64 `json:"xGA"`     // Expected goals against
	ExpectedGoalsDiff    float64 `json:"xGDiff"`  // xGF - xGA
	ExpectedGoalsPct     float64 `json:"xGF_pct"` // xGF / (xGF + xGA)

	// Per Game Averages
	XGFPerGame float64 `json:"xGF_pg"` // xG For per game
	XGAPerGame float64 `json:"xGA_pg"` // xG Against per game

	// Shooting/Save Performance vs Expected
	ShootingTalent    float64 `json:"shootingTalent"`    // (GF - xGF) / xGF
	GoaltendingTalent float64 `json:"goaltendingTalent"` // (xGA - GA) / xGA

	// Shot Attempt Quality
	ShotAttemptQuality     float64 `json:"shotAttemptQuality"`     // xGF / Shots For
	ShotSuppressionQuality float64 `json:"shotSuppressionQuality"` // xGA / Shots Against
}

// PossessionMetrics - Corsi, Fenwick, and advanced possession analytics
type PossessionMetrics struct {
	// Corsi (All Shot Attempts)
	CorsiFor     float64 `json:"CF"`     // Corsi For
	CorsiAgainst float64 `json:"CA"`     // Corsi Against
	CorsiDiff    float64 `json:"CDiff"`  // CF - CA
	CorsiForPct  float64 `json:"CF_pct"` // CF / (CF + CA)

	// Fenwick (Unblocked Shot Attempts)
	FenwickFor     float64 `json:"FF"`     // Fenwick For
	FenwickAgainst float64 `json:"FA"`     // Fenwick Against
	FenwickDiff    float64 `json:"FDiff"`  // FF - FA
	FenwickForPct  float64 `json:"FF_pct"` // FF / (FF + FA)

	// High Danger Chances
	HighDangerFor     float64 `json:"HDCF"`    // High danger chances for
	HighDangerAgainst float64 `json:"HDCA"`    // High danger chances against
	HighDangerPct     float64 `json:"HDC_pct"` // HDCF / (HDCF + HDCA)

	// Shot Generation Rates
	ShotGenerationRate  float64 `json:"shotGenRate"`  // Shots For / 60 minutes
	ShotSuppressionRate float64 `json:"shotSuppRate"` // Shots Against / 60 minutes

	// Zone Time & Possession Time
	OffensiveZoneTime float64 `json:"ozTime"` // % time in offensive zone
	DefensiveZoneTime float64 `json:"dzTime"` // % time in defensive zone
	NeutralZoneTime   float64 `json:"nzTime"` // % time in neutral zone
}

// ZonePlayStats - Zone entry/exit and transition analytics
type ZonePlayStats struct {
	// Zone Entries
	OzoneEntries            float64 `json:"ozEntries"`     // Offensive zone entries
	OzoneEntriesWithControl float64 `json:"ozEntriesCtrl"` // Controlled entries
	OzoneEntryPct           float64 `json:"ozEntryPct"`    // % of controlled entries

	// Zone Exits
	DzoneExits            float64 `json:"dzExits"`     // Defensive zone exits
	DzoneExitsWithControl float64 `json:"dzExitsCtrl"` // Controlled exits
	DzoneExitPct          float64 `json:"dzExitPct"`   // % of controlled exits

	// Neutral Zone Play
	NzRebounds  float64 `json:"nzRebounds"`  // Neutral zone rebounds
	NzTurnovers float64 `json:"nzTurnovers"` // Neutral zone turnovers

	// Transition Play
	TransitionFor        float64 `json:"transFor"`     // Transition opportunities for
	TransitionAgainst    float64 `json:"transAgainst"` // Transition opportunities against
	TransitionEfficiency float64 `json:"transEff"`     // Goals from transition / opportunities
}

// SpecialTeamsAdvanced - Enhanced special teams analytics
type SpecialTeamsAdvanced struct {
	// Power Play Advanced
	PowerPlayXG         float64 `json:"ppXG"`        // Power play expected goals
	PowerPlayEfficiency float64 `json:"ppEff"`       // PP goals / PP xG
	PowerPlayZoneTime   float64 `json:"ppZoneTime"`  // PP offensive zone time %
	PowerPlaySetupRate  float64 `json:"ppSetupRate"` // PP scoring chances / minute

	// Penalty Kill Advanced
	PenaltyKillXGA        float64 `json:"pkXGA"`      // Penalty kill xG against
	PenaltyKillEfficiency float64 `json:"pkEff"`      // (PK xGA - PK GA) / PK xGA
	PenaltyKillPressure   float64 `json:"pkPressure"` // PK offensive zone time %
	ShorthandedThreats    float64 `json:"shThreats"`  // Shorthanded scoring chances

	// Special Teams Discipline
	PenaltyDifferential float64 `json:"penDiff"`      // Penalties drawn - penalties taken
	MinorPenaltyRate    float64 `json:"minorPenRate"` // Minor penalties / 60 min
	MajorPenaltyRate    float64 `json:"majorPenRate"` // Major penalties / 60 min
}

// GoalieAdvancedStats - Professional goaltending analytics
type GoalieAdvancedStats struct {
	// Primary Goalie Metrics
	PrimaryGoalie      string `json:"primaryGoalie"` // Starting goalie name
	GoalieGamesStarted int    `json:"gamesStarted"`  // Games started

	// Advanced Save Metrics
	SavePctAll          float64 `json:"svPct"`   // Overall save percentage
	SavePctHighDanger   float64 `json:"svPctHD"` // High danger save %
	SavePctMediumDanger float64 `json:"svPctMD"` // Medium danger save %
	SavePctLowDanger    float64 `json:"svPctLD"` // Low danger save %

	// Expected Saves vs Actual
	ExpectedSaves      float64 `json:"xSv"`  // Expected saves based on shot quality
	SavesAboveExpected float64 `json:"svAE"` // Saves above expected (Sv - xSv)
	SavesAboveAverage  float64 `json:"svAA"` // Saves above league average

	// Goalie Style & Positioning
	ReachSaves     float64 `json:"reachSaves"`    // Saves requiring extension
	TrackingSaves  float64 `json:"trackingSaves"` // Saves with proper positioning
	ReboundControl float64 `json:"reboundCtrl"`   // % of shots with controlled rebounds

	// Breakaway & Penalty Shot
	BreakawaysSaved   float64 `json:"breakawaySv"`   // Breakaway save %
	PenaltyShotsSaved float64 `json:"penaltyShotSv"` // Penalty shot save %

	// Workload & Fatigue Indicators
	HighDangerFaced float64 `json:"hdFaced"`       // High danger shots faced / game
	WorkloadScore   float64 `json:"workloadScore"` // Overall workload rating (0-100)
	RestDaysAvg     float64 `json:"restDaysAvg"`   // Average rest between starts
}

// ShotQualityStats - Shot location and quality analytics
type ShotQualityStats struct {
	// Shot Location Breakdown
	ShotsFromSlot      float64 `json:"slotsShots"` // Shots from high-danger slot
	ShotsFromPoint     float64 `json:"pointShots"` // Shots from point
	ShotsFromWideAngle float64 `json:"wideShots"`  // Shots from wide angles

	// Shot Type Analysis
	WristShots  float64 `json:"wristShots"`  // Wrist shot attempts
	SlapShots   float64 `json:"slapShots"`   // Slap shot attempts
	SnapShots   float64 `json:"snapShots"`   // Snap shot attempts
	Deflections float64 `json:"deflections"` // Deflection attempts

	// Shot Quality Metrics
	AverageShotDistance float64 `json:"avgShotDist"`  // Average shot distance
	AverageShotAngle    float64 `json:"avgShotAngle"` // Average shot angle
	ShotQualityScore    float64 `json:"shotQuality"`  // Overall shot quality rating

	// Rebound Analytics
	ReboundShots      float64 `json:"reboundShots"` // Shots off rebounds
	ReboundGoals      float64 `json:"reboundGoals"` // Goals off rebounds
	ReboundConversion float64 `json:"reboundConv"`  // Rebound conversion %
}

// GameSituationStats - Performance in different game states
type GameSituationStats struct {
	// Score State Performance
	LeadingStats  ScoreStateStats `json:"leading"`  // When leading
	TiedStats     ScoreStateStats `json:"tied"`     // When tied
	TrailingStats ScoreStateStats `json:"trailing"` // When trailing

	// Period Performance
	FirstPeriodStats  PeriodStats `json:"period1"`  // 1st period performance
	SecondPeriodStats PeriodStats `json:"period2"`  // 2nd period performance
	ThirdPeriodStats  PeriodStats `json:"period3"`  // 3rd period performance
	OvertimeStats     PeriodStats `json:"overtime"` // OT performance

	// Clutch Situations
	CloseGameStats CloseGameStats `json:"closeGame"` // Close game performance
	BlowoutStats   BlowoutStats   `json:"blowout"`   // Blowout performance
}

// ScoreStateStats - Performance based on current score
type ScoreStateStats struct {
	Goals             float64 `json:"goals"`
	ExpectedGoals     float64 `json:"xGoals"`
	ShotAttempts      float64 `json:"shotAttempts"`
	HighDangerChances float64 `json:"hdChances"`
	TimeOnIce         float64 `json:"toi"` // Time in this state
}

// PeriodStats - Performance by period
type PeriodStats struct {
	Goals         float64 `json:"goals"`
	ExpectedGoals float64 `json:"xGoals"`
	CorsiForPct   float64 `json:"cfPct"`
	FenwickForPct float64 `json:"ffPct"`
	ZoneStartPct  float64 `json:"zoneStartPct"`
}

// CloseGameStats - Performance in close games (1-goal difference)
type CloseGameStats struct {
	Record              string  `json:"record"` // W-L-OTL in close games
	GoalsPerGame        float64 `json:"gpg"`    // Goals per game in close games
	GoalsAgainstPerGame float64 `json:"gapg"`   // Goals against per game
	PowerPlayPct        float64 `json:"ppPct"`  // PP% in close games
	PenaltyKillPct      float64 `json:"pkPct"`  // PK% in close games
	SavePct             float64 `json:"svPct"`  // Save% in close games
}

// BlowoutStats - Performance in blowout games (3+ goal difference)
type BlowoutStats struct {
	BlowoutWins      int             `json:"blowoutWins"`   // Games won by 3+ goals
	BlowoutLosses    int             `json:"blowoutLosses"` // Games lost by 3+ goals
	AverageMargin    float64         `json:"avgMargin"`     // Average goal margin in blowouts
	GarbageTimeStats ScoreStateStats `json:"garbageTime"`   // Performance when game decided
}
