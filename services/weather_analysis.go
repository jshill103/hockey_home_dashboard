package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// WeatherAnalysisService handles comprehensive weather impact analysis for hockey games
type WeatherAnalysisService struct {
	dataSources  []models.WeatherDataSource
	cache        map[string]*models.WeatherAnalysis
	cacheMutex   sync.RWMutex
	cacheExpiry  time.Duration
	isEnabled    bool
	outdoorGames map[string]models.OutdoorGameInfo // Game ID -> Outdoor game info
	cityCoords   map[string]CityCoordinates        // City name -> coordinates
}

// CityCoordinates represents latitude and longitude for a city
type CityCoordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	TimeZone  string  `json:"timeZone"`
}

// OpenWeatherMapResponse represents the API response from OpenWeatherMap
type OpenWeatherMapResponse struct {
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Humidity  float64 `json:"humidity"`
		Pressure  float64 `json:"pressure"`
	} `json:"main"`
	Visibility float64 `json:"visibility"`
	Wind       struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
		Gust  float64 `json:"gust"`
	} `json:"wind"`
	Clouds struct {
		All float64 `json:"all"`
	} `json:"clouds"`
	Rain struct {
		OneHour   float64 `json:"1h"`
		ThreeHour float64 `json:"3h"`
	} `json:"rain"`
	Snow struct {
		OneHour   float64 `json:"1h"`
		ThreeHour float64 `json:"3h"`
	} `json:"snow"`
	Sys struct {
		Country string `json:"country"`
	} `json:"sys"`
	Name string `json:"name"`
}

// NewWeatherAnalysisService creates a new weather analysis service
func NewWeatherAnalysisService() *WeatherAnalysisService {
	service := &WeatherAnalysisService{
		cache:        make(map[string]*models.WeatherAnalysis),
		cacheExpiry:  30 * time.Minute, // Cache weather data for 30 minutes
		isEnabled:    true,
		outdoorGames: make(map[string]models.OutdoorGameInfo),
		cityCoords:   initializeCityCoordinates(),
	}

	// Initialize weather data sources
	service.initializeDataSources()

	// Load known outdoor games
	service.loadOutdoorGames()

	log.Printf("🌦️ Weather Analysis Service initialized with %d data sources", len(service.dataSources))
	return service
}

// initializeDataSources sets up available weather data sources
func (w *WeatherAnalysisService) initializeDataSources() {
	enabledSources := 0

	// OpenWeatherMap (free tier with 1000 calls/day)
	openWeatherAPIKey := os.Getenv("OPENWEATHER_API_KEY")
	if openWeatherAPIKey != "" {
		w.dataSources = append(w.dataSources, models.WeatherDataSource{
			Name:           "OpenWeatherMap",
			APIKey:         openWeatherAPIKey,
			BaseURL:        "https://api.openweathermap.org/data/2.5",
			RateLimit:      1000, // per day for free tier
			Reliability:    0.85,
			CostPerRequest: 0.0, // Free tier
			Features:       []string{"current", "forecast", "alerts"},
			IsEnabled:      true,
			Priority:       1,
		})
		enabledSources++
		log.Printf("✅ Weather source enabled: OpenWeatherMap")
	} else {
		log.Printf("❌ Weather source disabled: OpenWeatherMap (no API key - set OPENWEATHER_API_KEY)")
	}

	// WeatherAPI (free tier with 1M calls/month)
	weatherAPIKey := os.Getenv("WEATHER_API_KEY")
	if weatherAPIKey != "" {
		w.dataSources = append(w.dataSources, models.WeatherDataSource{
			Name:           "WeatherAPI",
			APIKey:         weatherAPIKey,
			BaseURL:        "https://api.weatherapi.com/v1",
			RateLimit:      33333, // ~1M per month / 30 days / 24 hours
			Reliability:    0.90,
			CostPerRequest: 0.0, // Free tier
			Features:       []string{"current", "forecast", "alerts", "history", "sports"},
			IsEnabled:      true,
			Priority:       2,
		})
		enabledSources++
		log.Printf("✅ Weather source enabled: WeatherAPI")
	} else {
		log.Printf("❌ Weather source disabled: WeatherAPI (no API key - set WEATHER_API_KEY)")
	}

	// AccuWeather (free tier with 50 calls/day)
	accuWeatherAPIKey := os.Getenv("ACCUWEATHER_API_KEY")
	if accuWeatherAPIKey != "" {
		w.dataSources = append(w.dataSources, models.WeatherDataSource{
			Name:           "AccuWeather",
			APIKey:         accuWeatherAPIKey,
			BaseURL:        "https://dataservice.accuweather.com",
			RateLimit:      50, // per day for free tier
			Reliability:    0.88,
			CostPerRequest: 0.0, // Free tier
			Features:       []string{"current", "forecast", "alerts"},
			IsEnabled:      true,
			Priority:       3,
		})
		enabledSources++
		log.Printf("✅ Weather source enabled: AccuWeather")
	} else {
		log.Printf("❌ Weather source disabled: AccuWeather (no API key - set ACCUWEATHER_API_KEY)")
	}

	// Update service enabled status based on available data sources
	w.isEnabled = enabledSources > 0

	if enabledSources == 0 {
		log.Printf("⚠️ No weather API keys found - weather analysis disabled")
		log.Printf("💡 To enable weather analysis, set one or more environment variables:")
		log.Printf("   - OPENWEATHER_API_KEY (OpenWeatherMap)")
		log.Printf("   - WEATHER_API_KEY (WeatherAPI)")
		log.Printf("   - ACCUWEATHER_API_KEY (AccuWeather)")
	} else {
		log.Printf("🌦️ Weather analysis enabled with %d data source(s)", enabledSources)
	}
}

