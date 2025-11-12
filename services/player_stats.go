package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
)

// GetTeamRoster fetches the current roster for a specific team
func GetTeamRoster(teamCode string) (models.TeamRosterResponse, error) {
	fmt.Printf("Fetching %s team roster...\n", teamCode)

	urlIn := fmt.Sprintf("https://api-web.nhle.com/v1/roster/%s/current", teamCode)
	body, err := MakeAPICall(urlIn)
	if err != nil {
		fmt.Printf("Error calling roster API: %v\n", err)
		return models.TeamRosterResponse{}, err
	}

	var data models.TeamRosterResponse
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("Error unmarshaling roster JSON: %v\n", err)
		return models.TeamRosterResponse{}, err
	}

	fmt.Printf("Successfully fetched roster: %d forwards, %d defensemen, %d goalies\n",
		len(data.Forwards), len(data.Defensemen), len(data.Goalies))
	return data, nil
}

// GetPlayerStatsLeaders fetches the current NHL stats leaders
func GetPlayerStatsLeaders() (models.PlayerStatsLeaders, error) {
	fmt.Println("Fetching NHL player stats leaders...")

	// We'll fetch goals, assists, and points leaders
	goals, err := getStatsLeadersByCategory("goals")
	if err != nil {
		return models.PlayerStatsLeaders{}, fmt.Errorf("error fetching goals leaders: %v", err)
	}

	assists, err := getStatsLeadersByCategory("assists")
	if err != nil {
		return models.PlayerStatsLeaders{}, fmt.Errorf("error fetching assists leaders: %v", err)
	}

	points, err := getStatsLeadersByCategory("points")
	if err != nil {
		return models.PlayerStatsLeaders{}, fmt.Errorf("error fetching points leaders: %v", err)
	}

	leaders := models.PlayerStatsLeaders{
		Goals:   goals,
		Assists: assists,
		Points:  points,
	}

	fmt.Printf("Successfully fetched stats leaders: %d goals, %d assists, %d points\n",
		len(leaders.Goals), len(leaders.Assists), len(leaders.Points))
	return leaders, nil
}

// GetGoalieStatsLeaders fetches the current NHL goalie stats leaders
func GetGoalieStatsLeaders() (models.GoalieStatsLeaders, error) {
	fmt.Println("Fetching NHL goalie stats leaders...")

	// Fetch goalie stats leaders
	wins, err := getGoalieStatsLeadersByCategory("wins")
	if err != nil {
		return models.GoalieStatsLeaders{}, fmt.Errorf("error fetching goalie wins leaders: %v", err)
	}

	savePct, err := getGoalieStatsLeadersByCategory("savePct")
	if err != nil {
		return models.GoalieStatsLeaders{}, fmt.Errorf("error fetching goalie save%% leaders: %v", err)
	}

	gaa, err := getGoalieStatsLeadersByCategory("goalsAgainstAverage")
	if err != nil {
		return models.GoalieStatsLeaders{}, fmt.Errorf("error fetching goalie GAA leaders: %v", err)
	}

	leaders := models.GoalieStatsLeaders{
		Wins:    wins,
		SavePct: savePct,
		GAA:     gaa,
	}

	fmt.Printf("Successfully fetched goalie leaders: %d wins, %d save%%, %d GAA\n",
		len(leaders.Wins), len(leaders.SavePct), len(leaders.GAA))
	return leaders, nil
}

// getStatsLeadersByCategory fetches player stats leaders for a specific category
func getStatsLeadersByCategory(category string) ([]models.PlayerStats, error) {
	urlIn := fmt.Sprintf("https://api-web.nhle.com/v1/skater-stats-leaders/current?categories=%s&limit=50", category)
	body, err := MakeAPICall(urlIn)
	if err != nil {
		return nil, err
	}

	var response models.PlayerStatsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling %s stats JSON: %v", category, err)
	}

	return response.Data, nil
}

// getGoalieStatsLeadersByCategory fetches goalie stats leaders for a specific category
func getGoalieStatsLeadersByCategory(category string) ([]models.GoalieStats, error) {
	urlIn := fmt.Sprintf("https://api-web.nhle.com/v1/goalie-stats-leaders/current?categories=%s&limit=50", category)
	body, err := MakeAPICall(urlIn)
	if err != nil {
		return nil, err
	}

	var response models.GoalieStatsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling goalie %s stats JSON: %v", category, err)
	}

	return response.Data, nil
}

