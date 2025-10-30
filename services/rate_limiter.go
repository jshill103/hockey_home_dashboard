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
)

// NHLRateLimiter provides global rate limiting for NHL API calls
type NHLRateLimiter struct {
	requests    []time.Time
	maxRequests int           // Maximum requests in time window
	timeWindow  time.Duration // Time window for rate limiting
	minDelay    time.Duration // Minimum delay between consecutive requests
	mutex       sync.Mutex
	dataDir     string // Directory for metrics persistence

	// Metrics
	totalRequests   int64
	delayedRequests int64
	totalWaitTime   time.Duration
}

// Global NHL API rate limiter (singleton)
var (
	globalNHLRateLimiter *NHLRateLimiter
	rateLimiterOnce      sync.Once
)

// GetNHLRateLimiter returns the global NHL API rate limiter instance
func GetNHLRateLimiter() *NHLRateLimiter {
	rateLimiterOnce.Do(func() {
		globalNHLRateLimiter = &NHLRateLimiter{
			requests:    make([]time.Time, 0, 100),
			maxRequests: 60,                     // 60 requests per minute (conservative)
			timeWindow:  time.Minute,            // 1 minute window
			minDelay:    500 * time.Millisecond, // 500ms between calls (polite)
			dataDir:     "data/metrics",
		}

		// Create data directory
		os.MkdirAll(globalNHLRateLimiter.dataDir, 0755)

		// Load existing metrics
		if err := globalNHLRateLimiter.loadMetrics(); err != nil {
			log.Printf("⚠️ Could not load rate limiter metrics: %v (starting fresh)", err)
		}

		log.Printf("🛡️ NHL API Rate Limiter initialized: %d req/%v, %v min delay",
			globalNHLRateLimiter.maxRequests,
			globalNHLRateLimiter.timeWindow,
			globalNHLRateLimiter.minDelay)

		// Start periodic metrics save (every 5 minutes)
		go globalNHLRateLimiter.periodicSave()
	})
	return globalNHLRateLimiter
}

// Wait blocks until a request is allowed by the rate limiter
// This ensures we never exceed the configured rate limits
func (rl *NHLRateLimiter) Wait() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	rl.totalRequests++

	// Record API call in system stats
	systemStatsServ := GetSystemStatsService()
	if systemStatsServ != nil {
		systemStatsServ.IncrementAPIRequest()
	}

	// Step 1: Clean old requests outside the time window
	validRequests := make([]time.Time, 0, len(rl.requests))
	for _, reqTime := range rl.requests {
		if now.Sub(reqTime) < rl.timeWindow {
			validRequests = append(validRequests, reqTime)
		}
	}
	rl.requests = validRequests

	// Step 2: Check if we've hit the rate limit (requests per time window)
	if len(rl.requests) >= rl.maxRequests {
		// We're at the limit, wait until the oldest request expires
		oldestRequest := rl.requests[0]
		waitTime := rl.timeWindow - now.Sub(oldestRequest)

		if waitTime > 0 {
			rl.delayedRequests++
			rl.totalWaitTime += waitTime

			log.Printf("⏳ Rate limit reached (%d/%d requests in %v), waiting %.2fs...",
				len(rl.requests), rl.maxRequests, rl.timeWindow, waitTime.Seconds())

			// Unlock mutex while sleeping to allow other goroutines to check status
			// Use defer/recover to ensure mutex is re-locked even if panic occurs
			rl.mutex.Unlock()
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("⚠️ Panic during rate limiter sleep: %v", r)
					}
				}()
				time.Sleep(waitTime + 100*time.Millisecond) // Add 100ms buffer
			}()
			rl.mutex.Lock()

			// Re-clean after sleep
			now = time.Now()
			validRequests = make([]time.Time, 0, len(rl.requests))
			for _, reqTime := range rl.requests {
				if now.Sub(reqTime) < rl.timeWindow {
					validRequests = append(validRequests, reqTime)
				}
			}
			rl.requests = validRequests
		}
	}

	// Step 3: Enforce minimum delay between consecutive requests (be polite!)
	if len(rl.requests) > 0 {
		lastRequest := rl.requests[len(rl.requests)-1]
		timeSinceLastRequest := now.Sub(lastRequest)

		if timeSinceLastRequest < rl.minDelay {
			waitTime := rl.minDelay - timeSinceLastRequest
			rl.delayedRequests++
			rl.totalWaitTime += waitTime

			// Unlock mutex while sleeping (with panic recovery)
			rl.mutex.Unlock()
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("⚠️ Panic during rate limiter min delay sleep: %v", r)
					}
				}()
				time.Sleep(waitTime)
			}()
			rl.mutex.Lock()
		}
	}

	// Step 4: Record this request
	rl.requests = append(rl.requests, time.Now())
}