// loadOutdoorGames initializes known outdoor games and special events
func (w *WeatherAnalysisService) loadOutdoorGames() {
	// NHL Winter Classic and Stadium Series games would be loaded here
	// For now, we'll include some mock data for testing

	// Example: 2025 Winter Classic (hypothetical)
	w.outdoorGames["2025010001"] = models.OutdoorGameInfo{
		IsOutdoorGame: true,
		GameType:      "Winter Classic",
		VenueType:     "Football Stadium",
		SpecialConsiderations: []string{
			"Large venue with wind exposure",
			"Temporary ice surface",
			"Extended intermissions for ice maintenance",
			"Player bench heating",
		},
	}

	// Example: Stadium Series games
	w.outdoorGames["2025020150"] = models.OutdoorGameInfo{
		IsOutdoorGame: true,
		GameType:      "Stadium Series",
		VenueType:     "Baseball Stadium",
		SpecialConsiderations: []string{
			"Baseball stadium configuration",
			"Unique sight lines",
			"Weather exposure",
			"Temporary facilities",
		},
	}

	log.Printf("🏟️ Loaded %d outdoor game configurations", len(w.outdoorGames))
}

// DetectOutdoorGame attempts to detect if a game is played outdoors based on various factors
func (w *WeatherAnalysisService) DetectOutdoorGame(gameID, venueName, venueCity string, gameDate time.Time) (bool, models.OutdoorGameInfo) {
	// Check if game is explicitly configured as outdoor
	if outdoorInfo, exists := w.outdoorGames[gameID]; exists {
		return true, outdoorInfo
	}

	// Check venue name for outdoor indicators
	if w.isOutdoorVenue(venueName) {
		return true, models.OutdoorGameInfo{
			IsOutdoorGame: true,
			GameType:      "Special Event",
			VenueType:     w.determineVenueType(venueName),
			SpecialConsiderations: []string{
				"Outdoor venue",
				"Weather dependent conditions",
				"Potential delays or cancellations",
			},
		}
	}

	// Check for special game dates (Winter Classic typically Jan 1, Stadium Series in Feb)
	if w.isSpecialGameDate(gameDate) {
		return true, models.OutdoorGameInfo{
			IsOutdoorGame: true,
			GameType:      w.determineGameTypeByDate(gameDate),
			VenueType:     "Stadium",
			SpecialConsiderations: []string{
				"Annual outdoor event",
				"Enhanced weather monitoring",
				"Special broadcast considerations",
			},
		}
	}

	// Default to indoor
	return false, models.OutdoorGameInfo{IsOutdoorGame: false}
}

// isOutdoorVenue checks if a venue name suggests an outdoor location
func (w *WeatherAnalysisService) isOutdoorVenue(venueName string) bool {
	outdoorIndicators := []string{
		"Stadium", "Field", "Park", "Bowl", "Coliseum",
		"Fenway", "Yankee", "Wrigley", "Soldier", "MetLife",
		"Lambeau", "Arrowhead", "Mile High", "Memorial",
	}

	venueUpper := strings.ToUpper(venueName)
	for _, indicator := range outdoorIndicators {
		if strings.Contains(venueUpper, strings.ToUpper(indicator)) {
			return true
		}
	}

	return false
}

// determineVenueType determines the type of outdoor venue
func (w *WeatherAnalysisService) determineVenueType(venueName string) string {
	venueUpper := strings.ToUpper(venueName)

	if strings.Contains(venueUpper, "STADIUM") || strings.Contains(venueUpper, "FIELD") {
		if strings.Contains(venueUpper, "BASEBALL") || strings.Contains(venueUpper, "FENWAY") ||
			strings.Contains(venueUpper, "YANKEE") || strings.Contains(venueUpper, "WRIGLEY") {
			return "Baseball Stadium"
		}
		return "Football Stadium"
	}

	if strings.Contains(venueUpper, "BOWL") || strings.Contains(venueUpper, "COLISEUM") {
		return "Multi-purpose Stadium"
	}

	return "Outdoor Venue"
}

// isSpecialGameDate checks if the date corresponds to typical outdoor game dates
func (w *WeatherAnalysisService) isSpecialGameDate(gameDate time.Time) bool {
	month := gameDate.Month()
	day := gameDate.Day()

	// Winter Classic (typically January 1st)
	if month == time.January && day == 1 {
		return true
	}

	// Stadium Series (typically February)
	if month == time.February {
		return true
	}

	// Heritage Classic (typically in Canada, various dates)
	if month == time.October || month == time.March {
		// Could add more specific logic here
		return false
	}

	return false
}

// determineGameTypeByDate determines the game type based on the date
func (w *WeatherAnalysisService) determineGameTypeByDate(gameDate time.Time) string {
	month := gameDate.Month()
	day := gameDate.Day()

	if month == time.January && day == 1 {
		return "Winter Classic"
	}

	if month == time.February {
		return "Stadium Series"
	}

	if month == time.October || month == time.March {
		return "Heritage Classic"
	}

	return "Special Outdoor Game"
}

