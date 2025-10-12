package services

import (
	"net/http"
	"time"
)

// SharedHTTPClient is a shared HTTP client with connection pooling
// This reduces the overhead of creating new connections for each request
var SharedHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,              // Total idle connections across all hosts
		MaxIdleConnsPerHost: 10,               // Idle connections per host
		IdleConnTimeout:     90 * time.Second, // How long idle connections stay alive
		DisableCompression:  false,            // Enable compression
		ForceAttemptHTTP2:   true,             // Try HTTP/2
	},
}
