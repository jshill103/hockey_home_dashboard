package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// Global variables for player stats caching
var (
	cachedTeamPlayerStats *models.PlayerStatsLeaders
	cachedTeamGoalieStats *models.GoalieStatsLeaders
)

// InitPlayerStats initializes player stats cache with shared state
func InitPlayerStats(
	playerStats *models.PlayerStatsLeaders,
	goalieStats *models.GoalieStatsLeaders,
	teamPlayerStats *models.PlayerStatsLeaders,
	teamGoalieStats *models.GoalieStatsLeaders,
) {
	cachedTeamPlayerStats = teamPlayerStats
	cachedTeamGoalieStats = teamGoalieStats
}

// HandlePlayerStats returns Utah Hockey Club player statistics
func HandlePlayerStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if we have cached data
	if cachedTeamPlayerStats == nil || len(cachedTeamPlayerStats.Goals) == 0 {
		// Try to fetch fresh data
		playerLeaders, err := services.GetPlayerStatsLeaders()
		if err != nil {
			fmt.Fprintf(w, `<div class="error">Unable to load player stats: %v</div>`, err)
			return
		}
		teamCode := "UTA"
		if teamConfig != nil {
			teamCode = teamConfig.Code
		}
		teamStats := services.GetTeamPlayerStats(playerLeaders, teamCode)
		*cachedTeamPlayerStats = teamStats
	}

	html := formatPlayerStatsHTML(*cachedTeamPlayerStats)
	fmt.Fprint(w, html)
}

// HandleGoalieStats returns Utah Hockey Club goalie statistics
func HandleGoalieStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if we have cached data
	if cachedTeamGoalieStats == nil || len(cachedTeamGoalieStats.Wins) == 0 {
		// Try to fetch fresh data
		goalieLeaders, err := services.GetGoalieStatsLeaders()
		if err != nil {
			fmt.Fprintf(w, `<div class="error">Unable to load goalie stats: %v</div>`, err)
			return
		}
		teamCode := "UTA"
		if teamConfig != nil {
			teamCode = teamConfig.Code
		}
		teamStats := services.GetTeamGoalieStats(goalieLeaders, teamCode)
		*cachedTeamGoalieStats = teamStats
	}

	html := formatGoalieStatsHTML(*cachedTeamGoalieStats)
	fmt.Fprint(w, html)
}

// HandlePlayerStatsJSON returns player stats as JSON for API access
func HandlePlayerStatsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"playerStats": cachedTeamPlayerStats,
		"goalieStats": cachedTeamGoalieStats,
		"teamConfig":  teamConfig,
		"timestamp":   "now", // You can add proper timestamp
	}

	json.NewEncoder(w).Encode(response)
}

// formatPlayerStatsHTML formats player statistics into HTML
func formatPlayerStatsHTML(stats models.PlayerStatsLeaders) string {
	html := `<div class="player-stats-content">`

	if len(stats.Goals) == 0 && len(stats.Assists) == 0 && len(stats.Points) == 0 {
		html += `<div class="no-stats">No Utah Hockey Club players currently in NHL stat leaders during off-season.</div>`
		html += `</div>`
		return html
	}

	// Goals leaders
	if len(stats.Goals) > 0 {
		html += `<div class="stat-category">
			<h4>🥅 Goals Leaders</h4>
			<div class="player-grid">`

		for _, player := range stats.Goals {
			html += fmt.Sprintf(`
				<div class="player-card">
					<div class="player-info">
						<div class="player-name">%s %s</div>
						<div class="player-position">#%d • %s</div>
					</div>
					<div class="player-stats">
						<div class="stat-value">%d</div>
						<div class="stat-label">Goals</div>
					</div>
				</div>`,
				player.FirstName.Default,
				player.LastName.Default,
				0, // We'd need jersey number from roster
				player.Position,
				player.Goals)
		}
		html += `</div></div>`
	}

	// Assists leaders
	if len(stats.Assists) > 0 {
		html += `<div class="stat-category">
			<h4>🎯 Assists Leaders</h4>
			<div class="player-grid">`

		for _, player := range stats.Assists {
			html += fmt.Sprintf(`
				<div class="player-card">
					<div class="player-info">
						<div class="player-name">%s %s</div>
						<div class="player-position">#%d • %s</div>
					</div>
					<div class="player-stats">
						<div class="stat-value">%d</div>
						<div class="stat-label">Assists</div>
					</div>
				</div>`,
				player.FirstName.Default,
				player.LastName.Default,
				0, // We'd need jersey number from roster
				player.Position,
				player.Assists)
		}
		html += `</div></div>`
	}

	// Points leaders
	if len(stats.Points) > 0 {
		html += `<div class="stat-category">
			<h4>⭐ Points Leaders</h4>
			<div class="player-grid">`

		for _, player := range stats.Points {
			html += fmt.Sprintf(`
				<div class="player-card">
					<div class="player-info">
						<div class="player-name">%s %s</div>
						<div class="player-position">#%d • %s</div>
					</div>
					<div class="player-stats">
						<div class="stat-value">%d</div>
						<div class="stat-label">Points</div>
					</div>
				</div>`,
				player.FirstName.Default,
				player.LastName.Default,
				0, // We'd need jersey number from roster
				player.Position,
				player.Points)
		}
		html += `</div></div>`
	}

	html += `</div>`
	return html
}

