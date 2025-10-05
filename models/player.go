package models

import "time"

// PlayerStatsResponse represents the response from NHL player stats API
type PlayerStatsResponse struct {
	Data []PlayerStats `json:"data"`
}

// PlayerStats represents individual player statistics
type PlayerStats struct {
	PlayerID         int        `json:"playerId"`
	HeadshotURL      string     `json:"headshot"`
	FirstName        PlayerName `json:"firstName"`
	LastName         PlayerName `json:"lastName"`
	TeamAbbrev       string     `json:"teamAbbrev"`
	Position         string     `json:"position"`
	GamesPlayed      int        `json:"gamesPlayed"`
	Goals            int        `json:"goals"`
	Assists          int        `json:"assists"`
	Points           int        `json:"points"`
	PlusMinus        int        `json:"plusMinus"`
	PenaltyMinutes   int        `json:"penaltyMinutes"`
	PowerPlayGoals   int        `json:"powerPlayGoals"`
	ShortHandedGoals int        `json:"shortHandedGoals"`
	GameWinningGoals int        `json:"gameWinningGoals"`
	Shots            int        `json:"shots"`
	ShotPct          float64    `json:"shootingPct"`
	FaceoffWinPct    float64    `json:"faceoffWinPct"`
	AvgTimeOnIce     string     `json:"avgToi"`
}

// GoalieStatsResponse represents the response from NHL goalie stats API
type GoalieStatsResponse struct {
	Data []GoalieStats `json:"data"`
}

// ClubStatsResponse represents the response from NHL club stats API
type ClubStatsResponse struct {
	Skaters []ClubSkaterStats `json:"skaters"`
	Goalies []ClubGoalieStats `json:"goalies"`
}

// ClubSkaterStats represents a skater's stats from club-stats endpoint
type ClubSkaterStats struct {
	PlayerID          int        `json:"playerId"`
	HeadshotURL       string     `json:"headshot"`
	FirstName         PlayerName `json:"firstName"`
	LastName          PlayerName `json:"lastName"`
	SweaterNumber     int        `json:"sweaterNumber"`
	Position          string     `json:"positionCode"`
	GamesPlayed       int        `json:"gamesPlayed"`
	Goals             int        `json:"goals"`
	Assists           int        `json:"assists"`
	Points            int        `json:"points"`
	PlusMinus         int        `json:"plusMinus"`
	PenaltyMinutes    int        `json:"penaltyMinutes"`
	PowerPlayGoals    int        `json:"powerPlayGoals"`
	PowerPlayPoints   int        `json:"powerPlayPoints"`
	ShortHandedGoals  int        `json:"shorthandedGoals"`
	ShortHandedPoints int        `json:"shorthandedPoints"`
	GameWinningGoals  int        `json:"gameWinningGoals"`
	OvertimeGoals     int        `json:"overtimeGoals"`
	Shots             int        `json:"shots"`
	ShootingPct       float64    `json:"shootingPctg"`
	AvgTimeOnIce      string     `json:"avgToi"`
	FaceoffWinPct     float64    `json:"faceoffWinningPctg"`
}

// ClubGoalieStats represents a goalie's stats from club-stats endpoint
type ClubGoalieStats struct {
	PlayerID        int        `json:"playerId"`
	HeadshotURL     string     `json:"headshot"`
	FirstName       PlayerName `json:"firstName"`
	LastName        PlayerName `json:"lastName"`
	SweaterNumber   int        `json:"sweaterNumber"`
	Position        string     `json:"positionCode"`
	GamesPlayed     int        `json:"gamesPlayed"`
	GamesStarted    int        `json:"gamesStarted"`
	Wins            int        `json:"wins"`
	Losses          int        `json:"losses"`
	OvertimeLosses  int        `json:"overtimeLosses"`
	SavePct         float64    `json:"savePctg"`
	GoalsAgainstAvg float64    `json:"goalsAgainstAvg"`
	Shutouts        int        `json:"shutouts"`
	GoalsAgainst    int        `json:"goalsAgainst"`
	ShotsAgainst    int        `json:"shotsAgainst"`
	Saves           int        `json:"saves"`
	AvgTimeOnIce    string     `json:"avgToi"`
}

// GoalieStats represents individual goalie statistics
type GoalieStats struct {
	PlayerID        int        `json:"playerId"`
	HeadshotURL     string     `json:"headshot"`
	FirstName       PlayerName `json:"firstName"`
	LastName        PlayerName `json:"lastName"`
	TeamAbbrev      string     `json:"teamAbbrev"`
	GamesPlayed     int        `json:"gamesPlayed"`
	GamesStarted    int        `json:"gamesStarted"`
	Wins            int        `json:"wins"`
	Losses          int        `json:"losses"`
	Ties            int        `json:"ties"`
	OvertimeLosses  int        `json:"overtimeLosses"`
	Shutouts        int        `json:"shutouts"`
	GoalsAgainst    int        `json:"goalsAgainst"`
	ShotsAgainst    int        `json:"shotsAgainst"`
	Saves           int        `json:"saves"`
	SavePct         float64    `json:"savePct"`
	GoalsAgainstAvg float64    `json:"goalsAgainstAverage"`
	TimeOnIce       int        `json:"timeOnIce"`
}

