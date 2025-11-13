package models

import (
	"fmt"
	"time"
)

// GameResult represents the result of a completed or in-progress NHL game
type GameResult struct {
	GameID      int       `json:"gameId"`
	HomeTeam    string    `json:"homeTeam"`
	AwayTeam    string    `json:"awayTeam"`
	HomeScore   int       `json:"homeScore"`
	AwayScore   int       `json:"awayScore"`
	GameState   string    `json:"gameState"` // "LIVE", "FINAL", "PREVIEW", "OFF"
	Period      int       `json:"period"`
	TimeLeft    string    `json:"timeLeft"`
	GameDate    time.Time `json:"gameDate"`
	Venue       string    `json:"venue"`
	IsOvertime  bool      `json:"isOvertime"`
	IsShootout  bool      `json:"isShootout"`
	WinningTeam string    `json:"winningTeam"`
	LosingTeam  string    `json:"losingTeam"`
	UpdatedAt   time.Time `json:"updatedAt"`

	// Additional game details
	Shots       GameShots      `json:"shots"`
	PowerPlays  GamePowerPlays `json:"powerPlays"`
	Penalties   GamePenalties  `json:"penalties"`
	GoalDetails []GoalDetail   `json:"goalDetails"`

	// Prediction tracking
	WasPredicted       bool    `json:"wasPredicted"`
	PredictedWinner    string  `json:"predictedWinner"`
	PredictedScore     string  `json:"predictedScore"`
	PredictionAccuracy float64 `json:"predictionAccuracy"`
}

// GameShots represents shot statistics for a game
type GameShots struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

// GamePowerPlays represents power play statistics
type GamePowerPlays struct {
	HomeOpportunities int `json:"homeOpportunities"`
	HomeGoals         int `json:"homeGoals"`
	AwayOpportunities int `json:"awayOpportunities"`
	AwayGoals         int `json:"awayGoals"`
}

// GamePenalties represents penalty statistics
type GamePenalties struct {
	HomePenalties int `json:"homePenalties"`
	HomePIM       int `json:"homePIM"`
	AwayPenalties int `json:"awayPenalties"`
	AwayPIM       int `json:"awayPIM"`
}

// GoalDetail represents details about a specific goal
type GoalDetail struct {
	Team     string    `json:"team"`
	Player   string    `json:"player"`
	Assists  []string  `json:"assists"`
	Period   int       `json:"period"`
	Time     string    `json:"time"`
	GoalType string    `json:"goalType"` // "EV", "PP", "SH", "EN", "PS"
	GameTime time.Time `json:"gameTime"`
}

// GetHomeWinValue returns the win value for Elo calculations
// 1.0 = home win, 0.5 = overtime/shootout loss, 0.0 = regulation loss
func (gr *GameResult) GetHomeWinValue() float64 {
	if gr.GameState != "FINAL" && gr.GameState != "OFF" {
		return 0.5 // Game not finished
	}

	if gr.HomeScore > gr.AwayScore {
		return 1.0 // Home win
	} else if gr.HomeScore < gr.AwayScore {
		if gr.IsOvertime || gr.IsShootout {
			return 0.5 // Home OT/SO loss (gets 1 point)
		}
		return 0.0 // Home regulation loss
	}

	return 0.5 // Tie (shouldn't happen in modern NHL)
}

// GetAwayWinValue returns the win value for the away team
func (gr *GameResult) GetAwayWinValue() float64 {
	return 1.0 - gr.GetHomeWinValue()
}

// GetScoreDifferential returns the goal differential (positive = home team advantage)
func (gr *GameResult) GetScoreDifferential() int {
	return gr.HomeScore - gr.AwayScore
}

// GetTotalGoals returns the total goals scored in the game
func (gr *GameResult) GetTotalGoals() int {
	return gr.HomeScore + gr.AwayScore
}

// IsHighScoringGame returns true if the game had 6+ total goals
func (gr *GameResult) IsHighScoringGame() bool {
	return gr.GetTotalGoals() >= 6
}

