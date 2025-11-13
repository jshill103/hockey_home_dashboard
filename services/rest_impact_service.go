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
	restImpactInstance *RestImpactService
	restImpactOnce     sync.Once
)

// RestImpactService analyzes team performance based on rest days
type RestImpactService struct {
	mu            sync.RWMutex
	teamAnalysis  map[string]*models.RestImpactAnalysis // teamCode -> analysis
	dataDir       string
	currentSeason string
}

// InitializeRestImpact initializes the singleton RestImpactService
func InitializeRestImpact() error {
	var err error
	restImpactOnce.Do(func() {
		dataDir := filepath.Join("data", "rest_analysis")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create rest analysis directory: %w", err)
			return
		}

		restImpactInstance = &RestImpactService{
			teamAnalysis:  make(map[string]*models.RestImpactAnalysis),
			dataDir:       dataDir,
			currentSeason: GetCurrentSeason(),
		}

		// Load existing analysis
		if loadErr := restImpactInstance.loadAllAnalysis(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing rest analysis: %v\n", loadErr)
		}

		fmt.Println("✅ Rest Impact Service initialized")
	})
	return err
}

// GetRestImpactService returns the singleton instance
func GetRestImpactService() *RestImpactService {
	return restImpactInstance
}

// RecordGameWithRest records a game result with rest day information
func (ris *RestImpactService) RecordGameWithRest(teamCode string, restDays int, travelMiles float64,
	won bool, otLoss bool, goalsFor, goalsAgainst, shots, shotsAgainst, corsiFor int) error {

	ris.mu.Lock()
	defer ris.mu.Unlock()

	analysis := ris.getOrCreateAnalysis(teamCode)

	// Determine which record to update
	var record *models.TeamRestRecord
	switch {
	case restDays == 0:
		record = &analysis.BackToBackRecord
	case restDays == 1:
		record = &analysis.OneDayRestRecord
	case restDays == 2:
		record = &analysis.TwoDayRestRecord
	case restDays >= 3:
		record = &analysis.ThreePlusRestRecord
	}

	// Update the record
	ris.updateRecord(record, won, otLoss, goalsFor, goalsAgainst, shots, shotsAgainst, corsiFor)

	// If B2B with travel, also update that record
	if restDays == 0 && travelMiles > 500 {
		ris.updateRecord(&analysis.B2BWithTravelRecord, won, otLoss, goalsFor, goalsAgainst, shots, shotsAgainst, corsiFor)
	}

	// Update last 10 B2B record if applicable
	if restDays == 0 {
		ris.updateLast10B2B(analysis, won, otLoss, goalsFor, goalsAgainst)
	}

	// Recalculate derived metrics
	ris.calculateDerivedMetrics(analysis)

	analysis.LastUpdated = time.Now()

	// Save to disk
	return ris.saveAnalysis(teamCode, analysis)
}

