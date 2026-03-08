package services

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// SituationalAnalyzer detects and calculates contextual factors affecting game predictions
type SituationalAnalyzer struct {
	teamCode          string
	advancedAnalytics *AdvancedAnalyticsService
}

// NewSituationalAnalyzer creates a new situational analyzer
func NewSituationalAnalyzer(teamCode string) *SituationalAnalyzer {
	return &SituationalAnalyzer{
		teamCode:          teamCode,
		advancedAnalytics: NewAdvancedAnalyticsService(),
	}
}

// AnalyzeSituationalFactors calculates comprehensive situational factors with advanced analytics
func (sa *SituationalAnalyzer) AnalyzeSituationalFactors(teamCode, opponentCode string, gameVenue string, isHome bool) (*models.PredictionFactors, error) {
	fmt.Printf("🔍 Analyzing situational factors for %s vs %s...\n", teamCode, opponentCode)

	// Get base factors first
	baseFactors, err := sa.getBasePredictionFactors(teamCode, opponentCode, isHome)
	if err != nil {
		return nil, fmt.Errorf("error getting base factors: %v", err)
	}

	// Analyze each situational category
	travelFatigue := sa.analyzeTravelFatigue(teamCode, gameVenue, isHome)
	altitudeAdjust := sa.analyzeAltitudeEffects(teamCode, gameVenue)
	scheduleStrength := sa.analyzeScheduleStrength(teamCode, opponentCode)
	injuryImpact := sa.analyzeInjuryImpact(teamCode) // Basic injury estimation
	momentumFactors := sa.analyzeMomentumFactors(teamCode)

	// NEW: Get advanced analytics integration
	log.Printf("📊 Computing advanced analytics for %s...", teamCode)
	advancedStats, err := sa.advancedAnalytics.GetAdvancedAnalytics(teamCode, isHome)
	if err != nil {
		log.Printf("⚠️  Warning: Could not get advanced analytics for %s: %v", teamCode, err)
		// Use default advanced stats with reasonable values
		advancedStats = &models.AdvancedAnalytics{
			XGForPerGame:       2.8,        // League average
			XGAgainstPerGame:   2.8,        // League average
			XGDifferential:     0.0,        // Neutral
			CorsiForPct:        0.50,       // Even possession
			FenwickForPct:      0.50,       // Even unblocked shots
			HighDangerPct:      0.50,       // Even high danger chances
			PossessionQuality:  50.0,       // Average possession quality
			GoalieSvPctOverall: 0.910,      // League average save %
			OverallRating:      50.0,       // Average team rating
			StrengthAreas:      []string{}, // No specific strengths
			WeaknessAreas:      []string{}, // No specific weaknesses
		}
	}

	// Add all factors to base factors
	baseFactors.TravelFatigue = travelFatigue
	baseFactors.AltitudeAdjust = altitudeAdjust
	baseFactors.ScheduleStrength = scheduleStrength
	baseFactors.InjuryImpact = injuryImpact
	baseFactors.MomentumFactors = momentumFactors
	baseFactors.AdvancedStats = *advancedStats // Advanced analytics integration

	// Weather is intentionally disabled for prediction quality and consistency.
	baseFactors.WeatherAnalysis = sa.defaultWeatherAnalysis(teamCode, opponentCode, gameVenue)

	fmt.Printf("✅ Situational analysis complete for %s (Advanced Rating: %.1f)\n",
		teamCode, advancedStats.OverallRating)
	return baseFactors, nil
}

func (sa *SituationalAnalyzer) defaultWeatherAnalysis(teamCode, opponentCode, gameVenue string) models.WeatherAnalysis {
	return models.WeatherAnalysis{
		GameID:        fmt.Sprintf("%s_vs_%s", teamCode, opponentCode),
		HomeTeam:      teamCode,
		AwayTeam:      opponentCode,
		GameDate:      time.Now().Add(24 * time.Hour),
		VenueName:     gameVenue,
		VenueCity:     gameVenue,
		IsOutdoorGame: false,
		WeatherConditions: models.WeatherConditions{
			Temperature:   45.0,
			FeelsLike:     45.0,
			Humidity:      50.0,
			WindSpeed:     5.0,
			WindDirection: 180,
			WindGust:      8.0,
			Precipitation: 0.0,
			PrecipType:    "none",
			Visibility:    10.0,
			CloudCover:    30.0,
			DataSource:    "Disabled",
			Reliability:   0.0,
		},
		TravelImpact: models.TravelWeatherImpact{
			OverallImpact: 0.0,
		},
		GameImpact: models.GameWeatherImpact{
			OverallGameImpact: 0.0,
		},
		OverallImpact: 0.0,
		Confidence:    0.0,
		LastUpdated:   time.Now(),
	}
}

