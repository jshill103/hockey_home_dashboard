package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/services"
)

// GetRePredictionMetrics returns smart re-prediction statistics
func GetRePredictionMetrics(w http.ResponseWriter, r *http.Request) {
	smartRePrediction := services.GetSmartRePredictionService()
	if smartRePrediction == nil {
		http.Error(w, "Smart Re-Prediction Service not initialized", http.StatusServiceUnavailable)
		return
	}

	metrics := smartRePrediction.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