// IsLowScoringGame returns true if the game had 4 or fewer total goals
func (gr *GameResult) IsLowScoringGame() bool {
	return gr.GetTotalGoals() <= 4
}

// IsCloseGame returns true if the game was decided by 1 goal
func (gr *GameResult) IsCloseGame() bool {
	return abs(gr.GetScoreDifferential()) <= 1
}

// IsBlowout returns true if the game was decided by 3+ goals
func (gr *GameResult) IsBlowout() bool {
	return abs(gr.GetScoreDifferential()) >= 3
}

// GetGameType returns a classification of the game type
func (gr *GameResult) GetGameType() string {
	diff := abs(gr.GetScoreDifferential())

	if diff >= 3 {
		return "blowout"
	} else if diff <= 1 {
		return "close"
	} else {
		return "moderate"
	}
}

// WasUpset returns true if the predicted winner lost
func (gr *GameResult) WasUpset() bool {
	if !gr.WasPredicted || gr.PredictedWinner == "" {
		return false
	}

	actualWinner := gr.WinningTeam
	return gr.PredictedWinner != actualWinner
}

// CalculatePredictionAccuracy calculates how accurate the prediction was
func (gr *GameResult) CalculatePredictionAccuracy() float64 {
	if !gr.WasPredicted {
		return 0.0
	}

	accuracy := 0.0

	// Winner prediction accuracy (50% of total)
	if gr.PredictedWinner == gr.WinningTeam {
		accuracy += 0.5
	}

	// Score prediction accuracy (50% of total)
	if gr.PredictedScore != "" {
		// Parse predicted score and compare
		// This is a simplified version - you'd want more sophisticated scoring
		if gr.PredictedScore == fmt.Sprintf("%d-%d", gr.HomeScore, gr.AwayScore) {
			accuracy += 0.5 // Perfect score prediction
		} else {
			// Partial credit for close predictions
			// Implementation would go here
			accuracy += 0.1 // Minimal credit for trying
		}
	}

	gr.PredictionAccuracy = accuracy
	return accuracy
}

// Helper function for absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ============================================================================
// PHASE 4 HELPER METHODS
// ============================================================================

// DidTeamWin returns true if the specified team won the game
func (gr *GameResult) DidTeamWin(teamCode string) bool {
	return gr.WinningTeam == teamCode
}

// GetTeamResult returns the result from the perspective of the specified team
// Returns "W" for win, "L" for regulation loss, "OTL" for overtime/shootout loss
func (gr *GameResult) GetTeamResult(teamCode string) string {
	if gr.WinningTeam == teamCode {
		return "W"
	} else if gr.LosingTeam == teamCode {
		if gr.IsOvertime || gr.IsShootout {
			return "OTL"
		}
		return "L"
	}
	return "N/A" // Team not in this game
}

// GetTeamGoalDifferential returns the goal differential from the perspective of the specified team
// Positive means team scored more, negative means they allowed more
func (gr *GameResult) GetTeamGoalDifferential(teamCode string) int {
	if gr.HomeTeam == teamCode {
		return gr.GetScoreDifferential()
	} else if gr.AwayTeam == teamCode {
		return -gr.GetScoreDifferential()
	}
	return 0 // Team not in this game
}

// GetTeamScore returns the score for the specified team
func (gr *GameResult) GetTeamScore(teamCode string) int {
	if gr.HomeTeam == teamCode {
		return gr.HomeScore
	} else if gr.AwayTeam == teamCode {
		return gr.AwayScore
	}
	return 0
}

// GetOpponentTeam returns the opponent's team code
func (gr *GameResult) GetOpponentTeam(teamCode string) string {
	if gr.HomeTeam == teamCode {
		return gr.AwayTeam
	} else if gr.AwayTeam == teamCode {
		return gr.HomeTeam
	}
	return ""
}

// WentToOvertime is an alias for IsOvertime for backward compatibility
func (gr *GameResult) WentToOvertime() bool {
	return gr.IsOvertime
}

