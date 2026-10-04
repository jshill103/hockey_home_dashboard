package services

import (
	"fmt"
	"math"
	"testing"

	"github.com/jaredshillingburg/go_uhc/models"
)

func teamRates(code string, goalsFor, goalsAgainst float64) *models.PredictionFactors {
	return &models.PredictionFactors{TeamCode: code, GoalsFor: goalsFor, GoalsAgainst: goalsAgainst}
}

func parseScore(t *testing.T, score string) (int, int) {
	t.Helper()
	var home, away int
	if _, err := fmt.Sscanf(score, "%d-%d", &home, &away); err != nil {
		t.Fatalf("score %q not parseable: %v", score, err)
	}
	return home, away
}

// The defect these helpers replaced: the score ignored who was playing, so
// LSTM returned 3-2 for all 561 of its stored predictions.
func TestScorelineRespondsToTheTeams(t *testing.T) {
	highScoring := scorelineFromExpectedGoals(0.55, teamRates("RUN", 4.6, 4.4), teamRates("GUN", 4.5, 4.5))
	lowScoring := scorelineFromExpectedGoals(0.55, teamRates("DEF", 2.0, 1.8), teamRates("TRP", 1.9, 1.9))

	if highScoring == lowScoring {
		t.Fatalf("a run-and-gun game and a defensive one both gave %q", highScoring)
	}

	hf, ha := parseScore(t, highScoring)
	lf, la := parseScore(t, lowScoring)
	if hf+ha <= lf+la {
		t.Errorf("high-scoring matchup totalled %d, low-scoring %d; expected more goals in the former", hf+ha, lf+la)
	}
}

func TestScorelineIsNeverATie(t *testing.T) {
	for _, prob := range []float64{0.0, 0.2, 0.45, 0.5, 0.55, 0.8, 1.0} {
		for _, rate := range []float64{1.0, 2.5, 3.1, 4.0, 6.0} {
			score := scorelineFromExpectedGoals(prob, teamRates("A", rate, rate), teamRates("B", rate, rate))
			home, away := parseScore(t, score)
			if home == away {
				t.Errorf("prob %v rate %v gave the tie %q", prob, rate, score)
			}
			if home < 0 || away < 0 || home > maxSimulatedGoals || away > maxSimulatedGoals {
				t.Errorf("score %q outside 0-%d", score, maxSimulatedGoals)
			}
		}
	}
}

// The scoreline must name the same winner the model picked, including when
// the two teams' scoring rates point the other way. Random Forest shipped a
// 78% home win next to a 2-3 scoreline because only exact ties were broken.
func TestScorelineAgreesWithThePick(t *testing.T) {
	matchups := []struct {
		name       string
		home, away *models.PredictionFactors
	}{
		{"evenly matched", teamRates("HOM", 3.2, 3.0), teamRates("AWY", 3.1, 3.1)},
		{"rates favour away", teamRates("HOM", 1.9, 4.1), teamRates("AWY", 4.3, 1.8)},
		{"rates favour home", teamRates("HOM", 4.4, 1.9), teamRates("AWY", 1.8, 4.2)},
		{"low scoring", teamRates("HOM", 1.5, 1.4), teamRates("AWY", 1.4, 1.5)},
	}

	for _, m := range matchups {
		for _, prob := range []float64{0.05, 0.3, 0.49, 0.5, 0.51, 0.7, 0.95} {
			score := scorelineFromExpectedGoals(prob, m.home, m.away)
			h, a := parseScore(t, score)
			if (prob >= 0.5) != (h > a) {
				t.Errorf("%s at prob %v gave %q, which names the other side", m.name, prob, score)
			}
		}
	}
}

func TestScorelineSurvivesMissingOrOddData(t *testing.T) {
	cases := []struct {
		name        string
		home, away  *models.PredictionFactors
	}{
		{"both nil", nil, nil},
		{"zeroed", teamRates("A", 0, 0), teamRates("B", 0, 0)},
		{"negative", teamRates("A", -3, -2), teamRates("B", -1, -1)},
		{"season totals rather than per-game", teamRates("A", 260, 240), teamRates("B", 250, 250)},
		{"absurd", teamRates("A", 99, 99), teamRates("B", 99, 99)},
	}
	for _, c := range cases {
		score := scorelineFromExpectedGoals(0.6, c.home, c.away)
		h, a := parseScore(t, score)
		if h == a {
			t.Errorf("%s: gave the tie %q", c.name, score)
		}
		if h < 0 || a < 0 || h > maxSimulatedGoals || a > maxSimulatedGoals {
			t.Errorf("%s: %q outside 0-%d", c.name, score, maxSimulatedGoals)
		}
		if h+a == 0 {
			t.Errorf("%s: predicted a scoreless game", c.name)
		}
	}

	// A season total should be read as a sane per-game rate, not as a rout.
	score := scorelineFromExpectedGoals(0.5, teamRates("A", 260, 240), teamRates("B", 250, 250))
	h, a := parseScore(t, score)
	if h+a > 10 {
		t.Errorf("season totals leaked into the scoreline: %q", score)
	}
}

func TestPossessionShareNormalisesBothScales(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{0.52, 0.52},  // already a share
		{52.0, 0.52},  // percentage
		{100.0, 1.0},  // percentage ceiling
		{0, neutralPossessionShare},
		{-5, neutralPossessionShare},
	}
	for _, c := range cases {
		if got := possessionShare(c.in); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("possessionShare(%v) = %v, want %v", c.in, got, c.want)
		}
	}
	if got := possessionShare(math.NaN()); got != neutralPossessionShare {
		t.Errorf("NaN should fall back to neutral, got %v", got)
	}
}
