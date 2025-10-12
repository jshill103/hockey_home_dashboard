package services

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// PredictionModel interface for all prediction algorithms
type PredictionModel interface {
	Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error)
	GetName() string
	GetWeight() float64
}

// StatisticalModel - Enhanced statistical prediction model
type StatisticalModel struct {
	weight float64
}

func NewStatisticalModel() *StatisticalModel {
	return &StatisticalModel{weight: 0.35} // 35% weight in ensemble (reduced to make room for NN)
}

func (m *StatisticalModel) GetName() string {
	return "Enhanced Statistical"
}

func (m *StatisticalModel) GetWeight() float64 {
	return m.weight
}

func (m *StatisticalModel) Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error) {
	start := time.Now()

	// Enhanced statistical analysis with more factors
	homeScore := m.calculateAdvancedScore(homeFactors, true)
	awayScore := m.calculateAdvancedScore(awayFactors, false)

	// Safeguard: Ensure scores are valid numbers
	if math.IsNaN(homeScore) || math.IsInf(homeScore, 0) {
		homeScore = 50.0 // Default neutral score
	}
	if math.IsNaN(awayScore) || math.IsInf(awayScore, 0) {
		awayScore = 50.0 // Default neutral score
	}

	// Ensure scores are positive
	if homeScore <= 0 {
		homeScore = 0.1
	}
	if awayScore <= 0 {
		awayScore = 0.1
	}

	// Normalize scores to get probability
	totalScore := homeScore + awayScore

	// Safeguard: Prevent division by zero
	if totalScore <= 0 || math.IsNaN(totalScore) || math.IsInf(totalScore, 0) {
		totalScore = 100.0
		homeScore = 50.0
		awayScore = 50.0
	}

	homeWinProb := homeScore / totalScore

	// Safeguard: Ensure probability is valid
	if math.IsNaN(homeWinProb) || math.IsInf(homeWinProb, 0) {
		homeWinProb = 0.5 // Default to 50-50
	}

	// Clamp probability to valid range [0, 1]
	homeWinProb = math.Max(0.0, math.Min(1.0, homeWinProb))

	// Determine predicted score with more realistic distribution
	homeGoals, awayGoals := m.predictRealisticScore(homeFactors, awayFactors, homeWinProb)

	// Confidence based on score difference and data quality
	confidence := m.calculateConfidence(homeScore, awayScore, homeFactors, awayFactors)

	// Safeguard: Ensure confidence is valid
	if math.IsNaN(confidence) || math.IsInf(confidence, 0) || confidence < 0 {
		confidence = 0.5 // Default moderate confidence
	}

	// Clamp confidence to valid range [0, 1]
	confidence = math.Max(0.0, math.Min(1.0, confidence))

	// Determine final probability (for the home team)
	if homeWinProb < 0.5 {
		homeWinProb = 1.0 - homeWinProb
	}

	return &models.ModelResult{
		ModelName:      m.GetName(),
		WinProbability: homeWinProb,
		Confidence:     confidence,
		PredictedScore: fmt.Sprintf("%d-%d", homeGoals, awayGoals),
		Weight:         m.weight,
		ProcessingTime: time.Since(start).Milliseconds(),
	}, nil
}

