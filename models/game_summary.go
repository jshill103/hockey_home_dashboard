package models

import "time"

// GameSummaryResponse represents the NHL API game summary response
type GameSummaryResponse struct {
	ID               int                `json:"id"`
	Season           int                `json:"season"`
	GameType         int                `json:"gameType"`
	GameDate         string             `json:"gameDate"`
	Venue            VenueInfo          `json:"venue"`
	StartTimeUTC     string             `json:"startTimeUTC"`
	HomeTeam         SummaryTeam        `json:"homeTeam"`
	AwayTeam         SummaryTeam        `json:"awayTeam"`
	GameState        string             `json:"gameState"`
	GameStateText    string             `json:"gameStateText"`
	Period           int                `json:"period"`
	PeriodDescriptor PeriodDescriptor   `json:"periodDescriptor"`
	GameOutcome      SummaryGameOutcome `json:"gameOutcome"`
	Summary          GameSummary        `json:"summary"`
}

// SummaryTeam contains team data from game summary
type SummaryTeam struct {
	Abbrev      string               `json:"abbrev"`
	ID          int                  `json:"id"`
	Score       int                  `json:"score"`
	ShotsOnGoal int                  `json:"shotsOnGoal"`
	TeamStats   SummaryTeamStats     `json:"teamStats"`
	PlayerStats []SummaryPlayerStats `json:"playerStats,omitempty"`
}

// SummaryTeamStats contains comprehensive team statistics from summary
type SummaryTeamStats struct {
	// Basic Stats
	Goals        int `json:"goals"`
	ShotsOnGoal  int `json:"shotsOnGoal"`
	Shots        int `json:"shots"`
	Hits         int `json:"hits"`
	BlockedShots int `json:"blockedShots"`
	Giveaways    int `json:"giveaways"`
	Takeaways    int `json:"takeaways"`

	// Faceoffs
	FaceoffWins   int     `json:"faceoffWins"`
	FaceoffLosses int     `json:"faceoffLosses"`
	FaceoffWinPct float64 `json:"faceoffWinPct"`

	// Power Play
	PowerPlayGoals int     `json:"powerPlayGoals"`
	PowerPlayShots int     `json:"powerPlayShots"`
	PowerPlayPct   float64 `json:"powerPlayPct"`
	PowerPlayTime  float64 `json:"powerPlayTime"` // Minutes

	// Penalty Kill
	PenaltyKillGoals int     `json:"penaltyKillGoals"`
	PenaltyKillShots int     `json:"penaltyKillShots"`
	PenaltyKillPct   float64 `json:"penaltyKillPct"`
	PenaltyKillTime  float64 `json:"penaltyKillTime"` // Minutes

	// Advanced Stats
	TimeOnAttack      float64 `json:"timeOnAttack"`  // Minutes
	TimeOnDefense     float64 `json:"timeOnDefense"` // Minutes
	ControlledEntries int     `json:"controlledEntries"`
	ControlledExits   int     `json:"controlledExits"`

	// Zone Time
	OffensiveZoneTime float64 `json:"offensiveZoneTime"` // Minutes
	DefensiveZoneTime float64 `json:"defensiveZoneTime"` // Minutes
	NeutralZoneTime   float64 `json:"neutralZoneTime"`   // Minutes

	// Shot Quality
	HighDangerShots   int `json:"highDangerShots"`
	MediumDangerShots int `json:"mediumDangerShots"`
	LowDangerShots    int `json:"lowDangerShots"`

	// Penalties
	Penalties      int `json:"penalties"`
	PenaltyMinutes int `json:"penaltyMinutes"`
}

