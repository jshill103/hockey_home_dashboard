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

// ScheduleAPIResponse wraps schedule data for API responses
type ScheduleAPIResponse struct {
	Status       string      `json:"status"`
	TeamCode     string      `json:"teamCode"`
	LastUpdated  time.Time   `json:"lastUpdated"`
	NextGame     interface{} `json:"nextGame"`     // Can be Game or null
	UpcomingGames interface{} `json:"upcomingGames,omitempty"` // Optional: list of upcoming games
	SeasonGames  interface{} `json:"seasonGames,omitempty"`   // Optional: full season schedule
	Count        int         `json:"count,omitempty"` // Number of games returned
}

// HandleScheduleAPI handles JSON API requests for schedule data
// Endpoints:
//   GET /api/schedule/next?team=UTA         - Next upcoming game
//   GET /api/schedule/upcoming?team=UTA     - Upcoming games (next 7 days)
//   GET /api/schedule/season?team=UTA&year=20242025 - Full season schedule
func HandleScheduleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow external apps to access

	// Get team code from query params (default to UTA)
	teamCode := r.URL.Query().Get("team")
	if teamCode == "" {
		teamCode = "UTA"
	}

	// Determine which schedule data to return based on path
	switch r.URL.Path {
	case "/api/schedule/next":
		handleNextGame(w, teamCode)
	case "/api/schedule/upcoming":
		handleUpcomingGames(w, teamCode)
	case "/api/schedule/season":
		handleSeasonSchedule(w, r, teamCode)
	default:
		// Default: return next game
		handleNextGame(w, teamCode)
	}
}

// handleNextGame returns the next upcoming game
func handleNextGame(w http.ResponseWriter, teamCode string) {
	game, err := services.GetTeamSchedule(teamCode)
	if err != nil {
		response := ScheduleAPIResponse{
			Status:   "error",
			TeamCode: teamCode,
			NextGame: nil,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ScheduleAPIResponse{
		Status:      "success",
		TeamCode:    teamCode,
		LastUpdated: time.Now(),
		NextGame:    game,
		Count:       1,
	}

	if game.GameDate == "" {
		response.NextGame = nil
		response.Count = 0
	}

	json.NewEncoder(w).Encode(response)
}

// handleUpcomingGames returns upcoming games in the next 7 days
func handleUpcomingGames(w http.ResponseWriter, teamCode string) {
	games, err := services.GetTeamUpcomingGames(teamCode)
	if err != nil {
		response := ScheduleAPIResponse{
			Status:        "error",
			TeamCode:      teamCode,
			UpcomingGames: []interface{}{},
			Count:         0,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ScheduleAPIResponse{
		Status:        "success",
		TeamCode:      teamCode,
		LastUpdated:   time.Now(),
		UpcomingGames: games,
		Count:         len(games),
	}

	json.NewEncoder(w).Encode(response)
}

// handleSeasonSchedule returns the full season schedule
func handleSeasonSchedule(w http.ResponseWriter, r *http.Request, teamCode string) {
	// Get season year from query params (default to current season)
	seasonStr := r.URL.Query().Get("season")
	season := 20242025 // Default to 2024-2025 season

	if seasonStr != "" {
		if parsed, err := strconv.Atoi(seasonStr); err == nil {
			season = parsed
		}
	}

	games, err := services.GetTeamSeasonSchedule(teamCode, season)
	if err != nil {
		response := ScheduleAPIResponse{
			Status:      "error",
			TeamCode:    teamCode,
			SeasonGames: []interface{}{},
			Count:       0,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ScheduleAPIResponse{
		Status:      "success",
		TeamCode:    teamCode,
		LastUpdated: time.Now(),
		SeasonGames: games,
		Count:       len(games),
	}

	json.NewEncoder(w).Encode(response)
}

// HandleScheduleHealthAPI returns health status and metadata about the schedule API
func HandleScheduleHealthAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now(),
		"version":     "1.0.0",
		"endpoints": []string{
			"/api/schedule/next",
			"/api/schedule/upcoming",
			"/api/schedule/season",
		},
		"description": "NHL schedule API for external applications",
		"usage": map[string]string{
			"next":     "GET /api/schedule/next?team=UTA - Returns next upcoming game",
			"upcoming": "GET /api/schedule/upcoming?team=UTA - Returns games in next 7 days",
			"season":   "GET /api/schedule/season?team=UTA&season=20242025 - Returns full season schedule",
		},
	}

	json.NewEncoder(w).Encode(health)
}

// HandleAllTeamsScheduleAPI returns schedule data for all NHL teams
// This is useful for the video analyzer to know about all games happening
func HandleAllTeamsScheduleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get date from query params (default to today)
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	// Fetch league-wide schedule using NHL API
	url := fmt.Sprintf("https://api-web.nhle.com/v1/schedule/%s", dateStr)
	body, err := services.MakeAPICall(url)
	if err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Failed to fetch league schedule: %v", err),
			"games":  []interface{}{},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Parse the schedule response
	var scheduleData struct {
		GameWeek []struct {
			Date  string         `json:"date"`
			Games []models.Game  `json:"games"`
		} `json:"gameWeek"`
	}

	if err := json.Unmarshal(body, &scheduleData); err != nil {
		response := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Failed to parse schedule: %v", err),
			"games":  []interface{}{},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Flatten all games from the week
	allGames := []models.Game{}
	for _, day := range scheduleData.GameWeek {
		allGames = append(allGames, day.Games...)
	}

	response := map[string]interface{}{
		"status":      "success",
		"date":        dateStr,
		"lastUpdated": time.Now(),
		"games":       allGames,
		"count":       len(allGames),
	}

	json.NewEncoder(w).Encode(response)
}

