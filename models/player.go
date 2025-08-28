package models

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
	BirthCountry        PlayerName `json:"birthCountry"`
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
