package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// PreGameLineupService manages pre-game lineup data
type PreGameLineupService struct {
	teamCode      string
	lineupsCache  map[int]*models.LineupCache // gameID -> cached lineup
	cacheMu       sync.RWMutex
	httpClient    *http.Client
	cacheTTL      time.Duration
	dataDir       string
	monitorTicker *time.Ticker
	stopChan      chan bool
}

var (
	preGameLineupService *PreGameLineupService
	preGameLineupOnce    sync.Once
)

// InitPreGameLineupService initializes the global pre-game lineup service
func InitPreGameLineupService(teamCode string) {
	preGameLineupOnce.Do(func() {
		preGameLineupService = NewPreGameLineupService(teamCode)
		log.Println("✅ Pre-Game Lineup Service initialized")
	})
}

// GetPreGameLineupService returns the singleton instance
func GetPreGameLineupService() *PreGameLineupService {
	return preGameLineupService
}

// NewPreGameLineupService creates a new pre-game lineup service
func NewPreGameLineupService(teamCode string) *PreGameLineupService {
	service := &PreGameLineupService{
		teamCode:     teamCode,
		lineupsCache: make(map[int]*models.LineupCache),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		cacheTTL: 15 * time.Minute, // Refresh every 15 minutes when monitoring
		dataDir:  "data/lineups",
		stopChan: make(chan bool),
	}

	// Create data directory
	if err := os.MkdirAll(service.dataDir, 0755); err != nil {
		log.Printf("⚠️ Failed to create lineups directory: %v", err)
	}

	// Load cached lineups from disk
	service.loadLineupsFromDisk()

	// Start background monitoring for upcoming games
	service.startMonitoring()

	return service
}

// startMonitoring starts background monitoring for upcoming games
func (pgls *PreGameLineupService) startMonitoring() {
	// Check for lineups every 30 minutes
	pgls.monitorTicker = time.NewTicker(30 * time.Minute)

	go func() {
		log.Println("🔍 Pre-game lineup monitoring started")
		// Run initial check
		pgls.checkUpcomingGames()

		for {
			select {
			case <-pgls.monitorTicker.C:
				pgls.checkUpcomingGames()
			case <-pgls.stopChan:
				log.Println("⏹️ Pre-game lineup monitoring stopped")
				return
			}
		}
	}()
}

// StopMonitoring stops the background monitoring
func (pgls *PreGameLineupService) StopMonitoring() {
	if pgls.monitorTicker != nil {
		pgls.monitorTicker.Stop()
	}
	pgls.stopChan <- true
}

// checkUpcomingGames checks upcoming games and fetches lineups if available
func (pgls *PreGameLineupService) checkUpcomingGames() {
	log.Println("🔍 Checking for upcoming game lineups...")

	// Get next game from schedule
	games, err := GetTeamUpcomingGames(pgls.teamCode)
	if err != nil {
		log.Printf("⚠️ Failed to fetch schedule: %v", err)
		return
	}

	if len(games) == 0 {
		log.Println("📅 No upcoming games found")
		return
	}

	// Check next 2 games (in case of back-to-backs)
	for i := 0; i < len(games) && i < 2; i++ {
		game := games[i]
		gameTime, err := time.Parse(time.RFC3339, game.StartTime)
		if err != nil {
			continue
		}

		// Only fetch lineups for games within the next 6 hours
		timeUntilGame := time.Until(gameTime)
		if timeUntilGame > 0 && timeUntilGame <= 6*time.Hour {
			log.Printf("⏰ Game %d starts in %.1f hours, fetching lineup...", game.ID, timeUntilGame.Hours())

			_, err := pgls.FetchLineup(game.ID)
			if err != nil {
				log.Printf("⚠️ Lineup not yet available for game %d: %v", game.ID, err)
			} else {
				log.Printf("✅ Lineup fetched for game %d", game.ID)
			}
		}
	}
}

// FetchLineup fetches the pre-game lineup for a specific game
func (pgls *PreGameLineupService) FetchLineup(gameID int) (*models.PreGameLineup, error) {
	// Check cache first
	pgls.cacheMu.RLock()
	if cached, exists := pgls.lineupsCache[gameID]; exists {
		if time.Since(cached.CachedAt) < cached.TTL {
			pgls.cacheMu.RUnlock()
			log.Printf("🏒 Using cached lineup for game %d", gameID)
			return cached.Lineup, nil
		}
	}
	pgls.cacheMu.RUnlock()

	// Fetch from NHL API
	log.Printf("📥 Fetching lineup for game %d from NHL API...", gameID)

	url := fmt.Sprintf("https://api-web.nhle.com/v1/gamecenter/%d/boxscore", gameID)

	// Use rate limiter if available
	rateLimiter := GetNHLRateLimiter()
	if rateLimiter != nil {
		rateLimiter.Wait()
	}

	resp, err := pgls.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lineup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d (lineup may not be available yet)", resp.StatusCode)
	}

	var apiResp models.GameCenterLineupResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode lineup: %w", err)
	}

	// Parse and transform the lineup
	lineup := pgls.parseLineup(&apiResp)

	// Cache the lineup
	pgls.cacheMu.Lock()
	pgls.lineupsCache[gameID] = &models.LineupCache{
		Lineup:   lineup,
		CachedAt: time.Now(),
		TTL:      pgls.cacheTTL,
	}
	pgls.cacheMu.Unlock()

	// Save to disk
	if err := pgls.saveLineupToDisk(lineup); err != nil {
		log.Printf("⚠️ Failed to save lineup to disk: %v", err)
	}

	return lineup, nil
}

