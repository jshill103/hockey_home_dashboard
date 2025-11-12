package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleGetRefereeForPrediction returns referee information formatted for the prediction UI
func HandleGetRefereeForPrediction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get game ID from query parameter
	gameIDStr := r.URL.Query().Get("gameId")
	if gameIDStr == "" {
		http.Error(w, `{"error": "gameId parameter required"}`, http.StatusBadRequest)
		return
	}

	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid game id"}`, http.StatusBadRequest)
		return
	}

	// Get home and away teams
	homeTeam := r.URL.Query().Get("homeTeam")
	awayTeam := r.URL.Query().Get("awayTeam")
	if homeTeam == "" || awayTeam == "" {
		http.Error(w, `{"error": "homeTeam and awayTeam parameters required"}`, http.StatusBadRequest)
		return
	}

	// Get referee enrichment service
	enricher := services.NewRefereeEnrichmentService()
	summary := enricher.GetRefereeImpactSummary(gameID, homeTeam, awayTeam)

	json.NewEncoder(w).Encode(summary)
}

// HandleGetRefereeWidget returns referee data formatted for widget display
func HandleGetRefereeWidget(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get game ID from query parameter
	gameIDStr := r.URL.Query().Get("gameId")
	if gameIDStr == "" {
		http.Error(w, `{"error": "gameId parameter required"}`, http.StatusBadRequest)
		return
	}

	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid game id"}`, http.StatusBadRequest)
		return
	}

	// Get home and away teams
	homeTeam := r.URL.Query().Get("homeTeam")
	awayTeam := r.URL.Query().Get("awayTeam")
	if homeTeam == "" || awayTeam == "" {
		http.Error(w, `{"error": "homeTeam and awayTeam parameters required"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	// Get referee assignment
	assignment, err := refService.GetGameAssignment(gameID)
	if err != nil {
		// Return empty widget data
		emptyWidget := map[string]interface{}{
			"available":  false,
			"message":    "No referee data available for this game",
			"showWidget": false,
		}
		json.NewEncoder(w).Encode(emptyWidget)
		return
	}

	// Get referee impact
	impact, err := refService.AnalyzeRefereeImpact(gameID, homeTeam, awayTeam)
	if err != nil {
		http.Error(w, `{"error": "Could not analyze referee impact"}`, http.StatusNotFound)
		return
	}

	// Get referee profiles
	ref1Profile, _ := refService.GenerateRefereeProfile(assignment.Referee1ID)
	ref2Profile, _ := refService.GenerateRefereeProfile(assignment.Referee2ID)

	// Build widget data
	widget := map[string]interface{}{
		"available":  true,
		"showWidget": true,
		"referees": []map[string]interface{}{
			{
				"name":   assignment.Referee1Name,
				"number": assignment.Referee1ID,
			},
			{
				"name":   assignment.Referee2Name,
				"number": assignment.Referee2ID,
			},
		},
		"impact": map[string]interface{}{
			"expectedPenalties":   impact.ExpectedPenalties,
			"homeAdvantageAdjust": impact.HomeAdvantageAdjust,
			"homeTeamBias":        impact.HomeTeamBiasScore,
			"awayTeamBias":        impact.AwayTeamBiasScore,
			"confidenceLevel":     impact.ConfidenceLevel,
			"notes":               impact.Notes,
		},
	}

	// Add tendency information if available
	if ref1Profile != nil && ref1Profile.Tendencies != nil && ref2Profile != nil && ref2Profile.Tendencies != nil {
		avgPenaltyRate := (ref1Profile.Tendencies.PenaltyCallRate + ref2Profile.Tendencies.PenaltyCallRate) / 2
		
		// Use same logic as referee_enrichment.go for consistency
		tendency := "average"
		if ref1Profile.Tendencies.TendencyType == ref2Profile.Tendencies.TendencyType {
			tendency = ref1Profile.Tendencies.TendencyType
		} else if ref1Profile.Tendencies.TendencyType == "strict" || ref2Profile.Tendencies.TendencyType == "strict" {
			tendency = "strict"
		} else if ref1Profile.Tendencies.TendencyType == "lenient" || ref2Profile.Tendencies.TendencyType == "lenient" {
			tendency = "lenient"
		}

		widget["tendencies"] = map[string]interface{}{
			"type":               tendency,
			"penaltyCallRate":    avgPenaltyRate,
			"highScoringGame":    ref1Profile.Tendencies.HighScoringGames || ref2Profile.Tendencies.HighScoringGames,
			"overUnderTendency":  ref1Profile.Tendencies.OverUnderTendency,
		}

		// Generate warnings
		warnings := []string{}
		if avgPenaltyRate > 1.15 {
			warnings = append(warnings, "Strict referees - expect high penalty count")
		}
		if impact.HomeAdvantageAdjust > 0.5 {
			warnings = append(warnings, homeTeam+" has favorable history with these refs")
		} else if impact.HomeAdvantageAdjust < -0.5 {
			warnings = append(warnings, awayTeam+" has favorable history with these refs")
		}
		if ref1Profile.Tendencies.HighScoringGames || ref2Profile.Tendencies.HighScoringGames {
			warnings = append(warnings, "Referees tend toward high-scoring games")
		}

		widget["warnings"] = warnings
	}

	json.NewEncoder(w).Encode(widget)
}

// HandleGetRefereeInsights returns referee insights for a specific game
func HandleGetRefereeInsights(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get game ID from query parameter
	gameIDStr := r.URL.Query().Get("gameId")
	if gameIDStr == "" {
		http.Error(w, `{"error": "gameId parameter required"}`, http.StatusBadRequest)
		return
	}

	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid game id"}`, http.StatusBadRequest)
		return
	}

	// Get home and away teams
	homeTeam := r.URL.Query().Get("homeTeam")
	awayTeam := r.URL.Query().Get("awayTeam")
	if homeTeam == "" || awayTeam == "" {
		http.Error(w, `{"error": "homeTeam and awayTeam parameters required"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	// Get advanced impact analysis
	analysis, err := refService.GetAdvancedImpactAnalysis(gameID, homeTeam, awayTeam)
	if err != nil {
		http.Error(w, `{"error": "Could not generate referee insights"}`, http.StatusNotFound)
		return
	}

	// Format for UI display
	insights := map[string]interface{}{
		"available": true,
		"analysis":  analysis,
		"display": map[string]interface{}{
			"title":       "Referee Impact Analysis",
			"description": "How the referees assigned to this game may impact the outcome",
		},
	}

	json.NewEncoder(w).Encode(insights)
}

