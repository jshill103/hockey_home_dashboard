package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ShiftAnalysisService analyzes player shifts and line combinations
type ShiftAnalysisService struct {
	httpClient      *http.Client
	cache           map[int]*models.ShiftCache // gameID -> cached analytics
	cacheMu         sync.RWMutex
	teamHistory     map[string]*models.TeamShiftHistory // teamCode -> history
	historyMu       sync.RWMutex
	dataDir         string
	cacheTTL        time.Duration
	systemStatsServ *SystemStatsService
}

var (
	shiftAnalysisService *ShiftAnalysisService
	shiftAnalysisOnce    sync.Once
)

// InitShiftAnalysisService initializes the global shift analysis service
func InitShiftAnalysisService(statsServ *SystemStatsService) {
	shiftAnalysisOnce.Do(func() {
		shiftAnalysisService = NewShiftAnalysisService(statsServ)
		log.Println("✅ Shift Analysis Service initialized")
	})
}

// GetShiftAnalysisService returns the singleton instance
func GetShiftAnalysisService() *ShiftAnalysisService {
	return shiftAnalysisService
}

// NewShiftAnalysisService creates a new shift analysis service
func NewShiftAnalysisService(statsServ *SystemStatsService) *ShiftAnalysisService {
	service := &ShiftAnalysisService{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		cache:           make(map[int]*models.ShiftCache),
		teamHistory:     make(map[string]*models.TeamShiftHistory),
		dataDir:         "data/shifts",
		cacheTTL:        24 * time.Hour,
		systemStatsServ: statsServ,
	}

	// Create data directory
	if err := os.MkdirAll(service.dataDir, 0755); err != nil {
		log.Printf("⚠️ Failed to create shifts directory: %v", err)
	}

	// Load cached team history
	service.loadTeamHistory()

	return service
}

// FetchShiftData fetches and analyzes shift data for a game
func (sas *ShiftAnalysisService) FetchShiftData(gameID int) (*models.ShiftAnalytics, error) {
	// Check cache first
	sas.cacheMu.RLock()
	if cached, exists := sas.cache[gameID]; exists {
		if time.Since(cached.CachedAt) < cached.TTL {
			sas.cacheMu.RUnlock()
			log.Printf("⏱️ Using cached shift data for game %d", gameID)
			return cached.Analytics, nil
		}
	}
	sas.cacheMu.RUnlock()

	// Fetch from NHL API
	log.Printf("📥 Fetching shift data for game %d from NHL API...", gameID)

	startTime := time.Now()

	url := fmt.Sprintf("https://api-web.nhle.com/v1/gamecenter/%d/shifts", gameID)

	// Use MakeAPICall for caching and rate limiting
	body, err := MakeAPICall(url)
	if err != nil {
		// Record failure
		if sas.systemStatsServ != nil {
			sas.systemStatsServ.RecordBackfillFailure()
		}
		return nil, fmt.Errorf("failed to fetch shift data: %w", err)
	}

	var apiResp models.ShiftDataResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// Record failure
		if sas.systemStatsServ != nil {
			sas.systemStatsServ.RecordBackfillFailure()
		}
		return nil, fmt.Errorf("failed to decode shift data: %w", err)
	}

	// Analyze the shift data
	analytics := sas.analyzeShifts(&apiResp)
	if analytics == nil {
		return nil, fmt.Errorf("failed to analyze shift data")
	}
	// Check if analytics structs are properly initialized (they're structs, not pointers)
	if analytics.HomeAnalytics.TeamCode == "" || analytics.AwayAnalytics.TeamCode == "" {
		return nil, fmt.Errorf("failed to analyze shift data: incomplete analytics")
	}

	processingTime := time.Since(startTime)
	shiftsProcessed := analytics.HomeAnalytics.TotalShifts + analytics.AwayAnalytics.TotalShifts

	// Cache the results
	sas.cacheMu.Lock()
	sas.cache[gameID] = &models.ShiftCache{
		GameID:    gameID,
		Analytics: analytics,
		CachedAt:  time.Now(),
		TTL:       sas.cacheTTL,
	}
	sas.cacheMu.Unlock()

	// Update team history
	sas.updateTeamHistory(analytics)

	// Save to disk
	if err := sas.saveAnalytics(analytics); err != nil {
		log.Printf("⚠️ Failed to save shift analytics: %v", err)
	}

	// Record successful processing
	if sas.systemStatsServ != nil {
		sas.systemStatsServ.RecordBackfillGame("shift-data", shiftsProcessed, processingTime)
	}

	log.Printf("✅ Analyzed %d total shifts for game %d", shiftsProcessed, gameID)
	return analytics, nil
}

