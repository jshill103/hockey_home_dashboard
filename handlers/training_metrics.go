package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleTrainingMetrics returns system-wide training metrics
func HandleTrainingMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tms := services.GetTrainingMetricsService()
	if tms == nil {
		http.Error(w, `{"error": "Training Metrics Service not initialized"}`, http.StatusInternalServerError)
		return
	}

	metrics := tms.GetSystemMetrics()
	json.NewEncoder(w).Encode(metrics)
}

// HandleModelTrainingMetrics returns training metrics for a specific model
func HandleModelTrainingMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get model name from query parameter
	modelName := r.URL.Query().Get("model")
	if modelName == "" {
		http.Error(w, `{"error": "model parameter required"}`, http.StatusBadRequest)
		return
	}

	tms := services.GetTrainingMetricsService()
	if tms == nil {
		http.Error(w, `{"error": "Training Metrics Service not initialized"}`, http.StatusInternalServerError)
		return
	}

	metrics := tms.GetModelMetrics(modelName)
	if metrics == nil {
		http.Error(w, `{"error": "model not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(metrics)
}

// HandleRecentTrainingEvents returns recent training events
func HandleRecentTrainingEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get limit from query parameter (default 50)
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	tms := services.GetTrainingMetricsService()
	if tms == nil {
		http.Error(w, `{"error": "Training Metrics Service not initialized"}`, http.StatusInternalServerError)
		return
	}

	events := tms.GetRecentTrainingEvents(limit)
	json.NewEncoder(w).Encode(events)
}
