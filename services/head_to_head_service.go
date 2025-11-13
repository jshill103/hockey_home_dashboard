package services

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

var (
	headToHeadInstance *HeadToHeadService
	headToHeadOnce     sync.Once
)

// HeadToHeadService tracks historical matchup performance between teams
type HeadToHeadService struct {
	mu           sync.RWMutex
	records      map[string]*models.HeadToHeadRecord // key: "TEAM1_TEAM2" (alphabetically sorted)
	dataDir      string
	currentSeason string
}

// InitializeHeadToHead initializes the singleton HeadToHeadService
func InitializeHeadToHead() error {
	var err error
	headToHeadOnce.Do(func() {
		dataDir := filepath.Join("data", "head_to_head")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create head-to-head directory: %w", err)
			return
		}

		headToHeadInstance = &HeadToHeadService{
			records:       make(map[string]*models.HeadToHeadRecord),
			dataDir:       dataDir,
			currentSeason: GetCurrentSeason(),
		}

		// Load existing records
		if loadErr := headToHeadInstance.loadAllRecords(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing head-to-head records: %v\n", loadErr)
		}

		fmt.Println("✅ Head-to-Head Service initialized")
	})
	return err
}

// GetHeadToHeadService returns the singleton instance
func GetHeadToHeadService() *HeadToHeadService {
	return headToHeadInstance
}

// RecordGame adds a completed game to the head-to-head history
func (h2h *HeadToHeadService) RecordGame(homeTeam, awayTeam string, gameID int, gameDate time.Time,
	homeScore, awayScore int, wasOvertime bool, homeGoalie, awayGoalie string) error {
	
	h2h.mu.Lock()
	defer h2h.mu.Unlock()

	key := h2h.getMatchupKey(homeTeam, awayTeam)
	record := h2h.getOrCreateRecord(homeTeam, awayTeam)

	// Create game result
	winner := homeTeam
	if awayScore > homeScore {
		winner = awayTeam
	}

	game := models.H2HGame{
		GameID:      gameID,
		Date:        gameDate,
		HomeScore:   homeScore,
		AwayScore:   awayScore,
		Winner:      winner,
		WasBlowout:  math.Abs(float64(homeScore-awayScore)) >= 3,
		WasOvertime: wasOvertime,
		HomeGoalie:  homeGoalie,
		AwayGoalie:  awayGoalie,
		Recency:     1.0, // Will be recalculated
	}

	// Add to recent games (keep last 10)
	record.RecentGames = append(record.RecentGames, game)
	if len(record.RecentGames) > 10 {
		record.RecentGames = record.RecentGames[len(record.RecentGames)-10:]
	}

	// Update statistics
	record.TotalGames++
	if winner == homeTeam {
		record.HomeWins++
	} else if winner == awayTeam {
		record.AwayWins++
	} else {
		record.Ties++
	}

	if game.WasBlowout {
		if winner == homeTeam {
			record.HomeBlowoutWins++
		} else {
			record.AwayBlowoutWins++
		}
	}

	if wasOvertime {
		record.OvertimeGames++
	}

	if math.Abs(float64(homeScore-awayScore)) == 1 {
		record.CloseGames++
	}

	// Update averages
	totalHomeGoals := 0
	totalAwayGoals := 0
	for _, g := range record.RecentGames {
		totalHomeGoals += g.HomeScore
		totalAwayGoals += g.AwayScore
	}
	if len(record.RecentGames) > 0 {
		record.AvgHomeGoals = float64(totalHomeGoals) / float64(len(record.RecentGames))
		record.AvgAwayGoals = float64(totalAwayGoals) / float64(len(record.RecentGames))
	}

	// Update metadata
	record.LastMeetingDate = gameDate
	record.DaysSinceLastMeet = int(time.Since(gameDate).Hours() / 24)
	record.LastUpdated = time.Now()

	// Recalculate weighted advantage
	h2h.calculateWeightedAdvantage(record)

	// Update win streak
	h2h.updateWinStreak(record)

	// Save to disk
	h2h.records[key] = record
	return h2h.saveRecord(key, record)
}

// GetMatchupAnalysis returns the head-to-head analysis for a specific matchup
func (h2h *HeadToHeadService) GetMatchupAnalysis(homeTeam, awayTeam string) (*models.HeadToHeadRecord, error) {
	h2h.mu.RLock()
	defer h2h.mu.RUnlock()

	key := h2h.getMatchupKey(homeTeam, awayTeam)
	record, exists := h2h.records[key]
	if !exists {
		// Return empty record if no history
		return &models.HeadToHeadRecord{
			HomeTeam:          homeTeam,
			AwayTeam:          awayTeam,
			Season:            h2h.currentSeason,
			WeightedAdvantage: 0.0,
		}, nil
	}

	// Recalculate recency weights
	h2h.calculateRecencyWeights(record)

	// Update days since last meeting
	record.DaysSinceLastMeet = int(time.Since(record.LastMeetingDate).Hours() / 24)

	return record, nil
}

