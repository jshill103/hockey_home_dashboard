package services

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ScheduleContextService analyzes schedule situations and travel
type ScheduleContextService struct {
	cityCoords map[string]models.CityCoordinates
	mutex      sync.RWMutex
}

// NewScheduleContextService creates a new schedule context service
func NewScheduleContextService() *ScheduleContextService {
	return &ScheduleContextService{
		cityCoords: initializeNHLCityCoordinates(),
	}
}

// GetScheduleComparison analyzes schedule situation for both teams
func (scs *ScheduleContextService) GetScheduleComparison(homeTeam, awayTeam string, gameDate time.Time) (*models.ScheduleComparison, error) {
	scs.mutex.RLock()
	defer scs.mutex.RUnlock()

	// Build context for each team
	homeContext := scs.buildScheduleContext(homeTeam, true, gameDate)
	awayContext := scs.buildScheduleContext(awayTeam, false, gameDate)

	// Calculate comparison
	comparison := &models.ScheduleComparison{
		HomeContext: homeContext,
		AwayContext: awayContext,
	}

	// Determine advantages
	comparison.TravelImpact = scs.compareTravelFatigue(homeContext, awayContext)
	comparison.RestImpact = scs.compareRest(homeContext, awayContext)
	comparison.ScheduleImpact = scs.compareScheduleDensity(homeContext, awayContext)
	comparison.TrapGameImpact = scs.compareTrapGames(homeContext, awayContext)
	comparison.PlayoffImpact = scs.comparePlayoffImportance(homeContext, awayContext)

	// Calculate total impact
	comparison.TotalImpact = (comparison.TravelImpact*0.20 +
		comparison.RestImpact*0.30 +
		comparison.ScheduleImpact*0.20 +
		comparison.TrapGameImpact*0.15 +
		comparison.PlayoffImpact*0.15)

	// Determine overall advantage
	if comparison.TotalImpact > 0.02 {
		comparison.OverallAdvantage = "home"
	} else if comparison.TotalImpact < -0.02 {
		comparison.OverallAdvantage = "away"
	} else {
		comparison.OverallAdvantage = "even"
	}

	// Set specific advantages
	if comparison.TravelImpact > 0.02 {
		comparison.TravelAdvantage = "home"
	} else if comparison.TravelImpact < -0.02 {
		comparison.TravelAdvantage = "away"
	} else {
		comparison.TravelAdvantage = "even"
	}

	if comparison.RestImpact > 0.02 {
		comparison.RestAdvantage = "home"
	} else if comparison.RestImpact < -0.02 {
		comparison.RestAdvantage = "away"
	} else {
		comparison.RestAdvantage = "even"
	}

	if comparison.ScheduleImpact > 0.02 {
		comparison.ScheduleAdvantage = "home"
	} else if comparison.ScheduleImpact < -0.02 {
		comparison.ScheduleAdvantage = "away"
	} else {
		comparison.ScheduleAdvantage = "even"
	}

	comparison.Confidence = 0.7 // Default confidence
	comparison.LastUpdated = time.Now()

	return comparison, nil
}

// buildScheduleContext builds schedule context for a team
func (scs *ScheduleContextService) buildScheduleContext(team string, isHome bool, gameDate time.Time) *models.ScheduleContext {
	context := &models.ScheduleContext{
		TeamCode:    team,
		GameDate:    gameDate,
		IsHome:      isHome,
		LastUpdated: time.Now(),
	}

	// Calculate travel distance (simplified - assume away team traveled)
	if !isHome {
		context.TravelDistance = scs.estimateTravelDistance(team, gameDate)
		context.TravelFatigueScore = context.TravelDistance / 3000.0 // Normalize
		if context.TravelFatigueScore > 1.0 {
			context.TravelFatigueScore = 1.0
		}
	}

	// Rest days (simplified - would need actual schedule)
	context.RestDays = scs.estimateRestDays(team, gameDate)
	context.DaysSinceLastGame = context.RestDays

	// Back-to-back detection (simplified)
	context.IsBackToBack = context.RestDays == 0
	if context.IsBackToBack {
		context.BackToBackCount = 1
	}

	// Schedule density (simplified)
	context.GamesInLast7Days = scs.estimateGamesInLastWeek(team, gameDate)
	context.GamesInLast14Days = context.GamesInLast7Days * 2 // Rough estimate
	context.ScheduleDensityScore = float64(context.GamesInLast7Days) / 7.0

	// Trap game detection (simplified heuristic)
	context.IsTrapGame, context.TrapGameScore, context.TrapGameReason = scs.detectTrapGame(team, isHome, gameDate)

	// Playoff context (simplified - would need standings)
	context.InPlayoffRace = true    // Assume most teams are in race
	context.PlayoffImportance = 0.5 // Default medium importance

	return context
}

