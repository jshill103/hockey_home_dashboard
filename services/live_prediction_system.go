package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// LivePredictionSystem integrates all live update components
type LivePredictionSystem struct {
	liveDataService *LiveDataService
	modelScheduler  *ModelUpdateScheduler
	ensembleService *EnsemblePredictionService
	eloModel        *EloRatingModel
	poissonModel    *PoissonRegressionModel
	neuralNet       *NeuralNetworkModel
	isRunning       bool
	teamCode        string
}

// NewLivePredictionSystem creates a new integrated live prediction system
func NewLivePredictionSystem(teamCode string) *LivePredictionSystem {
	// Create the live data service
	liveDataService := NewLiveDataService()

	// Create the ensemble service (which already has the models)
	ensembleService := NewEnsemblePredictionService(teamCode)

	// Create the model scheduler
	modelScheduler := NewModelUpdateScheduler(liveDataService, ensembleService)

	// Register models with the scheduler
	// Note: We need to make the models implement UpdatableModel interface
	// For now, we'll register the ones we've enhanced
	eloModel := NewEloRatingModel()
	poissonModel := NewPoissonRegressionModel()
	neuralNet := NewNeuralNetworkModel()

	modelScheduler.RegisterModel(eloModel)
	modelScheduler.RegisterModel(poissonModel)
	// Neural Network doesn't implement UpdatableModel yet, trained differently

	return &LivePredictionSystem{
		liveDataService: liveDataService,
		modelScheduler:  modelScheduler,
		ensembleService: ensembleService,
		eloModel:        eloModel,
		poissonModel:    poissonModel,
		neuralNet:       neuralNet,
		teamCode:        teamCode,
	}
}

// Start initializes and starts the live prediction system
func (lps *LivePredictionSystem) Start() error {
	if lps.isRunning {
		return fmt.Errorf("live prediction system is already running")
	}

	log.Printf("🚀 Starting Live Prediction System for team %s...", lps.teamCode)

	// Start the live data service
	if err := lps.liveDataService.Start(); err != nil {
		return fmt.Errorf("failed to start live data service: %w", err)
	}

	// Start the model scheduler
	if err := lps.modelScheduler.Start(); err != nil {
		return fmt.Errorf("failed to start model scheduler: %w", err)
	}

	// Schedule initial data updates
	lps.scheduleInitialUpdates()

	lps.isRunning = true
	log.Printf("✅ Live Prediction System started successfully")

	return nil
}

// Stop stops the live prediction system
func (lps *LivePredictionSystem) Stop() error {
	if !lps.isRunning {
		return fmt.Errorf("live prediction system is not running")
	}

	log.Printf("⏹️ Stopping Live Prediction System...")

	// Stop components
	if err := lps.modelScheduler.Stop(); err != nil {
		log.Printf("⚠️ Error stopping model scheduler: %v", err)
	}

	if err := lps.liveDataService.Stop(); err != nil {
		log.Printf("⚠️ Error stopping live data service: %v", err)
	}

	lps.isRunning = false
	log.Printf("✅ Live Prediction System stopped")

	return nil
}

// scheduleInitialUpdates triggers initial data collection
func (lps *LivePredictionSystem) scheduleInitialUpdates() {
	log.Printf("📋 Scheduling initial data updates...")

	// Schedule immediate updates for key data
	lps.liveDataService.ScheduleUpdate(UpdateTask{
		TaskType: "standings",
		Priority: 1,
	})

	// Schedule team-specific updates
	allTeams := []string{lps.teamCode, "COL", "VGK", "SJS", "LAK", "EDM", "CGY", "WPG", "MIN", "CHI", "STL", "DAL", "NSH", "ANA"}

	lps.liveDataService.ScheduleUpdate(UpdateTask{
		TaskType:  "schedule",
		TeamCodes: allTeams,
		Priority:  2,
	})

	lps.liveDataService.ScheduleUpdate(UpdateTask{
		TaskType:  "injuries",
		TeamCodes: allTeams,
		Priority:  3,
	})
}

