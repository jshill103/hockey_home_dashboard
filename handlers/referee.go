package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleGetReferees returns all referees in the system
func HandleGetReferees(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	referees := refService.GetAllReferees()
	json.NewEncoder(w).Encode(referees)
}

// HandleGetReferee returns a specific referee by ID
func HandleGetReferee(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get referee ID from query parameter
	refereeIDStr := r.URL.Query().Get("id")
	if refereeIDStr == "" {
		http.Error(w, `{"error": "id parameter required"}`, http.StatusBadRequest)
		return
	}

	refereeID, err := strconv.Atoi(refereeIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid referee id"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	referee, err := refService.GetReferee(refereeID)
	if err != nil {
		http.Error(w, `{"error": "Referee not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(referee)
}

// HandleGetGameAssignment returns referee assignment for a specific game
func HandleGetGameAssignment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get game ID from query parameter
	gameIDStr := r.URL.Query().Get("gameId")
	if gameIDStr == "" {
		http.Error(w, `{"error": "gameId parameter required"}`, http.StatusBadRequest)
		return
	}

	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid game id"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	assignment, err := refService.GetGameAssignment(gameID)
	if err != nil {
		http.Error(w, `{"error": "No referee assignment found for this game"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(assignment)
}

// HandleGetDailySchedule returns all referee assignments for a specific date
func HandleGetDailySchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get date from query parameter (format: YYYY-MM-DD)
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		// Default to today
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, `{"error": "invalid date format (use YYYY-MM-DD)"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	schedule, err := refService.GetDailySchedule(date)
	if err != nil {
		// Return empty schedule instead of error
		emptySchedule := models.RefereeDailySchedule{
			Date:        date,
			Assignments: []models.RefereeGameAssignment{},
			LastUpdated: time.Now(),
		}
		json.NewEncoder(w).Encode(emptySchedule)
		return
	}

	json.NewEncoder(w).Encode(schedule)
}

// HandleGetRefereeStats returns statistics for a specific referee
func HandleGetRefereeStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get referee ID from query parameter
	refereeIDStr := r.URL.Query().Get("id")
	if refereeIDStr == "" {
		http.Error(w, `{"error": "id parameter required"}`, http.StatusBadRequest)
		return
	}

	refereeID, err := strconv.Atoi(refereeIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid referee id"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	stats, err := refService.GetRefereeStats(refereeID)
	if err != nil {
		http.Error(w, `{"error": "Stats not found for this referee"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// HandleGetRefereeImpact analyzes referee impact on a game
func HandleGetRefereeImpact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get parameters
	gameIDStr := r.URL.Query().Get("gameId")
	homeTeam := r.URL.Query().Get("homeTeam")
	awayTeam := r.URL.Query().Get("awayTeam")

	if gameIDStr == "" || homeTeam == "" || awayTeam == "" {
		http.Error(w, `{"error": "gameId, homeTeam, and awayTeam parameters required"}`, http.StatusBadRequest)
		return
	}

	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid game id"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	impact, err := refService.AnalyzeRefereeImpact(gameID, homeTeam, awayTeam)
	if err != nil {
		http.Error(w, `{"error": "Could not analyze referee impact"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(impact)
}

// HandleAddReferee adds or updates a referee (manual data entry)
func HandleAddReferee(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var referee models.Referee
	if err := json.NewDecoder(r.Body).Decode(&referee); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	if err := refService.AddReferee(&referee); err != nil {
		http.Error(w, `{"error": "Failed to add referee"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Referee added/updated successfully",
		"referee": referee,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleAddGameAssignment adds a referee assignment for a game (manual data entry)
func HandleAddGameAssignment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var assignment models.RefereeGameAssignment
	if err := json.NewDecoder(r.Body).Decode(&assignment); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	if err := refService.AddGameAssignment(&assignment); err != nil {
		http.Error(w, `{"error": "Failed to add game assignment"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":     "success",
		"message":    "Referee assignment added successfully",
		"assignment": assignment,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleUpdateRefereeStats updates statistics for a referee (manual data entry)
func HandleUpdateRefereeStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var stats models.RefereeSeasonStats
	if err := json.NewDecoder(r.Body).Decode(&stats); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	if err := refService.UpdateRefereeStats(&stats); err != nil {
		http.Error(w, `{"error": "Failed to update referee stats"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Referee stats updated successfully",
		"stats":   stats,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleTriggerScrape manually triggers a web scrape of referee data
func HandleTriggerScrape(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	// Return immediately and scrape in background
	response := map[string]interface{}{
		"status":  "processing",
		"message": "Referee data scraping started. Check server logs for progress.",
	}
	json.NewEncoder(w).Encode(response)

	// Run scrape in background
	go func() {
		if err := refService.TriggerScrape(); err != nil {
			fmt.Printf("❌ Scrape failed: %v\n", err)
		}
	}()
}

// HandleScraperStatus returns the status of the referee scraper
func HandleScraperStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	scraper := refService.GetScraper()
	if scraper == nil {
		http.Error(w, `{"error": "Scraper not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	status := map[string]interface{}{
		"status":     "active",
		"lastScrape": scraper.LastScrape,
		"message":    "Referee scraper is operational",
	}

	json.NewEncoder(w).Encode(status)
}

// ============================================================================
// PHASE 3: ADVANCED ANALYTICS HANDLERS
// ============================================================================

// HandleGetRefereeTendencies returns calculated tendencies for a referee
func HandleGetRefereeTendencies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get referee ID from query parameter
	refereeIDStr := r.URL.Query().Get("id")
	if refereeIDStr == "" {
		http.Error(w, `{"error": "id parameter required"}`, http.StatusBadRequest)
		return
	}

	refereeID, err := strconv.Atoi(refereeIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid referee id"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	tendencies, err := refService.CalculateRefereeTendencies(refereeID)
	if err != nil {
		http.Error(w, `{"error": "Could not calculate referee tendencies"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(tendencies)
}

// HandleGetRefereeProfile returns a comprehensive profile for a referee
func HandleGetRefereeProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get referee ID from query parameter
	refereeIDStr := r.URL.Query().Get("id")
	if refereeIDStr == "" {
		http.Error(w, `{"error": "id parameter required"}`, http.StatusBadRequest)
		return
	}

	refereeID, err := strconv.Atoi(refereeIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid referee id"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	profile, err := refService.GenerateRefereeProfile(refereeID)
	if err != nil {
		http.Error(w, `{"error": "Could not generate referee profile"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(profile)
}

// HandleGetTeamBias returns bias data for a referee-team pairing
func HandleGetTeamBias(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get parameters
	refereeIDStr := r.URL.Query().Get("refereeId")
	teamCode := r.URL.Query().Get("teamCode")
	seasonStr := r.URL.Query().Get("season")

	if refereeIDStr == "" || teamCode == "" {
		http.Error(w, `{"error": "refereeId and teamCode parameters required"}`, http.StatusBadRequest)
		return
	}

	refereeID, err := strconv.Atoi(refereeIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid referee id"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	// If season is not provided, use current season
	var season int
	if seasonStr == "" {
		// Get current season from service
		season = refService.GetCurrentSeason()
	} else {
		season, err = strconv.Atoi(seasonStr)
		if err != nil {
			http.Error(w, `{"error": "invalid season"}`, http.StatusBadRequest)
			return
		}
	}

	bias, err := refService.GetTeamBias(refereeID, teamCode, season)
	if err != nil {
		http.Error(w, `{"error": "No bias data found for this referee-team pairing"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(bias)
}

// HandleGetAllTeamBiases returns all bias data for a specific referee
func HandleGetAllTeamBiases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get referee ID from query parameter
	refereeIDStr := r.URL.Query().Get("refereeId")
	if refereeIDStr == "" {
		http.Error(w, `{"error": "refereeId parameter required"}`, http.StatusBadRequest)
		return
	}

	refereeID, err := strconv.Atoi(refereeIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid referee id"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	biases, err := refService.GetAllTeamBiases(refereeID)
	if err != nil {
		http.Error(w, `{"error": "No bias data found for this referee"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(biases)
}

// HandleGetAdvancedImpact returns advanced impact analysis with tendencies
func HandleGetAdvancedImpact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get parameters
	gameIDStr := r.URL.Query().Get("gameId")
	homeTeam := r.URL.Query().Get("homeTeam")
	awayTeam := r.URL.Query().Get("awayTeam")

	if gameIDStr == "" || homeTeam == "" || awayTeam == "" {
		http.Error(w, `{"error": "gameId, homeTeam, and awayTeam parameters required"}`, http.StatusBadRequest)
		return
	}

	gameID, err := strconv.Atoi(gameIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid game id"}`, http.StatusBadRequest)
		return
	}

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	analysis, err := refService.GetAdvancedImpactAnalysis(gameID, homeTeam, awayTeam)
	if err != nil {
		http.Error(w, `{"error": "Could not analyze referee impact"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(analysis)
}

// ============================================================================
// BACKFILL & DATA COLLECTION HANDLERS
// ============================================================================

// HandleBackfillRefereeData backfills referee assignments from completed games
func HandleBackfillRefereeData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	scraper := refService.GetScraper()
	if scraper == nil {
		http.Error(w, `{"error": "Scraper not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	// Parse date range from query parameters (optional)
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	if startDateStr != "" && endDateStr != "" {
		// Custom date range
		startDate, err1 := time.Parse("2006-01-02", startDateStr)
		endDate, err2 := time.Parse("2006-01-02", endDateStr)
		
		if err1 != nil || err2 != nil {
			http.Error(w, `{"error": "Invalid date format. Use YYYY-MM-DD"}`, http.StatusBadRequest)
			return
		}
		
		// Run backfill in a goroutine so we don't timeout
		go scraper.BackfillRefereeAssignments(startDate, endDate)
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Backfill started",
			"start_date": startDateStr,
			"end_date": endDateStr,
		})
	} else {
		// Backfill from season start
		go scraper.BackfillFromSeasonStart()
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Backfill started from season start",
		})
	}
}

// HandleTriggerScraperUpdate manually triggers a scraper update
func HandleTriggerScraperUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	refService := services.GetRefereeService()
	if refService == nil {
		http.Error(w, `{"error": "Referee service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	scraper := refService.GetScraper()
	if scraper == nil {
		http.Error(w, `{"error": "Scraper not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	// Run update in goroutine
	go scraper.RunDailyUpdate()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Scraper update triggered",
		"time": time.Now(),
	})
}

