package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleTriggerDailyPredictions manually triggers the daily prediction generation
func HandleTriggerDailyPredictions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed. Use POST."}`, http.StatusMethodNotAllowed)
		return
	}

	dailyService := services.GetDailyPredictionService()
	if dailyService == nil {
		http.Error(w, `{"error": "Daily Prediction Service not initialized"}`, http.StatusInternalServerError)
		return
	}

	// Check for force=true query parameter
	force := r.URL.Query().Get("force") == "true"

	var message string
	if force {
		message = "Force refresh triggered: All existing predictions will be deleted and regenerated. This may take 5-10 minutes."
		// Trigger prediction generation with force flag
		go dailyService.TriggerNowWithForce(true)
	} else {
		message = "Daily prediction generation triggered. Only new games will be predicted (existing predictions preserved). Check /api/predictions/all in a few moments."
		// Trigger normal prediction generation
		go dailyService.TriggerNow()
	}

	response := map[string]interface{}{
		"success": true,
		"force":   force,
		"message": message,
	}

	json.NewEncoder(w).Encode(response)
}
