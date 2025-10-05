package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// LiveDataService manages real-time NHL data updates
type LiveDataService struct {
	cache       *DataCache
	updateQueue chan UpdateTask
	isRunning   bool
	mutex       sync.RWMutex
	updateStats *UpdateStatistics
	lastUpdate  time.Time
	subscribers []UpdateSubscriber
	rateLimiter *RateLimiter
}

// UpdateTask represents a data update task
type UpdateTask struct {
	TaskType   string    `json:"taskType"` // "game_result", "standings", "schedule", "injuries"
	TeamCodes  []string  `json:"teamCodes"`
	GameID     int       `json:"gameId"`
	Priority   int       `json:"priority"` // 1=highest, 5=lowest
	CreatedAt  time.Time `json:"createdAt"`
	RetryCount int       `json:"retryCount"`
	MaxRetries int       `json:"maxRetries"`
}

// UpdateStatistics tracks update performance
type UpdateStatistics struct {
	TotalUpdates      int                      `json:"totalUpdates"`
	SuccessfulUpdates int                      `json:"successfulUpdates"`
	FailedUpdates     int                      `json:"failedUpdates"`
	LastUpdateTime    time.Time                `json:"lastUpdateTime"`
	AverageUpdateTime time.Duration            `json:"averageUpdateTime"`
	UpdatesByType     map[string]int           `json:"updatesByType"`
	ErrorsByType      map[string]int           `json:"errorsByType"`
	APIResponseTimes  map[string]time.Duration `json:"apiResponseTimes"`
}

// UpdateSubscriber interface for components that need update notifications
type UpdateSubscriber interface {
	OnDataUpdate(updateType string, data interface{}) error
	GetSubscriberName() string
}

// DataCache provides TTL-based caching for NHL data
type DataCache struct {
	gameResults sync.Map // gameID -> CachedGameResult
	teamStats   sync.Map // teamCode -> CachedTeamStats
	standings   sync.Map // "current" -> CachedStandings
	schedules   sync.Map // teamCode -> CachedSchedule
	injuries    sync.Map // teamCode -> CachedInjuries
	defaultTTL  time.Duration
	mutex       sync.RWMutex
}

// CachedData represents cached data with TTL
type CachedData struct {
	Data      interface{} `json:"data"`
	CachedAt  time.Time   `json:"cachedAt"`
	ExpiresAt time.Time   `json:"expiresAt"`
	Source    string      `json:"source"`
}

// RateLimiter prevents API abuse
type RateLimiter struct {
	requests    map[string][]time.Time
	maxRequests int
	timeWindow  time.Duration
	mutex       sync.Mutex
}

// NewLiveDataService creates a new live data service
func NewLiveDataService() *LiveDataService {
	service := &LiveDataService{
		cache:       NewDataCache(15 * time.Minute), // 15-minute default TTL
		updateQueue: make(chan UpdateTask, 1000),
		updateStats: &UpdateStatistics{
			UpdatesByType:    make(map[string]int),
			ErrorsByType:     make(map[string]int),
			APIResponseTimes: make(map[string]time.Duration),
		},
		subscribers: make([]UpdateSubscriber, 0),
		rateLimiter: NewRateLimiter(100, time.Minute), // 100 requests per minute
	}

	return service
}

// NewDataCache creates a new data cache
func NewDataCache(defaultTTL time.Duration) *DataCache {
	return &DataCache{
		defaultTTL: defaultTTL,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, timeWindow time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		timeWindow:  timeWindow,
	}
}

// Start begins the live data update service
func (lds *LiveDataService) Start() error {
	lds.mutex.Lock()
	defer lds.mutex.Unlock()

	if lds.isRunning {
		return fmt.Errorf("live data service is already running")
	}

	lds.isRunning = true

	// Start the update worker
	go lds.updateWorker()

	// Start the periodic update scheduler
	go lds.periodicUpdateScheduler()

	// Start cache cleanup routine
	go lds.cacheCleanupRoutine()

	log.Printf("🚀 Live Data Service started successfully")
	return nil
}

