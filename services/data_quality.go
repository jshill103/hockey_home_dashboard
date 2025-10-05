package services

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// DataQualityService evaluates the quality of prediction input data
type DataQualityService struct {
	teamCode     string
	lastUpdate   time.Time
	qualityCache map[string]*models.DataQualityScore
}

// DataQualityScore represents the quality assessment of prediction data
type DataQualityScore struct {
	OverallScore      float64                  `json:"overallScore"`      // 0-100 overall data quality
	FreshnessScore    float64                  `json:"freshnessScore"`    // How recent the data is
	CompletenessScore float64                  `json:"completenessScore"` // How complete the data is
	AccuracyScore     float64                  `json:"accuracyScore"`     // How accurate the data appears
	ConsistencyScore  float64                  `json:"consistencyScore"`  // How consistent across sources
	ReliabilityScore  float64                  `json:"reliabilityScore"`  // Source reliability
	CategoryScores    map[string]float64       `json:"categoryScores"`    // Scores by data category
	QualityFactors    []DataQualityFactor      `json:"qualityFactors"`    // Detailed quality factors
	ConfidenceImpact  float64                  `json:"confidenceImpact"`  // Impact on prediction confidence
	LastAssessment    time.Time                `json:"lastAssessment"`    // When assessment was made
	DataSources       map[string]SourceQuality `json:"dataSources"`       // Quality by data source
	IssuesDetected    []DataQualityIssue       `json:"issuesDetected"`    // Any data quality issues found
	Recommendations   []string                 `json:"recommendations"`   // Suggestions for improvement
}

// DataQualityFactor represents a specific factor affecting data quality
type DataQualityFactor struct {
	FactorName  string    `json:"factorName"`  // e.g., "API Response Time"
	Category    string    `json:"category"`    // e.g., "Freshness", "Completeness"
	Score       float64   `json:"score"`       // 0-100 score for this factor
	Weight      float64   `json:"weight"`      // Importance weight
	Description string    `json:"description"` // Human-readable description
	IsCritical  bool      `json:"isCritical"`  // Whether this is a critical factor
	Trend       string    `json:"trend"`       // "improving", "stable", "declining"
	LastUpdated time.Time `json:"lastUpdated"` // When this factor was last updated
}

// SourceQuality represents quality metrics for a specific data source
type SourceQuality struct {
	SourceName       string    `json:"sourceName"`       // e.g., "NHL API", "ESPN API"
	ReliabilityScore float64   `json:"reliabilityScore"` // Historical reliability 0-100
	ResponseTime     float64   `json:"responseTime"`     // Average response time (ms)
	UptimeScore      float64   `json:"uptimeScore"`      // Uptime percentage
	DataFreshness    float64   `json:"dataFreshness"`    // How fresh the data is (hours)
	ErrorRate        float64   `json:"errorRate"`        // Percentage of failed requests
	LastSuccess      time.Time `json:"lastSuccess"`      // Last successful data fetch
	LastError        time.Time `json:"lastError"`        // Last error encountered
	IsAvailable      bool      `json:"isAvailable"`      // Currently available
	QualityTrend     string    `json:"qualityTrend"`     // Recent quality trend
}

// DataQualityIssue represents a detected data quality problem
type DataQualityIssue struct {
	IssueType    string    `json:"issueType"`    // e.g., "stale_data", "missing_field"
	Severity     string    `json:"severity"`     // "low", "medium", "high", "critical"
	Description  string    `json:"description"`  // Human-readable description
	AffectedData []string  `json:"affectedData"` // Which data fields are affected
	DetectedAt   time.Time `json:"detectedAt"`   // When issue was detected
	Impact       float64   `json:"impact"`       // Impact on prediction quality (0-1)
	Suggestion   string    `json:"suggestion"`   // How to resolve the issue
	IsResolved   bool      `json:"isResolved"`   // Whether issue has been resolved
}

