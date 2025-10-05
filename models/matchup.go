package models

import "time"

// MatchupHistory tracks head-to-head history between two teams
type MatchupHistory struct {
	TeamA       string    `json:"teamA"`       // First team code
	TeamB       string    `json:"teamB"`       // Second team code
	LastUpdated time.Time `json:"lastUpdated"` // When last updated

	// Overall Record
	TotalGames int `json:"totalGames"` // Total games played
	TeamAWins  int `json:"teamAWins"`  // Team A wins
	TeamBWins  int `json:"teamBWins"`  // Team B wins
	OTGames    int `json:"otGames"`    // Games decided in OT/SO

	// Recent Performance (last 10 games)
	Recent10Games     int    `json:"recent10Games"`     // Games in last 10
	Recent10TeamAWins int    `json:"recent10TeamAWins"` // Team A wins in last 10
	Recent10TeamBWins int    `json:"recent10TeamBWins"` // Team B wins in last 10
	RecentTrend       string `json:"recentTrend"`       // "A_winning", "B_winning", "even"

	// Scoring Trends
	AvgGoalsTeamA    float64 `json:"avgGoalsTeamA"`    // Average goals by Team A
	AvgGoalsTeamB    float64 `json:"avgGoalsTeamB"`    // Average goals by Team B
	AvgTotalGoals    float64 `json:"avgTotalGoals"`    // Average total goals
	HighScoringGames int     `json:"highScoringGames"` // Games with 6+ goals
	LowScoringGames  int     `json:"lowScoringGames"`  // Games with <4 goals

	// Home/Away Splits
	HomeTeamWins  int     `json:"homeTeamWins"`  // Home team wins
	AwayTeamWins  int     `json:"awayTeamWins"`  // Away team wins
	HomeAdvantage float64 `json:"homeAdvantage"` // % home team wins

	// Venue-Specific (Team A at Team B, Team B at Team A)
	TeamAAtTeamB VenueRecord `json:"teamAAtTeamB"` // A's record in B's building
	TeamBAtTeamA VenueRecord `json:"teamBAtTeamA"` // B's record in A's building

	// Rivalry & Context
	IsRivalry       bool `json:"isRivalry"`       // Known rivalry
	IsDivisionGame  bool `json:"isDivisionGame"`  // Same division
	PlayoffHistory  int  `json:"playoffHistory"`  // Playoff series played
	LastPlayoffYear int  `json:"lastPlayoffYear"` // Last playoff matchup

	// Recency
	DaysSinceLastGame int       `json:"daysSinceLastGame"` // Days since last matchup
	LastGameDate      time.Time `json:"lastGameDate"`      // Date of last game
	LastGameScore     string    `json:"lastGameScore"`     // e.g., "4-3"
	LastGameWinner    string    `json:"lastGameWinner"`    // Team code

	// Game Results (store recent games for analysis)
	RecentGames []MatchupGame `json:"recentGames"` // Last 10 games
}

// VenueRecord tracks team performance in a specific venue
type VenueRecord struct {
	Games           int     `json:"games"`           // Games played
	Wins            int     `json:"wins"`            // Wins
	Losses          int     `json:"losses"`          // Losses
	WinPct          float64 `json:"winPct"`          // Win percentage
	AvgGoalsFor     float64 `json:"avgGoalsFor"`     // Average goals scored
	AvgGoalsAgainst float64 `json:"avgGoalsAgainst"` // Average goals allowed
}

// MatchupGame represents a single game in the matchup history
type MatchupGame struct {
	GameID       string    `json:"gameId"`
	Date         time.Time `json:"date"`
	HomeTeam     string    `json:"homeTeam"`
	AwayTeam     string    `json:"awayTeam"`
	HomeScore    int       `json:"homeScore"`
	AwayScore    int       `json:"awayScore"`
	Winner       string    `json:"winner"`
	OvertimeGame bool      `json:"overtimeGame"`
	Season       string    `json:"season"` // e.g., "2024-2025"
}

// MatchupIndex stores all matchup histories
type MatchupIndex struct {
	Matchups      map[string]*MatchupHistory `json:"matchups"` // Key: "TEAMA-TEAMB" (alphabetical)
	LastUpdated   time.Time                  `json:"lastUpdated"`
	TotalMatchups int                        `json:"totalMatchups"`
}

// MatchupAdvantage represents the calculated advantage based on matchup history
type MatchupAdvantage struct {
	HomeTeam string `json:"homeTeam"`
	AwayTeam string `json:"awayTeam"`

	// Historical Advantage
	HistoricalAdvantage float64 `json:"historicalAdvantage"` // -0.15 to +0.15
	RecentAdvantage     float64 `json:"recentAdvantage"`     // -0.10 to +0.10
	VenueAdvantage      float64 `json:"venueAdvantage"`      // -0.05 to +0.05

	// Rivalry & Context
	RivalryBoost        float64 `json:"rivalryBoost"`        // 0 to +0.05
	DivisionGameBoost   float64 `json:"divisionGameBoost"`   // 0 to +0.02
	PlayoffRematchBoost float64 `json:"playoffRematchBoost"` // 0 to +0.03

	// Total Impact
	TotalAdvantage  float64 `json:"totalAdvantage"`  // -0.20 to +0.20
	ConfidenceLevel float64 `json:"confidenceLevel"` // 0-1 based on sample size

	// Explanation
	Reasoning  string   `json:"reasoning"`  // Human-readable explanation
	KeyFactors []string `json:"keyFactors"` // Key factors list
}

// RivalryDefinition defines known NHL rivalries
type RivalryDefinition struct {
	Team1       string   `json:"team1"`
	Team2       string   `json:"team2"`
	RivalryName string   `json:"rivalryName"`
	Intensity   float64  `json:"intensity"`  // 0-1, how intense
	Historical  bool     `json:"historical"` // Long-standing rivalry
	Reasons     []string `json:"reasons"`    // Why it's a rivalry
}

// DivisionInfo maps teams to their divisions
type DivisionInfo struct {
	Atlantic     []string `json:"atlantic"`
	Metropolitan []string `json:"metropolitan"`
	Central      []string `json:"central"`
	Pacific      []string `json:"pacific"`
}

// MatchupPredictionFactors extends PredictionFactors with matchup data
type MatchupPredictionFactors struct {
	// Matchup History
	HeadToHeadAdvantage float64 `json:"headToHeadAdvantage"` // -0.15 to +0.15
	RecentMatchupTrend  float64 `json:"recentMatchupTrend"`  // -0.10 to +0.10
	VenueSpecificRecord float64 `json:"venueSpecificRecord"` // -0.05 to +0.05

	// Rivalry & Context
	IsRivalryGame    bool    `json:"isRivalryGame"`
	IsDivisionGame   bool    `json:"isDivisionGame"`
	IsPlayoffRematch bool    `json:"isPlayoffRematch"`
	RivalryIntensity float64 `json:"rivalryIntensity"` // 0-1

	// Game Context
	PlayoffImplication float64 `json:"playoffImplication"` // 0-1, how important
	MustWinSituation   bool    `json:"mustWinSituation"`   // Elimination/clinching
	MeaninglessGame    bool    `json:"meaninglessGame"`    // Both teams eliminated

	// Historical Performance
	GamesPlayed          int     `json:"gamesPlayed"`          // Total H2H games
	DaysSinceLastMeeting int     `json:"daysSinceLastMeeting"` // Recency
	AverageGoalDiff      float64 `json:"averageGoalDiff"`      // Goal differential
}
