package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// BettingMarketService fetches and analyzes betting market data
type BettingMarketService struct {
	apiKey      string
	baseURL     string
	cache       map[string]*models.BettingOdds
	consensus   map[string]*models.MarketConsensus
	history     map[string]*models.BettingMarketHistory
	dataDir     string
	mutex       sync.RWMutex
	httpClient  *http.Client
	isEnabled   bool
	lastUpdated time.Time
}

// NewBettingMarketService creates a new betting market service
func NewBettingMarketService() *BettingMarketService {
	apiKey := os.Getenv("ODDS_API_KEY")

	service := &BettingMarketService{
		apiKey:     apiKey,
		baseURL:    "https://api.the-odds-api.com/v4",
		cache:      make(map[string]*models.BettingOdds),
		consensus:  make(map[string]*models.MarketConsensus),
		history:    make(map[string]*models.BettingMarketHistory),
		dataDir:    "data/betting_markets",
		isEnabled:  apiKey != "",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	// Create data directory
	os.MkdirAll(service.dataDir, 0755)

	if !service.isEnabled {
		log.Printf("⚠️ Betting Market Service disabled (no ODDS_API_KEY)")
		log.Printf("💡 To enable: Get free API key from https://the-odds-api.com/")
		log.Printf("   Then set: export ODDS_API_KEY=your_key_here")
	} else {
		log.Printf("💰 Betting Market Service initialized")

		// Load existing market data
		if err := service.loadMarketData(); err != nil {
			log.Printf("⚠️ Could not load market data: %v (starting fresh)", err)
		}
	}

	return service
}

// GetMarketAdjustment gets betting market influence on prediction
func (bms *BettingMarketService) GetMarketAdjustment(homeTeam, awayTeam string, gameDate time.Time) (*models.MarketBasedAdjustment, error) {
	if !bms.isEnabled {
		return nil, fmt.Errorf("betting market service not enabled")
	}

	// Get market consensus for this game
	consensus, err := bms.GetMarketConsensus(homeTeam, awayTeam, gameDate)
	if err != nil {
		return nil, fmt.Errorf("could not get market consensus: %w", err)
	}

	// Detect market signals
	signal, err := bms.DetectMarketSignal(homeTeam, awayTeam, gameDate)
	if err != nil {
		log.Printf("⚠️ Could not detect market signal: %v", err)
	}

	// Calculate adjustment
	adjustment := &models.MarketBasedAdjustment{
		MarketPrediction: consensus.ConsensusHomeWinPct,
		MarketEfficiency: consensus.MarketConfidence,
		DataRecency:      bms.calculateDataRecency(consensus.LastUpdated),
		MarketWeight:     bms.calculateMarketWeight(consensus, signal),
	}

	return adjustment, nil
}

// GetMarketConsensus aggregates odds from multiple bookmakers
func (bms *BettingMarketService) GetMarketConsensus(homeTeam, awayTeam string, gameDate time.Time) (*models.MarketConsensus, error) {
	bms.mutex.RLock()
	key := fmt.Sprintf("%s_vs_%s_%s", homeTeam, awayTeam, gameDate.Format("2006-01-02"))
	cached := bms.consensus[key]
	bms.mutex.RUnlock()

	// Return cached if recent
	if cached != nil && time.Since(cached.LastUpdated) < 5*time.Minute {
		return cached, nil
	}

	// Fetch fresh odds
	odds, err := bms.fetchOddsFromAPI(homeTeam, awayTeam, gameDate)
	if err != nil {
		return nil, fmt.Errorf("could not fetch odds: %w", err)
	}

	// Calculate consensus
	consensus := bms.calculateConsensus(odds)

	// Cache result
	bms.mutex.Lock()
	bms.consensus[key] = consensus
	bms.mutex.Unlock()

	return consensus, nil
}

// DetectMarketSignal detects sharp money and significant line moves
func (bms *BettingMarketService) DetectMarketSignal(homeTeam, awayTeam string, gameDate time.Time) (*models.MarketSignal, error) {
	bms.mutex.RLock()
	key := fmt.Sprintf("%s_vs_%s_%s", homeTeam, awayTeam, gameDate.Format("2006-01-02"))
	history := bms.history[key]
	bms.mutex.RUnlock()

	if history == nil || len(history.DataPoints) < 2 {
		return nil, fmt.Errorf("insufficient historical data")
	}

	signal := &models.MarketSignal{
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
		GameDate: gameDate,
	}

	// Detect reverse line move (sharp money)
	if bms.detectReverseLineMove(history) {
		signal.SignalType = "reverse_line"
		signal.SignalStrength = 0.8
		signal.Description = "Line moving against public money (sharp action)"
		signal.Reasoning = "Professional bettors betting opposite of public"
	}

	// Detect steam move (sudden sharp action)
	if bms.detectSteamMove(history) {
		signal.SignalType = "steam"
		signal.SignalStrength = 0.9
		signal.Description = "Sudden significant line movement"
		signal.Reasoning = "Coordinated sharp money or breaking news"
	}

	// Determine which side the signal favors
	latestMove := history.KeyMovements[len(history.KeyMovements)-1]
	signal.SignalSide = latestMove.Direction

	signal.Confidence = signal.SignalStrength
	signal.LastUpdated = time.Now()

	return signal, nil
}

// fetchOddsFromAPI fetches odds from The Odds API
func (bms *BettingMarketService) fetchOddsFromAPI(homeTeam, awayTeam string, gameDate time.Time) ([]*models.BettingOdds, error) {
	// The Odds API endpoint for NHL
	url := fmt.Sprintf("%s/sports/icehockey_nhl/odds?apiKey=%s&regions=us&markets=h2h,spreads,totals",
		bms.baseURL, bms.apiKey)

	resp, err := bms.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var apiResponse []OddsAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to our format and filter for this game
	var odds []*models.BettingOdds
	for _, game := range apiResponse {
		if bms.matchesTeams(game, homeTeam, awayTeam) {
			converted := bms.convertAPIResponseToOdds(game, homeTeam, awayTeam)
			odds = append(odds, converted)
		}
	}

	return odds, nil
}

// OddsAPIResponse represents the API response format
type OddsAPIResponse struct {
	ID           string    `json:"id"`
	SportKey     string    `json:"sport_key"`
	CommenceTime time.Time `json:"commence_time"`
	HomeTeam     string    `json:"home_team"`
	AwayTeam     string    `json:"away_team"`
	Bookmakers   []struct {
		Key        string    `json:"key"`
		Title      string    `json:"title"`
		LastUpdate time.Time `json:"last_update"`
		Markets    []struct {
			Key        string    `json:"key"`
			LastUpdate time.Time `json:"last_update"`
			Outcomes   []struct {
				Name  string  `json:"name"`
				Price float64 `json:"price"`           // American odds
				Point float64 `json:"point,omitempty"` // For spreads/totals
			} `json:"outcomes"`
		} `json:"markets"`
	} `json:"bookmakers"`
}

// convertAPIResponseToOdds converts API format to our models
func (bms *BettingMarketService) convertAPIResponseToOdds(game OddsAPIResponse, homeTeam, awayTeam string) *models.BettingOdds {
	odds := &models.BettingOdds{
		HomeTeam:    homeTeam,
		AwayTeam:    awayTeam,
		GameDate:    game.CommenceTime,
		LastUpdated: time.Now(),
	}

	// Extract odds from first bookmaker (simplified)
	if len(game.Bookmakers) > 0 {
		bookmaker := game.Bookmakers[0]
		odds.Bookmaker = bookmaker.Title

		for _, market := range bookmaker.Markets {
			switch market.Key {
			case "h2h": // Moneyline
				for _, outcome := range market.Outcomes {
					if outcome.Name == homeTeam {
						odds.HomeMoneyline = int(outcome.Price)
					} else {
						odds.AwayMoneyline = int(outcome.Price)
					}
				}
			case "spreads": // Puck line
				for _, outcome := range market.Outcomes {
					if outcome.Name == homeTeam {
						odds.HomeSpread = outcome.Point
						odds.HomeSpreadOdds = int(outcome.Price)
					} else {
						odds.AwaySpread = outcome.Point
						odds.AwaySpreadOdds = int(outcome.Price)
					}
				}
			case "totals": // Over/Under
				if len(market.Outcomes) > 0 {
					odds.TotalLine = market.Outcomes[0].Point
					for _, outcome := range market.Outcomes {
						if outcome.Name == "Over" {
							odds.OverOdds = int(outcome.Price)
						} else {
							odds.UnderOdds = int(outcome.Price)
						}
					}
				}
			}
		}
	}

	// Calculate implied probabilities
	odds.ImpliedHomeWinPct = bms.americanOddsToImpliedProb(odds.HomeMoneyline)
	odds.ImpliedAwayWinPct = bms.americanOddsToImpliedProb(odds.AwayMoneyline)

	return odds
}

// americanOddsToImpliedProb converts American odds to implied probability
func (bms *BettingMarketService) americanOddsToImpliedProb(odds int) float64 {
	if odds == 0 {
		return 0.5 // Default 50% if no odds
	}

	if odds > 0 {
		// Positive odds (underdog)
		return 100.0 / (float64(odds) + 100.0)
	} else {
		// Negative odds (favorite)
		return math.Abs(float64(odds)) / (math.Abs(float64(odds)) + 100.0)
	}
}

// calculateConsensus aggregates odds from multiple bookmakers
func (bms *BettingMarketService) calculateConsensus(allOdds []*models.BettingOdds) *models.MarketConsensus {
	if len(allOdds) == 0 {
		return nil
	}

	consensus := &models.MarketConsensus{
		HomeTeam:      allOdds[0].HomeTeam,
		AwayTeam:      allOdds[0].AwayTeam,
		GameDate:      allOdds[0].GameDate,
		NumBookmakers: len(allOdds),
		LastUpdated:   time.Now(),
	}

	// Calculate averages
	var sumHomeML, sumAwayML, sumTotal, sumSpread float64
	var sumHomeWinPct, sumAwayWinPct float64
	bestHomeOdds := -10000
	worstHomeOdds := 10000
	bestAwayOdds := -10000
	worstAwayOdds := 10000

	for _, odds := range allOdds {
		sumHomeML += float64(odds.HomeMoneyline)
		sumAwayML += float64(odds.AwayMoneyline)
		sumTotal += odds.TotalLine
		sumSpread += odds.HomeSpread
		sumHomeWinPct += odds.ImpliedHomeWinPct
		sumAwayWinPct += odds.ImpliedAwayWinPct

		// Track best/worst odds
		if odds.HomeMoneyline > bestHomeOdds {
			bestHomeOdds = odds.HomeMoneyline
		}
		if odds.HomeMoneyline < worstHomeOdds {
			worstHomeOdds = odds.HomeMoneyline
		}
		if odds.AwayMoneyline > bestAwayOdds {
			bestAwayOdds = odds.AwayMoneyline
		}
		if odds.AwayMoneyline < worstAwayOdds {
			worstAwayOdds = odds.AwayMoneyline
		}

		consensus.Bookmakers = append(consensus.Bookmakers, odds.Bookmaker)
	}

	n := float64(len(allOdds))
	consensus.AvgHomeMoneyline = sumHomeML / n
	consensus.AvgAwayMoneyline = sumAwayML / n
	consensus.AvgTotalLine = sumTotal / n
	consensus.AvgHomeSpread = sumSpread / n
	consensus.ConsensusHomeWinPct = sumHomeWinPct / n
	consensus.ConsensusAwayWinPct = sumAwayWinPct / n

	consensus.BestHomeOdds = bestHomeOdds
	consensus.WorstHomeOdds = worstHomeOdds
	consensus.BestAwayOdds = bestAwayOdds
	consensus.WorstAwayOdds = worstAwayOdds

	// Calculate market agreement (low variance = high agreement)
	variance := float64(bestHomeOdds-worstHomeOdds) / math.Abs(consensus.AvgHomeMoneyline)
	consensus.MarketAgreement = math.Max(0, 1.0-variance)
	consensus.MarketConfidence = consensus.MarketAgreement // Simplified

	return consensus
}

// detectReverseLineMove detects line moving against public money
func (bms *BettingMarketService) detectReverseLineMove(history *models.BettingMarketHistory) bool {
	if len(history.DataPoints) < 3 {
		return false
	}

	// Check last few data points
	recent := history.DataPoints[len(history.DataPoints)-3:]

	// Line moving toward home but public betting away (or vice versa)
	lineMovingHome := recent[2].HomeMoneyline < recent[0].HomeMoneyline // Odds getting better for home
	publicBettingHome := recent[2].HomeBetPct > 55.0

	// Reverse line move: Line moving one way, public betting the other
	// Simplified: Just check if line is moving against public
	return (lineMovingHome && !publicBettingHome) || (!lineMovingHome && publicBettingHome)
}

// detectSteamMove detects sudden significant line movement
func (bms *BettingMarketService) detectSteamMove(history *models.BettingMarketHistory) bool {
	if len(history.DataPoints) < 2 {
		return false
	}

	// Compare last two data points
	latest := history.DataPoints[len(history.DataPoints)-1]
	previous := history.DataPoints[len(history.DataPoints)-2]

	// Calculate line movement
	homeOddsChange := math.Abs(float64(latest.HomeMoneyline - previous.HomeMoneyline))

	// Steam move: Significant change in short time (> 10 points in American odds)
	return homeOddsChange > 10.0
}

// calculateDataRecency calculates how fresh the data is
func (bms *BettingMarketService) calculateDataRecency(lastUpdate time.Time) float64 {
	age := time.Since(lastUpdate)

	if age < 5*time.Minute {
		return 1.0 // Very fresh
	} else if age < 30*time.Minute {
		return 0.8
	} else if age < 2*time.Hour {
		return 0.5
	} else {
		return 0.2 // Stale
	}
}

// calculateMarketWeight determines how much to trust market
func (bms *BettingMarketService) calculateMarketWeight(consensus *models.MarketConsensus, signal *models.MarketSignal) float64 {
	baseWeight := 0.25 // Default 25% market weight

	// Increase weight if market is confident
	if consensus != nil && consensus.MarketConfidence > 0.8 {
		baseWeight += 0.05
	}

	// Increase weight if sharp money detected
	if signal != nil && signal.SignalStrength > 0.7 {
		baseWeight += 0.1
	}

	return math.Min(0.40, baseWeight) // Cap at 40%
}

// matchesTeams checks if API response matches our teams
func (bms *BettingMarketService) matchesTeams(game OddsAPIResponse, homeTeam, awayTeam string) bool {
	// Simplified team name matching (would need proper mapping in production)
	return true // TODO: Implement proper team name matching
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// loadMarketData loads market data from disk
func (bms *BettingMarketService) loadMarketData() error {
	filePath := filepath.Join(bms.dataDir, "market_data.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("market data file not found")
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading market data file: %w", err)
	}

	var data struct {
		Cache       map[string]*models.BettingOdds          `json:"cache"`
		Consensus   map[string]*models.MarketConsensus      `json:"consensus"`
		History     map[string]*models.BettingMarketHistory `json:"history"`
		LastUpdated time.Time                               `json:"lastUpdated"`
	}

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return fmt.Errorf("error unmarshaling market data: %w", err)
	}

	bms.cache = data.Cache
	bms.consensus = data.Consensus
	bms.history = data.History
	bms.lastUpdated = data.LastUpdated

	return nil
}

// saveMarketData saves market data to disk
func (bms *BettingMarketService) saveMarketData() error {
	filePath := filepath.Join(bms.dataDir, "market_data.json")

	data := struct {
		Cache       map[string]*models.BettingOdds          `json:"cache"`
		Consensus   map[string]*models.MarketConsensus      `json:"consensus"`
		History     map[string]*models.BettingMarketHistory `json:"history"`
		LastUpdated time.Time                               `json:"lastUpdated"`
		Version     string                                  `json:"version"`
	}{
		Cache:       bms.cache,
		Consensus:   bms.consensus,
		History:     bms.history,
		LastUpdated: time.Now(),
		Version:     "1.0",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling market data: %w", err)
	}

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing market data file: %w", err)
	}

	return nil
}

// ============================================================================
// GLOBAL SERVICE
// ============================================================================

var (
	globalBettingMarketService *BettingMarketService
	bettingMarketMutex         sync.Mutex
)

// InitializeBettingMarketService initializes the global betting market service
func InitializeBettingMarketService() error {
	bettingMarketMutex.Lock()
	defer bettingMarketMutex.Unlock()

	if globalBettingMarketService != nil {
		return fmt.Errorf("betting market service already initialized")
	}

	globalBettingMarketService = NewBettingMarketService()
	log.Printf("💰 Betting Market Service initialized")

	return nil
}

// GetBettingMarketService returns the global betting market service
func GetBettingMarketService() *BettingMarketService {
	bettingMarketMutex.Lock()
	defer bettingMarketMutex.Unlock()
	return globalBettingMarketService
}
