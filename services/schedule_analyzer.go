package services

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ScheduleAnalyzer analyzes remaining schedule difficulty and identifies key games
type ScheduleAnalyzer struct {
	standings    map[string]*models.TeamStanding
	allStandings []*models.TeamStanding
}

// TeamScheduleStrength contains comprehensive schedule difficulty metrics
type TeamScheduleStrength struct {
	TeamCode               string
	RemainingGames         int
	HomeGamesRemaining     int
	AwayGamesRemaining     int
	
	// Opponent Strength Metrics
	AvgOpponentPoints      float64
	AvgOpponentWinPct      float64
	StdDevOpponentStrength float64
	MedianOpponentPoints   float64
	
	// Game Type Breakdown
	DivisionGamesRemaining   int
	ConferenceGamesRemaining int
	NonConferenceGames       int
	PlayoffTeamGames         int
	TopTeamGames             int // vs teams with >100 points
	BottomTeamGames          int // vs teams with <80 points
	
	// Schedule Patterns
	BackToBackGames        int
	ThreeInFourGames       int
	RestDisadvantageGames  int // Games where opponent has more rest
	
	// Overall Difficulty Metrics
	ScheduleDifficulty     float64 // 0-10 scale
	ScheduleRank           int     // 1 = hardest in conference
	DifficultyTier         string  // "Easy", "Average", "Hard", "Brutal"
	
	// Crucial Games
	CrucialGames           []CrucialGame
	MustWinGames           int // High-importance games
	
	// Comparative
	EasierThanAverage      bool
	RelativeDifficulty     float64 // vs conference average
}

// CrucialGame represents an important upcoming game
type CrucialGame struct {
	Date               time.Time
	Opponent           string
	IsHome             bool
	IsDivisionGame     bool
	IsDirectCompetitor bool
	IsPlayoffTeam      bool
	OpponentPoints     int
	OpponentRank       int
	Importance         float64 // 0-1 scale
	Description        string   // Human-readable description
}

// NewScheduleAnalyzer creates a new schedule analyzer
func NewScheduleAnalyzer(standings []*models.TeamStanding) *ScheduleAnalyzer {
	standingsMap := make(map[string]*models.TeamStanding)
	for _, team := range standings {
		standingsMap[team.TeamAbbrev.Default] = team
	}
	
	return &ScheduleAnalyzer{
		standings:    standingsMap,
		allStandings: standings,
	}
}

