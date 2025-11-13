package services

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

var (
	errorAnalysisInstance *ErrorAnalysisService
	errorAnalysisOnce     sync.Once
)

// ErrorAnalysisService tracks and analyzes prediction accuracy
type ErrorAnalysisService struct {
	dataDir          string
	records          []*models.PredictionAccuracyRecord
	summary          *models.AccuracySummary
	featureImportance map[string]*models.FeatureImportance
	errorPatterns    []*models.ErrorPattern
	mu               sync.RWMutex
}

// InitializeErrorAnalysis creates the error analysis service
func InitializeErrorAnalysis() error {
	var initErr error
	errorAnalysisOnce.Do(func() {
		dataDir := "data/accuracy"
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create accuracy directory: %w", err)
			return
		}

		errorAnalysisInstance = &ErrorAnalysisService{
			dataDir:           dataDir,
			records:           make([]*models.PredictionAccuracyRecord, 0),
			featureImportance: make(map[string]*models.FeatureImportance),
			errorPatterns:     make([]*models.ErrorPattern, 0),
		}

		// Load existing records
		if err := errorAnalysisInstance.loadRecords(); err != nil {
			fmt.Printf("⚠️ Warning: Could not load existing accuracy records: %v\n", err)
		}

		// Calculate initial summary
		errorAnalysisInstance.recalculateSummary()

		fmt.Println("✅ Error Analysis Service initialized")
	})

	return initErr
}

// GetErrorAnalysisService returns the singleton instance
func GetErrorAnalysisService() *ErrorAnalysisService {
	return errorAnalysisInstance
}

// RecordPrediction stores a prediction for later accuracy tracking
func (eas *ErrorAnalysisService) RecordPrediction(
	gameID int,
	gameDate time.Time,
	homeTeam, awayTeam string,
	predictionResult *models.PredictionResult,
	modelPredictions map[string]*models.ModelPredictionResult,
	context *PredictionContext,
) error {
	eas.mu.Lock()
	defer eas.mu.Unlock()

	// Get predicted winner and confidence from result
	predictedWinner := predictionResult.Winner
	confidence := predictionResult.WinProbability

	// Calculate home/away probabilities
	homeWinProb := predictionResult.WinProbability
	awayWinProb := 1.0 - predictionResult.WinProbability
	if predictedWinner == awayTeam {
		homeWinProb = 1.0 - predictionResult.WinProbability
		awayWinProb = predictionResult.WinProbability
	}

	record := &models.PredictionAccuracyRecord{
		GameID:               gameID,
		GameDate:             gameDate,
		HomeTeam:             homeTeam,
		AwayTeam:             awayTeam,
		PredictedWinner:      predictedWinner,
		PredictedHomeWinProb: homeWinProb,
		PredictedAwayWinProb: awayWinProb,
		PredictedTieProb:     0.0, // NHL predictions don't include tie probability
		Confidence:           confidence,
		ModelPredictions:     modelPredictions,
		PredictionTime:       time.Now(),
		Season:               GetCurrentSeason(),
	}

	// Add context information
	if context != nil {
		record.IsPlayoffGame = context.IsPlayoffs
		record.IsRivalryGame = context.IsRivalryGame
		record.IsDivisionGame = context.IsDivisionGame
		record.HomeBackToBack = context.RestDaysHome == 0
		record.AwayBackToBack = context.RestDaysAway == 0
	}

	// Save prediction record
	if err := eas.savePredictionRecord(record); err != nil {
		return fmt.Errorf("failed to save prediction record: %w", err)
	}

	fmt.Printf("📊 Recorded prediction for game %d: %s to win (%.1f%% confidence)\n",
		gameID, predictedWinner, confidence*100)

	return nil
}

