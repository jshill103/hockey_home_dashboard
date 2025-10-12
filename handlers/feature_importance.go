package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleFeatureImportance returns feature importance analysis
func HandleFeatureImportance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fis := services.GetFeatureImportanceService()
	if fis == nil {
		http.Error(w, `{"error": "Feature Importance Service not initialized"}`, http.StatusInternalServerError)
		return
	}

	report, err := fis.GetFeatureImportanceReport()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		http.Error(w, `{"error": "Failed to marshal response"}`, http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

// HandleFeatureImportanceMarkdown returns markdown report
func HandleFeatureImportanceMarkdown(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown")

	fis := services.GetFeatureImportanceService()
	if fis == nil {
		http.Error(w, "Feature Importance Service not initialized", http.StatusInternalServerError)
		return
	}

	markdown, err := fis.GenerateMarkdownReport()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(markdown))
}
