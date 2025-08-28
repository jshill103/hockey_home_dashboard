package models

// PlayoffOdds represents the current playoff situation and odds for a team
type PlayoffOdds struct {
	// Current Season Info
	CurrentSeason string `json:"currentSeason"`
	TeamName      string `json:"teamName"`

	// Current Standing
	CurrentRecord    string  `json:"currentRecord"` // "38-31-13"
	CurrentPoints    int     `json:"currentPoints"`
	GamesRemaining   int     `json:"gamesRemaining"`
	PointsPercentage float64 `json:"pointsPercentage"`

	// Division Standing
	DivisionName  string `json:"divisionName"`
	DivisionRank  int    `json:"divisionRank"`  // 1st, 2nd, 3rd, etc in division
	DivisionTeams int    `json:"divisionTeams"` // Total teams in division

	// Conference Standing
	ConferenceName string `json:"conferenceName"`
	ConferenceRank int    `json:"conferenceRank"` // Overall rank in conference
	WildCardRank   int    `json:"wildCardRank"`   // Rank for wild card (0 if in div top 3)

	// Playoff Position Analysis
	InPlayoffSpot      bool   `json:"inPlayoffSpot"`      // Currently in playoff position
	PlayoffSpotType    string `json:"playoffSpotType"`    // "division", "wildcard", "none"
	PointsFromPlayoffs int    `json:"pointsFromPlayoffs"` // Points from/to playoff spot
	PointsFrom8thSeed  int    `json:"pointsFrom8thSeed"`  // Points from 8th seed (last playoff spot)

	// Projections
	ProjectedPoints     int    `json:"projectedPoints"`     // Based on current pace
	ProjectedRecord     string `json:"projectedRecord"`     // Projected final record
	HistoricalThreshold int    `json:"historicalThreshold"` // Typical points needed for playoffs

	// Odds Calculation
	PlayoffOddsPercent  float64 `json:"playoffOddsPercent"`  // Overall playoff odds percentage
	DivisionOddsPercent float64 `json:"divisionOddsPercent"` // Odds of top 3 in division
	WildCardOddsPercent float64 `json:"wildCardOddsPercent"` // Odds of wild card spot

	// What Team Needs
	PointsNeeded      int     `json:"pointsNeeded"`      // Points needed for likely playoff spot
	WinsNeeded        int     `json:"winsNeeded"`        // Approximate wins needed
	RequiredPointPace float64 `json:"requiredPointPace"` // Points per game needed

	// Analysis Text
	PlayoffStatus string `json:"playoffStatus"` // "In Great Shape", "On Bubble", etc.
	KeyInsight    string `json:"keyInsight"`    // Main takeaway
	NextMilestone string `json:"nextMilestone"` // Next goal to reach
}

// ConferencePlayoffPicture shows the broader playoff race context
type ConferencePlayoffPicture struct {
	ConferenceName  string            `json:"conferenceName"`
	DivisionLeaders []PlayoffTeamInfo `json:"divisionLeaders"` // Top 3 from each division
	WildCardTeams   []PlayoffTeamInfo `json:"wildCardTeams"`   // Current wild card teams
	BubbleTeams     []PlayoffTeamInfo `json:"bubbleTeams"`     // Teams fighting for spots
	TeamPosition    PlayoffTeamInfo   `json:"teamPosition"`    // Where the team fits
}

type PlayoffTeamInfo struct {
	TeamName    string `json:"teamName"`
	TeamAbbrev  string `json:"teamAbbrev"`
	Points      int    `json:"points"`
	GamesPlayed int    `json:"gamesPlayed"`
	Record      string `json:"record"`
	PointsBack  int    `json:"pointsBack"`  // Points behind playoff line
	PlayoffSpot string `json:"playoffSpot"` // "D1", "D2", "D3", "WC1", "WC2", "OUT"
}
