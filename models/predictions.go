package models

import "time"

// GamePrediction represents AI prediction for an upcoming game
type GamePrediction struct {
	GameID      int              `json:"gameId"`
	GameDate    time.Time        `json:"gameDate"`
	HomeTeam    PredictionTeam   `json:"homeTeam"`
	AwayTeam    PredictionTeam   `json:"awayTeam"`
	Prediction  PredictionResult `json:"prediction"`
	Confidence  float64          `json:"confidence"`
	KeyFactors  []string         `json:"keyFactors"`
	GeneratedAt time.Time        `json:"generatedAt"`
}

// PredictionTeam holds team info for predictions
type PredictionTeam struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	WinProbability float64 `json:"winProbability"`
	ExpectedGoals  float64 `json:"expectedGoals"`
	RecentForm     string  `json:"recentForm"` // e.g., "W-L-W-W-L"
	Streak         string  `json:"streak"`     // e.g., "3W" or "2L"
}

// PredictionResult holds the main prediction outcome
type PredictionResult struct {
	Winner         string        `json:"winner"`         // Team code of predicted winner
	WinProbability float64       `json:"winProbability"` // 0.0 to 1.0
	PredictedScore string        `json:"predictedScore"` // e.g., "4-2"
	IsUpset        bool          `json:"isUpset"`        // True if underdog predicted to win
	GameType       string        `json:"gameType"`       // "blowout", "close", "toss-up"
	ModelResults   []ModelResult `json:"modelResults"`   // Results from individual models
	EnsembleMethod string        `json:"ensembleMethod"` // How models were combined
	Confidence     float64       `json:"confidence"`     // Overall ensemble confidence
}

// ModelResult represents prediction from a single model
type ModelResult struct {
	ModelName      string  `json:"modelName"`      // e.g., "Statistical", "Bayesian", "Monte Carlo"
	WinProbability float64 `json:"winProbability"` // This model's win probability
	Confidence     float64 `json:"confidence"`     // This model's confidence
	PredictedScore string  `json:"predictedScore"` // This model's score prediction
	Weight         float64 `json:"weight"`         // Weight in ensemble (0-1)
	ProcessingTime int64   `json:"processingTime"` // Time taken in milliseconds
}

