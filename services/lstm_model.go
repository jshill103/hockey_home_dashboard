package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ============================================================================
// RE-ENABLED. This model was previously excluded from live ensemble
// predictions because of two bugs, both now fixed:
//  1. trainSequence computed gradients for the LSTM gates but never applied
//     them -- every "trained" prediction still ran on the original random
//     Xavier-initialized weights. Fixed via forwardWithCache/backwardAndUpdate,
//     which cache per-timestep gate activations during the forward pass and
//     run real backpropagation-through-time, applying a gradient-descent
//     update (with clipping) to every gate's weights/biases and the output
//     layer after every sequence.
//  2. extractSequence fed the SAME current-game feature snapshot into all
//     `sequenceLen` (10) timesteps instead of that team's actual last 10
//     games, so the model never saw genuine temporal structure even after
//     bug 1 was fixed. Fixed via extractSequenceForTeam, which pulls the
//     team's real Last10Games history from RollingStatsService (oldest to
//     newest) and buildLSTMFeatures, which is now shared between training
//     (extractGameFeatures, from historical CompletedGame data) and inference
//     (extractSequenceForTeam, from live GameSummary data) so both sides
//     build the same feature vector out of the same underlying fields.
//
// A third bug was found alongside these: TrainOnGameResult (like GB/RF's
// before their own fix) never called the real Train() -- see
// ModelEvaluationService.trainModelBatch's "LSTM" case, which now calls
// lstm.Train() on a rolling window of history instead.
// ============================================================================

// LSTMModel implements a Long Short-Term Memory network for sequential game prediction
type LSTMModel struct {
	// Architecture
	inputSize   int // Features per game
	hiddenSize  int // LSTM hidden state size
	outputSize  int // Output classes (3: win/loss/ot)
	sequenceLen int // Number of past games to consider

	// Weights for LSTM gates (forget, input, output, cell)
	// Format: [gate][layer] where layer is input-to-hidden or hidden-to-hidden
	Wf [][]float64 // Forget gate weights
	Wi [][]float64 // Input gate weights
	Wo [][]float64 // Output gate weights
	Wc [][]float64 // Cell gate weights

	// Biases for LSTM gates
	bf []float64 // Forget gate bias
	bi []float64 // Input gate bias
	bo []float64 // Output gate bias
	bc []float64 // Cell gate bias

	// Output layer weights
	Wy [][]float64 // Hidden to output
	by []float64   // Output bias

	// Training parameters
	learningRate float64
	weight       float64 // Model weight in ensemble
	trained      bool

	// Persistence
	dataDir     string
	lastUpdated time.Time
	mutex       sync.RWMutex

	// Cached sequences for training
	gameSequences []GameSequence
}

// GameSequence represents a sequence of games for training
type GameSequence struct {
	Features [][]float64 // [sequenceLen][inputSize]
	Label    float64     // Outcome (1.0 = win, 0.0 = loss, 0.5 = OT loss)
	TeamCode string
}

// LSTMState represents the hidden and cell state at a timestep
type LSTMState struct {
	h []float64 // Hidden state
	c []float64 // Cell state
}

var (
	lstmModelInstance     *LSTMModel
	lstmModelInstanceOnce sync.Once
)

// NewLSTMModel creates a new LSTM prediction model
func NewLSTMModel() *LSTMModel {
	lstmModelInstanceOnce.Do(func() {
		inputSize := 30   // Features per game (goals, shots, power play, etc.)
		hiddenSize := 64  // LSTM hidden state size
		outputSize := 3   // Win/Loss/OT
		sequenceLen := 10 // Last 10 games

		lstmModelInstance = &LSTMModel{
			inputSize:     inputSize,
			hiddenSize:    hiddenSize,
			outputSize:    outputSize,
			sequenceLen:   sequenceLen,
			learningRate:  0.001,
			weight:        0.07, // 7% weight in ensemble, matching its documented base weight
			trained:       false,
			dataDir:       "data/models",
			lastUpdated:   time.Now(),
			gameSequences: []GameSequence{},
		}

		// Create data directory
		os.MkdirAll(lstmModelInstance.dataDir, 0755)

		// Try to load existing model
		lstmModelInstance.loadModel()

		// Try to load existing weights
		if err := lstmModelInstance.loadWeights(); err != nil {
			log.Printf("🔄 Initializing new LSTM model (no saved weights found)")
			lstmModelInstance.initializeWeights()
		} else {
			log.Printf("🔄 LSTM model loaded from disk")
			log.Printf("   Hidden size: %d, Sequence length: %d", hiddenSize, sequenceLen)
			log.Printf("   Last updated: %s", lstmModelInstance.lastUpdated.Format("2006-01-02 15:04:05"))
		}
	})

	return lstmModelInstance
}

// GetLSTMModel returns the singleton instance
func GetLSTMModel() *LSTMModel {
	if lstmModelInstance == nil {
		return NewLSTMModel()
	}
	return lstmModelInstance
}

