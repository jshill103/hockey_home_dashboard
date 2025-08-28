package services

import (
	"fmt"
	"strconv"

	"github.com/jaredshillingburg/go_uhc/models"
)

// AnalyzeTeamPerformance analyzes a team's performance based on standings
func AnalyzeTeamPerformance(teamConfig models.TeamConfig) (models.TeamPerformance, error) {
	// Get current standings
	standings, err := GetStandings()
	if err != nil {
		return models.TeamPerformance{}, fmt.Errorf("error fetching standings: %v", err)
	}

	// Find the team in the standings
	var teamStanding models.TeamStanding
	found := false
	for _, team := range standings.Standings {
		if team.TeamCommonName.Default == teamConfig.ShortName ||
			team.TeamCommonName.Default == teamConfig.Name ||
			team.TeamName.Default == teamConfig.Name {
			teamStanding = team
			found = true
			break
		}
	}

	if !found {
		return models.TeamPerformance{}, fmt.Errorf("%s not found in standings", teamConfig.Name)
	}

	// Analyze performance
	performance := models.TeamPerformance{
		TeamStanding: teamStanding,
	}

	// Trend Analysis
	if teamStanding.PointPctg > 0.600 {
		performance.TrendAnalysis = "Strong upward trend with excellent point percentage"
	} else if teamStanding.PointPctg > 0.500 {
		performance.TrendAnalysis = "Positive trend, competing for playoff position"
	} else {
		performance.TrendAnalysis = "Rebuilding phase, focusing on development"
	}

	// Playoff Chances
	if teamStanding.PointPctg > 0.650 {
		performance.PlayoffChances = "Excellent - Strong playoff contender"
	} else if teamStanding.PointPctg > 0.550 {
		performance.PlayoffChances = "Good - In playoff race"
	} else if teamStanding.PointPctg > 0.450 {
		performance.PlayoffChances = "Moderate - Need improvement"
	} else {
		performance.PlayoffChances = "Low - Rebuilding season"
	}

	// Key Strengths
	performance.KeyStrengths = analyzeStrengths(teamStanding)

	// Key Weaknesses
	performance.KeyWeaknesses = analyzeWeaknesses(teamStanding)

	// Recent Form
	if teamStanding.L10Points >= 16 {
		performance.RecentForm = "Excellent recent form (8+ points in last 10)"
	} else if teamStanding.L10Points >= 12 {
		performance.RecentForm = "Good recent form (6-7 points in last 10)"
	} else if teamStanding.L10Points >= 8 {
		performance.RecentForm = "Average recent form (4-5 points in last 10)"
	} else {
		performance.RecentForm = "Poor recent form (less than 4 points in last 10)"
	}

	// Home/Road Balance
	homeWinPct := float64(teamStanding.HomeWins) / float64(teamStanding.HomeWins+teamStanding.HomeLosses+teamStanding.HomeOtLosses)
	roadWinPct := float64(teamStanding.RoadWins) / float64(teamStanding.RoadWins+teamStanding.RoadLosses+teamStanding.RoadOtLosses)

	if homeWinPct > roadWinPct+0.200 {
		performance.HomeRoadBalance = "Strong home team, struggles on the road"
	} else if roadWinPct > homeWinPct+0.200 {
		performance.HomeRoadBalance = "Better road team than at home"
	} else {
		performance.HomeRoadBalance = "Well-balanced home and road performance"
	}

	// Goal Scoring Trend
	goalDiff := teamStanding.GoalDifferential
	if goalDiff > 20 {
		performance.GoalScoringTrend = "Excellent offensive production"
	} else if goalDiff > 0 {
		performance.GoalScoringTrend = "Positive goal differential"
	} else if goalDiff > -20 {
		performance.GoalScoringTrend = "Balanced scoring and defense"
	} else {
		performance.GoalScoringTrend = "Offensive struggles"
	}

	// Defensive Trend
	if teamStanding.GoalAgainst < teamStanding.GamesPlayed*2 {
		performance.DefensiveTrend = "Elite defensive play"
	} else if teamStanding.GoalAgainst < teamStanding.GamesPlayed*3 {
		performance.DefensiveTrend = "Solid defensive structure"
	} else {
		performance.DefensiveTrend = "Defensive improvements needed"
	}

	return performance, nil
}

func analyzeStrengths(standing models.TeamStanding) []string {
	var strengths []string

	if standing.PointPctg > 0.600 {
		strengths = append(strengths, "High point percentage")
	}

	if standing.GoalDifferential > 10 {
		strengths = append(strengths, "Strong goal differential")
	}

	homeWinPct := float64(standing.HomeWins) / float64(standing.HomeWins+standing.HomeLosses+standing.HomeOtLosses)
	if homeWinPct > 0.650 {
		strengths = append(strengths, "Dominant at home")
	}

	if standing.L10Points >= 14 {
		strengths = append(strengths, "Excellent recent form")
	}

	if len(strengths) == 0 {
		strengths = append(strengths, "Young team with potential")
	}

	return strengths
}

func analyzeWeaknesses(standing models.TeamStanding) []string {
	var weaknesses []string

	if standing.PointPctg < 0.450 {
		weaknesses = append(weaknesses, "Below .500 record")
	}

	if standing.GoalDifferential < -10 {
		weaknesses = append(weaknesses, "Poor goal differential")
	}

	roadWinPct := float64(standing.RoadWins) / float64(standing.RoadWins+standing.RoadLosses+standing.RoadOtLosses)
	if roadWinPct < 0.350 {
		weaknesses = append(weaknesses, "Struggles on the road")
	}

	if standing.L10Points < 8 {
		weaknesses = append(weaknesses, "Poor recent form")
	}

	if len(weaknesses) == 0 {
		weaknesses = append(weaknesses, "Room for improvement in consistency")
	}

	return weaknesses
}

// IsGameLive determines if a game is currently live based on game state
func IsGameLive(gameState string) bool {
	liveStates := []string{"LIVE", "CRIT", "IN_PROGRESS", "INTERMISSION"}
	for _, state := range liveStates {
		if gameState == state {
			return true
		}
	}
	return false
}

// FormatGameTime formats game time for display
func FormatGameTime(period int, periodTime string) string {
	if period == 0 {
		return "Not Started"
	}

	periodName := "Period " + strconv.Itoa(period)
	if period > 3 {
		overtimePeriod := period - 3
		if overtimePeriod == 1 {
			periodName = "Overtime"
		} else {
			periodName = "OT" + strconv.Itoa(overtimePeriod)
		}
	}

	if periodTime == "" {
		return periodName
	}

	return periodName + " - " + periodTime
}
