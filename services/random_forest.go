package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// RandomForestModel implements a random forest classifier for NHL predictions
type RandomForestModel struct {
	trees             []*RFTree
	numTrees          int
	maxDepth          int
	minSamplesLeaf    int
	maxFeatures       int // Number of features to consider at each split
	weight            float64
	trained           bool
	dataDir           string
	mutex             sync.RWMutex
	featureNames      []string
	featureImportance map[string]float64
}

// RFTree represents a single decision tree in the random forest
type RFTree struct {
	Root           *RFTreeNode
	MaxDepth       int
	MinSamplesLeaf int
	MaxFeatures    int
}

// RFTreeNode represents a node in the random forest tree (same structure as GBTreeNode)
type RFTreeNode struct {
	IsLeaf       bool
	Prediction   float64 // For leaf nodes (class probability)
	FeatureIndex int     // For internal nodes
	Threshold    float64 // For internal nodes
	Left         *RFTreeNode
	Right        *RFTreeNode
	SamplesCount int
	ClassCounts  map[int]int // Count of each class in this node
}

// NewRandomForestModel creates a new random forest model
func NewRandomForestModel() *RandomForestModel {
	dataDir := "data/models"
	os.MkdirAll(dataDir, 0755)

	numFeatures := 65
	maxFeatures := int(math.Sqrt(float64(numFeatures))) // sqrt(65) ≈ 8

	rfm := &RandomForestModel{
		trees:             []*RFTree{},
		numTrees:          100, // 100 trees in forest
		maxDepth:          6,   // Deeper than GB (less overfitting risk)
		minSamplesLeaf:    3,   // Allow smaller leaves
		maxFeatures:       maxFeatures,
		weight:            0.07, // 7% weight in ensemble
		trained:           false,
		dataDir:           dataDir,
		featureNames:      make([]string, numFeatures),
		featureImportance: make(map[string]float64),
	}

	// Initialize feature names
	for i := 0; i < numFeatures; i++ {
		rfm.featureNames[i] = fmt.Sprintf("feature_%d", i)
	}

	// Try to load existing model
	rfm.loadModel()

	return rfm
}

// Predict makes a prediction using random forest
func (rfm *RandomForestModel) Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error) {
	rfm.mutex.RLock()
	defer rfm.mutex.RUnlock()

	start := time.Now()

	if !rfm.trained || len(rfm.trees) == 0 {
		// Return neutral prediction if not trained
		return &models.ModelResult{
			ModelName:      "Random Forest",
			WinProbability: 0.50,
			Confidence:     0.30,
			PredictedScore: "3-2",
			Weight:         rfm.weight,
			ProcessingTime: time.Since(start).Milliseconds(),
		}, nil
	}

	// Extract features
	features := rfm.extractFeatures(homeFactors, awayFactors)

	// Get predictions from all trees
	votes := make([]float64, 3) // [win, loss, ot]

	for _, tree := range rfm.trees {
		prediction := rfm.predictTree(tree, features)
		// prediction is class probability from this tree
		if prediction > 0.6 {
			votes[0]++ // Win
		} else if prediction < 0.4 {
			votes[1]++ // Loss
		} else {
			votes[2]++ // OT
		}
	}

	// Majority voting
	totalVotes := votes[0] + votes[1] + votes[2]
	winProb := votes[0] / totalVotes

	// Ensure reasonable bounds
	winProb = math.Max(0.35, math.Min(0.85, winProb))

	// Calculate confidence based on vote agreement
	maxVotes := math.Max(votes[0], math.Max(votes[1], votes[2]))
	confidence := maxVotes / totalVotes // High if trees agree
	confidence = math.Max(0.40, math.Min(0.90, confidence))

	// Predict score
	predictedScore := rfm.predictScore(winProb, homeFactors, awayFactors)

	result := &models.ModelResult{
		ModelName:      "Random Forest",
		WinProbability: winProb,
		Confidence:     confidence,
		PredictedScore: predictedScore,
		Weight:         rfm.weight,
		ProcessingTime: time.Since(start).Milliseconds(),
	}

	return result, nil
}

// predictTree gets prediction from a single tree
func (rfm *RandomForestModel) predictTree(tree *RFTree, features []float64) float64 {
	return rfm.traverseTree(tree.Root, features)
}