// PredictionFactors holds data used for predictions with advanced analytics integration
type PredictionFactors struct {
	TeamCode          string  `json:"teamCode"`
	WinPercentage     float64 `json:"winPercentage"`
	HomeAdvantage     float64 `json:"homeAdvantage"`
	RecentForm        float64 `json:"recentForm"`   // Last 10 games performance
	HeadToHead        float64 `json:"headToHead"`   // Historical matchup record
	GoalsFor          float64 `json:"goalsFor"`     // Average goals scored
	GoalsAgainst      float64 `json:"goalsAgainst"` // Average goals allowed
	PowerPlayPct      float64 `json:"powerPlayPct"`
	PenaltyKillPct    float64 `json:"penaltyKillPct"`
	RestDays          int     `json:"restDays"`
	BackToBackPenalty float64 `json:"backToBackPenalty"`

	// Situational Factors
	TravelFatigue    TravelFatigue    `json:"travelFatigue"`
	AltitudeAdjust   AltitudeAdjust   `json:"altitudeAdjust"`
	ScheduleStrength ScheduleStrength `json:"scheduleStrength"`
	InjuryImpact     InjuryImpact     `json:"injuryImpact"`
	MomentumFactors  MomentumFactors  `json:"momentumFactors"`

	// Advanced Hockey Analytics Integration
	AdvancedStats AdvancedAnalytics `json:"advancedStats"`

	// Weather Impact Analysis
	WeatherAnalysis WeatherAnalysis `json:"weatherAnalysis"`

	// NEW: Market Data Integration
	MarketData MarketAdjustment `json:"marketData"`

	// ============================================================================
	// PHASE 2: ENHANCED DATA QUALITY (+6 features)
	// ============================================================================
	
	// Head-to-Head Matchup Analysis
	HeadToHeadAdvantage   float64 `json:"headToHeadAdvantage"`   // -0.30 to +0.30 win% adjustment
	H2HRecentForm         float64 `json:"h2hRecentForm"`         // Recent H2H performance (0-1)
	
	// Goalie Matchup History
	GoalieVsTeamRating    float64 `json:"goalieVsTeamRating"`    // Goalie's historical performance vs this opponent (-0.15 to +0.15)
	
	// Rest & Fatigue Analysis  
	RestAdvantageDetailed float64 `json:"restAdvantageDetailed"` // Enhanced rest analysis (-0.20 to +0.20)
	OpponentFatigue       float64 `json:"opponentFatigue"`       // Opponent's fatigue level (0-1, higher = more tired)
	
	// Lineup Stability
	LineupStabilityFactor float64 `json:"lineupStabilityFactor"` // Lineup continuity impact (0-1, higher = more stable)

	// ============================================================================
	// PHASE 4: Goalie Intelligence
	GoalieAdvantage      float64 `json:"goalieAdvantage"`      // -0.15 to +0.15 win % impact
	GoalieSavePctDiff    float64 `json:"goalieSavePctDiff"`    // Save % differential
	GoalieRecentFormDiff float64 `json:"goalieRecentFormDiff"` // Recent form differential
	GoalieFatigueDiff    float64 `json:"goalieFatigueDiff"`    // Workload differential

	// PHASE 4: Betting Market Data
	MarketConsensus     float64 `json:"marketConsensus"`     // Market win probability
	MarketLineMovement  float64 `json:"marketLineMovement"`  // Line movement indicator
	SharpMoneyIndicator float64 `json:"sharpMoneyIndicator"` // Sharp money signal (0-1)
	MarketConfidenceVal float64 `json:"marketConfidenceVal"` // Market confidence level

	// PHASE 4: Schedule Context
	TravelDistance      float64 `json:"travelDistance"`      // Miles traveled
	BackToBackIndicator float64 `json:"backToBackIndicator"` // 1.0 if B2B, 0.0 if not
	ScheduleDensity     float64 `json:"scheduleDensity"`     // Games in last 7 days
	TrapGameFactor      float64 `json:"trapGameFactor"`      // Trap game likelihood (0-1)
	PlayoffImportance   float64 `json:"playoffImportance"`   // Playoff stakes (0-1)
	RestAdvantage       float64 `json:"restAdvantage"`       // Rest days advantage

	// ============================================================================
	// PHASE 6: FEATURE ENGINEERING (+40 features)
	// ============================================================================

	// PHASE 6.1: Matchup Database (9 features - HeadToHeadAdvantage in Phase 2)
	// HeadToHeadAdvantage moved to Phase 2 for earlier integration
	RecentMatchupTrend   float64 `json:"recentMatchupTrend"`   // -0.10 to +0.10 recent trend
	VenueSpecificRecord  float64 `json:"venueSpecificRecord"`  // -0.05 to +0.05 venue advantage
	IsRivalryGame        bool    `json:"isRivalryGame"`        // Known rivalry
	RivalryIntensity     float64 `json:"rivalryIntensity"`     // 0-1 rivalry intensity
	IsDivisionGame       bool    `json:"isDivisionGame"`       // Same division
	IsPlayoffRematch     bool    `json:"isPlayoffRematch"`     // Recent playoff series
	GamesInSeries        int     `json:"gamesInSeries"`        // Total H2H games
	DaysSinceLastMeeting int     `json:"daysSinceLastMeeting"` // Days since last game
	AverageGoalDiff      float64 `json:"averageGoalDiff"`      // Avg goal differential

	// PHASE 6.2: Advanced Rolling Statistics (20 features)
	FormRating           float64 `json:"formRating"`           // 0-10 current form rating
	MomentumScore        float64 `json:"momentumScore"`        // -1 to +1 momentum
	IsHot                bool    `json:"isHot"`                // 4+ wins in last 5
	IsCold               bool    `json:"isCold"`               // 4+ losses in last 5
	IsStreaking          bool    `json:"isStreaking"`          // 3+ game streak
	WeightedWinPct       float64 `json:"weightedWinPct"`       // Time-weighted win %
	WeightedGoalsFor     float64 `json:"weightedGoalsFor"`     // Time-weighted GF
	WeightedGoalsAgainst float64 `json:"weightedGoalsAgainst"` // Time-weighted GA
	QualityOfWins        float64 `json:"qualityOfWins"`        // 0-1 opponent quality
	QualityOfLosses      float64 `json:"qualityOfLosses"`      // 0-1 opponent quality
	VsPlayoffTeamsPct    float64 `json:"vsPlayoffTeamsPct"`    // Win % vs playoff teams
	VsTop10TeamsPct      float64 `json:"vsTop10TeamsPct"`      // Win % vs top 10
	ClutchPerformance    float64 `json:"clutchPerformance"`    // Close game win %
	Last3GamesPoints     int     `json:"last3GamesPoints"`     // Points in last 3
	Last5GamesPoints     int     `json:"last5GamesPoints"`     // Points in last 5
	GoalDifferential3    int     `json:"goalDifferential3"`    // +/- last 3 games
	GoalDifferential5    int     `json:"goalDifferential5"`    // +/- last 5 games
	ScoringTrend         float64 `json:"scoringTrend"`         // Goals/game trend
	DefensiveTrend       float64 `json:"defensiveTrend"`       // GA/game trend
	StrengthOfSchedule   float64 `json:"strengthOfSchedule"`   // 0-1 opponent quality
	AdjustedWinPct       float64 `json:"adjustedWinPct"`       // Quality-adjusted win %
	PointsTrendDirection string  `json:"pointsTrendDirection"` // "accelerating", "stable", "declining"

	// PHASE 6.3: Player Impact (10 features)
	StarPowerRating float64 `json:"starPowerRating"` // 0-1 elite talent level
	TopScorerPPG    float64 `json:"topScorerPPG"`    // Top scorer PPG
	Top3CombinedPPG float64 `json:"top3CombinedPPG"` // Top 3 combined PPG
	DepthScoring    float64 `json:"depthScoring"`    // 0-1 depth quality
	SecondaryPPG    float64 `json:"secondaryPPG"`    // 4th-10th scorer PPG
	ScoringBalance  float64 `json:"scoringBalance"`  // 0-1 scoring distribution
	TopScorerForm   float64 `json:"topScorerForm"`   // 0-10 top scorer form
	DepthForm       float64 `json:"depthForm"`       // 0-10 depth form
	StarPowerEdge   float64 `json:"starPowerEdge"`   // -1 to +1 vs opponent
	DepthEdge       float64 `json:"depthEdge"`       // -1 to +1 vs opponent

	// ============================================================================
	// PLAY-BY-PLAY ANALYTICS: EXPECTED GOALS & SHOT QUALITY (12 features)
	// ============================================================================

	// Expected Goals (xG)
	ExpectedGoalsFor     float64 `json:"expectedGoalsFor"`     // xG per game
	ExpectedGoalsAgainst float64 `json:"expectedGoalsAgainst"` // xGA per game
	XGDifferential       float64 `json:"xgDifferential"`       // xGF - xGA
	XGPerShot            float64 `json:"xgPerShot"`            // Shot quality index

	// Shot Quality & Danger
	DangerousShotsPerGame float64 `json:"dangerousShotsPerGame"` // High-danger chances
	HighDangerXG          float64 `json:"highDangerXG"`          // xG from high-danger
	ShotQualityAdvantage  float64 `json:"shotQualityAdvantage"`  // vs opponent

	// Corsi & Fenwick
	CorsiForPct   float64 `json:"corsiForPct"`   // Shot attempt %
	FenwickForPct float64 `json:"fenwickForPct"` // Unblocked shot attempt %

	// Physical & Possession
	FaceoffWinPct     float64 `json:"faceoffWinPct"`     // Faceoff win %
	PossessionRatio   float64 `json:"possessionRatio"`   // Takeaways / (Takeaways + Giveaways)
	PhysicalPlayIndex float64 `json:"physicalPlayIndex"` // Hits + Blocks per game

	// ============================================================================
	// SHIFT ANALYSIS: LINE CHEMISTRY & COACHING TENDENCIES (8 features)
	// ============================================================================

	// Line Chemistry & Usage
	AvgShiftLength  float64 `json:"avgShiftLength"`  // Average shift length (seconds)
	LineConsistency float64 `json:"lineConsistency"` // 0-1 (higher = more stable lines)
	TopLineMinutes  float64 `json:"topLineMinutes"`  // Top line average TOI
	PlayersUsed     float64 `json:"playersUsed"`     // Roster depth usage

	// Coaching Tendencies
	ShortBench       float64 `json:"shortBench"`       // 0-1 (relies heavily on top players)
	BalancedLines    float64 `json:"balancedLines"`    // 0-1 (even ice time distribution)
	FatigueIndicator float64 `json:"fatigueIndicator"` // 0-1 (long shifts, overuse)
	RollerCoaster    float64 `json:"rollerCoaster"`    // 0-1 (frequent line changes)

	// ============================================================================
	// LANDING PAGE ANALYTICS: ENHANCED PHYSICAL PLAY & ZONE CONTROL (4 features)
	// ============================================================================

	// Zone Control & Time on Attack
	TimeOnAttack         float64 `json:"timeOnAttack"`         // Minutes per game
	ZoneControlRatio     float64 `json:"zoneControlRatio"`     // Offensive / (Offensive + Defensive)
	TransitionEfficiency float64 `json:"transitionEfficiency"` // Controlled entries / total transitions

	// Special Teams Enhancement
	SpecialTeamsIndex float64 `json:"specialTeamsIndex"` // Combined PP% + PK%

	// ============================================================================
	// GAME SUMMARY ANALYTICS: ENHANCED GAME CONTEXT (6 features)
	// ============================================================================

	// Enhanced Shot Quality & Discipline
	ShotQualityIndex float64 `json:"shotQualityIndex"` // Weighted shot quality (0-1)
	DisciplineIndex  float64 `json:"disciplineIndex"`  // Penalty discipline (lower = better)

	// Enhanced Special Teams Context
	PowerPlayTime   float64 `json:"powerPlayTime"`   // PP time per game (minutes)
	PenaltyKillTime float64 `json:"penaltyKillTime"` // PK time per game (minutes)

	// Enhanced Zone Control Context
	OffensiveZoneTime float64 `json:"offensiveZoneTime"` // Offensive zone time per game
	DefensiveZoneTime float64 `json:"defensiveZoneTime"` // Defensive zone time per game

	// ============================================================================
	// FEATURE INTERACTIONS: COMPOUND EFFECTS (20 features)
	// ============================================================================
	
	// Offensive Potency Interactions
	OffensivePotency          float64 `json:"offensivePotency"`          // GoalsFor * PowerPlayPct
	ScoringPressure           float64 `json:"scoringPressure"`           // ExpectedGoalsFor * ShotQualityIndex
	EliteOffense              float64 `json:"eliteOffense"`              // StarPowerRating * TopScorerPPG
	DepthOffense              float64 `json:"depthOffense"`              // DepthScoring * SecondaryPPG
	
	// Defensive Vulnerability Interactions
	DefensiveVulnerability    float64 `json:"defensiveVulnerability"`    // GoalsAgainst * (1 - PenaltyKillPct)
	GoalieSupport             float64 `json:"goalieSupport"`             // GoalieSavePctDiff * DefensiveTrend
	DefensiveStrength         float64 `json:"defensiveStrength"`         // (1 - GoalsAgainst/6) * PenaltyKillPct
	
	// Fatigue & Travel Compound Effects
	FatigueCompound           float64 `json:"fatigueCompound"`           // RestDays * TravelDistance (negative = tired)
	BackToBackTravel          float64 `json:"backToBackTravel"`          // BackToBackIndicator * TravelDistance
	ScheduleStress            float64 `json:"scheduleStress"`            // ScheduleDensity * TravelFatigue.FatigueLevel
	
	// Momentum & Home Advantage
	HomeMomentum              float64 `json:"homeMomentum"`              // RecentForm * HomeAdvantage
	HomeFieldStrength         float64 `json:"homeFieldStrength"`         // HomeAdvantage * WeightedWinPct
	RefereeHomeBias           float64 `json:"refereeHomeBias"`           // RefereeHomeAdvantage * HomeAdvantage
	
	// Elite Performance Interactions
	ClutchElite               float64 `json:"clutchElite"`               // StarPowerRating * ClutchPerformance
	HotStreak                 float64 `json:"hotStreak"`                 // MomentumScore * (IsHot ? 1.0 : 0.0)
	FormQuality               float64 `json:"formQuality"`               // RecentForm * QualityOfWins
	
	// Special Teams Differential
	SpecialTeamsDominance     float64 `json:"specialTeamsDominance"`     // (PowerPlayPct - 0.20) * (PenaltyKillPct - 0.80)
	PowerPlayOpportunity      float64 `json:"powerPlayOpportunity"`      // PowerPlayPct * RefereePenaltyRate
	
	// Situational Context
	RivalryIntensityFactor    float64 `json:"rivalryIntensityFactor"`    // IsRivalryGame * RivalryIntensity * RecentForm
	PlayoffPressure           float64 `json:"playoffPressure"`           // PlayoffImportance * ClutchPerformance
}

