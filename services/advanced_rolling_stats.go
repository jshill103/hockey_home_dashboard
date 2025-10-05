package services

import (
	"log"
	"math"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// AdvancedRollingStatsCalculator handles Phase 6 advanced rolling statistics
type AdvancedRollingStatsCalculator struct{}

// NewAdvancedRollingStatsCalculator creates a new calculator
func NewAdvancedRollingStatsCalculator() *AdvancedRollingStatsCalculator {
	return &AdvancedRollingStatsCalculator{}
}

// CalculateAdvancedMetrics computes Phase 6 advanced rolling statistics
func (arsc *AdvancedRollingStatsCalculator) CalculateAdvancedMetrics(
	stats *models.TeamRecentPerformance,
	allTeamStats map[string]*models.TeamRecentPerformance,
) {
	// Quality-Weighted Performance
	arsc.calculateQualityMetrics(stats)

	// Time-Weighted Metrics (exponential decay)
	arsc.calculateTimeWeightedMetrics(stats)

	// Momentum Indicators
	arsc.calculateMomentumIndicators(stats)

	// Hot/Cold Detection
	arsc.calculateHotColdIndicators(stats)

	// Scoring Trends
	arsc.calculateScoringTrends(stats)

	// Opponent-Adjusted Metrics
	arsc.calculateOpponentAdjustedMetrics(stats, allTeamStats)

	log.Printf("🎯 Calculated advanced metrics for %s: Form=%.1f, Momentum=%.2f",
		stats.TeamCode, stats.FormRating, stats.MomentumScore)
}

// calculateQualityMetrics computes quality-weighted performance
func (arsc *AdvancedRollingStatsCalculator) calculateQualityMetrics(stats *models.TeamRecentPerformance) {
	if len(stats.Last10Games) == 0 {
		return
	}

	var totalWinQuality float64
	var totalLossQuality float64
	var winsCount, lossesCount int
	var playoffTeamWins, playoffTeamGames int
	var top10Wins, top10Games int
	var clutchWins, clutchGames int
	var blowoutWins, totalWins int
	var closeWins, closeGames int

	for _, game := range stats.Last10Games {
		// Quality of opponent
		opponentStrength := game.OpponentStrength
		if opponentStrength == 0 {
			opponentStrength = 0.5 // Default if not set
		}

		// Track quality of wins/losses
		if game.Result == "W" {
			totalWinQuality += opponentStrength
			winsCount++
			totalWins++

			if game.WasBlowout {
				blowoutWins++
			}
		} else {
			totalLossQuality += opponentStrength
			lossesCount++
		}

		// vs Playoff teams (strength > 0.6)
		if opponentStrength > 0.6 {
			playoffTeamGames++
			if game.Result == "W" {
				playoffTeamWins++
			}
		}

		// vs Top 10 teams (strength > 0.7)
		if opponentStrength > 0.7 {
			top10Games++
			if game.Result == "W" {
				top10Wins++
			}
		}

		// Close games (1-goal)
		if game.WasCloseGame {
			closeGames++
			if game.Result == "W" {
				closeWins++
				clutchWins++
			}
			clutchGames++
		}
	}

	// Calculate averages
	if winsCount > 0 {
		stats.QualityOfWins = totalWinQuality / float64(winsCount)
		stats.BlowoutWinPct = float64(blowoutWins) / float64(totalWins)
	}

	if lossesCount > 0 {
		stats.QualityOfLosses = totalLossQuality / float64(lossesCount)
	}

	if playoffTeamGames > 0 {
		stats.VsPlayoffTeamsPct = float64(playoffTeamWins) / float64(playoffTeamGames)
	}

	if top10Games > 0 {
		stats.VsTop10TeamsPct = float64(top10Wins) / float64(top10Games)
	}

	if clutchGames > 0 {
		stats.ClutchPerformance = float64(clutchWins) / float64(clutchGames)
	}

	if closeGames > 0 {
		stats.CloseGameRecord = models.Record{
			Wins:   closeWins,
			Losses: closeGames - closeWins,
			Points: closeWins * 2,
		}
	}
}

// calculateTimeWeightedMetrics applies exponential decay to recent performance
func (arsc *AdvancedRollingStatsCalculator) calculateTimeWeightedMetrics(stats *models.TeamRecentPerformance) {
	if len(stats.Last10Games) == 0 {
		return
	}

	var weightedGoalsFor float64
	var weightedGoalsAgainst float64
	var weightedWins float64
	var totalWeight float64

	// Exponential decay: most recent game has highest weight
	// Decay factor: 0.85 (recent games matter more)
	decayFactor := 0.85

	for i, game := range stats.Last10Games {
		// Weight: 1.0 for most recent, decays exponentially
		weight := math.Pow(decayFactor, float64(i))
		totalWeight += weight

		weightedGoalsFor += float64(game.GoalsFor) * weight
		weightedGoalsAgainst += float64(game.GoalsAgainst) * weight

		if game.Result == "W" {
			weightedWins += weight
		} else if game.Result == "OTL" || game.Result == "SOL" {
			weightedWins += weight * 0.5 // OT loss counts as half
		}
	}

	if totalWeight > 0 {
		stats.WeightedGoalsFor = weightedGoalsFor / totalWeight
		stats.WeightedGoalsAgainst = weightedGoalsAgainst / totalWeight
		stats.WeightedWinPct = weightedWins / totalWeight
	}

	// Calculate momentum score (-1 to +1)
	// Positive momentum = recent games better than older games
	var recentPerformance float64 // Last 3 games
	var olderPerformance float64  // Games 4-10

	for i, game := range stats.Last10Games {
		performance := 0.0
		if game.Result == "W" {
			performance = 1.0
		} else if game.Result == "OTL" || game.Result == "SOL" {
			performance = 0.5
		}

		if i < 3 {
			recentPerformance += performance
		} else {
			olderPerformance += performance
		}
	}

	recentAvg := recentPerformance / 3.0
	olderAvg := olderPerformance / 7.0

	if olderAvg > 0 {
		// Momentum: how much better/worse is recent vs older
		stats.MomentumScore = (recentAvg - olderAvg)

		// Cap at -1 to +1
		if stats.MomentumScore > 1.0 {
			stats.MomentumScore = 1.0
		} else if stats.MomentumScore < -1.0 {
			stats.MomentumScore = -1.0
		}
	}

	// Form rating (0-10 scale)
	// Based on weighted win %, weighted goal diff, and momentum
	goalDiff := stats.WeightedGoalsFor - stats.WeightedGoalsAgainst
	goalDiffComponent := math.Tanh(goalDiff / 2.0) // Normalize to -1 to +1

	// Form = 5 + (weighted_win% * 3) + (goal_diff * 1) + (momentum * 1)
	stats.FormRating = 5.0 +
		(stats.WeightedWinPct * 3.0) +
		(goalDiffComponent * 1.0) +
		(stats.MomentumScore * 1.0)

	// Clamp to 0-10
	if stats.FormRating < 0 {
		stats.FormRating = 0
	} else if stats.FormRating > 10 {
		stats.FormRating = 10
	}
}

// calculateMomentumIndicators computes momentum and trend indicators
func (arsc *AdvancedRollingStatsCalculator) calculateMomentumIndicators(stats *models.TeamRecentPerformance) {
	if len(stats.Last10Games) == 0 {
		return
	}

	// Points in last N games
	stats.Last3GamesPoints = 0
	stats.Last5GamesPoints = 0
	stats.Last10GamesPoints = 0

	stats.GoalDifferential3 = 0
	stats.GoalDifferential5 = 0
	stats.GoalDifferential10 = 0

	for i, game := range stats.Last10Games {
		goalDiff := game.GoalsFor - game.GoalsAgainst

		if i < 3 {
			stats.Last3GamesPoints += game.Points
			stats.GoalDifferential3 += goalDiff
		}
		if i < 5 {
			stats.Last5GamesPoints += game.Points
			stats.GoalDifferential5 += goalDiff
		}
		stats.Last10GamesPoints += game.Points
		stats.GoalDifferential10 += goalDiff
	}

	// Determine trend direction
	threshold3High := float64(stats.Last5GamesPoints) * 0.6
	threshold3Low := float64(stats.Last5GamesPoints) * 0.4

	if float64(stats.Last3GamesPoints) > threshold3High {
		stats.PointsTrendDirection = "accelerating"
	} else if float64(stats.Last3GamesPoints) < threshold3Low {
		stats.PointsTrendDirection = "declining"
	} else {
		stats.PointsTrendDirection = "stable"
	}

	// Days since last win/loss
	now := time.Now()
	for _, game := range stats.Last10Games {
		daysSince := int(now.Sub(game.Date).Hours() / 24)

		if game.Result == "W" && stats.DaysSinceLastWin == 0 {
			stats.DaysSinceLastWin = daysSince
		}
		if game.Result == "L" && stats.DaysSinceLastLoss == 0 {
			stats.DaysSinceLastLoss = daysSince
		}
	}
}

// calculateHotColdIndicators determines if team is hot/cold/streaking
func (arsc *AdvancedRollingStatsCalculator) calculateHotColdIndicators(stats *models.TeamRecentPerformance) {
	if len(stats.Last5Games) < 5 {
		return
	}

	winsLast5 := 0
	lossesLast5 := 0

	for _, game := range stats.Last5Games {
		if game.Result == "W" {
			winsLast5++
		} else if game.Result == "L" {
			lossesLast5++
		}
	}

	// Hot: 4+ wins in last 5
	stats.IsHot = winsLast5 >= 4

	// Cold: 4+ losses in last 5
	stats.IsCold = lossesLast5 >= 4

	// Streaking: 3+ game win/loss streak
	stats.IsStreaking = (stats.CurrentStreak >= 3 || stats.CurrentStreak <= -3)
}

// calculateScoringTrends computes trend lines for various metrics
func (arsc *AdvancedRollingStatsCalculator) calculateScoringTrends(stats *models.TeamRecentPerformance) {
	if len(stats.Last10Games) < 5 {
		return
	}

	// Calculate simple linear trends (recent vs older)
	var recentGoalsFor, recentGoalsAgainst float64
	var olderGoalsFor, olderGoalsAgainst float64
	var recentPP, olderPP float64
	var recentSaves, olderSaves, recentShots, olderShots float64

	recentCount := 3
	olderCount := 5

	for i, game := range stats.Last10Games {
		if i < recentCount {
			recentGoalsFor += float64(game.GoalsFor)
			recentGoalsAgainst += float64(game.GoalsAgainst)
			recentShots += float64(game.Shots)
			if game.Shots > 0 {
				recentSaves += float64(game.Shots - game.GoalsFor)
			}
			if game.PowerPlayOpps > 0 {
				recentPP += float64(game.PowerPlayGoals) / float64(game.PowerPlayOpps)
			}
		} else if i < recentCount+olderCount {
			olderGoalsFor += float64(game.GoalsFor)
			olderGoalsAgainst += float64(game.GoalsAgainst)
			olderShots += float64(game.Shots)
			if game.Shots > 0 {
				olderSaves += float64(game.Shots - game.GoalsFor)
			}
			if game.PowerPlayOpps > 0 {
				olderPP += float64(game.PowerPlayGoals) / float64(game.PowerPlayOpps)
			}
		}
	}

	// Trends: recent avg - older avg
	stats.ScoringTrend = (recentGoalsFor / float64(recentCount)) - (olderGoalsFor / float64(olderCount))
	stats.DefensiveTrend = (olderGoalsAgainst / float64(olderCount)) - (recentGoalsAgainst / float64(recentCount)) // Reversed (lower is better)
	stats.PowerPlayTrend = (recentPP / float64(recentCount)) - (olderPP / float64(olderCount))

	// Goalie trend (save %)
	recentSavePct := 0.0
	olderSavePct := 0.0
	if recentShots > 0 {
		recentSavePct = recentSaves / recentShots
	}
	if olderShots > 0 {
		olderSavePct = olderSaves / olderShots
	}
	stats.GoalieTrend = recentSavePct - olderSavePct
}

// calculateOpponentAdjustedMetrics adjusts stats based on opponent quality
func (arsc *AdvancedRollingStatsCalculator) calculateOpponentAdjustedMetrics(
	stats *models.TeamRecentPerformance,
	allTeamStats map[string]*models.TeamRecentPerformance,
) {
	if len(stats.Last10Games) == 0 {
		return
	}

	var totalStrength float64
	var adjustedGoalsFor float64
	var adjustedGoalsAgainst float64
	var adjustedWins float64
	var gamesCount float64

	for _, game := range stats.Last10Games {
		// Get opponent strength
		opponentStrength := game.OpponentStrength
		if opponentStrength == 0 {
			// Try to get from live stats
			if oppStats, exists := allTeamStats[game.Opponent]; exists {
				opponentStrength = oppStats.RecentWinPct
				if opponentStrength == 0 {
					opponentStrength = 0.5 // Default
				}
			} else {
				opponentStrength = 0.5
			}
		}

		totalStrength += opponentStrength
		gamesCount++

		// Adjust goals based on opponent quality
		// Against strong team (0.7): goals worth more
		// Against weak team (0.3): goals worth less
		qualityFactor := 0.5 + (opponentStrength * 0.5) // 0.65 - 1.0

		adjustedGoalsFor += float64(game.GoalsFor) / qualityFactor
		adjustedGoalsAgainst += float64(game.GoalsAgainst) * qualityFactor

		// Adjust wins
		if game.Result == "W" {
			adjustedWins += 1.0 / qualityFactor // Win vs strong team counts more
		}
	}

	if gamesCount > 0 {
		stats.StrengthOfSchedule = totalStrength / gamesCount
		stats.AdjustedGoalsFor = adjustedGoalsFor / gamesCount
		stats.AdjustedGoalsAgainst = adjustedGoalsAgainst / gamesCount
		stats.AdjustedWinPct = adjustedWins / gamesCount
	}
}

// EnrichGameSummaryWithQuality adds opponent quality metrics to a game summary
func (arsc *AdvancedRollingStatsCalculator) EnrichGameSummaryWithQuality(
	game *models.GameSummary,
	opponentStats *models.TeamRecentPerformance,
	leagueRankings map[string]int,
) {
	// Set opponent rank
	if rank, exists := leagueRankings[game.Opponent]; exists {
		game.OpponentRank = rank
	}

	// Set opponent win %
	if opponentStats != nil {
		game.OpponentWinPct = opponentStats.RecentWinPct

		// Calculate opponent strength (0-1 scale)
		// Based on win %, rank, and form
		winPctComponent := opponentStats.RecentWinPct
		rankComponent := 1.0 - (float64(game.OpponentRank) / 32.0)
		formComponent := opponentStats.FormRating / 10.0

		game.OpponentStrength = (winPctComponent*0.5 + rankComponent*0.3 + formComponent*0.2)
	} else {
		game.OpponentStrength = 0.5 // Default
	}

	// Determine game type
	scoreDiff := game.GoalsFor - game.GoalsAgainst
	if scoreDiff == 1 || scoreDiff == -1 {
		game.WasCloseGame = true
	}
	if scoreDiff >= 3 || scoreDiff <= -3 {
		game.WasBlowout = true
	}

	// Game importance (placeholder - could be enhanced with playoff race data)
	game.GameImportance = 0.5 // Default medium importance
}
