package services

import (
	"fmt"
	"math"

	"github.com/jaredshillingburg/go_uhc/models"
)

// MagicNumbers represents playoff clinch and elimination scenarios
type MagicNumbers struct {
	// Clinch Numbers
	MagicNumber           int      `json:"magicNumber"`           // Points needed to clinch playoff spot (0 if clinched)
	MagicNumberWins       int      `json:"magicNumberWins"`       // Approximate wins needed
	CanClinchPlayoffs     bool     `json:"canClinchPlayoffs"`     // Can still make playoffs
	ClinchScenario        string   `json:"clinchScenario"`        // Human-readable clinch scenario
	ClinchPossibleDate    string   `json:"clinchPossibleDate"`    // Earliest possible clinch date
	
	// Elimination Numbers
	EliminationNumber     int      `json:"eliminationNumber"`     // Points that would eliminate (0 if eliminated)
	CanBeEliminated       bool     `json:"canBeEliminated"`       // Can still be eliminated
	EliminationScenario   string   `json:"eliminationScenario"`   // Human-readable elimination scenario
	EliminationDate       string   `json:"eliminationDate"`       // Possible elimination date
	
	// Additional Context
	MaxPossiblePoints     int      `json:"maxPossiblePoints"`     // Maximum points if win all remaining
	MinPossiblePoints     int      `json:"minPossiblePoints"`     // Minimum points if lose all remaining
	PointsBehind8th       int      `json:"pointsBehind8th"`       // Points behind 8th place
	PointsAhead9th        int      `json:"pointsAhead9th"`        // Points ahead of 9th place
	TeamsToJump           int      `json:"teamsToJump"`           // Teams to jump to make playoffs
	TeamsHoldingOff       int      `json:"teamsHoldingOff"`       // Teams trying to catch up
	
	// Tiebreaker Situations
	TiebreakerAdvantage   string   `json:"tiebreakerAdvantage"`   // "favorable", "unfavorable", "neutral"
	ROWvsNearbyTeams      int      `json:"rowVsNearbyTeams"`      // ROW differential vs 8th/9th place
}

// CalculateMagicNumbers computes playoff clinch and elimination scenarios
func CalculateMagicNumbers(
	team *models.TeamStanding,
	conferenceTeams []models.TeamStanding,
) *MagicNumbers {
	
	mn := &MagicNumbers{
		CanClinchPlayoffs: true,
		CanBeEliminated:   true,
	}
	
	// Calculate max and min possible points
	gamesRemaining := 82 - team.GamesPlayed
	mn.MaxPossiblePoints = team.Points + (gamesRemaining * 2) // 2 points per win
	mn.MinPossiblePoints = team.Points // If lose all remaining (0 points)
	
	// Sort teams by points
	sortedTeams := make([]models.TeamStanding, len(conferenceTeams))
	copy(sortedTeams, conferenceTeams)
	SortTeamsByNHLRules(sortedTeams)
	
	// Find team's current position and key positions
	var currentRank int
	var eighthPlaceTeam *models.TeamStanding
	var ninthPlaceTeam *models.TeamStanding
	
	for i := range sortedTeams {
		if sortedTeams[i].TeamAbbrev.Default == team.TeamAbbrev.Default {
			currentRank = i + 1
		}
		if i == 7 { // 8th place (0-indexed)
			eighthPlaceTeam = &sortedTeams[i]
		}
		if i == 8 { // 9th place
			ninthPlaceTeam = &sortedTeams[i]
		}
	}
	
	// Calculate points relationships
	if eighthPlaceTeam != nil {
		mn.PointsBehind8th = eighthPlaceTeam.Points - team.Points
		if mn.PointsBehind8th < 0 {
			mn.PointsBehind8th = 0 // Team is ahead
		}
	}
	
	if ninthPlaceTeam != nil {
		mn.PointsAhead9th = team.Points - ninthPlaceTeam.Points
		if mn.PointsAhead9th < 0 {
			mn.PointsAhead9th = 0 // Team is behind
		}
	}
	
	// Count teams to jump or hold off
	for i := range sortedTeams {
		rank := i + 1
		if rank <= 8 && rank < currentRank {
			mn.TeamsToJump++
		} else if rank > 8 && rank > currentRank {
			mn.TeamsHoldingOff++
		}
	}
	
	// Calculate magic number (clinch scenario)
	if currentRank <= 8 {
		// Currently in playoffs - calculate clinch number
		mn.MagicNumber = calculateClinchNumber(team, sortedTeams, currentRank)
		if mn.MagicNumber == 0 {
			mn.ClinchScenario = "CLINCHED: Playoff spot secured!"
			mn.CanBeEliminated = false
		} else {
			mn.MagicNumberWins = int(math.Ceil(float64(mn.MagicNumber) / 2.0))
			mn.ClinchScenario = fmt.Sprintf("Need %d points (%d wins) to clinch", 
				mn.MagicNumber, mn.MagicNumberWins)
		}
	} else {
		// Not in playoffs - calculate points needed to catch 8th
		if eighthPlaceTeam != nil {
			pointsNeeded := calculatePointsToReach8th(team, eighthPlaceTeam, sortedTeams)
			mn.MagicNumber = pointsNeeded
			mn.MagicNumberWins = int(math.Ceil(float64(pointsNeeded) / 2.0))
			
			if pointsNeeded > gamesRemaining*2 {
				mn.CanClinchPlayoffs = false
				mn.ClinchScenario = "ELIMINATED: Cannot reach playoff position"
			} else {
				mn.ClinchScenario = fmt.Sprintf("Need %d points (%d wins) to reach playoff position", 
					pointsNeeded, mn.MagicNumberWins)
			}
		}
	}
	
	// Calculate elimination number
	if currentRank > 8 {
		// Already eliminated or can't be eliminated (already out)
		if !mn.CanClinchPlayoffs {
			mn.EliminationNumber = 0
			mn.EliminationScenario = "ELIMINATED: Mathematically eliminated from playoffs"
			mn.CanBeEliminated = false
		} else {
			// Calculate how many can lose to stay alive
			mn.EliminationNumber = calculateEliminationBuffer(team, eighthPlaceTeam, gamesRemaining)
			mn.EliminationScenario = fmt.Sprintf("Can lose %d more games before elimination risk", 
				mn.EliminationNumber/2)
		}
	} else {
		// In playoffs - calculate elimination number
		if ninthPlaceTeam != nil {
			pointsBuffer := calculateEliminationBuffer(team, ninthPlaceTeam, gamesRemaining)
			mn.EliminationNumber = pointsBuffer
			
			if pointsBuffer > gamesRemaining*2 {
				// Can't be caught
				mn.CanBeEliminated = false
				mn.EliminationScenario = "SAFE: Cannot be eliminated"
			} else {
				mn.EliminationScenario = fmt.Sprintf("Buffer of %d points before elimination risk", pointsBuffer)
			}
		}
	}
	
	// Tiebreaker analysis
	mn.TiebreakerAdvantage = analyzeTiebreakerPosition(team, eighthPlaceTeam, ninthPlaceTeam)
	if eighthPlaceTeam != nil {
		mn.ROWvsNearbyTeams = team.GetROW() - eighthPlaceTeam.GetROW()
	} else if ninthPlaceTeam != nil {
		mn.ROWvsNearbyTeams = team.GetROW() - ninthPlaceTeam.GetROW()
	}
	
	return mn
}

