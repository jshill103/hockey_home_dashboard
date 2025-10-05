package models

import "time"

// WeatherAnalysis represents comprehensive weather impact analysis for a game
type WeatherAnalysis struct {
	GameID            string              `json:"gameId"`
	HomeTeam          string              `json:"homeTeam"`
	AwayTeam          string              `json:"awayTeam"`
	GameDate          time.Time           `json:"gameDate"`
	VenueName         string              `json:"venueName"`
	VenueCity         string              `json:"venueCity"`
	IsOutdoorGame     bool                `json:"isOutdoorGame"`
	WeatherConditions WeatherConditions   `json:"weatherConditions"`
	TravelImpact      TravelWeatherImpact `json:"travelImpact"`
	GameImpact        GameWeatherImpact   `json:"gameImpact"`
	OverallImpact     float64             `json:"overallImpact"` // -10.0 to +10.0 prediction adjustment
	Confidence        float64             `json:"confidence"`    // 0.0 to 1.0 confidence in weather data
	LastUpdated       time.Time           `json:"lastUpdated"`
}

// WeatherConditions represents current and forecasted weather conditions
type WeatherConditions struct {
	// Current Conditions
	Temperature   float64 `json:"temperature"`   // Fahrenheit
	FeelsLike     float64 `json:"feelsLike"`     // Wind chill/heat index
	Humidity      float64 `json:"humidity"`      // Percentage
	WindSpeed     float64 `json:"windSpeed"`     // MPH
	WindDirection int     `json:"windDirection"` // Degrees (0-360)
	WindGust      float64 `json:"windGust"`      // MPH
	Precipitation float64 `json:"precipitation"` // Inches
	PrecipType    string  `json:"precipType"`    // "rain", "snow", "sleet", "none"
	Visibility    float64 `json:"visibility"`    // Miles
	CloudCover    float64 `json:"cloudCover"`    // Percentage
	UVIndex       float64 `json:"uvIndex"`       // 0-11 scale
	AirQuality    int     `json:"airQuality"`    // AQI (0-500)

	// Forecast (3-hour intervals for next 12 hours)
	HourlyForecast []HourlyWeather `json:"hourlyForecast"`

	// Weather Alerts
	Alerts []WeatherAlert `json:"alerts"`

	// Data Quality
	DataSource  string        `json:"dataSource"`  // "OpenWeatherMap", "WeatherAPI", etc.
	DataAge     time.Duration `json:"dataAge"`     // How old is this data
	Reliability float64       `json:"reliability"` // 0.0 to 1.0 data reliability score
}

// HourlyWeather represents weather conditions for a specific hour
type HourlyWeather struct {
	Time          time.Time `json:"time"`
	Temperature   float64   `json:"temperature"`
	FeelsLike     float64   `json:"feelsLike"`
	WindSpeed     float64   `json:"windSpeed"`
	WindGust      float64   `json:"windGust"`
	Precipitation float64   `json:"precipitation"`
	PrecipType    string    `json:"precipType"`
	Visibility    float64   `json:"visibility"`
	CloudCover    float64   `json:"cloudCover"`
}

// WeatherAlert represents weather warnings or advisories
type WeatherAlert struct {
	Type        string    `json:"type"`     // "Winter Storm", "High Wind", etc.
	Severity    string    `json:"severity"` // "Minor", "Moderate", "Severe", "Extreme"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Impact      float64   `json:"impact"` // -5.0 to +5.0 game impact score
}

// TravelWeatherImpact analyzes weather impact on team travel
type TravelWeatherImpact struct {
	HomeTeamTravel TravelAnalysis `json:"homeTeamTravel"`
	AwayTeamTravel TravelAnalysis `json:"awayTeamTravel"`
	OverallImpact  float64        `json:"overallImpact"` // -5.0 to +5.0
}

// TravelAnalysis represents weather impact on a team's travel
type TravelAnalysis struct {
	TeamCode          string            `json:"teamCode"`
	DepartureCity     string            `json:"departureCity"`
	DepartureWeather  WeatherConditions `json:"departureWeather"`
	ArrivalCity       string            `json:"arrivalCity"`
	ArrivalWeather    WeatherConditions `json:"arrivalWeather"`
	FlightDelayRisk   string            `json:"flightDelayRisk"`   // "Low", "Medium", "High", "Severe"
	DelayProbability  float64           `json:"delayProbability"`  // 0.0 to 1.0
	EstimatedDelay    time.Duration     `json:"estimatedDelay"`    // Expected delay duration
	TravelStress      float64           `json:"travelStress"`      // 0.0 to 1.0 travel stress factor
	PreparationImpact float64           `json:"preparationImpact"` // -3.0 to +3.0 impact on game prep
	RestImpact        float64           `json:"restImpact"`        // -2.0 to +2.0 impact on player rest
	TotalTravelImpact float64           `json:"totalTravelImpact"` // -5.0 to +5.0 overall travel impact
}