// Stop stops the live data update service
func (lds *LiveDataService) Stop() error {
	lds.mutex.Lock()
	defer lds.mutex.Unlock()

	if !lds.isRunning {
		return fmt.Errorf("live data service is not running")
	}

	lds.isRunning = false
	close(lds.updateQueue)

	log.Printf("⏹️ Live Data Service stopped")
	return nil
}

// Subscribe adds a subscriber for data updates
func (lds *LiveDataService) Subscribe(subscriber UpdateSubscriber) {
	lds.mutex.Lock()
	defer lds.mutex.Unlock()

	lds.subscribers = append(lds.subscribers, subscriber)
	log.Printf("📡 Added subscriber: %s", subscriber.GetSubscriberName())
}

// ScheduleUpdate adds an update task to the queue
func (lds *LiveDataService) ScheduleUpdate(task UpdateTask) {
	task.CreatedAt = time.Now()
	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}

	select {
	case lds.updateQueue <- task:
		log.Printf("📝 Scheduled %s update (Priority: %d)", task.TaskType, task.Priority)
	default:
		log.Printf("⚠️ Update queue full, dropping task: %s", task.TaskType)
	}
}

// updateWorker processes update tasks from the queue
func (lds *LiveDataService) updateWorker() {
	for task := range lds.updateQueue {
		if !lds.isRunning {
			break
		}

		startTime := time.Now()
		err := lds.processUpdateTask(task)
		duration := time.Since(startTime)

		lds.updateUpdateStats(task.TaskType, err, duration)

		if err != nil {
			log.Printf("❌ Update task failed: %s - %v", task.TaskType, err)

			// Retry logic
			if task.RetryCount < task.MaxRetries {
				task.RetryCount++
				// Exponential backoff
				retryDelay := time.Duration(task.RetryCount*task.RetryCount) * time.Second
				time.AfterFunc(retryDelay, func() {
					lds.ScheduleUpdate(task)
				})
				log.Printf("🔄 Retrying %s update in %v (attempt %d/%d)",
					task.TaskType, retryDelay, task.RetryCount, task.MaxRetries)
			}
		} else {
			log.Printf("✅ Update task completed: %s (took %v)", task.TaskType, duration)
		}
	}
}

// processUpdateTask handles individual update tasks
func (lds *LiveDataService) processUpdateTask(task UpdateTask) error {
	// Check rate limit
	if !lds.rateLimiter.Allow(task.TaskType) {
		return fmt.Errorf("rate limit exceeded for %s", task.TaskType)
	}

	switch task.TaskType {
	case "game_result":
		return lds.updateGameResult(task.GameID)
	case "standings":
		return lds.updateStandings()
	case "schedule":
		return lds.updateSchedules(task.TeamCodes)
	case "injuries":
		return lds.updateInjuries(task.TeamCodes)
	case "team_stats":
		return lds.updateTeamStats(task.TeamCodes)
	default:
		return fmt.Errorf("unknown update task type: %s", task.TaskType)
	}
}

// updateGameResult fetches and caches a specific game result
func (lds *LiveDataService) updateGameResult(gameID int) error {
	log.Printf("🏒 Updating game result for game %d", gameID)

	// This would call the NHL API to get game results
	// For now, we'll simulate the data structure
	gameResult := &models.GameResult{
		GameID:    gameID,
		HomeTeam:  "UTA",
		AwayTeam:  "SJS",
		HomeScore: 3,
		AwayScore: 2,
		GameState: "FINAL",
		Period:    3,
		TimeLeft:  "00:00",
		UpdatedAt: time.Now(),
	}

	// Cache the result
	lds.cache.SetGameResult(gameID, gameResult, 24*time.Hour)

	// Notify subscribers
	lds.notifySubscribers("game_result", gameResult)

	return nil
}

