package models

import "time"

// LandingPageResponse represents the NHL API landing page response
type LandingPageResponse struct {
	ID               int                `json:"id"`
	Season           int                `json:"season"`
	GameType         int                `json:"gameType"`
	GameDate         string             `json:"gameDate"`
	Venue            VenueInfo          `json:"venue"`
	StartTimeUTC     string             `json:"startTimeUTC"`
	HomeTeam         LandingTeam        `json:"homeTeam"`
	AwayTeam         LandingTeam        `json:"awayTeam"`
	GameState        string             `json:"gameState"`
	GameStateText    string             `json:"gameStateText"`
	Period           int                `json:"period"`
	PeriodDescriptor PeriodDescriptor   `json:"periodDescriptor"`
	GameOutcome      LandingGameOutcome `json:"gameOutcome"`
}

// LandingTeam contains detailed team stats from landing page
type LandingTeam struct {
	Abbrev      string               `json:"abbrev"`
	ID          int                  `json:"id"`
	Score       int                  `json:"score"`
	ShotsOnGoal int                  `json:"shotsOnGoal"`
	TeamStats   LandingTeamStats     `json:"teamStats"`
	PlayerStats []LandingPlayerStats `json:"playerStats,omitempty"`
}

// LandingTeamStats contains comprehensive team statistics
type LandingTeamStats struct {
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

	// Penalty Kill
	PenaltyKillGoals int     `json:"penaltyKillGoals"`
	PenaltyKillShots int     `json:"penaltyKillShots"`
	PenaltyKillPct   float64 `json:"penaltyKillPct"`

	// Advanced Stats
	TimeOnAttack      float64 `json:"timeOnAttack"`  // Minutes
	TimeOnDefense     float64 `json:"timeOnDefense"` // Minutes
	ControlledEntries int     `json:"controlledEntries"`
	ControlledExits   int     `json:"controlledExits"`

	// Zone Time
	OffensiveZoneTime float64 `json:"offensiveZoneTime"` // Minutes
	DefensiveZoneTime float64 `json:"defensiveZoneTime"` // Minutes
	NeutralZoneTime   float64 `json:"neutralZoneTime"`   // Minutes
}

// LandingPlayerStats contains individual player stats from landing page
type LandingPlayerStats struct {
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
}

// LandingGameOutcome contains game result information from landing page
type LandingGameOutcome struct {
	LastPeriodType    string `json:"lastPeriodType"`
	WinningTeamID     int    `json:"winningTeamId,omitempty"`
	WinningTeamAbbrev string `json:"winningTeamAbbrev,omitempty"`
}

// ============================================================================
// AGGREGATED LANDING PAGE ANALYTICS
// ============================================================================

// LandingPageAnalytics contains aggregated analytics from landing page data
type LandingPageAnalytics struct {
	GameID        int                  `json:"gameId"`
	Season        int                  `json:"season"`
	GameDate      time.Time            `json:"gameDate"`
	HomeTeam      string               `json:"homeTeam"`
	AwayTeam      string               `json:"awayTeam"`
	HomeAnalytics TeamLandingAnalytics `json:"homeAnalytics"`
	AwayAnalytics TeamLandingAnalytics `json:"awayAnalytics"`
	ProcessedAt   time.Time            `json:"processedAt"`
	DataSource    string               `json:"dataSource"`
}

// TeamLandingAnalytics contains landing page analytics for a single team
type TeamLandingAnalytics struct {
	TeamCode string `json:"teamCode"`

	// Physical Play Metrics
	Hits              int     `json:"hits"`
	BlockedShots      int     `json:"blockedShots"`
	Giveaways         int     `json:"giveaways"`
	Takeaways         int     `json:"takeaways"`
	PhysicalPlayIndex float64 `json:"physicalPlayIndex"` // Hits + Blocks
	PossessionRatio   float64 `json:"possessionRatio"`   // Takeaways / (Takeaways + Giveaways)

	// Faceoff Performance
	FaceoffWins   int     `json:"faceoffWins"`
	FaceoffLosses int     `json:"faceoffLosses"`
	FaceoffWinPct float64 `json:"faceoffWinPct"`

	// Special Teams
	PowerPlayGoals   int     `json:"powerPlayGoals"`
	PowerPlayShots   int     `json:"powerPlayShots"`
	PowerPlayPct     float64 `json:"powerPlayPct"`
	PenaltyKillGoals int     `json:"penaltyKillGoals"`
	PenaltyKillShots int     `json:"penaltyKillShots"`
	PenaltyKillPct   float64 `json:"penaltyKillPct"`

	// Zone Control & Time on Attack
	TimeOnAttack      float64 `json:"timeOnAttack"`      // Minutes
	TimeOnDefense     float64 `json:"timeOnDefense"`     // Minutes
	OffensiveZoneTime float64 `json:"offensiveZoneTime"` // Minutes
	DefensiveZoneTime float64 `json:"defensiveZoneTime"` // Minutes
	NeutralZoneTime   float64 `json:"neutralZoneTime"`   // Minutes
	ZoneControlRatio  float64 `json:"zoneControlRatio"`  // Offensive / (Offensive + Defensive)

	// Transition Play
	ControlledEntries    int     `json:"controlledEntries"`
	ControlledExits      int     `json:"controlledExits"`
	TransitionEfficiency float64 `json:"transitionEfficiency"` // Entries / (Entries + Exits)

	// Shot Quality
	ShotsOnGoal  int     `json:"shotsOnGoal"`
	Shots        int     `json:"shots"`
	ShotAccuracy float64 `json:"shotAccuracy"` // SOG / Shots
}

// ============================================================================
// HISTORICAL LANDING PAGE TRACKING
// ============================================================================

// TeamLandingHistory contains rolling landing page analytics for a team
type TeamLandingHistory struct {
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

	AvgTimeOnAttack  float64 `json:"avgTimeOnAttack"`
	AvgZoneControl   float64 `json:"avgZoneControl"`
	AvgTransitionEff float64 `json:"avgTransitionEff"`

	// Team Identity
	PhysicalTeam     bool `json:"physicalTeam"`     // High hits/blocks
	PossessionTeam   bool `json:"possessionTeam"`   // High takeaways, low giveaways
	SpecialTeamsTeam bool `json:"specialTeamsTeam"` // Strong PP/PK
	ZoneControlTeam  bool `json:"zoneControlTeam"`  // High time on attack
}

// LandingPageCache represents cached landing page analytics
type LandingPageCache struct {
	GameID    int                   `json:"gameId"`
	Analytics *LandingPageAnalytics `json:"analytics"`
	CachedAt  time.Time             `json:"cachedAt"`
	TTL       time.Duration         `json:"ttl"`
}
