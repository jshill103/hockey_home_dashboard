package models

import "time"

// TacticalAdvantage represents a specific tactical edge
type TacticalAdvantage struct {
	Category      string    `json:"category"`      // "Special Teams", "Forechecking", "Neutral Zone", etc.
	HomeRating    float64   `json:"homeRating"`    // Home team rating (0-1)
	AwayRating    float64   `json:"awayRating"`    // Away team rating (0-1)
	Advantage     float64   `json:"advantage"`     // Net advantage (-1 to +1, positive = home)
	Impact        float64   `json:"impact"`        // Impact on prediction (-0.08 to +0.08)
	Confidence    float64   `json:"confidence"`    // Confidence in analysis (0-1)
	Explanation   string    `json:"explanation"`   // Human-readable explanation
	KeyMetrics    map[string]float64 `json:"keyMetrics"` // Supporting metrics
	LastUpdated   time.Time `json:"lastUpdated"`
}

// TeamTendencies tracks consistent team behavioral patterns
type TeamTendencies struct {
	TeamCode              string    `json:"teamCode"`
	Season                string    `json:"season"`
	
	// Period Performance
	StrongestPeriod       int       `json:"strongestPeriod"`       // 1, 2, or 3
	WeakestPeriod         int       `json:"weakestPeriod"`
	Period1GoalDiff       float64   `json:"period1GoalDiff"`       // Avg goal diff per period
	Period2GoalDiff       float64   `json:"period2GoalDiff"`
	Period3GoalDiff       float64   `json:"period3GoalDiff"`
	
	// Game Pace
	PreferredPace         string    `json:"preferredPace"`         // "Fast", "Slow", "Variable"
	AvgEventsPerGame      float64   `json:"avgEventsPerGame"`      // Shots + hits + faceoffs
	PaceControl           float64   `json:"paceControl"`           // Ability to dictate pace (0-1)
	
	// Lead/Trailing Behavior
	LeadBehavior          string    `json:"leadBehavior"`          // "Aggressive", "Defensive", "Balanced"
	TrailingBehavior      string    `json:"trailingBehavior"`      // "Desperate", "Measured", "Passive"
	LeadProtectionRate    float64   `json:"leadProtectionRate"`    // % of leads held
	ComebackFrequency     float64   `json:"comebackFrequency"`     // % of deficits overcome
	
	// Scoring Patterns
	FirstGoalImportance   float64   `json:"firstGoalImportance"`   // Win% when scoring first
	ScoringBurstTendency  float64   `json:"scoringBurstTendency"`  // Multiple goals in short time
	AnswerGoalRate        float64   `json:"answerGoalRate"`        // % time they score after allowing
	
	// Strategic Tendencies
	LineMatchupImportance float64   `json:"lineMatchupImportance"` // Coach's emphasis on matchups
	TimeoutUsage          string    `json:"timeoutUsage"`          // "Aggressive", "Conservative"
	GoaliePullTiming      float64   `json:"goaliePullTiming"`      // Avg time remaining when pulled
	ChallengeFrequency    float64   `json:"challengeFrequency"`    // Coach's challenge rate
	
	// System Characteristics
	ForeCheckSystem       string    `json:"foreCheckSystem"`       // "Aggressive", "Neutral", "Passive"
	DefensiveSystem       string    `json:"defensiveSystem"`       // "Collapse", "Man-to-Man", "Hybrid"
	PowerPlayStyle        string    `json:"powerPlayStyle"`        // "Umbrella", "Overload", "Movement"
	PenaltyKillStyle      string    `json:"penaltyKillStyle"`      // "Aggressive", "Box", "Hybrid"
	
	LastUpdated           time.Time `json:"lastUpdated"`
}

// TacticalFactor represents an individual tactical element
type TacticalFactor struct {
	Name         string  `json:"name"`
	Value        float64 `json:"value"`        // Numeric value
	Rating       string  `json:"rating"`       // "Excellent", "Good", "Average", "Poor"
	Importance   float64 `json:"importance"`   // Weight in overall analysis (0-1)
	Trend        string  `json:"trend"`        // "Improving", "Stable", "Declining"
	Description  string  `json:"description"`
}

// SpecialTeamsMatchup analyzes power play vs penalty kill
type SpecialTeamsMatchup struct {
	HomeTeam          string    `json:"homeTeam"`
	AwayTeam          string    `json:"awayTeam"`
	
	// Power Play Analysis
	HomePPPct         float64   `json:"homePPPct"`         // Home PP %
	AwayPKPct         float64   `json:"awayPKPct"`         // Away PK %
	PPAdvantage       float64   `json:"ppAdvantage"`       // Net PP advantage
	
	// Penalty Kill Analysis
	HomePKPct         float64   `json:"homePKPct"`         // Home PK %
	AwayPPPct         float64   `json:"awayPPPct"`         // Away PP %
	PKAdvantage       float64   `json:"pkAdvantage"`       // Net PK advantage
	
	// Overall
	NetSTAdvantage    float64   `json:"netSTAdvantage"`    // Combined ST advantage
	ImpactFactor      float64   `json:"impactFactor"`      // Prediction adjustment
	Confidence        float64   `json:"confidence"`
	ExpectedPPs       float64   `json:"expectedPPs"`       // Expected power plays in game
	LastUpdated       time.Time `json:"lastUpdated"`
}

