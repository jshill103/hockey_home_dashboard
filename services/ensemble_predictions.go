package services

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// DEBUG mode controls verbose logging (shared with prediction_models.go)
var ensembleDEBUG = os.Getenv("DEBUG") == "true"

// EnsemblePredictionService combines multiple prediction models with cross-validation
type EnsemblePredictionService struct {
	models          []PredictionModel
	metaLearner     *MetaLearnerModel // Optional: learns optimal model combination
	useMetaLearner  bool              // Flag to enable/disable meta-learner
	teamCode        string
	gameID          int // Optional: game ID for lineup data
	accuracyTracker *AccuracyTrackingService
	dataQuality     *DataQualityService
	dynamicWeights  *DynamicWeightingService
	crossValidation *CrossValidationService
}

// NewEnsemblePredictionService creates a new ensemble service with cross-validation
func NewEnsemblePredictionService(teamCode string) *EnsemblePredictionService {
	// Initialize meta-learner
	metaLearner := NewMetaLearnerModel()

	return &EnsemblePredictionService{
		teamCode:        teamCode,
		metaLearner:     metaLearner,
		useMetaLearner:  metaLearner.trained, // Use if trained, otherwise fall back to weighted average
		accuracyTracker: NewAccuracyTrackingService(),
		dataQuality:     NewDataQualityService(teamCode),
		dynamicWeights:  NewDynamicWeightingService(),
		crossValidation: NewCrossValidationService(),
		models: []PredictionModel{
			NewStatisticalModel(),       // 30% (if meta-learner not used)
			NewBayesianModel(),          // 12%
			NewMonteCarloModel(),        // 9%
			NewEloRatingModel(),         // 17%
			NewPoissonRegressionModel(), // 12%
			NewNeuralNetworkModel(),     // 6%
			NewGradientBoostingModel(),  // 7%
			NewLSTMModel(),              // 7%
			NewRandomForestModel(),      // 7%
		},
	}
}

// SetGameID sets the game ID for this prediction (used to fetch lineup data)
func (eps *EnsemblePredictionService) SetGameID(gameID int) {
	eps.gameID = gameID
}