// compareTravelFatigue compares travel situations
func (scs *ScheduleContextService) compareTravelFatigue(home, away *models.ScheduleContext) float64 {
	// Home team typically has no travel
	homeFatigue := home.TravelFatigueScore
	awayFatigue := away.TravelFatigueScore

	// Advantage to less fatigued team
	diff := awayFatigue - homeFatigue
	return math.Max(-0.10, math.Min(0.10, diff)) // Cap at ±10%
}

// compareRest compares rest days
func (scs *ScheduleContextService) compareRest(home, away *models.ScheduleContext) float64 {
	restDiff := home.RestDays - away.RestDays

	// Each extra rest day worth ~2%
	impact := float64(restDiff) * 0.02
	return math.Max(-0.10, math.Min(0.10, impact))
}

// compareScheduleDensity compares schedule load
func (scs *ScheduleContextService) compareScheduleDensity(home, away *models.ScheduleContext) float64 {
	// Team with lighter schedule has advantage
	densityDiff := away.ScheduleDensityScore - home.ScheduleDensityScore

	impact := densityDiff * 0.05 // Moderate impact
	return math.Max(-0.05, math.Min(0.05, impact))
}

// compareTrapGames checks for trap game scenarios
func (scs *ScheduleContextService) compareTrapGames(home, away *models.ScheduleContext) float64 {
	homeTrap := home.TrapGameScore
	awayTrap := away.TrapGameScore

	// Trap game hurts the affected team
	if homeTrap > 0.5 {
		return -0.03 // Home team in trap game
	}
	if awayTrap > 0.5 {
		return 0.03 // Away team in trap game
	}

	return 0.0
}

// comparePlayoffImportance compares stakes
func (scs *ScheduleContextService) comparePlayoffImportance(home, away *models.ScheduleContext) float64 {
	importanceDiff := home.PlayoffImportance - away.PlayoffImportance

	impact := importanceDiff * 0.05
	return math.Max(-0.05, math.Min(0.05, impact))
}

// ============================================================================
// HELPER FUNCTIONS (Simplified Implementations)
// ============================================================================

// estimateTravelDistance estimates travel distance (simplified)
func (scs *ScheduleContextService) estimateTravelDistance(team string, gameDate time.Time) float64 {
	// Simplified: Use city distance as rough estimate
	// In production, would track actual previous game location

	// For now, return average travel distance
	return 800.0 // Miles (rough average)
}

// estimateRestDays estimates rest days (simplified)
func (scs *ScheduleContextService) estimateRestDays(team string, gameDate time.Time) int {
	// Simplified: Return typical rest (1-2 days)
	// In production, would look at actual schedule

	return 1 // Default 1 day rest
}

// estimateGamesInLastWeek estimates recent game load (simplified)
func (scs *ScheduleContextService) estimateGamesInLastWeek(team string, gameDate time.Time) int {
	// Simplified: Typical NHL schedule is 3-4 games per week
	return 3 // Default
}

// detectTrapGame identifies trap game scenarios (simplified)
func (scs *ScheduleContextService) detectTrapGame(team string, isHome bool, gameDate time.Time) (bool, float64, string) {
	// Simplified trap game detection
	// Real implementation would check:
	// - Previous opponent strength
	// - Next opponent strength
	// - Team's recent success

	// For now, no trap game detected
	return false, 0.0, ""
}

// calculateDistance calculates distance between two coordinates using Haversine formula
func (scs *ScheduleContextService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMiles = 3959.0

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMiles * c
}

