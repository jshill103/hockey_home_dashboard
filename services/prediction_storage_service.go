package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// StoredPrediction represents a prediction stored before a game
type StoredPrediction struct {
	GameID       int                   `json:"gameId"`
	GameDate     time.Time             `json:"gameDate"`
	HomeTeam     string                `json:"homeTeam"`
	AwayTeam     string                `json:"awayTeam"`
	PredictedAt  time.Time             `json:"predictedAt"`
	Prediction   models.GamePrediction `json:"prediction"`
	ActualResult *models.GameResult    `json:"actualResult,omitempty"`
	Accuracy     *PredictionAccuracy   `json:"accuracy,omitempty"`
}

// PredictionAccuracy tracks how accurate a prediction was
type PredictionAccuracy struct {
	WinnerCorrect    bool    `json:"winnerCorrect"`
	ScoreDiff        int     `json:"scoreDiff"`        // Difference between predicted and actual score
	ConfidenceLevel  float64 `json:"confidenceLevel"`  // Ensemble confidence
	CalibrationError float64 `json:"calibrationError"` // |predicted prob - actual result|
}

// PredictionStorageService manages storing and retrieving predictions
type PredictionStorageService struct {
	dataDir string
	mutex   sync.RWMutex
}

var (
	predictionStorageService     *PredictionStorageService
	predictionStorageServiceOnce sync.Once
)

// InitPredictionStorageService initializes the singleton
func InitPredictionStorageService() *PredictionStorageService {
	predictionStorageServiceOnce.Do(func() {
		dataDir := "data/predictions"
		os.MkdirAll(dataDir, 0755)

		predictionStorageService = &PredictionStorageService{
			dataDir: dataDir,
		}

		log.Printf("📝 Prediction Storage Service initialized (dir: %s)", dataDir)
	})
	return predictionStorageService
}

// GetPredictionStorageService returns the singleton instance
func GetPredictionStorageService() *PredictionStorageService {
	return predictionStorageService
}

// StorePrediction saves a prediction for a game
func (pss *PredictionStorageService) StorePrediction(gameID int, gameDate time.Time, homeTeam, awayTeam string, prediction *models.GamePrediction) error {
	pss.mutex.Lock()
	defer pss.mutex.Unlock()

	stored := StoredPrediction{
		GameID:      gameID,
		GameDate:    gameDate,
		HomeTeam:    homeTeam,
		AwayTeam:    awayTeam,
		PredictedAt: time.Now(),
		Prediction:  *prediction,
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prediction: %w", err)
	}

	filename := filepath.Join(pss.dataDir, fmt.Sprintf("game_%d.json", gameID))
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write prediction file: %w", err)
	}

	log.Printf("📝 Stored prediction for game %d (%s @ %s)", gameID, awayTeam, homeTeam)
	return nil
}

// LoadPrediction retrieves a stored prediction
func (pss *PredictionStorageService) LoadPrediction(gameID int) (*StoredPrediction, error) {
	pss.mutex.RLock()
	defer pss.mutex.RUnlock()

	filename := filepath.Join(pss.dataDir, fmt.Sprintf("game_%d.json", gameID))

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No prediction stored
		}
		return nil, fmt.Errorf("failed to read prediction file: %w", err)
	}

	var stored StoredPrediction
	err = json.Unmarshal(data, &stored)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal prediction: %w", err)
	}

	return &stored, nil
}

