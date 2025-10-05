package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ModelEvaluationService handles train/test splits, batch training, and performance metrics
type ModelEvaluationService struct {
	dataDir        string
	metricsDir     string
	modelMetrics   map[string]*models.ModelEvaluationMetrics
	predictions    []models.PredictionOutcome
	completedGames []models.CompletedGame
	mutex          sync.RWMutex

	// Models to evaluate
	neuralNet    *NeuralNetworkModel
	eloModel     *EloRatingModel
	poissonModel *PoissonRegressionModel

	// Batch training
	batchSize    int
	pendingBatch []models.CompletedGame
	batchMutex   sync.Mutex
}

// NewModelEvaluationService creates a new evaluation service
func NewModelEvaluationService(neuralNet *NeuralNetworkModel, elo *EloRatingModel, poisson *PoissonRegressionModel) *ModelEvaluationService {
	service := &ModelEvaluationService{
		dataDir:      "data/evaluation",
		metricsDir:   "data/metrics",
		modelMetrics: make(map[string]*models.ModelEvaluationMetrics),
		predictions:  []models.PredictionOutcome{},
		neuralNet:    neuralNet,
		eloModel:     elo,
		poissonModel: poisson,
		batchSize:    10, // Train on batches of 10 games
		pendingBatch: []models.CompletedGame{},
	}

	// Create directories
	os.MkdirAll(service.dataDir, 0755)
	os.MkdirAll(service.metricsDir, 0755)

	// Load existing metrics
	if err := service.loadMetrics(); err != nil {
		log.Printf("⚠️ Could not load metrics: %v (starting fresh)", err)
	} else {
		log.Printf("📊 Loaded evaluation metrics for %d models", len(service.modelMetrics))
	}

	// Load completed games for evaluation
	if err := service.loadCompletedGames(); err != nil {
		log.Printf("⚠️ Could not load completed games: %v", err)
	}

	return service
}

// ============================================================================
// TRAIN/TEST SPLIT
// ============================================================================

// CreateTrainTestSplit splits games into training, validation, and test sets
func (mes *ModelEvaluationService) CreateTrainTestSplit(trainRatio, valRatio, testRatio float64) (*models.TrainTestSplit, error) {
	mes.mutex.RLock()
	defer mes.mutex.RUnlock()

	if len(mes.completedGames) == 0 {
		return nil, fmt.Errorf("no completed games available for split")
	}

	// Validate ratios
	if math.Abs(trainRatio+valRatio+testRatio-1.0) > 0.01 {
		return nil, fmt.Errorf("ratios must sum to 1.0")
	}

	// Sort games by date (temporal split)
	sortedGames := make([]models.CompletedGame, len(mes.completedGames))
	copy(sortedGames, mes.completedGames)
	sort.Slice(sortedGames, func(i, j int) bool {
		return sortedGames[i].GameDate.Before(sortedGames[j].GameDate)
	})

	// Calculate split sizes
	total := len(sortedGames)
	trainSize := int(float64(total) * trainRatio)
	valSize := int(float64(total) * valRatio)
	testSize := total - trainSize - valSize

	// Create split
	split := &models.TrainTestSplit{
		TrainingSet:    sortedGames[:trainSize],
		ValidationSet:  sortedGames[trainSize : trainSize+valSize],
		TestSet:        sortedGames[trainSize+valSize:],
		TrainSize:      trainSize,
		ValidationSize: valSize,
		TestSize:       testSize,
		SplitRatio:     []float64{trainRatio, valRatio, testRatio},
		SplitMethod:    "temporal",
		CreatedAt:      time.Now(),
	}

	log.Printf("📊 Train/Test Split Created:")
	log.Printf("   Training: %d games (%.1f%%)", trainSize, trainRatio*100)
	log.Printf("   Validation: %d games (%.1f%%)", valSize, valRatio*100)
	log.Printf("   Test: %d games (%.1f%%)", testSize, testRatio*100)

	return split, nil
}