// AnalyzeTeamSchedule performs comprehensive schedule analysis for a team
func (sa *ScheduleAnalyzer) AnalyzeTeamSchedule(
	teamCode string,
	remainingGames []RemainingGame,
) *TeamScheduleStrength {
	
	strength := &TeamScheduleStrength{
		TeamCode:      teamCode,
		CrucialGames:  make([]CrucialGame, 0),
	}
	
	team := sa.standings[teamCode]
	if team == nil {
		return strength
	}
	
	opponentPoints := make([]float64, 0)
	var totalOpponentWinPct float64
	
	// Track previous game for back-to-back detection
	var prevGameDate time.Time
	
	// Track games for three-in-four detection (rolling window)
	recentGames := make([]time.Time, 0)
	
	// Analyze each remaining game
	for i, game := range remainingGames {
		var opponent string
		var isHome bool
		
		// Determine if this team is playing in this game
		if game.HomeTeam == teamCode {
			opponent = game.AwayTeam
			isHome = true
			strength.HomeGamesRemaining++
		} else if game.AwayTeam == teamCode {
			opponent = game.HomeTeam
			isHome = false
			strength.AwayGamesRemaining++
		} else {
			continue // Not this team's game
		}
		
		strength.RemainingGames++
		
		// Get opponent's stats
		oppTeam := sa.standings[opponent]
		if oppTeam == nil {
			continue
		}
		
		// Opponent strength metrics
		opponentPoints = append(opponentPoints, float64(oppTeam.Points))
		totalOpponentWinPct += oppTeam.PointPctg
		
		// Check if playoff team (currently in top 8 of conference)
		if sa.isPlayoffTeam(oppTeam) {
			strength.PlayoffTeamGames++
		}
		
		// Check if top team (>100 points)
		if oppTeam.Points > 100 {
			strength.TopTeamGames++
		}
		
		// Check if bottom team (<80 points)
		if oppTeam.Points < 80 {
			strength.BottomTeamGames++
		}
		
		// Check game type
		if oppTeam.DivisionName == team.DivisionName {
			strength.DivisionGamesRemaining++
		}
		
		if oppTeam.ConferenceName == team.ConferenceName {
			strength.ConferenceGamesRemaining++
		} else {
			strength.NonConferenceGames++
		}
		
		// Back-to-back detection
		if i > 0 && !prevGameDate.IsZero() {
			hoursSinceLast := game.Date.Sub(prevGameDate).Hours()
			if hoursSinceLast >= 18 && hoursSinceLast <= 30 {
				strength.BackToBackGames++
			}
		}
		
		// Three-in-four detection (rolling 4-day window)
		recentGames = append(recentGames, game.Date)
		
		// Remove games older than 4 days from the window
		cutoffTime := game.Date.Add(-96 * time.Hour) // 4 days ago
		validGames := make([]time.Time, 0)
		for _, gDate := range recentGames {
			if gDate.After(cutoffTime) || gDate.Equal(cutoffTime) {
				validGames = append(validGames, gDate)
			}
		}
		recentGames = validGames
		
		// If 3+ games in the rolling 4-day window, count it
		if len(recentGames) >= 3 {
			strength.ThreeInFourGames++
		}
		
		prevGameDate = game.Date
		
		// Calculate game importance
		importance := sa.calculateGameImportance(team, oppTeam, isHome, game.Date)
		
		// Add to crucial games if important enough
		if importance > 0.65 {
			crucialGame := CrucialGame{
				Date:               game.Date,
				Opponent:           opponent,
				IsHome:             isHome,
				IsDivisionGame:     oppTeam.DivisionName == team.DivisionName,
				IsDirectCompetitor: sa.isDirectCompetitor(team, oppTeam),
				IsPlayoffTeam:      sa.isPlayoffTeam(oppTeam),
				OpponentPoints:     oppTeam.Points,
				OpponentRank:       sa.getTeamRank(oppTeam),
				Importance:         importance,
				Description:        sa.generateGameDescription(team, oppTeam, isHome, importance),
			}
			strength.CrucialGames = append(strength.CrucialGames, crucialGame)
			
			if importance > 0.80 {
				strength.MustWinGames++
			}
		}
	}
	
	// Calculate aggregate statistics
	if strength.RemainingGames > 0 {
		strength.AvgOpponentWinPct = totalOpponentWinPct / float64(strength.RemainingGames)
		
		// Calculate average and median opponent points
		var sum float64
		for _, pts := range opponentPoints {
			sum += pts
		}
		strength.AvgOpponentPoints = sum / float64(len(opponentPoints))
		
		// Median (proper calculation for even/odd counts)
		if len(opponentPoints) > 0 {
			sortedPts := make([]float64, len(opponentPoints))
			copy(sortedPts, opponentPoints)
			sort.Float64s(sortedPts)
			
			if len(sortedPts)%2 == 0 {
				// Even count: average of two middle values
				mid := len(sortedPts) / 2
				strength.MedianOpponentPoints = (sortedPts[mid-1] + sortedPts[mid]) / 2.0
			} else {
				// Odd count: take middle value
				strength.MedianOpponentPoints = sortedPts[len(sortedPts)/2]
			}
		}
		
		// Standard deviation
		var variance float64
		for _, pts := range opponentPoints {
			variance += math.Pow(pts-strength.AvgOpponentPoints, 2)
		}
		strength.StdDevOpponentStrength = math.Sqrt(variance / float64(len(opponentPoints)))
	}
	
	// Calculate overall schedule difficulty (0-10 scale)
	strength.ScheduleDifficulty = sa.calculateDifficulty(strength)
	
	// Assign difficulty tier
	strength.DifficultyTier = sa.getDifficultyTier(strength.ScheduleDifficulty)
	
	// Sort crucial games by date
	sort.Slice(strength.CrucialGames, func(i, j int) bool {
		return strength.CrucialGames[i].Date.Before(strength.CrucialGames[j].Date)
	})
	
	return strength
}

