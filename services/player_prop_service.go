package services

import (
	"fmt"
	"math"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/utils"
)

// Blend weights applied between a player's season-long per-game rate and
// their recent-10-game per-game rate when projecting props. This mirrors the
// spirit of PlayerImpactService's existing recent-form logic (which already
// leans on a last-10-games window via calculateRecentForm/fetchPlayerGameLog)
// without inventing a disconnected weighting scheme.
const (
	propSeasonWeight = 0.6
	propRecentWeight = 0.4
	propRecentWindow = 10

	// minPropLambda keeps Poisson lambdas away from exactly zero (which would
	// make every over-probability collapse to 0 for a player who simply
	// hasn't recorded a shot/assist/etc. yet in a very small sample).
	minPropLambda = 0.02
)

// PropLineProbability is P(stat > line) for one sportsbook-style line.
type PropLineProbability struct {
	Line            float64 `json:"line"`
	ProbabilityOver float64 `json:"probabilityOver"`
}

// PlayerPropDistribution is the Poisson-modeled projection for a single stat.
type PlayerPropDistribution struct {
	Stat              string                `json:"stat"`
	Lambda            float64               `json:"lambda"`
	OverProbabilities []PropLineProbability `json:"overProbabilities"`
}

// PlayerPropProjection bundles the goals/assists/points/shots-on-goal
// distributions for a player against a specific opponent.
type PlayerPropProjection struct {
	PlayerID   int    `json:"playerId"`
	PlayerName string `json:"playerName"`
	TeamCode   string `json:"teamCode"`
	Opponent   string `json:"opponent"`

	// OpponentDefensiveFactor is the opponent's normalized defensive rate
	// (1.0 == league average; >1 == weaker defense/more goals allowed; <1 ==
	// stronger defense), taken directly from PoissonRegressionModel so this
	// doesn't invent a second, disconnected opponent-strength scheme.
	OpponentDefensiveFactor float64 `json:"opponentDefensiveFactor"`

	GamesUsedForSeasonRate int `json:"gamesUsedForSeasonRate"`
	GamesUsedForRecentRate int `json:"gamesUsedForRecentRate"`

	Goals       PlayerPropDistribution `json:"goals"`
	Assists     PlayerPropDistribution `json:"assists"`
	Points      PlayerPropDistribution `json:"points"`
	ShotsOnGoal PlayerPropDistribution `json:"shotsOnGoal"`

	// Notes surfaces any data-availability caveats (e.g. game log fetch
	// failures) so API consumers know when a stat fell back to a coarser
	// estimate.
	Notes []string `json:"notes,omitempty"`
}

// buildPropDistribution constructs a PlayerPropDistribution for one stat,
// computing P(over) for each requested line via the shared Poisson PMF/CDF
// helpers in poisson_regression_model.go.
func buildPropDistribution(stat string, lambda float64, lines []float64) PlayerPropDistribution {
	dist := PlayerPropDistribution{Stat: stat, Lambda: lambda}
	for _, line := range lines {
		dist.OverProbabilities = append(dist.OverProbabilities, PropLineProbability{
			Line:            line,
			ProbabilityOver: poissonProbOverLine(line, lambda),
		})
	}
	return dist
}

