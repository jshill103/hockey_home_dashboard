package services

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// Maximum-likelihood fitting of the Dixon-Coles attack and defence rates.
//
// The model's expected goals are multiplicative:
//
//	lambda_home = off[home] * def[away] * leagueAvg * homeAdvantage
//	lambda_away = off[away] * def[home] * leagueAvg
//
// which is the standard Dixon-Coles parameterisation with off = exp(attack),
// def = exp(defence) and homeAdvantage = exp(gamma).
//
// Rates were previously only moved by an online error-correction step applied
// after each game (see UpdateRates and processGameResult). That tracks recent
// scoring, but it is not a fit: a team's rate absorbs credit for goals that
// were really explained by the strength of whichever opponent it happened to
// play, and the error is attributed to one team at a time rather than solved
// for jointly. Running a schedule-aware fit over accumulated history separates
// those effects, which is the whole point of the attack/defence decomposition.
//
// The fit uses weighted iterative scaling rather than gradient ascent. For
// this multiplicative Poisson family each parameter's conditional MLE has a
// closed form -- the ratio of goals observed to goals expected with the other
// parameters held fixed -- so the likelihood increases monotonically and there
// is no learning rate to tune or diverge.
const (
	// dixonColesFitHalfLifeDays sets the exponential time decay on game
	// weights. Dixon and Coles used a ~107 day half-life for football; a
	// comparable window here keeps roughly the current season's form dominant
	// without discarding last season entirely.
	dixonColesFitHalfLifeDays = 120.0

	// minGamesForDixonColesFit is the league-wide game count below which a
	// joint fit is too underdetermined to beat the online rates.
	minGamesForDixonColesFit = 60

	// dixonColesFitShrinkageGames controls how hard a team's fitted rate is
	// pulled back toward league average. A team with this many weighted games
	// keeps half its fitted deviation. Without it a team seen three times gets
	// an extreme rate off almost no evidence.
	dixonColesFitShrinkageGames = 20.0

	dixonColesFitMaxIterations = 200
	dixonColesFitTolerance     = 1e-7

	dixonColesRhoMin  = -0.30
	dixonColesRhoMax  = 0.30
	dixonColesRhoStep = 0.005

	// dixonColesRhoPriorSD is the standard deviation of the zero-centred prior
	// used to regularise rho. Published Dixon-Coles style fits put the
	// correction well inside +/-0.2, so this is weak enough to let a genuine
	// effect through once there is enough data to show one.
	dixonColesRhoPriorSD = 0.10
)

