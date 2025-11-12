package services

import (
	"math"

	"github.com/jaredshillingburg/go_uhc/models"
)

// SimulationConfig holds configuration for adaptive simulation count (Phase 5.2)
type SimulationConfig struct {
	MinSimulations     int
	MaxSimulations     int
	DefaultSimulations int
}

// DefaultSimulationConfig returns the default configuration
func DefaultSimulationConfig() *SimulationConfig {
	return &SimulationConfig{
		MinSimulations:     500,
		MaxSimulations:     10000,
		DefaultSimulations: 5000,
	}
}

// CalculateAdaptiveSimulationCount determines optimal simulation count based on situation urgency (Phase 5.2)
func CalculateAdaptiveSimulationCount(team *models.TeamStanding, conferenceTeams []models.TeamStanding, config *SimulationConfig) int {
	if config == nil {
		config = DefaultSimulationConfig()
	}
	
	// Base simulation count
	simCount := config.DefaultSimulations
	
	// Factor 1: Games Remaining (fewer games = need more precision)
	gamesRemaining := 82 - team.GamesPlayed
	if gamesRemaining <= 5 {
		simCount += 2000 // Critical endgame: need high precision
	} else if gamesRemaining <= 10 {
		simCount += 1000 // Late season: need more precision
	} else if gamesRemaining >= 40 {
		simCount -= 1000 // Early season: less precision needed
	}
	
	// Factor 2: Playoff Position (bubble teams need more precision)
	rank := getTeamRank(team, conferenceTeams)
	if rank >= 7 && rank <= 10 {
		simCount += 2000 // On playoff bubble: critical position
	} else if rank >= 5 && rank <= 12 {
		simCount += 1000 // Near bubble: important position
	} else if rank <= 3 || rank >= 14 {
		simCount -= 1000 // Safely in/out: less precision needed
	}
	
	// Factor 3: Points Gap (tight race = need more precision)
	pointsGap := calculatePointsGap(team, conferenceTeams, rank)
	if pointsGap <= 2 {
		simCount += 1500 // Very tight: need high precision
	} else if pointsGap <= 5 {
		simCount += 500 // Somewhat tight: need more precision
	} else if pointsGap >= 10 {
		simCount -= 1000 // Clear gap: less precision needed
	}
	
	// Factor 4: Point Percentage (extreme win rates = more predictable)
	if team.PointPctg >= 0.650 || team.PointPctg <= 0.350 {
		simCount -= 1000 // Very good or very bad teams are more predictable
	}
	
	// Factor 5: Division Race (in division race = need more precision)
	inDivisionRace := isInDivisionRace(team, conferenceTeams)
	if inDivisionRace {
		simCount += 1000 // Division race adds complexity
	}
	
	// Clamp to min/max bounds
	if simCount < config.MinSimulations {
		simCount = config.MinSimulations
	}
	if simCount > config.MaxSimulations {
		simCount = config.MaxSimulations
	}
	
	// Round to nearest 100 for cleaner numbers
	simCount = int(math.Round(float64(simCount)/100.0) * 100)
	
	return simCount
}

// getTeamRank returns the team's rank in the conference
func getTeamRank(team *models.TeamStanding, conferenceTeams []models.TeamStanding) int {
	sorted := make([]models.TeamStanding, len(conferenceTeams))
	copy(sorted, conferenceTeams)
	SortTeamsByNHLRules(sorted)
	
	for i, t := range sorted {
		if t.TeamAbbrev.Default == team.TeamAbbrev.Default {
			return i + 1
		}
	}
	return 99
}

// calculatePointsGap calculates the points gap to playoff line
func calculatePointsGap(team *models.TeamStanding, conferenceTeams []models.TeamStanding, rank int) int {
	sorted := make([]models.TeamStanding, len(conferenceTeams))
	copy(sorted, conferenceTeams)
	SortTeamsByNHLRules(sorted)
	
	if rank <= 8 {
		// In playoffs: gap to 9th place
		if len(sorted) > 8 {
			return team.Points - sorted[8].Points
		}
		return 10 // Default large gap
	} else {
		// Out of playoffs: gap to 8th place
		if len(sorted) > 7 {
			return sorted[7].Points - team.Points
		}
		return 10 // Default large gap
	}
}

// isInDivisionRace determines if team is in a tight division race
func isInDivisionRace(team *models.TeamStanding, conferenceTeams []models.TeamStanding) bool {
	// Find teams in same division
	divisionTeams := make([]models.TeamStanding, 0)
	for _, t := range conferenceTeams {
		if t.DivisionName == team.DivisionName {
			divisionTeams = append(divisionTeams, t)
		}
	}
	
	if len(divisionTeams) < 2 {
		return false
	}
	
	// Sort by points
	SortTeamsByNHLRules(divisionTeams)
	
	// Find team's division rank
	teamRank := 0
	for i, t := range divisionTeams {
		if t.TeamAbbrev.Default == team.TeamAbbrev.Default {
			teamRank = i + 1
			break
		}
	}
	
	// Check if within 5 points of 3rd place (top 3 make playoffs)
	if teamRank <= 3 && len(divisionTeams) > 3 {
		// In top 3: check gap to 4th place
		return (divisionTeams[2].Points - divisionTeams[3].Points) <= 5
	} else if teamRank == 4 {
		// Just outside: check gap to 3rd place
		return (divisionTeams[2].Points - team.Points) <= 5
	}
	
	return false
}

// GetSimulationRecommendation returns a human-readable explanation of simulation count (Phase 5.2)
func GetSimulationRecommendation(team *models.TeamStanding, conferenceTeams []models.TeamStanding, simCount int) string {
	rank := getTeamRank(team, conferenceTeams)
	gamesRemaining := 82 - team.GamesPlayed
	pointsGap := calculatePointsGap(team, conferenceTeams, rank)
	
	if simCount >= 8000 {
		return "🔥 CRITICAL: Using maximum simulations for high-precision analysis (tight race, few games left)"
	} else if simCount >= 6000 {
		return "⚠️ HIGH URGENCY: Using extra simulations for accurate playoff odds (bubble team or late season)"
	} else if simCount == 5000 {
		return "✅ STANDARD: Using default simulation count (typical scenario)"
	} else if simCount <= 2000 {
		return "📊 QUICK: Using fewer simulations (clear position, many games remaining)"
	}
	
	// Detailed explanation
	reasons := make([]string, 0)
	if gamesRemaining <= 10 {
		reasons = append(reasons, "late season")
	}
	if rank >= 7 && rank <= 10 {
		reasons = append(reasons, "on bubble")
	}
	if pointsGap <= 5 {
		reasons = append(reasons, "tight race")
	}
	
	if len(reasons) > 0 {
		return "📈 ELEVATED: Using more simulations due to: " + joinStrings(reasons, ", ")
	}
	
	return "✅ STANDARD: Using default simulation count"
}

// joinStrings joins strings with a separator (helper function)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

