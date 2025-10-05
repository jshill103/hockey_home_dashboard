package services

import (
	"log"
	"math"
	"sort"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ModelUncertaintyService quantifies and analyzes prediction uncertainty
type ModelUncertaintyService struct {
	historicalPredictions []UncertaintyDataPoint
	calibrationCurve      []CalibrationPoint
	uncertaintyMetrics    UncertaintyMetrics
	lastUpdate            time.Time
}

// UncertaintyDataPoint represents a historical prediction with uncertainty
type UncertaintyDataPoint struct {
	PredictionID      string    `json:"predictionId"`
	GameDate          time.Time `json:"gameDate"`
	HomeTeam          string    `json:"homeTeam"`
	AwayTeam          string    `json:"awayTeam"`
	PredictedWinner   string    `json:"predictedWinner"`
	WinProbability    float64   `json:"winProbability"`
	RawConfidence     float64   `json:"rawConfidence"`
	ModelUncertainty  float64   `json:"modelUncertainty"`
	DataUncertainty   float64   `json:"dataUncertainty"`
	TotalUncertainty  float64   `json:"totalUncertainty"`
	ActualWinner      string    `json:"actualWinner"`
	WasCorrect        bool      `json:"wasCorrect"`
	ConfidenceError   float64   `json:"confidenceError"`
	UncertaintyBucket string    `json:"uncertaintyBucket"` // "Low", "Medium", "High"
}

// CalibrationPoint represents a point on the calibration curve
type CalibrationPoint struct {
	PredictedProbability float64            `json:"predictedProbability"`
	ActualFrequency      float64            `json:"actualFrequency"`
	SampleSize           int                `json:"sampleSize"`
	ConfidenceInterval   ConfidenceInterval `json:"confidenceInterval"`
}

// UncertaintyMetrics represents overall uncertainty statistics
type UncertaintyMetrics struct {
	CalibrationError   float64                  `json:"calibrationError"`   // Brier score
	Overconfidence     float64                  `json:"overconfidence"`     // How much we overestimate
	Underconfidence    float64                  `json:"underconfidence"`    // How much we underestimate
	UncertaintyBias    float64                  `json:"uncertaintyBias"`    // Systematic uncertainty error
	ReliabilityScore   float64                  `json:"reliabilityScore"`   // Overall reliability (0-1)
	ResolutionScore    float64                  `json:"resolutionScore"`    // Ability to discriminate
	SharpnessScore     float64                  `json:"sharpnessScore"`     // How sharp predictions are
	UncertaintyBuckets map[string]BucketMetrics `json:"uncertaintyBuckets"` // Performance by uncertainty level
	LastUpdated        time.Time                `json:"lastUpdated"`
}

// BucketMetrics represents performance within an uncertainty bucket
type BucketMetrics struct {
	BucketName       string  `json:"bucketName"`       // "Low", "Medium", "High"
	PredictionCount  int     `json:"predictionCount"`  // Number of predictions
	Accuracy         float64 `json:"accuracy"`         // Accuracy within bucket
	AvgUncertainty   float64 `json:"avgUncertainty"`   // Average uncertainty
	CalibrationError float64 `json:"calibrationError"` // Calibration within bucket
	ExpectedAccuracy float64 `json:"expectedAccuracy"` // What accuracy should be
}

// UncertaintyQuantification represents uncertainty analysis for a prediction
type UncertaintyQuantification struct {
	ModelAgreement       float64             `json:"modelAgreement"`       // 0-1, how much models agree
	DataQuality          float64             `json:"dataQuality"`          // 0-1, quality of input data
	HistoricalVariance   float64             `json:"historicalVariance"`   // Historical prediction variance
	FeatureUncertainty   map[string]float64  `json:"featureUncertainty"`   // Uncertainty per feature
	EpistemicUncertainty float64             `json:"epistemicUncertainty"` // Model uncertainty
	AleatoryUncertainty  float64             `json:"aleatoryUncertainty"`  // Data uncertainty
	TotalUncertainty     float64             `json:"totalUncertainty"`     // Combined uncertainty
	UncertaintySource    []UncertaintySource `json:"uncertaintySource"`    // Sources of uncertainty
	ConfidenceInterval   ConfidenceInterval  `json:"confidenceInterval"`   // Prediction interval
	RecommendedAction    string              `json:"recommendedAction"`    // How to interpret
}

// UncertaintySource represents a source of prediction uncertainty
type UncertaintySource struct {
	Source       string  `json:"source"`       // "Model Disagreement", "Data Quality", etc.
	Contribution float64 `json:"contribution"` // 0-1, contribution to total uncertainty
	Description  string  `json:"description"`  // Human-readable explanation
	Severity     string  `json:"severity"`     // "Low", "Medium", "High"
}

// NewModelUncertaintyService creates a new uncertainty quantification service
func NewModelUncertaintyService() *ModelUncertaintyService {
	return &ModelUncertaintyService{
		historicalPredictions: make([]UncertaintyDataPoint, 0),
		calibrationCurve:      make([]CalibrationPoint, 0),
		uncertaintyMetrics: UncertaintyMetrics{
			UncertaintyBuckets: make(map[string]BucketMetrics),
		},
	}
}

// QuantifyUncertainty analyzes uncertainty for a prediction
func (mus *ModelUncertaintyService) QuantifyUncertainty(
	prediction *models.PredictionResult,
	homeFactors, awayFactors *models.PredictionFactors) *UncertaintyQuantification {

	log.Printf("🎯 Quantifying prediction uncertainty...")

	// Calculate model agreement
	modelAgreement := mus.calculateModelAgreement(prediction.ModelResults)

	// Calculate data quality
	dataQuality := mus.calculateDataQuality(homeFactors, awayFactors)

	// Calculate historical variance
	historicalVariance := mus.calculateHistoricalVariance(homeFactors.TeamCode, awayFactors.TeamCode)

	// Calculate feature uncertainty
	featureUncertainty := mus.calculateFeatureUncertainty(homeFactors, awayFactors)

	// Calculate epistemic uncertainty (model uncertainty)
	epistemicUncertainty := mus.calculateEpistemicUncertainty(prediction.ModelResults)

	// Calculate aleatory uncertainty (data uncertainty)
	aleatoryUncertainty := mus.calculateAleatoryUncertainty(homeFactors, awayFactors)

	// Calculate total uncertainty
	totalUncertainty := mus.combineTotalUncertainty(epistemicUncertainty, aleatoryUncertainty)

	// Identify uncertainty sources
	uncertaintySources := mus.identifyUncertaintySources(
		modelAgreement, dataQuality, epistemicUncertainty, aleatoryUncertainty)

	// Calculate confidence interval
	confidenceInterval := mus.calculateConfidenceInterval(prediction.WinProbability, totalUncertainty)

	// Determine recommended action
	recommendedAction := mus.determineRecommendedAction(totalUncertainty, modelAgreement)

	uncertainty := &UncertaintyQuantification{
		ModelAgreement:       modelAgreement,
		DataQuality:          dataQuality,
		HistoricalVariance:   historicalVariance,
		FeatureUncertainty:   featureUncertainty,
		EpistemicUncertainty: epistemicUncertainty,
		AleatoryUncertainty:  aleatoryUncertainty,
		TotalUncertainty:     totalUncertainty,
		UncertaintySource:    uncertaintySources,
		ConfidenceInterval:   confidenceInterval,
		RecommendedAction:    recommendedAction,
	}

	log.Printf("✅ Uncertainty quantification complete - Total: %.2f, Model Agreement: %.2f",
		totalUncertainty, modelAgreement)

	return uncertainty
}

// calculateModelAgreement measures how much the models agree
func (mus *ModelUncertaintyService) calculateModelAgreement(modelResults []models.ModelResult) float64 {
	if len(modelResults) < 2 {
		return 0.5 // Default if insufficient models
	}

	// Calculate variance in win probabilities
	probabilities := make([]float64, len(modelResults))
	var sum float64

	for i, result := range modelResults {
		probabilities[i] = result.WinProbability
		sum += result.WinProbability
	}

	mean := sum / float64(len(probabilities))

	var variance float64
	for _, prob := range probabilities {
		variance += math.Pow(prob-mean, 2)
	}
	variance /= float64(len(probabilities))

	// Convert variance to agreement (lower variance = higher agreement)
	// Scale so that 0.25 variance (max realistic) = 0 agreement
	agreement := math.Max(0, 1.0-(variance/0.25))

	return agreement
}

// calculateDataQuality assesses the quality of input data
func (mus *ModelUncertaintyService) calculateDataQuality(homeFactors, awayFactors *models.PredictionFactors) float64 {
	qualityFactors := make([]float64, 0)

	// Injury impact quality (lower impact = higher quality)
	injuryQuality := 1.0 - (homeFactors.InjuryImpact.ImpactScore/50.0+awayFactors.InjuryImpact.ImpactScore/50.0)/2.0
	qualityFactors = append(qualityFactors, math.Max(0, injuryQuality))

	// Weather data quality
	weatherQuality := (homeFactors.WeatherAnalysis.Confidence + awayFactors.WeatherAnalysis.Confidence) / 2.0
	qualityFactors = append(qualityFactors, weatherQuality)

	// Advanced analytics quality (based on overall rating)
	analyticsQuality := (homeFactors.AdvancedStats.OverallRating + awayFactors.AdvancedStats.OverallRating) / 200.0
	qualityFactors = append(qualityFactors, analyticsQuality)

	// Player analysis removed - no real data available

	// Calculate average quality
	var totalQuality float64
	for _, quality := range qualityFactors {
		totalQuality += quality
	}

	return totalQuality / float64(len(qualityFactors))
}

// calculateHistoricalVariance calculates variance in historical predictions for these teams
func (mus *ModelUncertaintyService) calculateHistoricalVariance(homeTeam, awayTeam string) float64 {
	// In a real implementation, this would query historical predictions
	// For now, return a default based on team codes

	teamVarianceMap := map[string]float64{
		"UTA": 0.15, "COL": 0.12, "VGK": 0.13, "SJS": 0.18, "LAK": 0.14,
		"EDM": 0.11, "CGY": 0.16, "WPG": 0.13, "MIN": 0.15, "CHI": 0.17,
	}

	homeVariance := teamVarianceMap[homeTeam]
	if homeVariance == 0 {
		homeVariance = 0.15 // Default
	}

	awayVariance := teamVarianceMap[awayTeam]
	if awayVariance == 0 {
		awayVariance = 0.15 // Default
	}

	// Average the team variances
	return (homeVariance + awayVariance) / 2.0
}

// calculateFeatureUncertainty calculates uncertainty for each prediction feature
func (mus *ModelUncertaintyService) calculateFeatureUncertainty(homeFactors, awayFactors *models.PredictionFactors) map[string]float64 {
	featureUncertainty := make(map[string]float64)

	// Win percentage uncertainty (based on sample size - games played)
	featureUncertainty["WinPercentage"] = mus.calculateSampleSizeUncertainty(82) // NHL season games

	// Recent form uncertainty (smaller sample)
	featureUncertainty["RecentForm"] = mus.calculateSampleSizeUncertainty(10) // Last 10 games

	// Head-to-head uncertainty (very small sample)
	featureUncertainty["HeadToHead"] = mus.calculateSampleSizeUncertainty(4) // ~4 games per season

	// Injury uncertainty (based on impact confidence)
	injuryUncertainty := 1.0 - (homeFactors.InjuryImpact.Confidence+awayFactors.InjuryImpact.Confidence)/2.0
	featureUncertainty["InjuryImpact"] = injuryUncertainty

	// Weather uncertainty
	weatherUncertainty := 1.0 - (homeFactors.WeatherAnalysis.Confidence+awayFactors.WeatherAnalysis.Confidence)/2.0
	featureUncertainty["WeatherAnalysis"] = weatherUncertainty

	// Advanced stats uncertainty (based on data completeness)
	featureUncertainty["AdvancedStats"] = 0.1 // Relatively low uncertainty for established metrics

	// Player analysis uncertainty
	// Player analysis removed - no real data available

	return featureUncertainty
}

// calculateSampleSizeUncertainty calculates uncertainty based on sample size
func (mus *ModelUncertaintyService) calculateSampleSizeUncertainty(sampleSize int) float64 {
	// Use inverse square root relationship
	// Larger samples have lower uncertainty
	if sampleSize <= 0 {
		return 1.0
	}

	uncertainty := 1.0 / math.Sqrt(float64(sampleSize))

	// Normalize to reasonable range (0-1)
	if uncertainty > 1.0 {
		uncertainty = 1.0
	}

	return uncertainty
}

// calculateEpistemicUncertainty calculates model uncertainty
func (mus *ModelUncertaintyService) calculateEpistemicUncertainty(modelResults []models.ModelResult) float64 {
	if len(modelResults) < 2 {
		return 0.3 // Default uncertainty
	}

	// Calculate coefficient of variation in model confidences
	confidences := make([]float64, len(modelResults))
	var sum float64

	for i, result := range modelResults {
		confidences[i] = result.Confidence
		sum += result.Confidence
	}

	mean := sum / float64(len(confidences))

	var variance float64
	for _, conf := range confidences {
		variance += math.Pow(conf-mean, 2)
	}
	variance /= float64(len(confidences))

	if mean == 0 {
		return 0.5 // Default if no confidence data
	}

	// Coefficient of variation
	cv := math.Sqrt(variance) / mean

	// Normalize to 0-1 range
	epistemicUncertainty := math.Min(cv, 1.0)

	return epistemicUncertainty
}

// calculateAleatoryUncertainty calculates data uncertainty
func (mus *ModelUncertaintyService) calculateAleatoryUncertainty(homeFactors, awayFactors *models.PredictionFactors) float64 {
	uncertaintyFactors := make([]float64, 0)

	// Travel fatigue uncertainty (use fatigue score as proxy)
	travelUncertainty := (homeFactors.TravelFatigue.FatigueScore + awayFactors.TravelFatigue.FatigueScore) / 2.0
	uncertaintyFactors = append(uncertaintyFactors, travelUncertainty)

	// Injury uncertainty
	injuryUncertainty := 1.0 - (homeFactors.InjuryImpact.Confidence+awayFactors.InjuryImpact.Confidence)/2.0
	uncertaintyFactors = append(uncertaintyFactors, injuryUncertainty)

	// Weather uncertainty
	weatherUncertainty := 1.0 - (homeFactors.WeatherAnalysis.Confidence+awayFactors.WeatherAnalysis.Confidence)/2.0
	uncertaintyFactors = append(uncertaintyFactors, weatherUncertainty)

	// Calculate average aleatory uncertainty
	var totalUncertainty float64
	for _, uncertainty := range uncertaintyFactors {
		totalUncertainty += uncertainty
	}

	return totalUncertainty / float64(len(uncertaintyFactors))
}

// combineTotalUncertainty combines epistemic and aleatory uncertainty
func (mus *ModelUncertaintyService) combineTotalUncertainty(epistemic, aleatory float64) float64 {
	// Use quadrature sum (square root of sum of squares)
	// This assumes uncertainties are independent
	totalUncertainty := math.Sqrt(epistemic*epistemic + aleatory*aleatory)

	// Ensure result is in [0, 1] range
	if totalUncertainty > 1.0 {
		totalUncertainty = 1.0
	}

	return totalUncertainty
}

// identifyUncertaintySources identifies the main sources of uncertainty
func (mus *ModelUncertaintyService) identifyUncertaintySources(
	modelAgreement, dataQuality, epistemic, aleatory float64) []UncertaintySource {

	sources := make([]UncertaintySource, 0)

	// Model disagreement
	if modelAgreement < 0.7 {
		severity := "Medium"
		if modelAgreement < 0.5 {
			severity = "High"
		}

		sources = append(sources, UncertaintySource{
			Source:       "Model Disagreement",
			Contribution: 1.0 - modelAgreement,
			Description:  "Prediction models disagree on the outcome",
			Severity:     severity,
		})
	}

	// Data quality issues
	if dataQuality < 0.8 {
		severity := "Medium"
		if dataQuality < 0.6 {
			severity = "High"
		}

		sources = append(sources, UncertaintySource{
			Source:       "Data Quality",
			Contribution: 1.0 - dataQuality,
			Description:  "Input data has quality or completeness issues",
			Severity:     severity,
		})
	}

	// High epistemic uncertainty
	if epistemic > 0.3 {
		severity := "Medium"
		if epistemic > 0.5 {
			severity = "High"
		}

		sources = append(sources, UncertaintySource{
			Source:       "Model Uncertainty",
			Contribution: epistemic,
			Description:  "Models have inherent limitations or insufficient training",
			Severity:     severity,
		})
	}

	// High aleatory uncertainty
	if aleatory > 0.3 {
		severity := "Medium"
		if aleatory > 0.5 {
			severity = "High"
		}

		sources = append(sources, UncertaintySource{
			Source:       "Data Uncertainty",
			Contribution: aleatory,
			Description:  "Inherent randomness or noise in the data",
			Severity:     severity,
		})
	}

	// Sort by contribution (highest first)
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Contribution > sources[j].Contribution
	})

	return sources
}