// ============================================================================
// COMPLETED GAME STORAGE STRUCTURES
// ============================================================================

// CompletedGame represents a finished game with comprehensive data for storage
type CompletedGame struct {
	// Game Identification
	GameID      int       `json:"gameId"`
	GameDate    time.Time `json:"gameDate"`
	Season      int       `json:"season"`
	GameType    int       `json:"gameType"` // 1=preseason, 2=regular, 3=playoffs
	ProcessedAt time.Time `json:"processedAt"`

	// Teams
	HomeTeam TeamGameResult `json:"homeTeam"`
	AwayTeam TeamGameResult `json:"awayTeam"`

	// Game Outcome
	Winner  string `json:"winner"`  // Team code
	WinType string `json:"winType"` // "REG", "OT", "SO"

	// Venue & Conditions
	Venue      string `json:"venue"`
	Attendance int    `json:"attendance"`

	// API Source
	DataSource  string `json:"dataSource"`  // "NHLE_API_v1"
	DataVersion string `json:"dataVersion"` // "1.0"
}

// TeamGameResult represents one team's performance in a game
type TeamGameResult struct {
	TeamCode string `json:"teamCode"`
	TeamName string `json:"teamName"`
	Score    int    `json:"score"`

	// Shooting Stats
	Shots   int     `json:"shots"`
	ShotPct float64 `json:"shotPct"`

	// Special Teams
	PowerPlayGoals   int     `json:"ppGoals"`
	PowerPlayOpps    int     `json:"ppOpps"`
	PowerPlayPct     float64 `json:"ppPct"`
	PenaltyKillSaves int     `json:"pkSaves"`
	PenaltyKillOpps  int     `json:"pkOpps"`
	PenaltyKillPct   float64 `json:"pkPct"`

	// Discipline
	PenaltyMinutes int `json:"pim"`

	// Possession
	FaceoffWins  int     `json:"faceoffWins"`
	FaceoffTotal int     `json:"faceoffTotal"`
	FaceoffPct   float64 `json:"faceoffPct"`

	// Physical
	Hits      int `json:"hits"`
	Blocks    int `json:"blocks"`
	Giveaways int `json:"giveaways"`
	Takeaways int `json:"takeaways"`

	// Goalie Performance
	GoalieName   string  `json:"goalieName"`
	Saves        int     `json:"saves"`
	SavePct      float64 `json:"savePct"`
	GoalsAgainst int     `json:"goalsAgainst"`

	// Top Performers (optional)
	TopScorer    string `json:"topScorer,omitempty"`
	TopScorerPts int    `json:"topScorerPts,omitempty"`
}

// ProcessedGamesIndex tracks which games have been processed
type ProcessedGamesIndex struct {
	LastUpdated    time.Time    `json:"lastUpdated"`
	ProcessedGames map[int]bool `json:"processedGames"` // gameID -> processed
	TotalProcessed int          `json:"totalProcessed"`
	Version        string       `json:"version"`
}

// ============================================================================
// NHL API RESPONSE STRUCTURES FOR BOXSCORE
// ============================================================================

// BoxscoreResponse represents the NHL API boxscore response
type BoxscoreResponse struct {
	GameID       int               `json:"id"`
	Season       int               `json:"season"`
	GameType     int               `json:"gameType"`
	Venue        VenueInfo         `json:"venue"`
	StartTimeUTC string            `json:"startTimeUTC"`
	GameState    string            `json:"gameState"`
	GameSchedule string            `json:"gameScheduleStateUTC"`
	PeriodDesc   PeriodDescriptor  `json:"periodDescriptor"`
	HomeTeam     BoxscoreTeam      `json:"homeTeam"`
	AwayTeam     BoxscoreTeam      `json:"awayTeam"`
	GameOutcome  GameOutcome       `json:"gameOutcome"`
	Summary      *BoxscoreSummary  `json:"summary,omitempty"`
	PlayerByGame interface{} `json:"playerByGameStats,omitempty"` // Can be object or array, ignoring for now
	BoxScore     *DetailedBoxscore `json:"boxscore,omitempty"`
}

