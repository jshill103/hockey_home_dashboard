package models

// Scoreboard Objects
type ScoreboardResponse struct {
	GamesByDate []GamesByDate `json:"gamesByDate"`
}

type GamesByDate struct {
	Date  string           `json:"date"`
	Games []ScoreboardGame `json:"games"`
}

type ScoreboardGame struct {
	GameID     int            `json:"id"`
	GameState  string         `json:"gameState"`
	Period     int            `json:"period"`
	PeriodTime string         `json:"periodTime"`
	HomeTeam   ScoreboardTeam `json:"homeTeam"`
	AwayTeam   ScoreboardTeam `json:"awayTeam"`
	StartTime  string         `json:"startTimeUTC"`
	EndTime    string         `json:"endTimeUTC"`
}

type ScoreboardTeam struct {
	ID        int      `json:"id"`
	Name      TeamName `json:"name"`
	Abbrev    string   `json:"abbrev"`
	Score     int      `json:"score"`
	Shots     int      `json:"sog"`
	PowerPlay string   `json:"powerPlayConversion"`
}

type TeamName struct {
	Default string `json:"default"`
} 