// UpdateWithResult updates a stored prediction with the actual game result
func (pss *PredictionStorageService) UpdateWithResult(gameID int, result *models.CompletedGame) error {
	// Load existing prediction
	stored, err := pss.LoadPrediction(gameID)
	if err != nil {
		return err
	}
	if stored == nil {
		// No prediction stored for this game
		return nil
	}

	pss.mutex.Lock()
	defer pss.mutex.Unlock()

	// Convert CompletedGame to GameResult
	gameResult := &models.GameResult{
		GameID:      result.GameID,
		HomeTeam:    result.HomeTeam.TeamCode,
		AwayTeam:    result.AwayTeam.TeamCode,
		HomeScore:   result.HomeTeam.Score,
		AwayScore:   result.AwayTeam.Score,
		WinningTeam: result.Winner,
		GameDate:    result.GameDate,
		GameState:   "FINAL",
	}
	stored.ActualResult = gameResult

	// Calculate accuracy
	accuracy := pss.calculateAccuracy(&stored.Prediction, result.Winner)
	stored.Accuracy = accuracy

	// Save updated prediction
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated prediction: %w", err)
	}

	filename := filepath.Join(pss.dataDir, fmt.Sprintf("game_%d.json", gameID))
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated prediction: %w", err)
	}

	log.Printf("✅ Updated prediction for game %d with actual result (Winner Correct: %v)",
		gameID, accuracy.WinnerCorrect)

	return nil
}

// calculateAccuracy determines how accurate the prediction was
func (pss *PredictionStorageService) calculateAccuracy(prediction *models.GamePrediction, actualWinner string) *PredictionAccuracy {
	accuracy := &PredictionAccuracy{
		ConfidenceLevel: prediction.Confidence,
	}

	// Check if winner was predicted correctly
	accuracy.WinnerCorrect = (prediction.Prediction.Winner == actualWinner)

	// Calculate calibration error
	// If we predicted 70% win probability and they won, error is |0.7 - 1.0| = 0.3
	// If we predicted 70% and they lost, error is |0.7 - 0.0| = 0.7
	actualOutcome := 0.0
	if prediction.Prediction.Winner == actualWinner {
		actualOutcome = 1.0
	}
	accuracy.CalibrationError = float64(int(1000*(prediction.Prediction.WinProbability-actualOutcome))) / 1000.0
	if accuracy.CalibrationError < 0 {
		accuracy.CalibrationError = -accuracy.CalibrationError
	}

	// Score difference will be calculated from actual result when available
	accuracy.ScoreDiff = 0

	return accuracy
}

// GetAllPredictions returns all stored predictions
func (pss *PredictionStorageService) GetAllPredictions() ([]*StoredPrediction, error) {
	pss.mutex.RLock()
	defer pss.mutex.RUnlock()

	files, err := ioutil.ReadDir(pss.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read predictions directory: %w", err)
	}

	predictions := []*StoredPrediction{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(pss.dataDir, file.Name()))
		if err != nil {
			continue
		}

		var stored StoredPrediction
		err = json.Unmarshal(data, &stored)
		if err != nil {
			continue
		}

		predictions = append(predictions, &stored)
	}

	return predictions, nil
}

// GetAccuracyStats returns overall accuracy statistics
func (pss *PredictionStorageService) GetAccuracyStats() map[string]interface{} {
	predictions, err := pss.GetAllPredictions()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	totalPredictions := 0
	correctPredictions := 0
	totalConfidence := 0.0
	totalCalibration := 0.0

	for _, pred := range predictions {
		if pred.Accuracy != nil {
			totalPredictions++
			if pred.Accuracy.WinnerCorrect {
				correctPredictions++
			}
			totalConfidence += pred.Accuracy.ConfidenceLevel
			totalCalibration += pred.Accuracy.CalibrationError
		}
	}

	accuracy := 0.0
	avgConfidence := 0.0
	avgCalibration := 0.0

	if totalPredictions > 0 {
		accuracy = float64(correctPredictions) / float64(totalPredictions)
		avgConfidence = totalConfidence / float64(totalPredictions)
		avgCalibration = totalCalibration / float64(totalPredictions)
	}

	return map[string]interface{}{
		"totalPredictions":   totalPredictions,
		"correctPredictions": correctPredictions,
		"accuracy":           accuracy,
		"averageConfidence":  avgConfidence,
		"averageCalibration": avgCalibration,
	}
}
