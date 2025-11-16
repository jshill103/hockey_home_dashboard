package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ModelUpdateScheduler manages the scheduling and execution of model updates
type ModelUpdateScheduler struct {
	models          []UpdatableModel
	updateInterval  time.Duration
	ticker          *time.Ticker
	isRunning       bool
	mutex           sync.RWMutex
	updateHistory   []UpdateRecord
	liveDataService *LiveDataService
	ensembleService *EnsemblePredictionService
	lastFullUpdate  time.Time
	updateQueue     chan ModelUpdateTask
	maxHistorySize  int
}

// UpdatableModel interface for models that can be updated with live data
type UpdatableModel interface {
	Update(data *LiveGameData) error
	GetLastUpdate() time.Time
	GetUpdateStats() ModelUpdateStats
	GetModelName() string
	RequiresUpdate(data *LiveGameData) bool
}

// ModelUpdateTask represents a model update task
type ModelUpdateTask struct {
	ModelName   string        `json:"modelName"`
	UpdateType  string        `json:"updateType"` // "game_result", "schedule", "full_refresh"
	Data        *LiveGameData `json:"data"`
	Priority    int           `json:"priority"`
	CreatedAt   time.Time     `json:"createdAt"`
	ProcessedAt time.Time     `json:"processedAt"`
}

