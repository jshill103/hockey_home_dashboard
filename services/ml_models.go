package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// NeuralNetworkModel implements a simple neural network for game prediction
type NeuralNetworkModel struct {
	weights      [][]float64 // Layer weights
	biases       [][]float64 // Layer biases
	learningRate float64
	layers       []int   // Architecture: [input, hidden1, hidden2, output]
	weight       float64 // Model weight in ensemble
	lastUpdated  time.Time
	mutex        sync.RWMutex // Thread safety for concurrent access
	dataDir      string       // Directory for persistence
}

// NewNeuralNetworkModel creates a new neural network prediction model
func NewNeuralNetworkModel() *NeuralNetworkModel {
	layers := []int{156, 64, 32, 3} // Input features (140 Phase 1 + 16 Phase 2), hidden layers, output (win/loss/ot)

	model := &NeuralNetworkModel{
		layers:       layers,
		learningRate: 0.001,
		weight:       0.05, // 5% weight in ensemble (starting conservative, will increase with training)
		lastUpdated:  time.Now(),
		dataDir:      "data/models",
	}

	// Create data directory if it doesn't exist
	os.MkdirAll(model.dataDir, 0755)

	// Try to load existing weights
	if err := model.loadWeights(); err != nil {
		// No existing weights, initialize new network
		log.Printf("🧠 Initializing new Neural Network (no saved weights found)")
		model.initializeNetwork()
	} else {
		log.Printf("🧠 Neural Network weights loaded from disk")
		log.Printf("   Last updated: %s", model.lastUpdated.Format("2006-01-02 15:04:05"))
	}

	return model
}

// XGBoostModel implements gradient boosting for hockey predictions
type XGBoostModel struct {
	trees             []DecisionTree
	learningRate      float64
	maxDepth          int
	numTrees          int
	weight            float64
	featureImportance map[string]float64
	lastUpdated       time.Time
}

// DecisionTree represents a single decision tree in the ensemble
type DecisionTree struct {
	Feature   string
	Threshold float64
	Left      *DecisionTree
	Right     *DecisionTree
	Value     float64
	IsLeaf    bool
}

// LSTM Model is now in lstm_model.go

// Random Forest Model is now in random_forest.go

// Predict implements neural network prediction
func (nn *NeuralNetworkModel) Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error) {
	nn.mutex.RLock()
	defer nn.mutex.RUnlock()

	start := time.Now()

	// Extract features from factors
	features := nn.extractFeatures(homeFactors, awayFactors)

	// Forward pass through network
	output := nn.forwardPass(features)

	// Convert output to win probability and score
	winProb := nn.sigmoid(output[0])
	confidence := nn.calculateConfidence(output)
	predictedScore := nn.outputToScore(output, homeFactors, awayFactors)

	return &models.ModelResult{
		ModelName:      "Neural Network",
		WinProbability: winProb,
		Confidence:     confidence,
		PredictedScore: predictedScore,
		Weight:         nn.weight,
		ProcessingTime: time.Since(start).Milliseconds(),
	}, nil
}