// CalculateAdvantage returns a specific advantage calculation for prediction
func (h2h *HeadToHeadService) CalculateAdvantage(homeTeam, awayTeam string) *models.HeadToHeadAdvantage {
	record, _ := h2h.GetMatchupAnalysis(homeTeam, awayTeam)

	advantage := &models.HeadToHeadAdvantage{
		HomeTeam:   homeTeam,
		AwayTeam:   awayTeam,
		Advantage:  record.WeightedAdvantage,
		SampleSize: record.TotalGames,
		KeyFactors: []string{},
	}

	// Calculate confidence based on sample size (sigmoid function)
	if record.TotalGames > 0 {
		advantage.Confidence = 1.0 - math.Exp(-float64(record.TotalGames)/5.0)
	}

	// Determine recent trend
	if len(record.RecentGames) >= 3 {
		recentHomeWins := 0
		recentAwayWins := 0
		for i := len(record.RecentGames) - 3; i < len(record.RecentGames); i++ {
			if record.RecentGames[i].Winner == homeTeam {
				recentHomeWins++
			} else if record.RecentGames[i].Winner == awayTeam {
				recentAwayWins++
			}
		}

		if recentHomeWins >= 2 {
			advantage.RecentTrend = "Home dominant"
			advantage.KeyFactors = append(advantage.KeyFactors, fmt.Sprintf("Won %d of last 3 meetings", recentHomeWins))
		} else if recentAwayWins >= 2 {
			advantage.RecentTrend = "Away surge"
			advantage.KeyFactors = append(advantage.KeyFactors, fmt.Sprintf("Won %d of last 3 meetings", recentAwayWins))
		} else {
			advantage.RecentTrend = "Balanced"
		}
	} else {
		advantage.RecentTrend = "Insufficient data"
	}

	// Add key factors
	if record.AvgHomeGoals > record.AvgAwayGoals+0.5 {
		advantage.KeyFactors = append(advantage.KeyFactors, fmt.Sprintf("Home averages %.1f goals, away %.1f", record.AvgHomeGoals, record.AvgAwayGoals))
	}

	if record.HomeBlowoutWins > record.AwayBlowoutWins {
		advantage.KeyFactors = append(advantage.KeyFactors, fmt.Sprintf("Home has %d blowout wins vs %d", record.HomeBlowoutWins, record.AwayBlowoutWins))
	}

	if record.CloseGames > record.TotalGames/2 {
		advantage.KeyFactors = append(advantage.KeyFactors, "Historically close matchup")
	}

	// Calculate recency bias (how much recent games are weighted)
	advantage.RecencyBias = h2h.calculateRecencyBias(record)

	return advantage
}