// initializeWeights initializes LSTM weights with Xavier initialization
func (lstm *LSTMModel) initializeWeights() {
	// Xavier initialization scale
	scaleIH := math.Sqrt(2.0 / float64(lstm.inputSize+lstm.hiddenSize))
	scaleHH := math.Sqrt(2.0 / float64(lstm.hiddenSize+lstm.hiddenSize))
	scaleHO := math.Sqrt(2.0 / float64(lstm.hiddenSize+lstm.outputSize))

	// Initialize forget gate
	lstm.Wf = make([][]float64, 2)
	lstm.Wf[0] = randomMatrix(lstm.hiddenSize, lstm.inputSize, scaleIH)  // Input to hidden
	lstm.Wf[1] = randomMatrix(lstm.hiddenSize, lstm.hiddenSize, scaleHH) // Hidden to hidden
	lstm.bf = make([]float64, lstm.hiddenSize)
	for i := range lstm.bf {
		lstm.bf[i] = 1.0 // Initialize forget bias to 1 (remember by default)
	}

	// Initialize input gate
	lstm.Wi = make([][]float64, 2)
	lstm.Wi[0] = randomMatrix(lstm.hiddenSize, lstm.inputSize, scaleIH)
	lstm.Wi[1] = randomMatrix(lstm.hiddenSize, lstm.hiddenSize, scaleHH)
	lstm.bi = make([]float64, lstm.hiddenSize)

	// Initialize output gate
	lstm.Wo = make([][]float64, 2)
	lstm.Wo[0] = randomMatrix(lstm.hiddenSize, lstm.inputSize, scaleIH)
	lstm.Wo[1] = randomMatrix(lstm.hiddenSize, lstm.hiddenSize, scaleHH)
	lstm.bo = make([]float64, lstm.hiddenSize)

	// Initialize cell gate
	lstm.Wc = make([][]float64, 2)
	lstm.Wc[0] = randomMatrix(lstm.hiddenSize, lstm.inputSize, scaleIH)
	lstm.Wc[1] = randomMatrix(lstm.hiddenSize, lstm.hiddenSize, scaleHH)
	lstm.bc = make([]float64, lstm.hiddenSize)

	// Initialize output layer
	lstm.Wy = randomMatrix2D(lstm.outputSize, lstm.hiddenSize, scaleHO)
	lstm.by = make([]float64, lstm.outputSize)

	log.Printf("🔄 LSTM weights initialized: %dx%d hidden units, %d sequence length",
		lstm.hiddenSize, lstm.inputSize, lstm.sequenceLen)
}

// randomMatrix creates a random matrix with Xavier initialization
func randomMatrix(rows, cols int, scale float64) []float64 {
	matrix := make([]float64, rows*cols)
	for i := range matrix {
		matrix[i] = (rand.Float64()*2 - 1) * scale
	}
	return matrix
}

// randomMatrix2D creates a 2D random matrix
func randomMatrix2D(rows, cols int, scale float64) [][]float64 {
	matrix := make([][]float64, rows)
	for i := range matrix {
		matrix[i] = make([]float64, cols)
		for j := range matrix[i] {
			matrix[i][j] = (rand.Float64()*2 - 1) * scale
		}
	}
	return matrix
}

// Predict makes a prediction using the LSTM model
func (lstm *LSTMModel) Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error) {
	lstm.mutex.RLock()
	defer lstm.mutex.RUnlock()

	start := time.Now()

	if !lstm.trained {
		// Return neutral prediction if not trained
		return &models.ModelResult{
			ModelName:      "LSTM",
			WinProbability: 0.50,
			Confidence:     0.30,
			PredictedScore: "3-2",
			Weight:         lstm.weight,
			ProcessingTime: time.Since(start).Milliseconds(),
		}, nil
	}

	// Get game sequences for both teams
	homeSequence := lstm.extractSequenceForTeam(homeFactors.TeamCode)
	awaySequence := lstm.extractSequenceForTeam(awayFactors.TeamCode)

	// Run LSTM forward pass for both teams
	homeOutput := lstm.forward(homeSequence)
	awayOutput := lstm.forward(awaySequence)

	// Compare outputs to determine win probability
	// homeOutput[0] = win prob, homeOutput[1] = loss prob, homeOutput[2] = OT prob
	homeStrength := homeOutput[0] - awayOutput[0]

	// Convert to win probability (sigmoid)
	winProb := 1.0 / (1.0 + math.Exp(-homeStrength))

	// Add home ice advantage
	winProb += 0.05

	// Ensure reasonable bounds
	winProb = math.Max(0.35, math.Min(0.85, winProb))

	// Calculate confidence based on prediction strength
	confidence := math.Abs(winProb-0.5) * 2.0
	confidence = math.Max(0.40, math.Min(0.90, confidence))

	// Predict score
	predictedScore := lstm.predictScore(winProb, homeFactors, awayFactors)

	result := &models.ModelResult{
		ModelName:      "LSTM",
		WinProbability: winProb,
		Confidence:     confidence,
		PredictedScore: predictedScore,
		Weight:         lstm.weight,
		ProcessingTime: time.Since(start).Milliseconds(),
	}

	return result, nil
}