// traverseTree traverses a tree to get prediction
func (rfm *RandomForestModel) traverseTree(node *RFTreeNode, features []float64) float64 {
	if node.IsLeaf {
		return node.Prediction
	}

	if features[node.FeatureIndex] <= node.Threshold {
		return rfm.traverseTree(node.Left, features)
	}
	return rfm.traverseTree(node.Right, features)
}

// Train trains the random forest model on game results
func (rfm *RandomForestModel) Train(games []models.CompletedGame) error {
	rfm.mutex.Lock()
	defer rfm.mutex.Unlock()

	if len(games) < 20 {
		return fmt.Errorf("insufficient training data: need at least 20 games, have %d", len(games))
	}

	log.Printf("🌲 Training Random Forest model on %d games...", len(games))
	start := time.Now()

	// Prepare training data
	features, labels := rfm.prepareTrainingData(games)
	numSamples := len(labels)

	// Train trees in parallel (key difference from GB!)
	rfm.trees = make([]*RFTree, rfm.numTrees)

	for t := 0; t < rfm.numTrees; t++ {
		// Bootstrap sampling (random subset with replacement)
		bootstrapIndices := rfm.bootstrapSample(numSamples)

		// Build tree on bootstrap sample
		tree := &RFTree{
			MaxDepth:       rfm.maxDepth,
			MinSamplesLeaf: rfm.minSamplesLeaf,
			MaxFeatures:    rfm.maxFeatures,
		}

		tree.Root = rfm.buildTree(features, labels, bootstrapIndices, 0)
		rfm.trees[t] = tree

		// Log progress every 20 trees
		if (t+1)%20 == 0 {
			log.Printf("   Tree %d/%d built", t+1, rfm.numTrees)
		}
	}

	// Calculate feature importance
	rfm.calculateFeatureImportance()

	rfm.trained = true

	trainingTime := time.Since(start)
	log.Printf("✅ Random Forest training complete!")
	log.Printf("   Trees: %d | Time: %.1fs", len(rfm.trees), trainingTime.Seconds())

	// Save model
	if err := rfm.saveModel(); err != nil {
		log.Printf("⚠️ Failed to save Random Forest model: %v", err)
	}

	return nil
}

// bootstrapSample creates a bootstrap sample (random indices with replacement)
func (rfm *RandomForestModel) bootstrapSample(numSamples int) []int {
	indices := make([]int, numSamples)
	for i := 0; i < numSamples; i++ {
		indices[i] = rand.Intn(numSamples)
	}
	return indices
}

// buildTree recursively builds a decision tree with random feature selection
func (rfm *RandomForestModel) buildTree(features [][]float64, labels []float64, indices []int, depth int) *RFTreeNode {
	// Create node
	node := &RFTreeNode{
		SamplesCount: len(indices),
		ClassCounts:  make(map[int]int),
	}

	// Count classes
	for _, idx := range indices {
		class := int(labels[idx])
		node.ClassCounts[class]++
	}

	// Check stopping criteria
	if depth >= rfm.maxDepth || len(indices) < rfm.minSamplesLeaf*2 || rfm.isPure(node.ClassCounts) {
		node.IsLeaf = true
		node.Prediction = rfm.getMajorityClass(node.ClassCounts)
		return node
	}

	// Find best split with RANDOM FEATURE SUBSET (key difference from GB!)
	bestSplit := rfm.findBestSplit(features, labels, indices)

	if bestSplit == nil || len(bestSplit.LeftIndices) < rfm.minSamplesLeaf || len(bestSplit.RightIndices) < rfm.minSamplesLeaf {
		node.IsLeaf = true
		node.Prediction = rfm.getMajorityClass(node.ClassCounts)
		return node
	}

	// Split node
	node.IsLeaf = false
	node.FeatureIndex = bestSplit.FeatureIndex
	node.Threshold = bestSplit.Threshold
	node.Left = rfm.buildTree(features, labels, bestSplit.LeftIndices, depth+1)
	node.Right = rfm.buildTree(features, labels, bestSplit.RightIndices, depth+1)

	return node
}

