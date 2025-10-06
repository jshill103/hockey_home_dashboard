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

// GameSummaryService analyzes game summary data for enhanced game context
type GameSummaryService struct {
	httpClient  *http.Client
	cache       map[int]*models.GameSummaryCache // gameID -> cached analytics
	cacheMu     sync.RWMutex
	teamHistory map[string]*models.TeamSummaryHistory // teamCode -> history
	historyMu   sync.RWMutex
	dataDir     string
	cacheTTL    time.Duration
}

var (
	gameSummaryService *GameSummaryService
	gameSummaryOnce    sync.Once
)

// InitGameSummaryService initializes the global game summary service
func InitGameSummaryService() {
	gameSummaryOnce.Do(func() {
		gameSummaryService = NewGameSummaryService()
		log.Println("✅ Game Summary Analytics Service initialized")
	})
}

// GetGameSummaryService returns the singleton instance
func GetGameSummaryService() *GameSummaryService {
	return gameSummaryService
}

// NewGameSummaryService creates a new game summary service
func NewGameSummaryService() *GameSummaryService {
	service := &GameSummaryService{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		cache:       make(map[int]*models.GameSummaryCache),
		teamHistory: make(map[string]*models.TeamSummaryHistory),
		dataDir:     "data/game_summary",
		cacheTTL:    24 * time.Hour,
	}

	// Create data directory
	if err := os.MkdirAll(service.dataDir, 0755); err != nil {
		log.Printf("⚠️ Failed to create game summary directory: %v", err)
	}

	// Load cached team history
	service.loadTeamHistory()

	return service
}

// FetchGameSummaryData fetches and analyzes game summary data for a game
func (gss *GameSummaryService) FetchGameSummaryData(gameID int) (*models.GameSummaryAnalytics, error) {
	// Check cache first
	gss.cacheMu.RLock()
	if cached, exists := gss.cache[gameID]; exists {
		if time.Since(cached.CachedAt) < cached.TTL {
			gss.cacheMu.RUnlock()
			log.Printf("📋 Using cached game summary data for game %d", gameID)
			return cached.Analytics, nil
		}
	}
	gss.cacheMu.RUnlock()

	// Fetch from NHL API
	log.Printf("📥 Fetching game summary data for game %d from NHL API...", gameID)

	url := fmt.Sprintf("https://api-web.nhle.com/v1/gamecenter/%d/summary", gameID)

	// Use rate limiter
	rateLimiter := GetNHLRateLimiter()
	if rateLimiter != nil {
		rateLimiter.Wait()
	}

	resp, err := gss.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game summary data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp models.GameSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode game summary data: %w", err)
	}

	// Analyze the game summary data
	analytics := gss.analyzeGameSummary(&apiResp)

	// Cache the results
	gss.cacheMu.Lock()
	gss.cache[gameID] = &models.GameSummaryCache{
		GameID:    gameID,
		Analytics: analytics,
		CachedAt:  time.Now(),
		TTL:       gss.cacheTTL,
	}
	gss.cacheMu.Unlock()

	// Update team history
	gss.updateTeamHistory(analytics)

	// Save to disk
	if err := gss.saveAnalytics(analytics); err != nil {
		log.Printf("⚠️ Failed to save game summary analytics: %v", err)
	}

	log.Printf("✅ Analyzed game summary data for game %d: %s vs %s",
		gameID, analytics.HomeTeam, analytics.AwayTeam)
	return analytics, nil
}

// analyzeGameSummary processes raw game summary data into analytics
func (gss *GameSummaryService) analyzeGameSummary(data *models.GameSummaryResponse) *models.GameSummaryAnalytics {
	gameDate, _ := time.Parse("2006-01-02", data.GameDate)

	analytics := &models.GameSummaryAnalytics{
		GameID:        data.ID,
		Season:        data.Season,
		GameDate:      gameDate,
		HomeTeam:      data.HomeTeam.Abbrev,
		AwayTeam:      data.AwayTeam.Abbrev,
		HomeAnalytics: models.TeamSummaryAnalytics{TeamCode: data.HomeTeam.Abbrev},
		AwayAnalytics: models.TeamSummaryAnalytics{TeamCode: data.AwayTeam.Abbrev},
		ProcessedAt:   time.Now(),
		DataSource:    "NHLE_API_v1_Summary",
	}

	// Analyze home team
	gss.analyzeTeamSummary(&analytics.HomeAnalytics, data.HomeTeam)

	// Analyze away team
	gss.analyzeTeamSummary(&analytics.AwayAnalytics, data.AwayTeam)

	return analytics
}

