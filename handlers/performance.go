package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/services"
)

// PerformanceDashboardHandler returns comprehensive performance metrics
func PerformanceDashboardHandler(w http.ResponseWriter, r *http.Request) {
	evalService := services.GetEvaluationService()

	if evalService == nil {
		http.Error(w, "Evaluation service not initialized", http.StatusServiceUnavailable)
		return
	}

	// Get ensemble metrics
	metrics := evalService.GetEnsembleMetrics()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// ModelMetricsHandler returns metrics for a specific model
func ModelMetricsHandler(w http.ResponseWriter, r *http.Request) {
	evalService := services.GetEvaluationService()

	if evalService == nil {
		http.Error(w, "Evaluation service not initialized", http.StatusServiceUnavailable)
		return
	}

	modelName := r.URL.Query().Get("model")
	if modelName == "" {
		// Return all models
		metrics := evalService.GetMetrics()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
		return
	}

	// Return specific model
	metrics := evalService.GetMetrics()
	if modelMetric, ok := metrics[modelName]; ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(modelMetric)
	} else {
		http.Error(w, "Model not found", http.StatusNotFound)
	}
}
