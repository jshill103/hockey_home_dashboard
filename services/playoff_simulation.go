package services

import (
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// PlayoffSimulationService uses ML models to simulate remaining season and calculate playoff odds
type PlayoffSimulationService struct {
	ensembleService *EnsemblePredictionService
	gamePredictor   GamePredictor // Phase 3: Configurable prediction strategy
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
	// Default to ELO predictor for simulations (fast, reasonable accuracy)
	// ML predictor is too slow without caching, and caching doesn't work with evolving team records
	eloPredictor := NewEloPredictor()
	
	return &PlayoffSimulationService{
		ensembleService: ensemble,
		gamePredictor:   eloPredictor, // Default to ELO (Phase 3 - Bug Fix)
		cachedResults:   make(map[string]*CachedSimulation),
	}
}

// NewPlayoffSimulationServiceWithPredictor creates a service with a custom predictor
func NewPlayoffSimulationServiceWithPredictor(ensemble *EnsemblePredictionService, predictor GamePredictor) *PlayoffSimulationService {
	return &PlayoffSimulationService{
		ensembleService: ensemble,
		gamePredictor:   predictor,
		cachedResults:   make(map[string]*CachedSimulation),
	}
}

// SetGamePredictor allows changing the prediction strategy (Phase 3)
func (ps *PlayoffSimulationService) SetGamePredictor(predictor GamePredictor) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.gamePredictor = predictor
	// Clear cache when strategy changes
	ps.cachedResults = make(map[string]*CachedSimulation)
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
	
	// Schedule Strength Analysis (Phase 2)
	ScheduleStrength    *TeamScheduleStrength `json:"scheduleStrength,omitempty"`
	
	// Percentile Distributions (Phase 4.2)
	PercentileP10       int     `json:"percentileP10,omitempty"`       // 10th percentile (pessimistic)
	PercentileP25       int     `json:"percentileP25,omitempty"`       // 25th percentile
	PercentileP50       int     `json:"percentileP50,omitempty"`       // 50th percentile (median)
	PercentileP75       int     `json:"percentileP75,omitempty"`       // 75th percentile
	PercentileP90       int     `json:"percentileP90,omitempty"`       // 90th percentile (optimistic)
	
	// Magic Numbers (Phase 4.1)
	MagicNumbers        *MagicNumbers `json:"magicNumbers,omitempty"`
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

	// Analyze schedule strength (Phase 2)
	fmt.Println("📊 Analyzing schedule strength...")
	analyzer := NewScheduleAnalyzer(conferenceTeams)
	scheduleStrength := analyzer.AnalyzeTeamSchedule(teamCode, remainingGames)
	
	fmt.Printf("  Schedule Difficulty: %.1f/10 (%s)\n", scheduleStrength.ScheduleDifficulty, scheduleStrength.DifficultyTier)
	fmt.Printf("  Remaining Games: %d (%d Home, %d Away)\n", scheduleStrength.RemainingGames, scheduleStrength.HomeGamesRemaining, scheduleStrength.AwayGamesRemaining)
	fmt.Printf("  Avg Opponent Win%%: %.3f\n", scheduleStrength.AvgOpponentWinPct)
	fmt.Printf("  Crucial Games: %d (Must-Win: %d)\n", len(scheduleStrength.CrucialGames), scheduleStrength.MustWinGames)

	// Phase 3.3: Pre-cache predictions for performance
	// DISABLED: Pre-caching doesn't work for Monte Carlo simulations because team records evolve
	// Cache keys based on initial records become stale as games are played in each simulation
	// Only useful for ELO predictor (which doesn't depend on full team records)
	// if simulations >= 1000 && ps.gamePredictor.Name() == "ELO" {
	// 	fmt.Printf("🚀 Pre-caching predictions for %d games (strategy: %s)...\n", len(remainingGames), ps.gamePredictor.Name())
	// 	ps.precachePredictions(remainingGames, conferenceTeams)
	// }

	// Run simulations (Phase 5.1: Parallel execution, Phase 5.5: With metrics)
	// Record start time and ensure metrics are recorded on completion
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		GetGlobalMetrics().RecordSimulation(duration, simulations, simulations >= 1000)
	}()
	
	results := make([]SimulationResult, simulations)
	pointsDistribution := make(map[int]int)
	
	// Use parallel execution for large simulation counts
	if simulations >= 1000 {
		fmt.Printf("🚀 Running %d simulations in parallel...\n", simulations)
		results = ps.runParallelSimulations(targetTeam, conferenceTeams, remainingGames, simulations)
	} else {
		// Sequential execution for small counts
		for i := 0; i < simulations; i++ {
			results[i] = ps.simulateSeason(targetTeam, conferenceTeams, remainingGames)
		}
	}
	
	// Build points distribution
	for _, result := range results {
		pointsDistribution[result.FinalPoints]++
	}

	// Aggregate results
	simulation := ps.aggregateResults(teamCode, results, pointsDistribution)
	
	// Add schedule strength analysis to results (Phase 2)
	simulation.ScheduleStrength = scheduleStrength

	// Calculate magic numbers (Phase 4.1)
	// Convert []*TeamStanding to []TeamStanding for magic numbers function
	conferenceTeamsSlice := make([]models.TeamStanding, len(conferenceTeams))
	for i, t := range conferenceTeams {
		conferenceTeamsSlice[i] = *t
	}
	magicNumbers := CalculateMagicNumbers(targetTeam, conferenceTeamsSlice)
	simulation.MagicNumbers = magicNumbers
	
	fmt.Printf("✅ Simulation complete: %.1f%% playoff odds\n", simulation.PlayoffOddsPercent)
	if magicNumbers != nil {
		fmt.Printf("   Magic Number: %d points | P50 projection: %d points\n", 
			magicNumbers.MagicNumber, simulation.PercentileP50)
	}

	// Cache the result
	ps.cacheResult(teamCode, simulation)

	return simulation, nil
}

