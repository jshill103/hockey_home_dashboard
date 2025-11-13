package models

import "time"

// PlaystyleProfile represents a team's playing style
type PlaystyleProfile struct {
	TeamCode         string    `json:"teamCode"`
	Season           string    `json:"season"`
	PrimaryStyle     string    `json:"primaryStyle"`     // "Offensive", "Defensive", "Possession", "Speed", "Physical", "Skilled"
	SecondaryStyle   string    `json:"secondaryStyle"`   // Secondary characteristic
	StyleConfidence  float64   `json:"styleConfidence"`  // How confident in classification (0-1)
	
	// Style Metrics (0-1 scale)
	OffensiveRating  float64   `json:"offensiveRating"`  // How offensive-minded
	DefensiveRating  float64   `json:"defensiveRating"`  // How defensive-minded
	PossessionRating float64   `json:"possessionRating"` // Puck control emphasis
	SpeedRating      float64   `json:"speedRating"`      // Transition game speed
	PhysicalRating   float64   `json:"physicalRating"`   // Physicality level
	SkillRating      float64   `json:"skillRating"`      // Finesse/skill emphasis
	
	// Supporting Stats
	AvgGoalsFor      float64   `json:"avgGoalsFor"`
	AvgGoalsAgainst  float64   `json:"avgGoalsAgainst"`
	AvgShots         float64   `json:"avgShots"`
	AvgHits          float64   `json:"avgHits"`
	PossessionTime   float64   `json:"possessionTime"`   // % of game with puck
	
	LastUpdated      time.Time `json:"lastUpdated"`
}

// PlaystyleMatchup analyzes compatibility between two team styles
type PlaystyleMatchup struct {
	HomeTeam         string    `json:"homeTeam"`
	AwayTeam         string    `json:"awayTeam"`
	HomeStyle        string    `json:"homeStyle"`
	AwayStyle        string    `json:"awayStyle"`
	Compatibility    float64   `json:"compatibility"`    // How styles match up (-1 to +1, positive favors home)
	Advantage        string    `json:"advantage"`        // "Home", "Away", "Neutral"
	ImpactFactor     float64   `json:"impactFactor"`     // Prediction adjustment (-0.10 to +0.10)
	Confidence       float64   `json:"confidence"`       // Confidence in matchup analysis (0-1)
	Explanation      string    `json:"explanation"`      // Why this matchup favors one team
	KeyFactors       []string  `json:"keyFactors"`       // Factors driving the matchup
	HistoricalBasis  bool      `json:"historicalBasis"`  // Based on historical data?
	LastUpdated      time.Time `json:"lastUpdated"`
}

// PlaystyleCompatibility defines how different styles match up
type PlaystyleCompatibility struct {
	Style1           string  `json:"style1"`
	Style2           string  `json:"style2"`
	BaseCompatibility float64 `json:"baseCompatibility"` // -1 to +1 (positive = Style1 advantage)
	Description      string  `json:"description"`
	Reasoning        string  `json:"reasoning"`
}

// StyleCategory defines characteristics of a playstyle
type StyleCategory struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Strengths        []string `json:"strengths"`
	Weaknesses       []string `json:"weaknesses"`
	TypicalStats     StyleStats `json:"typicalStats"`
	CounterStyles    []string `json:"counterStyles"`    // Styles this is weak against
	FavorableStyles  []string `json:"favorableStyles"`  // Styles this is strong against
}

// StyleStats provides typical statistical ranges for a style
type StyleStats struct {
	GoalsForRange    [2]float64 `json:"goalsForRange"`    // Min, Max per game
	GoalsAgainstRange [2]float64 `json:"goalsAgainstRange"`
	ShotsForRange    [2]float64 `json:"shotsForRange"`
	HitsRange        [2]float64 `json:"hitsRange"`
	PossessionRange  [2]float64 `json:"possessionRange"`  // % range
}

// MatchupHistoryPattern tracks patterns in specific matchups
type MatchupHistoryPattern struct {
	HomeTeam          string    `json:"homeTeam"`
	AwayTeam          string    `json:"awayTeam"`
	GamesAnalyzed     int       `json:"gamesAnalyzed"`
	HomeWins          int       `json:"homeWins"`
	AwayWins          int       `json:"awayWins"`
	AvgGoalDiff       float64   `json:"avgGoalDiff"`       // Positive = home scores more
	AvgTotalGoals     float64   `json:"avgTotalGoals"`
	ConsistencyScore  float64   `json:"consistencyScore"`  // How consistent (0-1)
	TrendDirection    string    `json:"trendDirection"`    // "Stable", "Shifting home", "Shifting away"
	PredictiveValue   float64   `json:"predictiveValue"`   // How useful (0-1)
	StyleStability    bool      `json:"styleStability"`    // Have styles been consistent?
	LastMeetingResult string    `json:"lastMeetingResult"` // Most recent result
	LastMeetingDate   time.Time `json:"lastMeetingDate"`
	LastUpdated       time.Time `json:"lastUpdated"`
}

// PlaystyleEvolution tracks how a team's style changes over time
type PlaystyleEvolution struct {
	TeamCode         string                   `json:"teamCode"`
	Season           string                   `json:"season"`
	StyleChanges     []StyleChange            `json:"styleChanges"`
	CurrentStability float64                  `json:"currentStability"` // How stable is current style (0-1)
	TrendDirection   string                   `json:"trendDirection"`   // Where style is heading
	LastUpdated      time.Time                `json:"lastUpdated"`
}

// StyleChange represents a shift in team playstyle
type StyleChange struct {
	Date         time.Time `json:"date"`
	FromStyle    string    `json:"fromStyle"`
	ToStyle      string    `json:"toStyle"`
	Trigger      string    `json:"trigger"`      // What caused the change
	Magnitude    float64   `json:"magnitude"`    // How significant (0-1)
	Permanent    bool      `json:"permanent"`    // Was it lasting?
}

// MatchupAdvantageMatrix provides a lookup for style matchups
type MatchupAdvantageMatrix struct {
	Compatibilities map[string]map[string]float64 `json:"compatibilities"` // [Style1][Style2] -> advantage
	LastUpdated     time.Time                     `json:"lastUpdated"`
	DataQuality     float64                       `json:"dataQuality"` // Confidence in matrix (0-1)
}

