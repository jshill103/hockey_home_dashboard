package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleSimulationMetrics returns performance metrics for playoff simulations (Phase 5.5)
func HandleSimulationMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	metrics := services.GetGlobalMetrics().GetMetrics()

	response := map[string]interface{}{
		"totalSimulations":      metrics.TotalSimulations,
		"totalDuration":         metrics.TotalDuration.String(),
		"avgSimulationTime":     metrics.AvgSimulationTime.String(),
		"fastestSimulation":     metrics.FastestSimulation.String(),
		"slowestSimulation":     metrics.SlowestSimulation.String(),
		"parallelSimulations":   metrics.ParallelSimulations,
		"sequentialSimulations": metrics.SequentialSimulations,
		"cacheHits":             metrics.CacheHits,
		"cacheMisses":           metrics.CacheMisses,
		"cacheHitRate":          metrics.GetCacheHitRate(),
		"whatIfCacheHits":       metrics.WhatIfCacheHits,
		"whatIfCacheMisses":     metrics.WhatIfCacheMisses,
		"whatIfCacheHitRate":    metrics.GetWhatIfCacheHitRate(),
		"lastUpdated":           metrics.LastUpdated,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleResetMetrics resets simulation metrics (Phase 5.5)
func HandleResetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	services.GetGlobalMetrics().Reset()

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Simulation metrics reset successfully",
	})
}

