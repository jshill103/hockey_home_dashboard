package models

import "time"

// BettingOdds represents betting market data for a game
type BettingOdds struct {
	GameID   int       `json:"gameId"`
	HomeTeam string    `json:"homeTeam"`
	AwayTeam string    `json:"awayTeam"`
	GameDate time.Time `json:"gameDate"`

	// Moneyline Odds
	HomeMoneyline int `json:"homeMoneyline"` // e.g., -150
	AwayMoneyline int `json:"awayMoneyline"` // e.g., +130

	// Spread/Puck Line
	HomeSpread     float64 `json:"homeSpread"`     // e.g., -1.5
	HomeSpreadOdds int     `json:"homeSpreadOdds"` // e.g., +180
	AwaySpread     float64 `json:"awaySpread"`     // e.g., +1.5
	AwaySpreadOdds int     `json:"awaySpreadOdds"` // e.g., -220

	// Total (Over/Under)
	TotalLine float64 `json:"totalLine"` // e.g., 6.5
	OverOdds  int     `json:"overOdds"`  // e.g., -110
	UnderOdds int     `json:"underOdds"` // e.g., -110

	// Market Consensus
	ImpliedHomeWinPct float64 `json:"impliedHomeWinPct"` // Derived from moneyline
	ImpliedAwayWinPct float64 `json:"impliedAwayWinPct"`

	// Line Movement
	OpeningHomeML int     `json:"openingHomeML"`
	OpeningAwayML int     `json:"openingAwayML"`
	LineMovement  float64 `json:"lineMovement"` // % change in implied probability

	// Betting Percentages
	HomeBetPct   float64 `json:"homeBetPct"`   // % of bets on home
	AwayBetPct   float64 `json:"awayBetPct"`   // % of bets on away
	HomeMoneyPct float64 `json:"homeMoneyPct"` // % of money on home
	AwayMoneyPct float64 `json:"awayMoneyPct"` // % of money on away

	// Sharp Money Indicators
	SharpMoneyOn    string `json:"sharpMoneyOn"`    // "home", "away", "none"
	ReverseLineMove bool   `json:"reverseLineMove"` // Line moving against public
	SteamMove       bool   `json:"steamMove"`       // Sudden sharp action

	// Metadata
	Bookmaker   string    `json:"bookmaker"` // Source of odds
	LastUpdated time.Time `json:"lastUpdated"`
	DataQuality float64   `json:"dataQuality"` // 0.0-1.0
}

// MarketConsensus represents aggregated market data from multiple bookmakers
type MarketConsensus struct {
	GameID   int       `json:"gameId"`
	HomeTeam string    `json:"homeTeam"`
	AwayTeam string    `json:"awayTeam"`
	GameDate time.Time `json:"gameDate"`

	// Consensus Lines
	AvgHomeMoneyline float64 `json:"avgHomeMoneyline"`
	AvgAwayMoneyline float64 `json:"avgAwayMoneyline"`
	AvgTotalLine     float64 `json:"avgTotalLine"`
	AvgHomeSpread    float64 `json:"avgHomeSpread"`

	// Consensus Probabilities
	ConsensusHomeWinPct float64 `json:"consensusHomeWinPct"`
	ConsensusAwayWinPct float64 `json:"consensusAwayWinPct"`

	// Market Efficiency
	MarketAgreement  float64 `json:"marketAgreement"`  // How much books agree (0-1)
	MarketConfidence float64 `json:"marketConfidence"` // Overall market confidence

	// Bookmaker Spread
	BestHomeOdds  int     `json:"bestHomeOdds"`
	BestAwayOdds  int     `json:"bestAwayOdds"`
	WorstHomeOdds int     `json:"worstHomeOdds"`
	WorstAwayOdds int     `json:"worstAwayOdds"`
	OddsSpread    float64 `json:"oddsSpread"` // Variance in odds

	// Sources
	NumBookmakers int       `json:"numBookmakers"`
	Bookmakers    []string  `json:"bookmakers"`
	LastUpdated   time.Time `json:"lastUpdated"`
}

