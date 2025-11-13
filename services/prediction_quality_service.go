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
	qualityInstance *PredictionQualityService
	qualityOnce     sync.Once
)

// PredictionQualityService assesses the quality and reliability of predictions
type PredictionQualityService struct {
	mu              sync.RWMutex
	qualityMetrics  *models.QualityMetrics
	thresholds      models.QualityThresholds
	dataDir         string
}

// InitializePredictionQuality initializes the singleton PredictionQualityService
func InitializePredictionQuality() error {
	var err error
	qualityOnce.Do(func() {
		dataDir := filepath.Join("data", "quality")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create quality directory: %w", err)
			return
		}

		qualityInstance = &PredictionQualityService{
			qualityMetrics: &models.QualityMetrics{
				LastUpdated: time.Now(),
			},
			thresholds: models.QualityThresholds{
				ExcellentMin:         0.90,
				GoodMin:              0.75,
				FairMin:              0.60,
				HighReliabilityMin:   0.85,
				MediumReliabilityMin: 0.70,
			},
			dataDir: dataDir,
		}

		// Load existing quality metrics
		if loadErr := qualityInstance.loadQualityMetrics(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing quality metrics: %v\n", loadErr)
		}

		fmt.Println("✅ Prediction Quality Service initialized")
	})
	return err
}

// GetPredictionQualityService returns the singleton instance
func GetPredictionQualityService() *PredictionQualityService {
	return qualityInstance
}

// AssessPredictionQuality performs a comprehensive quality assessment
func (pqs *PredictionQualityService) AssessPredictionQuality(
	prediction *models.PredictionResult,
	homeFactors, awayFactors *models.PredictionFactors,
	modelResults []models.ModelResult) *models.PredictionQuality {

	quality := &models.PredictionQuality{
		Confidence:       prediction.Confidence,
		Factors:          make(map[string]float64),
		Warnings:         []string{},
		MissingFeatures:  []string{},
		UncertainFactors: []string{},
		StrengthFactors:  []string{},
		Timestamp:        time.Now(),
	}

	// Calculate individual quality components
	quality.ModelAgreement = pqs.calculateModelAgreement(modelResults)
	quality.DataQuality = pqs.calculateDataQuality(homeFactors, awayFactors)
	quality.HistoricalAccuracy = pqs.getHistoricalAccuracy(prediction)
	quality.SampleSize = pqs.assessSampleSize(homeFactors, awayFactors)
	quality.DataRecency = pqs.assessDataRecency(homeFactors, awayFactors)

	// Calculate overall quality score (weighted average)
	weights := map[string]float64{
		"modelAgreement":     0.30,
		"dataQuality":        0.25,
		"historicalAccuracy": 0.25,
		"sampleSize":         0.10,
		"dataRecency":        0.10,
	}

	quality.Score = 
		quality.ModelAgreement*weights["modelAgreement"]*100 +
		quality.DataQuality*weights["dataQuality"]*100 +
		quality.HistoricalAccuracy*weights["historicalAccuracy"]*100 +
		quality.SampleSize*weights["sampleSize"]*100 +
		quality.DataRecency*weights["dataRecency"]*100

	// Store individual factor scores
	quality.Factors["modelAgreement"] = quality.ModelAgreement
	quality.Factors["dataQuality"] = quality.DataQuality
	quality.Factors["historicalAccuracy"] = quality.HistoricalAccuracy
	quality.Factors["sampleSize"] = quality.SampleSize
	quality.Factors["dataRecency"] = quality.DataRecency

	// Determine tier
	quality.Tier = pqs.determineTier(quality.Score)
	quality.Reliability = pqs.determineReliability(quality.Score)

	// Generate warnings and recommendations
	pqs.generateWarnings(quality, homeFactors, awayFactors, modelResults)
	pqs.generateStrengths(quality, homeFactors, awayFactors, modelResults)

	// Overall assessment
	pqs.generateAssessment(quality)

	// Adjust confidence based on quality
	quality.Confidence = pqs.adjustConfidenceForQuality(prediction.Confidence, quality.Score)

	return quality
}

// RecordPredictionQuality updates quality metrics after a prediction
func (pqs *PredictionQualityService) RecordPredictionQuality(quality *models.PredictionQuality, correct bool) error {
	pqs.mu.Lock()
	defer pqs.mu.Unlock()

	pqs.qualityMetrics.TotalPredictions++

	// Update tier counts
	switch quality.Tier {
	case "Excellent":
		pqs.qualityMetrics.ExcellentCount++
	case "Good":
		pqs.qualityMetrics.GoodCount++
	case "Fair":
		pqs.qualityMetrics.FairCount++
	case "Poor":
		pqs.qualityMetrics.PoorCount++
	}

	// Update rolling averages
	alpha := 0.1
	pqs.qualityMetrics.AvgQualityScore = alpha*quality.Score + (1-alpha)*pqs.qualityMetrics.AvgQualityScore
	pqs.qualityMetrics.AvgModelAgreement = alpha*quality.ModelAgreement + (1-alpha)*pqs.qualityMetrics.AvgModelAgreement
	pqs.qualityMetrics.AvgDataCompleteness = alpha*quality.DataQuality + (1-alpha)*pqs.qualityMetrics.AvgDataCompleteness

	pqs.qualityMetrics.LastUpdated = time.Now()

	return pqs.saveQualityMetrics()
}

