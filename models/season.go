package models

// Season Objects
type SeasonStatus struct {
	IsHockeySeason bool   `json:"isHockeySeason"`
	CurrentSeason  string `json:"currentSeason"`
	SeasonPhase    string `json:"seasonPhase"` // "preseason", "regular", "playoffs", "offseason"
} 