// FitFromHistory recovers team attack and defence rates, the home-ice
// multiplier and the league scoring level from completed games by maximum
// likelihood, then refits the Dixon-Coles low-score correlation and persists
// the result. Games are weighted by recency.
//
// It returns an error and leaves the model untouched when there is not enough
// history to fit, so callers can safely invoke it speculatively.
func (pr *PoissonRegressionModel) FitFromHistory(games []models.CompletedGame) error {
	fitGames, newest := filterGamesForDixonColesFit(games)
	if len(fitGames) < minGamesForDixonColesFit {
		return fmt.Errorf("dixon-coles fit needs %d usable games, have %d", minGamesForDixonColesFit, len(fitGames))
	}

	decay := math.Ln2 / dixonColesFitHalfLifeDays
	weights := make([]float64, len(fitGames))
	teamWeight := make(map[string]float64)
	for i, g := range fitGames {
		ageDays := newest.Sub(g.GameDate).Hours() / 24
		if ageDays < 0 {
			ageDays = 0
		}
		w := math.Exp(-decay * ageDays)
		weights[i] = w
		teamWeight[g.HomeTeam.TeamCode] += w
		teamWeight[g.AwayTeam.TeamCode] += w
	}

	// Deterministic team order so repeated fits on identical input produce
	// identical output; Go map iteration order is randomised.
	teams := make([]string, 0, len(teamWeight))
	for code := range teamWeight {
		teams = append(teams, code)
	}
	sort.Strings(teams)
	if len(teams) < 2 {
		return fmt.Errorf("dixon-coles fit needs at least 2 teams, have %d", len(teams))
	}

	pr.mutex.RLock()
	league := pr.leagueAvgGoalsPerGame
	homeAdv := pr.homeAdvantage
	pr.mutex.RUnlock()
	if league <= 0 {
		league = 3.1
	}
	if homeAdv <= 0 {
		homeAdv = 1.08
	}

	off := make(map[string]float64, len(teams))
	def := make(map[string]float64, len(teams))
	for _, code := range teams {
		off[code] = 1.0
		def[code] = 1.0
	}

	iterations := 0
	for iter := 0; iter < dixonColesFitMaxIterations; iter++ {
		iterations = iter + 1
		prevOff := make(map[string]float64, len(off))
		prevDef := make(map[string]float64, len(def))
		for code := range off {
			prevOff[code] = off[code]
			prevDef[code] = def[code]
		}
		prevHomeAdv := homeAdv

		// Attack: goals a team scored over the goals it would be expected to
		// score against the defences it actually faced.
		numOff := make(map[string]float64, len(teams))
		denOff := make(map[string]float64, len(teams))
		for i, g := range fitGames {
			w := weights[i]
			home, away := g.HomeTeam.TeamCode, g.AwayTeam.TeamCode
			numOff[home] += w * float64(g.HomeTeam.Score)
			denOff[home] += w * def[away] * league * homeAdv
			numOff[away] += w * float64(g.AwayTeam.Score)
			denOff[away] += w * def[home] * league
		}
		for _, code := range teams {
			if denOff[code] > 0 {
				off[code] = clampPositive(numOff[code]/denOff[code], 0.2, 3.0)
			}
		}

		// Defence: goals a team conceded over the goals the attacks it faced
		// would be expected to score.
		numDef := make(map[string]float64, len(teams))
		denDef := make(map[string]float64, len(teams))
		for i, g := range fitGames {
			w := weights[i]
			home, away := g.HomeTeam.TeamCode, g.AwayTeam.TeamCode
			numDef[home] += w * float64(g.AwayTeam.Score)
			denDef[home] += w * off[away] * league
			numDef[away] += w * float64(g.HomeTeam.Score)
			denDef[away] += w * off[home] * league * homeAdv
		}
		for _, code := range teams {
			if denDef[code] > 0 {
				def[code] = clampPositive(numDef[code]/denDef[code], 0.2, 3.0)
			}
		}

		// Home ice: total home goals over what the same fixtures would produce
		// on neutral ice.
		var numHome, denHome float64
		for i, g := range fitGames {
			w := weights[i]
			numHome += w * float64(g.HomeTeam.Score)
			denHome += w * off[g.HomeTeam.TeamCode] * def[g.AwayTeam.TeamCode] * league
		}
		if denHome > 0 {
			homeAdv = clampPositive(numHome/denHome, 1.0, 1.3)
		}

		// off*k and def/k describe the same lambdas, so pin the attack scale
		// each pass and push the compensating factor into defence. Purely for
		// numerical stability; it leaves every expected-goals value unchanged.
		if k := meanOf(off, teams); k > 0 {
			for _, code := range teams {
				off[code] /= k
				def[code] *= k
			}
		}

		delta := math.Abs(homeAdv - prevHomeAdv)
		for _, code := range teams {
			delta = math.Max(delta, math.Abs(off[code]-prevOff[code]))
			delta = math.Max(delta, math.Abs(def[code]-prevDef[code]))
		}
		if delta < dixonColesFitTolerance {
			break
		}
	}

	// Defence carries the overall scoring level after the attack scale was
	// pinned above. Fold that level into the league average so both rate maps
	// stay centred on 1.0, which is the range the rest of the model and its
	// bounds checks assume.
	if k := meanOf(def, teams); k > 0 {
		for _, code := range teams {
			def[code] /= k
		}
		league = clampPositive(league*k, 2.0, 4.5)
	}

	// Fit rho against the unshrunk maximum-likelihood rates. Shrinkage
	// deliberately compresses team strengths, which under-disperses the
	// lambdas; rho would otherwise widen to compensate for that compression
	// instead of describing the low-score dependence it is meant to capture.
	rho := fitDixonColesRhoWithRates(fitGames, weights, off, def, league, homeAdv)

	// Shrink each team toward league average in proportion to how little we
	// have seen of it.
	for _, code := range teams {
		s := teamWeight[code] / (teamWeight[code] + dixonColesFitShrinkageGames)
		off[code] = clampPositive(1+(off[code]-1)*s, 0.5, 1.8)
		def[code] = clampPositive(1+(def[code]-1)*s, 0.6, 1.5)
	}

	pr.mutex.Lock()
	for _, code := range teams {
		pr.teamOffensiveRates[code] = off[code]
		pr.teamDefensiveRates[code] = def[code]
	}
	pr.leagueAvgGoalsPerGame = league
	pr.homeAdvantage = homeAdv
	pr.rho = rho
	pr.rhoFitted = true
	pr.lastFittedAt = time.Now()
	pr.lastUpdated = time.Now()
	pr.mutex.Unlock()

	log.Printf("📐 Dixon-Coles fit: %d teams over %d games in %d iterations | league avg %.2f, home x%.3f, rho %.4f",
		len(teams), len(fitGames), iterations, league, homeAdv, rho)

	if err := pr.saveRates(); err != nil {
		log.Printf("⚠️ Failed to persist fitted Poisson rates: %v", err)
	}
	return nil
}