// EvaluateOnTestSet evaluates models on test set
func (mes *ModelEvaluationService) EvaluateOnTestSet(testGames []models.CompletedGame) error {
	log.Printf("🧪 Evaluating models on %d test games...", len(testGames))

	for _, game := range testGames {
		// Build prediction factors
		homeFactors := mes.buildFactors(game, true)
		awayFactors := mes.buildFactors(game, false)

		// Get predictions from each model
		if mes.neuralNet != nil {
			pred, err := mes.neuralNet.Predict(homeFactors, awayFactors)
			if err == nil {
				mes.recordPrediction("Neural Network", pred, &game, homeFactors, awayFactors)
			}
		}

		// Could add other models here
	}

	// Calculate and save metrics
	mes.calculateMetrics()
	mes.saveMetrics()

	log.Printf("✅ Evaluation complete")
	return nil
}

// ============================================================================
// BATCH TRAINING
// ============================================================================

// AddGameToBatch adds a game to the pending batch for training
func (mes *ModelEvaluationService) AddGameToBatch(game models.CompletedGame) error {
	mes.batchMutex.Lock()
	defer mes.batchMutex.Unlock()

	mes.pendingBatch = append(mes.pendingBatch, game)

	// Train when batch is full
	if len(mes.pendingBatch) >= mes.batchSize {
		log.Printf("📦 Batch full (%d games), starting batch training...", len(mes.pendingBatch))

		if err := mes.trainBatch(); err != nil {
			log.Printf("⚠️ Batch training failed: %v", err)
			return err
		}

		// Clear batch
		mes.pendingBatch = []models.CompletedGame{}
		log.Printf("✅ Batch training complete")
	} else {
		log.Printf("📦 Game added to batch (%d/%d)", len(mes.pendingBatch), mes.batchSize)
	}

	return nil
}

// trainBatch trains models on accumulated batch
func (mes *ModelEvaluationService) trainBatch() error {
	if len(mes.pendingBatch) == 0 {
		return nil
	}

	successCount := 0
	errorCount := 0

	// Train Neural Network on batch
	if mes.neuralNet != nil {
		for _, game := range mes.pendingBatch {
			homeFactors := mes.buildFactors(game, true)
			awayFactors := mes.buildFactors(game, false)
			gameResult := mes.convertToGameResult(&game)

			if err := mes.neuralNet.TrainOnGameResult(gameResult, homeFactors, awayFactors); err != nil {
				errorCount++
			} else {
				successCount++
			}
		}
		log.Printf("🧠 Neural Network trained on %d games (batch)", successCount)
	}

	// Recalculate and save metrics after batch training
	mes.calculateMetrics()
	if err := mes.saveMetrics(); err != nil {
		log.Printf("⚠️ Failed to save metrics after batch training: %v", err)
	}

	// Update other models
	if mes.eloModel != nil {
		for _, game := range mes.pendingBatch {
			gameResult := mes.convertToGameResult(&game)
			if err := mes.eloModel.processGameResult(gameResult); err != nil {
				errorCount++
			} else {
				successCount++
			}
		}
	}

	if mes.poissonModel != nil {
		for _, game := range mes.pendingBatch {
			gameResult := mes.convertToGameResult(&game)
			if err := mes.poissonModel.processGameResult(gameResult); err != nil {
				errorCount++
			} else {
				successCount++
			}
		}
	}

	return nil
}

// ForceBatchTraining trains on current batch regardless of size
func (mes *ModelEvaluationService) ForceBatchTraining() error {
	mes.batchMutex.Lock()
	defer mes.batchMutex.Unlock()

	if len(mes.pendingBatch) == 0 {
		return nil
	}

	log.Printf("📦 Force training on partial batch (%d games)...", len(mes.pendingBatch))

	if err := mes.trainBatch(); err != nil {
		return err
	}

	mes.pendingBatch = []models.CompletedGame{}
	return nil
}

// ============================================================================
// PERFORMANCE METRICS
// ============================================================================

