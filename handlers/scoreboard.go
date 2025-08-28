package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleScoreboard handles scoreboard requests
func HandleScoreboard(w http.ResponseWriter, r *http.Request) {
	// Use cached scoreboard data
	scoreboard := *cachedScoreboard

	// If no cached data, try to fetch fresh data as fallback
	if scoreboard.GameID == 0 {
		fmt.Println("No cached scoreboard data, fetching fresh data...")
		var err error
		scoreboard, err = services.GetTeamScoreboard(teamConfig.Code)
		if err != nil {
			w.Write([]byte("<p>Error fetching scoreboard: " + err.Error() + "</p>"))
			return
		}
		// Update cache
		*cachedScoreboard = scoreboard
	}

	if scoreboard.GameID == 0 {
		w.Write([]byte("<p>No active game at the moment.</p>"))
		return
	}

	html := formatScoreboardHTML(scoreboard)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func formatScoreboardHTML(scoreboard models.ScoreboardGame) string {
	// If no game data, return placeholder
	if scoreboard.GameID == 0 {
		return "<p>🏒 Live Scoreboard: No active game</p>"
	}

	var html strings.Builder

	html.WriteString("<div class='scoreboard' style='")
	html.WriteString("background: linear-gradient(135deg, #333, #555); ")
	html.WriteString("padding: 15px; border-radius: 8px; margin: 10px 0; ")
	html.WriteString("color: white; text-align: center; box-shadow: 0 4px 8px rgba(0,0,0,0.2);'>")

	html.WriteString("<h3 style='margin: 0 0 10px 0; font-size: 1.2em;'>🏒 Live Scoreboard</h3>")

	// Display teams and scores
	html.WriteString("<div style='display: flex; justify-content: space-between; align-items: center; margin: 15px 0;'>")

	// Away team
	html.WriteString("<div style='text-align: left; flex: 1;'>")
	html.WriteString("<div style='font-weight: bold; font-size: 1.1em;'>")
	html.WriteString(scoreboard.AwayTeam.Name.Default)
	html.WriteString("</div>")
	html.WriteString("<div style='font-size: 1.5em; font-weight: bold; color: #ff6b35;'>")
	html.WriteString(fmt.Sprintf("%d", scoreboard.AwayTeam.Score))
	html.WriteString("</div>")
	html.WriteString("</div>")

	// VS separator
	html.WriteString("<div style='text-align: center; padding: 0 20px;'>")
	html.WriteString("<div style='font-size: 0.8em; color: #ccc;'>VS</div>")
	html.WriteString("</div>")

	// Home team
	html.WriteString("<div style='text-align: right; flex: 1;'>")
	html.WriteString("<div style='font-weight: bold; font-size: 1.1em;'>")
	html.WriteString(scoreboard.HomeTeam.Name.Default)
	html.WriteString("</div>")
	html.WriteString("<div style='font-size: 1.5em; font-weight: bold; color: #ff6b35;'>")
	html.WriteString(fmt.Sprintf("%d", scoreboard.HomeTeam.Score))
	html.WriteString("</div>")
	html.WriteString("</div>")

	html.WriteString("</div>")

	// Game state and period info
	html.WriteString("<div style='font-size: 0.9em; margin: 10px 0; color: #ccc;'>")

	// Game state
	gameStateDisplay := scoreboard.GameState
	switch scoreboard.GameState {
	case "LIVE":
		gameStateDisplay = "🔴 LIVE"
	case "FUT":
		gameStateDisplay = "⏱️ UPCOMING"
	case "FINAL":
		gameStateDisplay = "✅ FINAL"
	case "OFF":
		gameStateDisplay = "⏸️ INTERMISSION"
	}

	html.WriteString("<div>" + gameStateDisplay + "</div>")

	// Period and time info
	if scoreboard.Period > 0 && scoreboard.PeriodTime != "" {
		html.WriteString("<div>")
		html.WriteString(fmt.Sprintf("Period %d - %s", scoreboard.Period, scoreboard.PeriodTime))
		html.WriteString("</div>")
	}

	html.WriteString("</div>")

	// Shots on goal
	if scoreboard.AwayTeam.Shots > 0 || scoreboard.HomeTeam.Shots > 0 {
		html.WriteString("<div style='font-size: 0.8em; margin: 10px 0; color: #aaa;'>")
		html.WriteString(fmt.Sprintf("Shots: %s %d - %d %s",
			scoreboard.AwayTeam.Abbrev,
			scoreboard.AwayTeam.Shots,
			scoreboard.HomeTeam.Shots,
			scoreboard.HomeTeam.Abbrev))
		html.WriteString("</div>")
	}

	html.WriteString("</div>")

	return html.String()
}
