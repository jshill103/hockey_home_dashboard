package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// Global referee service instance
var (
	globalRefereeService *RefereeService
	refereeServiceMutex  sync.Mutex
)

// RefereeService manages referee data collection and analysis
type RefereeService struct {
	dataDir               string
	referees              map[int]*models.Referee                 // refereeId -> Referee
	seasonStats           map[int]*models.RefereeSeasonStats      // refereeId -> current season stats
	teamBiases            map[string][]*models.RefereeTeamBias    // "refereeId_teamCode" -> bias data
	gameAssignments       map[int]*models.RefereeGameAssignment   // gameId -> assignment
	dailySchedules        map[string]*models.RefereeDailySchedule // "YYYY-MM-DD" -> schedule
	currentSeason         int
	scraper               *RefereeScraper                         // Web scraper instance
	mutex                 sync.RWMutex
	httpClient            *http.Client
	autoUpdateEnabled     bool
	lastUpdate            time.Time
	updateInterval        time.Duration
	stopChan              chan bool
}

// NewRefereeService creates a new referee tracking service
func NewRefereeService(currentSeason int) *RefereeService {
	service := &RefereeService{
		dataDir:           "data/referees",
		referees:          make(map[int]*models.Referee),
		seasonStats:       make(map[int]*models.RefereeSeasonStats),
		teamBiases:        make(map[string][]*models.RefereeTeamBias),
		gameAssignments:   make(map[int]*models.RefereeGameAssignment),
		dailySchedules:    make(map[string]*models.RefereeDailySchedule),
		currentSeason:     currentSeason,
		httpClient:        &http.Client{Timeout: 30 * time.Second},
		autoUpdateEnabled: true,
		updateInterval:    24 * time.Hour, // Update once per day
		stopChan:          make(chan bool),
	}

	// Create data directories
	os.MkdirAll(filepath.Join(service.dataDir, "referees"), 0755)
	os.MkdirAll(filepath.Join(service.dataDir, "assignments"), 0755)
	os.MkdirAll(filepath.Join(service.dataDir, "schedules"), 0755)
	os.MkdirAll(filepath.Join(service.dataDir, "stats"), 0755)
	os.MkdirAll(filepath.Join(service.dataDir, "bias"), 0755)

	// Load existing data
	if err := service.loadAllData(); err != nil {
		log.Printf("⚠️ Failed to load existing referee data: %v", err)
	}

	// Initialize web scraper
	service.scraper = NewRefereeScraper(service)

	return service
}

// InitializeRefereeService initializes the global referee service
func InitializeRefereeService(currentSeason int) error {
	refereeServiceMutex.Lock()
	defer refereeServiceMutex.Unlock()

	if globalRefereeService != nil {
		return fmt.Errorf("referee service already initialized")
	}

	globalRefereeService = NewRefereeService(currentSeason)
	log.Printf("✅ Referee Service initialized for season %d", currentSeason)

	return nil
}

// GetRefereeService returns the global referee service instance
func GetRefereeService() *RefereeService {
	refereeServiceMutex.Lock()
	defer refereeServiceMutex.Unlock()
	return globalRefereeService
}

// GetCurrentSeason returns the current season being tracked
func (rs *RefereeService) GetCurrentSeason() int {
	return rs.currentSeason
}

// Start begins the automatic referee data update process
func (rs *RefereeService) Start() {
	if !rs.autoUpdateEnabled {
		log.Printf("🏒 Referee Service: Auto-update disabled")
		return
	}

	log.Printf("🏒 Referee Service started (auto-update every %v)", rs.updateInterval)

	// Start web scraper automatic updates
	if rs.scraper != nil {
		rs.scraper.ScheduleAutomaticUpdates()
	}

	// Start background update goroutine
	go rs.autoUpdateLoop()
}

// Stop halts the automatic update process
func (rs *RefereeService) Stop() {
	if rs.autoUpdateEnabled {
		rs.stopChan <- true
		log.Printf("⏹️ Referee Service stopped")
	}
}