// calculateConfidenceInterval calculates prediction confidence interval
func (mus *ModelUncertaintyService) calculateConfidenceInterval(winProbability, totalUncertainty float64) ConfidenceInterval {
	// Use normal approximation for confidence interval
	// Standard error based on total uncertainty
	standardError := totalUncertainty / 2.0 // Rough approximation

	// 95% confidence interval (1.96 * SE)
	margin := 1.96 * standardError

	lower := math.Max(0.0, winProbability-margin)
	upper := math.Min(1.0, winProbability+margin)

	return ConfidenceInterval{
		Lower:           lower,
		Upper:           upper,
		ConfidenceLevel: 0.95,
	}
}

// determineRecommendedAction provides guidance on how to interpret the prediction
func (mus *ModelUncertaintyService) determineRecommendedAction(totalUncertainty, modelAgreement float64) string {
	if totalUncertainty < 0.2 && modelAgreement > 0.8 {
		return "High Confidence - Models agree and uncertainty is low"
	} else if totalUncertainty < 0.3 && modelAgreement > 0.7 {
		return "Medium Confidence - Good model agreement with moderate uncertainty"
	} else if totalUncertainty > 0.5 || modelAgreement < 0.5 {
		return "Low Confidence - High uncertainty or significant model disagreement"
	} else {
		return "Moderate Confidence - Use prediction with caution"
	}
}

