package services

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// HealthCheckService performs comprehensive health checks
type HealthCheckService struct {
	startTime time.Time
	version   string
}

var (
	globalHealthService *HealthCheckService
)

// InitHealthCheckService initializes the global health check service
func InitHealthCheckService(version string) {
	globalHealthService = &HealthCheckService{
		startTime: time.Now(),
		version:   version,
	}
}

// GetHealthCheckService returns the global health check service
func GetHealthCheckService() *HealthCheckService {
	if globalHealthService == nil {
		InitHealthCheckService("unknown")
	}
	return globalHealthService
}

// RunHealthChecks performs all health checks and returns overall status
func (hcs *HealthCheckService) RunHealthChecks() *models.HealthStatus {
	checks := make(map[string]models.HealthCheck)

	// 1. NHL API Health
	checks["nhl_api"] = hcs.checkNHLAPI()

	// 2. Data Persistence
	checks["data_persistence"] = hcs.checkDataPersistence()

	// 3. ML Models
	checks["ml_models"] = hcs.checkMLModels()

	// 4. Memory Usage
	checks["memory"] = hcs.checkMemoryUsage()

	// 5. Cache Health
	checks["cache"] = hcs.checkCacheHealth()

	// 6. API Cache Health
	checks["api_cache"] = hcs.checkAPICache()

	// Determine overall status
	overallStatus := hcs.determineOverallStatus(checks)

	return &models.HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    checks,
		Uptime:    hcs.getUptime(),
		Version:   hcs.version,
	}
}

// checkNHLAPI verifies NHL API is reachable and responsive
func (hcs *HealthCheckService) checkNHLAPI() models.HealthCheck {
	start := time.Now()

	// Try to fetch standings (lightweight endpoint)
	_, err := MakeAPICall("https://api-web.nhle.com/v1/standings/now")
	responseTime := time.Since(start)

	if err != nil {
		return models.HealthCheck{
			Name:         "NHL API",
			Status:       "unhealthy",
			Message:      fmt.Sprintf("API unreachable: %v", err),
			ResponseTime: responseTime,
			LastChecked:  time.Now(),
		}
	}

	// Check response time
	if responseTime > 5*time.Second {
		return models.HealthCheck{
			Name:         "NHL API",
			Status:       "degraded",
			Message:      "Slow response time",
			ResponseTime: responseTime,
			LastChecked:  time.Now(),
			Details: map[string]interface{}{
				"threshold_ms": 5000,
				"actual_ms":    responseTime.Milliseconds(),
			},
		}
	}

	return models.HealthCheck{
		Name:         "NHL API",
		Status:       "healthy",
		Message:      "API responsive",
		ResponseTime: responseTime,
		LastChecked:  time.Now(),
	}
}

// checkDataPersistence verifies data directories are writable
func (hcs *HealthCheckService) checkDataPersistence() models.HealthCheck {
	directories := []string{
		"data/accuracy",
		"data/models",
		"data/results",
		"data/cache/predictions",
		"data/matchups",
		"data/rolling_stats",
		"data/player_impact",
		"data/lineups",
	}

	issues := []string{}
	for _, dir := range directories {
		// Check if directory exists and is writable
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("%s: not found", dir))
			} else {
				issues = append(issues, fmt.Sprintf("%s: %v", dir, err))
			}
			continue
		}

		if !info.IsDir() {
			issues = append(issues, fmt.Sprintf("%s: not a directory", dir))
			continue
		}

		// Try to create a test file
		testFile := fmt.Sprintf("%s/.health_check", dir)
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			issues = append(issues, fmt.Sprintf("%s: not writable", dir))
		} else {
			os.Remove(testFile) // Clean up
		}
	}

	if len(issues) > 0 {
		return models.HealthCheck{
			Name:        "Data Persistence",
			Status:      "unhealthy",
			Message:     fmt.Sprintf("%d directories have issues", len(issues)),
			LastChecked: time.Now(),
			Details: map[string]interface{}{
				"issues":            issues,
				"total_directories": len(directories),
			},
		}
	}

	return models.HealthCheck{
		Name:        "Data Persistence",
		Status:      "healthy",
		Message:     fmt.Sprintf("All %d directories accessible", len(directories)),
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"directories": directories,
		},
	}
}

// checkMLModels verifies ML models are loaded and functional
func (hcs *HealthCheckService) checkMLModels() models.HealthCheck {
	loadedModels := []string{}
	issues := []string{}

	// PHASE 3: Check if models are loaded IN MEMORY (not just files)
	liveSys := GetLivePredictionSystem()
	if liveSys != nil {
		// Check Neural Network
		if nn := liveSys.GetNeuralNetwork(); nn != nil {
			loadedModels = append(loadedModels, "Neural Network")
		} else {
			issues = append(issues, "Neural Network: not loaded")
		}

		// Check Elo Rating
		if elo := liveSys.GetEloModel(); elo != nil {
			loadedModels = append(loadedModels, "Elo Rating")
		} else {
			issues = append(issues, "Elo Rating: not loaded")
		}

		// Check Poisson
		if poisson := liveSys.GetPoissonModel(); poisson != nil {
			loadedModels = append(loadedModels, "Poisson Regression")
		} else {
			issues = append(issues, "Poisson Regression: not loaded")
		}
	}

	// Check other models
	if gbModel := GetGradientBoostingModel(); gbModel != nil {
		loadedModels = append(loadedModels, "Gradient Boosting")
	} else {
		issues = append(issues, "Gradient Boosting: not loaded")
	}

	if lstmModel := GetLSTMModel(); lstmModel != nil {
		loadedModels = append(loadedModels, "LSTM")
	} else {
		issues = append(issues, "LSTM: not loaded")
	}

	if rfModel := GetRandomForestModel(); rfModel != nil {
		loadedModels = append(loadedModels, "Random Forest")
	} else {
		issues = append(issues, "Random Forest: not loaded")
	}

	if metaModel := GetMetaLearnerModel(); metaModel != nil {
		loadedModels = append(loadedModels, "Meta-Learner")
	} else {
		issues = append(issues, "Meta-Learner: not loaded")
	}

	// Determine status
	totalModels := 7
	status := "healthy"
	message := fmt.Sprintf("All %d models loaded", totalModels)

	if len(loadedModels) == 0 {
		status = "unhealthy"
		message = "No models loaded"
	} else if len(loadedModels) < 3 {
		status = "unhealthy"
		message = fmt.Sprintf("Critical: Only %d/%d models loaded", len(loadedModels), totalModels)
	} else if len(loadedModels) < totalModels {
		status = "degraded"
		message = fmt.Sprintf("Only %d/%d models loaded", len(loadedModels), totalModels)
	}

	return models.HealthCheck{
		Name:        "ML Models",
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"loaded_models": loadedModels,
			"issues":        issues,
			"total_models":  totalModels,
		},
	}
}

