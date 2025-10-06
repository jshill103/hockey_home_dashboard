package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleBackfillPlayByPlay triggers a backfill of play-by-play data
func HandleBackfillPlayByPlay(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	teamCode := r.URL.Query().Get("team")
	numGamesStr := r.URL.Query().Get("games")
	allTeams := r.URL.Query().Get("all") == "true"

	// Default to 10 games
	numGames := 10
	if numGamesStr != "" {
		if parsed, err := strconv.Atoi(numGamesStr); err == nil && parsed > 0 && parsed <= 20 {
			numGames = parsed
		}
	}

	pbpService := services.GetPlayByPlayService()
	if pbpService == nil {
		http.Error(w, "Play-by-Play service not available", http.StatusServiceUnavailable)
		return
	}

	// Respond immediately and process in background
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	response := map[string]interface{}{
		"status": "accepted",
	}

	if allTeams {
		response["message"] = fmt.Sprintf("Backfilling all teams (last %d games). Check server logs for progress.", numGames)
		json.NewEncoder(w).Encode(response)

		// Run in background
		go func() {
			pbpService.BackfillAllTeams(numGames)
		}()
	} else if teamCode != "" {
		response["message"] = fmt.Sprintf("Backfilling %s (last %d games). Check server logs for progress.", teamCode, numGames)
		response["team"] = teamCode
		response["games"] = numGames
		json.NewEncoder(w).Encode(response)

		// Run in background
		go func() {
			pbpService.BackfillPlayByPlayData(teamCode, numGames)
		}()
	} else {
		response["error"] = "Must specify 'team' parameter or 'all=true'"
		response["status"] = "error"
		json.NewEncoder(w).Encode(response)
	}
}

// HandlePlayByPlayStats returns current play-by-play statistics
func HandlePlayByPlayStats(w http.ResponseWriter, r *http.Request) {
	teamCode := r.URL.Query().Get("team")

	pbpService := services.GetPlayByPlayService()
	if pbpService == nil {
		http.Error(w, "Play-by-Play service not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if teamCode != "" {
		// Get stats for specific team
		stats := pbpService.GetTeamStats(teamCode)
		if stats == nil {
			http.Error(w, fmt.Sprintf("No stats found for team %s", teamCode), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(stats)
	} else {
		// Return all team stats
		// This requires adding a method to get all stats
		http.Error(w, "Team parameter required", http.StatusBadRequest)
	}
}