// analyzeTravelFatigue calculates travel-related fatigue factors
func (sa *SituationalAnalyzer) analyzeTravelFatigue(teamCode, gameVenue string, isHome bool) models.TravelFatigue {
	fmt.Printf("✈️ Analyzing travel fatigue for %s...\n", teamCode)

	// Get team's home city for distance calculations
	homeCity := sa.getTeamHomeCity(teamCode)
	venueCity := sa.getVenueCity(gameVenue)

	if isHome {
		// Home team has minimal travel fatigue
		return models.TravelFatigue{
			MilesTraveled:    0,
			TimeZonesCrossed: 0,
			DaysOnRoad:       0,
			FatigueScore:     0.0,
		}
	}

	// Calculate travel burden for away team
	miles := sa.calculateDistance(homeCity, venueCity)
	timeZones := sa.calculateTimeZoneDifference(homeCity, venueCity)
	daysOnRoad := sa.calculateDaysOnRoad(teamCode)

	// Calculate overall fatigue score (0.0 = no fatigue, 1.0 = maximum fatigue)
	fatigueScore := sa.calculateFatigueScore(miles, timeZones, daysOnRoad)

	return models.TravelFatigue{
		MilesTraveled:    miles,
		TimeZonesCrossed: timeZones,
		DaysOnRoad:       daysOnRoad,
		FatigueScore:     fatigueScore,
	}
}

// analyzeAltitudeEffects calculates altitude-related performance adjustments
func (sa *SituationalAnalyzer) analyzeAltitudeEffects(teamCode, gameVenue string) models.AltitudeAdjust {
	fmt.Printf("🏔️ Analyzing altitude effects for %s at %s...\n", teamCode, gameVenue)

	venueAltitude := sa.getVenueAltitude(gameVenue)
	teamHomeAltitude := sa.getTeamHomeAltitude(teamCode)
	altitudeDiff := venueAltitude - teamHomeAltitude

	// Calculate performance adjustment (-1.0 to +1.0)
	adjustmentFactor := sa.calculateAltitudeAdjustment(altitudeDiff)

	return models.AltitudeAdjust{
		VenueAltitude:    venueAltitude,
		TeamHomeAltitude: teamHomeAltitude,
		AltitudeDiff:     altitudeDiff,
		AdjustmentFactor: adjustmentFactor,
	}
}

// analyzeScheduleStrength evaluates recent schedule difficulty
func (sa *SituationalAnalyzer) analyzeScheduleStrength(teamCode, opponentCode string) models.ScheduleStrength {
	fmt.Printf("📅 Analyzing schedule strength for %s...\n", teamCode)

	gamesLast7 := sa.countRecentGames(teamCode, 7)
	opponentStrength := sa.getOpponentStrength(teamCode)
	restAdvantage := sa.calculateRestAdvantage(teamCode, opponentCode)
	scheduleDensity := sa.calculateScheduleDensity(teamCode)

	return models.ScheduleStrength{
		GamesInLast7Days: gamesLast7,
		OpponentStrength: opponentStrength,
		RestAdvantage:    restAdvantage,
		ScheduleDensity:  scheduleDensity,
	}
}

// analyzeInjuryImpact assesses roster and injury effects
func (sa *SituationalAnalyzer) analyzeInjuryImpact(teamCode string) models.InjuryImpact {
	fmt.Printf("🏥 Analyzing injury impact for %s...\n", teamCode)

	keyPlayersOut := sa.countKeyInjuries(teamCode)
	goalieStatus := sa.getGoalieStatus(teamCode)
	lineupChanges := sa.countRecentLineupChanges(teamCode)
	injuryScore := sa.calculateInjuryScore(keyPlayersOut, goalieStatus, lineupChanges)

	return models.InjuryImpact{
		KeyPlayersOut: keyPlayersOut,
		GoalieStatus:  goalieStatus,
		InjuryScore:   injuryScore,
		LineupChanges: lineupChanges,
	}
}