// runParallelSimulations runs multiple simulations in parallel using goroutines (Phase 5.1)
func (ps *PlayoffSimulationService) runParallelSimulations(
	targetTeam *models.TeamStanding,
	conferenceTeams []*models.TeamStanding,
	remainingGames []RemainingGame,
	simulations int,
) []SimulationResult {
	results := make([]SimulationResult, simulations)
	
	// Use worker pool pattern - number of workers = CPU cores
	numWorkers := runtime.NumCPU()
	if numWorkers > simulations {
		numWorkers = simulations
	}
	
	// Channel for work distribution
	jobs := make(chan int, simulations)
	var wg sync.WaitGroup
	
	// Progress tracking
	var completed int32
	progressTicker := time.NewTicker(1 * time.Second)
	defer progressTicker.Stop()
	
	// Progress reporter goroutine
	done := make(chan bool)
	defer func() {
		done <- true // Ensure goroutine exits even on panic
	}()
	
	go func() {
		for {
			select {
			case <-progressTicker.C:
				current := atomic.LoadInt32(&completed)
				if current > 0 {
					fmt.Printf("  Progress: %d/%d simulations (%.1f%%)\n", 
						current, simulations, float64(current)/float64(simulations)*100)
				}
			case <-done:
				return
			}
		}
	}()
	
	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				// Each worker runs one simulation
				result := ps.simulateSeason(targetTeam, conferenceTeams, remainingGames)
				results[i] = result
				atomic.AddInt32(&completed, 1)
			}
		}()
	}
	
	// Send jobs to workers
	for i := 0; i < simulations; i++ {
		jobs <- i
	}
	close(jobs)
	
	// Wait for all workers to finish
	wg.Wait()
	
	fmt.Printf("✅ Completed %d parallel simulations using %d workers\n", simulations, numWorkers)
	
	return results
}