// PredictGame runs all models and combines their predictions with dynamic weighting
func (eps *EnsemblePredictionService) PredictGame(homeFactors, awayFactors *models.PredictionFactors) (*models.PredictionResult, error) {
	start := time.Now()
	fmt.Printf("🤖 Running ensemble prediction with %d models...\n", len(eps.models))

	// ============================================================================
	// PHASE 4: ENHANCED DATA ENRICHMENT
	// ============================================================================

	// 1. Goalie Intelligence with Pre-Game Lineup Integration
	goalieService := GetGoalieService()
	if goalieService != nil {
		// Try to get confirmed lineup data if available
		var confirmedHomeGoalie, confirmedAwayGoalie *models.LineupGoalie
		lineupService := GetPreGameLineupService()
		if lineupService != nil && eps.gameID != 0 {
			lineup, err := lineupService.GetLineup(eps.gameID)
			if err == nil && lineup != nil && lineup.IsAvailable {
				// Check if lineup is fresh (within last 12 hours)
				lineupAge := time.Since(lineup.LastUpdated)
				if lineupAge < 12*time.Hour {
					if lineup.HomeLineup != nil && lineup.HomeLineup.StartingGoalie != nil {
						confirmedHomeGoalie = lineup.HomeLineup.StartingGoalie
					}
					if lineup.AwayLineup != nil && lineup.AwayLineup.StartingGoalie != nil {
						confirmedAwayGoalie = lineup.AwayLineup.StartingGoalie
					}
				} else {
					log.Printf("📋 Lineup data is stale (%.1f hours old), using predictions", lineupAge.Hours())
				}
			}
		}

		// Get full goalie comparison (with confirmed starters if available)
		comparison, err := goalieService.GetGoalieComparisonWithConfirmed(
			homeFactors.TeamCode,
			awayFactors.TeamCode,
			time.Now(),
			confirmedHomeGoalie,
			confirmedAwayGoalie,
		)

		if err == nil && comparison != nil {
			// Populate ALL goalie features for ML models
			homeFactors.GoalieAdvantage = comparison.WinProbabilityImpact
			homeFactors.GoalieSavePctDiff = comparison.SeasonPerformance
			homeFactors.GoalieRecentFormDiff = comparison.RecentForm
			homeFactors.GoalieFatigueDiff = comparison.WorkloadFatigue

			awayFactors.GoalieAdvantage = -comparison.WinProbabilityImpact
			awayFactors.GoalieSavePctDiff = -comparison.SeasonPerformance
			awayFactors.GoalieRecentFormDiff = -comparison.RecentForm
			awayFactors.GoalieFatigueDiff = -comparison.WorkloadFatigue

			// Log goalie impact
			if comparison.WinProbabilityImpact != 0 {
				if comparison.WinProbabilityImpact > 0 {
					fmt.Printf("🥅 Goalie Impact: Home advantage (%.1f%% swing)\n", comparison.WinProbabilityImpact*100)
				} else {
					fmt.Printf("🥅 Goalie Impact: Away advantage (%.1f%% swing)\n", math.Abs(comparison.WinProbabilityImpact)*100)
				}
				fmt.Printf("   Save%%: %.3f | Form: %.2f | Fatigue: %.2f\n",
					comparison.SeasonPerformance, comparison.RecentForm, comparison.WorkloadFatigue)
			}
		} else {
			log.Printf("⚠️ Could not get goalie comparison: %v", err)
		}

		// PHASE 2: Enhanced Goalie Matchup History
		if goalieService != nil && comparison != nil {
			// Get home goalie ID
			if comparison.HomeGoalie != nil {
				matchupAdj := goalieService.GetGoalieMatchupAdjustment(comparison.HomeGoalie.PlayerID, awayFactors.TeamCode)
				homeFactors.GoalieVsTeamRating = matchupAdj
				if matchupAdj != 0 {
					fmt.Printf("🥅 Goalie Matchup History: Home goalie %.1f%% vs %s\n", matchupAdj*100, awayFactors.TeamCode)
				}
			}
			// Get away goalie ID
			if comparison.AwayGoalie != nil {
				matchupAdj := goalieService.GetGoalieMatchupAdjustment(comparison.AwayGoalie.PlayerID, homeFactors.TeamCode)
				awayFactors.GoalieVsTeamRating = matchupAdj
				if matchupAdj != 0 {
					fmt.Printf("🥅 Goalie Matchup History: Away goalie %.1f%% vs %s\n", matchupAdj*100, homeFactors.TeamCode)
				}
			}
		}
	}

	// ============================================================================
	// PHASE 2: ENHANCED DATA QUALITY ENRICHMENT
	// ============================================================================

	// 1. Head-to-Head Matchup Database
	h2hService := GetHeadToHeadService()
	if h2hService != nil {
		advantage := h2hService.CalculateAdvantage(homeFactors.TeamCode, awayFactors.TeamCode)
		if advantage != nil {
			homeFactors.HeadToHeadAdvantage = advantage.Advantage
			awayFactors.HeadToHeadAdvantage = -advantage.Advantage

			// Calculate recent H2H form (0-1 scale based on recent wins)
			record, _ := h2hService.GetMatchupAnalysis(homeFactors.TeamCode, awayFactors.TeamCode)
			if record != nil && len(record.RecentGames) > 0 {
				recentWins := 0
				for _, game := range record.RecentGames {
					if game.Winner == homeFactors.TeamCode {
						recentWins++
					}
				}
				homeFactors.H2HRecentForm = float64(recentWins) / float64(len(record.RecentGames))
				awayFactors.H2HRecentForm = 1.0 - homeFactors.H2HRecentForm
			} else {
				// No H2H data available, use neutral values
				homeFactors.H2HRecentForm = 0.5
				awayFactors.H2HRecentForm = 0.5
			}

			if advantage.Advantage != 0 {
				fmt.Printf("🏒 Head-to-Head: %s advantage (%.1f%% from %d games)\n",
					homeFactors.TeamCode, advantage.Advantage*100, advantage.SampleSize)
			}
		}
	}

	// 2. Rest & Fatigue Impact Analysis
	restService := GetRestImpactService()
	if restService != nil {
		// Get rest days from schedule context (if available)
		homeRestDays := homeFactors.RestDays
		awayRestDays := awayFactors.RestDays

		// Calculate rest advantage
		restAdv := restService.CalculateRestAdvantage(
			homeFactors.TeamCode, awayFactors.TeamCode,
			homeRestDays, awayRestDays,
			homeFactors.TravelDistance, awayFactors.TravelDistance,
		)

		if restAdv != nil {
			homeFactors.RestAdvantageDetailed = restAdv.RestAdvantage
			awayFactors.RestAdvantageDetailed = -restAdv.RestAdvantage

			// Opponent fatigue (0-1 scale, higher = more tired)
			// Initialize with defaults
			homeFactors.OpponentFatigue = 0.0 // No fatigue by default
			awayFactors.OpponentFatigue = 0.0
			
			if restAdv.AwayOnB2B {
				homeFactors.OpponentFatigue = 0.75 + (restAdv.AwayB2BPenalty * -2.0) // Convert penalty to fatigue
			}
			if restAdv.HomeOnB2B {
				awayFactors.OpponentFatigue = 0.75 + (restAdv.HomeB2BPenalty * -2.0)
			}

			if restAdv.RestAdvantage != 0 {
				fmt.Printf("😴 Rest Impact: %s (%.1f%% advantage)\n",
					restAdv.FatigueAdvantage, math.Abs(restAdv.RestAdvantage)*100)
			}
		} else {
			// Service unavailable, use defaults
			homeFactors.RestAdvantageDetailed = 0.0
			awayFactors.RestAdvantageDetailed = 0.0
			homeFactors.OpponentFatigue = 0.0
			awayFactors.OpponentFatigue = 0.0
		}
	} else {
		// Service not initialized, use defaults
		homeFactors.RestAdvantageDetailed = 0.0
		awayFactors.RestAdvantageDetailed = 0.0
		homeFactors.OpponentFatigue = 0.0
		awayFactors.OpponentFatigue = 0.0
	}

	// 3. Lineup Stability (placeholder for now)
	// TODO: Integrate with PreGameLineupService for lineup change tracking
	homeFactors.LineupStabilityFactor = 0.75 // Assume relatively stable
	awayFactors.LineupStabilityFactor = 0.75

	// 2. Betting Market Intelligence
	marketService := GetBettingMarketService()
	if marketService != nil && marketService.isEnabled {
		marketAdj, err := marketService.GetMarketAdjustment(homeFactors.TeamCode, awayFactors.TeamCode, time.Now())
		if err == nil && marketAdj != nil {
			homeFactors.MarketConsensus = marketAdj.MarketPrediction
			awayFactors.MarketConsensus = 1.0 - marketAdj.MarketPrediction
			homeFactors.MarketConfidenceVal = marketAdj.MarketEfficiency

			if marketAdj.MarketPrediction > 0 {
				fmt.Printf("💰 Market Consensus: %.1f%% home win (confidence: %.1f%%)\n",
					marketAdj.MarketPrediction*100,
					marketAdj.MarketEfficiency*100)
			}
		}
	}

	// ============================================================================
	// NEW: FEATURE INTERACTION ENGINEERING (+20 compound features)
	// ============================================================================
	fmt.Printf("🔬 Calculating feature interactions...\n")
	interactionService := NewFeatureInteractionService()
	interactionService.EnrichWithInteractions(homeFactors)
	interactionService.EnrichWithInteractions(awayFactors)
	
	// Log key interactions for debugging
	if ensembleDEBUG {
		fmt.Printf("   Home Offensive Potency: %.2f | Defensive Strength: %.2f\n", 
			homeFactors.OffensivePotency, homeFactors.DefensiveStrength)
		fmt.Printf("   Away Offensive Potency: %.2f | Defensive Strength: %.2f\n", 
			awayFactors.OffensivePotency, awayFactors.DefensiveStrength)
		fmt.Printf("   Home Fatigue Compound: %.2f | Away Fatigue Compound: %.2f\n",
			homeFactors.FatigueCompound, awayFactors.FatigueCompound)
	}

	// 3. Schedule Context Analysis
	scheduleService := GetScheduleContextService()
	if scheduleService != nil {
		scheduleComp, err := scheduleService.GetScheduleComparison(homeFactors.TeamCode, awayFactors.TeamCode, time.Now())
		if err == nil && scheduleComp != nil {
			homeCtx := scheduleComp.HomeContext
			awayCtx := scheduleComp.AwayContext

			// Apply schedule factors
			homeFactors.TravelDistance = homeCtx.TravelDistance
			awayFactors.TravelDistance = awayCtx.TravelDistance

			homeFactors.BackToBackIndicator = 0.0
			if homeCtx.IsBackToBack {
				homeFactors.BackToBackIndicator = 1.0
			}
			awayFactors.BackToBackIndicator = 0.0
			if awayCtx.IsBackToBack {
				awayFactors.BackToBackIndicator = 1.0
			}

			homeFactors.ScheduleDensity = float64(homeCtx.GamesInLast7Days)
			awayFactors.ScheduleDensity = float64(awayCtx.GamesInLast7Days)

			homeFactors.TrapGameFactor = homeCtx.TrapGameScore
			awayFactors.TrapGameFactor = awayCtx.TrapGameScore

			homeFactors.PlayoffImportance = homeCtx.PlayoffImportance
			awayFactors.PlayoffImportance = awayCtx.PlayoffImportance

			homeFactors.RestAdvantage = float64(homeCtx.RestAdvantage)
			awayFactors.RestAdvantage = float64(-homeCtx.RestAdvantage)

			if scheduleComp.TotalImpact != 0 {
				fmt.Printf("📅 Schedule Impact: %s advantage (%.1f%% swing)\n",
					scheduleComp.OverallAdvantage,
					math.Abs(scheduleComp.TotalImpact)*100)
			}
		}
	}

	// ============================================================================
	// PHASE 6: FEATURE ENGINEERING ENRICHMENT
	// ============================================================================

	// 1. Matchup Database
	matchupService := GetMatchupService()
	if matchupService != nil {
		advantage := matchupService.CalculateMatchupAdvantage(homeFactors.TeamCode, awayFactors.TeamCode)

		homeFactors.HeadToHeadAdvantage = advantage.TotalAdvantage
		awayFactors.HeadToHeadAdvantage = -advantage.TotalAdvantage

		homeFactors.RecentMatchupTrend = advantage.RecentAdvantage
		awayFactors.RecentMatchupTrend = -advantage.RecentAdvantage

		homeFactors.VenueSpecificRecord = advantage.VenueAdvantage
		awayFactors.VenueSpecificRecord = -advantage.VenueAdvantage

		homeFactors.IsRivalryGame = advantage.RivalryBoost > 0
		awayFactors.IsRivalryGame = advantage.RivalryBoost > 0

		if advantage.RivalryBoost > 0 {
			homeFactors.RivalryIntensity = advantage.RivalryBoost / 0.05 // Normalize to 0-1
			awayFactors.RivalryIntensity = advantage.RivalryBoost / 0.05
		}

		homeFactors.IsDivisionGame = advantage.DivisionGameBoost > 0
		awayFactors.IsDivisionGame = advantage.DivisionGameBoost > 0

		homeFactors.IsPlayoffRematch = advantage.PlayoffRematchBoost > 0
		awayFactors.IsPlayoffRematch = advantage.PlayoffRematchBoost > 0

		// Get matchup history for additional details
		history := matchupService.GetMatchupHistory(homeFactors.TeamCode, awayFactors.TeamCode)
		if history != nil {
			homeFactors.GamesInSeries = history.TotalGames
			awayFactors.GamesInSeries = history.TotalGames

			if !history.LastGameDate.IsZero() {
				daysSince := int(time.Since(history.LastGameDate).Hours() / 24)
				homeFactors.DaysSinceLastMeeting = daysSince
				awayFactors.DaysSinceLastMeeting = daysSince
			}

			homeFactors.AverageGoalDiff = history.AvgGoalsTeamA - history.AvgGoalsTeamB
			awayFactors.AverageGoalDiff = history.AvgGoalsTeamB - history.AvgGoalsTeamA
		}

		if advantage.TotalAdvantage != 0 {
			fmt.Printf("📊 Matchup Advantage: %s (+%.1f%% from H2H)\n",
				homeFactors.TeamCode, advantage.TotalAdvantage*100)
		}
	}

	// 2. Advanced Rolling Statistics
	rollingService := GetRollingStatsService()
	if rollingService != nil {
		homeStats, _ := rollingService.GetTeamStats(homeFactors.TeamCode)
		awayStats, _ := rollingService.GetTeamStats(awayFactors.TeamCode)

		if homeStats != nil {
			homeFactors.FormRating = homeStats.FormRating
			homeFactors.MomentumScore = homeStats.MomentumScore
			homeFactors.IsHot = homeStats.IsHot
			homeFactors.IsCold = homeStats.IsCold
			homeFactors.IsStreaking = homeStats.IsStreaking
			homeFactors.WeightedWinPct = homeStats.WeightedWinPct
			homeFactors.WeightedGoalsFor = homeStats.WeightedGoalsFor
			homeFactors.WeightedGoalsAgainst = homeStats.WeightedGoalsAgainst
			homeFactors.QualityOfWins = homeStats.QualityOfWins
			homeFactors.QualityOfLosses = homeStats.QualityOfLosses
			homeFactors.VsPlayoffTeamsPct = homeStats.VsPlayoffTeamsPct
			homeFactors.VsTop10TeamsPct = homeStats.VsTop10TeamsPct
			homeFactors.ClutchPerformance = homeStats.ClutchPerformance
			homeFactors.Last3GamesPoints = homeStats.Last3GamesPoints
			homeFactors.Last5GamesPoints = homeStats.Last5GamesPoints
			homeFactors.GoalDifferential3 = homeStats.GoalDifferential3
			homeFactors.GoalDifferential5 = homeStats.GoalDifferential5
			homeFactors.ScoringTrend = homeStats.ScoringTrend
			homeFactors.DefensiveTrend = homeStats.DefensiveTrend
			homeFactors.StrengthOfSchedule = homeStats.StrengthOfSchedule
			homeFactors.AdjustedWinPct = homeStats.AdjustedWinPct
			homeFactors.PointsTrendDirection = homeStats.PointsTrendDirection

			if homeStats.IsHot {
				fmt.Printf("🔥 %s is HOT! (Form: %.1f/10, Momentum: %.2f)\n",
					homeFactors.TeamCode, homeStats.FormRating, homeStats.MomentumScore)
			} else if homeStats.IsCold {
				fmt.Printf("🧊 %s is COLD (Form: %.1f/10, Momentum: %.2f)\n",
					homeFactors.TeamCode, homeStats.FormRating, homeStats.MomentumScore)
			}
		}

		// Away team rolling stats
		awayStats, err := rollingService.GetTeamStats(awayFactors.TeamCode)
		if err == nil && awayStats != nil {
			// Populate away factors with rolling stats
			awayFactors.FormRating = awayStats.FormRating
			awayFactors.MomentumScore = awayStats.MomentumScore
			awayFactors.IsHot = awayStats.IsHot
			awayFactors.IsCold = awayStats.IsCold
			awayFactors.IsStreaking = awayStats.IsStreaking
			awayFactors.WeightedWinPct = awayStats.WeightedWinPct
			awayFactors.WeightedGoalsFor = awayStats.WeightedGoalsFor
			awayFactors.WeightedGoalsAgainst = awayStats.WeightedGoalsAgainst
			// ... (additional fields omitted for brevity)
		}
	}

	// ============================================================================
	// PLAY-BY-PLAY ANALYTICS: EXPECTED GOALS & SHOT QUALITY
	// ============================================================================

	playByPlayService := GetPlayByPlayService()
	if playByPlayService != nil {
		// Get play-by-play stats for both teams
		homePlayStats := playByPlayService.GetTeamStats(homeFactors.TeamCode)
		awayPlayStats := playByPlayService.GetTeamStats(awayFactors.TeamCode)

		if homePlayStats != nil {
			// Populate home team xG and shot quality metrics
			homeFactors.ExpectedGoalsFor = homePlayStats.AvgExpectedGoals
			homeFactors.ExpectedGoalsAgainst = homePlayStats.AvgXGAgainst
			homeFactors.XGDifferential = homePlayStats.AvgXGDifferential
			homeFactors.XGPerShot = homePlayStats.AvgShotQuality
			homeFactors.DangerousShotsPerGame = homePlayStats.AvgDangerousShots
			homeFactors.HighDangerXG = homePlayStats.AvgDangerousShots * homePlayStats.AvgShotQuality
			homeFactors.CorsiForPct = homePlayStats.AvgCorsiForPct
			homeFactors.FenwickForPct = homePlayStats.AvgFenwickForPct
			homeFactors.FaceoffWinPct = homePlayStats.AvgFaceoffWinPct
			homeFactors.PossessionRatio = homePlayStats.AvgPossessionRatio
			homeFactors.PhysicalPlayIndex = homePlayStats.AvgHits + homePlayStats.AvgBlockedShots

			fmt.Printf("📊 %s xG Stats: %.2f xGF/game, %.2f xGA/game (Δ%.2f)\n",
				homeFactors.TeamCode,
				homePlayStats.AvgExpectedGoals,
				homePlayStats.AvgXGAgainst,
				homePlayStats.AvgXGDifferential)
		}

		if awayPlayStats != nil {
			// Populate away team xG and shot quality metrics
			awayFactors.ExpectedGoalsFor = awayPlayStats.AvgExpectedGoals
			awayFactors.ExpectedGoalsAgainst = awayPlayStats.AvgXGAgainst
			awayFactors.XGDifferential = awayPlayStats.AvgXGDifferential
			awayFactors.XGPerShot = awayPlayStats.AvgShotQuality
			awayFactors.DangerousShotsPerGame = awayPlayStats.AvgDangerousShots
			awayFactors.HighDangerXG = awayPlayStats.AvgDangerousShots * awayPlayStats.AvgShotQuality
			awayFactors.CorsiForPct = awayPlayStats.AvgCorsiForPct
			awayFactors.FenwickForPct = awayPlayStats.AvgFenwickForPct
			awayFactors.FaceoffWinPct = awayPlayStats.AvgFaceoffWinPct
			awayFactors.PossessionRatio = awayPlayStats.AvgPossessionRatio
			awayFactors.PhysicalPlayIndex = awayPlayStats.AvgHits + awayPlayStats.AvgBlockedShots

			fmt.Printf("📊 %s xG Stats: %.2f xGF/game, %.2f xGA/game (Δ%.2f)\n",
				awayFactors.TeamCode,
				awayPlayStats.AvgExpectedGoals,
				awayPlayStats.AvgXGAgainst,
				awayPlayStats.AvgXGDifferential)
		}

		// Calculate shot quality advantage
		if homePlayStats != nil && awayPlayStats != nil {
			homeFactors.ShotQualityAdvantage = homePlayStats.AvgShotQuality - awayPlayStats.AvgShotQuality
			awayFactors.ShotQualityAdvantage = awayPlayStats.AvgShotQuality - homePlayStats.AvgShotQuality

			if math.Abs(homeFactors.ShotQualityAdvantage) > 0.01 {
				if homeFactors.ShotQualityAdvantage > 0 {
					fmt.Printf("🎯 Shot Quality Edge: %s (+%.3f xG/shot advantage)\n",
						homeFactors.TeamCode, homeFactors.ShotQualityAdvantage)
				} else {
					fmt.Printf("🎯 Shot Quality Edge: %s (+%.3f xG/shot advantage)\n",
						awayFactors.TeamCode, math.Abs(homeFactors.ShotQualityAdvantage))
				}
			}
		}
	} else {
		log.Println("⚠️ Play-by-Play service not available, xG metrics will be zero")
	}

	// ============================================================================
	// SHIFT ANALYSIS: LINE CHEMISTRY & COACHING TENDENCIES
	// ============================================================================

	// Helper function to convert bool to float64
	boolToFloat := func(b bool) float64 {
		if b {
			return 1.0
		}
		return 0.0
	}

	shiftService := GetShiftAnalysisService()
	if shiftService != nil {
		// Get shift history for both teams
		homeShiftHistory := shiftService.GetTeamHistory(homeFactors.TeamCode)
		awayShiftHistory := shiftService.GetTeamHistory(awayFactors.TeamCode)

		if homeShiftHistory != nil {
			// Populate home team shift metrics
			homeFactors.AvgShiftLength = homeShiftHistory.AvgShiftLength
			homeFactors.LineConsistency = homeShiftHistory.LineConsistency
			homeFactors.TopLineMinutes = homeShiftHistory.TopLineMinutes
			homeFactors.PlayersUsed = homeShiftHistory.AvgPlayersUsed
			homeFactors.ShortBench = boolToFloat(homeShiftHistory.ShortBench)
			homeFactors.BalancedLines = boolToFloat(homeShiftHistory.BalancedLines)
			homeFactors.FatigueIndicator = homeShiftHistory.AvgLongShifts / 20.0 // Normalize
			homeFactors.RollerCoaster = boolToFloat(homeShiftHistory.RollerCoaster)

			fmt.Printf("⏱️ %s Shift Stats: %.1fs avg shift, %.1f players used, %.1f%% top line TOI\n",
				homeFactors.TeamCode,
				homeShiftHistory.AvgShiftLength,
				homeShiftHistory.AvgPlayersUsed,
				homeShiftHistory.TopLineMinutes)
		}

		if awayShiftHistory != nil {
			// Populate away team shift metrics
			awayFactors.AvgShiftLength = awayShiftHistory.AvgShiftLength
			awayFactors.LineConsistency = awayShiftHistory.LineConsistency
			awayFactors.TopLineMinutes = awayShiftHistory.TopLineMinutes
			awayFactors.PlayersUsed = awayShiftHistory.AvgPlayersUsed
			awayFactors.ShortBench = boolToFloat(awayShiftHistory.ShortBench)
			awayFactors.BalancedLines = boolToFloat(awayShiftHistory.BalancedLines)
			awayFactors.FatigueIndicator = awayShiftHistory.AvgLongShifts / 20.0 // Normalize
			awayFactors.RollerCoaster = boolToFloat(awayShiftHistory.RollerCoaster)

			fmt.Printf("⏱️ %s Shift Stats: %.1fs avg shift, %.1f players used, %.1f%% top line TOI\n",
				awayFactors.TeamCode,
				awayShiftHistory.AvgShiftLength,
				awayShiftHistory.AvgPlayersUsed,
				awayShiftHistory.TopLineMinutes)
		}

		// Calculate coaching matchup advantages
		if homeShiftHistory != nil && awayShiftHistory != nil {
			// Short bench vs balanced lines
			if homeShiftHistory.ShortBench && !awayShiftHistory.ShortBench {
				fmt.Printf("🎯 Coaching Edge: %s (short bench) vs %s (balanced) - %s may have fresher legs late\n",
					homeFactors.TeamCode, awayFactors.TeamCode, awayFactors.TeamCode)
			} else if !homeShiftHistory.ShortBench && awayShiftHistory.ShortBench {
				fmt.Printf("🎯 Coaching Edge: %s (balanced) vs %s (short bench) - %s may have fresher legs late\n",
					homeFactors.TeamCode, awayFactors.TeamCode, homeFactors.TeamCode)
			}
		}
	} else {
		log.Println("⚠️ Shift Analysis service not available, shift metrics will be zero")
	}

	// ============================================================================
	// LANDING PAGE ANALYTICS: ENHANCED PHYSICAL PLAY & ZONE CONTROL
	// ============================================================================

	landingService := GetLandingPageService()
	if landingService != nil {
		// Get landing page history for both teams
		homeLandingHistory := landingService.GetTeamHistory(homeFactors.TeamCode)
		awayLandingHistory := landingService.GetTeamHistory(awayFactors.TeamCode)

		if homeLandingHistory != nil {
			// Populate home team landing page metrics
			homeFactors.PhysicalPlayIndex = homeLandingHistory.AvgPhysicalPlay
			homeFactors.PossessionRatio = homeLandingHistory.AvgPossessionRatio
			homeFactors.TimeOnAttack = homeLandingHistory.AvgTimeOnAttack
			homeFactors.ZoneControlRatio = homeLandingHistory.AvgZoneControl
			homeFactors.TransitionEfficiency = homeLandingHistory.AvgTransitionEff
			homeFactors.SpecialTeamsIndex = (homeLandingHistory.AvgPowerPlayPct + homeLandingHistory.AvgPenaltyKillPct) / 200.0 // Normalize

			fmt.Printf("📊 %s Landing Stats: %.1f physical play, %.1f%% possession, %.1f min attack\n",
				homeFactors.TeamCode,
				homeLandingHistory.AvgPhysicalPlay,
				homeLandingHistory.AvgPossessionRatio*100,
				homeLandingHistory.AvgTimeOnAttack)
		}

		if awayLandingHistory != nil {
			// Populate away team landing page metrics
			awayFactors.PhysicalPlayIndex = awayLandingHistory.AvgPhysicalPlay
			awayFactors.PossessionRatio = awayLandingHistory.AvgPossessionRatio
			awayFactors.TimeOnAttack = awayLandingHistory.AvgTimeOnAttack
			awayFactors.ZoneControlRatio = awayLandingHistory.AvgZoneControl
			awayFactors.TransitionEfficiency = awayLandingHistory.AvgTransitionEff
			awayFactors.SpecialTeamsIndex = (awayLandingHistory.AvgPowerPlayPct + awayLandingHistory.AvgPenaltyKillPct) / 200.0 // Normalize

			fmt.Printf("📊 %s Landing Stats: %.1f physical play, %.1f%% possession, %.1f min attack\n",
				awayFactors.TeamCode,
				awayLandingHistory.AvgPhysicalPlay,
				awayLandingHistory.AvgPossessionRatio*100,
				awayLandingHistory.AvgTimeOnAttack)
		}

		// Calculate team identity advantages
		if homeLandingHistory != nil && awayLandingHistory != nil {
			// Physical vs possession matchup
			if homeLandingHistory.PhysicalTeam && awayLandingHistory.PossessionTeam {
				fmt.Printf("🎯 Style Clash: %s (physical) vs %s (possession) - contrasting styles\n",
					homeFactors.TeamCode, awayFactors.TeamCode)
			} else if awayLandingHistory.PhysicalTeam && homeLandingHistory.PossessionTeam {
				fmt.Printf("🎯 Style Clash: %s (possession) vs %s (physical) - contrasting styles\n",
					homeFactors.TeamCode, awayFactors.TeamCode)
			}

			// Special teams advantage
			homeSpecialTeams := homeLandingHistory.AvgPowerPlayPct + homeLandingHistory.AvgPenaltyKillPct
			awaySpecialTeams := awayLandingHistory.AvgPowerPlayPct + awayLandingHistory.AvgPenaltyKillPct
			if math.Abs(homeSpecialTeams-awaySpecialTeams) > 10.0 {
				if homeSpecialTeams > awaySpecialTeams {
					fmt.Printf("🎯 Special Teams Edge: %s (+%.1f PP+PK advantage)\n",
						homeFactors.TeamCode, homeSpecialTeams-awaySpecialTeams)
				} else {
					fmt.Printf("🎯 Special Teams Edge: %s (+%.1f PP+PK advantage)\n",
						awayFactors.TeamCode, awaySpecialTeams-homeSpecialTeams)
				}
			}
		}
	} else {
		log.Println("⚠️ Landing Page service not available, enhanced metrics will be zero")
	}

	// ============================================================================
	// GAME SUMMARY ANALYTICS: ENHANCED GAME CONTEXT
	// ============================================================================

	summaryService := GetGameSummaryService()
	if summaryService != nil {
		// Get game summary history for both teams
		homeSummaryHistory := summaryService.GetTeamHistory(homeFactors.TeamCode)
		awaySummaryHistory := summaryService.GetTeamHistory(awayFactors.TeamCode)

		if homeSummaryHistory != nil {
			// Populate home team game summary metrics
			homeFactors.ShotQualityIndex = homeSummaryHistory.AvgShotQuality
			homeFactors.DisciplineIndex = homeSummaryHistory.AvgDiscipline
			homeFactors.PowerPlayTime = homeSummaryHistory.AvgPowerPlayPct * 5.0     // Estimate PP time from PP%
			homeFactors.PenaltyKillTime = homeSummaryHistory.AvgPenaltyKillPct * 5.0 // Estimate PK time from PK%
			homeFactors.OffensiveZoneTime = homeSummaryHistory.AvgZoneControl * 20.0 // Estimate zone time
			homeFactors.DefensiveZoneTime = (1.0 - homeSummaryHistory.AvgZoneControl) * 20.0

			fmt.Printf("📋 %s Summary Stats: %.2f shot quality, %.1f discipline, %.1f min PP/PK\n",
				homeFactors.TeamCode,
				homeSummaryHistory.AvgShotQuality,
				homeSummaryHistory.AvgDiscipline,
				homeSummaryHistory.AvgPowerPlayPct*5.0)
		}

		if awaySummaryHistory != nil {
			// Populate away team game summary metrics
			awayFactors.ShotQualityIndex = awaySummaryHistory.AvgShotQuality
			awayFactors.DisciplineIndex = awaySummaryHistory.AvgDiscipline
			awayFactors.PowerPlayTime = awaySummaryHistory.AvgPowerPlayPct * 5.0     // Estimate PP time from PP%
			awayFactors.PenaltyKillTime = awaySummaryHistory.AvgPenaltyKillPct * 5.0 // Estimate PK time from PK%
			awayFactors.OffensiveZoneTime = awaySummaryHistory.AvgZoneControl * 20.0 // Estimate zone time
			awayFactors.DefensiveZoneTime = (1.0 - awaySummaryHistory.AvgZoneControl) * 20.0

			fmt.Printf("📋 %s Summary Stats: %.2f shot quality, %.1f discipline, %.1f min PP/PK\n",
				awayFactors.TeamCode,
				awaySummaryHistory.AvgShotQuality,
				awaySummaryHistory.AvgDiscipline,
				awaySummaryHistory.AvgPowerPlayPct*5.0)
		}

		// Calculate game summary advantages
		if homeSummaryHistory != nil && awaySummaryHistory != nil {
			// Shot quality advantage
			shotQualityDiff := homeSummaryHistory.AvgShotQuality - awaySummaryHistory.AvgShotQuality
			if math.Abs(shotQualityDiff) > 0.1 {
				if shotQualityDiff > 0 {
					fmt.Printf("🎯 Shot Quality Edge: %s (+%.2f quality advantage)\n",
						homeFactors.TeamCode, shotQualityDiff)
				} else {
					fmt.Printf("🎯 Shot Quality Edge: %s (+%.2f quality advantage)\n",
						awayFactors.TeamCode, math.Abs(shotQualityDiff))
				}
			}

			// Discipline advantage
			disciplineDiff := awaySummaryHistory.AvgDiscipline - homeSummaryHistory.AvgDiscipline
			if math.Abs(disciplineDiff) > 0.5 {
				if disciplineDiff > 0 {
					fmt.Printf("🎯 Discipline Edge: %s (more disciplined, %.1f vs %.1f avg penalty)\n",
						homeFactors.TeamCode, homeSummaryHistory.AvgDiscipline, awaySummaryHistory.AvgDiscipline)
				} else {
					fmt.Printf("🎯 Discipline Edge: %s (more disciplined, %.1f vs %.1f avg penalty)\n",
						awayFactors.TeamCode, awaySummaryHistory.AvgDiscipline, homeSummaryHistory.AvgDiscipline)
				}
			}
		}
	} else {
		log.Println("⚠️ Game Summary service not available, enhanced context will be zero")
	}

	// 3. Player Impact
	playerService := GetPlayerImpactService()
	if playerService != nil {
		comparison := playerService.ComparePlayerImpact(homeFactors.TeamCode, awayFactors.TeamCode)

		homeFactors.StarPowerEdge = comparison.StarPowerAdvantage / 0.10 // Normalize to -1 to +1
		awayFactors.StarPowerEdge = -comparison.StarPowerAdvantage / 0.10

		homeFactors.DepthEdge = comparison.DepthAdvantage / 0.05 // Normalize to -1 to +1
		awayFactors.DepthEdge = -comparison.DepthAdvantage / 0.05

		// Get individual team impacts
		homeImpact := playerService.GetPlayerImpact(homeFactors.TeamCode)
		awayImpact := playerService.GetPlayerImpact(awayFactors.TeamCode)

		if homeImpact != nil {
			homeFactors.StarPowerRating = homeImpact.StarPower
			homeFactors.Top3CombinedPPG = homeImpact.Top3PPG
			homeFactors.DepthScoring = homeImpact.DepthScore
			homeFactors.SecondaryPPG = homeImpact.Secondary4to10
			homeFactors.ScoringBalance = homeImpact.BalanceRating
			homeFactors.TopScorerForm = homeImpact.TopScorerForm
			homeFactors.DepthForm = homeImpact.DepthForm

			if len(homeImpact.TopScorers) > 0 {
				homeFactors.TopScorerPPG = homeImpact.TopScorers[0].PointsPerGame
			}
		}

		if awayImpact != nil {
			awayFactors.StarPowerRating = awayImpact.StarPower
			awayFactors.Top3CombinedPPG = awayImpact.Top3PPG
			awayFactors.DepthScoring = awayImpact.DepthScore
			awayFactors.SecondaryPPG = awayImpact.Secondary4to10
			awayFactors.ScoringBalance = awayImpact.BalanceRating
			awayFactors.TopScorerForm = awayImpact.TopScorerForm
			awayFactors.DepthForm = awayImpact.DepthForm

			if len(awayImpact.TopScorers) > 0 {
				awayFactors.TopScorerPPG = awayImpact.TopScorers[0].PointsPerGame
			}
		}

		if comparison.TotalPlayerImpact != 0 {
			fmt.Printf("⭐ Player Advantage: %s (+%.1f%% from talent)\n",
				homeFactors.TeamCode, comparison.TotalPlayerImpact*100)
		}
	}

	var modelResults []models.ModelResult
	var totalWeight float64

	// 🚀 Get current dynamic weights
	currentWeights := eps.dynamicWeights.GetCurrentWeights()

	// ============================================================================
	// NEW: CONTEXT-AWARE MODEL SELECTION
	// ============================================================================
	fmt.Printf("🎯 Analyzing game context for model selection...\n")
	contextService := NewContextAwareWeightingService()
	gameContext := contextService.DetectGameContext(homeFactors, awayFactors)
	
	// Apply context-aware weight adjustments
	currentWeights = contextService.AdjustWeightsForContext(homeFactors, awayFactors, gameContext)
	
	// Log context explanation if any special context detected
	contextExplanation := contextService.GetContextExplanation(gameContext)
	if contextExplanation != "" {
		fmt.Printf("📋 Context: %s\n", contextExplanation)
	}
	
	// Show weight adjustments in debug mode
	if ensembleDEBUG {
		comparison := contextService.CompareWeights(currentWeights)
		fmt.Print(comparison)
	}

	// 📊 ADDITIONAL: Apply data quality boost when we have rich player data
	hasPlayerData := (homeFactors.TopScorerForm > 0 && awayFactors.TopScorerForm > 0 &&
		homeFactors.DepthForm > 0 && awayFactors.DepthForm > 0)

	if hasPlayerData {
		// Boost ML models that use player features
		currentWeights["Neural Network"] *= 1.15       // NN uses all 75 features
		currentWeights["Enhanced Statistical"] *= 1.10 // Statistical uses player impact
		currentWeights["Gradient Boosting"] *= 1.12    // GB uses player features

		// Slightly reduce simpler models
		currentWeights["Bayesian Inference"] *= 0.95
		currentWeights["Monte Carlo Simulation"] *= 0.95

		// Normalize weights back to 1.0
		totalNorm := 0.0
		for _, w := range currentWeights {
			totalNorm += w
		}
		for name := range currentWeights {
			currentWeights[name] /= totalNorm
		}

		fmt.Printf("📊 Data Quality Boost Applied: Full player intelligence available!\n")
	}

	fmt.Printf("⚖️ Current model weights: Statistical=%.1f%%, Bayesian=%.1f%%, Monte Carlo=%.1f%%, Elo=%.1f%%, Poisson=%.1f%%, Neural Net=%.1f%%\n",
		currentWeights["Enhanced Statistical"]*100,
		currentWeights["Bayesian Inference"]*100,
		currentWeights["Monte Carlo Simulation"]*100,
		currentWeights["Elo Rating"]*100,
		currentWeights["Poisson Regression"]*100,
		currentWeights["Neural Network"]*100)

	// Run all models with dynamic weights
	for _, model := range eps.models {
		result, err := model.Predict(homeFactors, awayFactors)
		if err != nil {
			fmt.Printf("⚠️ Model %s failed: %v\n", model.GetName(), err)
			continue
		}

		// 🔄 Update model weight with current dynamic weight
		if dynamicWeight, exists := currentWeights[model.GetName()]; exists {
			result.Weight = dynamicWeight
		} else {
			// Fallback to original weight if dynamic weight not found
			result.Weight = model.GetWeight()
		}

		modelResults = append(modelResults, *result)
		totalWeight += result.Weight

		// Only log individual model results in DEBUG mode
		if ensembleDEBUG {
			fmt.Printf("📊 %s: %.1f%% confidence, %s prediction (Weight: %.1f%%)\n",
				model.GetName(), result.Confidence*100, result.PredictedScore, result.Weight*100)
		}
	}

	if len(modelResults) == 0 {
		return nil, fmt.Errorf("all prediction models failed")
	}

	// Combine predictions - use meta-learner if trained, otherwise weighted average
	var combinedResult *models.PredictionResult

	if eps.useMetaLearner && eps.metaLearner.trained {
		// Use meta-learner to optimally combine predictions
		combinedResult = eps.combineWithMetaLearner(modelResults, homeFactors, awayFactors)
		combinedResult.EnsembleMethod = "Meta-Learner (Stacking)"
		fmt.Printf("🎯 Using Meta-Learner for optimal model combination\n")
	} else {
		// Fall back to weighted average
		combinedResult = eps.combineWeightedPredictions(modelResults, totalWeight, homeFactors, awayFactors)
		combinedResult.EnsembleMethod = "Weighted Average with Dynamic Weighting"
	}

	combinedResult.ModelResults = modelResults

	fmt.Printf("🎯 Ensemble Result: %s wins with %.1f%% probability (Score: %s, Confidence: %.1f%%)\n",
		combinedResult.Winner, combinedResult.WinProbability*100,
		combinedResult.PredictedScore, combinedResult.Confidence*100)

	fmt.Printf("⏱️ Total processing time: %dms\n", time.Since(start).Milliseconds())

	return combinedResult, nil
}