// extractFeatures converts prediction factors to neural network input
func (nn *NeuralNetworkModel) extractFeatures(home, away *models.PredictionFactors) []float64 {
	features := make([]float64, 156) // 156 input features (140 Phase 1 + 16 Phase 2)

	// Basic team stats
	features[0] = home.WinPercentage
	features[1] = away.WinPercentage
	features[2] = home.GoalsFor / 82.0
	features[3] = home.GoalsAgainst / 82.0
	features[4] = away.GoalsFor / 82.0
	features[5] = away.GoalsAgainst / 82.0

	// Advanced analytics
	features[6] = home.AdvancedStats.XGDifferential
	features[7] = away.AdvancedStats.XGDifferential
	features[8] = home.AdvancedStats.CorsiForPct / 100.0
	features[9] = away.AdvancedStats.CorsiForPct / 100.0

	// Situational factors
	features[10] = home.TravelFatigue.FatigueScore
	features[11] = away.TravelFatigue.FatigueScore
	features[12] = home.InjuryImpact.ImpactScore / 50.0
	features[13] = away.InjuryImpact.ImpactScore / 50.0

	// Weather impact
	features[14] = home.WeatherAnalysis.OverallImpact
	features[15] = away.WeatherAnalysis.OverallImpact

	// Momentum and form
	features[20] = home.MomentumFactors.MomentumScore
	features[21] = away.MomentumFactors.MomentumScore
	features[22] = home.RecentForm
	features[23] = away.RecentForm

	// Special teams
	features[24] = home.PowerPlayPct / 100.0
	features[25] = away.PowerPlayPct / 100.0
	features[26] = home.PenaltyKillPct / 100.0
	features[27] = away.PenaltyKillPct / 100.0

	// Rest and schedule
	features[28] = float64(home.RestDays) / 7.0
	features[29] = float64(away.RestDays) / 7.0
	features[30] = home.BackToBackPenalty
	features[31] = away.BackToBackPenalty

	// Head-to-head and historical
	features[32] = home.HeadToHead
	features[33] = away.HeadToHead

	// Advanced goaltending
	features[34] = home.AdvancedStats.GoalieSvPctOverall
	features[35] = away.AdvancedStats.GoalieSvPctOverall
	features[36] = home.AdvancedStats.SavesAboveExpected
	features[37] = away.AdvancedStats.SavesAboveExpected

	// Game state performance
	features[38] = home.AdvancedStats.LeadingPerformance
	features[39] = away.AdvancedStats.LeadingPerformance
	features[40] = home.AdvancedStats.TrailingPerformance
	features[41] = away.AdvancedStats.TrailingPerformance

	// Zone play and transitions
	features[42] = home.AdvancedStats.OffensiveZoneTime / 100.0
	features[43] = away.AdvancedStats.OffensiveZoneTime / 100.0
	features[44] = home.AdvancedStats.ControlledEntries / 100.0
	features[45] = away.AdvancedStats.ControlledEntries / 100.0

	// Overall team ratings
	features[46] = home.AdvancedStats.OverallRating / 100.0
	features[47] = away.AdvancedStats.OverallRating / 100.0

	// Home ice advantage indicator
	features[48] = 1.0 // Home team
	features[49] = 0.0 // Away team

	// ============================================================================
	// PHASE 4: NEW FEATURES (50-64)
	// ============================================================================

	// Goalie Intelligence (50-53)
	features[50] = home.GoalieAdvantage
	features[51] = away.GoalieAdvantage
	features[52] = home.GoalieSavePctDiff
	features[53] = home.GoalieRecentFormDiff

	// Betting Market Data (54-57)
	features[54] = home.MarketConsensus
	features[55] = away.MarketConsensus
	features[56] = home.SharpMoneyIndicator
	features[57] = home.MarketLineMovement

	// Schedule Context (58-64)
	features[58] = home.TravelDistance / 3000.0 // Normalize by max expected distance
	features[59] = away.TravelDistance / 3000.0
	features[60] = home.BackToBackIndicator
	features[61] = away.BackToBackIndicator
	features[62] = home.ScheduleDensity / 7.0 // Normalize by max games per week
	features[63] = away.ScheduleDensity / 7.0
	features[64] = home.RestAdvantage / 5.0 // Normalize by max rest advantage

	// ============================================================================
	// PLAYER INTELLIGENCE: TOP 10 TRACKING (65-74)
	// ============================================================================

	// Star Power & Top 3 Scoring (65-68)
	features[65] = home.StarPowerRating // 0-1 scale (star player quality)
	features[66] = away.StarPowerRating
	features[67] = home.Top3CombinedPPG / 4.0 // Normalize by ~max (4.0 PPG is elite)
	features[68] = away.Top3CombinedPPG / 4.0

	// Recent Form - Top 3 and Depth (69-72)
	features[69] = home.TopScorerForm / 10.0 // 0-10 scale → 0-1
	features[70] = away.TopScorerForm / 10.0
	features[71] = home.DepthForm / 10.0 // Depth scorers (4-10) form
	features[72] = away.DepthForm / 10.0

	// Player Advantages (73-74)
	features[73] = home.StarPowerEdge // Already normalized -1 to +1
	features[74] = home.DepthEdge     // Already normalized -1 to +1

	// ============================================================================
	// GOALIE FEATURES (75-80) - 6 new features
	// ============================================================================

	// Goalie Save Percentage Differential (75)
	// Positive = home goalie advantage, negative = away goalie advantage
	features[75] = home.GoalieSavePctDiff // Already calculated differential

	// Goalie Recent Form Differential (76)
	// Based on last 5 starts performance
	features[76] = home.GoalieRecentFormDiff // Already calculated differential

	// Goalie Workload/Fatigue Differential (77)
	// Positive = away goalie more fatigued, negative = home goalie more fatigued
	features[77] = home.GoalieFatigueDiff // Already calculated differential

	// Overall Goalie Advantage (78)
	// Combined goalie impact on win probability (-0.15 to +0.15)
	features[78] = home.GoalieAdvantage // From GoalieIntelligenceService

	// Advanced Goalie Stats - Home (79)
	// Saves Above Expected from AdvancedStats
	features[79] = home.AdvancedStats.SavesAboveExpected

	// Advanced Goalie Stats - Away (80)
	// Saves Above Expected from AdvancedStats
	features[80] = away.AdvancedStats.SavesAboveExpected

	// ============================================================================
	// PLAY-BY-PLAY ANALYTICS: EXPECTED GOALS & SHOT QUALITY (81-92) - 12 features
	// ============================================================================

	// Expected Goals (81-84)
	features[81] = home.ExpectedGoalsFor / 4.0 // Normalize by ~max (4 xGF/game is elite)
	features[82] = away.ExpectedGoalsFor / 4.0
	features[83] = home.ExpectedGoalsAgainst / 4.0 // Normalize by ~max (4 xGA/game is poor)
	features[84] = away.ExpectedGoalsAgainst / 4.0

	// xG Differential & Shot Quality (85-88)
	features[85] = home.XGDifferential / 2.0 // Normalize by typical range (-2 to +2)
	features[86] = away.XGDifferential / 2.0
	features[87] = home.XGPerShot / 0.15 // Normalize by ~max (0.15 xG/shot is elite)
	features[88] = away.XGPerShot / 0.15

	// Shot Quality Metrics (89-90)
	features[89] = home.DangerousShotsPerGame / 15.0 // Normalize by ~max (15 high-danger/game)
	features[90] = away.DangerousShotsPerGame / 15.0

	// Advanced Possession Metrics (91-92)
	features[91] = home.CorsiForPct // Already 0-1 (percentage)
	features[92] = away.CorsiForPct

	// ============================================================================
	// SHIFT ANALYSIS: LINE CHEMISTRY & COACHING TENDENCIES (93-100) - 8 features
	// ============================================================================

	// Line Chemistry & Usage (93-96)
	features[93] = home.AvgShiftLength / 60.0 // Normalize by ~max (60 seconds)
	features[94] = away.AvgShiftLength / 60.0
	features[95] = home.LineConsistency // Already 0-1
	features[96] = away.LineConsistency

	// Coaching Tendencies (97-100)
	features[97] = home.ShortBench // Already 0-1
	features[98] = away.ShortBench
	features[99] = home.FatigueIndicator // Already 0-1
	features[100] = away.FatigueIndicator

	// ============================================================================
	// LANDING PAGE ANALYTICS: ENHANCED PHYSICAL PLAY & ZONE CONTROL (101-104) - 4 features
	// ============================================================================

	// Zone Control & Time on Attack (101-102)
	features[101] = home.TimeOnAttack / 30.0 // Normalize by ~max (30 min attack)
	features[102] = away.TimeOnAttack / 30.0

	// Zone Control & Special Teams (103-104)
	features[103] = home.ZoneControlRatio // Already 0-1
	features[104] = away.ZoneControlRatio

	// ============================================================================
	// GAME SUMMARY ANALYTICS: ENHANCED GAME CONTEXT (105-110) - 6 features
	// ============================================================================

	// Enhanced Shot Quality & Discipline (105-106)
	features[105] = home.ShotQualityIndex // Already 0-1
	features[106] = away.ShotQualityIndex

	// Enhanced Special Teams Context (107-108)
	features[107] = home.PowerPlayTime / 10.0 // Normalize by ~max (10 min PP/game)
	features[108] = away.PowerPlayTime / 10.0

	// Enhanced Zone Control Context (109-110)
	features[109] = home.OffensiveZoneTime / 30.0 // Normalize by ~max (30 min/game)
	features[110] = away.OffensiveZoneTime / 30.0

	// ============================================================================
	// PHASE 1 EXPANSION: ROLLING STATISTICS (111-130) - 20 features
	// ============================================================================

	// Hot/Cold Streak Detection (111-116)
	if home.IsHot {
		features[111] = 1.0
	}
	if away.IsHot {
		features[112] = 1.0
	}
	if home.IsCold {
		features[113] = 1.0
	}
	if away.IsCold {
		features[114] = 1.0
	}
	if home.IsStreaking {
		features[115] = 1.0
	}
	if away.IsStreaking {
		features[116] = 1.0
	}

	// Time-Weighted Performance (117-122)
	features[117] = home.WeightedWinPct
	features[118] = away.WeightedWinPct
	features[119] = home.WeightedGoalsFor / 5.0 // Normalize by ~max (5 goals/game)
	features[120] = away.WeightedGoalsFor / 5.0
	features[121] = home.WeightedGoalsAgainst / 5.0
	features[122] = away.WeightedGoalsAgainst / 5.0

	// Quality of Competition (123-126)
	features[123] = home.VsPlayoffTeamsPct
	features[124] = away.VsPlayoffTeamsPct
	features[125] = home.ClutchPerformance
	features[126] = away.ClutchPerformance

	// Recent Points & Goal Differential (127-130)
	features[127] = float64(home.Last5GamesPoints) / 10.0 // Normalize by max (10 points)
	features[128] = float64(away.Last5GamesPoints) / 10.0
	features[129] = float64(home.GoalDifferential5) / 20.0 // Normalize by ~max (+/-20 goals)
	features[130] = float64(away.GoalDifferential5) / 20.0

	// ============================================================================
	// PHASE 1 EXPANSION: MATCHUP CONTEXT (131-139) - 9 features
	// ============================================================================

	// Rivalry & Division Context (131-135)
	if home.IsRivalryGame {
		features[131] = 1.0
	}
	features[132] = home.RivalryIntensity
	if home.IsDivisionGame {
		features[133] = 1.0
	}
	if home.IsPlayoffRematch {
		features[134] = 1.0
	}
	features[135] = home.HeadToHeadAdvantage

	// Matchup History (136-139)
	features[136] = home.RecentMatchupTrend
	features[137] = home.VenueSpecificRecord
	features[138] = float64(home.DaysSinceLastMeeting) / 365.0 // Normalize by max (1 year)
	features[139] = home.AverageGoalDiff / 5.0                 // Normalize by ~max (+/-5 goals)

	// ============================================================================
	// PHASE 2 EXPANSION: PLAYER/GOALIE EDGES (140-142) - 3 features
	// ============================================================================

	// Relative Advantages (140-142)
	features[140] = home.StarPowerEdge     // -1 to +1 star power advantage
	features[141] = home.DepthEdge         // -1 to +1 depth advantage
	features[142] = home.GoalieFatigueDiff // Goalie workload differential

	// ============================================================================
	// PHASE 2 EXPANSION: SITUATIONAL CONTEXT (143-147) - 5 features
	// ============================================================================

	// Motivation & Context (143-145)
	features[143] = home.TrapGameFactor    // 0-1 trap game likelihood
	features[144] = home.PlayoffImportance // 0-1 playoff stakes
	features[145] = home.TransitionEfficiency

	// Special Teams & Discipline (146-147)
	features[146] = home.SpecialTeamsIndex // Combined PP% + PK%
	features[147] = home.DisciplineIndex   // Penalty discipline

	// ============================================================================
	// PHASE 2 EXPANSION: WEATHER & ADVANCED (148-155) - 8 features
	// ============================================================================

	// Weather Impacts (148-153)
	features[148] = home.WeatherAnalysis.TravelImpact.OverallImpact / 5.0 // Normalize by max impact
	features[149] = away.WeatherAnalysis.TravelImpact.OverallImpact / 5.0
	features[150] = home.WeatherAnalysis.OverallImpact / 10.0 // Game day impact
	features[151] = away.WeatherAnalysis.OverallImpact / 10.0
	if home.WeatherAnalysis.IsOutdoorGame {
		features[152] = 1.0
	}
	if away.WeatherAnalysis.IsOutdoorGame {
		features[153] = 1.0
	}

	// Advanced Features (154-155)
	features[154] = home.MarketConfidenceVal      // Market confidence
	features[155] = home.DefensiveZoneTime / 30.0 // Defensive zone time

	return features
}

