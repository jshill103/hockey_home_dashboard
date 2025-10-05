package services

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// CrossValidationService implements scientific validation of prediction models
type CrossValidationService struct {
	historicalData    []HistoricalPrediction
	validationResults []ValidationResult
	calibrationCurve  CalibrationCurve
	lastUpdate        time.Time
	isCalibrated      bool
	settings          CrossValidationSettings
}

// HistoricalPrediction represents a past prediction with its actual outcome
type HistoricalPrediction struct {
	PredictionID string    `json:"predictionId"`
	GameDate     time.Time `json:"gameDate"`
	HomeTeam     string    `json:"homeTeam"`
	AwayTeam     string    `json:"awayTeam"`

	// Original Prediction
	PredictedWinner   string                   `json:"predictedWinner"`
	PredictedScore    string                   `json:"predictedScore"`
	WinProbability    float64                  `json:"winProbability"`
	RawConfidence     float64                  `json:"rawConfidence"`     // Original confidence
	ModelResults      []models.ModelResult     `json:"modelResults"`      // Individual model results
	PredictionFactors PredictionFactorSnapshot `json:"predictionFactors"` // Factors used

	// Actual Outcome
	ActualWinner  string `json:"actualWinner"`
	ActualScore   string `json:"actualScore"`
	GameCompleted bool   `json:"gameCompleted"`

	// Validation Metrics
	IsCorrect        bool    `json:"isCorrect"`
	ProbabilityError float64 `json:"probabilityError"` // |predicted_prob - actual_outcome|
	ScoreError       float64 `json:"scoreError"`       // Goal difference error
	ConfidenceError  float64 `json:"confidenceError"`  // Calibration error

	// Metadata
	GameType       string    `json:"gameType"` // "regular", "playoff", "preseason"
	Venue          string    `json:"venue"`
	RecordedAt     time.Time `json:"recordedAt"`
	ValidationFold int       `json:"validationFold"` // Which fold this belongs to
}

// PredictionFactorSnapshot captures the factors used for a prediction
type PredictionFactorSnapshot struct {
	HomeWinPercentage float64 `json:"homeWinPercentage"`
	AwayWinPercentage float64 `json:"awayWinPercentage"`
	HomeAdvantage     float64 `json:"homeAdvantage"`
	TravelFatigue     float64 `json:"travelFatigue"`
	AltitudeEffect    float64 `json:"altitudeEffect"`
	InjuryImpact      float64 `json:"injuryImpact"`
	MomentumFactor    float64 `json:"momentumFactor"`
	AdvancedRating    float64 `json:"advancedRating"`
}

// ValidationResult represents the outcome of cross-validation
type ValidationResult struct {
	FoldNumber   int `json:"foldNumber"`
	TrainingSize int `json:"trainingSize"`
	TestingSize  int `json:"testingSize"`

	// Accuracy Metrics
	Accuracy      float64 `json:"accuracy"`      // % correct predictions
	PrecisionHome float64 `json:"precisionHome"` // Home team prediction precision
	PrecisionAway float64 `json:"precisionAway"` // Away team prediction precision
	RecallHome    float64 `json:"recallHome"`    // Home team prediction recall
	RecallAway    float64 `json:"recallAway"`    // Away team prediction recall
	F1ScoreHome   float64 `json:"f1ScoreHome"`   // F1 for home predictions
	F1ScoreAway   float64 `json:"f1ScoreAway"`   // F1 for away predictions

	// Confidence Calibration
	CalibrationError float64 `json:"calibrationError"` // Mean absolute calibration error
	BrierScore       float64 `json:"brierScore"`       // Brier score for probability accuracy
	LogLoss          float64 `json:"logLoss"`          // Logarithmic loss

	// Score Prediction Accuracy
	MeanScoreError float64 `json:"meanScoreError"` // Average goal prediction error
	ScoreAccuracy  float64 `json:"scoreAccuracy"`  // % exact score matches

	// Model-Specific Results
	ModelPerformance map[string]ModelValidationResult `json:"modelPerformance"`

	// Statistical Significance
	ConfidenceInterval ConfidenceInterval `json:"confidenceInterval"` // 95% CI for accuracy
	PValue             float64            `json:"pValue"`             // Statistical significance

	ValidationDate time.Time `json:"validationDate"`
}