// findBestSplit finds the best split using RANDOM FEATURE SUBSET
func (rfm *RandomForestModel) findBestSplit(features [][]float64, labels []float64, indices []int) *SplitCandidate {
	if len(indices) == 0 {
		return nil
	}

	numFeatures := len(features[0])

	// RANDOM FEATURE SELECTION (key difference from GB!)
	selectedFeatures := rfm.selectRandomFeatures(numFeatures)

	var bestSplit *SplitCandidate
	bestGain := -math.MaxFloat64

	// Try each selected feature
	for _, featureIdx := range selectedFeatures {
		// Get unique values for this feature
		values := make([]float64, 0)
		for _, idx := range indices {
			values = append(values, features[idx][featureIdx])
		}

		// Try splits at unique values
		uniqueValues := rfm.getUniqueValues(values)

		for _, threshold := range uniqueValues {
			split := rfm.evaluateSplit(features, labels, indices, featureIdx, threshold)

			if split != nil && split.Gain > bestGain {
				bestGain = split.Gain
				bestSplit = split
			}
		}
	}

	return bestSplit
}

// selectRandomFeatures randomly selects maxFeatures features
func (rfm *RandomForestModel) selectRandomFeatures(numFeatures int) []int {
	// Create array of all feature indices
	allFeatures := make([]int, numFeatures)
	for i := 0; i < numFeatures; i++ {
		allFeatures[i] = i
	}

	// Shuffle and take first maxFeatures
	rand.Shuffle(len(allFeatures), func(i, j int) {
		allFeatures[i], allFeatures[j] = allFeatures[j], allFeatures[i]
	})

	maxFeatures := rfm.maxFeatures
	if maxFeatures > numFeatures {
		maxFeatures = numFeatures
	}

	return allFeatures[:maxFeatures]
}

// getUniqueValues returns sorted unique values
func (rfm *RandomForestModel) getUniqueValues(values []float64) []float64 {
	if len(values) == 0 {
		return []float64{}
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Get unique values
	unique := []float64{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1] {
			unique = append(unique, sorted[i])
		}
	}

	// Sample thresholds (use midpoints)
	thresholds := make([]float64, 0)
	for i := 0; i < len(unique)-1; i++ {
		thresholds = append(thresholds, (unique[i]+unique[i+1])/2.0)
	}

	// Limit to 10 thresholds max for efficiency
	if len(thresholds) > 10 {
		step := len(thresholds) / 10
		sampled := make([]float64, 0, 10)
		for i := 0; i < len(thresholds); i += step {
			sampled = append(sampled, thresholds[i])
		}
		return sampled
	}

	return thresholds
}

// evaluateSplit evaluates a potential split
func (rfm *RandomForestModel) evaluateSplit(features [][]float64, labels []float64, indices []int, featureIdx int, threshold float64) *SplitCandidate {
	leftIndices := make([]int, 0)
	rightIndices := make([]int, 0)

	for _, idx := range indices {
		if features[idx][featureIdx] <= threshold {
			leftIndices = append(leftIndices, idx)
		} else {
			rightIndices = append(rightIndices, idx)
		}
	}

	if len(leftIndices) == 0 || len(rightIndices) == 0 {
		return nil
	}

	// Calculate Gini impurity gain
	parentGini := rfm.calculateGini(labels, indices)
	leftGini := rfm.calculateGini(labels, leftIndices)
	rightGini := rfm.calculateGini(labels, rightIndices)

	n := float64(len(indices))
	nLeft := float64(len(leftIndices))
	nRight := float64(len(rightIndices))

	gain := parentGini - (nLeft/n)*leftGini - (nRight/n)*rightGini

	return &SplitCandidate{
		FeatureIndex: featureIdx,
		Threshold:    threshold,
		Gain:         gain,
		LeftIndices:  leftIndices,
		RightIndices: rightIndices,
	}
}

// calculateGini calculates Gini impurity
func (rfm *RandomForestModel) calculateGini(labels []float64, indices []int) float64 {
	if len(indices) == 0 {
		return 0.0
	}

	classCounts := make(map[int]int)
	for _, idx := range indices {
		class := int(labels[idx])
		classCounts[class]++
	}

	gini := 1.0
	n := float64(len(indices))

	for _, count := range classCounts {
		prob := float64(count) / n
		gini -= prob * prob
	}

	return gini
}

// isPure checks if all samples belong to same class
func (rfm *RandomForestModel) isPure(classCounts map[int]int) bool {
	return len(classCounts) == 1
}