// initializeNetwork initializes weights and biases with Xavier initialization
func (nn *NeuralNetworkModel) initializeNetwork() {
	numLayers := len(nn.layers)
	nn.weights = make([][]float64, numLayers-1)
	nn.biases = make([][]float64, numLayers-1)

	for i := 0; i < numLayers-1; i++ {
		inputSize := nn.layers[i]
		outputSize := nn.layers[i+1]

		// Xavier initialization
		limit := math.Sqrt(6.0 / float64(inputSize+outputSize))

		nn.weights[i] = make([]float64, inputSize*outputSize)
		nn.biases[i] = make([]float64, outputSize)

		for j := range nn.weights[i] {
			nn.weights[i][j] = (rand.Float64()*2 - 1) * limit
		}

		for j := range nn.biases[i] {
			nn.biases[i][j] = 0.0
		}
	}
}

// forwardPass performs forward propagation through the network
func (nn *NeuralNetworkModel) forwardPass(input []float64) []float64 {
	current := input

	for layer := 0; layer < len(nn.weights); layer++ {
		next := make([]float64, nn.layers[layer+1])

		for j := 0; j < nn.layers[layer+1]; j++ {
			sum := nn.biases[layer][j]
			for i := 0; i < nn.layers[layer]; i++ {
				sum += current[i] * nn.weights[layer][i*nn.layers[layer+1]+j]
			}

			// Apply activation function (ReLU for hidden layers, sigmoid for output)
			if layer == len(nn.weights)-1 {
				next[j] = nn.sigmoid(sum)
			} else {
				next[j] = nn.relu(sum)
			}
		}

		current = next
	}

	return current
}

