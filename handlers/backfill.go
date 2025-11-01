package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// HandleBackfillGameResults manually triggers processing of games from a specific date or date range
// Usage: /api/backfill-games?date=2025-01-27 or /api/backfill-games?days=7
func HandleBackfillGameResults(w http.ResponseWriter, r *http.Request) {
	grs := services.GetGameResultsService()
	if grs == nil {
		http.Error(w, "Game Results Service not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	dateStr := r.URL.Query().Get("date")
	daysStr := r.URL.Query().Get("days")

	response := map[string]interface{}{
		"status": "processing",
	}

	if dateStr != "" {
		// Process games from a specific date
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			response["status"] = "error"
			response["error"] = fmt.Sprintf("Invalid date format: %s (expected YYYY-MM-DD)", dateStr)
			json.NewEncoder(w).Encode(response)
			return
		}

		response["date"] = dateStr
		response["message"] = fmt.Sprintf("Processing games from %s. Check server logs for progress.", dateStr)
		json.NewEncoder(w).Encode(response)

		// Run in background
		go func() {
			grs.BackfillGamesForDate(parsedDate)
		}()
	} else if daysStr != "" {
		// Process games from the last N days
		days, err := strconv.Atoi(daysStr)
		if err != nil || days < 1 || days > 30 {
			response["status"] = "error"
			response["error"] = "Invalid days parameter (must be 1-30)"
			json.NewEncoder(w).Encode(response)
			return
		}

		response["days"] = days
		response["message"] = fmt.Sprintf("Processing games from last %d days. Check server logs for progress.", days)
		json.NewEncoder(w).Encode(response)

		// Run in background
		go func() {
			grs.BackfillGamesForDays(days)
		}()
	} else {
		// Just trigger a check for missed games
		response["message"] = "Checking for missed games from past 7 days..."
		json.NewEncoder(w).Encode(response)

		go func() {
			grs.CheckForMissedGames()
		}()
	}
}

// HandleCheckUnprocessedPredictions manually triggers checking for predictions without results
func HandleCheckUnprocessedPredictions(w http.ResponseWriter, r *http.Request) {
	grs := services.GetGameResultsService()
	if grs == nil {
		http.Error(w, "Game Results Service not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":  "processing",
		"message": "Checking for predictions without completed results. Check server logs for progress.",
	}
	json.NewEncoder(w).Encode(response)

	go func() {
		grs.CheckUnprocessedPredictions()
	}()
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