// UpdatePredictionWithResult updates a prediction record with the actual game result
func (eas *ErrorAnalysisService) UpdatePredictionWithResult(
	gameID int,
	homeScore, awayScore int,
	gameType string, // "regulation", "overtime", "shootout"
) error {
	eas.mu.Lock()
	defer eas.mu.Unlock()

	// Load the prediction record
	record, err := eas.loadPredictionRecord(gameID)
	if err != nil {
		return fmt.Errorf("failed to load prediction record: %w", err)
	}

	// Determine actual winner
	actualWinner := record.HomeTeam
	if awayScore > homeScore {
		actualWinner = record.AwayTeam
	}

	// Update record with results
	record.ActualWinner = actualWinner
	record.HomeScore = homeScore
	record.AwayScore = awayScore
	record.GameType = gameType
	record.IsCorrect = (record.PredictedWinner == actualWinner)
	record.WinningMargin = int(math.Abs(float64(homeScore - awayScore)))
	record.TotalGoals = homeScore + awayScore

	// Calculate game characteristics
	record.IsUpset = eas.isUpset(record)
	record.IsBlowout = record.WinningMargin > 3
	record.IsCloseGame = record.WinningMargin == 1
	record.IsHighScoring = record.TotalGoals > 7
	record.IsLowScoring = record.TotalGoals < 4

	// Calculate prediction error (Brier score component)
	actualHomeWin := 0.0
	if record.ActualWinner == record.HomeTeam {
		actualHomeWin = 1.0
	}
	record.PredictionError = math.Abs(record.PredictedHomeWinProb - actualHomeWin)

	// Calculate calibration score
	record.CalibrationScore = eas.calculateCalibration(record)

	// Update model-specific results
	for modelName, modelPred := range record.ModelPredictions {
		modelPred.IsCorrect = (modelPred.PredictedWinner == actualWinner)
		
		// Calculate model prediction error
		modelActualProb := modelPred.AwayWinProb
		if actualWinner == record.HomeTeam {
			modelActualProb = modelPred.HomeWinProb
		}
		modelPred.PredictionError = math.Abs(modelActualProb - 1.0)
		
		// Determine if this model helped or hurt the ensemble
		if modelPred.IsCorrect == record.IsCorrect {
			modelPred.ContributionSign = 0 // Neutral
		} else if modelPred.IsCorrect {
			modelPred.ContributionSign = 1 // Would have helped
		} else {
			modelPred.ContributionSign = -1 // Hurt the prediction
		}
		
		record.ModelPredictions[modelName] = modelPred
	}

	// Save updated record
	if err := eas.saveAccuracyRecord(record); err != nil {
		return fmt.Errorf("failed to save accuracy record: %w", err)
	}

	// Add to in-memory records
	eas.records = append(eas.records, record)

	// Recalculate summary statistics
	eas.recalculateSummary()

	// Analyze for error patterns
	eas.analyzeErrorPatterns()

	status := "✅ CORRECT"
	if !record.IsCorrect {
		status = "❌ INCORRECT"
	}
	fmt.Printf("📊 %s - Game %d: Predicted %s, Actual %s (%d-%d)\n",
		status, gameID, record.PredictedWinner, actualWinner, homeScore, awayScore)

	return nil
}

// isUpset determines if the result was an upset (underdog won)
func (eas *ErrorAnalysisService) isUpset(record *models.PredictionAccuracyRecord) bool {
	// An upset is when the team with lower win probability wins
	if record.ActualWinner == record.HomeTeam {
		return record.PredictedHomeWinProb < record.PredictedAwayWinProb
	}
	return record.PredictedAwayWinProb < record.PredictedHomeWinProb
}

// calculateCalibration measures how well-calibrated the prediction was
func (eas *ErrorAnalysisService) calculateCalibration(record *models.PredictionAccuracyRecord) float64 {
	// Perfect calibration = 1.0, poor calibration = 0.0
	// If we predicted 70% and the team won, calibration is good
	// If we predicted 70% and the team lost, calibration is poor

	winProb := record.PredictedHomeWinProb
	if record.PredictedWinner == record.AwayTeam {
		winProb = record.PredictedAwayWinProb
	}

	if record.IsCorrect {
		// Correct prediction: calibration = confidence
		return winProb
	}
	// Incorrect prediction: calibration = (1 - confidence)
	return 1.0 - winProb
}