// BackfillFromGames populates head-to-head records from historical game data
// Note: This is a placeholder. Full implementation requires game outcome data
func (h2h *HeadToHeadService) BackfillFromGames(games []models.Game) error {
	fmt.Println("📊 Head-to-Head backfill: placeholder (requires completed game data)")
	// TODO: Implement backfill when we have access to completed game results
	// This would require fetching data from game summary API or scoreboard
	return nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// getMatchupKey creates a consistent key for team pairs
func (h2h *HeadToHeadService) getMatchupKey(team1, team2 string) string {
	// Always use alphabetical order for consistency
	if team1 < team2 {
		return team1 + "_" + team2
	}
	return team2 + "_" + team1
}

// getOrCreateRecord retrieves or creates a new head-to-head record
func (h2h *HeadToHeadService) getOrCreateRecord(homeTeam, awayTeam string) *models.HeadToHeadRecord {
	key := h2h.getMatchupKey(homeTeam, awayTeam)
	record, exists := h2h.records[key]
	if !exists {
		record = &models.HeadToHeadRecord{
			HomeTeam:      homeTeam,
			AwayTeam:      awayTeam,
			Season:        h2h.currentSeason,
			RecentGames:   []models.H2HGame{},
			LastUpdated:   time.Now(),
		}
		h2h.records[key] = record
	}
	return record
}

// calculateWeightedAdvantage computes a composite advantage score
func (h2h *HeadToHeadService) calculateWeightedAdvantage(record *models.HeadToHeadRecord) {
	if record.TotalGames == 0 {
		record.WeightedAdvantage = 0.0
		return
	}

	// Base advantage from win percentage
	homeWinPct := float64(record.HomeWins) / float64(record.TotalGames)
	awayWinPct := float64(record.AwayWins) / float64(record.TotalGames)
	baseAdvantage := (homeWinPct - awayWinPct) * 0.3 // Scale to -0.3 to +0.3

	// Recent form adjustment (weight last 3 games more heavily)
	recentBonus := 0.0
	if len(record.RecentGames) >= 3 {
		recentHomeWins := 0
		for i := len(record.RecentGames) - 3; i < len(record.RecentGames); i++ {
			if record.RecentGames[i].Winner == record.HomeTeam {
				recentHomeWins++
			}
		}
		// Recent form can add up to ±0.1
		recentBonus = (float64(recentHomeWins) - 1.5) / 15.0 // Centered at 1.5 wins out of 3
	}

	// Goal differential adjustment
	goalDiffAdvantage := 0.0
	if record.AvgHomeGoals > 0 && record.AvgAwayGoals > 0 {
		goalDiff := record.AvgHomeGoals - record.AvgAwayGoals
		goalDiffAdvantage = math.Tanh(goalDiff / 3.0) * 0.1 // Scale to ±0.1
	}

	// Combine factors
	record.WeightedAdvantage = baseAdvantage + recentBonus + goalDiffAdvantage

	// Clamp to [-0.30, +0.30]
	if record.WeightedAdvantage > 0.30 {
		record.WeightedAdvantage = 0.30
	} else if record.WeightedAdvantage < -0.30 {
		record.WeightedAdvantage = -0.30
	}
}

// calculateRecencyWeights assigns exponential decay weights to historical games
func (h2h *HeadToHeadService) calculateRecencyWeights(record *models.HeadToHeadRecord) {
	if len(record.RecentGames) == 0 {
		return
	}

	// Sort games by date (most recent first)
	sort.Slice(record.RecentGames, func(i, j int) bool {
		return record.RecentGames[i].Date.After(record.RecentGames[j].Date)
	})

	// Assign exponential decay weights (most recent = 1.0, decay by 0.85)
	decayFactor := 0.85
	for i := range record.RecentGames {
		record.RecentGames[i].Recency = math.Pow(decayFactor, float64(i))
	}
}

// calculateRecencyBias determines how much recent games are weighted
func (h2h *HeadToHeadService) calculateRecencyBias(record *models.HeadToHeadRecord) float64 {
	if len(record.RecentGames) < 2 {
		return 0.0
	}

	// Calculate average recency weight
	totalRecency := 0.0
	for _, game := range record.RecentGames {
		totalRecency += game.Recency
	}
	avgRecency := totalRecency / float64(len(record.RecentGames))

	// Normalize to 0-1 (higher = more recent games weighted)
	return 1.0 - avgRecency
}

// updateWinStreak updates the current win/loss streak
func (h2h *HeadToHeadService) updateWinStreak(record *models.HeadToHeadRecord) {
	if len(record.RecentGames) == 0 {
		record.HomeWinStreak = 0
		return
	}

	// Count consecutive wins/losses from most recent game
	streak := 0
	lastWinner := record.RecentGames[len(record.RecentGames)-1].Winner
	isHomeStreak := lastWinner == record.HomeTeam

	for i := len(record.RecentGames) - 1; i >= 0; i-- {
		if record.RecentGames[i].Winner == lastWinner {
			if isHomeStreak {
				streak++
			} else {
				streak--
			}
		} else {
			break
		}
	}

	record.HomeWinStreak = streak
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (h2h *HeadToHeadService) getRecordFilePath(key string) string {
	return filepath.Join(h2h.dataDir, fmt.Sprintf("%s_%s.json", key, h2h.currentSeason))
}

func (h2h *HeadToHeadService) saveRecord(key string, record *models.HeadToHeadRecord) error {
	filePath := h2h.getRecordFilePath(key)
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal head-to-head record: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write head-to-head file: %w", err)
	}
	return nil
}

func (h2h *HeadToHeadService) loadRecord(key string) (*models.HeadToHeadRecord, error) {
	filePath := h2h.getRecordFilePath(key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read head-to-head file: %w", err)
	}

	var record models.HeadToHeadRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal head-to-head record: %w", err)
	}
	return &record, nil
}

func (h2h *HeadToHeadService) loadAllRecords() error {
	files, err := os.ReadDir(h2h.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read head-to-head directory: %w", err)
	}

	loaded := 0
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(h2h.dataDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("⚠️ Warning: Failed to load %s: %v\n", file.Name(), err)
			continue
		}

		var record models.HeadToHeadRecord
		if err := json.Unmarshal(data, &record); err != nil {
			fmt.Printf("⚠️ Warning: Failed to unmarshal %s: %v\n", file.Name(), err)
			continue
		}

		key := h2h.getMatchupKey(record.HomeTeam, record.AwayTeam)
		h2h.records[key] = &record
		loaded++
	}

	if loaded > 0 {
		fmt.Printf("📊 Loaded %d head-to-head records\n", loaded)
	}
	return nil
}

