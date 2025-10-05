package handlers

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// CalculatePlayoffOdds analyzes current standings to determine Utah's playoff chances
// Now uses ML-powered Monte Carlo simulation for accurate odds
func CalculatePlayoffOdds() (*models.PlayoffOdds, error) {
	// Get current standings
	standings, err := services.GetStandings()
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %v", err)
	}

	// Find Utah Mammoth in standings
	var utahTeam *models.TeamStanding
	for i := range standings.Standings {
		team := &standings.Standings[i]
		if strings.Contains(strings.ToLower(team.TeamName.Default), "utah") ||
			strings.Contains(strings.ToLower(team.TeamName.Default), "mammoth") ||
			team.TeamAbbrev.Default == teamConfig.Code {
			utahTeam = team
			break
		}
	}

	if utahTeam == nil {
		return nil, fmt.Errorf("Team not found in standings")
	}

	// Separate teams by conference
	westernTeams := make([]models.TeamStanding, 0)
	for _, team := range standings.Standings {
		if team.ConferenceName == "Western" {
			westernTeams = append(westernTeams, team)
		}
	}

	// Sort western teams by points (descending)
	sort.Slice(westernTeams, func(i, j int) bool {
		if westernTeams[i].Points == westernTeams[j].Points {
			// Tie-breaker: more games played means lower effective point percentage
			return westernTeams[i].GamesPlayed < westernTeams[j].GamesPlayed
		}
		return westernTeams[i].Points > westernTeams[j].Points
	})

	// Get division teams
	centralTeams := make([]models.TeamStanding, 0)
	for _, team := range westernTeams {
		if team.DivisionName == "Central" {
			centralTeams = append(centralTeams, team)
		}
	}

	// Calculate Utah's ranks
	conferenceRank := findTeamRank(westernTeams, utahTeam)
	divisionRank := findTeamRank(centralTeams, utahTeam)

	// Calculate playoff positioning
	playoffSpotType, inPlayoffSpot := determinePlayoffSpot(westernTeams, utahTeam, conferenceRank)

	// Calculate points needed and projections
	gamesRemaining := 82 - utahTeam.GamesPlayed
	currentPace := float64(utahTeam.Points) / float64(utahTeam.GamesPlayed)
	projectedPoints := utahTeam.Points + int(math.Round(currentPace*float64(gamesRemaining)))

	// Historical playoff threshold (typically 90-100 points)
	historicalThreshold := 96

	// Calculate playoff odds using ML simulation
	playoffOdds, divisionOdds, wildCardOdds, mlSimulation := calculateMLPlayoffOdds(utahTeam.TeamAbbrev.Default)

	// Determine what Utah needs
	pointsNeeded := calculatePointsNeeded(westernTeams, utahTeam, historicalThreshold)
	winsNeeded := int(math.Ceil(float64(pointsNeeded) / 2.0)) // Assume 2 points per win
	requiredPace := 0.0
	if gamesRemaining > 0 {
		requiredPace = float64(pointsNeeded) / float64(gamesRemaining)
	}

	// Generate analysis text
	playoffStatus := determinePlayoffStatus(playoffOdds, inPlayoffSpot, pointsNeeded)
	keyInsight := generateKeyInsight(utahTeam, playoffOdds, inPlayoffSpot, gamesRemaining)
	nextMilestone := generateNextMilestone(utahTeam, pointsNeeded, projectedPoints, historicalThreshold)

	// Calculate points from playoff line
	pointsFromPlayoffs := calculatePointsFromPlayoffLine(westernTeams, utahTeam)

	return &models.PlayoffOdds{
		CurrentSeason:       services.GetCurrentSeason(),
		TeamName:            utahTeam.TeamName.Default,
		CurrentRecord:       fmt.Sprintf("%d-%d-%d", utahTeam.Wins, utahTeam.Losses, utahTeam.OtLosses),
		CurrentPoints:       utahTeam.Points,
		GamesRemaining:      gamesRemaining,
		PointsPercentage:    utahTeam.PointPctg * 100,
		DivisionName:        utahTeam.DivisionName,
		DivisionRank:        divisionRank,
		DivisionTeams:       len(centralTeams),
		ConferenceName:      utahTeam.ConferenceName,
		ConferenceRank:      conferenceRank,
		WildCardRank:        calculateWildCardRank(westernTeams, utahTeam),
		InPlayoffSpot:       inPlayoffSpot,
		PlayoffSpotType:     playoffSpotType,
		PointsFromPlayoffs:  pointsFromPlayoffs,
		PointsFrom8thSeed:   calculatePointsFrom8thSeed(westernTeams, utahTeam),
		ProjectedPoints:     projectedPoints,
		ProjectedRecord:     calculateProjectedRecord(utahTeam, gamesRemaining, currentPace),
		HistoricalThreshold: historicalThreshold,
		PlayoffOddsPercent:  playoffOdds,
		DivisionOddsPercent: divisionOdds,
		WildCardOddsPercent: wildCardOdds,
		PointsNeeded:        pointsNeeded,
		WinsNeeded:          winsNeeded,
		RequiredPointPace:   requiredPace,
		PlayoffStatus:       playoffStatus,
		KeyInsight:          keyInsight,
		NextMilestone:       nextMilestone,
		// ML Simulation Data
		MLSimulations: mlSimulation.TotalSimulations,
		MLAvgPoints:   mlSimulation.AvgFinalPoints,
		MLBestCase:    mlSimulation.BestCasePoints,
		MLWorstCase:   mlSimulation.WorstCasePoints,
	}, nil
}