// analyzeTeamSummary processes game summary data for a single team
func (gss *GameSummaryService) analyzeTeamSummary(teamAnalytics *models.TeamSummaryAnalytics, team models.SummaryTeam) {
	stats := team.TeamStats

	// Enhanced Physical Play Metrics
	teamAnalytics.Hits = stats.Hits
	teamAnalytics.BlockedShots = stats.BlockedShots
	teamAnalytics.Giveaways = stats.Giveaways
	teamAnalytics.Takeaways = stats.Takeaways
	teamAnalytics.PhysicalPlayIndex = float64(stats.Hits + stats.BlockedShots)

	// Possession Ratio
	totalPossession := stats.Takeaways + stats.Giveaways
	if totalPossession > 0 {
		teamAnalytics.PossessionRatio = float64(stats.Takeaways) / float64(totalPossession)
	}

	// Enhanced Faceoff Performance
	teamAnalytics.FaceoffWins = stats.FaceoffWins
	teamAnalytics.FaceoffLosses = stats.FaceoffLosses
	teamAnalytics.FaceoffWinPct = stats.FaceoffWinPct

	// Enhanced Special Teams
	teamAnalytics.PowerPlayGoals = stats.PowerPlayGoals
	teamAnalytics.PowerPlayShots = stats.PowerPlayShots
	teamAnalytics.PowerPlayPct = stats.PowerPlayPct
	teamAnalytics.PowerPlayTime = stats.PowerPlayTime
	teamAnalytics.PenaltyKillGoals = stats.PenaltyKillGoals
	teamAnalytics.PenaltyKillShots = stats.PenaltyKillShots
	teamAnalytics.PenaltyKillPct = stats.PenaltyKillPct
	teamAnalytics.PenaltyKillTime = stats.PenaltyKillTime

	// Special Teams Index (combined PP% + PK%)
	teamAnalytics.SpecialTeamsIndex = (stats.PowerPlayPct + stats.PenaltyKillPct) / 200.0

	// Enhanced Zone Control & Time on Attack
	teamAnalytics.TimeOnAttack = stats.TimeOnAttack
	teamAnalytics.TimeOnDefense = stats.TimeOnDefense
	teamAnalytics.OffensiveZoneTime = stats.OffensiveZoneTime
	teamAnalytics.DefensiveZoneTime = stats.DefensiveZoneTime
	teamAnalytics.NeutralZoneTime = stats.NeutralZoneTime

	// Zone Control Ratio
	totalZoneTime := stats.OffensiveZoneTime + stats.DefensiveZoneTime
	if totalZoneTime > 0 {
		teamAnalytics.ZoneControlRatio = stats.OffensiveZoneTime / totalZoneTime
	}

	// Enhanced Transition Play
	teamAnalytics.ControlledEntries = stats.ControlledEntries
	teamAnalytics.ControlledExits = stats.ControlledExits
	totalTransitions := stats.ControlledEntries + stats.ControlledExits
	if totalTransitions > 0 {
		teamAnalytics.TransitionEfficiency = float64(stats.ControlledEntries) / float64(totalTransitions)
	}

	// Enhanced Shot Quality
	teamAnalytics.ShotsOnGoal = stats.ShotsOnGoal
	teamAnalytics.Shots = stats.Shots
	if stats.Shots > 0 {
		teamAnalytics.ShotAccuracy = float64(stats.ShotsOnGoal) / float64(stats.Shots)
	}

	// Shot Quality Index (weighted by danger level)
	teamAnalytics.HighDangerShots = stats.HighDangerShots
	teamAnalytics.MediumDangerShots = stats.MediumDangerShots
	teamAnalytics.LowDangerShots = stats.LowDangerShots

	totalShots := stats.HighDangerShots + stats.MediumDangerShots + stats.LowDangerShots
	if totalShots > 0 {
		// Weighted shot quality: High=3, Medium=2, Low=1
		weightedShots := float64(stats.HighDangerShots*3 + stats.MediumDangerShots*2 + stats.LowDangerShots*1)
		teamAnalytics.ShotQualityIndex = weightedShots / float64(totalShots*3) // Normalize to 0-1
	}

	// Penalty Discipline
	teamAnalytics.Penalties = stats.Penalties
	teamAnalytics.PenaltyMinutes = stats.PenaltyMinutes
	// Discipline Index: Lower is better (fewer penalties)
	if stats.Penalties > 0 {
		teamAnalytics.DisciplineIndex = float64(stats.PenaltyMinutes) / float64(stats.Penalties) // Avg penalty length
	}
}

