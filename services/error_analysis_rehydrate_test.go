package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// withPredictionStore points the prediction storage singleton at a temporary
// directory holding the given predictions, and restores it afterwards.
func withPredictionStore(t *testing.T, predictions ...*StoredPrediction) {
	t.Helper()

	dir := t.TempDir()
	for _, prediction := range predictions {
		data, err := json.Marshal(prediction)
		if err != nil {
			t.Fatalf("marshal prediction %d: %v", prediction.GameID, err)
		}
		name := filepath.Join(dir, "game_"+itoa(prediction.GameID)+".json")
		if err := os.WriteFile(name, data, 0644); err != nil {
			t.Fatalf("write prediction %d: %v", prediction.GameID, err)
		}
	}

	previous := predictionStorageService
	t.Cleanup(func() { predictionStorageService = previous })
	predictionStorageService = &PredictionStorageService{
		dataDir:     dir,
		allCacheTTL: time.Second,
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// newErrorAnalysisForTest builds a service backed by a temp dir, bypassing the
// singleton so tests do not fight over it.
func newErrorAnalysisForTest(t *testing.T) *ErrorAnalysisService {
	t.Helper()
	return &ErrorAnalysisService{
		dataDir:           t.TempDir(),
		records:           make([]*models.PredictionAccuracyRecord, 0),
		featureImportance: make(map[string]*models.FeatureImportance),
		errorPatterns:     make([]*models.ErrorPattern, 0),
	}
}

// settledPrediction builds a prediction with a result attached. homeWinProb is
// the home team's predicted probability, matching PredictionResult.
func settledPrediction(gameID int, home, away string, homeWinProb float64, homeScore, awayScore int) *StoredPrediction {
	winner := home
	if homeWinProb < 0.5 {
		winner = away
	}
	actualWinner := home
	if awayScore > homeScore {
		actualWinner = away
	}

	return &StoredPrediction{
		GameID:      gameID,
		GameDate:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, gameID),
		HomeTeam:    home,
		AwayTeam:    away,
		PredictedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Prediction: models.GamePrediction{
			GameID:   gameID,
			HomeTeam: models.PredictionTeam{Code: home},
			AwayTeam: models.PredictionTeam{Code: away},
			Prediction: models.PredictionResult{
				Winner:         winner,
				WinProbability: homeWinProb,
				ModelResults: []models.ModelResult{
					{ModelName: "Monte Carlo", WinProbability: homeWinProb, Confidence: 0.6, Weight: 0.5},
					{ModelName: "Elo", WinProbability: 1 - homeWinProb, Confidence: 0.6, Weight: 0.5},
				},
			},
		},
		ActualResult: &models.GameResult{
			GameID:      gameID,
			HomeTeam:    home,
			AwayTeam:    away,
			HomeScore:   homeScore,
			AwayScore:   awayScore,
			WinningTeam: actualWinner,
			GameState:   "FINAL",
		},
	}
}

// This is the bug the endpoints showed: hundreds of settled predictions on
// disk and a summary of zero.
func TestSummaryIsNotZeroWhenSettledPredictionsExist(t *testing.T) {
	withPredictionStore(t,
		settledPrediction(1, "NYR", "UTA", 0.70, 4, 2), // correct
		settledPrediction(2, "BOS", "TOR", 0.65, 1, 3), // wrong
		settledPrediction(3, "COL", "VGK", 0.55, 5, 1), // correct
	)

	eas := newErrorAnalysisForTest(t)
	if added := eas.rehydrateFromPredictionStorage(); added != 3 {
		t.Fatalf("expected to recover 3 records, got %d", added)
	}
	eas.recalculateSummary()

	summary := eas.GetSummary()
	if summary.TotalPredictions != 3 {
		t.Fatalf("TotalPredictions = %d, want 3", summary.TotalPredictions)
	}
	if summary.CorrectPredictions != 2 {
		t.Fatalf("CorrectPredictions = %d, want 2", summary.CorrectPredictions)
	}
	if summary.OverallAccuracy <= 0 {
		t.Fatalf("OverallAccuracy = %v, want a measured value", summary.OverallAccuracy)
	}
}

func TestUnsettledPredictionsAreNotCounted(t *testing.T) {
	pending := settledPrediction(10, "NYR", "UTA", 0.70, 0, 0)
	pending.ActualResult = nil

	// A game recorded as final but level cannot have its winner read off the
	// score, so it is not scoreable either.
	level := settledPrediction(11, "BOS", "TOR", 0.60, 2, 2)

	withPredictionStore(t, pending, level, settledPrediction(12, "COL", "VGK", 0.55, 3, 1))

	eas := newErrorAnalysisForTest(t)
	if added := eas.rehydrateFromPredictionStorage(); added != 1 {
		t.Fatalf("expected only the one scoreable game, got %d", added)
	}
	if eas.records[0].GameID != 12 {
		t.Fatalf("recovered game %d, want 12", eas.records[0].GameID)
	}
}

func TestRehydrationDoesNotDuplicateKnownGames(t *testing.T) {
	prediction := settledPrediction(20, "NYR", "UTA", 0.70, 4, 2)
	withPredictionStore(t, prediction, settledPrediction(21, "BOS", "TOR", 0.60, 1, 2))

	eas := newErrorAnalysisForTest(t)
	eas.records = append(eas.records, &models.PredictionAccuracyRecord{GameID: 20})

	if added := eas.rehydrateFromPredictionStorage(); added != 1 {
		t.Fatalf("expected 1 new record alongside the known one, got %d", added)
	}
	if len(eas.records) != 2 {
		t.Fatalf("records = %d, want 2", len(eas.records))
	}

	// And a second pass must be a no-op.
	if added := eas.rehydrateFromPredictionStorage(); added != 0 {
		t.Fatalf("second rehydration added %d records, want 0", added)
	}
}

// Rehydrated records have to be scored the same way as live ones, otherwise
// measured accuracy would shift depending on whether the process restarted.
func TestRehydratedRecordMatchesLiveScoring(t *testing.T) {
	prediction := settledPrediction(30, "NYR", "UTA", 0.30, 2, 5)
	withPredictionStore(t, prediction)

	eas := newErrorAnalysisForTest(t)
	if added := eas.rehydrateFromPredictionStorage(); added != 1 {
		t.Fatalf("expected 1 record, got %d", added)
	}
	got := eas.records[0]

	// Score the same prediction through the shared finalizer directly.
	want := buildAccuracyRecord(prediction)
	eas.finalizeRecord(want, 2, 5, "regulation")

	if got.ActualWinner != want.ActualWinner || got.IsCorrect != want.IsCorrect {
		t.Fatalf("winner/correctness differ: got %+v want %+v", got, want)
	}
	if got.PredictionError != want.PredictionError {
		t.Fatalf("PredictionError = %v, want %v", got.PredictionError, want.PredictionError)
	}
	if got.CalibrationScore != want.CalibrationScore {
		t.Fatalf("CalibrationScore = %v, want %v", got.CalibrationScore, want.CalibrationScore)
	}

	// UTA was the pick at 0.30 home probability and UTA won.
	if !got.IsCorrect {
		t.Fatalf("expected the away pick to be scored correct")
	}
	if got.IsUpset {
		t.Fatalf("picking the favourite and seeing it win is not an upset")
	}
}

func TestModelPredictionsAreScoredPerModel(t *testing.T) {
	// Home probability 0.70, so Monte Carlo picks the home side and Elo (at
	// 0.30) picks the away side. The home team then wins.
	withPredictionStore(t, settledPrediction(40, "NYR", "UTA", 0.70, 4, 2))

	eas := newErrorAnalysisForTest(t)
	eas.rehydrateFromPredictionStorage()

	models := eas.records[0].ModelPredictions
	if len(models) != 2 {
		t.Fatalf("expected 2 model results, got %d", len(models))
	}
	if !models["Monte Carlo"].IsCorrect {
		t.Errorf("Monte Carlo picked the home winner and should be correct")
	}
	if models["Elo"].IsCorrect {
		t.Errorf("Elo picked the away side and should be incorrect")
	}
}

func TestIngestSettledPredictionUpdatesSummaryWithoutRestart(t *testing.T) {
	eas := newErrorAnalysisForTest(t)
	eas.recalculateSummary()
	if eas.GetSummary().TotalPredictions != 0 {
		t.Fatalf("expected an empty summary to start")
	}

	eas.IngestSettledPrediction(settledPrediction(50, "NYR", "UTA", 0.70, 4, 2))

	summary := eas.GetSummary()
	if summary.TotalPredictions != 1 || summary.CorrectPredictions != 1 {
		t.Fatalf("summary not updated: %+v", summary)
	}

	// Settling the same game twice must not double-count it.
	eas.IngestSettledPrediction(settledPrediction(50, "NYR", "UTA", 0.70, 4, 2))
	if got := eas.GetSummary().TotalPredictions; got != 1 {
		t.Fatalf("TotalPredictions = %d after re-ingest, want 1", got)
	}
}

func TestIngestIgnoresUnsettledPrediction(t *testing.T) {
	eas := newErrorAnalysisForTest(t)

	pending := settledPrediction(60, "NYR", "UTA", 0.70, 0, 0)
	pending.ActualResult = nil
	eas.IngestSettledPrediction(pending)
	eas.IngestSettledPrediction(nil)

	if len(eas.records) != 0 {
		t.Fatalf("records = %d, want 0", len(eas.records))
	}
}
