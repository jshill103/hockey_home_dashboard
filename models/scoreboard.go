package models

// Scoreboard Objects
type ScoreboardResponse struct {
	FocusedDate string        `json:"focusedDate"`
	GamesByDate []GamesByDate `json:"gamesByDate"`
}

type GamesByDate struct {
	Date  string           `json:"date"`
	Games []ScoreboardGame `json:"games"`
}

type ScoreboardGame struct {
	GameID     int    `json:"id"`
	GameState  string `json:"gameState"`
	Period     int    `json:"period"`
	PeriodTime string `json:"periodTime"`
	// Clock carries the live period clock as actually returned by the NHL API
	// (nested "clock" object). Verified against api-web.nhle.com/v1/scoreboard
	// and /v1/score responses: there is no flat top-level "periodTime" string
	// field, so PeriodTime above is effectively always empty when parsed from
	// real API responses. Use Clock.SecondsRemaining/TimeRemaining instead for
	// any live-game-time logic (see services/live_win_probability_service.go).
	Clock     GameClock      `json:"clock"`
	HomeTeam  ScoreboardTeam `json:"homeTeam"`
	AwayTeam  ScoreboardTeam `json:"awayTeam"`
	StartTime string         `json:"startTimeUTC"`
	EndTime   string         `json:"endTimeUTC"`
}

// GameClock represents the NHL API's live period clock. TimeRemaining (a
// "MM:SS" string) and SecondsRemaining both count DOWN from 20:00/1200 to
// 0:00/0 within the current period -- confirmed by inspecting real
// api-web.nhle.com scoreboard/score/boxscore responses (a finished period
// shows "00:00" / 0, not "20:00" / 1200).
type GameClock struct {
	TimeRemaining    string `json:"timeRemaining"`
	SecondsRemaining int    `json:"secondsRemaining"`
	Running          bool   `json:"running"`
	InIntermission   bool   `json:"inIntermission"`
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