func (m *StatisticalModel) calculateAdvancedScore(factors *models.PredictionFactors, isHome bool) float64 {
	score := factors.WinPercentage * 100 // Base win percentage

	// Goal differential impact (more sophisticated)
	goalDiff := factors.GoalsFor - factors.GoalsAgainst
	score += goalDiff * 2.0

	// Recent form with exponential decay
	score += factors.RecentForm * 15.0

	// Special teams impact
	score += (factors.PowerPlayPct - 0.2) * 10.0  // Average PP% is ~20%
	score += (factors.PenaltyKillPct - 0.8) * 8.0 // Average PK% is ~80%

	// Home advantage
	if isHome && factors.HomeAdvantage > 0 {
		score += factors.HomeAdvantage * 5.0
	}

	// Rest and fatigue factors
	if factors.BackToBackPenalty > 0 {
		score -= factors.BackToBackPenalty * 10.0 // Significant penalty for tired teams
	}

	// Head-to-head historical performance
	score += factors.HeadToHead * 3.0

	// NEW: Situational Factors Integration

	// Travel fatigue impact
	score -= factors.TravelFatigue.FatigueScore * 8.0 // Up to 8 point penalty

	// Altitude adjustment
	score += factors.AltitudeAdjust.AdjustmentFactor * 12.0 // Significant altitude effect

	// Schedule strength considerations
	score -= factors.ScheduleStrength.ScheduleDensity * 6.0 // Tired from heavy schedule
	score += factors.ScheduleStrength.RestAdvantage * 4.0   // Rest advantage

	// Injury impact
	score -= factors.InjuryImpact.InjuryScore * 10.0 // Major impact from injuries

	// Momentum factors
	score += (factors.MomentumFactors.MomentumScore - 0.5) * 8.0 // Momentum swing

	// NEW: Advanced Hockey Analytics Integration
	advStats := &factors.AdvancedStats

	// Expected Goals Performance (25% weight in total score)
	xgDiffImpact := advStats.XGDifferential * 8.0 // Strong xG differential indicator
	score += xgDiffImpact

	// Shooting and Goaltending Talent (15% weight)
	talentImpact := (advStats.ShootingTalent + advStats.GoaltendingTalent) * 10.0
	score += talentImpact

	// Possession Dominance (20% weight)
	possessionImpact := (advStats.CorsiForPct - 0.50) * 30.0 // Major possession indicator
	score += possessionImpact

	// High Danger Chance Share (15% weight)
	hdcImpact := (advStats.HighDangerPct - 0.50) * 25.0
	score += hdcImpact

	// Special Teams Advanced (10% weight)
	stAdvancedImpact := advStats.SpecialTeamsEdge * 0.5
	score += stAdvancedImpact

	// Goaltending Performance Above Expected (10% weight)
	goalieImpact := advStats.SavesAboveExpected * 2.0
	score += goalieImpact

	// Game State Performance (5% weight)
	gameStateImpact := (advStats.LeadingPerformance + advStats.CloseGameRecord*50.0) * 0.1
	score += gameStateImpact

	// Player Intelligence (10% weight) - NOW WITH REAL DATA!
	playerImpact := m.calculatePlayerImpact(factors)
	score += playerImpact

	// Weather Impact Analysis Integration (5% weight in total score)
	weatherImpact := m.calculateWeatherImpact(&factors.WeatherAnalysis)
	score += weatherImpact

	// Defensive NaN check for WinPercentage before logging
	baseScore := factors.WinPercentage * 100
	if math.IsNaN(baseScore) || math.IsInf(baseScore, 0) {
		log.Printf("⚠️ WARNING: WinPercentage is NaN for %s, using default 50.0", factors.TeamCode)
		factors.WinPercentage = 0.5
		baseScore = 50.0
	}

	fmt.Printf("📊 %s Advanced Score Breakdown:\n", factors.TeamCode)
	fmt.Printf("   Base: %.1f, Travel: %.1f, Altitude: %.1f, Schedule: %.1f, Injuries: %.1f, Momentum: %.1f\n",
		baseScore,
		-factors.TravelFatigue.FatigueScore*8.0,
		factors.AltitudeAdjust.AdjustmentFactor*12.0,
		factors.ScheduleStrength.RestAdvantage*4.0-factors.ScheduleStrength.ScheduleDensity*6.0,
		-factors.InjuryImpact.InjuryScore*10.0,
		(factors.MomentumFactors.MomentumScore-0.5)*8.0)
	fmt.Printf("   🏒 Advanced: xG %.1f, Talent %.1f, Possession %.1f, HDC %.1f, Goalie %.1f, Overall Rating %.1f\n",
		xgDiffImpact, talentImpact, possessionImpact, hdcImpact, goalieImpact, advStats.OverallRating)
	fmt.Printf("   ⭐ Player: Star %.1f, TopForm %.1f, Depth %.1f, Total %.1f\n",
		(factors.StarPowerRating-0.75)*20.0, (factors.TopScorerForm-5.0)*2.0,
		(factors.DepthForm-5.0)*1.5, playerImpact)
	fmt.Printf("   🌦️ Weather: Overall %.1f, Travel %.1f, Game %.1f (Outdoor: %v)\n",
		weatherImpact, factors.WeatherAnalysis.TravelImpact.OverallImpact*2.0,
		factors.WeatherAnalysis.GameImpact.OverallGameImpact*3.0, factors.WeatherAnalysis.IsOutdoorGame)

	return math.Max(score, 10.0) // Minimum viable score
}

// calculatePlayerImpact calculates player intelligence impact (star power, form, depth)
func (m *StatisticalModel) calculatePlayerImpact(factors *models.PredictionFactors) float64 {
	impact := 0.0

	// Star Power Impact (5% weight)
	// 0.75 is league average star power, above = positive, below = negative
	starPowerImpact := (factors.StarPowerRating - 0.75) * 20.0
	impact += starPowerImpact

	// Top Scorer Form Impact (3% weight)
	// 5.0 is neutral form (0-10 scale), hot = positive, cold = negative
	topScorerFormImpact := (factors.TopScorerForm - 5.0) * 2.0
	impact += topScorerFormImpact

	// Depth Form Impact (2% weight)
	// 5.0 is neutral form for depth players (4-10)
	depthFormImpact := (factors.DepthForm - 5.0) * 1.5
	impact += depthFormImpact

	return impact
}