// ============================================================================
// QUALITY CALCULATION METHODS
// ============================================================================

func (pqs *PredictionQualityService) calculateModelAgreement(modelResults []models.ModelResult) float64 {
	if len(modelResults) < 2 {
		return 0.5 // Not enough models to assess agreement
	}

	// Calculate variance in predictions
	var predictions []float64
	for _, result := range modelResults {
		predictions = append(predictions, result.WinProbability)
	}

	// Calculate mean
	mean := 0.0
	for _, p := range predictions {
		mean += p
	}
	mean /= float64(len(predictions))

	// Calculate variance
	variance := 0.0
	for _, p := range predictions {
		variance += math.Pow(p-mean, 2)
	}
	variance /= float64(len(predictions))

	// Convert variance to agreement score (lower variance = higher agreement)
	// Variance ranges from 0 (perfect agreement) to ~0.0625 (complete disagreement)
	agreementScore := 1.0 - math.Min(variance*16.0, 1.0)

	return agreementScore
}

func (pqs *PredictionQualityService) calculateDataQuality(homeFactors, awayFactors *models.PredictionFactors) float64 {
	totalFeatures := 0
	populatedFeatures := 0

	// Check key features (simplified - would check all features in production)
	features := []float64{
		homeFactors.GoalsFor, awayFactors.GoalsFor,
		homeFactors.GoalsAgainst, awayFactors.GoalsAgainst,
		homeFactors.PowerPlayPct, awayFactors.PowerPlayPct,
		homeFactors.PenaltyKillPct, awayFactors.PenaltyKillPct,
		homeFactors.FaceoffWinPct, awayFactors.FaceoffWinPct,
	}

	for _, feature := range features {
		totalFeatures++
		if feature != 0.0 {
			populatedFeatures++
		}
	}

	if totalFeatures == 0 {
		return 0.5
	}

	return float64(populatedFeatures) / float64(totalFeatures)
}

func (pqs *PredictionQualityService) getHistoricalAccuracy(prediction *models.PredictionResult) float64 {
	// Check if we have calibration service
	calibService := GetConfidenceCalibrationService()
	if calibService == nil {
		return 0.70 // Default baseline
	}

	// Get the calibration curve
	curve := calibService.GetCalibrationCurve()
	if curve.TotalSamples < 20 {
		return 0.70 // Not enough data
	}

	// Find the bin for this confidence level
	for _, bin := range curve.Bins {
		if prediction.Confidence >= bin.MinConfidence && prediction.Confidence < bin.MaxConfidence {
			if bin.SampleSize >= 5 {
				return bin.ActualAccuracy
			}
		}
	}

	return 0.70 // Default if no matching bin
}

func (pqs *PredictionQualityService) assessSampleSize(homeFactors, awayFactors *models.PredictionFactors) float64 {
	// Check if we have sufficient historical data
	h2hService := GetHeadToHeadService()
	if h2hService == nil {
		return 0.60 // Default if service not available
	}

	matchup, err := h2hService.GetMatchupAnalysis(homeFactors.TeamCode, awayFactors.TeamCode)
	if err != nil || matchup == nil {
		return 0.50 // No H2H data
	}

	// Score based on sample size
	sampleSize := matchup.TotalGames
	if sampleSize >= 10 {
		return 1.0
	} else if sampleSize >= 5 {
		return 0.75
	} else if sampleSize >= 3 {
		return 0.60
	} else if sampleSize > 0 {
		return 0.40
	}

	return 0.30 // No samples
}

func (pqs *PredictionQualityService) assessDataRecency(homeFactors, awayFactors *models.PredictionFactors) float64 {
	// In a real implementation, we'd check the timestamp of the data sources
	// For now, assume reasonably recent data
	return 0.85
}

// ============================================================================
// ASSESSMENT METHODS
// ============================================================================

func (pqs *PredictionQualityService) determineTier(score float64) string {
	if score >= pqs.thresholds.ExcellentMin*100 {
		return "Excellent"
	} else if score >= pqs.thresholds.GoodMin*100 {
		return "Good"
	} else if score >= pqs.thresholds.FairMin*100 {
		return "Fair"
	}
	return "Poor"
}