// Helper activation functions
func (nn *NeuralNetworkModel) sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func (nn *NeuralNetworkModel) sigmoidDerivative(x float64) float64 {
	s := nn.sigmoid(x)
	return s * (1.0 - s)
}

func (nn *NeuralNetworkModel) relu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

func (nn *NeuralNetworkModel) reluDerivative(x float64) float64 {
	if x > 0 {
		return 1.0
	}
	return 0
}

func (nn *NeuralNetworkModel) calculateConfidence(output []float64) float64 {
	// Calculate confidence based on output certainty
	maxVal := 0.0
	for _, val := range output {
		if val > maxVal {
			maxVal = val
		}
	}

	// Higher max value = higher confidence
	return 0.6 + (maxVal * 0.35) // Scale to 0.6-0.95 range
}

func (nn *NeuralNetworkModel) outputToScore(output []float64, home, away *models.PredictionFactors) string {
	// Convert neural network output to realistic hockey score
	// output[0] = home win prob, output[1] = home goals, output[2] = away goals

	homeGoals := int(math.Round(output[1] * 8)) // Scale to 0-8 goals
	awayGoals := int(math.Round(output[2] * 8))

	// Ensure realistic bounds
	if homeGoals < 0 {
		homeGoals = 0
	}
	if awayGoals < 0 {
		awayGoals = 0
	}
	if homeGoals > 8 {
		homeGoals = 8
	}
	if awayGoals > 8 {
		awayGoals = 8
	}

	// Ensure at least one goal total
	if homeGoals == 0 && awayGoals == 0 {
		homeGoals = 1
	}

	return fmt.Sprintf("%d-%d", homeGoals, awayGoals)
}