// initializeNHLCityCoordinates returns city coordinates for all NHL teams
func initializeNHLCityCoordinates() map[string]models.CityCoordinates {
	return map[string]models.CityCoordinates{
		// Metropolitan Division
		"CAR": {City: "Raleigh", Latitude: 35.7796, Longitude: -78.6382, TimeZone: "EST"},
		"CBJ": {City: "Columbus", Latitude: 39.9612, Longitude: -82.9988, TimeZone: "EST"},
		"NJD": {City: "Newark", Latitude: 40.7357, Longitude: -74.1724, TimeZone: "EST"},
		"NYI": {City: "Elmont", Latitude: 40.7058, Longitude: -73.7097, TimeZone: "EST"},
		"NYR": {City: "New York", Latitude: 40.7128, Longitude: -74.0060, TimeZone: "EST"},
		"PHI": {City: "Philadelphia", Latitude: 39.9526, Longitude: -75.1652, TimeZone: "EST"},
		"PIT": {City: "Pittsburgh", Latitude: 40.4406, Longitude: -79.9959, TimeZone: "EST"},
		"WSH": {City: "Washington", Latitude: 38.9072, Longitude: -77.0369, TimeZone: "EST"},

		// Atlantic Division
		"BOS": {City: "Boston", Latitude: 42.3601, Longitude: -71.0589, TimeZone: "EST"},
		"BUF": {City: "Buffalo", Latitude: 42.8864, Longitude: -78.8784, TimeZone: "EST"},
		"DET": {City: "Detroit", Latitude: 42.3314, Longitude: -83.0458, TimeZone: "EST"},
		"FLA": {City: "Sunrise", Latitude: 26.1583, Longitude: -80.3256, TimeZone: "EST"},
		"MTL": {City: "Montreal", Latitude: 45.5017, Longitude: -73.5673, TimeZone: "EST"},
		"OTT": {City: "Ottawa", Latitude: 45.4215, Longitude: -75.6972, TimeZone: "EST"},
		"TBL": {City: "Tampa", Latitude: 27.9506, Longitude: -82.4572, TimeZone: "EST"},
		"TOR": {City: "Toronto", Latitude: 43.6532, Longitude: -79.3832, TimeZone: "EST"},

		// Central Division
		"ARI": {City: "Tempe", Latitude: 33.4484, Longitude: -111.9261, TimeZone: "MST"},
		"CHI": {City: "Chicago", Latitude: 41.8781, Longitude: -87.6298, TimeZone: "CST"},
		"COL": {City: "Denver", Latitude: 39.7392, Longitude: -104.9903, TimeZone: "MST"},
		"DAL": {City: "Dallas", Latitude: 32.7767, Longitude: -96.7970, TimeZone: "CST"},
		"MIN": {City: "St. Paul", Latitude: 44.9537, Longitude: -93.0900, TimeZone: "CST"},
		"NSH": {City: "Nashville", Latitude: 36.1627, Longitude: -86.7816, TimeZone: "CST"},
		"STL": {City: "St. Louis", Latitude: 38.6270, Longitude: -90.1994, TimeZone: "CST"},
		"UTA": {City: "Salt Lake City", Latitude: 40.7608, Longitude: -111.8910, TimeZone: "MST"},
		"WPG": {City: "Winnipeg", Latitude: 49.8951, Longitude: -97.1384, TimeZone: "CST"},

		// Pacific Division
		"ANA": {City: "Anaheim", Latitude: 33.8366, Longitude: -117.9143, TimeZone: "PST"},
		"CGY": {City: "Calgary", Latitude: 51.0447, Longitude: -114.0719, TimeZone: "MST"},
		"EDM": {City: "Edmonton", Latitude: 53.5461, Longitude: -113.4938, TimeZone: "MST"},
		"LAK": {City: "Los Angeles", Latitude: 34.0522, Longitude: -118.2437, TimeZone: "PST"},
		"SJS": {City: "San Jose", Latitude: 37.3382, Longitude: -121.8863, TimeZone: "PST"},
		"SEA": {City: "Seattle", Latitude: 47.6062, Longitude: -122.3321, TimeZone: "PST"},
		"VAN": {City: "Vancouver", Latitude: 49.2827, Longitude: -123.1207, TimeZone: "PST"},
		"VGK": {City: "Las Vegas", Latitude: 36.1699, Longitude: -115.1398, TimeZone: "PST"},
	}
}

// ============================================================================
// GLOBAL SERVICE
// ============================================================================

var (
	globalScheduleContextService *ScheduleContextService
	scheduleContextMutex         sync.Mutex
)

// InitializeScheduleContextService initializes the global schedule context service
func InitializeScheduleContextService() error {
	scheduleContextMutex.Lock()
	defer scheduleContextMutex.Unlock()

	if globalScheduleContextService != nil {
		return fmt.Errorf("schedule context service already initialized")
	}

	globalScheduleContextService = NewScheduleContextService()
	log.Printf("📅 Schedule Context Service initialized with %d city coordinates", len(globalScheduleContextService.cityCoords))

	return nil
}

// GetScheduleContextService returns the global schedule context service
func GetScheduleContextService() *ScheduleContextService {
	scheduleContextMutex.Lock()
	defer scheduleContextMutex.Unlock()
	return globalScheduleContextService
}
