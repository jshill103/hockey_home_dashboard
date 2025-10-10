package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleHealthCheck returns comprehensive health status
func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	healthService := services.GetHealthCheckService()

	// Run all health checks
	status := healthService.RunHealthChecks()

	// Set HTTP status code based on overall health
	statusCode := http.StatusOK
	switch status.Status {
	case "degraded":
		statusCode = http.StatusOK // 200 but degraded
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable // 503
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(status)
}

// HandleHealthCheckSimple returns basic health status (for load balancers)
func HandleHealthCheckSimple(w http.ResponseWriter, r *http.Request) {
	healthService := services.GetHealthCheckService()
	status := healthService.RunHealthChecks()

	// Simple response for load balancers
	w.Header().Set("Content-Type", "text/plain")

	if status.Status == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("unhealthy"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// HandleHealth is an alias for HandleHealthCheck (for backwards compatibility)
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	HandleHealthCheck(w, r)
}
