package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// PlayByPlayService analyzes play-by-play data and calculates advanced metrics
type PlayByPlayService struct {
	httpClient      *http.Client
	cache           map[int]*models.PlayByPlayCache // gameID -> cached analytics
	cacheMu         sync.RWMutex
	teamStats       map[string]*models.TeamPlayByPlayStats // teamCode -> rolling stats
	statsMu         sync.RWMutex
	dataDir         string
	cacheTTL        time.Duration
	systemStatsServ *SystemStatsService
}

var (
	playByPlayService *PlayByPlayService
	playByPlayOnce    sync.Once
)

// InitPlayByPlayService initializes the global play-by-play service
func InitPlayByPlayService(statsServ *SystemStatsService) {
	playByPlayOnce.Do(func() {
		playByPlayService = NewPlayByPlayService(statsServ)
		log.Println("✅ Play-by-Play Analytics Service initialized")
	})
}

// GetPlayByPlayService returns the singleton instance
func GetPlayByPlayService() *PlayByPlayService {
	return playByPlayService
}

// BackfillPlayByPlayData fetches play-by-play data for the last N completed games for all teams
func (pbp *PlayByPlayService) BackfillPlayByPlayData(teamCode string, numGames int) error {
	log.Printf("🔄 Starting play-by-play backfill for %s (last %d games)...", teamCode, numGames)

	// Get the team's season schedule (both past and future)
	completedGames, err := pbp.getCompletedGames(teamCode, numGames)
	if err != nil {
		return fmt.Errorf("failed to get completed games: %w", err)
	}

	log.Printf("📊 Found %d completed games to backfill", len(completedGames))

	// Process each game
	successCount := 0
	failCount := 0

	for i, game := range completedGames {
		log.Printf("🏒 [%d/%d] Processing game %d: %s vs %s",
			i+1, len(completedGames), game.ID, game.AwayTeam.Abbrev, game.HomeTeam.Abbrev)

		startTime := time.Now()

		// Fetch and analyze play-by-play
		analytics, err := pbp.FetchPlayByPlay(game.ID)
		if err != nil {
			log.Printf("⚠️ Failed to fetch play-by-play for game %d: %v", game.ID, err)
			failCount++
			if pbp.systemStatsServ != nil {
				pbp.systemStatsServ.RecordBackfillFailure()
			}
			continue
		}

		processingTime := time.Since(startTime)
		eventsProcessed := analytics.HomeAnalytics.TotalShots + analytics.AwayAnalytics.TotalShots +
			analytics.HomeAnalytics.Hits + analytics.AwayAnalytics.Hits +
			analytics.HomeAnalytics.Giveaways + analytics.AwayAnalytics.Giveaways

		log.Printf("✅ Processed game %d: %s %.2f xG vs %s %.2f xG",
			game.ID,
			analytics.HomeTeam, analytics.HomeAnalytics.ExpectedGoals,
			analytics.AwayTeam, analytics.AwayAnalytics.ExpectedGoals)

		// Record backfill stats
		if pbp.systemStatsServ != nil {
			pbp.systemStatsServ.RecordBackfillGame("play-by-play", eventsProcessed, processingTime)
		}

		successCount++

		// Brief delay to respect rate limits
		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("🎉 Backfill complete! Success: %d, Failed: %d", successCount, failCount)

	// Log the updated stats
	pbp.statsMu.RLock()
	for teamCode, stats := range pbp.teamStats {
		if stats.GamesAnalyzed > 0 {
			log.Printf("📊 %s stats: %.2f xGF/game, %.2f xGA/game, %.1f%% Corsi",
				teamCode, stats.AvgExpectedGoals, stats.AvgXGAgainst, stats.AvgCorsiForPct*100)
		}
	}
	pbp.statsMu.RUnlock()

	return nil
}

// getCompletedGames fetches the last N completed games for a team
func (pbp *PlayByPlayService) getCompletedGames(teamCode string, numGames int) ([]models.Game, error) {
	// Determine current and previous season
	currentSeason := getCurrentSeasonInt()
	previousSeason := currentSeason - 10001

	// Try current season first
	allGames, err := GetTeamSeasonSchedule(teamCode, currentSeason)
	if err != nil || len(allGames) == 0 {
		// Fall back to previous season if current season not available
		log.Printf("📅 Current season not available, fetching previous season %d", previousSeason)
		allGames, err = GetTeamSeasonSchedule(teamCode, previousSeason)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch season schedule: %w", err)
		}
	}

	// Filter for completed games (games that started more than 3 hours ago)
	completedGames := []models.Game{}
	now := time.Now()
	completionBuffer := 3 * time.Hour // Games started >3 hours ago are likely complete

	for _, game := range allGames {
		// Parse game time
		gameTime, err := time.Parse(time.RFC3339, game.StartTime)
		if err != nil {
			continue
		}

		// Check if game started more than 3 hours ago (likely completed)
		if gameTime.Before(now.Add(-completionBuffer)) {
			completedGames = append(completedGames, game)
		}
	}

	// Get the last N games
	if len(completedGames) > numGames {
		completedGames = completedGames[len(completedGames)-numGames:]
	}

	log.Printf("📊 Found %d completed games out of %d total games", len(completedGames), len(allGames))
	return completedGames, nil
}

