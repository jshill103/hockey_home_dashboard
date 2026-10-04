package services

import (
	"math"
	"testing"

	"github.com/jaredshillingburg/go_uhc/models"
)

// forwardPass already sigmoids the output layer and backpropagate trains
// against that activation, so Predict must report output[0] as it stands.
// Applying sigmoid a second time folded every forecast into (0.5, 0.731),
// which is why the model never predicted an away win across 569 games.
func TestPredictReportsTheLearnedProbability(t *testing.T) {
	nn := &NeuralNetworkModel{
		layers:       []int{182, 8, 3},
		learningRate: 0.001,
		weight:       0.1,
		dataDir:      t.TempDir(),
	}
	nn.initializeNetwork()

	home := teamRates("HOM", 3.4, 2.8)
	away := teamRates("AWY", 2.9, 3.3)

	result, err := nn.Predict(home, away)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}

	want := nn.forwardPass(nn.extractFeatures(home, away))[0]
	if math.Abs(result.WinProbability-want) > 1e-12 {
		t.Fatalf("WinProbability = %v, want the network output %v (a second sigmoid would give %v)",
			result.WinProbability, want, nn.sigmoid(want))
	}
}

// The double sigmoid made sub-0.5 output arithmetically impossible. Train the
// network toward away wins and confirm it can now say so.
func TestNetworkCanPredictAnAwayWin(t *testing.T) {
	nn := &NeuralNetworkModel{
		layers:       []int{182, 16, 3},
		learningRate: 0.05,
		weight:       0.1,
		dataDir:      t.TempDir(),
	}
	nn.initializeNetwork()

	home := teamRates("HOM", 2.2, 3.8)
	away := teamRates("AWY", 4.1, 2.3)
	features := nn.extractFeatures(home, away)

	// Target an away win: output[0] is the home win probability.
	target := []float64{0.0, 2.0 / 8.0, 5.0 / 8.0}
	for i := 0; i < 400; i++ {
		nn.backpropagate(features, target)
	}

	result, err := nn.Predict(home, away)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if result.WinProbability >= 0.5 {
		t.Fatalf("WinProbability = %v after training toward an away win; the old double sigmoid "+
			"made anything below 0.5 impossible", result.WinProbability)
	}
}

func TestPredictStaysInRangeAndNamesAWinner(t *testing.T) {
	nn := &NeuralNetworkModel{
		layers:       []int{182, 8, 3},
		learningRate: 0.001,
		weight:       0.1,
		dataDir:      t.TempDir(),
	}
	nn.initializeNetwork()

	pairs := [][2]*models.PredictionFactors{
		{teamRates("A", 3.1, 3.1), teamRates("B", 3.1, 3.1)},
		{teamRates("C", 4.8, 2.1), teamRates("D", 2.0, 4.6)},
		{teamRates("E", 0, 0), teamRates("F", 0, 0)},
	}
	for _, pair := range pairs {
		result, err := nn.Predict(pair[0], pair[1])
		if err != nil {
			t.Fatalf("Predict: %v", err)
		}
		if result.WinProbability < 0 || result.WinProbability > 1 {
			t.Errorf("WinProbability = %v, outside 0-1", result.WinProbability)
		}
		h, a := parseScore(t, result.PredictedScore)
		if h == a {
			t.Errorf("PredictedScore = %q, a tie", result.PredictedScore)
		}
	}
}
