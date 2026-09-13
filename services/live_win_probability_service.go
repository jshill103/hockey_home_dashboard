package services

import (
	"fmt"
	"time"
)

// NHL regulation is 3 x 20-minute periods.
const (
	liveRegulationPeriods = 3
	liveSecondsPerPeriod  = 1200                                         // 20 minutes
	liveRegulationSeconds = liveRegulationPeriods * liveSecondsPerPeriod // 3600

	// liveWinProbabilityTrials matches the trial count PoissonRegressionModel's
	// (independent-Poisson era) calculateWinProbability used, for consistency
	// with the rest of the codebase's Monte Carlo conventions.
	liveWinProbabilityTrials = 10000
)

// LiveWinProbabilityResult is the response payload for a live, in-game win
// probability calculation. It intentionally exposes the intermediate values
// (lambdas, trial tallies) so callers/debuggers can sanity-check the math.
type LiveWinProbabilityResult struct {
	HomeTeam string `json:"homeTeam"`
	AwayTeam string `json:"awayTeam"`

	HomeScore int `json:"homeScore"`
	AwayScore int `json:"awayScore"`
	Period    int `json:"period"`

	SecondsRemainingInPeriod   float64 `json:"secondsRemainingInPeriod"`
	RegulationSecondsRemaining float64 `json:"regulationSecondsRemaining"`
	TimeRemainingFraction      float64 `json:"timeRemainingFraction"`

	// Full-game pregame lambdas (same rate lookups the pregame Poisson model uses).
	HomeLambdaFullGame float64 `json:"homeLambdaFullGame"`
	AwayLambdaFullGame float64 `json:"awayLambdaFullGame"`

	// Lambdas scaled down to the time actually remaining in the game.
	HomeLambdaRemaining float64 `json:"homeLambdaRemaining"`
	AwayLambdaRemaining float64 `json:"awayLambdaRemaining"`

	// PregameHomeWinProbability (Elo, including its own home-ice adjustment) is
	// used only as a fair coin-weighting for simulated trials that are still
	// tied at the end of regulation, since real NHL games always resolve via
	// OT/shootout rather than ending in a tie.
	PregameHomeWinProbability float64 `json:"pregameHomeWinProbability"`

	Trials                       int `json:"trials"`
	HomeWinsOutrightInRegulation int `json:"homeWinsOutrightInRegulation"`
	AwayWinsOutrightInRegulation int `json:"awayWinsOutrightInRegulation"`
	TiedAfterRegulationTrials    int `json:"tiedAfterRegulationTrials"`

	HomeWinProbability float64 `json:"homeWinProbability"`
	AwayWinProbability float64 `json:"awayWinProbability"`
}

// getSharedPoissonModel returns the live prediction system's shared, already
// warmed-up (and rate-persisted) Poisson model when the system has been
// initialized, falling back to a freshly constructed disk-backed instance
// otherwise (e.g. if called before startup finishes, or from a standalone
// tool/test).
func getSharedPoissonModel() *PoissonRegressionModel {
	if lps := GetLivePredictionSystem(); lps != nil {
		if pm := lps.GetPoissonModel(); pm != nil {
			return pm
		}
	}
	return NewPoissonRegressionModel()
}

// getSharedEloModel mirrors getSharedPoissonModel for the Elo rating model.
func getSharedEloModel() *EloRatingModel {
	if lps := GetLivePredictionSystem(); lps != nil {
		if em := lps.GetEloModel(); em != nil {
			return em
		}
	}
	return NewEloRatingModel()
}

// boundGoalsLambda mirrors the bounds PoissonRegressionModel.calculateExpectedGoals
// applies to a team's full-game expected goals (NHL games realistically see
// 0.5-7.0 goals per team per game).
func boundGoalsLambda(lambda float64) float64 {
	if lambda < 0.5 {
		return 0.5
	}
	if lambda > 7.0 {
		return 7.0
	}
	return lambda
}