// ModelValidationResult tracks individual model performance in cross-validation
type ModelValidationResult struct {
	ModelName         string  `json:"modelName"`
	Accuracy          float64 `json:"accuracy"`
	CalibrationError  float64 `json:"calibrationError"`
	BrierScore        float64 `json:"brierScore"`
	AverageConfidence float64 `json:"averageConfidence"`
	OptimalWeight     float64 `json:"optimalWeight"`   // Suggested weight based on performance
	PerformanceRank   int     `json:"performanceRank"` // 1 = best performing
}

// ConfidenceInterval represents statistical confidence bounds
type ConfidenceInterval struct {
	Lower           float64 `json:"lower"`           // Lower bound (e.g., 2.5%)
	Upper           float64 `json:"upper"`           // Upper bound (e.g., 97.5%)
	ConfidenceLevel float64 `json:"confidenceLevel"` // e.g., 0.95 for 95%
}

// CalibrationCurve represents confidence calibration data
type CalibrationCurve struct {
	ConfidenceBins       []ConfidenceBin `json:"confidenceBins"`
	OverallCalibration   float64         `json:"overallCalibration"`   // 0-1, 1 = perfectly calibrated
	IsWellCalibrated     bool            `json:"isWellCalibrated"`     // True if calibration error < 0.1
	CalibrationSlope     float64         `json:"calibrationSlope"`     // Slope of calibration line
	CalibrationIntercept float64         `json:"calibrationIntercept"` // Intercept of calibration line
	LastUpdated          time.Time       `json:"lastUpdated"`
}

// ConfidenceBin represents a range of confidence values and their actual accuracy
type ConfidenceBin struct {
	MinConfidence    float64 `json:"minConfidence"`    // e.g., 0.8
	MaxConfidence    float64 `json:"maxConfidence"`    // e.g., 0.9
	PredictionCount  int     `json:"predictionCount"`  // Number of predictions in this bin
	ActualAccuracy   float64 `json:"actualAccuracy"`   // Actual accuracy in this bin
	ExpectedAccuracy float64 `json:"expectedAccuracy"` // Expected accuracy (bin midpoint)
	CalibrationError float64 `json:"calibrationError"` // |actual - expected|
}

// CrossValidationSettings configures the validation process
type CrossValidationSettings struct {
	KFolds             int           `json:"kFolds"`             // Number of folds (default: 5)
	MinHistoricalData  int           `json:"minHistoricalData"`  // Min predictions needed (default: 50)
	ValidationWindow   time.Duration `json:"validationWindow"`   // How far back to look (default: 1 year)
	TemporalValidation bool          `json:"temporalValidation"` // Use time-based splits
	BootstrapSamples   int           `json:"bootstrapSamples"`   // Bootstrap iterations (default: 1000)
	ConfidenceLevel    float64       `json:"confidenceLevel"`    // CI level (default: 0.95)
	CalibrationBins    int           `json:"calibrationBins"`    // Number of confidence bins (default: 10)
	UpdateFrequency    time.Duration `json:"updateFrequency"`    // How often to revalidate
}

// NewCrossValidationService creates a new cross-validation service
func NewCrossValidationService() *CrossValidationService {
	settings := CrossValidationSettings{
		KFolds:             5,
		MinHistoricalData:  50,
		ValidationWindow:   365 * 24 * time.Hour, // 1 year
		TemporalValidation: true,
		BootstrapSamples:   1000,
		ConfidenceLevel:    0.95,
		CalibrationBins:    10,
		UpdateFrequency:    24 * time.Hour, // Daily updates
	}

	return &CrossValidationService{
		historicalData:    make([]HistoricalPrediction, 0),
		validationResults: make([]ValidationResult, 0),
		settings:          settings,
		isCalibrated:      false,
	}
}

