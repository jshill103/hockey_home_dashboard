package models

// Standings Objects
type StandingsResponse struct {
	WildCardIndicator    bool           `json:"wildCardIndicator"`
	StandingsDateTimeUtc string         `json:"standingsDateTimeUtc"`
	Standings           []TeamStanding  `json:"standings"`
}

type TeamStanding struct {
	SeasonId              int          `json:"seasonId"`
	TeamName              TeamNameInfo `json:"teamName"`
	TeamCommonName        TeamNameInfo `json:"teamCommonName"`
	TeamAbbrev            TeamNameInfo `json:"teamAbbrev"`
	TeamLogo              string       `json:"teamLogo"`
	PlaceName             TeamNameInfo `json:"placeName"`

	// Conference/Division Info
	ConferenceName        string `json:"conferenceName"`
	ConferenceAbbrev      string `json:"conferenceAbbrev"`
	ConferenceSequence    int    `json:"conferenceSequence"`
	DivisionName          string `json:"divisionName"`
	DivisionAbbrev        string `json:"divisionAbbrev"`
	DivisionSequence      int    `json:"divisionSequence"`

	// Season Stats
	GamesPlayed           int     `json:"gamesPlayed"`
	Wins                  int     `json:"wins"`
	Losses                int     `json:"losses"`
	OtLosses              int     `json:"otLosses"`
	Ties                  int     `json:"ties"`
	Points                int     `json:"points"`
	PointPctg             float64 `json:"pointPctg"`
	WinPctg               float64 `json:"winPctg"`

	// Goal Stats
	GoalFor               int     `json:"goalFor"`
	GoalAgainst           int     `json:"goalAgainst"`
	GoalDifferential      int     `json:"goalDifferential"`
	GoalDifferentialPctg  float64 `json:"goalDifferentialPctg"`

	// Home/Road Splits
	HomeWins              int `json:"homeWins"`
	HomeLosses            int `json:"homeLosses"`
	HomeOtLosses          int `json:"homeOtLosses"`
	HomePoints            int `json:"homePoints"`
	RoadWins              int `json:"roadWins"`
	RoadLosses            int `json:"roadLosses"`
	RoadOtLosses          int `json:"roadOtLosses"`
	RoadPoints            int `json:"roadPoints"`

	// Recent Performance
	L10Wins               int    `json:"l10Wins"`
	L10Losses             int    `json:"l10Losses"`
	L10OtLosses           int    `json:"l10OtLosses"`
	L10Points             int    `json:"l10Points"`
	StreakCode            string `json:"streakCode"`
	StreakCount           int    `json:"streakCount"`

	// Playoff Status
	ClinchIndicator       string `json:"clinchIndicator"`
	WildcardSequence      int    `json:"wildcardSequence"`
}

type TeamNameInfo struct {
	Default string `json:"default"`
	Fr      string `json:"fr,omitempty"`
} 