// recalculateSummary recalculates the accuracy summary from all records
func (eas *ErrorAnalysisService) recalculateSummary() {
	if len(eas.records) == 0 {
		eas.summary = &models.AccuracySummary{
			ModelAccuracies: make(map[string]float64),
			LastUpdate:      time.Now(),
		}
		return
	}

	summary := &models.AccuracySummary{
		TotalPredictions:  len(eas.records),
		ModelAccuracies:   make(map[string]float64),
		LastUpdate:        time.Now(),
	}

	// Count correct predictions and calculate metrics
	var totalConfidence, totalError, totalBrier, totalLogLoss float64
	var regulationCorrect, regulationTotal int
	var overtimeCorrect, overtimeTotal int
	var shootoutCorrect, shootoutTotal int
	var upsetsPredicted, upsetsActual, upsetsCaught int
	var blowoutCorrect, blowoutTotal int
	var closeCorrect, closeTotal int
	var highScoringCorrect, highScoringTotal int
	var lowScoringCorrect, lowScoringTotal int
	var rivalryCorrect, rivalryTotal int
	var divisionCorrect, divisionTotal int
	var backToBackCorrect, backToBackTotal int
	var playoffCorrect, playoffTotal int
	var highConfCorrect, highConfTotal int
	var medConfCorrect, medConfTotal int
	var lowConfCorrect, lowConfTotal int

	modelCorrect := make(map[string]int)
	modelTotal := make(map[string]int)

	for _, record := range eas.records {
		if record.IsCorrect {
			summary.CorrectPredictions++
		} else {
			summary.IncorrectPredictions++
		}

		totalConfidence += record.Confidence
		totalError += record.PredictionError

		// Brier score contribution
		actualHomeWin := 0.0
		if record.ActualWinner == record.HomeTeam {
			actualHomeWin = 1.0
		}
		brierContribution := math.Pow(record.PredictedHomeWinProb-actualHomeWin, 2)
		totalBrier += brierContribution

		// Log loss contribution
		predictedProb := record.PredictedHomeWinProb
		if record.ActualWinner == record.AwayTeam {
			predictedProb = record.PredictedAwayWinProb
		}
		if predictedProb > 0 {
			totalLogLoss += -math.Log(predictedProb)
		}

		// Game type breakdown
		switch record.GameType {
		case "regulation":
			regulationTotal++
			if record.IsCorrect {
				regulationCorrect++
			}
		case "overtime":
			overtimeTotal++
			if record.IsCorrect {
				overtimeCorrect++
			}
		case "shootout":
			shootoutTotal++
			if record.IsCorrect {
				shootoutCorrect++
			}
		}

		// Upsets
		if record.IsUpset {
			upsetsActual++
			if record.IsCorrect {
				upsetsCaught++
			}
		}
		if record.PredictedWinner != record.HomeTeam && record.PredictedHomeWinProb > 0.5 {
			upsetsPredicted++
		}

		// Game characteristics
		if record.IsBlowout {
			blowoutTotal++
			if record.IsCorrect {
				blowoutCorrect++
			}
		}
		if record.IsCloseGame {
			closeTotal++
			if record.IsCorrect {
				closeCorrect++
			}
		}
		if record.IsHighScoring {
			highScoringTotal++
			if record.IsCorrect {
				highScoringCorrect++
			}
		}
		if record.IsLowScoring {
			lowScoringTotal++
			if record.IsCorrect {
				lowScoringCorrect++
			}
		}
		if record.IsRivalryGame {
			rivalryTotal++
			if record.IsCorrect {
				rivalryCorrect++
			}
		}
		if record.IsDivisionGame {
			divisionTotal++
			if record.IsCorrect {
				divisionCorrect++
			}
		}
		if record.HomeBackToBack || record.AwayBackToBack {
			backToBackTotal++
			if record.IsCorrect {
				backToBackCorrect++
			}
		}
		if record.IsPlayoffGame {
			playoffTotal++
			if record.IsCorrect {
				playoffCorrect++
			}
		}

		// Confidence brackets
		if record.Confidence > 0.70 {
			highConfTotal++
			if record.IsCorrect {
				highConfCorrect++
			}
		} else if record.Confidence > 0.50 {
			medConfTotal++
			if record.IsCorrect {
				medConfCorrect++
			}
		} else {
			lowConfTotal++
			if record.IsCorrect {
				lowConfCorrect++
			}
		}

		// Model-specific tracking
		for modelName, modelPred := range record.ModelPredictions {
			modelTotal[modelName]++
			if modelPred.IsCorrect {
				modelCorrect[modelName]++
			}
		}
	}

	// Calculate percentages
	summary.OverallAccuracy = float64(summary.CorrectPredictions) / float64(summary.TotalPredictions)
	summary.AverageConfidence = totalConfidence / float64(summary.TotalPredictions)
	summary.AveragePredictionError = totalError / float64(summary.TotalPredictions)
	summary.BrierScore = totalBrier / float64(summary.TotalPredictions)
	summary.LogLoss = totalLogLoss / float64(summary.TotalPredictions)

	if regulationTotal > 0 {
		summary.RegulationAccuracy = float64(regulationCorrect) / float64(regulationTotal)
	}
	if overtimeTotal > 0 {
		summary.OvertimeAccuracy = float64(overtimeCorrect) / float64(overtimeTotal)
	}
	if shootoutTotal > 0 {
		summary.ShootoutAccuracy = float64(shootoutCorrect) / float64(shootoutTotal)
	}
	if upsetsPredicted > 0 {
		summary.UpsetPredictionRate = float64(upsetsPredicted) / float64(summary.TotalPredictions)
	}
	if upsetsActual > 0 {
		summary.UpsetCatchRate = float64(upsetsCaught) / float64(upsetsActual)
	}
	if blowoutTotal > 0 {
		summary.BlowoutAccuracy = float64(blowoutCorrect) / float64(blowoutTotal)
	}
	if closeTotal > 0 {
		summary.CloseGameAccuracy = float64(closeCorrect) / float64(closeTotal)
	}
	if highScoringTotal > 0 {
		summary.HighScoringAccuracy = float64(highScoringCorrect) / float64(highScoringTotal)
	}
	if lowScoringTotal > 0 {
		summary.LowScoringAccuracy = float64(lowScoringCorrect) / float64(lowScoringTotal)
	}
	if rivalryTotal > 0 {
		summary.RivalryGameAccuracy = float64(rivalryCorrect) / float64(rivalryTotal)
	}
	if divisionTotal > 0 {
		summary.DivisionGameAccuracy = float64(divisionCorrect) / float64(divisionTotal)
	}
	if backToBackTotal > 0 {
		summary.BackToBackAccuracy = float64(backToBackCorrect) / float64(backToBackTotal)
	}
	if playoffTotal > 0 {
		summary.PlayoffGameAccuracy = float64(playoffCorrect) / float64(playoffTotal)
	}
	if highConfTotal > 0 {
		summary.HighConfidenceAccuracy = float64(highConfCorrect) / float64(highConfTotal)
	}
	if medConfTotal > 0 {
		summary.MediumConfidenceAccuracy = float64(medConfCorrect) / float64(medConfTotal)
	}
	if lowConfTotal > 0 {
		summary.LowConfidenceAccuracy = float64(lowConfCorrect) / float64(lowConfTotal)
	}

	// Recent accuracy
	summary.Last10GamesAccuracy = eas.calculateRecentAccuracy(10)
	summary.Last30GamesAccuracy = eas.calculateRecentAccuracy(30)
	summary.Last50GamesAccuracy = eas.calculateRecentAccuracy(50)

	// Model accuracies
	for modelName, correct := range modelCorrect {
		if modelTotal[modelName] > 0 {
			summary.ModelAccuracies[modelName] = float64(correct) / float64(modelTotal[modelName])
		}
	}

	// Set dates
	if len(eas.records) > 0 {
		summary.StartDate = eas.records[0].GameDate
		summary.EndDate = eas.records[len(eas.records)-1].GameDate
		summary.Season = eas.records[0].Season
	}

	eas.summary = summary

	// Save summary
	eas.saveSummary()
}