// checkMemoryUsage monitors memory consumption
func (hcs *HealthCheckService) checkMemoryUsage() models.HealthCheck {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := m.Alloc / 1024 / 1024
	sysMB := m.Sys / 1024 / 1024

	// Thresholds
	const (
		warningThreshold  = 500  // MB
		criticalThreshold = 1000 // MB
	)

	status := "healthy"
	message := fmt.Sprintf("Memory usage: %dMB / %dMB", allocMB, sysMB)

	if allocMB > criticalThreshold {
		status = "unhealthy"
		message = fmt.Sprintf("Critical memory usage: %dMB (threshold: %dMB)", allocMB, criticalThreshold)
	} else if allocMB > warningThreshold {
		status = "degraded"
		message = fmt.Sprintf("High memory usage: %dMB (threshold: %dMB)", allocMB, warningThreshold)
	}

	return models.HealthCheck{
		Name:        "Memory Usage",
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"alloc_mb":    allocMB,
			"sys_mb":      sysMB,
			"num_gc":      m.NumGC,
			"goroutines":  runtime.NumGoroutine(),
			"warning_mb":  warningThreshold,
			"critical_mb": criticalThreshold,
		},
	}
}

// checkCacheHealth verifies prediction cache is functioning
func (hcs *HealthCheckService) checkCacheHealth() models.HealthCheck {
	cache := GetPredictionCache()
	if cache == nil {
		return models.HealthCheck{
			Name:        "Prediction Cache",
			Status:      "unhealthy",
			Message:     "Cache not initialized",
			LastChecked: time.Now(),
		}
	}

	stats := cache.GetCacheStats()

	status := "healthy"
	message := fmt.Sprintf("%d predictions cached", stats["total_cached"])

	// Check for high degradation rate
	if degradedCount, ok := stats["degraded_count"].(int); ok {
		if totalCached, ok := stats["total_cached"].(int); ok && totalCached > 0 {
			degradationRate := float64(degradedCount) / float64(totalCached)
			if degradationRate > 0.5 {
				status = "degraded"
				message = fmt.Sprintf("High degradation rate: %.1f%%", degradationRate*100)
			}
		}
	}

	return models.HealthCheck{
		Name:        "Prediction Cache",
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Details:     stats,
	}
}

// determineOverallStatus calculates overall system status from individual checks
func (hcs *HealthCheckService) determineOverallStatus(checks map[string]models.HealthCheck) string {
	unhealthyCount := 0
	degradedCount := 0

	for _, check := range checks {
		switch check.Status {
		case "unhealthy":
			unhealthyCount++
		case "degraded":
			degradedCount++
		}
	}

	// If any check is unhealthy, system is unhealthy
	if unhealthyCount > 0 {
		return "unhealthy"
	}

	// If any check is degraded, system is degraded
	if degradedCount > 0 {
		return "degraded"
	}

	return "healthy"
}

// checkAPICache verifies the API response cache is functioning
func (hcs *HealthCheckService) checkAPICache() models.HealthCheck {
	cache := GetAPICacheService()
	if cache == nil {
		return models.HealthCheck{
			Name:        "API Cache",
			Status:      "unhealthy",
			Message:     "API cache service not initialized",
			LastChecked: time.Now(),
		}
	}

	stats := cache.GetStats()
	cacheSize := stats["cache_size"].(int)
	hitRate := stats["hit_rate"].(string)
	totalRequests := stats["total_requests"].(int64)

	// Determine status based on cache performance
	status := "healthy"
	message := "API cache operational"

	if totalRequests > 100 {
		// If we have enough data, check hit rate
		hits := stats["hits"].(int64)
		hitRatio := float64(hits) / float64(totalRequests)

		if hitRatio < 0.20 { // Less than 20% hit rate
			status = "degraded"
			message = "Low cache hit rate"
		}
	}

	return models.HealthCheck{
		Name:        "API Cache",
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"cache_size":     cacheSize,
			"hit_rate":       hitRate,
			"total_requests": totalRequests,
			"hits":           stats["hits"],
			"misses":         stats["misses"],
			"evictions":      stats["evictions"],
		},
	}
}

// getUptime returns formatted uptime string
func (hcs *HealthCheckService) getUptime() string {
	uptime := time.Since(hcs.startTime)

	days := int(uptime.Hours() / 24)
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
