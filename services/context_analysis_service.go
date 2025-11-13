package services

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

var (
	contextAnalysisInstance *ContextAnalysisService
	contextAnalysisOnce     sync.Once
)

// ContextAnalysisService analyzes game context for intelligent model selection
type ContextAnalysisService struct {
	mu                sync.RWMutex
	contextPerformance map[string]*models.ContextPerformance  // contextType -> performance
	modelPreferences  map[string]*models.ContextualModelPreference // contextType -> preferences
	dataDir           string
}

// InitializeContextAnalysis initializes the singleton ContextAnalysisService
func InitializeContextAnalysis() error {
	var err error
	contextAnalysisOnce.Do(func() {
		dataDir := filepath.Join("data", "context")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create context directory: %w", err)
			return
		}

		contextAnalysisInstance = &ContextAnalysisService{
			contextPerformance: make(map[string]*models.ContextPerformance),
			modelPreferences:   make(map[string]*models.ContextualModelPreference),
			dataDir:            dataDir,
		}

		// Load existing context data
		if loadErr := contextAnalysisInstance.loadContextData(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing context data: %v\n", loadErr)
		}

		fmt.Println("✅ Context Analysis Service initialized")
	})
	return err
}

// GetContextAnalysisService returns the singleton instance
func GetContextAnalysisService() *ContextAnalysisService {
	return contextAnalysisInstance
}

// AnalyzeGameContext determines the context of a game for model selection
func (cas *ContextAnalysisService) AnalyzeGameContext(homeTeam, awayTeam string, gameDate time.Time) *models.GameContext {
	context := &models.GameContext{
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
		GameDate: gameDate,
	}

	// Calculate season progress (assuming Oct 1 start, Apr 30 end)
	seasonStart := time.Date(gameDate.Year(), 10, 1, 0, 0, 0, 0, time.UTC)
	if gameDate.Month() < 10 {
		seasonStart = time.Date(gameDate.Year()-1, 10, 1, 0, 0, 0, 0, time.UTC)
	}
	daysSinceStart := gameDate.Sub(seasonStart).Hours() / 24
	context.SeasonProgress = math.Min(daysSinceStart/212.0, 1.0) // ~212 days in season

	// Determine context flags
	context.IsSeasonOpener = daysSinceStart < 7
	context.IsSeasonFinale = context.SeasonProgress > 0.95

	// Check if playoff game (after April 10)
	context.IsPlayoffGame = gameDate.Month() >= 4 && gameDate.Day() >= 10

	// Check if rivalry (simplified - would need rivalry data)
	context.IsRivalryGame = cas.isRivalryGame(homeTeam, awayTeam)

	// Check if division game (simplified - would need division data)
	context.IsDivisionGame = cas.isDivisionGame(homeTeam, awayTeam)

	// Calculate importance and difficulty
	cas.calculateImportance(context)
	cas.calculateDifficulty(context)
	cas.determineStrategy(context)

	return context
}

