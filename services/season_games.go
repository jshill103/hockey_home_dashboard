package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/utils"
)

// Regular-season length (games per team) is determined per season, in order:
//  1. NHL stats API season record (`numberOfGames`)
//  2. Count of regular-season games (gameType=2) in a club's season schedule
//  3. FallbackGamesPerSeason static lookup

const (
	seasonGamesAPITTL      = 24 * time.Hour
	seasonGamesFallbackTTL = 15 * time.Minute
	regularSeasonGameType  = 2
)

type seasonGamesEntry struct {
	games     int
	fetchedAt time.Time
	ttl       time.Duration
}

var (
	seasonGamesMu    sync.RWMutex
	seasonGamesCache = map[int]seasonGamesEntry{}

	seasonGamesTeamMu   sync.RWMutex
	seasonGamesTeamCode = "UTA"

	// Package-level so tests can inject fakes without network access.
	fetchSeasonGamesFromStatsAPI = fetchSeasonGamesFromStatsAPIImpl
	fetchSeasonGamesFromSchedule = fetchSeasonGamesFromScheduleImpl
	seasonGamesNow               = time.Now
	currentSeasonForGames        = utils.GetCurrentSeason
)

// SetSeasonGamesTeamCode sets the club whose schedule is used as the secondary
// source when the stats API is unavailable.
func SetSeasonGamesTeamCode(teamCode string) {
	if teamCode == "" {
		return
	}
	seasonGamesTeamMu.Lock()
	seasonGamesTeamCode = teamCode
	seasonGamesTeamMu.Unlock()
}

func getSeasonGamesTeamCode() string {
	seasonGamesTeamMu.RLock()
	defer seasonGamesTeamMu.RUnlock()
	return seasonGamesTeamCode
}

// FallbackGamesPerSeason returns the known regular-season length for a season
// (YYYYZZZZ format) without any API access.
func FallbackGamesPerSeason(season int) int {
	switch season {
	case 20122013:
		return 48 // lockout-shortened
	case 20192020:
		return 82 // scheduled length (season paused for COVID)
	case 20202021:
		return 56 // COVID-shortened
	}
	if season > 0 && season <= 20252026 {
		return 82
	}
	// 2026-27 onward (and any unknown future season) is 84 games
	return 84
}

// GetGamesPerSeason returns the number of regular-season games per team for
// the given season, using live NHL data when available (cached).
func GetGamesPerSeason(season int) int {
	// Read fast path: called from Monte Carlo simulation hot loops.
	now := seasonGamesNow()
	seasonGamesMu.RLock()
	entry, ok := seasonGamesCache[season]
	seasonGamesMu.RUnlock()
	if ok && now.Sub(entry.fetchedAt) < entry.ttl {
		return entry.games
	}

	seasonGamesMu.Lock()
	defer seasonGamesMu.Unlock()
	if entry, ok := seasonGamesCache[season]; ok && now.Sub(entry.fetchedAt) < entry.ttl {
		return entry.games
	}

	games, source := resolveGamesPerSeason(season)
	ttl := seasonGamesAPITTL
	if source == "fallback" {
		ttl = seasonGamesFallbackTTL
	}
	seasonGamesCache[season] = seasonGamesEntry{games: games, fetchedAt: now, ttl: ttl}
	fmt.Printf("📅 Season %d regular-season length: %d games (source: %s)\n", season, games, source)
	return games
}

func resolveGamesPerSeason(season int) (int, string) {
	games, err := fetchSeasonGamesFromStatsAPI(season)
	if err == nil && games > 0 {
		return games, "nhl-stats-api"
	}
	if err != nil {
		fmt.Printf("⚠️ Season length from stats API failed for %d: %v\n", season, err)
	}

	games, err = fetchSeasonGamesFromSchedule(getSeasonGamesTeamCode(), season)
	if err == nil && games > 0 {
		return games, "club-schedule"
	}
	if err != nil {
		fmt.Printf("⚠️ Season length from club schedule failed for %d: %v\n", season, err)
	}

	return FallbackGamesPerSeason(season), "fallback"
}

// GetCurrentGamesPerSeason returns the regular-season length for the current season.
func GetCurrentGamesPerSeason() int {
	return GetGamesPerSeason(currentSeasonForGames())
}

// GamesRemaining returns totalGames - gamesPlayed, never negative.
func GamesRemaining(totalGames, gamesPlayed int) int {
	remaining := totalGames - gamesPlayed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GamesRemainingForTeam returns the team's remaining regular-season games in the current season.
func GamesRemainingForTeam(team *models.TeamStanding) int {
	if team == nil {
		return 0
	}
	return GamesRemaining(GetCurrentGamesPerSeason(), team.GamesPlayed)
}

// ResetSeasonGamesCache clears cached season lengths.
func ResetSeasonGamesCache() {
	seasonGamesMu.Lock()
	seasonGamesCache = map[int]seasonGamesEntry{}
	seasonGamesMu.Unlock()
}

func fetchSeasonGamesFromStatsAPIImpl(season int) (int, error) {
	url := fmt.Sprintf("https://api.nhle.com/stats/rest/en/season?cayenneExp=id=%d", season)
	body, err := MakeAPICall(url)
	if err != nil {
		return 0, err
	}
	return parseStatsAPISeasonGames(body, season)
}

func parseStatsAPISeasonGames(body []byte, season int) (int, error) {
	var resp struct {
		Data []struct {
			ID            int `json:"id"`
			NumberOfGames int `json:"numberOfGames"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("parse stats season response: %w", err)
	}
	for _, s := range resp.Data {
		if s.ID == season && s.NumberOfGames > 0 {
			return s.NumberOfGames, nil
		}
	}
	return 0, fmt.Errorf("season %d not found in stats API response", season)
}

func fetchSeasonGamesFromScheduleImpl(teamCode string, season int) (int, error) {
	url := fmt.Sprintf("https://api-web.nhle.com/v1/club-schedule-season/%s/%d", teamCode, season)
	body, err := MakeAPICall(url)
	if err != nil {
		return 0, err
	}
	return countRegularSeasonGames(body)
}

func countRegularSeasonGames(body []byte) (int, error) {
	var resp struct {
		Games []struct {
			GameType int `json:"gameType"`
		} `json:"games"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("parse club schedule response: %w", err)
	}
	count := 0
	for _, g := range resp.Games {
		if g.GameType == regularSeasonGameType {
			count++
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("no regular-season games in schedule")
	}
	return count, nil
}
