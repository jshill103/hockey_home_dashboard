package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// ============================================================================
// PHASE 2: ENHANCED DATA QUALITY API HANDLERS
// ============================================================================

// GetPhase2AnalyticsDashboard returns comprehensive Phase 2 analytics overview
func GetPhase2AnalyticsDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardData := make(map[string]interface{})

	// Head-to-Head service status
	h2hService := services.GetHeadToHeadService()
	if h2hService != nil {
		dashboardData["headToHeadStatus"] = "operational"
	} else {
		dashboardData["headToHeadStatus"] = "unavailable"
	}

	// Rest Impact service status
	restService := services.GetRestImpactService()
	if restService != nil {
		dashboardData["restAnalysisStatus"] = "operational"
		
		// Get rest impact summaries for all teams
		summaries := restService.GetAllTeamSummaries()
		dashboardData["restImpactSummaries"] = summaries
		dashboardData["totalTeamsAnalyzed"] = len(summaries)
	} else {
		dashboardData["restAnalysisStatus"] = "unavailable"
	}

	// Goalie matchup tracking status
	goalieService := services.GetGoalieService()
	if goalieService != nil {
		dashboardData["goalieMatchupStatus"] = "operational"
	} else {
		dashboardData["goalieMatchupStatus"] = "unavailable"
	}

	// Phase 2 feature status
	dashboardData["phase2Features"] = map[string]bool{
		"headToHeadDatabase":   true,
		"goalieMatchupHistory": true,
		"restImpactAnalysis":   true,
		"lineupStability":      false, // Not fully implemented yet
	}

	dashboardData["status"] = "Phase 2 Enhanced Data Quality Operational"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboardData)
}

// GetHeadToHeadMatchup returns head-to-head analysis for two teams
func GetHeadToHeadMatchup(w http.ResponseWriter, r *http.Request) {
	// Parse teams from URL path: /api/head-to-head/:home/:away
	path := strings.TrimPrefix(r.URL.Path, "/api/head-to-head/")
	teams := strings.Split(path, "/")
	
	if len(teams) < 2 {
		http.Error(w, "Missing home or away team parameter", http.StatusBadRequest)
		return
	}

	homeTeam := strings.ToUpper(teams[0])
	awayTeam := strings.ToUpper(teams[1])

	h2hService := services.GetHeadToHeadService()
	if h2hService == nil {
		http.Error(w, "Head-to-head service not available", http.StatusServiceUnavailable)
		return
	}

	record, err := h2hService.GetMatchupAnalysis(homeTeam, awayTeam)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get matchup analysis: %v", err), http.StatusInternalServerError)
		return
	}

	// Also get the advantage calculation
	advantage := h2hService.CalculateAdvantage(homeTeam, awayTeam)

	response := map[string]interface{}{
		"matchup":   record,
		"advantage": advantage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetGoalieMatchupHistory returns a goalie's history against a specific team
func GetGoalieMatchupHistory(w http.ResponseWriter, r *http.Request) {
	// Parse from query params: /api/goalie-matchup?id=123&opponent=BOS
	goalieIDStr := r.URL.Query().Get("id")
	opponentTeam := strings.ToUpper(r.URL.Query().Get("opponent"))

	if goalieIDStr == "" || opponentTeam == "" {
		http.Error(w, "Missing goalie ID or opponent parameter", http.StatusBadRequest)
		return
	}

	goalieID := 0
	if _, err := fmt.Sscanf(goalieIDStr, "%d", &goalieID); err != nil {
		http.Error(w, "Invalid goalie ID", http.StatusBadRequest)
		return
	}

	goalieService := services.GetGoalieService()
	if goalieService == nil {
		http.Error(w, "Goalie service not available", http.StatusServiceUnavailable)
		return
	}

	matchup := goalieService.GetGoalieMatchupHistory(goalieID, opponentTeam)
	if matchup == nil {
		// Return empty matchup instead of error
		matchup = &models.GoalieMatchup{
			GoalieID:     goalieID,
			OpponentTeam: opponentTeam,
		}
	}

	// Calculate adjustment factor
	adjustment := goalieService.GetGoalieMatchupAdjustment(goalieID, opponentTeam)

	response := map[string]interface{}{
		"matchup":    matchup,
		"adjustment": adjustment,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRestImpactAnalysis returns rest impact analysis for a team
func GetRestImpactAnalysis(w http.ResponseWriter, r *http.Request) {
	// Parse from URL path: /api/rest-impact/:team
	path := strings.TrimPrefix(r.URL.Path, "/api/rest-impact/")
	teamCode := strings.ToUpper(path)

	if teamCode == "" {
		http.Error(w, "Missing team parameter", http.StatusBadRequest)
		return
	}

	restService := services.GetRestImpactService()
	if restService == nil {
		http.Error(w, "Rest impact service not available", http.StatusServiceUnavailable)
		return
	}

	analysis := restService.GetTeamAnalysis(teamCode)
	if analysis == nil {
		http.Error(w, fmt.Sprintf("No rest analysis available for team %s", teamCode), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// GetRestAdvantageComparison compares rest situations for two teams
func GetRestAdvantageComparison(w http.ResponseWriter, r *http.Request) {
	homeTeam := strings.ToUpper(r.URL.Query().Get("home"))
	awayTeam := strings.ToUpper(r.URL.Query().Get("away"))
	homeRestStr := r.URL.Query().Get("homeRest")
	awayRestStr := r.URL.Query().Get("awayRest")

	if homeTeam == "" || awayTeam == "" {
		http.Error(w, "Missing home or away team parameter", http.StatusBadRequest)
		return
	}

	homeRestDays := 2 // Default
	awayRestDays := 2
	fmt.Sscanf(homeRestStr, "%d", &homeRestDays)
	fmt.Sscanf(awayRestStr, "%d", &awayRestDays)

	restService := services.GetRestImpactService()
	if restService == nil {
		http.Error(w, "Rest impact service not available", http.StatusServiceUnavailable)
		return
	}

	// Calculate rest advantage
	advantage := restService.CalculateRestAdvantage(homeTeam, awayTeam, homeRestDays, awayRestDays, 0, 0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(advantage)
}

// GetAllRestImpactRankings returns rest performance rankings for all teams
func GetAllRestImpactRankings(w http.ResponseWriter, r *http.Request) {
	restService := services.GetRestImpactService()
	if restService == nil {
		http.Error(w, "Rest impact service not available", http.StatusServiceUnavailable)
		return
	}

	summaries := restService.GetAllTeamSummaries()

	response := map[string]interface{}{
		"rankings": summaries,
		"count":    len(summaries),
		"description": "Teams ranked by back-to-back performance (best to worst)",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetLineupImpact returns lineup change impact for a team (placeholder)
func GetLineupImpact(w http.ResponseWriter, r *http.Request) {
	teamCode := strings.ToUpper(r.URL.Query().Get("team"))
	
	if teamCode == "" {
		http.Error(w, "Missing team parameter", http.StatusBadRequest)
		return
	}

	// Placeholder response - full implementation in Phase 2.5
	response := map[string]interface{}{
		"team":   teamCode,
		"status": "Lineup impact analysis coming in Phase 2.5",
		"note":   "This feature requires confirmed lineup data integration",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