// Helper functions
func findTeamRank(teams []models.TeamStanding, targetTeam *models.TeamStanding) int {
	for i, team := range teams {
		if team.TeamName.Default == targetTeam.TeamName.Default {
			return i + 1
		}
	}
	return len(teams)
}

func determinePlayoffSpot(westernTeams []models.TeamStanding, utahTeam *models.TeamStanding, conferenceRank int) (string, bool) {
	if conferenceRank <= 8 {
		// Check if in top 3 of division or wild card
		centralRank := 0
		centralCount := 0
		for _, team := range westernTeams {
			if team.DivisionName == "Central" {
				centralCount++
				if team.TeamName.Default == utahTeam.TeamName.Default {
					centralRank = centralCount
					break
				}
			}
		}

		if centralRank <= 3 {
			return "division", true
		} else if conferenceRank <= 8 {
			return "wildcard", true
		}
	}
	return "none", false
}

func calculatePlayoffOddsPercentage(utahTeam *models.TeamStanding, westernTeams []models.TeamStanding, projectedPoints, threshold int) float64 {
	// Simple odds calculation based on projected points vs historical threshold
	if projectedPoints >= threshold+5 {
		return 95.0
	} else if projectedPoints >= threshold {
		return 75.0
	} else if projectedPoints >= threshold-5 {
		return 50.0
	} else if projectedPoints >= threshold-10 {
		return 25.0
	} else {
		return 10.0
	}
}

func calculatePointsNeeded(westernTeams []models.TeamStanding, utahTeam *models.TeamStanding, threshold int) int {
	needed := threshold - utahTeam.Points
	if needed < 0 {
		return 0
	}
	return needed
}

func determinePlayoffStatus(odds float64, inPlayoffSpot bool, pointsNeeded int) string {
	if inPlayoffSpot && odds >= 75 {
		return "In Great Shape"
	} else if inPlayoffSpot && odds >= 50 {
		return "In Good Position"
	} else if odds >= 50 {
		return "On the Bubble"
	} else if pointsNeeded <= 10 {
		return "Fighting for Spot"
	} else {
		return "Need Strong Finish"
	}
}

func generateKeyInsight(utahTeam *models.TeamStanding, odds float64, inPlayoffSpot bool, gamesRemaining int) string {
	if inPlayoffSpot {
		return fmt.Sprintf("Currently holding a playoff spot with %.0f%% odds. Stay consistent!", odds)
	} else if odds >= 50 {
		return fmt.Sprintf("Right in the playoff race with %.0f%% odds. Every game matters!", odds)
	} else {
		return fmt.Sprintf("Need to step up - %.0f%% odds with %d games left to make a run", odds, gamesRemaining)
	}
}