// UpdateRecord tracks model update history
type UpdateRecord struct {
	ModelName   string        `json:"modelName"`
	UpdateType  string        `json:"updateType"`
	StartTime   time.Time     `json:"startTime"`
	EndTime     time.Time     `json:"endTime"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
	DataQuality float64       `json:"dataQuality"`
	ImpactScore float64       `json:"impactScore"` // How much the update changed the model
}

// ModelUpdateStats provides statistics about model updates
type ModelUpdateStats struct {
	TotalUpdates      int           `json:"totalUpdates"`
	SuccessfulUpdates int           `json:"successfulUpdates"`
	FailedUpdates     int           `json:"failedUpdates"`
	LastUpdateTime    time.Time     `json:"lastUpdateTime"`
	AverageUpdateTime time.Duration `json:"averageUpdateTime"`
	LastError         string        `json:"lastError,omitempty"`
	UpdateFrequency   time.Duration `json:"updateFrequency"`
	DataFreshness     time.Duration `json:"dataFreshness"`
}

// LiveGameData contains all the data needed for model updates
type LiveGameData struct {
	GameResults   []*models.GameResult                `json:"gameResults"`
	Standings     *models.StandingsResponse           `json:"standings"`
	Schedules     map[string]*models.ScheduleResponse `json:"schedules"`
	InjuryReports map[string]interface{}              `json:"injuryReports"`
	TeamStats     map[string]*models.TeamStanding     `json:"teamStats"`
	UpdateTime    time.Time                           `json:"updateTime"`
	DataSources   []string                            `json:"dataSources"`
	QualityScore  float64                             `json:"qualityScore"`
}

// NewModelUpdateScheduler creates a new model update scheduler
func NewModelUpdateScheduler(liveDataService *LiveDataService, ensembleService *EnsemblePredictionService) *ModelUpdateScheduler {
	scheduler := &ModelUpdateScheduler{
		models:          make([]UpdatableModel, 0),
		updateInterval:  1 * time.Hour, // Default hourly updates
		liveDataService: liveDataService,
		ensembleService: ensembleService,
		updateHistory:   make([]UpdateRecord, 0),
		updateQueue:     make(chan ModelUpdateTask, 100),
		maxHistorySize:  1000,
	}

	return scheduler
}

// RegisterModel adds a model to the update scheduler
func (mus *ModelUpdateScheduler) RegisterModel(model UpdatableModel) {
	mus.mutex.Lock()
	defer mus.mutex.Unlock()

	mus.models = append(mus.models, model)
	log.Printf("📝 Registered model for updates: %s", model.GetModelName())
}

// Start begins the model update scheduler
func (mus *ModelUpdateScheduler) Start() error {
	mus.mutex.Lock()
	defer mus.mutex.Unlock()

	if mus.isRunning {
		return fmt.Errorf("model update scheduler is already running")
	}

	mus.isRunning = true
	mus.ticker = time.NewTicker(mus.updateInterval)

	// Start the update worker
	go mus.updateWorker()

	// Start the periodic scheduler
	go mus.periodicScheduler()

	// Subscribe to live data updates
	mus.liveDataService.Subscribe(mus)

	log.Printf("🚀 Model Update Scheduler started with %d models", len(mus.models))
	return nil
}

// Stop stops the model update scheduler
func (mus *ModelUpdateScheduler) Stop() error {
	mus.mutex.Lock()
	defer mus.mutex.Unlock()

	if !mus.isRunning {
		return fmt.Errorf("model update scheduler is not running")
	}

	mus.isRunning = false
	if mus.ticker != nil {
		mus.ticker.Stop()
	}
	close(mus.updateQueue)

	log.Printf("⏹️ Model Update Scheduler stopped")
	return nil
}

// OnDataUpdate implements UpdateSubscriber interface
func (mus *ModelUpdateScheduler) OnDataUpdate(updateType string, data interface{}) error {
	log.Printf("📡 Received data update: %s", updateType)

	// Convert the update to LiveGameData
	liveData, err := mus.convertToLiveGameData(updateType, data)
	if err != nil {
		return fmt.Errorf("failed to convert update data: %w", err)
	}

	// Schedule model updates based on the data type
	switch updateType {
	case "game_result":
		mus.scheduleGameResultUpdates(liveData)
	case "standings":
		mus.scheduleStandingsUpdates(liveData)
	case "schedule":
		mus.scheduleScheduleUpdates(liveData)
	case "injuries":
		mus.scheduleInjuryUpdates(liveData)
	case "team_stats":
		mus.scheduleTeamStatsUpdates(liveData)
	}

	return nil
}

// GetSubscriberName implements UpdateSubscriber interface
func (mus *ModelUpdateScheduler) GetSubscriberName() string {
	return "ModelUpdateScheduler"
}

// updateWorker processes model update tasks
func (mus *ModelUpdateScheduler) updateWorker() {
	for task := range mus.updateQueue {
		if !mus.isRunning {
			break
		}

		mus.processUpdateTask(task)
	}
}

// periodicScheduler handles scheduled updates
func (mus *ModelUpdateScheduler) periodicScheduler() {
	for range mus.ticker.C {
		if !mus.isRunning {
			break
		}

		mus.schedulePeriodicUpdates()
	}
}

// processUpdateTask processes a single model update task
func (mus *ModelUpdateScheduler) processUpdateTask(task ModelUpdateTask) {
	startTime := time.Now()
	task.ProcessedAt = startTime

	log.Printf("🔄 Processing update task: %s for model %s", task.UpdateType, task.ModelName)

	// Find the model
	var targetModel UpdatableModel
	for _, model := range mus.models {
		if model.GetModelName() == task.ModelName {
			targetModel = model
			break
		}
	}

	if targetModel == nil {
		log.Printf("❌ Model not found: %s", task.ModelName)
		return
	}

	// Check if model requires update
	if !targetModel.RequiresUpdate(task.Data) {
		log.Printf("⏭️ Model %s doesn't require update, skipping", task.ModelName)
		return
	}

	// Perform the update
	var updateErr error
	var impactScore float64

	// Capture model state before update for impact measurement
	beforeStats := targetModel.GetUpdateStats()

	updateErr = targetModel.Update(task.Data)

	// Measure impact
	afterStats := targetModel.GetUpdateStats()
	impactScore = mus.calculateUpdateImpact(beforeStats, afterStats)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Record the update
	record := UpdateRecord{
		ModelName:   task.ModelName,
		UpdateType:  task.UpdateType,
		StartTime:   startTime,
		EndTime:     endTime,
		Duration:    duration,
		Success:     updateErr == nil,
		DataQuality: task.Data.QualityScore,
		ImpactScore: impactScore,
	}

	if updateErr != nil {
		record.Error = updateErr.Error()
		log.Printf("❌ Model update failed: %s - %v", task.ModelName, updateErr)
	} else {
		log.Printf("✅ Model update completed: %s (Impact: %.3f, Duration: %v)",
			task.ModelName, impactScore, duration)
	}

	mus.addUpdateRecord(record)

	// If this was a significant update, trigger ensemble recalculation
	if impactScore > 0.1 && updateErr == nil {
		mus.triggerEnsembleUpdate(task.ModelName)
	}
}

// convertToLiveGameData converts update data to LiveGameData format
func (mus *ModelUpdateScheduler) convertToLiveGameData(updateType string, data interface{}) (*LiveGameData, error) {
	liveData := &LiveGameData{
		UpdateTime:   time.Now(),
		DataSources:  []string{updateType},
		QualityScore: 1.0, // Default quality score
	}

	switch updateType {
	case "game_result":
		if gameResult, ok := data.(*models.GameResult); ok {
			liveData.GameResults = []*models.GameResult{gameResult}
		}
	case "standings":
		if standings, ok := data.(*models.StandingsResponse); ok {
			liveData.Standings = standings
		}
	case "schedule":
		if scheduleData, ok := data.(map[string]interface{}); ok {
			if teamCode, exists := scheduleData["teamCode"].(string); exists {
				if schedule, scheduleExists := scheduleData["schedule"].(*models.ScheduleResponse); scheduleExists {
					liveData.Schedules = map[string]*models.ScheduleResponse{
						teamCode: schedule,
					}
				}
			}
		}
	case "injuries":
		if injuryData, ok := data.(map[string]interface{}); ok {
			liveData.InjuryReports = map[string]interface{}{
				injuryData["teamCode"].(string): injuryData["injuryReport"],
			}
		}
	case "team_stats":
		if statsData, ok := data.(map[string]interface{}); ok {
			if teamCode, exists := statsData["teamCode"].(string); exists {
				if stats, statsExists := statsData["teamStats"].(*models.TeamStanding); statsExists {
					liveData.TeamStats = map[string]*models.TeamStanding{
						teamCode: stats,
					}
				}
			}
		}
	default:
		return nil, fmt.Errorf("unknown update type: %s", updateType)
	}

	return liveData, nil
}

// scheduleGameResultUpdates schedules updates for game result data
func (mus *ModelUpdateScheduler) scheduleGameResultUpdates(data *LiveGameData) {
	for _, model := range mus.models {
		task := ModelUpdateTask{
			ModelName:  model.GetModelName(),
			UpdateType: "game_result",
			Data:       data,
			Priority:   1, // High priority for game results
			CreatedAt:  time.Now(),
		}

		select {
		case mus.updateQueue <- task:
			log.Printf("📝 Scheduled game result update for %s", model.GetModelName())
		default:
			log.Printf("⚠️ Update queue full, dropping task for %s", model.GetModelName())
		}
	}
}

// scheduleStandingsUpdates schedules updates for standings data
func (mus *ModelUpdateScheduler) scheduleStandingsUpdates(data *LiveGameData) {
	for _, model := range mus.models {
		task := ModelUpdateTask{
			ModelName:  model.GetModelName(),
			UpdateType: "standings",
			Data:       data,
			Priority:   2, // Medium priority
			CreatedAt:  time.Now(),
		}

		select {
		case mus.updateQueue <- task:
			log.Printf("📝 Scheduled standings update for %s", model.GetModelName())
		default:
			log.Printf("⚠️ Update queue full, dropping task for %s", model.GetModelName())
		}
	}
}

// scheduleScheduleUpdates schedules updates for schedule data
func (mus *ModelUpdateScheduler) scheduleScheduleUpdates(data *LiveGameData) {
	for _, model := range mus.models {
		task := ModelUpdateTask{
			ModelName:  model.GetModelName(),
			UpdateType: "schedule",
			Data:       data,
			Priority:   3, // Lower priority
			CreatedAt:  time.Now(),
		}

		select {
		case mus.updateQueue <- task:
			log.Printf("📝 Scheduled schedule update for %s", model.GetModelName())
		default:
			log.Printf("⚠️ Update queue full, dropping task for %s", model.GetModelName())
		}
	}
}

// scheduleInjuryUpdates schedules updates for injury data
func (mus *ModelUpdateScheduler) scheduleInjuryUpdates(data *LiveGameData) {
	for _, model := range mus.models {
		task := ModelUpdateTask{
			ModelName:  model.GetModelName(),
			UpdateType: "injuries",
			Data:       data,
			Priority:   2, // Medium priority
			CreatedAt:  time.Now(),
		}

		select {
		case mus.updateQueue <- task:
			log.Printf("📝 Scheduled injury update for %s", model.GetModelName())
		default:
			log.Printf("⚠️ Update queue full, dropping task for %s", model.GetModelName())
		}
	}
}

// scheduleTeamStatsUpdates schedules updates for team stats data
func (mus *ModelUpdateScheduler) scheduleTeamStatsUpdates(data *LiveGameData) {
	for _, model := range mus.models {
		task := ModelUpdateTask{
			ModelName:  model.GetModelName(),
			UpdateType: "team_stats",
			Data:       data,
			Priority:   3, // Lower priority
			CreatedAt:  time.Now(),
		}

		select {
		case mus.updateQueue <- task:
			log.Printf("📝 Scheduled team stats update for %s", model.GetModelName())
		default:
			log.Printf("⚠️ Update queue full, dropping task for %s", model.GetModelName())
		}
	}
}

// schedulePeriodicUpdates schedules regular periodic updates
func (mus *ModelUpdateScheduler) schedulePeriodicUpdates() {
	log.Printf("⏰ Scheduling periodic model updates")

	// Get fresh data from live data service
	liveData := mus.gatherLiveData()

	// Schedule full refresh for all models
	for _, model := range mus.models {
		task := ModelUpdateTask{
			ModelName:  model.GetModelName(),
			UpdateType: "full_refresh",
			Data:       liveData,
			Priority:   4, // Lower priority for periodic updates
			CreatedAt:  time.Now(),
		}

		select {
		case mus.updateQueue <- task:
			log.Printf("📝 Scheduled periodic update for %s", model.GetModelName())
		default:
			log.Printf("⚠️ Update queue full, dropping periodic task for %s", model.GetModelName())
		}
	}

	mus.lastFullUpdate = time.Now()
}

// gatherLiveData collects current data from all sources
func (mus *ModelUpdateScheduler) gatherLiveData() *LiveGameData {
	liveData := &LiveGameData{
		UpdateTime:    time.Now(),
		DataSources:   []string{"periodic_update"},
		QualityScore:  1.0,
		GameResults:   make([]*models.GameResult, 0),
		Schedules:     make(map[string]*models.ScheduleResponse),
		InjuryReports: make(map[string]interface{}),
		TeamStats:     make(map[string]*models.TeamStanding),
	}

	// Get standings from cache or API
	if standings, exists := mus.liveDataService.cache.GetStandings(); exists {
		liveData.Standings = standings
	}

	// Add more data gathering logic here
	// This would typically fetch recent game results, schedules, etc.

	return liveData
}

// calculateUpdateImpact measures how much an update changed the model
func (mus *ModelUpdateScheduler) calculateUpdateImpact(before, after ModelUpdateStats) float64 {
	// Simple impact calculation based on update frequency change
	// In a real implementation, this would be more sophisticated

	if before.LastUpdateTime.IsZero() {
		return 1.0 // First update has maximum impact
	}

	// Calculate impact based on time since last update
	timeSinceUpdate := time.Since(before.LastUpdateTime)
	expectedInterval := before.UpdateFrequency

	if expectedInterval == 0 {
		expectedInterval = 1 * time.Hour // Default
	}

	// Impact is higher if it's been longer since the last update
	impact := float64(timeSinceUpdate) / float64(expectedInterval)

	// Cap impact at 1.0
	if impact > 1.0 {
		impact = 1.0
	}

	return impact
}

// triggerEnsembleUpdate triggers a recalculation of ensemble predictions
func (mus *ModelUpdateScheduler) triggerEnsembleUpdate(updatedModel string) {
	log.Printf("🎯 Triggering ensemble update due to significant change in %s", updatedModel)

	// This would trigger the ensemble service to recalculate predictions
	// For now, we'll just log it
	// In a full implementation, this would call the ensemble service
}

// addUpdateRecord adds an update record to history
func (mus *ModelUpdateScheduler) addUpdateRecord(record UpdateRecord) {
	mus.mutex.Lock()
	defer mus.mutex.Unlock()

	mus.updateHistory = append(mus.updateHistory, record)

	// Limit history size
	if len(mus.updateHistory) > mus.maxHistorySize {
		mus.updateHistory = mus.updateHistory[1:]
	}
}

// GetUpdateHistory returns the update history
func (mus *ModelUpdateScheduler) GetUpdateHistory(limit int) []UpdateRecord {
	mus.mutex.RLock()
	defer mus.mutex.RUnlock()

	if limit <= 0 || limit > len(mus.updateHistory) {
		limit = len(mus.updateHistory)
	}

	// Return the most recent records
	start := len(mus.updateHistory) - limit
	if start < 0 {
		start = 0
	}

	history := make([]UpdateRecord, limit)
	copy(history, mus.updateHistory[start:])

	return history
}

// GetModelStats returns statistics for all registered models
func (mus *ModelUpdateScheduler) GetModelStats() map[string]ModelUpdateStats {
	mus.mutex.RLock()
	defer mus.mutex.RUnlock()

	stats := make(map[string]ModelUpdateStats)

	for _, model := range mus.models {
		stats[model.GetModelName()] = model.GetUpdateStats()
	}

	return stats
}

// SetUpdateInterval changes the update interval
func (mus *ModelUpdateScheduler) SetUpdateInterval(interval time.Duration) {
	mus.mutex.Lock()
	defer mus.mutex.Unlock()

	mus.updateInterval = interval

	if mus.ticker != nil {
		mus.ticker.Stop()
		mus.ticker = time.NewTicker(interval)
	}

	log.Printf("⏱️ Update interval changed to %v", interval)
}