// calculateWeatherImpact calculates the impact of weather conditions on team performance
func (m *StatisticalModel) calculateWeatherImpact(weatherAnalysis *models.WeatherAnalysis) float64 {
	if weatherAnalysis == nil {
		return 0.0 // No weather data = neutral impact
	}

	impact := 0.0

	// Travel weather impact (affects preparation and rest)
	travelImpact := weatherAnalysis.TravelImpact.OverallImpact * 2.0 // Up to ±10 points
	impact += travelImpact

	// Game weather impact (affects gameplay, especially for outdoor games)
	gameImpact := weatherAnalysis.GameImpact.OverallGameImpact * 3.0 // Up to ±15 points for outdoor games
	impact += gameImpact

	// Confidence weighting - reduce impact if weather data is unreliable
	confidenceWeight := weatherAnalysis.Confidence
	if confidenceWeight < 0.5 {
		confidenceWeight = 0.5 // Minimum confidence weighting
	}
	impact *= confidenceWeight

	// Special considerations for outdoor games
	if weatherAnalysis.IsOutdoorGame {
		// Outdoor games have more variable conditions
		impact *= 1.5 // 50% amplification for outdoor games

		// Additional factors for extreme conditions
		conditions := &weatherAnalysis.WeatherConditions

		// Extreme temperature penalty
		if conditions.Temperature < 10 || conditions.Temperature > 80 {
			impact -= 2.0 // Additional penalty for extreme temperatures
		}

		// High wind penalty
		if conditions.WindSpeed > 25 {
			impact -= 1.5 // Additional penalty for high winds
		}

		// Precipitation penalty
		if conditions.Precipitation > 0.1 {
			impact -= 1.0 // Additional penalty for precipitation
		}
	}

	// Cap the weather impact
	if impact > 8.0 {
		impact = 8.0
	} else if impact < -8.0 {
		impact = -8.0
	}

	return impact
}

func (m *StatisticalModel) predictRealisticScore(homeFactors, awayFactors *models.PredictionFactors, homeWinProb float64) (int, int) {
	// More realistic goal prediction based on team averages
	expectedHomeGoals := homeFactors.GoalsFor + (homeWinProb-0.5)*1.5
	expectedAwayGoals := awayFactors.GoalsFor + (1.0-homeWinProb-0.5)*1.5

	// Add some randomness within reason
	homeGoals := int(math.Round(expectedHomeGoals + (rand.Float64()-0.5)*0.8))
	awayGoals := int(math.Round(expectedAwayGoals + (rand.Float64()-0.5)*0.8))

	// Ensure minimum goals and reasonable range
	homeGoals = int(math.Max(1, math.Min(float64(homeGoals), 7)))
	awayGoals = int(math.Max(1, math.Min(float64(awayGoals), 7)))

	return homeGoals, awayGoals
}

func (m *StatisticalModel) calculateConfidence(homeScore, awayScore float64, homeFactors, awayFactors *models.PredictionFactors) float64 {
	// Confidence based on score margin
	scoreDiff := math.Abs(homeScore - awayScore)
	marginConfidence := math.Min(scoreDiff/50.0, 1.0)

	// Data quality confidence (rest days, recent games, etc.)
	dataQuality := 0.8 // Base confidence
	if homeFactors.RestDays < 1 || awayFactors.RestDays < 1 {
		dataQuality -= 0.2 // Less confident with tired teams
	}

	return (marginConfidence + dataQuality) / 2.0
}

// BayesianModel - Bayesian inference prediction model
type BayesianModel struct {
	weight float64
}

func NewBayesianModel() *BayesianModel {
	return &BayesianModel{weight: 0.15} // 15% weight in ensemble (reduced to make room for NN)
}

func (m *BayesianModel) GetName() string {
	return "Bayesian Inference"
}

func (m *BayesianModel) GetWeight() float64 {
	return m.weight
}