// getCurrentSeasonInt returns the current NHL season as an integer (e.g., 20252026)
func getCurrentSeasonInt() int {
	now := time.Now()
	year := now.Year()
	month := now.Month()

	// NHL season starts in October and runs through April of next year
	if month >= time.October {
		// Current season (e.g., October 2025 = 20252026 season)
		return year*10000 + (year + 1)
	}
	// Previous season (e.g., April 2025 = 20242025 season)
	return (year-1)*10000 + year
}

// BackfillAllTeams fetches play-by-play data for all NHL teams
func (pbp *PlayByPlayService) BackfillAllTeams(numGames int) error {
	log.Printf("🔄 Starting league-wide play-by-play backfill (last %d games per team)...", numGames)

	// NHL team codes
	teams := []string{
		"ANA", "BOS", "BUF", "CAR", "CBJ", "CGY", "CHI", "COL", "DAL", "DET",
		"EDM", "FLA", "LAK", "MIN", "MTL", "NJD", "NSH", "NYI", "NYR", "OTT",
		"PHI", "PIT", "SEA", "SJS", "STL", "TBL", "TOR", "UTA", "VAN", "VGK",
		"WPG", "WSH",
	}

	totalSuccess := 0
	totalFail := 0

	for i, team := range teams {
		log.Printf("🏒 [%d/%d] Backfilling team: %s", i+1, len(teams), team)

		if err := pbp.BackfillPlayByPlayData(team, numGames); err != nil {
			log.Printf("⚠️ Failed to backfill %s: %v", team, err)
			totalFail++
		} else {
			totalSuccess++
		}

		// Delay between teams to respect rate limits
		time.Sleep(2 * time.Second)
	}

	log.Printf("🎉 League-wide backfill complete! Teams: %d success, %d failed", totalSuccess, totalFail)
	return nil
}

// NewPlayByPlayService creates a new play-by-play service
func NewPlayByPlayService(statsServ *SystemStatsService) *PlayByPlayService {
	service := &PlayByPlayService{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		cache:           make(map[int]*models.PlayByPlayCache),
		teamStats:       make(map[string]*models.TeamPlayByPlayStats),
		dataDir:         "data/play_by_play",
		cacheTTL:        24 * time.Hour, // Cache for 24 hours
		systemStatsServ: statsServ,
	}

	// Create data directory
	if err := os.MkdirAll(service.dataDir, 0755); err != nil {
		log.Printf("⚠️ Failed to create play-by-play directory: %v", err)
	}

	// Load cached team stats
	service.loadTeamStats()

	return service
}

