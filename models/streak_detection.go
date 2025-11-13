package models

import "time"

// StreakPattern represents a current streak for a team
type StreakPattern struct {
	TeamCode         string    `json:"teamCode"`
	Type             string    `json:"type"`             // "Win", "Loss", "Scoring", "Defense"
	Length           int       `json:"length"`           // Current streak length
	MaxLength        int       `json:"maxLength"`        // Longest this season
	StartDate        time.Time `json:"startDate"`        // When streak started
	Confidence       float64   `json:"confidence"`       // Statistical confidence (0-1)
	Historical       float64   `json:"historical"`       // How often streaks this length continue
	BreakProbability float64   `json:"breakProbability"` // Probability of ending (0-1)
	ImpactFactor     float64   `json:"impactFactor"`     // Prediction adjustment (-0.15 to +0.15)
	IsHot            bool      `json:"isHot"`            // 3+ win streak
	IsCold           bool      `json:"isCold"`           // 3+ loss streak
	IsDominant       bool      `json:"isDominant"`       // 5+ win streak
	IsInCrisis       bool      `json:"isInCrisis"`       // 5+ loss streak
}

// StreakAnalysis provides comprehensive streak analysis for a team
type StreakAnalysis struct {
	TeamCode          string          `json:"teamCode"`
	Season            string          `json:"season"`
	CurrentWinStreak  int             `json:"currentWinStreak"`
	CurrentLossStreak int             `json:"currentLossStreak"`
	LongestWinStreak  int             `json:"longestWinStreak"`
	LongestLossStreak int             `json:"longestLossStreak"`
	HomeStreaks       StreakSummary   `json:"homeStreaks"`
	AwayStreaks       StreakSummary   `json:"awayStreaks"`
	ActiveStreaks     []StreakPattern `json:"activeStreaks"`     // All current streaks
	StreakTendency    string          `json:"streakTendency"`    // "Streaky", "Consistent", "Volatile"
	AverageStreakLength float64       `json:"averageStreakLength"`
	StreakMomentum    float64         `json:"streakMomentum"`    // Rate of change in streak
	LastUpdated       time.Time       `json:"lastUpdated"`
}

// StreakSummary summarizes streak performance for a specific condition (home/away)
type StreakSummary struct {
	CurrentStreak     int     `json:"currentStreak"`     // Can be positive (wins) or negative (losses)
	LongestWinStreak  int     `json:"longestWinStreak"`
	LongestLossStreak int     `json:"longestLossStreak"`
	WinStreakCount    int     `json:"winStreakCount"`    // Number of win streaks this season
	LossStreakCount   int     `json:"lossStreakCount"`   // Number of loss streaks this season
	StreakImpact      float64 `json:"streakImpact"`      // Historical impact on performance
}

// StreakHistory tracks historical streak data
type StreakHistory struct {
	TeamCode          string               `json:"teamCode"`
	Season            string               `json:"season"`
	Streaks           []HistoricalStreak   `json:"streaks"`
	StreakBreakdowns  map[int]StreakStats  `json:"streakBreakdowns"` // Stats by streak length
	ContinuationRates map[int]float64      `json:"continuationRates"` // Probability of continuing by length
	LastUpdated       time.Time            `json:"lastUpdated"`
}

// HistoricalStreak represents a completed streak
type HistoricalStreak struct {
	Type      string    `json:"type"`      // "Win" or "Loss"
	Length    int       `json:"length"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
	Venue     string    `json:"venue"`     // "Home", "Away", "Mixed"
	Ended     bool      `json:"ended"`     // Whether streak is complete
	EndReason string    `json:"endReason"` // How it ended (if applicable)
}

// StreakStats provides statistics for streaks of a specific length
type StreakStats struct {
	Length            int     `json:"length"`
	Occurrences       int     `json:"occurrences"`       // How many times occurred
	ContinuedCount    int     `json:"continuedCount"`    // How many continued to next game
	ContinuationRate  float64 `json:"continuationRate"`  // Probability of continuing
	AverageEndLength  float64 `json:"averageEndLength"`  // Average final length when started
	Impact            float64 `json:"impact"`            // Historical impact on next game
}

// StreakBreakProbability calculates the probability of a streak ending
type StreakBreakProbability struct {
	CurrentLength     int     `json:"currentLength"`
	Type              string  `json:"type"` // "Win" or "Loss"
	BaseBreakRate     float64 `json:"baseBreakRate"`     // Historical break rate for this length
	OpponentAdjusted  float64 `json:"opponentAdjusted"`  // Adjusted for opponent strength
	VenueAdjusted     float64 `json:"venueAdjusted"`     // Adjusted for home/away
	FinalProbability  float64 `json:"finalProbability"`  // Final break probability
	Confidence        float64 `json:"confidence"`        // Confidence in calculation
}

// StreakComparison compares streaks between two teams
type StreakComparison struct {
	HomeTeam         string        `json:"homeTeam"`
	AwayTeam         string        `json:"awayTeam"`
	HomeStreak       StreakPattern `json:"homeStreak"`
	AwayStreak       StreakPattern `json:"awayStreak"`
	Advantage        string        `json:"advantage"`        // "Home", "Away", "Neutral"
	ImpactDifferential float64     `json:"impactDifferential"` // Net impact on prediction
	Confidence       float64       `json:"confidence"`
	Analysis         string        `json:"analysis"`         // Human-readable analysis
}

// StreakMomentum tracks the acceleration of streaks
type StreakMomentum struct {
	CurrentStreak    int     `json:"currentStreak"`
	PreviousStreak   int     `json:"previousStreak"`
	Acceleration     float64 `json:"acceleration"`     // Rate of change
	IsAccelerating   bool    `json:"isAccelerating"`   // Getting stronger
	IsDecelerating   bool    `json:"isDecelerating"`   // Getting weaker
	MomentumFactor   float64 `json:"momentumFactor"`   // Additional adjustment
}