// analyzeMomentumFactors evaluates psychological/momentum factors
func (sa *SituationalAnalyzer) analyzeMomentumFactors(teamCode string) models.MomentumFactors {
	fmt.Printf("🔥 Analyzing momentum factors for %s...\n", teamCode)

	winStreak := sa.getCurrentStreak(teamCode)
	homeStandLength := sa.getHomeStandLength(teamCode)
	lastGameMargin := sa.getLastGameMargin(teamCode)
	recentBlowouts := sa.countRecentBlowouts(teamCode, 5)
	momentumScore := sa.calculateMomentumScore(winStreak, homeStandLength, lastGameMargin, recentBlowouts)

	return models.MomentumFactors{
		WinStreak:       winStreak,
		HomeStandLength: homeStandLength,
		LastGameMargin:  lastGameMargin,
		RecentBlowouts:  recentBlowouts,
		MomentumScore:   momentumScore,
	}
}

// Helper methods for specific calculations

func (sa *SituationalAnalyzer) getTeamHomeCity(teamCode string) string {
	cityMap := map[string]string{
		"UTA": "Salt Lake City, UT", "VGK": "Las Vegas, NV", "COL": "Denver, CO",
		"SJS": "San Jose, CA", "LAK": "Los Angeles, CA", "ANA": "Anaheim, CA",
		"SEA": "Seattle, WA", "VAN": "Vancouver, BC", "CGY": "Calgary, AB",
		"EDM": "Edmonton, AB", "WPG": "Winnipeg, MB", "MIN": "Minneapolis, MN",
		"CHI": "Chicago, IL", "STL": "St. Louis, MO", "NSH": "Nashville, TN",
		"DAL": "Dallas, TX", "BOS": "Boston, MA", "NYR": "New York, NY",
		"NYI": "New York, NY", "NJD": "Newark, NJ", "PHI": "Philadelphia, PA",
		"PIT": "Pittsburgh, PA", "WSH": "Washington, DC", "CAR": "Raleigh, NC",
		"FLA": "Sunrise, FL", "TBL": "Tampa, FL", "TOR": "Toronto, ON",
		"MTL": "Montreal, QC", "OTT": "Ottawa, ON", "BUF": "Buffalo, NY",
		"DET": "Detroit, MI", "CBJ": "Columbus, OH",
	}

	if city, exists := cityMap[teamCode]; exists {
		return city
	}
	return "Unknown"
}

func (sa *SituationalAnalyzer) getVenueCity(venue string) string {
	// Map venue names to cities (simplified)
	venueMap := map[string]string{
		"Delta Center":             "Salt Lake City, UT",
		"T-Mobile Arena":           "Las Vegas, NV",
		"Ball Arena":               "Denver, CO",
		"SAP Center":               "San Jose, CA",
		"Crypto.com Arena":         "Los Angeles, CA",
		"Honda Center":             "Anaheim, CA",
		"Climate Pledge Arena":     "Seattle, WA",
		"Rogers Arena":             "Vancouver, BC",
		"Scotiabank Saddledome":    "Calgary, AB",
		"Rogers Place":             "Edmonton, AB",
		"Canada Life Centre":       "Winnipeg, MB",
		"Xcel Energy Center":       "Minneapolis, MN",
		"United Center":            "Chicago, IL",
		"Enterprise Center":        "St. Louis, MO",
		"Bridgestone Arena":        "Nashville, TN",
		"American Airlines Center": "Dallas, TX",
	}

	if city, exists := venueMap[venue]; exists {
		return city
	}
	return "Unknown"
}

