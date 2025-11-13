package models

import "time"

// HeadToHeadRecord tracks historical performance between two specific teams
type HeadToHeadRecord struct {
	HomeTeam          string        `json:"homeTeam"`
	AwayTeam          string        `json:"awayTeam"`
	Season            string        `json:"season"`
	TotalGames        int           `json:"totalGames"`
	HomeWins          int           `json:"homeWins"`
	AwayWins          int           `json:"awayWins"`
	Ties              int           `json:"ties"`              // For overtime/shootout records
	RecentGames       []H2HGame     `json:"recentGames"`       // Last 10 H2H games
	AvgHomeGoals      float64       `json:"avgHomeGoals"`
	AvgAwayGoals      float64       `json:"avgAwayGoals"`
	LastMeetingDate   time.Time     `json:"lastMeetingDate"`
	WeightedAdvantage float64       `json:"weightedAdvantage"` // Composite score (-1 to +1, positive favors home)
	HomeWinStreak     int           `json:"homeWinStreak"`     // Current streak (positive = wins, negative = losses)
	DaysSinceLastMeet int           `json:"daysSinceLastMeet"`
	
	// Advanced metrics
	HomeBlowoutWins   int     `json:"homeBlowoutWins"`   // Wins by 3+ goals
	AwayBlowoutWins   int     `json:"awayBlowoutWins"`
	CloseGames        int     `json:"closeGames"`        // Games decided by 1 goal
	OvertimeGames     int     `json:"overtimeGames"`
	
	// Goalie head-to-head (if available)
	TopGoalieMatchups []GoalieH2H `json:"topGoalieMatchups,omitempty"`
	
	LastUpdated       time.Time `json:"lastUpdated"`
}

// H2HGame represents a single head-to-head game result
type H2HGame struct {
	GameID      int       `json:"gameID"`
	Date        time.Time `json:"date"`
	HomeScore   int       `json:"homeScore"`
	AwayScore   int       `json:"awayScore"`
	Winner      string    `json:"winner"`      // Team code of winner
	WasBlowout  bool      `json:"wasBlowout"`  // 3+ goal differential
	WasOvertime bool      `json:"wasOvertime"`
	HomeGoalie  string    `json:"homeGoalie,omitempty"`
	AwayGoalie  string    `json:"awayGoalie,omitempty"`
	Recency     float64   `json:"recency"`     // Weight factor based on how recent (1.0 = most recent, decays exponentially)
}

// GoalieH2H tracks goalie performance in this specific matchup
type GoalieH2H struct {
	GoalieName string  `json:"goalieName"`
	Team       string  `json:"team"`
	Starts     int     `json:"starts"`
	Wins       int     `json:"wins"`
	SavePct    float64 `json:"savePct"`
	GAA        float64 `json:"gaa"`
}

// HeadToHeadSummary provides a quick overview for API responses
type HeadToHeadSummary struct {
	Matchup           string  `json:"matchup"`           // "BOS vs NYR"
	TotalMeetings     int     `json:"totalMeetings"`
	HomeAdvantage     float64 `json:"homeAdvantage"`     // -1 to +1
	RecentFormHome    string  `json:"recentFormHome"`    // "W-L-W-W-L" (last 5)
	RecentFormAway    string  `json:"recentFormAway"`
	AvgGoalDiff       float64 `json:"avgGoalDiff"`       // Home - Away
	LastMeeting       string  `json:"lastMeeting"`       // "2025-10-15"
	DaysSinceLastMeet int     `json:"daysSinceLastMeet"`
	PredictedImpact   string  `json:"predictedImpact"`   // "Strong home advantage", "Even matchup", etc.
}

// HeadToHeadAdvantage calculates the advantage for a specific matchup
type HeadToHeadAdvantage struct {
	HomeTeam        string  `json:"homeTeam"`
	AwayTeam        string  `json:"awayTeam"`
	Advantage       float64 `json:"advantage"`       // -0.30 to +0.30 (adjustment to home win probability)
	Confidence      float64 `json:"confidence"`      // 0.0 to 1.0 (based on sample size)
	RecentTrend     string  `json:"recentTrend"`     // "Home dominant", "Away surge", "Balanced"
	KeyFactors      []string `json:"keyFactors"`     // Reasons for the advantage
	SampleSize      int      `json:"sampleSize"`     // Number of games in analysis
	RecencyBias     float64  `json:"recencyBias"`    // How much recent games are weighted
}

