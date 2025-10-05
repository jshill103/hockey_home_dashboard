package services

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// PlayoffSimulationService uses ML models to simulate remaining season and calculate playoff odds
type PlayoffSimulationService struct {
	ensembleService *EnsemblePredictionService
	mu              sync.RWMutex
	cachedResults   map[string]*CachedSimulation
	cacheMu         sync.RWMutex
}

// CachedSimulation holds cached simulation results
type CachedSimulation struct {
	Result    *SeasonSimulation
	Timestamp time.Time
}

// NewPlayoffSimulationService creates a new playoff simulation service
func NewPlayoffSimulationService(ensemble *EnsemblePredictionService) *PlayoffSimulationService {
	return &PlayoffSimulationService{
		ensembleService: ensemble,
		cachedResults:   make(map[string]*CachedSimulation),
	}
}

// SimulationResult holds the results of a playoff simulation
type SimulationResult struct {
	MadePlayoffs    bool
	FinalPoints     int
	FinalWins       int
	FinalLosses     int
	FinalOTLosses   int
	ConferenceRank  int
	DivisionRank    int
	PlayoffSpotType string // "division", "wildcard", "none"
}

// SeasonSimulation holds the results of multiple season simulations
type SeasonSimulation struct {
	TeamCode            string
	TotalSimulations    int
	PlayoffAppearances  int
	PlayoffOddsPercent  float64
	DivisionTop3Count   int
	DivisionOddsPercent float64
	WildCardCount       int
	WildCardOddsPercent float64
	AvgFinalPoints      float64
	BestCasePoints      int
	WorstCasePoints     int
	AvgConferenceRank   float64
	PointsDistribution  map[int]int // Points -> Count
}

// RemainingGame represents a game yet to be played
type RemainingGame struct {
	HomeTeam string
	AwayTeam string
	Date     time.Time
}

// SimulatePlayoffOdds runs Monte Carlo simulation of the remaining season
func (ps *PlayoffSimulationService) SimulatePlayoffOdds(teamCode string, simulations int) (*SeasonSimulation, error) {
	// Check cache first
	if cached := ps.getCachedResult(teamCode); cached != nil {
		fmt.Printf("📦 Using cached playoff odds for %s (age: %v)\n", teamCode, time.Since(cached.Timestamp))
		return cached.Result, nil
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	fmt.Printf("🎲 Starting playoff simulation for %s (%d simulations)...\n", teamCode, simulations)

	// Get current standings
	standings, err := GetStandings()
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %v", err)
	}

	// Find target team and get their conference
	var targetTeam *models.TeamStanding
	var conferenceName string
	for i := range standings.Standings {
		team := &standings.Standings[i]
		if team.TeamAbbrev.Default == teamCode {
			targetTeam = team
			conferenceName = team.ConferenceName
			break
		}
	}

	if targetTeam == nil {
		return nil, fmt.Errorf("team %s not found in standings", teamCode)
	}

	// Get all teams in the same conference
	conferenceTeams := make([]*models.TeamStanding, 0)
	for i := range standings.Standings {
		team := &standings.Standings[i]
		if team.ConferenceName == conferenceName {
			conferenceTeams = append(conferenceTeams, team)
		}
	}

	// Get remaining schedule for all teams
	remainingGames, err := ps.getRemainingGames(conferenceTeams)
	if err != nil {
		return nil, fmt.Errorf("failed to get remaining games: %v", err)
	}

	fmt.Printf("📅 Found %d remaining games in %s conference\n", len(remainingGames), conferenceName)

	// Run simulations
	results := make([]SimulationResult, simulations)
	pointsDistribution := make(map[int]int)

	for i := 0; i < simulations; i++ {
		result := ps.simulateSeason(targetTeam, conferenceTeams, remainingGames)
		results[i] = result
		pointsDistribution[result.FinalPoints]++

		if (i+1)%1000 == 0 {
			fmt.Printf("  Completed %d/%d simulations\n", i+1, simulations)
		}
	}

	// Aggregate results
	simulation := ps.aggregateResults(teamCode, results, pointsDistribution)

	fmt.Printf("✅ Simulation complete: %.1f%% playoff odds\n", simulation.PlayoffOddsPercent)

	// Cache the result
	ps.cacheResult(teamCode, simulation)

	return simulation, nil
}

