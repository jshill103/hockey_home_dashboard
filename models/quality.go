package models

import "time"

// PredictionQuality assesses the quality and reliability of a prediction
type PredictionQuality struct {
	Score       float64            `json:"score"`       // Overall quality score (0-100)
	Tier        string             `json:"tier"`        // "Excellent", "Good", "Fair", "Poor"
	Confidence  float64            `json:"confidence"`  // Adjusted/calibrated confidence
	Reliability string             `json:"reliability"` // "High", "Medium", "Low"
	Warnings    []string           `json:"warnings"`    // Any quality concerns
	Factors     map[string]float64 `json:"factors"`     // Individual quality factor scores
	
	// Component Scores
	ModelAgreement      float64 `json:"modelAgreement"`      // How much models agree (0-1)
	DataQuality         float64 `json:"dataQuality"`         // Completeness of data (0-1)
	HistoricalAccuracy  float64 `json:"historicalAccuracy"`  // Past accuracy in similar contexts (0-1)
	SampleSize          float64 `json:"sampleSize"`          // Adequate data for H2H/rest (0-1)
	DataRecency         float64 `json:"dataRecency"`         // How recent is the data (0-1)
	
	// Detailed Assessment
	MissingFeatures     []string `json:"missingFeatures"`     // Features with no data
	UncertainFactors    []string `json:"uncertainFactors"`    // Factors with uncertainty
	StrengthFactors     []string `json:"strengthFactors"`     // What makes this prediction strong
	
	// Recommendation
	UseWithCaution      bool   `json:"useWithCaution"`      // Flag if quality is low
	RecommendedAction   string `json:"recommendedAction"`   // What to do with this prediction
	QualityExplanation  string `json:"qualityExplanation"`  // Human-readable quality summary
	
	Timestamp time.Time `json:"timestamp"`
}

// QualityFactor represents an individual aspect of prediction quality
type QualityFactor struct {
	Name        string  `json:"name"`        // e.g., "Data Completeness", "Model Agreement"
	Score       float64 `json:"score"`       // 0-100
	Weight      float64 `json:"weight"`      // Importance weight (0-1)
	Status      string  `json:"status"`      // "Excellent", "Good", "Fair", "Poor"
	Impact      string  `json:"impact"`      // How this affects overall quality
	Description string  `json:"description"` // What this factor measures
}

// DataCompleteness tracks which features have data and which don't
type DataCompleteness struct {
	TotalFeatures      int      `json:"totalFeatures"`
	PopulatedFeatures  int      `json:"populatedFeatures"`
	CompletenessScore  float64  `json:"completenessScore"` // 0-1
	MissingFeatures    []string `json:"missingFeatures"`
	CriticalMissing    []string `json:"criticalMissing"`    // High-importance missing features
	Tier               string   `json:"tier"`               // "Complete", "Good", "Partial", "Poor"
}

// ModelAgreementAnalysis measures how much models agree on prediction
type ModelAgreementAnalysis struct {
	AgreementScore     float64            `json:"agreementScore"`     // 0-1 (1 = perfect agreement)
	VarianceScore      float64            `json:"varianceScore"`      // Variance in predictions
	ConsensusWinner    string             `json:"consensusWinner"`    // Team most models predict
	ConsensusStrength  float64            `json:"consensusStrength"`  // % of models agreeing
	Outliers           []string           `json:"outliers"`           // Models with very different predictions
	ModelPredictions   map[string]string  `json:"modelPredictions"`   // Each model's prediction
	AgreementLevel     string             `json:"agreementLevel"`     // "Strong", "Moderate", "Weak", "Conflicting"
}

// HistoricalAccuracyContext provides accuracy in similar past situations
type HistoricalAccuracyContext struct {
	ContextType        string  `json:"contextType"`        // Type of similar situation
	SimilarGames       int     `json:"similarGames"`       // Number of similar past games
	AccuracyInContext  float64 `json:"accuracyInContext"`  // Historical accuracy
	ConfidenceLevel    float64 `json:"confidenceLevel"`    // Confidence in this assessment
	SampleAdequacy     string  `json:"sampleAdequacy"`     // "Adequate", "Limited", "Insufficient"
}

// PredictionRiskAssessment identifies risks with a prediction
type PredictionRiskAssessment struct {
	RiskLevel          string   `json:"riskLevel"`          // "Low", "Medium", "High", "Very High"
	RiskScore          float64  `json:"riskScore"`          // 0-1 (higher = riskier)
	RiskFactors        []string `json:"riskFactors"`        // Identified risks
	MitigatingFactors  []string `json:"mitigatingFactors"`  // What reduces risk
	RecommendedConfidence float64 `json:"recommendedConfidence"` // Adjusted confidence
	UseInBetting       bool     `json:"useInBetting"`       // Safe for betting decisions
}

// QualityThresholds defines standards for quality tiers
type QualityThresholds struct {
	ExcellentMin float64 `json:"excellentMin"` // 90+
	GoodMin      float64 `json:"goodMin"`      // 75+
	FairMin      float64 `json:"fairMin"`      // 60+
	// Below FairMin = Poor
	
	HighReliabilityMin float64 `json:"highReliabilityMin"` // 85+
	MediumReliabilityMin float64 `json:"mediumReliabilityMin"` // 70+
	// Below MediumReliabilityMin = Low
}

// QualityReport provides a comprehensive quality assessment
type QualityReport struct {
	GameID             int                       `json:"gameID"`
	HomeTeam           string                    `json:"homeTeam"`
	AwayTeam           string                    `json:"awayTeam"`
	PredictionQuality  PredictionQuality         `json:"predictionQuality"`
	DataCompleteness   DataCompleteness          `json:"dataCompleteness"`
	ModelAgreement     ModelAgreementAnalysis    `json:"modelAgreement"`
	HistoricalContext  HistoricalAccuracyContext `json:"historicalContext"`
	RiskAssessment     PredictionRiskAssessment  `json:"riskAssessment"`
	OverallAssessment  string                    `json:"overallAssessment"`  // Summary
	Recommendations    []string                  `json:"recommendations"`    // Action items
	Timestamp          time.Time                 `json:"timestamp"`
}

// QualityMetrics tracks quality over time
type QualityMetrics struct {
	TotalPredictions     int     `json:"totalPredictions"`
	ExcellentCount       int     `json:"excellentCount"`
	GoodCount            int     `json:"goodCount"`
	FairCount            int     `json:"fairCount"`
	PoorCount            int     `json:"poorCount"`
	AvgQualityScore      float64 `json:"avgQualityScore"`
	AvgModelAgreement    float64 `json:"avgModelAgreement"`
	AvgDataCompleteness  float64 `json:"avgDataCompleteness"`
	QualityTrend         float64 `json:"qualityTrend"` // Positive = improving
	LastUpdated          time.Time `json:"lastUpdated"`
}

