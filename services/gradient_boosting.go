package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// GradientBoostingModel implements a gradient boosting classifier for NHL predictions
type GradientBoostingModel struct {
	trees             []*GBTree
	learningRate      float64
	numTrees          int
	maxDepth          int
	minSamplesLeaf    int
	weight            float64
	trained           bool
	dataDir           string
	mutex             sync.RWMutex
	featureNames      []string
	featureImportance map[string]float64
}

// GBTree represents a single decision tree in the gradient boosting ensemble
type GBTree struct {
	Root           *GBTreeNode
	MaxDepth       int
	MinSamplesLeaf int
}

// GBTreeNode represents a node in the gradient boosting tree
type GBTreeNode struct {
	IsLeaf       bool
	Prediction   float64 // For leaf nodes
	FeatureIndex int     // For internal nodes
	Threshold    float64 // For internal nodes
	Left         *GBTreeNode
	Right        *GBTreeNode
	SamplesCount int
}

// SplitCandidate represents a potential split point
type SplitCandidate struct {
	FeatureIndex int
	Threshold    float64
	Gain         float64
	LeftIndices  []int
	RightIndices []int
}

// NewGradientBoostingModel creates a new gradient boosting model
func NewGradientBoostingModel() *GradientBoostingModel {
	dataDir := "data/models"
	os.MkdirAll(dataDir, 0755)

	gbm := &GradientBoostingModel{
		trees:             []*GBTree{},
		learningRate:      0.1,
		numTrees:          100,
		maxDepth:          3,
		minSamplesLeaf:    5,
		weight:            0.10, // 10% weight in ensemble
		trained:           false,
		dataDir:           dataDir,
		featureNames:      make([]string, 111), // 111 features (including xG + shift + landing + summary metrics)
		featureImportance: make(map[string]float64),
	}

	// Initialize feature names
	for i := 0; i < 111; i++ {
		gbm.featureNames[i] = fmt.Sprintf("feature_%d", i)
	}

	// Try to load existing model
	gbm.loadModel()

	return gbm
}

// Predict makes a prediction using gradient boosting
func (gbm *GradientBoostingModel) Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error) {
	gbm.mutex.RLock()
	defer gbm.mutex.RUnlock()

	start := time.Now()

	if !gbm.trained || len(gbm.trees) == 0 {
		// Return neutral prediction if not trained
		return &models.ModelResult{
			ModelName:      "Gradient Boosting",
			WinProbability: 0.50,
			Confidence:     0.30,
			PredictedScore: "3-2",
			Weight:         gbm.weight,
			ProcessingTime: time.Since(start).Milliseconds(),
		}, nil
	}

	// Extract features
	features := gbm.extractFeatures(homeFactors, awayFactors)

	// Get prediction from all trees
	prediction := gbm.predictProbability(features)

	// Convert to win probability (sigmoid)
	winProb := 1.0 / (1.0 + math.Exp(-prediction))

	// Ensure reasonable bounds
	winProb = math.Max(0.35, math.Min(0.85, winProb))

	// Calculate confidence based on prediction strength
	confidence := math.Abs(winProb-0.5) * 2.0 // 0 at 50%, 1.0 at 0% or 100%
	confidence = math.Max(0.40, math.Min(0.90, confidence))

	// Predict score
	predictedScore := gbm.predictScore(winProb, homeFactors, awayFactors)

	result := &models.ModelResult{
		ModelName:      "Gradient Boosting",
		WinProbability: winProb,
		Confidence:     confidence,
		PredictedScore: predictedScore,
		Weight:         gbm.weight,
		ProcessingTime: time.Since(start).Milliseconds(),
	}

	return result, nil
}

