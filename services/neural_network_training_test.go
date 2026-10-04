package services

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaredshillingburg/go_uhc/models"
)

func newTrainableNetwork(t *testing.T) *NeuralNetworkModel {
	t.Helper()
	nn := &NeuralNetworkModel{
		layers:       []int{182, 64, 32, 3},
		learningRate: 0.01,
		weight:       0.1,
		dataDir:      t.TempDir(),
	}
	nn.initializeNetwork()
	return nn
}

// separableGames builds a batch where the stronger side always wins, so a
// network that learns anything at all should beat a coin flip on it.
func separableGames(n int) []NeuralTrainingSample {
	samples := make([]NeuralTrainingSample, 0, n)
	for i := 0; i < n; i++ {
		homeStrong := i%2 == 0

		homeRate, awayRate := 2.2, 4.0
		homeScore, awayScore := 2, 5
		if homeStrong {
			homeRate, awayRate = 4.0, 2.2
			homeScore, awayScore = 5, 2
		}

		home := &models.PredictionFactors{TeamCode: "HOM", GoalsFor: homeRate, GoalsAgainst: awayRate, WinPercentage: 0.7}
		away := &models.PredictionFactors{TeamCode: "AWY", GoalsFor: awayRate, GoalsAgainst: homeRate, WinPercentage: 0.3}
		if !homeStrong {
			home.WinPercentage, away.WinPercentage = 0.3, 0.7
		}

		samples = append(samples, NeuralTrainingSample{
			Result:      &models.GameResult{HomeScore: homeScore, AwayScore: awayScore},
			HomeFactors: home,
			AwayFactors: away,
		})
	}
	return samples
}

func copyWeightLayers(nn *NeuralNetworkModel) [][]float64 {
	out := make([][]float64, len(nn.weights))
	for i, layer := range nn.weights {
		out[i] = append([]float64(nil), layer...)
	}
	return out
}

func maxAbsDelta(before, after []float64) float64 {
	worst := 0.0
	for i := range before {
		if d := math.Abs(after[i] - before[i]); d > worst {
			worst = d
		}
	}
	return worst
}

// The defect: in the saved production weights, layers 0, 1 and 2 sat at
// exactly their Xavier initialisation. Only the output layer had moved, and
// only by a few percent, because a saturated sigmoid under squared-error loss
// shrank every gradient by a factor of about 145.
func TestEveryLayerTrains(t *testing.T) {
	nn := newTrainableNetwork(t)
	before := copyWeightLayers(nn)

	if err := nn.TrainOnGameResults(separableGames(40)); err != nil {
		t.Fatalf("TrainOnGameResults: %v", err)
	}

	for layer := range nn.weights {
		moved := maxAbsDelta(before[layer], nn.weights[layer])
		if moved < 1e-6 {
			t.Errorf("layer %d did not move (max weight change %.3g)", layer, moved)
		}
	}
}

// Training has to actually improve the prediction, not merely perturb it.
func TestTrainingImprovesSeparableAccuracy(t *testing.T) {
	nn := newTrainableNetwork(t)
	games := separableGames(60)

	accuracy := func() float64 {
		correct := 0
		for _, s := range games {
			result, err := nn.Predict(s.HomeFactors, s.AwayFactors)
			if err != nil {
				t.Fatalf("Predict: %v", err)
			}
			homeWon := s.Result.HomeScore > s.Result.AwayScore
			if (result.WinProbability >= 0.5) == homeWon {
				correct++
			}
		}
		return float64(correct) / float64(len(games))
	}

	if err := nn.TrainOnGameResults(games); err != nil {
		t.Fatalf("TrainOnGameResults: %v", err)
	}
	after := accuracy()

	// The two classes are perfectly separable, so anything that learns should
	// be well clear of a coin flip.
	if after < 0.9 {
		t.Errorf("accuracy after training = %.2f on perfectly separable data, want >= 0.90", after)
	}
}

