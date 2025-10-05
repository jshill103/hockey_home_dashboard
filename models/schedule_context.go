package models

import "time"

// ScheduleContext represents comprehensive schedule situation for a team
type ScheduleContext struct {
	TeamCode string    `json:"teamCode"`
	GameDate time.Time `json:"gameDate"`

	// Travel
	TravelDistance      float64 `json:"travelDistance"`      // Miles traveled for this game
	TravelDistanceLast7 float64 `json:"travelDistanceLast7"` // Total miles last 7 days
	TimeZoneChanges     int     `json:"timeZoneChanges"`     // # of TZ changes last 7 days
	CrossCountryTrip    bool    `json:"crossCountryTrip"`    // Coast-to-coast travel
	TravelFatigueScore  float64 `json:"travelFatigueScore"`  // 0.0 (fresh) to 1.0 (exhausted)

	// Rest & Schedule Density
	DaysSinceLastGame    int     `json:"daysSinceLastGame"`
	GamesInLast7Days     int     `json:"gamesInLast7Days"`
	GamesInLast14Days    int     `json:"gamesInLast14Days"`
	IsBackToBack         bool    `json:"isBackToBack"`
	BackToBackCount      int     `json:"backToBackCount"`      // # of B2B in last 14 days
	ScheduleDensityScore float64 `json:"scheduleDensityScore"` // 0.0 (light) to 1.0 (heavy)

	// Rest Advantage
	RestDays           int     `json:"restDays"`
	OpponentRestDays   int     `json:"opponentRestDays"`
	RestAdvantage      int     `json:"restAdvantage"`      // Difference in rest days
	RestAdvantageScore float64 `json:"restAdvantageScore"` // -1.0 to +1.0

	// Home/Road Context
	IsHome             bool `json:"isHome"`
	GamesIntoRoadTrip  int  `json:"gamesIntoRoadTrip"`  // Which game of road trip (0 = home)
	TotalRoadTripGames int  `json:"totalRoadTripGames"` // Length of current road trip
	FirstGameBackHome  bool `json:"firstGameBackHome"`  // Just returned home
	EndOfRoadTrip      bool `json:"endOfRoadTrip"`      // Last game of road trip

	// Looking Ahead/Behind
	NextGameDate        time.Time `json:"nextGameDate"`
	DaysUntilNextGame   int       `json:"daysUntilNextGame"`
	PreviousOpponent    string    `json:"previousOpponent"`
	PreviousOpponentStr string    `json:"previousOpponentStr"` // "strong", "weak", "medium"
	NextOpponent        string    `json:"nextOpponent"`
	NextOpponentStr     string    `json:"nextOpponentStr"`

	// Trap Game Indicators
	IsTrapGame     bool    `json:"isTrapGame"`
	TrapGameScore  float64 `json:"trapGameScore"` // 0.0-1.0 likelihood
	TrapGameReason string  `json:"trapGameReason"`

	// Playoff Context
	InPlayoffRace       bool    `json:"inPlayoffRace"`
	PointsBehindPlayoff int     `json:"pointsBehindPlayoff"`
	PointsAheadPlayoff  int     `json:"pointsAheadPlayoff"`
	PlayoffImportance   float64 `json:"playoffImportance"` // 0.0 (eliminated) to 1.0 (must-win)
	DesperationFactor   float64 `json:"desperationFactor"` // 0.0-1.0

	// Special Circumstances
	SeasonOpener bool `json:"seasonOpener"`
	HomeOpener   bool `json:"homeOpener"`
	SpecialEvent bool `json:"specialEvent"` // Stadium Series, Winter Classic, etc.
	RivalryGame  bool `json:"rivalryGame"`

	LastUpdated time.Time `json:"lastUpdated"`
}

// ScheduleComparison compares schedule situations for both teams
type ScheduleComparison struct {
	HomeContext *ScheduleContext `json:"homeContext"`
	AwayContext *ScheduleContext `json:"awayContext"`

	// Advantage Breakdown
	TravelAdvantage   string `json:"travelAdvantage"` // "home", "away", "even"
	RestAdvantage     string `json:"restAdvantage"`
	ScheduleAdvantage string `json:"scheduleAdvantage"` // Who has easier recent schedule
	OverallAdvantage  string `json:"overallAdvantage"`

	// Impact Scores
	TravelImpact   float64 `json:"travelImpact"` // -1.0 to +1.0
	RestImpact     float64 `json:"restImpact"`
	ScheduleImpact float64 `json:"scheduleImpact"`
	TrapGameImpact float64 `json:"trapGameImpact"`
	PlayoffImpact  float64 `json:"playoffImpact"`

	// Combined Impact
	TotalImpact float64 `json:"totalImpact"` // -0.10 to +0.10 (win % adjustment)
	Confidence  float64 `json:"confidence"`  // 0.0-1.0

	LastUpdated time.Time `json:"lastUpdated"`
}

// CityCoordinates represents a city's location
type CityCoordinates struct {
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	TimeZone  string  `json:"timeZone"`
}

// TravelSegment represents a single trip
type TravelSegment struct {
	FromCity       string    `json:"fromCity"`
	ToCity         string    `json:"toCity"`
	Distance       float64   `json:"distance"` // Miles
	Date           time.Time `json:"date"`
	TimeZoneChange int       `json:"timeZoneChange"` // Hours
}

// RoadTripInfo represents a road trip
type RoadTripInfo struct {
	StartDate     time.Time `json:"startDate"`
	EndDate       time.Time `json:"endDate"`
	NumGames      int       `json:"numGames"`
	Cities        []string  `json:"cities"`
	TotalDistance float64   `json:"totalDistance"`
	TimeZonesSpan int       `json:"timeZonesSpan"`
	IsActive      bool      `json:"isActive"`
}
