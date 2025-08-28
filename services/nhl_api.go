package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// MakeAPICall makes a generic HTTP GET request to the specified URL
func MakeAPICall(urlIn string) ([]byte, error) {
	fmt.Printf("Making API call to: %s\n", urlIn)

	req, err := http.NewRequest("GET", urlIn, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return nil, err
	}

	// Add User-Agent header to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; UHC-Bot/1.0)")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return nil, err
	}
	defer res.Body.Close()

	fmt.Printf("API Response Status: %d\n", res.StatusCode)

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status code: %d", res.StatusCode)
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		fmt.Printf("Error reading response body: %v\n", readErr)
		return nil, readErr
	}

	fmt.Printf("Response body length: %d bytes\n", len(body))

	return body, nil
}

// GetTeamSchedule fetches the next upcoming game for a specific team
func GetTeamSchedule(teamCode string) (models.Game, error) {
	fmt.Printf("Fetching %s schedule...\n", teamCode)

	urlIn := fmt.Sprintf("https://api-web.nhle.com/v1/club-schedule/%s/week/now", teamCode)
	body, err := MakeAPICall(urlIn)
	if err != nil {
		fmt.Printf("Error calling schedule API: %v\n", err)
		return models.Game{}, err
	}

	now := time.Now().UTC()
	location, locErr := time.LoadLocation("America/Denver")
	if locErr != nil {
		fmt.Printf("Error loading timezone: %v\n", locErr)
		return models.Game{}, locErr
	}

	var data models.ScheduleResponse

	// Add debug output for raw response
	fmt.Printf("Raw API response: %s\n", string(body))

	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("Error unmarshaling schedule JSON: %v\n", err)
		return models.Game{}, err
	}

	fmt.Printf("Found %d games in schedule response\n", len(data.Games))

	// Find the next upcoming game
	for i, game := range data.Games {
		fmt.Printf("Processing game %d: %s vs %s at %s\n", i,
			game.AwayTeam.CommonName.Default,
			game.HomeTeam.CommonName.Default,
			game.StartTime)

		// Parse the game time
		parsedTime, err := time.Parse(time.RFC3339, game.StartTime)
		if err != nil {
			fmt.Printf("Error parsing game time %s: %v\n", game.StartTime, err)
			continue
		}

		// Format time for display
		game.FormattedTime = parsedTime.In(location).Format("2006-01-02 15:04:05")

		// Check if this is a future game
		if parsedTime.After(now) {
			fmt.Printf("Found next game: %s vs %s on %s\n",
				game.AwayTeam.CommonName.Default,
				game.HomeTeam.CommonName.Default,
				game.FormattedTime)
			return game, nil
		}

		fmt.Printf("Game %d is in the past, skipping\n", i)
	}

	fmt.Println("No upcoming games found")
	return models.Game{}, nil
}

// GetUHCSchedule fetches the next upcoming game for Utah Hockey Club (backward compatibility)
func GetUHCSchedule() (models.Game, error) {
	return GetTeamSchedule("UTA")
}

// GetTeamUpcomingGames returns all upcoming games in the next 7 days for a specific team
func GetTeamUpcomingGames(teamCode string) ([]models.Game, error) {
	fmt.Printf("Fetching upcoming games for %s...\n", teamCode)

	urlIn := fmt.Sprintf("https://api-web.nhle.com/v1/club-schedule/%s/week/now", teamCode)
	body, err := MakeAPICall(urlIn)
	if err != nil {
		fmt.Printf("Error fetching upcoming games: %v\n", err)
		return nil, err
	}

	now := time.Now().UTC()
	location, locErr := time.LoadLocation("America/Denver")
	if locErr != nil {
		return nil, locErr
	}

	var data models.ScheduleResponse
	var upcomingGames []models.Game

	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("Error unmarshaling upcoming games JSON: %v\n", err)
		return nil, err
	}

	fmt.Printf("Processing %d games for upcoming games list\n", len(data.Games))

	for _, game := range data.Games {
		// Convert UTC time to MST, Denver timezone
		startTime, err := time.Parse(time.RFC3339, game.StartTime)
		if err != nil {
			fmt.Printf("Error parsing game time: %v\n", err)
			continue
		}

		// Format time for display
		game.FormattedTime = startTime.In(location).Format("2006-01-02 15:04:05")

		// Only include future games within the next 7 days
		if startTime.After(now) {
			// Check if game is within next 7 days
			sevenDaysFromNow := now.Add(7 * 24 * time.Hour)
			if startTime.Before(sevenDaysFromNow) {
				upcomingGames = append(upcomingGames, game)
				fmt.Printf("Added upcoming game: %s vs %s on %s\n",
					game.AwayTeam.CommonName.Default,
					game.HomeTeam.CommonName.Default,
					game.FormattedTime)
			}
		}
	}

	fmt.Printf("Found %d upcoming games in next 7 days\n", len(upcomingGames))
	return upcomingGames, nil
}

