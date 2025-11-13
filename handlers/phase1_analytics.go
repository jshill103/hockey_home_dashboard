package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jaredshillingburg/go_uhc/services"
)

// GetAccuracySummary returns the current prediction accuracy summary
func GetAccuracySummary(w http.ResponseWriter, r *http.Request) {
	errorAnalysis := services.GetErrorAnalysisService()
	if errorAnalysis == nil {
		http.Error(w, "Error analysis service not available", http.StatusServiceUnavailable)
		return
	}

	summary := errorAnalysis.GetSummary()
	if summary == nil {
		http.Error(w, "No accuracy data available yet", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetErrorPatterns returns identified prediction error patterns
func GetErrorPatterns(w http.ResponseWriter, r *http.Request) {
	errorAnalysis := services.GetErrorAnalysisService()
	if errorAnalysis == nil {
		http.Error(w, "Error analysis service not available", http.StatusServiceUnavailable)
		return
	}

	patterns := errorAnalysis.GetErrorPatterns()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}

// GetRecentPredictions returns recent prediction records
func GetRecentPredictions(w http.ResponseWriter, r *http.Request) {
	errorAnalysis := services.GetErrorAnalysisService()
	if errorAnalysis == nil {
		http.Error(w, "Error analysis service not available", http.StatusServiceUnavailable)
		return
	}

	// Get count from query params (default 10)
	countStr := r.URL.Query().Get("count")
	count := 10
	if countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 {
			count = c
		}
	}

	records := errorAnalysis.GetRecentRecords(count)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// GetFeatureImportance returns feature importance rankings
func GetFeatureImportance(w http.ResponseWriter, r *http.Request) {
	featureAnalyzer := services.GetFeatureImportanceAnalyzer()
	if featureAnalyzer == nil {
		http.Error(w, "Feature importance analyzer not available", http.StatusServiceUnavailable)
		return
	}

	// Get top N features (default all)
	topStr := r.URL.Query().Get("top")
	if topStr != "" {
		if top, err := strconv.Atoi(topStr); err == nil && top > 0 {
			features := featureAnalyzer.GetTopFeatures(top)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(features)
			return
		}
	}

	// Return all features
	features := featureAnalyzer.GetAllFeatures()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(features)
}

// GetLowValueFeatures returns features with low importance
func GetLowValueFeatures(w http.ResponseWriter, r *http.Request) {
	featureAnalyzer := services.GetFeatureImportanceAnalyzer()
	if featureAnalyzer == nil {
		http.Error(w, "Feature importance analyzer not available", http.StatusServiceUnavailable)
		return
	}

	// Get threshold from query params (default 0.2)
	thresholdStr := r.URL.Query().Get("threshold")
	threshold := 0.2
	if thresholdStr != "" {
		if t, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			threshold = t
		}
	}

	lowValueFeatures := featureAnalyzer.GetLowValueFeatures(threshold)
	
	response := map[string]interface{}{
		"threshold": threshold,
		"count":     len(lowValueFeatures),
		"features":  lowValueFeatures,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTimeWeightedStats returns time-weighted statistics for a team
func GetTimeWeightedStats(w http.ResponseWriter, r *http.Request) {
	timeWeightedService := services.GetTimeWeightedStatsService()
	if timeWeightedService == nil {
		http.Error(w, "Time-weighted stats service not available", http.StatusServiceUnavailable)
		return
	}

	teamCode := r.URL.Query().Get("team")
	if teamCode == "" {
		http.Error(w, "Missing 'team' parameter", http.StatusBadRequest)
		return
	}

	stats := timeWeightedService.GetStats(teamCode)
	if stats == nil {
		http.Error(w, fmt.Sprintf("No time-weighted stats available for team %s", teamCode), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// AnalyzeSpecialTeamsMatchup analyzes special teams matchup between two teams
func AnalyzeSpecialTeamsMatchup(w http.ResponseWriter, r *http.Request) {
	homeTeam := r.URL.Query().Get("homeTeam")
	awayTeam := r.URL.Query().Get("awayTeam")

	if homeTeam == "" || awayTeam == "" {
		http.Error(w, "Missing 'homeTeam' or 'awayTeam' parameter", http.StatusBadRequest)
		return
	}

	// TODO: Integrate with existing prediction factors service
	// For now, return a message indicating this endpoint is being developed
	response := map[string]interface{}{
		"message":  "Special teams matchup analysis endpoint under development",
		"homeTeam": homeTeam,
		"awayTeam": awayTeam,
		"note":     "This will be integrated with the prediction pipeline to analyze PP vs PK matchups",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPhase1AnalyticsDashboard returns a comprehensive Phase 1 analytics dashboard
func GetPhase1AnalyticsDashboard(w http.ResponseWriter, r *http.Request) {
	errorAnalysis := services.GetErrorAnalysisService()
	featureAnalyzer := services.GetFeatureImportanceAnalyzer()

	dashboard := map[string]interface{}{
		"status": "Phase 1 Analytics Operational",
	}

	// Accuracy summary
	if errorAnalysis != nil {
		summary := errorAnalysis.GetSummary()
		if summary != nil {
			dashboard["accuracy"] = map[string]interface{}{
				"overallAccuracy":       summary.OverallAccuracy,
				"totalPredictions":      summary.TotalPredictions,
				"correctPredictions":    summary.CorrectPredictions,
				"last10GamesAccuracy":   summary.Last10GamesAccuracy,
				"last30GamesAccuracy":   summary.Last30GamesAccuracy,
				"brierScore":            summary.BrierScore,
				"highConfidenceAccuracy": summary.HighConfidenceAccuracy,
			}

			// Error patterns summary
			patterns := errorAnalysis.GetErrorPatterns()
			if len(patterns) > 0 {
				dashboard["topErrorPatterns"] = patterns[:min(3, len(patterns))]
			}

			// Model performance
			dashboard["modelAccuracies"] = summary.ModelAccuracies
		}
	}

	// Feature importance summary
	if featureAnalyzer != nil {
		topFeatures := featureAnalyzer.GetTopFeatures(10)
		lowValueFeatures := featureAnalyzer.GetLowValueFeatures(0.2)
		
		dashboard["featureImportance"] = map[string]interface{}{
			"top10Features":       topFeatures,
			"lowValueCount":       len(lowValueFeatures),
			"lowValueFeatures":    lowValueFeatures,
		}
	}

	// Phase 1 improvements status
	dashboard["improvements"] = map[string]interface{}{
		"errorAnalysis":       errorAnalysis != nil,
		"featureImportance":   featureAnalyzer != nil,
		"timeWeightedStats":   services.GetTimeWeightedStatsService() != nil,
		"specialTeamsMatchup": true, // Always available
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

