package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// RosterValidationService handles team roster fetching and validation
type RosterValidationService struct {
	rosters   map[string]*models.TeamRoster // "TEAM_SEASON" -> Roster
	changes   []models.RosterChange
	dataDir   string
	mutex     sync.RWMutex
	lastFetch time.Time
	cacheTTL  time.Duration
}

// NewRosterValidationService creates a new roster validation service
func NewRosterValidationService() *RosterValidationService {
	service := &RosterValidationService{
		rosters:  make(map[string]*models.TeamRoster),
		changes:  []models.RosterChange{},
		dataDir:  "data/rosters",
		cacheTTL: 24 * time.Hour, // Cache rosters for 24 hours
	}

	// Create data directory
	os.MkdirAll(service.dataDir, 0755)

	// Load existing rosters
	if err := service.loadRosters(); err != nil {
		log.Printf("⚠️ Could not load roster data: %v (starting fresh)", err)
	} else {
		log.Printf("🏒 Loaded roster data for %d teams", len(service.rosters))
	}

	return service
}

// FetchRoster fetches and caches a team's roster for a specific season
func (rvs *RosterValidationService) FetchRoster(teamCode string, season int) (*models.TeamRoster, error) {
	key := fmt.Sprintf("%s_%d", teamCode, season)

	// Check cache first
	rvs.mutex.RLock()
	if cached, exists := rvs.rosters[key]; exists {
		// Use cache if less than TTL old
		if time.Since(cached.LastUpdated) < rvs.cacheTTL {
			rvs.mutex.RUnlock()
			log.Printf("🏒 Using cached roster for %s (season %d)", teamCode, season)
			return cached, nil
		}
	}
	rvs.mutex.RUnlock()

	// Fetch from API
	log.Printf("🏒 Fetching roster for %s (season %d) from NHL API...", teamCode, season)

	url := fmt.Sprintf("https://api-web.nhle.com/v1/roster/%s/%d", teamCode, season)
	body, err := MakeAPICall(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roster: %w", err)
	}

	var rosterResp models.TeamRosterResponse
	if err := json.Unmarshal(body, &rosterResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roster: %w", err)
	}

	// Build TeamRoster with quick lookup map
	roster := &models.TeamRoster{
		TeamCode:    teamCode,
		Season:      season,
		Forwards:    rosterResp.Forwards,
		Defensemen:  rosterResp.Defensemen,
		Goalies:     rosterResp.Goalies,
		PlayerIDs:   make(map[int]bool),
		LastUpdated: time.Now(),
		Version:     "1.0",
	}

	// Populate player ID lookup map
	for _, p := range roster.Forwards {
		roster.PlayerIDs[p.ID] = true
	}
	for _, p := range roster.Defensemen {
		roster.PlayerIDs[p.ID] = true
	}
	for _, p := range roster.Goalies {
		roster.PlayerIDs[p.ID] = true
	}

	log.Printf("🏒 Fetched roster: %d forwards, %d defensemen, %d goalies",
		len(roster.Forwards), len(roster.Defensemen), len(roster.Goalies))

	// Cache roster
	rvs.mutex.Lock()
	rvs.rosters[key] = roster
	rvs.mutex.Unlock()

	// Save to disk
	if err := rvs.saveRosters(); err != nil {
		log.Printf("⚠️ Failed to save roster data: %v", err)
	}

	return roster, nil
}

// IsOnRoster checks if a player ID is on the team's current roster
func (rvs *RosterValidationService) IsOnRoster(teamCode string, season int, playerID int) bool {
	roster, err := rvs.FetchRoster(teamCode, season)
	if err != nil {
		log.Printf("⚠️ Could not validate roster for %s: %v", teamCode, err)
		return true // Assume valid if we can't fetch roster
	}

	return roster.IsOnRoster(playerID)
}

// ValidatePlayerIDs checks which player IDs are on the current roster
func (rvs *RosterValidationService) ValidatePlayerIDs(teamCode string, season int, playerIDs []int) (valid []int, invalid []int) {
	roster, err := rvs.FetchRoster(teamCode, season)
	if err != nil {
		log.Printf("⚠️ Could not validate roster for %s: %v", teamCode, err)
		return playerIDs, []int{} // Assume all valid if we can't fetch roster
	}

	for _, playerID := range playerIDs {
		if roster.IsOnRoster(playerID) {
			valid = append(valid, playerID)
		} else {
			invalid = append(invalid, playerID)
			// Try to get player name for logging
			player := roster.GetPlayerByID(playerID)
			if player != nil {
				playerName := player.FirstName.Default + " " + player.LastName.Default
				log.Printf("⚠️ Player %s (ID %d) not on current %s roster", playerName, playerID, teamCode)
			} else {
				log.Printf("⚠️ Player ID %d not on current %s roster", playerID, teamCode)
			}
		}
	}

	return valid, invalid
}