// recordPrediction records a prediction outcome
func (mes *ModelEvaluationService) recordPrediction(modelName string, pred *models.ModelResult, game *models.CompletedGame, homeFactors, awayFactors *models.PredictionFactors) {
	outcome := models.PredictionOutcome{
		GameID:          game.GameID,
		PredictedWinner: homeFactors.TeamCode,
		ActualWinner:    game.Winner,
		PredictedScore:  pred.PredictedScore,
		ActualScore:     fmt.Sprintf("%d-%d", game.HomeTeam.Score, game.AwayTeam.Score),
		WinProbability:  pred.WinProbability,
		Confidence:      pred.Confidence,
		IsCorrect:       homeFactors.TeamCode == game.Winner,
		HomeTeam:        game.HomeTeam.TeamCode,
		AwayTeam:        game.AwayTeam.TeamCode,
		PredictionTime:  time.Now(),
		GameTime:        game.GameDate,
	}

	// Determine if underdog won (upset)
	if pred.WinProbability < 0.5 {
		outcome.IsUpset = outcome.IsCorrect
		outcome.PredictedUpset = true
	}

	mes.mutex.Lock()
	mes.predictions = append(mes.predictions, outcome)
	mes.mutex.Unlock()
}

// calculateMetrics calculates performance metrics for all models
func (mes *ModelEvaluationService) calculateMetrics() {
	mes.mutex.Lock()
	defer mes.mutex.Unlock()

	// Group predictions by model
	modelPredictions := make(map[string][]models.PredictionOutcome)
	for _, pred := range mes.predictions {
		// Extract model name (simplified - would need to track this properly)
		modelPredictions["Neural Network"] = append(modelPredictions["Neural Network"], pred)
	}

	// Calculate metrics for each model
	for modelName, preds := range modelPredictions {
		metrics := mes.calculateModelMetrics(modelName, preds)
		mes.modelMetrics[modelName] = metrics
	}
}

// calculateModelMetrics calculates metrics for a specific model
func (mes *ModelEvaluationService) calculateModelMetrics(modelName string, predictions []models.PredictionOutcome) *models.ModelEvaluationMetrics {
	metrics := &models.ModelEvaluationMetrics{
		ModelName:        modelName,
		TotalPredictions: len(predictions),
		LastEvaluated:    time.Now(),
	}

	if len(predictions) == 0 {
		return metrics
	}

	var correctCount int
	var brierSum float64
	var confidenceSum float64
	var homeCorrect, awayCorrect int
	var homeTotal, awayTotal int
	var upsetCorrect, upsetTotal int

	cm := &models.ConfusionMatrix{}

	for _, pred := range predictions {
		// Accuracy
		if pred.IsCorrect {
			correctCount++
		}

		// Confusion Matrix
		if pred.PredictedWinner == pred.HomeTeam {
			if pred.IsCorrect {
				cm.TruePositives++
			} else {
				cm.FalsePositives++
			}
		} else {
			if pred.IsCorrect {
				cm.TrueNegatives++
			} else {
				cm.FalseNegatives++
			}
		}

		// Brier Score (for probability calibration)
		actual := 0.0
		if pred.IsCorrect {
			actual = 1.0
		}
		brierSum += math.Pow(pred.WinProbability-actual, 2)

		// Confidence
		confidenceSum += pred.Confidence

		// Home/Away accuracy
		if pred.PredictedWinner == pred.HomeTeam {
			homeTotal++
			if pred.IsCorrect {
				homeCorrect++
			}
		} else {
			awayTotal++
			if pred.IsCorrect {
				awayCorrect++
			}
		}

		// Upset detection
		if pred.IsUpset {
			upsetTotal++
			if pred.PredictedUpset && pred.IsCorrect {
				upsetCorrect++
			}
		}
	}

	// Calculate final metrics
	metrics.CorrectPredictions = correctCount
	metrics.Accuracy = float64(correctCount) / float64(len(predictions))
	metrics.BrierScore = brierSum / float64(len(predictions))
	metrics.AvgConfidence = confidenceSum / float64(len(predictions))
	metrics.CalibrationError = math.Abs(metrics.AvgConfidence - metrics.Accuracy)

	// Confusion matrix metrics
	metrics.TruePositives = cm.TruePositives
	metrics.TrueNegatives = cm.TrueNegatives
	metrics.FalsePositives = cm.FalsePositives
	metrics.FalseNegatives = cm.FalseNegatives
	metrics.Precision, metrics.Recall, metrics.F1Score, _ = cm.CalculateMetrics()

	// Context-specific
	if homeTotal > 0 {
		metrics.HomeAccuracy = float64(homeCorrect) / float64(homeTotal)
	}
	if awayTotal > 0 {
		metrics.AwayAccuracy = float64(awayCorrect) / float64(awayTotal)
	}
	if upsetTotal > 0 {
		metrics.UpsetDetection = float64(upsetCorrect) / float64(upsetTotal)
	}

	// Last N accuracy
	if len(predictions) >= 10 {
		last10 := predictions[len(predictions)-10:]
		last10Correct := 0
		for _, pred := range last10 {
			if pred.IsCorrect {
				last10Correct++
			}
		}
		metrics.Last10Accuracy = float64(last10Correct) / 10.0
	}

	return metrics
}

