package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MetaLearnerModel implements stacking ensemble learning
// It learns the optimal way to combine predictions from base models
type MetaLearnerModel struct {
	// Meta-model (learns to combine base models)
	weights      []float64 // Weights for each base model + context features
	bias         float64
	learningRate float64

	// Architecture
	numBaseModels   int // 9 base models
	numContextFeats int // Additional context features

	// Training
	trained bool
	weight  float64 // Weight in final ensemble (if used)

	// Persistence
	dataDir     string
	lastUpdated time.Time
	mutex       sync.RWMutex

	// Performance tracking
	trainAccuracy float64
	valAccuracy   float64
}

// ModelPredictions holds predictions from all base models
type ModelPredictions struct {
	Statistical      float64
	Bayesian         float64
	MonteCarlo       float64
	Elo              float64
	Poisson          float64
	NeuralNetwork    float64
	GradientBoosting float64
	LSTM             float64
	RandomForest     float64
}

// MetaGameContext holds additional context features for meta-learning
type MetaGameContext struct {
	IsDivisionalGame bool
	IsPlayoffGame    bool
	IsRivalryGame    bool
	HomeTeamHot      bool // 4+ wins in last 5
	AwayTeamHot      bool
	HomeTeamCold     bool // 4+ losses in last 5
	AwayTeamCold     bool
	RestAdvantage    float64 // Difference in rest days
	TravelDistance   float64 // Travel distance for away team
	BackToBack       bool    // Either team on back-to-back
}

// NewMetaLearnerModel creates a new meta-learner
func NewMetaLearnerModel() *MetaLearnerModel {
	dataDir := "data/models"
	os.MkdirAll(dataDir, 0755)

	numBaseModels := 9    // 9 base prediction models
	numContextFeats := 10 // 10 context features
	totalFeatures := numBaseModels + numContextFeats

	mlm := &MetaLearnerModel{
		weights:         make([]float64, totalFeatures),
		bias:            0.0,
		learningRate:    0.01,
		numBaseModels:   numBaseModels,
		numContextFeats: numContextFeats,
		trained:         false,
		weight:          1.0, // Meta-learner gets 100% weight (it combines others)
		dataDir:         dataDir,
		lastUpdated:     time.Now(),
	}

	// Try to load existing model
	if err := mlm.loadModel(); err != nil {
		log.Printf("🎯 Initializing new Meta-Learner (no saved model found)")
		mlm.initializeWeights()
	} else {
		log.Printf("🎯 Meta-Learner loaded from disk")
		log.Printf("   Train Acc: %.2f%%, Val Acc: %.2f%%", mlm.trainAccuracy*100, mlm.valAccuracy*100)
		log.Printf("   Last updated: %s", mlm.lastUpdated.Format("2006-01-02 15:04:05"))
	}

	return mlm
}

// initializeWeights initializes meta-learner weights
func (mlm *MetaLearnerModel) initializeWeights() {
	// Initialize with small random values
	for i := range mlm.weights {
		mlm.weights[i] = (rand.Float64()*2 - 1) * 0.1
	}

	// Bias starts at 0.5 (neutral)
	mlm.bias = 0.5

	log.Printf("🎯 Meta-Learner weights initialized")
}

// PredictFromModels makes a meta-prediction from base model predictions
func (mlm *MetaLearnerModel) PredictFromModels(predictions *ModelPredictions, context *MetaGameContext) float64 {
	mlm.mutex.RLock()
	defer mlm.mutex.RUnlock()

	if !mlm.trained {
		// If not trained, return weighted average of base models
		return mlm.weightedAverage(predictions)
	}

	// Extract features
	features := mlm.extractFeatures(predictions, context)

	// Linear combination
	sum := mlm.bias
	for i, feat := range features {
		sum += mlm.weights[i] * feat
	}

	// Apply sigmoid to get probability
	winProb := 1.0 / (1.0 + math.Exp(-sum))

	// Ensure reasonable bounds
	winProb = math.Max(0.30, math.Min(0.90, winProb))

	return winProb
}