// FetchPlayByPlay fetches and analyzes play-by-play data for a game
func (pbp *PlayByPlayService) FetchPlayByPlay(gameID int) (*models.PlayByPlayAnalytics, error) {
	// Check cache first
	pbp.cacheMu.RLock()
	if cached, exists := pbp.cache[gameID]; exists {
		if time.Since(cached.CachedAt) < cached.TTL {
			pbp.cacheMu.RUnlock()
			log.Printf("📊 Using cached play-by-play for game %d", gameID)
			return cached.Analytics, nil
		}
	}
	pbp.cacheMu.RUnlock()

	// Fetch from NHL API
	log.Printf("📥 Fetching play-by-play data for game %d from NHL API...", gameID)

	url := fmt.Sprintf("https://api-web.nhle.com/v1/gamecenter/%d/play-by-play", gameID)

	// Use rate limiter if available
	rateLimiter := GetNHLRateLimiter()
	if rateLimiter != nil {
		rateLimiter.Wait()
	}

	resp, err := pbp.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch play-by-play: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp models.PlayByPlayResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode play-by-play: %w", err)
	}

	// Analyze the play-by-play data
	analytics := pbp.analyzePlayByPlay(&apiResp)

	// Cache the results
	pbp.cacheMu.Lock()
	pbp.cache[gameID] = &models.PlayByPlayCache{
		GameID:    gameID,
		Analytics: analytics,
		CachedAt:  time.Now(),
		TTL:       pbp.cacheTTL,
	}
	pbp.cacheMu.Unlock()

	// Update team rolling stats
	pbp.updateTeamStats(analytics)

	// Save to disk
	if err := pbp.saveAnalytics(analytics); err != nil {
		log.Printf("⚠️ Failed to save play-by-play analytics: %v", err)
	}

	log.Printf("✅ Analyzed %d play events for game %d", len(apiResp.Plays), gameID)
	return analytics, nil
}

// analyzePlayByPlay processes raw play-by-play data into analytics
func (pbp *PlayByPlayService) analyzePlayByPlay(data *models.PlayByPlayResponse) *models.PlayByPlayAnalytics {
	gameDate, _ := time.Parse("2006-01-02", data.GameDate)

	analytics := &models.PlayByPlayAnalytics{
		GameID:        data.ID,
		Season:        data.Season,
		GameDate:      gameDate,
		HomeTeam:      data.HomeTeam.Abbrev,
		AwayTeam:      data.AwayTeam.Abbrev,
		HomeAnalytics: models.TeamPlayAnalytics{TeamCode: data.HomeTeam.Abbrev},
		AwayAnalytics: models.TeamPlayAnalytics{TeamCode: data.AwayTeam.Abbrev},
		ProcessedAt:   time.Now(),
		DataSource:    "NHLE_API_v1_PlayByPlay",
	}

	homeID := data.HomeTeam.ID
	// awayID := data.AwayTeam.ID // Currently unused

	// Process each play event
	for i, play := range data.Plays {
		isHomeEvent := play.Details.EventOwnerTeamID == homeID
		teamAnalytics := &analytics.HomeAnalytics
		opponentAnalytics := &analytics.AwayAnalytics
		if !isHomeEvent {
			teamAnalytics = &analytics.AwayAnalytics
			opponentAnalytics = &analytics.HomeAnalytics
		}

		// Determine if this is a rebound (shot within 3 seconds of previous shot)
		isRebound := false
		if i > 0 && (play.TypeDescKey == "shot-on-goal" || play.TypeDescKey == "goal") {
			prevPlay := data.Plays[i-1]
			if prevPlay.TypeDescKey == "shot-on-goal" || prevPlay.TypeDescKey == "missed-shot" {
				// Simple rebound detection (would need time parsing for accuracy)
				isRebound = true
			}
		}

		// Process by event type
		switch play.TypeDescKey {
		case "shot-on-goal":
			pbp.processShotOnGoal(play, teamAnalytics, opponentAnalytics, isRebound)

		case "missed-shot":
			pbp.processMissedShot(play, teamAnalytics)

		case "blocked-shot":
			pbp.processBlockedShot(play, teamAnalytics, opponentAnalytics)

		case "goal":
			pbp.processGoal(play, teamAnalytics, opponentAnalytics, isRebound)

		case "hit":
			pbp.processHit(play, teamAnalytics, opponentAnalytics, homeID)

		case "faceoff":
			pbp.processFaceoff(play, teamAnalytics, opponentAnalytics, homeID)

		case "giveaway":
			teamAnalytics.Giveaways++

		case "takeaway":
			teamAnalytics.Takeaways++

		case "penalty":
			teamAnalytics.PenaltiesTaken++
			if play.Details.Duration > 0 {
				teamAnalytics.PenaltyMinutes += play.Details.Duration
			}
		}
	}

	// Calculate derived metrics
	pbp.calculateDerivedMetrics(&analytics.HomeAnalytics)
	pbp.calculateDerivedMetrics(&analytics.AwayAnalytics)

	return analytics
}