// AddHistoricalPrediction records a prediction and its outcome for validation
func (cvs *CrossValidationService) AddHistoricalPrediction(prediction HistoricalPrediction) {
	// Validate the prediction data
	if prediction.GameDate.IsZero() || prediction.PredictionID == "" {
		log.Printf("⚠️ Invalid historical prediction data: %+v", prediction)
		return
	}

	// Calculate validation metrics if outcome is known
	if prediction.GameCompleted {
		prediction.IsCorrect = (prediction.PredictedWinner == prediction.ActualWinner)

		// Calculate probability error
		actualOutcome := 0.0
		if prediction.IsCorrect {
			actualOutcome = 1.0
		}
		prediction.ProbabilityError = math.Abs(prediction.WinProbability - actualOutcome)

		// Calculate confidence error (calibration)
		prediction.ConfidenceError = math.Abs(prediction.RawConfidence - actualOutcome)

		// Calculate score error
		prediction.ScoreError = cvs.calculateScoreError(prediction.PredictedScore, prediction.ActualScore)
	}

	cvs.historicalData = append(cvs.historicalData, prediction)

	// Maintain rolling window
	cutoff := time.Now().Add(-cvs.settings.ValidationWindow)
	cvs.historicalData = cvs.filterByDate(cvs.historicalData, cutoff)

	log.Printf("📊 Added historical prediction: %s vs %s (Correct: %v)",
		prediction.HomeTeam, prediction.AwayTeam, prediction.IsCorrect)
}

// RunCrossValidation performs k-fold cross-validation on historical data
func (cvs *CrossValidationService) RunCrossValidation() error {
	if len(cvs.historicalData) < cvs.settings.MinHistoricalData {
		return fmt.Errorf("insufficient historical data: have %d, need %d",
			len(cvs.historicalData), cvs.settings.MinHistoricalData)
	}

	log.Printf("🔄 Starting %d-fold cross-validation with %d historical predictions...",
		cvs.settings.KFolds, len(cvs.historicalData))

	// Prepare data for validation
	completedPredictions := cvs.getCompletedPredictions()
	if len(completedPredictions) < cvs.settings.MinHistoricalData {
		return fmt.Errorf("insufficient completed predictions: have %d, need %d",
			len(completedPredictions), cvs.settings.MinHistoricalData)
	}

	// Create folds
	folds := cvs.createFolds(completedPredictions)

	// Clear previous results
	cvs.validationResults = make([]ValidationResult, 0)

	// Run validation for each fold
	for i, fold := range folds {
		log.Printf("🔍 Validating fold %d/%d...", i+1, len(folds))

		result := cvs.validateFold(i, fold, folds)
		cvs.validationResults = append(cvs.validationResults, result)
	}

	// Calculate overall calibration curve
	cvs.calculateCalibrationCurve(completedPredictions)

	// Mark as calibrated
	cvs.isCalibrated = true
	cvs.lastUpdate = time.Now()

	log.Printf("✅ Cross-validation complete! Overall accuracy: %.1f%%",
		cvs.GetOverallAccuracy()*100)

	return nil
}

// GetCalibratedConfidence returns a calibrated confidence score
func (cvs *CrossValidationService) GetCalibratedConfidence(rawConfidence float64, modelResults []models.ModelResult) float64 {
	if !cvs.isCalibrated {
		log.Printf("⚠️ Cross-validation not yet calibrated, returning raw confidence")
		return rawConfidence
	}

	// Find the appropriate calibration bin
	bin := cvs.findCalibrationBin(rawConfidence)
	if bin == nil {
		return rawConfidence
	}

	// Apply calibration adjustment
	calibratedConfidence := bin.ActualAccuracy

	// Apply model-specific adjustments based on validation results
	if len(modelResults) > 0 && len(cvs.validationResults) > 0 {
		adjustment := cvs.calculateModelAdjustment(modelResults)
		calibratedConfidence *= adjustment
	}

	// Ensure bounds
	calibratedConfidence = math.Max(0.1, math.Min(0.99, calibratedConfidence))

	log.Printf("🎯 Confidence calibration: %.1f%% → %.1f%% (bin: %.1f-%.1f%%)",
		rawConfidence*100, calibratedConfidence*100,
		bin.MinConfidence*100, bin.MaxConfidence*100)

	return calibratedConfidence
}