// NewDataQualityService creates a new data quality assessment service
func NewDataQualityService(teamCode string) *DataQualityService {
	return &DataQualityService{
		teamCode:     teamCode,
		qualityCache: make(map[string]*models.DataQualityScore),
	}
}

// AssessDataQuality performs comprehensive data quality assessment
func (dqs *DataQualityService) AssessDataQuality(factors *models.PredictionFactors) *models.DataQualityScore {
	log.Printf("📊 Assessing data quality for %s...", factors.TeamCode)

	cacheKey := fmt.Sprintf("%s_%s", factors.TeamCode, time.Now().Format("2006-01-02-15"))

	// Check cache first (refresh hourly)
	if cached, exists := dqs.qualityCache[cacheKey]; exists {
		return cached
	}

	result := &models.DataQualityScore{
		CategoryScores: make(map[string]float64),
		DataSources:    make(map[string]models.SourceQuality),
		LastAssessment: time.Now(),
	}

	// Assess different data quality dimensions
	result.FreshnessScore = dqs.assessDataFreshness(factors)
	result.CompletenessScore = dqs.assessDataCompleteness(factors)
	result.AccuracyScore = dqs.assessDataAccuracy(factors)
	result.ConsistencyScore = dqs.assessDataConsistency(factors)
	result.ReliabilityScore = dqs.assessSourceReliability(factors)

	// Calculate category scores
	result.CategoryScores["team_stats"] = dqs.assessTeamStatsQuality(factors)
	result.CategoryScores["recent_performance"] = dqs.assessRecentPerformanceQuality(factors)
	result.CategoryScores["situational_factors"] = dqs.assessSituationalFactorsQuality(factors)
	result.CategoryScores["advanced_analytics"] = dqs.assessAdvancedAnalyticsQuality(factors)
	// Player analysis removed - no real data available

	// Calculate overall score (weighted average)
	result.OverallScore = dqs.calculateOverallScore(result)

	// Assess quality factors
	result.QualityFactors = dqs.identifyQualityFactors(factors, result)

	// Detect issues
	result.IssuesDetected = dqs.detectQualityIssues(factors, result)

	// Generate recommendations
	result.Recommendations = dqs.generateRecommendations(result)

	// Calculate confidence impact
	result.ConfidenceImpact = dqs.calculateConfidenceImpact(result)

	// Cache the result
	dqs.qualityCache[cacheKey] = result

	log.Printf("✅ Data quality assessment complete: %.1f/100 (Confidence Impact: %.2fx)",
		result.OverallScore, result.ConfidenceImpact)

	return result
}

// assessDataFreshness evaluates how recent the data is
func (dqs *DataQualityService) assessDataFreshness(factors *models.PredictionFactors) float64 {
	score := 100.0

	// Penalize for stale data (mock implementation - would check actual timestamps)
	if factors.RestDays > 3 {
		score -= 10.0 // Old rest data
	}

	// Recent form should be very recent
	if factors.RecentForm < 70 {
		score -= 5.0 // Potentially stale recent form
	}

	// Advanced analytics should be current season
	if factors.AdvancedStats.OverallRating == 0 {
		score -= 15.0 // Missing current advanced stats
	}

	return math.Max(0, math.Min(100, score))
}

