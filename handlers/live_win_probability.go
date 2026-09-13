package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleLiveWinProbability returns a live, in-game win probability for a
// team's current game by re-simulating the remaining regulation time with the
// existing Poisson goal-rate model. If the team has no live game right now,
// it returns a clear JSON message (not an HTTP error).
//
// GET /api/live-win-probability?team=<TEAM_CODE>
func HandleLiveWinProbability(w http.ResponseWriter, r *http.Request) {
	teamCode := strings.ToUpper(r.URL.Query().Get("team"))
	if teamCode == "" && teamConfig != nil {
		teamCode = teamConfig.Code
	}
	if teamCode == "" {
		http.Error(w, "Missing team parameter (and no default team configured)", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	game, err := services.GetTeamScoreboard(teamCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch scoreboard for %s: %v", teamCode, err), http.StatusInternalServerError)
		return
	}

	// "CRIT" is the NHL API's game state for a close game in its final
	// minutes -- still in progress for our purposes, just like "LIVE".
	isLive := game.GameID != 0 && (game.GameState == "LIVE" || game.GameState == "CRIT")
	if !isLive {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"live":    false,
			"team":    teamCode,
			"message": fmt.Sprintf("No live game currently in progress for %s (state: %q)", teamCode, game.GameState),
		})
		return
	}

	periodTimeRemaining := time.Duration(game.Clock.SecondsRemaining) * time.Second

	result, err := services.CalculateLiveWinProbability(
		game.HomeTeam.Abbrev,
		game.AwayTeam.Abbrev,
		game.HomeTeam.Score,
		game.AwayTeam.Score,
		game.Period,
		periodTimeRemaining,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to calculate live win probability: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"live":   true,
		"gameId": game.GameID,
		"result": result,
	})
}