// UpdateCalibration updates the calibration curve with new prediction outcomes
func (mus *ModelUncertaintyService) UpdateCalibration(predictions []UncertaintyDataPoint) {
	log.Printf("📊 Updating uncertainty calibration with %d predictions", len(predictions))

	// Add to historical data
	mus.historicalPredictions = append(mus.historicalPredictions, predictions...)

	// Rebuild calibration curve
	mus.rebuildCalibrationCurve()

	// Update uncertainty metrics
	mus.updateUncertaintyMetrics()

	mus.lastUpdate = time.Now()

	log.Printf("✅ Calibration updated - Reliability: %.3f, Resolution: %.3f",
		mus.uncertaintyMetrics.ReliabilityScore, mus.uncertaintyMetrics.ResolutionScore)
}

// rebuildCalibrationCurve rebuilds the calibration curve from historical data
func (mus *ModelUncertaintyService) rebuildCalibrationCurve() {
	if len(mus.historicalPredictions) < 10 {
		return // Need minimum data
	}

	// Create probability bins
	bins := []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0}
	calibrationPoints := make([]CalibrationPoint, 0)

	for i := 0; i < len(bins)-1; i++ {
		binLower := bins[i]
		binUpper := bins[i+1]
		binCenter := (binLower + binUpper) / 2.0

		// Find predictions in this bin
		binPredictions := make([]UncertaintyDataPoint, 0)
		for _, pred := range mus.historicalPredictions {
			if pred.WinProbability >= binLower && pred.WinProbability < binUpper {
				binPredictions = append(binPredictions, pred)
			}
		}

		if len(binPredictions) < 3 {
			continue // Skip bins with too few samples
		}

		// Calculate actual frequency in this bin
		correctPredictions := 0
		for _, pred := range binPredictions {
			if pred.WasCorrect {
				correctPredictions++
			}
		}

		actualFrequency := float64(correctPredictions) / float64(len(binPredictions))

		// Calculate confidence interval for this bin
		ci := mus.calculateBinConfidenceInterval(actualFrequency, len(binPredictions))

		calibrationPoints = append(calibrationPoints, CalibrationPoint{
			PredictedProbability: binCenter,
			ActualFrequency:      actualFrequency,
			SampleSize:           len(binPredictions),
			ConfidenceInterval:   ci,
		})
	}

	mus.calibrationCurve = calibrationPoints
}

