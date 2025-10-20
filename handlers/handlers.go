package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// Shared state for handlers
var (
	cachedSchedule        *models.Game
	cachedScheduleUpdated *time.Time
	cachedNews            *[]models.NewsHeadline
	cachedUpcomingGames   *[]models.Game
	currentSeasonStatus   *models.SeasonStatus
	isGameCurrentlyLive   *bool
	teamConfig            *models.TeamConfig
)

// Init initializes the handlers with shared state from main
func Init(
	schedule *models.Game,
	scheduleUpdated *time.Time,
	news *[]models.NewsHeadline,
	upcomingGames *[]models.Game,
	seasonStatus *models.SeasonStatus,
	gameLive *bool,
) {
	cachedSchedule = schedule
	cachedScheduleUpdated = scheduleUpdated
	cachedNews = news
	cachedUpcomingGames = upcomingGames
	currentSeasonStatus = seasonStatus
	isGameCurrentlyLive = gameLive
}

// InitTeamConfig initializes the team configuration for handlers
func InitTeamConfig(config *models.TeamConfig) {
	teamConfig = config
}

// HandleAPITest provides a web endpoint to test all NHL API endpoints
func HandleAPITest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head>
	<title>NHL API Test Results</title>
	<style>
		body { 
			font-family: 'Segoe UI', monospace; 
			background: #1a1a1a; 
			color: #ffffff; 
			margin: 20px; 
			line-height: 1.6;
		}
		.container { 
			max-width: 1200px; 
			margin: 0 auto; 
			background: #2d2d2d; 
			padding: 20px; 
			border-radius: 10px; 
		}
		h1 { 
			color: #ff6b35; 
			text-align: center; 
			margin-bottom: 30px;
		}
		pre { 
			background: #000000; 
			padding: 15px; 
			border-radius: 5px; 
			white-space: pre-wrap; 
			font-family: 'Courier New', monospace;
			border-left: 4px solid #ff6b35;
		}
		.success { color: #4CAF50; }
		.error { color: #f44336; }
		.info { color: #2196F3; }
		.back-link {
			display: inline-block;
			margin: 20px 0;
			padding: 10px 20px;
			background: #ff6b35;
			color: white;
			text-decoration: none;
			border-radius: 5px;
		}
		.back-link:hover {
			background: #e55a2b;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>🏒 NHL API Test Results</h1>
		<a href="/" class="back-link">← Back to Home</a>
		<pre>`)

	// Capture output by redirecting to our writer
	originalOutput := fmt.Sprintf("API Testing started at: %s\n\n",
		fmt.Sprintf("%v", http.StatusText(http.StatusOK)))
	fmt.Fprint(w, originalOutput)

	// Run comprehensive API tests
	fmt.Fprint(w, "=== Running Comprehensive NHL API Tests ===\n\n")

	// Test GetTeamSchedule
	fmt.Fprintf(w, "--- Testing GetTeamSchedule for %s ---\n", teamConfig.Code)
	schedule, err := services.GetTeamSchedule(teamConfig.Code)
	if err != nil {
		fmt.Fprintf(w, "❌ GetTeamSchedule failed: %v\n", err)
	} else if schedule.HomeTeam.CommonName.Default == "" {
		fmt.Fprint(w, "⚠️  GetTeamSchedule: No upcoming games found\n")
	} else {
		fmt.Fprintf(w, "✅ GetTeamSchedule success: %s vs %s on %s\n",
			schedule.AwayTeam.CommonName.Default,
			schedule.HomeTeam.CommonName.Default,
			schedule.FormattedTime)
	}
	fmt.Fprint(w, "\n")

	// Test GetTeamUpcomingGames
	fmt.Fprintf(w, "--- Testing GetTeamUpcomingGames for %s ---\n", teamConfig.Code)
	upcomingGames, err := services.GetTeamUpcomingGames(teamConfig.Code)
	if err != nil {
		fmt.Fprintf(w, "❌ GetTeamUpcomingGames failed: %v\n", err)
	} else {
		fmt.Fprintf(w, "✅ GetTeamUpcomingGames success: Found %d games in next 7 days\n", len(upcomingGames))
		for i, game := range upcomingGames {
			if i < 3 { // Show first 3 games
				fmt.Fprintf(w, "   Game %d: %s vs %s on %s\n", i+1,
					game.AwayTeam.CommonName.Default,
					game.HomeTeam.CommonName.Default,
					game.FormattedTime)
			}
		}
		if len(upcomingGames) > 3 {
			fmt.Fprintf(w, "   ... and %d more games\n", len(upcomingGames)-3)
		}
	}
	fmt.Fprint(w, "\n")

	// Test GetStandings
	fmt.Fprint(w, "--- Testing GetStandings ---\n")
	standings, err := services.GetStandings()
	if err != nil {
		fmt.Fprintf(w, "❌ GetStandings failed: %v\n", err)
	} else {
		fmt.Fprint(w, "✅ GetStandings success: Got standings data\n")
		_ = standings // Use the variable to avoid unused variable warning
	}
	fmt.Fprint(w, "\n")

	// Test Season Validation
	fmt.Fprint(w, "--- Testing Season Validation ---\n")
	hasGames, err := services.ValidateSeasonWithAPI()
	if err != nil {
		fmt.Fprintf(w, "❌ ValidateSeasonWithAPI failed: %v\n", err)
	} else {
		fmt.Fprintf(w, "✅ ValidateSeasonWithAPI success: Games found = %t\n", hasGames)
	}
	fmt.Fprint(w, "\n")

	// Test raw API endpoints
	fmt.Fprint(w, "--- Testing Raw API Endpoints ---\n")
	endpoints := []string{
		"https://api-web.nhle.com/v1/club-schedule/UTA/week/now",
		"https://api-web.nhle.com/v1/scoreboard/UTA/now",
		"https://api-web.nhle.com/v1/standings/now",
		"https://api-web.nhle.com/v1/schedule/now",
	}

	for _, endpoint := range endpoints {
		fmt.Fprintf(w, "\nTesting: %s\n", endpoint)
		body, err := services.MakeAPICall(endpoint)
		if err != nil {
			fmt.Fprintf(w, "❌ FAILED: %v\n", err)
		} else {
			fmt.Fprintf(w, "✅ SUCCESS: Got %d bytes of data\n", len(body))
		}
	}

	fmt.Fprint(w, "\n=== API Testing Complete ===\n")

	fmt.Fprint(w, `
		</pre>
		<a href="/" class="back-link">← Back to Home</a>
	</div>
</body>
</html>`)
}

// HandleRateLimiterMetrics provides an endpoint to view NHL API rate limiter metrics
func HandleRateLimiterMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rateLimiter := services.GetNHLRateLimiter()
	metrics := rateLimiter.GetMetrics()

	// Format as JSON
	fmt.Fprintf(w, `{
  "totalRequests": %d,
  "requestsInWindow": %d,
  "maxRequests": %d,
  "timeWindow": "%v",
  "minDelay": "%v",
  "delayedRequests": %d,
  "totalWaitTime": "%v",
  "averageWaitTime": "%v",
  "currentUtilization": %.2f
}`,
		metrics.TotalRequests,
		metrics.RequestsInWindow,
		metrics.MaxRequests,
		metrics.TimeWindow,
		metrics.MinDelay,
		metrics.DelayedRequests,
		metrics.TotalWaitTime,
		metrics.AverageWaitTime,
		metrics.CurrentUtilization)
}