// Cross-entropy on the classification output means the delta is exactly
// (a - y), with no sigmoid derivative to shrink it. Under the old squared
// error a saturated output produced a delta around 145 times smaller.
func TestClassificationGradientSurvivesSaturation(t *testing.T) {
	nn := newTrainableNetwork(t)

	// Drive the win output hard toward 1 while the truth is an away win.
	home := &models.PredictionFactors{TeamCode: "HOM", GoalsFor: 4.0, GoalsAgainst: 2.0}
	away := &models.PredictionFactors{TeamCode: "AWY", GoalsFor: 2.0, GoalsAgainst: 4.0}
	features := nn.extractFeatures(home, away)

	// Push the output layer into saturation.
	for j := range nn.biases[len(nn.biases)-1] {
		nn.biases[len(nn.biases)-1][j] = 6.0
	}

	saturated := nn.forwardPass(features)[0]
	if saturated < 0.99 {
		t.Fatalf("setup failed: output %.4f is not saturated", saturated)
	}

	target := []float64{0.0, 2.0 / 8.0, 5.0 / 8.0}
	beforeBias := nn.biases[len(nn.biases)-1][0]
	nn.backpropagate(features, target)
	moved := math.Abs(nn.biases[len(nn.biases)-1][0] - beforeBias)

	// Cross-entropy delta is (a - y) ~= 1.0, so the bias should move by
	// roughly the learning rate. The squared-error version would move it by
	// about lr * 0.007.
	wantAtLeast := nn.learningRate * 0.5
	if moved < wantAtLeast {
		t.Errorf("saturated output moved the bias by %.3g, want at least %.3g; "+
			"the sigmoid derivative is still attenuating the gradient", moved, wantAtLeast)
	}
}

func TestFeaturesAreHeldInRange(t *testing.T) {
	nn := newTrainableNetwork(t)

	// Values far outside any sane scale, of the sort the unnormalised
	// assignments can produce.
	home := &models.PredictionFactors{TeamCode: "HOM", GoalsFor: 5000, GoalsAgainst: -4000}
	away := &models.PredictionFactors{TeamCode: "AWY", GoalsFor: math.NaN(), GoalsAgainst: 9999}

	for i, v := range nn.extractFeatures(home, away) {
		if math.IsNaN(v) {
			t.Fatalf("feature %d is NaN", i)
		}
		if v > featureBound || v < -featureBound {
			t.Errorf("feature %d = %v, outside ±%v", i, v, featureBound)
		}
	}
}

// A clamped input must leave the freshly initialised network near a coin
// flip rather than pinned against a bound, which is what saturated it.
func TestFreshNetworkStartsUnsaturated(t *testing.T) {
	nn := newTrainableNetwork(t)
	home := &models.PredictionFactors{TeamCode: "HOM", GoalsFor: 3.2, GoalsAgainst: 3.0}
	away := &models.PredictionFactors{TeamCode: "AWY", GoalsFor: 3.1, GoalsAgainst: 3.1}

	result, err := nn.Predict(home, away)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if result.WinProbability < 0.2 || result.WinProbability > 0.8 {
		t.Errorf("an untrained network reported %.4f; it should be near a coin flip, "+
			"not saturated", result.WinProbability)
	}
}

func TestStaleWeightsAreRejected(t *testing.T) {
	nn := newTrainableNetwork(t)
	nn.layers = []int{182, 512, 256, 128, 3}
	nn.initializeNetwork()

	if err := nn.saveWeights(); err != nil {
		t.Fatalf("saveWeights: %v", err)
	}
	if err := nn.loadWeights(); err != nil {
		t.Fatalf("weights written by this version should load: %v", err)
	}

	// Rewrite them as the previous version and confirm they are refused.
	path := filepath.Join(nn.dataDir, "neural_network.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var data NeuralNetworkData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data.Version = "1.0"
	rewritten, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, rewritten, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := nn.loadWeights(); err == nil {
		t.Fatal("weights from the old training regime were accepted; they should be discarded")
	}
}