// autoUpdateLoop runs periodic updates
func (rs *RefereeService) autoUpdateLoop() {
	ticker := time.NewTicker(rs.updateInterval)
	defer ticker.Stop()

	// Do an initial update
	rs.updateRefereeData()

	for {
		select {
		case <-ticker.C:
			rs.updateRefereeData()
		case <-rs.stopChan:
			return
		}
	}
}

// updateRefereeData fetches latest referee assignments and stats
func (rs *RefereeService) updateRefereeData() {
	log.Printf("🔄 Updating referee data...")
	start := time.Now()

	// Run scraper update
	if rs.scraper != nil {
		rs.scraper.RunDailyUpdate()
	} else {
		log.Printf("⚠️ No scraper available - manual updates only")
	}

	rs.mutex.Lock()
	rs.lastUpdate = time.Now()
	rs.mutex.Unlock()

	log.Printf("✅ Referee data update complete (took %v)", time.Since(start))
}

// GetScraper returns the web scraper instance
func (rs *RefereeService) GetScraper() *RefereeScraper {
	return rs.scraper
}

// TriggerScrape manually triggers a web scrape
func (rs *RefereeService) TriggerScrape() error {
	if rs.scraper == nil {
		return fmt.Errorf("scraper not initialized")
	}
	
	return rs.scraper.RunDailyUpdate()
}

// ============================================================================
// REFEREE MANAGEMENT
// ============================================================================

// AddReferee adds or updates a referee in the system
func (rs *RefereeService) AddReferee(referee *models.Referee) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	referee.LastUpdated = time.Now()
	rs.referees[referee.RefereeID] = referee

	// Save to disk
	if err := rs.saveReferee(referee); err != nil {
		return fmt.Errorf("failed to save referee: %w", err)
	}

	log.Printf("👔 Added/Updated referee: %s (#%d)", referee.FullName, referee.JerseyNumber)
	return nil
}

// GetReferee retrieves a referee by ID
func (rs *RefereeService) GetReferee(refereeID int) (*models.Referee, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	referee, exists := rs.referees[refereeID]
	if !exists {
		return nil, fmt.Errorf("referee %d not found", refereeID)
	}

	return referee, nil
}

// GetAllReferees returns all referees in the system
func (rs *RefereeService) GetAllReferees() []*models.Referee {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	referees := make([]*models.Referee, 0, len(rs.referees))
	for _, ref := range rs.referees {
		referees = append(referees, ref)
	}

	return referees
}

// ============================================================================
// GAME ASSIGNMENT MANAGEMENT
// ============================================================================

// AddGameAssignment adds a referee assignment for a game
func (rs *RefereeService) AddGameAssignment(assignment *models.RefereeGameAssignment) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	assignment.LastUpdated = time.Now()
	rs.gameAssignments[assignment.GameID] = assignment

	// Add to daily schedule
	dateKey := assignment.GameDate.Format("2006-01-02")
	if schedule, exists := rs.dailySchedules[dateKey]; exists {
		// Update existing schedule
		found := false
		for i, existing := range schedule.Assignments {
			if existing.GameID == assignment.GameID {
				schedule.Assignments[i] = *assignment
				found = true
				break
			}
		}
		if !found {
			schedule.Assignments = append(schedule.Assignments, *assignment)
		}
		schedule.LastUpdated = time.Now()
	} else {
		// Create new schedule
		rs.dailySchedules[dateKey] = &models.RefereeDailySchedule{
			Date:        assignment.GameDate,
			Assignments: []models.RefereeGameAssignment{*assignment},
			LastUpdated: time.Now(),
		}
	}

	// Save to disk
	if err := rs.saveGameAssignment(assignment); err != nil {
		return fmt.Errorf("failed to save game assignment: %w", err)
	}

	log.Printf("📋 Referee assignment added for game %d: %s & %s",
		assignment.GameID, assignment.Referee1Name, assignment.Referee2Name)

	return nil
}

