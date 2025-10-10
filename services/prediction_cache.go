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

// PredictionCache handles caching of game predictions for graceful degradation
type PredictionCache struct {
	predictions map[int]*CachedPrediction // gameID -> cached prediction
	mu          sync.RWMutex
	cacheDir    string
	ttl         time.Duration
}

// CachedPrediction stores a prediction with metadata
type CachedPrediction struct {
	GameID      int                    `json:"gameId"`
	Prediction  *models.GamePrediction `json:"prediction"`
	CachedAt    time.Time              `json:"cachedAt"`
	HomeTeam    string                 `json:"homeTeam"`
	AwayTeam    string                 `json:"awayTeam"`
	DataQuality float64                `json:"dataQuality"` // 0-1 score
	IsDegraded  bool                   `json:"isDegraded"`  // If this was a degraded prediction
}

var (
	globalPredictionCache *PredictionCache
	predictionCacheOnce   sync.Once
)

// GetPredictionCache returns the global prediction cache instance
func GetPredictionCache() *PredictionCache {
	predictionCacheOnce.Do(func() {
		globalPredictionCache = NewPredictionCache()
	})
	return globalPredictionCache
}

// NewPredictionCache creates a new prediction cache
func NewPredictionCache() *PredictionCache {
	cache := &PredictionCache{
		predictions: make(map[int]*CachedPrediction),
		cacheDir:    "data/cache/predictions",
		ttl:         6 * time.Hour, // Predictions valid for 6 hours
	}

	// Create cache directory
	if err := os.MkdirAll(cache.cacheDir, 0755); err != nil {
		log.Printf("⚠️ Failed to create prediction cache directory: %v", err)
	}

	// Load existing cache from disk
	cache.loadCache()

	log.Println("✅ Prediction Cache initialized")
	return cache
}

// CachePrediction stores a prediction in the cache
func (pc *PredictionCache) CachePrediction(gameID int, homeTeam, awayTeam string, prediction *models.GamePrediction, dataQuality float64, isDegraded bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	cached := &CachedPrediction{
		GameID:      gameID,
		Prediction:  prediction,
		CachedAt:    time.Now(),
		HomeTeam:    homeTeam,
		AwayTeam:    awayTeam,
		DataQuality: dataQuality,
		IsDegraded:  isDegraded,
	}

	pc.predictions[gameID] = cached

	// Save to disk asynchronously
	go pc.savePrediction(cached)

	if isDegraded {
		log.Printf("💾 Cached DEGRADED prediction for game %d: %s vs %s (quality: %.1f%%)",
			gameID, awayTeam, homeTeam, dataQuality*100)
	} else {
		log.Printf("💾 Cached prediction for game %d: %s vs %s (quality: %.1f%%)",
			gameID, awayTeam, homeTeam, dataQuality*100)
	}
}

// GetCachedPrediction retrieves a cached prediction if available and fresh
func (pc *PredictionCache) GetCachedPrediction(gameID int) *CachedPrediction {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	cached, exists := pc.predictions[gameID]
	if !exists {
		return nil
	}

	// Check if cache is still valid
	age := time.Since(cached.CachedAt)
	if age > pc.ttl {
		log.Printf("⏰ Cached prediction for game %d is stale (%.1f hours old)", gameID, age.Hours())
		return nil
	}

	log.Printf("✅ Using cached prediction for game %d (%.1f minutes old, quality: %.1f%%)",
		gameID, age.Minutes(), cached.DataQuality*100)
	return cached
}

// HasFreshPrediction checks if a fresh prediction exists for a game
func (pc *PredictionCache) HasFreshPrediction(gameID int) bool {
	return pc.GetCachedPrediction(gameID) != nil
}

// CleanStaleEntries removes predictions older than TTL
func (pc *PredictionCache) CleanStaleEntries() int {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	now := time.Now()
	removed := 0

	for gameID, cached := range pc.predictions {
		if now.Sub(cached.CachedAt) > pc.ttl {
			delete(pc.predictions, gameID)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("🧹 Cleaned %d stale prediction(s) from cache", removed)
	}

	return removed
}

// savePrediction persists a single prediction to disk
func (pc *PredictionCache) savePrediction(cached *CachedPrediction) {
	filename := filepath.Join(pc.cacheDir, fmt.Sprintf("game_%d.json", cached.GameID))

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		log.Printf("⚠️ Failed to marshal cached prediction: %v", err)
		return
	}

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		log.Printf("⚠️ Failed to save cached prediction: %v", err)
	}
}

// loadCache loads all cached predictions from disk
func (pc *PredictionCache) loadCache() {
	files, err := filepath.Glob(filepath.Join(pc.cacheDir, "game_*.json"))
	if err != nil {
		log.Printf("⚠️ Failed to load prediction cache: %v", err)
		return
	}

	loaded := 0
	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		var cached CachedPrediction
		if err := json.Unmarshal(data, &cached); err != nil {
			continue
		}

		// Only load if not stale
		if time.Since(cached.CachedAt) <= pc.ttl {
			pc.predictions[cached.GameID] = &cached
			loaded++
		}
	}

	if loaded > 0 {
		log.Printf("📂 Loaded %d cached prediction(s) from disk", loaded)
	}
}

// GetCacheStats returns statistics about the cache
func (pc *PredictionCache) GetCacheStats() map[string]interface{} {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	fresh := 0
	degraded := 0
	avgQuality := 0.0

	for _, cached := range pc.predictions {
		if time.Since(cached.CachedAt) <= pc.ttl {
			fresh++
			if cached.IsDegraded {
				degraded++
			}
			avgQuality += cached.DataQuality
		}
	}

	if fresh > 0 {
		avgQuality /= float64(fresh)
	}

	return map[string]interface{}{
		"total_cached":      len(pc.predictions),
		"fresh_predictions": fresh,
		"degraded_count":    degraded,
		"avg_data_quality":  avgQuality,
		"cache_ttl_hours":   pc.ttl.Hours(),
	}
}