func (pqs *PredictionQualityService) determineReliability(score float64) string {
	if score >= pqs.thresholds.HighReliabilityMin*100 {
		return "High"
	} else if score >= pqs.thresholds.MediumReliabilityMin*100 {
		return "Medium"
	}
	return "Low"
}

func (pqs *PredictionQualityService) generateWarnings(
	quality *models.PredictionQuality,
	homeFactors, awayFactors *models.PredictionFactors,
	modelResults []models.ModelResult) {

	if quality.ModelAgreement < 0.60 {
		quality.Warnings = append(quality.Warnings, "Low model agreement - models disagree significantly")
		quality.UncertainFactors = append(quality.UncertainFactors, "Model consensus")
	}

	if quality.DataQuality < 0.70 {
		quality.Warnings = append(quality.Warnings, "Incomplete data - some key features missing")
		quality.UncertainFactors = append(quality.UncertainFactors, "Data completeness")
	}

	if quality.SampleSize < 0.60 {
		quality.Warnings = append(quality.Warnings, "Limited historical matchup data")
		quality.UncertainFactors = append(quality.UncertainFactors, "Sample size")
	}

	if quality.Score < 60 {
		quality.UseWithCaution = true
		quality.RecommendedAction = "Use with caution - consider additional research"
	}
}

func (pqs *PredictionQualityService) generateStrengths(
	quality *models.PredictionQuality,
	homeFactors, awayFactors *models.PredictionFactors,
	modelResults []models.ModelResult) {

	if quality.ModelAgreement > 0.85 {
		quality.StrengthFactors = append(quality.StrengthFactors, "Strong model consensus")
	}

	if quality.DataQuality > 0.90 {
		quality.StrengthFactors = append(quality.StrengthFactors, "Complete data coverage")
	}

	if quality.HistoricalAccuracy > 0.75 {
		quality.StrengthFactors = append(quality.StrengthFactors, "High historical accuracy")
	}

	if quality.SampleSize > 0.80 {
		quality.StrengthFactors = append(quality.StrengthFactors, "Substantial historical data")
	}
}

func (pqs *PredictionQualityService) generateAssessment(quality *models.PredictionQuality) {
	assessment := fmt.Sprintf("Prediction Quality: %s (%.1f/100). ", quality.Tier, quality.Score)

	if quality.Reliability == "High" {
		assessment += "High reliability - confident in this prediction. "
	} else if quality.Reliability == "Medium" {
		assessment += "Medium reliability - reasonable confidence. "
	} else {
		assessment += "Low reliability - use with caution. "
	}

	if len(quality.StrengthFactors) > 0 {
		assessment += fmt.Sprintf("Strengths: %s. ", quality.StrengthFactors[0])
	}

	if len(quality.Warnings) > 0 {
		assessment += fmt.Sprintf("Note: %s", quality.Warnings[0])
	}

	quality.QualityExplanation = assessment

	if quality.UseWithCaution {
		quality.RecommendedAction = "Use with caution - consider additional analysis"
	} else if quality.Tier == "Excellent" {
		quality.RecommendedAction = "High confidence - suitable for decision-making"
	} else {
		quality.RecommendedAction = "Good quality - proceed with normal confidence"
	}
}

func (pqs *PredictionQualityService) adjustConfidenceForQuality(originalConf, qualityScore float64) float64 {
	// Reduce confidence if quality is low
	qualityFactor := qualityScore / 100.0

	// Apply adjustment with smoothing
	adjustedConf := originalConf * (0.7 + 0.3*qualityFactor)

	// Ensure reasonable bounds
	return math.Max(0.50, math.Min(0.95, adjustedConf))
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (pqs *PredictionQualityService) getQualityMetricsPath() string {
	return filepath.Join(pqs.dataDir, "quality_metrics.json")
}

func (pqs *PredictionQualityService) saveQualityMetrics() error {
	data, err := json.MarshalIndent(pqs.qualityMetrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal quality metrics: %w", err)
	}
	if err := os.WriteFile(pqs.getQualityMetricsPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write quality metrics: %w", err)
	}
	return nil
}

func (pqs *PredictionQualityService) loadQualityMetrics() error {
	filePath := pqs.getQualityMetricsPath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data yet
		}
		return fmt.Errorf("failed to read quality metrics: %w", err)
	}

	if err := json.Unmarshal(data, &pqs.qualityMetrics); err != nil {
		return fmt.Errorf("failed to unmarshal quality metrics: %w", err)
	}

	fmt.Printf("📊 Loaded quality metrics: %d predictions tracked\n", pqs.qualityMetrics.TotalPredictions)
	return nil
}

// GetQualityMetrics returns current quality metrics
func (pqs *PredictionQualityService) GetQualityMetrics() *models.QualityMetrics {
	pqs.mu.RLock()
	defer pqs.mu.RUnlock()

	// Return a copy
	metricsCopy := *pqs.qualityMetrics
	return &metricsCopy
}