// combineWeightedPredictions combines model results using weighted averaging
func (eps *EnsemblePredictionService) combineWeightedPredictions(results []models.ModelResult, totalWeight float64, homeFactors, awayFactors *models.PredictionFactors) *models.PredictionResult {
	var weightedHomeProb, weightedAwayProb float64
	var weightedConfidence float64
	var homeGoalsSum, awayGoalsSum float64
	var validScores int

	// Calculate weighted averages
	for _, result := range results {
		normalizedWeight := result.Weight / totalWeight

		// Weight probabilities by model confidence as well
		confidenceBoost := 1.0 + (result.Confidence-0.5)*0.4 // Boost high-confidence models
		adjustedWeight := normalizedWeight * confidenceBoost

		if result.WinProbability > 0.5 {
			// Model predicts home team wins
			weightedHomeProb += result.WinProbability * adjustedWeight
		} else {
			// Model predicts away team wins
			weightedAwayProb += (1.0 - result.WinProbability) * adjustedWeight
		}

		weightedConfidence += result.Confidence * normalizedWeight

		// Parse and combine scores
		homeGoals, awayGoals := eps.parseScore(result.PredictedScore)
		if homeGoals >= 0 && awayGoals >= 0 {
			homeGoalsSum += float64(homeGoals) * normalizedWeight
			awayGoalsSum += float64(awayGoals) * normalizedWeight
			validScores++
		}
	}

	// Determine final winner and probability
	var winner string
	var finalProb float64

	if weightedHomeProb > weightedAwayProb {
		winner = homeFactors.TeamCode
		finalProb = weightedHomeProb / (weightedHomeProb + weightedAwayProb)
	} else {
		winner = awayFactors.TeamCode
		finalProb = weightedAwayProb / (weightedHomeProb + weightedAwayProb)
	}

	// Create final score prediction
	predictedScore := "3-2" // Default fallback
	if validScores > 0 {
		avgHomeGoals := int(math.Round(homeGoalsSum))
		avgAwayGoals := int(math.Round(awayGoalsSum))
		predictedScore = fmt.Sprintf("%d-%d", avgHomeGoals, avgAwayGoals)
	}

	// Determine game type based on probability margin
	gameType := eps.determineGameType(finalProb, homeGoalsSum, awayGoalsSum)

	// Check if it's an upset (lower-ranked team predicted to win)
	isUpset := eps.isUpsetPrediction(homeFactors, awayFactors, winner)

	// Ensemble confidence considers model agreement, historical accuracy, and cross-validation
	ensembleConfidence := eps.calculateEnsembleConfidence(results, weightedConfidence, homeFactors, awayFactors)

	// 🚀 NEW: Apply cross-validation calibration to final confidence
	if eps.crossValidation.IsCalibrated() {
		calibratedConfidence := eps.crossValidation.GetCalibratedConfidence(ensembleConfidence, results)
		fmt.Printf("🎯 Cross-validation calibration: %.1f%% → %.1f%%\n",
			ensembleConfidence*100, calibratedConfidence*100)
		ensembleConfidence = calibratedConfidence
	}

	result := &models.PredictionResult{
		Winner:         winner,
		WinProbability: finalProb,
		PredictedScore: predictedScore,
		IsUpset:        isUpset,
		GameType:       gameType,
		Confidence:     ensembleConfidence,
		ModelResults:   results,
		EnsembleMethod: "Weighted Average with Cross-Validation",
	}

	// Record prediction for future accuracy tracking
	systemStatsServ := GetSystemStatsService()
	if systemStatsServ != nil && eps.gameID != 0 {
		// Build model predictions map from results
		modelPredictions := make(map[string]models.ModelPredictionRecord)
		for _, modelResult := range results {
			// Determine winner from win probability
			modelWinner := homeFactors.TeamCode
			if modelResult.WinProbability < 0.5 {
				modelWinner = awayFactors.TeamCode
			}
			modelPredictions[modelResult.ModelName] = models.ModelPredictionRecord{
				Winner:     modelWinner,
				Score:      modelResult.PredictedScore,
				Confidence: modelResult.Confidence,
			}
		}

		systemStatsServ.RecordPrediction(
			eps.gameID,
			time.Now(),
			homeFactors.TeamCode,
			awayFactors.TeamCode,
			winner,
			predictedScore,
			ensembleConfidence,
			modelPredictions,
		)
	}

	return result
}

