package utils

import (
	"testing"
	"time"
)

func TestGetSeasonForDate(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected int
	}{
		{
			name:     "January 2025 (mid-season)",
			date:     time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC),
			expected: 20242025,
		},
		{
			name:     "June 2025 (end of season)",
			date:     time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC),
			expected: 20242025,
		},
		{
			name:     "July 2025 (off-season)",
			date:     time.Date(2025, time.July, 15, 0, 0, 0, 0, time.UTC),
			expected: 20252026,
		},
		{
			name:     "September 2025 (off-season)",
			date:     time.Date(2025, time.September, 30, 0, 0, 0, 0, time.UTC),
			expected: 20252026,
		},
		{
			name:     "October 2025 (season start)",
			date:     time.Date(2025, time.October, 1, 0, 0, 0, 0, time.UTC),
			expected: 20252026,
		},
		{
			name:     "December 2025 (mid-season)",
			date:     time.Date(2025, time.December, 25, 0, 0, 0, 0, time.UTC),
			expected: 20252026,
		},
		{
			name:     "October 2026 (next season start)",
			date:     time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
			expected: 20262027,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSeasonForDate(tt.date)
			if result != tt.expected {
				t.Errorf("GetSeasonForDate(%v) = %d, want %d", tt.date, result, tt.expected)
			}
		})
	}
}

func TestGetPreviousSeason(t *testing.T) {
	// Mock current season as 20252026
	// Note: This test will use the actual current date, so we test the logic
	current := GetCurrentSeason()
	previous := GetPreviousSeason()

	expected := current - 10001
	if previous != expected {
		t.Errorf("GetPreviousSeason() = %d, want %d", previous, expected)
	}
}

func TestGetSeasonYears(t *testing.T) {
	tests := []struct {
		season        int
		expectedStart int
		expectedEnd   int
	}{
		{20252026, 2025, 2026},
		{20242025, 2024, 2025},
		{20232024, 2023, 2024},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			start, end := GetSeasonYears(tt.season)
			if start != tt.expectedStart || end != tt.expectedEnd {
				t.Errorf("GetSeasonYears(%d) = (%d, %d), want (%d, %d)",
					tt.season, start, end, tt.expectedStart, tt.expectedEnd)
			}
		})
	}
}

func TestFormatSeason(t *testing.T) {
	tests := []struct {
		season   int
		expected string
	}{
		{20252026, "2025-2026"},
		{20242025, "2024-2025"},
		{20232024, "2023-2024"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := FormatSeason(tt.season)
			if result != tt.expected {
				t.Errorf("FormatSeason(%d) = %s, want %s", tt.season, result, tt.expected)
			}
		})
	}
}

func TestIsOffseasonForDate(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{
			name:     "January (not off-season)",
			date:     time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "July (off-season)",
			date:     time.Date(2025, time.July, 15, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "August (off-season)",
			date:     time.Date(2025, time.August, 15, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "September (off-season)",
			date:     time.Date(2025, time.September, 15, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "October (not off-season)",
			date:     time.Date(2025, time.October, 1, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOffseasonForDate(tt.date)
			if result != tt.expected {
				t.Errorf("IsOffseasonForDate(%v) = %v, want %v", tt.date, result, tt.expected)
			}
		})
	}
}

func TestGetSeasonForYear(t *testing.T) {
	tests := []struct {
		year     int
		expected int
	}{
		{2025, 20252026},
		{2024, 20242025},
		{2023, 20232024},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := GetSeasonForYear(tt.year)
			if result != tt.expected {
				t.Errorf("GetSeasonForYear(%d) = %d, want %d", tt.year, result, tt.expected)
			}
		})
	}
}