// GetGameAssignment retrieves referee assignment for a game
func (rs *RefereeService) GetGameAssignment(gameID int) (*models.RefereeGameAssignment, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	assignment, exists := rs.gameAssignments[gameID]
	if !exists {
		return nil, fmt.Errorf("no referee assignment found for game %d", gameID)
	}

	return assignment, nil
}

// GetDailySchedule retrieves all referee assignments for a specific date
func (rs *RefereeService) GetDailySchedule(date time.Time) (*models.RefereeDailySchedule, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	dateKey := date.Format("2006-01-02")
	schedule, exists := rs.dailySchedules[dateKey]
	if !exists {
		return nil, fmt.Errorf("no referee schedule found for %s", dateKey)
	}

	return schedule, nil
}

// ============================================================================
// STATS MANAGEMENT
// ============================================================================

// UpdateRefereeStats updates or adds season statistics for a referee
func (rs *RefereeService) UpdateRefereeStats(stats *models.RefereeSeasonStats) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	stats.LastUpdated = time.Now()
	rs.seasonStats[stats.RefereeID] = stats

	// Save to disk
	if err := rs.saveRefereeStats(stats); err != nil {
		return fmt.Errorf("failed to save referee stats: %w", err)
	}

	log.Printf("📊 Updated stats for referee %d: %d games, %.2f pen/game",
		stats.RefereeID, stats.GamesOfficiated, stats.AvgPenaltiesPerGame)

	return nil
}

// GetRefereeStats retrieves season statistics for a referee
func (rs *RefereeService) GetRefereeStats(refereeID int) (*models.RefereeSeasonStats, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	stats, exists := rs.seasonStats[refereeID]
	if !exists {
		return nil, fmt.Errorf("no stats found for referee %d", refereeID)
	}

	return stats, nil
}

// ============================================================================
// BIAS TRACKING
// ============================================================================

// AddTeamBias adds or updates referee bias data for a team
func (rs *RefereeService) AddTeamBias(bias *models.RefereeTeamBias) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	bias.LastUpdated = time.Now()
	key := fmt.Sprintf("%d_%s", bias.RefereeID, bias.TeamCode)

	// Find existing or add new
	biases := rs.teamBiases[key]
	found := false
	for i, existing := range biases {
		if existing.Season == bias.Season {
			biases[i] = bias
			found = true
			break
		}
	}
	if !found {
		biases = append(biases, bias)
	}
	rs.teamBiases[key] = biases

	// Calculate bias score
	rs.calculateBiasScore(bias)

	// Save to disk
	if err := rs.saveTeamBias(bias); err != nil {
		return fmt.Errorf("failed to save team bias: %w", err)
	}

	log.Printf("📈 Updated bias for referee %d vs %s: %.2f penalties/game (bias: %.2f)",
		bias.RefereeID, bias.TeamCode, bias.AvgPenaltiesPerGame, bias.BiasScore)

	return nil
}

// calculateBiasScore calculates a bias score for a referee-team pairing
// Positive = favors team, Negative = against team
func (rs *RefereeService) calculateBiasScore(bias *models.RefereeTeamBias) {
	// Get referee's overall average
	refStats, exists := rs.seasonStats[bias.RefereeID]
	if !exists {
		bias.BiasScore = 0
		return
	}

	// Compare team-specific rate to overall average
	// Bias score is the difference in penalties per game
	overallAvg := refStats.AvgPenaltiesPerGame
	teamAvg := bias.AvgPenaltiesPerGame

	// Negative score = more penalties than average (against team)
	// Positive score = fewer penalties than average (favors team)
	bias.BiasScore = overallAvg - teamAvg

	// Factor in home/away split if significant
	if bias.HomeGames > 0 && bias.AwayGames > 0 {
		homePenRate := float64(bias.HomePenalties) / float64(bias.HomeGames)
		awayPenRate := float64(bias.AwayPenalties) / float64(bias.AwayGames)
		homeAwayDiff := homePenRate - awayPenRate

		// Adjust bias score based on home/away difference
		bias.BiasScore += homeAwayDiff * 0.5
	}
}