// calculateDifficulty computes overall schedule difficulty (0-10 scale)
func (sa *ScheduleAnalyzer) calculateDifficulty(s *TeamScheduleStrength) float64 {
	if s.RemainingGames == 0 {
		return 5.0 // Neutral if no games (also prevents division by zero in percentage calculations below)
	}
	
	// Weighted factors for schedule difficulty
	
	// 1. Opponent win percentage (35% weight)
	// Normalize: 0.400 = 0, 0.500 = 5, 0.600 = 10
	opponentFactor := ((s.AvgOpponentWinPct - 0.400) / 0.200) * 10 * 0.35
	
	// 2. Playoff team percentage (25% weight)
	playoffPct := float64(s.PlayoffTeamGames) / float64(s.RemainingGames)
	playoffFactor := playoffPct * 10 * 0.25
	
	// 3. Top team percentage (15% weight)
	topTeamPct := float64(s.TopTeamGames) / float64(s.RemainingGames)
	topTeamFactor := topTeamPct * 10 * 0.15
	
	// 4. Road game percentage (10% weight) - road is harder
	roadPct := float64(s.AwayGamesRemaining) / float64(s.RemainingGames)
	roadFactor := roadPct * 10 * 0.10
	
	// 5. Division game percentage (10% weight) - division games are harder
	divisionPct := float64(s.DivisionGamesRemaining) / float64(s.RemainingGames)
	divisionFactor := divisionPct * 10 * 0.10
	
	// 6. Back-to-back percentage (5% weight)
	b2bPct := float64(s.BackToBackGames) / float64(s.RemainingGames)
	b2bFactor := b2bPct * 10 * 0.05
	
	difficulty := opponentFactor + playoffFactor + topTeamFactor + roadFactor + divisionFactor + b2bFactor
	
	// Clamp to 0-10
	return math.Max(0, math.Min(10, difficulty))
}

// getDifficultyTier assigns a human-readable tier
func (sa *ScheduleAnalyzer) getDifficultyTier(difficulty float64) string {
	if difficulty < 3.5 {
		return "Easy"
	} else if difficulty < 5.5 {
		return "Average"
	} else if difficulty < 7.5 {
		return "Hard"
	} else {
		return "Brutal"
	}
}

// calculateGameImportance determines how important a game is (0-1 scale)
func (sa *ScheduleAnalyzer) calculateGameImportance(
	team, opponent *models.TeamStanding,
	isHome bool,
	gameDate time.Time,
) float64 {
	importance := 0.4 // Base importance
	
	// Factor 1: Division game (+0.15)
	if team.DivisionName == opponent.DivisionName {
		importance += 0.15
	}
	
	// Factor 2: Direct playoff competitor (+0.20)
	if sa.isDirectCompetitor(team, opponent) {
		importance += 0.20
	}
	
	// Factor 3: Against playoff team (+0.10)
	if sa.isPlayoffTeam(opponent) {
		importance += 0.10
	}
	
	// Factor 4: Late in season (+0.15 max)
	// Games get more important as season progresses
	gamesRemaining := 82 - team.GamesPlayed
	if gamesRemaining < 20 {
		lateSeasonBonus := (20.0 - float64(gamesRemaining)) / 20.0 * 0.15
		importance += lateSeasonBonus
	}
	
	// Factor 5: Road game vs strong team (+0.05)
	if !isHome && opponent.PointPctg > 0.600 {
		importance += 0.05
	}
	
	// Factor 6: Home game vs direct competitor (+0.05)
	if isHome && sa.isDirectCompetitor(team, opponent) {
		importance += 0.05
	}
	
	return math.Min(1.0, importance)
}

// isPlayoffTeam checks if a team is currently in playoff position
func (sa *ScheduleAnalyzer) isPlayoffTeam(team *models.TeamStanding) bool {
	rank := sa.getTeamRank(team)
	return rank <= 8
}