// forward performs LSTM forward pass on a sequence
func (lstm *LSTMModel) forward(sequence [][]float64) []float64 {
	// Initialize hidden and cell states
	h := make([]float64, lstm.hiddenSize)
	c := make([]float64, lstm.hiddenSize)

	// Process each timestep in the sequence
	for t := 0; t < len(sequence); t++ {
		x := sequence[t]

		// Forget gate: f_t = sigmoid(Wf * [h_{t-1}, x_t] + bf)
		ft := lstm.gate(lstm.Wf, x, h, lstm.bf, sigmoid)

		// Input gate: i_t = sigmoid(Wi * [h_{t-1}, x_t] + bi)
		it := lstm.gate(lstm.Wi, x, h, lstm.bi, sigmoid)

		// Cell gate: c_tilde = tanh(Wc * [h_{t-1}, x_t] + bc)
		cTilde := lstm.gate(lstm.Wc, x, h, lstm.bc, tanhActivation)

		// Update cell state: c_t = f_t * c_{t-1} + i_t * c_tilde
		for i := 0; i < lstm.hiddenSize; i++ {
			c[i] = ft[i]*c[i] + it[i]*cTilde[i]
		}

		// Output gate: o_t = sigmoid(Wo * [h_{t-1}, x_t] + bo)
		ot := lstm.gate(lstm.Wo, x, h, lstm.bo, sigmoid)

		// Update hidden state: h_t = o_t * tanh(c_t)
		for i := 0; i < lstm.hiddenSize; i++ {
			h[i] = ot[i] * tanhActivation(c[i])
		}
	}

	// Output layer: y = softmax(Wy * h + by)
	output := make([]float64, lstm.outputSize)
	for i := 0; i < lstm.outputSize; i++ {
		sum := lstm.by[i]
		for j := 0; j < lstm.hiddenSize; j++ {
			sum += lstm.Wy[i][j] * h[j]
		}
		output[i] = sum
	}

	// Apply softmax
	return softmax(output)
}

// gate computes a single LSTM gate
func (lstm *LSTMModel) gate(W [][]float64, x, h []float64, b []float64, activation func(float64) float64) []float64 {
	result := make([]float64, lstm.hiddenSize)

	// Validate inputs
	if len(W) < 2 {
		log.Printf("⚠️ Warning: Invalid weight matrix dimensions in LSTM gate")
		return result
	}
	if len(W[0]) < lstm.hiddenSize*lstm.inputSize || len(W[1]) < lstm.hiddenSize*lstm.hiddenSize {
		log.Printf("⚠️ Warning: Weight matrix too small for LSTM gate")
		return result
	}
	if len(b) < lstm.hiddenSize {
		log.Printf("⚠️ Warning: Bias vector too small for LSTM gate")
		return result
	}
	if len(x) < lstm.inputSize {
		log.Printf("⚠️ Warning: Input vector too small for LSTM gate")
		return result
	}
	if len(h) < lstm.hiddenSize {
		log.Printf("⚠️ Warning: Hidden vector too small for LSTM gate")
		return result
	}

	for i := 0; i < lstm.hiddenSize; i++ {
		sum := b[i]

		// Input contribution: W[0] * x (with bounds checking)
		for j := 0; j < lstm.inputSize && j < len(x); j++ {
			idx := i*lstm.inputSize + j
			if idx < len(W[0]) {
				sum += W[0][idx] * x[j]
			}
		}

		// Hidden contribution: W[1] * h (with bounds checking)
		for j := 0; j < lstm.hiddenSize && j < len(h); j++ {
			idx := i*lstm.hiddenSize + j
			if idx < len(W[1]) {
				sum += W[1][idx] * h[j]
			}
		}

		result[i] = activation(sum)
	}

	return result
}

// buildLSTMFeatures converts one game's box-score stats (from a specific
// team's perspective) into a fixed-length feature vector for one LSTM
// timestep. Shared by extractGameFeatures (training, from historical
// CompletedGame data) and extractSequenceForTeam (inference, from live
// RollingStatsService GameSummary data) so both sides build features out of
// the same fields the same way.
func (lstm *LSTMModel) buildLSTMFeatures(goalsFor, goalsAgainst, shotsFor, shotsAgainst, ppGoals, ppOpps int, isHome, won bool, points int) []float64 {
	features := make([]float64, lstm.inputSize)
	idx := 0

	features[idx] = float64(goalsFor) / 8.0
	idx++
	features[idx] = float64(goalsAgainst) / 8.0
	idx++
	features[idx] = float64(goalsFor-goalsAgainst) / 8.0
	idx++
	features[idx] = float64(shotsFor) / 45.0
	idx++
	features[idx] = float64(shotsAgainst) / 45.0
	idx++
	ppPct := 0.0
	if ppOpps > 0 {
		ppPct = float64(ppGoals) / float64(ppOpps)
	}
	features[idx] = ppPct
	idx++
	features[idx] = float64(ppOpps) / 6.0
	idx++
	if isHome {
		features[idx] = 1.0
	}
	idx++
	if won {
		features[idx] = 1.0
	}
	idx++
	features[idx] = float64(points) / 2.0
	idx++

	// Remaining features reserved for future signals (rest days, opponent
	// strength, etc.) once those are reliably tracked per historical game.
	for idx < lstm.inputSize {
		features[idx] = 0.0
		idx++
	}

	return features
}