// updateStandings fetches and caches current NHL standings
func (lds *LiveDataService) updateStandings() error {
	log.Printf("📊 Updating NHL standings")

	standings, err := GetStandings()
	if err != nil {
		return fmt.Errorf("failed to fetch standings: %w", err)
	}

	// Cache the standings
	lds.cache.SetStandings(&standings, 1*time.Hour)

	// Notify subscribers
	lds.notifySubscribers("standings", standings)

	return nil
}

// updateSchedules fetches and caches team schedules
func (lds *LiveDataService) updateSchedules(teamCodes []string) error {
	log.Printf("📅 Updating schedules for teams: %v", teamCodes)

	for _, teamCode := range teamCodes {
		schedule, err := lds.fetchTeamSchedule(teamCode)
		if err != nil {
			log.Printf("⚠️ Failed to update schedule for %s: %v", teamCode, err)
			continue
		}

		// Cache the schedule
		lds.cache.SetSchedule(teamCode, schedule, 2*time.Hour)

		// Notify subscribers
		lds.notifySubscribers("schedule", map[string]interface{}{
			"teamCode": teamCode,
			"schedule": schedule,
		})
	}

	return nil
}

// updateInjuries fetches and caches team injury reports
func (lds *LiveDataService) updateInjuries(teamCodes []string) error {
	log.Printf("🏥 Injury reports disabled - no real data source available")
	// Injury service removed - no real API data available
	// Future: Integrate real injury API when available
	return nil
}

// updateTeamStats fetches and caches team statistics
func (lds *LiveDataService) updateTeamStats(teamCodes []string) error {
	log.Printf("📈 Updating team stats for teams: %v", teamCodes)

	for _, teamCode := range teamCodes {
		// This would fetch detailed team stats from NHL API
		// For now, we'll use the standings data as a proxy
		standings, err := GetStandings()
		if err != nil {
			return fmt.Errorf("failed to fetch standings for team stats: %w", err)
		}

		// Find team in standings
		var teamStats *models.TeamStanding
		for _, team := range standings.Standings {
			if team.TeamAbbrev.Default == teamCode {
				teamStats = &team
				break
			}
		}

		if teamStats != nil {
			// Cache the team stats
			lds.cache.SetTeamStats(teamCode, teamStats, 1*time.Hour)

			// Notify subscribers
			lds.notifySubscribers("team_stats", map[string]interface{}{
				"teamCode":  teamCode,
				"teamStats": teamStats,
			})
		}
	}

	return nil
}

// fetchTeamSchedule fetches a team's schedule from the NHL API
func (lds *LiveDataService) fetchTeamSchedule(teamCode string) (*models.ScheduleResponse, error) {
	// This would make an API call to get the team's schedule
	// For now, we'll return a placeholder
	return &models.ScheduleResponse{
		Games: []models.Game{},
	}, nil
}

// periodicUpdateScheduler runs periodic updates
func (lds *LiveDataService) periodicUpdateScheduler() {
	// Hourly updates
	hourlyTicker := time.NewTicker(1 * time.Hour)
	defer hourlyTicker.Stop()

	// 15-minute game checks during active periods
	gameCheckTicker := time.NewTicker(15 * time.Minute)
	defer gameCheckTicker.Stop()

	// Daily deep updates
	dailyTicker := time.NewTicker(24 * time.Hour)
	defer dailyTicker.Stop()

	for {
		select {
		case <-hourlyTicker.C:
			if lds.isRunning {
				lds.scheduleHourlyUpdates()
			}
		case <-gameCheckTicker.C:
			if lds.isRunning {
				lds.scheduleGameChecks()
			}
		case <-dailyTicker.C:
			if lds.isRunning {
				lds.scheduleDailyUpdates()
			}
		}

		if !lds.isRunning {
			break
		}
	}
}

