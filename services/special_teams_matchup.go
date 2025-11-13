package services

import (
	"fmt"
	"math"

	"github.com/jaredshillingburg/go_uhc/models"
)

// SpecialTeamsMatchupAnalyzer analyzes special teams matchups between two teams
type SpecialTeamsMatchupAnalyzer struct{}

// NewSpecialTeamsMatchupAnalyzer creates a new special teams matchup analyzer
func NewSpecialTeamsMatchupAnalyzer() *SpecialTeamsMatchupAnalyzer {
	return &SpecialTeamsMatchupAnalyzer{}
}

// SpecialTeamsMatchup represents a comprehensive special teams matchup analysis
type SpecialTeamsMatchup struct {
	// Home team advantages
	HomePPvsPKAdvantage float64 `json:"homePPvsPKAdvantage"` // Home PP vs Away PK
	HomePKvsPPAdvantage float64 `json:"homePKvsPPAdvantage"` // Home PK vs Away PP
	HomeSTNetAdvantage  float64 `json:"homeSTNetAdvantage"`  // Net special teams advantage

	// Away team advantages (mirror of home)
	AwayPPvsPKAdvantage float64 `json:"awayPPvsPKAdvantage"` // Away PP vs Home PK
	AwayPKvsPPAdvantage float64 `json:"awayPKvsPPAdvantage"` // Away PK vs Home PP
	AwaySTNetAdvantage  float64 `json:"awaySTNetAdvantage"`  // Net special teams advantage

	// Matchup quality scores
	HomePPEffectiveness float64 `json:"homePPEffectiveness"` // Expected PP success rate
	AwayPPEffectiveness float64 `json:"awayPPEffectiveness"` // Expected PP success rate
	HomePKEffectiveness float64 `json:"homePKEffectiveness"` // Expected PK success rate
	AwayPKEffectiveness float64 `json:"awayPKEffectiveness"` // Expected PK success rate

	// Differential impact
	ExpectedPPGoalsDiff float64 `json:"expectedPPGoalsDiff"` // Expected PP goals differential
	SpecialTeamsImpact  float64 `json:"specialTeamsImpact"`  // Overall game impact (-1 to +1)

	// Matchup characterization
	IsPPMismatch     bool    `json:"isPPMismatch"`     // One team has major PP advantage
	IsPKMismatch     bool    `json:"isPKMismatch"`     // One team has major PK advantage
	IsSTEquilibrium  bool    `json:"isSTEquilibrium"`  // Special teams relatively even
	MismatchSeverity float64 `json:"mismatchSeverity"` // How lopsided is the matchup (0-1)
}

