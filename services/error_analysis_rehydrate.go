package services

import (
	"fmt"
	"sort"

	"github.com/jaredshillingburg/go_uhc/models"
)

// The accuracy endpoints reported zeros for the entire life of the service.
// The cause was not a loading bug: ErrorAnalysisService.RecordPrediction and
// UpdatePredictionWithResult had no callers anywhere, so the accuracy_*.json
// files loadRecords globs for were never written and data/accuracy sat empty
// while data/predictions held hundreds of settled games.
//
// Rather than add writes to a second store that would then have to be kept in
// step with the first, the accuracy records are derived from the prediction
// store, which is already the thing the result pipeline updates. There is one
// place a prediction becomes settled -- PredictionStorageService.UpdateWithResult
// -- so that is the only place the two can disagree, and it feeds this file.

// rehydrateFromPredictionStorage rebuilds accuracy records from every settled
// prediction on disk and returns how many it added. Records already held in
// memory win, so this composes with whatever loadRecords found.
func (eas *ErrorAnalysisService) rehydrateFromPredictionStorage() int {
	storage := GetPredictionStorageService()
	if storage == nil {
		return 0
	}

	// Read the store before taking eas.mu. Settling a prediction takes the
	// storage lock and then this one, so acquiring them in that order here
	// too is what keeps the two paths from deadlocking against each other.
	stored, err := storage.GetAllPredictions()
	if err != nil {
		fmt.Printf("⚠️ Could not read prediction store for accuracy rehydration: %v\n", err)
		return 0
	}

	eas.mu.Lock()
	defer eas.mu.Unlock()

	seen := make(map[int]bool, len(eas.records))
	for _, record := range eas.records {
		seen[record.GameID] = true
	}

	added := 0
	for _, prediction := range stored {
		if record := accuracyRecordFor(eas, prediction, seen); record != nil {
			eas.records = append(eas.records, record)
			seen[record.GameID] = true
			added++
		}
	}

	if added > 0 {
		sort.Slice(eas.records, func(i, j int) bool {
			return eas.records[i].GameDate.Before(eas.records[j].GameDate)
		})
	}

	return added
}

// IngestSettledPrediction folds a prediction that has just received its result
// into the accuracy records, so the endpoints reflect last night's games
// without waiting for a restart.
func (eas *ErrorAnalysisService) IngestSettledPrediction(prediction *StoredPrediction) {
	eas.mu.Lock()
	defer eas.mu.Unlock()

	seen := make(map[int]bool, len(eas.records))
	for _, record := range eas.records {
		seen[record.GameID] = true
	}

	record := accuracyRecordFor(eas, prediction, seen)
	if record == nil {
		return
	}

	eas.records = append(eas.records, record)
	sort.Slice(eas.records, func(i, j int) bool {
		return eas.records[i].GameDate.Before(eas.records[j].GameDate)
	})

	eas.recalculateSummary()
	eas.analyzeErrorPatterns()
	if err := eas.saveSummary(); err != nil {
		fmt.Printf("⚠️ Could not save accuracy summary: %v\n", err)
	}
}

// accuracyRecordFor converts a settled prediction into a scored accuracy
// record, or returns nil if it is unsettled, already known, or too incomplete
// to score. Callers must hold eas.mu.
func accuracyRecordFor(eas *ErrorAnalysisService, prediction *StoredPrediction, seen map[int]bool) *models.PredictionAccuracyRecord {
	if prediction == nil || prediction.ActualResult == nil || seen[prediction.GameID] {
		return nil
	}

	result := prediction.ActualResult

	// No NHL game ends level, so equal scores mean the result is not actually
	// final and the winner cannot be read off the score.
	if result.HomeScore == result.AwayScore {
		return nil
	}

	record := buildAccuracyRecord(prediction)
	if record == nil {
		return nil
	}

	eas.finalizeRecord(record, result.HomeScore, result.AwayScore, gameTypeFromResult(result))
	return record
}

// buildAccuracyRecord copies across what was known at prediction time. The
// result-dependent fields are left to finalizeRecord.
func buildAccuracyRecord(prediction *StoredPrediction) *models.PredictionAccuracyRecord {
	outcome := prediction.Prediction.Prediction

	homeTeam := prediction.HomeTeam
	if homeTeam == "" {
		homeTeam = prediction.Prediction.HomeTeam.Code
	}
	awayTeam := prediction.AwayTeam
	if awayTeam == "" {
		awayTeam = prediction.Prediction.AwayTeam.Code
	}
	if homeTeam == "" || awayTeam == "" || outcome.Winner == "" {
		return nil
	}

	return &models.PredictionAccuracyRecord{
		GameID:   prediction.GameID,
		GameDate: prediction.GameDate,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,

		PredictedWinner: outcome.Winner,
		// WinProbability is the home team's, not the winner's.
		PredictedHomeWinProb: outcome.WinProbability,
		PredictedAwayWinProb: 1.0 - outcome.WinProbability,
		// Match the live path, which records how sure the pick was rather
		// than the separate ensemble confidence field.
		Confidence: outcome.WinnerProbability(),

		ModelPredictions: modelPredictionsFromResults(outcome.ModelResults, homeTeam, awayTeam),
		PredictionTime:   prediction.PredictedAt,
		Season:           GetCurrentSeason(),
	}
}

// modelPredictionsFromResults reshapes the per-model slice stored with a
// prediction into the map the accuracy record keeps.
func modelPredictionsFromResults(results []models.ModelResult, homeTeam, awayTeam string) map[string]*models.ModelPredictionResult {
	if len(results) == 0 {
		return nil
	}

	predictions := make(map[string]*models.ModelPredictionResult, len(results))
	for _, result := range results {
		if result.ModelName == "" {
			continue
		}

		// Each model reports only the home win probability, so its pick is
		// whichever side that favours.
		winner := homeTeam
		if result.WinProbability < 0.5 {
			winner = awayTeam
		}

		predictions[result.ModelName] = &models.ModelPredictionResult{
			ModelName:       result.ModelName,
			PredictedWinner: winner,
			HomeWinProb:     result.WinProbability,
			AwayWinProb:     1.0 - result.WinProbability,
			Confidence:      result.Confidence,
			ModelWeight:     result.Weight,
		}
	}

	return predictions
}

func gameTypeFromResult(result *models.GameResult) string {
	switch {
	case result.IsShootout:
		return "shootout"
	case result.IsOvertime:
		return "overtime"
	default:
		return "regulation"
	}
}
