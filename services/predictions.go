package services

import (
	"fmt"
	"math"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// PredictionService handles game predictions using ensemble of AI models
type PredictionService struct {
	teamCode        string
	ensembleService *EnsemblePredictionService
}

// NewPredictionService creates a new prediction service
func NewPredictionService(teamCode string) *PredictionService {
	return &PredictionService{
		teamCode:        teamCode,
		ensembleService: NewEnsemblePredictionService(teamCode),
	}
}

// PredictNextGame generates AI prediction for the team's next upcoming game
func (ps *PredictionService) PredictNextGame() (*models.GamePrediction, error) {
	fmt.Printf("🚀 Generating advanced AI prediction for %s next game...\n", ps.teamCode)

	// Get upcoming games
	games, err := GetTeamUpcomingGames(ps.teamCode)
	if err != nil {
		return nil, fmt.Errorf("error getting upcoming games: %v", err)
	}

	if len(games) == 0 {
		return nil, fmt.Errorf("no upcoming games found for %s", ps.teamCode)
	}

	nextGame := games[0]
	fmt.Printf("🏒 Analyzing matchup: %s @ %s on %s\n",
		nextGame.AwayTeam.CommonName.Default,
		nextGame.HomeTeam.CommonName.Default,
		nextGame.GameDate)

	// Fetch enhanced factors for both teams using situational analysis
	analyzer := NewSituationalAnalyzer(ps.teamCode)
	homeFactors, err := analyzer.AnalyzeSituationalFactors(nextGame.HomeTeam.Abbrev, nextGame.AwayTeam.Abbrev, nextGame.Venue.Default, true)
	if err != nil {
		return nil, fmt.Errorf("error getting home team factors: %v", err)
	}
	awayFactors, err := analyzer.AnalyzeSituationalFactors(nextGame.AwayTeam.Abbrev, nextGame.HomeTeam.Abbrev, nextGame.Venue.Default, false)
	if err != nil {
		return nil, fmt.Errorf("error getting away team factors: %v", err)
	}

	// Run ensemble prediction
	prediction, err := ps.ensembleService.PredictGame(homeFactors, awayFactors)
	if err != nil {
		return nil, fmt.Errorf("ensemble prediction failed: %v", err)
	}

	// Create full prediction object
	homeWinProb := prediction.WinProbability
	awayWinProb := 1.0 - prediction.WinProbability
	if prediction.Winner == nextGame.AwayTeam.Abbrev {
		homeWinProb = 1.0 - prediction.WinProbability
		awayWinProb = prediction.WinProbability
	}

	gamePrediction := &models.GamePrediction{
		GameID: 0, // Schedule games don't have IDs
		GameDate: func() time.Time {
			if t, err := time.Parse("2006-01-02", nextGame.GameDate); err == nil {
				return t
			}
			return time.Now()
		}(),
		HomeTeam: models.PredictionTeam{
			Code:           nextGame.HomeTeam.Abbrev,
			Name:           nextGame.HomeTeam.CommonName.Default,
			WinProbability: homeWinProb,
			ExpectedGoals:  homeFactors.GoalsFor,
			RecentForm:     ps.getRecentFormString(homeFactors.TeamCode),
			Streak:         ps.getCurrentStreak(homeFactors.TeamCode),
		},
		AwayTeam: models.PredictionTeam{
			Code:           nextGame.AwayTeam.Abbrev,
			Name:           nextGame.AwayTeam.CommonName.Default,
			WinProbability: awayWinProb,
			ExpectedGoals:  awayFactors.GoalsFor,
			RecentForm:     ps.getRecentFormString(awayFactors.TeamCode),
			Streak:         ps.getCurrentStreak(awayFactors.TeamCode),
		},
		Prediction:  *prediction,
		Confidence:  prediction.Confidence,
		KeyFactors:  ps.generateAdvancedKeyFactors(homeFactors, awayFactors, prediction),
		GeneratedAt: time.Now(),
	}

	return gamePrediction, nil
}

// getEnhancedPredictionFactors fetches comprehensive prediction factors
func (ps *PredictionService) getEnhancedPredictionFactors(teamCode, opponentCode string, isHome bool) (*models.PredictionFactors, error) {
	fmt.Printf("📊 Calculating enhanced factors for %s vs %s (home: %t)\n", teamCode, opponentCode, isHome)

	// Get standings for basic stats
	standings, err := GetStandings()
	if err != nil {
		return nil, fmt.Errorf("error getting standings: %v", err)
	}

	// Find team in standings
	var teamStanding *models.TeamStanding
	for _, standing := range standings.Standings {
		if standing.TeamAbbrev.Default == teamCode {
			teamStanding = &standing
			break
		}
	}

	if teamStanding == nil {
		return nil, fmt.Errorf("team %s not found in standings", teamCode)
	}

	// Calculate advanced factors
	factors := &models.PredictionFactors{
		TeamCode:          teamCode,
		WinPercentage:     float64(teamStanding.Wins) / float64(teamStanding.GamesPlayed),
		HomeAdvantage:     ps.calculateHomeAdvantage(teamCode, isHome),
		RecentForm:        ps.calculateAdvancedRecentForm(teamCode),
		HeadToHead:        ps.calculateHeadToHead(teamCode, opponentCode),
		GoalsFor:          float64(teamStanding.GoalFor) / float64(teamStanding.GamesPlayed),
		GoalsAgainst:      float64(teamStanding.GoalAgainst) / float64(teamStanding.GamesPlayed),
		PowerPlayPct:      ps.estimatePowerPlayPct(teamCode),
		PenaltyKillPct:    ps.estimatePenaltyKillPct(teamCode),
		RestDays:          ps.calculateRestDays(teamCode),
		BackToBackPenalty: ps.calculateBackToBackPenalty(teamCode),
	}

	return factors, nil
}

// Enhanced calculation methods
func (ps *PredictionService) calculateHomeAdvantage(teamCode string, isHome bool) float64 {
	if !isHome {
		return 0.0
	}

	// Different teams have different home advantages
	baseAdvantage := 0.12 // League average

	// Some teams have stronger home advantages (simplified logic)
	strongHomeTeams := []string{"MTL", "BOS", "CGY", "EDM", "WPG"}
	for _, team := range strongHomeTeams {
		if team == teamCode {
			return baseAdvantage + 0.03 // Extra 3% for strong home teams
		}
	}

	return baseAdvantage
}

func (ps *PredictionService) calculateAdvancedRecentForm(teamCode string) float64 {
	// Enhanced recent form calculation (still simplified)
	// In reality, would check last 10 games from API

	// Simulate recent form based on team characteristics
	hashVal := 0
	for _, r := range teamCode {
		hashVal += int(r)
	}

	// Create variation between 0.3 and 0.7
	form := 0.3 + float64(hashVal%40)/100.0
	return form
}

func (ps *PredictionService) calculateHeadToHead(teamCode, opponentCode string) float64 {
	// Simplified head-to-head calculation
	// In reality, would fetch historical matchup data

	// Create some variation based on team codes
	hashDiff := 0
	for i, r := range teamCode {
		if i < len(opponentCode) {
			hashDiff += int(r) - int(rune(opponentCode[i]))
		}
	}

	// Normalize to -0.2 to +0.2
	h2h := float64(hashDiff%40-20) / 100.0
	return h2h
}

func (ps *PredictionService) estimatePowerPlayPct(teamCode string) float64 {
	// Estimate power play percentage (league average ~20%)
	hashVal := 0
	for _, r := range teamCode {
		hashVal += int(r)
	}

	// Vary between 15% and 25%
	return 0.15 + float64(hashVal%10)/100.0
}

func (ps *PredictionService) estimatePenaltyKillPct(teamCode string) float64 {
	// Estimate penalty kill percentage (league average ~80%)
	hashVal := 0
	for _, r := range teamCode {
		hashVal += int(r) * 2
	}

	// Vary between 75% and 85%
	return 0.75 + float64(hashVal%10)/100.0
}

func (ps *PredictionService) calculateRestDays(teamCode string) int {
	// Simplified rest calculation
	// In reality, would check team's last game date
	return 1 + (len(teamCode) % 3)
}

func (ps *PredictionService) calculateBackToBackPenalty(teamCode string) float64 {
	// Check if team is on back-to-back games
	restDays := ps.calculateRestDays(teamCode)
	if restDays == 0 {
		return 0.15 // 15% penalty for back-to-back
	}
	return 0.0
}

// generateAdvancedKeyFactors creates sophisticated key factors including situational insights
func (ps *PredictionService) generateAdvancedKeyFactors(homeFactors, awayFactors *models.PredictionFactors, prediction *models.PredictionResult) []string {
	factors := []string{}

	// Model-specific insights
	if len(prediction.ModelResults) > 0 {
		var bestModel models.ModelResult
		for _, model := range prediction.ModelResults {
			if model.Confidence > bestModel.Confidence {
				bestModel = model
			}
		}
		factors = append(factors, fmt.Sprintf("🎯 %s model shows highest confidence (%.1f%%)", bestModel.ModelName, bestModel.Confidence*100))
	}

	// Ensemble agreement
	if len(prediction.ModelResults) > 1 {
		agreement := ps.calculateModelAgreement(prediction.ModelResults)
		if agreement > 0.8 {
			factors = append(factors, "✅ All models agree on the outcome")
		} else if agreement < 0.6 {
			factors = append(factors, "⚠️ Models show mixed predictions")
		}
	}

	// Statistical advantages
	if prediction.Winner == homeFactors.TeamCode {
		if homeFactors.WinPercentage > awayFactors.WinPercentage+0.1 {
			factors = append(factors, fmt.Sprintf("📈 %s has superior record (%.1f%% vs %.1f%%)", homeFactors.TeamCode, homeFactors.WinPercentage*100, awayFactors.WinPercentage*100))
		}
		if homeFactors.GoalsFor > awayFactors.GoalsFor+0.5 {
			factors = append(factors, fmt.Sprintf("⚡ %s has stronger offense (%.1f vs %.1f goals/game)", homeFactors.TeamCode, homeFactors.GoalsFor, awayFactors.GoalsFor))
		}
		if homeFactors.HomeAdvantage > 0.1 {
			factors = append(factors, fmt.Sprintf("🏠 Home advantage favors %s (+%.1f%%)", homeFactors.TeamCode, homeFactors.HomeAdvantage*100))
		}
	} else {
		if awayFactors.WinPercentage > homeFactors.WinPercentage+0.1 {
			factors = append(factors, fmt.Sprintf("📈 %s has superior record (%.1f%% vs %.1f%%)", awayFactors.TeamCode, awayFactors.WinPercentage*100, homeFactors.WinPercentage*100))
		}
		if awayFactors.GoalsFor > homeFactors.GoalsFor+0.5 {
			factors = append(factors, fmt.Sprintf("⚡ %s has stronger offense (%.1f vs %.1f goals/game)", awayFactors.TeamCode, awayFactors.GoalsFor, homeFactors.GoalsFor))
		}
	}

	// NEW: Situational Factor Insights

	// Travel fatigue analysis
	if homeFactors.TravelFatigue.FatigueScore > 0.3 {
		factors = append(factors, fmt.Sprintf("✈️ %s experiencing significant travel fatigue (%.0f miles, %d time zones)",
			homeFactors.TeamCode, homeFactors.TravelFatigue.MilesTraveled, homeFactors.TravelFatigue.TimeZonesCrossed))
	}
	if awayFactors.TravelFatigue.FatigueScore > 0.3 {
		factors = append(factors, fmt.Sprintf("✈️ %s experiencing significant travel fatigue (%.0f miles, %d time zones)",
			awayFactors.TeamCode, awayFactors.TravelFatigue.MilesTraveled, awayFactors.TravelFatigue.TimeZonesCrossed))
	}

	// Altitude effects
	if math.Abs(homeFactors.AltitudeAdjust.AdjustmentFactor) > 0.05 {
		if homeFactors.AltitudeAdjust.AdjustmentFactor > 0 {
			factors = append(factors, fmt.Sprintf("🏔️ %s benefits from altitude advantage (%.0f ft difference)",
				homeFactors.TeamCode, homeFactors.AltitudeAdjust.AltitudeDiff))
		} else {
			factors = append(factors, fmt.Sprintf("🏔️ %s struggling with altitude change (%.0f ft difference)",
				homeFactors.TeamCode, math.Abs(homeFactors.AltitudeAdjust.AltitudeDiff)))
		}
	}
	if math.Abs(awayFactors.AltitudeAdjust.AdjustmentFactor) > 0.05 {
		if awayFactors.AltitudeAdjust.AdjustmentFactor > 0 {
			factors = append(factors, fmt.Sprintf("🏔️ %s benefits from altitude advantage (%.0f ft difference)",
				awayFactors.TeamCode, awayFactors.AltitudeAdjust.AltitudeDiff))
		} else {
			factors = append(factors, fmt.Sprintf("🏔️ %s struggling with altitude change (%.0f ft difference)",
				awayFactors.TeamCode, math.Abs(awayFactors.AltitudeAdjust.AltitudeDiff)))
		}
	}

	// Schedule analysis
	if homeFactors.ScheduleStrength.GamesInLast7Days > 3 {
		factors = append(factors, fmt.Sprintf("📅 %s playing heavy schedule (%d games in 7 days)",
			homeFactors.TeamCode, homeFactors.ScheduleStrength.GamesInLast7Days))
	}
	if awayFactors.ScheduleStrength.GamesInLast7Days > 3 {
		factors = append(factors, fmt.Sprintf("📅 %s playing heavy schedule (%d games in 7 days)",
			awayFactors.TeamCode, awayFactors.ScheduleStrength.GamesInLast7Days))
	}

	// Injury impact
	if homeFactors.InjuryImpact.InjuryScore > 0.15 {
		factors = append(factors, fmt.Sprintf("🏥 %s dealing with significant injuries (%s goalie, %d key players out)",
			homeFactors.TeamCode, homeFactors.InjuryImpact.GoalieStatus, homeFactors.InjuryImpact.KeyPlayersOut))
	}
	if awayFactors.InjuryImpact.InjuryScore > 0.15 {
		factors = append(factors, fmt.Sprintf("🏥 %s dealing with significant injuries (%s goalie, %d key players out)",
			awayFactors.TeamCode, awayFactors.InjuryImpact.GoalieStatus, awayFactors.InjuryImpact.KeyPlayersOut))
	}

	// Momentum analysis
	if homeFactors.MomentumFactors.MomentumScore > 0.7 {
		factors = append(factors, fmt.Sprintf("🔥 %s riding high momentum (%d streak, %d recent blowouts)",
			homeFactors.TeamCode, homeFactors.MomentumFactors.WinStreak, homeFactors.MomentumFactors.RecentBlowouts))
	} else if homeFactors.MomentumFactors.MomentumScore < 0.3 {
		factors = append(factors, fmt.Sprintf("❄️ %s struggling with low momentum (%d streak)",
			homeFactors.TeamCode, homeFactors.MomentumFactors.WinStreak))
	}
	if awayFactors.MomentumFactors.MomentumScore > 0.7 {
		factors = append(factors, fmt.Sprintf("🔥 %s riding high momentum (%d streak, %d recent blowouts)",
			awayFactors.TeamCode, awayFactors.MomentumFactors.WinStreak, awayFactors.MomentumFactors.RecentBlowouts))
	} else if awayFactors.MomentumFactors.MomentumScore < 0.3 {
		factors = append(factors, fmt.Sprintf("❄️ %s struggling with low momentum (%d streak)",
			awayFactors.TeamCode, awayFactors.MomentumFactors.WinStreak))
	}

	// Rest advantage
	if math.Abs(homeFactors.ScheduleStrength.RestAdvantage) > 0.5 {
		if homeFactors.ScheduleStrength.RestAdvantage > 0 {
			factors = append(factors, fmt.Sprintf("😴 %s has rest advantage over opponent", homeFactors.TeamCode))
		} else {
			factors = append(factors, fmt.Sprintf("😴 %s at rest disadvantage vs opponent", homeFactors.TeamCode))
		}
	}

	// Traditional factors
	if homeFactors.BackToBackPenalty > 0 {
		factors = append(factors, fmt.Sprintf("😴 %s playing back-to-back games", homeFactors.TeamCode))
	}
	if awayFactors.BackToBackPenalty > 0 {
		factors = append(factors, fmt.Sprintf("😴 %s playing back-to-back games", awayFactors.TeamCode))
	}

	// Upset potential
	if prediction.IsUpset {
		factors = append(factors, "🚨 Potential upset alert based on advanced metrics")
	}

	// Game type insights
	switch prediction.GameType {
	case "blowout":
		factors = append(factors, "💥 Models predict a decisive victory")
	case "toss-up":
		factors = append(factors, "🎲 Extremely close matchup predicted")
	case "close":
		factors = append(factors, "⚔️ Tight game expected")
	}

	// Confidence level
	if prediction.Confidence > 0.8 {
		factors = append(factors, "🔒 High confidence prediction")
	} else if prediction.Confidence < 0.6 {
		factors = append(factors, "❓ Lower confidence due to uncertainty")
	}

	if len(factors) == 0 {
		factors = append(factors, "📊 Teams are closely matched across all metrics")
	}

	return factors
}

// calculateModelAgreement determines how much models agree
func (ps *PredictionService) calculateModelAgreement(results []models.ModelResult) float64 {
	if len(results) <= 1 {
		return 1.0
	}

	// Check winner agreement
	winners := make(map[string]int)
	for _, result := range results {
		if result.WinProbability > 0.5 {
			winners["home"]++
		} else {
			winners["away"]++
		}
	}

	maxCount := 0
	for _, count := range winners {
		if count > maxCount {
			maxCount = count
		}
	}

	return float64(maxCount) / float64(len(results))
}

// Helper functions (simplified versions for demonstration)
func (ps *PredictionService) getRecentFormString(teamCode string) string {
	forms := []string{"W-W-L-W-W", "L-W-W-L-W", "W-L-L-W-L", "W-W-W-L-W", "L-L-W-W-W"}
	return forms[len(teamCode)%len(forms)]
}

func (ps *PredictionService) getCurrentStreak(teamCode string) string {
	streaks := []string{"3W", "2L", "1W", "4W", "1L", "2W", "1L"}
	return streaks[len(teamCode)%len(streaks)]
}