// processShotOnGoal handles shot-on-goal events
func (pbp *PlayByPlayService) processShotOnGoal(play models.PlayEvent, team, opponent *models.TeamPlayAnalytics, isRebound bool) {
	team.ShotsOnGoal++
	team.TotalShots++
	team.ShotAttempts++ // Corsi

	// Calculate Expected Goals (xG)
	shotCtx := pbp.buildShotContext(play, isRebound)
	xgResult := pbp.calculateExpectedGoal(shotCtx)

	team.ExpectedGoals += xgResult.XG
	opponent.ExpectedGoalsAgainst += xgResult.XG

	// Track danger level
	switch xgResult.DangerLevel {
	case "high":
		team.DangerousShots++
		team.HighDangerXG += xgResult.XG
	case "medium":
		team.MediumDangerShots++
	case "low":
		team.LowDangerShots++
	}

	// Track shot type
	pbp.trackShotType(play.Details.ShotType, team)

	// Track zone
	if play.Details.ZoneCode == "O" {
		team.OffensiveZoneShots++
	}

	// Track rebound
	if isRebound {
		team.ReboundShots++
	}

	// Update average distance/angle
	pbp.updateShotLocationMetrics(play, team)
}

// processMissedShot handles missed shot events
func (pbp *PlayByPlayService) processMissedShot(play models.PlayEvent, team *models.TeamPlayAnalytics) {
	team.MissedShots++
	team.TotalShots++
	team.ShotAttempts++ // Corsi
}

// processBlockedShot handles blocked shot events
func (pbp *PlayByPlayService) processBlockedShot(play models.PlayEvent, team, opponent *models.TeamPlayAnalytics) {
	team.BlockedShots++ // Shot was blocked (against this team)
	team.TotalShots++
	team.ShotAttempts++        // Corsi
	opponent.BlockedShotsFor++ // Opponent blocked it (for opponent)
}

// processGoal handles goal events
func (pbp *PlayByPlayService) processGoal(play models.PlayEvent, team, opponent *models.TeamPlayAnalytics, isRebound bool) {
	team.ActualGoals++
	team.ShotsOnGoal++
	team.TotalShots++
	team.ShotAttempts++

	// Calculate xG for the goal
	shotCtx := pbp.buildShotContext(play, isRebound)
	xgResult := pbp.calculateExpectedGoal(shotCtx)
	xgResult.IsGoal = true

	team.ExpectedGoals += xgResult.XG
	opponent.ExpectedGoalsAgainst += xgResult.XG

	// Track danger level
	if xgResult.DangerLevel == "high" {
		team.DangerousShots++
		team.HighDangerXG += xgResult.XG
	}

	// Track shot type
	pbp.trackShotType(play.Details.ShotType, team)

	if isRebound {
		team.ReboundShots++
	}
}

// processHit handles hit events
func (pbp *PlayByPlayService) processHit(play models.PlayEvent, team, opponent *models.TeamPlayAnalytics, homeID int) {
	// Determine which team made the hit
	if play.Details.HittingPlayerID != 0 {
		team.Hits++
	}
}

// processFaceoff handles faceoff events
func (pbp *PlayByPlayService) processFaceoff(play models.PlayEvent, team, opponent *models.TeamPlayAnalytics, homeID int) {
	// Determine winner
	if play.Details.WinningPlayerID != 0 {
		team.FaceoffsWon++
		opponent.FaceoffsLost++

		// Track by zone
		switch play.Details.ZoneCode {
		case "O":
			team.OffensiveZoneFOWins++
		case "D":
			team.DefensiveZoneFOWins++
		case "N":
			team.NeutralZoneFOWins++
		}
	} else if play.Details.LosingPlayerID != 0 {
		team.FaceoffsLost++
		opponent.FaceoffsWon++

		// Track opponent zones
		switch play.Details.ZoneCode {
		case "O":
			opponent.DefensiveZoneFOWins++ // Opponent's defensive zone
		case "D":
			opponent.OffensiveZoneFOWins++ // Opponent's offensive zone
		case "N":
			opponent.NeutralZoneFOWins++
		}
	}
}