func generateNextMilestone(utahTeam *models.TeamStanding, pointsNeeded int, projectedPoints, threshold int) string {
	if pointsNeeded <= 0 {
		return fmt.Sprintf("Maintain pace to stay above %d point playoff threshold", threshold)
	} else if pointsNeeded <= 5 {
		return fmt.Sprintf("Just %d more points needed to reach playoff safety", pointsNeeded)
	} else {
		return fmt.Sprintf("Need %d more points to reach typical playoff threshold", pointsNeeded)
	}
}

func calculatePointsFromPlayoffLine(westernTeams []models.TeamStanding, utahTeam *models.TeamStanding) int {
	// 8th seed is last playoff spot
	if len(westernTeams) >= 8 {
		eighthSeedPoints := westernTeams[7].Points
		return utahTeam.Points - eighthSeedPoints
	}
	return 0
}

func calculatePointsFrom8thSeed(westernTeams []models.TeamStanding, utahTeam *models.TeamStanding) int {
	if len(westernTeams) >= 8 {
		eighthSeedPoints := westernTeams[7].Points
		return utahTeam.Points - eighthSeedPoints
	}
	return 0
}

func calculateWildCardRank(westernTeams []models.TeamStanding, utahTeam *models.TeamStanding) int {
	// Count teams ahead of Utah that aren't in top 3 of their division
	wildCardRank := 0
	for _, team := range westernTeams {
		if team.Points > utahTeam.Points {
			// This is a simplified calculation - would need more complex logic for real wild card ranking
			wildCardRank++
		}
	}
	return wildCardRank
}

func calculateProjectedRecord(utahTeam *models.TeamStanding, gamesRemaining int, currentPace float64) string {
	if gamesRemaining <= 0 {
		return fmt.Sprintf("%d-%d-%d", utahTeam.Wins, utahTeam.Losses, utahTeam.OtLosses)
	}

	// Project additional wins (assuming roughly 50% of points come from regulation wins)
	additionalPoints := int(currentPace * float64(gamesRemaining))
	additionalWins := additionalPoints / 2
	additionalOT := additionalPoints % 2
	additionalLosses := gamesRemaining - additionalWins - additionalOT

	return fmt.Sprintf("%d-%d-%d",
		utahTeam.Wins+additionalWins,
		utahTeam.Losses+additionalLosses,
		utahTeam.OtLosses+additionalOT)
}

func calculateDivisionOdds(centralTeams []models.TeamStanding, utahTeam *models.TeamStanding, divisionRank int, projectedPoints int) float64 {
	if divisionRank <= 3 {
		return 60.0 // Currently in top 3
	} else if projectedPoints >= 100 {
		return 30.0 // Strong projection
	} else {
		return 15.0 // Lower chances
	}
}

func calculateWildCardOdds(westernTeams []models.TeamStanding, utahTeam *models.TeamStanding, projectedPoints int) float64 {
	// Count teams that would likely finish ahead
	teamsAhead := 0
	for _, team := range westernTeams {
		if team.Points > utahTeam.Points+10 { // Teams significantly ahead
			teamsAhead++
		}
	}

	if teamsAhead <= 6 {
		return 50.0 // Good wild card chances
	} else if teamsAhead <= 8 {
		return 25.0 // Some chances
	} else {
		return 10.0 // Long shot
	}
}

