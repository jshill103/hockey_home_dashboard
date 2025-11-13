package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/services"
)

// Global Twitter collector instance
var globalTwitterCollector *services.TwitterRefereeCollector

// InitializeTwitterCollector initializes the Twitter referee collector
func InitializeTwitterCollector(refereeService *services.RefereeService, bearerToken string) {
	globalTwitterCollector = services.NewTwitterRefereeCollector(refereeService)
	
	if bearerToken != "" {
		globalTwitterCollector.SetTwitterCredentials(bearerToken)
		log.Printf("🐦 Twitter API integration enabled")
	} else {
		log.Printf("🌐 Twitter web scraping fallback enabled (no API token)")
	}
	
	// Start automated collection every 4 hours
	globalTwitterCollector.StartAutomatedCollection(4)
	log.Printf("🤖 Automated Twitter collection started (every 4 hours)")
}

// HandleCollectTwitterReferees manually triggers Twitter collection
func HandleCollectTwitterReferees(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if globalTwitterCollector == nil {
		http.Error(w, "Twitter collector not initialized", http.StatusServiceUnavailable)
		return
	}
	
	log.Printf("📲 Manual Twitter collection triggered")
	
	count, err := globalTwitterCollector.CollectRefereeAssignments()
	if err != nil {
		log.Printf("❌ Twitter collection failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"success":              true,
		"assignmentsCollected": count,
		"message":              fmt.Sprintf("Collected %d referee assignments from Twitter", count),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	log.Printf("✅ Manual Twitter collection complete: %d assignments", count)
}

// HandleGetTwitterStatus returns Twitter collector status
func HandleGetTwitterStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if globalTwitterCollector == nil {
		http.Error(w, "Twitter collector not initialized", http.StatusServiceUnavailable)
		return
	}
	
	// Determine collection method
	method := "web_scraping"
	if globalTwitterCollector.BearerToken != "" {
		method = "twitter_api"
	}
	
	response := map[string]interface{}{
		"enabled":                       true,
		"method":                        method,
		"automatedCollectionInterval":  "4 hours",
		"monitoredAccounts":             []string{"@ScoutingTheRefs", "@NHLOfficials", "@NHLRefWatcher"},
		"message":                       "Twitter referee collection is active",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

