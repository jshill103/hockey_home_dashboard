package models

import "time"

// HealthStatus represents the overall health of the application
type HealthStatus struct {
	Status    string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]HealthCheck `json:"checks"`
	Uptime    string                 `json:"uptime"`
	Version   string                 `json:"version,omitempty"`
}

// HealthCheck represents a single health check result
type HealthCheck struct {
	Name         string        `json:"name"`
	Status       string        `json:"status"` // "healthy", "degraded", "unhealthy"
	Message      string        `json:"message,omitempty"`
	ResponseTime time.Duration `json:"responseTime,omitempty"`
	LastChecked  time.Time     `json:"lastChecked"`
	Details      interface{}   `json:"details,omitempty"`
}