// calculateBinConfidenceInterval calculates confidence interval for a calibration bin
func (mus *ModelUncertaintyService) calculateBinConfidenceInterval(frequency float64, sampleSize int) ConfidenceInterval {
	if sampleSize < 3 {
		return ConfidenceInterval{Lower: 0, Upper: 1, ConfidenceLevel: 0.95}
	}

	// Wilson score interval for binomial proportion
	z := 1.96 // 95% confidence
	n := float64(sampleSize)
	p := frequency

	denominator := 1 + z*z/n
	center := (p + z*z/(2*n)) / denominator
	margin := z * math.Sqrt((p*(1-p)+z*z/(4*n))/n) / denominator

	lower := math.Max(0, center-margin)
	upper := math.Min(1, center+margin)

	return ConfidenceInterval{
		Lower:           lower,
		Upper:           upper,
		ConfidenceLevel: 0.95,
	}
}

// updateUncertaintyMetrics calculates overall uncertainty metrics
func (mus *ModelUncertaintyService) updateUncertaintyMetrics() {
	if len(mus.historicalPredictions) < 10 {
		return
	}

	// Calculate Brier score (calibration error)
	brierScore := mus.calculateBrierScore()

	// Calculate overconfidence and underconfidence
	overconfidence, underconfidence := mus.calculateConfidenceBias()

	// Calculate reliability, resolution, and sharpness
	reliability := mus.calculateReliability()
	resolution := mus.calculateResolution()
	sharpness := mus.calculateSharpness()

	// Update uncertainty buckets
	uncertaintyBuckets := mus.calculateUncertaintyBuckets()

	mus.uncertaintyMetrics = UncertaintyMetrics{
		CalibrationError:   brierScore,
		Overconfidence:     overconfidence,
		Underconfidence:    underconfidence,
		UncertaintyBias:    overconfidence - underconfidence,
		ReliabilityScore:   reliability,
		ResolutionScore:    resolution,
		SharpnessScore:     sharpness,
		UncertaintyBuckets: uncertaintyBuckets,
		LastUpdated:        time.Now(),
	}
}