func (m *BayesianModel) Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error) {
	start := time.Now()

	// Bayesian approach: Update prior beliefs with observed data
	priorHomeWin := 0.55 // Home teams win ~55% of games historically

	// Calculate likelihood based on current factors
	homeLikelihood := m.calculateLikelihood(homeFactors, true)
	awayLikelihood := m.calculateLikelihood(awayFactors, false)

	// Safeguard: Ensure likelihoods are valid
	if math.IsNaN(homeLikelihood) || math.IsInf(homeLikelihood, 0) || homeLikelihood <= 0 {
		homeLikelihood = 0.5
	}
	if math.IsNaN(awayLikelihood) || math.IsInf(awayLikelihood, 0) || awayLikelihood <= 0 {
		awayLikelihood = 0.5
	}

	// Clamp likelihoods to reasonable range
	homeLikelihood = math.Max(0.01, math.Min(2.0, homeLikelihood))
	awayLikelihood = math.Max(0.01, math.Min(2.0, awayLikelihood))

	// Bayes' theorem: P(Home Win | Evidence) = P(Evidence | Home Win) * P(Home Win) / P(Evidence)
	evidence := homeLikelihood*priorHomeWin + awayLikelihood*(1-priorHomeWin)

	// Safeguard: Prevent division by zero in Bayes' theorem
	if evidence <= 0 || math.IsNaN(evidence) || math.IsInf(evidence, 0) {
		evidence = 1.0
	}

	posteriorHomeWin := (homeLikelihood * priorHomeWin) / evidence

	// Safeguard: Ensure posterior is valid
	if math.IsNaN(posteriorHomeWin) || math.IsInf(posteriorHomeWin, 0) {
		posteriorHomeWin = 0.5
	}

	// Clamp posterior to valid range [0, 1]
	posteriorHomeWin = math.Max(0.0, math.Min(1.0, posteriorHomeWin))

	// Predict score using Bayesian expectations
	homeGoals, awayGoals := m.bayesianScorePrediction(homeFactors, awayFactors, posteriorHomeWin)

	// Confidence from Bayesian credible interval
	confidence := m.calculateBayesianConfidence(posteriorHomeWin, evidence)

	// Safeguard: Ensure confidence is valid
	if math.IsNaN(confidence) || math.IsInf(confidence, 0) || confidence < 0 {
		confidence = 0.5
	}

	// Clamp confidence to valid range [0, 1]
	confidence = math.Max(0.0, math.Min(1.0, confidence))

	// Determine final probability (for the home team)
	if posteriorHomeWin < 0.5 {
		posteriorHomeWin = 1.0 - posteriorHomeWin
	}

	return &models.ModelResult{
		ModelName:      m.GetName(),
		WinProbability: posteriorHomeWin,
		Confidence:     confidence,
		PredictedScore: fmt.Sprintf("%d-%d", homeGoals, awayGoals),
		Weight:         m.weight,
		ProcessingTime: time.Since(start).Milliseconds(),
	}, nil
}