func (nn *NeuralNetworkModel) GetName() string {
	return "Neural Network"
}

func (nn *NeuralNetworkModel) GetWeight() float64 {
	return nn.weight
}

// TrainOnGameResult updates the neural network with actual game results
func (nn *NeuralNetworkModel) TrainOnGameResult(gameResult *models.GameResult, homeFactors, awayFactors *models.PredictionFactors) error {
	nn.mutex.Lock()
	defer nn.mutex.Unlock()

	// Extract features
	features := nn.extractFeatures(homeFactors, awayFactors)

	// Create target output
	target := make([]float64, 3)
	if gameResult.HomeScore > gameResult.AwayScore {
		target[0] = 1.0 // Home win
	} else {
		target[0] = 0.0 // Away win
	}
	target[1] = float64(gameResult.HomeScore) / 8.0 // Normalized home goals
	target[2] = float64(gameResult.AwayScore) / 8.0 // Normalized away goals

	// Perform backpropagation (simplified)
	nn.backpropagate(features, target)

	nn.lastUpdated = time.Now()

	// Auto-save weights after training
	// Release lock before saving to avoid deadlock
	nn.mutex.Unlock()
	if err := nn.saveWeights(); err != nil {
		log.Printf("⚠️ Failed to save Neural Network weights: %v", err)
	}
	nn.mutex.Lock() // Re-acquire for defer unlock

	return nil
}