// GetPlayerPropProjection produces Poisson-distribution-based prop
// projections (goals, assists, points, shots-on-goal) for one player against
// a specific opponent.
//
// It reuses PlayerImpactService's existing cached season data (models.TopScorer)
// for the player's identity/season totals, and fetches a fresh game log (via
// the private fetchPlayerGameLog helper, which is why this method lives on
// PlayerImpactService rather than as a free-standing function) to derive
// per-game rates -- both a season-average and a recent-10-game average -- for
// every stat, including shots-on-goal, which isn't tracked anywhere in the
// cached PlayerImpact/TopScorer data (only the game log carries per-game
// shots).
func (pis *PlayerImpactService) GetPlayerPropProjection(teamCode, opponentCode string, playerID int) (*PlayerPropProjection, error) {
	teamCode = strings.ToUpper(strings.TrimSpace(teamCode))
	opponentCode = strings.ToUpper(strings.TrimSpace(opponentCode))

	if teamCode == "" || opponentCode == "" {
		return nil, fmt.Errorf("teamCode and opponentCode are required")
	}

	season := utils.GetCurrentSeason()

	// Make sure we have season data cached for this team (mirrors the
	// NeedsUpdate/UpdatePlayerImpact pattern already used in predictions.go).
	if pis.NeedsUpdate(teamCode) {
		if err := pis.UpdatePlayerImpact(teamCode, season); err != nil {
			// Non-fatal: fall through and try to use whatever is cached
			// (possibly stale, possibly empty) rather than failing outright.
			_ = err
		}
	}

	impact := pis.GetPlayerImpact(teamCode)
	if impact == nil || len(impact.TopScorers) == 0 {
		return nil, fmt.Errorf("no cached player data available for team %s", teamCode)
	}
	if impact.Season != 0 {
		season = impact.Season
	}

	var player *models.TopScorer
	for i := range impact.TopScorers {
		if impact.TopScorers[i].PlayerID == playerID {
			player = &impact.TopScorers[i]
			break
		}
	}
	if player == nil {
		return nil, fmt.Errorf("player %d not found among cached top scorers for team %s (only top-10 scorers are tracked)", playerID, teamCode)
	}

	notes := []string{}

	// Fetch the player's game log once. Passing a generous numGames (larger
	// than an NHL season) is a no-op on the trimming logic in
	// fetchPlayerGameLogForSeason -- the whole season is fetched in a single
	// API call either way, and we derive both the season-average and the
	// recent-10-game average from that one slice, keeping every stat
	// (including shots-on-goal, which isn't cached elsewhere) on a
	// consistent games-played denominator.
	const fullSeasonGameCap = 100
	fullLog, err := pis.fetchPlayerGameLog(playerID, season, fullSeasonGameCap)

	var seasonGoalsPG, seasonAssistsPG, seasonPointsPG, seasonShotsPG float64
	var recentGoalsPG, recentAssistsPG, recentPointsPG, recentShotsPG float64
	seasonGames := player.GamesPlayed
	recentGames := 0

	if err != nil || len(fullLog) == 0 {
		// Fall back to the cached season totals only; shots-on-goal isn't
		// available anywhere except the per-game log, so it's explicitly
		// unavailable in this fallback path.
		notes = append(notes, fmt.Sprintf("game log unavailable (%v); using cached season totals only, shots-on-goal projection is not available", err))

		games := math.Max(1, float64(player.GamesPlayed))
		seasonGoalsPG = float64(player.Goals) / games
		seasonAssistsPG = float64(player.Assists) / games
		seasonPointsPG = player.PointsPerGame
		recentGoalsPG, recentAssistsPG, recentPointsPG = seasonGoalsPG, seasonAssistsPG, seasonPointsPG
		seasonShotsPG, recentShotsPG = 0, 0
	} else {
		n := len(fullLog)
		seasonGames = n

		var sumGoals, sumAssists, sumPoints, sumShots int
		for _, g := range fullLog {
			sumGoals += g.Goals
			sumAssists += g.Assists
			sumPoints += g.Points
			sumShots += g.Shots
		}
		seasonGoalsPG = float64(sumGoals) / float64(n)
		seasonAssistsPG = float64(sumAssists) / float64(n)
		seasonPointsPG = float64(sumPoints) / float64(n)
		seasonShotsPG = float64(sumShots) / float64(n)

		recentWindow := propRecentWindow
		if recentWindow > n {
			recentWindow = n
		}
		recentLog := fullLog[n-recentWindow:]
		recentGames = len(recentLog)

		// Reuse the existing recent-form helper for goals/assists/points so
		// this matches the rest of the codebase's recent-form math exactly.
		rGoals, rAssists, rPoints, _, _ := pis.calculateRecentForm(recentLog)
		recentGoalsPG = float64(rGoals) / float64(recentGames)
		recentAssistsPG = float64(rAssists) / float64(recentGames)
		recentPointsPG = float64(rPoints) / float64(recentGames)

		var recentShots int
		for _, g := range recentLog {
			recentShots += g.Shots
		}
		recentShotsPG = float64(recentShots) / float64(recentGames)
	}

	blend := func(seasonRate, recentRate float64) float64 {
		return propSeasonWeight*seasonRate + propRecentWeight*recentRate
	}

	// Opponent adjustment: reuse PoissonRegressionModel's existing normalized
	// defensive rate (league average == 1.0) rather than a separate scheme.
	poisson := getSharedPoissonModel()
	defensiveFactor := poisson.getDefensiveRate(opponentCode)

	goalsLambda := math.Max(minPropLambda, blend(seasonGoalsPG, recentGoalsPG)*defensiveFactor)
	assistsLambda := math.Max(minPropLambda, blend(seasonAssistsPG, recentAssistsPG)*defensiveFactor)
	pointsLambda := math.Max(minPropLambda, blend(seasonPointsPG, recentPointsPG)*defensiveFactor)

	var shotsDist PlayerPropDistribution
	if seasonShotsPG == 0 && recentShotsPG == 0 {
		notes = append(notes, "shots-on-goal lambda is 0 (no per-game shot data available for this player)")
		shotsDist = buildPropDistribution("shotsOnGoal", 0, []float64{1.5, 2.5, 3.5})
	} else {
		shotsLambda := math.Max(minPropLambda, blend(seasonShotsPG, recentShotsPG)*defensiveFactor)
		shotsDist = buildPropDistribution("shotsOnGoal", shotsLambda, []float64{1.5, 2.5, 3.5})
	}

	return &PlayerPropProjection{
		PlayerID:                playerID,
		PlayerName:              player.Name,
		TeamCode:                teamCode,
		Opponent:                opponentCode,
		OpponentDefensiveFactor: defensiveFactor,
		GamesUsedForSeasonRate:  seasonGames,
		GamesUsedForRecentRate:  recentGames,
		Goals:                   buildPropDistribution("goals", goalsLambda, []float64{0.5, 1.5}),
		Assists:                 buildPropDistribution("assists", assistsLambda, []float64{0.5, 1.5}),
		Points:                  buildPropDistribution("points", pointsLambda, []float64{0.5, 1.5, 2.5}),
		ShotsOnGoal:             shotsDist,
		Notes:                   notes,
	}, nil
}
