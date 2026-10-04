package services

import (
	"fmt"
	"math"

	"github.com/jaredshillingburg/go_uhc/models"
)

// Several models built their predicted score from the win probability alone.
// LSTM and Random Forest shared a byte-identical helper that took the two
// teams' factors and never read them, and Gradient Boosting mapped the
// probability onto one of five fixed strings. The result was a scoreline
// carrying no information about who was playing: across 569 stored
// predictions LSTM returned 3-2 every single time, Random Forest and Gradient
// Boosting did so in 91%, and the Neural Network returned 7-3 in 74%.
//
// A score should come from how much the two sides actually score and concede.

const (
	// League-average goals per game, used when a team's own rates are
	// missing or outside anything hockey produces.
	leagueAverageGoalsFor = 3.1

	// Plausible bounds for one team's expected goals in a single game.
	minExpectedGoals = 1.2
	maxExpectedGoals = 5.5

	// How far the win probability is allowed to pull the scoreline, in goals.
	// A coin flip shifts nothing; near-certainty shifts about three quarters
	// of a goal each way.
	scorelineTiltGoals = 1.5
)

// scorelineFromExpectedGoals builds a predicted final score from what the two
// sides score and concede, tilted by the model's own win probability so the
// score and the pick agree. The result never names a tie, because the NHL
// cannot produce one.
func scorelineFromExpectedGoals(homeWinProb float64, home, away *models.PredictionFactors) string {
	homeExpected := expectedGoalsFor(home, away)
	awayExpected := expectedGoalsFor(away, home)

	// Tilt toward whichever side the model favours.
	tilt := (homeWinProb - 0.5) * scorelineTiltGoals
	homeExpected += tilt
	awayExpected -= tilt

	homeGoals := clampGoals(int(math.Round(homeExpected)))
	awayGoals := clampGoals(int(math.Round(awayExpected)))

	if homeGoals == awayGoals {
		if homeWinProb >= 0.5 {
			homeGoals = clampGoals(homeGoals + 1)
			// If the cap blocked the increment, take one off the other side.
			if homeGoals == awayGoals {
				awayGoals = clampGoals(awayGoals - 1)
			}
		} else {
			awayGoals = clampGoals(awayGoals + 1)
			if homeGoals == awayGoals {
				homeGoals = clampGoals(homeGoals - 1)
			}
		}
	}

	return fmt.Sprintf("%d-%d", homeGoals, awayGoals)
}

// expectedGoalsFor estimates what one side scores against the other, by
// averaging its own offence with the opponent's defence.
func expectedGoalsFor(team, opponent *models.PredictionFactors) float64 {
	attack := plausibleRate(goalsPerGame(team, true))
	defence := plausibleRate(goalsPerGame(opponent, false))
	return (attack + defence) / 2.0
}

// goalsPerGame reads a team's scoring or conceding rate. The field is a
// per-game figure, but some callers populate it with a season total, so a
// value far past anything a team averages is rescaled over a full season
// rather than being taken at face value.
func goalsPerGame(factors *models.PredictionFactors, scoring bool) float64 {
	if factors == nil {
		return leagueAverageGoalsFor
	}

	rate := factors.GoalsAgainst
	if scoring {
		rate = factors.GoalsFor
	}
	if rate > 20 {
		rate /= 82.0
	}
	return rate
}

// plausibleRate keeps a scoring rate inside what hockey produces, falling
// back to the league average when the input is missing or nonsense.
func plausibleRate(rate float64) float64 {
	if math.IsNaN(rate) || rate <= 0 {
		return leagueAverageGoalsFor
	}
	return math.Max(minExpectedGoals, math.Min(maxExpectedGoals, rate))
}

// possessionShare normalises a Corsi or Fenwick figure to 0-1. The codebase
// populates these on both scales -- PlayByPlayService stores a 0-1 share
// while a few older paths store 0-100 -- so consumers cannot assume either.
// A value above 1 can only be a percentage, since a share cannot exceed 1.
func possessionShare(value float64) float64 {
	if math.IsNaN(value) || value <= 0 {
		return neutralPossessionShare
	}
	if value > 1.0 {
		value /= 100.0
	}
	return math.Min(value, 1.0)
}

func clampGoals(goals int) int {
	if goals < 0 {
		return 0
	}
	if goals > maxSimulatedGoals {
		return maxSimulatedGoals
	}
	return goals
}
