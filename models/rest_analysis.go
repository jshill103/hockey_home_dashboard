package models

import "time"

// RestImpactAnalysis tracks team performance based on rest days
type RestImpactAnalysis struct {
	TeamCode    string    `json:"teamCode"`
	Season      string    `json:"season"`
	LastUpdated time.Time `json:"lastUpdated"`
	
	// Performance by rest days
	BackToBackRecord    TeamRestRecord `json:"backToBackRecord"`    // 0 days rest
	OneDayRestRecord    TeamRestRecord `json:"oneDayRestRecord"`    // 1 day rest
	TwoDayRestRecord    TeamRestRecord `json:"twoDayRestRecord"`    // 2 days rest
	ThreePlusRestRecord TeamRestRecord `json:"threePlusRestRecord"` // 3+ days rest
	
	// Comparative metrics
	B2BPenalty          float64 `json:"b2bPenalty"`          // Win% drop on B2B (e.g., -0.15)
	OptimalRestDays     int     `json:"optimalRestDays"`     // Days of rest for peak performance
	FatigueResistance   float64 `json:"fatigueResistance"`   // How well team handles fatigue (0-1, higher = better)
	RestSensitivity     float64 `json:"restSensitivity"`     // How much rest affects this team (0-1, higher = more sensitive)
	
	// Advanced metrics
	ShotsDeclineB2B     float64 `json:"shotsDeclineB2B"`     // Shot volume drop on B2B (e.g., -3.5 shots/game)
	SavePctDeclineB2B   float64 `json:"savePctDeclineB2B"`   // Goalie performance drop on B2B
	PowerPlayPctB2B     float64 `json:"powerPlayPctB2B"`     // PP% on back-to-backs
	PenaltyKillPctB2B   float64 `json:"penaltyKillPctB2B"`   // PK% on back-to-backs
	
	// Travel-adjusted metrics
	B2BWithTravelRecord TeamRestRecord `json:"b2bWithTravelRecord"` // B2B + significant travel
	RestAfterTravelBonus float64       `json:"restAfterTravelBonus"` // Performance boost after rest following travel
	
	// Recent trend
	Last10B2BRecord     TeamRestRecord `json:"last10B2BRecord"`     // Recent B2B performance
	ImprovingTrend      bool           `json:"improvingTrend"`      // Is B2B performance improving?
}

// TeamRestRecord stores performance data for a specific rest scenario
type TeamRestRecord struct {
	Games           int     `json:"games"`
	Wins            int     `json:"wins"`
	Losses          int     `json:"losses"`
	OTLosses        int     `json:"otLosses"`
	WinPct          float64 `json:"winPct"`
	Points          int     `json:"points"`           // Total points earned
	PointsPct       float64 `json:"pointsPct"`        // Points% (for playoff race context)
	AvgGoalsFor     float64 `json:"avgGoalsFor"`
	AvgGoalsAgainst float64 `json:"avgGoalsAgainst"`
	AvgShots        float64 `json:"avgShots"`
	AvgShotsAgainst float64 `json:"avgShotsAgainst"`
	AvgCorsiFor     float64 `json:"avgCorsiFor"`      // Possession metric
}

// RestAdvantageCalculation compares two teams' rest situations
type RestAdvantageCalculation struct {
	HomeTeam            string  `json:"homeTeam"`
	AwayTeam            string  `json:"awayTeam"`
	HomeRestDays        int     `json:"homeRestDays"`
	AwayRestDays        int     `json:"awayRestDays"`
	RestAdvantage       float64 `json:"restAdvantage"`       // -0.20 to +0.20 (positive favors home)
	HomeOnB2B           bool    `json:"homeOnB2B"`
	AwayOnB2B           bool    `json:"awayOnB2B"`
	HomeB2BPenalty      float64 `json:"homeB2BPenalty"`      // Team-specific B2B impact
	AwayB2BPenalty      float64 `json:"awayB2BPenalty"`
	HomeTravelMiles     float64 `json:"homeTravelMiles"`     // Miles traveled to this game
	AwayTravelMiles     float64 `json:"awayTravelMiles"`
	FatigueAdvantage    string  `json:"fatigueAdvantage"`    // "Home", "Away", "Neutral"
	KeyFactors          []string `json:"keyFactors"`         // Explanation of advantage
	ConfidenceLevel     float64  `json:"confidenceLevel"`    // 0-1 based on sample size
}

// RestImpactSummary provides a quick overview of rest impact
type RestImpactSummary struct {
	TeamCode           string  `json:"teamCode"`
	B2BWinPct          float64 `json:"b2bWinPct"`
	NormalWinPct       float64 `json:"normalWinPct"`
	B2BPenalty         float64 `json:"b2bPenalty"`
	FatigueResistance  float64 `json:"fatigueResistance"`
	OptimalRestDays    int     `json:"optimalRestDays"`
	Rank               int     `json:"rank"`               // Rank among all teams for B2B performance (1 = best)
	Assessment         string  `json:"assessment"`         // "Elite", "Above Average", "Below Average", "Poor"
}

// RestScenario defines a specific rest situation for analysis
type RestScenario struct {
	RestDays      int     `json:"restDays"`
	TravelMiles   float64 `json:"travelMiles"`
	IsBackToBack  bool    `json:"isBackToBack"`
	PreviousOT    bool    `json:"previousOT"`          // Previous game went to OT/SO
	TimeZoneChange int    `json:"timeZoneChange"`      // Hours of timezone change
	ExpectedImpact float64 `json:"expectedImpact"`     // Predicted impact on performance (-0.25 to +0.25)
}