func (sa *SituationalAnalyzer) calculateDistance(city1, city2 string) float64 {
	// Simplified distance calculation using city pairs
	// In real implementation, would use GPS coordinates and haversine formula

	if city1 == city2 {
		return 0
	}

	// Sample distances for common routes (miles)
	distanceMap := map[string]float64{
		"Salt Lake City, UT-Las Vegas, NV":   350,
		"Salt Lake City, UT-Denver, CO":      350,
		"Salt Lake City, UT-San Jose, CA":    650,
		"Salt Lake City, UT-Los Angeles, CA": 580,
		"Salt Lake City, UT-Seattle, WA":     650,
		"Las Vegas, NV-Los Angeles, CA":      270,
		"Los Angeles, CA-San Jose, CA":       350,
		"Denver, CO-Dallas, TX":              650,
		"Chicago, IL-Detroit, MI":            280,
		"New York, NY-Boston, MA":            200,
		"Toronto, ON-Montreal, QC":           340,
	}

	key1 := city1 + "-" + city2
	key2 := city2 + "-" + city1

	if distance, exists := distanceMap[key1]; exists {
		return distance
	}
	if distance, exists := distanceMap[key2]; exists {
		return distance
	}

	// Default moderate distance for unknown routes
	return 500
}

func (sa *SituationalAnalyzer) calculateTimeZoneDifference(city1, city2 string) int {
	timeZoneMap := map[string]int{
		"Salt Lake City, UT": -7, "Denver, CO": -7, "Las Vegas, NV": -8,
		"Los Angeles, CA": -8, "San Jose, CA": -8, "Anaheim, CA": -8, "Seattle, WA": -8,
		"Vancouver, BC": -8, "Calgary, AB": -7, "Edmonton, AB": -7, "Winnipeg, MB": -6,
		"Minneapolis, MN": -6, "Chicago, IL": -6, "St. Louis, MO": -6, "Nashville, TN": -6,
		"Dallas, TX": -6, "Boston, MA": -5, "New York, NY": -5, "Philadelphia, PA": -5,
		"Pittsburgh, PA": -5, "Washington, DC": -5, "Detroit, MI": -5, "Toronto, ON": -5,
		"Montreal, QC": -5, "Ottawa, ON": -5, "Buffalo, NY": -5, "Columbus, OH": -5,
		"Raleigh, NC": -5, "Tampa, FL": -5, "Sunrise, FL": -5,
	}

	tz1, exists1 := timeZoneMap[city1]
	tz2, exists2 := timeZoneMap[city2]

	if exists1 && exists2 {
		return int(math.Abs(float64(tz1 - tz2)))
	}
	return 0
}

func (sa *SituationalAnalyzer) calculateFatigueScore(miles float64, timeZones int, daysOnRoad int) float64 {
	// Combine factors to create fatigue score (0.0 to 1.0)
	milesFactor := math.Min(miles/2000.0, 1.0)              // Max at 2000 miles
	timeZoneFactor := math.Min(float64(timeZones)/4.0, 1.0) // Max at 4 time zones
	roadFactor := math.Min(float64(daysOnRoad)/10.0, 1.0)   // Max at 10 days on road

	return (milesFactor*0.4 + timeZoneFactor*0.3 + roadFactor*0.3)
}

func (sa *SituationalAnalyzer) getVenueAltitude(venue string) float64 {
	altitudeMap := map[string]float64{
		"Delta Center":             4226, // Salt Lake City
		"Ball Arena":               5280, // Denver (Mile High)
		"Scotiabank Saddledome":    3557, // Calgary
		"Rogers Place":             2200, // Edmonton
		"Xcel Energy Center":       850,  // Minneapolis
		"T-Mobile Arena":           2000, // Las Vegas
		"SAP Center":               82,   // San Jose (sea level)
		"Crypto.com Arena":         285,  // Los Angeles
		"Honda Center":             160,  // Anaheim
		"Climate Pledge Arena":     56,   // Seattle
		"Rogers Arena":             200,  // Vancouver
		"United Center":            600,  // Chicago
		"Enterprise Center":        465,  // St. Louis
		"Bridgestone Arena":        550,  // Nashville
		"American Airlines Center": 430,  // Dallas
	}

	if altitude, exists := altitudeMap[venue]; exists {
		return altitude
	}
	return 1000 // Default moderate altitude
}

func (sa *SituationalAnalyzer) getTeamHomeAltitude(teamCode string) float64 {
	altitudeMap := map[string]float64{
		"UTA": 4226, "COL": 5280, "CGY": 3557, "EDM": 2200,
		"VGK": 2000, "SJS": 82, "LAK": 285, "ANA": 160,
		"SEA": 56, "VAN": 200, "CHI": 600, "STL": 465,
		"NSH": 550, "DAL": 430, "MIN": 850,
	}

	if altitude, exists := altitudeMap[teamCode]; exists {
		return altitude
	}
	return 1000 // Default
}