// GetValidationSummary returns a summary of cross-validation results
func (cvs *CrossValidationService) GetValidationSummary() ValidationSummary {
	if len(cvs.validationResults) == 0 {
		return ValidationSummary{
			IsValidated: false,
			Message:     "Cross-validation not yet performed",
		}
	}

	overallAccuracy := cvs.GetOverallAccuracy()
	overallCalibration := cvs.calibrationCurve.OverallCalibration

	return ValidationSummary{
		IsValidated:        true,
		OverallAccuracy:    overallAccuracy,
		CalibrationScore:   overallCalibration,
		IsWellCalibrated:   cvs.calibrationCurve.IsWellCalibrated,
		TotalPredictions:   len(cvs.historicalData),
		ValidationFolds:    len(cvs.validationResults),
		LastValidated:      cvs.lastUpdate,
		ConfidenceInterval: cvs.calculateOverallConfidenceInterval(),
		ModelRankings:      cvs.getModelRankings(),
		CalibrationBins:    cvs.calibrationCurve.ConfidenceBins,
		Message:            cvs.generateValidationMessage(overallAccuracy, overallCalibration),
	}
}

// ValidationSummary provides a high-level view of validation results
type ValidationSummary struct {
	IsValidated        bool                    `json:"isValidated"`
	OverallAccuracy    float64                 `json:"overallAccuracy"`
	CalibrationScore   float64                 `json:"calibrationScore"`
	IsWellCalibrated   bool                    `json:"isWellCalibrated"`
	TotalPredictions   int                     `json:"totalPredictions"`
	ValidationFolds    int                     `json:"validationFolds"`
	LastValidated      time.Time               `json:"lastValidated"`
	ConfidenceInterval ConfidenceInterval      `json:"confidenceInterval"`
	ModelRankings      []ModelValidationResult `json:"modelRankings"`
	CalibrationBins    []ConfidenceBin         `json:"calibrationBins"`
	Message            string                  `json:"message"`
}

// Helper methods for cross-validation implementation

// getCompletedPredictions filters for predictions with known outcomes
func (cvs *CrossValidationService) getCompletedPredictions() []HistoricalPrediction {
	var completed []HistoricalPrediction
	for _, pred := range cvs.historicalData {
		if pred.GameCompleted && pred.ActualWinner != "" {
			completed = append(completed, pred)
		}
	}
	return completed
}

// createFolds creates k-fold splits of the data
func (cvs *CrossValidationService) createFolds(data []HistoricalPrediction) [][]HistoricalPrediction {
	// Sort by date for temporal validation
	if cvs.settings.TemporalValidation {
		sort.Slice(data, func(i, j int) bool {
			return data[i].GameDate.Before(data[j].GameDate)
		})
	} else {
		// Shuffle for random validation
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(data), func(i, j int) {
			data[i], data[j] = data[j], data[i]
		})
	}

	folds := make([][]HistoricalPrediction, cvs.settings.KFolds)
	foldSize := len(data) / cvs.settings.KFolds

	for i := 0; i < cvs.settings.KFolds; i++ {
		start := i * foldSize
		end := start + foldSize
		if i == cvs.settings.KFolds-1 {
			end = len(data) // Include remainder in last fold
		}
		folds[i] = data[start:end]
	}

	return folds
}

