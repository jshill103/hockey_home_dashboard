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

	// ML Simulation Data
	MLSimulations int     `json:"mlSimulations"` // Number of Monte Carlo simulations (e.g., 5000)
	MLAvgPoints   float64 `json:"mlAvgPoints"`   // Average final points from simulations
	MLBestCase    int     `json:"mlBestCase"`    // Best case scenario (max points)
	MLWorstCase   int     `json:"mlWorstCase"`   // Worst case scenario (min points)
	
	// Schedule Strength (Phase 2)
	ScheduleDifficulty        float64 `json:"scheduleDifficulty,omitempty"`        // 0-10 scale
	ScheduleDifficultyTier    string  `json:"scheduleDifficultyTier,omitempty"`    // "Easy", "Average", "Hard", "Brutal"
	AvgOpponentWinPct         float64 `json:"avgOpponentWinPct,omitempty"`         // Average opponent win percentage
	HomeGamesRemaining        int     `json:"homeGamesRemaining,omitempty"`        // Home games left
	AwayGamesRemaining        int     `json:"awayGamesRemaining,omitempty"`        // Away games left
	DivisionGamesRemaining    int     `json:"divisionGamesRemaining,omitempty"`    // Division games left
	PlayoffTeamGamesRemaining int     `json:"playoffTeamGamesRemaining,omitempty"` // Games vs playoff teams
	CrucialGamesCount         int     `json:"crucialGamesCount,omitempty"`         // High-importance games
	MustWinGamesCount         int     `json:"mustWinGamesCount,omitempty"`         // Critical must-win games
	BackToBackGames           int     `json:"backToBackGames,omitempty"`           // Back-to-back games remaining
	CrucialGames              []map[string]interface{} `json:"crucialGames,omitempty"` // Detailed crucial game info (Phase 4.4)
	
	// Percentile Distributions (Phase 4.2)
	PercentileP10 int `json:"percentileP10,omitempty"` // 10th percentile projection (pessimistic)
	PercentileP25 int `json:"percentileP25,omitempty"` // 25th percentile projection
	PercentileP50 int `json:"percentileP50,omitempty"` // 50th percentile (median)
	PercentileP75 int `json:"percentileP75,omitempty"` // 75th percentile projection
	PercentileP90 int `json:"percentileP90,omitempty"` // 90th percentile projection (optimistic)
	
	// Magic Numbers (Phase 4.1)
	MagicNumber           int    `json:"magicNumber,omitempty"`           // Points needed to clinch
	MagicNumberWins       int    `json:"magicNumberWins,omitempty"`       // Approximate wins needed to clinch
	CanClinchPlayoffs     bool   `json:"canClinchPlayoffs,omitempty"`     // Can still make playoffs
	ClinchScenario        string `json:"clinchScenario,omitempty"`        // Human-readable clinch scenario
	EliminationNumber     int    `json:"eliminationNumber,omitempty"`     // Points buffer before elimination
	CanBeEliminated       bool   `json:"canBeEliminated,omitempty"`       // Can still be eliminated
	EliminationScenario   string `json:"eliminationScenario,omitempty"`   // Human-readable elimination scenario
	MaxPossiblePoints     int    `json:"maxPossiblePoints,omitempty"`     // Maximum points if win all remaining
	PointsBehind8th       int    `json:"pointsBehind8th,omitempty"`       // Points behind 8th place
	PointsAhead9th        int    `json:"pointsAhead9th,omitempty"`        // Points ahead of 9th place
	TiebreakerAdvantage   string `json:"tiebreakerAdvantage,omitempty"`   // "favorable", "unfavorable", "neutral"
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