// GetSystemStatus returns the current status of the live prediction system
func (lps *LivePredictionSystem) GetSystemStatus() *SystemStatus {
	return &SystemStatus{
		IsRunning:        lps.isRunning,
		TeamCode:         lps.teamCode,
		DataServiceStats: lps.liveDataService.GetUpdateStats(),
		ModelStats:       lps.modelScheduler.GetModelStats(),
		LastUpdate:       time.Now(), // This would be the actual last update time
	}
}

// SystemStatus represents the current status of the live prediction system
type SystemStatus struct {
	IsRunning        bool                        `json:"isRunning"`
	TeamCode         string                      `json:"teamCode"`
	DataServiceStats *UpdateStatistics           `json:"dataServiceStats"`
	ModelStats       map[string]ModelUpdateStats `json:"modelStats"`
	LastUpdate       time.Time                   `json:"lastUpdate"`
}

// ========== HTTP Handlers for Live System Management ==========

// HandleSystemStatus returns the system status as JSON
func (lps *LivePredictionSystem) HandleSystemStatus(w http.ResponseWriter, r *http.Request) {
	status := lps.GetSystemStatus()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode status", http.StatusInternalServerError)
		return
	}
}

// HandleForceUpdate forces an immediate update of all models
func (lps *LivePredictionSystem) HandleForceUpdate(w http.ResponseWriter, r *http.Request) {
	if !lps.isRunning {
		http.Error(w, "Live prediction system is not running", http.StatusServiceUnavailable)
		return
	}

	log.Printf("🔄 Manual update triggered via API")

	// Schedule high-priority updates
	lps.liveDataService.ScheduleUpdate(UpdateTask{
		TaskType: "standings",
		Priority: 1,
	})

	// Check for recent game results (this would be enhanced with actual game IDs)
	recentGameIDs := []int{2024020001, 2024020002, 2024020003}
	for _, gameID := range recentGameIDs {
		lps.liveDataService.ScheduleUpdate(UpdateTask{
			TaskType: "game_result",
			GameID:   gameID,
			Priority: 1,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Update triggered successfully",
	})
}

// HandleModelHistory returns the update history for models
func (lps *LivePredictionSystem) HandleModelHistory(w http.ResponseWriter, r *http.Request) {
	limit := 50 // Default limit

	history := lps.modelScheduler.GetUpdateHistory(limit)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(history); err != nil {
		http.Error(w, "Failed to encode history", http.StatusInternalServerError)
		return
	}
}

// GetEloModel returns the Elo rating model
func (lps *LivePredictionSystem) GetEloModel() *EloRatingModel {
	return lps.eloModel
}

// GetPoissonModel returns the Poisson regression model
func (lps *LivePredictionSystem) GetPoissonModel() *PoissonRegressionModel {
	return lps.poissonModel
}

// GetNeuralNetwork returns the Neural Network model
func (lps *LivePredictionSystem) GetNeuralNetwork() *NeuralNetworkModel {
	return lps.neuralNet
}

// GetEnsemble returns the ensemble service
func (lps *LivePredictionSystem) GetEnsemble() *EnsemblePredictionService {
	return lps.ensembleService
}

// Global instance for the live prediction system
var globalLivePredictionSystem *LivePredictionSystem

// InitializeLivePredictionSystem initializes the global live prediction system
func InitializeLivePredictionSystem(teamCode string) error {
	if globalLivePredictionSystem != nil {
		return fmt.Errorf("live prediction system already initialized")
	}

	globalLivePredictionSystem = NewLivePredictionSystem(teamCode)
	return globalLivePredictionSystem.Start()
}

// GetLivePredictionSystem returns the global live prediction system instance
func GetLivePredictionSystem() *LivePredictionSystem {
	return globalLivePredictionSystem
}

// StopLivePredictionSystem stops the global live prediction system
func StopLivePredictionSystem() error {
	if globalLivePredictionSystem == nil {
		return fmt.Errorf("live prediction system not initialized")
	}

	err := globalLivePredictionSystem.Stop()
	globalLivePredictionSystem = nil
	return err
}