// GetTeamBias retrieves bias data for a referee-team pairing
func (rs *RefereeService) GetTeamBias(refereeID int, teamCode string, season int) (*models.RefereeTeamBias, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	key := fmt.Sprintf("%d_%s", refereeID, teamCode)
	biases, exists := rs.teamBiases[key]
	if !exists {
		return nil, fmt.Errorf("no bias data found for referee %d vs %s", refereeID, teamCode)
	}

	// Find the specific season
	for _, bias := range biases {
		if bias.Season == season {
			return bias, nil
		}
	}

	return nil, fmt.Errorf("no bias data found for referee %d vs %s in season %d", refereeID, teamCode, season)
}

// ============================================================================
// IMPACT ANALYSIS
// ============================================================================

// AnalyzeRefereeImpact calculates the potential impact of referees on a game
func (rs *RefereeService) AnalyzeRefereeImpact(gameID int, homeTeam, awayTeam string) (*models.RefereeImpactAnalysis, error) {
	assignment, err := rs.GetGameAssignment(gameID)
	if err != nil {
		return nil, fmt.Errorf("no referee assignment for game: %w", err)
	}

	analysis := &models.RefereeImpactAnalysis{
		GameID:     gameID,
		Referee1ID: assignment.Referee1ID,
		Referee2ID: assignment.Referee2ID,
	}

	// Analyze both referees
	ref1Impact := rs.analyzeRefereeBias(assignment.Referee1ID, homeTeam, awayTeam)
	ref2Impact := rs.analyzeRefereeBias(assignment.Referee2ID, homeTeam, awayTeam)

	// Average the impacts from both referees
	analysis.HomeTeamBiasScore = (ref1Impact.HomeTeamBiasScore + ref2Impact.HomeTeamBiasScore) / 2
	analysis.AwayTeamBiasScore = (ref1Impact.AwayTeamBiasScore + ref2Impact.AwayTeamBiasScore) / 2
	analysis.ExpectedPenalties = (ref1Impact.ExpectedPenalties + ref2Impact.ExpectedPenalties) / 2
	analysis.TotalGoalsImpact = (ref1Impact.TotalGoalsImpact + ref2Impact.TotalGoalsImpact) / 2

	// Calculate home advantage adjustment based on bias
	analysis.HomeAdvantageAdjust = analysis.HomeTeamBiasScore - analysis.AwayTeamBiasScore

	// Calculate confidence based on data availability
	analysis.ConfidenceLevel = (ref1Impact.ConfidenceLevel + ref2Impact.ConfidenceLevel) / 2

	// Generate notes
	if analysis.HomeAdvantageAdjust > 0.5 {
		analysis.Notes = fmt.Sprintf("Referees historically favor home team %s", homeTeam)
	} else if analysis.HomeAdvantageAdjust < -0.5 {
		analysis.Notes = fmt.Sprintf("Referees historically favor away team %s", awayTeam)
	} else {
		analysis.Notes = "No significant referee bias detected"
	}

	return analysis, nil
}

// analyzeRefereeBias analyzes a single referee's bias for both teams
func (rs *RefereeService) analyzeRefereeBias(refereeID int, homeTeam, awayTeam string) *models.RefereeImpactAnalysis {
	analysis := &models.RefereeImpactAnalysis{
		Referee1ID: refereeID,
	}

	// Get referee stats
	stats, err := rs.GetRefereeStats(refereeID)
	if err != nil {
		analysis.ConfidenceLevel = 0
		return analysis
	}

	analysis.ExpectedPenalties = stats.AvgPenaltiesPerGame
	analysis.TotalGoalsImpact = stats.AvgTotalGoals

	// Get team-specific bias for home team
	if homeBias, err := rs.GetTeamBias(refereeID, homeTeam, rs.currentSeason); err == nil {
		analysis.HomeTeamBiasScore = homeBias.BiasScore
		analysis.ConfidenceLevel += 0.5
	}

	// Get team-specific bias for away team
	if awayBias, err := rs.GetTeamBias(refereeID, awayTeam, rs.currentSeason); err == nil {
		analysis.AwayTeamBiasScore = awayBias.BiasScore
		analysis.ConfidenceLevel += 0.5
	}

	// Confidence based on games officiated
	if stats.GamesOfficiated < 10 {
		analysis.ConfidenceLevel *= 0.5
	}

	return analysis
}