// Train trains the gradient boosting model on game results
func (gbm *GradientBoostingModel) Train(games []models.CompletedGame) error {
	gbm.mutex.Lock()
	defer gbm.mutex.Unlock()

	if len(games) < 10 {
		return fmt.Errorf("insufficient training data: need at least 10 games, have %d", len(games))
	}

	log.Printf("🌳 Training Gradient Boosting model on %d games...", len(games))
	start := time.Now()

	// Prepare training data
	features, labels := gbm.prepareTrainingData(games)

	// Initialize predictions with 0 (neutral)
	predictions := make([]float64, len(labels))

	// Train trees sequentially
	gbm.trees = []*GBTree{}

	for t := 0; t < gbm.numTrees; t++ {
		// Calculate residuals (negative gradients for log loss)
		residuals := make([]float64, len(labels))
		for i := range labels {
			// Convert to probability
			prob := 1.0 / (1.0 + math.Exp(-predictions[i]))
			// Residual is (actual - predicted)
			residuals[i] = labels[i] - prob
		}

		// Build tree to predict residuals
		tree := &GBTree{
			MaxDepth:       gbm.maxDepth,
			MinSamplesLeaf: gbm.minSamplesLeaf,
		}

		indices := make([]int, len(features))
		for i := range indices {
			indices[i] = i
		}

		tree.Root = gbm.buildTree(features, residuals, indices, 0)
		gbm.trees = append(gbm.trees, tree)

		// Update predictions
		for i := range predictions {
			treeOut := gbm.predictTree(tree, features[i])
			predictions[i] += gbm.learningRate * treeOut
		}

		// Log progress every 20 trees
		if (t+1)%20 == 0 {
			accuracy := gbm.calculateAccuracy(predictions, labels)
			log.Printf("   Tree %d/%d: Accuracy %.2f%%", t+1, gbm.numTrees, accuracy*100)
		}
	}

	// Calculate final training accuracy
	finalAccuracy := gbm.calculateAccuracy(predictions, labels)

	// Calculate feature importance
	gbm.calculateFeatureImportance()

	gbm.trained = true

	trainingTime := time.Since(start)
	log.Printf("✅ Gradient Boosting training complete!")
	log.Printf("   Trees: %d | Accuracy: %.2f%% | Time: %.1fs",
		len(gbm.trees), finalAccuracy*100, trainingTime.Seconds())

	// Save model
	gbm.saveModel()

	return nil
}

// buildTree recursively builds a decision tree
func (gbm *GradientBoostingModel) buildTree(features [][]float64, targets []float64, indices []int, depth int) *GBTreeNode {
	// Create node
	node := &GBTreeNode{
		SamplesCount: len(indices),
	}

	// Stop conditions
	if depth >= gbm.maxDepth || len(indices) <= gbm.minSamplesLeaf {
		// Make leaf node
		node.IsLeaf = true
		node.Prediction = gbm.mean(targets, indices)
		return node
	}

	// Check if all targets are the same (pure node)
	if gbm.isHomogeneous(targets, indices) {
		node.IsLeaf = true
		node.Prediction = targets[indices[0]]
		return node
	}

	// Find best split
	bestSplit := gbm.findBestSplit(features, targets, indices)

	if bestSplit == nil || bestSplit.Gain <= 0.0001 {
		// No good split found, make leaf
		node.IsLeaf = true
		node.Prediction = gbm.mean(targets, indices)
		return node
	}

	// Create internal node
	node.IsLeaf = false
	node.FeatureIndex = bestSplit.FeatureIndex
	node.Threshold = bestSplit.Threshold

	// Recursively build children
	node.Left = gbm.buildTree(features, targets, bestSplit.LeftIndices, depth+1)
	node.Right = gbm.buildTree(features, targets, bestSplit.RightIndices, depth+1)

	return node
}

// findBestSplit finds the best split for a node
func (gbm *GradientBoostingModel) findBestSplit(features [][]float64, targets []float64, indices []int) *SplitCandidate {
	if len(indices) < 2 {
		return nil
	}

	numFeatures := len(features[0])
	var bestSplit *SplitCandidate
	bestGain := -1.0

	// Current variance
	parentVariance := gbm.variance(targets, indices)

	// Try each feature
	for featureIdx := 0; featureIdx < numFeatures; featureIdx++ {
		// Get unique values for this feature
		values := make([]float64, len(indices))
		for i, idx := range indices {
			values[i] = features[idx][featureIdx]
		}

		// Try different thresholds
		uniqueValues := gbm.uniqueSorted(values)
		if len(uniqueValues) < 2 {
			continue
		}

		// Try splits between consecutive values
		for i := 0; i < len(uniqueValues)-1; i++ {
			threshold := (uniqueValues[i] + uniqueValues[i+1]) / 2.0

			// Split indices
			var leftIndices, rightIndices []int
			for _, idx := range indices {
				if features[idx][featureIdx] <= threshold {
					leftIndices = append(leftIndices, idx)
				} else {
					rightIndices = append(rightIndices, idx)
				}
			}

			// Check minimum samples
			if len(leftIndices) < gbm.minSamplesLeaf || len(rightIndices) < gbm.minSamplesLeaf {
				continue
			}

			// Calculate gain
			leftVariance := gbm.variance(targets, leftIndices)
			rightVariance := gbm.variance(targets, rightIndices)

			leftWeight := float64(len(leftIndices)) / float64(len(indices))
			rightWeight := float64(len(rightIndices)) / float64(len(indices))

			gain := parentVariance - (leftWeight*leftVariance + rightWeight*rightVariance)

			if gain > bestGain {
				bestGain = gain
				bestSplit = &SplitCandidate{
					FeatureIndex: featureIdx,
					Threshold:    threshold,
					Gain:         gain,
					LeftIndices:  leftIndices,
					RightIndices: rightIndices,
				}
			}
		}
	}

	return bestSplit
}