// PredictGameWithRecovery wraps PredictGame with error recovery for graceful degradation
func (eps *EnsemblePredictionService) PredictGameWithRecovery(homeFactors, awayFactors *models.PredictionFactors) (*models.PredictionResult, error) {
	// Try to run the full prediction
	result, err := eps.PredictGame(homeFactors, awayFactors)
	if err == nil {
		// Success - cache the prediction
		cache := GetPredictionCache()
		if cache != nil && eps.gameID != 0 {
			// Calculate data quality score
			dataQuality := eps.calculateDataQuality(homeFactors, awayFactors)

			// Convert to GamePrediction for caching
			// Note: GamePrediction stores a PredictionResult, not individual fields
			gamePrediction := &models.GamePrediction{
				Prediction:  *result,
				Confidence:  result.Confidence,
				GeneratedAt: time.Now(),
			}

			cache.CachePrediction(
				eps.gameID,
				homeFactors.TeamCode,
				awayFactors.TeamCode,
				gamePrediction,
				dataQuality,
				false, // Not degraded
			)
		}
		return result, nil
	}

	// If prediction failed, log the error and return it
	// The calling function will handle degradation
	log.Printf("⚠️ Full prediction failed: %v", err)
	return nil, err
}

// calculateDataQuality assesses the quality of data available for prediction
func (eps *EnsemblePredictionService) calculateDataQuality(homeFactors, awayFactors *models.PredictionFactors) float64 {
	quality := 1.0

	// Reduce quality if using defaults
	if homeFactors.WinPercentage == 0.5 && awayFactors.WinPercentage == 0.5 {
		quality *= 0.8 // No real win % data
	}

	// Check player data availability
	if homeFactors.TopScorerForm == 0 && awayFactors.TopScorerForm == 0 {
		quality *= 0.85 // No player form data
	}

	// Check goalie data
	if homeFactors.GoalieAdvantage == 0 && awayFactors.GoalieAdvantage == 0 {
		quality *= 0.9 // Limited goalie intel
	}

	// Check advanced stats
	if homeFactors.AdvancedStats.OverallRating == 50.0 && awayFactors.AdvancedStats.OverallRating == 50.0 {
		quality *= 0.95 // Using default ratings
	}

	return quality
}