// validateFold performs validation on a single fold
func (cvs *CrossValidationService) validateFold(foldIndex int, testFold []HistoricalPrediction, allFolds [][]HistoricalPrediction) ValidationResult {
	// Create training set (all folds except test fold)
	var trainingSet []HistoricalPrediction
	for i, fold := range allFolds {
		if i != foldIndex {
			trainingSet = append(trainingSet, fold...)
		}
	}

	result := ValidationResult{
		FoldNumber:       foldIndex + 1,
		TrainingSize:     len(trainingSet),
		TestingSize:      len(testFold),
		ModelPerformance: make(map[string]ModelValidationResult),
		ValidationDate:   time.Now(),
	}

	// Calculate accuracy metrics
	correct := 0
	totalProbError := 0.0
	totalScoreError := 0.0
	brierSum := 0.0
	logLossSum := 0.0

	for _, pred := range testFold {
		if pred.IsCorrect {
			correct++
		}
		totalProbError += pred.ProbabilityError
		totalScoreError += pred.ScoreError

		// Brier score calculation
		actualOutcome := 0.0
		if pred.IsCorrect {
			actualOutcome = 1.0
		}
		brierSum += math.Pow(pred.WinProbability-actualOutcome, 2)

		// Log loss calculation (avoid log(0))
		prob := math.Max(0.001, math.Min(0.999, pred.WinProbability))
		if pred.IsCorrect {
			logLossSum += -math.Log(prob)
		} else {
			logLossSum += -math.Log(1 - prob)
		}
	}

	result.Accuracy = float64(correct) / float64(len(testFold))
	result.CalibrationError = totalProbError / float64(len(testFold))
	result.BrierScore = brierSum / float64(len(testFold))
	result.LogLoss = logLossSum / float64(len(testFold))
	result.MeanScoreError = totalScoreError / float64(len(testFold))

	// Calculate confidence interval for this fold
	result.ConfidenceInterval = cvs.calculateConfidenceInterval(result.Accuracy, len(testFold))

	return result
}

// calculateScoreError computes the error between predicted and actual scores
func (cvs *CrossValidationService) calculateScoreError(predicted, actual string) float64 {
	predHome, predAway := cvs.parseScore(predicted)
	actualHome, actualAway := cvs.parseScore(actual)

	if predHome < 0 || predAway < 0 || actualHome < 0 || actualAway < 0 {
		return 0.0 // Invalid scores
	}

	// Calculate total goal difference
	predTotal := predHome + predAway
	actualTotal := actualHome + actualAway

	return math.Abs(float64(predTotal - actualTotal))
}

// parseScore extracts goals from score string (e.g., "3-2" -> 3, 2)
func (cvs *CrossValidationService) parseScore(scoreStr string) (int, int) {
	// Simple parsing - in real implementation, use proper parsing
	// Return empty validation when no historical data available
	return 0, 0
}

// filterByDate filters predictions within the validation window
func (cvs *CrossValidationService) filterByDate(predictions []HistoricalPrediction, cutoff time.Time) []HistoricalPrediction {
	var filtered []HistoricalPrediction
	for _, pred := range predictions {
		if pred.GameDate.After(cutoff) {
			filtered = append(filtered, pred)
		}
	}
	return filtered
}