// extractSequenceForTeam builds a real chronological (oldest-to-newest)
// sequence of teamCode's last games from RollingStatsService, replacing the
// old extractSequence which fed the same current-game snapshot into every
// timestep. Teams with fewer than sequenceLen games return a shorter
// sequence -- forward() handles any sequence length.
func (lstm *LSTMModel) extractSequenceForTeam(teamCode string) [][]float64 {
	rollingStats := GetRollingStatsService()
	if rollingStats == nil {
		return [][]float64{}
	}
	stats, err := rollingStats.GetTeamStats(teamCode)
	if err != nil || stats == nil || len(stats.Last10Games) == 0 {
		return [][]float64{}
	}

	// Last10Games is stored newest-first; reverse to oldest-first so the
	// final timestep (and thus the final hidden state driving the output)
	// reflects the most recent game.
	games := stats.Last10Games
	sequence := make([][]float64, len(games))
	for i, g := range games {
		sequence[len(games)-1-i] = lstm.buildLSTMFeatures(
			g.GoalsFor, g.GoalsAgainst, g.Shots, g.ShotsAgainst,
			g.PowerPlayGoals, g.PowerPlayOpps, g.IsHome, g.Result == "W", g.Points,
		)
	}
	return sequence
}

// predictScore predicts the final score
func (lstm *LSTMModel) predictScore(winProb float64, homeFactors, awayFactors *models.PredictionFactors) string {
	// Base expected goals
	homeGoals := 3.0
	awayGoals := 2.5

	// Adjust based on win probability
	if winProb > 0.5 {
		homeGoals += (winProb - 0.5) * 2.0
		awayGoals -= (winProb - 0.5) * 1.5
	} else {
		homeGoals -= (0.5 - winProb) * 1.5
		awayGoals += (0.5 - winProb) * 2.0
	}

	// Round to integers
	homeScore := int(math.Round(homeGoals))
	awayScore := int(math.Round(awayGoals))

	// Ensure minimum score difference
	if homeScore == awayScore {
		if winProb > 0.5 {
			homeScore++
		} else {
			awayScore++
		}
	}

	return fmt.Sprintf("%d-%d", homeScore, awayScore)
}

// Train trains the LSTM model on game sequences
func (lstm *LSTMModel) Train(games []models.CompletedGame) error {
	lstm.mutex.Lock()
	defer lstm.mutex.Unlock()

	if len(games) < lstm.sequenceLen {
		return fmt.Errorf("insufficient training data: need at least %d games, have %d", lstm.sequenceLen, len(games))
	}

	log.Printf("🔄 Training LSTM model on %d games...", len(games))
	start := time.Now()

	// Prepare sequences from games
	sequences := lstm.prepareSequences(games)

	if len(sequences) == 0 {
		return fmt.Errorf("no valid sequences created from games")
	}

	// Training epochs
	epochs := 10
	for epoch := 0; epoch < epochs; epoch++ {
		totalLoss := 0.0

		for _, seq := range sequences {
			loss := lstm.trainSequence(seq)
			totalLoss += loss
		}

		avgLoss := totalLoss / float64(len(sequences))
		if (epoch+1)%2 == 0 {
			log.Printf("   Epoch %d/%d: Loss %.4f", epoch+1, epochs, avgLoss)
		}
	}

	lstm.trained = true
	lstm.lastUpdated = time.Now()

	trainingTime := time.Since(start)
	log.Printf("✅ LSTM training complete!")
	log.Printf("   Sequences: %d | Time: %.1fs", len(sequences), trainingTime.Seconds())

	// Save the trained model
	if err := lstm.saveModel(); err != nil {
		log.Printf("⚠️ Failed to save LSTM weights: %v", err)
	}

	return nil
}