// MarketSignal represents actionable betting intelligence
type MarketSignal struct {
	GameID   int       `json:"gameId"`
	HomeTeam string    `json:"homeTeam"`
	AwayTeam string    `json:"awayTeam"`
	GameDate time.Time `json:"gameDate"`

	// Signal Type
	SignalType     string  `json:"signalType"`     // "sharp_money", "reverse_line", "steam", "consensus"
	SignalStrength float64 `json:"signalStrength"` // 0.0-1.0
	SignalSide     string  `json:"signalSide"`     // "home", "away"

	// Signal Details
	Description string `json:"description"`
	Reasoning   string `json:"reasoning"`

	// Market vs Our Models
	OurPrediction    float64 `json:"ourPrediction"`    // Our win %
	MarketPrediction float64 `json:"marketPrediction"` // Market win %
	Disagreement     float64 `json:"disagreement"`     // Absolute difference
	EdgeDetected     bool    `json:"edgeDetected"`     // Do we have an edge?
	EdgeSide         string  `json:"edgeSide"`         // Who we think is undervalued
	EdgeMagnitude    float64 `json:"edgeMagnitude"`    // Size of edge (%)

	// Confidence
	Confidence  float64   `json:"confidence"` // 0.0-1.0
	LastUpdated time.Time `json:"lastUpdated"`
}

// MarketBasedAdjustment represents how market data influences our prediction
type MarketBasedAdjustment struct {
	// Market Influence
	MarketWeight       float64 `json:"marketWeight"`       // How much to trust market (0-1)
	BaselinePrediction float64 `json:"baselinePrediction"` // Our original %
	MarketPrediction   float64 `json:"marketPrediction"`   // Market %
	AdjustedPrediction float64 `json:"adjustedPrediction"` // Blended %

	// Adjustment Factors
	MarketEfficiency float64 `json:"marketEfficiency"` // Market quality
	OurConfidence    float64 `json:"ourConfidence"`    // Our model confidence
	DataRecency      float64 `json:"dataRecency"`      // How fresh is market data

	// Decision
	ShouldAdjust  bool    `json:"shouldAdjust"`  // Should we use market data?
	AdjustmentPct float64 `json:"adjustmentPct"` // % change from baseline
	Reasoning     string  `json:"reasoning"`     // Why we adjusted (or not)
}

// BettingMarketHistory tracks market movement over time
type BettingMarketHistory struct {
	GameID   int    `json:"gameId"`
	HomeTeam string `json:"homeTeam"`
	AwayTeam string `json:"awayTeam"`

	// Historical Data Points
	DataPoints []MarketDataPoint `json:"dataPoints"`

	// Movement Analysis
	TotalMovement      float64 `json:"totalMovement"`      // Total line movement
	SharpMovements     int     `json:"sharpMovements"`     // # of sharp moves
	PublicMovements    int     `json:"publicMovements"`    // # of public moves
	FinalLineDirection string  `json:"finalLineDirection"` // "toward_home", "toward_away", "stable"

	// Key Events
	KeyMovements []KeyMarketMovement `json:"keyMovements"`

	LastUpdated time.Time `json:"lastUpdated"`
}

// MarketDataPoint represents odds at a specific time
type MarketDataPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	HomeMoneyline int       `json:"homeMoneyline"`
	AwayMoneyline int       `json:"awayMoneyline"`
	TotalLine     float64   `json:"totalLine"`
	HomeBetPct    float64   `json:"homeBetPct"`
	HomeMoneyPct  float64   `json:"homeMoneyPct"`
}

// KeyMarketMovement represents significant line moves
type KeyMarketMovement struct {
	Timestamp    time.Time `json:"timestamp"`
	MovementType string    `json:"movementType"` // "sharp", "steam", "public"
	Direction    string    `json:"direction"`    // "toward_home", "toward_away"
	Magnitude    float64   `json:"magnitude"`    // Size of move
	Description  string    `json:"description"`
}