// scheduleHourlyUpdates schedules regular hourly updates
func (lds *LiveDataService) scheduleHourlyUpdates() {
	log.Printf("⏰ Scheduling hourly updates")

	// Update standings
	lds.ScheduleUpdate(UpdateTask{
		TaskType: "standings",
		Priority: 2,
	})

	// Update schedules for all teams (you might want to limit this)
	allTeams := []string{"UTA", "COL", "VGK", "SJS", "LAK"} // Add more teams as needed
	lds.ScheduleUpdate(UpdateTask{
		TaskType:  "schedule",
		TeamCodes: allTeams,
		Priority:  3,
	})

	// Update injury reports
	lds.ScheduleUpdate(UpdateTask{
		TaskType:  "injuries",
		TeamCodes: allTeams,
		Priority:  4,
	})
}

// scheduleGameChecks checks for completed games
func (lds *LiveDataService) scheduleGameChecks() {
	log.Printf("🎮 Scheduling game completion checks")

	// This would check for recently completed games
	// For now, we'll simulate checking a few game IDs
	recentGameIDs := []int{2024020001, 2024020002, 2024020003}

	for _, gameID := range recentGameIDs {
		lds.ScheduleUpdate(UpdateTask{
			TaskType: "game_result",
			GameID:   gameID,
			Priority: 1, // High priority for game results
		})
	}
}

// scheduleDailyUpdates schedules comprehensive daily updates
func (lds *LiveDataService) scheduleDailyUpdates() {
	log.Printf("🌅 Scheduling daily deep updates")

	// This would schedule comprehensive updates
	// Including cache cleanup, model recalibration, etc.

	// Clear old cache entries
	lds.cache.Cleanup()

	// Schedule comprehensive data refresh
	allTeams := []string{"UTA", "COL", "VGK", "SJS", "LAK", "EDM", "CGY", "WPG", "MIN", "CHI", "STL", "DAL", "NSH", "ANA"} // Add all NHL teams

	lds.ScheduleUpdate(UpdateTask{
		TaskType:  "team_stats",
		TeamCodes: allTeams,
		Priority:  5,
	})
}

// cacheCleanupRoutine periodically cleans expired cache entries
func (lds *LiveDataService) cacheCleanupRoutine() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if !lds.isRunning {
			break
		}

		lds.cache.Cleanup()
	}
}

// notifySubscribers notifies all subscribers of data updates
func (lds *LiveDataService) notifySubscribers(updateType string, data interface{}) {
	for _, subscriber := range lds.subscribers {
		go func(sub UpdateSubscriber) {
			if err := sub.OnDataUpdate(updateType, data); err != nil {
				log.Printf("⚠️ Subscriber %s failed to process %s update: %v",
					sub.GetSubscriberName(), updateType, err)
			}
		}(subscriber)
	}
}

// updateUpdateStats updates performance statistics
func (lds *LiveDataService) updateUpdateStats(taskType string, err error, duration time.Duration) {
	lds.mutex.Lock()
	defer lds.mutex.Unlock()

	lds.updateStats.TotalUpdates++
	lds.updateStats.UpdatesByType[taskType]++

	if err != nil {
		lds.updateStats.FailedUpdates++
		lds.updateStats.ErrorsByType[taskType]++
	} else {
		lds.updateStats.SuccessfulUpdates++
	}

	lds.updateStats.LastUpdateTime = time.Now()
	lds.updateStats.APIResponseTimes[taskType] = duration

	// Calculate average update time
	if lds.updateStats.TotalUpdates > 0 {
		totalDuration := time.Duration(0)
		for _, d := range lds.updateStats.APIResponseTimes {
			totalDuration += d
		}
		lds.updateStats.AverageUpdateTime = totalDuration / time.Duration(len(lds.updateStats.APIResponseTimes))
	}
}

// GetUpdateStats returns current update statistics
func (lds *LiveDataService) GetUpdateStats() *UpdateStatistics {
	lds.mutex.RLock()
	defer lds.mutex.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := *lds.updateStats
	return &statsCopy
}