// forwardPassWithActivations performs forward pass and stores all activations
func (nn *NeuralNetworkModel) forwardPassWithActivations(input []float64) ([][]float64, [][]float64) {
	// Store activations (post-activation) and pre-activations (z values)
	numLayers := len(nn.layers)
	activations := make([][]float64, numLayers)
	preActivations := make([][]float64, numLayers)

	// Input layer
	activations[0] = make([]float64, len(input))
	copy(activations[0], input)
	preActivations[0] = nil // No pre-activation for input layer

	// Forward through each layer
	for layer := 0; layer < numLayers-1; layer++ {
		inputSize := nn.layers[layer]
		outputSize := nn.layers[layer+1]

		preActivations[layer+1] = make([]float64, outputSize)
		activations[layer+1] = make([]float64, outputSize)

		for j := 0; j < outputSize; j++ {
			// Calculate weighted sum (z = W*a + b)
			z := nn.biases[layer][j]
			for i := 0; i < inputSize; i++ {
				z += activations[layer][i] * nn.weights[layer][i*outputSize+j]
			}

			preActivations[layer+1][j] = z

			// Apply activation function
			if layer == numLayers-2 { // Output layer
				activations[layer+1][j] = nn.sigmoid(z)
			} else { // Hidden layers
				activations[layer+1][j] = nn.relu(z)
			}
		}
	}

	return activations, preActivations
}

// backpropagate performs proper backpropagation with gradient descent
func (nn *NeuralNetworkModel) backpropagate(input, target []float64) {
	// Forward pass with activation storage
	activations, preActivations := nn.forwardPassWithActivations(input)
	numLayers := len(nn.layers)

	// Initialize error arrays for each layer
	errors := make([][]float64, numLayers)
	for i := range errors {
		errors[i] = make([]float64, nn.layers[i])
	}

	// Calculate output layer error (δ = (a - y) * σ'(z))
	outputLayer := numLayers - 1
	for j := 0; j < nn.layers[outputLayer]; j++ {
		// Mean Squared Error derivative: (predicted - actual)
		outputError := activations[outputLayer][j] - target[j]

		// Multiply by activation derivative
		activationDeriv := nn.sigmoidDerivative(preActivations[outputLayer][j])

		errors[outputLayer][j] = outputError * activationDeriv
	}

	// Backpropagate errors through hidden layers
	for layer := numLayers - 2; layer > 0; layer-- {
		for j := 0; j < nn.layers[layer]; j++ {
			error := 0.0

			// Sum weighted errors from next layer
			nextLayerSize := nn.layers[layer+1]
			for k := 0; k < nextLayerSize; k++ {
				weightIndex := j*nextLayerSize + k
				error += nn.weights[layer][weightIndex] * errors[layer+1][k]
			}

			// Multiply by activation derivative (ReLU for hidden layers)
			activationDeriv := nn.reluDerivative(preActivations[layer][j])
			errors[layer][j] = error * activationDeriv
		}
	}

	// Update weights and biases using computed errors
	for layer := 0; layer < numLayers-1; layer++ {
		inputSize := nn.layers[layer]
		outputSize := nn.layers[layer+1]

		// Update weights: W = W - α * (a^(l) * δ^(l+1))
		for i := 0; i < inputSize; i++ {
			for j := 0; j < outputSize; j++ {
				weightIndex := i*outputSize + j

				// Gradient = activation from previous layer * error of current neuron
				gradient := activations[layer][i] * errors[layer+1][j]

				// Update weight with learning rate
				nn.weights[layer][weightIndex] -= nn.learningRate * gradient
			}
		}

		// Update biases: b = b - α * δ
		for j := 0; j < outputSize; j++ {
			nn.biases[layer][j] -= nn.learningRate * errors[layer+1][j]
		}
	}
}

