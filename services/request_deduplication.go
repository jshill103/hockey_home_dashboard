package services

import (
	"fmt"
	"sync"
)

// RequestDeduplicator prevents concurrent duplicate API requests
// If a request is already in flight, subsequent requests will wait for the result
type RequestDeduplicator struct {
	inflight map[string]*inflightRequest
	mu       sync.Mutex
}

type inflightRequest struct {
	wg     sync.WaitGroup
	result []byte
	err    error
}

var (
	requestDeduplicator     *RequestDeduplicator
	requestDeduplicatorOnce sync.Once
)

// InitRequestDeduplicator initializes the global request deduplicator
func InitRequestDeduplicator() {
	requestDeduplicatorOnce.Do(func() {
		requestDeduplicator = &RequestDeduplicator{
			inflight: make(map[string]*inflightRequest),
		}
	})
}

// GetRequestDeduplicator returns the singleton instance
func GetRequestDeduplicator() *RequestDeduplicator {
	return requestDeduplicator
}

// Do executes fn only once for the given key, even if called concurrently
// If a request is already in flight for this key, it waits for that result
func (rd *RequestDeduplicator) Do(key string, fn func() ([]byte, error)) ([]byte, error) {
	rd.mu.Lock()

	// Check if request already in flight
	if req, ok := rd.inflight[key]; ok {
		// Request in flight - wait for it
		rd.mu.Unlock()
		fmt.Printf("⏳ Waiting for in-flight request: %s\n", key)
		req.wg.Wait()
		return req.result, req.err
	}

	// Start new request
	req := &inflightRequest{}
	req.wg.Add(1)
	rd.inflight[key] = req
	rd.mu.Unlock()

	// Execute the function
	result, err := fn()

	// Store result and notify waiters
	req.result = result
	req.err = err
	req.wg.Done()

	// Remove from inflight map
	rd.mu.Lock()
	delete(rd.inflight, key)
	rd.mu.Unlock()

	return result, err
}
