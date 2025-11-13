package services

import (
	"math"

	"github.com/jaredshillingburg/go_uhc/models"
)

// FeatureInteractionService calculates complex feature interactions
// These compound features capture non-linear relationships that improve ML accuracy
type FeatureInteractionService struct{}

// NewFeatureInteractionService creates a new feature interaction service
func NewFeatureInteractionService() *FeatureInteractionService {
	return &FeatureInteractionService{}
}

// EnrichWithInteractions calculates all interaction features for prediction factors
// This adds 20 powerful compound features that capture complex relationships
func (fis *FeatureInteractionService) EnrichWithInteractions(factors *models.PredictionFactors) {
	// ============================================================================
	// OFFENSIVE POTENCY INTERACTIONS
	// ============================================================================
	
	// OffensivePotency: Combines raw scoring with PP effectiveness
	// High scoring team with strong PP = elite offense
	factors.OffensivePotency = factors.GoalsFor * factors.PowerPlayPct

	// ScoringPressure: Expected goals weighted by shot quality
	// Team that creates high-danger chances with quality shots
	factors.ScoringPressure = factors.ExpectedGoalsFor * factors.ShotQualityIndex

	// EliteOffense: Star power combined with production
	// Elite talent that's actually producing
	factors.EliteOffense = factors.StarPowerRating * factors.TopScorerPPG

	// DepthOffense: Depth scoring combined with secondary production
	// Balanced attack throughout lineup
	factors.DepthOffense = factors.DepthScoring * factors.SecondaryPPG

	// ============================================================================
	// DEFENSIVE VULNERABILITY INTERACTIONS
	// ============================================================================
	
	// DefensiveVulnerability: Goals allowed compounded by poor PK
	// Leaky defense that also struggles shorthanded
	factors.DefensiveVulnerability = factors.GoalsAgainst * (1.0 - factors.PenaltyKillPct)

	// GoalieSupport: Goalie performance trend with defensive support
	// Strong goalie getting help from defense (or vice versa)
	factors.GoalieSupport = factors.GoalieSavePctDiff * factors.DefensiveTrend

	// DefensiveStrength: Combined defensive metrics
	// Normalize goals against to 0-1 scale (assuming 6 GA/game is max), multiply by PK
	normalizedGA := math.Max(0, 1.0 - (factors.GoalsAgainst / 6.0))
	factors.DefensiveStrength = normalizedGA * factors.PenaltyKillPct

	// ============================================================================
	// FATIGUE & TRAVEL COMPOUND EFFECTS
	// ============================================================================
	
	// FatigueCompound: Rest days offset by travel distance
	// Well-rested team that traveled far = still somewhat fatigued
	// Note: Negative travel distance means more fatigue impact
	factors.FatigueCompound = float64(factors.RestDays) - (factors.TravelDistance / 1000.0)

	// BackToBackTravel: Back-to-back games compounded by travel
	// B2B games are hard, B2B with travel is brutal
	factors.BackToBackTravel = factors.BackToBackIndicator * (factors.TravelDistance / 1000.0)

	// ScheduleStress: Dense schedule compounded by travel fatigue
	// Many recent games + travel fatigue = exhaustion
	factors.ScheduleStress = factors.ScheduleDensity * factors.TravelFatigue.FatigueScore

	// ============================================================================
	// MOMENTUM & HOME ADVANTAGE
	// ============================================================================
	
	// HomeMomentum: Recent form amplified at home
	// Hot team at home = dangerous
	factors.HomeMomentum = factors.RecentForm * factors.HomeAdvantage

	// HomeFieldStrength: Home advantage weighted by overall quality
	// Home advantage means more for good teams
	factors.HomeFieldStrength = factors.HomeAdvantage * factors.WeightedWinPct

	// RefereeHomeBias: Set to 0 (referee data removed)
	factors.RefereeHomeBias = 0.0

	// ============================================================================
	// ELITE PERFORMANCE INTERACTIONS
	// ============================================================================
	
	// ClutchElite: Star power in clutch situations
	// Elite players that perform in big moments
	factors.ClutchElite = factors.StarPowerRating * factors.ClutchPerformance

	// HotStreak: Momentum score amplified by hot streak
	// Momentum is real when team is on fire
	hotMultiplier := 0.0
	if factors.IsHot {
		hotMultiplier = 1.0
	}
	factors.HotStreak = factors.MomentumScore * hotMultiplier

	// FormQuality: Recent form weighted by opponent quality
	// Winning against good teams = real form
	factors.FormQuality = factors.RecentForm * factors.QualityOfWins

	// ============================================================================
	// SPECIAL TEAMS DIFFERENTIAL (PHASE 1.4 ENHANCED)
	// ============================================================================
	
	// SpecialTeamsDominance: PP and PK excellence combined
	// Normalized around league averages (PP ~20%, PK ~80%)
	// Positive = elite special teams, negative = poor special teams
	ppAboveAvg := factors.PowerPlayPct - 0.20
	pkAboveAvg := factors.PenaltyKillPct - 0.80
	factors.SpecialTeamsDominance = ppAboveAvg * pkAboveAvg

	// PowerPlayOpportunity: PP effectiveness
	factors.PowerPlayOpportunity = factors.PowerPlayPct

	// ============================================================================
	// SITUATIONAL CONTEXT
	// ============================================================================
	
	// RivalryIntensityFactor: Rivalry games with high intensity and form
	// Hot team in rivalry game = maximum motivation
	rivalryMultiplier := 0.0
	if factors.IsRivalryGame {
		rivalryMultiplier = factors.RivalryIntensity
	}
	factors.RivalryIntensityFactor = rivalryMultiplier * factors.RecentForm

	// PlayoffPressure: High stakes combined with clutch performance
	// Important games where clutch matters most
	factors.PlayoffPressure = factors.PlayoffImportance * factors.ClutchPerformance
}