// GetContextualWeights calculates model weight adjustments based on context
func (cas *ContextAnalysisService) GetContextualWeights(context *models.GameContext) *models.ContextualWeights {
	// Base weights (from dynamic weighting service if available, otherwise defaults)
	baseWeights := map[string]float64{
		"Enhanced Statistical":   0.30,
		"Bayesian Inference":     0.12,
		"Monte Carlo Simulation": 0.09,
		"Elo Rating":             0.17,
		"Poisson Regression":     0.12,
		"Neural Network":         0.06,
		"Gradient Boosting":      0.07,
		"LSTM":                   0.07,
		"Random Forest":          0.07,
	}

	adjustments := make(map[string]float64)
	reasoning := make(map[string]string)

	// Initialize all adjustments to 1.0 (no change)
	for model := range baseWeights {
		adjustments[model] = 1.0
	}

	// Context-based adjustments
	if context.IsPlayoffGame {
		// Playoffs: Boost statistical/historical models, reduce ML
		adjustments["Enhanced Statistical"] = 1.15
		adjustments["Elo Rating"] = 1.10
		adjustments["Neural Network"] = 0.95
		reasoning["Enhanced Statistical"] = "Playoff game - historical performance emphasized"
		reasoning["Elo Rating"] = "Playoff game - rating system reliable"
	}

	if context.IsRivalryGame {
		// Rivalry: Boost H2H-aware models
		adjustments["Enhanced Statistical"] = adjustments["Enhanced Statistical"] * 1.08
		adjustments["Bayesian Inference"] = 1.05
		reasoning["Enhanced Statistical"] = "Rivalry game - historical matchup matters"
	}

	if context.IsBackToBack {
		// B2B: Boost models that consider rest
		adjustments["Gradient Boosting"] = 1.15
		adjustments["Neural Network"] = 1.10
		reasoning["Gradient Boosting"] = "B2B game - rest impact important"
		reasoning["Neural Network"] = "B2B game - fatigue features emphasized"
	}

	if context.SeasonProgress < 0.15 {
		// Early season: More conservative, less reliance on trends
		adjustments["Monte Carlo Simulation"] = 1.15
		adjustments["LSTM"] = 0.90 // Sequence models need more data
		reasoning["Monte Carlo Simulation"] = "Early season - simulation robust with limited data"
	}

	if context.SeasonProgress > 0.85 {
		// Late season: Heavy emphasis on current form
		adjustments["LSTM"] = 1.15
		adjustments["Neural Network"] = 1.10
		reasoning["LSTM"] = "Late season - recent trends highly predictive"
	}

	if context.ImpactLevel == "Critical" {
		// Critical games: Boost most reliable models
		adjustments["Enhanced Statistical"] = adjustments["Enhanced Statistical"] * 1.05
		adjustments["Elo Rating"] = adjustments["Elo Rating"] * 1.05
		reasoning["Elo Rating"] = "Critical game - using most reliable models"
	}

	// Calculate final weights
	finalWeights := make(map[string]float64)
	totalWeight := 0.0
	for model, baseWeight := range baseWeights {
		finalWeights[model] = baseWeight * adjustments[model]
		totalWeight += finalWeights[model]
	}

	// Normalize to sum to 1.0
	for model := range finalWeights {
		finalWeights[model] /= totalWeight
	}

	// Calculate confidence in these weights
	confidence := cas.calculateWeightConfidence(context)

	return &models.ContextualWeights{
		Context:      *context,
		BaseWeights:  baseWeights,
		Adjustments:  adjustments,
		FinalWeights: finalWeights,
		Reasoning:    reasoning,
		Confidence:   confidence,
	}
}

