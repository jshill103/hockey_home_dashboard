package models

import "time"

// PlayerImpact tracks simplified player impact metrics for predictions
type PlayerImpact struct {
	TeamCode    string    `json:"teamCode"`
	Season      int       `json:"season"`
	LastUpdated time.Time `json:"lastUpdated"`

	// Top Scorers (Top 3)
	TopScorers []TopScorer `json:"topScorers"` // Top 3 point leaders
	Top3PPG    float64     `json:"top3PPG"`    // Combined PPG of top 3
	StarPower  float64     `json:"starPower"`  // 0-1 scale, elite talent

	// Depth Scoring
	Secondary4to10 float64 `json:"secondary4to10"` // PPG of 4th-10th scorers
	DepthScore     float64 `json:"depthScore"`     // 0-1 scale, depth quality
	BalanceRating  float64 `json:"balanceRating"`  // How balanced is scoring

	// Key Players
	TopGoalie     PlayerSnapshot `json:"topGoalie"`     // #1 goalie
	TopDefenseman PlayerSnapshot `json:"topDefenseman"` // #1 D-man
	Captain       PlayerSnapshot `json:"captain"`       // Team captain

	// Recent Performance
	TopScorerForm float64 `json:"topScorerForm"` // Top scorers' recent form
	DepthForm     float64 `json:"depthForm"`     // Depth players' recent form
	GoalieForm    float64 `json:"goalieForm"`    // Goalie recent performance

	// Differentials (vs league average)
	StarPowerDiff float64 `json:"starPowerDiff"` // How much better than avg
	DepthDiff     float64 `json:"depthDiff"`     // Depth vs league avg
	GoalieDiff    float64 `json:"goalieDiff"`    // Goalie vs league avg
}

// TopScorer represents a team's leading scorer
type TopScorer struct {
	PlayerID int    `json:"playerId"`
	Name     string `json:"name"`
	Position string `json:"position"` // "C", "LW", "RW", "D"
	Number   int    `json:"number"`

	// Season Stats
	GamesPlayed   int     `json:"gamesPlayed"`
	Goals         int     `json:"goals"`
	Assists       int     `json:"assists"`
	Points        int     `json:"points"`
	PointsPerGame float64 `json:"pointsPerGame"`

	// Recent Form (Last 10 Games)
	Last10Goals   int     `json:"last10Goals"`
	Last10Assists int     `json:"last10Assists"`
	Last10Points  int     `json:"last10Points"`
	Last10PPG     float64 `json:"last10PPG"`
	FormRating    float64 `json:"formRating"` // 0-10 current form

	// Status
	IsPlaying     bool `json:"isPlaying"`     // Active in recent games
	DaysSinceGame int  `json:"daysSinceGame"` // Days since last appearance
	IsCaptain     bool `json:"isCaptain"`
	IsAlternate   bool `json:"isAlternate"`
}

// PlayerSnapshot represents key info about a player
type PlayerSnapshot struct {
	PlayerID    int    `json:"playerId"`
	Name        string `json:"name"`
	Position    string `json:"position"`
	IsStarter   bool   `json:"isStarter"`
	GamesPlayed int    `json:"gamesPlayed"`
	IsPlaying   bool   `json:"isPlaying"`

	// Performance (position-specific)
	StatValue     float64 `json:"statValue"`     // Save% for G, Points for skaters
	Recent10Value float64 `json:"recent10Value"` // Recent performance
	FormRating    float64 `json:"formRating"`    // 0-10 form
}

// PlayerImpactComparison compares two teams' player impact
type PlayerImpactComparison struct {
	HomeTeam string `json:"homeTeam"`
	AwayTeam string `json:"awayTeam"`

	// Star Power Comparison
	StarPowerAdvantage    float64 `json:"starPowerAdvantage"`    // -0.10 to +0.10
	TopScorerDifferential float64 `json:"topScorerDifferential"` // PPG difference
	ElitePlayerEdge       string  `json:"elitePlayerEdge"`       // Which team has elite talent

	// Depth Comparison
	DepthAdvantage   float64 `json:"depthAdvantage"`   // -0.05 to +0.05
	BalanceAdvantage float64 `json:"balanceAdvantage"` // Which team more balanced

	// Form Comparison
	TopScorerFormDiff float64 `json:"topScorerFormDiff"` // Recent form differential
	DepthFormDiff     float64 `json:"depthFormDiff"`     // Depth form differential

	// Total Impact
	TotalPlayerImpact float64 `json:"totalPlayerImpact"` // -0.15 to +0.15
	ConfidenceLevel   float64 `json:"confidenceLevel"`   // 0-1 based on data quality

	// Explanation
	KeyFactors []string `json:"keyFactors"` // Key differences
	Reasoning  string   `json:"reasoning"`  // Human-readable
}

// TeamPlayerIndex stores all team player impacts
type TeamPlayerIndex struct {
	Teams          map[string]*PlayerImpact `json:"teams"` // Key: team code
	LastUpdated    time.Time                `json:"lastUpdated"`
	Season         int                      `json:"season"`
	LeagueAverages LeagueAverages           `json:"leagueAverages"`
}

// LeagueAverages tracks league-wide averages for comparison
type LeagueAverages struct {
	AvgTop3PPG       float64 `json:"avgTop3PPG"`
	AvgSecondaryPPG  float64 `json:"avgSecondaryPPG"`
	AvgGoalieSavePct float64 `json:"avgGoalieSavePct"`
	AvgTopScorerPPG  float64 `json:"avgTopScorerPPG"`
	AvgDepthScore    float64 `json:"avgDepthScore"`
}

// PlayerImpactFactors extends PredictionFactors with player data
type PlayerImpactFactors struct {
	// Star Power
	StarPowerRating float64 `json:"starPowerRating"` // 0-1, elite talent level
	TopScorerPPG    float64 `json:"topScorerPPG"`    // Top scorer's PPG
	Top3CombinedPPG float64 `json:"top3CombinedPPG"` // Top 3 combined PPG

	// Depth
	DepthScoring   float64 `json:"depthScoring"`   // 0-1, depth quality
	SecondaryPPG   float64 `json:"secondaryPPG"`   // 4th-10th scorer PPG
	ScoringBalance float64 `json:"scoringBalance"` // 0-1, how balanced

	// Form
	TopScorerForm float64 `json:"topScorerForm"` // 0-10, recent form
	DepthForm     float64 `json:"depthForm"`     // 0-10, depth form

	// Key Players
	StarGoalieActive bool `json:"starGoalieActive"` // #1 goalie playing
	TopDManActive    bool `json:"topDManActive"`    // #1 D-man playing
	CaptainActive    bool `json:"captainActive"`    // Captain playing

	// Differentials (vs opponent)
	StarPowerEdge float64 `json:"starPowerEdge"` // -1 to +1
	DepthEdge     float64 `json:"depthEdge"`     // -1 to +1
	FormEdge      float64 `json:"formEdge"`      // -1 to +1
}