// parseLineup parses the NHL API response into our lineup model
func (pgls *PreGameLineupService) parseLineup(apiResp *models.GameCenterLineupResponse) *models.PreGameLineup {
	gameDate, _ := time.Parse("2006-01-02", apiResp.GameDate)

	lineup := &models.PreGameLineup{
		GameID:      apiResp.ID,
		GameDate:    gameDate,
		HomeTeam:    apiResp.HomeTeam.Abbrev,
		AwayTeam:    apiResp.AwayTeam.Abbrev,
		IsAvailable: len(apiResp.RosterSpots) > 0,
		LastUpdated: time.Now(),
		DataSource:  "NHLE_API_v1",
	}

	// Parse home team lineup
	if lineup.IsAvailable {
		lineup.HomeLineup = pgls.parseTeamLineup(apiResp, apiResp.HomeTeam.ID, true)
		lineup.AwayLineup = pgls.parseTeamLineup(apiResp, apiResp.AwayTeam.ID, false)
	}

	return lineup
}

// parseTeamLineup parses lineup data for a specific team
func (pgls *PreGameLineupService) parseTeamLineup(apiResp *models.GameCenterLineupResponse, teamID int, isHome bool) *models.TeamLineup {
	teamCode := apiResp.HomeTeam.Abbrev
	if !isHome {
		teamCode = apiResp.AwayTeam.Abbrev
	}

	teamLineup := &models.TeamLineup{
		TeamCode:     teamCode,
		ForwardLines: []models.ForwardLine{},
		DefensePairs: []models.DefensePair{},
		Scratches:    []models.ScratchedPlayer{},
		ExtraSkaters: []models.LineupPlayer{},
	}

	// Parse goalies from summary if available
	if apiResp.Summary != nil && len(apiResp.Summary.Goalies) > 0 {
		for _, goalie := range apiResp.Summary.Goalies {
			if goalie.TeamID == teamID {
				goalieInfo := &models.LineupGoalie{
					PlayerID:      goalie.PlayerID,
					PlayerName:    fmt.Sprintf("%s %s", goalie.FirstName.Default, goalie.LastName.Default),
					SweaterNumber: goalie.SweaterNumber,
					IsStarting:    goalie.IsStarter,
				}

				if goalie.IsStarter {
					teamLineup.StartingGoalie = goalieInfo
				} else {
					teamLineup.BackupGoalie = goalieInfo
				}
			}
		}
	}

	// Parse roster spots
	var forwards []models.LineupPlayer
	var defensemen []models.LineupPlayer

	for _, spot := range apiResp.RosterSpots {
		if spot.TeamID != teamID {
			continue
		}

		player := models.LineupPlayer{
			PlayerID:      spot.PlayerID,
			PlayerName:    fmt.Sprintf("%s %s", spot.FirstName.Default, spot.LastName.Default),
			SweaterNumber: spot.SweaterNumber,
			Position:      spot.Position,
		}

		switch spot.Position {
		case "C", "L", "LW", "R", "RW":
			forwards = append(forwards, player)
		case "D":
			defensemen = append(defensemen, player)
		}
	}

	// Organize forwards into lines (4 lines of 3 players each)
	// Note: Without explicit line data, we just group them sequentially
	for i := 0; i < len(forwards) && i < 12; i += 3 {
		line := models.ForwardLine{
			LineNumber: (i / 3) + 1,
		}
		if i < len(forwards) {
			line.LeftWing = &forwards[i]
		}
		if i+1 < len(forwards) {
			line.Center = &forwards[i+1]
		}
		if i+2 < len(forwards) {
			line.RightWing = &forwards[i+2]
		}
		teamLineup.ForwardLines = append(teamLineup.ForwardLines, line)
	}

	// Extra forwards (beyond 12)
	if len(forwards) > 12 {
		teamLineup.ExtraSkaters = forwards[12:]
	}

	// Organize defensemen into pairs (3 pairs of 2 players each)
	for i := 0; i < len(defensemen) && i < 6; i += 2 {
		pair := models.DefensePair{
			PairNumber: (i / 2) + 1,
		}
		if i < len(defensemen) {
			pair.LeftDefense = &defensemen[i]
		}
		if i+1 < len(defensemen) {
			pair.RightDefense = &defensemen[i+1]
		}
		teamLineup.DefensePairs = append(teamLineup.DefensePairs, pair)
	}

	// Parse scratches if available
	if apiResp.Scratches != nil {
		scratchKey := "away"
		if isHome {
			scratchKey = "home"
		}

		if scratchedIDs, exists := apiResp.Scratches[scratchKey]; exists {
			for _, playerID := range scratchedIDs {
				// Find player in roster spots
				for _, spot := range apiResp.RosterSpots {
					if spot.PlayerID == playerID {
						scratch := models.ScratchedPlayer{
							PlayerID:   spot.PlayerID,
							PlayerName: fmt.Sprintf("%s %s", spot.FirstName.Default, spot.LastName.Default),
							Position:   spot.Position,
							Reason:     "Scratch", // API doesn't provide reason
						}
						teamLineup.Scratches = append(teamLineup.Scratches, scratch)
						break
					}
				}
			}
		}
	}

	return teamLineup
}