// extractFeatures extracts features from model predictions and context
func (mlm *MetaLearnerModel) extractFeatures(predictions *ModelPredictions, context *MetaGameContext) []float64 {
	features := make([]float64, mlm.numBaseModels+mlm.numContextFeats)
	idx := 0

	// Base model predictions (9 features)
	features[idx] = predictions.Statistical
	idx++
	features[idx] = predictions.Bayesian
	idx++
	features[idx] = predictions.MonteCarlo
	idx++
	features[idx] = predictions.Elo
	idx++
	features[idx] = predictions.Poisson
	idx++
	features[idx] = predictions.NeuralNetwork
	idx++
	features[idx] = predictions.GradientBoosting
	idx++
	features[idx] = predictions.LSTM
	idx++
	features[idx] = predictions.RandomForest
	idx++

	// Context features (10 features)
	if context.IsDivisionalGame {
		features[idx] = 1.0
	}
	idx++
	if context.IsPlayoffGame {
		features[idx] = 1.0
	}
	idx++
	if context.IsRivalryGame {
		features[idx] = 1.0
	}
	idx++
	if context.HomeTeamHot {
		features[idx] = 1.0
	}
	idx++
	if context.AwayTeamHot {
		features[idx] = 1.0
	}
	idx++
	if context.HomeTeamCold {
		features[idx] = 1.0
	}
	idx++
	if context.AwayTeamCold {
		features[idx] = 1.0
	}
	idx++
	features[idx] = context.RestAdvantage / 5.0 // Normalize
	idx++
	features[idx] = context.TravelDistance / 3000.0 // Normalize
	idx++
	if context.BackToBack {
		features[idx] = 1.0
	}

	return features
}

// weightedAverage returns simple weighted average if not trained
func (mlm *MetaLearnerModel) weightedAverage(predictions *ModelPredictions) float64 {
	// Use base weights from ensemble
	sum := predictions.Statistical*0.30 +
		predictions.Bayesian*0.12 +
		predictions.MonteCarlo*0.09 +
		predictions.Elo*0.17 +
		predictions.Poisson*0.12 +
		predictions.NeuralNetwork*0.06 +
		predictions.GradientBoosting*0.07 +
		predictions.LSTM*0.07 +
		predictions.RandomForest*0.07

	return sum
}

// Train trains the meta-learner on completed games with base model predictions
func (mlm *MetaLearnerModel) Train(trainingData []MetaTrainingExample) error {
	mlm.mutex.Lock()
	defer mlm.mutex.Unlock()

	if len(trainingData) < 20 {
		return fmt.Errorf("insufficient training data: need at least 20 examples, have %d", len(trainingData))
	}

	log.Printf("🎯 Training Meta-Learner on %d examples...", len(trainingData))
	start := time.Now()

	// Split into train/validation (80/20)
	splitIdx := int(float64(len(trainingData)) * 0.8)
	trainSet := trainingData[:splitIdx]
	valSet := trainingData[splitIdx:]

	// Training epochs
	epochs := 100
	bestValAccuracy := 0.0
	bestWeights := make([]float64, len(mlm.weights))
	bestBias := mlm.bias

	for epoch := 0; epoch < epochs; epoch++ {
		// Shuffle training data
		rand.Shuffle(len(trainSet), func(i, j int) {
			trainSet[i], trainSet[j] = trainSet[j], trainSet[i]
		})

		// Train on each example
		totalLoss := 0.0
		for _, example := range trainSet {
			loss := mlm.trainOnExample(&example)
			totalLoss += loss
		}

		// Evaluate on validation set
		if (epoch+1)%10 == 0 {
			trainAcc := mlm.evaluateAccuracy(trainSet)
			valAcc := mlm.evaluateAccuracy(valSet)

			log.Printf("   Epoch %d/%d: Train Acc=%.2f%%, Val Acc=%.2f%%, Loss=%.4f",
				epoch+1, epochs, trainAcc*100, valAcc*100, totalLoss/float64(len(trainSet)))

			// Save best model (based on validation accuracy)
			if valAcc > bestValAccuracy {
				bestValAccuracy = valAcc
				copy(bestWeights, mlm.weights)
				bestBias = mlm.bias
			}
		}
	}

	// Restore best weights
	copy(mlm.weights, bestWeights)
	mlm.bias = bestBias
	mlm.trainAccuracy = mlm.evaluateAccuracy(trainSet)
	mlm.valAccuracy = bestValAccuracy

	mlm.trained = true
	mlm.lastUpdated = time.Now()

	trainingTime := time.Since(start)
	log.Printf("✅ Meta-Learner training complete!")
	log.Printf("   Train Accuracy: %.2f%%", mlm.trainAccuracy*100)
	log.Printf("   Val Accuracy: %.2f%%", mlm.valAccuracy*100)
	log.Printf("   Time: %.1fs", trainingTime.Seconds())

	// Save model
	if err := mlm.saveModel(); err != nil {
		log.Printf("⚠️ Failed to save Meta-Learner: %v", err)
	}

	return nil
}

// trainOnExample trains on a single example using gradient descent
func (mlm *MetaLearnerModel) trainOnExample(example *MetaTrainingExample) float64 {
	// Forward pass
	features := mlm.extractFeatures(&example.Predictions, &example.Context)

	sum := mlm.bias
	for i, feat := range features {
		sum += mlm.weights[i] * feat
	}

	prediction := 1.0 / (1.0 + math.Exp(-sum)) // Sigmoid

	// Calculate loss (binary cross-entropy)
	actualLabel := example.ActualOutcome // 1.0 for win, 0.0 for loss
	loss := -actualLabel*math.Log(math.Max(prediction, 1e-10)) -
		(1-actualLabel)*math.Log(math.Max(1-prediction, 1e-10))

	// Backward pass (gradient descent)
	error := prediction - actualLabel

	// Update weights
	for i, feat := range features {
		gradient := error * feat
		mlm.weights[i] -= mlm.learningRate * gradient
	}

	// Update bias
	mlm.bias -= mlm.learningRate * error

	return loss
}

