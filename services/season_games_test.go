package services

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// TestMain keeps season-length lookups offline for every test in this package
// (simulation code paths call GetCurrentGamesPerSeason).
func TestMain(m *testing.M) {
	fetchSeasonGamesFromStatsAPI = failingStats
	fetchSeasonGamesFromSchedule = failingSchedule
	os.Exit(m.Run())
}

// withSeasonGamesFakes installs fake fetchers/clock/current-season and restores them on cleanup.
func withSeasonGamesFakes(t *testing.T, statsAPI func(int) (int, error), schedule func(string, int) (int, error), currentSeason int) *time.Time {
	t.Helper()
	origStats, origSched, origNow, origSeason := fetchSeasonGamesFromStatsAPI, fetchSeasonGamesFromSchedule, seasonGamesNow, currentSeasonForGames
	now := time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC)
	fetchSeasonGamesFromStatsAPI = statsAPI
	fetchSeasonGamesFromSchedule = schedule
	seasonGamesNow = func() time.Time { return now }
	currentSeasonForGames = func() int { return currentSeason }
	ResetSeasonGamesCache()
	t.Cleanup(func() {
		fetchSeasonGamesFromStatsAPI, fetchSeasonGamesFromSchedule, seasonGamesNow, currentSeasonForGames = origStats, origSched, origNow, origSeason
		ResetSeasonGamesCache()
	})
	return &now
}

func failingStats(int) (int, error)            { return 0, errors.New("stats down") }
func failingSchedule(string, int) (int, error) { return 0, errors.New("schedule down") }
func fixedStats(n int) func(int) (int, error)  { return func(int) (int, error) { return n, nil } }

func TestFallbackGamesPerSeason(t *testing.T) {
	tests := []struct {
		season, want int
	}{
		{20122013, 48},
		{20192020, 82},
		{20202021, 56},
		{20232024, 82},
		{20242025, 82},
		{20252026, 82},
		{20262027, 84},
		{20272028, 84},
		{20352036, 84},
	}
	for _, tt := range tests {
		if got := FallbackGamesPerSeason(tt.season); got != tt.want {
			t.Errorf("FallbackGamesPerSeason(%d) = %d, want %d", tt.season, got, tt.want)
		}
	}
}

func TestGetGamesPerSeason_UsesStatsAPI(t *testing.T) {
	scheduleCalled := false
	withSeasonGamesFakes(t, fixedStats(84), func(string, int) (int, error) {
		scheduleCalled = true
		return 0, nil
	}, 20262027)

	if got := GetGamesPerSeason(20262027); got != 84 {
		t.Fatalf("GetGamesPerSeason = %d, want 84", got)
	}
	if scheduleCalled {
		t.Error("schedule fetcher should not be called when stats API succeeds")
	}
}

func TestGetGamesPerSeason_FallsBackToSchedule(t *testing.T) {
	var gotTeam string
	withSeasonGamesFakes(t, failingStats, func(team string, season int) (int, error) {
		gotTeam = team
		return 84, nil
	}, 20262027)
	SetSeasonGamesTeamCode("COL")
	t.Cleanup(func() { SetSeasonGamesTeamCode("UTA") })

	if got := GetGamesPerSeason(20262027); got != 84 {
		t.Fatalf("GetGamesPerSeason = %d, want 84", got)
	}
	if gotTeam != "COL" {
		t.Errorf("schedule fetched for team %q, want COL", gotTeam)
	}
}

func TestGetGamesPerSeason_FallsBackToTable(t *testing.T) {
	withSeasonGamesFakes(t, failingStats, failingSchedule, 20262027)

	if got := GetGamesPerSeason(20262027); got != 84 {
		t.Errorf("2026-27 fallback = %d, want 84", got)
	}
	if got := GetGamesPerSeason(20252026); got != 82 {
		t.Errorf("2025-26 fallback = %d, want 82", got)
	}
}