func (sa *SituationalAnalyzer) calculateAltitudeAdjustment(altitudeDiff float64) float64 {
	// Teams from lower altitude struggle at high altitude
	// Teams from higher altitude get slight advantage at lower altitude

	if math.Abs(altitudeDiff) < 1000 {
		return 0.0 // Minimal effect
	}

	if altitudeDiff > 0 {
		// Playing at higher altitude than home
		return -math.Min(altitudeDiff/5000.0, 0.15) // Max 15% penalty
	} else {
		// Playing at lower altitude than home
		return math.Min(math.Abs(altitudeDiff)/8000.0, 0.08) // Max 8% bonus
	}
}

// Simplified calculation methods (in real implementation, would query APIs/databases)

func (sa *SituationalAnalyzer) calculateDaysOnRoad(teamCode string) int {
	// Simplified: estimate based on team code
	return (len(teamCode) + 2) % 5 // 0-4 days
}

func (sa *SituationalAnalyzer) countRecentGames(teamCode string, days int) int {
	// Simplified: estimate games in last N days
	return int(math.Min(float64(days)/2, 4)) // Max 4 games in 7 days
}

func (sa *SituationalAnalyzer) getOpponentStrength(teamCode string) float64 {
	// Simplified: average recent opponent win percentage
	return 0.5 + (float64(len(teamCode)%6)-3)/10.0 // 0.2 to 0.8
}

func (sa *SituationalAnalyzer) calculateRestAdvantage(teamCode, opponentCode string) float64 {
	// Simplified: rest days difference
	teamRest := (len(teamCode) % 4) + 1
	oppRest := (len(opponentCode) % 4) + 1
	return float64(teamRest-oppRest) / 4.0 // -0.75 to +0.75
}

func (sa *SituationalAnalyzer) calculateScheduleDensity(teamCode string) float64 {
	games := sa.countRecentGames(teamCode, 14)
	return float64(games) / 14.0 // Games per day over 2 weeks
}

func (sa *SituationalAnalyzer) countKeyInjuries(teamCode string) int {
	// Simplified: estimate key injuries
	return len(teamCode) % 3 // 0-2 key injuries
}

func (sa *SituationalAnalyzer) getGoalieStatus(teamCode string) string {
	statuses := []string{"starter", "backup", "emergency"}
	return statuses[len(teamCode)%3]
}

func (sa *SituationalAnalyzer) countRecentLineupChanges(teamCode string) int {
	return (len(teamCode) * 2) % 5 // 0-4 changes
}

func (sa *SituationalAnalyzer) calculateInjuryScore(keyInjuries int, goalieStatus string, lineupChanges int) float64 {
	score := float64(keyInjuries) * 0.15 // Each key injury = 15%

	if goalieStatus == "backup" {
		score += 0.10
	} else if goalieStatus == "emergency" {
		score += 0.25
	}

	score += float64(lineupChanges) * 0.02 // Each change = 2%

	return math.Min(score, 1.0)
}

func (sa *SituationalAnalyzer) getCurrentStreak(teamCode string) int {
	// Simplified: return current win/loss streak (negative for losses)
	streak := (len(teamCode) * 3 % 10) - 5 // -5 to +4
	return streak
}

func (sa *SituationalAnalyzer) getHomeStandLength(teamCode string) int {
	return (len(teamCode) % 6) + 1 // 1-6 games in home stand
}

func (sa *SituationalAnalyzer) getLastGameMargin(teamCode string) int {
	return ((len(teamCode) * 7) % 11) - 5 // -5 to +5 goal margin
}

func (sa *SituationalAnalyzer) countRecentBlowouts(teamCode string, games int) int {
	return (len(teamCode) % 3) // 0-2 blowouts in recent games
}