// GameWeatherImpact analyzes weather impact on actual gameplay
type GameWeatherImpact struct {
	// Performance Impacts
	SkatingConditions SkatingImpact    `json:"skatingConditions"`
	PuckHandling      PuckImpact       `json:"puckHandling"`
	ShootingAccuracy  ShootingImpact   `json:"shootingAccuracy"`
	GoaltendingImpact GoalieImpact     `json:"goaltendingImpact"`
	VisibilityImpact  VisibilityImpact `json:"visibilityImpact"`

	// Environmental Factors
	IceQuality      IceQualityImpact `json:"iceQuality"`
	EquipmentImpact EquipmentImpact  `json:"equipmentImpact"`
	FanComfort      FanComfortImpact `json:"fanComfort"`

	// Overall Game Impact
	HomeAdvantageChange float64 `json:"homeAdvantageChange"` // -2.0 to +2.0
	GamePaceImpact      float64 `json:"gamePaceImpact"`      // -1.0 to +1.0
	ScoringImpact       float64 `json:"scoringImpact"`       // -2.0 to +2.0
	OverallGameImpact   float64 `json:"overallGameImpact"`   // -5.0 to +5.0
}

// SkatingImpact represents how weather affects player skating
type SkatingImpact struct {
	SpeedReduction  float64 `json:"speedReduction"`  // 0.0 to 1.0 (percentage reduction)
	AgilityImpact   float64 `json:"agilityImpact"`   // -1.0 to +1.0
	EnduranceImpact float64 `json:"enduranceImpact"` // -1.0 to +1.0
	InjuryRisk      float64 `json:"injuryRisk"`      // 0.0 to 1.0 (increased risk)
	OverallImpact   float64 `json:"overallImpact"`   // -2.0 to +2.0
	Description     string  `json:"description"`     // Human-readable explanation
}

// PuckImpact represents how weather affects puck handling
type PuckImpact struct {
	BounceUnpredictability float64 `json:"bounceUnpredictability"` // 0.0 to 1.0
	PassingAccuracy        float64 `json:"passingAccuracy"`        // -1.0 to +1.0
	StickHandling          float64 `json:"stickHandling"`          // -1.0 to +1.0
	PuckSpeed              float64 `json:"puckSpeed"`              // -1.0 to +1.0
	OverallImpact          float64 `json:"overallImpact"`          // -2.0 to +2.0
	Description            string  `json:"description"`
}

// ShootingImpact represents how weather affects shooting
type ShootingImpact struct {
	AccuracyReduction float64 `json:"accuracyReduction"` // 0.0 to 1.0 (percentage reduction)
	PowerReduction    float64 `json:"powerReduction"`    // 0.0 to 1.0 (percentage reduction)
	PuckTrajectory    float64 `json:"puckTrajectory"`    // -1.0 to +1.0 (wind effect)
	ShotSelection     float64 `json:"shotSelection"`     // -1.0 to +1.0 (strategy change)
	OverallImpact     float64 `json:"overallImpact"`     // -2.0 to +2.0
	Description       string  `json:"description"`
}

// GoalieImpact represents how weather affects goaltending
type GoalieImpact struct {
	VisionImpairment  float64 `json:"visionImpairment"`  // 0.0 to 1.0
	ReactionTime      float64 `json:"reactionTime"`      // -1.0 to +1.0
	EquipmentFunction float64 `json:"equipmentFunction"` // -1.0 to +1.0
	PositioningImpact float64 `json:"positioningImpact"` // -1.0 to +1.0
	TrackingAbility   float64 `json:"trackingAbility"`   // -1.0 to +1.0
	OverallImpact     float64 `json:"overallImpact"`     // -2.0 to +2.0
	Description       string  `json:"description"`
}

