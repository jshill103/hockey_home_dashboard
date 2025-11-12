package services

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// WhatIfCache holds cached what-if simulation results (Phase 5.3)
type WhatIfCache struct {
	results map[string]*CachedWhatIfResult
	mu      sync.RWMutex
}

// CachedWhatIfResult holds a cached what-if result with expiration
type CachedWhatIfResult struct {
	Result    *WhatIfResult
	ExpiresAt time.Time
}

var whatIfCache = &WhatIfCache{
	results: make(map[string]*CachedWhatIfResult),
}

// WhatIfScenario represents a hypothetical scenario to simulate
type WhatIfScenario struct {
	Name        string  `json:"name"`        // "Win Next 5", "Lose Next 3", etc.
	Description string  `json:"description"` // Human-readable description
	WinNext     int     `json:"winNext"`     // Number of next games to win
	LoseNext    int     `json:"loseNext"`    // Number of next games to lose
	WinRate     float64 `json:"winRate"`     // Win rate for remaining games after scenario
}

// WhatIfResult holds the results of a what-if simulation
type WhatIfResult struct {
	Scenario            WhatIfScenario  `json:"scenario"`
	ProjectedPoints     int             `json:"projectedPoints"`
	ProjectedRecord     string          `json:"projectedRecord"`
	PlayoffOdds         float64         `json:"playoffOdds"`
	PlayoffOddsChange   float64         `json:"playoffOddsChange"`   // vs current odds
	AvgFinalPoints      float64         `json:"avgFinalPoints"`
	MedianFinalPoints   int             `json:"medianFinalPoints"`
	RankImprovement     float64         `json:"rankImprovement"`     // Expected rank change
	MagicNumberChange   int             `json:"magicNumberChange"`   // Change in magic number
	Likelihood          string          `json:"likelihood"`          // "likely", "possible", "unlikely"
}

// generateWhatIfCacheKey generates a unique cache key for a what-if scenario (Phase 5.3)
func generateWhatIfCacheKey(teamCode string, scenario WhatIfScenario, teamPoints int, gamesPlayed int) string {
	// Include team state + scenario in hash
	data := fmt.Sprintf("%s|%s|%d|%d|%d|%d|%.3f", 
		teamCode, scenario.Name, teamPoints, gamesPlayed, 
		scenario.WinNext, scenario.LoseNext, scenario.WinRate)
	
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:16]) // Use first 128 bits
}

// getFromCache retrieves a cached what-if result if available (Phase 5.3)
func (wc *WhatIfCache) Get(key string) (*WhatIfResult, bool) {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	
	cached, exists := wc.results[key]
	if !exists {
		return nil, false
	}
	
	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}
	
	return cached.Result, true
}

// setInCache stores a what-if result in cache (Phase 5.3)
func (wc *WhatIfCache) Set(key string, result *WhatIfResult, duration time.Duration) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	
	wc.results[key] = &CachedWhatIfResult{
		Result:    result,
		ExpiresAt: time.Now().Add(duration),
	}
}

// clearExpired removes expired entries from cache (Phase 5.3)
func (wc *WhatIfCache) ClearExpired() int {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	
	now := time.Now()
	removed := 0
	for key, cached := range wc.results {
		if now.After(cached.ExpiresAt) {
			delete(wc.results, key)
			removed++
		}
	}
	return removed
}