// RecordPredictionOutcome records the actual game outcome for dynamic weight adjustment
func (eps *EnsemblePredictionService) RecordPredictionOutcome(homeTeam, awayTeam string, gameDate time.Time, actualWinner string, actualScore string) error {
	if eps.dynamicWeights == nil || !eps.dynamicWeights.IsEnabled() {
		return nil // Dynamic weighting not enabled
	}

	// For each model, create an accuracy record
	for _, model := range eps.models {
		// This would need to be enhanced to store the original prediction
		// For now, we'll create a basic record structure
		record := AccuracyRecord{
			PredictionID: fmt.Sprintf("%s_vs_%s_%s_%s", homeTeam, awayTeam, gameDate.Format("2006-01-02"), model.GetName()),
			GameDate:     gameDate,
			HomeTeam:     homeTeam,
			AwayTeam:     awayTeam,
			ActualWinner: actualWinner,
			RecordedAt:   time.Now(),
			GameContext: GameContext{
				IsPlayoffGame:     false,     // Would need to determine this
				TeamStrengthGap:   0.1,       // Would calculate from team stats
				IsUpsetPrediction: false,     // Would determine from prediction
				IsBackToBack:       false,     // Would check B2B status
				RestDaysAdvantage:  0,         // Rest days differential
				TravelDistance:     0.0,       // Miles traveled
				AltitudeDifference: 0.0,       // Altitude differential
				GameImportance:    "medium",  // Would assess importance
				OpponentType:      "average", // Would classify opponent
			},
		}

		// Record the outcome for this model
		err := eps.dynamicWeights.RecordPredictionOutcome(model.GetName(), record)
		if err != nil {
			fmt.Printf("⚠️ Error recording outcome for %s: %v\n", model.GetName(), err)
		}
	}

	return nil
}