func (m *BayesianModel) calculateLikelihood(factors *models.PredictionFactors, isHome bool) float64 {
	// Calculate how likely the current evidence is given team performance
	likelihood := factors.WinPercentage

	// Safeguard: Ensure win percentage is valid
	if math.IsNaN(likelihood) || math.IsInf(likelihood, 0) {
		likelihood = 0.5
	}
	likelihood = math.Max(0.0, math.Min(1.0, likelihood))

	// Weight recent performance more heavily (Bayesian updating)
	recentForm := factors.RecentForm
	if math.IsNaN(recentForm) || math.IsInf(recentForm, 0) {
		recentForm = 0.5
	}
	recentForm = math.Max(0.0, math.Min(1.0, recentForm))
	likelihood = 0.7*likelihood + 0.3*recentForm

	// Incorporate goal differential with uncertainty
	goalsFor := factors.GoalsFor
	goalsAgainst := factors.GoalsAgainst
	if math.IsNaN(goalsFor) || math.IsInf(goalsFor, 0) {
		goalsFor = 2.8 // League average
	}
	if math.IsNaN(goalsAgainst) || math.IsInf(goalsAgainst, 0) {
		goalsAgainst = 2.8 // League average
	}

	goalFactor := 1.0 + (goalsFor-goalsAgainst)/10.0
	// Clamp goal factor to reasonable range
	goalFactor = math.Max(0.5, math.Min(1.5, goalFactor))
	likelihood *= goalFactor

	// Safeguard after goal factor multiplication
	if math.IsNaN(likelihood) || math.IsInf(likelihood, 0) || likelihood <= 0 {
		likelihood = 0.5
	}

	// Home advantage in likelihood
	if isHome {
		homeAdv := factors.HomeAdvantage
		if math.IsNaN(homeAdv) || math.IsInf(homeAdv, 0) {
			homeAdv = 0.05 // Default 5% home advantage
		}
		homeAdv = math.Max(0.0, math.Min(0.2, homeAdv))
		likelihood *= (1.0 + homeAdv)
	}

	// Safeguard after home advantage
	if math.IsNaN(likelihood) || math.IsInf(likelihood, 0) || likelihood <= 0 {
		likelihood = 0.5
	}

	// NEW: Bayesian integration of situational factors

	// Travel fatigue reduces likelihood of good performance
	likelihood *= (1.0 - factors.TravelFatigue.FatigueScore*0.15)

	// Altitude effects on likelihood
	if factors.AltitudeAdjust.AdjustmentFactor < 0 {
		likelihood *= (1.0 + factors.AltitudeAdjust.AdjustmentFactor*0.2) // Penalty for altitude struggles
	} else {
		likelihood *= (1.0 + factors.AltitudeAdjust.AdjustmentFactor*0.1) // Smaller bonus for altitude advantage
	}

	// Injury impact on likelihood
	likelihood *= (1.0 - factors.InjuryImpact.InjuryScore*0.18)

	// Momentum as Bayesian prior adjustment
	momentumAdjust := (factors.MomentumFactors.MomentumScore - 0.5) * 0.12
	likelihood *= (1.0 + momentumAdjust)

	// Schedule strength affects uncertainty
	if factors.ScheduleStrength.ScheduleDensity > 0.25 {
		likelihood *= 0.95 // Slightly reduce confidence for tired teams
	}

	// NEW: Advanced Hockey Analytics Bayesian Integration
	advStats := &factors.AdvancedStats

	// Expected Goals influence on likelihood (strong predictor)
	xgFactor := 1.0 + (advStats.XGDifferential * 0.15) // xG differential impact
	likelihood *= xgFactor

	// Shooting/Goaltending talent influence
	talentFactor := 1.0 + ((advStats.ShootingTalent + advStats.GoaltendingTalent) * 0.10)
	likelihood *= talentFactor

	// Possession metrics (Corsi) - strong underlying performance indicator
	possessionFactor := 1.0 + ((advStats.CorsiForPct - 0.50) * 0.25) // Corsi impact on likelihood
	likelihood *= possessionFactor

	// High danger chances share
	hdcFactor := 1.0 + ((advStats.HighDangerPct - 0.50) * 0.20)
	likelihood *= hdcFactor

	// Goaltending performance above expected
	goalieFactor := 1.0 + (advStats.SavesAboveExpected * 0.08)
	likelihood *= goalieFactor

	// Close game performance (clutch factor)
	clutchFactor := 1.0 + ((advStats.CloseGameRecord - 0.50) * 0.15)
	likelihood *= clutchFactor

	// NEW: Weather Impact on Bayesian Likelihood
	weatherFactor := m.calculateWeatherLikelihoodFactor(&factors.WeatherAnalysis)
	// Safeguard: Ensure weather factor is valid
	if math.IsNaN(weatherFactor) || math.IsInf(weatherFactor, 0) || weatherFactor <= 0 {
		weatherFactor = 1.0
	}
	weatherFactor = math.Max(0.5, math.Min(weatherFactor, 1.5))
	likelihood *= weatherFactor

	// Final safeguard: Ensure likelihood is a valid number
	if math.IsNaN(likelihood) || math.IsInf(likelihood, 0) || likelihood <= 0 {
		likelihood = 0.5
		fmt.Printf("⚠️ %s Bayesian Likelihood was invalid, defaulting to 0.5\n", factors.TeamCode)
	}

	fmt.Printf("🧠 %s Bayesian Likelihood: %.3f (with situational adjustments)\n", factors.TeamCode, likelihood)

	return math.Max(0.1, math.Min(likelihood, 2.0))
}

// calculateWeatherLikelihoodFactor calculates weather impact on Bayesian likelihood
func (m *BayesianModel) calculateWeatherLikelihoodFactor(weatherAnalysis *models.WeatherAnalysis) float64 {
	if weatherAnalysis == nil {
		return 1.0 // No weather data = neutral impact
	}

	factor := 1.0

	// Travel weather impact affects team preparation likelihood
	travelImpact := weatherAnalysis.TravelImpact.OverallImpact
	if travelImpact != 0 {
		factor *= (1.0 + travelImpact*0.05) // Up to ±25% likelihood adjustment
	}

	// Game weather impact affects performance likelihood
	gameImpact := weatherAnalysis.GameImpact.OverallGameImpact
	if gameImpact != 0 {
		factor *= (1.0 + gameImpact*0.08) // Up to ±40% likelihood adjustment for outdoor games
	}

	// Confidence weighting
	confidenceWeight := weatherAnalysis.Confidence
	if confidenceWeight < 0.7 {
		// Reduce weather impact if confidence is low
		adjustmentMagnitude := math.Abs(factor - 1.0)
		factor = 1.0 + (adjustmentMagnitude * confidenceWeight)
	}

	// Special outdoor game considerations
	if weatherAnalysis.IsOutdoorGame {
		conditions := &weatherAnalysis.WeatherConditions

		// Extreme conditions reduce likelihood of normal performance
		if conditions.Temperature < 15 || conditions.Temperature > 75 {
			factor *= 0.95 // 5% reduction in likelihood for extreme temps
		}

		if conditions.WindSpeed > 20 {
			factor *= 0.93 // 7% reduction for high winds
		}

		if conditions.Precipitation > 0.05 {
			factor *= 0.92 // 8% reduction for precipitation
		}
	}

	// Ensure factor stays within reasonable bounds
	return math.Max(0.7, math.Min(factor, 1.3))
}

