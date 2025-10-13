package services

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// LazyCache provides lazy-loaded data with disk persistence
// Data is only loaded from disk when first accessed, reducing memory usage
type LazyCache[T any] struct {
	data       *T
	diskPath   string
	loaded     bool
	lastAccess time.Time
	ttl        time.Duration
	mu         sync.RWMutex

	// Stats
	loadCount int
	saveCount int
	hitCount  int
	missCount int
}

// NewLazyCache creates a new lazy cache with disk persistence
func NewLazyCache[T any](diskPath string, ttl time.Duration) *LazyCache[T] {
	return &LazyCache[T]{
		diskPath: diskPath,
		ttl:      ttl,
	}
}

// Get returns the cached data, loading from disk if necessary
func (lc *LazyCache[T]) Get() (*T, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// Check if data is already loaded and fresh
	if lc.loaded && lc.data != nil {
		if lc.ttl == 0 || time.Since(lc.lastAccess) < lc.ttl {
			lc.hitCount++
			lc.lastAccess = time.Now()
			return lc.data, nil
		}
	}

	// Data not loaded or stale - load from disk
	lc.missCount++
	if err := lc.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load from disk: %w", err)
	}

	lc.lastAccess = time.Now()
	return lc.data, nil
}

// Set updates the cached data and optionally saves to disk
func (lc *LazyCache[T]) Set(data *T, saveToDisk bool) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.data = data
	lc.loaded = true
	lc.lastAccess = time.Now()

	if saveToDisk {
		return lc.saveToDiskInternal()
	}

	return nil
}

// loadFromDisk loads data from disk (internal, assumes lock is held)
func (lc *LazyCache[T]) loadFromDisk() error {
	if _, err := os.Stat(lc.diskPath); os.IsNotExist(err) {
		// File doesn't exist - initialize with zero value
		var zero T
		lc.data = &zero
		lc.loaded = true
		return nil
	}

	fileData, err := os.ReadFile(lc.diskPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var data T
	if err := json.Unmarshal(fileData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	lc.data = &data
	lc.loaded = true
	lc.loadCount++

	return nil
}

// saveToDiskInternal saves data to disk (internal, assumes lock is held)
func (lc *LazyCache[T]) saveToDiskInternal() error {
	if lc.data == nil {
		return fmt.Errorf("no data to save")
	}

	fileData, err := json.MarshalIndent(lc.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(lc.diskPath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	lc.saveCount++
	return nil
}

// SaveToDisk explicitly saves current data to disk
func (lc *LazyCache[T]) SaveToDisk() error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if !lc.loaded || lc.data == nil {
		return fmt.Errorf("no data loaded to save")
	}

	return lc.saveToDiskInternal()
}

// Unload clears data from memory (keeps on disk)
func (lc *LazyCache[T]) Unload() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.data = nil
	lc.loaded = false
}

// IsLoaded returns whether data is currently in memory
func (lc *LazyCache[T]) IsLoaded() bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	return lc.loaded
}

// GetStats returns cache statistics
func (lc *LazyCache[T]) GetStats() map[string]interface{} {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	hitRate := 0.0
	totalAccess := lc.hitCount + lc.missCount
	if totalAccess > 0 {
		hitRate = float64(lc.hitCount) / float64(totalAccess) * 100.0
	}

	return map[string]interface{}{
		"loaded":      lc.loaded,
		"last_access": lc.lastAccess.Format(time.RFC3339),
		"ttl":         lc.ttl.String(),
		"load_count":  lc.loadCount,
		"save_count":  lc.saveCount,
		"hit_count":   lc.hitCount,
		"miss_count":  lc.missCount,
		"hit_rate":    fmt.Sprintf("%.1f%%", hitRate),
	}
}

// LazyCacheManager manages multiple lazy caches
type LazyCacheManager struct {
	caches map[string]interface{} // map of cache name to LazyCache
	mu     sync.RWMutex
}

var (
	lazyCacheManager     *LazyCacheManager
	lazyCacheManagerOnce sync.Once
)

// InitLazyCacheManager initializes the global cache manager
func InitLazyCacheManager() {
	lazyCacheManagerOnce.Do(func() {
		lazyCacheManager = &LazyCacheManager{
			caches: make(map[string]interface{}),
		}
		fmt.Println("✅ Lazy Cache Manager initialized")
	})
}

// GetLazyCacheManager returns the singleton instance
func GetLazyCacheManager() *LazyCacheManager {
	return lazyCacheManager
}

// Register adds a cache to the manager
func (lcm *LazyCacheManager) Register(name string, cache interface{}) {
	if lcm == nil {
		return
	}

	lcm.mu.Lock()
	defer lcm.mu.Unlock()

	lcm.caches[name] = cache
}

// UnloadAll unloads all registered caches from memory
func (lcm *LazyCacheManager) UnloadAll() {
	if lcm == nil {
		return
	}

	lcm.mu.RLock()
	defer lcm.mu.RUnlock()

	for name, cache := range lcm.caches {
		switch c := cache.(type) {
		case interface{ Unload() }:
			c.Unload()
			fmt.Printf("🗑️ Unloaded cache: %s\n", name)
		}
	}
}

// SaveAll saves all registered caches to disk
func (lcm *LazyCacheManager) SaveAll() error {
	if lcm == nil {
		return nil
	}

	lcm.mu.RLock()
	defer lcm.mu.RUnlock()

	var errors []error
	for name, cache := range lcm.caches {
		switch c := cache.(type) {
		case interface{ SaveToDisk() error }:
			if err := c.SaveToDisk(); err != nil {
				errors = append(errors, fmt.Errorf("%s: %w", name, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to save %d caches: %v", len(errors), errors)
	}

	return nil
}

// GetAllStats returns statistics for all registered caches
func (lcm *LazyCacheManager) GetAllStats() map[string]interface{} {
	if lcm == nil {
		return nil
	}

	lcm.mu.RLock()
	defer lcm.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, cache := range lcm.caches {
		switch c := cache.(type) {
		case interface{ GetStats() map[string]interface{} }:
			stats[name] = c.GetStats()
		}
	}

	return stats
}