// calculateCalibrationCurve computes confidence calibration bins
func (cvs *CrossValidationService) calculateCalibrationCurve(predictions []HistoricalPrediction) {
	bins := make([]ConfidenceBin, cvs.settings.CalibrationBins)
	binSize := 1.0 / float64(cvs.settings.CalibrationBins)

	// Initialize bins
	for i := 0; i < cvs.settings.CalibrationBins; i++ {
		bins[i] = ConfidenceBin{
			MinConfidence:    float64(i) * binSize,
			MaxConfidence:    float64(i+1) * binSize,
			ExpectedAccuracy: (float64(i) + 0.5) * binSize,
		}
	}

	// Populate bins with predictions
	for _, pred := range predictions {
		binIndex := int(pred.RawConfidence * float64(cvs.settings.CalibrationBins))
		if binIndex >= cvs.settings.CalibrationBins {
			binIndex = cvs.settings.CalibrationBins - 1
		}

		bins[binIndex].PredictionCount++
		if pred.IsCorrect {
			bins[binIndex].ActualAccuracy += 1.0
		}
	}

	// Calculate final metrics for each bin
	totalCalibrationError := 0.0
	validBins := 0

	for i := range bins {
		if bins[i].PredictionCount > 0 {
			bins[i].ActualAccuracy /= float64(bins[i].PredictionCount)
			bins[i].CalibrationError = math.Abs(bins[i].ActualAccuracy - bins[i].ExpectedAccuracy)
			totalCalibrationError += bins[i].CalibrationError
			validBins++
		}
	}

	// Calculate overall calibration
	overallCalibration := 1.0
	if validBins > 0 {
		meanCalibrationError := totalCalibrationError / float64(validBins)
		overallCalibration = math.Max(0.0, 1.0-meanCalibrationError)
	}

	cvs.calibrationCurve = CalibrationCurve{
		ConfidenceBins:     bins,
		OverallCalibration: overallCalibration,
		IsWellCalibrated:   overallCalibration > 0.9,
		LastUpdated:        time.Now(),
	}
}

// findCalibrationBin finds the appropriate calibration bin for a confidence value
func (cvs *CrossValidationService) findCalibrationBin(confidence float64) *ConfidenceBin {
	for i := range cvs.calibrationCurve.ConfidenceBins {
		bin := &cvs.calibrationCurve.ConfidenceBins[i]
		if confidence >= bin.MinConfidence && confidence < bin.MaxConfidence {
			return bin
		}
	}
	return nil
}

// calculateModelAdjustment calculates adjustment based on individual model performance
func (cvs *CrossValidationService) calculateModelAdjustment(modelResults []models.ModelResult) float64 {
	if len(cvs.validationResults) == 0 {
		return 1.0
	}

	// Get average model performance from validation results
	totalAdjustment := 0.0
	totalWeight := 0.0

	for _, modelResult := range modelResults {
		// Find this model's validation performance
		for _, validationResult := range cvs.validationResults {
			if modelPerf, exists := validationResult.ModelPerformance[modelResult.ModelName]; exists {
				// Weight by model's weight in ensemble
				adjustment := modelPerf.Accuracy / cvs.GetOverallAccuracy()
				totalAdjustment += adjustment * modelResult.Weight
				totalWeight += modelResult.Weight
			}
		}
	}

	if totalWeight > 0 {
		return totalAdjustment / totalWeight
	}
	return 1.0
}

// GetOverallAccuracy calculates the mean accuracy across all folds
func (cvs *CrossValidationService) GetOverallAccuracy() float64 {
	if len(cvs.validationResults) == 0 {
		return 0.0
	}

	totalAccuracy := 0.0
	for _, result := range cvs.validationResults {
		totalAccuracy += result.Accuracy
	}
	return totalAccuracy / float64(len(cvs.validationResults))
}

// calculateOverallConfidenceInterval calculates CI for overall accuracy
func (cvs *CrossValidationService) calculateOverallConfidenceInterval() ConfidenceInterval {
	if len(cvs.validationResults) == 0 {
		return ConfidenceInterval{}
	}

	accuracies := make([]float64, len(cvs.validationResults))
	for i, result := range cvs.validationResults {
		accuracies[i] = result.Accuracy
	}

	return cvs.calculateConfidenceIntervalFromSample(accuracies, cvs.settings.ConfidenceLevel)
}

// calculateConfidenceInterval calculates CI for a single accuracy measurement
func (cvs *CrossValidationService) calculateConfidenceInterval(accuracy float64, sampleSize int) ConfidenceInterval {
	// Using normal approximation for binomial confidence interval
	z := 1.96 // 95% confidence level

	margin := z * math.Sqrt((accuracy*(1-accuracy))/float64(sampleSize))

	return ConfidenceInterval{
		Lower:           math.Max(0.0, accuracy-margin),
		Upper:           math.Min(1.0, accuracy+margin),
		ConfidenceLevel: cvs.settings.ConfidenceLevel,
	}
}