// buildShotContext creates a shot context for xG calculation
func (pbp *PlayByPlayService) buildShotContext(play models.PlayEvent, isRebound bool) models.ShotContext {
	location := models.ShotLocation{
		X:        play.Details.XCoord,
		Y:        play.Details.YCoord,
		ZoneCode: play.Details.ZoneCode,
	}

	// Calculate distance and angle from net
	// NHL rink: center is (0,0), goal is at (-89, 0) or (89, 0)
	// Assuming offensive zone shots have positive X
	goalX := 89.0
	if location.X < 0 {
		goalX = -89.0
	}

	dx := float64(location.X) - goalX
	dy := float64(location.Y)
	location.Distance = math.Sqrt(dx*dx + dy*dy)
	location.Angle = math.Abs(math.Atan2(dy, math.Abs(dx)) * 180 / math.Pi)

	// Determine danger zone (slot area: within 25 feet, angle < 40 degrees)
	location.IsDangerZone = location.Distance < 25 && location.Angle < 40

	// Determine situation from situation code
	// situationCode format: "1551" = 5v5, "1541" = 5v4 (PP), etc.
	isPP := false
	isSH := false
	// TODO: Parse situationCode for PP/SH detection

	return models.ShotContext{
		Location:      location,
		ShotType:      play.Details.ShotType,
		IsRebound:     isRebound,
		IsRush:        false, // TODO: Detect rush chances
		IsPowerPlay:   isPP,
		IsShortHanded: isSH,
		IsEmptyNet:    false, // TODO: Detect empty net
	}
}

// calculateExpectedGoal calculates xG for a shot
// Based on distance, angle, shot type, and context
func (pbp *PlayByPlayService) calculateExpectedGoal(ctx models.ShotContext) models.ExpectedGoalResult {
	// Simple xG model (production would use trained ML model)
	baseXG := 0.05 // 5% baseline

	// Distance factor (closer = better)
	distanceFactor := 1.0
	if ctx.Location.Distance < 10 {
		distanceFactor = 4.0 // Very close
	} else if ctx.Location.Distance < 20 {
		distanceFactor = 2.5
	} else if ctx.Location.Distance < 30 {
		distanceFactor = 1.5
	} else if ctx.Location.Distance < 40 {
		distanceFactor = 1.0
	} else {
		distanceFactor = 0.5 // Long shot
	}

	// Angle factor (center is better)
	angleFactor := 1.0
	if ctx.Location.Angle < 15 {
		angleFactor = 2.0 // Right in front
	} else if ctx.Location.Angle < 30 {
		angleFactor = 1.5
	} else if ctx.Location.Angle < 45 {
		angleFactor = 1.0
	} else {
		angleFactor = 0.6 // Sharp angle
	}

	// Shot type factor
	shotTypeFactor := 1.0
	switch ctx.ShotType {
	case "Slap":
		shotTypeFactor = 1.3
	case "Snap":
		shotTypeFactor = 1.2
	case "Wrist":
		shotTypeFactor = 1.0
	case "Backhand":
		shotTypeFactor = 0.8
	case "Tip-In", "Deflected":
		shotTypeFactor = 1.8
	}

	// Context multipliers
	if ctx.IsRebound {
		baseXG *= 2.0 // Rebounds are much more dangerous
	}
	if ctx.IsRush {
		baseXG *= 1.5
	}
	if ctx.IsPowerPlay {
		baseXG *= 1.3
	}
	if ctx.IsEmptyNet {
		return models.ExpectedGoalResult{
			XG:          0.95, // Almost guaranteed
			DangerLevel: "high",
		}
	}

	// Calculate final xG
	xg := baseXG * distanceFactor * angleFactor * shotTypeFactor

	// Cap at 0.95 (nothing is 100%)
	if xg > 0.95 {
		xg = 0.95
	}

	// Determine danger level
	dangerLevel := "low"
	if xg > 0.20 {
		dangerLevel = "high"
	} else if xg > 0.10 {
		dangerLevel = "medium"
	}

	return models.ExpectedGoalResult{
		XG:              xg,
		DangerLevel:     dangerLevel,
		Distance:        ctx.Location.Distance,
		Angle:           ctx.Location.Angle,
		ShotType:        ctx.ShotType,
		ConfidenceLevel: 0.75, // Model confidence
	}
}