// predictTree makes a prediction using a single tree
func (gbm *GradientBoostingModel) predictTree(tree *GBTree, features []float64) float64 {
	return gbm.traverseTree(tree.Root, features)
}

// traverseTree recursively traverses a tree to get prediction
func (gbm *GradientBoostingModel) traverseTree(node *GBTreeNode, features []float64) float64 {
	if node.IsLeaf {
		return node.Prediction
	}

	if features[node.FeatureIndex] <= node.Threshold {
		return gbm.traverseTree(node.Left, features)
	}
	return gbm.traverseTree(node.Right, features)
}

// predictProbability gets probability from all trees
func (gbm *GradientBoostingModel) predictProbability(features []float64) float64 {
	prediction := 0.0
	for _, tree := range gbm.trees {
		prediction += gbm.learningRate * gbm.predictTree(tree, features)
	}
	return prediction
}

// Helper functions

func (gbm *GradientBoostingModel) mean(values []float64, indices []int) float64 {
	if len(indices) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, idx := range indices {
		sum += values[idx]
	}
	return sum / float64(len(indices))
}

func (gbm *GradientBoostingModel) variance(values []float64, indices []int) float64 {
	if len(indices) == 0 {
		return 0.0
	}
	mean := gbm.mean(values, indices)
	variance := 0.0
	for _, idx := range indices {
		diff := values[idx] - mean
		variance += diff * diff
	}
	return variance / float64(len(indices))
}

func (gbm *GradientBoostingModel) isHomogeneous(values []float64, indices []int) bool {
	if len(indices) <= 1 {
		return true
	}
	first := values[indices[0]]
	for _, idx := range indices[1:] {
		if math.Abs(values[idx]-first) > 0.0001 {
			return false
		}
	}
	return true
}

func (gbm *GradientBoostingModel) uniqueSorted(values []float64) []float64 {
	uniqueMap := make(map[float64]bool)
	for _, v := range values {
		uniqueMap[v] = true
	}

	unique := make([]float64, 0, len(uniqueMap))
	for v := range uniqueMap {
		unique = append(unique, v)
	}

	sort.Float64s(unique)
	return unique
}

func (gbm *GradientBoostingModel) calculateAccuracy(predictions, labels []float64) float64 {
	correct := 0
	for i := range predictions {
		prob := 1.0 / (1.0 + math.Exp(-predictions[i]))
		predicted := 0.0
		if prob > 0.5 {
			predicted = 1.0
		}
		if predicted == labels[i] {
			correct++
		}
	}
	return float64(correct) / float64(len(labels))
}

// prepareTrainingData converts games to feature matrix and labels
func (gbm *GradientBoostingModel) prepareTrainingData(games []models.CompletedGame) ([][]float64, []float64) {
	features := make([][]float64, len(games))
	labels := make([]float64, len(games))

	for i, game := range games {
		// Extract features (same as Neural Network)
		homeFactors := &models.PredictionFactors{TeamCode: game.HomeTeam.TeamCode}
		awayFactors := &models.PredictionFactors{TeamCode: game.AwayTeam.TeamCode}

		features[i] = gbm.extractFeatures(homeFactors, awayFactors)

		// Label: 1.0 if home won, 0.0 if away won
		if game.HomeTeam.Score > game.AwayTeam.Score {
			labels[i] = 1.0
		} else {
			labels[i] = 0.0
		}
	}

	return features, labels
}

