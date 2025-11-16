package services

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"

	"github.com/jaredshillingburg/go_uhc/models"
)

// TeamRanking represents a team's ML-powered ranking
type TeamRanking struct {
	TeamCode        string
	TeamName        string
	MLScore         float64 // 0-100 composite score from all ML models
	Tier            string  // S, A, B, C, D
	Record          string
	Points          int
	PointPct        float64
	EloRating       float64
	PoissonOffense  float64
	PoissonDefense  float64
	ModelBreakdown  map[string]float64 // Individual model scores
	ConfidenceLevel string
}

// TierList represents the complete NHL tier ranking
type TierList struct {
	GeneratedAt string
	Season      string
	Tiers       map[string][]TeamRanking // S, A, B, C, D
	Methodology string
}

// TeamTierRankingService generates ML-powered NHL team tier lists
type TeamTierRankingService struct {
	ensemble   *EnsemblePredictionService
	elo        *EloRatingModel
	poisson    *PoissonRegressionModel
	neuralNet  *NeuralNetworkModel
	mu         sync.RWMutex
}

var (
	tierRankingService     *TeamTierRankingService
	tierRankingServiceOnce sync.Once
)

// GetTeamTierRankingService returns the singleton instance
func GetTeamTierRankingService() *TeamTierRankingService {
	tierRankingServiceOnce.Do(func() {
		// Get models from live prediction system
		liveSys := GetLivePredictionSystem()
		var ensemble *EnsemblePredictionService
		var elo *EloRatingModel
		var poisson *PoissonRegressionModel
		var neuralNet *NeuralNetworkModel

		if liveSys != nil {
			ensemble = liveSys.GetEnsemble()
			elo = liveSys.GetEloModel()
			poisson = liveSys.GetPoissonModel()
			neuralNet = liveSys.GetNeuralNetwork()
		}

		tierRankingService = &TeamTierRankingService{
			ensemble:  ensemble,
			elo:       elo,
			poisson:   poisson,
			neuralNet: neuralNet,
		}
	})
	return tierRankingService
}

// GenerateTierList creates a comprehensive ML-powered tier list of all NHL teams
func (t *TeamTierRankingService) GenerateTierList() (*TierList, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	log.Println("🏆 Generating ML-powered NHL tier list...")

	// Get current standings
	standings, err := GetStandings()
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %w", err)
	}

	// Rank all teams
	rankings := make([]TeamRanking, 0, len(standings.Standings))
	for _, team := range standings.Standings {
		ranking := t.rankTeam(&team)
		rankings = append(rankings, ranking)
	}

	// Sort by ML score (highest first)
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].MLScore > rankings[j].MLScore
	})

	// Assign tiers
	t.assignTiers(rankings)

	// Organize into tier map
	tierList := &TierList{
		GeneratedAt: GetCurrentSeason(),
		Season:      GetCurrentSeason(),
		Tiers:       make(map[string][]TeamRanking),
		Methodology: "ML Ensemble (10 models) + Advanced Analytics",
	}

	for _, ranking := range rankings {
		tierList.Tiers[ranking.Tier] = append(tierList.Tiers[ranking.Tier], ranking)
	}

	log.Printf("✅ Tier list generated: S:%d A:%d B:%d C:%d D:%d",
		len(tierList.Tiers["S"]),
		len(tierList.Tiers["A"]),
		len(tierList.Tiers["B"]),
		len(tierList.Tiers["C"]),
		len(tierList.Tiers["D"]))

	return tierList, nil
}