// PlayerName represents a player's name in different languages
type PlayerName struct {
	Default string `json:"default"`
}

// TeamRosterResponse represents the team roster API response
type TeamRosterResponse struct {
	Forwards   []RosterPlayer `json:"forwards"`
	Defensemen []RosterPlayer `json:"defensemen"`
	Goalies    []RosterPlayer `json:"goalies"`
}

// RosterPlayer represents a player on the team roster
type RosterPlayer struct {
	ID                  int        `json:"id"`
	HeadshotURL         string     `json:"headshot"`
	FirstName           PlayerName `json:"firstName"`
	LastName            PlayerName `json:"lastName"`
	SweaterNumber       int        `json:"sweaterNumber"`
	Position            string     `json:"positionCode"`
	ShootsCatches       string     `json:"shootsCatches"`
	HeightInInches      int        `json:"heightInInches"`
	WeightInPounds      int        `json:"weightInPounds"`
	HeightInCentimeters int        `json:"heightInCentimeters"`
	WeightInKilograms   int        `json:"weightInKilograms"`
	BirthDate           string     `json:"birthDate"`
	BirthCity           PlayerName `json:"birthCity"`
	BirthCountry        string     `json:"birthCountry"` // API returns string, not PlayerName
}

// PlayerStatsLeaders represents the stats leaders API response
type PlayerStatsLeaders struct {
	Goals   []PlayerStats `json:"goals"`
	Assists []PlayerStats `json:"assists"`
	Points  []PlayerStats `json:"points"`
}

// GoalieStatsLeaders represents the goalie stats leaders API response
type GoalieStatsLeaders struct {
	Wins    []GoalieStats `json:"wins"`
	SavePct []GoalieStats `json:"savePct"`
	GAA     []GoalieStats `json:"goalsAgainstAverage"`
}

// PlayerGameLogResponse represents the response from NHL player game log API
type PlayerGameLogResponse struct {
	GameLog []PlayerGameLogEntry `json:"gameLog"`
}

// PlayerGameLogEntry represents a single game in a player's game log
type PlayerGameLogEntry struct {
	GameID         int    `json:"gameId"`
	GameDate       string `json:"gameDate"`
	HomeRoadFlag   string `json:"homeRoadFlag"` // "H" or "R"
	TeamAbbrev     string `json:"teamAbbrev"`
	OpponentAbbrev string `json:"opponentAbbrev"`

	// Game stats
	Goals     int `json:"goals"`
	Assists   int `json:"assists"`
	Points    int `json:"points"`
	PlusMinus int `json:"plusMinus"`

	PowerPlayGoals   int `json:"powerPlayGoals"`
	ShorthandedGoals int `json:"shorthandedGoals"`
	GameWinningGoals int `json:"gameWinningGoals"`
	OvertimeGoals    int `json:"overtimeGoals"`

	Shots  int    `json:"shots"`
	Shifts int    `json:"shifts"`
	TOI    string `json:"toi"` // Time on ice (MM:SS format)

	// Game outcome
	GameOutcome string `json:"gameOutcome"` // "W", "L", "OTL", etc.

	// Opponent info
	OpponentLogo string `json:"opponentLogo"`
}

// TeamRoster represents a validated team roster with metadata
type TeamRoster struct {
	TeamCode    string         `json:"teamCode"`
	Season      int            `json:"season"`
	Forwards    []RosterPlayer `json:"forwards"`
	Defensemen  []RosterPlayer `json:"defensemen"`
	Goalies     []RosterPlayer `json:"goalies"`
	PlayerIDs   map[int]bool   `json:"playerIds"` // For quick lookup
	LastUpdated time.Time      `json:"lastUpdated"`
	Version     string         `json:"version"`
}

// RosterChange represents a change between two rosters
type RosterChange struct {
	PlayerID   int       `json:"playerId"`
	PlayerName string    `json:"playerName"`
	Position   string    `json:"position"`
	ChangeType string    `json:"changeType"` // "added", "removed"
	FromSeason int       `json:"fromSeason"`
	ToSeason   int       `json:"toSeason"`
	DetectedAt time.Time `json:"detectedAt"`
}

// IsOnRoster checks if a player ID is on the roster
func (tr *TeamRoster) IsOnRoster(playerID int) bool {
	return tr.PlayerIDs[playerID]
}

// GetPlayerByID finds a player on the roster by ID
func (tr *TeamRoster) GetPlayerByID(playerID int) *RosterPlayer {
	// Check forwards
	for i := range tr.Forwards {
		if tr.Forwards[i].ID == playerID {
			return &tr.Forwards[i]
		}
	}

	// Check defensemen
	for i := range tr.Defensemen {
		if tr.Defensemen[i].ID == playerID {
			return &tr.Defensemen[i]
		}
	}

	// Check goalies
	for i := range tr.Goalies {
		if tr.Goalies[i].ID == playerID {
			return &tr.Goalies[i]
		}
	}

	return nil
}