// getMajorityClass returns the majority class as a probability
func (rfm *RandomForestModel) getMajorityClass(classCounts map[int]int) float64 {
	maxCount := 0
	majorityClass := 0
	total := 0

	for class, count := range classCounts {
		total += count
		if count > maxCount {
			maxCount = count
			majorityClass = class
		}
	}

	// Return probability (0.0 = loss, 0.5 = OT, 1.0 = win)
	if majorityClass == 1 {
		return 1.0 // Win
	} else if majorityClass == 2 {
		return 0.5 // OT
	}
	return 0.0 // Loss
}

// prepareTrainingData prepares features and labels from games
func (rfm *RandomForestModel) prepareTrainingData(games []models.CompletedGame) ([][]float64, []float64) {
	features := make([][]float64, 0, len(games)*2)
	labels := make([]float64, 0, len(games)*2)

	for _, game := range games {
		// Extract features for home team perspective
		homeFeatures := rfm.extractGameFeatures(&game, true)
		features = append(features, homeFeatures)

		// Label: 1 = win, 0 = loss, 2 = OT loss
		if game.HomeTeam.Score > game.AwayTeam.Score {
			labels = append(labels, 1.0)
		} else if game.WinType == "OT" || game.WinType == "SO" {
			labels = append(labels, 2.0) // OT loss
		} else {
			labels = append(labels, 0.0)
		}

		// Extract features for away team perspective
		awayFeatures := rfm.extractGameFeatures(&game, false)
		features = append(features, awayFeatures)

		if game.AwayTeam.Score > game.HomeTeam.Score {
			labels = append(labels, 1.0)
		} else if game.WinType == "OT" || game.WinType == "SO" {
			labels = append(labels, 2.0)
		} else {
			labels = append(labels, 0.0)
		}
	}

	return features, labels
}

// extractGameFeatures extracts features from a completed game
func (rfm *RandomForestModel) extractGameFeatures(game *models.CompletedGame, isHome bool) []float64 {
	features := make([]float64, 65)

	// Basic features
	if isHome {
		features[0] = float64(game.HomeTeam.Score) / 10.0
		features[1] = float64(game.AwayTeam.Score) / 10.0
		features[2] = float64(game.HomeTeam.Shots) / 40.0
		features[3] = float64(game.AwayTeam.Shots) / 40.0
	} else {
		features[0] = float64(game.AwayTeam.Score) / 10.0
		features[1] = float64(game.HomeTeam.Score) / 10.0
		features[2] = float64(game.AwayTeam.Shots) / 40.0
		features[3] = float64(game.HomeTeam.Shots) / 40.0
	}

	// Win type
	if game.WinType == "OT" {
		features[4] = 0.5
	} else if game.WinType == "SO" {
		features[4] = 0.7
	} else {
		features[4] = 1.0
	}

	// Pad remaining features
	for i := 5; i < 65; i++ {
		features[i] = 0.5 // Neutral value
	}

	return features
}

// extractFeatures extracts features for prediction
func (rfm *RandomForestModel) extractFeatures(homeFactors, awayFactors *models.PredictionFactors) []float64 {
	features := make([]float64, 65)
	idx := 0

	// Home team features
	features[idx] = homeFactors.WinPercentage
	idx++
	features[idx] = homeFactors.GoalsFor / 5.0
	idx++
	features[idx] = homeFactors.GoalsAgainst / 5.0
	idx++
	features[idx] = homeFactors.PowerPlayPct
	idx++
	features[idx] = homeFactors.PenaltyKillPct
	idx++

	// Away team features
	features[idx] = awayFactors.WinPercentage
	idx++
	features[idx] = awayFactors.GoalsFor / 5.0
	idx++
	features[idx] = awayFactors.GoalsAgainst / 5.0
	idx++
	features[idx] = awayFactors.PowerPlayPct
	idx++
	features[idx] = awayFactors.PenaltyKillPct
	idx++

	// Advanced features
	features[idx] = homeFactors.MomentumScore
	idx++
	features[idx] = awayFactors.MomentumScore
	idx++
	features[idx] = homeFactors.GoalieAdvantage
	idx++
	features[idx] = awayFactors.GoalieAdvantage
	idx++

	// Pad remaining
	for idx < 65 {
		features[idx] = 0.5
		idx++
	}

	return features
}

