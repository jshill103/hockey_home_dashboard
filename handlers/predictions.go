package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// Global prediction cache
var (
	cachedPrediction  *models.GamePrediction
	predictionService *services.PredictionService
)

// InitPredictions initializes the prediction service
func InitPredictions(teamCode string) {
	predictionService = services.NewPredictionService(teamCode)
	fmt.Printf("Initialized AI prediction service for %s\n", teamCode)
}

// HandleGamePrediction returns AI prediction for the next game as JSON
func HandleGamePrediction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if predictionService == nil {
		http.Error(w, `{"error": "Prediction service not initialized"}`, http.StatusInternalServerError)
		return
	}

	// Check if force refresh is requested via query parameter
	forceRefresh := r.URL.Query().Get("refresh") == "true"
	if forceRefresh {
		fmt.Printf("🔄 Force refresh requested - clearing prediction cache...\n")
		// Clear service-level prediction cache
		cache := services.GetPredictionCache()
		if cache != nil {
			cache.ClearCache()
			fmt.Printf("✅ Prediction cache cleared\n")
		}
		// Clear standings cache to get latest NHL data
		standingsCache := services.GetStandingsCacheService()
		if standingsCache != nil {
			standingsCache.InvalidateCache()
			fmt.Printf("✅ Standings cache invalidated\n")
		}
	}

	// Generate fresh prediction (DON'T cache - always get latest data)
	prediction, err := predictionService.PredictNextGame()
	if err != nil {
		fmt.Printf("Error generating prediction: %v\n", err)
		http.Error(w, fmt.Sprintf(`{"error": "Failed to generate prediction: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// NOTE: Removed handler-level caching to ensure fresh predictions with latest standings data
	// The service layer has its own caching with proper TTLs

	// Return as JSON
	if err := json.NewEncoder(w).Encode(prediction); err != nil {
		http.Error(w, `{"error": "Failed to encode prediction"}`, http.StatusInternalServerError)
		return
	}
}

// HandlePredictionWidget returns HTML widget displaying the game prediction
func HandlePredictionWidget(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if predictionService == nil {
		fmt.Fprint(w, `<div class="prediction-error">AI Predictions not available</div>`)
		return
	}

	// Use cached prediction or generate new one
	var prediction *models.GamePrediction
	if cachedPrediction != nil {
		prediction = cachedPrediction
	} else {
		var err error
		prediction, err = predictionService.PredictNextGame()
		if err != nil {
			fmt.Printf("Error generating prediction: %v\n", err)
			fmt.Fprintf(w, `<div class="prediction-error">Unable to generate prediction: %v</div>`, err)
			return
		}
		cachedPrediction = prediction
	}

	html := formatPredictionHTML(prediction)
	fmt.Fprint(w, html)
}

// formatPredictionHTML creates comprehensive HTML for the prediction widget with advanced hockey analytics
func formatPredictionHTML(prediction *models.GamePrediction) string {
	// Determine winner display
	var winnerDisplay, winnerTeam string

	if prediction.Prediction.Winner == prediction.HomeTeam.Code {
		winnerTeam = prediction.HomeTeam.Name
		winnerDisplay = fmt.Sprintf("%s (Home)", winnerTeam)
	} else {
		winnerTeam = prediction.AwayTeam.Name
		winnerDisplay = fmt.Sprintf("%s (Away)", winnerTeam)
	}

	// Get confidence color and game type icon
	confidenceColor := getConfidenceColor(prediction.Confidence)
	gameTypeIcon := getGameTypeIcon(prediction.Prediction.GameType)

	// NEW: Build Advanced Analytics Dashboard
	advancedAnalyticsHTML := buildAdvancedAnalyticsDashboard(prediction)

	// Build situational factors HTML
	situationalHTML := buildSituationalAnalysisHTML(prediction.KeyFactors)

	// Build model analysis HTML
	modelAnalysisHTML := buildModelAnalysisHTML(prediction.Prediction.ModelResults)

	// Build key factors HTML
	keyFactorsHTML := buildKeyFactorsHTML(prediction.KeyFactors)

	html := fmt.Sprintf(`
	<div class="ai-prediction-widget-enhanced">
		<div class="prediction-header">
			<h3>🎯 Advanced AI Prediction</h3>
			<div class="confidence-badge" style="background-color: %s">
				%.0f%% Confidence
			</div>
		</div>

		<div class="prediction-matchup">
			<div class="team away-team">
				<div class="team-name">✈️ %s</div>
				<div class="team-form">Form: %s</div>
				<div class="team-streak">%s</div>
				<div class="win-probability">%.1f%% chance</div>
			</div>
			
			<div class="prediction-center">
				<div class="predicted-score">%s</div>
				<div class="vs-divider">VS</div>
				<div class="game-date">%s</div>
				<div class="ensemble-method">%s</div>
			</div>
			
			<div class="team home-team">
				<div class="team-name">🏠 %s</div>
				<div class="team-form">Form: %s</div>
				<div class="team-streak">%s</div>
				<div class="win-probability">%.1f%% chance</div>
			</div>
		</div>

		<div class="prediction-result">
			<div class="prediction-winner">
				%s <strong>%s</strong> to win %s
			</div>
			<div class="predicted-final">Predicted Score: <strong>%s</strong></div>
		</div>

		%s <!-- NEW: Advanced Analytics Dashboard -->
		
		%s <!-- Situational Analysis -->
		
		%s <!-- Model Analysis -->
		
		%s <!-- Key Strategic Factors -->

		<div class="prediction-footer">
			<small>🤖 Generated: %s | Game Type: %s %s | Ensemble: %s</small>
		</div>
	</div>

	<style>
	.ai-prediction-widget-enhanced {
		background: linear-gradient(135deg, #1a1a1a 0%%, #2d2d2d 100%%);
		color: #ffffff;
		border-radius: 16px;
		padding: 24px;
		margin: 12px 0;
		border: 2px solid rgba(76, 175, 80, 0.3);
		box-shadow: 0 12px 40px rgba(0,0,0,0.4);
		font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
	}

	.prediction-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 20px;
		padding-bottom: 15px;
		border-bottom: 2px solid rgba(76, 175, 80, 0.2);
	}

	.prediction-header h3 {
		margin: 0;
		color: #4CAF50;
		font-size: 1.4em;
		font-weight: 600;
	}

	.confidence-badge {
		padding: 8px 16px;
		border-radius: 20px;
		font-weight: bold;
		font-size: 0.9em;
		color: white;
		text-shadow: 0 1px 2px rgba(0,0,0,0.5);
	}

	.prediction-matchup {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		gap: 20px;
		margin-bottom: 20px;
		align-items: center;
	}

	.team {
		text-align: center;
		padding: 15px;
		background: rgba(255,255,255,0.05);
		border-radius: 12px;
		border: 1px solid rgba(255,255,255,0.1);
	}

	.team-name {
		font-size: 1.1em;
		font-weight: bold;
		margin-bottom: 8px;
		color: #4CAF50;
	}

	.team-form, .team-streak {
		font-size: 0.85em;
		margin: 4px 0;
		color: #cccccc;
	}

	.win-probability {
		font-size: 0.9em;
		font-weight: bold;
		color: #FFC107;
		margin-top: 8px;
	}

	.prediction-center {
		text-align: center;
		padding: 15px;
	}

	.predicted-score {
		font-size: 2.2em;
		font-weight: bold;
		color: #4CAF50;
		margin-bottom: 8px;
		text-shadow: 0 2px 4px rgba(0,0,0,0.5);
	}

	.vs-divider {
		font-size: 0.9em;
		color: #999;
		margin: 5px 0;
	}

	.game-date {
		font-size: 0.85em;
		color: #ccc;
		margin: 5px 0;
	}

	.ensemble-method {
		font-size: 0.8em;
		color: #4CAF50;
		font-style: italic;
	}

	.prediction-result {
		text-align: center;
		padding: 20px;
		background: rgba(76, 175, 80, 0.1);
		border-radius: 12px;
		margin-bottom: 20px;
		border: 1px solid rgba(76, 175, 80, 0.3);
	}

	.prediction-winner {
		font-size: 1.2em;
		margin-bottom: 8px;
		color: #4CAF50;
	}

	.predicted-final {
		font-size: 1.1em;
		color: #ffffff;
	}

	/* NEW: Advanced Analytics Dashboard Styles */
	.advanced-analytics-dashboard {
		background: linear-gradient(135deg, #0f1419 0%%, #1a2332 100%%);
		border-radius: 12px;
		padding: 20px;
		margin: 20px 0;
		border: 2px solid rgba(33, 150, 243, 0.3);
	}

	.analytics-header {
		text-align: center;
		margin-bottom: 20px;
		color: #2196F3;
		font-size: 1.2em;
		font-weight: bold;
	}

	.analytics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 16px;
		margin-bottom: 20px;
	}

	.analytics-card {
		background: rgba(33, 150, 243, 0.1);
		border-radius: 10px;
		padding: 16px;
		border: 1px solid rgba(33, 150, 243, 0.2);
	}

	.analytics-card-title {
		font-size: 0.9em;
		color: #2196F3;
		font-weight: bold;
		margin-bottom: 12px;
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.analytics-metric {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin: 8px 0;
		padding: 6px 0;
		border-bottom: 1px solid rgba(255,255,255,0.1);
	}

	.analytics-metric:last-child {
		border-bottom: none;
	}

	.metric-label {
		font-size: 0.85em;
		color: #cccccc;
	}

	.metric-value {
		font-weight: bold;
		color: #ffffff;
	}

	.metric-bar {
		width: 100%%;
		height: 6px;
		background: rgba(255,255,255,0.1);
		border-radius: 3px;
		margin-top: 4px;
		overflow: hidden;
	}

	.metric-fill {
		height: 100%%;
		background: linear-gradient(90deg, #2196F3, #4CAF50);
		border-radius: 3px;
		transition: width 0.3s ease;
	}

	.rating-display {
		text-align: center;
		background: rgba(76, 175, 80, 0.2);
		border-radius: 8px;
		padding: 12px;
		margin-top: 12px;
	}

	.rating-number {
		font-size: 1.8em;
		font-weight: bold;
		color: #4CAF50;
	}

	.rating-label {
		font-size: 0.8em;
		color: #cccccc;
		margin-top: 4px;
	}

	/* Responsive Design */
	@media (max-width: 768px) {
		.prediction-matchup {
			grid-template-columns: 1fr;
			gap: 12px;
		}
		
		.analytics-grid {
			grid-template-columns: 1fr;
		}
		
		.predicted-score {
			font-size: 1.8em;
		}
	}
	</style>
	`,
		confidenceColor, prediction.Confidence*100,
		prediction.AwayTeam.Name, prediction.AwayTeam.RecentForm, prediction.AwayTeam.Streak, (1-prediction.Prediction.WinProbability)*100,
		prediction.Prediction.PredictedScore, prediction.GameDate.Format("Jan 2, 3:04 PM"), prediction.Prediction.EnsembleMethod,
		prediction.HomeTeam.Name, prediction.HomeTeam.RecentForm, prediction.HomeTeam.Streak, prediction.Prediction.WinProbability*100,
		gameTypeIcon, winnerTeam, winnerDisplay, prediction.Prediction.PredictedScore,
		advancedAnalyticsHTML,
		situationalHTML,
		modelAnalysisHTML,
		keyFactorsHTML,
		prediction.GeneratedAt.Format("3:04 PM"), prediction.Prediction.GameType, gameTypeIcon, prediction.Prediction.EnsembleMethod)

	return html
}

// NEW: buildAdvancedAnalyticsDashboard creates a comprehensive analytics dashboard
func buildAdvancedAnalyticsDashboard(prediction *models.GamePrediction) string {
	// Get real advanced analytics from the AdvancedAnalyticsService
	analyticsService := services.NewAdvancedAnalyticsService()

	homeTeamAnalytics, err := analyticsService.GetAdvancedAnalytics(prediction.HomeTeam.Code, true)
	if err != nil {
		// Fallback to basic analytics if service fails
		homeTeamAnalytics = createFallbackAnalytics(prediction.HomeTeam.Code, true)
	}

	awayTeamAnalytics, err := analyticsService.GetAdvancedAnalytics(prediction.AwayTeam.Code, false)
	if err != nil {
		// Fallback to basic analytics if service fails
		awayTeamAnalytics = createFallbackAnalytics(prediction.AwayTeam.Code, false)
	}

	return fmt.Sprintf(`
	<div class="advanced-analytics-dashboard">
		<div class="analytics-header">
			📊 Advanced Hockey Analytics Dashboard
		</div>
		
		<div class="analytics-grid">
			<!-- Expected Goals Analysis -->
			<div class="analytics-card">
				<div class="analytics-card-title">
					🎯 Expected Goals (xG)
				</div>
				<div class="analytics-metric">
					<span class="metric-label">%s xG/Game</span>
					<span class="metric-value">%.2f</span>
				</div>
				<div class="metric-bar">
					<div class="metric-fill" style="width: %.0f%%"></div>
				</div>
				<div class="analytics-metric">
					<span class="metric-label">%s xG/Game</span>
					<span class="metric-value">%.2f</span>
				</div>
				<div class="metric-bar">
					<div class="metric-fill" style="width: %.0f%%"></div>
				</div>
				<div class="analytics-metric">
					<span class="metric-label">xG Differential</span>
					<span class="metric-value">%+.2f</span>
				</div>
			</div>
			
			<!-- Possession Dominance -->
			<div class="analytics-card">
				<div class="analytics-card-title">
					💪 Possession Metrics
				</div>
				<div class="analytics-metric">
					<span class="metric-label">%s Corsi %%</span>
					<span class="metric-value">%.1f%%</span>
				</div>
				<div class="metric-bar">
					<div class="metric-fill" style="width: %.0f%%"></div>
				</div>
				<div class="analytics-metric">
					<span class="metric-label">%s Corsi %%</span>
					<span class="metric-value">%.1f%%</span>
				</div>
				<div class="metric-bar">
					<div class="metric-fill" style="width: %.0f%%"></div>
				</div>
				<div class="analytics-metric">
					<span class="metric-label">High Danger %%</span>
					<span class="metric-value">%.1f%% vs %.1f%%</span>
				</div>
			</div>
			
			<!-- Goaltending Performance -->
			<div class="analytics-card">
				<div class="analytics-card-title">
					🥅 Goaltending Analytics
				</div>
				<div class="analytics-metric">
					<span class="metric-label">%s Save %%</span>
					<span class="metric-value">%.3f</span>
				</div>
				<div class="metric-bar">
					<div class="metric-fill" style="width: %.0f%%"></div>
				</div>
				<div class="analytics-metric">
					<span class="metric-label">%s Save %%</span>
					<span class="metric-value">%.3f</span>
				</div>
				<div class="metric-bar">
					<div class="metric-fill" style="width: %.0f%%"></div>
				</div>
				<div class="analytics-metric">
					<span class="metric-label">Saves Above Expected</span>
					<span class="metric-value">%+.1f vs %+.1f</span>
				</div>
			</div>
			
			<!-- Overall Team Ratings -->
			<div class="analytics-card">
				<div class="analytics-card-title">
					⭐ Team Ratings
				</div>
				<div class="rating-display">
					<div class="rating-number">%.0f</div>
					<div class="rating-label">%s Overall Rating</div>
				</div>
				<div class="rating-display">
					<div class="rating-number">%.0f</div>
					<div class="rating-label">%s Overall Rating</div>
				</div>
				<div class="analytics-metric">
					<span class="metric-label">Rating Advantage</span>
					<span class="metric-value">%+.1f</span>
				</div>
			</div>
		</div>
		
		<!-- Team Strengths & Weaknesses -->
		<div class="analytics-grid">
			<div class="analytics-card">
				<div class="analytics-card-title">
					💪 %s Strengths
				</div>
				%s
			</div>
			<div class="analytics-card">
				<div class="analytics-card-title">
					⚠️ %s Weaknesses  
				</div>
				%s
			</div>
		</div>
	</div>
	`,
		// Expected Goals data
		prediction.HomeTeam.Name, homeTeamAnalytics.XGForPerGame, homeTeamAnalytics.XGForPerGame*20,
		prediction.AwayTeam.Name, awayTeamAnalytics.XGForPerGame, awayTeamAnalytics.XGForPerGame*20,
		homeTeamAnalytics.XGDifferential,

		// Possession data
		prediction.HomeTeam.Name, homeTeamAnalytics.CorsiForPct*100, homeTeamAnalytics.CorsiForPct*100,
		prediction.AwayTeam.Name, awayTeamAnalytics.CorsiForPct*100, awayTeamAnalytics.CorsiForPct*100,
		homeTeamAnalytics.HighDangerPct*100, awayTeamAnalytics.HighDangerPct*100,

		// Goaltending data
		prediction.HomeTeam.Name, homeTeamAnalytics.GoalieSvPctOverall, homeTeamAnalytics.GoalieSvPctOverall*100,
		prediction.AwayTeam.Name, awayTeamAnalytics.GoalieSvPctOverall, awayTeamAnalytics.GoalieSvPctOverall*100,
		homeTeamAnalytics.SavesAboveExpected, awayTeamAnalytics.SavesAboveExpected,

		// Overall ratings
		homeTeamAnalytics.OverallRating, prediction.HomeTeam.Name,
		awayTeamAnalytics.OverallRating, prediction.AwayTeam.Name,
		homeTeamAnalytics.OverallRating-awayTeamAnalytics.OverallRating,

		// Strengths and weaknesses
		prediction.HomeTeam.Name, formatStrengthsList(homeTeamAnalytics.StrengthAreas),
		prediction.AwayTeam.Name, formatStrengthsList(awayTeamAnalytics.WeaknessAreas))
}

// createFallbackAnalytics creates basic analytics when service is unavailable
// This is a fallback only - the real AdvancedAnalyticsService uses NHL API data
func createFallbackAnalytics(teamCode string, isHome bool) *models.AdvancedAnalytics {
	// Return league-average analytics as safe fallback
	return &models.AdvancedAnalytics{
		XGForPerGame:        2.8,   // League average
		XGAgainstPerGame:    2.8,   // League average
		XGDifferential:      0.0,   // Neutral
		ShootingTalent:      0.0,   // Average
		GoaltendingTalent:   0.0,   // Average
		CorsiForPct:         0.50,  // Even possession
		FenwickForPct:       0.50,  // Even possession
		HighDangerPct:       0.50,  // Even chances
		PossessionQuality:   50.0,  // Average
		ShotGenerationRate:  30.0,  // League average
		ShotSuppressionRate: 30.0,  // League average
		ShotQualityFor:      0.10,  // Average
		ShotQualityAgainst:  0.10,  // Average
		PowerPlayXG:         1.5,   // Average
		PenaltyKillXGA:      1.8,   // Average
		SpecialTeamsEdge:    0.0,   // Neutral
		GoalieSvPctOverall:  0.910, // League average
		GoalieSvPctHD:       0.850, // League average
		SavesAboveExpected:  0.0,   // Average
		GoalieWorkload:      50.0,  // Average
		LeadingPerformance:  50.0,  // Average
		TrailingPerformance: 50.0,  // Average
		CloseGameRecord:     0.50,  // .500 record
		OverallRating:       50.0,  // Average rating
		StrengthAreas:       []string{"Balanced Team"},
		WeaknessAreas:       []string{"No Data Available"},
	}
}

// formatStrengthsList formats a list of strengths/weaknesses for HTML display
func formatStrengthsList(items []string) string {
	html := ""
	for _, item := range items {
		html += fmt.Sprintf(`
		<div class="analytics-metric">
			<span class="metric-label">✓ %s</span>
			<span class="metric-value">●</span>
		</div>`, item)
	}
	return html
}

// Helper functions for building UI sections

// buildSituationalAnalysisHTML creates the situational factors analysis section
func buildSituationalAnalysisHTML(keyFactors []string) string {
	var html strings.Builder

	html.WriteString(`<div class="situational-analysis">`)
	html.WriteString(`<h4 class="section-title">🔍 Situational Intelligence Analysis</h4>`)
	html.WriteString(`<div class="situational-grid">`)

	// Travel & Geography Analysis
	html.WriteString(`<div class="situational-card travel-card">`)
	html.WriteString(`<div class="card-header"><span class="card-icon">✈️</span><span class="card-title">Travel & Geography</span></div>`)
	html.WriteString(`<div class="card-content">`)

	travelFactors := []string{}
	altitudeFactors := []string{}
	for _, factor := range keyFactors {
		if strings.Contains(factor, "travel fatigue") || strings.Contains(factor, "miles") || strings.Contains(factor, "time zones") {
			travelFactors = append(travelFactors, factor)
		}
		if strings.Contains(factor, "altitude") || strings.Contains(factor, "ft difference") {
			altitudeFactors = append(altitudeFactors, factor)
		}
	}

	if len(travelFactors) > 0 {
		for _, factor := range travelFactors {
			html.WriteString(fmt.Sprintf(`<div class="factor-item">%s</div>`, factor))
		}
	}
	if len(altitudeFactors) > 0 {
		for _, factor := range altitudeFactors {
			html.WriteString(fmt.Sprintf(`<div class="factor-item">%s</div>`, factor))
		}
	}
	if len(travelFactors) == 0 && len(altitudeFactors) == 0 {
		html.WriteString(`<div class="factor-item neutral">🟢 No significant travel or altitude impact</div>`)
	}

	html.WriteString(`</div></div>`)

	// Schedule & Rest Analysis
	html.WriteString(`<div class="situational-card schedule-card">`)
	html.WriteString(`<div class="card-header"><span class="card-icon">📅</span><span class="card-title">Schedule & Rest</span></div>`)
	html.WriteString(`<div class="card-content">`)

	scheduleFactors := []string{}
	restFactors := []string{}
	for _, factor := range keyFactors {
		if strings.Contains(factor, "schedule") || strings.Contains(factor, "games in") || strings.Contains(factor, "heavy") {
			scheduleFactors = append(scheduleFactors, factor)
		}
		if strings.Contains(factor, "back-to-back") || strings.Contains(factor, "rest") {
			restFactors = append(restFactors, factor)
		}
	}

	if len(scheduleFactors) > 0 {
		for _, factor := range scheduleFactors {
			html.WriteString(fmt.Sprintf(`<div class="factor-item">%s</div>`, factor))
		}
	}
	if len(restFactors) > 0 {
		for _, factor := range restFactors {
			html.WriteString(fmt.Sprintf(`<div class="factor-item">%s</div>`, factor))
		}
	}
	if len(scheduleFactors) == 0 && len(restFactors) == 0 {
		html.WriteString(`<div class="factor-item neutral">🟢 Normal schedule density</div>`)
	}

	html.WriteString(`</div></div>`)

	// Health & Roster Analysis
	html.WriteString(`<div class="situational-card health-card">`)
	html.WriteString(`<div class="card-header"><span class="card-icon">🏥</span><span class="card-title">Health & Roster</span></div>`)
	html.WriteString(`<div class="card-content">`)

	injuryFactors := []string{}
	for _, factor := range keyFactors {
		if strings.Contains(factor, "injuries") || strings.Contains(factor, "goalie") || strings.Contains(factor, "players out") {
			injuryFactors = append(injuryFactors, factor)
		}
	}

	if len(injuryFactors) > 0 {
		for _, factor := range injuryFactors {
			html.WriteString(fmt.Sprintf(`<div class="factor-item">%s</div>`, factor))
		}
	} else {
		html.WriteString(`<div class="factor-item neutral">🟢 Teams relatively healthy</div>`)
	}

	html.WriteString(`</div></div>`)

	// Momentum & Psychology Analysis
	html.WriteString(`<div class="situational-card momentum-card">`)
	html.WriteString(`<div class="card-header"><span class="card-icon">🔥</span><span class="card-title">Momentum & Psychology</span></div>`)
	html.WriteString(`<div class="card-content">`)

	momentumFactors := []string{}
	for _, factor := range keyFactors {
		if strings.Contains(factor, "momentum") || strings.Contains(factor, "streak") || strings.Contains(factor, "blowouts") || strings.Contains(factor, "riding") || strings.Contains(factor, "struggling") {
			momentumFactors = append(momentumFactors, factor)
		}
	}

	if len(momentumFactors) > 0 {
		for _, factor := range momentumFactors {
			html.WriteString(fmt.Sprintf(`<div class="factor-item">%s</div>`, factor))
		}
	} else {
		html.WriteString(`<div class="factor-item neutral">🟡 Balanced momentum</div>`)
	}

	html.WriteString(`</div></div>`)
	html.WriteString(`</div></div>`) // Close grid and section

	return html.String()
}

// buildModelAnalysisHTML creates the AI model analysis section
func buildModelAnalysisHTML(modelResults []models.ModelResult) string {
	if len(modelResults) == 0 {
		return ""
	}

	var html strings.Builder

	html.WriteString(`<div class="model-analysis">`)
	html.WriteString(`<h4 class="section-title">🤖 AI Model Analysis</h4>`)
	html.WriteString(`<div class="models-grid">`)

	for _, mr := range modelResults {
		confidenceClass := "confidence-low"
		if mr.Confidence > 0.7 {
			confidenceClass = "confidence-high"
		} else if mr.Confidence > 0.5 {
			confidenceClass = "confidence-medium"
		}

		html.WriteString(fmt.Sprintf(`
			<div class="model-card">
				<div class="model-header">
					<span class="model-name">%s</span>
					<span class="model-confidence %s">%.1f%%</span>
				</div>
				<div class="model-details">
					<div>Prediction: <strong>%s</strong></div>
					<div>Weight: %.0f%%</div>
					<div>Processing: %dms</div>
				</div>
			</div>
		`, mr.ModelName, confidenceClass, mr.Confidence*100, mr.PredictedScore, mr.Weight*100, mr.ProcessingTime))
	}

	html.WriteString(`</div></div>`)
	return html.String()
}

// buildKeyFactorsHTML creates the key strategic factors section
func buildKeyFactorsHTML(keyFactors []string) string {
	var html strings.Builder

	html.WriteString(`<div class="key-factors-section">`)
	html.WriteString(`<h4 class="section-title">🔑 Key Strategic Factors</h4>`)
	html.WriteString(`<ul class="key-factors-list">`)

	for _, factor := range keyFactors {
		html.WriteString(fmt.Sprintf(`<li class="key-factor-item">%s</li>`, factor))
	}

	html.WriteString(`</ul></div>`)
	return html.String()
}

// Helper functions for formatting
func getConfidenceColor(confidence float64) string {
	if confidence > 0.8 {
		return "#4CAF50" // Green - High confidence
	} else if confidence > 0.6 {
		return "#FF9800" // Orange - Medium confidence
	} else {
		return "#f44336" // Red - Low confidence
	}
}

func getGameTypeIcon(gameType string) string {
	switch gameType {
	case "blowout":
		return "💥"
	case "toss-up":
		return "🎲"
	case "close":
		return "⚔️"
	default:
		return "🏒"
	}
}

// HandleLeagueWidePredictions returns all stored predictions
func HandleLeagueWidePredictions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	predictionStorage := services.GetPredictionStorageService()
	if predictionStorage == nil {
		http.Error(w, `{"error": "Prediction Storage Service not initialized"}`, http.StatusInternalServerError)
		return
	}

	predictions, err := predictionStorage.GetAllPredictions()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(predictions)
}

// HandlePredictionAccuracy returns accuracy statistics
func HandlePredictionAccuracy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	predictionStorage := services.GetPredictionStorageService()
	if predictionStorage == nil {
		http.Error(w, `{"error": "Prediction Storage Service not initialized"}`, http.StatusInternalServerError)
		return
	}

	stats := predictionStorage.GetAccuracyStats()
	json.NewEncoder(w).Encode(stats)
}

// HandleDailyPredictionStats returns daily prediction service statistics
func HandleDailyPredictionStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dailyPredictionService := services.GetDailyPredictionService()
	if dailyPredictionService == nil {
		http.Error(w, `{"error": "Daily Prediction Service not initialized"}`, http.StatusInternalServerError)
		return
	}

	stats := dailyPredictionService.GetStats()
	json.NewEncoder(w).Encode(stats)
}

// HandlePredictionsStatsPopup returns the HTML for the predictions stats popup
func HandlePredictionsStatsPopup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	predictionStorage := services.GetPredictionStorageService()
	if predictionStorage == nil {
		fmt.Fprint(w, `<div class="stats-error">Prediction stats unavailable</div>`)
		return
	}

	// Get all predictions
	allPredictions, err := predictionStorage.GetAllPredictions()
	if err != nil {
		fmt.Fprint(w, `<div class="stats-error">Failed to load predictions</div>`)
		return
	}

	// Get accuracy stats
	accuracyStats := predictionStorage.GetAccuracyStats()

	// Generate HTML
	html := generatePredictionsPopupHTML(allPredictions, accuracyStats)
	fmt.Fprint(w, html)
}

// generatePredictionsPopupHTML generates the HTML for the predictions stats popup
func generatePredictionsPopupHTML(allPredictions []*services.StoredPrediction, accuracyStats interface{}) string {
	// Count upcoming and completed games
	upcomingCount := 0
	completedCount := 0
	correctCount := 0

	for _, pred := range allPredictions {
		if pred.ActualResult == nil {
			upcomingCount++
		} else {
			completedCount++
			if pred.Accuracy != nil && pred.Accuracy.WinnerCorrect {
				correctCount++
			}
		}
	}

	// Calculate accuracy percentage
	accuracyPct := 0.0
	if completedCount > 0 {
		accuracyPct = float64(correctCount) / float64(completedCount) * 100
	}

	// Use strings.Builder for efficient string concatenation
	var html strings.Builder
	html.WriteString(`
<div class="system-stats-popup">
	<div class="stats-header">
		<h3>📊 Prediction Statistics</h3>
		<button class="close-popup" onclick="closePredictionsPopup()">✕</button>
	</div>
	
	<div class="stats-section">
		<h4>📈 Overall Accuracy</h4>
		<div class="stats-grid">
			<div class="stat-item">
				<span class="stat-label">Total Predictions:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", len(allPredictions)) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Completed Games:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", completedCount) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Correct Predictions:</span>
				<span class="stat-value stat-success">` + fmt.Sprintf("%d", correctCount) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Accuracy Rate:</span>
				<span class="stat-value stat-success">` + fmt.Sprintf("%.1f%%", accuracyPct) + `</span>
			</div>
			<div class="stat-item">
				<span class="stat-label">Upcoming Games:</span>
				<span class="stat-value">` + fmt.Sprintf("%d", upcomingCount) + `</span>
			</div>
		</div>
	</div>

	<div class="stats-section">
		<h4>🎯 Upcoming Predictions</h4>
		<div class="predictions-list">`)

	// Add upcoming predictions
	upcomingAdded := 0
	for _, pred := range allPredictions {
		if pred.ActualResult == nil && upcomingAdded < 10 {
			winningTeam := pred.HomeTeam
			winProb := pred.Prediction.HomeTeam.WinProbability
			if pred.Prediction.AwayTeam.WinProbability > pred.Prediction.HomeTeam.WinProbability {
				winningTeam = pred.AwayTeam
				winProb = pred.Prediction.AwayTeam.WinProbability
			}

			html.WriteString(`
			<div class="prediction-item">
				<div class="prediction-game">`)
			html.WriteString(pred.AwayTeam + ` @ ` + pred.HomeTeam)
			html.WriteString(`</div>
				<div class="prediction-details">
					<span class="prediction-winner">Predicted: `)
			html.WriteString(winningTeam)
			html.WriteString(`</span>
					<span class="prediction-confidence">`)
			html.WriteString(fmt.Sprintf("%.1f%% confidence", winProb*100))
			html.WriteString(`</span>
				</div>
				<div class="prediction-date">`)
			html.WriteString(pred.GameDate.Format("Jan 2, 2006"))
			html.WriteString(`</div>
			</div>`)
			upcomingAdded++
		}
	}

	if upcomingAdded == 0 {
		html.WriteString(`<p style="text-align: center; color: #aaa; font-style: italic;">No upcoming predictions</p>`)
	}

	html.WriteString(`
		</div>
	</div>

	<div class="stats-section">
		<h4>✅ Recent Results</h4>
		<div class="predictions-list">`)

	// Add recent completed predictions
	recentAdded := 0
	for i := len(allPredictions) - 1; i >= 0 && recentAdded < 10; i-- {
		pred := allPredictions[i]
		if pred.ActualResult != nil {
			isCorrect := pred.Accuracy != nil && pred.Accuracy.WinnerCorrect
			correctIcon := "❌"
			correctClass := "incorrect"
			if isCorrect {
				correctIcon = "✅"
				correctClass = "correct"
			}

			winningTeam := pred.HomeTeam
			winProb := pred.Prediction.HomeTeam.WinProbability
			if pred.Prediction.AwayTeam.WinProbability > pred.Prediction.HomeTeam.WinProbability {
				winningTeam = pred.AwayTeam
				winProb = pred.Prediction.AwayTeam.WinProbability
			}

			actualWinner := pred.ActualResult.WinningTeam

			html.WriteString(`
			<div class="prediction-item `)
			html.WriteString(correctClass)
			html.WriteString(`">
				<div class="prediction-game">`)
			html.WriteString(pred.AwayTeam + ` @ ` + pred.HomeTeam + ` ` + correctIcon)
			html.WriteString(`</div>
				<div class="prediction-details">
					<span class="prediction-winner">Predicted: `)
			html.WriteString(winningTeam + ` (` + fmt.Sprintf("%.1f%%", winProb*100) + `)`)
			html.WriteString(`</span>
					<span class="prediction-actual">Actual: `)
			html.WriteString(actualWinner)
			html.WriteString(`</span>
				</div>
				<div class="prediction-date">`)
			html.WriteString(pred.GameDate.Format("Jan 2"))
			html.WriteString(`</div>
			</div>`)
			recentAdded++
		}
	}

	if recentAdded == 0 {
		html.WriteString(`<p style="text-align: center; color: #aaa; font-style: italic;">No completed predictions yet</p>`)
	}

	html.WriteString(`
		</div>
	</div>
</div>
`)

	return html.String()
}

// HandleTriggerDailyPredictions manually triggers the daily prediction generation
