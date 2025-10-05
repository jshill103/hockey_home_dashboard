package services

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// AdvancedAnalyticsService handles professional hockey analytics calculations
type AdvancedAnalyticsService struct {
	cache      map[string]*models.AdvancedTeamStats
	lastUpdate time.Time
}

// NewAdvancedAnalyticsService creates a new advanced analytics service
func NewAdvancedAnalyticsService() *AdvancedAnalyticsService {
	return &AdvancedAnalyticsService{
		cache: make(map[string]*models.AdvancedTeamStats),
	}
}

// GetAdvancedAnalytics returns processed advanced analytics for prediction models
func (s *AdvancedAnalyticsService) GetAdvancedAnalytics(teamCode string, isHome bool) (*models.AdvancedAnalytics, error) {
	log.Printf("📊 Computing advanced analytics for %s (home: %v)...", teamCode, isHome)

	// Get base team stats from NHL API
	standings, err := GetStandings()
	if err != nil {
		log.Printf("❌ Error fetching standings for advanced analytics: %v", err)
		return nil, fmt.Errorf("NHL standings data unavailable: %v", err)
	}

	// Find team in standings
	var teamStanding *models.TeamStanding
	for _, team := range standings.Standings {
		if team.TeamAbbrev.Default == teamCode {
			teamStanding = &team
			break
		}
	}

	if teamStanding == nil {
		log.Printf("❌ Team %s not found in standings", teamCode)
		return nil, fmt.Errorf("team %s not found in NHL standings data", teamCode)
	}

	// Calculate advanced analytics from available data
	analytics := s.calculateAdvancedAnalytics(teamStanding, isHome)

	log.Printf("✅ Advanced analytics computed for %s: Rating %.1f", teamCode, analytics.OverallRating)
	return analytics, nil
}

// calculateAdvancedAnalytics processes NHL data into advanced metrics
func (s *AdvancedAnalyticsService) calculateAdvancedAnalytics(team *models.TeamStanding, isHome bool) *models.AdvancedAnalytics {
	// Base calculations from available NHL data
	gamesPlayed := float64(team.GamesPlayed)
	if gamesPlayed == 0 {
		gamesPlayed = 1 // Prevent division by zero
	}

	goalsFor := float64(team.GoalFor)
	goalsAgainst := float64(team.GoalAgainst)

	// Expected Goals calculations (estimated from goal scoring patterns)
	xgForPerGame := s.estimateExpectedGoals(goalsFor/gamesPlayed, team.TeamAbbrev.Default)
	xgAgainstPerGame := s.estimateExpectedGoals(goalsAgainst/gamesPlayed, team.TeamAbbrev.Default)

	// Possession metrics (estimated from team performance)
	corsiForPct := s.estimateCorsiPct(team, isHome)
	fenwickForPct := corsiForPct * 0.95 // Fenwick typically slightly lower than Corsi

	// Shot quality estimates
	shotGenRate := s.estimateShotGeneration(team)
	shotSuppRate := s.estimateShotSuppression(team)

	// Special teams advanced metrics
	ppXG := s.estimatePowerPlayXG(team)
	pkXGA := s.estimatePenaltyKillXGA(team)

	// Goaltending performance
	svPct := s.estimateSavePercentage(team)
	svPctHD := svPct * 0.85 // High danger save % typically lower

	// Game state performance estimates
	leadingPerf := s.estimateLeadingPerformance(team)
	trailingPerf := s.estimateTrailingPerformance(team)
	closeGameRecord := s.estimateCloseGameRecord(team)

	// Zone play estimates
	ozTime := s.estimateOffensiveZoneTime(team, isHome)
	controlledEntries := s.estimateControlledEntries(team)
	controlledExits := s.estimateControlledExits(team)

	// Overall rating calculation
	overallRating := s.calculateOverallRating(team, xgForPerGame, xgAgainstPerGame, corsiForPct)

	// Determine strengths and weaknesses
	strengths, weaknesses := s.analyzeStrengthsWeaknesses(team, xgForPerGame, xgAgainstPerGame, corsiForPct)

	return &models.AdvancedAnalytics{
		// Expected Goals Performance
		XGForPerGame:      xgForPerGame,
		XGAgainstPerGame:  xgAgainstPerGame,
		XGDifferential:    xgForPerGame - xgAgainstPerGame,
		ShootingTalent:    (goalsFor/gamesPlayed - xgForPerGame) / xgForPerGame,
		GoaltendingTalent: (xgAgainstPerGame - goalsAgainst/gamesPlayed) / xgAgainstPerGame,

		// Possession Dominance
		CorsiForPct:       corsiForPct,
		FenwickForPct:     fenwickForPct,
		HighDangerPct:     s.estimateHighDangerPct(team),
		PossessionQuality: (corsiForPct + fenwickForPct) / 2,

		// Shot Quality & Generation
		ShotGenerationRate:  shotGenRate,
		ShotSuppressionRate: shotSuppRate,
		ShotQualityFor:      s.estimateShotQuality(team, true),
		ShotQualityAgainst:  s.estimateShotQuality(team, false),

		// Special Teams Excellence
		PowerPlayXG:      ppXG,
		PenaltyKillXGA:   pkXGA,
		SpecialTeamsEdge: s.calculateSpecialTeamsEdge(team),

		// Goaltending Performance
		GoalieSvPctOverall: svPct,
		GoalieSvPctHD:      svPctHD,
		SavesAboveExpected: s.estimateSavesAboveExpected(team),
		GoalieWorkload:     s.estimateGoalieWorkload(team),

		// Game State Performance
		LeadingPerformance:  leadingPerf,
		TrailingPerformance: trailingPerf,
		CloseGameRecord:     closeGameRecord,
		PeriodStrength:      s.estimatePeriodStrength(team),

		// Transition & Zone Play
		OffensiveZoneTime:    ozTime,
		ControlledEntries:    controlledEntries,
		ControlledExits:      controlledExits,
		TransitionEfficiency: s.estimateTransitionEfficiency(team),

		// Advanced Rating
		OverallRating: overallRating,
		StrengthAreas: strengths,
		WeaknessAreas: weaknesses,
	}
}