// SimulateWhatIf runs a what-if scenario simulation (Phase 5.3: With caching)
func (ps *PlayoffSimulationService) SimulateWhatIf(teamCode string, scenario WhatIfScenario, simulations int) (*WhatIfResult, error) {
	// Get current standings
	standings, err := GetStandings()
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %v", err)
	}

	// Find target team
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
		return nil, fmt.Errorf("team %s not found", teamCode)
	}

	// Check cache first (Phase 5.3, Phase 5.5: Track metrics)
	cacheKey := generateWhatIfCacheKey(teamCode, scenario, targetTeam.Points, targetTeam.GamesPlayed)
	if cachedResult, found := whatIfCache.Get(cacheKey); found {
		fmt.Printf("✅ What-If cache hit for '%s'\n", scenario.Name)
		GetGlobalMetrics().RecordCacheHit(true)
		return cachedResult, nil
	}
	GetGlobalMetrics().RecordCacheMiss(true)

	// Create a modified team record based on scenario
	modifiedTeam := *targetTeam
	gamesRemaining := 82 - modifiedTeam.GamesPlayed

	// Apply scenario wins/losses
	gamesInScenario := scenario.WinNext + scenario.LoseNext
	if gamesInScenario > gamesRemaining {
		gamesInScenario = gamesRemaining
	}

	// Apply wins
	if scenario.WinNext > 0 {
		modifiedTeam.Wins += scenario.WinNext
		modifiedTeam.Points += scenario.WinNext * 2
		modifiedTeam.GamesPlayed += scenario.WinNext
	}

	// Apply losses
	if scenario.LoseNext > 0 {
		modifiedTeam.Losses += scenario.LoseNext
		modifiedTeam.GamesPlayed += scenario.LoseNext
	}

	// Recalculate point percentage
	if modifiedTeam.GamesPlayed > 0 {
		modifiedTeam.PointPctg = float64(modifiedTeam.Points) / float64(modifiedTeam.GamesPlayed*2)
	}

	// Get all conference teams (Phase 5.4: Pre-allocate with estimated capacity)
	conferenceTeams := make([]*models.TeamStanding, 0, 16) // Typical conference size
	for i := range standings.Standings {
		team := &standings.Standings[i]
		if team.ConferenceName == conferenceName {
			if team.TeamAbbrev.Default == teamCode {
				conferenceTeams = append(conferenceTeams, &modifiedTeam)
			} else {
				conferenceTeams = append(conferenceTeams, team)
			}
		}
	}

	// Get remaining schedule
	remainingGames, err := ps.getRemainingGames(conferenceTeams)
	if err != nil {
		return nil, fmt.Errorf("failed to get remaining games: %v", err)
	}

	// Filter out games that are part of the scenario (already "played")
	// Phase 5.4: Pre-allocate with capacity
	filteredGames := make([]RemainingGame, 0, len(remainingGames))
	scenarioGamesSkipped := 0
	for _, game := range remainingGames {
		if game.HomeTeam == teamCode || game.AwayTeam == teamCode {
			if scenarioGamesSkipped < gamesInScenario {
				scenarioGamesSkipped++
				continue // Skip this game (it's part of the scenario)
			}
		}
		filteredGames = append(filteredGames, game)
	}

	// Run simulations with modified team
	fmt.Printf("🎯 What-If: Simulating '%s' scenario...\n", scenario.Name)
	results := make([]SimulationResult, simulations)
	pointsDistribution := make(map[int]int)

	for i := 0; i < simulations; i++ {
		result := ps.simulateSeason(&modifiedTeam, conferenceTeams, filteredGames)
		results[i] = result
		pointsDistribution[result.FinalPoints]++
	}

	// Aggregate results
	simulation := ps.aggregateResults(teamCode, results, pointsDistribution)

	// Calculate magic numbers for modified scenario
	conferenceTeamsSlice := make([]models.TeamStanding, len(conferenceTeams))
	for i, t := range conferenceTeams {
		conferenceTeamsSlice[i] = *t
	}
	magicNumbers := CalculateMagicNumbers(&modifiedTeam, conferenceTeamsSlice)

	// Build result
	result := &WhatIfResult{
		Scenario:          scenario,
		ProjectedPoints:   modifiedTeam.Points,
		ProjectedRecord:   fmt.Sprintf("%d-%d-%d", modifiedTeam.Wins, modifiedTeam.Losses, modifiedTeam.OtLosses),
		PlayoffOdds:       simulation.PlayoffOddsPercent,
		AvgFinalPoints:    simulation.AvgFinalPoints,
		MedianFinalPoints: simulation.PercentileP50,
	}

	// Calculate likelihood based on historical win rate
	if scenario.WinNext > 0 {
		expectedWins := float64(scenario.WinNext) * targetTeam.PointPctg / 0.5 // Normalize
		if expectedWins >= float64(scenario.WinNext)*0.8 {
			result.Likelihood = "likely"
		} else if expectedWins >= float64(scenario.WinNext)*0.5 {
			result.Likelihood = "possible"
		} else {
			result.Likelihood = "unlikely"
		}
	} else {
		result.Likelihood = "neutral"
	}

	// Magic number change
	originalMagicNumbers := CalculateMagicNumbers(targetTeam, conferenceTeamsSlice)
	if originalMagicNumbers != nil && magicNumbers != nil {
		result.MagicNumberChange = originalMagicNumbers.MagicNumber - magicNumbers.MagicNumber
	}

	// Cache the result (Phase 5.3: Cache for 5 minutes)
	whatIfCache.Set(cacheKey, result, 5*time.Minute)

	return result, nil
}

// GetCommonWhatIfScenarios returns a list of common what-if scenarios
func GetCommonWhatIfScenarios(team *models.TeamStanding) []WhatIfScenario {
	gamesRemaining := 82 - team.GamesPlayed
	
	scenarios := []WhatIfScenario{
		{
			Name:        "Win Next 3",
			Description: "What if we win the next 3 games?",
			WinNext:     3,
			LoseNext:    0,
		},
		{
			Name:        "Win Next 5",
			Description: "What if we win the next 5 games?",
			WinNext:     5,
			LoseNext:    0,
		},
		{
			Name:        "Win Next 10",
			Description: "What if we win the next 10 games?",
			WinNext:     10,
			LoseNext:    0,
		},
		{
			Name:        "Lose Next 3",
			Description: "What if we lose the next 3 games?",
			WinNext:     0,
			LoseNext:    3,
		},
		{
			Name:        "Lose Next 5",
			Description: "What if we lose the next 5 games?",
			WinNext:     0,
			LoseNext:    5,
		},
		{
			Name:        "Go .500",
			Description: "What if we go 50/50 the rest of the way?",
			WinNext:     0,
			LoseNext:    0,
			WinRate:     0.50,
		},
		{
			Name:        "Hot Streak",
			Description: "What if we go .700 the rest of the way?",
			WinNext:     0,
			LoseNext:    0,
			WinRate:     0.70,
		},
		{
			Name:        "Cold Streak",
			Description: "What if we go .300 the rest of the way?",
			WinNext:     0,
			LoseNext:    0,
			WinRate:     0.30,
		},
	}
	
	// Filter scenarios that are impossible
	validScenarios := make([]WhatIfScenario, 0)
	for _, scenario := range scenarios {
		if scenario.WinNext+scenario.LoseNext <= gamesRemaining {
			validScenarios = append(validScenarios, scenario)
		}
	}
	
	return validScenarios
}