// calculateBrierScore calculates the Brier score for calibration
func (mus *ModelUncertaintyService) calculateBrierScore() float64 {
	if len(mus.historicalPredictions) == 0 {
		return 0.5
	}

	var totalSquaredError float64
	for _, pred := range mus.historicalPredictions {
		actual := 0.0
		if pred.WasCorrect {
			actual = 1.0
		}

		error := pred.WinProbability - actual
		totalSquaredError += error * error
	}

	return totalSquaredError / float64(len(mus.historicalPredictions))
}

// calculateConfidenceBias calculates overconfidence and underconfidence
func (mus *ModelUncertaintyService) calculateConfidenceBias() (float64, float64) {
	if len(mus.historicalPredictions) == 0 {
		return 0, 0
	}

	var overconfidentCount, underconfidentCount int
	var overconfidenceSum, underconfidenceSum float64

	for _, pred := range mus.historicalPredictions {
		if pred.WasCorrect {
			// Correct prediction
			if pred.RawConfidence < pred.WinProbability {
				underconfidentCount++
				underconfidenceSum += pred.WinProbability - pred.RawConfidence
			}
		} else {
			// Incorrect prediction
			if pred.RawConfidence > (1.0 - pred.WinProbability) {
				overconfidentCount++
				overconfidenceSum += pred.RawConfidence - (1.0 - pred.WinProbability)
			}
		}
	}

	overconfidence := 0.0
	if overconfidentCount > 0 {
		overconfidence = overconfidenceSum / float64(overconfidentCount)
	}

	underconfidence := 0.0
	if underconfidentCount > 0 {
		underconfidence = underconfidenceSum / float64(underconfidentCount)
	}

	return overconfidence, underconfidence
}

