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

// LandingPageService analyzes landing page data for enhanced game metrics
type LandingPageService struct {
	httpClient  *http.Client
	cache       map[int]*models.LandingPageCache // gameID -> cached analytics
	cacheMu     sync.RWMutex
	teamHistory map[string]*models.TeamLandingHistory // teamCode -> history
	historyMu   sync.RWMutex
	dataDir     string
	cacheTTL    time.Duration
}

var (
	landingPageService *LandingPageService
	landingPageOnce    sync.Once
)

// InitLandingPageService initializes the global landing page service
func InitLandingPageService() {
	landingPageOnce.Do(func() {
		landingPageService = NewLandingPageService()
		log.Println("✅ Landing Page Analytics Service initialized")
	})
}

// GetLandingPageService returns the singleton instance
func GetLandingPageService() *LandingPageService {
	return landingPageService
}

// NewLandingPageService creates a new landing page service
func NewLandingPageService() *LandingPageService {
	service := &LandingPageService{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		cache:       make(map[int]*models.LandingPageCache),
		teamHistory: make(map[string]*models.TeamLandingHistory),
		dataDir:     "data/landing_page",
		cacheTTL:    24 * time.Hour,
	}

	// Create data directory
	if err := os.MkdirAll(service.dataDir, 0755); err != nil {
		log.Printf("⚠️ Failed to create landing page directory: %v", err)
	}

	// Load cached team history
	service.loadTeamHistory()

	return service
}

// FetchLandingPageData fetches and analyzes landing page data for a game
func (lps *LandingPageService) FetchLandingPageData(gameID int) (*models.LandingPageAnalytics, error) {
	// Check cache first
	lps.cacheMu.RLock()
	if cached, exists := lps.cache[gameID]; exists {
		if time.Since(cached.CachedAt) < cached.TTL {
			lps.cacheMu.RUnlock()
			log.Printf("📊 Using cached landing page data for game %d", gameID)
			return cached.Analytics, nil
		}
	}
	lps.cacheMu.RUnlock()

	// Fetch from NHL API
	log.Printf("📥 Fetching landing page data for game %d from NHL API...", gameID)

	url := fmt.Sprintf("https://api-web.nhle.com/v1/gamecenter/%d/landing", gameID)

	// Use rate limiter
	rateLimiter := GetNHLRateLimiter()
	if rateLimiter != nil {
		rateLimiter.Wait()
	}

	resp, err := lps.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch landing page data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp models.LandingPageResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode landing page data: %w", err)
	}

	// Analyze the landing page data
	analytics := lps.analyzeLandingPage(&apiResp)

	// Cache the results
	lps.cacheMu.Lock()
	lps.cache[gameID] = &models.LandingPageCache{
		GameID:    gameID,
		Analytics: analytics,
		CachedAt:  time.Now(),
		TTL:       lps.cacheTTL,
	}
	lps.cacheMu.Unlock()

	// Update team history
	lps.updateTeamHistory(analytics)

	// Save to disk
	if err := lps.saveAnalytics(analytics); err != nil {
		log.Printf("⚠️ Failed to save landing page analytics: %v", err)
	}

	log.Printf("✅ Analyzed landing page data for game %d: %s vs %s",
		gameID, analytics.HomeTeam, analytics.AwayTeam)
	return analytics, nil
}

// analyzeLandingPage processes raw landing page data into analytics
func (lps *LandingPageService) analyzeLandingPage(data *models.LandingPageResponse) *models.LandingPageAnalytics {
	gameDate, _ := time.Parse("2006-01-02", data.GameDate)

	analytics := &models.LandingPageAnalytics{
		GameID:        data.ID,
		Season:        data.Season,
		GameDate:      gameDate,
		HomeTeam:      data.HomeTeam.Abbrev,
		AwayTeam:      data.AwayTeam.Abbrev,
		HomeAnalytics: models.TeamLandingAnalytics{TeamCode: data.HomeTeam.Abbrev},
		AwayAnalytics: models.TeamLandingAnalytics{TeamCode: data.AwayTeam.Abbrev},
		ProcessedAt:   time.Now(),
		DataSource:    "NHLE_API_v1_Landing",
	}

	// Analyze home team
	lps.analyzeTeamLanding(&analytics.HomeAnalytics, data.HomeTeam)

	// Analyze away team
	lps.analyzeTeamLanding(&analytics.AwayAnalytics, data.AwayTeam)

	return analytics
}