// ============================================================================
// ADVANCED ANALYTICS (PHASE 3)
// ============================================================================

// CalculateRefereeTendencies analyzes a referee's calling patterns and tendencies
func (rs *RefereeService) CalculateRefereeTendencies(refereeID int) (*models.RefereeTendency, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	// Get referee stats
	stats, exists := rs.seasonStats[refereeID]
	if !exists {
		return nil, fmt.Errorf("no stats found for referee %d", refereeID)
	}

	// Calculate league averages for comparison
	leagueAvg := rs.calculateLeagueAverages()

	tendency := &models.RefereeTendency{
		RefereeID:   refereeID,
		Season:      stats.Season,
		LastUpdated: time.Now(),
	}

	// 1. Penalty Call Rate (compared to league average)
	if leagueAvg.AvgPenaltiesPerGame > 0 {
		tendency.PenaltyCallRate = stats.AvgPenaltiesPerGame / leagueAvg.AvgPenaltiesPerGame
	}

	// 2. Determine Tendency Type
	if tendency.PenaltyCallRate > 1.15 {
		tendency.TendencyType = "strict"
	} else if tendency.PenaltyCallRate < 0.85 {
		tendency.TendencyType = "lenient"
	} else {
		tendency.TendencyType = "average"
	}

	// 3. Home Advantage Impact (difference in home vs away win %)
	tendency.HomeAdvantageImpact = stats.HomeWinPct - stats.AwayWinPct
	tendency.HomeWinBias = stats.HomeWinPct - 0.55 // League average home win% is ~55%

	// 4. High Scoring Games
	if leagueAvg.AvgTotalGoals > 0 {
		tendency.HighScoringGames = stats.AvgTotalGoals > leagueAvg.AvgTotalGoals*1.1
	}

	// 5. Penalty Type Breakdown
	if stats.TotalPenalties > 0 {
		tendency.MinorPenaltyRate = float64(stats.MinorPenalties) / float64(stats.TotalPenalties)
		tendency.MajorPenaltyRate = float64(stats.MajorPenalties) / float64(stats.TotalPenalties)
	}

	// 6. Power Play Impact (based on penalties per game)
	tendency.PowerPlayImpact = stats.AvgPenaltiesPerGame - leagueAvg.AvgPenaltiesPerGame

	// 7. Consistency Score (based on game count and data quality)
	if stats.GamesOfficiated >= 50 {
		tendency.ConsistencyScore = 0.9
	} else if stats.GamesOfficiated >= 30 {
		tendency.ConsistencyScore = 0.75
	} else if stats.GamesOfficiated >= 15 {
		tendency.ConsistencyScore = 0.6
	} else {
		tendency.ConsistencyScore = 0.4
	}

	// 8. Over/Under Tendency
	if stats.OverPct > 0.55 {
		tendency.OverUnderTendency = "over"
	} else if stats.UnderPct > 0.55 {
		tendency.OverUnderTendency = "under"
	} else {
		tendency.OverUnderTendency = "neutral"
	}

	// 9. Physicality Tolerance (based on major penalties)
	if tendency.MajorPenaltyRate > 0.15 {
		tendency.PhysicalityTolerance = "low"
	} else if tendency.MajorPenaltyRate < 0.05 {
		tendency.PhysicalityTolerance = "high"
	} else {
		tendency.PhysicalityTolerance = "medium"
	}

	return tendency, nil
}