// estimateExpectedGoals calculates estimated xG from team performance
func (s *AdvancedAnalyticsService) estimateExpectedGoals(goalsPerGame float64, teamCode string) float64 {
	// Base estimate with team-specific adjustments
	baseXG := goalsPerGame * 0.95 // Slight regression toward mean

	// Team-specific adjustments based on playing style
	teamMultiplier := s.getTeamStyleMultiplier(teamCode)

	return math.Max(0.5, baseXG*teamMultiplier)
}

// estimateCorsiPct estimates shot attempt share from team performance
func (s *AdvancedAnalyticsService) estimateCorsiPct(team *models.TeamStanding, isHome bool) float64 {
	// Base calculation from goal differential and record
	gamesPlayed := float64(team.GamesPlayed)
	if gamesPlayed == 0 {
		return 0.50
	}

	// Goal differential impact
	goalDiff := float64(team.GoalFor - team.GoalAgainst)
	goalDiffImpact := goalDiff / (gamesPlayed * 6.0) // Normalize per game, scale by typical range

	// Win percentage impact
	winPct := float64(team.Points) / (gamesPlayed * 2.0)
	winPctImpact := (winPct - 0.5) * 0.3

	// Home adjustment
	homeAdj := 0.0
	if isHome {
		homeAdj = 0.02 // Small home ice advantage in possession
	}

	corsiPct := 0.50 + goalDiffImpact + winPctImpact + homeAdj

	// Clamp between reasonable bounds
	return math.Min(0.65, math.Max(0.35, corsiPct))
}

// Additional estimation methods...
func (s *AdvancedAnalyticsService) estimateShotGeneration(team *models.TeamStanding) float64 {
	gf := float64(team.GoalFor)
	gp := float64(team.GamesPlayed)
	if gp == 0 {
		return 30.0
	}

	// Estimate shots from goals (typical conversion ~10%)
	shotsPerGame := (gf / gp) / 0.10
	return math.Min(40.0, math.Max(20.0, shotsPerGame))
}