// assessDataCompleteness evaluates how complete the data is
func (dqs *DataQualityService) assessDataCompleteness(factors *models.PredictionFactors) float64 {
	requiredFields := 0
	presentFields := 0

	// Check core team stats
	requiredFields += 6
	if factors.WinPercentage > 0 {
		presentFields++
	}
	if factors.GoalsFor > 0 {
		presentFields++
	}
	if factors.GoalsAgainst > 0 {
		presentFields++
	}
	if factors.PowerPlayPct > 0 {
		presentFields++
	}
	if factors.PenaltyKillPct > 0 {
		presentFields++
	}
	if factors.HomeAdvantage > 0 {
		presentFields++
	}

	// Check completeness of situational factors
	requiredFields += 5
	if factors.TravelFatigue.MilesTraveled >= 0 {
		presentFields++
	}
	if factors.AltitudeAdjust.AltitudeDiff != 0 {
		presentFields++
	}
	if factors.ScheduleStrength.OpponentStrength > 0 {
		presentFields++
	}
	if factors.InjuryImpact.InjuryScore >= 0 {
		presentFields++
	}
	if factors.MomentumFactors.WinStreak >= 0 {
		presentFields++
	}

	// Check advanced analytics
	requiredFields += 3
	if factors.AdvancedStats.XGForPerGame > 0 {
		presentFields++
	}
	if factors.AdvancedStats.CorsiForPct > 0 {
		presentFields++
	}
	if factors.AdvancedStats.OverallRating > 0 {
		presentFields++
	}

	// Player analysis removed - no real data available

	completenessRatio := float64(presentFields) / float64(requiredFields)
	return completenessRatio * 100.0
}

// assessDataAccuracy evaluates apparent accuracy of the data
func (dqs *DataQualityService) assessDataAccuracy(factors *models.PredictionFactors) float64 {
	score := 100.0

	// Check for unrealistic values
	if factors.WinPercentage < 0 || factors.WinPercentage > 1 {
		score -= 20.0
	}

	if factors.GoalsFor < 0 || factors.GoalsFor > 8 {
		score -= 10.0 // Unrealistic goals per game
	}

	if factors.PowerPlayPct < 0 || factors.PowerPlayPct > 1 {
		score -= 15.0
	}

	if factors.PenaltyKillPct < 0 || factors.PenaltyKillPct > 1 {
		score -= 15.0
	}

	// Check for logical consistency
	if factors.AdvancedStats.OverallRating > 100 {
		score -= 10.0
	}

	return math.Max(0, math.Min(100, score))
}

// assessDataConsistency evaluates consistency across different data sources
func (dqs *DataQualityService) assessDataConsistency(factors *models.PredictionFactors) float64 {
	score := 90.0 // Start high, deduct for inconsistencies

	// Check if advanced stats align with basic stats
	expectedRating := (factors.WinPercentage * 100)
	actualRating := factors.AdvancedStats.OverallRating

	if math.Abs(expectedRating-actualRating) > 20 {
		score -= 15.0 // Significant inconsistency
	} else if math.Abs(expectedRating-actualRating) > 10 {
		score -= 5.0 // Minor inconsistency
	}

	// Check if momentum aligns with recent form
	if factors.MomentumFactors.WinStreak > 3 && factors.RecentForm < 60 {
		score -= 8.0 // Inconsistent: winning streak but poor recent form
	}

	return math.Max(0, math.Min(100, score))
}

// assessSourceReliability evaluates the reliability of data sources
func (dqs *DataQualityService) assessSourceReliability(factors *models.PredictionFactors) float64 {
	// Mock implementation - would track actual source reliability
	sources := map[string]float64{
		"NHL_API":        95.0,
		"ESPN_API":       85.0,
		"ADVANCED_STATS": 90.0,
		"PLAYER_DATA":    80.0,
		"INJURY_REPORTS": 75.0,
	}

	totalReliability := 0.0
	sourceCount := 0

	for _, reliability := range sources {
		totalReliability += reliability
		sourceCount++
	}

	if sourceCount == 0 {
		return 50.0
	}

	return totalReliability / float64(sourceCount)
}

// Category-specific quality assessments
func (dqs *DataQualityService) assessTeamStatsQuality(factors *models.PredictionFactors) float64 {
	score := 100.0

	// Check completeness of team stats
	if factors.WinPercentage == 0 {
		score -= 25.0
	}
	if factors.GoalsFor == 0 {
		score -= 20.0
	}
	if factors.GoalsAgainst == 0 {
		score -= 20.0
	}
	if factors.PowerPlayPct == 0 {
		score -= 15.0
	}
	if factors.PenaltyKillPct == 0 {
		score -= 15.0
	}

	// Check reasonableness
	if factors.WinPercentage > 0.8 || factors.WinPercentage < 0.2 {
		score -= 5.0 // Extreme win percentage (possible but unusual)
	}

	return math.Max(0, score)
}

