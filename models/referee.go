package models

import "time"

// Referee represents an NHL referee with their career statistics
type Referee struct {
	RefereeID        int       `json:"refereeId"`        // Unique identifier
	FirstName        string    `json:"firstName"`        // First name
	LastName         string    `json:"lastName"`         // Last name
	FullName         string    `json:"fullName"`         // Full name
	JerseyNumber     int       `json:"jerseyNumber"`     // Jersey number
	Active           bool      `json:"active"`           // Currently active referee
	CareerGames      int       `json:"careerGames"`      // Total career games officiated
	SeasonGames      int       `json:"seasonGames"`      // Games this season
	LastUpdated      time.Time `json:"lastUpdated"`      // When this data was last updated
	ExternalSourceID string    `json:"externalSourceId"` // ID from external source (e.g., ScoutingTheRefs)
}

// RefereeSeasonStats represents a referee's statistics for a season
type RefereeSeasonStats struct {
	RefereeID           int       `json:"refereeId"`           // Reference to referee
	Season              int       `json:"season"`              // Season (e.g., 20242025)
	GamesOfficiated     int       `json:"gamesOfficiated"`     // Total games officiated
	TotalPenalties      int       `json:"totalPenalties"`      // Total penalties called
	AvgPenaltiesPerGame float64   `json:"avgPenaltiesPerGame"` // Average penalties per game
	MinorPenalties      int       `json:"minorPenalties"`      // Minor penalties called
	MajorPenalties      int       `json:"majorPenalties"`      // Major penalties called
	Misconducts         int       `json:"misconducts"`         // Misconducts called
	GameMisconducts     int       `json:"gameMisconducts"`     // Game misconducts
	MatchPenalties      int       `json:"matchPenalties"`      // Match penalties
	PenaltyShotsCalled  int       `json:"penaltyShotsCalled"`  // Penalty shots awarded
	HomeWinPct          float64   `json:"homeWinPct"`          // Home team win percentage
	AwayWinPct          float64   `json:"awayWinPct"`          // Away team win percentage
	AvgHomeScore        float64   `json:"avgHomeScore"`        // Average home team score
	AvgAwayScore        float64   `json:"avgAwayScore"`        // Average away team score
	AvgTotalGoals       float64   `json:"avgTotalGoals"`       // Average total goals per game
	OverPct             float64   `json:"overPct"`             // Over percentage for betting
	UnderPct            float64   `json:"underPct"`            // Under percentage for betting
	LastUpdated         time.Time `json:"lastUpdated"`         // When this data was last updated
}

// RefereeTeamBias represents a referee's historical penalty calling against specific teams
type RefereeTeamBias struct {
	RefereeID           int       `json:"refereeId"`           // Reference to referee
	TeamCode            string    `json:"teamCode"`            // Team abbreviation
	Season              int       `json:"season"`              // Season
	GamesOfficiated     int       `json:"gamesOfficiated"`     // Games with this team
	TotalPenalties      int       `json:"totalPenalties"`      // Total penalties against team
	AvgPenaltiesPerGame float64   `json:"avgPenaltiesPerGame"` // Avg penalties per game
	HomeGames           int       `json:"homeGames"`           // Games when team was home
	AwayGames           int       `json:"awayGames"`           // Games when team was away
	HomePenalties       int       `json:"homePenalties"`       // Penalties when team at home
	AwayPenalties       int       `json:"awayPenalties"`       // Penalties when team away
	TeamWins            int       `json:"teamWins"`            // Times team won with this ref
	TeamLosses          int       `json:"teamLosses"`          // Times team lost with this ref
	WinPct              float64   `json:"winPct"`              // Win percentage
	BiasScore           float64   `json:"biasScore"`           // Calculated bias score (+ favors team, - against team)
	LastUpdated         time.Time `json:"lastUpdated"`         // When this data was last updated
}

