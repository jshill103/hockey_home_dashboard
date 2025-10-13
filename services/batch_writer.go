package services

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// BatchWrite represents a pending write operation
type BatchWrite struct {
	Data      interface{}
	FilePath  string
	Timestamp time.Time
}

// BatchWriter accumulates writes and flushes them in batches
type BatchWriter struct {
	pendingWrites map[string]*BatchWrite // map of filePath to BatchWrite
	mu            sync.Mutex
	flushInterval time.Duration
	maxBatchSize  int
	stopChan      chan struct{}
	wg            sync.WaitGroup

	// Stats
	totalWrites   int
	totalFlushes  int
	writesSkipped int // Writes that were overwritten before flush
}

var (
	batchWriter     *BatchWriter
	batchWriterOnce sync.Once
)

// InitBatchWriter initializes the global batch writer
func InitBatchWriter(flushInterval time.Duration, maxBatchSize int) {
	batchWriterOnce.Do(func() {
		batchWriter = &BatchWriter{
			pendingWrites: make(map[string]*BatchWrite),
			flushInterval: flushInterval,
			maxBatchSize:  maxBatchSize,
			stopChan:      make(chan struct{}),
		}

		// Start background flusher
		batchWriter.wg.Add(1)
		go batchWriter.backgroundFlusher()

		fmt.Printf("✅ Batch Writer initialized (flush: %s, max batch: %d)\n",
			flushInterval, maxBatchSize)
	})
}

// GetBatchWriter returns the singleton instance
func GetBatchWriter() *BatchWriter {
	return batchWriter
}

// QueueWrite adds a write to the batch queue
func (bw *BatchWriter) QueueWrite(filePath string, data interface{}) {
	if bw == nil {
		// Fallback: write immediately if batch writer not initialized
		immediateWrite(filePath, data)
		return
	}

	bw.mu.Lock()
	defer bw.mu.Unlock()

	// Check if we already have a pending write for this file
	if existing, exists := bw.pendingWrites[filePath]; exists {
		bw.writesSkipped++ // Previous write was overwritten
		if DEBUG {
			fmt.Printf("🔄 Batch write overwritten: %s (age: %s)\n",
				filePath, time.Since(existing.Timestamp))
		}
	}

	bw.pendingWrites[filePath] = &BatchWrite{
		Data:      data,
		FilePath:  filePath,
		Timestamp: time.Now(),
	}
	bw.totalWrites++

	// If batch is full, trigger immediate flush
	if len(bw.pendingWrites) >= bw.maxBatchSize {
		if DEBUG {
			fmt.Printf("📦 Batch size limit reached (%d), flushing...\n", bw.maxBatchSize)
		}
		bw.flushInternal()
	}
}

// Flush writes all pending data to disk immediately
func (bw *BatchWriter) Flush() error {
	if bw == nil {
		return nil
	}

	bw.mu.Lock()
	defer bw.mu.Unlock()

	return bw.flushInternal()
}

// flushInternal performs the actual flush (assumes lock is held)
func (bw *BatchWriter) flushInternal() error {
	if len(bw.pendingWrites) == 0 {
		return nil
	}

	startTime := time.Now()
	writeCount := len(bw.pendingWrites)
	errors := []error{}

	// Write all pending data
	for filePath, write := range bw.pendingWrites {
		if err := writeToFile(filePath, write.Data); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", filePath, err))
		}
	}

	// Clear pending writes
	bw.pendingWrites = make(map[string]*BatchWrite)
	bw.totalFlushes++

	duration := time.Since(startTime)
	if DEBUG {
		fmt.Printf("💾 Batch flush: %d writes in %s (%.2f writes/sec)\n",
			writeCount, duration, float64(writeCount)/duration.Seconds())
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch flush failed for %d/%d files: %v",
			len(errors), writeCount, errors)
	}

	return nil
}

// backgroundFlusher periodically flushes pending writes
func (bw *BatchWriter) backgroundFlusher() {
	defer bw.wg.Done()

	ticker := time.NewTicker(bw.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bw.mu.Lock()
			if len(bw.pendingWrites) > 0 {
				if DEBUG {
					fmt.Printf("⏰ Scheduled batch flush (%d pending writes)\n",
						len(bw.pendingWrites))
				}
				bw.flushInternal()
			}
			bw.mu.Unlock()

		case <-bw.stopChan:
			// Final flush before shutdown
			bw.mu.Lock()
			if len(bw.pendingWrites) > 0 {
				fmt.Printf("🛑 Final batch flush (%d pending writes)\n",
					len(bw.pendingWrites))
				bw.flushInternal()
			}
			bw.mu.Unlock()
			return
		}
	}
}

// Stop stops the background flusher and flushes remaining writes
func (bw *BatchWriter) Stop() error {
	if bw == nil {
		return nil
	}

	close(bw.stopChan)
	bw.wg.Wait()

	return nil
}

// GetStats returns batch writer statistics
func (bw *BatchWriter) GetStats() map[string]interface{} {
	if bw == nil {
		return nil
	}

	bw.mu.Lock()
	defer bw.mu.Unlock()

	avgBatchSize := 0.0
	if bw.totalFlushes > 0 {
		avgBatchSize = float64(bw.totalWrites-bw.writesSkipped) / float64(bw.totalFlushes)
	}

	efficiencyGain := 0.0
	if bw.totalWrites > 0 {
		efficiencyGain = float64(bw.writesSkipped) / float64(bw.totalWrites) * 100.0
	}

	return map[string]interface{}{
		"pending_writes":  len(bw.pendingWrites),
		"total_writes":    bw.totalWrites,
		"total_flushes":   bw.totalFlushes,
		"writes_skipped":  bw.writesSkipped,
		"avg_batch_size":  fmt.Sprintf("%.1f", avgBatchSize),
		"efficiency_gain": fmt.Sprintf("%.1f%%", efficiencyGain),
		"flush_interval":  bw.flushInterval.String(),
		"max_batch_size":  bw.maxBatchSize,
	}
}

// GetPendingCount returns the number of pending writes
func (bw *BatchWriter) GetPendingCount() int {
	if bw == nil {
		return 0
	}

	bw.mu.Lock()
	defer bw.mu.Unlock()

	return len(bw.pendingWrites)
}

// writeToFile is a helper that writes data to a file
func writeToFile(filePath string, data interface{}) error {
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// immediateWrite is a fallback that writes immediately (used when batch writer not initialized)
func immediateWrite(filePath string, data interface{}) {
	if err := writeToFile(filePath, data); err != nil {
		fmt.Printf("⚠️ Immediate write failed for %s: %v\n", filePath, err)
	}
}

// BatchWriteJSON is a convenience function for queuing JSON writes
func BatchWriteJSON(filePath string, data interface{}) {
	bw := GetBatchWriter()
	if bw != nil {
		bw.QueueWrite(filePath, data)
	} else {
		// Fallback to immediate write
		immediateWrite(filePath, data)
	}
}

// FlushAll is a convenience function for flushing all pending writes
func FlushAllBatchWrites() error {
	bw := GetBatchWriter()
	if bw != nil {
		return bw.Flush()
	}
	return nil
}
