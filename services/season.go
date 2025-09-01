package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// GetCurrentSeason returns the current NHL season string (e.g., "20242025")
func GetCurrentSeason() string {
	now := time.Now()
	currentYear := now.Year()

	// NHL season typically starts in October and ends in June of the following year
	// If we're in Jan-June, we're in the season that started the previous year
	// If we're in July-December, we're in the season that starts this year

	if now.Month() >= time.July {
		// July-December: season starts this year
		return fmt.Sprintf("%d%d", currentYear, currentYear+1)
	} else {
		// January-June: season started last year
		return fmt.Sprintf("%d%d", currentYear-1, currentYear)
	}
}

// GetSeasonStatus determines if we're currently in hockey season
func GetSeasonStatus() models.SeasonStatus {
	now := time.Now()
	currentSeason := GetCurrentSeason()

	// Define season phases based on typical NHL calendar
	var seasonPhase string
	var isHockeySeason bool

	switch now.Month() {
	case time.July, time.August:
		seasonPhase = "offseason"
		isHockeySeason = false
	case time.September:
		seasonPhase = "preseason"
		isHockeySeason = false
	case time.October, time.November, time.December, time.January, time.February, time.March:
		seasonPhase = "regular"
		isHockeySeason = true
	case time.April, time.May:
		seasonPhase = "playoffs"
		isHockeySeason = true
	case time.June:
		// June can be either playoffs (Stanley Cup Finals) or offseason
		if now.Day() <= 15 {
			seasonPhase = "playoffs"
			isHockeySeason = true
		} else {
			seasonPhase = "offseason"
			isHockeySeason = false
		}
	}

	return models.SeasonStatus{
		IsHockeySeason: isHockeySeason,
		CurrentSeason:  currentSeason,
		SeasonPhase:    seasonPhase,
	}
}

// ValidateSeasonWithAPI checks the NHL API to validate season status
func ValidateSeasonWithAPI() (bool, error) {
	fmt.Println("Validating season status with NHL API...")

	// Check if there are any games scheduled today or in the near future
	urlIn := "https://api-web.nhle.com/v1/schedule/now"
	body, err := MakeAPICall(urlIn)
	if err != nil {
		fmt.Printf("Error calling NHL API for validation: %v\n", err)
		return false, fmt.Errorf("error calling NHL API: %v", err)
	}

	fmt.Printf("Validation API response length: %d bytes\n", len(body))

	// Try to parse the JSON to see if there's actual game data
	var scheduleData map[string]interface{}
	if err := json.Unmarshal(body, &scheduleData); err != nil {
		fmt.Printf("Error parsing validation JSON: %v\n", err)
		// If we can't parse it but got data, assume there are games
		return len(body) > 100, nil
	}

	// Check if there are any games in the response
	if gameWeeks, ok := scheduleData["gameWeek"].([]interface{}); ok && len(gameWeeks) > 0 {
		fmt.Printf("Found %d game weeks in validation response\n", len(gameWeeks))

		// Look for actual games within the game weeks
		totalGames := 0
		for _, week := range gameWeeks {
			if weekMap, ok := week.(map[string]interface{}); ok {
				if games, ok := weekMap["games"].([]interface{}); ok {
					totalGames += len(games)
				}
			}
		}

		fmt.Printf("Found %d total games in validation response\n", totalGames)
		return totalGames > 0, nil
	}

	// If we get any response data, there are likely games scheduled
	hasData := len(body) > 100
	fmt.Printf("Validation result: hasData = %t (based on response size)\n", hasData)
	return hasData, nil
}

// TestAllAPIs runs comprehensive tests on all NHL API endpoints
func TestAllAPIs() {
	fmt.Println("\n=== Running Comprehensive API Tests ===")

	// Test individual API functions
	fmt.Println("\n--- Testing GetUHCSchedule ---")
	schedule, err := GetUHCSchedule()
	if err != nil {
		fmt.Printf("❌ GetUHCSchedule failed: %v\n", err)
	} else {
		fmt.Printf("✅ GetUHCSchedule success: %s vs %s\n",
			schedule.AwayTeam.CommonName.Default,
			schedule.HomeTeam.CommonName.Default)
	}

	fmt.Println("\n--- Testing GetUHCScoreboard ---")
	scoreboard, err := GetUHCScoreboard()
	if err != nil {
		fmt.Printf("❌ GetUHCScoreboard failed: %v\n", err)
	} else if scoreboard.GameID == 0 {
		fmt.Printf("✅ GetUHCScoreboard success: No active games\n")
	} else {
		fmt.Printf("✅ GetUHCScoreboard success: Game ID %d, %s vs %s\n",
			scoreboard.GameID,
			scoreboard.AwayTeam.Name.Default,
			scoreboard.HomeTeam.Name.Default)
	}

	fmt.Println("\n--- Testing GetUpcomingGames ---")
	upcomingGames, err := GetUpcomingGames()
	if err != nil {
		fmt.Printf("❌ GetUpcomingGames failed: %v\n", err)
	} else {
		fmt.Printf("✅ GetUpcomingGames success: Found %d games\n", len(upcomingGames))
	}

	fmt.Println("\n--- Testing GetStandings ---")
	standings, err := GetStandings()
	if err != nil {
		fmt.Printf("❌ GetStandings failed: %v\n", err)
	} else {
		fmt.Printf("✅ GetStandings success: Got standings data\n")
		_ = standings // Use the variable to avoid unused variable warning
	}

	fmt.Println("\n--- Testing Season Validation ---")
	hasGames, err := ValidateSeasonWithAPI()
	if err != nil {
		fmt.Printf("❌ ValidateSeasonWithAPI failed: %v\n", err)
	} else {
		fmt.Printf("✅ ValidateSeasonWithAPI success: Games found = %t\n", hasGames)
	}

	// Test all endpoints directly
	TestAPIEndpoints()

	fmt.Println("=== API Testing Complete ===\n")
}