// simulateSeason simulates one possible season outcome
func (ps *PlayoffSimulationService) simulateSeason(
	targetTeam *models.TeamStanding,
	conferenceTeams []*models.TeamStanding,
	remainingGames []RemainingGame,
) SimulationResult {
	// Create a copy of current standings
	teamRecords := make(map[string]*models.TeamStanding)
	for _, team := range conferenceTeams {
		// Create a copy
		teamCopy := *team
		teamRecords[team.TeamAbbrev.Default] = &teamCopy
	}

	// Simulate each remaining game
	for _, game := range remainingGames {
		ps.simulateGame(game, teamRecords)
	}

	// Determine final standings
	targetFinal := teamRecords[targetTeam.TeamAbbrev.Default]

	// Sort conference by points
	sortedTeams := make([]*models.TeamStanding, 0, len(teamRecords))
	for _, team := range teamRecords {
		sortedTeams = append(sortedTeams, team)
	}
	sort.Slice(sortedTeams, func(i, j int) bool {
		return sortedTeams[i].Points > sortedTeams[j].Points
	})

	// Find target team's rank
	conferenceRank := 0
	for i, team := range sortedTeams {
		if team.TeamAbbrev.Default == targetTeam.TeamAbbrev.Default {
			conferenceRank = i + 1
			break
		}
	}

	// Determine if made playoffs (top 8 in conference)
	madePlayoffs := conferenceRank <= 8
	playoffSpotType := "none"

	if madePlayoffs {
		// Determine if division spot or wild card
		divisionRank := ps.getDivisionRank(targetFinal, sortedTeams)
		if divisionRank <= 3 {
			playoffSpotType = "division"
		} else {
			playoffSpotType = "wildcard"
		}
	}

	return SimulationResult{
		MadePlayoffs:    madePlayoffs,
		FinalPoints:     targetFinal.Points,
		FinalWins:       targetFinal.Wins,
		FinalLosses:     targetFinal.Losses,
		FinalOTLosses:   targetFinal.OtLosses,
		ConferenceRank:  conferenceRank,
		PlayoffSpotType: playoffSpotType,
	}
}

// simulateGame simulates a single game and updates team records
func (ps *PlayoffSimulationService) simulateGame(game RemainingGame, teamRecords map[string]*models.TeamStanding) {
	homeTeam := teamRecords[game.HomeTeam]
	awayTeam := teamRecords[game.AwayTeam]

	if homeTeam == nil || awayTeam == nil {
		return // Teams not in our conference
	}

	// Use ensemble to predict game outcome
	// For simulation speed, use a simplified prediction
	winProbability := ps.quickPredict(game.HomeTeam, game.AwayTeam, homeTeam, awayTeam)

	// Simulate game outcome based on probability
	roll := rand.Float64()

	if roll < winProbability {
		// Home team wins
		homeWinsRegulation := rand.Float64() < 0.85 // 85% of wins are regulation

		if homeWinsRegulation {
			homeTeam.Wins++
			homeTeam.Points += 2
			awayTeam.Losses++
		} else {
			// Home wins in OT/SO
			homeTeam.Wins++
			homeTeam.Points += 2
			awayTeam.OtLosses++
			awayTeam.Points += 1
		}
	} else {
		// Away team wins
		awayWinsRegulation := rand.Float64() < 0.85

		if awayWinsRegulation {
			awayTeam.Wins++
			awayTeam.Points += 2
			homeTeam.Losses++
		} else {
			// Away wins in OT/SO
			awayTeam.Wins++
			awayTeam.Points += 2
			homeTeam.OtLosses++
			homeTeam.Points += 1
		}
	}

	homeTeam.GamesPlayed++
	awayTeam.GamesPlayed++
}