// simulateSeason simulates one possible season outcome
func (ps *PlayoffSimulationService) simulateSeason(
	targetTeam *models.TeamStanding,
	conferenceTeams []*models.TeamStanding,
	remainingGames []RemainingGame,
) SimulationResult {
	// Create a copy of current standings (Phase 5.4: Optimized with pre-sized map)
	teamRecords := make(map[string]*models.TeamStanding, len(conferenceTeams))
	for _, team := range conferenceTeams {
		// Create a copy
		teamCopy := *team
		teamRecords[team.TeamAbbrev.Default] = &teamCopy
	}

	// Track previous game dates for rest day calculation (Phase 3)
	prevGameDates := make(map[string]time.Time)

	// Simulate each remaining game
	for _, game := range remainingGames {
		ps.simulateGame(game, teamRecords, prevGameDates)
	}

	// Determine final standings
	targetFinal := teamRecords[targetTeam.TeamAbbrev.Default]

	// Sort conference by NHL tiebreaker rules
	sortedTeams := make([]*models.TeamStanding, 0, len(teamRecords))
	for _, team := range teamRecords {
		sortedTeams = append(sortedTeams, team)
	}
	ps.sortByNHLRules(sortedTeams)

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
func (ps *PlayoffSimulationService) simulateGame(game RemainingGame, teamRecords map[string]*models.TeamStanding, prevGameDates map[string]time.Time) {
	homeTeam := teamRecords[game.HomeTeam]
	awayTeam := teamRecords[game.AwayTeam]

	if homeTeam == nil || awayTeam == nil {
		return // Teams not in our conference
	}

	// Phase 3: Use GamePredictor with full context
	context := ps.buildPredictionContext(game, homeTeam, awayTeam, prevGameDates)
	
	// Get win probability from configured predictor (with nil check)
	var winProbability float64
	if ps.gamePredictor == nil {
		// Fallback to simple prediction if predictor not set
		winProbability = ps.quickPredict(game.HomeTeam, game.AwayTeam, homeTeam, awayTeam)
	} else {
		var err error
		winProbability, err = ps.gamePredictor.PredictWinProbability(game.HomeTeam, game.AwayTeam, context)
		if err != nil {
			// Fallback to simple prediction if predictor fails
			winProbability = ps.quickPredict(game.HomeTeam, game.AwayTeam, homeTeam, awayTeam)
		}
	}

	// Simulate game outcome based on probability
	roll := rand.Float64()

	if roll < winProbability {
		// Home team wins - determine win type
		winType := rand.Float64()
		
		if winType < 0.85 {
			// Regulation win (85%)
			homeTeam.Wins++
			homeTeam.RegulationWins++
			homeTeam.RegulationPlusOtWins++
			homeTeam.Points += 2
			awayTeam.Losses++
		} else if winType < 0.95 {
			// Overtime win (10%) - counts toward ROW
			homeTeam.Wins++
			homeTeam.RegulationPlusOtWins++
			homeTeam.Points += 2
			awayTeam.OtLosses++
			awayTeam.Points += 1
		} else {
			// Shootout win (5%) - does NOT count toward ROW
			homeTeam.Wins++
			homeTeam.Points += 2
			awayTeam.OtLosses++
			awayTeam.Points += 1
		}
	} else {
		// Away team wins - determine win type
		winType := rand.Float64()
		
		if winType < 0.85 {
			// Regulation win (85%)
			awayTeam.Wins++
			awayTeam.RegulationWins++
			awayTeam.RegulationPlusOtWins++
			awayTeam.Points += 2
			homeTeam.Losses++
		} else if winType < 0.95 {
			// Overtime win (10%) - counts toward ROW
			awayTeam.Wins++
			awayTeam.RegulationPlusOtWins++
			awayTeam.Points += 2
			homeTeam.OtLosses++
			homeTeam.Points += 1
		} else {
			// Shootout win (5%) - does NOT count toward ROW
			awayTeam.Wins++
			awayTeam.Points += 2
			homeTeam.OtLosses++
			homeTeam.Points += 1
		}
	}

	homeTeam.GamesPlayed++
	awayTeam.GamesPlayed++
	
	// Track last game date for rest day calculation
	prevGameDates[game.HomeTeam] = game.Date
	prevGameDates[game.AwayTeam] = game.Date
}

// precachePredictions pre-computes predictions for all remaining games (Phase 3.3)
func (ps *PlayoffSimulationService) precachePredictions(games []RemainingGame, conferenceTeams []*models.TeamStanding) {
	// Build team lookup map
	teamMap := make(map[string]*models.TeamStanding)
	for _, team := range conferenceTeams {
		teamMap[team.TeamAbbrev.Default] = team
	}
	
	// Pre-compute predictions for all games
	cached := 0
	for _, game := range games {
		homeTeam := teamMap[game.HomeTeam]
		awayTeam := teamMap[game.AwayTeam]
		
		if homeTeam == nil || awayTeam == nil {
			continue
		}
		
		// Build context for this game
		context := &PredictionContext{
			Date:           game.Date,
			HomeRecord:     homeTeam,
			AwayRecord:     awayTeam,
			RestDaysHome:   3, // Default
			RestDaysAway:   3, // Default
			IsDivisionGame: (homeTeam.DivisionName == awayTeam.DivisionName),
		}
		
		// Call predictor to cache the result
		_, err := ps.gamePredictor.PredictWinProbability(game.HomeTeam, game.AwayTeam, context)
		if err == nil {
			cached++
		}
	}
	
	fmt.Printf("  ✅ Pre-cached %d/%d game predictions\n", cached, len(games))
}

// buildPredictionContext creates context for game prediction (Phase 3)
func (ps *PlayoffSimulationService) buildPredictionContext(
	game RemainingGame,
	homeTeam, awayTeam *models.TeamStanding,
	prevGameDates map[string]time.Time,
) *PredictionContext {
	context := &PredictionContext{
		Date:       game.Date,
		HomeRecord: homeTeam,
		AwayRecord: awayTeam,
	}
	
	// Calculate rest days (with bounds checking)
	if lastHome, ok := prevGameDates[game.HomeTeam]; ok {
		restDays := int(game.Date.Sub(lastHome).Hours() / 24)
		if restDays < 0 {
			restDays = 3 // Default if dates are wonky
		}
		context.RestDaysHome = restDays
	} else {
		context.RestDaysHome = 3 // Default
	}
	
	if lastAway, ok := prevGameDates[game.AwayTeam]; ok {
		restDays := int(game.Date.Sub(lastAway).Hours() / 24)
		if restDays < 0 {
			restDays = 3 // Default if dates are wonky
		}
		context.RestDaysAway = restDays
	} else {
		context.RestDaysAway = 3 // Default
	}
	
	// Check if division/rivalry game (homeTeam/awayTeam already verified non-nil above)
	context.IsDivisionGame = (homeTeam.DivisionName == awayTeam.DivisionName)
	// Simple rivalry detection (within 10 points)
	pointsDiff := homeTeam.Points - awayTeam.Points
	if pointsDiff < 0 {
		pointsDiff = -pointsDiff
	}
	context.IsRivalryGame = (pointsDiff <= 10 && context.IsDivisionGame)
	
	// Playoffs detection (late season games)
	gamesRemaining := 82 - homeTeam.GamesPlayed
	context.IsPlayoffs = (gamesRemaining <= 10)
	
	return context
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

// compareTeamsByNHLRules compares two teams using official NHL tiebreaker rules
// Returns true if team a should rank higher than team b
func compareTeamsByNHLRules(a, b *models.TeamStanding) bool {
	// 1. Points (primary criteria)
	if a.Points != b.Points {
		return a.Points > b.Points
	}
	
	// 2. Games Played (fewer is better when tied on points)
	if a.GamesPlayed != b.GamesPlayed {
		return a.GamesPlayed < b.GamesPlayed
	}
	
	// 3. Regulation + Overtime Wins (ROW) - excludes shootout wins
	aROW := a.GetROW()
	bROW := b.GetROW()
	if aROW != bROW {
		return aROW > bROW
	}
	
	// 4. Total Wins (including shootout)
	if a.Wins != b.Wins {
		return a.Wins > b.Wins
	}
	
	// 5. Points earned in head-to-head (not available, skip)
	
	// 6. Goal Differential
	aGD := a.GetGoalsFor() - a.GetGoalsAgainst()
	bGD := b.GetGoalsFor() - b.GetGoalsAgainst()
	if aGD != bGD {
		return aGD > bGD
	}
	
	// 7. Goals For
	if a.GetGoalsFor() != b.GetGoalsFor() {
		return a.GetGoalsFor() > b.GetGoalsFor()
	}
	
	// 8. Last resort: alphabetical by team name
	return a.TeamName.Default < b.TeamName.Default
}

// SortTeamsByNHLRules sorts teams using official NHL tiebreaker rules (public utility function)
func SortTeamsByNHLRules(teams []models.TeamStanding) {
	sort.Slice(teams, func(i, j int) bool {
		return compareTeamsByNHLRules(&teams[i], &teams[j])
	})
}

// sortByNHLRules sorts teams using official NHL tiebreaker rules (for pointers)
func (ps *PlayoffSimulationService) sortByNHLRules(teams []*models.TeamStanding) {
	sort.Slice(teams, func(i, j int) bool {
		return compareTeamsByNHLRules(teams[i], teams[j])
	})
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

	// Calculate percentiles (Phase 4.2)
	percentiles := calculatePercentiles(results)

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
		PercentileP10:       percentiles[10],
		PercentileP25:       percentiles[25],
		PercentileP50:       percentiles[50],
		PercentileP75:       percentiles[75],
		PercentileP90:       percentiles[90],
	}
}

// calculatePercentiles computes percentile values from simulation results (Phase 4.2)
func calculatePercentiles(results []SimulationResult) map[int]int {
	// Extract all final points
	points := make([]int, len(results))
	for i, result := range results {
		points[i] = result.FinalPoints
	}
	
	// Sort points
	sort.Ints(points)
	
	percentiles := make(map[int]int)
	n := len(points)
	
	// Calculate percentiles
	percentiles[10] = points[int(float64(n)*0.10)]
	percentiles[25] = points[int(float64(n)*0.25)]
	percentiles[50] = points[int(float64(n)*0.50)] // Median
	percentiles[75] = points[int(float64(n)*0.75)]
	percentiles[90] = points[int(float64(n)*0.90)]
	
	return percentiles
}

// getRemainingGames fetches ACTUAL remaining schedule for all conference teams from NHL API
func (ps *PlayoffSimulationService) getRemainingGames(conferenceTeams []*models.TeamStanding) ([]RemainingGame, error) {
	// Phase 5.4: Pre-allocate with estimated capacity (avg 15 games/team * teams / 2 for duplicates)
	estimatedGames := (len(conferenceTeams) * 15) / 2
	games := make([]RemainingGame, 0, estimatedGames)
	now := time.Now()
	seasonStr := GetCurrentSeason()
	
	// Convert season string "20242025" to int
	season, err := strconv.Atoi(seasonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse season: %v", err)
	}
	
	fmt.Println("📅 Fetching real remaining schedule from NHL API...")
	
	// Track unique games to avoid duplicates (same game appears in both team schedules)
	gameKeys := make(map[string]bool)
	successCount := 0
	errorCount := 0
	
	// Fetch schedule for each team in the conference
	for _, team := range conferenceTeams {
		teamCode := team.TeamAbbrev.Default
		
		// Fetch full season schedule for this team
		schedule, err := GetTeamSeasonSchedule(teamCode, season)
		if err != nil {
			fmt.Printf("⚠️ Failed to fetch schedule for %s: %v\n", teamCode, err)
			errorCount++
			continue
		}
		successCount++
		
		// Filter for remaining games (not yet played)
		for _, game := range schedule {
			// Parse game time
			gameTime, err := time.Parse(time.RFC3339, game.StartTime)
			if err != nil {
				continue
			}
			
			// Only include future games
			if !gameTime.After(now) {
				continue
			}
			
			// Get home and away team codes
			homeCode := game.HomeTeam.Abbrev
			awayCode := game.AwayTeam.Abbrev
			
			// Only include games where BOTH teams are in this conference
			if !ps.isTeamInList(homeCode, conferenceTeams) || !ps.isTeamInList(awayCode, conferenceTeams) {
				continue
			}
			
			// Create unique key to avoid duplicates
			// Use sorted team codes and date to ensure consistency
			var teamA, teamB string
			if homeCode < awayCode {
				teamA, teamB = homeCode, awayCode
			} else {
				teamA, teamB = awayCode, homeCode
			}
			gameKey := fmt.Sprintf("%s_%s_%s", gameTime.Format("2006-01-02"), teamA, teamB)
			
			// Skip if already added
			if gameKeys[gameKey] {
				continue
			}
			gameKeys[gameKey] = true
			
			// Add game to list
			games = append(games, RemainingGame{
				HomeTeam: homeCode,
				AwayTeam: awayCode,
				Date:     gameTime,
			})
		}
	}
	
	// Sort games by date (chronological order)
	sort.Slice(games, func(i, j int) bool {
		return games[i].Date.Before(games[j].Date)
	})
	
	fmt.Printf("✅ Fetched real schedule: %d teams processed (%d success, %d errors)\n", 
		len(conferenceTeams), successCount, errorCount)
	fmt.Printf("📊 Found %d remaining conference games\n", len(games))
	
	// If we couldn't fetch any schedules, fall back to estimation
	if len(games) == 0 && errorCount > 0 {
		fmt.Println("⚠️ No schedules fetched, falling back to estimation...")
		return ps.getEstimatedRemainingGames(conferenceTeams)
	}
	
	return games, nil
}

// isTeamInList checks if a team code is in the conference teams list
func (ps *PlayoffSimulationService) isTeamInList(teamCode string, teams []*models.TeamStanding) bool {
	for _, team := range teams {
		if team.TeamAbbrev.Default == teamCode {
			return true
		}
	}
	return false
}

// getEstimatedRemainingGames provides a fallback estimation when API fails
func (ps *PlayoffSimulationService) getEstimatedRemainingGames(conferenceTeams []*models.TeamStanding) ([]RemainingGame, error) {
	games := make([]RemainingGame, 0)
	gameKeys := make(map[string]bool) // Track unique games to avoid duplicates
	now := time.Now()
	
	fmt.Println("📊 Generating estimated schedule based on games played...")
	
	// For each team, calculate remaining games
	for _, team := range conferenceTeams {
		gamesRemaining := 82 - team.GamesPlayed
		
		// Estimate games (simplified - better than random, uses round-robin style)
		gamesPerOpponent := gamesRemaining / (len(conferenceTeams) - 1)
		if gamesPerOpponent == 0 {
			gamesPerOpponent = 1
		}
		
		gameCount := 0
		for _, opponent := range conferenceTeams {
			if opponent.TeamAbbrev.Default == team.TeamAbbrev.Default {
				continue
			}
			
			for g := 0; g < gamesPerOpponent && gameCount < gamesRemaining/2; g++ {
				gameDate := now.Add(time.Duration(gameCount*2) * 24 * time.Hour)
				var homeTeam, awayTeam string
				
				// Alternate home/away
				if gameCount%2 == 0 {
					homeTeam = team.TeamAbbrev.Default
					awayTeam = opponent.TeamAbbrev.Default
				} else {
					homeTeam = opponent.TeamAbbrev.Default
					awayTeam = team.TeamAbbrev.Default
				}
				
				// Create unique key to avoid duplicates
				var teamA, teamB string
				if homeTeam < awayTeam {
					teamA, teamB = homeTeam, awayTeam
				} else {
					teamA, teamB = awayTeam, homeTeam
				}
				gameKey := fmt.Sprintf("%s_%s_%s", gameDate.Format("2006-01-02"), teamA, teamB)
				
				// Skip if already added
				if !gameKeys[gameKey] {
					gameKeys[gameKey] = true
					games = append(games, RemainingGame{
						HomeTeam: homeTeam,
						AwayTeam: awayTeam,
						Date:     gameDate,
					})
				}
				
				gameCount++
			}
		}
	}
	
	fmt.Printf("📊 Estimated %d games (deduplicated)\n", len(games))
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
