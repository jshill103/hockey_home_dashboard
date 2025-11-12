package services

import (
	"fmt"
	"log"

	"github.com/jaredshillingburg/go_uhc/models"
)

// RefereeEnrichmentService enriches prediction factors with referee data
type RefereeEnrichmentService struct {
	refereeService *RefereeService
}

// NewRefereeEnrichmentService creates a new referee enrichment service
func NewRefereeEnrichmentService() *RefereeEnrichmentService {
	return &RefereeEnrichmentService{
		refereeService: GetRefereeService(),
	}
}

// EnrichWithRefereeData adds referee impact to prediction factors
func (res *RefereeEnrichmentService) EnrichWithRefereeData(
	homeFactors *models.PredictionFactors,
	awayFactors *models.PredictionFactors,
	gameID int,
	homeTeam string,
	awayTeam string,
) error {
	if res.refereeService == nil {
		log.Printf("⚠️ Referee service not available - skipping referee enrichment")
		return fmt.Errorf("referee service not initialized")
	}

	// Get referee assignment for the game
	assignment, err := res.refereeService.GetGameAssignment(gameID)
	if err != nil {
		// No referee data available - not an error, just log and continue
		log.Printf("📋 No referee assignment found for game %d - using defaults", gameID)
		return nil
	}

	log.Printf("👔 Found referee assignment for game %d: %s & %s", 
		gameID, assignment.Referee1Name, assignment.Referee2Name)

	// Get advanced impact analysis
	impact, err := res.refereeService.AnalyzeRefereeImpact(gameID, homeTeam, awayTeam)
	if err != nil {
		log.Printf("⚠️ Could not analyze referee impact: %v", err)
		return err
	}

	// Get referee tendencies for both refs
	ref1Tendency, _ := res.refereeService.CalculateRefereeTendencies(assignment.Referee1ID)
	ref2Tendency, _ := res.refereeService.CalculateRefereeTendencies(assignment.Referee2ID)

	// Enrich home team factors
	homeFactors.RefereeAssigned = true
	homeFactors.RefereeTeamBias = impact.HomeTeamBiasScore
	homeFactors.RefereeHomeAdvantage = impact.HomeAdvantageAdjust
	homeFactors.RefereeConfidence = impact.ConfidenceLevel

	// Enrich away team factors
	awayFactors.RefereeAssigned = true
	awayFactors.RefereeTeamBias = impact.AwayTeamBiasScore
	awayFactors.RefereeHomeAdvantage = -impact.HomeAdvantageAdjust // Negative for away team
	awayFactors.RefereeConfidence = impact.ConfidenceLevel

	// Average referee tendencies if both available
	if ref1Tendency != nil && ref2Tendency != nil {
		avgPenaltyRate := (ref1Tendency.PenaltyCallRate + ref2Tendency.PenaltyCallRate) / 2
		avgConsistency := (ref1Tendency.ConsistencyScore + ref2Tendency.ConsistencyScore) / 2
		isOverTendency := (ref1Tendency.OverUnderTendency == "over" || ref2Tendency.OverUnderTendency == "over")

		// Determine overall tendency
		tendency := "average"
		if ref1Tendency.TendencyType == ref2Tendency.TendencyType {
			tendency = ref1Tendency.TendencyType
		} else if ref1Tendency.TendencyType == "strict" || ref2Tendency.TendencyType == "strict" {
			tendency = "strict"
		} else if ref1Tendency.TendencyType == "lenient" || ref2Tendency.TendencyType == "lenient" {
			tendency = "lenient"
		}

		// Apply to both teams
		homeFactors.RefereePenaltyRate = avgPenaltyRate
		homeFactors.RefereeTendency = tendency
		homeFactors.RefereeConsistency = avgConsistency
		homeFactors.RefereeOverTendency = isOverTendency

		awayFactors.RefereePenaltyRate = avgPenaltyRate
		awayFactors.RefereeTendency = tendency
		awayFactors.RefereeConsistency = avgConsistency
		awayFactors.RefereeOverTendency = isOverTendency

		log.Printf("🎯 Referee tendencies (both): %s (penalty rate: %.2fx, consistency: %.2f, over: %v)",
			tendency, avgPenaltyRate, avgConsistency, isOverTendency)
	} else if ref1Tendency != nil {
		// Use ref1 data if available
		homeFactors.RefereePenaltyRate = ref1Tendency.PenaltyCallRate
		homeFactors.RefereeTendency = ref1Tendency.TendencyType
		homeFactors.RefereeConsistency = ref1Tendency.ConsistencyScore
		homeFactors.RefereeOverTendency = (ref1Tendency.OverUnderTendency == "over")

		awayFactors.RefereePenaltyRate = ref1Tendency.PenaltyCallRate
		awayFactors.RefereeTendency = ref1Tendency.TendencyType
		awayFactors.RefereeConsistency = ref1Tendency.ConsistencyScore
		awayFactors.RefereeOverTendency = (ref1Tendency.OverUnderTendency == "over")
		
		log.Printf("🎯 Referee tendencies (ref1 only): %s (penalty rate: %.2fx)", 
			ref1Tendency.TendencyType, ref1Tendency.PenaltyCallRate)
	} else if ref2Tendency != nil {
		// Use ref2 data if ref1 failed but ref2 succeeded
		homeFactors.RefereePenaltyRate = ref2Tendency.PenaltyCallRate
		homeFactors.RefereeTendency = ref2Tendency.TendencyType
		homeFactors.RefereeConsistency = ref2Tendency.ConsistencyScore
		homeFactors.RefereeOverTendency = (ref2Tendency.OverUnderTendency == "over")

		awayFactors.RefereePenaltyRate = ref2Tendency.PenaltyCallRate
		awayFactors.RefereeTendency = ref2Tendency.TendencyType
		awayFactors.RefereeConsistency = ref2Tendency.ConsistencyScore
		awayFactors.RefereeOverTendency = (ref2Tendency.OverUnderTendency == "over")
		
		log.Printf("🎯 Referee tendencies (ref2 only): %s (penalty rate: %.2fx)", 
			ref2Tendency.TendencyType, ref2Tendency.PenaltyCallRate)
	}

	// Adjust home advantage based on referee impact
	if impact.HomeAdvantageAdjust != 0 {
		originalHomeAdv := homeFactors.HomeAdvantage
		homeFactors.HomeAdvantage += impact.HomeAdvantageAdjust * 0.01 // Convert to decimal
		log.Printf("🏠 Adjusted home advantage: %.3f -> %.3f (referee impact: %+.2f)", 
			originalHomeAdv, homeFactors.HomeAdvantage, impact.HomeAdvantageAdjust)
	}

	log.Printf("✅ Referee enrichment complete (confidence: %.1f%%)", impact.ConfidenceLevel*100)
	return nil
}