// GetTeamHistory returns game summary history for a team
func (gss *GameSummaryService) GetTeamHistory(teamCode string) *models.TeamSummaryHistory {
	gss.historyMu.RLock()
	defer gss.historyMu.RUnlock()

	if history, exists := gss.teamHistory[teamCode]; exists {
		return history
	}
	return nil
}

// updateTeamHistory updates rolling game summary history for teams
func (gss *GameSummaryService) updateTeamHistory(analytics *models.GameSummaryAnalytics) {
	gss.updateTeamHistoryForSide(analytics.HomeTeam, &analytics.HomeAnalytics, analytics.Season)
	gss.updateTeamHistoryForSide(analytics.AwayTeam, &analytics.AwayAnalytics, analytics.Season)
}

// updateTeamHistoryForSide updates history for one team
func (gss *GameSummaryService) updateTeamHistoryForSide(teamCode string, game *models.TeamSummaryAnalytics, season int) {
	gss.historyMu.Lock()
	defer gss.historyMu.Unlock()

	history, exists := gss.teamHistory[teamCode]
	if !exists {
		history = &models.TeamSummaryHistory{
			TeamCode: teamCode,
			Season:   season,
		}
		gss.teamHistory[teamCode] = history
	}

	// Update game count
	history.GamesAnalyzed++
	history.LastUpdated = time.Now()

	// Update rolling averages (last 10 games)
	weight := 1.0 / float64(minInt(history.GamesAnalyzed, 10))

	history.AvgHits += (float64(game.Hits) - history.AvgHits) * weight
	history.AvgBlockedShots += (float64(game.BlockedShots) - history.AvgBlockedShots) * weight
	history.AvgPhysicalPlay += (game.PhysicalPlayIndex - history.AvgPhysicalPlay) * weight
	history.AvgPossessionRatio += (game.PossessionRatio - history.AvgPossessionRatio) * weight

	history.AvgFaceoffWinPct += (game.FaceoffWinPct - history.AvgFaceoffWinPct) * weight
	history.AvgPowerPlayPct += (game.PowerPlayPct - history.AvgPowerPlayPct) * weight
	history.AvgPenaltyKillPct += (game.PenaltyKillPct - history.AvgPenaltyKillPct) * weight
	history.AvgSpecialTeams += (game.SpecialTeamsIndex - history.AvgSpecialTeams) * weight

	history.AvgTimeOnAttack += (game.TimeOnAttack - history.AvgTimeOnAttack) * weight
	history.AvgZoneControl += (game.ZoneControlRatio - history.AvgZoneControl) * weight
	history.AvgTransitionEff += (game.TransitionEfficiency - history.AvgTransitionEff) * weight

	history.AvgShotQuality += (game.ShotQualityIndex - history.AvgShotQuality) * weight
	history.AvgDiscipline += (game.DisciplineIndex - history.AvgDiscipline) * weight

	// Update team identity flags
	history.PhysicalTeam = history.AvgPhysicalPlay > 50.0      // High hits + blocks
	history.PossessionTeam = history.AvgPossessionRatio > 0.55 // High takeaways
	history.SpecialTeamsTeam = history.AvgSpecialTeams > 0.5   // Strong PP + PK
	history.ZoneControlTeam = history.AvgZoneControl > 0.55    // High offensive zone time
	history.DisciplinedTeam = history.AvgDiscipline < 2.0      // Low penalty minutes per penalty
	history.ShotQualityTeam = history.AvgShotQuality > 0.6     // High danger shots

	// Save to disk
	if err := gss.saveTeamHistory(); err != nil {
		log.Printf("⚠️ Failed to save team history: %v", err)
	}
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// saveAnalytics saves game summary analytics to disk
func (gss *GameSummaryService) saveAnalytics(analytics *models.GameSummaryAnalytics) error {
	filename := filepath.Join(gss.dataDir, fmt.Sprintf("game_%d.json", analytics.GameID))

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
func (gss *GameSummaryService) saveTeamHistory() error {
	filename := filepath.Join(gss.dataDir, "team_history.json")

	data, err := json.MarshalIndent(gss.teamHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal team history: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write team history file: %w", err)
	}

	return nil
}

// loadTeamHistory loads team history from disk
func (gss *GameSummaryService) loadTeamHistory() {
	filename := filepath.Join(gss.dataDir, "team_history.json")

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("📂 No existing game summary history found (will create on first update)")
		return
	}

	var history map[string]*models.TeamSummaryHistory
	if err := json.Unmarshal(data, &history); err != nil {
		log.Printf("⚠️ Failed to unmarshal team history: %v", err)
		return
	}

	gss.historyMu.Lock()
	gss.teamHistory = history
	gss.historyMu.Unlock()

	log.Printf("📂 Loaded game summary history for %d teams", len(history))
}
