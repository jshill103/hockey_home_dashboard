package services

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// Synthetic-recovery tests for the Dixon-Coles fit. Goals are generated from
// known attack/defence/home-ice parameters, so the fit can be checked against
// ground truth rather than against itself.

const (
	syntheticTeamCount   = 20
	syntheticLeagueAvg   = 3.05
	syntheticHomeAdv     = 1.09
	syntheticRoundsEach  = 3 // home-and-away meetings per opponent pair
	syntheticSeasonDays  = 150
	syntheticRandomSeed  = 20261003
	syntheticGoalsPerMax = 15
)

// syntheticLeague holds the ground-truth parameters a synthetic season was
// generated from.
type syntheticLeague struct {
	teams []string
	off   map[string]float64
	def   map[string]float64
}

func newSyntheticLeague() syntheticLeague {
	sl := syntheticLeague{
		teams: make([]string, 0, syntheticTeamCount),
		off:   make(map[string]float64, syntheticTeamCount),
		def:   make(map[string]float64, syntheticTeamCount),
	}
	// Spread attack and defence strengths deterministically across a
	// realistic band rather than randomly, so the expected ordering is known.
	for i := 0; i < syntheticTeamCount; i++ {
		code := fmt.Sprintf("T%02d", i)
		frac := float64(i) / float64(syntheticTeamCount-1) // 0..1
		sl.teams = append(sl.teams, code)
		sl.off[code] = 0.75 + 0.50*frac // 0.75 .. 1.25
		sl.def[code] = 1.25 - 0.50*frac // 1.25 .. 0.75, anti-correlated
	}
	return sl
}

func (sl syntheticLeague) lambdas(home, away string) (float64, float64) {
	lambdaHome := sl.off[home] * sl.def[away] * syntheticLeagueAvg * syntheticHomeAdv
	lambdaAway := sl.off[away] * sl.def[home] * syntheticLeagueAvg
	return lambdaHome, lambdaAway
}

func samplePoissonRand(rng *rand.Rand, lambda float64) int {
	// Knuth's method; lambda here is always small enough for it.
	l := math.Exp(-lambda)
	k := 0
	p := 1.0
	for {
		p *= rng.Float64()
		if p <= l {
			return k
		}
		k++
		if k > syntheticGoalsPerMax {
			return syntheticGoalsPerMax
		}
	}
}

// generateSyntheticSeason plays a multi-round double round robin and samples
// each score from the true lambdas.
func (sl syntheticLeague) generateSeason(rng *rand.Rand) []models.CompletedGame {
	start := time.Now().AddDate(0, 0, -syntheticSeasonDays)
	var games []models.CompletedGame

	totalSlots := syntheticRoundsEach * syntheticTeamCount * (syntheticTeamCount - 1)
	slot := 0

	for round := 0; round < syntheticRoundsEach; round++ {
		for _, home := range sl.teams {
			for _, away := range sl.teams {
				if home == away {
					continue
				}
				lambdaHome, lambdaAway := sl.lambdas(home, away)
				homeScore := samplePoissonRand(rng, lambdaHome)
				awayScore := samplePoissonRand(rng, lambdaAway)

				// Spread games evenly across the season window.
				offset := float64(slot) / float64(totalSlots) * syntheticSeasonDays
				date := start.Add(time.Duration(offset * float64(24*time.Hour)))
				slot++

				winner := home
				if awayScore > homeScore {
					winner = away
				}

				games = append(games, models.CompletedGame{
					GameID:   slot,
					GameDate: date,
					GameType: 2,
					HomeTeam: models.TeamGameResult{TeamCode: home, Score: homeScore},
					AwayTeam: models.TeamGameResult{TeamCode: away, Score: awayScore},
					Winner:   winner,
				})
			}
		}
	}

	return games
}

// newTestPoissonModel builds a model that does not touch the shared singleton
// or the data directory used by the running service.
func newTestPoissonModel(t *testing.T) *PoissonRegressionModel {
	t.Helper()
	return &PoissonRegressionModel{
		leagueAvgGoalsPerGame: 3.1,
		teamOffensiveRates:    make(map[string]float64),
		teamDefensiveRates:    make(map[string]float64),
		rateHistory:           make(map[string][]RateRecord),
		confidenceTracking:    make(map[string]float64),
		homeAdvantage:         1.08,
		weight:                0.15,
		learningRate:          0.1,
		seasonDecayRate:       0.98,
		rand:                  rand.New(rand.NewSource(1)),
		dataDir:               t.TempDir(),
		rho:                   defaultDixonColesRho,
	}
}

func pearson(xs, ys []float64) float64 {
	n := float64(len(xs))
	var sx, sy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
	}
	mx, my := sx/n, sy/n

	var num, dx, dy float64
	for i := range xs {
		a, b := xs[i]-mx, ys[i]-my
		num += a * b
		dx += a * a
		dy += b * b
	}
	if dx == 0 || dy == 0 {
		return 0
	}
	return num / math.Sqrt(dx*dy)
}