// CalculateInteractionImportance returns feature importance scores for analysis
// This helps understand which interactions are most predictive
func (fis *FeatureInteractionService) CalculateInteractionImportance() map[string]float64 {
	// These are estimated based on hockey domain knowledge
	// Will be refined as models train and we measure actual importance
	return map[string]float64{
		"OffensivePotency":         0.85, // Very important: scoring ability
		"ScoringPressure":          0.80, // High: quality chances
		"EliteOffense":             0.75, // High: star power matters
		"DepthOffense":             0.70, // Medium-high: depth helps
		"DefensiveVulnerability":   0.85, // Very important: defensive weaknesses
		"GoalieSupport":            0.90, // Critical: goalie + defense
		"DefensiveStrength":        0.80, // High: combined defense
		"FatigueCompound":          0.75, // High: fatigue is real
		"BackToBackTravel":         0.80, // High: B2B travel kills teams
		"ScheduleStress":           0.70, // Medium-high: cumulative fatigue
		"HomeMomentum":             0.85, // Very important: hot at home
		"HomeFieldStrength":        0.75, // High: good teams at home
		"RefereeHomeBias":          0.60, // Medium: referee impact
		"ClutchElite":              0.80, // High: stars in big moments
		"HotStreak":                0.75, // High: momentum matters
		"FormQuality":              0.85, // Very important: quality wins
		"SpecialTeamsDominance":    0.90, // Critical: PP/PK combined
		"PowerPlayOpportunity":     0.70, // Medium-high: situational
		"RivalryIntensityFactor":   0.65, // Medium: rivalry boost
		"PlayoffPressure":          0.80, // High: clutch in big games
	}
}

// GetInteractionDescription returns human-readable descriptions
func (fis *FeatureInteractionService) GetInteractionDescription(featureName string) string {
	descriptions := map[string]string{
		"OffensivePotency":         "Scoring ability combined with power play effectiveness",
		"ScoringPressure":          "Expected goals weighted by shot quality",
		"EliteOffense":             "Star power combined with top scorer production",
		"DepthOffense":             "Depth scoring throughout lineup",
		"DefensiveVulnerability":   "Goals allowed compounded by weak penalty kill",
		"GoalieSupport":            "Goalie performance with defensive support",
		"DefensiveStrength":        "Combined defensive metrics and penalty kill",
		"FatigueCompound":          "Rest days offset by travel distance",
		"BackToBackTravel":         "Back-to-back games with travel fatigue",
		"ScheduleStress":           "Dense schedule with cumulative fatigue",
		"HomeMomentum":             "Recent form amplified at home ice",
		"HomeFieldStrength":        "Home advantage for quality teams",
		"RefereeHomeBias":          "Referee home bias compounded with venue advantage",
		"ClutchElite":              "Star players performing in clutch situations",
		"HotStreak":                "Momentum score during hot streaks",
		"FormQuality":              "Recent form against quality opponents",
		"SpecialTeamsDominance":    "Power play and penalty kill excellence combined",
		"PowerPlayOpportunity":     "Power play with referee that calls penalties",
		"RivalryIntensityFactor":   "Motivation boost in rivalry games",
		"PlayoffPressure":          "Clutch performance in high-stakes games",
	}
	
	if desc, exists := descriptions[featureName]; exists {
		return desc
	}
	return "Unknown interaction feature"
}