// prepareSequences creates training sequences from completed games
func (lstm *LSTMModel) prepareSequences(games []models.CompletedGame) []GameSequence {
	sequences := []GameSequence{}

	// Group games by team
	teamGames := make(map[string][]models.CompletedGame)
	for _, game := range games {
		teamGames[game.HomeTeam.TeamCode] = append(teamGames[game.HomeTeam.TeamCode], game)
		teamGames[game.AwayTeam.TeamCode] = append(teamGames[game.AwayTeam.TeamCode], game)
	}

	// Create sequences for each team
	for teamCode, tGames := range teamGames {
		if len(tGames) < lstm.sequenceLen+1 {
			continue
		}

		// Create sliding window sequences
		for i := 0; i <= len(tGames)-lstm.sequenceLen-1; i++ {
			sequence := make([][]float64, lstm.sequenceLen)

			// Extract features from sequence
			for t := 0; t < lstm.sequenceLen; t++ {
				sequence[t] = lstm.extractGameFeatures(&tGames[i+t], teamCode)
			}

			// Label is the outcome of the next game
			nextGame := &tGames[i+lstm.sequenceLen]
			label := lstm.getGameLabel(nextGame, teamCode)

			sequences = append(sequences, GameSequence{
				Features: sequence,
				Label:    label,
				TeamCode: teamCode,
			})
		}
	}

	return sequences
}

// extractGameFeatures extracts features from a completed game for teamCode's
// perspective, using the same buildLSTMFeatures extractor as inference.
func (lstm *LSTMModel) extractGameFeatures(game *models.CompletedGame, teamCode string) []float64 {
	var team, opponent models.TeamGameResult
	isHome := game.HomeTeam.TeamCode == teamCode
	if isHome {
		team, opponent = game.HomeTeam, game.AwayTeam
	} else {
		team, opponent = game.AwayTeam, game.HomeTeam
	}

	won := team.Score > opponent.Score
	points := 0
	if won {
		points = 2
	} else if game.WinType == "OT" || game.WinType == "SO" {
		points = 1
	}

	return lstm.buildLSTMFeatures(
		team.Score, opponent.Score, team.Shots, opponent.Shots,
		team.PowerPlayGoals, team.PowerPlayOpps, isHome, won, points,
	)
}

// getGameLabel returns the label for a game (1.0 = win, 0.0 = loss, 0.5 = OT loss)
func (lstm *LSTMModel) getGameLabel(game *models.CompletedGame, teamCode string) float64 {
	isHome := game.HomeTeam.TeamCode == teamCode
	won := (isHome && game.HomeTeam.Score > game.AwayTeam.Score) || (!isHome && game.AwayTeam.Score > game.HomeTeam.Score)

	if won {
		return 1.0
	}

	// Check for OT/SO loss (gets a point)
	if game.WinType == "OT" || game.WinType == "SO" {
		return 0.5
	}

	return 0.0
}

// lstmTimestepCache holds the intermediate values from one forward-pass
// timestep that backwardAndUpdate needs to compute gradients.
type lstmTimestepCache struct {
	x, hPrev, cPrev []float64
	f, i, o, cTilde []float64
	c, h            []float64
}

// forwardWithCache is forward's twin for training: same computation, but it
// also stashes every timestep's gate activations and cell/hidden states so
// backwardAndUpdate can run backpropagation-through-time afterward. forward
// itself stays cache-free since Predict() runs far more often than training.
func (lstm *LSTMModel) forwardWithCache(sequence [][]float64) ([]float64, []lstmTimestepCache) {
	h := make([]float64, lstm.hiddenSize)
	c := make([]float64, lstm.hiddenSize)
	caches := make([]lstmTimestepCache, len(sequence))

	for t := 0; t < len(sequence); t++ {
		x := sequence[t]
		hPrev := append([]float64(nil), h...)
		cPrev := append([]float64(nil), c...)

		ft := lstm.gate(lstm.Wf, x, h, lstm.bf, sigmoid)
		it := lstm.gate(lstm.Wi, x, h, lstm.bi, sigmoid)
		cTilde := lstm.gate(lstm.Wc, x, h, lstm.bc, tanhActivation)

		newC := make([]float64, lstm.hiddenSize)
		for k := 0; k < lstm.hiddenSize; k++ {
			newC[k] = ft[k]*cPrev[k] + it[k]*cTilde[k]
		}

		ot := lstm.gate(lstm.Wo, x, h, lstm.bo, sigmoid)

		newH := make([]float64, lstm.hiddenSize)
		for k := 0; k < lstm.hiddenSize; k++ {
			newH[k] = ot[k] * tanhActivation(newC[k])
		}

		caches[t] = lstmTimestepCache{x: x, hPrev: hPrev, cPrev: cPrev, f: ft, i: it, o: ot, cTilde: cTilde, c: newC, h: newH}
		h, c = newH, newC
	}

	output := make([]float64, lstm.outputSize)
	for i := 0; i < lstm.outputSize; i++ {
		sum := lstm.by[i]
		for j := 0; j < lstm.hiddenSize; j++ {
			sum += lstm.Wy[i][j] * h[j]
		}
		output[i] = sum
	}

	return softmax(output), caches
}