// fitDixonColesRhoWithRates fits the low-score correlation parameter against
// each game's own expected goals instead of one league-average pair.
//
// rho is only meant to absorb the mild dependence between the two teams'
// scores in low-scoring games. Scoring every game against league-average
// lambdas (as fitDixonColesRho does, for want of per-team rates at the time it
// runs) leaves ordinary team-strength variation in the residuals, and the grid
// search soaks that up as correlation: on synthetic games generated from
// independent Poissons it recovers rho near 0.27 instead of near 0. Using the
// fitted rates removes that confound.
func fitDixonColesRhoWithRates(
	games []models.CompletedGame,
	weights []float64,
	off, def map[string]float64,
	league, homeAdv float64,
) float64 {
	if len(games) == 0 {
		return defaultDixonColesRho
	}

	type gameLambdas struct {
		home, away       int
		lambdaH, lambdaA float64
		weight           float64
	}

	prepared := make([]gameLambdas, 0, len(games))
	for i, g := range games {
		ho, exists := off[g.HomeTeam.TeamCode]
		if !exists {
			continue
		}
		ao, exists := off[g.AwayTeam.TeamCode]
		if !exists {
			continue
		}
		hd, ad := def[g.HomeTeam.TeamCode], def[g.AwayTeam.TeamCode]

		w := 1.0
		if i < len(weights) {
			w = weights[i]
		}
		prepared = append(prepared, gameLambdas{
			home:    g.HomeTeam.Score,
			away:    g.AwayTeam.Score,
			lambdaH: boundGoalsLambda(ho * ad * league * homeAdv),
			lambdaA: boundGoalsLambda(ao * hd * league),
			weight:  w,
		})
	}
	if len(prepared) == 0 {
		return defaultDixonColesRho
	}

	nllAt := func(rho float64) float64 {
		nll := 0.0
		for _, p := range prepared {
			tau := dixonColesTau(p.home, p.away, p.lambdaH, p.lambdaA, rho)
			joint := tau * poissonPMF(p.home, p.lambdaH) * poissonPMF(p.away, p.lambdaA)
			if joint <= 0 {
				return math.Inf(1)
			}
			nll -= p.weight * math.Log(joint)
		}
		return nll
	}

	bestRho := 0.0
	bestNLL := math.Inf(1)
	for rho := dixonColesRhoMin; rho <= dixonColesRhoMax+1e-9; rho += dixonColesRhoStep {
		if nll := nllAt(rho); nll < bestNLL {
			bestNLL = nll
			bestRho = rho
		}
	}
	if math.IsInf(bestNLL, 1) {
		return defaultDixonColesRho
	}

	// Only four score combinations (0-0, 1-0, 0-1, 1-1) carry any information
	// about rho, so a season's worth of games pins it down far more loosely
	// than the raw game count suggests: on synthetic games with a true rho of
	// zero, the unregularised grid search returns +0.265 at 1,140 games and
	// only settles near zero past ~20,000. Left alone it lands at the edge of
	// the grid and distorts exactly the low-scoring outcomes that make up most
	// of hockey.
	//
	// Shrink toward zero by how sharply the data actually determines rho. The
	// curvature of the negative log-likelihood at the optimum is the observed
	// Fisher information, which combined with a weak zero-centred prior gives
	// the posterior mean below. Plenty of evidence leaves the estimate nearly
	// untouched; thin evidence collapses it toward no correction at all.
	const h = 0.05
	lo, hi := bestRho-h, bestRho+h
	if lo < dixonColesRhoMin || hi > dixonColesRhoMax {
		return 0.0
	}

	curvature := (nllAt(lo) - 2*bestNLL + nllAt(hi)) / (h * h)
	if math.IsNaN(curvature) || math.IsInf(curvature, 0) || curvature <= 0 {
		return 0.0
	}

	priorPrecision := 1.0 / (dixonColesRhoPriorSD * dixonColesRhoPriorSD)
	return bestRho * curvature / (curvature + priorPrecision)
}

// filterGamesForDixonColesFit drops games that would distort the fit and
// reports the most recent game date, which anchors the recency weighting.
func filterGamesForDixonColesFit(games []models.CompletedGame) ([]models.CompletedGame, time.Time) {
	fitGames := make([]models.CompletedGame, 0, len(games))
	var newest time.Time

	for _, g := range games {
		// Preseason lineups are not the lineups being predicted on.
		if g.GameType == 1 {
			continue
		}
		if g.HomeTeam.TeamCode == "" || g.AwayTeam.TeamCode == "" {
			continue
		}
		if g.HomeTeam.TeamCode == g.AwayTeam.TeamCode {
			continue
		}
		// Scores outside this range are data errors rather than blowouts.
		if g.HomeTeam.Score < 0 || g.AwayTeam.Score < 0 || g.HomeTeam.Score > 15 || g.AwayTeam.Score > 15 {
			continue
		}
		if g.GameDate.IsZero() {
			continue
		}
		fitGames = append(fitGames, g)
		if g.GameDate.After(newest) {
			newest = g.GameDate
		}
	}

	return fitGames, newest
}

func clampPositive(v, min, max float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 1.0
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func meanOf(m map[string]float64, keys []string) float64 {
	if len(keys) == 0 {
		return 0
	}
	var sum float64
	for _, k := range keys {
		sum += m[k]
	}
	return sum / float64(len(keys))
}