// RefereeGameAssignment represents a referee assigned to a specific game
type RefereeGameAssignment struct {
	GameID       int       `json:"gameId"`       // NHL game ID
	GameDate     time.Time `json:"gameDate"`     // Game date
	HomeTeam     string    `json:"homeTeam"`     // Home team code
	AwayTeam     string    `json:"awayTeam"`     // Away team code
	Referee1ID   int       `json:"referee1Id"`   // First referee ID
	Referee1Name string    `json:"referee1Name"` // First referee name
	Referee2ID   int       `json:"referee2Id"`   // Second referee ID
	Referee2Name string    `json:"referee2Name"` // Second referee name
	Linesman1ID  int       `json:"linesman1Id"`  // First linesman ID (optional)
	Linesman1Name string   `json:"linesman1Name"` // First linesman name
	Linesman2ID  int       `json:"linesman2Id"`  // Second linesman ID (optional)
	Linesman2Name string   `json:"linesman2Name"` // Second linesman name
	Source       string    `json:"source"`       // Data source (e.g., "ScoutingTheRefs")
	LastUpdated  time.Time `json:"lastUpdated"`  // When this assignment was recorded
}

// RefereeImpactAnalysis represents the calculated impact of a referee on game predictions
type RefereeImpactAnalysis struct {
	GameID              int     `json:"gameId"`              // NHL game ID
	Referee1ID          int     `json:"referee1Id"`          // First referee
	Referee2ID          int     `json:"referee2Id"`          // Second referee
	HomeTeamBiasScore   float64 `json:"homeTeamBiasScore"`   // Bias toward home team (+ favors, - against)
	AwayTeamBiasScore   float64 `json:"awayTeamBiasScore"`   // Bias toward away team (+ favors, - against)
	ExpectedPenalties   float64 `json:"expectedPenalties"`   // Expected total penalties
	HomeAdvantageAdjust float64 `json:"homeAdvantageAdjust"` // Adjustment to home advantage
	TotalGoalsImpact    float64 `json:"totalGoalsImpact"`    // Impact on total goals (O/U)
	ConfidenceLevel     float64 `json:"confidenceLevel"`     // Confidence in this analysis (0-1)
	Notes               string  `json:"notes"`               // Analysis notes
}

// RefereeDailySchedule represents the day's referee assignments
type RefereeDailySchedule struct {
	Date        time.Time                `json:"date"`        // Schedule date
	Assignments []RefereeGameAssignment  `json:"assignments"` // All assignments for this date
	LastUpdated time.Time                `json:"lastUpdated"` // When this schedule was fetched
}

// RefereeTendency represents a referee's overall calling patterns and tendencies
type RefereeTendency struct {
	RefereeID             int       `json:"refereeId"`             // Reference to referee
	Season                int       `json:"season"`                // Season
	TendencyType          string    `json:"tendencyType"`          // "lenient", "average", "strict"
	PenaltyCallRate       float64   `json:"penaltyCallRate"`       // Penalties per game vs league avg
	HomeAdvantageImpact   float64   `json:"homeAdvantageImpact"`   // How much ref impacts home advantage
	HighScoringGames      bool      `json:"highScoringGames"`      // Tends to have high-scoring games
	MinorPenaltyRate      float64   `json:"minorPenaltyRate"`      // % of penalties that are minors
	MajorPenaltyRate      float64   `json:"majorPenaltyRate"`      // % of penalties that are majors
	PowerPlayImpact       float64   `json:"powerPlayImpact"`       // How much ref impacts PP opportunities
	ConsistencyScore      float64   `json:"consistencyScore"`      // How consistent is the referee (0-1)
	HomeWinBias           float64   `json:"homeWinBias"`           // Tendency toward home team wins
	OverUnderTendency     string    `json:"overUnderTendency"`     // "over", "under", "neutral"
	PhysicalityTolerance  string    `json:"physicalityTolerance"`  // "low", "medium", "high"
	LastUpdated           time.Time `json:"lastUpdated"`           // When this analysis was updated
}

// RefereeProfile is a comprehensive profile combining all referee data and analytics
type RefereeProfile struct {
	Referee          Referee               `json:"referee"`          // Basic referee info
	CurrentStats     *RefereeSeasonStats   `json:"currentStats"`     // Current season stats
	Tendencies       *RefereeTendency      `json:"tendencies"`       // Calculated tendencies
	TeamBiases       []RefereeTeamBias     `json:"teamBiases"`       // All team biases
	RecentGames      []RefereeGameAssignment `json:"recentGames"`    // Recent game assignments
	CareerSummary    map[string]interface{} `json:"careerSummary"`   // Career statistics
	PredictionImpact map[string]float64    `json:"predictionImpact"` // Impact factors for predictions
	ConfidenceLevel  float64               `json:"confidenceLevel"`  // Data confidence (0-1)
	LastUpdated      time.Time             `json:"lastUpdated"`      // When this profile was generated
}

