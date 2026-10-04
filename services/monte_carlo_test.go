package services

import (
	"fmt"
	"math"
	"testing"

	"github.com/jaredshillingburg/go_uhc/models"
)

// symmetricFactors builds two sides that are identical in every respect, with
// no home advantage, so the only fair answer is a coin flip.
func symmetricFactors() (*models.PredictionFactors, *models.PredictionFactors) {
	build := func(code string) *models.PredictionFactors {
		return &models.PredictionFactors{
			TeamCode:      code,
			WinPercentage: 0.5,
			GoalsFor:      3.0,
			GoalsAgainst:  3.0,
			RecentForm:    0.0,
			HomeAdvantage: 0.0,
		}
	}
	return build("HOM"), build("AWY")
}

// Drawn simulations used to go to the away team by default, which pushed the
// home probability below a coin flip for sides that were in fact identical.
func TestEvenlyMatchedTeamsAreACoinFlip(t *testing.T) {
	model := NewMonteCarloModel()
	home, away := symmetricFactors()

	result, err := model.Predict(home, away)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}

	if math.Abs(result.WinProbability-0.5) > 0.05 {
		t.Fatalf("WinProbability = %.4f for identical teams, want ~0.50", result.WinProbability)
	}
}

// Quantifies the bias the old rule carried, and guards the assumption behind
// the fix: drawn simulations are common enough that discarding them matters.
func TestDrawnSimulationsAreCommonEnoughToMatter(t *testing.T) {
	model := NewMonteCarloModel()
	home, away := symmetricFactors()

	draws, homeAhead := 0, 0
	const runs = 5000
	for i := 0; i < runs; i++ {
		homeGoals, awayGoals := model.simulateGame(home, away)
		switch {
		case homeGoals > awayGoals:
			homeAhead++
		case homeGoals == awayGoals:
			draws++
		}
	}

	drawRate := float64(draws) / runs
	if drawRate < 0.05 {
		t.Skipf("draws too rare (%.1f%%) for this to be a meaningful guard", 100*drawRate)
	}

	// What the old rule published: draws silently handed to the away side.
	oldProbability := float64(homeAhead) / runs
	if oldProbability > 0.45 {
		t.Errorf("expected the discarded draws to visibly depress the old estimate, got %.3f", oldProbability)
	}
	t.Logf("draws %.1f%% of simulations; old rule gave identical teams %.3f, corrected gives ~0.5",
		100*drawRate, oldProbability)
}

func TestWinProbabilityIsNeverCertain(t *testing.T) {
	cases := []struct{ wins, sims int }{
		{0, 2000},    // every simulation lost
		{2000, 2000}, // every simulation won
		{1000, 2000},
	}
	for _, c := range cases {
		got := monteCarloWinProbability(c.wins, c.sims)
		if got <= 0 || got >= 1 {
			t.Errorf("monteCarloWinProbability(%d, %d) = %v, want strictly inside (0,1)", c.wins, c.sims, got)
		}
	}

	// A sweep must stay inside the open interval and stay ordered.
	previous := -1.0
	for wins := 0; wins <= 2000; wins += 100 {
		got := monteCarloWinProbability(wins, 2000)
		if got <= 0 || got >= 1 {
			t.Fatalf("probability escaped (0,1) at wins=%d: %v", wins, got)
		}
		if got <= previous {
			t.Fatalf("probability not increasing at wins=%d: %v after %v", wins, got, previous)
		}
		previous = got
	}

	if got := monteCarloWinProbability(0, 0); got != 0.5 {
		t.Errorf("no simulations should give 0.5, got %v", got)
	}
}

// The NHL has no ties, so a predicted scoreline must not be level.
func TestPredictedScorelineIsNeverATie(t *testing.T) {
	cases := []struct {
		avgHome, avgAway, homeWinProb float64
		wantHome, wantAway            int
	}{
		{3.4, 3.4, 0.60, 4, 3}, // level after rounding, home favoured
		{3.4, 3.4, 0.40, 3, 4}, // level after rounding, away favoured
		{2.2, 2.2, 0.50, 3, 2}, // exactly even goes to the home side
		{4.4, 2.3, 0.70, 4, 2}, // already decisive, left alone
		{2.3, 4.4, 0.30, 2, 4},
	}
	for _, c := range cases {
		home, away := monteCarloScoreline(c.avgHome, c.avgAway, c.homeWinProb)
		if home == away {
			t.Errorf("scoreline(%v,%v,%v) = %d-%d, a tie", c.avgHome, c.avgAway, c.homeWinProb, home, away)
		}
		if home != c.wantHome || away != c.wantAway {
			t.Errorf("scoreline(%v,%v,%v) = %d-%d, want %d-%d",
				c.avgHome, c.avgAway, c.homeWinProb, home, away, c.wantHome, c.wantAway)
		}
		// The favoured side must be the one with more goals.
		if (c.homeWinProb >= 0.5) != (home > away) {
			t.Errorf("scoreline %d-%d contradicts homeWinProb %v", home, away, c.homeWinProb)
		}
	}
}

// Confidence was pinned near 1.0 for every game: all 487 settled predictions
// reported at least 0.70, so it could not tell a sure thing from a toss-up.
func TestConfidenceVariesWithTheForecast(t *testing.T) {
	model := NewMonteCarloModel()

	coinFlip := model.calculateMonteCarloConfidence(0.50)
	slight := model.calculateMonteCarloConfidence(0.60)
	strong := model.calculateMonteCarloConfidence(0.90)

	if coinFlip > 0.01 {
		t.Errorf("a coin flip should carry no confidence, got %v", coinFlip)
	}
	if !(coinFlip < slight && slight < strong) {
		t.Errorf("confidence should rise with the margin: %v, %v, %v", coinFlip, slight, strong)
	}
	if strong > 1.0 || coinFlip < 0.0 {
		t.Errorf("confidence left 0-1: %v, %v", coinFlip, strong)
	}

	// Symmetric about a coin flip: being sure of the away side is just as
	// confident as being sure of the home side.
	if math.Abs(model.calculateMonteCarloConfidence(0.20)-model.calculateMonteCarloConfidence(0.80)) > 1e-9 {
		t.Errorf("confidence is not symmetric about 0.5")
	}
}