// CalculateLiveWinProbability computes an in-game win probability for the home
// team.
//
// It is a principled extension of the existing Poisson goal-rate model rather
// than a separate ad hoc heuristic: it takes the same pregame offensive/
// defensive rate lookups PoissonRegressionModel.calculateExpectedGoals uses,
// scales the resulting full-game lambdas down by the fraction of regulation
// time remaining, then Monte Carlo re-simulates only the *remaining* goals
// (reusing the model's existing Poisson sampler) on top of the actual current
// score. Trials still tied after regulation are resolved using the Elo
// model's pregame win probability as a fair (not flat 50/50) OT/shootout
// coin-weighting, since OT outcomes are close to -- but not exactly -- a coin
// flip between mismatched teams.
func CalculateLiveWinProbability(homeTeam, awayTeam string, homeScore, awayScore, period int, periodTimeRemaining time.Duration) (*LiveWinProbabilityResult, error) {
	if homeTeam == "" || awayTeam == "" {
		return nil, fmt.Errorf("homeTeam and awayTeam are required")
	}
	if period < 1 {
		period = 1
	}

	secondsRemainingInPeriod := periodTimeRemaining.Seconds()
	if secondsRemainingInPeriod < 0 {
		secondsRemainingInPeriod = 0
	}

	// Seconds of REGULATION left = time left in the current period, plus one
	// full period for every regulation period still to come. Once we're past
	// regulation (period > 3, i.e. OT/shootout), there is no regulation time
	// left -- the score at that point is (by construction) tied, and the tie
	// resolution below takes over immediately.
	regulationSecondsRemaining := 0.0
	if period <= liveRegulationPeriods {
		periodsFullyRemaining := liveRegulationPeriods - period
		regulationSecondsRemaining = secondsRemainingInPeriod + float64(periodsFullyRemaining)*liveSecondsPerPeriod
	}
	if regulationSecondsRemaining > liveRegulationSeconds {
		regulationSecondsRemaining = liveRegulationSeconds
	}
	if regulationSecondsRemaining < 0 {
		regulationSecondsRemaining = 0
	}

	timeRemainingFraction := regulationSecondsRemaining / float64(liveRegulationSeconds)

	poisson := getSharedPoissonModel()
	elo := getSharedEloModel()

	// Full-game expected goals using the same underlying offensive/defensive
	// rate lookups and home-ice multiplier as
	// PoissonRegressionModel.calculateExpectedGoals -- deliberately skipping
	// that function's situational/analytics multipliers, which require a full
	// models.PredictionFactors for both teams that isn't available (or
	// particularly meaningful) mid-game here.
	homeOffensive := poisson.getOffensiveRate(homeTeam)
	homeDefensive := poisson.getDefensiveRate(homeTeam)
	awayOffensive := poisson.getOffensiveRate(awayTeam)
	awayDefensive := poisson.getDefensiveRate(awayTeam)

	homeLambdaFullGame := boundGoalsLambda(homeOffensive * awayDefensive * poisson.leagueAvgGoalsPerGame * poisson.homeAdvantage)
	awayLambdaFullGame := boundGoalsLambda(awayOffensive * homeDefensive * poisson.leagueAvgGoalsPerGame)

	homeLambdaRemaining := homeLambdaFullGame * timeRemainingFraction
	awayLambdaRemaining := awayLambdaFullGame * timeRemainingFraction

	// Pregame-style win probability (Elo, including its own home-ice
	// adjustment), used purely as the OT/SO tie-resolution weight.
	homeRating := elo.GetTeamRating(homeTeam)
	awayRating := elo.GetTeamRating(awayTeam)
	pregameHomeWinProb := elo.calculateWinProbability(homeRating+elo.homeAdvantage, awayRating)

	trials := liveWinProbabilityTrials
	homeWinsOutright := 0
	awayWinsOutright := 0
	ties := 0

	for i := 0; i < trials; i++ {
		addedHomeGoals := poisson.samplePoisson(homeLambdaRemaining)
		addedAwayGoals := poisson.samplePoisson(awayLambdaRemaining)

		finalHome := homeScore + addedHomeGoals
		finalAway := awayScore + addedAwayGoals

		switch {
		case finalHome > finalAway:
			homeWinsOutright++
		case finalAway > finalHome:
			awayWinsOutright++
		default:
			ties++
		}
	}

	homeWinProbability := (float64(homeWinsOutright) + float64(ties)*pregameHomeWinProb) / float64(trials)
	awayWinProbability := 1.0 - homeWinProbability

	return &LiveWinProbabilityResult{
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,

		HomeScore: homeScore,
		AwayScore: awayScore,
		Period:    period,

		SecondsRemainingInPeriod:   secondsRemainingInPeriod,
		RegulationSecondsRemaining: regulationSecondsRemaining,
		TimeRemainingFraction:      timeRemainingFraction,

		HomeLambdaFullGame: homeLambdaFullGame,
		AwayLambdaFullGame: awayLambdaFullGame,

		HomeLambdaRemaining: homeLambdaRemaining,
		AwayLambdaRemaining: awayLambdaRemaining,

		PregameHomeWinProbability: pregameHomeWinProb,

		Trials:                       trials,
		HomeWinsOutrightInRegulation: homeWinsOutright,
		AwayWinsOutrightInRegulation: awayWinsOutright,
		TiedAfterRegulationTrials:    ties,

		HomeWinProbability: homeWinProbability,
		AwayWinProbability: awayWinProbability,
	}, nil
}
