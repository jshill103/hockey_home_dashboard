package models

import "time"

// PreGameLineup represents the lineup information for an upcoming game
type PreGameLineup struct {
	GameID      int         `json:"gameId"`
	GameDate    time.Time   `json:"gameDate"`
	HomeTeam    string      `json:"homeTeam"`
	AwayTeam    string      `json:"awayTeam"`
	HomeLineup  *TeamLineup `json:"homeLineup,omitempty"`
	AwayLineup  *TeamLineup `json:"awayLineup,omitempty"`
	IsAvailable bool        `json:"isAvailable"`
	LastUpdated time.Time   `json:"lastUpdated"`
	DataSource  string      `json:"dataSource"`
}

// TeamLineup represents a single team's lineup for a game
type TeamLineup struct {
	TeamCode       string            `json:"teamCode"`
	StartingGoalie *LineupGoalie     `json:"startingGoalie,omitempty"`
	BackupGoalie   *LineupGoalie     `json:"backupGoalie,omitempty"`
	ForwardLines   []ForwardLine     `json:"forwardLines,omitempty"`
	DefensePairs   []DefensePair     `json:"defensePairs,omitempty"`
	Scratches      []ScratchedPlayer `json:"scratches,omitempty"`
	ExtraSkaters   []LineupPlayer    `json:"extraSkaters,omitempty"`
}

// LineupGoalie represents a goalie in the lineup
type LineupGoalie struct {
	PlayerID      int    `json:"playerId"`
	PlayerName    string `json:"playerName"`
	SweaterNumber int    `json:"sweaterNumber"`
	IsStarting    bool   `json:"isStarting"`
}

// ForwardLine represents a line of forwards
type ForwardLine struct {
	LineNumber int           `json:"lineNumber"` // 1-4
	LeftWing   *LineupPlayer `json:"leftWing,omitempty"`
	Center     *LineupPlayer `json:"center,omitempty"`
	RightWing  *LineupPlayer `json:"rightWing,omitempty"`
}

// DefensePair represents a defensive pairing
type DefensePair struct {
	PairNumber   int           `json:"pairNumber"` // 1-3
	LeftDefense  *LineupPlayer `json:"leftDefense,omitempty"`
	RightDefense *LineupPlayer `json:"rightDefense,omitempty"`
}

// LineupPlayer represents a skater in the lineup
type LineupPlayer struct {
	PlayerID      int    `json:"playerId"`
	PlayerName    string `json:"playerName"`
	SweaterNumber int    `json:"sweaterNumber"`
	Position      string `json:"position"` // C, LW, RW, D
}

// ScratchedPlayer represents a player who is scratched
type ScratchedPlayer struct {
	PlayerID   int    `json:"playerId"`
	PlayerName string `json:"playerName"`
	Position   string `json:"position"`
	Reason     string `json:"reason,omitempty"` // injury, healthy scratch, etc.
}

// LineupCache represents a cached lineup with metadata
type LineupCache struct {
	Lineup   *PreGameLineup `json:"lineup"`
	CachedAt time.Time      `json:"cachedAt"`
	TTL      time.Duration  `json:"ttl"`
}

// GameCenterLineupResponse represents the raw API response for lineup data
// This maps to the NHL API's gamecenter/{gameID}/boxscore structure
type GameCenterLineupResponse struct {
	ID          int              `json:"id"`
	GameDate    string           `json:"gameDate"`
	HomeTeam    BoxscoreTeam     `json:"homeTeam"`
	AwayTeam    BoxscoreTeam     `json:"awayTeam"`
	RosterSpots []RosterSpot     `json:"rosterSpots,omitempty"`
	Scratches   map[string][]int `json:"scratches,omitempty"` // "home" or "away" -> player IDs
	Summary     *LineupSummary   `json:"summary,omitempty"`
}

// RosterSpot represents a player's roster spot in the game
type RosterSpot struct {
	PlayerID      int    `json:"playerId"`
	FirstName     Name   `json:"firstName"`
	LastName      Name   `json:"lastName"`
	SweaterNumber int    `json:"sweaterNumber"`
	Position      string `json:"positionCode"`
	TeamID        int    `json:"teamId"`
}

// LineupSummary contains summary information including starting goalies
type LineupSummary struct {
	Goalies []GoalieLineupInfo `json:"goalies,omitempty"`
}

// GoalieLineupInfo contains goalie-specific lineup information
type GoalieLineupInfo struct {
	PlayerID      int  `json:"playerId"`
	TeamID        int  `json:"teamId"`
	FirstName     Name `json:"firstName"`
	LastName      Name `json:"lastName"`
	SweaterNumber int  `json:"sweaterNumber"`
	IsStarter     bool `json:"starter,omitempty"`
}
