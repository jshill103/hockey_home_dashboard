package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// MarketDataService integrates betting market data for enhanced predictions
type MarketDataService struct {
	apiKeys     map[string]string
	cache       map[string]*MarketData
	cacheMutex  sync.RWMutex
	lastUpdate  time.Time
	rateLimiter *RateLimiter
}

// MarketData represents betting market information
type MarketData struct {
	GameID         string              `json:"gameId"`
	HomeTeam       string              `json:"homeTeam"`
	AwayTeam       string              `json:"awayTeam"`
	MoneyLine      MoneyLineOdds       `json:"moneyLine"`
	Spread         SpreadOdds          `json:"spread"`
	TotalGoals     TotalOdds           `json:"totalGoals"`
	PropBets       []PropBet           `json:"propBets"`
	MarketMovement []OddsMovement      `json:"marketMovement"`
	Volume         BettingVolume       `json:"volume"`
	SharpMoney     SharpMoneyIndicator `json:"sharpMoney"`
	PublicBetting  PublicBettingData   `json:"publicBetting"`
	LastUpdated    time.Time           `json:"lastUpdated"`
	Confidence     float64             `json:"confidence"`
}

// MoneyLineOdds represents money line betting odds
type MoneyLineOdds struct {
	HomeOdds        int     `json:"homeOdds"`        // e.g., -150
	AwayOdds        int     `json:"awayOdds"`        // e.g., +130
	HomeImpliedProb float64 `json:"homeImpliedProb"` // Implied probability
	AwayImpliedProb float64 `json:"awayImpliedProb"` // Implied probability
	Vig             float64 `json:"vig"`             // Bookmaker's edge
	ConsensusLine   int     `json:"consensusLine"`   // Average across books
}

// SpreadOdds represents puck line/spread betting
type SpreadOdds struct {
	HomeSpread     float64 `json:"homeSpread"`     // e.g., -1.5
	HomeSpreadOdds int     `json:"homeSpreadOdds"` // e.g., +120
	AwaySpread     float64 `json:"awaySpread"`     // e.g., +1.5
	AwaySpreadOdds int     `json:"awaySpreadOdds"` // e.g., -140
	ImpliedMargin  float64 `json:"impliedMargin"`  // Expected goal differential
}

// TotalOdds represents over/under total goals
type TotalOdds struct {
	Total        float64 `json:"total"`        // e.g., 6.5
	OverOdds     int     `json:"overOdds"`     // e.g., -110
	UnderOdds    int     `json:"underOdds"`    // e.g., -110
	ImpliedTotal float64 `json:"impliedTotal"` // Market's expected total
}

// PropBet represents proposition bets
type PropBet struct {
	Type       string  `json:"type"`       // "player_goals", "team_shots", etc.
	Player     string  `json:"player"`     // Player name if applicable
	Line       float64 `json:"line"`       // Prop line
	OverOdds   int     `json:"overOdds"`   // Over odds
	UnderOdds  int     `json:"underOdds"`  // Under odds
	Confidence float64 `json:"confidence"` // Market confidence
}

// OddsMovement tracks how odds have changed
type OddsMovement struct {
	Timestamp time.Time `json:"timestamp"`
	HomeOdds  int       `json:"homeOdds"`
	AwayOdds  int       `json:"awayOdds"`
	TotalLine float64   `json:"totalLine"`
	Volume    int64     `json:"volume"`
	Direction string    `json:"direction"` // "home", "away", "over", "under"
}

// BettingVolume represents betting activity
type BettingVolume struct {
	TotalHandle      int64   `json:"totalHandle"`      // Total money wagered
	HomePercent      float64 `json:"homePercent"`      // % of bets on home
	AwayPercent      float64 `json:"awayPercent"`      // % of bets on away
	HomeMoneyPercent float64 `json:"homeMoneyPercent"` // % of money on home
	AwayMoneyPercent float64 `json:"awayMoneyPercent"` // % of money on away
	BetCount         int     `json:"betCount"`         // Number of bets
}

// SharpMoneyIndicator detects professional betting activity
type SharpMoneyIndicator struct {
	IsSharpAction   bool    `json:"isSharpAction"`   // Sharp money detected
	SharpSide       string  `json:"sharpSide"`       // Which side sharps favor
	SharpConfidence float64 `json:"sharpConfidence"` // Confidence in sharp action
	ReverseLineMove bool    `json:"reverseLineMove"` // Line moved against public
	SteamMove       bool    `json:"steamMove"`       // Rapid line movement
	Explanation     string  `json:"explanation"`     // Why it's sharp action
}