// MarketAdjustment represents betting market influence on predictions
type MarketAdjustment struct {
	MarketConfidence      float64 `json:"marketConfidence"`      // 0-1, confidence in market data
	ImpliedProbability    float64 `json:"impliedProbability"`    // Market's implied win probability
	SharpMoneyFactor      float64 `json:"sharpMoneyFactor"`      // -0.1 to 0.1, sharp money influence
	PublicFadeFactor      float64 `json:"publicFadeFactor"`      // -0.1 to 0.1, contrarian value
	VolumeConfidence      float64 `json:"volumeConfidence"`      // 0-1, confidence based on volume
	MarketEfficiency      float64 `json:"marketEfficiency"`      // 0-1, how efficient the market is
	RecommendedAdjustment float64 `json:"recommendedAdjustment"` // -0.2 to 0.2, overall adjustment
}

// AdvancedAnalytics represents processed advanced stats for prediction models
type AdvancedAnalytics struct {
	// Expected Goals Performance
	XGForPerGame      float64 `json:"xgForPerGame"`      // Expected goals for per game
	XGAgainstPerGame  float64 `json:"xgAgainstPerGame"`  // Expected goals against per game
	XGDifferential    float64 `json:"xgDifferential"`    // xGF - xGA per game
	ShootingTalent    float64 `json:"shootingTalent"`    // Goals above/below expected
	GoaltendingTalent float64 `json:"goaltendingTalent"` // Saves above/below expected

	// Possession Dominance
	CorsiForPct       float64 `json:"corsiForPct"`       // Shot attempt share
	FenwickForPct     float64 `json:"fenwickForPct"`     // Unblocked shot attempt share
	HighDangerPct     float64 `json:"highDangerPct"`     // High danger chance share
	PossessionQuality float64 `json:"possessionQuality"` // Overall possession rating

	// Shot Quality & Generation
	ShotGenerationRate  float64 `json:"shotGenRate"`        // Shots for per 60 min
	ShotSuppressionRate float64 `json:"shotSuppRate"`       // Shots against per 60 min
	ShotQualityFor      float64 `json:"shotQualityFor"`     // Quality of shots taken
	ShotQualityAgainst  float64 `json:"shotQualityAgainst"` // Quality of shots allowed

	// Special Teams Excellence
	PowerPlayXG      float64 `json:"powerPlayXG"`      // PP expected goals rate
	PenaltyKillXGA   float64 `json:"penaltyKillXGA"`   // PK expected goals against
	SpecialTeamsEdge float64 `json:"specialTeamsEdge"` // Overall ST advantage

	// Goaltending Performance
	GoalieSvPctOverall float64 `json:"goalieSvPct"`    // Overall save percentage
	GoalieSvPctHD      float64 `json:"goalieSvPctHD"`  // High danger save percentage
	SavesAboveExpected float64 `json:"savesAboveExp"`  // Goalie performance vs expected
	GoalieWorkload     float64 `json:"goalieWorkload"` // Workload stress factor

	// Game State Performance
	LeadingPerformance  float64              `json:"leadingPerf"`     // Performance when leading
	TrailingPerformance float64              `json:"trailingPerf"`    // Performance when trailing
	CloseGameRecord     float64              `json:"closeGameRecord"` // Win% in close games
	PeriodStrength      PeriodStrengthRating `json:"periodStrength"`  // Period-by-period performance

	// Transition & Zone Play
	OffensiveZoneTime    float64 `json:"offensiveZoneTime"` // % time in offensive zone
	ControlledEntries    float64 `json:"controlledEntries"` // Zone entry success rate
	ControlledExits      float64 `json:"controlledExits"`   // Zone exit success rate
	TransitionEfficiency float64 `json:"transitionEff"`     // Transition goal conversion

	// Advanced Rating
	OverallRating float64  `json:"overallRating"` // Composite advanced rating (0-100)
	StrengthAreas []string `json:"strengthAreas"` // Top 3 strength categories
	WeaknessAreas []string `json:"weaknessAreas"` // Top 3 weakness categories
}

