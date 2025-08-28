package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleTeamAnalysis handles team analysis requests
func HandleTeamAnalysis(w http.ResponseWriter, r *http.Request) {
	// Fetch team performance analysis
	performance, err := services.AnalyzeTeamPerformance(*teamConfig)
	if err != nil {
		w.Write([]byte("<p>Error fetching team analysis: " + err.Error() + "</p>"))
		return
	}

	html := formatAnalysisHTML(performance)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// HandleSeasonStatus handles season status requests
func HandleSeasonStatus(w http.ResponseWriter, r *http.Request) {
	// Validate with NHL API
	apiValidation, err := services.ValidateSeasonWithAPI()

	response := map[string]interface{}{
		"seasonStatus": *currentSeasonStatus,
		"apiValidation": map[string]interface{}{
			"gamesFound": apiValidation,
			"error":      nil,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err != nil {
		response["apiValidation"].(map[string]interface{})["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	jsonBytes, _ := json.Marshal(response)
	w.Write(jsonBytes)
}

func formatAnalysisHTML(performance models.TeamPerformance) string {
	var html strings.Builder

	// Container for rotating sections
	html.WriteString("<div class='analysis-container'>")

	// Navigation controls
	html.WriteString("<div class='analysis-nav'>")
	html.WriteString("<div class='nav-dots'>")
	for i := 1; i <= 5; i++ {
		activeClass := ""
		if i == 1 {
			activeClass = " active"
		}
		html.WriteString(fmt.Sprintf("<span class='nav-dot%s' onclick='showAnalysisSection(%d)'></span>", activeClass, i))
	}
	html.WriteString("</div>")
	html.WriteString("<div class='section-indicator'>")
	html.WriteString("<span id='current-section'>1</span> / 5")
	html.WriteString("</div>")
	html.WriteString("</div>")

	// Section 1: Overview & Current Form
	html.WriteString("<div class='analysis-section-rotating active' id='analysis-section-1'>")
	html.WriteString("<div class='section-header'>")
	teamName := "Team"
	if teamConfig != nil {
		teamName = teamConfig.ShortName
	}
	html.WriteString(fmt.Sprintf("<h3>🏒 %s Overview</h3>", teamName))
	html.WriteString("<div class='section-subtitle'>Current Form & Season Progress</div>")
	html.WriteString("</div>")

	html.WriteString("<div class='overview-grid'>")
	html.WriteString("<div class='overview-main'>")
	html.WriteString(fmt.Sprintf("<div class='team-record-large'>%d-%d-%d</div>",
		performance.Wins, performance.Losses, performance.OtLosses))
	html.WriteString(fmt.Sprintf("<div class='team-points'>%d Points (%.1f%%)</div>",
		performance.Points, performance.PointPctg*100))
	html.WriteString(fmt.Sprintf("<div class='division-info'>%s %s</div>",
		performance.ConferenceName, performance.DivisionName))
	html.WriteString("</div>")

	html.WriteString("<div class='current-form'>")
	streakType := "Unknown"
	if len(performance.StreakCode) > 0 {
		switch performance.StreakCode[0:1] {
		case "W":
			streakType = "Win"
		case "L":
			streakType = "Loss"
		case "O":
			streakType = "OT Loss"
		}
	}
	html.WriteString(fmt.Sprintf("<div class='streak-info'><strong>Current:</strong> %s %d</div>", streakType, performance.StreakCount))
	html.WriteString(fmt.Sprintf("<div class='last10-info'><strong>Last 10:</strong> %d-%d-%d (%d pts)</div>",
		performance.L10Wins, performance.L10Losses, performance.L10OtLosses, performance.L10Points))
	html.WriteString(fmt.Sprintf("<div class='games-progress'><strong>Progress:</strong> %d / 82 games</div>", performance.GamesPlayed))
	html.WriteString("</div>")
	html.WriteString("</div>")
	html.WriteString("</div>")

	// Section 2: Performance Metrics
	html.WriteString("<div class='analysis-section-rotating' id='analysis-section-2'>")
	html.WriteString("<div class='section-header'>")
	html.WriteString("<h3>📊 Performance Metrics</h3>")
	html.WriteString("<div class='section-subtitle'>Goals, Percentages & Analytics</div>")
	html.WriteString("</div>")

	html.WriteString("<div class='metrics-compact-grid'>")
	html.WriteString("<div class='metric-compact'>")
	html.WriteString("<div class='metric-label'>Goals For</div>")
	html.WriteString(fmt.Sprintf("<div class='metric-value-large'>%d</div>", performance.GoalFor))
	html.WriteString("</div>")
	html.WriteString("<div class='metric-compact'>")
	html.WriteString("<div class='metric-label'>Goals Against</div>")
	html.WriteString(fmt.Sprintf("<div class='metric-value-large'>%d</div>", performance.GoalAgainst))
	html.WriteString("</div>")
	goalDiffClass := "neutral"
	if performance.GoalDifferential > 0 {
		goalDiffClass = "positive"
	}
	if performance.GoalDifferential < 0 {
		goalDiffClass = "negative"
	}
	html.WriteString(fmt.Sprintf("<div class='metric-compact %s'>", goalDiffClass))
	html.WriteString("<div class='metric-label'>Goal Diff</div>")
	html.WriteString(fmt.Sprintf("<div class='metric-value-large'>%+d</div>", performance.GoalDifferential))
	html.WriteString("</div>")
	html.WriteString("</div>")

	html.WriteString("<div class='analytics-compact'>")
	if performance.GamesPlayed > 0 {
		avgGF := float64(performance.GoalFor) / float64(performance.GamesPlayed)
		avgGA := float64(performance.GoalAgainst) / float64(performance.GamesPlayed)
		pointsPace := float64(performance.Points) / float64(performance.GamesPlayed) * 82

		html.WriteString(fmt.Sprintf("<div class='analytics-item'><strong>Goals/Game:</strong> %.2f for, %.2f against</div>", avgGF, avgGA))
		html.WriteString(fmt.Sprintf("<div class='analytics-item'><strong>Win Rate:</strong> %.1f%% (%.1f%% points)</div>", performance.WinPctg*100, performance.PointPctg*100))
		html.WriteString(fmt.Sprintf("<div class='analytics-item'><strong>82-Game Pace:</strong> %.0f points</div>", pointsPace))
	}
	html.WriteString("</div>")
	html.WriteString("</div>")

	// Section 3: Home vs Road & League Standing
	html.WriteString("<div class='analysis-section-rotating' id='analysis-section-3'>")
	html.WriteString("<div class='section-header'>")
	html.WriteString("<h3>🏠 Location & League Standing</h3>")
	html.WriteString("<div class='section-subtitle'>Home/Road Splits & Playoff Position</div>")
	html.WriteString("</div>")

	html.WriteString("<div class='location-standings-grid'>")
	html.WriteString("<div class='location-splits'>")
	html.WriteString("<div class='split-item home'>")
	homeVenue := "Home"
	if teamConfig != nil {
		homeVenue = fmt.Sprintf("Home (%s)", teamConfig.Arena)
	}
	html.WriteString(fmt.Sprintf("<div class='split-label'>🏠 %s</div>", homeVenue))
	homeGames := performance.HomeWins + performance.HomeLosses + performance.HomeOtLosses
	html.WriteString(fmt.Sprintf("<div class='split-record'>%d-%d-%d (%d pts)</div>",
		performance.HomeWins, performance.HomeLosses, performance.HomeOtLosses, performance.HomePoints))
	if homeGames > 0 {
		homePct := float64(performance.HomePoints) / float64(homeGames*2) * 100
		html.WriteString(fmt.Sprintf("<div class='split-pct'>%.1f%% rate</div>", homePct))
	}
	html.WriteString("</div>")

	html.WriteString("<div class='split-item road'>")
	html.WriteString("<div class='split-label'>✈️ Road Games</div>")
	roadGames := performance.RoadWins + performance.RoadLosses + performance.RoadOtLosses
	html.WriteString(fmt.Sprintf("<div class='split-record'>%d-%d-%d (%d pts)</div>",
		performance.RoadWins, performance.RoadLosses, performance.RoadOtLosses, performance.RoadPoints))
	if roadGames > 0 {
		roadPct := float64(performance.RoadPoints) / float64(roadGames*2) * 100
		html.WriteString(fmt.Sprintf("<div class='split-pct'>%.1f%% rate</div>", roadPct))
	}
	html.WriteString("</div>")
	html.WriteString("</div>")

	html.WriteString("<div class='league-position'>")
	html.WriteString(fmt.Sprintf("<div class='position-item'><strong>Conference:</strong> %s</div>", performance.ConferenceName))
	html.WriteString(fmt.Sprintf("<div class='position-item'><strong>Division:</strong> %s</div>", performance.DivisionName))
	if performance.ClinchIndicator != "" {
		html.WriteString(fmt.Sprintf("<div class='position-item playoff-indicator'><strong>Status:</strong> %s</div>", performance.ClinchIndicator))
	}
	if performance.WildcardSequence > 0 {
		html.WriteString(fmt.Sprintf("<div class='position-item'><strong>Wildcard:</strong> #%d</div>", performance.WildcardSequence))
	}
	html.WriteString("</div>")
	html.WriteString("</div>")
	html.WriteString("</div>")

	// Section 4: Analysis & Trends (Simplified)
	html.WriteString("<div class='analysis-section-rotating' id='analysis-section-4'>")
	html.WriteString("<div class='section-header'>")
	html.WriteString("<h3>🔍 Team Analysis</h3>")
	html.WriteString("<div class='section-subtitle'>Key Insights & Performance Trends</div>")
	html.WriteString("</div>")

	// Simplified two-column layout
	html.WriteString("<div class='analysis-two-col'>")

	// Left column - Key insights
	html.WriteString("<div class='analysis-left'>")
	html.WriteString("<div class='insight-brief trend'>" + performance.TrendAnalysis + "</div>")
	html.WriteString("<div class='insight-brief recent'>" + performance.RecentForm + "</div>")
	html.WriteString("</div>")

	// Right column - Performance trends
	html.WriteString("<div class='analysis-right'>")
	html.WriteString("<div class='trend-brief offensive'><strong>🥅 Offense:</strong> " + performance.GoalScoringTrend + "</div>")
	html.WriteString("<div class='trend-brief defensive'><strong>🛡️ Defense:</strong> " + performance.DefensiveTrend + "</div>")
	html.WriteString("</div>")

	html.WriteString("</div>")
	html.WriteString("</div>")

	// Section 5: Season Projection (Simplified)
	html.WriteString("<div class='analysis-section-rotating' id='analysis-section-5'>")
	html.WriteString("<div class='section-header'>")
	html.WriteString("<h3>🎯 Season Outlook</h3>")
	html.WriteString("<div class='section-subtitle'>Playoff Projection & Team Strengths</div>")
	html.WriteString("</div>")

	html.WriteString("<div class='projection-simplified'>")
	if performance.GamesPlayed > 0 {
		remainingGames := 82 - performance.GamesPlayed
		currentPace := float64(performance.Points) / float64(performance.GamesPlayed)
		projectedPoints := float64(performance.Points) + (currentPace * float64(remainingGames))
		playoffThreshold := 96.0
		pointsNeeded := playoffThreshold - float64(performance.Points)

		// Compact projection display
		html.WriteString("<div class='projection-compact'>")
		html.WriteString(fmt.Sprintf("<div class='proj-main'><strong>%.0f</strong> projected pts</div>", projectedPoints))
		html.WriteString(fmt.Sprintf("<div class='proj-detail'>%d games left • %.1f pace</div>", remainingGames, currentPace))

		if pointsNeeded > 0 {
			html.WriteString(fmt.Sprintf("<div class='proj-target'>Need %.0f more for playoffs</div>", pointsNeeded))
		} else {
			html.WriteString("<div class='proj-good'>✅ On playoff pace</div>")
		}
		html.WriteString("</div>")
	}

	// Team strengths and weaknesses in compact format
	html.WriteString("<div class='team-summary'>")
	html.WriteString("<div class='summary-col'>")
	html.WriteString("<h4>💪 Key Strengths</h4>")
	maxStrengths := len(performance.KeyStrengths)
	if maxStrengths > 3 {
		maxStrengths = 3 // Limit to 3 to prevent overflow
	}
	for i := 0; i < maxStrengths; i++ {
		html.WriteString("<div class='summary-item'>• " + performance.KeyStrengths[i] + "</div>")
	}
	html.WriteString("</div>")

	html.WriteString("<div class='summary-col'>")
	html.WriteString("<h4>⚠️ Focus Areas</h4>")
	maxWeaknesses := len(performance.KeyWeaknesses)
	if maxWeaknesses > 3 {
		maxWeaknesses = 3 // Limit to 3 to prevent overflow
	}
	for i := 0; i < maxWeaknesses; i++ {
		html.WriteString("<div class='summary-item'>• " + performance.KeyWeaknesses[i] + "</div>")
	}
	html.WriteString("</div>")
	html.WriteString("</div>")

	html.WriteString("</div>")
	html.WriteString("</div>")

	html.WriteString("</div>") // Close analysis-container

	// JavaScript for rotation
	html.WriteString(`
	<script>
	let currentAnalysisSection = 1;
	const totalAnalysisSections = 5;
	let analysisInterval;

	function showAnalysisSection(sectionNum) {
		// Hide all sections
		for (let i = 1; i <= totalAnalysisSections; i++) {
			const section = document.getElementById('analysis-section-' + i);
			const dot = document.querySelectorAll('.nav-dot')[i-1];
			if (section) section.classList.remove('active');
			if (dot) dot.classList.remove('active');
		}
		
		// Show selected section
		const activeSection = document.getElementById('analysis-section-' + sectionNum);
		const activeDot = document.querySelectorAll('.nav-dot')[sectionNum-1];
		if (activeSection) activeSection.classList.add('active');
		if (activeDot) activeDot.classList.add('active');
		
		// Update indicator
		const indicator = document.getElementById('current-section');
		if (indicator) indicator.textContent = sectionNum;
		
		currentAnalysisSection = sectionNum;
		
		// Reset auto-rotation timer
		clearInterval(analysisInterval);
		analysisInterval = setInterval(autoRotateAnalysis, 8000);
	}

	function autoRotateAnalysis() {
		currentAnalysisSection = currentAnalysisSection >= totalAnalysisSections ? 1 : currentAnalysisSection + 1;
		showAnalysisSection(currentAnalysisSection);
	}

	// Start auto-rotation after page load
	document.addEventListener('DOMContentLoaded', function() {
		analysisInterval = setInterval(autoRotateAnalysis, 8000);
	});
	</script>
	`)

	return html.String()
}