// PublicBettingData represents public betting patterns
type PublicBettingData struct {
	PublicFavorite   string  `json:"publicFavorite"`   // Team public favors
	PublicPercentage float64 `json:"publicPercentage"` // % of public bets
	FadeValue        float64 `json:"fadeValue"`        // Value in fading public
	ContrarianSignal bool    `json:"contrarianSignal"` // Strong contrarian opportunity
}

// NewMarketDataService creates a new market data service
func NewMarketDataService() *MarketDataService {
	return &MarketDataService{
		apiKeys: map[string]string{
			"odds_api":   "", // The Odds API
			"sportsbook": "", // Sportsbook API
			"pinnacle":   "", // Pinnacle API
			"betfair":    "", // Betfair Exchange API
		},
		cache:       make(map[string]*MarketData),
		rateLimiter: NewRateLimiter(60, time.Minute), // 60 requests per minute
	}
}

// GetMarketData fetches current betting market data for a game
func (mds *MarketDataService) GetMarketData(homeTeam, awayTeam string, gameDate time.Time) (*MarketData, error) {
	gameID := fmt.Sprintf("%s_vs_%s_%s", homeTeam, awayTeam, gameDate.Format("2006-01-02"))

	// Check cache first
	mds.cacheMutex.RLock()
	if cached, exists := mds.cache[gameID]; exists {
		if time.Since(cached.LastUpdated) < 5*time.Minute {
			mds.cacheMutex.RUnlock()
			return cached, nil
		}
	}
	mds.cacheMutex.RUnlock()

	// Rate limit check
	if !mds.rateLimiter.Allow("market_data") {
		return nil, fmt.Errorf("rate limit exceeded for market data")
	}

	log.Printf("📊 Fetching market data for %s vs %s", homeTeam, awayTeam)

	// Fetch from multiple sources and aggregate
	marketData, err := mds.fetchAndAggregateMarketData(homeTeam, awayTeam, gameDate)
	if err != nil {
		return nil, err
	}

	// Cache the result
	mds.cacheMutex.Lock()
	mds.cache[gameID] = marketData
	mds.cacheMutex.Unlock()

	return marketData, nil
}

// fetchAndAggregateMarketData fetches from multiple sources
func (mds *MarketDataService) fetchAndAggregateMarketData(homeTeam, awayTeam string, gameDate time.Time) (*MarketData, error) {
	// In a real implementation, this would fetch from multiple sportsbooks
	// For now, we'll simulate market data

	marketData := &MarketData{
		GameID:      fmt.Sprintf("%s_vs_%s_%s", homeTeam, awayTeam, gameDate.Format("2006-01-02")),
		HomeTeam:    homeTeam,
		AwayTeam:    awayTeam,
		LastUpdated: time.Now(),
		Confidence:  0.85,
	}

	// Simulate realistic NHL betting odds
	marketData.MoneyLine = mds.generateMoneyLineOdds(homeTeam, awayTeam)
	marketData.Spread = mds.generateSpreadOdds()
	marketData.TotalGoals = mds.generateTotalOdds()
	marketData.PropBets = mds.generatePropBets(homeTeam, awayTeam)
	marketData.MarketMovement = mds.generateMarketMovement()
	marketData.Volume = mds.generateBettingVolume()
	marketData.SharpMoney = mds.detectSharpMoney(marketData)
	marketData.PublicBetting = mds.analyzePublicBetting(marketData)

	return marketData, nil
}