// predictScore predicts the final score
func (rfm *RandomForestModel) predictScore(winProb float64, homeFactors, awayFactors *models.PredictionFactors) string {
	homeGoals := 3.0
	awayGoals := 2.5

	if winProb > 0.5 {
		homeGoals += (winProb - 0.5) * 2.0
		awayGoals -= (winProb - 0.5) * 1.5
	} else {
		homeGoals -= (0.5 - winProb) * 1.5
		awayGoals += (0.5 - winProb) * 2.0
	}

	homeScore := int(math.Round(homeGoals))
	awayScore := int(math.Round(awayGoals))

	if homeScore == awayScore {
		if winProb > 0.5 {
			homeScore++
		} else {
			awayScore++
		}
	}

	return fmt.Sprintf("%d-%d", homeScore, awayScore)
}

// calculateFeatureImportance calculates which features are most important
func (rfm *RandomForestModel) calculateFeatureImportance() {
	// Simplified: Count feature usage across all trees
	usage := make(map[string]int)

	for _, tree := range rfm.trees {
		rfm.countFeatureUsage(tree.Root, usage)
	}

	// Normalize to probabilities
	total := 0
	for _, count := range usage {
		total += count
	}

	if total > 0 {
		for feature, count := range usage {
			rfm.featureImportance[feature] = float64(count) / float64(total)
		}
	}
}

// countFeatureUsage recursively counts feature usage in a tree
func (rfm *RandomForestModel) countFeatureUsage(node *RFTreeNode, usage map[string]int) {
	if node == nil || node.IsLeaf {
		return
	}

	featureName := rfm.featureNames[node.FeatureIndex]
	usage[featureName]++

	rfm.countFeatureUsage(node.Left, usage)
	rfm.countFeatureUsage(node.Right, usage)
}

// GetName returns the model name
func (rfm *RandomForestModel) GetName() string {
	return "Random Forest"
}

// GetWeight returns the model weight in ensemble
func (rfm *RandomForestModel) GetWeight() float64 {
	rfm.mutex.RLock()
	defer rfm.mutex.RUnlock()
	return rfm.weight
}

// TrainOnGameResult trains the model on a completed game
func (rfm *RandomForestModel) TrainOnGameResult(game models.CompletedGame) error {
	// Random Forest needs batch training
	log.Printf("🌲 Random Forest: Received game result (batch training needed)")
	return nil
}

// RandomForestModelData represents serializable model data
type RandomForestModelData struct {
	Trees             []SerializedRFTree `json:"trees"`
	NumTrees          int                `json:"numTrees"`
	MaxDepth          int                `json:"maxDepth"`
	MinSamplesLeaf    int                `json:"minSamplesLeaf"`
	MaxFeatures       int                `json:"maxFeatures"`
	Weight            float64            `json:"weight"`
	Trained           bool               `json:"trained"`
	FeatureNames      []string           `json:"featureNames"`
	FeatureImportance map[string]float64 `json:"featureImportance"`
	LastUpdated       time.Time          `json:"lastUpdated"`
	Version           string             `json:"version"`
}

// SerializedRFTree represents a serializable decision tree
type SerializedRFTree struct {
	Root           *SerializedRFTreeNode `json:"root"`
	MaxDepth       int                   `json:"maxDepth"`
	MinSamplesLeaf int                   `json:"minSamplesLeaf"`
	MaxFeatures    int                   `json:"maxFeatures"`
}

// SerializedRFTreeNode represents a serializable tree node
type SerializedRFTreeNode struct {
	IsLeaf       bool                  `json:"isLeaf"`
	Prediction   float64               `json:"prediction"`
	FeatureIndex int                   `json:"featureIndex"`
	Threshold    float64               `json:"threshold"`
	Left         *SerializedRFTreeNode `json:"left,omitempty"`
	Right        *SerializedRFTreeNode `json:"right,omitempty"`
	SamplesCount int                   `json:"samplesCount"`
	ClassCounts  map[int]int           `json:"classCounts"`
}