// EnrichPredictionContext adds referee information to prediction context
func (res *RefereeEnrichmentService) EnrichPredictionContext(
	context *PredictionContext,
	homeTeam string,
	awayTeam string,
) error {
	if res.refereeService == nil || context.GameID == 0 {
		return nil
	}

	// Get referee assignment
	assignment, err := res.refereeService.GetGameAssignment(context.GameID)
	if err != nil {
		// No referee data available
		return nil
	}

	// Get referee impact
	impact, err := res.refereeService.AnalyzeRefereeImpact(context.GameID, homeTeam, awayTeam)
	if err != nil {
		return err
	}

	// Update context
	context.RefereeAssignment = assignment
	context.RefereeImpact = impact

	return nil
}

// GetRefereeImpactSummary returns a summary of referee impact for display
func (res *RefereeEnrichmentService) GetRefereeImpactSummary(gameID int, homeTeam, awayTeam string) map[string]interface{} {
	summary := make(map[string]interface{})
	summary["available"] = false

	if res.refereeService == nil {
		summary["message"] = "Referee service not available"
		return summary
	}

	// Get referee assignment
	assignment, err := res.refereeService.GetGameAssignment(gameID)
	if err != nil {
		summary["message"] = "No referee assignment data available"
		return summary
	}

	// Get advanced impact analysis
	advancedImpact, err := res.refereeService.GetAdvancedImpactAnalysis(gameID, homeTeam, awayTeam)
	if err != nil {
		summary["message"] = "Could not analyze referee impact"
		return summary
	}

	summary["available"] = true
	summary["referees"] = []string{assignment.Referee1Name, assignment.Referee2Name}
	summary["impact"] = advancedImpact
	
	return summary
}