// VisibilityImpact represents how weather affects visibility
type VisibilityImpact struct {
	PlayerVisibility float64 `json:"playerVisibility"` // 0.0 to 1.0 (reduction)
	PuckVisibility   float64 `json:"puckVisibility"`   // 0.0 to 1.0 (reduction)
	DepthPerception  float64 `json:"depthPerception"`  // -1.0 to +1.0
	LightingImpact   float64 `json:"lightingImpact"`   // -1.0 to +1.0
	OverallImpact    float64 `json:"overallImpact"`    // -2.0 to +2.0
	Description      string  `json:"description"`
}

// IceQualityImpact represents how weather affects ice conditions
type IceQualityImpact struct {
	SurfaceHardness   float64 `json:"surfaceHardness"`   // -1.0 to +1.0
	SurfaceSmoothness float64 `json:"surfaceSmoothness"` // -1.0 to +1.0
	IceTemperature    float64 `json:"iceTemperature"`    // Actual ice temperature
	MeltingRate       float64 `json:"meltingRate"`       // 0.0 to 1.0 (increased melting)
	ChipBuildup       float64 `json:"chipBuildup"`       // 0.0 to 1.0 (ice chip accumulation)
	OverallImpact     float64 `json:"overallImpact"`     // -2.0 to +2.0
	Description       string  `json:"description"`
}

// EquipmentImpact represents how weather affects equipment performance
type EquipmentImpact struct {
	SkatePerformance    float64 `json:"skatePerformance"`    // -1.0 to +1.0
	StickHandling       float64 `json:"stickHandling"`       // -1.0 to +1.0
	GearComfort         float64 `json:"gearComfort"`         // -1.0 to +1.0
	EquipmentDurability float64 `json:"equipmentDurability"` // -1.0 to +1.0
	OverallImpact       float64 `json:"overallImpact"`       // -2.0 to +2.0
	Description         string  `json:"description"`
}

// FanComfortImpact represents how weather affects fan attendance and energy
type FanComfortImpact struct {
	AttendanceImpact float64 `json:"attendanceImpact"` // -1.0 to +1.0
	FanEnergyLevel   float64 `json:"fanEnergyLevel"`   // -1.0 to +1.0
	NoiseLevel       float64 `json:"noiseLevel"`       // -1.0 to +1.0
	HomeAdvantage    float64 `json:"homeAdvantage"`    // -1.0 to +1.0
	OverallImpact    float64 `json:"overallImpact"`    // -2.0 to +2.0
	Description      string  `json:"description"`
}

// OutdoorGameInfo represents special considerations for outdoor games
type OutdoorGameInfo struct {
	IsOutdoorGame         bool               `json:"isOutdoorGame"`
	GameType              string             `json:"gameType"`              // "Winter Classic", "Stadium Series", etc.
	VenueType             string             `json:"venueType"`             // "Football Stadium", "Baseball Stadium", etc.
	SpecialConsiderations []string           `json:"specialConsiderations"` // Additional factors to consider
	HistoricalData        OutdoorGameHistory `json:"historicalData"`
}

// OutdoorGameHistory represents historical data for outdoor games
type OutdoorGameHistory struct {
	SimilarConditionsGames []HistoricalOutdoorGame `json:"similarConditionsGames"`
	AverageScoring         float64                 `json:"averageScoring"`
	AverageGameTime        time.Duration           `json:"averageGameTime"`
	CommonIssues           []string                `json:"commonIssues"`
	SuccessFactors         []string                `json:"successFactors"`
}

// HistoricalOutdoorGame represents data from a previous outdoor game
type HistoricalOutdoorGame struct {
	Date          time.Time     `json:"date"`
	Teams         []string      `json:"teams"`
	Temperature   float64       `json:"temperature"`
	WindSpeed     float64       `json:"windSpeed"`
	Precipitation float64       `json:"precipitation"`
	FinalScore    string        `json:"finalScore"`
	TotalGoals    int           `json:"totalGoals"`
	GameDuration  time.Duration `json:"gameDuration"`
	WeatherIssues []string      `json:"weatherIssues"`
	Attendance    int           `json:"attendance"`
}

// WeatherDataSource represents configuration for weather data APIs
type WeatherDataSource struct {
	Name           string   `json:"name"`
	APIKey         string   `json:"apiKey"`
	BaseURL        string   `json:"baseURL"`
	RateLimit      int      `json:"rateLimit"`      // Requests per hour
	Reliability    float64  `json:"reliability"`    // 0.0 to 1.0
	CostPerRequest float64  `json:"costPerRequest"` // USD
	Features       []string `json:"features"`       // Supported features
	IsEnabled      bool     `json:"isEnabled"`
	Priority       int      `json:"priority"` // 1 = highest priority
}

