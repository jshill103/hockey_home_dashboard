package services

import "math"

// safeDiv safely divides two numbers, returning a default value if division is invalid
// This prevents NaN and Inf values that can cascade through predictions
func safeDiv(numerator, denominator, defaultValue float64) float64 {
	if denominator == 0 || math.IsNaN(denominator) || math.IsInf(denominator, 0) {
		return defaultValue
	}
	result := numerator / denominator
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return defaultValue
	}
	return result
}

// clampValue clamps a value between min and max
func clampValue(value, min, max float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return (min + max) / 2.0 // Return midpoint if invalid
	}
	return math.Max(min, math.Min(max, value))
}

// isValidNumber checks if a number is valid (not NaN or Inf)
func isValidNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