func (dqs *DataQualityService) assessRecentPerformanceQuality(factors *models.PredictionFactors) float64 {
	score := 100.0

	if factors.RecentForm == 0 {
		score -= 30.0 // Missing recent form data
	}

	if factors.RestDays < 0 {
		score -= 20.0 // Invalid rest days
	}

	if factors.BackToBackPenalty < 0 || factors.BackToBackPenalty > 1 {
		score -= 15.0 // Invalid penalty value
	}

	return math.Max(0, score)
}

func (dqs *DataQualityService) assessSituationalFactorsQuality(factors *models.PredictionFactors) float64 {
	score := 100.0

	// Check travel fatigue data
	if factors.TravelFatigue.MilesTraveled < 0 {
		score -= 15.0
	}

	// Check altitude data
	if math.Abs(factors.AltitudeAdjust.AltitudeDiff) > 10000 {
		score -= 10.0 // Unrealistic altitude difference
	}

	// Check injury impact
	if factors.InjuryImpact.InjuryScore < 0 || factors.InjuryImpact.InjuryScore > 1 {
		score -= 20.0
	}

	return math.Max(0, score)
}

func (dqs *DataQualityService) assessAdvancedAnalyticsQuality(factors *models.PredictionFactors) float64 {
	score := 100.0

	if factors.AdvancedStats.OverallRating == 0 {
		score -= 40.0 // Missing advanced stats
	}

	if factors.AdvancedStats.XGForPerGame < 0 || factors.AdvancedStats.XGForPerGame > 6 {
		score -= 15.0 // Unrealistic expected goals
	}

	if factors.AdvancedStats.CorsiForPct < 0.3 || factors.AdvancedStats.CorsiForPct > 0.7 {
		score -= 10.0 // Extreme Corsi values
	}

	return math.Max(0, score)
}

// calculateOverallScore computes weighted overall data quality score
func (dqs *DataQualityService) calculateOverallScore(score *models.DataQualityScore) float64 {
	// Weighted combination of different quality dimensions
	weights := map[string]float64{
		"freshness":    0.20,
		"completeness": 0.25,
		"accuracy":     0.25,
		"consistency":  0.15,
		"reliability":  0.15,
	}

	overall := 0.0
	overall += score.FreshnessScore * weights["freshness"]
	overall += score.CompletenessScore * weights["completeness"]
	overall += score.AccuracyScore * weights["accuracy"]
	overall += score.ConsistencyScore * weights["consistency"]
	overall += score.ReliabilityScore * weights["reliability"]

	return math.Max(0, math.Min(100, overall))
}

// identifyQualityFactors identifies specific factors affecting data quality
func (dqs *DataQualityService) identifyQualityFactors(factors *models.PredictionFactors, score *models.DataQualityScore) []models.DataQualityFactor {
	var qualityFactors []models.DataQualityFactor

	// API Response Quality
	qualityFactors = append(qualityFactors, models.DataQualityFactor{
		FactorName:  "API Response Time",
		Category:    "Freshness",
		Score:       95.0, // Mock - would measure actual response times
		Weight:      0.15,
		Description: "NHL API response time and availability",
		IsCritical:  true,
		Trend:       "stable",
		LastUpdated: time.Now(),
	})

	// Data Completeness
	completenessScore := score.CompletenessScore
	qualityFactors = append(qualityFactors, models.DataQualityFactor{
		FactorName:  "Data Completeness",
		Category:    "Completeness",
		Score:       completenessScore,
		Weight:      0.25,
		Description: fmt.Sprintf("%.1f%% of required data fields present", completenessScore),
		IsCritical:  completenessScore < 80,
		Trend:       "stable",
		LastUpdated: time.Now(),
	})

	// Advanced Analytics Quality
	advancedScore := score.CategoryScores["advanced_analytics"]
	qualityFactors = append(qualityFactors, models.DataQualityFactor{
		FactorName:  "Advanced Analytics",
		Category:    "Accuracy",
		Score:       advancedScore,
		Weight:      0.20,
		Description: fmt.Sprintf("Quality of advanced hockey analytics (%.1f/100)", advancedScore),
		IsCritical:  advancedScore < 70,
		Trend:       "improving",
		LastUpdated: time.Now(),
	})

	return qualityFactors
}

