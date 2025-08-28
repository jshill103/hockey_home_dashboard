package models

// Schedule Objects
type ScheduleResponse struct {
	Games []Game
}

type Game struct {
	GameDate      string      `json:"gameDate"`
	StartTime     string      `json:"startTimeUTC"`
	FormattedTime string      // Computed field for display
	Broadcasts    []Broadcast `json:"tvBroadcasts"`
	HomeTeam      Team        `json:"homeTeam"`
	AwayTeam      Team        `json:"awayTeam"`
	Venue         Venue       `json:"venue"`
}

type Broadcast struct {
	Network string `json:"network"`
}

type Team struct {
	CommonName CommonNameInfo `json:"commonName"`
	Abbrev     string         `json:"abbrev"`
}

type CommonNameInfo struct {
	Default string `json:"default"`
}

type Venue struct {
	Default string `json:"default"`
}