// SummaryPlayerStats contains individual player stats from summary
type SummaryPlayerStats struct {
	PlayerID        int    `json:"playerId"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Position        string `json:"position"`
	Goals           int    `json:"goals"`
	Assists         int    `json:"assists"`
	Points          int    `json:"points"`
	PlusMinus       int    `json:"plusMinus"`
	Shots           int    `json:"shots"`
	Hits            int    `json:"hits"`
	BlockedShots    int    `json:"blockedShots"`
	Giveaways       int    `json:"giveaways"`
	Takeaways       int    `json:"takeaways"`
	FaceoffWins     int    `json:"faceoffWins"`
	FaceoffLosses   int    `json:"faceoffLosses"`
	TimeOnIce       string `json:"timeOnIce"`       // e.g., "20:15"
	PowerPlayTime   string `json:"powerPlayTime"`   // e.g., "3:45"
	ShortHandedTime string `json:"shortHandedTime"` // e.g., "1:30"
	Penalties       int    `json:"penalties"`
	PenaltyMinutes  int    `json:"penaltyMinutes"`
}

// SummaryGameOutcome contains game result information from summary
type SummaryGameOutcome struct {
	LastPeriodType    string `json:"lastPeriodType"`
	WinningTeamID     int    `json:"winningTeamId,omitempty"`
	WinningTeamAbbrev string `json:"winningTeamAbbrev,omitempty"`
}

// GameSummaryInfo contains overall game summary information
type GameSummaryInfo struct {
	GameID         int    `json:"gameId"`
	Season         int    `json:"season"`
	GameDate       string `json:"gameDate"`
	HomeTeam       string `json:"homeTeam"`
	AwayTeam       string `json:"awayTeam"`
	FinalScore     string `json:"finalScore"`
	GameState      string `json:"gameState"`
	TotalShots     int    `json:"totalShots"`
	TotalHits      int    `json:"totalHits"`
	TotalPenalties int    `json:"totalPenalties"`
	TotalFaceoffs  int    `json:"totalFaceoffs"`
}

// ============================================================================
// AGGREGATED GAME SUMMARY ANALYTICS
// ============================================================================

// GameSummaryAnalytics contains aggregated analytics from game summary data
type GameSummaryAnalytics struct {
	GameID        int                  `json:"gameId"`
	Season        int                  `json:"season"`
	GameDate      time.Time            `json:"gameDate"`
	HomeTeam      string               `json:"homeTeam"`
	AwayTeam      string               `json:"awayTeam"`
	HomeAnalytics TeamSummaryAnalytics `json:"homeAnalytics"`
	AwayAnalytics TeamSummaryAnalytics `json:"awayAnalytics"`
	ProcessedAt   time.Time            `json:"processedAt"`
	DataSource    string               `json:"dataSource"`
}

// TeamSummaryAnalytics contains game summary analytics for a single team
type TeamSummaryAnalytics struct {
	TeamCode string `json:"teamCode"`

	// Enhanced Physical Play Metrics
	Hits              int     `json:"hits"`
	BlockedShots      int     `json:"blockedShots"`
	Giveaways         int     `json:"giveaways"`
	Takeaways         int     `json:"takeaways"`
	PhysicalPlayIndex float64 `json:"physicalPlayIndex"` // Hits + Blocks
	PossessionRatio   float64 `json:"possessionRatio"`   // Takeaways / (Takeaways + Giveaways)

	// Enhanced Faceoff Performance
	FaceoffWins   int     `json:"faceoffWins"`
	FaceoffLosses int     `json:"faceoffLosses"`
	FaceoffWinPct float64 `json:"faceoffWinPct"`

	// Enhanced Special Teams
	PowerPlayGoals    int     `json:"powerPlayGoals"`
	PowerPlayShots    int     `json:"powerPlayShots"`
	PowerPlayPct      float64 `json:"powerPlayPct"`
	PowerPlayTime     float64 `json:"powerPlayTime"` // Minutes
	PenaltyKillGoals  int     `json:"penaltyKillGoals"`
	PenaltyKillShots  int     `json:"penaltyKillShots"`
	PenaltyKillPct    float64 `json:"penaltyKillPct"`
	PenaltyKillTime   float64 `json:"penaltyKillTime"`   // Minutes
	SpecialTeamsIndex float64 `json:"specialTeamsIndex"` // Combined PP% + PK%

	// Enhanced Zone Control & Time on Attack
	TimeOnAttack      float64 `json:"timeOnAttack"`      // Minutes
	TimeOnDefense     float64 `json:"timeOnDefense"`     // Minutes
	OffensiveZoneTime float64 `json:"offensiveZoneTime"` // Minutes
	DefensiveZoneTime float64 `json:"defensiveZoneTime"` // Minutes
	NeutralZoneTime   float64 `json:"neutralZoneTime"`   // Minutes
	ZoneControlRatio  float64 `json:"zoneControlRatio"`  // Offensive / (Offensive + Defensive)

	// Enhanced Transition Play
	ControlledEntries    int     `json:"controlledEntries"`
	ControlledExits      int     `json:"controlledExits"`
	TransitionEfficiency float64 `json:"transitionEfficiency"` // Entries / (Entries + Exits)

	// Enhanced Shot Quality
	ShotsOnGoal       int     `json:"shotsOnGoal"`
	Shots             int     `json:"shots"`
	ShotAccuracy      float64 `json:"shotAccuracy"` // SOG / Shots
	HighDangerShots   int     `json:"highDangerShots"`
	MediumDangerShots int     `json:"mediumDangerShots"`
	LowDangerShots    int     `json:"lowDangerShots"`
	ShotQualityIndex  float64 `json:"shotQualityIndex"` // Weighted shot quality

	// Penalty Discipline
	Penalties       int     `json:"penalties"`
	PenaltyMinutes  int     `json:"penaltyMinutes"`
	DisciplineIndex float64 `json:"disciplineIndex"` // Lower = more disciplined
}

// ============================================================================
// HISTORICAL GAME SUMMARY TRACKING
// ============================================================================

// TeamSummaryHistory contains rolling game summary analytics for a team
type TeamSummaryHistory struct {
	TeamCode      string    `json:"teamCode"`
	Season        int       `json:"season"`
	GamesAnalyzed int       `json:"gamesAnalyzed"`
	LastUpdated   time.Time `json:"lastUpdated"`

	// Rolling Averages (Last 10 games)
	AvgHits            float64 `json:"avgHits"`
	AvgBlockedShots    float64 `json:"avgBlockedShots"`
	AvgPhysicalPlay    float64 `json:"avgPhysicalPlay"`
	AvgPossessionRatio float64 `json:"avgPossessionRatio"`

	AvgFaceoffWinPct  float64 `json:"avgFaceoffWinPct"`
	AvgPowerPlayPct   float64 `json:"avgPowerPlayPct"`
	AvgPenaltyKillPct float64 `json:"avgPenaltyKillPct"`
	AvgSpecialTeams   float64 `json:"avgSpecialTeams"`

	AvgTimeOnAttack  float64 `json:"avgTimeOnAttack"`
	AvgZoneControl   float64 `json:"avgZoneControl"`
	AvgTransitionEff float64 `json:"avgTransitionEff"`

	AvgShotQuality float64 `json:"avgShotQuality"`
	AvgDiscipline  float64 `json:"avgDiscipline"`

	// Team Identity
	PhysicalTeam     bool `json:"physicalTeam"`     // High hits/blocks
	PossessionTeam   bool `json:"possessionTeam"`   // High takeaways, low giveaways
	SpecialTeamsTeam bool `json:"specialTeamsTeam"` // Strong PP/PK
	ZoneControlTeam  bool `json:"zoneControlTeam"`  // High time on attack
	DisciplinedTeam  bool `json:"disciplinedTeam"`  // Low penalties
	ShotQualityTeam  bool `json:"shotQualityTeam"`  // High danger shots
}

// GameSummaryCache represents cached game summary analytics
type GameSummaryCache struct {
	GameID    int                   `json:"gameId"`
	Analytics *GameSummaryAnalytics `json:"analytics"`
	CachedAt  time.Time             `json:"cachedAt"`
	TTL       time.Duration         `json:"ttl"`
}