// AnalyzeMatchup performs comprehensive special teams matchup analysis
func (stma *SpecialTeamsMatchupAnalyzer) AnalyzeMatchup(
	homeFactors, awayFactors *models.PredictionFactors,
) *SpecialTeamsMatchup {
	matchup := &SpecialTeamsMatchup{}

	// League averages for normalization
	const leaguePP = 0.20  // 20% power play success
	const leaguePK = 0.80  // 80% penalty kill success

	// ============================================================================
	// HOME TEAM SPECIAL TEAMS ANALYSIS
	// ============================================================================

	// Home PP vs Away PK
	// Question: How well does home team's PP perform against away team's PK?
	// Formula: Home PP% relative to league avg vs Away PK% relative to league avg
	homePPStrength := homeFactors.PowerPlayPct - leaguePP
	awayPKStrength := awayFactors.PenaltyKillPct - leaguePK
	
	// Positive homePPvsPKAdvantage = Home PP is better than Away PK can handle
	matchup.HomePPvsPKAdvantage = homePPStrength - (-awayPKStrength) // Double negative: weak PK helps PP
	
	// Normalize to [-1, 1] range
	matchup.HomePPvsPKAdvantage = math.Max(-1.0, math.Min(1.0, matchup.HomePPvsPKAdvantage * 5.0))

	// Home PK vs Away PP
	// Question: How well does home team's PK perform against away team's PP?
	homePKStrength := homeFactors.PenaltyKillPct - leaguePK
	awayPPStrength := awayFactors.PowerPlayPct - leaguePP
	
	// Positive homePKvsPPAdvantage = Home PK can contain Away PP
	matchup.HomePKvsPPAdvantage = homePKStrength - awayPPStrength
	
	// Normalize to [-1, 1] range
	matchup.HomePKvsPPAdvantage = math.Max(-1.0, math.Min(1.0, matchup.HomePKvsPPAdvantage * 5.0))

	// ============================================================================
	// AWAY TEAM SPECIAL TEAMS ANALYSIS (mirror calculation)
	// ============================================================================

	// Away PP vs Home PK
	matchup.AwayPPvsPKAdvantage = awayPPStrength - (-homePKStrength)
	matchup.AwayPPvsPKAdvantage = math.Max(-1.0, math.Min(1.0, matchup.AwayPPvsPKAdvantage * 5.0))

	// Away PK vs Home PP
	matchup.AwayPKvsPPAdvantage = awayPKStrength - homePPStrength
	matchup.AwayPKvsPPAdvantage = math.Max(-1.0, math.Min(1.0, matchup.AwayPKvsPPAdvantage * 5.0))

	// ============================================================================
	// NET ADVANTAGE CALCULATION
	// ============================================================================

	// Combine PP and PK advantages
	matchup.HomeSTNetAdvantage = (matchup.HomePPvsPKAdvantage + matchup.HomePKvsPPAdvantage) / 2.0
	matchup.AwaySTNetAdvantage = (matchup.AwayPPvsPKAdvantage + matchup.AwayPKvsPPAdvantage) / 2.0

	// ============================================================================
	// EFFECTIVENESS PREDICTION
	// ============================================================================

	// Predict actual effectiveness in this matchup
	// Base effectiveness adjusted by opponent quality
	
	// Home PP effectiveness against Away PK
	awayPKEfficiency := awayFactors.PenaltyKillPct
	matchup.HomePPEffectiveness = homeFactors.PowerPlayPct * (1.0 + (1.0 - awayPKEfficiency))
	matchup.HomePPEffectiveness = math.Max(0.0, math.Min(1.0, matchup.HomePPEffectiveness))

	// Away PP effectiveness against Home PK
	homePKEfficiency := homeFactors.PenaltyKillPct
	matchup.AwayPPEffectiveness = awayFactors.PowerPlayPct * (1.0 + (1.0 - homePKEfficiency))
	matchup.AwayPPEffectiveness = math.Max(0.0, math.Min(1.0, matchup.AwayPPEffectiveness))

	// PK effectiveness (probability of killing penalty)
	matchup.HomePKEffectiveness = homeFactors.PenaltyKillPct * (1.0 - awayFactors.PowerPlayPct/leaguePP*0.5)
	matchup.HomePKEffectiveness = math.Max(0.0, math.Min(1.0, matchup.HomePKEffectiveness))

	matchup.AwayPKEffectiveness = awayFactors.PenaltyKillPct * (1.0 - homeFactors.PowerPlayPct/leaguePP*0.5)
	matchup.AwayPKEffectiveness = math.Max(0.0, math.Min(1.0, matchup.AwayPKEffectiveness))

	// ============================================================================
	// EXPECTED PP GOALS DIFFERENTIAL
	// ============================================================================

	// Estimate expected PP opportunities per game (league avg ~3-4 per team)
	avgPPOppsPerTeam := 3.5
	
	// Expected PP goals
	homeExpectedPPGoals := matchup.HomePPEffectiveness * avgPPOppsPerTeam
	awayExpectedPPGoals := matchup.AwayPPEffectiveness * avgPPOppsPerTeam
	
	matchup.ExpectedPPGoalsDiff = homeExpectedPPGoals - awayExpectedPPGoals

	// ============================================================================
	// OVERALL SPECIAL TEAMS IMPACT
	// ============================================================================

	// Calculate overall impact on game outcome
	// Special teams typically account for ~20-25% of goals
	// A 1-goal PP advantage in a 5-6 goal game is significant
	
	// Impact score: how much do special teams favor one team?
	matchup.SpecialTeamsImpact = matchup.HomeSTNetAdvantage
	
	// Amplify if there's a large expected goals differential
	if math.Abs(matchup.ExpectedPPGoalsDiff) > 0.5 {
		matchup.SpecialTeamsImpact *= 1.5
	}
	
	// Clamp to [-1, 1]
	matchup.SpecialTeamsImpact = math.Max(-1.0, math.Min(1.0, matchup.SpecialTeamsImpact))

	// ============================================================================
	// MATCHUP CHARACTERIZATION
	// ============================================================================

	// PP Mismatch: one team has significantly better PP vs opponent's PK
	ppDiff := math.Abs(matchup.HomePPvsPKAdvantage - matchup.AwayPPvsPKAdvantage)
	matchup.IsPPMismatch = ppDiff > 0.4

	// PK Mismatch: one team has significantly better PK vs opponent's PP
	pkDiff := math.Abs(matchup.HomePKvsPPAdvantage - matchup.AwayPKvsPPAdvantage)
	matchup.IsPKMismatch = pkDiff > 0.4

	// ST Equilibrium: teams are evenly matched on special teams
	totalDiff := math.Abs(matchup.HomeSTNetAdvantage - matchup.AwaySTNetAdvantage)
	matchup.IsSTEquilibrium = totalDiff < 0.2

	// Mismatch severity
	matchup.MismatchSeverity = math.Max(ppDiff, pkDiff)

	return matchup
}

