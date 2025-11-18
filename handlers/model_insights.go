package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleModelInsights returns detailed insights from all ML models for the next game
func HandleModelInsights(w http.ResponseWriter, r *http.Request) {
	// Get the next game prediction
	if predictionService == nil {
		w.Write([]byte(`<div class="model-insight-error">Prediction service not available</div>`))
		return
	}

	// Get fresh prediction with all model insights
	prediction, err := predictionService.PredictNextGame()
	if err != nil {
		w.Write([]byte(`<div class="model-insight-error">Unable to generate insights: ` + template.HTMLEscapeString(err.Error()) + `</div>`))
		return
	}

	if prediction == nil {
		w.Write([]byte(`<div class="model-insight-error">No upcoming games scheduled</div>`))
		return
	}

	// Get Phase 6 services
	matchupService := services.GetMatchupService()
	rollingService := services.GetRollingStatsService()
	playerService := services.GetPlayerImpactService()

	// Build HTML response
	html := buildModelInsightsHTML(prediction, matchupService, rollingService, playerService)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func buildModelInsightsHTML(
	prediction *models.GamePrediction,
	matchupService *services.MatchupDatabaseService,
	rollingService *services.RollingStatsService,
	playerService *services.PlayerImpactService,
) string {
	homeTeam := prediction.HomeTeam.Name
	awayTeam := prediction.AwayTeam.Name
	homeCode := prediction.HomeTeam.Code
	awayCode := prediction.AwayTeam.Code

	result := prediction.Prediction

	html := fmt.Sprintf(`
<div class="model-insights-container">
	<div class="game-matchup-header">
		<div class="team-info away-team">
			<span class="team-name">%s</span>
			<span class="team-record">Away</span>
		</div>
		<div class="vs-separator">VS</div>
		<div class="team-info home-team">
			<span class="team-name">%s</span>
			<span class="team-record">Home</span>
		</div>
	</div>

	<div class="overall-prediction">
		<div class="prediction-header">🎯 Ensemble Prediction</div>
		<div class="prediction-winner">
			<span class="winner-name">%s</span>
			<span class="win-probability">%.1f%%</span>
		</div>
		<div class="confidence-bar">
			<div class="confidence-fill" style="width: %.1f%%; background: linear-gradient(90deg, #4CAF50, #66BB6A);"></div>
		</div>
		<div class="model-agreement">Confidence: %.1f%%</div>
	</div>
`, awayTeam, homeTeam, result.Winner, result.WinProbability*100,
		result.WinProbability*100, prediction.Confidence*100)

	// Individual model predictions
	html += `<div class="individual-models-section">`
	html += `<div class="models-grid">`

	if len(result.ModelResults) > 0 {
		for _, modelResult := range result.ModelResults {
			confidence := "medium"
			if modelResult.Confidence > 0.75 {
				confidence = "high"
			} else if modelResult.Confidence < 0.5 {
				confidence = "low"
			}

		winner := homeTeam
		winnerProb := modelResult.WinProbability
		if modelResult.WinProbability < 0.5 {
			winner = awayTeam
			winnerProb = 1.0 - modelResult.WinProbability // Show away team's probability
		}

		html += fmt.Sprintf(`
	<div class="model-card">
		<div class="model-name">%s</div>
		<div class="model-prediction">
			<span class="model-winner">%s</span>
			<span class="model-confidence confidence-%s">%.1f%%</span>
		</div>
		<div class="model-weight">Weight: %.1f%%</div>
	</div>
		`, modelResult.ModelName, winner, confidence, winnerProb*100, modelResult.Weight*100)
		}
	} else {
		html += `<div class="model-card"><div class="model-name">Model data loading...</div></div>`
	}

	html += `</div></div>` // Close models-grid and individual-models-section

	// Phase 6: Matchup Intelligence
	if matchupService != nil {
		advantage := matchupService.CalculateMatchupAdvantage(homeCode, awayCode)

		html += `<div class="phase6-section matchup-section">`
		html += `<div class="section-title">📊 Matchup Intelligence</div>`
		html += `<div class="insight-grid">`

		history := matchupService.GetMatchupHistory(homeCode, awayCode)
		if history != nil && history.TotalGames > 0 {
			html += fmt.Sprintf(`
			<div class="insight-item">
				✓ All-Time Record: %s leads %d-%d (Total: %d games)
			</div>
			`, homeCode, history.TeamAWins, history.TeamBWins, history.TotalGames)

			if len(history.RecentGames) > 0 {
				recentWins := 0
				for _, game := range history.RecentGames {
					if game.Winner == homeCode {
						recentWins++
					}
				}
				html += fmt.Sprintf(`
				<div class="insight-item">
					✓ Last 10 Games: %s %d-%d %s
				</div>
				`, homeCode, recentWins, len(history.RecentGames)-recentWins, awayCode)
			}

			if history.IsRivalry {
				html += `
				<div class="insight-item highlight">
					🔥 RIVALRY GAME!
				</div>
				`
			}
		}

		if advantage.TotalAdvantage != 0 {
			sign := "+"
			if advantage.TotalAdvantage < 0 {
				sign = ""
			}
			team := homeCode
			if advantage.TotalAdvantage < 0 {
				team = awayCode
			}
			html += fmt.Sprintf(`
			<div class="insight-item highlight">
				<strong>H2H Advantage:</strong> %s%.1f%% for %s
			</div>
			`, sign, advantage.TotalAdvantage*100, team)
		}

		html += `</div></div>` // Close insight-grid and matchup-section
	}

	// Phase 6: Form & Momentum
	if rollingService != nil {
		homeStats, _ := rollingService.GetTeamStats(homeCode)
		awayStats, _ := rollingService.GetTeamStats(awayCode)

		if homeStats != nil && awayStats != nil {
			html += `<div class="phase6-section form-section">`
			html += `<div class="section-title">🔥 Current Form & Momentum</div>`
			html += `<div class="form-comparison">`

			// Home team
			homeStatus := "neutral"
			homeIcon := "📊"
			if homeStats.IsHot {
				homeStatus = "hot"
				homeIcon = "🔥"
			} else if homeStats.IsCold {
				homeStatus = "cold"
				homeIcon = "🧊"
			}

			streakText := fmt.Sprintf("%d game", homeStats.CurrentStreak)
			if homeStats.CurrentStreak != 1 && homeStats.CurrentStreak != -1 {
				streakText += "s"
			}

			html += fmt.Sprintf(`
			<div class="team-form %s">
				<div class="team-label">%s %s</div>
				<div class="form-rating">Form: %.1f/10</div>
				<div class="momentum-score">Momentum: %+.2f</div>
				<div class="form-details">
					Last 5: %d pts | Streak: %s
				</div>
			</div>
			`, homeStatus, homeIcon, homeTeam, homeStats.FormRating,
				homeStats.MomentumScore, homeStats.Last5GamesPoints, streakText)

			// Away team
			awayStatus := "neutral"
			awayIcon := "📊"
			if awayStats.IsHot {
				awayStatus = "hot"
				awayIcon = "🔥"
			} else if awayStats.IsCold {
				awayStatus = "cold"
				awayIcon = "🧊"
			}

			awayStreakText := fmt.Sprintf("%d game", awayStats.CurrentStreak)
			if awayStats.CurrentStreak != 1 && awayStats.CurrentStreak != -1 {
				awayStreakText += "s"
			}

			html += fmt.Sprintf(`
			<div class="team-form %s">
				<div class="team-label">%s %s</div>
				<div class="form-rating">Form: %.1f/10</div>
				<div class="momentum-score">Momentum: %+.2f</div>
				<div class="form-details">
					Last 5: %d pts | Streak: %s
				</div>
			</div>
			`, awayStatus, awayIcon, awayTeam, awayStats.FormRating,
				awayStats.MomentumScore, awayStats.Last5GamesPoints, awayStreakText)

			html += `</div></div>` // Close form-comparison and form-section
		}
	}

	// Phase 6: Player Impact
	if playerService != nil {
		comparison := playerService.ComparePlayerImpact(homeCode, awayCode)

		html += `<div class="phase6-section player-section">`
		html += `<div class="section-title">⭐ Player Impact Analysis</div>`
		html += `<div class="insight-grid">`

		if comparison.StarPowerAdvantage != 0 {
			team := homeCode
			if comparison.StarPowerAdvantage < 0 {
				team = awayCode
			}
			html += fmt.Sprintf(`
			<div class="insight-item">
				⭐ Star Power Edge: %s (%+.1f%%)
			</div>
			`, team, comparison.StarPowerAdvantage*100)
		}

		if comparison.DepthAdvantage != 0 {
			team := homeCode
			if comparison.DepthAdvantage < 0 {
				team = awayCode
			}
			html += fmt.Sprintf(`
			<div class="insight-item">
				📊 Depth Advantage: %s (%+.1f%%)
			</div>
			`, team, comparison.DepthAdvantage*100)
		}

		if comparison.TotalPlayerImpact != 0 {
			team := homeCode
			if comparison.TotalPlayerImpact < 0 {
				team = awayCode
			}
			html += fmt.Sprintf(`
			<div class="insight-item highlight">
				<strong>Total Player Edge:</strong> %s (%+.1f%%)
			</div>
			`, team, comparison.TotalPlayerImpact*100)
		}

		html += `</div></div>` // Close insight-grid and player-section
	}

	// Model learning status
	liveSys := services.GetLivePredictionSystem()
	if liveSys != nil {
		html += `<div class="learning-status-section">`
		html += `<div class="section-title">📚 Model Learning Status</div>`
		html += `<div class="learning-grid">`

		eloModel := liveSys.GetEloModel()
		if eloModel != nil {
			homeRating := eloModel.GetTeamRating(homeCode)
			awayRating := eloModel.GetTeamRating(awayCode)
			html += fmt.Sprintf(`
		<div class="learning-card">
			<div class="model-name">Elo Ratings</div>
			<div class="rating-info">%s: %.0f | %s: %.0f</div>
			<div class="learning-note">Learns from every game result</div>
		</div>
			`, homeCode, homeRating, awayCode, awayRating)
		}

		poissonModel := liveSys.GetPoissonModel()
		if poissonModel != nil {
			html += `
		<div class="learning-card">
			<div class="model-name">Poisson Model</div>
			<div class="learning-note">Tracks scoring patterns</div>
		</div>
			`
		}

		neuralNet := liveSys.GetNeuralNetwork()
		if neuralNet != nil {
			html += `
		<div class="learning-card">
			<div class="model-name">Neural Network</div>
			<div class="learning-note">105 features, deep learning</div>
		</div>
			`
		}

		html += `</div></div>` // Close learning-grid and learning-status-section
	}

	html += `</div>` // Close model-insights-container

	return html
}