// calculateClinchNumber calculates points needed to clinch playoff spot
func calculateClinchNumber(team *models.TeamStanding, sortedTeams []models.TeamStanding, currentRank int) int {
	// If already way ahead, clinched
	if currentRank <= 5 && team.Points > 100 {
		return 0 // Very likely clinched
	}
	
	// Find 9th place team (first team out)
	if len(sortedTeams) < 9 {
		return 0 // Not enough teams
	}
	
	ninthPlace := sortedTeams[8]
	ninthGamesRemaining := 82 - ninthPlace.GamesPlayed
	ninthMaxPoints := ninthPlace.Points + (ninthGamesRemaining * 2)
	
	// Magic number = points needed to guarantee ahead of 9th place max
	magicNumber := ninthMaxPoints - team.Points + 1
	
	if magicNumber <= 0 {
		return 0 // Already clinched
	}
	
	gamesRemaining := 82 - team.GamesPlayed
	if magicNumber > gamesRemaining*2 {
		return gamesRemaining * 2 // Can't clinch yet
	}
	
	return magicNumber
}

// calculatePointsToReach8th calculates points needed to reach 8th place
func calculatePointsToReach8th(team *models.TeamStanding, eighthPlace *models.TeamStanding, sortedTeams []models.TeamStanding) int {
	if eighthPlace == nil {
		return 99
	}
	
	// Need to match 8th place max possible points, plus 1 for tiebreaker safety
	eighthGamesRemaining := 82 - eighthPlace.GamesPlayed
	eighthMaxPoints := eighthPlace.Points + (eighthGamesRemaining * 2)
	
	pointsNeeded := eighthMaxPoints - team.Points + 1
	
	if pointsNeeded < 0 {
		pointsNeeded = 0
	}
	
	return pointsNeeded
}

// calculateEliminationBuffer calculates point buffer before elimination risk
func calculateEliminationBuffer(team *models.TeamStanding, compareTeam *models.TeamStanding, gamesRemaining int) int {
	if compareTeam == nil {
		return 999 // Can't be eliminated
	}
	
	// Buffer = current points lead over compare team
	buffer := team.Points - compareTeam.Points
	
	if buffer < 0 {
		return 0 // Already behind
	}
	
	return buffer
}

// analyzeTiebreakerPosition analyzes team's tiebreaker situation
func analyzeTiebreakerPosition(team *models.TeamStanding, eighthPlace, ninthPlace *models.TeamStanding) string {
	if team == nil {
		return "neutral"
	}
	
	// Check ROW (primary tiebreaker)
	teamROW := team.GetROW()
	
	favorableCount := 0
	unfavorableCount := 0
	
	if eighthPlace != nil {
		if teamROW > eighthPlace.GetROW() {
			favorableCount++
		} else if teamROW < eighthPlace.GetROW() {
			unfavorableCount++
		}
	}
	
	if ninthPlace != nil {
		if teamROW > ninthPlace.GetROW() {
			favorableCount++
		} else if teamROW < ninthPlace.GetROW() {
			unfavorableCount++
		}
	}
	
	if favorableCount > unfavorableCount {
		return "favorable"
	} else if unfavorableCount > favorableCount {
		return "unfavorable"
	}
	
	return "neutral"
}