func (sa *SituationalAnalyzer) calculateMomentumScore(streak, homeStand, margin, blowouts int) float64 {
	score := 0.5 // Neutral baseline

	// Win streak adds momentum
	if streak > 0 {
		score += math.Min(float64(streak)*0.05, 0.3) // Max 30% boost
	} else if streak < 0 {
		score -= math.Min(float64(-streak)*0.04, 0.25) // Max 25% penalty
	}

	// Recent blowout wins add momentum
	score += float64(blowouts) * 0.08

	// Last game margin influences momentum
	score += float64(margin) * 0.02

	// Long home stands can be either good (comfort) or bad (fatigue)
	if homeStand > 4 {
		score -= 0.05 // Slight penalty for very long home stands
	}

	return math.Max(0.0, math.Min(score, 1.0))
}

// getBasePredictionFactors gets the standard prediction factors
func (sa *SituationalAnalyzer) getBasePredictionFactors(teamCode, opponentCode string, isHome bool) (*models.PredictionFactors, error) {
	// Get standings for basic stats
	standings, err := GetStandings()
	if err != nil {
		return nil, fmt.Errorf("error getting standings: %v", err)
	}

	// Find team in standings
	var teamStanding *models.TeamStanding
	for _, standing := range standings.Standings {
		if standing.TeamAbbrev.Default == teamCode {
			teamStanding = &standing
			break
		}
	}

	if teamStanding == nil {
		return nil, fmt.Errorf("team %s not found in standings", teamCode)
	}

	// Calculate basic factors with safe division to prevent NaN
	gamesPlayed := float64(teamStanding.GamesPlayed)
	factors := &models.PredictionFactors{
		TeamCode:          teamCode,
		WinPercentage:     safeDiv(float64(teamStanding.Wins), gamesPlayed, 0.5), // Default 50% if no games
		HomeAdvantage:     sa.calculateHomeAdvantage(teamCode, isHome),
		RecentForm:        sa.calculateAdvancedRecentForm(teamCode),
		HeadToHead:        sa.calculateHeadToHead(teamCode, opponentCode),
		GoalsFor:          safeDiv(float64(teamStanding.GoalFor), gamesPlayed, 2.8),     // League avg ~2.8 goals
		GoalsAgainst:      safeDiv(float64(teamStanding.GoalAgainst), gamesPlayed, 2.8), // League avg
		PowerPlayPct:      sa.estimatePowerPlayPct(teamCode),
		PenaltyKillPct:    sa.estimatePenaltyKillPct(teamCode),
		RestDays:          sa.calculateRestDays(teamCode),
		BackToBackPenalty: sa.calculateBackToBackPenalty(teamCode),
	}

	return factors, nil
}

// Helper methods from original predictions service
func (sa *SituationalAnalyzer) calculateHomeAdvantage(teamCode string, isHome bool) float64 {
	if !isHome {
		return 0.0
	}

	baseAdvantage := 0.12
	strongHomeTeams := []string{"MTL", "BOS", "CGY", "EDM", "WPG"}
	for _, team := range strongHomeTeams {
		if team == teamCode {
			return baseAdvantage + 0.03
		}
	}
	return baseAdvantage
}

func (sa *SituationalAnalyzer) calculateAdvancedRecentForm(teamCode string) float64 {
	hashVal := 0
	for _, r := range teamCode {
		hashVal += int(r)
	}
	return 0.3 + float64(hashVal%40)/100.0
}

func (sa *SituationalAnalyzer) calculateHeadToHead(teamCode, opponentCode string) float64 {
	hashDiff := 0
	for i, r := range teamCode {
		if i < len(opponentCode) {
			hashDiff += int(r) - int(rune(opponentCode[i]))
		}
	}
	return float64(hashDiff%40-20) / 100.0
}

func (sa *SituationalAnalyzer) estimatePowerPlayPct(teamCode string) float64 {
	hashVal := 0
	for _, r := range teamCode {
		hashVal += int(r)
	}
	return 0.15 + float64(hashVal%10)/100.0
}

func (sa *SituationalAnalyzer) estimatePenaltyKillPct(teamCode string) float64 {
	hashVal := 0
	for _, r := range teamCode {
		hashVal += int(r) * 2
	}
	return 0.75 + float64(hashVal%10)/100.0
}

func (sa *SituationalAnalyzer) calculateRestDays(teamCode string) int {
	return 1 + (len(teamCode) % 3)
}

func (sa *SituationalAnalyzer) calculateBackToBackPenalty(teamCode string) float64 {
	restDays := sa.calculateRestDays(teamCode)
	if restDays == 0 {
		return 0.15
	}
	return 0.0
}