// GetLineup retrieves a cached lineup or fetches it if needed
func (pgls *PreGameLineupService) GetLineup(gameID int) (*models.PreGameLineup, error) {
	return pgls.FetchLineup(gameID)
}

// GetStartingGoalie returns the starting goalie for a team in a specific game
func (pgls *PreGameLineupService) GetStartingGoalie(gameID int, teamCode string) (*models.LineupGoalie, error) {
	lineup, err := pgls.GetLineup(gameID)
	if err != nil {
		return nil, err
	}

	if !lineup.IsAvailable {
		return nil, fmt.Errorf("lineup not yet available for game %d", gameID)
	}

	if lineup.HomeTeam == teamCode && lineup.HomeLineup != nil {
		return lineup.HomeLineup.StartingGoalie, nil
	}

	if lineup.AwayTeam == teamCode && lineup.AwayLineup != nil {
		return lineup.AwayLineup.StartingGoalie, nil
	}

	return nil, fmt.Errorf("team %s not found in game %d", teamCode, gameID)
}

// IsLineupAvailable checks if lineup data is available for a game
func (pgls *PreGameLineupService) IsLineupAvailable(gameID int) bool {
	lineup, err := pgls.GetLineup(gameID)
	if err != nil {
		return false
	}
	return lineup.IsAvailable
}

// saveLineupToDisk saves lineup to disk for persistence
func (pgls *PreGameLineupService) saveLineupToDisk(lineup *models.PreGameLineup) error {
	filename := filepath.Join(pgls.dataDir, fmt.Sprintf("game_%d.json", lineup.GameID))

	data, err := json.MarshalIndent(lineup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lineup: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write lineup file: %w", err)
	}

	log.Printf("💾 Saved lineup for game %d to disk", lineup.GameID)
	return nil
}

// loadLineupsFromDisk loads cached lineups from disk
func (pgls *PreGameLineupService) loadLineupsFromDisk() {
	files, err := os.ReadDir(pgls.dataDir)
	if err != nil {
		log.Printf("⚠️ Could not read lineups directory: %v", err)
		return
	}

	loaded := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(pgls.dataDir, file.Name()))
		if err != nil {
			continue
		}

		var lineup models.PreGameLineup
		if err := json.Unmarshal(data, &lineup); err != nil {
			continue
		}

		// Only cache lineups for future games or games from today
		if lineup.GameDate.After(time.Now().Add(-24 * time.Hour)) {
			pgls.cacheMu.Lock()
			pgls.lineupsCache[lineup.GameID] = &models.LineupCache{
				Lineup:   &lineup,
				CachedAt: lineup.LastUpdated,
				TTL:      pgls.cacheTTL,
			}
			pgls.cacheMu.Unlock()
			loaded++
		}
	}

	if loaded > 0 {
		log.Printf("📂 Loaded %d lineup(s) from disk", loaded)
	}
}

// ClearOldLineups removes old lineup data from cache and disk
func (pgls *PreGameLineupService) ClearOldLineups() error {
	cutoff := time.Now().Add(-7 * 24 * time.Hour) // Keep lineups for 7 days

	// Clear from cache
	pgls.cacheMu.Lock()
	for gameID, cached := range pgls.lineupsCache {
		if cached.Lineup.GameDate.Before(cutoff) {
			delete(pgls.lineupsCache, gameID)
		}
	}
	pgls.cacheMu.Unlock()

	// Clear from disk
	files, err := os.ReadDir(pgls.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read lineups directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(pgls.dataDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var lineup models.PreGameLineup
		if err := json.Unmarshal(data, &lineup); err != nil {
			continue
		}

		if lineup.GameDate.Before(cutoff) {
			os.Remove(filePath)
			log.Printf("🗑️ Removed old lineup file: %s", file.Name())
		}
	}

	return nil
}