func (m *BayesianModel) bayesianScorePrediction(homeFactors, awayFactors *models.PredictionFactors, homeWinProb float64) (int, int) {
	// Use Poisson distribution assumptions for goal scoring
	homeExpected := homeFactors.GoalsFor * (0.8 + homeWinProb*0.4)
	awayExpected := awayFactors.GoalsFor * (0.8 + (1-homeWinProb)*0.4)

	// Bayesian point estimates
	homeGoals := int(math.Round(homeExpected))
	awayGoals := int(math.Round(awayExpected))

	// Ensure reasonable bounds
	homeGoals = int(math.Max(1, math.Min(float64(homeGoals), 6)))
	awayGoals = int(math.Max(1, math.Min(float64(awayGoals), 6)))

	return homeGoals, awayGoals
}

func (m *BayesianModel) calculateBayesianConfidence(posterior, evidence float64) float64 {
	// Confidence based on how much evidence supports the conclusion
	certainty := math.Abs(posterior-0.5) * 2 // Distance from 50-50
	evidenceStrength := evidence

	return (certainty + evidenceStrength) / 2.0
}

// MonteCarloModel - Monte Carlo simulation prediction model
type MonteCarloModel struct {
	weight      float64
	simulations int
}

func NewMonteCarloModel() *MonteCarloModel {
	return &MonteCarloModel{
		weight:      0.10, // 10% weight in ensemble (reduced to make room for NN)
		simulations: 2000, // Number of simulations to run (optimized for speed vs accuracy)
	}
}

func (m *MonteCarloModel) GetName() string {
	return "Monte Carlo Simulation"
}

func (m *MonteCarloModel) GetWeight() float64 {
	return m.weight
}

func (m *MonteCarloModel) Predict(homeFactors, awayFactors *models.PredictionFactors) (*models.ModelResult, error) {
	start := time.Now()

	homeWins := 0
	var homeGoalsSum, awayGoalsSum int

	// Run Monte Carlo simulations
	for i := 0; i < m.simulations; i++ {
		homeGoals, awayGoals := m.simulateGame(homeFactors, awayFactors)
		homeGoalsSum += homeGoals
		awayGoalsSum += awayGoals

		if homeGoals > awayGoals {
			homeWins++
		}
	}

	// Calculate results from simulations
	homeWinProb := float64(homeWins) / float64(m.simulations)
	avgHomeGoals := float64(homeGoalsSum) / float64(m.simulations)
	avgAwayGoals := float64(awayGoalsSum) / float64(m.simulations)

	// Confidence based on consistency of results
	confidence := m.calculateMonteCarloConfidence(homeWinProb, homeWins)

	// Determine final probability (for the home team)
	if homeWinProb < 0.5 {
		homeWinProb = 1.0 - homeWinProb
	}

	return &models.ModelResult{
		ModelName:      m.GetName(),
		WinProbability: homeWinProb,
		Confidence:     confidence,
		PredictedScore: fmt.Sprintf("%.0f-%.0f", avgHomeGoals, avgAwayGoals),
		Weight:         m.weight,
		ProcessingTime: time.Since(start).Milliseconds(),
	}, nil
}