// RecordContextPerformance updates performance tracking for a context type
func (cas *ContextAnalysisService) RecordContextPerformance(contextType string, correct bool, confidence float64) error {
	cas.mu.Lock()
	defer cas.mu.Unlock()

	perf, exists := cas.contextPerformance[contextType]
	if !exists {
		perf = &models.ContextPerformance{
			ContextType: contextType,
		}
		cas.contextPerformance[contextType] = perf
	}

	perf.TotalPredictions++
	if correct {
		perf.CorrectPredictions++
	}

	// Update rolling averages
	if perf.TotalPredictions > 0 {
		perf.Accuracy = float64(perf.CorrectPredictions) / float64(perf.TotalPredictions)
	}

	// Update average confidence (exponential moving average)
	alpha := 0.1
	perf.AvgConfidence = alpha*confidence + (1-alpha)*perf.AvgConfidence
	perf.CalibrationGap = perf.AvgConfidence - perf.Accuracy
	perf.LastUpdated = time.Now()

	return cas.saveContextData()
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (cas *ContextAnalysisService) isRivalryGame(homeTeam, awayTeam string) bool {
	// Simplified rivalry detection
	rivalries := map[string][]string{
		"BOS": {"MTL", "NYR"},
		"MTL": {"BOS", "TOR"},
		"TOR": {"MTL", "OTT"},
		"NYR": {"NYI", "NJD", "PHI"},
		"NYI": {"NYR", "NJD"},
		"PHI": {"PIT", "NYR"},
		"PIT": {"PHI", "WSH"},
		"WSH": {"PIT", "CAR"},
		"CAR": {"WSH", "TBL"},
		"TBL": {"FLA", "CAR"},
		"FLA": {"TBL"},
		"DET": {"CHI", "COL"},
		"CHI": {"DET", "STL"},
		"STL": {"CHI", "MIN"},
		"COL": {"DET"},
		"EDM": {"CGY", "VAN"},
		"CGY": {"EDM", "VAN"},
		"VAN": {"EDM", "CGY"},
		"LAK": {"ANA", "SJS", "VGK"},
		"ANA": {"LAK", "SJS"},
		"SJS": {"LAK", "ANA"},
	}

	if rivals, ok := rivalries[homeTeam]; ok {
		for _, rival := range rivals {
			if rival == awayTeam {
				return true
			}
		}
	}
	return false
}

func (cas *ContextAnalysisService) isDivisionGame(homeTeam, awayTeam string) bool {
	// Simplified division detection
	divisions := map[string][]string{
		"Atlantic": {"BOS", "BUF", "DET", "FLA", "MTL", "OTT", "TBL", "TOR"},
		"Metropolitan": {"CAR", "CBJ", "NJD", "NYI", "NYR", "PHI", "PIT", "WSH"},
		"Central": {"ARI", "CHI", "COL", "DAL", "MIN", "NSH", "STL", "WPG", "UTA"},
		"Pacific": {"ANA", "CGY", "EDM", "LAK", "SJS", "SEA", "VAN", "VGK"},
	}

	for _, teams := range divisions {
		homeInDiv := false
		awayInDiv := false
		for _, team := range teams {
			if team == homeTeam {
				homeInDiv = true
			}
			if team == awayTeam {
				awayInDiv = true
			}
		}
		if homeInDiv && awayInDiv {
			return true
		}
	}
	return false
}

func (cas *ContextAnalysisService) calculateImportance(context *models.GameContext) {
	importance := 0.5 // Base importance

	if context.IsPlayoffGame {
		importance = 1.0
		context.ImpactLevel = "Critical"
	} else if context.SeasonProgress > 0.85 {
		importance += 0.3
		context.ImpactLevel = "High"
	} else if context.IsRivalryGame {
		importance += 0.2
		context.ImpactLevel = "High"
	} else if context.SeasonProgress < 0.15 {
		importance -= 0.1
		context.ImpactLevel = "Low"
	} else {
		context.ImpactLevel = "Medium"
	}

	context.ImportanceScore = math.Max(0.0, math.Min(1.0, importance))
}

func (cas *ContextAnalysisService) calculateDifficulty(context *models.GameContext) {
	difficulty := 0.5 // Base difficulty

	// Early season = harder to predict
	if context.SeasonProgress < 0.15 {
		difficulty += 0.2
	}

	// Rivalry games = harder (emotion factor)
	if context.IsRivalryGame {
		difficulty += 0.1
	}

	// B2B games = easier (clear fatigue factor)
	if context.IsBackToBack {
		difficulty -= 0.1
	}

	// Playoff games = easier (teams play to form)
	if context.IsPlayoffGame {
		difficulty -= 0.15
	}

	difficulty = math.Max(0.0, math.Min(1.0, difficulty))
	context.DifficultyScore = difficulty

	if difficulty > 0.75 {
		context.PredictionDifficulty = "Very Hard"
	} else if difficulty > 0.6 {
		context.PredictionDifficulty = "Hard"
	} else if difficulty > 0.4 {
		context.PredictionDifficulty = "Medium"
	} else {
		context.PredictionDifficulty = "Easy"
	}
}

func (cas *ContextAnalysisService) determineStrategy(context *models.GameContext) {
	if context.DifficultyScore > 0.7 {
		context.RecommendedStrategy = "Conservative"
		context.StrategyReasoning = "High prediction difficulty - use proven reliable models"
	} else if context.SeasonProgress < 0.15 {
		context.RecommendedStrategy = "Balanced"
		context.StrategyReasoning = "Early season - balance historical and current data"
	} else if context.SeasonProgress > 0.85 {
		context.RecommendedStrategy = "Aggressive"
		context.StrategyReasoning = "Late season - emphasize current form and trends"
	} else {
		context.RecommendedStrategy = "Data-Driven"
		context.StrategyReasoning = "Mid-season - full data-driven ensemble approach"
	}
}

func (cas *ContextAnalysisService) calculateWeightConfidence(context *models.GameContext) float64 {
	confidence := 0.8 // Base confidence

	// More confident in late season (more data)
	if context.SeasonProgress > 0.5 {
		confidence += (context.SeasonProgress - 0.5) * 0.3
	}

	// Less confident in very difficult predictions
	confidence -= context.DifficultyScore * 0.2

	return math.Max(0.5, math.Min(1.0, confidence))
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (cas *ContextAnalysisService) getContextDataPath() string {
	return filepath.Join(cas.dataDir, "context_performance.json")
}

func (cas *ContextAnalysisService) saveContextData() error {
	data, err := json.MarshalIndent(cas.contextPerformance, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}
	if err := os.WriteFile(cas.getContextDataPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write context data: %w", err)
	}
	return nil
}

func (cas *ContextAnalysisService) loadContextData() error {
	filePath := cas.getContextDataPath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data yet
		}
		return fmt.Errorf("failed to read context data: %w", err)
	}

	if err := json.Unmarshal(data, &cas.contextPerformance); err != nil {
		return fmt.Errorf("failed to unmarshal context data: %w", err)
	}

	fmt.Printf("🎯 Loaded context performance for %d context types\n", len(cas.contextPerformance))
	return nil
}