// clipGrad bounds a single gradient component to avoid exploding gradients
// destabilizing this hand-rolled (no optimizer momentum/normalization) SGD.
func clipGrad(v float64) float64 {
	const bound = 5.0
	if v > bound {
		return bound
	}
	if v < -bound {
		return -bound
	}
	return v
}

// backwardAndUpdate runs backpropagation-through-time over the cached
// forward pass and applies a plain SGD update to every gate's weights/biases
// and the output layer. This is the piece that was previously missing
// entirely: trainSequence computed a loss but never adjusted any weight.
func (lstm *LSTMModel) backwardAndUpdate(caches []lstmTimestepCache, predicted, target []float64) {
	T := len(caches)
	if T == 0 {
		return
	}
	hiddenSize, inputSize := lstm.hiddenSize, lstm.inputSize
	finalH := caches[T-1].h

	// Output layer gradients (standard softmax + cross-entropy gradient)
	dy := make([]float64, lstm.outputSize)
	for i := range dy {
		dy[i] = predicted[i] - target[i]
	}
	dWy := make([][]float64, lstm.outputSize)
	dby := make([]float64, lstm.outputSize)
	for i := 0; i < lstm.outputSize; i++ {
		dWy[i] = make([]float64, hiddenSize)
		for j := 0; j < hiddenSize; j++ {
			dWy[i][j] = dy[i] * finalH[j]
		}
		dby[i] = dy[i]
	}

	dhNext := make([]float64, hiddenSize)
	for j := 0; j < hiddenSize; j++ {
		sum := 0.0
		for i := 0; i < lstm.outputSize; i++ {
			sum += lstm.Wy[i][j] * dy[i]
		}
		dhNext[j] = sum
	}
	dcNext := make([]float64, hiddenSize)

	// Gradient accumulators, flattened with the same [ih | hh] layout gate() uses.
	dWfIH, dWfHH, dbf := make([]float64, hiddenSize*inputSize), make([]float64, hiddenSize*hiddenSize), make([]float64, hiddenSize)
	dWiIH, dWiHH, dbi := make([]float64, hiddenSize*inputSize), make([]float64, hiddenSize*hiddenSize), make([]float64, hiddenSize)
	dWoIH, dWoHH, dbo := make([]float64, hiddenSize*inputSize), make([]float64, hiddenSize*hiddenSize), make([]float64, hiddenSize)
	dWcIH, dWcHH, dbc := make([]float64, hiddenSize*inputSize), make([]float64, hiddenSize*hiddenSize), make([]float64, hiddenSize)

	accumulateGateGrads := func(dgate, x, hPrev, dWih, dWhh, db []float64) {
		for i := 0; i < hiddenSize; i++ {
			for j := 0; j < inputSize; j++ {
				dWih[i*inputSize+j] += dgate[i] * x[j]
			}
			for j := 0; j < hiddenSize; j++ {
				dWhh[i*hiddenSize+j] += dgate[i] * hPrev[j]
			}
			db[i] += dgate[i]
		}
	}

	for t := T - 1; t >= 0; t-- {
		ck := caches[t]

		do := make([]float64, hiddenSize)
		dcTotal := make([]float64, hiddenSize)
		for k := 0; k < hiddenSize; k++ {
			tanhC := tanhActivation(ck.c[k])
			do[k] = dhNext[k] * tanhC * ck.o[k] * (1 - ck.o[k])
			dcTotal[k] = dcNext[k] + dhNext[k]*ck.o[k]*(1-tanhC*tanhC)
		}

		df := make([]float64, hiddenSize)
		di := make([]float64, hiddenSize)
		dcTilde := make([]float64, hiddenSize)
		dcPrev := make([]float64, hiddenSize)
		for k := 0; k < hiddenSize; k++ {
			df[k] = dcTotal[k] * ck.cPrev[k] * ck.f[k] * (1 - ck.f[k])
			di[k] = dcTotal[k] * ck.cTilde[k] * ck.i[k] * (1 - ck.i[k])
			dcTilde[k] = dcTotal[k] * ck.i[k] * (1 - ck.cTilde[k]*ck.cTilde[k])
			dcPrev[k] = dcTotal[k] * ck.f[k]
		}

		accumulateGateGrads(df, ck.x, ck.hPrev, dWfIH, dWfHH, dbf)
		accumulateGateGrads(di, ck.x, ck.hPrev, dWiIH, dWiHH, dbi)
		accumulateGateGrads(do, ck.x, ck.hPrev, dWoIH, dWoHH, dbo)
		accumulateGateGrads(dcTilde, ck.x, ck.hPrev, dWcIH, dWcHH, dbc)

		dhPrev := make([]float64, hiddenSize)
		for j := 0; j < hiddenSize; j++ {
			sum := 0.0
			for i := 0; i < hiddenSize; i++ {
				sum += lstm.Wf[1][i*hiddenSize+j] * df[i]
				sum += lstm.Wi[1][i*hiddenSize+j] * di[i]
				sum += lstm.Wo[1][i*hiddenSize+j] * do[i]
				sum += lstm.Wc[1][i*hiddenSize+j] * dcTilde[i]
			}
			dhPrev[j] = sum
		}

		dhNext = dhPrev
		dcNext = dcPrev
	}

	applyUpdate := func(W, dW []float64) {
		for i := range W {
			W[i] -= lstm.learningRate * clipGrad(dW[i])
		}
	}
	applyUpdate(lstm.Wf[0], dWfIH)
	applyUpdate(lstm.Wf[1], dWfHH)
	applyUpdate(lstm.Wi[0], dWiIH)
	applyUpdate(lstm.Wi[1], dWiHH)
	applyUpdate(lstm.Wo[0], dWoIH)
	applyUpdate(lstm.Wo[1], dWoHH)
	applyUpdate(lstm.Wc[0], dWcIH)
	applyUpdate(lstm.Wc[1], dWcHH)
	for i := range lstm.bf {
		lstm.bf[i] -= lstm.learningRate * clipGrad(dbf[i])
	}
	for i := range lstm.bi {
		lstm.bi[i] -= lstm.learningRate * clipGrad(dbi[i])
	}
	for i := range lstm.bo {
		lstm.bo[i] -= lstm.learningRate * clipGrad(dbo[i])
	}
	for i := range lstm.bc {
		lstm.bc[i] -= lstm.learningRate * clipGrad(dbc[i])
	}
	for i := range lstm.Wy {
		for j := range lstm.Wy[i] {
			lstm.Wy[i][j] -= lstm.learningRate * clipGrad(dWy[i][j])
		}
		lstm.by[i] -= lstm.learningRate * clipGrad(dby[i])
	}
}