// calculateRecentAccuracy calculates accuracy for the last N games
func (eas *ErrorAnalysisService) calculateRecentAccuracy(n int) float64 {
	if len(eas.records) == 0 {
		return 0.0
	}

	start := len(eas.records) - n
	if start < 0 {
		start = 0
	}

	recentRecords := eas.records[start:]
	correct := 0
	for _, record := range recentRecords {
		if record.IsCorrect {
			correct++
		}
	}

	return float64(correct) / float64(len(recentRecords))
}

// analyzeErrorPatterns identifies common prediction failure patterns
func (eas *ErrorAnalysisService) analyzeErrorPatterns() {
	patterns := make(map[string]*models.ErrorPattern)

	for _, record := range eas.records {
		if record.IsCorrect {
			continue // Only analyze errors
		}

		// Pattern: Upset misses
		if record.IsUpset {
			key := "upset_miss"
			if patterns[key] == nil {
				patterns[key] = &models.ErrorPattern{
					PatternType: "Upset Miss",
					CommonCharacteristics: []string{
						"Underdog won",
						"High confidence in favorite",
					},
					PotentialFix: "Improve underdog detection, add upset indicators",
					Severity:     "high",
					ExampleGameIDs: []int{},
				}
			}
			patterns[key].Frequency++
			if len(patterns[key].ExampleGameIDs) < 5 {
				patterns[key].ExampleGameIDs = append(patterns[key].ExampleGameIDs, record.GameID)
			}
			patterns[key].LastSeen = record.GameDate
		}

		// Pattern: Close game losses
		if record.IsCloseGame {
			key := "close_game_loss"
			if patterns[key] == nil {
				patterns[key] = &models.ErrorPattern{
					PatternType: "Close Game Loss",
					CommonCharacteristics: []string{
						"1-goal game",
						"Could have gone either way",
					},
					PotentialFix: "Add clutch performance metrics, late-game statistics",
					Severity:     "medium",
					ExampleGameIDs: []int{},
				}
			}
			patterns[key].Frequency++
			if len(patterns[key].ExampleGameIDs) < 5 {
				patterns[key].ExampleGameIDs = append(patterns[key].ExampleGameIDs, record.GameID)
			}
			patterns[key].LastSeen = record.GameDate
		}

		// Pattern: Blowout direction wrong
		if record.IsBlowout {
			key := "blowout_wrong"
			if patterns[key] == nil {
				patterns[key] = &models.ErrorPattern{
					PatternType: "Blowout Direction Wrong",
					CommonCharacteristics: []string{
						"Margin > 3 goals",
						"Predicted wrong winner in lopsided game",
					},
					PotentialFix: "Improve team strength differential analysis",
					Severity:     "high",
					ExampleGameIDs: []int{},
				}
			}
			patterns[key].Frequency++
			if len(patterns[key].ExampleGameIDs) < 5 {
				patterns[key].ExampleGameIDs = append(patterns[key].ExampleGameIDs, record.GameID)
			}
			patterns[key].LastSeen = record.GameDate
		}

		// Pattern: Overtime/Shootout losses
		if record.GameType != "regulation" {
			key := "overtime_shootout_loss"
			if patterns[key] == nil {
				patterns[key] = &models.ErrorPattern{
					PatternType: "Overtime/Shootout Loss",
					CommonCharacteristics: []string{
						"Game went to OT/SO",
						"Harder to predict due to randomness",
					},
					PotentialFix: "Add OT/SO-specific features, shootout percentage",
					Severity:     "low",
					ExampleGameIDs: []int{},
				}
			}
			patterns[key].Frequency++
			if len(patterns[key].ExampleGameIDs) < 5 {
				patterns[key].ExampleGameIDs = append(patterns[key].ExampleGameIDs, record.GameID)
			}
			patterns[key].LastSeen = record.GameDate
		}

		// Pattern: Back-to-back fatigue underestimation
		if record.HomeBackToBack || record.AwayBackToBack {
			key := "back_to_back_fatigue"
			if patterns[key] == nil {
				patterns[key] = &models.ErrorPattern{
					PatternType: "Back-to-Back Fatigue",
					CommonCharacteristics: []string{
						"Team on back-to-back",
						"Fatigue impact underestimated",
					},
					PotentialFix: "Increase weight of rest/fatigue factors",
					Severity:     "medium",
					ExampleGameIDs: []int{},
				}
			}
			patterns[key].Frequency++
			if len(patterns[key].ExampleGameIDs) < 5 {
				patterns[key].ExampleGameIDs = append(patterns[key].ExampleGameIDs, record.GameID)
			}
			patterns[key].LastSeen = record.GameDate
		}
	}

	// Convert map to slice and calculate percentages
	totalErrors := len(eas.records) - eas.summary.CorrectPredictions
	patternSlice := make([]*models.ErrorPattern, 0, len(patterns))
	for _, pattern := range patterns {
		if totalErrors > 0 {
			pattern.PercentageOfErrors = float64(pattern.Frequency) / float64(totalErrors) * 100.0
		}
		patternSlice = append(patternSlice, pattern)
	}

	// Sort by frequency
	sort.Slice(patternSlice, func(i, j int) bool {
		return patternSlice[i].Frequency > patternSlice[j].Frequency
	})

	eas.errorPatterns = patternSlice
}

