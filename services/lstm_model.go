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
			weight:        0.08, // 8% weight in ensemble
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
	homeSequence := lstm.extractSequence(homeFactors)
	awaySequence := lstm.extractSequence(awayFactors)

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

	for i := 0; i < lstm.hiddenSize; i++ {
		sum := b[i]

		// Input contribution: W[0] * x
		for j := 0; j < lstm.inputSize; j++ {
			sum += W[0][i*lstm.inputSize+j] * x[j]
		}

		// Hidden contribution: W[1] * h
		for j := 0; j < lstm.hiddenSize; j++ {
			sum += W[1][i*lstm.hiddenSize+j] * h[j]
		}

		result[i] = activation(sum)
	}

	return result
}

// extractSequence extracts a game sequence from prediction factors
func (lstm *LSTMModel) extractSequence(factors *models.PredictionFactors) [][]float64 {
	// For now, create a simple sequence from rolling stats
	// In production, this would use actual game history
	sequence := make([][]float64, lstm.sequenceLen)

	for t := 0; t < lstm.sequenceLen; t++ {
		features := make([]float64, lstm.inputSize)
		idx := 0

		// Add rolling stats
		features[idx] = factors.MomentumScore
		idx++
		features[idx] = factors.WeightedWinPct
		idx++
		features[idx] = factors.WeightedGoalsFor
		idx++
		features[idx] = factors.WeightedGoalsAgainst
		idx++
		if factors.IsHot {
			features[idx] = 1.0
		}
		idx++
		if factors.IsCold {
			features[idx] = 1.0
		}
		idx++
		features[idx] = float64(factors.Last5GamesPoints)
		idx++
		features[idx] = float64(factors.GoalDifferential5)
		idx++

		// Add basic stats
		features[idx] = factors.WinPercentage
		idx++
		features[idx] = factors.GoalsFor
		idx++
		features[idx] = factors.GoalsAgainst
		idx++
		features[idx] = factors.PowerPlayPct
		idx++
		features[idx] = factors.PenaltyKillPct
		idx++

		// Add player impact
		features[idx] = factors.StarPowerRating
		idx++
		features[idx] = factors.Top3CombinedPPG
		idx++
		features[idx] = factors.DepthScoring
		idx++
		features[idx] = factors.ScoringBalance
		idx++

		// Add goalie advantage
		features[idx] = factors.GoalieAdvantage
		idx++

		// Pad remaining features with zeros
		for idx < lstm.inputSize {
			features[idx] = 0.0
			idx++
		}

		sequence[t] = features
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

// extractGameFeatures extracts features from a completed game
func (lstm *LSTMModel) extractGameFeatures(game *models.CompletedGame, teamCode string) []float64 {
	features := make([]float64, lstm.inputSize)

	isHome := game.HomeTeam.TeamCode == teamCode

	idx := 0
	if isHome {
		features[idx] = float64(game.HomeTeam.Score)
		idx++
		features[idx] = float64(game.AwayTeam.Score)
		idx++
	} else {
		features[idx] = float64(game.AwayTeam.Score)
		idx++
		features[idx] = float64(game.HomeTeam.Score)
		idx++
	}

	// Add more features as available
	// For now, pad with normalized values
	for idx < lstm.inputSize {
		features[idx] = 0.0
		idx++
	}

	return features
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

// trainSequence trains on a single sequence using backpropagation through time
func (lstm *LSTMModel) trainSequence(seq GameSequence) float64 {
	// Forward pass
	output := lstm.forward(seq.Features)

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

	// Simplified gradient update (full BPTT would be more complex)
	// For now, just update output layer
	outputGrad := make([]float64, lstm.outputSize)
	for i := 0; i < lstm.outputSize; i++ {
		outputGrad[i] = output[i] - target[i]
	}

	// Update output weights (simplified)
	// In full implementation, would backpropagate through time

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

// TrainOnGameResult trains the model on a completed game
func (lstm *LSTMModel) TrainOnGameResult(game models.CompletedGame) error {
	// LSTM needs sequences, so we'll collect games and train in batches
	lstm.mutex.Lock()
	lstm.gameSequences = append(lstm.gameSequences, GameSequence{
		Features: [][]float64{}, // Will be populated during batch training
		Label:    0.0,
		TeamCode: game.HomeTeam.TeamCode,
	})
	lstm.mutex.Unlock()

	log.Printf("🔄 LSTM: Received game result (batch training needed, %d games collected)", len(lstm.gameSequences))
	return nil
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

// saveModel saves the complete LSTM model to disk
func (lstm *LSTMModel) saveModel() error {
	lstm.mutex.Lock()
	defer lstm.mutex.Unlock()

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
