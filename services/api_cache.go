package services

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a cached API response
type CacheEntry struct {
	Key        string    `json:"key"`
	Data       []byte    `json:"data"`
	CachedAt   time.Time `json:"cached_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Endpoint   string    `json:"endpoint"`
	HitCount   int       `json:"hit_count"`
	LastAccess time.Time `json:"last_access"`
}

// APICacheService manages caching of NHL API responses
type APICacheService struct {
	cache     map[string]*CacheEntry
	cacheMu   sync.RWMutex
	cacheDir  string
	hits      int64
	misses    int64
	statsMu   sync.RWMutex
	evictions int64
}

var (
	apiCacheService     *APICacheService
	apiCacheServiceOnce sync.Once
)

const (
	apiCacheDir = "data/cache/api"
)

// InitAPICacheService initializes the global API cache service
func InitAPICacheService() {
	apiCacheServiceOnce.Do(func() {
		apiCacheService = &APICacheService{
			cache:    make(map[string]*CacheEntry),
			cacheDir: apiCacheDir,
		}
		apiCacheService.LoadCache()
		// Start background cleanup goroutine
		go apiCacheService.cleanupExpiredEntries()
	})
}

// GetAPICacheService returns the singleton instance
func GetAPICacheService() *APICacheService {
	return apiCacheService
}

// generateCacheKey creates a consistent cache key from the URL
func (acs *APICacheService) generateCacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes for shorter keys
}

// Get retrieves a cached response if it exists and is not expired
func (acs *APICacheService) Get(url string) ([]byte, bool) {
	key := acs.generateCacheKey(url)

	acs.cacheMu.RLock()
	entry, exists := acs.cache[key]
	acs.cacheMu.RUnlock()

	if !exists {
		acs.recordMiss()
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		acs.recordMiss()
		// Remove expired entry
		acs.cacheMu.Lock()
		delete(acs.cache, key)
		acs.cacheMu.Unlock()
		return nil, false
	}

	// Update hit count and last access
	acs.cacheMu.Lock()
	entry.HitCount++
	entry.LastAccess = time.Now()
	acs.cacheMu.Unlock()

	acs.recordHit()
	return entry.Data, true
}

// Set stores an API response in the cache with a TTL
func (acs *APICacheService) Set(url string, data []byte, ttl time.Duration) {
	key := acs.generateCacheKey(url)

	entry := &CacheEntry{
		Key:        key,
		Data:       data,
		CachedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(ttl),
		Endpoint:   url,
		HitCount:   0,
		LastAccess: time.Now(),
	}

	acs.cacheMu.Lock()
	acs.cache[key] = entry
	acs.cacheMu.Unlock()
}

// GetTTLForEndpoint returns the appropriate TTL based on the endpoint type
func GetTTLForEndpoint(url string) time.Duration {
	// Parse URL to determine cache duration
	switch {
	// Live/real-time data - very short cache
	case contains(url, "/gamecenter/") && contains(url, "/play-by-play"):
		return 30 * time.Second // Play-by-play updates frequently during games
	case contains(url, "/gamecenter/") && contains(url, "/boxscore"):
		return 1 * time.Minute // Boxscores update every minute during games
	case contains(url, "/scoreboard/now"):
		return 1 * time.Minute // Current scores update frequently

	// Player/team stats - medium cache
	case contains(url, "/club-stats/"):
		return 1 * time.Hour // Team stats update daily
	case contains(url, "/player/") && contains(url, "/game-log"):
		return 1 * time.Hour // Player game logs update after games
	case contains(url, "/standings/now"):
		return 5 * time.Minute // Standings update after games

	// Schedule data - longer cache
	case contains(url, "/club-schedule/"):
		return 24 * time.Hour // Schedules rarely change
	case contains(url, "/club-schedule-season/"):
		return 24 * time.Hour // Season schedules are static

	// Roster data - very long cache
	case contains(url, "/roster/") || contains(url, "/roster-season/"):
		return 24 * time.Hour // Rosters change infrequently

	// Default for unknown endpoints
	default:
		return 5 * time.Minute
	}
}

// contains is a helper to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Invalidate removes a specific cache entry
func (acs *APICacheService) Invalidate(url string) {
	key := acs.generateCacheKey(url)
	acs.cacheMu.Lock()
	delete(acs.cache, key)
	acs.cacheMu.Unlock()
}

// InvalidatePattern removes all cache entries matching a pattern
func (acs *APICacheService) InvalidatePattern(pattern string) int {
	count := 0
	acs.cacheMu.Lock()
	for key, entry := range acs.cache {
		if contains(entry.Endpoint, pattern) {
			delete(acs.cache, key)
			count++
		}
	}
	acs.cacheMu.Unlock()
	acs.statsMu.Lock()
	acs.evictions += int64(count)
	acs.statsMu.Unlock()
	return count
}

// Clear removes all cache entries
func (acs *APICacheService) Clear() {
	acs.cacheMu.Lock()
	count := len(acs.cache)
	acs.cache = make(map[string]*CacheEntry)
	acs.cacheMu.Unlock()
	acs.statsMu.Lock()
	acs.evictions += int64(count)
	acs.statsMu.Unlock()
}

// GetStats returns cache statistics
func (acs *APICacheService) GetStats() map[string]interface{} {
	acs.cacheMu.RLock()
	cacheSize := len(acs.cache)
	acs.cacheMu.RUnlock()

	acs.statsMu.RLock()
	hits := acs.hits
	misses := acs.misses
	evictions := acs.evictions
	acs.statsMu.RUnlock()

	total := hits + misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"cache_size":     cacheSize,
		"hits":           hits,
		"misses":         misses,
		"evictions":      evictions,
		"hit_rate":       fmt.Sprintf("%.1f%%", hitRate),
		"total_requests": total,
	}
}

// recordHit increments the hit counter
func (acs *APICacheService) recordHit() {
	acs.statsMu.Lock()
	acs.hits++
	acs.statsMu.Unlock()
}

// recordMiss increments the miss counter
func (acs *APICacheService) recordMiss() {
	acs.statsMu.Lock()
	acs.misses++
	acs.statsMu.Unlock()
}

// cleanupExpiredEntries removes expired entries every 5 minutes
func (acs *APICacheService) cleanupExpiredEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		count := 0

		acs.cacheMu.Lock()
		for key, entry := range acs.cache {
			if now.After(entry.ExpiresAt) {
				delete(acs.cache, key)
				count++
			}
		}
		acs.cacheMu.Unlock()

		if count > 0 {
			acs.statsMu.Lock()
			acs.evictions += int64(count)
			acs.statsMu.Unlock()
			fmt.Printf("🗑️ Cleaned up %d expired cache entries\n", count)
		}
	}
}

// LoadCache loads the cache from disk
func (acs *APICacheService) LoadCache() {
	cacheFile := filepath.Join(acs.cacheDir, "api_cache.json")

	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		// Cache file doesn't exist yet - that's fine
		fmt.Printf("📦 No existing API cache found, starting fresh\n")
		return
	}

	var entries []*CacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		fmt.Printf("⚠️ Failed to unmarshal API cache: %v\n", err)
		return
	}

	now := time.Now()
	loaded := 0
	expired := 0

	acs.cacheMu.Lock()
	for _, entry := range entries {
		// Only load entries that haven't expired
		if now.Before(entry.ExpiresAt) {
			acs.cache[entry.Key] = entry
			loaded++
		} else {
			expired++
		}
	}
	acs.cacheMu.Unlock()

	fmt.Printf("✅ API cache loaded: %d entries (skipped %d expired)\n", loaded, expired)
}

// SaveCache persists the cache to disk
func (acs *APICacheService) SaveCache() error {
	acs.cacheMu.RLock()
	entries := make([]*CacheEntry, 0, len(acs.cache))
	for _, entry := range acs.cache {
		entries = append(entries, entry)
	}
	acs.cacheMu.RUnlock()

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	cacheFile := filepath.Join(acs.cacheDir, "api_cache.json")
	if err := ioutil.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}