// analyzeShifts processes raw shift data into analytics
func (sas *ShiftAnalysisService) analyzeShifts(data *models.ShiftDataResponse) *models.ShiftAnalytics {
	gameDate, _ := time.Parse("2006-01-02", data.GameDate)

	analytics := &models.ShiftAnalytics{
		GameID:        data.GameID,
		Season:        data.Season,
		GameDate:      gameDate,
		HomeTeam:      data.HomeTeam.Abbrev,
		AwayTeam:      data.AwayTeam.Abbrev,
		HomeAnalytics: models.TeamShiftAnalytics{TeamCode: data.HomeTeam.Abbrev},
		AwayAnalytics: models.TeamShiftAnalytics{TeamCode: data.AwayTeam.Abbrev},
		ProcessedAt:   time.Now(),
		DataSource:    "NHLE_API_v1_Shifts",
	}

	// Analyze home team
	sas.analyzeTeamShifts(&analytics.HomeAnalytics, data.HomeTeam.Players)

	// Analyze away team
	sas.analyzeTeamShifts(&analytics.AwayAnalytics, data.AwayTeam.Players)

	return analytics
}

// analyzeTeamShifts processes shift data for a single team
func (sas *ShiftAnalysisService) analyzeTeamShifts(teamAnalytics *models.TeamShiftAnalytics, players []models.PlayerShift) {
	teamAnalytics.PlayersUsed = len(players)
	teamAnalytics.PlayerShiftStats = make([]models.PlayerShiftStats, 0, len(players))

	var totalTeamShifts int
	var totalShiftDuration float64
	var topPlayerMinutes float64
	var topPlayerShifts int

	for _, player := range players {
		if len(player.Shifts) == 0 {
			continue
		}

		stats := models.PlayerShiftStats{
			PlayerID:    player.PlayerID,
			PlayerName:  fmt.Sprintf("%s %s", player.FirstName, player.LastName),
			TotalShifts: len(player.Shifts),
		}

		var playerTotalTime float64
		var longestShift float64
		var shortestShift float64 = 999.0
		var longShiftCount int
		var veryLongShiftCount int
		var shortShiftCount int

		for _, shift := range player.Shifts {
			duration := sas.parseDuration(shift.Duration)
			playerTotalTime += duration

			totalShiftDuration += duration
			totalTeamShifts++

			// Track longest/shortest
			if duration > longestShift {
				longestShift = duration
			}
			if duration < shortestShift {
				shortestShift = duration
			}

			// Categorize shifts
			if duration > 90 {
				veryLongShiftCount++
				teamAnalytics.VeryLongShifts++
			} else if duration > 60 {
				longShiftCount++
				teamAnalytics.LongShifts++
			} else if duration < 30 {
				shortShiftCount++
				teamAnalytics.ShortShifts++
			}
		}

		stats.TotalIceTime = playerTotalTime / 60.0 // Convert to minutes
		stats.AvgShiftLength = playerTotalTime / float64(len(player.Shifts))
		stats.LongestShift = longestShift
		stats.ShortestShift = shortestShift
		stats.UsageRate = (stats.TotalIceTime / 60.0) * 100 // % of game (60 min)

		// Calculate fatigue score (based on long shifts and total ice time)
		fatigueFromLongShifts := float64(longShiftCount+veryLongShiftCount*2) / float64(len(player.Shifts))
		fatigueFromTotalTime := math.Min(stats.TotalIceTime/30.0, 1.0) // Cap at 30 minutes
		stats.FatigueScore = (fatigueFromLongShifts + fatigueFromTotalTime) / 2.0

		teamAnalytics.PlayerShiftStats = append(teamAnalytics.PlayerShiftStats, stats)

		// Track top player
		if stats.TotalIceTime > topPlayerMinutes {
			topPlayerMinutes = stats.TotalIceTime
			topPlayerShifts = stats.TotalShifts
		}
	}

	// Calculate team-level metrics
	teamAnalytics.TotalShifts = totalTeamShifts
	if totalTeamShifts > 0 {
		teamAnalytics.AvgShiftLength = totalShiftDuration / float64(totalTeamShifts)
	}
	teamAnalytics.TotalIceTime = totalShiftDuration / 60.0 // Minutes
	teamAnalytics.TopPlayerMinutes = topPlayerMinutes
	teamAnalytics.TopPlayerShifts = topPlayerShifts

	// Sort players by ice time
	sort.Slice(teamAnalytics.PlayerShiftStats, func(i, j int) bool {
		return teamAnalytics.PlayerShiftStats[i].TotalIceTime > teamAnalytics.PlayerShiftStats[j].TotalIceTime
	})

	// Detect line combinations (simplified - just track top TOI players together)
	// In a real implementation, you'd analyze overlapping shifts
	teamAnalytics.TopLineCombos = sas.detectLineCombinations(players)
}

// parseDuration converts NHL shift duration string (e.g., "1:23") to seconds
func (sas *ShiftAnalysisService) parseDuration(duration string) float64 {
	parts := strings.Split(duration, ":")
	if len(parts) != 2 {
		return 0
	}

	minutes, _ := strconv.Atoi(parts[0])
	seconds, _ := strconv.Atoi(parts[1])

	return float64(minutes*60 + seconds)
}