// SystemMatchup analyzes how team systems interact
type SystemMatchup struct {
	HomeTeam             string    `json:"homeTeam"`
	AwayTeam             string    `json:"awayTeam"`
	
	// Forechecking vs Breakout
	HomeForeCheckRating  float64   `json:"homeForeCheckRating"`
	AwayBreakoutRating   float64   `json:"awayBreakoutRating"`
	ForeCheckAdvantage   float64   `json:"foreCheckAdvantage"`
	
	// Neutral Zone
	HomeNZControl        float64   `json:"homeNZControl"`
	AwayNZControl        float64   `json:"awayNZControl"`
	NeutralZoneAdvantage float64   `json:"neutralZoneAdvantage"`
	
	// Defensive Zone
	HomeDefSystem        string    `json:"homeDefSystem"`
	AwayDefSystem        string    `json:"awayDefSystem"`
	DefensiveCompatibility float64 `json:"defensiveCompatibility"` // How systems match up
	
	// Overall
	SystemCompatibility  float64   `json:"systemCompatibility"`  // -1 to +1
	ImpactFactor         float64   `json:"impactFactor"`
	Confidence           float64   `json:"confidence"`
	LastUpdated          time.Time `json:"lastUpdated"`
}

// TendencyMatchup analyzes how team tendencies interact
type TendencyMatchup struct {
	HomeTeam            string                `json:"homeTeam"`
	AwayTeam            string                `json:"awayTeam"`
	HomeTendencies      TeamTendencies        `json:"homeTendencies"`
	AwayTendencies      TeamTendencies        `json:"awayTendencies"`
	
	// Matchup Analysis
	PaceCompatibility   float64               `json:"paceCompatibility"`   // Do pace preferences align?
	PeriodAdvantages    map[int]float64       `json:"periodAdvantages"`    // Advantage by period
	FirstGoalImportance float64               `json:"firstGoalImportance"` // How critical is first goal?
	GameScriptFit       float64               `json:"gameScriptFit"`       // How well teams match up
	
	// Key Insights
	KeyAdvantages       []string              `json:"keyAdvantages"`       // Main advantages
	KeyDisadvantages    []string              `json:"keyDisadvantages"`    // Main disadvantages
	CriticalFactors     []TacticalFactor      `json:"criticalFactors"`     // Most important factors
	
	// Overall
	NetTendencyAdvantage float64              `json:"netTendencyAdvantage"` // -1 to +1
	ImpactFactor        float64               `json:"impactFactor"`
	Confidence          float64               `json:"confidence"`
	LastUpdated         time.Time             `json:"lastUpdated"`
}

// OpponentSpecificAdjustment combines all opponent-specific factors
type OpponentSpecificAdjustment struct {
	HomeTeam              string                 `json:"homeTeam"`
	AwayTeam              string                 `json:"awayTeam"`
	PlaystyleAdvantage    float64                `json:"playstyleAdvantage"`
	TacticalAdvantage     float64                `json:"tacticalAdvantage"`
	SpecialTeamsAdvantage float64                `json:"specialTeamsAdvantage"`
	SystemAdvantage       float64                `json:"systemAdvantage"`
	TendencyAdvantage     float64                `json:"tendencyAdvantage"`
	MatchupHistory        float64                `json:"matchupHistory"`
	CombinedAdjustment    float64                `json:"combinedAdjustment"`    // Total adjustment
	Confidence            float64                `json:"confidence"`
	KeyFactors            []string               `json:"keyFactors"`
	Explanation           string                 `json:"explanation"`
	LastUpdated           time.Time              `json:"lastUpdated"`
}

// TacticalAnalysisSummary provides a comprehensive tactical breakdown
type TacticalAnalysisSummary struct {
	HomeTeam          string               `json:"homeTeam"`
	AwayTeam          string               `json:"awayTeam"`
	Advantages        []TacticalAdvantage  `json:"advantages"`
	SpecialTeams      SpecialTeamsMatchup  `json:"specialTeams"`
	SystemMatchup     SystemMatchup        `json:"systemMatchup"`
	TendencyMatchup   TendencyMatchup      `json:"tendencyMatchup"`
	OverallAdvantage  string               `json:"overallAdvantage"`  // "Home", "Away", "Neutral"
	TotalImpact       float64              `json:"totalImpact"`       // Combined tactical impact
	Confidence        float64              `json:"confidence"`
	TopFactors        []string             `json:"topFactors"`        // Most important factors
	LastUpdated       time.Time            `json:"lastUpdated"`
}