// Allow checks if a request is allowed by the rate limiter
func (rl *RateLimiter) Allow(requestType string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Clean old requests
	if requests, exists := rl.requests[requestType]; exists {
		validRequests := make([]time.Time, 0)
		for _, reqTime := range requests {
			if now.Sub(reqTime) < rl.timeWindow {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[requestType] = validRequests
	}

	// Check if we can make a new request
	if len(rl.requests[requestType]) >= rl.maxRequests {
		return false
	}

	// Add the new request
	if rl.requests[requestType] == nil {
		rl.requests[requestType] = make([]time.Time, 0)
	}
	rl.requests[requestType] = append(rl.requests[requestType], now)

	return true
}

// Cache methods
func (dc *DataCache) SetGameResult(gameID int, result *models.GameResult, ttl time.Duration) {
	data := &CachedData{
		Data:      result,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Source:    "live_update",
	}
	dc.gameResults.Store(gameID, data)
}

func (dc *DataCache) GetGameResult(gameID int) (*models.GameResult, bool) {
	if cached, exists := dc.gameResults.Load(gameID); exists {
		cachedData := cached.(*CachedData)
		if time.Now().Before(cachedData.ExpiresAt) {
			return cachedData.Data.(*models.GameResult), true
		}
		dc.gameResults.Delete(gameID)
	}
	return nil, false
}

func (dc *DataCache) SetStandings(standings *models.StandingsResponse, ttl time.Duration) {
	data := &CachedData{
		Data:      standings,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Source:    "live_update",
	}
	dc.standings.Store("current", data)
}

func (dc *DataCache) GetStandings() (*models.StandingsResponse, bool) {
	if cached, exists := dc.standings.Load("current"); exists {
		cachedData := cached.(*CachedData)
		if time.Now().Before(cachedData.ExpiresAt) {
			return cachedData.Data.(*models.StandingsResponse), true
		}
		dc.standings.Delete("current")
	}
	return nil, false
}

func (dc *DataCache) SetSchedule(teamCode string, schedule *models.ScheduleResponse, ttl time.Duration) {
	data := &CachedData{
		Data:      schedule,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Source:    "live_update",
	}
	dc.schedules.Store(teamCode, data)
}

func (dc *DataCache) SetInjuries(teamCode string, injuries interface{}, ttl time.Duration) {
	data := &CachedData{
		Data:      injuries,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Source:    "live_update",
	}
	dc.injuries.Store(teamCode, data)
}

func (dc *DataCache) SetTeamStats(teamCode string, stats *models.TeamStanding, ttl time.Duration) {
	data := &CachedData{
		Data:      stats,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Source:    "live_update",
	}
	dc.teamStats.Store(teamCode, data)
}

// Cleanup removes expired cache entries
func (dc *DataCache) Cleanup() {
	now := time.Now()

	// Clean game results
	dc.gameResults.Range(func(key, value interface{}) bool {
		cachedData := value.(*CachedData)
		if now.After(cachedData.ExpiresAt) {
			dc.gameResults.Delete(key)
		}
		return true
	})

	// Clean other cache types similarly
	dc.standings.Range(func(key, value interface{}) bool {
		cachedData := value.(*CachedData)
		if now.After(cachedData.ExpiresAt) {
			dc.standings.Delete(key)
		}
		return true
	})

	dc.schedules.Range(func(key, value interface{}) bool {
		cachedData := value.(*CachedData)
		if now.After(cachedData.ExpiresAt) {
			dc.schedules.Delete(key)
		}
		return true
	})

	dc.injuries.Range(func(key, value interface{}) bool {
		cachedData := value.(*CachedData)
		if now.After(cachedData.ExpiresAt) {
			dc.injuries.Delete(key)
		}
		return true
	})

	dc.teamStats.Range(func(key, value interface{}) bool {
		cachedData := value.(*CachedData)
		if now.After(cachedData.ExpiresAt) {
			dc.teamStats.Delete(key)
		}
		return true
	})

	log.Printf("🧹 Cache cleanup completed")
}
