package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// StandingsCacheService provides in-memory caching for NHL standings
// to prevent redundant API calls within a short timeframe
type StandingsCacheService struct {
	standings []models.TeamStanding
	cachedAt  time.Time
	cacheMu   sync.RWMutex
	cacheTTL  time.Duration
	hitCount  int64
	missCount int64
	statsMu   sync.RWMutex
}

var (
	standingsCacheService     *StandingsCacheService
	standingsCacheServiceOnce sync.Once
)

const (
	standingsCacheTTL = 5 * time.Minute // Standings change after games
)

// InitStandingsCacheService initializes the global standings cache service
func InitStandingsCacheService() {
	standingsCacheServiceOnce.Do(func() {
		standingsCacheService = &StandingsCacheService{
			cacheTTL: standingsCacheTTL,
		}
	})
}

// GetStandingsCacheService returns the singleton instance
func GetStandingsCacheService() *StandingsCacheService {
	return standingsCacheService
}

// GetStandings returns cached standings or fetches from API if stale
func (scs *StandingsCacheService) GetStandings() ([]models.TeamStanding, error) {
	// Check cache first
	scs.cacheMu.RLock()
	if len(scs.standings) > 0 && time.Since(scs.cachedAt) < scs.cacheTTL {
		standings := scs.standings
		scs.cacheMu.RUnlock()
		scs.recordHit()
		return standings, nil
	}
	scs.cacheMu.RUnlock()

	// Cache miss or stale - fetch from API
	scs.recordMiss()

	fmt.Println("Fetching NHL standings...")
	url := "https://api-web.nhle.com/v1/standings/now"
	body, err := MakeAPICall(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch standings: %w", err)
	}

	var response models.StandingsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal standings: %w", err)
	}

	standings := response.Standings

	// Update cache
	scs.cacheMu.Lock()
	scs.standings = standings
	scs.cachedAt = time.Now()
	scs.cacheMu.Unlock()

	fmt.Println("Successfully fetched NHL standings")
	return standings, nil
}

// GetTeamStanding returns a specific team's standing from cached data
func (scs *StandingsCacheService) GetTeamStanding(teamCode string) (*models.TeamStanding, error) {
	standings, err := scs.GetStandings()
	if err != nil {
		return nil, err
	}

	for _, standing := range standings {
		if standing.TeamAbbrev.Default == teamCode {
			return &standing, nil
		}
	}

	return nil, fmt.Errorf("team %s not found in standings", teamCode)
}

// InvalidateCache forces a cache refresh on the next request
func (scs *StandingsCacheService) InvalidateCache() {
	scs.cacheMu.Lock()
	scs.standings = nil
	scs.cachedAt = time.Time{}
	scs.cacheMu.Unlock()
}

// GetStats returns cache statistics
func (scs *StandingsCacheService) GetStats() map[string]interface{} {
	scs.statsMu.RLock()
	hits := scs.hitCount
	misses := scs.missCount
	scs.statsMu.RUnlock()

	total := hits + misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	scs.cacheMu.RLock()
	age := time.Since(scs.cachedAt)
	cacheActive := len(scs.standings) > 0
	scs.cacheMu.RUnlock()

	return map[string]interface{}{
		"hits":         hits,
		"misses":       misses,
		"hit_rate":     fmt.Sprintf("%.1f%%", hitRate),
		"cache_active": cacheActive,
		"cache_age":    age.Round(time.Second).String(),
		"ttl":          scs.cacheTTL.String(),
	}
}

// recordHit increments the hit counter
func (scs *StandingsCacheService) recordHit() {
	scs.statsMu.Lock()
	scs.hitCount++
	scs.statsMu.Unlock()
}

// recordMiss increments the miss counter
func (scs *StandingsCacheService) recordMiss() {
	scs.statsMu.Lock()
	scs.missCount++
	scs.statsMu.Unlock()
}
