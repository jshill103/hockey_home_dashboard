package services

import (
	"fmt"
	"log"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ContextAwareWeightingService dynamically adjusts model weights based on game context
// Different models excel in different situations - use the right tool for the job
type ContextAwareWeightingService struct {
	baseWeights map[string]float64 // Default model weights
}

// NewContextAwareWeightingService creates a new context-aware weighting service
func NewContextAwareWeightingService() *ContextAwareWeightingService {
	// Base weights - these are the default ensemble weights
	baseWeights := map[string]float64{
		"Enhanced Statistical":   0.30,
		"Bayesian":               0.12,
		"Monte Carlo":            0.09,
		"Elo Rating":             0.17,
		"Poisson Regression":     0.12,
		"Neural Network":         0.06,
		"Gradient Boosting":      0.07,
		"LSTM":                   0.07,
		"Random Forest":          0.07,
	}

	return &ContextAwareWeightingService{
		baseWeights: baseWeights,
	}
}

// PredictionGameContext represents the situational context of a game
type PredictionGameContext struct{
	IsPlayoffs       bool
	IsPlayoffPush    bool // Fighting for playoff spot
	IsRivalryGame    bool
	IsDivisionGame   bool
	IsBackToBack     bool
	IsHighStakes     bool // Playoff implications
	IsUnderdogGame   bool // Big talent disparity
	IsCloseMatchup   bool // Evenly matched teams
	IsTrapGame       bool // Letdown spot
	HasInjuries      bool // Key players out
	HasTravelFatigue bool
	IsLateSeason     bool // March/April
	IsEarlySeason    bool // October/November
}

// AdjustWeightsForContext dynamically adjusts model weights based on game situation
// Returns adjusted weights that sum to 1.0
func (caw *ContextAwareWeightingService) AdjustWeightsForContext(
	homeFactors, awayFactors *models.PredictionFactors,
	context *PredictionGameContext,
) map[string]float64 {
	// Start with base weights
	weights := make(map[string]float64)
	for model, weight := range caw.baseWeights {
		weights[model] = weight
	}

	// Apply context-specific adjustments
	// Each adjustment is multiplicative on the base weight

	// ============================================================================
	// PLAYOFF CONTEXT
	// ============================================================================
	if context.IsPlayoffs {
		// In playoffs, historical stats and clutch performance matter more
		weights["Enhanced Statistical"] *= 1.30  // History repeats in playoffs
		weights["Bayesian"] *= 1.20              // Prior probabilities more stable
		weights["Elo Rating"] *= 1.25            // Quality matters more
		weights["Monte Carlo"] *= 0.70           // Less randomness in playoffs
		weights["Neural Network"] *= 1.10        // Pattern recognition for pressure
		
		log.Printf("📊 Context: Playoff game - boosting statistical models (+30%%), reducing randomness")
	}

	// ============================================================================
	// PLAYOFF PUSH (Late season, fighting for spot)
	// ============================================================================
	if context.IsPlayoffPush {
		// Motivation and recent form matter most when teams are desperate
		weights["Enhanced Statistical"] *= 1.20  // Recent stats matter
		weights["LSTM"] *= 1.30                  // Temporal patterns (momentum)
		weights["Neural Network"] *= 1.15        // Captures urgency signals
		
		log.Printf("📊 Context: Playoff push - boosting momentum models (+30%% LSTM)")
	}

	// ============================================================================
	// RIVALRY GAMES
	// ============================================================================
	if context.IsRivalryGame {
		// Rivalry games are unpredictable - historical matchups matter
		weights["Enhanced Statistical"] *= 1.40  // H2H history very important
		weights["Bayesian"] *= 0.80              // Traditional stats less reliable
		weights["Monte Carlo"] *= 1.30           // More variance in outcomes
		weights["Elo Rating"] *= 0.90            // Rankings less predictive
		
		log.Printf("📊 Context: Rivalry game - boosting H2H history (+40%%)")
	}

	// ============================================================================
	// BACK-TO-BACK GAMES
	// ============================================================================
	if context.IsBackToBack {
		// Physical factors dominate - fatigue is real
		weights["Enhanced Statistical"] *= 1.35  // Fatigue stats matter
		weights["Neural Network"] *= 1.25        // Captures complex fatigue patterns
		weights["Gradient Boosting"] *= 1.20     // Non-linear fatigue effects
		weights["Random Forest"] *= 1.20         // Good at fatigue modeling
		
		log.Printf("📊 Context: Back-to-back - boosting fatigue-aware models (+35%%)")
	}

	// ============================================================================
	// HIGH STAKES GAMES
	// ============================================================================
	if context.IsHighStakes {
		// Clutch performance and experience matter
		weights["Enhanced Statistical"] *= 1.25  // Past performance in pressure
		weights["Bayesian"] *= 1.15              // More conservative predictions
		weights["Neural Network"] *= 1.20        // Pressure situation patterns
		weights["LSTM"] *= 0.85                  // Momentum less important
		
		log.Printf("📊 Context: High stakes - boosting clutch-aware models (+25%%)")
	}

	// ============================================================================
	// UNDERDOG SCENARIOS (Upset potential)
	// ============================================================================
	if context.IsUnderdogGame {
		// Look for upset signals - ML models better at finding non-obvious patterns
		weights["Neural Network"] *= 1.40        // Best at finding upset signals
		weights["Gradient Boosting"] *= 1.35     // Good at complex interactions
		weights["Random Forest"] *= 1.30         // Ensemble wisdom
		weights["Enhanced Statistical"] *= 0.75  // Traditional stats misleading
		weights["Elo Rating"] *= 0.70            // Rankings favor favorites
		
		log.Printf("📊 Context: Underdog scenario - boosting ML models for upset detection (+40%% NN)")
	}

	// ============================================================================
	// CLOSE MATCHUPS
	// ============================================================================
	if context.IsCloseMatchup {
		// When teams are even, small edges matter - use most sophisticated models
		weights["Neural Network"] *= 1.30        // Best at subtle patterns
		weights["Gradient Boosting"] *= 1.25     // Complex non-linear effects
		weights["LSTM"] *= 1.20                  // Recent momentum matters
		weights["Meta Learner"] *= 1.35          // Ensemble of ensembles
		
		log.Printf("📊 Context: Close matchup - boosting sophisticated ML (+30%% NN)")
	}

	// ============================================================================
	// TRAP GAMES (Letdown spots)
	// ============================================================================
	if context.IsTrapGame {
		// Psychological factors matter - historical patterns
		weights["Enhanced Statistical"] *= 1.30  // Trap game history
		weights["LSTM"] *= 0.75                  // Momentum misleading
		weights["Bayesian"] *= 1.25              // Contrarian predictions
		weights["Neural Network"] *= 1.20        // Pattern recognition
		
		log.Printf("📊 Context: Trap game - adjusting for letdown spot")
	}

	// ============================================================================
	// INJURY CONTEXT
	// ============================================================================
	if context.HasInjuries {
		// When key players out, depth and system matter more than star power
		weights["Enhanced Statistical"] *= 1.25  // Depth stats matter
		weights["Neural Network"] *= 1.30        // Captures lineup changes
		weights["Gradient Boosting"] *= 1.25     // Non-linear roster effects
		weights["Elo Rating"] *= 0.80            // Team ratings less accurate
		
		log.Printf("📊 Context: Key injuries - adjusting for roster changes")
	}

	// ============================================================================
	// TRAVEL FATIGUE
	// ============================================================================
	if context.HasTravelFatigue {
		// Physical wear matters - similar to B2B but different pattern
		weights["Enhanced Statistical"] *= 1.30  // Travel stats
		weights["Neural Network"] *= 1.25        // Fatigue patterns
		weights["Random Forest"] *= 1.20         // Good at travel modeling
		
		log.Printf("📊 Context: Travel fatigue detected")
	}

	// ============================================================================
	// SEASON PHASE ADJUSTMENTS
	// ============================================================================
	if context.IsEarlySeason {
		// Early season: less data, more variance
		weights["Bayesian"] *= 1.30              // Priors more important
		weights["Elo Rating"] *= 1.25            // Preseason ratings matter
		weights["Enhanced Statistical"] *= 0.80  // Less season data
		weights["LSTM"] *= 0.70                  // Not enough temporal data
		
		log.Printf("📊 Context: Early season - relying on priors")
	}

	if context.IsLateSeason {
		// Late season: lots of data, patterns clear
		weights["Enhanced Statistical"] *= 1.25  // Full season of data
		weights["LSTM"] *= 1.30                  // Strong temporal patterns
		weights["Neural Network"] *= 1.20        // Learned patterns clear
		weights["Gradient Boosting"] *= 1.15     // Lots of training data
		
		log.Printf("📊 Context: Late season - strong data patterns")
	}

	// ============================================================================
	// NORMALIZE WEIGHTS (must sum to 1.0)
	// ============================================================================
	totalWeight := 0.0
	for _, weight := range weights {
		totalWeight += weight
	}

	if totalWeight > 0 {
		for model := range weights {
			weights[model] /= totalWeight
		}
	}

	return weights
}

// DetectGameContext analyzes prediction factors to determine game context
func (caw *ContextAwareWeightingService) DetectGameContext(
	homeFactors, awayFactors *models.PredictionFactors,
) *PredictionGameContext {
	context := &PredictionGameContext{}

	// Detect playoff game
	context.IsPlayoffs = homeFactors.PlayoffImportance > 0.8

	// Detect playoff push (important game, late season)
	context.IsPlayoffPush = homeFactors.PlayoffImportance > 0.6 && !context.IsPlayoffs

	// Detect rivalry
	context.IsRivalryGame = homeFactors.IsRivalryGame

	// Detect division game
	context.IsDivisionGame = homeFactors.IsDivisionGame

	// Detect back-to-back
	context.IsBackToBack = homeFactors.BackToBackIndicator > 0.5 || awayFactors.BackToBackIndicator > 0.5

	// Detect high stakes
	context.IsHighStakes = homeFactors.PlayoffImportance > 0.7 || context.IsPlayoffs

	// Detect underdog scenario (large talent disparity)
	talentGap := homeFactors.StarPowerRating - awayFactors.StarPowerRating
	winPctGap := homeFactors.WeightedWinPct - awayFactors.WeightedWinPct
	context.IsUnderdogGame = (talentGap > 0.3 || talentGap < -0.3) || (winPctGap > 0.25 || winPctGap < -0.25)

	// Detect close matchup
	context.IsCloseMatchup = (talentGap > -0.15 && talentGap < 0.15) && (winPctGap > -0.10 && winPctGap < 0.10)

	// Detect trap game
	// Hot team playing cold opponent (letdown spot)
	homeHot := homeFactors.IsHot && awayFactors.IsCold
	awayHot := awayFactors.IsHot && homeFactors.IsCold
	context.IsTrapGame = (homeHot || awayHot) && (homeFactors.TrapGameFactor > 0.6 || awayFactors.TrapGameFactor > 0.6)

	// Detect injury situation
	// High injury impact on either side
	context.HasInjuries = homeFactors.InjuryImpact.ImpactScore > 15 || awayFactors.InjuryImpact.ImpactScore > 15

	// Detect travel fatigue
	context.HasTravelFatigue = homeFactors.TravelFatigue.FatigueScore > 0.6 || awayFactors.TravelFatigue.FatigueScore > 0.6

	// Detect season phase (would need date info - using playoff importance as proxy for now)
	// Early season: low playoff importance, not much momentum
	// Late season: high playoff importance, strong patterns
	context.IsEarlySeason = homeFactors.PlayoffImportance < 0.2
	context.IsLateSeason = homeFactors.PlayoffImportance > 0.5

	// Log detected context
	contextFlags := []string{}
	if context.IsPlayoffs {
		contextFlags = append(contextFlags, "PLAYOFFS")
	}
	if context.IsPlayoffPush {
		contextFlags = append(contextFlags, "PLAYOFF_PUSH")
	}
	if context.IsRivalryGame {
		contextFlags = append(contextFlags, "RIVALRY")
	}
	if context.IsBackToBack {
		contextFlags = append(contextFlags, "BACK_TO_BACK")
	}
	if context.IsUnderdogGame {
		contextFlags = append(contextFlags, "UNDERDOG")
	}
	if context.IsCloseMatchup {
		contextFlags = append(contextFlags, "CLOSE_MATCHUP")
	}
	if context.IsTrapGame {
		contextFlags = append(contextFlags, "TRAP_GAME")
	}
	if context.HasInjuries {
		contextFlags = append(contextFlags, "INJURIES")
	}
	if context.HasTravelFatigue {
		contextFlags = append(contextFlags, "TRAVEL_FATIGUE")
	}

	if len(contextFlags) > 0 {
		log.Printf("🎯 Game context detected: %v", contextFlags)
	}

	return context
}

// GetContextExplanation returns human-readable explanation of weight adjustments
func (caw *ContextAwareWeightingService) GetContextExplanation(context *PredictionGameContext) string {
	explanation := ""

	if context.IsPlayoffs {
		explanation += "Playoff game: Historical stats and clutch performance matter more. "
	}
	if context.IsPlayoffPush {
		explanation += "Playoff push: Momentum and recent form are critical. "
	}
	if context.IsRivalryGame {
		explanation += "Rivalry game: Head-to-head history very important. "
	}
	if context.IsBackToBack {
		explanation += "Back-to-back: Fatigue factors dominate. "
	}
	if context.IsUnderdogGame {
		explanation += "Underdog scenario: ML models better at finding upset signals. "
	}
	if context.IsCloseMatchup {
		explanation += "Close matchup: Using most sophisticated models for subtle edges. "
	}
	if context.IsTrapGame {
		explanation += "Trap game: Adjusting for potential letdown. "
	}
	if context.HasInjuries {
		explanation += "Key injuries: Depth and system matter more. "
	}
	if context.HasTravelFatigue {
		explanation += "Travel fatigue: Physical factors important. "
	}
	if context.IsEarlySeason {
		explanation += "Early season: Relying on preseason ratings and priors. "
	}
	if context.IsLateSeason {
		explanation += "Late season: Strong data patterns established. "
	}

	if explanation == "" {
		explanation = "Standard game context: Using balanced model weights."
	}

	return explanation
}

// CompareWeights shows the difference between base and adjusted weights
func (caw *ContextAwareWeightingService) CompareWeights(adjusted map[string]float64) string {
	comparison := "\n🔧 Model Weight Adjustments:\n"
	
	for model, baseWeight := range caw.baseWeights {
		adjustedWeight := adjusted[model]
		change := ((adjustedWeight - baseWeight) / baseWeight) * 100
		
		if change > 5 {
			comparison += fmt.Sprintf("  ↑ %s: %.1f%% → %.1f%% (+%.0f%%)\n", 
				model, baseWeight*100, adjustedWeight*100, change)
		} else if change < -5 {
			comparison += fmt.Sprintf("  ↓ %s: %.1f%% → %.1f%% (%.0f%%)\n", 
				model, baseWeight*100, adjustedWeight*100, change)
		}
	}
	
	return comparison
}

