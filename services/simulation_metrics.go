package services

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// SimulationMetrics tracks performance metrics for playoff simulations (Phase 5.5)
type SimulationMetrics struct {
	TotalSimulations      int64         `json:"totalSimulations"`
	TotalDuration         time.Duration `json:"totalDuration"`
	AvgSimulationTime     time.Duration `json:"avgSimulationTime"`
	FastestSimulation     time.Duration `json:"fastestSimulation"`
	SlowestSimulation     time.Duration `json:"slowestSimulation"`
	ParallelSimulations   int64         `json:"parallelSimulations"`
	SequentialSimulations int64         `json:"sequentialSimulations"`
	CacheHits             int64         `json:"cacheHits"`
	CacheMisses           int64         `json:"cacheMisses"`
	WhatIfCacheHits       int64         `json:"whatIfCacheHits"`
	WhatIfCacheMisses     int64         `json:"whatIfCacheMisses"`
	LastUpdated           time.Time     `json:"lastUpdated"`
	mu                    sync.RWMutex
}

var globalSimulationMetrics = &SimulationMetrics{
	FastestSimulation: time.Hour, // Initialize to high value
}

// RecordSimulation records metrics for a completed simulation (Phase 5.5)
func (sm *SimulationMetrics) RecordSimulation(duration time.Duration, count int, parallel bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sm.TotalSimulations += int64(count)
	sm.TotalDuration += duration
	
	if parallel {
		sm.ParallelSimulations += int64(count)
	} else {
		sm.SequentialSimulations += int64(count)
	}
	
	if duration < sm.FastestSimulation {
		sm.FastestSimulation = duration
	}
	if duration > sm.SlowestSimulation {
		sm.SlowestSimulation = duration
	}
	
	// Calculate average (with division by zero protection)
	if sm.TotalSimulations > 0 {
		sm.AvgSimulationTime = time.Duration(int64(sm.TotalDuration) / sm.TotalSimulations)
	}
	sm.LastUpdated = time.Now()
}

// RecordCacheHit records a cache hit (Phase 5.5)
func (sm *SimulationMetrics) RecordCacheHit(whatIf bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if whatIf {
		sm.WhatIfCacheHits++
	} else {
		sm.CacheHits++
	}
	sm.LastUpdated = time.Now()
}

// RecordCacheMiss records a cache miss (Phase 5.5)
func (sm *SimulationMetrics) RecordCacheMiss(whatIf bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if whatIf {
		sm.WhatIfCacheMisses++
	} else {
		sm.CacheMisses++
	}
	sm.LastUpdated = time.Now()
}

// GetMetrics returns a copy of current metrics (Phase 5.5)
func (sm *SimulationMetrics) GetMetrics() *SimulationMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	return &SimulationMetrics{
		TotalSimulations:      sm.TotalSimulations,
		TotalDuration:         sm.TotalDuration,
		AvgSimulationTime:     sm.AvgSimulationTime,
		FastestSimulation:     sm.FastestSimulation,
		SlowestSimulation:     sm.SlowestSimulation,
		ParallelSimulations:   sm.ParallelSimulations,
		SequentialSimulations: sm.SequentialSimulations,
		CacheHits:             sm.CacheHits,
		CacheMisses:           sm.CacheMisses,
		WhatIfCacheHits:       sm.WhatIfCacheHits,
		WhatIfCacheMisses:     sm.WhatIfCacheMisses,
		LastUpdated:           sm.LastUpdated,
	}
}

// GetCacheHitRate returns the cache hit rate as a percentage (Phase 5.5)
func (sm *SimulationMetrics) GetCacheHitRate() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	total := sm.CacheHits + sm.CacheMisses
	if total == 0 {
		return 0.0
	}
	return float64(sm.CacheHits) / float64(total) * 100.0
}

// GetWhatIfCacheHitRate returns the what-if cache hit rate as a percentage (Phase 5.5)
func (sm *SimulationMetrics) GetWhatIfCacheHitRate() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	total := sm.WhatIfCacheHits + sm.WhatIfCacheMisses
	if total == 0 {
		return 0.0
	}
	return float64(sm.WhatIfCacheHits) / float64(total) * 100.0
}

// Reset resets all metrics (Phase 5.5)
func (sm *SimulationMetrics) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	*sm = SimulationMetrics{
		FastestSimulation: time.Hour,
		LastUpdated:       time.Now(),
	}
}

// PrintSummary prints a human-readable summary of metrics (Phase 5.5)
func (sm *SimulationMetrics) PrintSummary() {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	fmt.Println("\n📊 Simulation Performance Metrics:")
	fmt.Printf("   Total Simulations: %d\n", sm.TotalSimulations)
	fmt.Printf("   Parallel: %d (%.1f%%), Sequential: %d (%.1f%%)\n",
		sm.ParallelSimulations,
		float64(sm.ParallelSimulations)/float64(sm.TotalSimulations)*100.0,
		sm.SequentialSimulations,
		float64(sm.SequentialSimulations)/float64(sm.TotalSimulations)*100.0)
	fmt.Printf("   Total Duration: %v\n", sm.TotalDuration)
	fmt.Printf("   Avg Simulation: %v\n", sm.AvgSimulationTime)
	fmt.Printf("   Fastest: %v, Slowest: %v\n", sm.FastestSimulation, sm.SlowestSimulation)
	
	if sm.CacheHits+sm.CacheMisses > 0 {
		fmt.Printf("   Cache Hit Rate: %.1f%% (%d hits, %d misses)\n",
			sm.GetCacheHitRate(), sm.CacheHits, sm.CacheMisses)
	}
	
	if sm.WhatIfCacheHits+sm.WhatIfCacheMisses > 0 {
		fmt.Printf("   What-If Cache Hit Rate: %.1f%% (%d hits, %d misses)\n",
			sm.GetWhatIfCacheHitRate(), sm.WhatIfCacheHits, sm.WhatIfCacheMisses)
	}
	
	fmt.Printf("   Last Updated: %s\n", sm.LastUpdated.Format("2006-01-02 15:04:05"))
}

// SaveToFile persists metrics to a JSON file (Phase 5.5)
func (sm *SimulationMetrics) SaveToFile(filepath string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	data, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}
	
	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write metrics file: %v", err)
	}
	
	return nil
}

// LoadFromFile loads metrics from a JSON file (Phase 5.5)
func (sm *SimulationMetrics) LoadFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, not an error
		}
		return fmt.Errorf("failed to read metrics file: %v", err)
	}
	
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	err = json.Unmarshal(data, sm)
	if err != nil {
		return fmt.Errorf("failed to unmarshal metrics: %v", err)
	}
	
	return nil
}

// GetGlobalMetrics returns the global simulation metrics instance (Phase 5.5)
func GetGlobalMetrics() *SimulationMetrics {
	return globalSimulationMetrics
}

// RecordSimulationStart returns a function to record simulation completion (Phase 5.5)
func RecordSimulationStart() func(count int, parallel bool) {
	start := time.Now()
	return func(count int, parallel bool) {
		duration := time.Since(start)
		globalSimulationMetrics.RecordSimulation(duration, count, parallel)
	}
}

