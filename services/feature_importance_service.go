package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// FeatureImportance represents importance scores for a single feature
type FeatureImportance struct {
	Rank         int                `json:"rank"`
	FeatureIndex int                `json:"featureIndex"`
	FeatureName  string             `json:"featureName"`
	Category     string             `json:"category"`
	Importance   float64            `json:"importance"`
	ModelScores  map[string]float64 `json:"modelScores"`
}

// CategoryImportance represents aggregated importance by category
type CategoryImportance struct {
	Category   string  `json:"category"`
	Importance float64 `json:"importance"`
	Count      int     `json:"count"`
}

// FeatureImportanceReport is the complete analysis report
type FeatureImportanceReport struct {
	TopFeatures           []FeatureImportance   `json:"topFeatures"`
	ByCategory            []CategoryImportance  `json:"byCategory"`
	TotalFeatures         int                   `json:"totalFeatures"`
	FeaturesAnalyzed      int                   `json:"featuresAnalyzed"`
	ModelsIncluded        []string              `json:"modelsIncluded"`
	PruningRecommendation PruningRecommendation `json:"pruningRecommendation"`
}

// PruningRecommendation suggests which features to keep/remove
type PruningRecommendation struct {
	KeepCount   int `json:"keepCount"`   // >0.1% importance
	ReviewCount int `json:"reviewCount"` // 0.01-0.1% importance
	PruneCount  int `json:"pruneCount"`  // <0.01% importance
}

// FeatureImportanceService analyzes feature importance across models
type FeatureImportanceService struct {
	mutex sync.RWMutex
}

var (
	featureImportanceService     *FeatureImportanceService
	featureImportanceServiceOnce sync.Once
)

// InitFeatureImportanceService initializes the singleton
func InitFeatureImportanceService() {
	featureImportanceServiceOnce.Do(func() {
		featureImportanceService = &FeatureImportanceService{}
	})
}

// GetFeatureImportanceService returns the singleton instance
func GetFeatureImportanceService() *FeatureImportanceService {
	return featureImportanceService
}

// GetFeatureImportanceReport generates a comprehensive feature importance analysis
func (fis *FeatureImportanceService) GetFeatureImportanceReport() (*FeatureImportanceReport, error) {
	fis.mutex.RLock()
	defer fis.mutex.RUnlock()

	// Get importance from Gradient Boosting
	gbModel := GetGradientBoostingModel()
	gbImportance := make(map[int]float64)
	if gbModel != nil && gbModel.trained {
		for featureName, importance := range gbModel.featureImportance {
			var idx int
			fmt.Sscanf(featureName, "feature_%d", &idx)
			gbImportance[idx] = importance
		}
	}

	// Get importance from Random Forest
	rfModel := GetRandomForestModel()
	rfImportance := make(map[int]float64)
	if rfModel != nil && rfModel.trained {
		for featureName, importance := range rfModel.featureImportance {
			var idx int
			fmt.Sscanf(featureName, "feature_%d", &idx)
			rfImportance[idx] = importance
		}
	}

	// Combine importance scores
	combinedImportance := make(map[int]float64)
	modelsIncluded := []string{}

	if len(gbImportance) > 0 {
		modelsIncluded = append(modelsIncluded, "Gradient Boosting")
		for idx, importance := range gbImportance {
			combinedImportance[idx] += importance
		}
	}

	if len(rfImportance) > 0 {
		modelsIncluded = append(modelsIncluded, "Random Forest")
		for idx, importance := range rfImportance {
			combinedImportance[idx] += importance
		}
	}

	// Average if multiple models
	if len(modelsIncluded) > 1 {
		for idx := range combinedImportance {
			combinedImportance[idx] /= float64(len(modelsIncluded))
		}
	}

	// Create feature importance list
	features := []FeatureImportance{}
	for idx := 0; idx < 156; idx++ {
		importance := combinedImportance[idx]

		modelScores := make(map[string]float64)
		if gbScore, ok := gbImportance[idx]; ok {
			modelScores["Gradient Boosting"] = gbScore
		}
		if rfScore, ok := rfImportance[idx]; ok {
			modelScores["Random Forest"] = rfScore
		}

		features = append(features, FeatureImportance{
			FeatureIndex: idx,
			FeatureName:  getFeatureName(idx),
			Category:     getFeatureCategory(idx),
			Importance:   importance,
			ModelScores:  modelScores,
		})
	}

	// Sort by importance
	sort.Slice(features, func(i, j int) bool {
		return features[i].Importance > features[j].Importance
	})

	// Assign ranks
	for i := range features {
		features[i].Rank = i + 1
	}

	// Get top 20
	topCount := 20
	if len(features) < topCount {
		topCount = len(features)
	}
	topFeatures := features[:topCount]

	// Aggregate by category
	categoryMap := make(map[string]*CategoryImportance)
	for _, feature := range features {
		if _, exists := categoryMap[feature.Category]; !exists {
			categoryMap[feature.Category] = &CategoryImportance{
				Category: feature.Category,
			}
		}
		categoryMap[feature.Category].Importance += feature.Importance
		categoryMap[feature.Category].Count++
	}

	// Convert to slice and sort
	byCategory := []CategoryImportance{}
	for _, cat := range categoryMap {
		byCategory = append(byCategory, *cat)
	}
	sort.Slice(byCategory, func(i, j int) bool {
		return byCategory[i].Importance > byCategory[j].Importance
	})

	// Pruning recommendation
	pruning := PruningRecommendation{}
	for _, feature := range features {
		if feature.Importance > 0.001 {
			pruning.KeepCount++
		} else if feature.Importance > 0.0001 {
			pruning.ReviewCount++
		} else {
			pruning.PruneCount++
		}
	}

	return &FeatureImportanceReport{
		TopFeatures:           topFeatures,
		ByCategory:            byCategory,
		TotalFeatures:         156,
		FeaturesAnalyzed:      len(features),
		ModelsIncluded:        modelsIncluded,
		PruningRecommendation: pruning,
	}, nil
}