func (m *MonteCarloModel) simulateGame(homeFactors, awayFactors *models.PredictionFactors) (int, int) {
	// Simulate a single game using random variables

	// Base goal expectation with random variation
	homeExpected := homeFactors.GoalsFor * (0.7 + rand.Float64()*0.6)
	awayExpected := awayFactors.GoalsFor * (0.7 + rand.Float64()*0.6)

	// Add performance factors with randomness
	homePerf := homeFactors.RecentForm + (rand.Float64()-0.5)*0.4
	awayPerf := awayFactors.RecentForm + (rand.Float64()-0.5)*0.4

	homeExpected *= (1.0 + homePerf)
	awayExpected *= (1.0 + awayPerf)

	// Home advantage with variation
	homeExpected *= (1.0 + homeFactors.HomeAdvantage*(0.8+rand.Float64()*0.4))

	// Fatigue factors
	if homeFactors.BackToBackPenalty > 0 {
		homeExpected *= (1.0 - homeFactors.BackToBackPenalty*rand.Float64())
	}
	if awayFactors.BackToBackPenalty > 0 {
		awayExpected *= (1.0 - awayFactors.BackToBackPenalty*rand.Float64())
	}

	// NEW: Monte Carlo integration of situational factors

	// Travel fatigue with random variation
	homeTravelPenalty := homeFactors.TravelFatigue.FatigueScore * (0.5 + rand.Float64()*0.5)
	awayTravelPenalty := awayFactors.TravelFatigue.FatigueScore * (0.5 + rand.Float64()*0.5)
	homeExpected *= (1.0 - homeTravelPenalty*0.15)
	awayExpected *= (1.0 - awayTravelPenalty*0.15)

	// Altitude effects with random variation
	homeAltEffect := homeFactors.AltitudeAdjust.AdjustmentFactor * (0.8 + rand.Float64()*0.4)
	awayAltEffect := awayFactors.AltitudeAdjust.AdjustmentFactor * (0.8 + rand.Float64()*0.4)
	homeExpected *= (1.0 + homeAltEffect*0.1)
	awayExpected *= (1.0 + awayAltEffect*0.1)

	// Injury impact with random severity
	homeInjuryPenalty := homeFactors.InjuryImpact.InjuryScore * (rand.Float64()*0.8 + 0.2)
	awayInjuryPenalty := awayFactors.InjuryImpact.InjuryScore * (rand.Float64()*0.8 + 0.2)
	homeExpected *= (1.0 - homeInjuryPenalty*0.12)
	awayExpected *= (1.0 - awayInjuryPenalty*0.12)

	// Momentum with psychological randomness
	homeMomentum := (homeFactors.MomentumFactors.MomentumScore - 0.5) * (0.5 + rand.Float64()*0.5)
	awayMomentum := (awayFactors.MomentumFactors.MomentumScore - 0.5) * (0.5 + rand.Float64()*0.5)
	homeExpected *= (1.0 + homeMomentum*0.08)
	awayExpected *= (1.0 + awayMomentum*0.08)

	// Schedule density fatigue
	if homeFactors.ScheduleStrength.ScheduleDensity > 0.25 {
		homeExpected *= (0.95 - rand.Float64()*0.1) // Random fatigue effect
	}
	if awayFactors.ScheduleStrength.ScheduleDensity > 0.25 {
		awayExpected *= (0.95 - rand.Float64()*0.1)
	}

	// NEW: Advanced Hockey Analytics Monte Carlo Integration
	homeAdvStats := &homeFactors.AdvancedStats
	awayAdvStats := &awayFactors.AdvancedStats

	// Expected Goals with random variation (primary predictor)
	homeXGEffect := homeAdvStats.XGForPerGame * (0.8 + rand.Float64()*0.4)
	awayXGEffect := awayAdvStats.XGForPerGame * (0.8 + rand.Float64()*0.4)
	homeExpected = (homeExpected + homeXGEffect) / 2.0 // Blend traditional and xG
	awayExpected = (awayExpected + awayXGEffect) / 2.0

	// Shooting/Goaltending talent with random performance
	homeShootingBonus := homeAdvStats.ShootingTalent * (0.5 + rand.Float64()*1.0)
	awayShootingBonus := awayAdvStats.ShootingTalent * (0.5 + rand.Float64()*1.0)
	homeExpected *= (1.0 + homeShootingBonus*0.1)
	awayExpected *= (1.0 + awayShootingBonus*0.1)

	// Goaltending performance with random hot/cold streaks
	homeGoalieEffect := homeAdvStats.SavesAboveExpected * (rand.Float64()*2.0 - 1.0) // Can be negative
	awayGoalieEffect := awayAdvStats.SavesAboveExpected * (rand.Float64()*2.0 - 1.0)
	homeExpected *= (1.0 - awayGoalieEffect*0.08) // Opponent's goalie affects your scoring
	awayExpected *= (1.0 - homeGoalieEffect*0.08)

	// Possession dominance with game flow randomness
	homePossessionBonus := (homeAdvStats.CorsiForPct - 0.50) * (0.5 + rand.Float64()*1.0)
	awayPossessionBonus := (awayAdvStats.CorsiForPct - 0.50) * (0.5 + rand.Float64()*1.0)
	homeExpected *= (1.0 + homePossessionBonus*0.15)
	awayExpected *= (1.0 + awayPossessionBonus*0.15)

	// High danger chances with random hot/cold shooting
	homeHDCBonus := (homeAdvStats.HighDangerPct - 0.50) * (0.3 + rand.Float64()*0.7)
	awayHDCBonus := (awayAdvStats.HighDangerPct - 0.50) * (0.3 + rand.Float64()*0.7)
	homeExpected *= (1.0 + homeHDCBonus*0.12)
	awayExpected *= (1.0 + awayHDCBonus*0.12)

	// Close game performance (clutch factor) with random execution
	if rand.Float64() < 0.3 { // 30% chance this becomes a "clutch moment"
		homeClutchFactor := (homeAdvStats.CloseGameRecord - 0.50) * (0.5 + rand.Float64()*1.5)
		awayClutchFactor := (awayAdvStats.CloseGameRecord - 0.50) * (0.5 + rand.Float64()*1.5)
		homeExpected *= (1.0 + homeClutchFactor*0.2)
		awayExpected *= (1.0 + awayClutchFactor*0.2)
	}

	// NEW: Weather Impact Monte Carlo Integration
	homeWeatherEffect := m.simulateWeatherImpact(&homeFactors.WeatherAnalysis)
	awayWeatherEffect := m.simulateWeatherImpact(&awayFactors.WeatherAnalysis)
	homeExpected *= homeWeatherEffect
	awayExpected *= awayWeatherEffect

	// Convert to discrete goals (using Poisson-like distribution)
	homeGoals := int(math.Max(0, homeExpected+rand.NormFloat64()*0.8))
	awayGoals := int(math.Max(0, awayExpected+rand.NormFloat64()*0.8))

	// Cap at reasonable maximums
	homeGoals = int(math.Min(float64(homeGoals), 8))
	awayGoals = int(math.Min(float64(awayGoals), 8))

	return homeGoals, awayGoals
}