// quickPredict provides a fast prediction without full ensemble computation
func (ps *PlayoffSimulationService) quickPredict(
	homeCode, awayCode string,
	homeTeam, awayTeam *models.TeamStanding,
) float64 {
	// Use point percentage as base probability
	homeWinPct := homeTeam.PointPctg
	awayWinPct := awayTeam.PointPctg

	// Home ice advantage (roughly 5-8%)
	homeAdvantage := 0.06

	// Calculate relative strength
	if homeWinPct+awayWinPct > 0 {
		baseProbability := homeWinPct / (homeWinPct + awayWinPct)
		winProbability := baseProbability + homeAdvantage

		// Clamp to reasonable range
		if winProbability < 0.30 {
			winProbability = 0.30
		} else if winProbability > 0.80 {
			winProbability = 0.80
		}

		return winProbability
	}

	return 0.55 // Default with home advantage
}

// getDivisionRank calculates division rank from sorted conference teams
func (ps *PlayoffSimulationService) getDivisionRank(team *models.TeamStanding, sortedTeams []*models.TeamStanding) int {
	divisionRank := 0
	for _, t := range sortedTeams {
		if t.DivisionName == team.DivisionName {
			divisionRank++
			if t.TeamAbbrev.Default == team.TeamAbbrev.Default {
				return divisionRank
			}
		}
	}
	return divisionRank
}

// aggregateResults combines simulation results into final statistics
func (ps *PlayoffSimulationService) aggregateResults(
	teamCode string,
	results []SimulationResult,
	pointsDistribution map[int]int,
) *SeasonSimulation {
	totalSims := len(results)
	playoffCount := 0
	divisionCount := 0
	wildCardCount := 0
	totalPoints := 0
	totalRank := 0
	bestPoints := 0
	worstPoints := 200

	for _, result := range results {
		if result.MadePlayoffs {
			playoffCount++
			if result.PlayoffSpotType == "division" {
				divisionCount++
			} else if result.PlayoffSpotType == "wildcard" {
				wildCardCount++
			}
		}

		totalPoints += result.FinalPoints
		totalRank += result.ConferenceRank

		if result.FinalPoints > bestPoints {
			bestPoints = result.FinalPoints
		}
		if result.FinalPoints < worstPoints {
			worstPoints = result.FinalPoints
		}
	}

	return &SeasonSimulation{
		TeamCode:            teamCode,
		TotalSimulations:    totalSims,
		PlayoffAppearances:  playoffCount,
		PlayoffOddsPercent:  float64(playoffCount) / float64(totalSims) * 100,
		DivisionTop3Count:   divisionCount,
		DivisionOddsPercent: float64(divisionCount) / float64(totalSims) * 100,
		WildCardCount:       wildCardCount,
		WildCardOddsPercent: float64(wildCardCount) / float64(totalSims) * 100,
		AvgFinalPoints:      float64(totalPoints) / float64(totalSims),
		BestCasePoints:      bestPoints,
		WorstCasePoints:     worstPoints,
		AvgConferenceRank:   float64(totalRank) / float64(totalSims),
		PointsDistribution:  pointsDistribution,
	}
}