// evaluateAccuracy calculates accuracy on a dataset
func (mlm *MetaLearnerModel) evaluateAccuracy(data []MetaTrainingExample) float64 {
	correct := 0
	total := len(data)

	for _, example := range data {
		prediction := mlm.PredictFromModels(&example.Predictions, &example.Context)

		predictedWin := prediction > 0.5
		actualWin := example.ActualOutcome > 0.5

		if predictedWin == actualWin {
			correct++
		}
	}

	return float64(correct) / float64(total)
}

// MetaTrainingExample represents a training example for the meta-learner
type MetaTrainingExample struct {
	Predictions   ModelPredictions
	Context       MetaGameContext
	ActualOutcome float64 // 1.0 = home win, 0.0 = home loss
	GameID        int
	GameDate      time.Time
}

// GetName returns the model name
func (mlm *MetaLearnerModel) GetName() string {
	return "Meta-Learner"
}

// GetWeight returns the model weight
func (mlm *MetaLearnerModel) GetWeight() float64 {
	mlm.mutex.RLock()
	defer mlm.mutex.RUnlock()
	return mlm.weight
}

// GetLearnedWeights returns the learned weights for each base model
func (mlm *MetaLearnerModel) GetLearnedWeights() map[string]float64 {
	mlm.mutex.RLock()
	defer mlm.mutex.RUnlock()

	if !mlm.trained {
		return nil
	}

	weights := map[string]float64{
		"Statistical":      mlm.weights[0],
		"Bayesian":         mlm.weights[1],
		"MonteCarlo":       mlm.weights[2],
		"Elo":              mlm.weights[3],
		"Poisson":          mlm.weights[4],
		"NeuralNetwork":    mlm.weights[5],
		"GradientBoosting": mlm.weights[6],
		"LSTM":             mlm.weights[7],
		"RandomForest":     mlm.weights[8],
	}

	return weights
}

// Persistence methods

// MetaLearnerModelData represents serializable model data
type MetaLearnerModelData struct {
	Weights         []float64 `json:"weights"`
	Bias            float64   `json:"bias"`
	LearningRate    float64   `json:"learningRate"`
	NumBaseModels   int       `json:"numBaseModels"`
	NumContextFeats int       `json:"numContextFeats"`
	Trained         bool      `json:"trained"`
	Weight          float64   `json:"weight"`
	TrainAccuracy   float64   `json:"trainAccuracy"`
	ValAccuracy     float64   `json:"valAccuracy"`
	LastUpdated     time.Time `json:"lastUpdated"`
	Version         string    `json:"version"`
}

func (mlm *MetaLearnerModel) saveModel() error {
	filePath := filepath.Join(mlm.dataDir, "meta_learner.json")

	modelData := MetaLearnerModelData{
		Weights:         mlm.weights,
		Bias:            mlm.bias,
		LearningRate:    mlm.learningRate,
		NumBaseModels:   mlm.numBaseModels,
		NumContextFeats: mlm.numContextFeats,
		Trained:         mlm.trained,
		Weight:          mlm.weight,
		TrainAccuracy:   mlm.trainAccuracy,
		ValAccuracy:     mlm.valAccuracy,
		LastUpdated:     time.Now(),
		Version:         "1.0",
	}

	data, err := json.MarshalIndent(modelData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling meta-learner: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing meta-learner: %w", err)
	}

	log.Printf("💾 Meta-Learner saved: trained=%v, val_acc=%.2f%%", mlm.trained, mlm.valAccuracy*100)
	return nil
}

func (mlm *MetaLearnerModel) loadModel() error {
	filePath := filepath.Join(mlm.dataDir, "meta_learner.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("no saved model found")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading meta-learner: %w", err)
	}

	var modelData MetaLearnerModelData
	err = json.Unmarshal(data, &modelData)
	if err != nil {
		return fmt.Errorf("error unmarshaling meta-learner: %w", err)
	}

	mlm.weights = modelData.Weights
	mlm.bias = modelData.Bias
	mlm.learningRate = modelData.LearningRate
	mlm.numBaseModels = modelData.NumBaseModels
	mlm.numContextFeats = modelData.NumContextFeats
	mlm.trained = modelData.Trained
	mlm.weight = modelData.Weight
	mlm.trainAccuracy = modelData.TrainAccuracy
	mlm.valAccuracy = modelData.ValAccuracy
	mlm.lastUpdated = modelData.LastUpdated

	return nil
}