// generateMoneyLineOdds creates realistic money line odds
func (mds *MarketDataService) generateMoneyLineOdds(homeTeam, awayTeam string) MoneyLineOdds {
	// Simulate based on team strength (simplified)
	teamStrengths := map[string]float64{
		"UTA": 0.48, "COL": 0.65, "VGK": 0.62, "SJS": 0.35, "LAK": 0.55,
		"EDM": 0.70, "CGY": 0.52, "WPG": 0.58, "MIN": 0.50, "CHI": 0.45,
	}

	homeStrength := teamStrengths[homeTeam]
	if homeStrength == 0 {
		homeStrength = 0.50 // Default
	}

	awayStrength := teamStrengths[awayTeam]
	if awayStrength == 0 {
		awayStrength = 0.50 // Default
	}

	// Add home ice advantage
	homeAdvantage := homeStrength + 0.05

	// Calculate implied probabilities
	totalStrength := homeAdvantage + awayStrength
	homeImpliedProb := homeAdvantage / totalStrength
	awayImpliedProb := awayStrength / totalStrength

	// Convert to American odds
	homeOdds := mds.probabilityToAmericanOdds(homeImpliedProb)
	awayOdds := mds.probabilityToAmericanOdds(awayImpliedProb)

	// Calculate vig
	vig := (homeImpliedProb + awayImpliedProb) - 1.0

	return MoneyLineOdds{
		HomeOdds:        homeOdds,
		AwayOdds:        awayOdds,
		HomeImpliedProb: homeImpliedProb,
		AwayImpliedProb: awayImpliedProb,
		Vig:             vig,
		ConsensusLine:   homeOdds, // Simplified
	}
}

// generateSpreadOdds creates puck line odds
func (mds *MarketDataService) generateSpreadOdds() SpreadOdds {
	return SpreadOdds{
		HomeSpread:     -1.5,
		HomeSpreadOdds: 120,
		AwaySpread:     1.5,
		AwaySpreadOdds: -140,
		ImpliedMargin:  0.8, // Expected goal differential
	}
}

// generateTotalOdds creates over/under odds
func (mds *MarketDataService) generateTotalOdds() TotalOdds {
	return TotalOdds{
		Total:        6.5,
		OverOdds:     -110,
		UnderOdds:    -110,
		ImpliedTotal: 6.3, // Market's expected total
	}
}

// generatePropBets creates proposition bet data
func (mds *MarketDataService) generatePropBets(homeTeam, awayTeam string) []PropBet {
	return []PropBet{
		{
			Type:       "team_total_goals",
			Player:     homeTeam,
			Line:       3.5,
			OverOdds:   105,
			UnderOdds:  -125,
			Confidence: 0.78,
		},
		{
			Type:       "team_total_goals",
			Player:     awayTeam,
			Line:       2.5,
			OverOdds:   -115,
			UnderOdds:  -105,
			Confidence: 0.82,
		},
	}
}

// generateMarketMovement simulates odds movement
func (mds *MarketDataService) generateMarketMovement() []OddsMovement {
	now := time.Now()
	return []OddsMovement{
		{
			Timestamp: now.Add(-2 * time.Hour),
			HomeOdds:  -140,
			AwayOdds:  120,
			TotalLine: 6.0,
			Volume:    15000,
			Direction: "home",
		},
		{
			Timestamp: now.Add(-1 * time.Hour),
			HomeOdds:  -150,
			AwayOdds:  130,
			TotalLine: 6.5,
			Volume:    28000,
			Direction: "home",
		},
	}
}

// generateBettingVolume simulates betting activity
func (mds *MarketDataService) generateBettingVolume() BettingVolume {
	return BettingVolume{
		TotalHandle:      500000, // $500k wagered
		HomePercent:      65.0,   // 65% of bets on home
		AwayPercent:      35.0,   // 35% of bets on away
		HomeMoneyPercent: 58.0,   // 58% of money on home
		AwayMoneyPercent: 42.0,   // 42% of money on away
		BetCount:         1250,   // 1,250 bets placed
	}
}

// detectSharpMoney analyzes for professional betting patterns
func (mds *MarketDataService) detectSharpMoney(data *MarketData) SharpMoneyIndicator {
	// Detect reverse line movement (line moves against public betting)
	reverseLineMove := data.Volume.HomePercent > 60.0 && data.MoneyLine.HomeOdds > -140

	// Steam move detection (rapid line movement)
	steamMove := len(data.MarketMovement) > 1 &&
		abs(data.MarketMovement[len(data.MarketMovement)-1].HomeOdds-data.MarketMovement[0].HomeOdds) > 20

	sharpAction := reverseLineMove || steamMove

	var sharpSide string
	var explanation string

	if sharpAction {
		if data.Volume.HomeMoneyPercent > data.Volume.HomePercent {
			sharpSide = data.HomeTeam
			explanation = "Sharp money on home team - higher money percentage than bet percentage"
		} else {
			sharpSide = data.AwayTeam
			explanation = "Sharp money on away team - line moved against public betting"
		}
	}

	return SharpMoneyIndicator{
		IsSharpAction:   sharpAction,
		SharpSide:       sharpSide,
		SharpConfidence: 0.75,
		ReverseLineMove: reverseLineMove,
		SteamMove:       steamMove,
		Explanation:     explanation,
	}
}

