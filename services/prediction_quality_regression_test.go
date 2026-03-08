package services

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaredshillingburg/go_uhc/models"
)

func TestNormalizeWinningScore_HomeWinner(t *testing.T) {
	homeGoals, awayGoals := normalizeWinningScore(3, 3, "UTA", "UTA", "CHI")

	if homeGoals <= awayGoals {
		t.Fatalf("expected home team to have a winning score, got %d-%d", homeGoals, awayGoals)
	}
}

func TestNormalizeWinningScore_AwayWinner(t *testing.T) {
	homeGoals, awayGoals := normalizeWinningScore(4, 2, "CHI", "UTA", "CHI")

	if awayGoals <= homeGoals {
		t.Fatalf("expected away team to have a winning score, got %d-%d", homeGoals, awayGoals)
	}
}

func TestGenerateDegradedPrediction_NeverReturnsTieForWinner(t *testing.T) {
	ps := &PredictionService{}
	homeFactors := &models.PredictionFactors{
		TeamCode:      "UTA",
		WinPercentage: 0.50,
		GoalsFor:      3.0,
		GoalsAgainst:  3.0,
		RestDays:      1,
	}
	awayFactors := &models.PredictionFactors{
		TeamCode:      "CHI",
		WinPercentage: 0.50,
		GoalsFor:      3.0,
		GoalsAgainst:  3.0,
		RestDays:      1,
	}

	result := ps.generateDegradedPrediction(homeFactors, awayFactors, nil)
	if result == nil {
		t.Fatal("expected degraded prediction result")
	}

	parts := strings.Split(result.PredictedScore, "-")
	if len(parts) != 2 {
		t.Fatalf("expected score in home-away format, got %q", result.PredictedScore)
	}

	if parts[0] == parts[1] {
		t.Fatalf("expected non-tie score for declared winner %s, got %q", result.Winner, result.PredictedScore)
	}
}

func TestClubGoalieStats_UnmarshalCurrentNHLAPIFields(t *testing.T) {
	payload := []byte(`{
		"goalies": [
			{
				"playerId": 8478872,
				"firstName": {"default": "Karel"},
				"lastName": {"default": "Vejmelka"},
				"gamesPlayed": 49,
				"wins": 30,
				"losses": 16,
				"overtimeLosses": 2,
				"savePercentage": 0.900238,
				"goalsAgainstAverage": 2.640653,
				"shutouts": 1,
				"goalsAgainst": 129,
				"shotsAgainst": 1294,
				"saves": 1165
			}
		]
	}`)

	var response models.ClubStatsResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(response.Goalies) != 1 {
		t.Fatalf("expected 1 goalie, got %d", len(response.Goalies))
	}

	goalie := response.Goalies[0]
	if goalie.SavePct != 0.900238 {
		t.Fatalf("expected save percentage to parse, got %f", goalie.SavePct)
	}
	if goalie.GoalsAgainstAvg != 2.640653 {
		t.Fatalf("expected GAA to parse, got %f", goalie.GoalsAgainstAvg)
	}
}
