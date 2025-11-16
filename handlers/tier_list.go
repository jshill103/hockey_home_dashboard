package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleTierListPopup generates the ML-powered NHL tier list popup
func HandleTierListPopup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// Get tier ranking service
	tierService := services.GetTeamTierRankingService()
	if tierService == nil {
		http.Error(w, "Tier ranking service not available", http.StatusServiceUnavailable)
		return
	}

	// Generate tier list
	tierList, err := tierService.GenerateTierList()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate tier list: %v", err), http.StatusInternalServerError)
		return
	}

	html := generateTierListHTML(tierList)
	w.Write([]byte(html))
}

// generateTierListHTML creates the popup HTML for the tier list
func generateTierListHTML(tierList *services.TierList) string {
	var html strings.Builder

	// Tier colors and descriptions
	tierInfo := map[string]struct {
		Color       string
		GradientStart string
		GradientEnd string
		Description string
		Icon        string
	}{
		"S": {"#FFD700", "#FFD700", "#FFA500", "Elite Championship Contenders", "👑"},
		"A": {"#C0C0C0", "#E8E8E8", "#C0C0C0", "Strong Playoff Teams", "⭐"},
		"B": {"#CD7F32", "#DAA520", "#CD7F32", "Playoff Bubble / Wild Card", "🔥"},
		"C": {"#4A90E2", "#6CA6E8", "#4A90E2", "Developing / Fringe Teams", "📈"},
		"D": {"#8B4513", "#A0522D", "#8B4513", "Rebuilding / Lottery", "🔧"},
	}

	html.WriteString(`
	<div class="system-stats-popup" style="max-width: 95%; width: 1400px;">
		<div class="stats-header" style="background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);">
			<h3>🏆 NHL Power Rankings - ML Tier List</h3>
			<button class="close-popup" onclick="closeTierListPopup()">✕</button>
		</div>
		<div class="stats-section" style="padding: 20px;">
			<div style="text-align: center; margin-bottom: 20px; padding: 15px; background: rgba(255,255,255,0.05); border-radius: 10px;">
				<div style="font-size: 0.9em; color: #aaa;">Season: ` + tierList.Season + `</div>
				<div style="font-size: 0.85em; color: #888; margin-top: 5px;">`)
	html.WriteString(tierList.Methodology)
	html.WriteString(`</div>
				<div style="font-size: 0.8em; color: #666; margin-top: 8px; font-style: italic;">
					Rankings use: Elo (25%), Recent Form (20%), Poisson Model (20%), Goal Diff (15%), Point % (15%), Playoff Position (10%), Road Strength (10%)
				</div>
			</div>
	`)

	// Render each tier
	tierOrder := []string{"S", "A", "B", "C", "D"}
	for _, tier := range tierOrder {
		teams := tierList.Tiers[tier]
		if len(teams) == 0 {
			continue
		}

		info := tierInfo[tier]
		html.WriteString(fmt.Sprintf(`
			<div style="margin-bottom: 25px;">
				<div style="background: linear-gradient(90deg, %s, %s); padding: 12px 20px; border-radius: 10px 10px 0 0; display: flex; justify-content: space-between; align-items: center;">
					<div>
						<span style="font-size: 1.8em; font-weight: bold; color: #000;">%s %s Tier</span>
						<span style="font-size: 0.9em; color: #333; margin-left: 15px;">%s</span>
					</div>
					<div style="font-size: 1.2em; color: #333; font-weight: bold;">%d Teams</div>
				</div>
				<div style="background: rgba(0,0,0,0.3); padding: 15px; border-radius: 0 0 10px 10px; display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px;">
		`, info.GradientStart, info.GradientEnd, info.Icon, tier, info.Description, len(teams)))

		// Render each team in the tier
		for i, team := range teams {
			rank := 0
			for j, t := range tierOrder {
				if t == tier {
					for k := 0; k < j; k++ {
						rank += len(tierList.Tiers[tierOrder[k]])
					}
					rank += i + 1
					break
				}
			}

			// Team card
			html.WriteString(fmt.Sprintf(`
				<div style="background: rgba(255,255,255,0.08); padding: 12px; border-radius: 8px; border-left: 4px solid %s; transition: all 0.3s; cursor: pointer;" 
				     onmouseover="this.style.background='rgba(255,255,255,0.15)'; this.style.transform='translateX(5px)';" 
				     onmouseout="this.style.background='rgba(255,255,255,0.08)'; this.style.transform='translateX(0)';">
					<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
						<div>
							<span style="font-weight: bold; font-size: 1.1em; color: white;">#%d %s</span>
							<span style="font-size: 0.85em; color: #aaa; margin-left: 8px;">%s</span>
						</div>
						<div style="background: %s; color: #000; padding: 2px 8px; border-radius: 12px; font-size: 0.75em; font-weight: bold;">
							%.1f
						</div>
					</div>
					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 6px; font-size: 0.85em;">
						<div style="color: #ddd;">Record: <span style="color: white; font-weight: 600;">%s</span></div>
						<div style="color: #ddd;">Points: <span style="color: white; font-weight: 600;">%d</span></div>
						<div style="color: #ddd;">Win%%: <span style="color: white; font-weight: 600;">%.1f%%</span></div>
						<div style="color: #ddd;">Elo: <span style="color: white; font-weight: 600;">%.0f</span></div>
					</div>
				</div>
			`, info.Color, rank, team.TeamCode, team.TeamName, info.Color, team.MLScore, 
			   team.Record, team.Points, team.PointPct*100, team.EloRating))
		}

		html.WriteString(`
				</div>
			</div>
		`)
	}

	html.WriteString(`
		</div>
		<div style="text-align: center; padding: 15px; background: rgba(0,0,0,0.2); font-size: 0.8em; color: #888;">
			<div>💡 Rankings update in real-time based on current season performance</div>
			<div style="margin-top: 5px;">Click outside or press ESC to close</div>
		</div>
	</div>
	`)

	return html.String()
}

// HandleTierListAPI returns tier list data as JSON
func HandleTierListAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tierService := services.GetTeamTierRankingService()
	if tierService == nil {
		http.Error(w, `{"error": "Tier ranking service not available"}`, http.StatusServiceUnavailable)
		return
	}

	tierList, err := tierService.GenerateTierList()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(tierList)
}