// calculateLeagueAverages computes league-wide averages for comparison
func (rs *RefereeService) calculateLeagueAverages() *models.RefereeSeasonStats {
	var totalGames, totalPenalties int
	var totalPenaltiesPerGame, totalGoalsPerGame float64
	var count int

	for _, stats := range rs.seasonStats {
		if stats.GamesOfficiated >= 10 { // Only include refs with sufficient games
			totalGames += stats.GamesOfficiated
			totalPenalties += stats.TotalPenalties
			totalPenaltiesPerGame += stats.AvgPenaltiesPerGame
			totalGoalsPerGame += stats.AvgTotalGoals
			count++
		}
	}

	avg := &models.RefereeSeasonStats{}
	if count > 0 {
		avg.AvgPenaltiesPerGame = totalPenaltiesPerGame / float64(count)
		avg.AvgTotalGoals = totalGoalsPerGame / float64(count)
		avg.GamesOfficiated = totalGames / count
	}

	return avg
}

// GenerateRefereeProfile creates a comprehensive profile for a referee
func (rs *RefereeService) GenerateRefereeProfile(refereeID int) (*models.RefereeProfile, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	// Get basic referee info
	referee, exists := rs.referees[refereeID]
	if !exists {
		return nil, fmt.Errorf("referee %d not found", refereeID)
	}

	profile := &models.RefereeProfile{
		Referee:          *referee,
		LastUpdated:      time.Now(),
		PredictionImpact: make(map[string]float64),
		CareerSummary:    make(map[string]interface{}),
	}

	// Get current season stats
	if stats, exists := rs.seasonStats[refereeID]; exists {
		profile.CurrentStats = stats
		profile.ConfidenceLevel += 0.3
	}

	// Calculate tendencies
	if tendencies, err := rs.CalculateRefereeTendencies(refereeID); err == nil {
		profile.Tendencies = tendencies
		profile.ConfidenceLevel += 0.3
	}

	// Gather team biases
	teamBiases := []models.RefereeTeamBias{}
	for _, biases := range rs.teamBiases {
		for _, bias := range biases {
			if bias.RefereeID == refereeID {
				teamBiases = append(teamBiases, *bias)
			}
		}
	}
	profile.TeamBiases = teamBiases
	if len(teamBiases) > 0 {
		profile.ConfidenceLevel += 0.2
	}

	// Find recent game assignments
	recentGames := []models.RefereeGameAssignment{}
	for _, assignment := range rs.gameAssignments {
		if assignment.Referee1ID == refereeID || assignment.Referee2ID == refereeID {
			recentGames = append(recentGames, *assignment)
			if len(recentGames) >= 10 {
				break
			}
		}
	}
	profile.RecentGames = recentGames
	if len(recentGames) > 0 {
		profile.ConfidenceLevel += 0.2
	}

	// Build career summary
	profile.CareerSummary["totalGames"] = referee.CareerGames
	profile.CareerSummary["seasonGames"] = referee.SeasonGames
	if profile.CurrentStats != nil {
		profile.CareerSummary["avgPenaltiesPerGame"] = profile.CurrentStats.AvgPenaltiesPerGame
		profile.CareerSummary["homeWinPct"] = profile.CurrentStats.HomeWinPct
		profile.CareerSummary["avgTotalGoals"] = profile.CurrentStats.AvgTotalGoals
	}

	// Calculate prediction impact factors
	if profile.Tendencies != nil {
		profile.PredictionImpact["penaltyRate"] = profile.Tendencies.PenaltyCallRate
		profile.PredictionImpact["homeAdvantage"] = profile.Tendencies.HomeAdvantageImpact
		profile.PredictionImpact["totalGoalsImpact"] = profile.Tendencies.PowerPlayImpact
		profile.PredictionImpact["consistency"] = profile.Tendencies.ConsistencyScore
	}

	// Normalize confidence level (max 1.0)
	if profile.ConfidenceLevel > 1.0 {
		profile.ConfidenceLevel = 1.0
	}

	return profile, nil
}

