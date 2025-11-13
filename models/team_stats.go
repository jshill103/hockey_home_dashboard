package models

// TeamStats represents basic team statistics for a season
type TeamStats struct {
	TeamCode       string  `json:"teamCode"`
	GamesPlayed    int     `json:"gamesPlayed"`
	Wins           int     `json:"wins"`
	Losses         int     `json:"losses"`
	OvertimeLosses int     `json:"overtimeLosses"`
	Points         int     `json:"points"`
	GoalsFor       float64 `json:"goalsFor"`       // Total goals scored
	GoalsAgainst   float64 `json:"goalsAgainst"`   // Total goals allowed
	PowerPlayPct   float64 `json:"powerPlayPct"`   // Power play percentage (0-1)
	PenaltyKillPct float64 `json:"penaltyKillPct"` // Penalty kill percentage (0-1)
	ShotsFor       float64 `json:"shotsFor"`       // Total shots
	ShotsAgainst   float64 `json:"shotsAgainst"`   // Total shots against
	FaceoffWinPct  float64 `json:"faceoffWinPct"`  // Faceoff win percentage (0-1)
	PenaltyMinutes int     `json:"penaltyMinutes"` // Total penalty minutes
}