// trainSequence trains on a single sequence using backpropagation through time
func (lstm *LSTMModel) trainSequence(seq GameSequence) float64 {
	if len(seq.Features) == 0 {
		return 0.0
	}

	output, caches := lstm.forwardWithCache(seq.Features)

	// Compute loss (cross-entropy)
	target := make([]float64, lstm.outputSize)
	if seq.Label == 1.0 {
		target[0] = 1.0 // Win
	} else if seq.Label == 0.5 {
		target[2] = 1.0 // OT
	} else {
		target[1] = 1.0 // Loss
	}

	loss := 0.0
	for i := 0; i < lstm.outputSize; i++ {
		if target[i] > 0 {
			loss -= target[i] * math.Log(math.Max(output[i], 1e-10))
		}
	}

	lstm.backwardAndUpdate(caches, output, target)

	return loss
}

// Activation functions
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func tanhActivation(x float64) float64 {
	return math.Tanh(x)
}

func softmax(x []float64) []float64 {
	max := x[0]
	for _, v := range x {
		if v > max {
			max = v
		}
	}

	exp := make([]float64, len(x))
	sum := 0.0
	for i, v := range x {
		exp[i] = math.Exp(v - max)
		sum += exp[i]
	}

	for i := range exp {
		exp[i] /= sum
	}

	return exp
}

// GetName returns the model name
func (lstm *LSTMModel) GetName() string {
	return "LSTM"
}

// GetWeight returns the model weight in ensemble
func (lstm *LSTMModel) GetWeight() float64 {
	lstm.mutex.RLock()
	defer lstm.mutex.RUnlock()
	return lstm.weight
}

// LSTMModelData represents serializable LSTM model data
type LSTMModelData struct {
	InputSize    int         `json:"inputSize"`
	HiddenSize   int         `json:"hiddenSize"`
	OutputSize   int         `json:"outputSize"`
	SequenceLen  int         `json:"sequenceLen"`
	Wf           [][]float64 `json:"wf"`
	Wi           [][]float64 `json:"wi"`
	Wo           [][]float64 `json:"wo"`
	Wc           [][]float64 `json:"wc"`
	Bf           []float64   `json:"bf"`
	Bi           []float64   `json:"bi"`
	Bo           []float64   `json:"bo"`
	Bc           []float64   `json:"bc"`
	Wy           [][]float64 `json:"wy"`
	By           []float64   `json:"by"`
	LearningRate float64     `json:"learningRate"`
	Weight       float64     `json:"weight"`
	Trained      bool        `json:"trained"`
	LastUpdated  time.Time   `json:"lastUpdated"`
	Version      string      `json:"version"`
}