// RecordHistoricalPrediction records a prediction for cross-validation analysis
func (eps *EnsemblePredictionService) RecordHistoricalPrediction(homeTeam, awayTeam string, gameDate time.Time,
	prediction *models.PredictionResult, actualWinner string, actualScore string) error {

	if eps.crossValidation == nil {
		return nil // Cross-validation not enabled
	}

	// Create historical prediction record
	historicalPred := HistoricalPrediction{
		PredictionID:    fmt.Sprintf("%s_vs_%s_%s", homeTeam, awayTeam, gameDate.Format("2006-01-02")),
		GameDate:        gameDate,
		HomeTeam:        homeTeam,
		AwayTeam:        awayTeam,
		PredictedWinner: prediction.Winner,
		PredictedScore:  prediction.PredictedScore,
		WinProbability:  prediction.WinProbability,
		RawConfidence:   prediction.Confidence,
		ModelResults:    prediction.ModelResults,
		ActualWinner:    actualWinner,
		ActualScore:     actualScore,
		GameCompleted:   actualWinner != "",
		GameType:        "regular", // Could be enhanced to detect game type
		RecordedAt:      time.Now(),
	}

	// Add to cross-validation service
	eps.crossValidation.AddHistoricalPrediction(historicalPred)

	// Check if we should run cross-validation
	if eps.crossValidation.ShouldRevalidate() {
		go func() {
			log.Printf("🔄 Running cross-validation update...")
			if err := eps.crossValidation.RunCrossValidation(); err != nil {
				log.Printf("⚠️ Cross-validation failed: %v", err)
			} else {
				log.Printf("✅ Cross-validation completed successfully")
			}
		}()
	}

	return nil
}