func (s *AdvancedAnalyticsService) estimateShotSuppression(team *models.TeamStanding) float64 {
	ga := float64(team.GoalAgainst)
	gp := float64(team.GamesPlayed)
	if gp == 0 {
		return 30.0
	}

	// Estimate shots against from goals against
	shotsAgainstPerGame := (ga / gp) / 0.10
	return math.Min(40.0, math.Max(20.0, shotsAgainstPerGame))
}

func (s *AdvancedAnalyticsService) estimatePowerPlayXG(team *models.TeamStanding) float64 {
	// Estimate from team performance (mock calculation)
	gp := float64(team.GamesPlayed)
	gf := float64(team.GoalFor)
	if gp == 0 {
		return 1.5
	}

	// Estimate PP performance from overall offense
	ppEst := (gf / gp) * 0.25 // Assume ~25% of goals are PP goals
	return math.Max(0.5, ppEst)
}

func (s *AdvancedAnalyticsService) estimatePenaltyKillXGA(team *models.TeamStanding) float64 {
	// Estimate from team defense (mock calculation)
	gp := float64(team.GamesPlayed)
	ga := float64(team.GoalAgainst)
	if gp == 0 {
		return 1.8
	}

	// Estimate PK performance from overall defense
	pkEst := (ga / gp) * 0.2 // Assume ~20% of goals against are PP goals
	return math.Max(0.8, pkEst)
}

func (s *AdvancedAnalyticsService) estimateSavePercentage(team *models.TeamStanding) float64 {
	gp := float64(team.GamesPlayed)
	if gp == 0 {
		return 0.910
	}

	ga := float64(team.GoalAgainst)
	// Estimate save percentage from goals against
	estimatedShots := ga / 0.09 // Assume ~9% shooting percentage against
	svPct := 1.0 - (ga / estimatedShots)

	return math.Min(0.950, math.Max(0.880, svPct))
}

func (s *AdvancedAnalyticsService) calculateOverallRating(team *models.TeamStanding, xgFor, xgAgainst, corsiPct float64) float64 {
	// Weighted combination of key metrics
	winPctWeight := 0.25
	xgDiffWeight := 0.30
	corsiWeight := 0.25
	specialTeamsWeight := 0.20

	gp := float64(team.GamesPlayed)
	if gp == 0 {
		return 50.0
	}

	winPctScore := (float64(team.Points) / (gp * 2.0)) * 100.0
	xgDiffScore := 50.0 + ((xgFor - xgAgainst) * 10.0)
	corsiScore := corsiPct * 100.0
	stScore := 50.0 // Mock special teams score

	rating := (winPctScore * winPctWeight) +
		(xgDiffScore * xgDiffWeight) +
		(corsiScore * corsiWeight) +
		(stScore * specialTeamsWeight)

	return math.Min(95.0, math.Max(5.0, rating))
}

// getTeamStyleMultiplier returns team-specific playing style adjustments
func (s *AdvancedAnalyticsService) getTeamStyleMultiplier(teamCode string) float64 {
	styleMultipliers := map[string]float64{
		"TOR": 1.05, // High-offense team
		"EDM": 1.06, // McDavid effect
		"VGK": 0.98, // Defensive system
		"UTA": 1.00, // New team, neutral
		"BOS": 0.99, // Structured system
		"TBL": 1.03, // Offensive system
		"COL": 1.04, // High-tempo offense
		"CAR": 1.02, // Aggressive forecheck
		"FLA": 1.01, // Balanced attack
		"DAL": 0.97, // Defensive structure
		"NYR": 1.01, // Skilled forwards
		"NJD": 1.02, // Young, fast team
	}

	if multiplier, exists := styleMultipliers[teamCode]; exists {
		return multiplier
	}
	return 1.00 // Default neutral
}

