package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleLineup handles the lineup API endpoint
func HandleLineup(w http.ResponseWriter, r *http.Request) {
	// Get gameID from query parameter
	gameIDStr := r.URL.Query().Get("gameId")
	if gameIDStr == "" {
		http.Error(w, "gameId parameter required", http.StatusBadRequest)
		return
	}

	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, "invalid gameId", http.StatusBadRequest)
		return
	}

	lineupService := services.GetPreGameLineupService()
	if lineupService == nil {
		http.Error(w, "lineup service not initialized", http.StatusInternalServerError)
		return
	}

	lineup, err := lineupService.GetLineup(gameID)
	if err != nil {
		http.Error(w, fmt.Sprintf("lineup not available: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lineup)
}

// HandleLineupHTML handles the HTML formatted lineup display
func HandleLineupHTML(w http.ResponseWriter, r *http.Request) {
	// Get gameID from query parameter
	gameIDStr := r.URL.Query().Get("gameId")
	if gameIDStr == "" {
		http.Error(w, "gameId parameter required", http.StatusBadRequest)
		return
	}

	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, "invalid gameId", http.StatusBadRequest)
		return
	}

	lineupService := services.GetPreGameLineupService()
	if lineupService == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<p>Lineup service not available</p>"))
		return
	}

	lineup, err := lineupService.GetLineup(gameID)
	if err != nil || !lineup.IsAvailable {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<p>📋 Pre-game lineup not yet available. Check back closer to game time!</p>"))
		return
	}

	// Format lineup as HTML
	html := formatLineupHTML(lineup)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// formatLineupHTML formats the lineup data as HTML
func formatLineupHTML(lineup *models.PreGameLineup) string {
	var html strings.Builder

	html.WriteString("<div class='lineup-container'>")

	// Header
	html.WriteString(fmt.Sprintf("<div class='lineup-header'>"))
	html.WriteString(fmt.Sprintf("<h3>🏒 %s @ %s</h3>", lineup.AwayTeam, lineup.HomeTeam))
	html.WriteString(fmt.Sprintf("<div class='lineup-date'>%s</div>", lineup.GameDate.Format("January 2, 2006 at 3:04 PM MST")))
	html.WriteString("</div>")

	// Two-column layout for both teams
	html.WriteString("<div class='lineup-teams'>")

	// Away Team
	if lineup.AwayLineup != nil {
		html.WriteString("<div class='lineup-team'>")
		html.WriteString(fmt.Sprintf("<h4>%s (Away)</h4>", lineup.AwayTeam))
		html.WriteString(formatTeamLineupHTML(lineup.AwayLineup))
		html.WriteString("</div>")
	}

	// Home Team
	if lineup.HomeLineup != nil {
		html.WriteString("<div class='lineup-team'>")
		html.WriteString(fmt.Sprintf("<h4>%s (Home)</h4>", lineup.HomeTeam))
		html.WriteString(formatTeamLineupHTML(lineup.HomeLineup))
		html.WriteString("</div>")
	}

	html.WriteString("</div>") // lineup-teams

	html.WriteString("</div>") // lineup-container

	return html.String()
}

// formatTeamLineupHTML formats a single team's lineup
func formatTeamLineupHTML(teamLineup *models.TeamLineup) string {
	var html strings.Builder

	// Goalies
	html.WriteString("<div class='lineup-section'>")
	html.WriteString("<div class='lineup-section-title'>🥅 Goalies</div>")

	if teamLineup.StartingGoalie != nil {
		html.WriteString(fmt.Sprintf("<div class='lineup-player starter'>"))
		html.WriteString(fmt.Sprintf("⭐ #%d %s <span class='starter-badge'>STARTER</span>",
			teamLineup.StartingGoalie.SweaterNumber,
			teamLineup.StartingGoalie.PlayerName))
		html.WriteString("</div>")
	}

	if teamLineup.BackupGoalie != nil {
		html.WriteString(fmt.Sprintf("<div class='lineup-player'>"))
		html.WriteString(fmt.Sprintf("#%d %s",
			teamLineup.BackupGoalie.SweaterNumber,
			teamLineup.BackupGoalie.PlayerName))
		html.WriteString("</div>")
	}

	html.WriteString("</div>")

	// Forward Lines
	if len(teamLineup.ForwardLines) > 0 {
		html.WriteString("<div class='lineup-section'>")
		html.WriteString("<div class='lineup-section-title'>⚔️ Forward Lines</div>")

		for _, line := range teamLineup.ForwardLines {
			html.WriteString(fmt.Sprintf("<div class='lineup-line'>"))
			html.WriteString(fmt.Sprintf("<div class='line-number'>Line %d</div>", line.LineNumber))
			html.WriteString("<div class='line-players'>")

			if line.LeftWing != nil {
				html.WriteString(fmt.Sprintf("<span>#%d %s (LW)</span>", line.LeftWing.SweaterNumber, line.LeftWing.PlayerName))
			}
			if line.Center != nil {
				html.WriteString(fmt.Sprintf("<span>#%d %s (C)</span>", line.Center.SweaterNumber, line.Center.PlayerName))
			}
			if line.RightWing != nil {
				html.WriteString(fmt.Sprintf("<span>#%d %s (RW)</span>", line.RightWing.SweaterNumber, line.RightWing.PlayerName))
			}

			html.WriteString("</div>")
			html.WriteString("</div>")
		}

		html.WriteString("</div>")
	}

	// Defense Pairs
	if len(teamLineup.DefensePairs) > 0 {
		html.WriteString("<div class='lineup-section'>")
		html.WriteString("<div class='lineup-section-title'>🛡️ Defense Pairs</div>")

		for _, pair := range teamLineup.DefensePairs {
			html.WriteString(fmt.Sprintf("<div class='lineup-line'>"))
			html.WriteString(fmt.Sprintf("<div class='line-number'>Pair %d</div>", pair.PairNumber))
			html.WriteString("<div class='line-players'>")

			if pair.LeftDefense != nil {
				html.WriteString(fmt.Sprintf("<span>#%d %s (LD)</span>", pair.LeftDefense.SweaterNumber, pair.LeftDefense.PlayerName))
			}
			if pair.RightDefense != nil {
				html.WriteString(fmt.Sprintf("<span>#%d %s (RD)</span>", pair.RightDefense.SweaterNumber, pair.RightDefense.PlayerName))
			}

			html.WriteString("</div>")
			html.WriteString("</div>")
		}

		html.WriteString("</div>")
	}

	// Scratches
	if len(teamLineup.Scratches) > 0 {
		html.WriteString("<div class='lineup-section scratches'>")
		html.WriteString("<div class='lineup-section-title'>⚠️ Scratches</div>")

		for _, scratch := range teamLineup.Scratches {
			html.WriteString(fmt.Sprintf("<div class='lineup-player scratched'>"))
			html.WriteString(fmt.Sprintf("#%d %s (%s)",
				scratch.PlayerID, // Using PlayerID as we don't have SweaterNumber
				scratch.PlayerName,
				scratch.Position))
			if scratch.Reason != "" {
				html.WriteString(fmt.Sprintf(" - %s", scratch.Reason))
			}
			html.WriteString("</div>")
		}

		html.WriteString("</div>")
	}

	return html.String()
}
