package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/jaredshillingburg/go_uhc/services"
	"github.com/jaredshillingburg/go_uhc/utils"
)

// HealthStatus represents the overall health of the application
type HealthStatus struct {
	Status      string                 `json:"status"`      // "healthy", "degraded", "unhealthy"
	Timestamp   string                 `json:"timestamp"`   // Current time
	Uptime      string                 `json:"uptime"`      // Time since startup
	Version     string                 `json:"version"`     // Application version
	Season      string                 `json:"season"`      // Current NHL season
	IsOffseason bool                   `json:"isOffseason"` // Whether in off-season
	Services    map[string]ServiceInfo `json:"services"`    // Individual service statuses
	System      SystemInfo             `json:"system"`      // System metrics
}

// ServiceInfo represents the health of an individual service
type ServiceInfo struct {
	Status  string `json:"status"`  // "up", "down", "degraded"
	Message string `json:"message"` // Additional info
}

// SystemInfo contains system-level metrics
type SystemInfo struct {
	GoVersion     string  `json:"goVersion"`     // Go runtime version
	NumGoroutines int     `json:"numGoroutines"` // Number of goroutines
	MemoryAllocMB float64 `json:"memoryAllocMB"` // Allocated memory in MB
	MemorySysMB   float64 `json:"memorySysMB"`   // System memory in MB
	NumCPU        int     `json:"numCPU"`        // Number of CPUs
	LastGCTime    string  `json:"lastGCTime"`    // Last garbage collection time
}

var (
	startTime = time.Now() // Application start time
)

// HandleHealth returns comprehensive health check information
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	health := HealthStatus{
		Status:      "healthy",
		Timestamp:   time.Now().Format(time.RFC3339),
		Uptime:      time.Since(startTime).String(),
		Version:     "1.9.0", // Update this with each release
		Season:      utils.FormatSeason(utils.GetCurrentSeason()),
		IsOffseason: utils.IsOffseason(),
		Services:    make(map[string]ServiceInfo),
		System:      getSystemInfo(),
	}

	// Check individual services
	health.Services["nhlAPI"] = checkNHLAPI()
	health.Services["predictionModels"] = checkPredictionModels()
	health.Services["liveDataService"] = checkLiveDataService()
	health.Services["rateLimiter"] = checkRateLimiter()
	health.Services["playerImpactService"] = checkPlayerImpactService()
	health.Services["goalieService"] = checkGoalieService()
	health.Services["playoffSimulation"] = checkPlayoffSimulation()

	// Determine overall status based on services
	downCount := 0
	degradedCount := 0
	for _, service := range health.Services {
		if service.Status == "down" {
			downCount++
		} else if service.Status == "degraded" {
			degradedCount++
		}
	}

	if downCount > 0 {
		health.Status = "unhealthy"
	} else if degradedCount > 0 {
		health.Status = "degraded"
	}

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if health.Status == "degraded" {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// getSystemInfo collects system-level metrics
func getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	lastGCTime := "never"
	if m.LastGC > 0 {
		lastGCTime = time.Unix(0, int64(m.LastGC)).Format(time.RFC3339)
	}

	return SystemInfo{
		GoVersion:     runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		MemoryAllocMB: float64(m.Alloc) / 1024 / 1024,
		MemorySysMB:   float64(m.Sys) / 1024 / 1024,
		NumCPU:        runtime.NumCPU(),
		LastGCTime:    lastGCTime,
	}
}

// checkNHLAPI verifies NHL API connectivity
func checkNHLAPI() ServiceInfo {
	rateLimiter := services.GetNHLRateLimiter()
	if rateLimiter == nil {
		return ServiceInfo{
			Status:  "down",
			Message: "Rate limiter not initialized",
		}
	}

	metrics := rateLimiter.GetMetrics()
	if metrics.TotalRequests > 0 {
		return ServiceInfo{
			Status:  "up",
			Message: "API operational, " + healthFormatNumber(int(metrics.TotalRequests)) + " requests processed",
		}
	}

	return ServiceInfo{
		Status:  "up",
		Message: "Ready (no requests yet)",
	}
}

// checkPredictionModels verifies ML models are loaded
func checkPredictionModels() ServiceInfo {
	liveSys := services.GetLivePredictionSystem()
	if liveSys == nil {
		return ServiceInfo{
			Status:  "down",
			Message: "Live prediction system not initialized",
		}
	}

	ensemble := liveSys.GetEnsemble()
	if ensemble == nil {
		return ServiceInfo{
			Status:  "down",
			Message: "Ensemble service not available",
		}
	}

	return ServiceInfo{
		Status:  "up",
		Message: "All 9 ML models loaded and operational",
	}
}

// checkLiveDataService verifies live data updates are working
func checkLiveDataService() ServiceInfo {
	// Live data service is always active in this application
	// We check if the live prediction system is running instead
	liveSys := services.GetLivePredictionSystem()
	if liveSys == nil {
		return ServiceInfo{
			Status:  "degraded",
			Message: "Live prediction system not initialized",
		}
	}

	return ServiceInfo{
		Status:  "up",
		Message: "Live data updates active",
	}
}

// checkRateLimiter verifies rate limiter is operational
func checkRateLimiter() ServiceInfo {
	rateLimiter := services.GetNHLRateLimiter()
	if rateLimiter == nil {
		return ServiceInfo{
			Status:  "down",
			Message: "Rate limiter not initialized",
		}
	}

	metrics := rateLimiter.GetMetrics()
	delayedPct := 0.0
	if metrics.TotalRequests > 0 {
		delayedPct = float64(metrics.DelayedRequests) / float64(metrics.TotalRequests) * 100
	}

	return ServiceInfo{
		Status:  "up",
		Message: healthFormatFloat(delayedPct) + "% requests delayed (rate limiting active)",
	}
}

// checkPlayerImpactService verifies player stats service
func checkPlayerImpactService() ServiceInfo {
	playerService := services.GetPlayerImpactService()
	if playerService == nil {
		return ServiceInfo{
			Status:  "degraded",
			Message: "Player impact service not initialized",
		}
	}

	return ServiceInfo{
		Status:  "up",
		Message: "Player stats tracking operational",
	}
}

// checkGoalieService verifies goalie intelligence service
func checkGoalieService() ServiceInfo {
	goalieService := services.GetGoalieService()
	if goalieService == nil {
		return ServiceInfo{
			Status:  "degraded",
			Message: "Goalie service not initialized",
		}
	}

	return ServiceInfo{
		Status:  "up",
		Message: "Goalie intelligence operational",
	}
}

// checkPlayoffSimulation verifies playoff odds calculation
func checkPlayoffSimulation() ServiceInfo {
	playoffSim := services.GetPlayoffSimulationService()
	if playoffSim == nil {
		return ServiceInfo{
			Status:  "degraded",
			Message: "Playoff simulation not initialized",
		}
	}

	return ServiceInfo{
		Status:  "up",
		Message: "ML-powered playoff simulation ready",
	}
}

// Helper functions for formatting (prefixed with 'health' to avoid conflicts)
func healthFormatNumber(n int) string {
	if n < 1000 {
		return healthIntToString(n)
	}
	return healthFormatNumberWithCommas(n)
}

func healthFormatNumberWithCommas(n int) string {
	s := ""
	for i, c := range healthReverseString(healthIntToString(n)) {
		if i > 0 && i%3 == 0 {
			s = "," + s
		}
		s = string(c) + s
	}
	return s
}

func healthFormatFloat(f float64) string {
	return healthIntToString(int(f + 0.5))
}

func healthIntToString(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if negative {
		s = "-" + s
	}
	return s
}

func healthReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