// GetUpcomingGames returns all upcoming games in the next 7 days (backward compatibility)
func GetUpcomingGames() ([]models.Game, error) {
	return GetTeamUpcomingGames("UTA")
}

// GetTeamScoreboard fetches the current scoreboard data for a specific team
func GetTeamScoreboard(teamCode string) (models.ScoreboardGame, error) {
	fmt.Printf("Fetching %s scoreboard...\n", teamCode)

	urlIn := fmt.Sprintf("https://api-web.nhle.com/v1/scoreboard/%s/now", teamCode)
	body, err := MakeAPICall(urlIn)
	if err != nil {
		fmt.Printf("Error calling scoreboard API: %v\n", err)
		return models.ScoreboardGame{}, err
	}

	// Add debug output for raw response
	fmt.Printf("Raw scoreboard API response: %s\n", string(body))

	var data models.ScoreboardResponse
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("Error unmarshaling scoreboard JSON: %v\n", err)
		return models.ScoreboardGame{}, err
	}

	fmt.Printf("Found %d date entries in scoreboard response\n", len(data.GamesByDate))

	// Find the most recent game
	for _, gamesByDate := range data.GamesByDate {
		fmt.Printf("Processing date %s with %d games\n", gamesByDate.Date, len(gamesByDate.Games))

		for _, game := range gamesByDate.Games {
			if game.GameID != 0 {
				fmt.Printf("Found active game: ID %d, %s vs %s, State: %s\n",
					game.GameID,
					game.AwayTeam.Name.Default,
					game.HomeTeam.Name.Default,
					game.GameState)
				return game, nil
			}
		}
	}

	fmt.Println("No active games found in scoreboard")
	return models.ScoreboardGame{}, nil
}

// GetUHCScoreboard fetches the current scoreboard data for Utah Hockey Club (backward compatibility)
func GetUHCScoreboard() (models.ScoreboardGame, error) {
	return GetTeamScoreboard("UTA")
}

// GetStandings fetches the current NHL standings
func GetStandings() (models.StandingsResponse, error) {
	fmt.Println("Fetching NHL standings...")

	urlIn := "https://api-web.nhle.com/v1/standings/now"
	body, err := MakeAPICall(urlIn)
	if err != nil {
		fmt.Printf("Error fetching standings: %v\n", err)
		return models.StandingsResponse{}, err
	}

	var data models.StandingsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("Error parsing standings JSON: %v\n", err)
		return models.StandingsResponse{}, err
	}

	fmt.Println("Successfully fetched NHL standings")
	return data, nil
}

// TestAPIEndpoints tests all NHL API endpoints to verify they're working
func TestAPIEndpoints() {
	fmt.Println("\n=== Testing NHL API Endpoints ===")

	endpoints := []string{
		"https://api-web.nhle.com/v1/club-schedule/EDM/week/now", // Use EDM as test team
		"https://api-web.nhle.com/v1/scoreboard/EDM/now",
		"https://api-web.nhle.com/v1/standings/now",
		"https://api-web.nhle.com/v1/schedule/now",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("\nTesting endpoint: %s\n", endpoint)
		body, err := MakeAPICall(endpoint)
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
		} else {
			fmt.Printf("✅ SUCCESS: Got %d bytes of data\n", len(body))
			// Show first 200 chars of response for debugging
			preview := string(body)
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Printf("Preview: %s\n", preview)
		}
	}

	fmt.Println("\n=== API Endpoint Testing Complete ===")
}
