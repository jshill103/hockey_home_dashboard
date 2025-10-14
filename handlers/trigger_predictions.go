package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleTriggerDailyPredictions manually triggers the daily prediction generation
func HandleTriggerDailyPredictions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dailyService := services.GetDailyPredictionService()
	if dailyService == nil {
		http.Error(w, `{"error": "Daily Prediction Service not initialized"}`, http.StatusInternalServerError)
		return
	}

	// Trigger prediction generation in a goroutine so we can return immediately
	go dailyService.TriggerNow()

	response := map[string]string{
		"status":  "triggered",
		"message": "Daily prediction generation started. Check /api/predictions/all in a few moments.",
	}

	json.NewEncoder(w).Encode(response)
}