// PeriodStrengthRating represents performance strength by period
type PeriodStrengthRating struct {
	FirstPeriod     float64 `json:"period1"`   // 1st period strength rating
	SecondPeriod    float64 `json:"period2"`   // 2nd period strength rating
	ThirdPeriod     float64 `json:"period3"`   // 3rd period strength rating
	OvertimePeriod  float64 `json:"overtime"`  // Overtime strength rating
	StrongestPeriod string  `json:"strongest"` // Best performing period
	WeakestPeriod   string  `json:"weakest"`   // Worst performing period
}

// TravelFatigue represents team travel burden
type TravelFatigue struct {
	MilesTraveled    float64 `json:"milesTraveled"`    // Miles since last game
	TimeZonesCrossed int     `json:"timeZonesCrossed"` // Time zones crossed
	DaysOnRoad       int     `json:"daysOnRoad"`       // Consecutive road games
	FatigueScore     float64 `json:"fatigueScore"`     // Overall fatigue (0.0-1.0)
}

// AltitudeAdjust represents altitude-related factors
type AltitudeAdjust struct {
	VenueAltitude    float64 `json:"venueAltitude"`    // Feet above sea level
	TeamHomeAltitude float64 `json:"teamHomeAltitude"` // Team's home altitude
	AltitudeDiff     float64 `json:"altitudeDiff"`     // Difference in altitude
	AdjustmentFactor float64 `json:"adjustmentFactor"` // Performance adjustment
}