// getTeamRank returns the team's rank in their conference
func (sa *ScheduleAnalyzer) getTeamRank(team *models.TeamStanding) int {
	conferenceTeams := make([]*models.TeamStanding, 0)
	for _, t := range sa.allStandings {
		if t.ConferenceName == team.ConferenceName {
			conferenceTeams = append(conferenceTeams, t)
		}
	}
	
	// Sort by NHL rules
	sortedTeams := make([]*models.TeamStanding, len(conferenceTeams))
	copy(sortedTeams, conferenceTeams)
	sort.Slice(sortedTeams, func(i, j int) bool {
		return compareTeamsByNHLRules(sortedTeams[i], sortedTeams[j])
	})
	
	// Find team's rank
	for i, t := range sortedTeams {
		if t.TeamAbbrev.Default == team.TeamAbbrev.Default {
			return i + 1
		}
	}
	
	return 99
}

// isDirectCompetitor checks if two teams are competing for same playoff spot
func (sa *ScheduleAnalyzer) isDirectCompetitor(team1, team2 *models.TeamStanding) bool {
	// Must be same conference
	if team1.ConferenceName != team2.ConferenceName {
		return false
	}
	
	// Within 10 points = direct competition
	pointsDiff := math.Abs(float64(team1.Points - team2.Points))
	if pointsDiff > 10 {
		return false
	}
	
	// Both in playoff race (5th to 12th place typically)
	rank1 := sa.getTeamRank(team1)
	rank2 := sa.getTeamRank(team2)
	
	bothInBubble := (rank1 >= 5 && rank1 <= 12) && (rank2 >= 5 && rank2 <= 12)
	
	return bothInBubble || pointsDiff <= 5
}

// generateGameDescription creates human-readable description
func (sa *ScheduleAnalyzer) generateGameDescription(
	team, opponent *models.TeamStanding,
	isHome bool,
	importance float64,
) string {
	location := "vs"
	if !isHome {
		location = "@"
	}
	
	var description string
	
	if importance > 0.85 {
		description = fmt.Sprintf("CRITICAL: %s %s (Rank #%d)", location, opponent.TeamName.Default, sa.getTeamRank(opponent))
	} else if importance > 0.75 {
		description = fmt.Sprintf("Must-Win: %s %s", location, opponent.TeamName.Default)
	} else {
		description = fmt.Sprintf("Key Game: %s %s", location, opponent.TeamName.Default)
	}
	
	// Add context
	if sa.isDirectCompetitor(team, opponent) {
		description += " (Direct Competitor)"
	} else if sa.isPlayoffTeam(opponent) {
		description += " (Playoff Team)"
	}
	
	if team.DivisionName == opponent.DivisionName {
		description += " [Division]"
	}
	
	return description
}

// CalculateConferenceAverages computes average schedule difficulty for conference
func (sa *ScheduleAnalyzer) CalculateConferenceAverages(
	conference string,
	remainingGames []RemainingGame,
) map[string]float64 {
	
	difficulties := make([]float64, 0)
	
	for _, team := range sa.allStandings {
		if team.ConferenceName == conference {
			strength := sa.AnalyzeTeamSchedule(team.TeamAbbrev.Default, remainingGames)
			difficulties = append(difficulties, strength.ScheduleDifficulty)
		}
	}
	
	// Guard against empty conference (prevents division by zero)
	if len(difficulties) == 0 {
		return map[string]float64{
			"average": 0,
			"stddev":  0,
			"min":     0,
			"max":     0,
		}
	}
	
	// Calculate average, min, and max in one pass
	var sum float64
	minDiff := math.MaxFloat64
	maxDiff := -math.MaxFloat64
	
	for _, d := range difficulties {
		sum += d
		if d < minDiff {
			minDiff = d
		}
		if d > maxDiff {
			maxDiff = d
		}
	}
	
	avg := sum / float64(len(difficulties))
	
	// Calculate standard deviation
	var variance float64
	for _, d := range difficulties {
		variance += math.Pow(d-avg, 2)
	}
	stdDev := math.Sqrt(variance / float64(len(difficulties)))
	
	return map[string]float64{
		"average": avg,
		"stddev":  stdDev,
		"min":     minDiff,
		"max":     maxDiff,
	}
}