// GetMetrics returns current rate limiter metrics
func (rl *NHLRateLimiter) GetMetrics() RateLimiterMetrics {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Count requests in current window
	requestsInWindow := 0
	for _, reqTime := range rl.requests {
		if now.Sub(reqTime) < rl.timeWindow {
			requestsInWindow++
		}
	}

	avgWaitTime := time.Duration(0)
	if rl.delayedRequests > 0 {
		avgWaitTime = rl.totalWaitTime / time.Duration(rl.delayedRequests)
	}

	return RateLimiterMetrics{
		TotalRequests:      rl.totalRequests,
		RequestsInWindow:   requestsInWindow,
		MaxRequests:        rl.maxRequests,
		TimeWindow:         rl.timeWindow,
		MinDelay:           rl.minDelay,
		DelayedRequests:    rl.delayedRequests,
		TotalWaitTime:      rl.totalWaitTime,
		AverageWaitTime:    avgWaitTime,
		CurrentUtilization: float64(requestsInWindow) / float64(rl.maxRequests) * 100,
	}
}

// RateLimiterMetrics contains rate limiter statistics
type RateLimiterMetrics struct {
	Timestamp          time.Time     `json:"timestamp"`
	TotalRequests      int64         `json:"totalRequests"`
	RequestsInWindow   int           `json:"requestsInWindow"`
	MaxRequests        int           `json:"maxRequests"`
	TimeWindow         time.Duration `json:"timeWindow"`
	MinDelay           time.Duration `json:"minDelay"`
	DelayedRequests    int64         `json:"delayedRequests"`
	TotalWaitTime      time.Duration `json:"totalWaitTime"`
	AverageWaitTime    time.Duration `json:"averageWaitTime"`
	CurrentUtilization float64       `json:"currentUtilization"` // Percentage
}

// LogMetrics logs current rate limiter metrics
func (rl *NHLRateLimiter) LogMetrics() {
	metrics := rl.GetMetrics()

	log.Printf("📊 NHL API Rate Limiter Metrics:")
	log.Printf("   Total Requests: %d", metrics.TotalRequests)
	log.Printf("   Current Window: %d/%d (%.1f%% utilization)",
		metrics.RequestsInWindow, metrics.MaxRequests, metrics.CurrentUtilization)
	log.Printf("   Delayed Requests: %d (%.1f%%)",
		metrics.DelayedRequests,
		float64(metrics.DelayedRequests)/float64(metrics.TotalRequests)*100)
	log.Printf("   Total Wait Time: %v", metrics.TotalWaitTime)
	log.Printf("   Average Wait: %v", metrics.AverageWaitTime)
}

// Reset clears all rate limiter history (useful for testing)
func (rl *NHLRateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.requests = make([]time.Time, 0, 100)
	rl.totalRequests = 0
	rl.delayedRequests = 0
	rl.totalWaitTime = 0

	log.Printf("🔄 Rate limiter reset")
}

// ============================================================================
// PERSISTENCE
// ============================================================================

// saveMetrics saves rate limiter metrics to disk
func (rl *NHLRateLimiter) saveMetrics() error {
	filePath := filepath.Join(rl.dataDir, "rate_limiter_metrics.json")

	metrics := rl.GetMetrics()
	metrics.Timestamp = time.Now()

	jsonData, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling rate limiter metrics: %w", err)
	}

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing rate limiter metrics: %w", err)
	}

	return nil
}

// loadMetrics loads rate limiter metrics from disk
func (rl *NHLRateLimiter) loadMetrics() error {
	filePath := filepath.Join(rl.dataDir, "rate_limiter_metrics.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("rate limiter metrics file not found")
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading rate limiter metrics: %w", err)
	}

	var metrics RateLimiterMetrics
	err = json.Unmarshal(jsonData, &metrics)
	if err != nil {
		return fmt.Errorf("error unmarshaling rate limiter metrics: %w", err)
	}

	// Restore cumulative metrics (not current window)
	rl.totalRequests = metrics.TotalRequests
	rl.delayedRequests = metrics.DelayedRequests
	rl.totalWaitTime = metrics.TotalWaitTime

	log.Printf("📊 Loaded rate limiter metrics: %d total requests, %d delayed",
		rl.totalRequests, rl.delayedRequests)

	return nil
}

// periodicSave saves metrics every 5 minutes
func (rl *NHLRateLimiter) periodicSave() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := rl.saveMetrics(); err != nil {
			log.Printf("⚠️ Failed to save rate limiter metrics: %v", err)
		} else {
			log.Printf("💾 Rate limiter metrics saved")
		}
	}
}