// ScheduleStrength represents recent schedule difficulty
type ScheduleStrength struct {
	GamesInLast7Days int     `json:"gamesInLast7Days"` // Games played recently
	OpponentStrength float64 `json:"opponentStrength"` // Avg opponent win %
	RestAdvantage    float64 `json:"restAdvantage"`    // Rest vs opponent
	ScheduleDensity  float64 `json:"scheduleDensity"`  // Games per day ratio
}

// InjuryImpact represents roster and injury effects
// InjuryImpact represents comprehensive injury impact analysis
type InjuryImpact struct {
	// Basic Impact
	KeyPlayersOut int     `json:"keyPlayersOut"` // Number of key injuries
	GoalieStatus  string  `json:"goalieStatus"`  // "starter", "backup", "emergency"
	InjuryScore   float64 `json:"injuryScore"`   // Overall impact (0.0-1.0)
	LineupChanges int     `json:"lineupChanges"` // Recent lineup changes

	// Enhanced Real-Time Data
	InjuredPlayers   int       `json:"injuredPlayers"`   // Total injured players
	KeyInjuries      int       `json:"keyInjuries"`      // High-impact injuries
	ImpactScore      float64   `json:"impactScore"`      // Real-time impact score (0-50)
	HealthPercentage float64   `json:"healthPercentage"` // Team health percentage (0-100)
	InjuryTrend      string    `json:"injuryTrend"`      // "improving", "stable", "declining"
	Description      string    `json:"description"`      // Human-readable description
	LastUpdated      time.Time `json:"lastUpdated"`      // When data was last updated
	Confidence       float64   `json:"confidence"`       // Data confidence (0-1)
}

// MomentumFactors represents psychological/momentum factors
type MomentumFactors struct {
	WinStreak       int     `json:"winStreak"`       // Current win/loss streak
	HomeStandLength int     `json:"homeStandLength"` // Games in current home stand
	LastGameMargin  int     `json:"lastGameMargin"`  // Goal margin in last game
	RecentBlowouts  int     `json:"recentBlowouts"`  // Blowout wins in last 5
	MomentumScore   float64 `json:"momentumScore"`   // Overall momentum (0.0-1.0)
}