// VenueInfo contains venue details
type VenueInfo struct {
	Default string `json:"default"`
}

// PeriodDescriptor describes the current/final period
type PeriodDescriptor struct {
	Number     int    `json:"number"`
	PeriodType string `json:"periodType"` // "REG", "OT", "SO"
}

// BoxscoreTeam represents team info in boxscore
type BoxscoreTeam struct {
	ID     int    `json:"id"`
	Name   Name   `json:"name"`
	Abbrev string `json:"abbrev"`
	Score  int    `json:"score"`
	SOG    int    `json:"sog"` // Shots on goal
	Logo   string `json:"logo"`
}

// Name structure for team names
type Name struct {
	Default string `json:"default"`
}

// GameOutcome provides the game result details
type GameOutcome struct {
	LastPeriodType string `json:"lastPeriodType"` // "REG", "OT", "SO"
}

// BoxscoreSummary contains game summary statistics
type BoxscoreSummary struct {
	TeamGameStats []TeamGameStats `json:"teamGameStats,omitempty"`
	Scoring       []ScoringPlay   `json:"scoring,omitempty"`
}

// TeamGameStats contains detailed team statistics
type TeamGameStats struct {
	Category string `json:"category"`
	HomeTeam string `json:"homeValue"`
	AwayTeam string `json:"awayValue"`
}

// ScoringPlay represents a goal in the game
type ScoringPlay struct {
	Period     int    `json:"period"`
	Time       string `json:"time"`
	Team       string `json:"teamAbbrev"`
	GoalScorer string `json:"name"`
	Assists    string `json:"assists"`
	Strength   string `json:"strength"`
}

// PlayerGameStats contains player-level statistics
type PlayerGameStats struct {
	PlayerID int    `json:"playerId"`
	Name     string `json:"name"`
	Position string `json:"position"`
	TeamID   int    `json:"teamId"`
	Goals    int    `json:"goals"`
	Assists  int    `json:"assists"`
	Points   int    `json:"points"`
	PIM      int    `json:"pim"`
	Shots    int    `json:"shots"`
}

// DetailedBoxscore contains more granular game statistics
type DetailedBoxscore struct {
	LinesScore  *LinesScore          `json:"linescore,omitempty"`
	PlayerStats interface{} `json:"playerByGameStats,omitempty"` // Can be object or array, ignoring for now
	GameInfo    *GameInfo            `json:"gameInfo,omitempty"`
}

// LinesScore shows period-by-period scoring
type LinesScore struct {
	ByPeriod []PeriodScore `json:"byPeriod,omitempty"`
	Totals   ScoreTotals   `json:"totals,omitempty"`
}

// PeriodScore represents scoring in a specific period
type PeriodScore struct {
	Period int `json:"period"`
	Home   int `json:"home"`
	Away   int `json:"away"`
}

// ScoreTotals represents final totals
type ScoreTotals struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

// BoxscorePlayerStats contains all player statistics from boxscore
type BoxscorePlayerStats struct {
	HomeTeam []PlayerStat `json:"homeTeam,omitempty"`
	AwayTeam []PlayerStat `json:"awayTeam,omitempty"`
}

// PlayerStat represents individual player performance
type PlayerStat struct {
	Name     string `json:"name"`
	Position string `json:"position"`
	Goals    int    `json:"goals"`
	Assists  int    `json:"assists"`
	Points   int    `json:"points"`
	Shots    int    `json:"shots"`
	PIM      int    `json:"pim"`
	TOI      string `json:"timeOnIce"`
}

// GameInfo contains general game information
type GameInfo struct {
	Referees   []string `json:"referees,omitempty"`
	Linesmen   []string `json:"linesmen,omitempty"`
	Attendance int      `json:"attendance,omitempty"`
}