// HandlePlayoffOdds serves the playoff odds as HTML
func HandlePlayoffOdds(w http.ResponseWriter, r *http.Request) {
	odds, err := CalculatePlayoffOdds()
	if err != nil {
		http.Error(w, "Error calculating playoff odds: "+err.Error(), http.StatusInternalServerError)
		return
	}

	html := formatPlayoffOddsHTML(*odds)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// formatPlayoffOddsHTML formats the playoff odds data as HTML
func formatPlayoffOddsHTML(odds models.PlayoffOdds) string {
	var html strings.Builder

	// Main playoff status header
	html.WriteString("<div class='playoff-odds-container'>")

	// Status header with big odds percentage
	html.WriteString("<div class='playoff-status-header'>")
	html.WriteString(fmt.Sprintf("<div class='playoff-odds-main'>%.0f%%</div>", odds.PlayoffOddsPercent))
	html.WriteString(fmt.Sprintf("<div class='playoff-status-text'>%s</div>", odds.PlayoffStatus))
	html.WriteString("</div>")

	// Current position section
	html.WriteString("<div class='current-position'>")
	html.WriteString("<h4>Current Position</h4>")
	html.WriteString("<div class='position-grid'>")

	html.WriteString("<div class='position-item'>")
	html.WriteString("<div class='position-label'>Record</div>")
	html.WriteString(fmt.Sprintf("<div class='position-value'>%s (%d pts)</div>", odds.CurrentRecord, odds.CurrentPoints))
	html.WriteString("</div>")

	html.WriteString("<div class='position-item'>")
	html.WriteString("<div class='position-label'>Division</div>")
	html.WriteString(fmt.Sprintf("<div class='position-value'>#%d in %s</div>", odds.DivisionRank, odds.DivisionName))
	html.WriteString("</div>")

	html.WriteString("<div class='position-item'>")
	html.WriteString("<div class='position-label'>Conference</div>")
	html.WriteString(fmt.Sprintf("<div class='position-value'>#%d in %s</div>", odds.ConferenceRank, odds.ConferenceName))
	html.WriteString("</div>")

	// Playoff spot indicator
	playoffSpotClass := "out-of-playoffs"
	playoffSpotText := "Out of Playoffs"
	if odds.InPlayoffSpot {
		if odds.PlayoffSpotType == "division" {
			playoffSpotClass = "division-spot"
			playoffSpotText = "Division Spot"
		} else {
			playoffSpotClass = "wildcard-spot"
			playoffSpotText = "Wild Card Spot"
		}
	}

	html.WriteString(fmt.Sprintf("<div class='position-item %s'>", playoffSpotClass))
	html.WriteString("<div class='position-label'>Status</div>")
	html.WriteString(fmt.Sprintf("<div class='position-value'>%s</div>", playoffSpotText))
	html.WriteString("</div>")

	html.WriteString("</div>")
	html.WriteString("</div>")

	// ML Simulation Insights section
	html.WriteString("<div class='ml-simulation-insights'>")
	html.WriteString("<h4>🤖 ML Simulation Analysis</h4>")

	// Show simulation count if available
	if odds.MLSimulations > 0 {
		html.WriteString("<div class='ml-badge'>")
		html.WriteString(fmt.Sprintf("Based on %s Monte Carlo simulations", formatNumber(odds.MLSimulations)))
		html.WriteString("</div>")
	}

	html.WriteString("<div class='ml-scenarios-grid'>")

	// Best Case Scenario
	if odds.MLBestCase > 0 {
		html.WriteString("<div class='ml-scenario best-case'>")
		html.WriteString("<div class='scenario-label'>🌟 Best Case</div>")
		html.WriteString(fmt.Sprintf("<div class='scenario-value'>%d pts</div>", odds.MLBestCase))
		html.WriteString("</div>")
	}

	// Average Expected Outcome
	if odds.MLAvgPoints > 0 {
		html.WriteString("<div class='ml-scenario average'>")
		html.WriteString("<div class='scenario-label'>📊 Expected</div>")
		html.WriteString(fmt.Sprintf("<div class='scenario-value'>%.1f pts</div>", odds.MLAvgPoints))
		html.WriteString("</div>")
	}

	// Worst Case Scenario
	if odds.MLWorstCase > 0 {
		html.WriteString("<div class='ml-scenario worst-case'>")
		html.WriteString("<div class='scenario-label'>⚠️ Worst Case</div>")
		html.WriteString(fmt.Sprintf("<div class='scenario-value'>%d pts</div>", odds.MLWorstCase))
		html.WriteString("</div>")
	}

	// Games Remaining
	html.WriteString("<div class='ml-scenario games-left'>")
	html.WriteString("<div class='scenario-label'>🎮 Games Left</div>")
	html.WriteString(fmt.Sprintf("<div class='scenario-value'>%d</div>", odds.GamesRemaining))
	html.WriteString("</div>")

	html.WriteString("</div>")
	html.WriteString("</div>")

	// What Utah needs section
	if odds.PointsNeeded > 0 {
		html.WriteString("<div class='whats-needed'>")
		html.WriteString("<h4>What Utah Needs</h4>")
		html.WriteString("<div class='needs-grid'>")

		html.WriteString("<div class='need-item'>")
		html.WriteString("<div class='need-label'>Points Needed</div>")
		html.WriteString(fmt.Sprintf("<div class='need-value'>%d</div>", odds.PointsNeeded))
		html.WriteString("</div>")

		html.WriteString("<div class='need-item'>")
		html.WriteString("<div class='need-label'>≈ Wins Needed</div>")
		html.WriteString(fmt.Sprintf("<div class='need-value'>%d</div>", odds.WinsNeeded))
		html.WriteString("</div>")

		if odds.GamesRemaining > 0 {
			html.WriteString("<div class='need-item'>")
			html.WriteString("<div class='need-label'>Required Pace</div>")
			html.WriteString(fmt.Sprintf("<div class='need-value'>%.1f pts/game</div>", odds.RequiredPointPace))
			html.WriteString("</div>")
		}

		html.WriteString("</div>")
		html.WriteString("</div>")
	}

	// Key insight
	html.WriteString("<div class='playoff-insight'>")
	html.WriteString(fmt.Sprintf("<div class='insight-text'>%s</div>", odds.KeyInsight))
	html.WriteString(fmt.Sprintf("<div class='milestone-text'>%s</div>", odds.NextMilestone))
	html.WriteString("</div>")

	// Odds breakdown
	html.WriteString("<div class='odds-breakdown'>")
	html.WriteString("<h4>Playoff Paths</h4>")
	html.WriteString("<div class='odds-grid'>")

	html.WriteString("<div class='odds-item'>")
	html.WriteString("<div class='odds-label'>Division Top 3</div>")
	html.WriteString(fmt.Sprintf("<div class='odds-value'>%.0f%%</div>", odds.DivisionOddsPercent))
	html.WriteString("</div>")

	html.WriteString("<div class='odds-item'>")
	html.WriteString("<div class='odds-label'>Wild Card</div>")
	html.WriteString(fmt.Sprintf("<div class='odds-value'>%.0f%%</div>", odds.WildCardOddsPercent))
	html.WriteString("</div>")

	html.WriteString("</div>")
	html.WriteString("</div>")

	html.WriteString("</div>") // Close playoff-odds-container

	return html.String()
}

// formatNumber formats a number with commas (e.g., 5000 -> "5,000")
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", formatNumber(n/1000), n%1000)
}