// extractFeatures extracts 75 features from prediction factors
func (gbm *GradientBoostingModel) extractFeatures(home, away *models.PredictionFactors) []float64 {
	features := make([]float64, 111) // Expanded to 111 to include xG, play-by-play, shift, landing, and summary analytics

	// Basic features (indices 0-9)
	features[0] = home.WinPercentage
	features[1] = away.WinPercentage
	features[2] = home.GoalsFor / 82.0
	features[3] = away.GoalsFor / 82.0
	features[4] = home.GoalsAgainst / 82.0
	features[5] = away.GoalsAgainst / 82.0
	features[6] = home.PowerPlayPct
	features[7] = away.PowerPlayPct
	features[8] = home.PenaltyKillPct
	features[9] = away.PenaltyKillPct

	// Phase 4 features (indices 50-64)
	features[50] = home.GoalieAdvantage
	features[51] = away.GoalieAdvantage
	features[52] = home.MarketConsensus
	features[53] = away.MarketConsensus
	features[54] = home.TravelDistance / 3000.0
	features[55] = away.TravelDistance / 3000.0
	features[56] = home.BackToBackIndicator
	features[57] = away.BackToBackIndicator
	features[58] = home.ScheduleDensity / 7.0
	features[59] = away.ScheduleDensity / 7.0
	features[60] = home.RestAdvantage / 5.0
	features[61] = away.RestAdvantage / 5.0
	features[62] = float64(home.RestDays) / 5.0
	features[63] = float64(away.RestDays) / 5.0
	features[64] = 1.0 // Home ice indicator

	// Player Intelligence features (indices 65-74)
	features[65] = home.StarPowerRating
	features[66] = away.StarPowerRating
	features[67] = home.Top3CombinedPPG / 4.0
	features[68] = away.Top3CombinedPPG / 4.0
	features[69] = home.TopScorerForm / 10.0
	features[70] = away.TopScorerForm / 10.0
	features[71] = home.DepthForm / 10.0
	features[72] = away.DepthForm / 10.0
	features[73] = home.StarPowerEdge
	features[74] = home.DepthEdge

	// Play-by-Play Analytics: xG and Shot Quality (indices 75-92)
	features[75] = home.ExpectedGoalsFor / 4.0
	features[76] = away.ExpectedGoalsFor / 4.0
	features[77] = home.ExpectedGoalsAgainst / 4.0
	features[78] = away.ExpectedGoalsAgainst / 4.0
	features[79] = home.XGDifferential / 2.0
	features[80] = away.XGDifferential / 2.0
	features[81] = home.XGPerShot / 0.15
	features[82] = away.XGPerShot / 0.15
	features[83] = home.DangerousShotsPerGame / 15.0
	features[84] = away.DangerousShotsPerGame / 15.0
	features[85] = home.HighDangerXG / 3.0
	features[86] = away.HighDangerXG / 3.0
	features[87] = home.ShotQualityAdvantage / 0.05
	features[88] = home.CorsiForPct
	features[89] = away.CorsiForPct
	features[90] = home.FenwickForPct
	features[91] = away.FenwickForPct
	features[92] = home.FaceoffWinPct

	// Shift Analysis: Line Chemistry & Coaching Tendencies (indices 93-100)
	features[93] = home.AvgShiftLength / 60.0
	features[94] = away.AvgShiftLength / 60.0
	features[95] = home.LineConsistency
	features[96] = away.LineConsistency
	features[97] = home.ShortBench
	features[98] = away.ShortBench
	features[99] = home.FatigueIndicator
	features[100] = away.FatigueIndicator

	// Landing Page Analytics: Enhanced Physical Play & Zone Control (indices 101-104)
	features[101] = home.TimeOnAttack / 30.0
	features[102] = away.TimeOnAttack / 30.0
	features[103] = home.ZoneControlRatio
	features[104] = away.ZoneControlRatio

	// Game Summary Analytics: Enhanced Game Context (indices 105-110)
	features[105] = home.ShotQualityIndex
	features[106] = away.ShotQualityIndex
	features[107] = home.PowerPlayTime / 10.0
	features[108] = away.PowerPlayTime / 10.0
	features[109] = home.OffensiveZoneTime / 30.0
	features[110] = away.OffensiveZoneTime / 30.0

	return features
}