// analyzeTeamLanding processes landing page data for a single team
func (lps *LandingPageService) analyzeTeamLanding(teamAnalytics *models.TeamLandingAnalytics, team models.LandingTeam) {
	stats := team.TeamStats

	// Physical Play Metrics
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

	// Faceoff Performance
	teamAnalytics.FaceoffWins = stats.FaceoffWins
	teamAnalytics.FaceoffLosses = stats.FaceoffLosses
	teamAnalytics.FaceoffWinPct = stats.FaceoffWinPct

	// Special Teams
	teamAnalytics.PowerPlayGoals = stats.PowerPlayGoals
	teamAnalytics.PowerPlayShots = stats.PowerPlayShots
	teamAnalytics.PowerPlayPct = stats.PowerPlayPct
	teamAnalytics.PenaltyKillGoals = stats.PenaltyKillGoals
	teamAnalytics.PenaltyKillShots = stats.PenaltyKillShots
	teamAnalytics.PenaltyKillPct = stats.PenaltyKillPct

	// Zone Control & Time on Attack
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

	// Transition Play
	teamAnalytics.ControlledEntries = stats.ControlledEntries
	teamAnalytics.ControlledExits = stats.ControlledExits
	totalTransitions := stats.ControlledEntries + stats.ControlledExits
	if totalTransitions > 0 {
		teamAnalytics.TransitionEfficiency = float64(stats.ControlledEntries) / float64(totalTransitions)
	}

	// Shot Quality
	teamAnalytics.ShotsOnGoal = stats.ShotsOnGoal
	teamAnalytics.Shots = stats.Shots
	if stats.Shots > 0 {
		teamAnalytics.ShotAccuracy = float64(stats.ShotsOnGoal) / float64(stats.Shots)
	}
}

// GetTeamHistory returns landing page history for a team
func (lps *LandingPageService) GetTeamHistory(teamCode string) *models.TeamLandingHistory {
	lps.historyMu.RLock()
	defer lps.historyMu.RUnlock()

	if history, exists := lps.teamHistory[teamCode]; exists {
		return history
	}
	return nil
}

// updateTeamHistory updates rolling landing page history for teams
func (lps *LandingPageService) updateTeamHistory(analytics *models.LandingPageAnalytics) {
	lps.updateTeamHistoryForSide(analytics.HomeTeam, &analytics.HomeAnalytics, analytics.Season)
	lps.updateTeamHistoryForSide(analytics.AwayTeam, &analytics.AwayAnalytics, analytics.Season)
}

// updateTeamHistoryForSide updates history for one team
func (lps *LandingPageService) updateTeamHistoryForSide(teamCode string, game *models.TeamLandingAnalytics, season int) {
	lps.historyMu.Lock()
	defer lps.historyMu.Unlock()

	history, exists := lps.teamHistory[teamCode]
	if !exists {
		history = &models.TeamLandingHistory{
			TeamCode: teamCode,
			Season:   season,
		}
		lps.teamHistory[teamCode] = history
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

	history.AvgTimeOnAttack += (game.TimeOnAttack - history.AvgTimeOnAttack) * weight
	history.AvgZoneControl += (game.ZoneControlRatio - history.AvgZoneControl) * weight
	history.AvgTransitionEff += (game.TransitionEfficiency - history.AvgTransitionEff) * weight

	// Update team identity flags
	history.PhysicalTeam = history.AvgPhysicalPlay > 50.0                                    // High hits + blocks
	history.PossessionTeam = history.AvgPossessionRatio > 0.55                               // High takeaways
	history.SpecialTeamsTeam = (history.AvgPowerPlayPct + history.AvgPenaltyKillPct) > 100.0 // Strong PP + PK
	history.ZoneControlTeam = history.AvgZoneControl > 0.55                                  // High offensive zone time

	// Save to disk
	if err := lps.saveTeamHistory(); err != nil {
		log.Printf("⚠️ Failed to save team history: %v", err)
	}
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// saveAnalytics saves landing page analytics to disk
func (lps *LandingPageService) saveAnalytics(analytics *models.LandingPageAnalytics) error {
	filename := filepath.Join(lps.dataDir, fmt.Sprintf("game_%d.json", analytics.GameID))

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
func (lps *LandingPageService) saveTeamHistory() error {
	filename := filepath.Join(lps.dataDir, "team_history.json")

	data, err := json.MarshalIndent(lps.teamHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal team history: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write team history file: %w", err)
	}

	return nil
}

// loadTeamHistory loads team history from disk
func (lps *LandingPageService) loadTeamHistory() {
	filename := filepath.Join(lps.dataDir, "team_history.json")

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("📂 No existing landing page history found (will create on first update)")
		return
	}

	var history map[string]*models.TeamLandingHistory
	if err := json.Unmarshal(data, &history); err != nil {
		log.Printf("⚠️ Failed to unmarshal team history: %v", err)
		return
	}

	lps.historyMu.Lock()
	lps.teamHistory = history
	lps.historyMu.Unlock()

	log.Printf("📂 Loaded landing page history for %d teams", len(history))
}