// GetSummary returns the current accuracy summary
func (eas *ErrorAnalysisService) GetSummary() *models.AccuracySummary {
	eas.mu.RLock()
	defer eas.mu.RUnlock()
	return eas.summary
}

// GetErrorPatterns returns identified error patterns
func (eas *ErrorAnalysisService) GetErrorPatterns() []*models.ErrorPattern {
	eas.mu.RLock()
	defer eas.mu.RUnlock()
	return eas.errorPatterns
}

// GetRecentRecords returns the N most recent prediction records
func (eas *ErrorAnalysisService) GetRecentRecords(n int) []*models.PredictionAccuracyRecord {
	eas.mu.RLock()
	defer eas.mu.RUnlock()

	if len(eas.records) == 0 {
		return []*models.PredictionAccuracyRecord{}
	}

	start := len(eas.records) - n
	if start < 0 {
		start = 0
	}

	return eas.records[start:]
}

// File operations

func (eas *ErrorAnalysisService) savePredictionRecord(record *models.PredictionAccuracyRecord) error {
	filename := filepath.Join(eas.dataDir, fmt.Sprintf("prediction_%d.json", record.GameID))
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (eas *ErrorAnalysisService) loadPredictionRecord(gameID int) (*models.PredictionAccuracyRecord, error) {
	filename := filepath.Join(eas.dataDir, fmt.Sprintf("prediction_%d.json", gameID))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var record models.PredictionAccuracyRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}

	return &record, nil
}

func (eas *ErrorAnalysisService) saveAccuracyRecord(record *models.PredictionAccuracyRecord) error {
	filename := filepath.Join(eas.dataDir, fmt.Sprintf("accuracy_%d.json", record.GameID))
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (eas *ErrorAnalysisService) loadRecords() error {
	files, err := filepath.Glob(filepath.Join(eas.dataDir, "accuracy_*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var record models.PredictionAccuracyRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}

		eas.records = append(eas.records, &record)
	}

	// Sort by game date
	sort.Slice(eas.records, func(i, j int) bool {
		return eas.records[i].GameDate.Before(eas.records[j].GameDate)
	})

	fmt.Printf("📊 Loaded %d accuracy records\n", len(eas.records))
	return nil
}

func (eas *ErrorAnalysisService) saveSummary() error {
	if eas.summary == nil {
		return nil
	}

	filename := filepath.Join(eas.dataDir, "summary.json")
	data, err := json.MarshalIndent(eas.summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

