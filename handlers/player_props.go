package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jaredshillingburg/go_uhc/services"
)

// GetPlayerProps returns Poisson-distribution-based prop projections (goals,
// assists, points, shots-on-goal) for an individual player against a
// specific opponent, including P(over) for common sportsbook-style lines.
//
// GET /api/player-props?playerId=<id>&opponent=<TEAM_CODE>[&team=<TEAM_CODE>]
//
// "team" (the player's own team) follows the same query-param convention as
// /api/goalie-matchup's "id"/"opponent" pair; it defaults to the configured
// primary team if omitted.
func GetPlayerProps(w http.ResponseWriter, r *http.Request) {
	playerIDStr := r.URL.Query().Get("playerId")
	opponent := strings.ToUpper(r.URL.Query().Get("opponent"))
	teamCode := strings.ToUpper(r.URL.Query().Get("team"))

	if playerIDStr == "" || opponent == "" {
		http.Error(w, "Missing required parameters: playerId and opponent", http.StatusBadRequest)
		return
	}

	if teamCode == "" && teamConfig != nil {
		teamCode = teamConfig.Code
	}
	if teamCode == "" {
		http.Error(w, "Missing team parameter (and no default team configured)", http.StatusBadRequest)
		return
	}

	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		http.Error(w, "Invalid playerId parameter", http.StatusBadRequest)
		return
	}

	playerService := services.GetPlayerImpactService()
	if playerService == nil {
		http.Error(w, "Player impact service not available", http.StatusServiceUnavailable)
		return
	}

	projection, err := playerService.GetPlayerPropProjection(teamCode, opponent, playerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to compute player prop projection: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projection)
}