// predictScore predicts the final score
func (gbm *GradientBoostingModel) predictScore(winProb float64, home, away *models.PredictionFactors) string {
	// Simple score prediction based on probability
	if winProb > 0.65 {
		return "4-2" // Comfortable home win
	} else if winProb > 0.55 {
		return "3-2" // Close home win
	} else if winProb > 0.45 {
		return "3-3 (OT)" // Could go either way
	} else if winProb > 0.35 {
		return "2-3" // Close away win
	}
	return "2-4" // Comfortable away win
}

// calculateFeatureImportance calculates which features are most important
func (gbm *GradientBoostingModel) calculateFeatureImportance() {
	importance := make(map[int]float64)

	// Sum up splits across all trees
	for _, tree := range gbm.trees {
		gbm.addImportanceFromTree(tree.Root, importance)
	}

	// Convert to named features
	gbm.featureImportance = make(map[string]float64)
	for idx, imp := range importance {
		if idx < len(gbm.featureNames) {
			gbm.featureImportance[gbm.featureNames[idx]] = imp
		}
	}

	// Log top 10 features
	type featureImp struct {
		name       string
		importance float64
	}

	features := make([]featureImp, 0, len(gbm.featureImportance))
	for name, imp := range gbm.featureImportance {
		features = append(features, featureImp{name, imp})
	}

	sort.Slice(features, func(i, j int) bool {
		return features[i].importance > features[j].importance
	})

	log.Printf("📊 Top 10 Important Features:")
	for i := 0; i < 10 && i < len(features); i++ {
		log.Printf("   %d. %s: %.2f", i+1, features[i].name, features[i].importance)
	}
}

func (gbm *GradientBoostingModel) addImportanceFromTree(node *GBTreeNode, importance map[int]float64) {
	if node == nil || node.IsLeaf {
		return
	}

	// Add importance for this split
	importance[node.FeatureIndex] += float64(node.SamplesCount)

	// Recurse
	gbm.addImportanceFromTree(node.Left, importance)
	gbm.addImportanceFromTree(node.Right, importance)
}

// Model persistence

// GradientBoostingModelData represents serializable model data
type GradientBoostingModelData struct {
	Trees             []SerializedGBTree `json:"trees"`
	LearningRate      float64            `json:"learningRate"`
	NumTrees          int                `json:"numTrees"`
	MaxDepth          int                `json:"maxDepth"`
	MinSamplesLeaf    int                `json:"minSamplesLeaf"`
	Weight            float64            `json:"weight"`
	Trained           bool               `json:"trained"`
	FeatureNames      []string           `json:"featureNames"`
	FeatureImportance map[string]float64 `json:"featureImportance"`
	LastUpdated       time.Time          `json:"lastUpdated"`
	Version           string             `json:"version"`
}

// SerializedGBTree represents a serializable decision tree
type SerializedGBTree struct {
	Root           *SerializedGBTreeNode `json:"root"`
	MaxDepth       int                   `json:"maxDepth"`
	MinSamplesLeaf int                   `json:"minSamplesLeaf"`
}

// SerializedGBTreeNode represents a serializable tree node
type SerializedGBTreeNode struct {
	IsLeaf       bool                  `json:"isLeaf"`
	Prediction   float64               `json:"prediction"`
	FeatureIndex int                   `json:"featureIndex"`
	Threshold    float64               `json:"threshold"`
	Left         *SerializedGBTreeNode `json:"left,omitempty"`
	Right        *SerializedGBTreeNode `json:"right,omitempty"`
	SamplesCount int                   `json:"samplesCount"`
}

func (gbm *GradientBoostingModel) saveModel() error {
	filePath := filepath.Join(gbm.dataDir, "gradient_boosting.json")

	// Serialize trees
	serializedTrees := make([]SerializedGBTree, len(gbm.trees))
	for i, tree := range gbm.trees {
		serializedTrees[i] = SerializedGBTree{
			Root:           serializeGBTreeNode(tree.Root),
			MaxDepth:       tree.MaxDepth,
			MinSamplesLeaf: tree.MinSamplesLeaf,
		}
	}

	// Create model data
	modelData := GradientBoostingModelData{
		Trees:             serializedTrees,
		LearningRate:      gbm.learningRate,
		NumTrees:          gbm.numTrees,
		MaxDepth:          gbm.maxDepth,
		MinSamplesLeaf:    gbm.minSamplesLeaf,
		Weight:            gbm.weight,
		Trained:           gbm.trained,
		FeatureNames:      gbm.featureNames,
		FeatureImportance: gbm.featureImportance,
		LastUpdated:       time.Now(),
		Version:           "1.0",
	}

	data, err := json.MarshalIndent(modelData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling gradient boosting model: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing gradient boosting model: %w", err)
	}

	log.Printf("💾 Gradient Boosting model saved: %d trees, trained=%v", len(gbm.trees), gbm.trained)
	return nil
}