// The old formula improved as the simulation count rose, so the model could
// talk itself into certainty without learning anything about hockey.
func TestConfidenceDoesNotDependOnSimulationCount(t *testing.T) {
	few := &MonteCarloModel{simulations: 100}
	many := &MonteCarloModel{simulations: 100000}

	if few.calculateMonteCarloConfidence(0.55) != many.calculateMonteCarloConfidence(0.55) {
		t.Errorf("confidence changed with simulation count alone")
	}
}

// Goals are close to Poisson in hockey, so the sampler's spread should be
// about sqrt(mean). The old truncated-normal draw used a fixed 0.8, less than
// half of that, which is what let the simulator separate teams unrealistically.
func TestGoalSamplerHasHockeyLikeSpread(t *testing.T) {
	const lambda = 3.2
	const draws = 200000

	sum, sumSq := 0.0, 0.0
	for i := 0; i < draws; i++ {
		g := float64(samplePoisson(lambda))
		sum += g
		sumSq += g * g
	}
	mean := sum / draws
	variance := sumSq/draws - mean*mean
	stdDev := math.Sqrt(variance)

	if math.Abs(mean-lambda) > 0.05 {
		t.Errorf("mean = %.3f, want ~%.1f", mean, lambda)
	}
	// Poisson variance equals its mean, so the standard deviation is sqrt(λ).
	if want := math.Sqrt(lambda); math.Abs(stdDev-want) > 0.1 {
		t.Errorf("stdDev = %.3f, want ~%.3f (old sampler used 0.8)", stdDev, want)
	}
	if stdDev < 1.5 {
		t.Errorf("spread %.3f is too tight to be hockey", stdDev)
	}
}

func TestGoalSamplerStaysInRange(t *testing.T) {
	for _, lambda := range []float64{0, -1, 0.5, 3.2, 50} {
		for i := 0; i < 2000; i++ {
			g := drawGoals(lambda)
			if g < 0 || g > maxSimulatedGoals {
				t.Fatalf("drawGoals(%v) = %d, outside 0-%d", lambda, g, maxSimulatedGoals)
			}
		}
	}
	if got := samplePoisson(0); got != 0 {
		t.Errorf("samplePoisson(0) = %d, want 0", got)
	}
}

// A mismatch should favour the better side without claiming near-certainty.
// The model published a 0.25% chance for one team in a real game, which no
// NHL matchup justifies.
func TestLopsidedMatchupIsNotNearCertain(t *testing.T) {
	model := NewMonteCarloModel()
	strong := &models.PredictionFactors{TeamCode: "STR", WinPercentage: 0.75, GoalsFor: 4.2, GoalsAgainst: 2.3, RecentForm: 0.3}
	weak := &models.PredictionFactors{TeamCode: "WEK", WinPercentage: 0.25, GoalsFor: 2.3, GoalsAgainst: 4.2, RecentForm: -0.3}

	result, err := model.Predict(strong, weak)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}

	if result.WinProbability <= 0.5 {
		t.Errorf("the stronger home side should be favoured, got %.4f", result.WinProbability)
	}
	// Even a thorough mismatch is not a lock in a sport this high-variance.
	if result.WinProbability > 0.95 {
		t.Errorf("WinProbability = %.4f, too close to certain for one hockey game", result.WinProbability)
	}
	t.Logf("lopsided matchup -> %.4f", result.WinProbability)
}

func TestBreakTiedScoreline(t *testing.T) {
	cases := []struct {
		in          string
		homeWinProb float64
		want        string
	}{
		{"3-3", 0.60, "4-3"},
		{"3-3", 0.40, "3-4"},
		{"2-2", 0.50, "3-2"},
		{"4-2", 0.70, "4-2"}, // already decided, untouched
		{"2-4", 0.30, "2-4"},
		{"", 0.50, ""},            // unparseable, left alone
		{"not-a-score", 0.5, "not-a-score"},
	}
	for _, c := range cases {
		if got := breakTiedScoreline(c.in, c.homeWinProb); got != c.want {
			t.Errorf("breakTiedScoreline(%q, %v) = %q, want %q", c.in, c.homeWinProb, got, c.want)
		}
	}
}

func TestPredictProducesUsableOutput(t *testing.T) {
	model := NewMonteCarloModel()
	home, away := symmetricFactors()

	result, err := model.Predict(home, away)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}

	if result.WinProbability <= 0 || result.WinProbability >= 1 {
		t.Errorf("WinProbability = %v, want strictly inside (0,1)", result.WinProbability)
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence = %v, want 0-1", result.Confidence)
	}

	// Identical teams are a toss-up, so the model should say so rather than
	// reporting the near-certainty the old formula produced.
	if result.Confidence > 0.25 {
		t.Errorf("Confidence = %v for identical teams, want a low value", result.Confidence)
	}

	var h, a int
	if _, err := fmt.Sscanf(result.PredictedScore, "%d-%d", &h, &a); err != nil {
		t.Fatalf("PredictedScore %q not parseable: %v", result.PredictedScore, err)
	}
	if h == a {
		t.Errorf("PredictedScore = %q, a tie", result.PredictedScore)
	}
}
