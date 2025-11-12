package services

import (
	"testing"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ================================================================================================
// NHL TIEBREAKER TESTS
// ================================================================================================

func TestNHLTiebreakers_Points(t *testing.T) {
	teams := []models.TeamStanding{
		{TeamName: models.TeamNameInfo{Default: "Team A"}, Points: 90, GamesPlayed: 75},
		{TeamName: models.TeamNameInfo{Default: "Team B"}, Points: 95, GamesPlayed: 75},
		{TeamName: models.TeamNameInfo{Default: "Team C"}, Points: 85, GamesPlayed: 75},
	}

	SortTeamsByNHLRules(teams)

	if teams[0].TeamName.Default != "Team B" {
		t.Errorf("Expected Team B first (95 points), got %s", teams[0].TeamName.Default)
	}
	if teams[1].TeamName.Default != "Team A" {
		t.Errorf("Expected Team A second (90 points), got %s", teams[1].TeamName.Default)
	}
	if teams[2].TeamName.Default != "Team C" {
		t.Errorf("Expected Team C third (85 points), got %s", teams[2].TeamName.Default)
	}
}

func TestNHLTiebreakers_GamesPlayed(t *testing.T) {
	teams := []models.TeamStanding{
		{TeamName: models.TeamNameInfo{Default: "Team A"}, Points: 90, GamesPlayed: 75},
		{TeamName: models.TeamNameInfo{Default: "Team B"}, Points: 90, GamesPlayed: 74},
		{TeamName: models.TeamNameInfo{Default: "Team C"}, Points: 90, GamesPlayed: 76},
	}

	SortTeamsByNHLRules(teams)

	// Team B should be first (fewer games played with same points)
	if teams[0].TeamName.Default != "Team B" {
		t.Errorf("Expected Team B first (74 GP), got %s (%d GP)",
			teams[0].TeamName.Default, teams[0].GamesPlayed)
	}
	if teams[2].TeamName.Default != "Team C" {
		t.Errorf("Expected Team C last (76 GP), got %s (%d GP)",
			teams[2].TeamName.Default, teams[2].GamesPlayed)
	}
}

func TestNHLTiebreakers_ROW(t *testing.T) {
	teams := []models.TeamStanding{
		{
			TeamName:             models.TeamNameInfo{Default: "Team A"},
			Points:               90,
			GamesPlayed:          75,
			Wins:                 40,
			RegulationPlusOtWins: 35,
		},
		{
			TeamName:             models.TeamNameInfo{Default: "Team B"},
			Points:               90,
			GamesPlayed:          75,
			Wins:                 40,
			RegulationPlusOtWins: 38,
		},
	}

	SortTeamsByNHLRules(teams)

	// Team B should be first (more ROW)
	if teams[0].TeamName.Default != "Team B" {
		t.Errorf("Expected Team B first (38 ROW), got %s (%d ROW)",
			teams[0].TeamName.Default, teams[0].RegulationPlusOtWins)
	}
}

func TestNHLTiebreakers_GoalDifferential(t *testing.T) {
	teams := []models.TeamStanding{
		{
			TeamName:             models.TeamNameInfo{Default: "Team A"},
			Points:               90,
			GamesPlayed:          75,
			Wins:                 40,
			RegulationPlusOtWins: 35,
			GoalFor:              250,
			GoalAgainst:          240, // +10 goal diff
		},
		{
			TeamName:             models.TeamNameInfo{Default: "Team B"},
			Points:               90,
			GamesPlayed:          75,
			Wins:                 40,
			RegulationPlusOtWins: 35,
			GoalFor:              260,
			GoalAgainst:          245, // +15 goal diff
		},
	}

	SortTeamsByNHLRules(teams)

	// Team B should be first (better goal differential)
	if teams[0].TeamName.Default != "Team B" {
		t.Errorf("Expected Team B first (+15 goal diff), got %s",
			teams[0].TeamName.Default)
	}
}

// ================================================================================================
// GAME PREDICTOR TESTS
// ================================================================================================

func TestSimplePredictor(t *testing.T) {
	predictor := NewSimplePredictor()

	if predictor.Name() != "Simple" {
		t.Errorf("Expected name 'Simple', got %s", predictor.Name())
	}

	context := &PredictionContext{
		HomeRecord: &models.TeamStanding{PointPctg: 0.650},
		AwayRecord: &models.TeamStanding{PointPctg: 0.450},
	}

	prob, err := predictor.PredictWinProbability("HOME", "AWAY", context)
	if err != nil {
		t.Fatalf("Prediction failed: %v", err)
	}

	// Home team should have higher probability
	if prob <= 0.5 {
		t.Errorf("Expected home team advantage, got probability: %.3f", prob)
	}

	// Should be within bounds
	if prob < 0.3 || prob > 0.8 {
		t.Errorf("Probability out of expected bounds (0.3-0.8): %.3f", prob)
	}
}

func TestSimplePredictor_NilContext(t *testing.T) {
	predictor := NewSimplePredictor()

	context := &PredictionContext{
		HomeRecord: nil,
		AwayRecord: nil,
	}

	prob, err := predictor.PredictWinProbability("HOME", "AWAY", context)
	if err != nil {
		t.Fatalf("Prediction failed: %v", err)
	}

	// Should return default value with home advantage
	if prob != 0.55 {
		t.Errorf("Expected default probability 0.55, got %.3f", prob)
	}
}

func TestEloPredictor(t *testing.T) {
	predictor := NewEloPredictor()

	if predictor.Name() != "ELO" {
		t.Errorf("Expected name 'ELO', got %s", predictor.Name())
	}

	// Test ELO calculation from record
	team := &models.TeamStanding{
		PointPctg: 0.600,
	}

	elo := predictor.calculateEloFromRecord(team)
	if elo <= 1500 {
		t.Errorf("Expected ELO > 1500 for winning team (0.600 win pct), got %.1f", elo)
	}

	// Test prediction
	context := &PredictionContext{
		HomeRecord: &models.TeamStanding{PointPctg: 0.600},
		AwayRecord: &models.TeamStanding{PointPctg: 0.400},
	}

	prob, err := predictor.PredictWinProbability("HOME", "AWAY", context)
	if err != nil {
		t.Fatalf("Prediction failed: %v", err)
	}

	// Better team at home should have strong advantage
	if prob <= 0.6 {
		t.Errorf("Expected strong home advantage, got %.3f", prob)
	}

	// Should be within bounds
	if prob < 0.25 || prob > 0.85 {
		t.Errorf("Probability out of bounds: %.3f", prob)
	}
}

func TestEloPredictor_UpdateRatings(t *testing.T) {
	predictor := NewEloPredictor()

	// Set initial ratings
	predictor.SetElo("TEAM_A", 1600)
	predictor.SetElo("TEAM_B", 1400)

	initialA := predictor.GetElo("TEAM_A")
	initialB := predictor.GetElo("TEAM_B")

	// Team B (underdog) beats Team A
	predictor.UpdateElo("TEAM_B", "TEAM_A", 1)

	newA := predictor.GetElo("TEAM_A")
	newB := predictor.GetElo("TEAM_B")

	// Team A should lose more rating (lost to underdog)
	if newA >= initialA {
		t.Errorf("Team A should have lost rating, but went from %.1f to %.1f", initialA, newA)
	}

	// Team B should gain rating (upset win)
	if newB <= initialB {
		t.Errorf("Team B should have gained rating, but went from %.1f to %.1f", initialB, newB)
	}

	// Rating change for underdog win should be significant
	ratingGain := newB - initialB
	if ratingGain < 20 {
		t.Errorf("Expected significant rating gain for underdog, got %.1f", ratingGain)
	}
}

func TestHybridPredictor_ImportanceCalculation(t *testing.T) {
	ml := NewMLPredictor(nil) // Pass nil for testing
	elo := NewEloPredictor()
	hybrid := NewHybridPredictor(ml, elo)

	// Test division game importance
	context := &PredictionContext{
		IsDivisionGame: true,
		HomeRecord:     &models.TeamStanding{Points: 90},
		AwayRecord:     &models.TeamStanding{Points: 88},
	}

	importance := hybrid.calculateGameImportance(context)
	if importance <= 0.5 {
		t.Errorf("Division game should have high importance, got %.2f", importance)
	}

	// Test playoff game importance
	context.IsPlayoffs = true
	importance = hybrid.calculateGameImportance(context)
	if importance != 1.0 {
		t.Errorf("Playoff game should have maximum importance (1.0), got %.2f", importance)
	}
}

func TestHybridPredictor_MLThreshold(t *testing.T) {
	ml := NewMLPredictor(nil)
	elo := NewEloPredictor()
	hybrid := NewHybridPredictor(ml, elo)

	// Test setting threshold
	hybrid.SetMLThreshold(0.8)

	// Test boundary conditions
	hybrid.SetMLThreshold(-0.5) // Should clamp to 0
	hybrid.SetMLThreshold(1.5)  // Should clamp to 1

	// Verify stats tracking
	mlUsage, eloUsage := hybrid.GetStats()
	if mlUsage != 0 || eloUsage != 0 {
		t.Errorf("Expected zero usage initially, got ML: %d, ELO: %d", mlUsage, eloUsage)
	}

	hybrid.ResetStats()
}

// ================================================================================================
// PLAYOFF SIMULATION TESTS
// ================================================================================================

func TestSortByNHLRules_ConsistencyCheck(t *testing.T) {
	// Create test teams with various tie scenarios
	teams := []models.TeamStanding{
		{TeamName: models.TeamNameInfo{Default: "Team A"}, Points: 95, GamesPlayed: 75, Wins: 42, RegulationPlusOtWins: 38, GoalFor: 260, GoalAgainst: 240},
		{TeamName: models.TeamNameInfo{Default: "Team B"}, Points: 95, GamesPlayed: 75, Wins: 42, RegulationPlusOtWins: 40, GoalFor: 250, GoalAgainst: 235},
		{TeamName: models.TeamNameInfo{Default: "Team C"}, Points: 95, GamesPlayed: 74, Wins: 42, RegulationPlusOtWins: 38, GoalFor: 255, GoalAgainst: 238},
		{TeamName: models.TeamNameInfo{Default: "Team D"}, Points: 90, GamesPlayed: 75, Wins: 40, RegulationPlusOtWins: 35, GoalFor: 245, GoalAgainst: 250},
	}

	SortTeamsByNHLRules(teams)

	// Verify ordering
	expectedOrder := []string{"Team C", "Team B", "Team A", "Team D"}
	for i, expectedName := range expectedOrder {
		if teams[i].TeamName.Default != expectedName {
			t.Errorf("Position %d: expected %s, got %s", i+1, expectedName, teams[i].TeamName.Default)
		}
	}
}

func TestPredictionContext_BasicFields(t *testing.T) {
	// Test that PredictionContext struct is properly defined
	context := &PredictionContext{
		IsDivisionGame: true,
		IsRivalryGame:  false,
		IsPlayoffs:     false,
		RestDaysHome:   2,
		RestDaysAway:   1,
	}

	if !context.IsDivisionGame {
		t.Error("Expected IsDivisionGame to be true")
	}
	if context.RestDaysHome != 2 {
		t.Errorf("Expected RestDaysHome to be 2, got %d", context.RestDaysHome)
	}
}

// ================================================================================================
// INTEGRATION TESTS
// ================================================================================================

func TestPredictorComparison(t *testing.T) {
	// Create a scenario where we can compare predictors
	context := &PredictionContext{
		HomeRecord: &models.TeamStanding{
			PointPctg:            0.650,
			Points:               95,
			GamesPlayed:          75,
			Wins:                 42,
			RegulationPlusOtWins: 38,
		},
		AwayRecord: &models.TeamStanding{
			PointPctg:            0.450,
			Points:               70,
			GamesPlayed:          75,
			Wins:                 30,
			RegulationPlusOtWins: 28,
		},
		IsDivisionGame: true,
	}

	simple := NewSimplePredictor()
	elo := NewEloPredictor()

	simpleProb, err1 := simple.PredictWinProbability("HOME", "AWAY", context)
	eloProb, err2 := elo.PredictWinProbability("HOME", "AWAY", context)

	if err1 != nil {
		t.Fatalf("Simple predictor failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("ELO predictor failed: %v", err2)
	}

	// Both should favor home team
	if simpleProb <= 0.5 {
		t.Errorf("Simple predictor should favor home team, got %.3f", simpleProb)
	}
	if eloProb <= 0.5 {
		t.Errorf("ELO predictor should favor home team, got %.3f", eloProb)
	}

	// Predictions should be reasonably close
	diff := simpleProb - eloProb
	if diff < -0.2 || diff > 0.2 {
		t.Logf("Warning: Predictors differ significantly - Simple: %.3f, ELO: %.3f", simpleProb, eloProb)
	}
}

func TestDefaultPredictor(t *testing.T) {
	// Test with nil ensemble (should return ELO)
	predictor := GetDefaultPredictor(nil)
	if predictor.Name() != "ELO" {
		t.Errorf("Expected ELO predictor when ensemble is nil, got %s", predictor.Name())
	}

	// Can't test with actual ensemble without full initialization
	// But we can verify the function doesn't panic
}

// ================================================================================================
// BENCHMARKS
// ================================================================================================

func BenchmarkSimplePredictor(b *testing.B) {
	predictor := NewSimplePredictor()
	context := &PredictionContext{
		HomeRecord: &models.TeamStanding{PointPctg: 0.600},
		AwayRecord: &models.TeamStanding{PointPctg: 0.500},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		predictor.PredictWinProbability("HOME", "AWAY", context)
	}
}

func BenchmarkEloPredictor(b *testing.B) {
	predictor := NewEloPredictor()
	context := &PredictionContext{
		HomeRecord: &models.TeamStanding{PointPctg: 0.600},
		AwayRecord: &models.TeamStanding{PointPctg: 0.500},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		predictor.PredictWinProbability("HOME", "AWAY", context)
	}
}

func BenchmarkSortTeamsByNHLRules(b *testing.B) {
	// Create a slice of 16 teams (typical conference size)
	teams := make([]models.TeamStanding, 16)
	for i := 0; i < 16; i++ {
		teams[i] = models.TeamStanding{
			TeamName:             models.TeamNameInfo{Default: "Team"},
			Points:               90 - i,
			GamesPlayed:          75,
			Wins:                 40,
			RegulationPlusOtWins: 35,
			GoalFor:              250,
			GoalAgainst:          240,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		teamsCopy := make([]models.TeamStanding, len(teams))
		copy(teamsCopy, teams)
		SortTeamsByNHLRules(teamsCopy)
	}
}

// ================================================================================================
// BUG FIX TESTS
// ================================================================================================

// TestGetROW_WithoutAPIData verifies Bug Fix #1
func TestGetROW_WithoutAPIData(t *testing.T) {
	// Test when API provides ROW
	teamWithROW := &models.TeamStanding{
		Wins:                 40,
		RegulationPlusOtWins: 35,
	}
	row := teamWithROW.GetROW()
	if row != 35 {
		t.Errorf("Expected ROW to be 35, got %d", row)
	}

	// Test when API doesn't provide ROW (Bug #1 fix)
	teamWithoutROW := &models.TeamStanding{
		Wins:                 40,
		RegulationPlusOtWins: 0, // API didn't provide
	}
	row = teamWithoutROW.GetROW()
	// Should estimate ~85% of wins (excluding shootouts)
	if row != 34 { // 40 * 0.85 = 34
		t.Errorf("Expected estimated ROW to be 34 (85%% of 40), got %d", row)
	}

	// Edge case: zero wins
	teamZero := &models.TeamStanding{
		Wins:                 0,
		RegulationPlusOtWins: 0,
	}
	row = teamZero.GetROW()
	if row != 0 {
		t.Errorf("Expected ROW to be 0 for team with no wins, got %d", row)
	}
}

// TestSimulation_TracksRegulationWins verifies Bug Fix #2
func TestSimulation_TracksRegulationWins(t *testing.T) {
	// Create mock simulation
	ps := &PlayoffSimulationService{}

	homeTeam := &models.TeamStanding{
		TeamName:    models.TeamNameInfo{Default: "Home"},
		PointPctg:   0.600,
		GamesPlayed: 0,
		Wins:        0,
	}

	awayTeam := &models.TeamStanding{
		TeamName:    models.TeamNameInfo{Default: "Away"},
		PointPctg:   0.400,
		GamesPlayed: 0,
		Wins:        0,
	}

	teamRecords := map[string]*models.TeamStanding{
		"HOME": homeTeam,
		"AWAY": awayTeam,
	}

	game := RemainingGame{
		HomeTeam: "HOME",
		AwayTeam: "AWAY",
	}

	// Run 100 simulations
	prevGameDates := make(map[string]time.Time)
	for i := 0; i < 100; i++ {
		ps.simulateGame(game, teamRecords, prevGameDates)
	}

	// Verify regulation wins are tracked
	totalGames := homeTeam.Wins + awayTeam.Wins
	totalROW := homeTeam.RegulationPlusOtWins + awayTeam.RegulationPlusOtWins

	if totalGames == 0 {
		t.Fatal("No games were simulated")
	}

	// ROW should be non-zero
	if totalROW == 0 {
		t.Error("Bug #2 NOT fixed: RegulationPlusOtWins is still 0 after simulations")
	}

	// ROW should be less than or equal to total wins
	if totalROW > totalGames {
		t.Errorf("ROW (%d) cannot exceed total wins (%d)", totalROW, totalGames)
	}

	// ROW should be approximately 95% of wins (85% reg + 10% OT)
	expectedROW := int(float64(totalGames) * 0.95)
	if totalROW < expectedROW-10 || totalROW > expectedROW+10 {
		t.Logf("Warning: ROW (%d) is not close to expected value (%d) for %d games", totalROW, expectedROW, totalGames)
	}

	// Regulation wins should be ~85% of total wins
	totalRegWins := homeTeam.RegulationWins + awayTeam.RegulationWins
	if totalRegWins == 0 {
		t.Error("Bug #2 NOT fully fixed: RegulationWins is still 0 after simulations")
	}
}

// TestEloPredictor_NilContext verifies Bug Fix #3
func TestEloPredictor_NilContext(t *testing.T) {
	predictor := NewEloPredictor()

	// Should not panic with nil context
	prob, err := predictor.PredictWinProbability("UTA", "COL", nil)
	if err != nil {
		t.Fatalf("Prediction with nil context failed: %v", err)
	}

	// Should return reasonable default (with home advantage)
	if prob < 0.25 || prob > 0.85 {
		t.Errorf("Probability out of bounds with nil context: %.3f", prob)
	}

	// Should favor home team slightly (default ELO with home advantage)
	if prob < 0.5 {
		t.Errorf("Expected home advantage with nil context, got %.3f", prob)
	}
}

// TestSimplePredictor_NilContextBugFix verifies Bug Fix #4
func TestSimplePredictor_NilContextBugFix(t *testing.T) {
	predictor := NewSimplePredictor()

	// Should not panic with truly nil context (not just nil records)
	prob, err := predictor.PredictWinProbability("UTA", "COL", nil)
	if err != nil {
		t.Fatalf("Prediction with nil context failed: %v", err)
	}

	// Should return default value
	if prob != 0.55 {
		t.Errorf("Expected default probability 0.55 with nil context, got %.3f", prob)
	}
}

// TestCompareTeamsByNHLRules_NoDuplication verifies Bug Fix #5
func TestCompareTeamsByNHLRules_NoDuplication(t *testing.T) {
	// Create test teams
	teamA := &models.TeamStanding{
		TeamName:             models.TeamNameInfo{Default: "Team A"},
		Points:               90,
		GamesPlayed:          75,
		Wins:                 40,
		RegulationPlusOtWins: 35,
		GoalFor:              250,
		GoalAgainst:          240,
	}

	teamB := &models.TeamStanding{
		TeamName:             models.TeamNameInfo{Default: "Team B"},
		Points:               90,
		GamesPlayed:          75,
		Wins:                 40,
		RegulationPlusOtWins: 38,
		GoalFor:              245,
		GoalAgainst:          238,
	}

	// Team B should rank higher (more ROW)
	result := compareTeamsByNHLRules(teamB, teamA)
	if !result {
		t.Error("Team B should rank higher than Team A (more ROW)")
	}

	// Verify consistency with both sort functions
	teamsValue := []models.TeamStanding{*teamA, *teamB}
	SortTeamsByNHLRules(teamsValue)

	teamsPointer := []*models.TeamStanding{teamA, teamB}
	ps := &PlayoffSimulationService{}
	ps.sortByNHLRules(teamsPointer)

	// Both should produce same ordering
	if teamsValue[0].TeamName.Default != teamsPointer[0].TeamName.Default {
		t.Errorf("Sort functions produced different results: %s vs %s",
			teamsValue[0].TeamName.Default, teamsPointer[0].TeamName.Default)
	}
}

