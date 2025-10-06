package models

import "time"

// ShiftData represents the NHL API shift data response
type ShiftDataResponse struct {
	GameID   int       `json:"id"`
	Season   int       `json:"season"`
	GameType int       `json:"gameType"`
	GameDate string    `json:"gameDate"`
	HomeTeam ShiftTeam `json:"homeTeam"`
	AwayTeam ShiftTeam `json:"awayTeam"`
}

// ShiftTeam contains shift data for one team
type ShiftTeam struct {
	Abbrev  string        `json:"abbrev"`
	ID      int           `json:"id"`
	Players []PlayerShift `json:"players,omitempty"`
}

// PlayerShift contains all shifts for a single player
type PlayerShift struct {
	PlayerID  int     `json:"playerId"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Shifts    []Shift `json:"shifts"`
}

// Shift represents a single shift on the ice
type Shift struct {
	Period      int    `json:"period"`
	StartTime   string `json:"startTime"` // e.g., "00:00"
	EndTime     string `json:"endTime"`   // e.g., "01:23"
	Duration    string `json:"duration"`  // e.g., "1:23"
	EventNumber int    `json:"eventNumber,omitempty"`
}

// ============================================================================
// AGGREGATED SHIFT ANALYTICS
// ============================================================================

// ShiftAnalytics contains aggregated shift analysis for a game
type ShiftAnalytics struct {
	GameID        int                `json:"gameId"`
	Season        int                `json:"season"`
	GameDate      time.Time          `json:"gameDate"`
	HomeTeam      string             `json:"homeTeam"`
	AwayTeam      string             `json:"awayTeam"`
	HomeAnalytics TeamShiftAnalytics `json:"homeAnalytics"`
	AwayAnalytics TeamShiftAnalytics `json:"awayAnalytics"`
	ProcessedAt   time.Time          `json:"processedAt"`
	DataSource    string             `json:"dataSource"`
}

// TeamShiftAnalytics contains shift analytics for a single team
type TeamShiftAnalytics struct {
	TeamCode string `json:"teamCode"`

	// Overall Team Metrics
	TotalShifts    int     `json:"totalShifts"`
	AvgShiftLength float64 `json:"avgShiftLength"` // Seconds
	TotalIceTime   float64 `json:"totalIceTime"`   // Minutes

	// Player Metrics
	PlayersUsed      int     `json:"playersUsed"`
	TopPlayerMinutes float64 `json:"topPlayerMinutes"` // Most ice time
	TopPlayerShifts  int     `json:"topPlayerShifts"`  // Most shifts

	// Line Combinations (Top 5 most used)
	TopLineCombos []LineCombination `json:"topLineCombos"`

	// Fatigue Indicators
	LongShifts     int `json:"longShifts"`     // Shifts > 60 seconds
	VeryLongShifts int `json:"veryLongShifts"` // Shifts > 90 seconds
	ShortShifts    int `json:"shortShifts"`    // Shifts < 30 seconds

	// Player Details
	PlayerShiftStats []PlayerShiftStats `json:"playerShiftStats"`
}

// LineCombination represents players who played together frequently
type LineCombination struct {
	Players        []string `json:"players"` // Player IDs
	PlayerNames    []string `json:"playerNames"`
	TimeTogether   float64  `json:"timeTogether"` // Minutes
	ShiftsTogether int      `json:"shiftsTogether"`
	ChemistryScore float64  `json:"chemistryScore"` // 0-1 based on time together
}

// PlayerShiftStats contains shift statistics for a single player
type PlayerShiftStats struct {
	PlayerID       int     `json:"playerId"`
	PlayerName     string  `json:"playerName"`
	Position       string  `json:"position"` // F, D, G
	TotalShifts    int     `json:"totalShifts"`
	TotalIceTime   float64 `json:"totalIceTime"`   // Minutes
	AvgShiftLength float64 `json:"avgShiftLength"` // Seconds
	LongestShift   float64 `json:"longestShift"`   // Seconds
	ShortestShift  float64 `json:"shortestShift"`  // Seconds
	FatigueScore   float64 `json:"fatigueScore"`   // 0-1 (higher = more fatigued)
	UsageRate      float64 `json:"usageRate"`      // % of game (0-100)
}

// ============================================================================
// HISTORICAL SHIFT TRACKING
// ============================================================================

// TeamShiftHistory contains rolling shift analytics for a team
type TeamShiftHistory struct {
	TeamCode      string    `json:"teamCode"`
	Season        int       `json:"season"`
	GamesAnalyzed int       `json:"gamesAnalyzed"`
	LastUpdated   time.Time `json:"lastUpdated"`

	// Rolling Averages (Last 10 games)
	AvgShiftLength float64 `json:"avgShiftLength"` // Team avg shift length
	AvgPlayersUsed float64 `json:"avgPlayersUsed"` // Roster depth usage
	AvgLongShifts  float64 `json:"avgLongShifts"`  // Fatigue indicator

	// Line Stability
	LineConsistency float64 `json:"lineConsistency"` // 0-1 (higher = more stable lines)
	TopLineMinutes  float64 `json:"topLineMinutes"`  // Top line average TOI

	// Coaching Tendencies
	RollerCoaster bool `json:"rollerCoaster"` // Frequent line changes?
	ShortBench    bool `json:"shortBench"`    // Relies heavily on top players?
	BalancedLines bool `json:"balancedLines"` // Even ice time distribution?

	// Key Player Workloads
	TopPlayer        string  `json:"topPlayer"`        // Most used player
	TopPlayerAvgTOI  float64 `json:"topPlayerAvgTOI"`  // Minutes per game
	TopPlayerFatigue float64 `json:"topPlayerFatigue"` // 0-1 fatigue score
}

// ShiftCache represents cached shift analytics
type ShiftCache struct {
	GameID    int             `json:"gameId"`
	Analytics *ShiftAnalytics `json:"analytics"`
	CachedAt  time.Time       `json:"cachedAt"`
	TTL       time.Duration   `json:"ttl"`
}