// getFeatureName returns human-readable name for feature index
func getFeatureName(idx int) string {
	names := map[int]string{
		0: "Home Win %", 1: "Away Win %",
		2: "Home Goals For", 3: "Home Goals Against",
		4: "Away Goals For", 5: "Away Goals Against",
		6: "Home xG Differential", 7: "Away xG Differential",
		8: "Home Corsi For %", 9: "Away Corsi For %",
		10: "Home Travel Fatigue", 11: "Away Travel Fatigue",
		12: "Home Injury Impact", 13: "Away Injury Impact",
		14: "Home Weather Impact", 15: "Away Weather Impact",
		20: "Home Momentum", 21: "Away Momentum",
		22: "Home Recent Form", 23: "Away Recent Form",
		24: "Home PP%", 25: "Away PP%",
		26: "Home PK%", 27: "Away PK%",
		28: "Home Rest Days", 29: "Away Rest Days",
		30: "Home B2B Penalty", 31: "Away B2B Penalty",
		32: "Home H2H Record", 33: "Away H2H Record",
		50: "Home Goalie Advantage", 51: "Away Goalie Advantage",
		52: "Goalie Save % Diff", 53: "Goalie Recent Form Diff",
		54: "Home Market Consensus", 55: "Away Market Consensus",
		56: "Sharp Money Indicator", 57: "Market Line Movement",
		65: "Home Star Power", 66: "Away Star Power",
		67: "Home Top Scorer PPG", 68: "Away Top Scorer PPG",
		69: "Home Top 3 Combined PPG", 70: "Away Top 3 Combined PPG",
		111: "Home IsHot", 112: "Away IsHot",
		113: "Home IsCold", 114: "Away IsCold",
		115: "Home IsStreaking", 116: "Away IsStreaking",
		117: "Home Weighted Win %", 118: "Away Weighted Win %",
		119: "Home Weighted GF", 120: "Away Weighted GF",
		121: "Home Weighted GA", 122: "Away Weighted GA",
		123: "Home vs Playoff Teams %", 124: "Away vs Playoff Teams %",
		125: "Home Clutch Performance", 126: "Away Clutch Performance",
		127: "Home Last 5 Points", 128: "Away Last 5 Points",
		129: "Home Goal Diff 5", 130: "Away Goal Diff 5",
		131: "Is Rivalry Game", 132: "Rivalry Intensity",
		133: "Is Division Game", 134: "Is Playoff Rematch",
		135: "H2H Advantage", 136: "Recent Matchup Trend",
		137: "Venue Specific Record", 138: "Days Since Last Meeting",
		139: "Average Goal Diff", 140: "Star Power Edge",
		141: "Depth Edge", 142: "Goalie Fatigue Diff",
		143: "Trap Game Factor", 144: "Playoff Importance",
		145: "Transition Efficiency", 146: "Special Teams Index",
		147: "Discipline Index",
		148: "Home Weather Travel Impact", 149: "Away Weather Travel Impact",
		150: "Home Weather Overall Impact", 151: "Away Weather Overall Impact",
		152: "Home Is Outdoor Game", 153: "Away Is Outdoor Game",
		154: "Market Confidence", 155: "Defensive Zone Time",
	}

	if name, ok := names[idx]; ok {
		return name
	}
	return fmt.Sprintf("Feature %d", idx)
}