// serializeGBTreeNode recursively serializes a tree node
func serializeGBTreeNode(node *GBTreeNode) *SerializedGBTreeNode {
	if node == nil {
		return nil
	}

	serialized := &SerializedGBTreeNode{
		IsLeaf:       node.IsLeaf,
		Prediction:   node.Prediction,
		FeatureIndex: node.FeatureIndex,
		Threshold:    node.Threshold,
		SamplesCount: node.SamplesCount,
	}

	if node.Left != nil {
		serialized.Left = serializeGBTreeNode(node.Left)
	}
	if node.Right != nil {
		serialized.Right = serializeGBTreeNode(node.Right)
	}

	return serialized
}

func (gbm *GradientBoostingModel) loadModel() error {
	filePath := filepath.Join(gbm.dataDir, "gradient_boosting.json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("🌳 No saved Gradient Boosting model found, starting fresh")
		return nil // Not an error
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading gradient boosting model: %w", err)
	}

	var modelData GradientBoostingModelData
	err = json.Unmarshal(data, &modelData)
	if err != nil {
		return fmt.Errorf("error unmarshaling gradient boosting model: %w", err)
	}

	// Deserialize trees
	gbm.trees = make([]*GBTree, len(modelData.Trees))
	for i, serializedTree := range modelData.Trees {
		gbm.trees[i] = &GBTree{
			Root:           deserializeGBTreeNode(serializedTree.Root),
			MaxDepth:       serializedTree.MaxDepth,
			MinSamplesLeaf: serializedTree.MinSamplesLeaf,
		}
	}

	// Load other fields
	gbm.learningRate = modelData.LearningRate
	gbm.numTrees = modelData.NumTrees
	gbm.maxDepth = modelData.MaxDepth
	gbm.minSamplesLeaf = modelData.MinSamplesLeaf
	gbm.weight = modelData.Weight
	gbm.trained = modelData.Trained
	gbm.featureNames = modelData.FeatureNames
	gbm.featureImportance = modelData.FeatureImportance

	log.Printf("🌳 Gradient Boosting model loaded: %d trees, trained=%v", len(gbm.trees), gbm.trained)
	log.Printf("   Last updated: %s", modelData.LastUpdated.Format("2006-01-02 15:04:05"))
	return nil
}

// deserializeGBTreeNode recursively deserializes a tree node
func deserializeGBTreeNode(serialized *SerializedGBTreeNode) *GBTreeNode {
	if serialized == nil {
		return nil
	}

	node := &GBTreeNode{
		IsLeaf:       serialized.IsLeaf,
		Prediction:   serialized.Prediction,
		FeatureIndex: serialized.FeatureIndex,
		Threshold:    serialized.Threshold,
		SamplesCount: serialized.SamplesCount,
	}

	if serialized.Left != nil {
		node.Left = deserializeGBTreeNode(serialized.Left)
	}
	if serialized.Right != nil {
		node.Right = deserializeGBTreeNode(serialized.Right)
	}

	return node
}

// GetName returns the model name
func (gbm *GradientBoostingModel) GetName() string {
	return "Gradient Boosting"
}

// GetWeight returns the model weight in ensemble
func (gbm *GradientBoostingModel) GetWeight() float64 {
	gbm.mutex.RLock()
	defer gbm.mutex.RUnlock()
	return gbm.weight
}

// TrainOnGameResult trains the model on a completed game
func (gbm *GradientBoostingModel) TrainOnGameResult(game models.CompletedGame) error {
	// Gradient Boosting needs batch training, so we'll collect games and train periodically
	// For now, just log that we received a game
	log.Printf("🌳 Gradient Boosting: Received game result (batch training needed)")
	return nil
}