// calculateReliability calculates how well calibrated the predictions are
func (mus *ModelUncertaintyService) calculateReliability() float64 {
	if len(mus.calibrationCurve) == 0 {
		return 0.5
	}

	var totalWeightedError float64
	var totalWeight float64

	for _, point := range mus.calibrationCurve {
		weight := float64(point.SampleSize)
		error := math.Abs(point.PredictedProbability - point.ActualFrequency)

		totalWeightedError += weight * error
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.5
	}

	avgError := totalWeightedError / totalWeight

	// Convert to reliability score (lower error = higher reliability)
	reliability := math.Max(0, 1.0-avgError*2) // Scale so 0.5 error = 0 reliability

	return reliability
}

// calculateResolution calculates the ability to discriminate between outcomes
func (mus *ModelUncertaintyService) calculateResolution() float64 {
	if len(mus.historicalPredictions) == 0 {
		return 0.5
	}

	// Calculate overall accuracy
	correctPredictions := 0
	for _, pred := range mus.historicalPredictions {
		if pred.WasCorrect {
			correctPredictions++
		}
	}

	overallAccuracy := float64(correctPredictions) / float64(len(mus.historicalPredictions))

	// Calculate weighted variance of bin accuracies
	var weightedVariance float64
	var totalWeight float64

	for _, point := range mus.calibrationCurve {
		weight := float64(point.SampleSize)
		deviation := point.ActualFrequency - overallAccuracy

		weightedVariance += weight * deviation * deviation
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.5
	}

	resolution := weightedVariance / totalWeight

	// Normalize to 0-1 scale
	resolution = math.Min(resolution*4, 1.0) // Scale appropriately

	return resolution
}