// initializeCityCoordinates sets up coordinate mapping for NHL cities
func initializeCityCoordinates() map[string]CityCoordinates {
	coords := make(map[string]CityCoordinates)

	// NHL city coordinates
	coords["Anaheim"] = CityCoordinates{33.8366, -117.9143, "America/Los_Angeles"}
	coords["Boston"] = CityCoordinates{42.3601, -71.0589, "America/New_York"}
	coords["Buffalo"] = CityCoordinates{42.8864, -78.8784, "America/New_York"}
	coords["Calgary"] = CityCoordinates{51.0447, -114.0719, "America/Edmonton"}
	coords["Carolina"] = CityCoordinates{35.8032, -78.7811, "America/New_York"} // Raleigh
	coords["Chicago"] = CityCoordinates{41.8781, -87.6298, "America/Chicago"}
	coords["Colorado"] = CityCoordinates{39.7392, -104.9903, "America/Denver"} // Denver
	coords["Columbus"] = CityCoordinates{39.9612, -82.9988, "America/New_York"}
	coords["Dallas"] = CityCoordinates{32.7767, -96.7970, "America/Chicago"}
	coords["Detroit"] = CityCoordinates{42.3314, -83.0458, "America/New_York"}
	coords["Edmonton"] = CityCoordinates{53.5461, -113.4938, "America/Edmonton"}
	coords["Florida"] = CityCoordinates{26.1584, -80.3267, "America/New_York"} // Sunrise
	coords["Los Angeles"] = CityCoordinates{34.0522, -118.2437, "America/Los_Angeles"}
	coords["Minnesota"] = CityCoordinates{44.9778, -93.2650, "America/Chicago"} // Minneapolis
	coords["Montreal"] = CityCoordinates{45.5017, -73.5673, "America/New_York"}
	coords["Nashville"] = CityCoordinates{36.1627, -86.7816, "America/Chicago"}
	coords["New Jersey"] = CityCoordinates{40.7282, -74.1776, "America/New_York"} // Newark
	coords["New York"] = CityCoordinates{40.7128, -74.0060, "America/New_York"}
	coords["New York Islanders"] = CityCoordinates{40.7505, -73.5901, "America/New_York"} // Elmont
	coords["Ottawa"] = CityCoordinates{45.4215, -75.6972, "America/New_York"}
	coords["Philadelphia"] = CityCoordinates{39.9526, -75.1652, "America/New_York"}
	coords["Pittsburgh"] = CityCoordinates{40.4406, -79.9959, "America/New_York"}
	coords["San Jose"] = CityCoordinates{37.3382, -121.8863, "America/Los_Angeles"}
	coords["Seattle"] = CityCoordinates{47.6062, -122.3321, "America/Los_Angeles"}
	coords["St. Louis"] = CityCoordinates{38.6270, -90.1994, "America/Chicago"}
	coords["Tampa Bay"] = CityCoordinates{27.9506, -82.4572, "America/New_York"} // Tampa
	coords["Toronto"] = CityCoordinates{43.6532, -79.3832, "America/New_York"}
	coords["Utah"] = CityCoordinates{40.7608, -111.8910, "America/Denver"} // Salt Lake City
	coords["Vancouver"] = CityCoordinates{49.2827, -123.1207, "America/Vancouver"}
	coords["Vegas"] = CityCoordinates{36.1699, -115.1398, "America/Los_Angeles"} // Las Vegas
	coords["Washington"] = CityCoordinates{38.9072, -77.0369, "America/New_York"}
	coords["Winnipeg"] = CityCoordinates{49.8951, -97.1384, "America/Winnipeg"}

	return coords
}

// IsEnabled returns whether the weather analysis service is enabled
func (w *WeatherAnalysisService) IsEnabled() bool {
	return w.isEnabled
}

// getEnabledDataSources returns all enabled weather data sources sorted by priority
func (w *WeatherAnalysisService) getEnabledDataSources() []models.WeatherDataSource {
	var enabled []models.WeatherDataSource
	for _, source := range w.dataSources {
		if source.IsEnabled {
			enabled = append(enabled, source)
		}
	}

	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(enabled)-1; i++ {
		for j := i + 1; j < len(enabled); j++ {
			if enabled[i].Priority > enabled[j].Priority {
				enabled[i], enabled[j] = enabled[j], enabled[i]
			}
		}
	}

	return enabled
}