func TestFitFromHistoryRecoversTeamStrengths(t *testing.T) {
	sl := newSyntheticLeague()
	rng := rand.New(rand.NewSource(syntheticRandomSeed))
	games := sl.generateSeason(rng)

	pr := newTestPoissonModel(t)
	if err := pr.FitFromHistory(games); err != nil {
		t.Fatalf("FitFromHistory returned error: %v", err)
	}

	trueOff := make([]float64, 0, len(sl.teams))
	gotOff := make([]float64, 0, len(sl.teams))
	trueDef := make([]float64, 0, len(sl.teams))
	gotDef := make([]float64, 0, len(sl.teams))

	for _, code := range sl.teams {
		o, d := pr.GetTeamRates(code)
		trueOff = append(trueOff, sl.off[code])
		gotOff = append(gotOff, o)
		trueDef = append(trueDef, sl.def[code])
		gotDef = append(gotDef, d)
	}

	// Shrinkage deliberately compresses the magnitude of fitted deviations, so
	// the thing to check is that the fit ranks and scales team strength with
	// the truth, not that it reproduces it exactly.
	if r := pearson(trueOff, gotOff); r < 0.90 {
		t.Errorf("fitted attack rates correlate with truth at only r=%.3f, want >= 0.90", r)
	}
	if r := pearson(trueDef, gotDef); r < 0.90 {
		t.Errorf("fitted defence rates correlate with truth at only r=%.3f, want >= 0.90", r)
	}
}

func TestFitFromHistoryRecoversHomeAdvantageAndLeagueLevel(t *testing.T) {
	sl := newSyntheticLeague()
	rng := rand.New(rand.NewSource(syntheticRandomSeed))
	games := sl.generateSeason(rng)

	pr := newTestPoissonModel(t)
	if err := pr.FitFromHistory(games); err != nil {
		t.Fatalf("FitFromHistory returned error: %v", err)
	}

	pr.mutex.RLock()
	gotHomeAdv := pr.homeAdvantage
	gotLeague := pr.leagueAvgGoalsPerGame
	pr.mutex.RUnlock()

	if math.Abs(gotHomeAdv-syntheticHomeAdv) > 0.03 {
		t.Errorf("home advantage = %.4f, want %.4f +/- 0.03", gotHomeAdv, syntheticHomeAdv)
	}

	// The attack and defence maps are both centred on 1.0 after the fit, so the
	// league average has to carry the true scoring level.
	if math.Abs(gotLeague-syntheticLeagueAvg) > 0.25 {
		t.Errorf("league average = %.3f, want %.3f +/- 0.25", gotLeague, syntheticLeagueAvg)
	}
}

// TestFitFromHistoryBeatsFlatRates is the test that actually justifies the
// change: the fitted parameters must explain held-out games better than the
// league-average rates the model starts with.
func TestFitFromHistoryBeatsFlatRates(t *testing.T) {
	sl := newSyntheticLeague()
	rng := rand.New(rand.NewSource(syntheticRandomSeed))
	games := sl.generateSeason(rng)

	split := len(games) * 7 / 10
	train, test := games[:split], games[split:]

	pr := newTestPoissonModel(t)
	if err := pr.FitFromHistory(train); err != nil {
		t.Fatalf("FitFromHistory returned error: %v", err)
	}

	pr.mutex.RLock()
	league := pr.leagueAvgGoalsPerGame
	homeAdv := pr.homeAdvantage
	pr.mutex.RUnlock()

	// Mean log-likelihood per score under fitted rates vs flat 1.0 rates.
	logLik := func(useFit bool) float64 {
		var total float64
		var n int
		for _, g := range test {
			ho, hd := 1.0, 1.0
			ao, ad := 1.0, 1.0
			if useFit {
				ho, hd = pr.GetTeamRates(g.HomeTeam.TeamCode)
				ao, ad = pr.GetTeamRates(g.AwayTeam.TeamCode)
			}
			lambdaHome := ho * ad * league * homeAdv
			lambdaAway := ao * hd * league

			total += math.Log(math.Max(poissonPMF(g.HomeTeam.Score, lambdaHome), 1e-12))
			total += math.Log(math.Max(poissonPMF(g.AwayTeam.Score, lambdaAway), 1e-12))
			n += 2
		}
		return total / float64(n)
	}

	fitted := logLik(true)
	flat := logLik(false)

	if fitted <= flat {
		t.Errorf("fitted rates do not improve held-out log-likelihood: fitted=%.5f flat=%.5f", fitted, flat)
	}
	t.Logf("held-out mean log-likelihood: fitted=%.5f flat=%.5f (improvement %.5f)", fitted, flat, fitted-flat)
}