// GetMetrics returns current metrics for all models
func (mes *ModelEvaluationService) GetMetrics() map[string]*models.ModelEvaluationMetrics {
	mes.mutex.RLock()
	defer mes.mutex.RUnlock()

	// Return copy
	metricsCopy := make(map[string]*models.ModelEvaluationMetrics)
	for k, v := range mes.modelMetrics {
		metricsCopy[k] = v
	}

	return metricsCopy
}

// GetEnsembleMetrics returns overall ensemble performance
func (mes *ModelEvaluationService) GetEnsembleMetrics() *models.EnsembleMetrics {
	mes.mutex.RLock()
	defer mes.mutex.RUnlock()

	ensemble := &models.EnsembleMetrics{
		Timestamp:           time.Now(),
		ModelMetrics:        mes.modelMetrics,
		TotalGamesEvaluated: len(mes.predictions),
		Version:             "1.0",
	}

	// Find best and worst models
	var bestAccuracy, worstAccuracy float64 = 0, 1.0
	for name, metrics := range mes.modelMetrics {
		if metrics.Accuracy > bestAccuracy {
			bestAccuracy = metrics.Accuracy
			ensemble.BestModel = name
			ensemble.BestAccuracy = bestAccuracy
		}
		if metrics.Accuracy < worstAccuracy {
			worstAccuracy = metrics.Accuracy
			ensemble.WorstModel = name
			ensemble.WorstAccuracy = worstAccuracy
		}
	}

	return ensemble
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// saveMetrics saves performance metrics to disk
func (mes *ModelEvaluationService) saveMetrics() error {
	filePath := filepath.Join(mes.metricsDir, "model_metrics.json")

	ensemble := mes.GetEnsembleMetrics()

	jsonData, err := json.MarshalIndent(ensemble, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling metrics: %w", err)
	}

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing metrics file: %w", err)
	}

	return nil
}

// loadMetrics loads performance metrics from disk
func (mes *ModelEvaluationService) loadMetrics() error {
	filePath := filepath.Join(mes.metricsDir, "model_metrics.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("metrics file not found")
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading metrics file: %w", err)
	}

	var ensemble models.EnsembleMetrics
	err = json.Unmarshal(jsonData, &ensemble)
	if err != nil {
		return fmt.Errorf("error unmarshaling metrics: %w", err)
	}

	mes.modelMetrics = ensemble.ModelMetrics

	return nil
}