// GetValidationSummary returns cross-validation results
func (eps *EnsemblePredictionService) GetValidationSummary() ValidationSummary {
	if eps.crossValidation == nil {
		return ValidationSummary{
			IsValidated: false,
			Message:     "Cross-validation not enabled",
		}
	}

	return eps.crossValidation.GetValidationSummary()
}

// parseScore extracts home and away goals from score string
func (eps *EnsemblePredictionService) parseScore(scoreStr string) (int, int) {
	parts := strings.Split(scoreStr, "-")
	if len(parts) != 2 {
		return -1, -1 // Invalid score format
	}

	homeGoals, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	awayGoals, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)

	if err1 != nil || err2 != nil {
		return -1, -1
	}

	return int(homeGoals), int(awayGoals)
}

// determineGameType categorizes the predicted game outcome
func (eps *EnsemblePredictionService) determineGameType(probability, homeGoals, awayGoals float64) string {
	scoreDiff := math.Abs(homeGoals - awayGoals)
	probMargin := math.Abs(probability - 0.5)

	if probMargin < 0.1 || scoreDiff <= 1 {
		return "toss-up"
	} else if probMargin > 0.25 || scoreDiff > 2.5 {
		return "blowout"
	}
	return "close"
}

// isUpsetPrediction determines if prediction is an upset
func (eps *EnsemblePredictionService) isUpsetPrediction(homeFactors, awayFactors *models.PredictionFactors, winner string) bool {
	// Simple upset detection based on win percentage
	if winner == homeFactors.TeamCode {
		return homeFactors.WinPercentage < awayFactors.WinPercentage-0.1
	}
	return awayFactors.WinPercentage < homeFactors.WinPercentage-0.1
}