// Additional helper methods for comprehensive analytics...
func (s *AdvancedAnalyticsService) analyzeStrengthsWeaknesses(team *models.TeamStanding, xgFor, xgAgainst, corsiPct float64) ([]string, []string) {
	strengths := []string{}
	weaknesses := []string{}

	gp := float64(team.GamesPlayed)
	if gp == 0 {
		return []string{"Unknown"}, []string{"Unknown"}
	}

	// Analyze different areas based on available data
	gfPerGame := float64(team.GoalFor) / gp
	gaPerGame := float64(team.GoalAgainst) / gp

	if gfPerGame > 3.2 {
		strengths = append(strengths, "High-Scoring Offense")
	} else if gfPerGame < 2.5 {
		weaknesses = append(weaknesses, "Offensive Struggles")
	}

	if gaPerGame < 2.8 {
		strengths = append(strengths, "Stingy Defense")
	} else if gaPerGame > 3.5 {
		weaknesses = append(weaknesses, "Defensive Issues")
	}

	if xgFor > 3.2 {
		strengths = append(strengths, "High-Quality Chances")
	} else if xgFor < 2.5 {
		weaknesses = append(weaknesses, "Chance Generation")
	}

	if xgAgainst < 2.8 {
		strengths = append(strengths, "Defensive Structure")
	} else if xgAgainst > 3.5 {
		weaknesses = append(weaknesses, "Defensive Breakdowns")
	}

	if corsiPct > 0.55 {
		strengths = append(strengths, "Possession Dominance")
	} else if corsiPct < 0.45 {
		weaknesses = append(weaknesses, "Possession Problems")
	}

	// Limit to top 3 each
	if len(strengths) > 3 {
		strengths = strengths[:3]
	}
	if len(weaknesses) > 3 {
		weaknesses = weaknesses[:3]
	}

	// Ensure we have at least something
	if len(strengths) == 0 {
		strengths = []string{"Balanced Team"}
	}
	if len(weaknesses) == 0 {
		weaknesses = []string{"Well-Rounded"}
	}

	return strengths, weaknesses
}

// Additional estimation methods for comprehensive analytics
func (s *AdvancedAnalyticsService) estimateHighDangerPct(team *models.TeamStanding) float64 {
	return 0.50
}
func (s *AdvancedAnalyticsService) estimateShotQuality(team *models.TeamStanding, isFor bool) float64 {
	return 0.10
}
func (s *AdvancedAnalyticsService) calculateSpecialTeamsEdge(team *models.TeamStanding) float64 {
	// Mock calculation based on goal differential
	gp := float64(team.GamesPlayed)
	if gp == 0 {
		return 0.0
	}

	goalDiff := float64(team.GoalDifferential)
	return goalDiff / gp // Simple approximation
}
func (s *AdvancedAnalyticsService) estimateSavesAboveExpected(team *models.TeamStanding) float64 {
	return 0.0
}
func (s *AdvancedAnalyticsService) estimateGoalieWorkload(team *models.TeamStanding) float64 {
	return 50.0
}
func (s *AdvancedAnalyticsService) estimateLeadingPerformance(team *models.TeamStanding) float64 {
	return 55.0
}
func (s *AdvancedAnalyticsService) estimateTrailingPerformance(team *models.TeamStanding) float64 {
	return 45.0
}
func (s *AdvancedAnalyticsService) estimateCloseGameRecord(team *models.TeamStanding) float64 {
	return 0.50
}
func (s *AdvancedAnalyticsService) estimateOffensiveZoneTime(team *models.TeamStanding, isHome bool) float64 {
	return 0.50
}
func (s *AdvancedAnalyticsService) estimateControlledEntries(team *models.TeamStanding) float64 {
	return 0.60
}
func (s *AdvancedAnalyticsService) estimateControlledExits(team *models.TeamStanding) float64 {
	return 0.65
}
func (s *AdvancedAnalyticsService) estimateTransitionEfficiency(team *models.TeamStanding) float64 {
	return 0.15
}
func (s *AdvancedAnalyticsService) estimatePeriodStrength(team *models.TeamStanding) models.PeriodStrengthRating {
	return models.PeriodStrengthRating{
		FirstPeriod: 52.0, SecondPeriod: 50.0, ThirdPeriod: 53.0, OvertimePeriod: 55.0,
		StrongestPeriod: "3rd", WeakestPeriod: "2nd",
	}
}