// CalculateRestAdvantage compares rest situations between two teams
func (ris *RestImpactService) CalculateRestAdvantage(homeTeam, awayTeam string,
	homeRestDays, awayRestDays int, homeTravelMiles, awayTravelMiles float64) *models.RestAdvantageCalculation {

	ris.mu.RLock()
	defer ris.mu.RUnlock()

	homeAnalysis := ris.teamAnalysis[homeTeam]
	awayAnalysis := ris.teamAnalysis[awayTeam]

	calc := &models.RestAdvantageCalculation{
		HomeTeam:        homeTeam,
		AwayTeam:        awayTeam,
		HomeRestDays:    homeRestDays,
		AwayRestDays:    awayRestDays,
		HomeOnB2B:       homeRestDays == 0,
		AwayOnB2B:       awayRestDays == 0,
		HomeTravelMiles: homeTravelMiles,
		AwayTravelMiles: awayTravelMiles,
		KeyFactors:      []string{},
	}

	// Calculate team-specific B2B penalties
	if homeAnalysis != nil {
		calc.HomeB2BPenalty = homeAnalysis.B2BPenalty
	}
	if awayAnalysis != nil {
		calc.AwayB2BPenalty = awayAnalysis.B2BPenalty
	}

	// Calculate rest advantage
	restAdvantage := 0.0

	// B2B penalties
	if calc.HomeOnB2B && !calc.AwayOnB2B {
		restAdvantage += calc.HomeB2BPenalty // Negative value
		calc.KeyFactors = append(calc.KeyFactors, fmt.Sprintf("%s on back-to-back (%.1f%% penalty)", homeTeam, calc.HomeB2BPenalty*100))
	} else if calc.AwayOnB2B && !calc.HomeOnB2B {
		restAdvantage -= calc.AwayB2BPenalty // Negative value for away becomes positive for home
		calc.KeyFactors = append(calc.KeyFactors, fmt.Sprintf("%s on back-to-back (%.1f%% penalty)", awayTeam, calc.AwayB2BPenalty*100))
	} else if calc.HomeOnB2B && calc.AwayOnB2B {
		// Both on B2B, compare who handles it better
		diff := calc.AwayB2BPenalty - calc.HomeB2BPenalty
		restAdvantage += diff * 0.5 // Reduced impact when both tired
		calc.KeyFactors = append(calc.KeyFactors, "Both teams on back-to-back")
	}

	// Rest day differential (non-B2B)
	if !calc.HomeOnB2B && !calc.AwayOnB2B {
		restDiff := homeRestDays - awayRestDays
		if math.Abs(float64(restDiff)) >= 2 {
			// Significant rest advantage
			restBonus := math.Tanh(float64(restDiff)/3.0) * 0.05 // Up to ±0.05
			restAdvantage += restBonus
			if restDiff > 0 {
				calc.KeyFactors = append(calc.KeyFactors, fmt.Sprintf("%s has %d more days rest", homeTeam, restDiff))
			} else {
				calc.KeyFactors = append(calc.KeyFactors, fmt.Sprintf("%s has %d more days rest", awayTeam, -restDiff))
			}
		}
	}

	// Travel fatigue
	travelDiff := awayTravelMiles - homeTravelMiles
	if travelDiff > 1000 {
		travelPenalty := math.Tanh(travelDiff/3000.0) * 0.03 // Up to +0.03 for home
		restAdvantage += travelPenalty
		calc.KeyFactors = append(calc.KeyFactors, fmt.Sprintf("%s traveled %.0f miles", awayTeam, awayTravelMiles))
	} else if travelDiff < -1000 {
		travelPenalty := math.Tanh(-travelDiff/3000.0) * 0.03 // Up to +0.03 for away
		restAdvantage -= travelPenalty
		calc.KeyFactors = append(calc.KeyFactors, fmt.Sprintf("%s traveled %.0f miles", homeTeam, homeTravelMiles))
	}

	calc.RestAdvantage = restAdvantage

	// Clamp to [-0.20, +0.20]
	if calc.RestAdvantage > 0.20 {
		calc.RestAdvantage = 0.20
	} else if calc.RestAdvantage < -0.20 {
		calc.RestAdvantage = -0.20
	}

	// Determine fatigue advantage
	if calc.RestAdvantage > 0.05 {
		calc.FatigueAdvantage = "Home"
	} else if calc.RestAdvantage < -0.05 {
		calc.FatigueAdvantage = "Away"
	} else {
		calc.FatigueAdvantage = "Neutral"
	}

	// Calculate confidence based on sample size
	sampleSize := 0
	if homeAnalysis != nil {
		sampleSize += homeAnalysis.BackToBackRecord.Games
	}
	if awayAnalysis != nil {
		sampleSize += awayAnalysis.BackToBackRecord.Games
	}
	calc.ConfidenceLevel = 1.0 - math.Exp(-float64(sampleSize)/10.0)

	return calc
}

// GetTeamAnalysis returns the rest analysis for a specific team
func (ris *RestImpactService) GetTeamAnalysis(teamCode string) *models.RestImpactAnalysis {
	ris.mu.RLock()
	defer ris.mu.RUnlock()
	return ris.teamAnalysis[teamCode]
}