// calculateEnsembleConfidence determines overall confidence with accuracy tracking boost
func (eps *EnsemblePredictionService) calculateEnsembleConfidence(results []models.ModelResult, avgConfidence float64, homeFactors, awayFactors *models.PredictionFactors) float64 {
	if len(results) <= 1 {
		return avgConfidence
	}

	// Calculate agreement between models
	var probabilities []float64
	for _, result := range results {
		probabilities = append(probabilities, result.WinProbability)
	}

	// Sort to find median and range
	sort.Float64s(probabilities)

	// Agreement bonus based on how close models are
	span := probabilities[len(probabilities)-1] - probabilities[0]
	agreementBonus := math.Max(0, (0.3-span)/0.3*0.2) // Up to 20% bonus for agreement

	// Processing time penalty (longer = less confident in timeliness)
	var avgProcessingTime float64
	for _, result := range results {
		avgProcessingTime += float64(result.ProcessingTime)
	}
	avgProcessingTime /= float64(len(results))

	timeConfidence := math.Max(0.8, 1.0-avgProcessingTime/5000.0) // Penalty after 5 seconds

	// 🚀 NEW: Historical Accuracy Boost
	var accuracyBoost float64 = 1.0
	if eps.accuracyTracker != nil {
		// Get confidence boost for the winning team's factors
		var winningFactors *models.PredictionFactors
		if homeFactors.WinPercentage > awayFactors.WinPercentage {
			winningFactors = homeFactors
		} else {
			winningFactors = awayFactors
		}

		// Calculate weighted accuracy boost from all models
		totalBoost := 0.0
		totalWeight := 0.0

		for _, result := range results {
			boostFactors := eps.accuracyTracker.GetConfidenceBoost(result.ModelName, winningFactors)
			totalBoost += boostFactors.OverallConfidenceBoost * result.Weight
			totalWeight += result.Weight
		}

		if totalWeight > 0 {
			accuracyBoost = totalBoost / totalWeight
		}

		fmt.Printf("🚀 Historical accuracy boost: %.2fx\n", accuracyBoost)
	}

	// 🚀 NEW: Data Quality Boost
	var dataQualityBoost float64 = 1.0
	if eps.dataQuality != nil {
		// Assess data quality for both teams
		homeQuality := eps.dataQuality.AssessDataQuality(homeFactors)
		awayQuality := eps.dataQuality.AssessDataQuality(awayFactors)

		// Use average data quality impact
		avgQualityImpact := (homeQuality.ConfidenceImpact + awayQuality.ConfidenceImpact) / 2.0
		dataQualityBoost = avgQualityImpact

		fmt.Printf("📊 Data quality boost: %.2fx (Home: %.1f/100, Away: %.1f/100)\n",
			dataQualityBoost, homeQuality.OverallScore, awayQuality.OverallScore)
	}

	// Combine all confidence factors
	baseConfidence := (avgConfidence + agreementBonus) * timeConfidence
	finalConfidence := baseConfidence * accuracyBoost * dataQualityBoost

	return math.Min(1.0, math.Max(0.1, finalConfidence))
}

// GetModelNames returns names of all models in the ensemble
func (eps *EnsemblePredictionService) GetModelNames() []string {
	var names []string
	for _, model := range eps.models {
		names = append(names, model.GetName())
	}
	return names
}

// GetModelWeights returns weights of all models in the ensemble
func (eps *EnsemblePredictionService) GetModelWeights() map[string]float64 {
	weights := make(map[string]float64)
	for _, model := range eps.models {
		weights[model.GetName()] = model.GetWeight()
	}
	return weights
}

// GetAccuracyTracker returns the accuracy tracking service
func (eps *EnsemblePredictionService) GetAccuracyTracker() *AccuracyTrackingService {
	return eps.accuracyTracker
}

// combineWithMetaLearner uses the meta-learner to optimally combine model predictions
func (eps *EnsemblePredictionService) combineWithMetaLearner(results []models.ModelResult, homeFactors, awayFactors *models.PredictionFactors) *models.PredictionResult {
	// Extract predictions from each model
	predictions := &ModelPredictions{}

	for _, result := range results {
		switch result.ModelName {
		case "Enhanced Statistical":
			predictions.Statistical = result.WinProbability
		case "Bayesian Inference":
			predictions.Bayesian = result.WinProbability
		case "Monte Carlo Simulation":
			predictions.MonteCarlo = result.WinProbability
		case "Elo Rating":
			predictions.Elo = result.WinProbability
		case "Poisson Regression":
			predictions.Poisson = result.WinProbability
		case "Neural Network":
			predictions.NeuralNetwork = result.WinProbability
		case "Gradient Boosting":
			predictions.GradientBoosting = result.WinProbability
		case "LSTM":
			predictions.LSTM = result.WinProbability
		case "Random Forest":
			predictions.RandomForest = result.WinProbability
		}
	}

	// Build game context
	context := &MetaGameContext{
		IsDivisionalGame: false, // TODO: Determine from team codes
		IsPlayoffGame:    false, // TODO: Determine from game type
		IsRivalryGame:    false, // TODO: Determine from matchup
		HomeTeamHot:      homeFactors.IsHot,
		AwayTeamHot:      awayFactors.IsHot,
		HomeTeamCold:     homeFactors.IsCold,
		AwayTeamCold:     awayFactors.IsCold,
		RestAdvantage:    float64(homeFactors.RestDays - awayFactors.RestDays),
		TravelDistance:   awayFactors.TravelFatigue.MilesTraveled,
		BackToBack:       homeFactors.BackToBackPenalty > 0 || awayFactors.BackToBackPenalty > 0,
	}

	// Get meta-learner prediction
	winProb := eps.metaLearner.PredictFromModels(predictions, context)

	// Calculate confidence (based on model agreement)
	var sumSquaredDiff float64
	modelCount := 0
	for _, result := range results {
		diff := result.WinProbability - winProb
		sumSquaredDiff += diff * diff
		modelCount++
	}
	variance := sumSquaredDiff / float64(modelCount)
	confidence := 1.0 - math.Min(variance*2, 0.6) // Lower variance = higher confidence
	confidence = math.Max(0.40, math.Min(0.95, confidence))

	// Predict score (use weighted average of model scores)
	var homeGoalsSum, awayGoalsSum float64
	validScores := 0

	for _, result := range results {
		if result.PredictedScore != "" {
			parts := strings.Split(result.PredictedScore, "-")
			if len(parts) == 2 {
				if homeGoals, err := strconv.Atoi(parts[0]); err == nil {
					if awayGoals, err := strconv.Atoi(parts[1]); err == nil {
						homeGoalsSum += float64(homeGoals)
						awayGoalsSum += float64(awayGoals)
						validScores++
					}
				}
			}
		}
	}

	predictedScore := "3-2"
	if validScores > 0 {
		homeScore := int(math.Round(homeGoalsSum / float64(validScores)))
		awayScore := int(math.Round(awayGoalsSum / float64(validScores)))

		// Adjust based on win probability
		if winProb > 0.6 && homeScore <= awayScore {
			homeScore = awayScore + 1
		} else if winProb < 0.4 && awayScore <= homeScore {
			awayScore = homeScore + 1
		}

		predictedScore = fmt.Sprintf("%d-%d", homeScore, awayScore)
	}

	// Determine winner
	winner := homeFactors.TeamCode
	if winProb < 0.5 {
		winner = awayFactors.TeamCode
		winProb = 1.0 - winProb
	}

	return &models.PredictionResult{
		Winner:         winner,
		WinProbability: winProb,
		Confidence:     confidence,
		PredictedScore: predictedScore,
	}
}