// analyzePublicBetting analyzes public betting patterns
func (mds *MarketDataService) analyzePublicBetting(data *MarketData) PublicBettingData {
	publicFavorite := data.HomeTeam
	publicPercentage := data.Volume.HomePercent

	if data.Volume.AwayPercent > data.Volume.HomePercent {
		publicFavorite = data.AwayTeam
		publicPercentage = data.Volume.AwayPercent
	}

	// Strong contrarian opportunity if public is heavily on one side
	contrarianSignal := publicPercentage > 75.0

	// Fade value increases with public percentage
	fadeValue := (publicPercentage - 50.0) / 50.0

	return PublicBettingData{
		PublicFavorite:   publicFavorite,
		PublicPercentage: publicPercentage,
		FadeValue:        fadeValue,
		ContrarianSignal: contrarianSignal,
	}
}

// probabilityToAmericanOdds converts probability to American odds format
func (mds *MarketDataService) probabilityToAmericanOdds(prob float64) int {
	if prob >= 0.5 {
		return int(-100 * prob / (1 - prob))
	} else {
		return int(100 * (1 - prob) / prob)
	}
}

// GetMarketPredictionAdjustment calculates prediction adjustments based on market data
func (mds *MarketDataService) GetMarketPredictionAdjustment(marketData *MarketData) models.MarketAdjustment {
	adjustment := models.MarketAdjustment{
		MarketConfidence:      marketData.Confidence,
		ImpliedProbability:    marketData.MoneyLine.HomeImpliedProb,
		SharpMoneyFactor:      0.0,
		PublicFadeFactor:      0.0,
		VolumeConfidence:      mds.calculateVolumeConfidence(marketData.Volume),
		MarketEfficiency:      mds.calculateMarketEfficiency(marketData),
		RecommendedAdjustment: 0.0,
	}

	// Sharp money adjustment
	if marketData.SharpMoney.IsSharpAction {
		if marketData.SharpMoney.SharpSide == marketData.HomeTeam {
			adjustment.SharpMoneyFactor = 0.05 // Boost home team
		} else {
			adjustment.SharpMoneyFactor = -0.05 // Boost away team
		}
	}

	// Public fade adjustment
	if marketData.PublicBetting.ContrarianSignal {
		adjustment.PublicFadeFactor = marketData.PublicBetting.FadeValue * 0.03
		if marketData.PublicBetting.PublicFavorite != marketData.HomeTeam {
			adjustment.PublicFadeFactor *= -1
		}
	}

	// Calculate overall recommended adjustment
	adjustment.RecommendedAdjustment = adjustment.SharpMoneyFactor + adjustment.PublicFadeFactor

	return adjustment
}

// calculateVolumeConfidence determines confidence based on betting volume
func (mds *MarketDataService) calculateVolumeConfidence(volume BettingVolume) float64 {
	// Higher volume = higher confidence in market price
	if volume.TotalHandle > 1000000 { // $1M+
		return 0.95
	} else if volume.TotalHandle > 500000 { // $500k+
		return 0.85
	} else if volume.TotalHandle > 100000 { // $100k+
		return 0.75
	} else {
		return 0.65
	}
}

// calculateMarketEfficiency measures how efficient the betting market is
func (mds *MarketDataService) calculateMarketEfficiency(data *MarketData) float64 {
	// Lower vig = more efficient market
	vigPenalty := data.MoneyLine.Vig * 0.5

	// Sharp money presence increases efficiency
	sharpBonus := 0.0
	if data.SharpMoney.IsSharpAction {
		sharpBonus = 0.1
	}

	// High volume increases efficiency
	volumeBonus := mds.calculateVolumeConfidence(data.Volume) * 0.1

	efficiency := 0.8 - vigPenalty + sharpBonus + volumeBonus

	if efficiency > 1.0 {
		efficiency = 1.0
	}
	if efficiency < 0.3 {
		efficiency = 0.3
	}

	return efficiency
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
