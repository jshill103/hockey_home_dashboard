package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleSchedule handles schedule requests
func HandleSchedule(w http.ResponseWriter, r *http.Request) {
	// Check if we have cached data first
	if cachedSchedule.GameDate != "" {
		html := formatBannerHTML(*cachedSchedule)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}

	// If no cached data, fetch fresh data
	game, err := services.GetTeamSchedule(teamConfig.Code)
	if err != nil {
		w.Write([]byte("<p>Error fetching schedule: " + err.Error() + "</p>"))
		return
	}

	// Cache the result
	*cachedSchedule = game

	if game.GameDate == "" {
		w.Write([]byte("<p>No upcoming games found for Utah this week.</p>"))
		return
	}

	html := formatBannerHTML(game)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// HandleBanner handles banner requests
func HandleBanner(w http.ResponseWriter, r *http.Request) {
	// Check if we have cached schedule data
	if cachedSchedule.GameDate == "" {
		// If no cached data, try to fetch fresh data
		fmt.Println("No cached schedule data, fetching fresh data...")
		game, err := services.GetTeamSchedule(teamConfig.Code)
		if err != nil {
			w.Write([]byte("<p>Error fetching banner data: " + err.Error() + "</p>"))
			return
		}
		*cachedSchedule = game
	}

	if cachedSchedule.GameDate == "" {
		w.Write([]byte("<p>No upcoming games scheduled for Utah this week.</p>"))
		return
	}

	html := formatBannerHTML(*cachedSchedule)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func formatBannerHTML(game models.Game) string {
	// If no game data, return loading message
	if game.GameDate == "" || game.HomeTeam.CommonName.Default == "" {
		return "<span style='color: white; font-size: 1.1em; font-weight: bold;'>🏒 Loading next game information...</span>"
	}

	// Format as a compact horizontal ticker
	var html strings.Builder

	html.WriteString("<span style='color: white; font-size: 1.1em; font-weight: bold; text-shadow: 2px 2px 4px rgba(0,0,0,0.3);'>")

	// Compact game info: NEXT GAME: (AWAY) Away @ (HOME) Home • Date • Time • Venue • TV
	html.WriteString("🏒 NEXT GAME: ")
	html.WriteString("<span class='away-indicator'>AWAY</span>")
	html.WriteString(game.AwayTeam.CommonName.Default)
	html.WriteString(" @ ")
	html.WriteString("<span class='home-indicator'>HOME</span>")
	html.WriteString(game.HomeTeam.CommonName.Default)

	// Add date/time
	if game.FormattedTime != "" {
		html.WriteString(" • ")
		html.WriteString(game.FormattedTime)
		html.WriteString(" MT")
	}

	// Add venue
	if game.Venue.Default != "" {
		html.WriteString(" • ")
		html.WriteString(game.Venue.Default)
	}

	// Add TV info
	if len(game.Broadcasts) > 0 {
		html.WriteString(" • TV: ")
		networks := make([]string, len(game.Broadcasts))
		for i, broadcast := range game.Broadcasts {
			networks[i] = broadcast.Network
		}
		html.WriteString(strings.Join(networks, ", "))
	} else {
		html.WriteString(" • TV: TBA")
	}

	html.WriteString("</span>")

	return html.String()
}

// HandleUpcomingGames handles upcoming games requests
func HandleUpcomingGames(w http.ResponseWriter, r *http.Request) {
	// Use cached upcoming games data
	games := *cachedUpcomingGames

	// If no cached data, try to fetch fresh data as fallback
	if len(games) == 0 {
		fmt.Println("No cached upcoming games data, fetching fresh data...")
		var err error
		games, err = services.GetTeamUpcomingGames(teamConfig.Code)
		if err != nil {
			w.Write([]byte("<p>Error fetching upcoming games: " + err.Error() + "</p>"))
			return
		}
		// Update cache
		*cachedUpcomingGames = games
	}

	if len(games) == 0 {
		w.Write([]byte("<p>No upcoming games found in the next 7 days.</p>"))
		return
	}

	html := formatUpcomingGamesHTML(games)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func formatUpcomingGamesHTML(games []models.Game) string {
	var html strings.Builder

	html.WriteString("<div class='upcoming-games' style='")
	html.WriteString("background: rgba(255, 255, 255, 0.1); ")
	html.WriteString("padding: 15px; border-radius: 8px; margin: 10px 0;'>")

	html.WriteString("<h4 style='color: #ff6b35; margin: 0 0 15px 0;'>")
	html.WriteString(fmt.Sprintf("🗓️ Upcoming Games (%d)", len(games)))
	html.WriteString("</h4>")

	for i, game := range games {
		if i >= 5 { // Limit to first 5 games
			break
		}

		html.WriteString("<div style='")
		html.WriteString("background: rgba(255, 255, 255, 0.1); ")
		html.WriteString("padding: 10px; margin: 8px 0; border-radius: 5px; ")
		html.WriteString("border-left: 3px solid #ff6b35;'>")

		html.WriteString("<div style='font-weight: bold;'>")
		html.WriteString(fmt.Sprintf("%s vs %s",
			game.AwayTeam.CommonName.Default,
			game.HomeTeam.CommonName.Default))
		html.WriteString("</div>")

		if game.FormattedTime != "" {
			html.WriteString("<div style='font-size: 0.9em; color: #ccc; margin-top: 5px;'>")
			html.WriteString("📅 " + game.FormattedTime)
			html.WriteString("</div>")
		}

		html.WriteString("</div>")
	}

	if len(games) > 5 {
		html.WriteString("<div style='text-align: center; font-style: italic; margin-top: 10px;'>")
		html.WriteString(fmt.Sprintf("... and %d more games", len(games)-5))
		html.WriteString("</div>")
	}

	html.WriteString("</div>")

	return html.String()
}