// ============================================================================
// NEURAL NETWORK PERSISTENCE
// ============================================================================

// NeuralNetworkData represents the serializable state of the neural network
type NeuralNetworkData struct {
	Weights      [][][]float64 `json:"weights"` // 3D array for all layer weights
	Biases       [][]float64   `json:"biases"`
	Layers       []int         `json:"layers"`
	LearningRate float64       `json:"learningRate"`
	Weight       float64       `json:"weight"`
	LastUpdated  time.Time     `json:"lastUpdated"`
	Version      string        `json:"version"`
	TrainingInfo struct {
		TotalGames int    `json:"totalGames"`
		Notes      string `json:"notes"`
	} `json:"trainingInfo"`
}

// saveWeights saves the neural network weights to disk
func (nn *NeuralNetworkModel) saveWeights() error {
	nn.mutex.RLock()
	defer nn.mutex.RUnlock()

	filePath := filepath.Join(nn.dataDir, "neural_network.json")

	// Convert 2D weight slices to 3D for JSON serialization
	// weights[layer][neuron] needs to become weights[layer][neuron][connection]
	weights3D := make([][][]float64, len(nn.weights))
	for layer := range nn.weights {
		weights3D[layer] = make([][]float64, len(nn.weights[layer]))
		for neuron := range nn.weights[layer] {
			// Each weight is a single value, wrap in slice
			weights3D[layer][neuron] = []float64{nn.weights[layer][neuron]}
		}
	}

	data := NeuralNetworkData{
		Weights:      weights3D,
		Biases:       nn.biases,
		Layers:       nn.layers,
		LearningRate: nn.learningRate,
		Weight:       nn.weight,
		LastUpdated:  nn.lastUpdated,
		Version:      "1.0",
	}
	data.TrainingInfo.Notes = "Neural Network for NHL game prediction"

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling neural network data: %v", err)
	}

	err = ioutil.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing neural network file: %v", err)
	}

	return nil
}

// loadWeights loads the neural network weights from disk
func (nn *NeuralNetworkModel) loadWeights() error {
	filePath := filepath.Join(nn.dataDir, "neural_network.json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("neural network weights file not found")
	}

	// Read file
	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading neural network file: %v", err)
	}

	// Unmarshal data
	var data NeuralNetworkData
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return fmt.Errorf("error unmarshaling neural network data: %v", err)
	}

	// Validate architecture matches
	if len(data.Layers) != len(nn.layers) {
		return fmt.Errorf("loaded architecture doesn't match: expected %v, got %v", nn.layers, data.Layers)
	}
	for i := range data.Layers {
		if data.Layers[i] != nn.layers[i] {
			return fmt.Errorf("layer %d size mismatch: expected %d, got %d", i, nn.layers[i], data.Layers[i])
		}
	}

	// Convert 3D weights back to 2D
	nn.weights = make([][]float64, len(data.Weights))
	for layer := range data.Weights {
		nn.weights[layer] = make([]float64, len(data.Weights[layer]))
		for neuron := range data.Weights[layer] {
			if len(data.Weights[layer][neuron]) > 0 {
				nn.weights[layer][neuron] = data.Weights[layer][neuron][0]
			}
		}
	}

	// Load other fields
	nn.biases = data.Biases
	nn.learningRate = data.LearningRate
	nn.weight = data.Weight
	nn.lastUpdated = data.LastUpdated

	return nil
}

// GetLastUpdate returns when the neural network was last updated
func (nn *NeuralNetworkModel) GetLastUpdate() time.Time {
	nn.mutex.RLock()
	defer nn.mutex.RUnlock()
	return nn.lastUpdated
}

// ============================================================================
// OTHER MODEL IMPLEMENTATIONS (placeholder getters)
// ============================================================================

// GetName, GetWeight implementations for other models...
func (xgb *XGBoostModel) GetName() string    { return "XGBoost" }
func (xgb *XGBoostModel) GetWeight() float64 { return xgb.weight }

// LSTM methods are now in lstm_model.go
// Random Forest methods are now in random_forest.go