// simulateWeatherImpact simulates weather effects with random variations
func (m *MonteCarloModel) simulateWeatherImpact(weatherAnalysis *models.WeatherAnalysis) float64 {
	if weatherAnalysis == nil {
		return 1.0 // No weather data = neutral impact
	}

	effect := 1.0

	// Travel weather impact with random variation
	travelImpact := weatherAnalysis.TravelImpact.OverallImpact
	if travelImpact != 0 {
		// Random variation in how much travel affects performance
		randomFactor := 0.5 + rand.Float64()             // 0.5 to 1.5 multiplier
		effect *= (1.0 + travelImpact*0.03*randomFactor) // Up to ±15% with variation
	}

	// Game weather impact with random variation
	gameImpact := weatherAnalysis.GameImpact.OverallGameImpact
	if gameImpact != 0 {
		// Random variation in how teams adapt to conditions
		adaptationFactor := 0.3 + rand.Float64()*0.7       // 0.3 to 1.0 adaptation
		effect *= (1.0 + gameImpact*0.05*adaptationFactor) // Up to ±25% with adaptation
	}

	// Special outdoor game simulation
	if weatherAnalysis.IsOutdoorGame {
		conditions := &weatherAnalysis.WeatherConditions

		// Temperature effects with random player tolerance
		if conditions.Temperature < 20 || conditions.Temperature > 70 {
			tempTolerance := rand.Float64() // Random tolerance (0-1)
			if tempTolerance < 0.3 {        // 30% chance of poor adaptation
				effect *= (0.85 + rand.Float64()*0.1) // 85-95% performance
			} else if tempTolerance > 0.8 { // 20% chance of good adaptation
				effect *= (1.0 + rand.Float64()*0.05) // Up to 5% bonus
			}
		}

		// Wind effects with random gusts
		if conditions.WindSpeed > 15 {
			windVariation := rand.Float64() * 0.5 // Random wind impact
			effect *= (0.92 + windVariation*0.08) // 92-100% performance
		}

		// Precipitation effects with random intensity
		if conditions.Precipitation > 0.05 {
			precipIntensity := rand.Float64()       // Random precipitation intensity
			effect *= (0.88 + precipIntensity*0.12) // 88-100% performance
		}

		// Random equipment/ice quality variations
		if rand.Float64() < 0.2 { // 20% chance of equipment/ice issues
			effect *= (0.93 + rand.Float64()*0.07) // 93-100% performance
		}
	}

	// Confidence affects variability - low confidence = more random outcomes
	confidenceWeight := weatherAnalysis.Confidence
	if confidenceWeight < 0.7 {
		// Add more randomness for low confidence weather data
		randomVariation := (rand.Float64() - 0.5) * 0.1 * (1.0 - confidenceWeight)
		effect *= (1.0 + randomVariation)
	}

	// Ensure effect stays within reasonable simulation bounds
	return math.Max(0.75, math.Min(effect, 1.25))
}

func (m *MonteCarloModel) calculateMonteCarloConfidence(winProb float64, wins int) float64 {
	// Statistical confidence based on sample size and variance
	sampleSize := float64(m.simulations)
	variance := winProb * (1 - winProb) / sampleSize
	standardError := math.Sqrt(variance)

	// Higher confidence with larger sample and clearer results
	marginConfidence := math.Max(0, 1.0-standardError*4) // 4 standard deviations

	// Additional confidence from clear win/loss margin
	clarityBonus := math.Abs(winProb-0.5) * 0.5

	return math.Min(1.0, marginConfidence+clarityBonus)
}