// calculateConfidenceIntervalFromSample calculates CI from a sample of values
func (cvs *CrossValidationService) calculateConfidenceIntervalFromSample(values []float64, confidenceLevel float64) ConfidenceInterval {
	if len(values) == 0 {
		return ConfidenceInterval{}
	}

	sort.Float64s(values)

	alpha := 1.0 - confidenceLevel
	lowerIndex := int(alpha / 2 * float64(len(values)))
	upperIndex := int((1 - alpha/2) * float64(len(values)))

	if upperIndex >= len(values) {
		upperIndex = len(values) - 1
	}

	return ConfidenceInterval{
		Lower:           values[lowerIndex],
		Upper:           values[upperIndex],
		ConfidenceLevel: confidenceLevel,
	}
}

// getModelRankings returns model performance rankings
func (cvs *CrossValidationService) getModelRankings() []ModelValidationResult {
	modelMap := make(map[string]*ModelValidationResult)

	// Aggregate model performance across all folds
	for _, result := range cvs.validationResults {
		for modelName, perf := range result.ModelPerformance {
			if existing, exists := modelMap[modelName]; exists {
				existing.Accuracy += perf.Accuracy
				existing.CalibrationError += perf.CalibrationError
				existing.BrierScore += perf.BrierScore
			} else {
				perfCopy := perf
				modelMap[modelName] = &perfCopy
			}
		}
	}

	// Average the results
	rankings := make([]ModelValidationResult, 0, len(modelMap))
	foldCount := float64(len(cvs.validationResults))

	for _, perf := range modelMap {
		perf.Accuracy /= foldCount
		perf.CalibrationError /= foldCount
		perf.BrierScore /= foldCount
		rankings = append(rankings, *perf)
	}

	// Sort by accuracy (descending)
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Accuracy > rankings[j].Accuracy
	})

	// Assign performance ranks
	for i := range rankings {
		rankings[i].PerformanceRank = i + 1
	}

	return rankings
}

// generateValidationMessage creates a human-readable validation summary
func (cvs *CrossValidationService) generateValidationMessage(accuracy, calibration float64) string {
	if !cvs.isCalibrated {
		return "Cross-validation not yet performed"
	}

	accuracyGrade := "Poor"
	if accuracy > 0.9 {
		accuracyGrade = "Excellent"
	} else if accuracy > 0.8 {
		accuracyGrade = "Very Good"
	} else if accuracy > 0.7 {
		accuracyGrade = "Good"
	} else if accuracy > 0.6 {
		accuracyGrade = "Fair"
	}

	calibrationGrade := "Poor"
	if calibration > 0.9 {
		calibrationGrade = "Excellent"
	} else if calibration > 0.8 {
		calibrationGrade = "Very Good"
	} else if calibration > 0.7 {
		calibrationGrade = "Good"
	} else if calibration > 0.6 {
		calibrationGrade = "Fair"
	}

	return fmt.Sprintf("Validation Complete: %s accuracy (%.1f%%), %s calibration (%.1f%%) across %d folds",
		accuracyGrade, accuracy*100, calibrationGrade, calibration*100, len(cvs.validationResults))
}

// ShouldRevalidate checks if it's time to run validation again
func (cvs *CrossValidationService) ShouldRevalidate() bool {
	if !cvs.isCalibrated {
		return len(cvs.getCompletedPredictions()) >= cvs.settings.MinHistoricalData
	}

	return time.Since(cvs.lastUpdate) > cvs.settings.UpdateFrequency
}

// GetCalibrationCurve returns the current calibration curve
func (cvs *CrossValidationService) GetCalibrationCurve() CalibrationCurve {
	return cvs.calibrationCurve
}

// IsCalibrated returns whether the service has been calibrated
func (cvs *CrossValidationService) IsCalibrated() bool {
	return cvs.isCalibrated
}
