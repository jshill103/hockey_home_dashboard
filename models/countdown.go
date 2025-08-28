package models

import "time"

// SeasonCountdown represents information about the countdown to the next season
type SeasonCountdown struct {
	DaysUntilSeason        int       `json:"daysUntilSeason"`
	FirstGameDate          time.Time `json:"firstGameDate"`
	FirstGameFormatted     string    `json:"firstGameFormatted"`
	FirstGameTeams         string    `json:"firstGameTeams"`
	FirstGameVenue         string    `json:"firstGameVenue"`
	FirstGameTime          string    `json:"firstGameTime"`
	SeasonStarted          bool      `json:"seasonStarted"`
	TeamFirstGameDate      time.Time `json:"teamFirstGameDate,omitempty"`
	TeamFirstGameFormatted string    `json:"teamFirstGameFormatted,omitempty"`
	TeamFirstGameTeams     string    `json:"teamFirstGameTeams,omitempty"`
	TeamFirstGameVenue     string    `json:"teamFirstGameVenue,omitempty"`
	TeamFirstGameTime      string    `json:"teamFirstGameTime,omitempty"`
	DaysUntilTeamGame      int       `json:"daysUntilTeamGame,omitempty"`
}

// SeasonScheduleResponse represents the NHL API response for season schedule
type SeasonScheduleResponse struct {
	GameWeeks              []GameWeek `json:"gameWeek"`
	RegularSeasonStartDate string     `json:"regularSeasonStartDate"`
	RegularSeasonEndDate   string     `json:"regularSeasonEndDate"`
	PreSeasonStartDate     string     `json:"preSeasonStartDate"`
	PlayoffEndDate         string     `json:"playoffEndDate"`
}

// GameWeek represents a week of games in the schedule
type GameWeek struct {
	Date  string `json:"date"`
	Games []Game `json:"games"`
}