// EnrichPredictionFactors adds special teams matchup features to prediction factors
func (stma *SpecialTeamsMatchupAnalyzer) EnrichPredictionFactors(
	homeFactors, awayFactors *models.PredictionFactors,
	matchup *SpecialTeamsMatchup,
) {
	// Add matchup-specific features to home factors
	homeFactors.SpecialTeamsDominance = matchup.HomeSTNetAdvantage
	homeFactors.PowerPlayOpportunity = matchup.HomePPEffectiveness
	
	// Add matchup-specific features to away factors
	awayFactors.SpecialTeamsDominance = matchup.AwaySTNetAdvantage
	awayFactors.PowerPlayOpportunity = matchup.AwayPPEffectiveness
}

// GetSpecialTeamsImpactDescription returns a human-readable description
func (stma *SpecialTeamsMatchupAnalyzer) GetSpecialTeamsImpactDescription(
	matchup *SpecialTeamsMatchup,
	homeTeam, awayTeam string,
) string {
	if matchup.IsSTEquilibrium {
		return "Special teams are evenly matched"
	}

	if matchup.HomeSTNetAdvantage > 0.3 {
		if matchup.IsPPMismatch {
			return fmt.Sprintf("%s has a significant power play advantage", homeTeam)
		}
		if matchup.IsPKMismatch {
			return fmt.Sprintf("%s has a dominant penalty kill vs %s PP", homeTeam, awayTeam)
		}
		return fmt.Sprintf("%s has the special teams edge", homeTeam)
	}

	if matchup.AwaySTNetAdvantage > 0.3 {
		if matchup.IsPPMismatch {
			return fmt.Sprintf("%s has a significant power play advantage", awayTeam)
		}
		if matchup.IsPKMismatch {
			return fmt.Sprintf("%s has a dominant penalty kill vs %s PP", awayTeam, homeTeam)
		}
		return fmt.Sprintf("%s has the special teams edge", awayTeam)
	}

	return "Special teams slight advantage exists but minimal impact expected"
}

// CalculateSpecialTeamsGoalSwing estimates goal swing from special teams
func (stma *SpecialTeamsMatchupAnalyzer) CalculateSpecialTeamsGoalSwing(
	matchup *SpecialTeamsMatchup,
) float64 {
	// Returns expected goal differential contribution from special teams
	// Positive = home team expected to score more on PP
	// Negative = away team expected to score more on PP
	return matchup.ExpectedPPGoalsDiff
}

