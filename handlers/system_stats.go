package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

var systemStatsService *services.SystemStatsService

// InitSystemStatsService initializes the system stats service
func InitSystemStatsService(service *services.SystemStatsService) {
	systemStatsService = service
}

// HandleSystemStats returns the system statistics page
func HandleSystemStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if systemStatsService == nil {
		http.Error(w, `{"error": "System stats service not initialized"}`, http.StatusInternalServerError)
		return
	}

	stats := systemStatsService.GetStats()
	json.NewEncoder(w).Encode(stats)
}

// HandleSystemStatsPopup returns the HTML for the stats popup
func HandleSystemStatsPopup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if systemStatsService == nil {
		fmt.Fprint(w, `<div class="stats-error">System stats unavailable</div>`)
		return
	}

	stats := systemStatsService.GetStats()

	html := generateStatsPopupHTML(stats)
	fmt.Fprint(w, html)
}

// generateStatsPopupHTML generates the HTML for the stats popup
func generateStatsPopupHTML(statsInterface interface{}) string {
	systemStats, ok := statsInterface.(models.SystemStats)
	if !ok {
		return `<div class="stats-error">Unable to load stats</div>`
	}

	html := `
<div class="system-stats-popup">
	<div class="stats-header">
		<h3>System Statistics</h3>
		<button class="close-popup" onclick="closeStatsPopup()">✕</button>
	</div>
	
	<div class="stats-section">
		<h4>📦 Backfill Status</h4>
		<div class="stats-grid">
			<div class="stat-item">
				<span class="stat-label">Games Processed:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", systemStats.BackfillStats.TotalGamesProcessed) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Play-by-Play:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", systemStats.BackfillStats.PlayByPlayGames) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Shift Data:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", systemStats.BackfillStats.ShiftDataGames) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Game Summaries:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", systemStats.BackfillStats.GameSummaryGames) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Failed:</span>
				<span class="stat-value stat-warning">` + fmt.Sprintf("%d", systemStats.BackfillStats.FailedGames) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Events Processed:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", systemStats.BackfillStats.TotalEventsProcessed) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Avg Processing:</span>
				<span class="stat-value">` + fmt.Sprintf("%.2fms", systemStats.BackfillStats.ProcessingTimeAvg) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Last Backfill:</span>
				<span class="stat-value">` + formatTimeAgo(systemStats.BackfillStats.LastBackfillTime) + `</span>
			</div>
		</div>
	</div>

	<div class="stats-section">
		<h4>🎯 Prediction Accuracy</h4>
		<div class="stats-grid">
			<div class="stat-item stat-highlight">
				<span class="stat-label">Total Predictions:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", systemStats.PredictionStats.TotalPredictions) + `</span>
			</div>
			<div class="stat-item stat-highlight">
				<span class="stat-label">Correct:</span>
				<span class="stat-value stat-success">` + fmt.Sprintf("%d", systemStats.PredictionStats.CorrectPredictions) + `</span>
			</div>
			<div class="stat-item stat-highlight">
				<span class="stat-label">Overall Accuracy:</span>
				<span class="stat-value stat-success">` + fmt.Sprintf("%.1f%%", systemStats.PredictionStats.OverallAccuracy) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Best Model:</span>
				<span class="stat-value">` + systemStats.PredictionStats.BestPerformingModel + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Last Prediction:</span>
				<span class="stat-value">` + formatTimeAgo(systemStats.PredictionStats.LastPredictionTime) + `</span>
			</div>
		</div>
	</div>

	<div class="stats-section">
		<h4>🤖 Model Performance</h4>
		<div class="model-stats">
`

	// Add model accuracy details
	for modelName, modelData := range systemStats.PredictionStats.ModelAccuracy {
		if modelData == nil {
			continue
		}

		if modelData.TotalPredictions > 0 {
			html += `
			<div class="model-stat-row">
				<span class="model-name">` + modelName + `</span>
				<div class="model-accuracy-bar">
					<div class="accuracy-fill" style="width: ` + fmt.Sprintf("%.1f%%", modelData.Accuracy) + `"></div>
				</div>
				<span class="model-accuracy">` + fmt.Sprintf("%.1f%%", modelData.Accuracy) + `</span>
				<span class="model-count">` + fmt.Sprintf("(%d/%d)", modelData.CorrectPredictions, modelData.TotalPredictions) + `</span>
			</div>
`
		}
	}

	html += `
		</div>
	</div>

	<div class="stats-section">
		<h4>⚙️ System Info</h4>
		<div class="stats-grid">
			<div class="stat-item">
				<span class="stat-label">Uptime:</span>
				<span class="stat-value">` + formatDuration(systemStats.SystemUptime) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">API Requests:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", systemStats.TotalAPIRequests) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Last Updated:</span>
				<span class="stat-value">` + formatTimeAgo(systemStats.LastUpdated) + `</span>
			</div>
		</div>
	</div>
</div>
`

	return html
}

// formatTimeAgo formats a time as "X ago"
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	} else {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}