// GetTeamPlayerStats filters stats leaders to show only players from a specific team
func GetTeamPlayerStats(leaders models.PlayerStatsLeaders, teamCode string) models.PlayerStatsLeaders {
	fmt.Printf("Filtering stats for %s players...\n", teamCode)

	var teamLeaders models.PlayerStatsLeaders

	// Filter goals leaders
	for _, player := range leaders.Goals {
		if strings.ToUpper(player.TeamAbbrev) == strings.ToUpper(teamCode) {
			teamLeaders.Goals = append(teamLeaders.Goals, player)
		}
	}

	// Filter assists leaders
	for _, player := range leaders.Assists {
		if strings.ToUpper(player.TeamAbbrev) == strings.ToUpper(teamCode) {
			teamLeaders.Assists = append(teamLeaders.Assists, player)
		}
	}

	// Filter points leaders
	for _, player := range leaders.Points {
		if strings.ToUpper(player.TeamAbbrev) == strings.ToUpper(teamCode) {
			teamLeaders.Points = append(teamLeaders.Points, player)
		}
	}

	fmt.Printf("Found %s players: %d goals, %d assists, %d points\n",
		teamCode, len(teamLeaders.Goals), len(teamLeaders.Assists), len(teamLeaders.Points))

	return teamLeaders
}

// GetUTAPlayerStats filters stats leaders to show only Utah Hockey Club players (backward compatibility)
func GetUTAPlayerStats(leaders models.PlayerStatsLeaders) models.PlayerStatsLeaders {
	return GetTeamPlayerStats(leaders, "UTA")
}

// GetTeamGoalieStats filters goalie stats leaders to show only goalies from a specific team
func GetTeamGoalieStats(leaders models.GoalieStatsLeaders, teamCode string) models.GoalieStatsLeaders {
	fmt.Printf("Filtering goalie stats for %s...\n", teamCode)

	var teamLeaders models.GoalieStatsLeaders

	// Filter wins leaders
	for _, goalie := range leaders.Wins {
		if strings.ToUpper(goalie.TeamAbbrev) == strings.ToUpper(teamCode) {
			teamLeaders.Wins = append(teamLeaders.Wins, goalie)
		}
	}

	// Filter save percentage leaders
	for _, goalie := range leaders.SavePct {
		if strings.ToUpper(goalie.TeamAbbrev) == strings.ToUpper(teamCode) {
			teamLeaders.SavePct = append(teamLeaders.SavePct, goalie)
		}
	}

	// Filter GAA leaders
	for _, goalie := range leaders.GAA {
		if strings.ToUpper(goalie.TeamAbbrev) == strings.ToUpper(teamCode) {
			teamLeaders.GAA = append(teamLeaders.GAA, goalie)
		}
	}

	fmt.Printf("Found %s goalies: %d wins, %d save%%, %d GAA\n",
		teamCode, len(teamLeaders.Wins), len(teamLeaders.SavePct), len(teamLeaders.GAA))

	return teamLeaders
}

// GetUTAGoalieStats filters goalie stats leaders to show only Utah Hockey Club goalies (backward compatibility)
func GetUTAGoalieStats(leaders models.GoalieStatsLeaders) models.GoalieStatsLeaders {
	return GetTeamGoalieStats(leaders, "UTA")
}

// GetUTARoster fetches the current roster for Utah Hockey Club (backward compatibility)
func GetUTARoster() (models.TeamRosterResponse, error) {
	return GetTeamRoster("UTA")
}

// FindPlayerRankInLeaders finds a player's rank in the NHL leaders for a specific stat
func FindPlayerRankInLeaders(playerID int, leaders []models.PlayerStats) int {
	for i, player := range leaders {
		if player.PlayerID == playerID {
			return i + 1 // Rank is 1-based
		}
	}
	return -1 // Player not found in leaders
}

// FindGoalieRankInLeaders finds a goalie's rank in the NHL leaders for a specific stat
func FindGoalieRankInLeaders(playerID int, leaders []models.GoalieStats) int {
	for i, goalie := range leaders {
		if goalie.PlayerID == playerID {
			return i + 1 // Rank is 1-based
		}
	}
	return -1 // Goalie not found in leaders
}