// calculateMLPlayoffOdds uses ML-powered Monte Carlo simulation for accurate odds
func calculateMLPlayoffOdds(teamCode string) (playoffOdds, divisionOdds, wildCardOdds float64, simulation *services.SeasonSimulation) {
	// Get the playoff simulation service
	simService := services.GetPlayoffSimulationService()

	if simService == nil {
		// Fallback to simple calculation if service not available
		fmt.Println("⚠️ Playoff simulation service not available, using fallback")
		return 50.0, 30.0, 20.0, &services.SeasonSimulation{
			TotalSimulations: 0,
			AvgFinalPoints:   0,
			BestCasePoints:   0,
			WorstCasePoints:  0,
		}
	}

	// Run 5000 simulations (good balance of accuracy vs speed)
	sim, err := simService.SimulatePlayoffOdds(teamCode, 5000)
	if err != nil {
		fmt.Printf("⚠️ Playoff simulation error: %v\n", err)
		// Fallback to simple calculation
		return 50.0, 30.0, 20.0, &services.SeasonSimulation{
			TotalSimulations: 0,
			AvgFinalPoints:   0,
			BestCasePoints:   0,
			WorstCasePoints:  0,
		}
	}

	return sim.PlayoffOddsPercent, sim.DivisionOddsPercent, sim.WildCardOddsPercent, sim
}
