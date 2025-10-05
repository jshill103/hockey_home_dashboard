package utils

import (
	"fmt"
	"time"
)

// GetCurrentSeason returns the current NHL season in YYYYZZZZ format
// Example: 20252026 for the 2025-2026 season
// NHL seasons run from October to June, with July-September as off-season
func GetCurrentSeason() int {
	return GetSeasonForDate(time.Now())
}

// GetSeasonForDate returns the NHL season for a specific date
// This allows for testing with historical dates
func GetSeasonForDate(date time.Time) int {
	year := date.Year()
	month := date.Month()

	// NHL season runs October to June
	// July-September is off-season (preparing for next season)
	if month >= time.July && month <= time.September {
		// Off-season: next season hasn't started yet, but we reference it
		// for draft, free agency, etc.
		return year*10000 + (year + 1)
	}

	// October-December: season started this year, ends next year
	if month >= time.October {
		return year*10000 + (year + 1)
	}

	// January-June: season started last year, ends this year
	return (year-1)*10000 + year
}

// GetPreviousSeason returns the previous NHL season
// Example: If current is 20252026, returns 20242025
func GetPreviousSeason() int {
	return GetCurrentSeason() - 10001
}

// GetSeasonForYear returns the season that starts in the given year
// Example: GetSeasonForYear(2025) returns 20252026
func GetSeasonForYear(year int) int {
	return year*10000 + (year + 1)
}

// GetSeasonYears returns the start and end year of a season
// Example: GetSeasonYears(20252026) returns (2025, 2026)
func GetSeasonYears(season int) (int, int) {
	startYear := season / 10000
	endYear := season % 10000
	return startYear, endYear
}

// FormatSeason returns a human-readable season string
// Example: FormatSeason(20252026) returns "2025-2026"
func FormatSeason(season int) string {
	startYear, endYear := GetSeasonYears(season)
	return fmt.Sprintf("%d-%d", startYear, endYear)
}

// IsOffseason returns true if the current date is in the NHL off-season
// Off-season is typically July through September
func IsOffseason() bool {
	return IsOffseasonForDate(time.Now())
}

// IsOffseasonForDate returns true if the given date is in the NHL off-season
func IsOffseasonForDate(date time.Time) bool {
	month := date.Month()
	return month >= time.July && month <= time.September
}