// detectLineCombinations identifies frequent line combinations (simplified)
func (sas *ShiftAnalysisService) detectLineCombinations(players []models.PlayerShift) []models.LineCombination {
	// Simplified: Just return top 3 players by ice time as "top line"
	// Real implementation would analyze overlapping shifts

	// Sort by total ice time
	playerTimes := make([]struct {
		PlayerID  int
		Name      string
		TotalTime float64
	}, 0, len(players))

	for _, player := range players {
		var totalTime float64
		for _, shift := range player.Shifts {
			totalTime += sas.parseDuration(shift.Duration)
		}
		playerTimes = append(playerTimes, struct {
			PlayerID  int
			Name      string
			TotalTime float64
		}{
			PlayerID:  player.PlayerID,
			Name:      fmt.Sprintf("%s %s", player.FirstName, player.LastName),
			TotalTime: totalTime,
		})
	}

	sort.Slice(playerTimes, func(i, j int) bool {
		return playerTimes[i].TotalTime > playerTimes[j].TotalTime
	})

	// Create "top line" from top 3 players
	if len(playerTimes) >= 3 {
		topLine := models.LineCombination{
			Players:        []string{strconv.Itoa(playerTimes[0].PlayerID), strconv.Itoa(playerTimes[1].PlayerID), strconv.Itoa(playerTimes[2].PlayerID)},
			PlayerNames:    []string{playerTimes[0].Name, playerTimes[1].Name, playerTimes[2].Name},
			TimeTogether:   (playerTimes[0].TotalTime + playerTimes[1].TotalTime + playerTimes[2].TotalTime) / 180.0, // Avg minutes
			ChemistryScore: 0.8,                                                                                      // Placeholder
		}
		return []models.LineCombination{topLine}
	}

	return []models.LineCombination{}
}

// GetTeamHistory returns shift history for a team
func (sas *ShiftAnalysisService) GetTeamHistory(teamCode string) *models.TeamShiftHistory {
	sas.historyMu.RLock()
	defer sas.historyMu.RUnlock()

	if history, exists := sas.teamHistory[teamCode]; exists {
		return history
	}
	return nil
}

// updateTeamHistory updates rolling shift history for teams
func (sas *ShiftAnalysisService) updateTeamHistory(analytics *models.ShiftAnalytics) {
	sas.updateTeamHistoryForSide(analytics.HomeTeam, &analytics.HomeAnalytics, analytics.Season)
	sas.updateTeamHistoryForSide(analytics.AwayTeam, &analytics.AwayAnalytics, analytics.Season)
}

// updateTeamHistoryForSide updates history for one team
func (sas *ShiftAnalysisService) updateTeamHistoryForSide(teamCode string, game *models.TeamShiftAnalytics, season int) {
	sas.historyMu.Lock()
	defer sas.historyMu.Unlock()

	history, exists := sas.teamHistory[teamCode]
	if !exists {
		history = &models.TeamShiftHistory{
			TeamCode: teamCode,
			Season:   season,
		}
		sas.teamHistory[teamCode] = history
	}

	// Update game count
	history.GamesAnalyzed++
	history.LastUpdated = time.Now()

	// Update rolling averages (last 10 games)
	weight := 1.0 / float64(minInt(history.GamesAnalyzed, 10))

	history.AvgShiftLength += (game.AvgShiftLength - history.AvgShiftLength) * weight
	history.AvgPlayersUsed += (float64(game.PlayersUsed) - history.AvgPlayersUsed) * weight
	history.AvgLongShifts += (float64(game.LongShifts+game.VeryLongShifts) - history.AvgLongShifts) * weight

	// Update top line minutes
	if game.TopPlayerMinutes > 0 {
		history.TopLineMinutes += (game.TopPlayerMinutes - history.TopLineMinutes) * weight
	}

	// Coaching tendencies (simple heuristics)
	history.ShortBench = history.AvgPlayersUsed < 17
	history.BalancedLines = history.TopLineMinutes < 22

	// Save to disk
	if err := sas.saveTeamHistory(); err != nil {
		log.Printf("⚠️ Failed to save team history: %v", err)
	}
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// saveAnalytics saves shift analytics to disk
func (sas *ShiftAnalysisService) saveAnalytics(analytics *models.ShiftAnalytics) error {
	filename := filepath.Join(sas.dataDir, fmt.Sprintf("game_%d.json", analytics.GameID))

	data, err := json.MarshalIndent(analytics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analytics: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write analytics file: %w", err)
	}

	return nil
}

// saveTeamHistory saves team history to disk
func (sas *ShiftAnalysisService) saveTeamHistory() error {
	filename := filepath.Join(sas.dataDir, "team_history.json")

	data, err := json.MarshalIndent(sas.teamHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal team history: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write team history file: %w", err)
	}

	return nil
}

// loadTeamHistory loads team history from disk
func (sas *ShiftAnalysisService) loadTeamHistory() {
	filename := filepath.Join(sas.dataDir, "team_history.json")

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("📂 No existing shift history found (will create on first update)")
		return
	}

	var history map[string]*models.TeamShiftHistory
	if err := json.Unmarshal(data, &history); err != nil {
		log.Printf("⚠️ Failed to unmarshal team history: %v", err)
		return
	}

	sas.historyMu.Lock()
	sas.teamHistory = history
	sas.historyMu.Unlock()

	log.Printf("📂 Loaded shift history for %d teams", len(history))
}