// getFeatureCategory returns category for feature index
func getFeatureCategory(idx int) string {
	switch {
	case idx >= 0 && idx <= 5:
		return "Basic Stats"
	case idx >= 6 && idx <= 9:
		return "Advanced Analytics"
	case idx >= 10 && idx <= 31:
		return "Situational Factors"
	case idx >= 32 && idx <= 49:
		return "Historical"
	case idx >= 50 && idx <= 64:
		return "Phase 4 Features"
	case idx >= 65 && idx <= 74:
		return "Player Impact"
	case idx >= 75 && idx <= 92:
		return "xG & Shot Quality"
	case idx >= 93 && idx <= 100:
		return "Shift Analysis"
	case idx >= 101 && idx <= 110:
		return "Zone Control & Context"
	case idx >= 111 && idx <= 130:
		return "Rolling Statistics"
	case idx >= 131 && idx <= 139:
		return "Matchup Context"
	case idx >= 140 && idx <= 142:
		return "Player/Goalie Edges"
	case idx >= 143 && idx <= 147:
		return "Situational Context"
	case idx >= 148 && idx <= 155:
		return "Weather & Advanced"
	default:
		return "Unknown"
	}
}

// GenerateMarkdownReport creates a markdown report of feature importance
func (fis *FeatureImportanceService) GenerateMarkdownReport() (string, error) {
	report, err := fis.GetFeatureImportanceReport()
	if err != nil {
		return "", err
	}

	md := "# Feature Importance Analysis Report\n\n"
	md += fmt.Sprintf("**Total Features**: %d\n", report.TotalFeatures)
	md += fmt.Sprintf("**Features Analyzed**: %d\n", report.FeaturesAnalyzed)
	md += fmt.Sprintf("**Models Included**: %v\n\n", report.ModelsIncluded)

	md += "## Top 20 Features\n\n"
	md += "| Rank | Feature | Category | Importance |\n"
	md += "|------|---------|----------|------------|\n"
	for _, feature := range report.TopFeatures {
		md += fmt.Sprintf("| %d | %s | %s | %.4f |\n",
			feature.Rank, feature.FeatureName, feature.Category, feature.Importance)
	}

	md += "\n## Importance by Category\n\n"
	md += "| Category | Total Importance | Feature Count | Avg Importance |\n"
	md += "|----------|-----------------|---------------|----------------|\n"
	for _, cat := range report.ByCategory {
		avgImportance := cat.Importance / float64(cat.Count)
		md += fmt.Sprintf("| %s | %.4f | %d | %.4f |\n",
			cat.Category, cat.Importance, cat.Count, avgImportance)
	}

	md += "\n## Pruning Recommendation\n\n"
	md += fmt.Sprintf("- **Keep**: %d features (>0.1%% importance)\n", report.PruningRecommendation.KeepCount)
	md += fmt.Sprintf("- **Review**: %d features (0.01-0.1%% importance)\n", report.PruningRecommendation.ReviewCount)
	md += fmt.Sprintf("- **Prune**: %d features (<0.01%% importance)\n", report.PruningRecommendation.PruneCount)

	return md, nil
}

// GetFeatureImportanceJSON returns JSON representation
func (fis *FeatureImportanceService) GetFeatureImportanceJSON() ([]byte, error) {
	report, err := fis.GetFeatureImportanceReport()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(report, "", "  ")
}