// detectQualityIssues identifies specific data quality problems
func (dqs *DataQualityService) detectQualityIssues(factors *models.PredictionFactors, score *models.DataQualityScore) []models.DataQualityIssue {
	var issues []models.DataQualityIssue

	// Check for missing critical data
	if factors.WinPercentage == 0 {
		issues = append(issues, models.DataQualityIssue{
			IssueType:    "missing_critical_data",
			Severity:     "high",
			Description:  "Missing win percentage data",
			AffectedData: []string{"WinPercentage"},
			DetectedAt:   time.Now(),
			Impact:       0.3,
			Suggestion:   "Fetch current season standings data",
			IsResolved:   false,
		})
	}

	// Check for stale data
	if score.FreshnessScore < 70 {
		issues = append(issues, models.DataQualityIssue{
			IssueType:    "stale_data",
			Severity:     "medium",
			Description:  "Some data appears to be stale or outdated",
			AffectedData: []string{"RecentForm", "PlayerHealth"},
			DetectedAt:   time.Now(),
			Impact:       0.15,
			Suggestion:   "Refresh data from primary sources",
			IsResolved:   false,
		})
	}

	// Check for inconsistent data
	if score.ConsistencyScore < 80 {
		issues = append(issues, models.DataQualityIssue{
			IssueType:    "data_inconsistency",
			Severity:     "medium",
			Description:  "Inconsistency detected between different data sources",
			AffectedData: []string{"AdvancedStats", "BasicStats"},
			DetectedAt:   time.Now(),
			Impact:       0.2,
			Suggestion:   "Cross-validate data from multiple sources",
			IsResolved:   false,
		})
	}

	return issues
}

// generateRecommendations provides suggestions for improving data quality
func (dqs *DataQualityService) generateRecommendations(score *models.DataQualityScore) []string {
	var recommendations []string

	if score.FreshnessScore < 80 {
		recommendations = append(recommendations, "Increase data refresh frequency for real-time accuracy")
	}

	if score.CompletenessScore < 85 {
		recommendations = append(recommendations, "Add additional data sources to fill missing information")
	}

	if score.AccuracyScore < 90 {
		recommendations = append(recommendations, "Implement data validation rules to catch errors")
	}

	if score.ConsistencyScore < 85 {
		recommendations = append(recommendations, "Set up cross-source validation to detect inconsistencies")
	}

	if len(score.IssuesDetected) > 2 {
		recommendations = append(recommendations, "Address high-priority data quality issues first")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Data quality is excellent - maintain current practices")
	}

	return recommendations
}

// calculateConfidenceImpact determines how data quality affects prediction confidence
func (dqs *DataQualityService) calculateConfidenceImpact(score *models.DataQualityScore) float64 {
	// Convert quality score to confidence multiplier
	// 100 = 1.2x confidence, 50 = 1.0x confidence, 0 = 0.7x confidence

	baseMultiplier := 1.0
	qualityBonus := (score.OverallScore - 50.0) / 50.0 * 0.2 // -0.2 to +0.2

	// Apply penalty for critical issues
	criticalIssues := 0
	for _, issue := range score.IssuesDetected {
		if issue.Severity == "critical" || issue.Severity == "high" {
			criticalIssues++
		}
	}

	criticalPenalty := float64(criticalIssues) * 0.1 // -0.1 per critical issue

	finalMultiplier := baseMultiplier + qualityBonus - criticalPenalty

	return math.Max(0.7, math.Min(1.3, finalMultiplier))
}
