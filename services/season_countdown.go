package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// GetSeasonCountdown fetches information about the countdown to the next NHL season
func GetSeasonCountdown(teamConfig models.TeamConfig) (models.SeasonCountdown, error) {
	fmt.Println("Fetching season countdown information...")

	// For 2025-26 season, we know it starts October 7, 2025
	// But let's fetch the actual schedule to be dynamic
	seasonDate := "2025-10-07" // Start with the known season opener date

	urlIn := fmt.Sprintf("https://api-web.nhle.com/v1/schedule/%s", seasonDate)
	body, err := MakeAPICall(urlIn)
	if err != nil {
		return models.SeasonCountdown{}, fmt.Errorf("error fetching season schedule: %v", err)
	}

	var scheduleResponse models.SeasonScheduleResponse
	if err := json.Unmarshal(body, &scheduleResponse); err != nil {
		return models.SeasonCountdown{}, fmt.Errorf("error unmarshaling season schedule JSON: %v", err)
	}

	// Find the first game of the season
	if len(scheduleResponse.GameWeeks) == 0 || len(scheduleResponse.GameWeeks[0].Games) == 0 {
		return models.SeasonCountdown{}, fmt.Errorf("no games found in season schedule")
	}

	firstGame := scheduleResponse.GameWeeks[0].Games[0]

	// Parse the game start time
	gameTime, err := time.Parse(time.RFC3339, firstGame.StartTime)
	if err != nil {
		return models.SeasonCountdown{}, fmt.Errorf("error parsing game time: %v", err)
	}

	// Convert to team's timezone (default to Mountain Time)
	timezone := "America/Denver" // Default timezone
	if teamConfig.Code == "EDM" {
		timezone = "America/Edmonton"
	} else if teamConfig.Code == "TOR" {
		timezone = "America/Toronto"
	} else if teamConfig.Code == "BOS" || teamConfig.Code == "NYR" {
		timezone = "America/New_York"
	}

	teamTime, err := time.LoadLocation(timezone)
	if err != nil {
		return models.SeasonCountdown{}, fmt.Errorf("error loading timezone: %v", err)
	}
	gameTimeTeam := gameTime.In(teamTime)

	// Calculate days until season
	now := time.Now()
	daysUntil := int(gameTime.Sub(now).Hours() / 24)
	seasonStarted := daysUntil <= 0

	countdown := models.SeasonCountdown{
		DaysUntilSeason:    daysUntil,
		FirstGameDate:      gameTime,
		FirstGameFormatted: gameTimeTeam.Format("Monday, January 2, 2006"),
		FirstGameTeams:     fmt.Sprintf("%s @ %s", firstGame.AwayTeam.CommonName.Default, firstGame.HomeTeam.CommonName.Default),
		FirstGameVenue:     firstGame.Venue.Default,
		FirstGameTime:      gameTimeTeam.Format("3:04 PM MST"),
		SeasonStarted:      seasonStarted,
	}

	// Now find the team's first game
	teamGame, err := findTeamFirstGame(scheduleResponse, teamConfig)
	if err == nil {
		teamGameTime, err := time.Parse(time.RFC3339, teamGame.StartTime)
		if err == nil {
			teamGameTimeLocal := teamGameTime.In(teamTime)
			teamDaysUntil := int(teamGameTime.Sub(now).Hours() / 24)

			countdown.TeamFirstGameDate = teamGameTime
			countdown.TeamFirstGameFormatted = teamGameTimeLocal.Format("Monday, January 2, 2006")
			countdown.TeamFirstGameTime = teamGameTimeLocal.Format("3:04 PM MST")
			countdown.DaysUntilTeamGame = teamDaysUntil

			// Determine if team is home or away
			if teamGame.HomeTeam.CommonName.Default == teamConfig.ShortName {
				countdown.TeamFirstGameTeams = fmt.Sprintf("%s vs %s", teamGame.HomeTeam.CommonName.Default, teamGame.AwayTeam.CommonName.Default)
				countdown.TeamFirstGameVenue = teamGame.Venue.Default
			} else {
				countdown.TeamFirstGameTeams = fmt.Sprintf("%s @ %s", teamGame.AwayTeam.CommonName.Default, teamGame.HomeTeam.CommonName.Default)
				countdown.TeamFirstGameVenue = teamGame.Venue.Default
			}
		}
	}

	fmt.Printf("Season countdown: %d days until season starts (%s)\n", daysUntil, countdown.FirstGameFormatted)
	if countdown.TeamFirstGameDate.IsZero() == false {
		fmt.Printf("%s's first game: %d days until %s game (%s)\n", teamConfig.ShortName, countdown.DaysUntilTeamGame, teamConfig.Code, countdown.TeamFirstGameFormatted)
	}

	return countdown, nil
}

// findTeamFirstGame searches through the schedule to find the team's first game
func findTeamFirstGame(schedule models.SeasonScheduleResponse, teamConfig models.TeamConfig) (models.Game, error) {
	for _, week := range schedule.GameWeeks {
		for _, game := range week.Games {
			// Check both common name and team code/abbreviation for more robust matching
			homeMatches := game.HomeTeam.CommonName.Default == teamConfig.ShortName ||
				game.HomeTeam.Abbrev == teamConfig.Code
			awayMatches := game.AwayTeam.CommonName.Default == teamConfig.ShortName ||
				game.AwayTeam.Abbrev == teamConfig.Code

			if homeMatches || awayMatches {
				fmt.Printf("Found %s game: %s @ %s\n", teamConfig.Code, game.AwayTeam.CommonName.Default, game.HomeTeam.CommonName.Default)
				return game, nil
			}
		}
	}
	return models.Game{}, fmt.Errorf("no %s games found in schedule", teamConfig.Code)
}