// calculateSharpness calculates how confident/sharp the predictions are
func (mus *ModelUncertaintyService) calculateSharpness() float64 {
	if len(mus.historicalPredictions) == 0 {
		return 0.5
	}

	// Calculate average distance from 0.5 (neutral)
	var totalDistance float64
	for _, pred := range mus.historicalPredictions {
		distance := math.Abs(pred.WinProbability - 0.5)
		totalDistance += distance
	}

	avgDistance := totalDistance / float64(len(mus.historicalPredictions))

	// Normalize to 0-1 scale (max distance is 0.5)
	sharpness := avgDistance / 0.5

	return sharpness
}

// calculateUncertaintyBuckets calculates performance within uncertainty buckets
func (mus *ModelUncertaintyService) calculateUncertaintyBuckets() map[string]BucketMetrics {
	buckets := map[string]BucketMetrics{
		"Low":    {BucketName: "Low"},
		"Medium": {BucketName: "Medium"},
		"High":   {BucketName: "High"},
	}

	if len(mus.historicalPredictions) == 0 {
		return buckets
	}

	// Categorize predictions by uncertainty
	lowUncertainty := make([]UncertaintyDataPoint, 0)
	mediumUncertainty := make([]UncertaintyDataPoint, 0)
	highUncertainty := make([]UncertaintyDataPoint, 0)

	for _, pred := range mus.historicalPredictions {
		if pred.TotalUncertainty < 0.3 {
			lowUncertainty = append(lowUncertainty, pred)
		} else if pred.TotalUncertainty < 0.6 {
			mediumUncertainty = append(mediumUncertainty, pred)
		} else {
			highUncertainty = append(highUncertainty, pred)
		}
	}

	// Calculate metrics for each bucket
	buckets["Low"] = mus.calculateBucketMetrics("Low", lowUncertainty)
	buckets["Medium"] = mus.calculateBucketMetrics("Medium", mediumUncertainty)
	buckets["High"] = mus.calculateBucketMetrics("High", highUncertainty)

	return buckets
}

// calculateBucketMetrics calculates metrics for a specific uncertainty bucket
func (mus *ModelUncertaintyService) calculateBucketMetrics(bucketName string, predictions []UncertaintyDataPoint) BucketMetrics {
	if len(predictions) == 0 {
		return BucketMetrics{
			BucketName:       bucketName,
			PredictionCount:  0,
			Accuracy:         0.5,
			AvgUncertainty:   0.5,
			CalibrationError: 0.5,
			ExpectedAccuracy: 0.5,
		}
	}

	// Calculate accuracy
	correctPredictions := 0
	var totalUncertainty float64
	var totalSquaredError float64

	for _, pred := range predictions {
		if pred.WasCorrect {
			correctPredictions++
		}

		totalUncertainty += pred.TotalUncertainty

		actual := 0.0
		if pred.WasCorrect {
			actual = 1.0
		}
		error := pred.WinProbability - actual
		totalSquaredError += error * error
	}

	accuracy := float64(correctPredictions) / float64(len(predictions))
	avgUncertainty := totalUncertainty / float64(len(predictions))
	calibrationError := totalSquaredError / float64(len(predictions))

	// Expected accuracy based on uncertainty level
	expectedAccuracy := 1.0 - avgUncertainty

	return BucketMetrics{
		BucketName:       bucketName,
		PredictionCount:  len(predictions),
		Accuracy:         accuracy,
		AvgUncertainty:   avgUncertainty,
		CalibrationError: calibrationError,
		ExpectedAccuracy: expectedAccuracy,
	}
}

// GetUncertaintyMetrics returns the current uncertainty metrics
func (mus *ModelUncertaintyService) GetUncertaintyMetrics() UncertaintyMetrics {
	return mus.uncertaintyMetrics
}

// GetCalibrationCurve returns the current calibration curve
func (mus *ModelUncertaintyService) GetCalibrationCurve() []CalibrationPoint {
	return mus.calibrationCurve
}
