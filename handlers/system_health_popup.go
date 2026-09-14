package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleSystemHealthPopup returns the HTML for the system health popup, giving a
// single at-a-glance view of every background model/service the dashboard depends on.
func HandleSystemHealthPopup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	healthService := services.GetHealthCheckService()
	status := healthService.RunHealthChecks()

	fmt.Fprint(w, generateHealthPopupHTML(status))
}

// healthCheckOrder pins the display order of well-known checks; anything else
// reported by the health service is appended after these in map order.
var healthCheckOrder = []string{"tier1_features", "model_accuracy", "nhl_api", "ml_models", "data_persistence", "cache", "api_cache", "memory"}

var healthCheckIcons = map[string]string{
	"tier1_features":   "🎯",
	"model_accuracy":   "📈",
	"nhl_api":          "🌐",
	"ml_models":        "🤖",
	"data_persistence": "💾",
	"cache":            "🗂️",
	"api_cache":        "📦",
	"memory":           "🧠",
}

var healthStatusLabels = map[string]string{
	"healthy":   "🟢 Healthy",
	"degraded":  "🟡 Degraded",
	"unhealthy": "🔴 Unhealthy",
}

func healthStatusBadge(status string) string {
	label, ok := healthStatusLabels[status]
	if !ok {
		label = "⚪ Unknown"
		status = "unknown"
	}
	return fmt.Sprintf(`<span class="health-badge %s">%s</span>`, status, label)
}

// generateHealthPopupHTML renders the overall health status plus a card per
// component check (NHL API, ML models, persistence, caches, memory).
func generateHealthPopupHTML(status *models.HealthStatus) string {
	if status == nil {
		return `<div class="system-stats-popup health-popup"><div class="stats-error">Unable to load system health</div></div>`
	}

	var b strings.Builder

	b.WriteString(`<div class="system-stats-popup health-popup">`)
	b.WriteString(`<div class="stats-header"><h3>🩺 System Health</h3><button class="close-popup" onclick="closeSystemHealthPopup()">✕</button></div>`)

	b.WriteString(`<div class="stats-section health-overall">`)
	b.WriteString(`<div class="health-overall-row"><span class="stat-label">Overall Status:</span>`)
	b.WriteString(healthStatusBadge(status.Status))
	b.WriteString(`</div>`)
	b.WriteString(`<div class="stats-grid">`)
	b.WriteString(`<div class="stat-item"><span class="stat-label">Uptime:</span><span class="stat-value">` + status.Uptime + `</span></div>`)
	version := status.Version
	if version == "" {
		version = "unknown"
	}
	b.WriteString(`<div class="stat-item"><span class="stat-label">Version:</span><span class="stat-value">` + version + `</span></div>`)
	b.WriteString(`<div class="stat-item"><span class="stat-label">Checked:</span><span class="stat-value">` + formatTimeAgo(status.Timestamp) + `</span></div>`)
	b.WriteString(`</div></div>`)

	b.WriteString(`<div class="stats-section"><h4>🔍 Component Checks</h4><div class="health-check-grid">`)

	orderedKeys := append([]string{}, healthCheckOrder...)
	for k := range status.Checks {
		found := false
		for _, existing := range orderedKeys {
			if existing == k {
				found = true
				break
			}
		}
		if !found {
			orderedKeys = append(orderedKeys, k)
		}
	}

	for _, key := range orderedKeys {
		check, ok := status.Checks[key]
		if !ok {
			continue
		}

		icon := healthCheckIcons[key]
		if icon == "" {
			icon = "🔧"
		}

		b.WriteString(`<div class="health-check-card ` + check.Status + `">`)
		b.WriteString(`<div class="health-check-title">` + icon + ` ` + check.Name + `</div>`)
		b.WriteString(healthStatusBadge(check.Status))
		if check.Message != "" {
			b.WriteString(`<div class="health-check-message">` + check.Message + `</div>`)
		}
		if check.ResponseTime > 0 {
			b.WriteString(`<div class="health-check-meta">Response: ` + check.ResponseTime.String() + `</div>`)
		}
		if key == "model_accuracy" {
			b.WriteString(renderModelAccuracyDetails(check.Details))
		}
		b.WriteString(`<div class="health-check-meta">Checked ` + formatTimeAgo(check.LastChecked) + `</div>`)
		b.WriteString(`</div>`)
	}

	b.WriteString(`</div></div></div>`)

	return b.String()
}

// renderModelAccuracyDetails renders the per-model accuracy breakdown
// (populated by HealthCheckService.checkModelAccuracy's "per_model" detail)
// as a mini version of the model-stat-row table used elsewhere in the popup.
func renderModelAccuracyDetails(details interface{}) string {
	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return ""
	}
	perModel, ok := detailsMap["per_model"].(map[string]interface{})
	if !ok || len(perModel) == 0 {
		return ""
	}

	names := make([]string, 0, len(perModel))
	for name := range perModel {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString(`<div class="model-stats">`)
	for _, name := range names {
		entry, ok := perModel[name].(map[string]interface{})
		if !ok {
			continue
		}
		accuracy, _ := entry["accuracy"].(float64)
		total, _ := entry["totalPredictions"].(int)
		correct, _ := entry["correctPredictions"].(int)

		b.WriteString(`<div class="model-stat-row">`)
		b.WriteString(`<span class="model-name">` + name + `</span>`)
		b.WriteString(`<div class="model-accuracy-bar"><div class="accuracy-fill" style="width: ` + fmt.Sprintf("%.1f%%", accuracy*100) + `"></div></div>`)
		b.WriteString(`<span class="model-accuracy">` + fmt.Sprintf("%.1f%%", accuracy*100) + `</span>`)
		b.WriteString(`<span class="model-count">` + fmt.Sprintf("(%d/%d)", correct, total) + `</span>`)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)

	return b.String()
}