// saveWeights saves LSTM weights to disk
func (lstm *LSTMModel) saveWeights() error {
	filePath := filepath.Join(lstm.dataDir, "lstm_weights.json")

	modelData := LSTMModelData{
		InputSize:    lstm.inputSize,
		HiddenSize:   lstm.hiddenSize,
		OutputSize:   lstm.outputSize,
		SequenceLen:  lstm.sequenceLen,
		Wf:           lstm.Wf,
		Wi:           lstm.Wi,
		Wo:           lstm.Wo,
		Wc:           lstm.Wc,
		Bf:           lstm.bf,
		Bi:           lstm.bi,
		Bo:           lstm.bo,
		Bc:           lstm.bc,
		Wy:           lstm.Wy,
		By:           lstm.by,
		LearningRate: lstm.learningRate,
		Weight:       lstm.weight,
		Trained:      lstm.trained,
		LastUpdated:  time.Now(),
		Version:      "1.0",
	}

	data, err := json.MarshalIndent(modelData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling LSTM weights: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing LSTM weights: %w", err)
	}

	log.Printf("💾 LSTM weights saved: %dx%d hidden, trained=%v", lstm.hiddenSize, lstm.inputSize, lstm.trained)
	return nil
}

// loadWeights loads LSTM weights from disk
func (lstm *LSTMModel) loadWeights() error {
	filePath := filepath.Join(lstm.dataDir, "lstm_weights.json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("no saved weights found")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading LSTM weights: %w", err)
	}

	var modelData LSTMModelData
	err = json.Unmarshal(data, &modelData)
	if err != nil {
		return fmt.Errorf("error unmarshaling LSTM weights: %w", err)
	}

	// Load all weights
	lstm.inputSize = modelData.InputSize
	lstm.hiddenSize = modelData.HiddenSize
	lstm.outputSize = modelData.OutputSize
	lstm.sequenceLen = modelData.SequenceLen
	lstm.Wf = modelData.Wf
	lstm.Wi = modelData.Wi
	lstm.Wo = modelData.Wo
	lstm.Wc = modelData.Wc
	lstm.bf = modelData.Bf
	lstm.bi = modelData.Bi
	lstm.bo = modelData.Bo
	lstm.bc = modelData.Bc
	lstm.Wy = modelData.Wy
	lstm.by = modelData.By
	lstm.learningRate = modelData.LearningRate
	lstm.weight = modelData.Weight
	lstm.trained = modelData.Trained
	lstm.lastUpdated = modelData.LastUpdated

	return nil
}

// loadModel loads the complete LSTM model from disk
func (lstm *LSTMModel) loadModel() {
	// Try to load weights first
	if err := lstm.loadWeights(); err != nil {
		log.Printf("🔄 No saved LSTM model found, initializing new model")
		lstm.initializeWeights()
		return
	}

	// Try to load game sequences
	if err := lstm.loadGameSequences(); err != nil {
		log.Printf("⚠️ Could not load LSTM game sequences: %v", err)
		lstm.gameSequences = []GameSequence{}
	}

	log.Printf("✅ LSTM model loaded: %dx%d hidden, %d sequences, trained=%v",
		lstm.hiddenSize, lstm.inputSize, len(lstm.gameSequences), lstm.trained)
}

// saveModel saves the complete LSTM model to disk. Does not lock mutex --
// matches the convention GB/RF/Meta-Learner's saveModel already use, since
// Train() calls this while still holding lstm.mutex.Lock() itself; locking
// again here would deadlock (sync.Mutex is not reentrant). This was latent
// and harmless before because nothing ever called Train() in production;
// now that ModelEvaluationService.trainModelBatch does, it would otherwise
// hang the whole app the first time LSTM's batch threshold is hit.
func (lstm *LSTMModel) saveModel() error {
	// Save weights
	if err := lstm.saveWeights(); err != nil {
		return fmt.Errorf("failed to save LSTM weights: %w", err)
	}

	// Save game sequences
	if err := lstm.saveGameSequences(); err != nil {
		return fmt.Errorf("failed to save LSTM sequences: %w", err)
	}

	lstm.lastUpdated = time.Now()
	return nil
}

// saveGameSequences saves game sequences to disk
func (lstm *LSTMModel) saveGameSequences() error {
	filePath := filepath.Join(lstm.dataDir, "lstm_sequences.json")

	data := struct {
		Sequences   []GameSequence `json:"sequences"`
		LastUpdated time.Time      `json:"lastUpdated"`
		Version     string         `json:"version"`
	}{
		Sequences:   lstm.gameSequences,
		LastUpdated: time.Now(),
		Version:     "1.0",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling LSTM sequences: %w", err)
	}

	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing LSTM sequences file: %w", err)
	}

	return nil
}

// loadGameSequences loads game sequences from disk
func (lstm *LSTMModel) loadGameSequences() error {
	filePath := filepath.Join(lstm.dataDir, "lstm_sequences.json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("no saved sequences found")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading LSTM sequences: %w", err)
	}

	var sequenceData struct {
		Sequences   []GameSequence `json:"sequences"`
		LastUpdated time.Time      `json:"lastUpdated"`
		Version     string         `json:"version"`
	}

	err = json.Unmarshal(data, &sequenceData)
	if err != nil {
		return fmt.Errorf("error unmarshaling LSTM sequences: %w", err)
	}

	lstm.gameSequences = sequenceData.Sequences
	return nil
}