// GetAllTeamSummaries returns rest impact summaries for all teams, sorted by B2B performance
func (ris *RestImpactService) GetAllTeamSummaries() []models.RestImpactSummary {
	ris.mu.RLock()
	defer ris.mu.RUnlock()

	summaries := make([]models.RestImpactSummary, 0, len(ris.teamAnalysis))

	for teamCode, analysis := range ris.teamAnalysis {
		summary := models.RestImpactSummary{
			TeamCode:          teamCode,
			B2BWinPct:         analysis.BackToBackRecord.WinPct,
			NormalWinPct:      analysis.TwoDayRestRecord.WinPct, // Use 2-day rest as "normal"
			B2BPenalty:        analysis.B2BPenalty,
			FatigueResistance: analysis.FatigueResistance,
			OptimalRestDays:   analysis.OptimalRestDays,
		}

		// Determine assessment
		if analysis.FatigueResistance > 0.75 {
			summary.Assessment = "Elite"
		} else if analysis.FatigueResistance > 0.50 {
			summary.Assessment = "Above Average"
		} else if analysis.FatigueResistance > 0.25 {
			summary.Assessment = "Below Average"
		} else {
			summary.Assessment = "Poor"
		}

		summaries = append(summaries, summary)
	}

	// Sort by B2B performance (best first)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].B2BWinPct > summaries[j].B2BWinPct
	})

	// Assign ranks
	for i := range summaries {
		summaries[i].Rank = i + 1
	}

	return summaries
}