func TestGetGamesPerSeason_CachesAPIResult(t *testing.T) {
	calls := 0
	now := withSeasonGamesFakes(t, func(int) (int, error) {
		calls++
		return 84, nil
	}, failingSchedule, 20262027)

	for i := 0; i < 5; i++ {
		GetGamesPerSeason(20262027)
	}
	if calls != 1 {
		t.Fatalf("stats API called %d times, want 1 (cached)", calls)
	}

	*now = now.Add(seasonGamesAPITTL + time.Minute)
	GetGamesPerSeason(20262027)
	if calls != 2 {
		t.Errorf("stats API called %d times after TTL expiry, want 2", calls)
	}
}

func TestGetGamesPerSeason_RetriesSoonAfterFallback(t *testing.T) {
	statsUp := false
	now := withSeasonGamesFakes(t, func(int) (int, error) {
		if !statsUp {
			return 0, errors.New("down")
		}
		return 86, nil
	}, failingSchedule, 20262027)

	if got := GetGamesPerSeason(20262027); got != 84 {
		t.Fatalf("fallback = %d, want 84", got)
	}
	statsUp = true
	if got := GetGamesPerSeason(20262027); got != 84 {
		t.Fatalf("within fallback TTL = %d, want cached 84", got)
	}
	*now = now.Add(seasonGamesFallbackTTL + time.Second)
	if got := GetGamesPerSeason(20262027); got != 86 {
		t.Errorf("after fallback TTL = %d, want 86 from API", got)
	}
}

func TestGamesRemaining(t *testing.T) {
	tests := []struct {
		total, played, want int
	}{
		{84, 0, 84},
		{84, 2, 82},
		{84, 60, 24},
		{84, 84, 0},
		{82, 82, 0},
		{84, 90, 0},
	}
	for _, tt := range tests {
		if got := GamesRemaining(tt.total, tt.played); got != tt.want {
			t.Errorf("GamesRemaining(%d, %d) = %d, want %d", tt.total, tt.played, got, tt.want)
		}
	}
}

func TestGamesRemainingForTeam(t *testing.T) {
	withSeasonGamesFakes(t, fixedStats(84), failingSchedule, 20262027)

	if got := GamesRemainingForTeam(&models.TeamStanding{GamesPlayed: 0}); got != 84 {
		t.Errorf("GP=0: got %d, want 84", got)
	}
	if got := GamesRemainingForTeam(&models.TeamStanding{GamesPlayed: 10}); got != 74 {
		t.Errorf("GP=10: got %d, want 74", got)
	}
	if got := GamesRemainingForTeam(nil); got != 0 {
		t.Errorf("nil team: got %d, want 0", got)
	}
}

func TestParseStatsAPISeasonGames(t *testing.T) {
	body := []byte(`{"data":[{"id":20262027,"numberOfGames":84,"totalRegularSeasonGames":1344}],"total":1}`)
	got, err := parseStatsAPISeasonGames(body, 20262027)
	if err != nil || got != 84 {
		t.Fatalf("parse = %d, %v; want 84, nil", got, err)
	}
	if _, err := parseStatsAPISeasonGames(body, 20272028); err == nil {
		t.Error("expected error for season not in response")
	}
	if _, err := parseStatsAPISeasonGames([]byte(`{"data":[]}`), 20262027); err == nil {
		t.Error("expected error for empty data")
	}
}

func TestCountRegularSeasonGames(t *testing.T) {
	body := []byte(`{"games":[{"gameType":1},{"gameType":1},{"gameType":2},{"gameType":2},{"gameType":2},{"gameType":3}]}`)
	got, err := countRegularSeasonGames(body)
	if err != nil || got != 3 {
		t.Fatalf("count = %d, %v; want 3, nil", got, err)
	}
	if _, err := countRegularSeasonGames([]byte(`{"games":[{"gameType":1}]}`)); err == nil {
		t.Error("expected error when no regular-season games")
	}
}

func TestCalculateMagicNumbers_UsesSeasonLength(t *testing.T) {
	withSeasonGamesFakes(t, fixedStats(84), failingSchedule, 20262027)

	team := models.TeamStanding{TeamAbbrev: models.TeamNameInfo{Default: "UTA"}, Points: 10, GamesPlayed: 6}
	mn := CalculateMagicNumbers(&team, []models.TeamStanding{team})
	if want := 10 + (84-6)*2; mn.MaxPossiblePoints != want {
		t.Errorf("MaxPossiblePoints = %d, want %d", mn.MaxPossiblePoints, want)
	}
}