// formatGoalieStatsHTML formats goalie statistics into HTML
func formatGoalieStatsHTML(stats models.GoalieStatsLeaders) string {
	html := `<div class="goalie-stats-content">`

	if len(stats.Wins) == 0 && len(stats.SavePct) == 0 && len(stats.GAA) == 0 {
		html += `<div class="no-stats">No Utah Hockey Club goalies currently in NHL stat leaders during off-season.</div>`
		html += `</div>`
		return html
	}

	// Wins leaders
	if len(stats.Wins) > 0 {
		html += `<div class="stat-category">
			<h4>🏆 Wins Leaders</h4>
			<div class="goalie-grid">`

		for _, goalie := range stats.Wins {
			html += fmt.Sprintf(`
				<div class="goalie-card">
					<div class="goalie-info">
						<div class="goalie-name">%s %s</div>
						<div class="goalie-position">Goaltender</div>
					</div>
					<div class="goalie-stats">
						<div class="stat-value">%d</div>
						<div class="stat-label">Wins</div>
					</div>
				</div>`,
				goalie.FirstName.Default,
				goalie.LastName.Default,
				goalie.Wins)
		}
		html += `</div></div>`
	}

	// Save percentage leaders
	if len(stats.SavePct) > 0 {
		html += `<div class="stat-category">
			<h4>🛡️ Save % Leaders</h4>
			<div class="goalie-grid">`

		for _, goalie := range stats.SavePct {
			html += fmt.Sprintf(`
				<div class="goalie-card">
					<div class="goalie-info">
						<div class="goalie-name">%s %s</div>
						<div class="goalie-position">Goaltender</div>
					</div>
					<div class="goalie-stats">
						<div class="stat-value">%.3f</div>
						<div class="stat-label">Save %%</div>
					</div>
				</div>`,
				goalie.FirstName.Default,
				goalie.LastName.Default,
				goalie.SavePct)
		}
		html += `</div></div>`
	}

	// Goals Against Average leaders
	if len(stats.GAA) > 0 {
		html += `<div class="stat-category">
			<h4>🥅 GAA Leaders</h4>
			<div class="goalie-grid">`

		for _, goalie := range stats.GAA {
			html += fmt.Sprintf(`
				<div class="goalie-card">
					<div class="goalie-info">
						<div class="goalie-name">%s %s</div>
						<div class="goalie-position">Goaltender</div>
					</div>
					<div class="goalie-stats">
						<div class="stat-value">%.2f</div>
						<div class="stat-label">GAA</div>
					</div>
				</div>`,
				goalie.FirstName.Default,
				goalie.LastName.Default,
				goalie.GoalsAgainstAvg)
		}
		html += `</div></div>`
	}

	html += `</div>`
	return html
}