// The synthetic season samples the two scores from independent Poissons, so
// the true low-score correlation is zero. Fitting rho against league-average
// lambdas instead recovers ~0.27 purely from team-strength variation, which is
// what fitDixonColesRhoWithRates exists to avoid.
func TestFitFromHistoryRecoversNearZeroRhoOnIndependentScores(t *testing.T) {
	sl := newSyntheticLeague()
	rng := rand.New(rand.NewSource(syntheticRandomSeed))
	games := sl.generateSeason(rng)

	pr := newTestPoissonModel(t)
	if err := pr.FitFromHistory(games); err != nil {
		t.Fatalf("FitFromHistory returned error: %v", err)
	}

	pr.mutex.RLock()
	gotRho := pr.rho
	pr.mutex.RUnlock()

	if math.Abs(gotRho) > 0.05 {
		t.Errorf("rho = %.4f on independently sampled scores, want within 0.05 of 0", gotRho)
	}
}

// sampleDixonColesScore draws a correlated score pair from the Dixon-Coles
// joint distribution, so a known non-zero rho can be planted in the data.
func sampleDixonColesScore(rng *rand.Rand, lambdaHome, lambdaAway, rho float64) (int, int) {
	const maxGoals = 12

	var joint [maxGoals + 1][maxGoals + 1]float64
	total := 0.0
	for x := 0; x <= maxGoals; x++ {
		for y := 0; y <= maxGoals; y++ {
			p := dixonColesTau(x, y, lambdaHome, lambdaAway, rho) *
				poissonPMF(x, lambdaHome) * poissonPMF(y, lambdaAway)
			if p < 0 {
				p = 0
			}
			joint[x][y] = p
			total += p
		}
	}

	target := rng.Float64() * total
	cum := 0.0
	for x := 0; x <= maxGoals; x++ {
		for y := 0; y <= maxGoals; y++ {
			cum += joint[x][y]
			if cum >= target {
				return x, y
			}
		}
	}
	return 0, 0
}

// The regularisation must not simply zero rho out. With a real effect planted
// in the data and enough games to show it, the fit has to find it.
func TestFitFromHistoryRecoversGenuineRho(t *testing.T) {
	const trueRho = -0.15

	sl := newSyntheticLeague()
	rng := rand.New(rand.NewSource(syntheticRandomSeed))

	var games []models.CompletedGame
	id := 0
	start := time.Now().AddDate(0, 0, -30)
	// Enough games that a real effect is distinguishable from noise.
	for round := 0; round < 40; round++ {
		for _, home := range sl.teams {
			for _, away := range sl.teams {
				if home == away {
					continue
				}
				lh, la := sl.lambdas(home, away)
				x, y := sampleDixonColesScore(rng, lh, la, trueRho)
				id++
				winner := home
				if y > x {
					winner = away
				}
				games = append(games, models.CompletedGame{
					GameID:   id,
					GameDate: start.Add(time.Duration(id%30) * 24 * time.Hour),
					GameType: 2,
					HomeTeam: models.TeamGameResult{TeamCode: home, Score: x},
					AwayTeam: models.TeamGameResult{TeamCode: away, Score: y},
					Winner:   winner,
				})
			}
		}
	}

	pr := newTestPoissonModel(t)
	if err := pr.FitFromHistory(games); err != nil {
		t.Fatalf("FitFromHistory returned error: %v", err)
	}

	pr.mutex.RLock()
	gotRho := pr.rho
	pr.mutex.RUnlock()

	t.Logf("planted rho=%.3f over %d games, recovered %.4f", trueRho, len(games), gotRho)

	if gotRho >= 0 {
		t.Errorf("rho = %.4f, expected the planted negative correlation to be detected", gotRho)
	}
	if math.Abs(gotRho-trueRho) > 0.06 {
		t.Errorf("rho = %.4f, want %.3f +/- 0.06", gotRho, trueRho)
	}
}

func TestFitFromHistoryRejectsInsufficientHistory(t *testing.T) {
	pr := newTestPoissonModel(t)

	before, _ := pr.GetTeamRates("T00")
	if err := pr.FitFromHistory(nil); err == nil {
		t.Fatal("expected an error when fitting with no games")
	}
	after, _ := pr.GetTeamRates("T00")

	if before != after {
		t.Errorf("rates changed despite a failed fit: %.4f -> %.4f", before, after)
	}
}

// Preseason games are excluded from the fit; a season made only of them should
// never satisfy the minimum.
func TestFitFromHistoryIgnoresPreseason(t *testing.T) {
	sl := newSyntheticLeague()
	rng := rand.New(rand.NewSource(syntheticRandomSeed))
	games := sl.generateSeason(rng)
	for i := range games {
		games[i].GameType = 1
	}

	pr := newTestPoissonModel(t)
	if err := pr.FitFromHistory(games); err == nil {
		t.Fatal("expected preseason-only history to be rejected")
	}
}
