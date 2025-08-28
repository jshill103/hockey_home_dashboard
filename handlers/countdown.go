package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleSeasonCountdown returns the countdown to the next NHL season
func HandleSeasonCountdown(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Only show countdown during off-season
	if currentSeasonStatus == nil || currentSeasonStatus.IsHockeySeason {
		fmt.Fprint(w, `<div class="no-countdown">Season countdown is only available during the off-season.</div>`)
		return
	}

	countdown, err := services.GetSeasonCountdown(*teamConfig)
	if err != nil {
		fmt.Fprintf(w, `<div class="countdown-error">Error loading season countdown: %v</div>`, err)
		return
	}

	if countdown.SeasonStarted {
		fmt.Fprint(w, `<div class="season-started">🏒 The 2025-26 NHL season has started!</div>`)
		return
	}

	html := formatCountdownHTML(countdown)
	fmt.Fprint(w, html)
}

// HandleSeasonCountdownJSON returns the countdown data as JSON
func HandleSeasonCountdownJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	countdown, err := services.GetSeasonCountdown(*teamConfig)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(countdown)
}

// formatCountdownHTML formats the countdown data into HTML
func formatCountdownHTML(countdown models.SeasonCountdown) string {
	var html strings.Builder

	html.WriteString(`<div class="countdown-container">`)

	// Main season countdown
	html.WriteString(`<div class="main-countdown">`)
	html.WriteString(`<div class="countdown-header">`)
	html.WriteString(`<h3>🏒 2025-26 NHL Season Countdown</h3>`)
	html.WriteString(`</div>`)

	html.WriteString(`<div class="countdown-main">`)
	html.WriteString(fmt.Sprintf(`<div class="days-counter" data-target-date="%s">`, countdown.FirstGameDate.Format("2006-01-02T15:04:05Z07:00")))
	html.WriteString(fmt.Sprintf(`<span class="days-number">%d</span>`, countdown.DaysUntilSeason))
	html.WriteString(`<span class="days-label">Days Until Season</span>`)
	html.WriteString(`</div>`)
	html.WriteString(`</div>`)

	// First game details
	html.WriteString(`<div class="first-game-details">`)
	html.WriteString(`<div class="game-info">`)
	html.WriteString(fmt.Sprintf(`<div class="game-teams">%s</div>`, countdown.FirstGameTeams))
	html.WriteString(fmt.Sprintf(`<div class="game-date">%s at %s</div>`, countdown.FirstGameFormatted, countdown.FirstGameTime))
	html.WriteString(fmt.Sprintf(`<div class="game-venue">%s</div>`, countdown.FirstGameVenue))
	html.WriteString(`</div>`)
	html.WriteString(`</div>`)
	html.WriteString(`</div>`)

	// Team's first game (if different and found)
	if !countdown.TeamFirstGameDate.IsZero() && countdown.DaysUntilTeamGame != countdown.DaysUntilSeason {
		html.WriteString(`<div class="uta-countdown">`)
		html.WriteString(`<div class="countdown-header">`)
		html.WriteString(fmt.Sprintf(`<h4>🏒 %s's Season Opener</h4>`, teamConfig.ShortName))
		html.WriteString(`</div>`)

		html.WriteString(`<div class="uta-countdown-main">`)
		html.WriteString(fmt.Sprintf(`<div class="uta-days-counter" data-target-date="%s">`, countdown.TeamFirstGameDate.Format("2006-01-02T15:04:05Z07:00")))
		html.WriteString(fmt.Sprintf(`<span class="uta-days-number">%d</span>`, countdown.DaysUntilTeamGame))
		html.WriteString(fmt.Sprintf(`<span class="uta-days-label">Days Until %s's First Game</span>`, teamConfig.ShortName))
		html.WriteString(`</div>`)
		html.WriteString(`</div>`)

		html.WriteString(`<div class="uta-game-details">`)
		html.WriteString(`<div class="uta-game-info">`)
		html.WriteString(fmt.Sprintf(`<div class="uta-game-teams">%s</div>`, countdown.TeamFirstGameTeams))
		html.WriteString(fmt.Sprintf(`<div class="uta-game-date">%s at %s</div>`, countdown.TeamFirstGameFormatted, countdown.TeamFirstGameTime))
		html.WriteString(fmt.Sprintf(`<div class="uta-game-venue">%s</div>`, countdown.TeamFirstGameVenue))
		html.WriteString(`</div>`)
		html.WriteString(`</div>`)
		html.WriteString(`</div>`)
	}

	html.WriteString(`</div>`)
	return html.String()
}