// getRemainingGames fetches remaining schedule for all conference teams
func (ps *PlayoffSimulationService) getRemainingGames(conferenceTeams []*models.TeamStanding) ([]RemainingGame, error) {
	// For now, generate a simplified remaining schedule based on games played
	// In production, this would fetch actual scheduled games from NHL API

	games := make([]RemainingGame, 0)
	now := time.Now()

	// For each team, calculate remaining games (82 - games_played)
	// and create approximate matchups
	for _, team := range conferenceTeams {
		gamesRemaining := 82 - team.GamesPlayed

		// Create games for this team (simplified - random opponents from conference)
		for g := 0; g < gamesRemaining/2; g++ {
			// Pick a random opponent
			opponent := conferenceTeams[rand.Intn(len(conferenceTeams))]
			if opponent.TeamAbbrev.Default == team.TeamAbbrev.Default {
				continue
			}

			// Alternate home/away
			if g%2 == 0 {
				games = append(games, RemainingGame{
					HomeTeam: team.TeamAbbrev.Default,
					AwayTeam: opponent.TeamAbbrev.Default,
					Date:     now.Add(time.Duration(g*2) * 24 * time.Hour),
				})
			} else {
				games = append(games, RemainingGame{
					HomeTeam: opponent.TeamAbbrev.Default,
					AwayTeam: team.TeamAbbrev.Default,
					Date:     now.Add(time.Duration(g*2) * 24 * time.Hour),
				})
			}
		}
	}

	return games, nil
}

// Global instance
var playoffSimulationService *PlayoffSimulationService
var playoffSimOnce sync.Once

// InitPlayoffSimulationService initializes the global playoff simulation service
func InitPlayoffSimulationService(ensemble *EnsemblePredictionService) {
	playoffSimOnce.Do(func() {
		playoffSimulationService = NewPlayoffSimulationService(ensemble)
		fmt.Println("✅ Playoff Simulation Service initialized")
	})
}

// GetPlayoffSimulationService returns the global playoff simulation service
func GetPlayoffSimulationService() *PlayoffSimulationService {
	return playoffSimulationService
}

// getCachedResult retrieves cached simulation results if available and fresh
func (ps *PlayoffSimulationService) getCachedResult(teamCode string) *CachedSimulation {
	ps.cacheMu.RLock()
	defer ps.cacheMu.RUnlock()

	cached, exists := ps.cachedResults[teamCode]
	if !exists {
		return nil
	}

	// Cache is valid for 1 hour (or until a game completes)
	if time.Since(cached.Timestamp) > 1*time.Hour {
		return nil
	}

	return cached
}

// cacheResult stores simulation results in cache
func (ps *PlayoffSimulationService) cacheResult(teamCode string, result *SeasonSimulation) {
	ps.cacheMu.Lock()
	defer ps.cacheMu.Unlock()

	ps.cachedResults[teamCode] = &CachedSimulation{
		Result:    result,
		Timestamp: time.Now(),
	}

	fmt.Printf("💾 Playoff odds cached for %s\n", teamCode)
}

// InvalidateCache clears cached results for a team (called after game completion)
func (ps *PlayoffSimulationService) InvalidateCache(teamCode string) {
	ps.cacheMu.Lock()
	defer ps.cacheMu.Unlock()

	delete(ps.cachedResults, teamCode)
	fmt.Printf("🗑️ Playoff odds cache invalidated for %s\n", teamCode)
}

// InvalidateAllCache clears all cached results (called after any game in conference)
func (ps *PlayoffSimulationService) InvalidateAllCache() {
	ps.cacheMu.Lock()
	defer ps.cacheMu.Unlock()

	ps.cachedResults = make(map[string]*CachedSimulation)
	fmt.Printf("🗑️ All playoff odds caches invalidated\n")
}

// RecalculatePlayoffOdds forces recalculation and caches the result
// This is called automatically after games complete
func (ps *PlayoffSimulationService) RecalculatePlayoffOdds(teamCode string) error {
	// Invalidate existing cache
	ps.InvalidateCache(teamCode)

	// Run new simulation (will auto-cache)
	fmt.Printf("🔄 Recalculating playoff odds for %s after game completion...\n", teamCode)
	_, err := ps.SimulatePlayoffOdds(teamCode, 5000)
	if err != nil {
		return fmt.Errorf("failed to recalculate playoff odds: %v", err)
	}

	fmt.Printf("✅ Playoff odds recalculated and cached for %s\n", teamCode)
	return nil
}