// GetAllTeamBiases retrieves all bias data for a specific referee
func (rs *RefereeService) GetAllTeamBiases(refereeID int) ([]models.RefereeTeamBias, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	biases := []models.RefereeTeamBias{}
	for _, biasGroup := range rs.teamBiases {
		for _, bias := range biasGroup {
			if bias.RefereeID == refereeID {
				biases = append(biases, *bias)
			}
		}
	}

	if len(biases) == 0 {
		return nil, fmt.Errorf("no bias data found for referee %d", refereeID)
	}

	return biases, nil
}

// GetAdvancedImpactAnalysis provides detailed impact analysis with tendencies
func (rs *RefereeService) GetAdvancedImpactAnalysis(gameID int, homeTeam, awayTeam string) (map[string]interface{}, error) {
	// Get basic impact analysis
	basicImpact, err := rs.AnalyzeRefereeImpact(gameID, homeTeam, awayTeam)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	result["basicImpact"] = basicImpact

	// Get referee profiles for both refs
	ref1Profile, err1 := rs.GenerateRefereeProfile(basicImpact.Referee1ID)
	ref2Profile, err2 := rs.GenerateRefereeProfile(basicImpact.Referee2ID)

	if err1 == nil {
		result["referee1Profile"] = ref1Profile
	}
	if err2 == nil {
		result["referee2Profile"] = ref2Profile
	}

	// Calculate combined tendencies
	combinedTendencies := make(map[string]interface{})
	if err1 == nil && err2 == nil && ref1Profile.Tendencies != nil && ref2Profile.Tendencies != nil {
		combinedTendencies["avgPenaltyCallRate"] = (ref1Profile.Tendencies.PenaltyCallRate + ref2Profile.Tendencies.PenaltyCallRate) / 2
		combinedTendencies["avgHomeAdvantageImpact"] = (ref1Profile.Tendencies.HomeAdvantageImpact + ref2Profile.Tendencies.HomeAdvantageImpact) / 2
		combinedTendencies["avgConsistency"] = (ref1Profile.Tendencies.ConsistencyScore + ref2Profile.Tendencies.ConsistencyScore) / 2
		
		// Determine overall tendency type
		if ref1Profile.Tendencies.TendencyType == ref2Profile.Tendencies.TendencyType {
			combinedTendencies["overallTendency"] = ref1Profile.Tendencies.TendencyType
		} else {
			combinedTendencies["overallTendency"] = "mixed"
		}
		
		result["combinedTendencies"] = combinedTendencies
	}

	// Add specific recommendations
	recommendations := []string{}
	if basicImpact.ExpectedPenalties > 7.0 {
		recommendations = append(recommendations, "Expect higher than average penalty count")
	}
	if basicImpact.HomeAdvantageAdjust > 0.5 {
		recommendations = append(recommendations, fmt.Sprintf("Home team %s has historical advantage with these refs", homeTeam))
	} else if basicImpact.HomeAdvantageAdjust < -0.5 {
		recommendations = append(recommendations, fmt.Sprintf("Away team %s has historical advantage with these refs", awayTeam))
	}
	if ref1Profile != nil && ref1Profile.Tendencies != nil && ref1Profile.Tendencies.HighScoringGames {
		recommendations = append(recommendations, "Referee tendencies suggest high-scoring game possible")
	}
	
	result["recommendations"] = recommendations
	result["confidenceLevel"] = basicImpact.ConfidenceLevel

	return result, nil
}

// ============================================================================
// PERSISTENCE LAYER
// ============================================================================