// BackfillFromGames populates rest analysis from historical games
// Note: This is a placeholder. Full implementation requires game outcome data
func (ris *RestImpactService) BackfillFromGames(teamCode string, games []models.Game) error {
	fmt.Printf("📊 Rest analysis backfill for %s: placeholder (requires completed game data)\n", teamCode)
	// TODO: Implement backfill when we have access to completed game results
	// This would require fetching data from game summary API with rest day calculations
	return nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (ris *RestImpactService) getOrCreateAnalysis(teamCode string) *models.RestImpactAnalysis {
	analysis, exists := ris.teamAnalysis[teamCode]
	if !exists {
		analysis = &models.RestImpactAnalysis{
			TeamCode:    teamCode,
			Season:      ris.currentSeason,
			LastUpdated: time.Now(),
		}
		ris.teamAnalysis[teamCode] = analysis
	}
	return analysis
}

func (ris *RestImpactService) updateRecord(record *models.TeamRestRecord, won, otLoss bool,
	goalsFor, goalsAgainst, shots, shotsAgainst, corsiFor int) {

	record.Games++
	if won {
		record.Wins++
		record.Points += 2
	} else if otLoss {
		record.OTLosses++
		record.Points += 1
	} else {
		record.Losses++
	}

	// Update averages
	record.AvgGoalsFor = (record.AvgGoalsFor*float64(record.Games-1) + float64(goalsFor)) / float64(record.Games)
	record.AvgGoalsAgainst = (record.AvgGoalsAgainst*float64(record.Games-1) + float64(goalsAgainst)) / float64(record.Games)
	record.AvgShots = (record.AvgShots*float64(record.Games-1) + float64(shots)) / float64(record.Games)
	record.AvgShotsAgainst = (record.AvgShotsAgainst*float64(record.Games-1) + float64(shotsAgainst)) / float64(record.Games)
	record.AvgCorsiFor = (record.AvgCorsiFor*float64(record.Games-1) + float64(corsiFor)) / float64(record.Games)

	// Calculate win% and points%
	if record.Games > 0 {
		record.WinPct = float64(record.Wins) / float64(record.Games)
		maxPoints := record.Games * 2
		record.PointsPct = float64(record.Points) / float64(maxPoints)
	}
}

func (ris *RestImpactService) updateLast10B2B(analysis *models.RestImpactAnalysis, won, otLoss bool,
	goalsFor, goalsAgainst int) {
	
	// This is a simplified tracking - in a full implementation, we'd maintain a sliding window
	// For now, just track cumulative stats with exponential decay
	decayFactor := 0.9
	existing := &analysis.Last10B2BRecord

	if existing.Games < 10 {
		ris.updateRecord(existing, won, otLoss, goalsFor, goalsAgainst, 30, 30, 50)
	} else {
		// Apply decay and add new game
		existing.AvgGoalsFor = existing.AvgGoalsFor*decayFactor + float64(goalsFor)*(1-decayFactor)
		existing.AvgGoalsAgainst = existing.AvgGoalsAgainst*decayFactor + float64(goalsAgainst)*(1-decayFactor)
		if won {
			existing.Wins = int(float64(existing.Wins)*decayFactor) + 1
		}
		existing.Games = 10
		existing.WinPct = float64(existing.Wins) / 10.0
	}
}

func (ris *RestImpactService) calculateDerivedMetrics(analysis *models.RestImpactAnalysis) {
	// Calculate B2B penalty
	normalWinPct := analysis.TwoDayRestRecord.WinPct
	if normalWinPct == 0 {
		normalWinPct = 0.50 // League average assumption
	}
	analysis.B2BPenalty = analysis.BackToBackRecord.WinPct - normalWinPct

	// Calculate fatigue resistance (0-1, higher = better at handling fatigue)
	if analysis.BackToBackRecord.Games > 0 {
		// Compare B2B performance to normal
		resistance := 1.0 + (analysis.B2BPenalty * 2.0) // Scale to 0-1 range
		analysis.FatigueResistance = math.Max(0, math.Min(1.0, resistance))
	}

	// Determine optimal rest days
	records := []struct {
		days   int
		winPct float64
	}{
		{0, analysis.BackToBackRecord.WinPct},
		{1, analysis.OneDayRestRecord.WinPct},
		{2, analysis.TwoDayRestRecord.WinPct},
		{3, analysis.ThreePlusRestRecord.WinPct},
	}

	maxWinPct := 0.0
	optimalDays := 2
	for _, r := range records {
		if r.winPct > maxWinPct {
			maxWinPct = r.winPct
			optimalDays = r.days
		}
	}
	analysis.OptimalRestDays = optimalDays

	// Calculate rest sensitivity
	if len(records) > 1 {
		variance := 0.0
		meanWinPct := (records[0].winPct + records[1].winPct + records[2].winPct + records[3].winPct) / 4.0
		for _, r := range records {
			variance += math.Pow(r.winPct-meanWinPct, 2)
		}
		analysis.RestSensitivity = math.Sqrt(variance / 4.0)
	}

	// Calculate specific B2B metrics
	if analysis.BackToBackRecord.Games > 0 {
		analysis.ShotsDeclineB2B = analysis.BackToBackRecord.AvgShots - analysis.TwoDayRestRecord.AvgShots
		// Placeholder for save % (would need goalie-specific data)
		analysis.SavePctDeclineB2B = -0.010 // Assume 1% drop
	}

	// Check if improving trend
	if analysis.Last10B2BRecord.Games >= 10 {
		analysis.ImprovingTrend = analysis.Last10B2BRecord.WinPct > analysis.BackToBackRecord.WinPct
	}
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (ris *RestImpactService) getAnalysisFilePath(teamCode string) string {
	return filepath.Join(ris.dataDir, fmt.Sprintf("%s_%s.json", teamCode, ris.currentSeason))
}

func (ris *RestImpactService) saveAnalysis(teamCode string, analysis *models.RestImpactAnalysis) error {
	filePath := ris.getAnalysisFilePath(teamCode)
	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rest analysis: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write rest analysis file: %w", err)
	}
	return nil
}

func (ris *RestImpactService) loadAnalysis(teamCode string) (*models.RestImpactAnalysis, error) {
	filePath := ris.getAnalysisFilePath(teamCode)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read rest analysis file: %w", err)
	}

	var analysis models.RestImpactAnalysis
	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rest analysis: %w", err)
	}
	return &analysis, nil
}

func (ris *RestImpactService) loadAllAnalysis() error {
	files, err := os.ReadDir(ris.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read rest analysis directory: %w", err)
	}

	loaded := 0
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(ris.dataDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("⚠️ Warning: Failed to load %s: %v\n", file.Name(), err)
			continue
		}

		var analysis models.RestImpactAnalysis
		if err := json.Unmarshal(data, &analysis); err != nil {
			fmt.Printf("⚠️ Warning: Failed to unmarshal %s: %v\n", file.Name(), err)
			continue
		}

		ris.teamAnalysis[analysis.TeamCode] = &analysis
		loaded++
	}

	if loaded > 0 {
		fmt.Printf("📊 Loaded %d rest analysis records\n", loaded)
	}
	return nil
}