// trackShotType increments the counter for a shot type
func (pbp *PlayByPlayService) trackShotType(shotType string, team *models.TeamPlayAnalytics) {
	switch shotType {
	case "Wrist":
		team.WristShots++
	case "Slap":
		team.SlapShots++
	case "Snap":
		team.SnapShots++
	case "Backhand":
		team.BackhandShots++
	case "Tip-In":
		team.TipInShots++
	case "Deflected":
		team.DeflectedShots++
	}
}

// updateShotLocationMetrics updates average shot distance and angle
func (pbp *PlayByPlayService) updateShotLocationMetrics(play models.PlayEvent, team *models.TeamPlayAnalytics) {
	if play.Details.XCoord == 0 && play.Details.YCoord == 0 {
		return // No location data
	}

	// Calculate distance
	goalX := 89.0
	if play.Details.XCoord < 0 {
		goalX = -89.0
	}
	dx := float64(play.Details.XCoord) - goalX
	dy := float64(play.Details.YCoord)
	distance := math.Sqrt(dx*dx + dy*dy)

	// Update running average (simple method)
	currentTotal := team.AvgShotDistance * float64(team.TotalShots-1)
	team.AvgShotDistance = (currentTotal + distance) / float64(team.TotalShots)

	// Calculate angle
	angle := math.Abs(math.Atan2(dy, math.Abs(dx)) * 180 / math.Pi)
	currentAngleTotal := team.AvgShotAngle * float64(team.TotalShots-1)
	team.AvgShotAngle = (currentAngleTotal + angle) / float64(team.TotalShots)
}

// calculateDerivedMetrics calculates all derived/percentage metrics
func (pbp *PlayByPlayService) calculateDerivedMetrics(team *models.TeamPlayAnalytics) {
	// xG differential
	team.XGDifferential = team.ExpectedGoals - team.ExpectedGoalsAgainst

	// Goals vs Expected (luck/skill indicator)
	team.GoalsVsExpected = float64(team.ActualGoals) - team.ExpectedGoals

	// Faceoff win percentage
	totalFaceoffs := team.FaceoffsWon + team.FaceoffsLost
	if totalFaceoffs > 0 {
		team.FaceoffWinPct = float64(team.FaceoffsWon) / float64(totalFaceoffs)
	}

	// Corsi For % (shot attempt differential)
	team.CorsiFor = team.ShotAttempts
	totalCorsi := team.CorsiFor + team.CorsiAgainst
	if totalCorsi > 0 {
		team.CorsiForPct = float64(team.CorsiFor) / float64(totalCorsi)
	}

	// Fenwick (unblocked shot attempts)
	team.FenwickFor = team.ShotsOnGoal + team.MissedShots
	totalFenwick := team.FenwickFor + team.FenwickAgainst
	if totalFenwick > 0 {
		team.FenwickForPct = float64(team.FenwickFor) / float64(totalFenwick)
	}

	// Shot quality index (xG per shot)
	if team.TotalShots > 0 {
		team.ShotQualityIndex = team.ExpectedGoals / float64(team.TotalShots)
	}

	// Possession ratio (takeaways / (takeaways + giveaways))
	totalPossession := team.Takeaways + team.Giveaways
	if totalPossession > 0 {
		team.PossessionRatio = float64(team.Takeaways) / float64(totalPossession)
	}
}

// GetTeamStats returns rolling play-by-play stats for a team
func (pbp *PlayByPlayService) GetTeamStats(teamCode string) *models.TeamPlayByPlayStats {
	pbp.statsMu.RLock()
	defer pbp.statsMu.RUnlock()

	if stats, exists := pbp.teamStats[teamCode]; exists {
		return stats
	}
	return nil
}

