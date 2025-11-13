package models

import "time"

// Note: ConfidenceBin and CalibrationCurve are defined in accuracy_tracking.go

// ModelPerformanceMetrics tracks model performance over time (Phase 3)
type ModelPerformanceMetrics struct {
	ModelName      string    `json:"modelName"`
	LastUpdated    time.Time `json:"lastUpdated"`
	
	// Overall Performance
	TotalPredictions    int     `json:"totalPredictions"`
	CorrectPredictions  int     `json:"correctPredictions"`
	OverallAccuracy     float64 `json:"overallAccuracy"`
	
	// Recent Performance (last 20 predictions)
	RecentPredictions   int     `json:"recentPredictions"`
	RecentCorrect       int     `json:"recentCorrect"`
	RecentAccuracy      float64 `json:"recentAccuracy"`
	
	// Confidence Metrics
	AvgConfidence       float64 `json:"avgConfidence"`
	ConfidenceAccuracyGap float64 `json:"confidenceAccuracyGap"` // How overconfident
	
	// Trend Analysis
	ImprovingTrend      bool    `json:"improvingTrend"`
	AccuracyTrend       float64 `json:"accuracyTrend"` // Positive = improving
	
	// Weights
	CurrentWeight       float64 `json:"currentWeight"`
	BaseWeight          float64 `json:"baseWeight"`
	WeightAdjustment    float64 `json:"weightAdjustment"` // Current - Base
	
	// Performance by Context
	PerformanceByContext map[string]float64 `json:"performanceByContext"`
}

// RecalibrationEvent tracks when and why ensemble weights were adjusted
type RecalibrationEvent struct {
	Timestamp      time.Time          `json:"timestamp"`
	Trigger        string             `json:"trigger"`        // Why recalibration happened
	OldWeights     map[string]float64 `json:"oldWeights"`
	NewWeights     map[string]float64 `json:"newWeights"`
	WeightChanges  map[string]float64 `json:"weightChanges"`  // Delta for each model
	PerformanceSnapshot map[string]float64 `json:"performanceSnapshot"` // Accuracy before change
	Reasoning      string             `json:"reasoning"`      // Human-readable explanation
	PredictionCount int               `json:"predictionCount"` // Total predictions so far
}

// RecalibrationHistory stores the history of weight adjustments
type RecalibrationHistory struct {
	Events       []RecalibrationEvent `json:"events"`
	LastRecalibration time.Time        `json:"lastRecalibration"`
	TotalRecalibrations int            `json:"totalRecalibrations"`
	AvgTimeBetween float64            `json:"avgTimeBetween"` // Hours between recalibrations
}

// ModelPerformanceComparison compares models against each other
type ModelPerformanceComparison struct {
	Timestamp     time.Time                      `json:"timestamp"`
	Models        []string                       `json:"models"`
	Accuracies    map[string]float64             `json:"accuracies"`
	Rankings      map[string]int                 `json:"rankings"` // 1 = best
	BestModel     string                         `json:"bestModel"`
	WorstModel    string                         `json:"worstModel"`
	AccuracySpread float64                       `json:"accuracySpread"` // Max - Min
	Agreement     float64                        `json:"agreement"`      // How much models agree
}

// WeightConstraints defines limits on weight adjustments
type WeightConstraints struct {
	MinWeight         float64 `json:"minWeight"`         // Minimum weight any model can have
	MaxWeight         float64 `json:"maxWeight"`         // Maximum weight any model can have
	MaxShiftPerUpdate float64 `json:"maxShiftPerUpdate"` // Max change per recalibration
	MinSampleSize     int     `json:"minSampleSize"`     // Min predictions before adjusting
	SmoothingFactor   float64 `json:"smoothingFactor"`   // EMA smoothing (0-1)
}

// RecalibrationConfig stores configuration for recalibration
type RecalibrationConfig struct {
	Enabled             bool              `json:"enabled"`
	UpdateFrequency     int               `json:"updateFrequency"`     // Predictions between updates
	AccuracyThreshold   float64           `json:"accuracyThreshold"`   // Min accuracy drop to trigger
	LearningRate        float64           `json:"learningRate"`        // How aggressively to adjust
	Constraints         WeightConstraints `json:"constraints"`
	AutoRecalibrate     bool              `json:"autoRecalibrate"`     // Auto-trigger recalibration
	RequireManualApproval bool            `json:"requireManualApproval"` // Need approval for changes
}