// loadAllData loads all referee data from disk
func (rs *RefereeService) loadAllData() error {
	log.Printf("📂 Loading referee data from disk...")

	// Load referees
	if err := rs.loadReferees(); err != nil {
		log.Printf("⚠️ Failed to load referees: %v", err)
	}

	// Load stats
	if err := rs.loadAllStats(); err != nil {
		log.Printf("⚠️ Failed to load stats: %v", err)
	}

	// Load assignments
	if err := rs.loadAllAssignments(); err != nil {
		log.Printf("⚠️ Failed to load assignments: %v", err)
	}

	log.Printf("✅ Loaded %d referees, %d stats entries, %d assignments",
		len(rs.referees), len(rs.seasonStats), len(rs.gameAssignments))

	return nil
}

// saveReferee saves a referee to disk
func (rs *RefereeService) saveReferee(referee *models.Referee) error {
	filePath := filepath.Join(rs.dataDir, "referees", fmt.Sprintf("referee_%d.json", referee.RefereeID))
	data, err := json.MarshalIndent(referee, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, data, 0644)
}

// loadReferees loads all referees from disk
func (rs *RefereeService) loadReferees() error {
	refDir := filepath.Join(rs.dataDir, "referees")
	files, err := ioutil.ReadDir(refDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			data, err := ioutil.ReadFile(filepath.Join(refDir, file.Name()))
			if err != nil {
				continue
			}

			var ref models.Referee
			if err := json.Unmarshal(data, &ref); err != nil {
				continue
			}

			rs.referees[ref.RefereeID] = &ref
		}
	}

	return nil
}

// saveRefereeStats saves referee stats to disk
func (rs *RefereeService) saveRefereeStats(stats *models.RefereeSeasonStats) error {
	filePath := filepath.Join(rs.dataDir, "stats", fmt.Sprintf("referee_%d_season_%d.json", stats.RefereeID, stats.Season))
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, data, 0644)
}

// loadAllStats loads all referee stats from disk
func (rs *RefereeService) loadAllStats() error {
	statsDir := filepath.Join(rs.dataDir, "stats")
	files, err := ioutil.ReadDir(statsDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			data, err := ioutil.ReadFile(filepath.Join(statsDir, file.Name()))
			if err != nil {
				continue
			}

			var stats models.RefereeSeasonStats
			if err := json.Unmarshal(data, &stats); err != nil {
				continue
			}

			rs.seasonStats[stats.RefereeID] = &stats
		}
	}

	return nil
}

// saveGameAssignment saves a game assignment to disk
func (rs *RefereeService) saveGameAssignment(assignment *models.RefereeGameAssignment) error {
	filePath := filepath.Join(rs.dataDir, "assignments", fmt.Sprintf("game_%d.json", assignment.GameID))
	data, err := json.MarshalIndent(assignment, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, data, 0644)
}

// loadAllAssignments loads all game assignments from disk
func (rs *RefereeService) loadAllAssignments() error {
	assignDir := filepath.Join(rs.dataDir, "assignments")
	files, err := ioutil.ReadDir(assignDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			data, err := ioutil.ReadFile(filepath.Join(assignDir, file.Name()))
			if err != nil {
				continue
			}

			var assignment models.RefereeGameAssignment
			if err := json.Unmarshal(data, &assignment); err != nil {
				continue
			}

			rs.gameAssignments[assignment.GameID] = &assignment

			// Add to daily schedule
			dateKey := assignment.GameDate.Format("2006-01-02")
			if schedule, exists := rs.dailySchedules[dateKey]; exists {
				schedule.Assignments = append(schedule.Assignments, assignment)
			} else {
				rs.dailySchedules[dateKey] = &models.RefereeDailySchedule{
					Date:        assignment.GameDate,
					Assignments: []models.RefereeGameAssignment{assignment},
					LastUpdated: time.Now(),
				}
			}
		}
	}

	return nil
}

// saveTeamBias saves team bias data to disk
func (rs *RefereeService) saveTeamBias(bias *models.RefereeTeamBias) error {
	filePath := filepath.Join(rs.dataDir, "bias", fmt.Sprintf("referee_%d_team_%s_season_%d.json",
		bias.RefereeID, bias.TeamCode, bias.Season))
	data, err := json.MarshalIndent(bias, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, data, 0644)
}