// updateTeamStats updates rolling averages for a team
func (pbp *PlayByPlayService) updateTeamStats(analytics *models.PlayByPlayAnalytics) {
	pbp.updateTeamStatsForSide(analytics.HomeTeam, &analytics.HomeAnalytics, analytics.Season)
	pbp.updateTeamStatsForSide(analytics.AwayTeam, &analytics.AwayAnalytics, analytics.Season)
}

// updateTeamStatsForSide updates stats for one team
func (pbp *PlayByPlayService) updateTeamStatsForSide(teamCode string, game *models.TeamPlayAnalytics, season int) {
	pbp.statsMu.Lock()
	defer pbp.statsMu.Unlock()

	stats, exists := pbp.teamStats[teamCode]
	if !exists {
		stats = &models.TeamPlayByPlayStats{
			TeamCode: teamCode,
			Season:   season,
		}
		pbp.teamStats[teamCode] = stats
	}

	// Update game count
	stats.GamesAnalyzed++
	stats.LastUpdated = time.Now()

	// Update rolling averages (simple moving average, last 10 games)
	weight := 1.0 / float64(minInt(stats.GamesAnalyzed, 10))

	stats.AvgExpectedGoals += (game.ExpectedGoals - stats.AvgExpectedGoals) * weight
	stats.AvgXGAgainst += (game.ExpectedGoalsAgainst - stats.AvgXGAgainst) * weight
	stats.AvgXGDifferential += (game.XGDifferential - stats.AvgXGDifferential) * weight
	stats.AvgShotQuality += (game.ShotQualityIndex - stats.AvgShotQuality) * weight
	stats.AvgDangerousShots += (float64(game.DangerousShots) - stats.AvgDangerousShots) * weight

	stats.AvgCorsiForPct += (game.CorsiForPct - stats.AvgCorsiForPct) * weight
	stats.AvgFenwickForPct += (game.FenwickForPct - stats.AvgFenwickForPct) * weight
	stats.AvgFaceoffWinPct += (game.FaceoffWinPct - stats.AvgFaceoffWinPct) * weight

	stats.AvgHits += (float64(game.Hits) - stats.AvgHits) * weight
	stats.AvgBlockedShots += (float64(game.BlockedShotsFor) - stats.AvgBlockedShots) * weight
	stats.AvgPossessionRatio += (game.PossessionRatio - stats.AvgPossessionRatio) * weight

	// Update luck/skill indicators
	stats.TotalGoalsVsExpected += game.GoalsVsExpected
	stats.ShootingTalent = stats.TotalGoalsVsExpected / float64(stats.GamesAnalyzed)

	// Save to disk
	if err := pbp.saveTeamStats(); err != nil {
		log.Printf("⚠️ Failed to save team stats: %v", err)
	}
}

// minInt is a helper function (renamed to avoid conflict with player_impact_service)
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// saveAnalytics saves play-by-play analytics to disk
func (pbp *PlayByPlayService) saveAnalytics(analytics *models.PlayByPlayAnalytics) error {
	filename := filepath.Join(pbp.dataDir, fmt.Sprintf("game_%d.json", analytics.GameID))

	data, err := json.MarshalIndent(analytics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analytics: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write analytics file: %w", err)
	}

	return nil
}

// saveTeamStats saves team stats to disk
func (pbp *PlayByPlayService) saveTeamStats() error {
	filename := filepath.Join(pbp.dataDir, "team_stats.json")

	data, err := json.MarshalIndent(pbp.teamStats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal team stats: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write team stats file: %w", err)
	}

	return nil
}

// loadTeamStats loads team stats from disk
func (pbp *PlayByPlayService) loadTeamStats() {
	filename := filepath.Join(pbp.dataDir, "team_stats.json")

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("📂 No existing play-by-play team stats found (will create on first update)")
		return
	}

	var stats map[string]*models.TeamPlayByPlayStats
	if err := json.Unmarshal(data, &stats); err != nil {
		log.Printf("⚠️ Failed to unmarshal team stats: %v", err)
		return
	}

	pbp.statsMu.Lock()
	pbp.teamStats = stats
	pbp.statsMu.Unlock()

	log.Printf("📂 Loaded play-by-play stats for %d teams", len(stats))
}