// DetectRosterChanges compares two seasons and identifies roster changes
func (rvs *RosterValidationService) DetectRosterChanges(teamCode string, oldSeason, newSeason int) ([]models.RosterChange, error) {
	oldRoster, err := rvs.FetchRoster(teamCode, oldSeason)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch old roster: %w", err)
	}

	newRoster, err := rvs.FetchRoster(teamCode, newSeason)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch new roster: %w", err)
	}

	var changes []models.RosterChange

	// Find players removed (in old but not in new)
	for playerID := range oldRoster.PlayerIDs {
		if !newRoster.PlayerIDs[playerID] {
			player := oldRoster.GetPlayerByID(playerID)
			if player != nil {
				playerName := player.FirstName.Default + " " + player.LastName.Default
				changes = append(changes, models.RosterChange{
					PlayerID:   playerID,
					PlayerName: playerName,
					Position:   player.Position,
					ChangeType: "removed",
					FromSeason: oldSeason,
					ToSeason:   newSeason,
					DetectedAt: time.Now(),
				})
			}
		}
	}

	// Find players added (in new but not in old)
	for playerID := range newRoster.PlayerIDs {
		if !oldRoster.PlayerIDs[playerID] {
			player := newRoster.GetPlayerByID(playerID)
			if player != nil {
				playerName := player.FirstName.Default + " " + player.LastName.Default
				changes = append(changes, models.RosterChange{
					PlayerID:   playerID,
					PlayerName: playerName,
					Position:   player.Position,
					ChangeType: "added",
					FromSeason: oldSeason,
					ToSeason:   newSeason,
					DetectedAt: time.Now(),
				})
			}
		}
	}

	// Log changes
	if len(changes) > 0 {
		log.Printf("🏒 Roster changes detected for %s (%d -> %d): %d additions, %d departures",
			teamCode, oldSeason, newSeason,
			countByType(changes, "added"),
			countByType(changes, "removed"))
	}

	// Store changes
	rvs.mutex.Lock()
	rvs.changes = append(rvs.changes, changes...)
	rvs.mutex.Unlock()

	return changes, nil
}

// countByType counts roster changes by type
func countByType(changes []models.RosterChange, changeType string) int {
	count := 0
	for _, c := range changes {
		if c.ChangeType == changeType {
			count++
		}
	}
	return count
}

// GetRoster returns a cached roster if available
func (rvs *RosterValidationService) GetRoster(teamCode string, season int) (*models.TeamRoster, error) {
	key := fmt.Sprintf("%s_%d", teamCode, season)

	rvs.mutex.RLock()
	defer rvs.mutex.RUnlock()

	if roster, exists := rvs.rosters[key]; exists {
		return roster, nil
	}

	return nil, fmt.Errorf("roster not found for %s (season %d)", teamCode, season)
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// saveRosters saves all rosters to disk
func (rvs *RosterValidationService) saveRosters() error {
	rvs.mutex.RLock()
	defer rvs.mutex.RUnlock()

	filePath := filepath.Join(rvs.dataDir, "rosters.json")

	data := struct {
		Rosters     map[string]*models.TeamRoster `json:"rosters"`
		Changes     []models.RosterChange         `json:"changes"`
		LastUpdated time.Time                     `json:"lastUpdated"`
		Version     string                        `json:"version"`
	}{
		Rosters:     rvs.rosters,
		Changes:     rvs.changes,
		LastUpdated: time.Now(),
		Version:     "1.0",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling roster data: %w", err)
	}

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing roster data: %w", err)
	}

	return nil
}

// loadRosters loads rosters from disk
func (rvs *RosterValidationService) loadRosters() error {
	filePath := filepath.Join(rvs.dataDir, "rosters.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("roster file not found")
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading roster file: %w", err)
	}

	var data struct {
		Rosters     map[string]*models.TeamRoster `json:"rosters"`
		Changes     []models.RosterChange         `json:"changes"`
		LastUpdated time.Time                     `json:"lastUpdated"`
		Version     string                        `json:"version"`
	}

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return fmt.Errorf("error unmarshaling roster data: %w", err)
	}

	rvs.rosters = data.Rosters
	rvs.changes = data.Changes

	return nil
}

// ============================================================================
// GLOBAL SERVICE
// ============================================================================

var (
	globalRosterService *RosterValidationService
	rosterServiceMutex  sync.Mutex
)

// InitializeRosterValidationService initializes the global roster validation service
func InitializeRosterValidationService() error {
	rosterServiceMutex.Lock()
	defer rosterServiceMutex.Unlock()

	if globalRosterService != nil {
		return fmt.Errorf("roster validation service already initialized")
	}

	globalRosterService = NewRosterValidationService()
	log.Printf("🏒 Roster Validation Service initialized")

	return nil
}

// GetRosterValidationService returns the global roster validation service
func GetRosterValidationService() *RosterValidationService {
	rosterServiceMutex.Lock()
	defer rosterServiceMutex.Unlock()
	return globalRosterService
}