// loadCompletedGames loads completed games from results directory
func (mes *ModelEvaluationService) loadCompletedGames() error {
	resultsDir := "data/results"

	// Walk through all subdirectories
	err := filepath.Walk(resultsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Process JSON files
		if !info.IsDir() && filepath.Ext(path) == ".json" && filepath.Base(path) != "processed_games.json" {
			jsonData, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			var game models.CompletedGame
			if err := json.Unmarshal(jsonData, &game); err != nil {
				// Skip files that don't match structure
				return nil
			}

			mes.completedGames = append(mes.completedGames, game)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error loading completed games: %w", err)
	}

	log.Printf("📊 Loaded %d completed games for evaluation", len(mes.completedGames))
	return nil
}

// Helper methods
func (mes *ModelEvaluationService) buildFactors(game models.CompletedGame, isHome bool) *models.PredictionFactors {
	var team, opponent models.TeamGameResult

	if isHome {
		team = game.HomeTeam
		opponent = game.AwayTeam
	} else {
		team = game.AwayTeam
		opponent = game.HomeTeam
	}

	homeAdvantage := 0.0
	if isHome {
		homeAdvantage = 1.0
	}

	return &models.PredictionFactors{
		TeamCode:          team.TeamCode,
		GoalsFor:          float64(team.Score),
		GoalsAgainst:      float64(opponent.Score),
		PowerPlayPct:      team.PowerPlayPct,
		PenaltyKillPct:    team.PenaltyKillPct,
		WinPercentage:     0.5,
		RecentForm:        0.5,
		RestDays:          1,
		HomeAdvantage:     homeAdvantage,
		BackToBackPenalty: 0.0,
		HeadToHead:        0.5,
		TravelFatigue:     models.TravelFatigue{},
		AltitudeAdjust:    models.AltitudeAdjust{},
		ScheduleStrength:  models.ScheduleStrength{},
		InjuryImpact:      models.InjuryImpact{},
		MomentumFactors:   models.MomentumFactors{},
		AdvancedStats:     models.AdvancedAnalytics{},
		WeatherAnalysis:   models.WeatherAnalysis{},
		MarketData:        models.MarketAdjustment{},
	}
}

func (mes *ModelEvaluationService) convertToGameResult(game *models.CompletedGame) *models.GameResult {
	isOT := game.WinType == "OT"
	isSO := game.WinType == "SO"

	return &models.GameResult{
		GameID:      game.GameID,
		HomeTeam:    game.HomeTeam.TeamCode,
		AwayTeam:    game.AwayTeam.TeamCode,
		HomeScore:   game.HomeTeam.Score,
		AwayScore:   game.AwayTeam.Score,
		GameState:   "FINAL",
		Period:      3,
		TimeLeft:    "0:00",
		GameDate:    game.GameDate,
		Venue:       game.Venue,
		IsOvertime:  isOT,
		IsShootout:  isSO,
		WinningTeam: game.Winner,
		LosingTeam:  mes.getLosingTeam(game),
		UpdatedAt:   time.Now(),
	}
}

func (mes *ModelEvaluationService) getLosingTeam(game *models.CompletedGame) string {
	if game.Winner == game.HomeTeam.TeamCode {
		return game.AwayTeam.TeamCode
	}
	return game.HomeTeam.TeamCode
}

// Global evaluation service
var (
	globalEvaluationService *ModelEvaluationService
	evaluationMutex         sync.Mutex
)

// InitializeEvaluationService initializes the global evaluation service
func InitializeEvaluationService(neuralNet *NeuralNetworkModel, elo *EloRatingModel, poisson *PoissonRegressionModel) error {
	evaluationMutex.Lock()
	defer evaluationMutex.Unlock()

	if globalEvaluationService != nil {
		return fmt.Errorf("evaluation service already initialized")
	}

	globalEvaluationService = NewModelEvaluationService(neuralNet, elo, poisson)

	return nil
}

// GetEvaluationService returns the global evaluation service
func GetEvaluationService() *ModelEvaluationService {
	evaluationMutex.Lock()
	defer evaluationMutex.Unlock()
	return globalEvaluationService
}