func (rfm *RandomForestModel) saveModel() error {
	filePath := filepath.Join(rfm.dataDir, "random_forest.json")

	// Serialize trees
	serializedTrees := make([]SerializedRFTree, len(rfm.trees))
	for i, tree := range rfm.trees {
		serializedTrees[i] = SerializedRFTree{
			Root:           serializeRFTreeNode(tree.Root),
			MaxDepth:       tree.MaxDepth,
			MinSamplesLeaf: tree.MinSamplesLeaf,
			MaxFeatures:    tree.MaxFeatures,
		}
	}

	// Create model data
	modelData := RandomForestModelData{
		Trees:             serializedTrees,
		NumTrees:          rfm.numTrees,
		MaxDepth:          rfm.maxDepth,
		MinSamplesLeaf:    rfm.minSamplesLeaf,
		MaxFeatures:       rfm.maxFeatures,
		Weight:            rfm.weight,
		Trained:           rfm.trained,
		FeatureNames:      rfm.featureNames,
		FeatureImportance: rfm.featureImportance,
		LastUpdated:       time.Now(),
		Version:           "1.0",
	}

	data, err := json.MarshalIndent(modelData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling random forest model: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing random forest model: %w", err)
	}

	log.Printf("💾 Random Forest model saved: %d trees, trained=%v", len(rfm.trees), rfm.trained)
	return nil
}

// serializeRFTreeNode recursively serializes a tree node
func serializeRFTreeNode(node *RFTreeNode) *SerializedRFTreeNode {
	if node == nil {
		return nil
	}

	serialized := &SerializedRFTreeNode{
		IsLeaf:       node.IsLeaf,
		Prediction:   node.Prediction,
		FeatureIndex: node.FeatureIndex,
		Threshold:    node.Threshold,
		SamplesCount: node.SamplesCount,
		ClassCounts:  node.ClassCounts,
	}

	if node.Left != nil {
		serialized.Left = serializeRFTreeNode(node.Left)
	}
	if node.Right != nil {
		serialized.Right = serializeRFTreeNode(node.Right)
	}

	return serialized
}

func (rfm *RandomForestModel) loadModel() error {
	filePath := filepath.Join(rfm.dataDir, "random_forest.json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("🌲 No saved Random Forest model found, starting fresh")
		return nil // Not an error
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading random forest model: %w", err)
	}

	var modelData RandomForestModelData
	err = json.Unmarshal(data, &modelData)
	if err != nil {
		return fmt.Errorf("error unmarshaling random forest model: %w", err)
	}

	// Deserialize trees
	rfm.trees = make([]*RFTree, len(modelData.Trees))
	for i, serializedTree := range modelData.Trees {
		rfm.trees[i] = &RFTree{
			Root:           deserializeRFTreeNode(serializedTree.Root),
			MaxDepth:       serializedTree.MaxDepth,
			MinSamplesLeaf: serializedTree.MinSamplesLeaf,
			MaxFeatures:    serializedTree.MaxFeatures,
		}
	}

	// Load other fields
	rfm.numTrees = modelData.NumTrees
	rfm.maxDepth = modelData.MaxDepth
	rfm.minSamplesLeaf = modelData.MinSamplesLeaf
	rfm.maxFeatures = modelData.MaxFeatures
	rfm.weight = modelData.Weight
	rfm.trained = modelData.Trained
	rfm.featureNames = modelData.FeatureNames
	rfm.featureImportance = modelData.FeatureImportance

	log.Printf("🌲 Random Forest model loaded: %d trees, trained=%v", len(rfm.trees), rfm.trained)
	log.Printf("   Last updated: %s", modelData.LastUpdated.Format("2006-01-02 15:04:05"))
	return nil
}

// deserializeRFTreeNode recursively deserializes a tree node
func deserializeRFTreeNode(serialized *SerializedRFTreeNode) *RFTreeNode {
	if serialized == nil {
		return nil
	}

	node := &RFTreeNode{
		IsLeaf:       serialized.IsLeaf,
		Prediction:   serialized.Prediction,
		FeatureIndex: serialized.FeatureIndex,
		Threshold:    serialized.Threshold,
		SamplesCount: serialized.SamplesCount,
		ClassCounts:  serialized.ClassCounts,
	}

	if serialized.Left != nil {
		node.Left = deserializeRFTreeNode(serialized.Left)
	}
	if serialized.Right != nil {
		node.Right = deserializeRFTreeNode(serialized.Right)
	}

	return node
}