// AnalyzeWeatherImpact performs comprehensive weather analysis for a game
func (w *WeatherAnalysisService) AnalyzeWeatherImpact(homeTeam, awayTeam, venueCity string, gameDate time.Time, gameID string) (*models.WeatherAnalysis, error) {
	log.Printf("🌦️ Analyzing weather impact for %s vs %s in %s on %s", awayTeam, homeTeam, venueCity, gameDate.Format("2006-01-02"))

	if !w.IsEnabled() {
		log.Printf("⚠️ Weather analysis service disabled - no API keys provided")
		return nil, fmt.Errorf("weather analysis disabled: no API keys configured")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s_%s_%s_%s", homeTeam, awayTeam, venueCity, gameDate.Format("2006-01-02"))
	if cached := w.getCachedAnalysis(cacheKey); cached != nil {
		log.Printf("✅ Using cached weather analysis for %s", cacheKey)
		return cached, nil
	}

	// Get weather data
	weatherConditions, err := w.fetchWeatherData(venueCity, gameDate)
	if err != nil {
		log.Printf("❌ Error fetching weather data: %v", err)
		return nil, fmt.Errorf("failed to fetch weather data: %v", err)
	}

	// Check if this is an outdoor game
	isOutdoor, outdoorInfo := w.DetectOutdoorGame(gameID, w.getVenueName(venueCity), venueCity, gameDate)

	// Analyze travel weather impact
	travelImpact := w.analyzeTravelWeatherImpact(homeTeam, awayTeam, venueCity, gameDate, weatherConditions)

	// Analyze game weather impact
	gameImpact := w.analyzeGameWeatherImpact(weatherConditions, isOutdoor, outdoorInfo)

	// Calculate overall impact
	overallImpact := w.calculateOverallWeatherImpact(travelImpact, gameImpact, isOutdoor)

	// Create weather analysis
	analysis := &models.WeatherAnalysis{
		GameID:            gameID,
		HomeTeam:          homeTeam,
		AwayTeam:          awayTeam,
		GameDate:          gameDate,
		VenueName:         w.getVenueName(venueCity),
		VenueCity:         venueCity,
		IsOutdoorGame:     isOutdoor,
		WeatherConditions: *weatherConditions,
		TravelImpact:      travelImpact,
		GameImpact:        gameImpact,
		OverallImpact:     overallImpact,
		Confidence:        weatherConditions.Reliability,
		LastUpdated:       time.Now(),
	}

	// Cache the analysis
	w.cacheAnalysis(cacheKey, analysis)

	log.Printf("✅ Weather analysis complete: Overall Impact %.2f, Confidence %.1f%%",
		overallImpact, weatherConditions.Reliability*100)

	return analysis, nil
}

// fetchWeatherData retrieves weather data from available sources
func (w *WeatherAnalysisService) fetchWeatherData(city string, gameDate time.Time) (*models.WeatherConditions, error) {
	coords, exists := w.cityCoords[city]
	if !exists {
		return nil, fmt.Errorf("coordinates not found for city: %s", city)
	}

	enabledSources := w.getEnabledDataSources()
	if len(enabledSources) == 0 {
		return nil, fmt.Errorf("no enabled weather data sources")
	}

	// Try each data source in priority order
	for _, source := range enabledSources {
		conditions, err := w.fetchFromSource(source, coords, gameDate)
		if err != nil {
			log.Printf("⚠️ Failed to fetch from %s: %v", source.Name, err)
			continue
		}

		conditions.DataSource = source.Name
		conditions.Reliability = source.Reliability
		conditions.DataAge = time.Since(time.Now()) // Will be updated with actual data age

		return conditions, nil
	}

	return nil, fmt.Errorf("all weather data sources failed")
}

// fetchFromSource fetches weather data from a specific source
func (w *WeatherAnalysisService) fetchFromSource(source models.WeatherDataSource, coords CityCoordinates, gameDate time.Time) (*models.WeatherConditions, error) {
	switch source.Name {
	case "OpenWeatherMap":
		return w.fetchFromOpenWeatherMap(source, coords, gameDate)
	case "WeatherAPI":
		return w.fetchFromWeatherAPI(source, coords, gameDate)
	case "MockWeatherService":
		return nil, fmt.Errorf("mock weather service no longer supported")
	default:
		return nil, fmt.Errorf("unsupported weather data source: %s", source.Name)
	}
}

// fetchFromOpenWeatherMap fetches data from OpenWeatherMap API
func (w *WeatherAnalysisService) fetchFromOpenWeatherMap(source models.WeatherDataSource, coords CityCoordinates, gameDate time.Time) (*models.WeatherConditions, error) {
	if source.APIKey == "" {
		return nil, fmt.Errorf("OpenWeatherMap API key not configured")
	}

	// Build API URL
	apiURL := fmt.Sprintf("%s/weather?lat=%f&lon=%f&appid=%s&units=imperial",
		source.BaseURL, coords.Latitude, coords.Longitude, source.APIKey)

	// Make HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var owmResp OpenWeatherMapResponse
	if err := json.Unmarshal(body, &owmResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to our weather conditions format
	conditions := &models.WeatherConditions{
		Temperature:   owmResp.Main.Temp,
		FeelsLike:     owmResp.Main.FeelsLike,
		Humidity:      owmResp.Main.Humidity,
		WindSpeed:     owmResp.Wind.Speed,
		WindDirection: owmResp.Wind.Deg,
		WindGust:      owmResp.Wind.Gust,
		Visibility:    owmResp.Visibility / 1609.34, // Convert meters to miles
		CloudCover:    owmResp.Clouds.All,
		Precipitation: owmResp.Rain.OneHour + owmResp.Snow.OneHour,
		PrecipType:    w.determinePrecipType(owmResp),
		DataSource:    source.Name,
		Reliability:   source.Reliability,
		DataAge:       0, // Current data
	}

	return conditions, nil
}

// fetchFromWeatherAPI fetches data from WeatherAPI
func (w *WeatherAnalysisService) fetchFromWeatherAPI(source models.WeatherDataSource, coords CityCoordinates, gameDate time.Time) (*models.WeatherConditions, error) {
	if source.APIKey == "" {
		return nil, fmt.Errorf("WeatherAPI key not configured")
	}

	// WeatherAPI implementation would go here
	// Real API implementation needed
	return nil, fmt.Errorf("WeatherAPI implementation not yet available")
}

// determinePrecipType determines precipitation type from OpenWeatherMap response
func (w *WeatherAnalysisService) determinePrecipType(resp OpenWeatherMapResponse) string {
	if resp.Snow.OneHour > 0 || resp.Snow.ThreeHour > 0 {
		return "snow"
	}
	if resp.Rain.OneHour > 0 || resp.Rain.ThreeHour > 0 {
		return "rain"
	}

	// Check weather description for other types
	for _, weather := range resp.Weather {
		desc := strings.ToLower(weather.Description)
		if strings.Contains(desc, "sleet") || strings.Contains(desc, "freezing") {
			return "sleet"
		}
	}

	return "none"
}

// getVenueName returns the venue name for a city
func (w *WeatherAnalysisService) getVenueName(city string) string {
	venueMap := map[string]string{
		"Anaheim":      "Honda Center",
		"Boston":       "TD Garden",
		"Buffalo":      "KeyBank Center",
		"Calgary":      "Scotiabank Saddledome",
		"Carolina":     "PNC Arena",
		"Chicago":      "United Center",
		"Colorado":     "Ball Arena",
		"Columbus":     "Nationwide Arena",
		"Dallas":       "American Airlines Center",
		"Detroit":      "Little Caesars Arena",
		"Edmonton":     "Rogers Place",
		"Florida":      "FLA Live Arena",
		"Los Angeles":  "Crypto.com Arena",
		"Minnesota":    "Xcel Energy Center",
		"Montreal":     "Bell Centre",
		"Nashville":    "Bridgestone Arena",
		"New Jersey":   "Prudential Center",
		"New York":     "Madison Square Garden",
		"Ottawa":       "Canadian Tire Centre",
		"Philadelphia": "Wells Fargo Center",
		"Pittsburgh":   "PPG Paints Arena",
		"San Jose":     "SAP Center",
		"Seattle":      "Climate Pledge Arena",
		"St. Louis":    "Enterprise Center",
		"Tampa Bay":    "Amalie Arena",
		"Toronto":      "Scotiabank Arena",
		"Utah":         "Delta Center",
		"Vancouver":    "Rogers Arena",
		"Vegas":        "T-Mobile Arena",
		"Washington":   "Capital One Arena",
		"Winnipeg":     "Canada Life Centre",
	}

	if venue, exists := venueMap[city]; exists {
		return venue
	}
	return fmt.Sprintf("%s Arena", city)
}

// getCachedAnalysis retrieves cached weather analysis
func (w *WeatherAnalysisService) getCachedAnalysis(key string) *models.WeatherAnalysis {
	w.cacheMutex.RLock()
	defer w.cacheMutex.RUnlock()

	if analysis, exists := w.cache[key]; exists {
		if time.Since(analysis.LastUpdated) < w.cacheExpiry {
			return analysis
		}
		// Cache expired, remove it
		delete(w.cache, key)
	}

	return nil
}

// cacheAnalysis stores weather analysis in cache
func (w *WeatherAnalysisService) cacheAnalysis(key string, analysis *models.WeatherAnalysis) {
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()

	w.cache[key] = analysis
}

// analyzeTravelWeatherImpact analyzes how weather affects team travel
func (w *WeatherAnalysisService) analyzeTravelWeatherImpact(homeTeam, awayTeam, venueCity string, gameDate time.Time, conditions *models.WeatherConditions) models.TravelWeatherImpact {
	log.Printf("✈️ Analyzing travel weather impact for %s vs %s", awayTeam, homeTeam)

	// Analyze home team travel (minimal - already in city)
	homeTravel := models.TravelAnalysis{
		TeamCode:          homeTeam,
		DepartureCity:     venueCity,
		ArrivalCity:       venueCity,
		FlightDelayRisk:   "None",
		DelayProbability:  0.0,
		TravelStress:      0.0,
		PreparationImpact: 0.0,
		RestImpact:        0.0,
		TotalTravelImpact: 0.0,
	}

	// Analyze away team travel
	awayTravel := w.analyzeAwayTeamTravel(awayTeam, venueCity, gameDate, conditions)

	// Calculate overall travel impact
	overallImpact := (homeTravel.TotalTravelImpact + awayTravel.TotalTravelImpact) / 2.0

	return models.TravelWeatherImpact{
		HomeTeamTravel: homeTravel,
		AwayTeamTravel: awayTravel,
		OverallImpact:  overallImpact,
	}
}

// analyzeAwayTeamTravel analyzes weather impact on away team's travel
func (w *WeatherAnalysisService) analyzeAwayTeamTravel(awayTeam, venueCity string, gameDate time.Time, conditions *models.WeatherConditions) models.TravelAnalysis {
	// Estimate departure city (would be improved with actual schedule data)
	departureCity := w.estimateDepartureCity(awayTeam)

	// Calculate flight delay risk based on weather conditions
	delayRisk, delayProb := w.calculateFlightDelayRisk(conditions, venueCity)

	// Estimate delay duration
	estimatedDelay := w.estimateDelayDuration(delayProb, conditions)

	// Calculate travel stress factors
	travelStress := w.calculateTravelStress(conditions, delayProb, estimatedDelay)

	// Calculate impact on game preparation
	prepImpact := w.calculatePreparationImpact(estimatedDelay, travelStress)

	// Calculate impact on player rest
	restImpact := w.calculateRestImpact(estimatedDelay, travelStress)

	// Total travel impact
	totalImpact := prepImpact + restImpact - (travelStress * 2.0)

	return models.TravelAnalysis{
		TeamCode:          awayTeam,
		DepartureCity:     departureCity,
		ArrivalCity:       venueCity,
		FlightDelayRisk:   delayRisk,
		DelayProbability:  delayProb,
		EstimatedDelay:    estimatedDelay,
		TravelStress:      travelStress,
		PreparationImpact: prepImpact,
		RestImpact:        restImpact,
		TotalTravelImpact: totalImpact,
	}
}

// estimateDepartureCity estimates where the away team is traveling from
func (w *WeatherAnalysisService) estimateDepartureCity(teamCode string) string {
	// Map team codes to their home cities
	teamCities := map[string]string{
		"ANA": "Anaheim", "BOS": "Boston", "BUF": "Buffalo", "CGY": "Calgary",
		"CAR": "Carolina", "CHI": "Chicago", "COL": "Colorado", "CBJ": "Columbus",
		"DAL": "Dallas", "DET": "Detroit", "EDM": "Edmonton", "FLA": "Florida",
		"LAK": "Los Angeles", "MIN": "Minnesota", "MTL": "Montreal", "NSH": "Nashville",
		"NJD": "New Jersey", "NYR": "New York", "NYI": "New York Islanders",
		"OTT": "Ottawa", "PHI": "Philadelphia", "PIT": "Pittsburgh", "SJS": "San Jose",
		"SEA": "Seattle", "STL": "St. Louis", "TBL": "Tampa Bay", "TOR": "Toronto",
		"UTA": "Utah", "VAN": "Vancouver", "VGK": "Vegas", "WSH": "Washington",
		"WPG": "Winnipeg",
	}

	if city, exists := teamCities[teamCode]; exists {
		return city
	}
	return "Unknown"
}

// calculateFlightDelayRisk determines flight delay risk based on weather
func (w *WeatherAnalysisService) calculateFlightDelayRisk(conditions *models.WeatherConditions, city string) (string, float64) {
	riskScore := 0.0

	// Temperature extremes
	if conditions.Temperature < 10 || conditions.Temperature > 100 {
		riskScore += 0.3
	} else if conditions.Temperature < 20 || conditions.Temperature > 90 {
		riskScore += 0.1
	}

	// Wind conditions
	if conditions.WindSpeed > 35 {
		riskScore += 0.4
	} else if conditions.WindSpeed > 25 {
		riskScore += 0.2
	} else if conditions.WindSpeed > 15 {
		riskScore += 0.1
	}

	// Precipitation
	if conditions.Precipitation > 0.5 {
		riskScore += 0.3
	} else if conditions.Precipitation > 0.1 {
		riskScore += 0.1
	}

	// Visibility
	if conditions.Visibility < 1.0 {
		riskScore += 0.4
	} else if conditions.Visibility < 3.0 {
		riskScore += 0.2
	}

	// Snow/ice conditions
	if conditions.PrecipType == "snow" && conditions.Temperature < 32 {
		riskScore += 0.3
	}

	// Cap at 1.0
	if riskScore > 1.0 {
		riskScore = 1.0
	}

	// Convert to risk categories
	var riskLevel string
	if riskScore < 0.2 {
		riskLevel = "Low"
	} else if riskScore < 0.5 {
		riskLevel = "Medium"
	} else if riskScore < 0.8 {
		riskLevel = "High"
	} else {
		riskLevel = "Severe"
	}

	return riskLevel, riskScore
}

// estimateDelayDuration estimates how long delays might last
func (w *WeatherAnalysisService) estimateDelayDuration(delayProb float64, conditions *models.WeatherConditions) time.Duration {
	if delayProb < 0.2 {
		return 0
	}

	baseDelay := delayProb * 180 // Up to 3 hours for severe conditions

	// Adjust based on specific conditions
	if conditions.PrecipType == "snow" {
		baseDelay *= 1.5 // Snow causes longer delays
	}

	if conditions.WindSpeed > 35 {
		baseDelay *= 1.3 // High winds extend delays
	}

	return time.Duration(baseDelay) * time.Minute
}

// calculateTravelStress calculates overall travel stress
func (w *WeatherAnalysisService) calculateTravelStress(conditions *models.WeatherConditions, delayProb float64, delay time.Duration) float64 {
	stress := delayProb * 0.5 // Base stress from delay probability

	// Add stress from delay duration
	if delay > time.Hour {
		stress += 0.3
	} else if delay > 30*time.Minute {
		stress += 0.2
	}

	// Weather-specific stress
	if conditions.PrecipType == "snow" || conditions.Temperature < 10 {
		stress += 0.2
	}

	// Cap at 1.0
	if stress > 1.0 {
		stress = 1.0
	}

	return stress
}

// calculatePreparationImpact calculates impact on game preparation
func (w *WeatherAnalysisService) calculatePreparationImpact(delay time.Duration, stress float64) float64 {
	impact := 0.0

	// Delay impact
	if delay > 2*time.Hour {
		impact -= 2.0
	} else if delay > time.Hour {
		impact -= 1.0
	} else if delay > 30*time.Minute {
		impact -= 0.5
	}

	// Stress impact
	impact -= stress * 1.5

	// Cap at -3.0
	if impact < -3.0 {
		impact = -3.0
	}

	return impact
}

// calculateRestImpact calculates impact on player rest
func (w *WeatherAnalysisService) calculateRestImpact(delay time.Duration, stress float64) float64 {
	impact := 0.0

	// Delay impact on rest
	if delay > 3*time.Hour {
		impact -= 1.5
	} else if delay > time.Hour {
		impact -= 1.0
	} else if delay > 30*time.Minute {
		impact -= 0.5
	}

	// Stress impact on rest quality
	impact -= stress * 1.0

	// Cap at -2.0
	if impact < -2.0 {
		impact = -2.0
	}

	return impact
}

// analyzeGameWeatherImpact analyzes how weather affects actual gameplay
func (w *WeatherAnalysisService) analyzeGameWeatherImpact(conditions *models.WeatherConditions, isOutdoor bool, outdoorInfo models.OutdoorGameInfo) models.GameWeatherImpact {
	if !isOutdoor {
		// Indoor games have minimal weather impact
		return models.GameWeatherImpact{
			HomeAdvantageChange: 0.0,
			GamePaceImpact:      0.0,
			ScoringImpact:       0.0,
			OverallGameImpact:   0.0,
		}
	}

	log.Printf("🏟️ Analyzing outdoor game weather impact")

	// Analyze skating conditions
	skatingImpact := w.analyzeSkatingImpact(conditions)

	// Analyze puck handling
	puckImpact := w.analyzePuckImpact(conditions)

	// Analyze shooting accuracy
	shootingImpact := w.analyzeShootingImpact(conditions)

	// Analyze goaltending impact
	goalieImpact := w.analyzeGoalieImpact(conditions)

	// Analyze visibility
	visibilityImpact := w.analyzeVisibilityImpact(conditions)

	// Analyze ice quality
	iceImpact := w.analyzeIceQualityImpact(conditions)

	// Analyze equipment impact
	equipmentImpact := w.analyzeEquipmentImpact(conditions)

	// Analyze fan comfort
	fanImpact := w.analyzeFanComfortImpact(conditions)

	// Calculate overall impacts
	homeAdvantageChange := fanImpact.HomeAdvantage
	gamePaceImpact := (skatingImpact.OverallImpact + puckImpact.OverallImpact) / 2.0
	scoringImpact := (shootingImpact.OverallImpact + goalieImpact.OverallImpact) / 2.0

	overallImpact := (homeAdvantageChange + gamePaceImpact + scoringImpact) / 3.0

	return models.GameWeatherImpact{
		SkatingConditions:   skatingImpact,
		PuckHandling:        puckImpact,
		ShootingAccuracy:    shootingImpact,
		GoaltendingImpact:   goalieImpact,
		VisibilityImpact:    visibilityImpact,
		IceQuality:          iceImpact,
		EquipmentImpact:     equipmentImpact,
		FanComfort:          fanImpact,
		HomeAdvantageChange: homeAdvantageChange,
		GamePaceImpact:      gamePaceImpact,
		ScoringImpact:       scoringImpact,
		OverallGameImpact:   overallImpact,
	}
}

// analyzeSkatingImpact analyzes weather impact on skating
func (w *WeatherAnalysisService) analyzeSkatingImpact(conditions *models.WeatherConditions) models.SkatingImpact {
	impact := 0.0
	speedReduction := 0.0
	agilityImpact := 0.0
	enduranceImpact := 0.0
	injuryRisk := 0.0
	description := "Normal skating conditions"

	// Temperature effects
	if conditions.Temperature < 0 {
		impact -= 1.5
		speedReduction += 0.15
		agilityImpact -= 0.8
		enduranceImpact -= 0.6
		injuryRisk += 0.3
		description = "Extremely cold conditions significantly impact skating"
	} else if conditions.Temperature < 20 {
		impact -= 1.0
		speedReduction += 0.10
		agilityImpact -= 0.5
		enduranceImpact -= 0.4
		injuryRisk += 0.2
		description = "Cold conditions moderately impact skating"
	} else if conditions.Temperature > 50 {
		impact -= 0.5
		speedReduction += 0.05
		enduranceImpact -= 0.3
		description = "Warmer conditions may affect ice quality"
	}

	// Wind effects
	if conditions.WindSpeed > 25 {
		impact -= 0.8
		agilityImpact -= 0.4
		description += "; strong winds affect balance and control"
	} else if conditions.WindSpeed > 15 {
		impact -= 0.4
		agilityImpact -= 0.2
		description += "; moderate winds affect player control"
	}

	return models.SkatingImpact{
		SpeedReduction:  speedReduction,
		AgilityImpact:   agilityImpact,
		EnduranceImpact: enduranceImpact,
		InjuryRisk:      injuryRisk,
		OverallImpact:   impact,
		Description:     description,
	}
}

// analyzePuckImpact analyzes weather impact on puck handling
func (w *WeatherAnalysisService) analyzePuckImpact(conditions *models.WeatherConditions) models.PuckImpact {
	impact := 0.0
	bounceUnpredictability := 0.0
	passingAccuracy := 0.0
	stickHandling := 0.0
	puckSpeed := 0.0
	description := "Normal puck conditions"

	// Temperature effects on puck bounce
	if conditions.Temperature < 10 {
		bounceUnpredictability += 0.4
		passingAccuracy -= 0.6
		stickHandling -= 0.5
		impact -= 1.2
		description = "Cold conditions make puck handling difficult"
	} else if conditions.Temperature < 25 {
		bounceUnpredictability += 0.2
		passingAccuracy -= 0.3
		stickHandling -= 0.2
		impact -= 0.6
		description = "Cool conditions slightly affect puck handling"
	}

	// Wind effects on puck trajectory
	if conditions.WindSpeed > 20 {
		passingAccuracy -= 0.8
		puckSpeed -= 0.4
		impact -= 1.0
		description += "; wind significantly affects puck trajectory"
	} else if conditions.WindSpeed > 10 {
		passingAccuracy -= 0.4
		puckSpeed -= 0.2
		impact -= 0.5
		description += "; wind moderately affects puck movement"
	}

	// Precipitation effects
	if conditions.Precipitation > 0.1 {
		bounceUnpredictability += 0.3
		stickHandling -= 0.4
		impact -= 0.7
		description += "; precipitation affects puck control"
	}

	return models.PuckImpact{
		BounceUnpredictability: bounceUnpredictability,
		PassingAccuracy:        passingAccuracy,
		StickHandling:          stickHandling,
		PuckSpeed:              puckSpeed,
		OverallImpact:          impact,
		Description:            description,
	}
}

// analyzeShootingImpact analyzes weather impact on shooting
func (w *WeatherAnalysisService) analyzeShootingImpact(conditions *models.WeatherConditions) models.ShootingImpact {
	impact := 0.0
	accuracyReduction := 0.0
	powerReduction := 0.0
	trajectoryEffect := 0.0
	selectionImpact := 0.0
	description := "Normal shooting conditions"

	// Wind effects on shot accuracy and trajectory
	if conditions.WindSpeed > 25 {
		accuracyReduction += 0.25
		trajectoryEffect -= 0.8
		selectionImpact -= 0.6
		impact -= 1.5
		description = "Strong winds severely impact shooting accuracy"
	} else if conditions.WindSpeed > 15 {
		accuracyReduction += 0.15
		trajectoryEffect -= 0.5
		selectionImpact -= 0.3
		impact -= 1.0
		description = "Moderate winds affect shooting accuracy"
	} else if conditions.WindSpeed > 10 {
		accuracyReduction += 0.08
		trajectoryEffect -= 0.2
		impact -= 0.5
		description = "Light winds slightly affect shots"
	}

	// Temperature effects on shot power
	if conditions.Temperature < 10 {
		powerReduction += 0.15
		accuracyReduction += 0.10
		impact -= 0.8
		description += "; cold conditions reduce shot power"
	}

	return models.ShootingImpact{
		AccuracyReduction: accuracyReduction,
		PowerReduction:    powerReduction,
		PuckTrajectory:    trajectoryEffect,
		ShotSelection:     selectionImpact,
		OverallImpact:     impact,
		Description:       description,
	}
}

// analyzeGoalieImpact analyzes weather impact on goaltending
func (w *WeatherAnalysisService) analyzeGoalieImpact(conditions *models.WeatherConditions) models.GoalieImpact {
	impact := 0.0
	visionImpairment := 0.0
	reactionTime := 0.0
	equipmentFunction := 0.0
	positioningImpact := 0.0
	trackingAbility := 0.0
	description := "Normal goaltending conditions"

	// Visibility effects
	if conditions.Visibility < 2.0 {
		visionImpairment += 0.6
		trackingAbility -= 0.8
		impact -= 1.5
		description = "Poor visibility severely impacts goaltending"
	} else if conditions.Visibility < 5.0 {
		visionImpairment += 0.3
		trackingAbility -= 0.4
		impact -= 0.8
		description = "Reduced visibility affects goaltending"
	}

	// Precipitation effects on vision
	if conditions.Precipitation > 0.2 {
		visionImpairment += 0.4
		trackingAbility -= 0.6
		impact -= 1.0
		description += "; precipitation impairs vision"
	} else if conditions.Precipitation > 0.05 {
		visionImpairment += 0.2
		trackingAbility -= 0.3
		impact -= 0.5
		description += "; light precipitation affects visibility"
	}

	// Wind effects on positioning
	if conditions.WindSpeed > 20 {
		positioningImpact -= 0.5
		reactionTime -= 0.3
		impact -= 0.6
		description += "; wind affects goalie positioning"
	}

	// Temperature effects on equipment
	if conditions.Temperature < 15 {
		equipmentFunction -= 0.4
		reactionTime -= 0.2
		impact -= 0.4
		description += "; cold affects equipment flexibility"
	}

	return models.GoalieImpact{
		VisionImpairment:  visionImpairment,
		ReactionTime:      reactionTime,
		EquipmentFunction: equipmentFunction,
		PositioningImpact: positioningImpact,
		TrackingAbility:   trackingAbility,
		OverallImpact:     impact,
		Description:       description,
	}
}

// analyzeVisibilityImpact analyzes weather impact on visibility
func (w *WeatherAnalysisService) analyzeVisibilityImpact(conditions *models.WeatherConditions) models.VisibilityImpact {
	impact := 0.0
	playerVisibility := 0.0
	puckVisibility := 0.0
	depthPerception := 0.0
	lightingImpact := 0.0
	description := "Good visibility conditions"

	// Base visibility reduction
	if conditions.Visibility < 1.0 {
		playerVisibility += 0.7
		puckVisibility += 0.8
		depthPerception -= 0.8
		impact -= 2.0
		description = "Severely reduced visibility"
	} else if conditions.Visibility < 3.0 {
		playerVisibility += 0.4
		puckVisibility += 0.5
		depthPerception -= 0.5
		impact -= 1.2
		description = "Moderately reduced visibility"
	} else if conditions.Visibility < 6.0 {
		playerVisibility += 0.2
		puckVisibility += 0.3
		depthPerception -= 0.2
		impact -= 0.6
		description = "Slightly reduced visibility"
	}

	// Cloud cover effects on lighting
	if conditions.CloudCover > 90 {
		lightingImpact -= 0.4
		impact -= 0.3
		description += "; overcast conditions affect lighting"
	}

	return models.VisibilityImpact{
		PlayerVisibility: playerVisibility,
		PuckVisibility:   puckVisibility,
		DepthPerception:  depthPerception,
		LightingImpact:   lightingImpact,
		OverallImpact:    impact,
		Description:      description,
	}
}

// analyzeIceQualityImpact analyzes weather impact on ice conditions
func (w *WeatherAnalysisService) analyzeIceQualityImpact(conditions *models.WeatherConditions) models.IceQualityImpact {
	impact := 0.0
	surfaceHardness := 0.0
	surfaceSmoothness := 0.0
	iceTemp := 22.0 // Ideal ice temperature
	meltingRate := 0.0
	chipBuildup := 0.0
	description := "Good ice conditions"

	// Temperature effects on ice quality
	if conditions.Temperature > 45 {
		surfaceHardness -= 0.8
		surfaceSmoothness -= 0.6
		meltingRate += 0.6
		iceTemp += 8
		impact -= 1.5
		description = "Warm conditions significantly affect ice quality"
	} else if conditions.Temperature > 35 {
		surfaceHardness -= 0.5
		surfaceSmoothness -= 0.3
		meltingRate += 0.3
		iceTemp += 4
		impact -= 1.0
		description = "Moderate temperatures affect ice quality"
	} else if conditions.Temperature < 10 {
		surfaceHardness += 0.3
		chipBuildup += 0.4
		iceTemp -= 5
		impact -= 0.5
		description = "Very cold conditions make ice brittle"
	}

	// Wind effects on ice surface
	if conditions.WindSpeed > 20 {
		chipBuildup += 0.3
		surfaceSmoothness -= 0.2
		impact -= 0.4
		description += "; wind causes ice debris buildup"
	}

	return models.IceQualityImpact{
		SurfaceHardness:   surfaceHardness,
		SurfaceSmoothness: surfaceSmoothness,
		IceTemperature:    iceTemp,
		MeltingRate:       meltingRate,
		ChipBuildup:       chipBuildup,
		OverallImpact:     impact,
		Description:       description,
	}
}

// analyzeEquipmentImpact analyzes weather impact on equipment
func (w *WeatherAnalysisService) analyzeEquipmentImpact(conditions *models.WeatherConditions) models.EquipmentImpact {
	impact := 0.0
	skatePerformance := 0.0
	stickHandling := 0.0
	gearComfort := 0.0
	durability := 0.0
	description := "Normal equipment performance"

	// Temperature effects on equipment
	if conditions.Temperature < 10 {
		skatePerformance -= 0.4
		stickHandling -= 0.3
		gearComfort -= 0.6
		impact -= 0.8
		description = "Cold conditions affect equipment flexibility"
	} else if conditions.Temperature > 50 {
		gearComfort -= 0.8
		durability -= 0.2
		impact -= 0.6
		description = "Warm conditions affect player comfort"
	}

	// Precipitation effects
	if conditions.Precipitation > 0.1 {
		skatePerformance -= 0.2
		durability -= 0.3
		impact -= 0.3
		description += "; moisture affects equipment"
	}

	return models.EquipmentImpact{
		SkatePerformance:    skatePerformance,
		StickHandling:       stickHandling,
		GearComfort:         gearComfort,
		EquipmentDurability: durability,
		OverallImpact:       impact,
		Description:         description,
	}
}

// analyzeFanComfortImpact analyzes weather impact on fan attendance and energy
func (w *WeatherAnalysisService) analyzeFanComfortImpact(conditions *models.WeatherConditions) models.FanComfortImpact {
	impact := 0.0
	attendanceImpact := 0.0
	energyLevel := 0.0
	noiseLevel := 0.0
	homeAdvantage := 0.0
	description := "Good fan conditions"

	// Temperature effects on fan comfort
	if conditions.Temperature < 20 {
		attendanceImpact -= 0.3
		energyLevel -= 0.4
		noiseLevel -= 0.3
		homeAdvantage -= 0.4
		impact -= 0.8
		description = "Cold conditions reduce fan comfort"
	} else if conditions.Temperature > 80 {
		attendanceImpact -= 0.2
		energyLevel -= 0.3
		homeAdvantage -= 0.2
		impact -= 0.5
		description = "Hot conditions affect fan comfort"
	} else if conditions.Temperature >= 30 && conditions.Temperature <= 50 {
		energyLevel += 0.2
		noiseLevel += 0.2
		homeAdvantage += 0.3
		impact += 0.4
		description = "Ideal conditions enhance fan experience"
	}

	// Precipitation effects
	if conditions.Precipitation > 0.2 {
		attendanceImpact -= 0.5
		energyLevel -= 0.6
		noiseLevel -= 0.4
		homeAdvantage -= 0.5
		impact -= 1.0
		description += "; precipitation significantly impacts fans"
	} else if conditions.Precipitation > 0.05 {
		attendanceImpact -= 0.2
		energyLevel -= 0.3
		homeAdvantage -= 0.2
		impact -= 0.4
		description += "; light precipitation affects fans"
	}

	// Wind effects
	if conditions.WindSpeed > 25 {
		energyLevel -= 0.4
		noiseLevel -= 0.3
		homeAdvantage -= 0.3
		impact -= 0.6
		description += "; strong winds affect fan comfort"
	}

	return models.FanComfortImpact{
		AttendanceImpact: attendanceImpact,
		FanEnergyLevel:   energyLevel,
		NoiseLevel:       noiseLevel,
		HomeAdvantage:    homeAdvantage,
		OverallImpact:    impact,
		Description:      description,
	}
}

// calculateOverallWeatherImpact calculates the overall weather impact on the game
func (w *WeatherAnalysisService) calculateOverallWeatherImpact(travelImpact models.TravelWeatherImpact, gameImpact models.GameWeatherImpact, isOutdoor bool) float64 {
	if !isOutdoor {
		// Indoor games - only travel impact matters
		return travelImpact.OverallImpact * 0.3 // Reduced weight for indoor games
	}

	// Outdoor games - both travel and game impacts matter
	travelWeight := 0.3
	gameWeight := 0.7

	overallImpact := (travelImpact.OverallImpact * travelWeight) + (gameImpact.OverallGameImpact * gameWeight)

	// Cap the overall impact
	if overallImpact > 10.0 {
		overallImpact = 10.0
	} else if overallImpact < -10.0 {
		overallImpact = -10.0
	}

	return overallImpact
}