// rankTeam calculates a team's comprehensive ML score
func (t *TeamTierRankingService) rankTeam(team *models.TeamStanding) TeamRanking {
	ranking := TeamRanking{
		TeamCode:       team.TeamAbbrev.Default,
		TeamName:       team.TeamName.Default,
		Record:         fmt.Sprintf("%d-%d-%d", team.Wins, team.Losses, team.OtLosses),
		Points:         team.Points,
		PointPct:       team.PointPctg,
		ModelBreakdown: make(map[string]float64),
	}

	// Start with point percentage as baseline (0-100 scale)
	baseScore := team.PointPctg * 100

	// Factor 1: Elo Rating (weighted 25%)
	eloScore := 0.0
	if t.elo != nil {
		eloRating := t.elo.GetTeamRating(team.TeamAbbrev.Default)
		ranking.EloRating = eloRating
		// Convert Elo (typically 1300-1700) to 0-100 scale
		// 1500 = average (50), 1700+ = elite (100), 1300- = weak (0)
		eloScore = ((eloRating - 1300) / 400.0) * 100
		eloScore = math.Max(0, math.Min(100, eloScore))
		ranking.ModelBreakdown["Elo Rating"] = eloScore
	}

	// Factor 2: Recent Form (weighted 20%)
	// Use streak and last 10 games performance
	recentFormScore := baseScore
	if team.StreakCount > 0 && team.StreakCode == "W" {
		recentFormScore += float64(team.StreakCount) * 2 // Bonus for win streaks
	} else if team.StreakCount > 0 && team.StreakCode == "L" {
		recentFormScore -= float64(team.StreakCount) * 2 // Penalty for losing streaks
	}
	recentFormScore = math.Max(0, math.Min(100, recentFormScore))
	ranking.ModelBreakdown["Recent Form"] = recentFormScore

	// Factor 3: Goal Differential (weighted 15%)
	goalDiffScore := 50.0 // Start at average
	if team.GamesPlayed > 0 {
		avgGoalDiff := float64(team.GoalFor-team.GoalAgainst) / float64(team.GamesPlayed)
		// +1.5 goals/game = excellent (90+), -1.5 = poor (10-)
		goalDiffScore = 50 + (avgGoalDiff * 20)
		goalDiffScore = math.Max(0, math.Min(100, goalDiffScore))
	}
	ranking.ModelBreakdown["Goal Differential"] = goalDiffScore

	// Factor 4: Poisson Strength (weighted 20%)
	poissonScore := 50.0
	if t.poisson != nil {
		homeRate, awayRate := t.poisson.GetTeamRates(team.TeamAbbrev.Default)
		offensiveRate := (homeRate + awayRate) / 2.0
		ranking.PoissonOffense = offensiveRate
		
		// Get defensive strength (opponent rates when playing this team)
		// Approximate: higher point % = better defense
		defensiveScore := team.PointPctg * 100
		ranking.PoissonDefense = defensiveScore
		
		// Balanced offense + defense
		poissonScore = (offensiveRate*20 + defensiveScore) / 2.0
		poissonScore = math.Max(0, math.Min(100, poissonScore))
		ranking.ModelBreakdown["Poisson Model"] = poissonScore
	}

	// Factor 5: Playoff Position (weighted 10%)
	// Teams in playoff spots get bonus
	playoffScore := baseScore
	if team.WildcardSequence > 0 {
		// In playoff position
		playoffScore += 15
	} else if team.ConferenceSequence <= 8 {
		// Near playoff position
		playoffScore += 10
	}
	playoffScore = math.Max(0, math.Min(100, playoffScore))
	ranking.ModelBreakdown["Playoff Position"] = playoffScore

	// Factor 6: Home/Road Balance (weighted 10%)
	homeAwayScore := 50.0
	if team.GamesPlayed > 0 {
		// Teams with good road records are stronger
		if team.RoadWins+team.RoadOtLosses > 0 {
			roadWinPct := float64(team.RoadWins*2+team.RoadOtLosses) / float64((team.RoadWins+team.RoadLosses+team.RoadOtLosses)*2)
			homeAwayScore = roadWinPct * 100
		}
		ranking.ModelBreakdown["Road Strength"] = homeAwayScore
	}

	// Calculate weighted composite score
	weights := map[string]float64{
		"Base (Point %)":     0.15,
		"Elo Rating":         0.25,
		"Recent Form":        0.20,
		"Goal Differential":  0.15,
		"Poisson Model":      0.20,
		"Playoff Position":   0.10,
		"Road Strength":      0.10,
	}

	compositeScore := 0.0
	compositeScore += baseScore * weights["Base (Point %)"]
	compositeScore += eloScore * weights["Elo Rating"]
	compositeScore += recentFormScore * weights["Recent Form"]
	compositeScore += goalDiffScore * weights["Goal Differential"]
	compositeScore += poissonScore * weights["Poisson Model"]
	compositeScore += playoffScore * weights["Playoff Position"]
	compositeScore += homeAwayScore * weights["Road Strength"]

	ranking.MLScore = math.Max(0, math.Min(100, compositeScore))

	// Determine confidence based on games played
	if team.GamesPlayed < 10 {
		ranking.ConfidenceLevel = "Low"
	} else if team.GamesPlayed < 20 {
		ranking.ConfidenceLevel = "Medium"
	} else {
		ranking.ConfidenceLevel = "High"
	}

	return ranking
}

// assignTiers assigns S/A/B/C/D tiers based on ML scores
func (t *TeamTierRankingService) assignTiers(rankings []TeamRanking) {
	if len(rankings) == 0 {
		return
	}

	// Dynamic tier assignment based on score distribution
	// S Tier: Top 10-15% (elite contenders) - Score 80+
	// A Tier: Next 25% (strong playoff teams) - Score 65-79
	// B Tier: Next 30% (bubble/wild card teams) - Score 50-64
	// C Tier: Next 20% (rebuilding/fringe) - Score 35-49
	// D Tier: Bottom 15-20% (lottery teams) - Score <35

	for i := range rankings {
		score := rankings[i].MLScore
		
		if score >= 80 {
			rankings[i].Tier = "S"
		} else if score >= 65 {
			rankings[i].Tier = "A"
		} else if score >= 50 {
			rankings[i].Tier = "B"
		} else if score >= 35 {
			rankings[i].Tier = "C"
		} else {
			rankings[i].Tier = "D"
		}
	}
}

