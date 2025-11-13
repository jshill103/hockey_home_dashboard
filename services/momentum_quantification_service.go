package services

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

var (
	momentumInstance *MomentumQuantificationService
	momentumOnce     sync.Once
)

// MomentumQuantificationService quantifies team momentum
type MomentumQuantificationService struct {
	mu            sync.RWMutex
	teamMomentum  map[string]*models.MomentumScore // teamCode -> momentum
	dataDir       string
}

// InitializeMomentumQuantification initializes the singleton MomentumQuantificationService
func InitializeMomentumQuantification() error {
	var err error
	momentumOnce.Do(func() {
		dataDir := filepath.Join("data", "momentum")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create momentum directory: %w", err)
			return
		}

		momentumInstance = &MomentumQuantificationService{
			teamMomentum: make(map[string]*models.MomentumScore),
			dataDir:      dataDir,
		}

		// Load existing momentum data
		if loadErr := momentumInstance.loadMomentumData(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing momentum data: %v\n", loadErr)
		}

		fmt.Println("✅ Momentum Quantification Service initialized")
	})
	return err
}

// GetMomentumService returns the singleton instance
func GetMomentumService() *MomentumQuantificationService {
	return momentumInstance
}

// CalculateMomentum calculates momentum for a team based on recent games
func (mqs *MomentumQuantificationService) CalculateMomentum(teamCode string, recentGames []models.GameResult) *models.MomentumScore {
	mqs.mu.Lock()
	defer mqs.mu.Unlock()

	momentum := &models.MomentumScore{
		TeamCode:    teamCode,
		Season:      getCurrentSeason(),
		LastUpdated: time.Now(),
	}

	if len(recentGames) == 0 {
		// Return neutral momentum
		momentum.Overall = 0.0
		momentum.Trend = 0.0
		momentum.Confidence = 0.3
		mqs.teamMomentum[teamCode] = momentum
		return momentum
	}

	// Calculate exponentially weighted performance (most recent games matter most)
	var weightedScore float64
	var totalWeight float64
	var scoringMomentum float64
	var qualityMomentum float64

	for i := len(recentGames) - 1; i >= 0; i-- {
		game := recentGames[i]
		position := len(recentGames) - 1 - i // 0 = most recent
		weight := math.Exp(-float64(position) * 0.3) // Exponential decay
		
		// Performance score
		var gameScore float64
		if game.Result == "W" {
			gameScore = 1.0
			// Bigger wins = more momentum
			if game.GoalDifferential > 0 {
				gameScore += math.Min(float64(game.GoalDifferential)*0.1, 0.3)
			}
		} else if game.Result == "OTL" {
			gameScore = 0.1 // Small positive for OT loss
		} else {
			gameScore = -1.0
			// Bad losses hurt more
			if game.GoalDifferential < 0 {
				gameScore -= math.Min(float64(-game.GoalDifferential)*0.1, 0.3)
			}
		}

		weightedScore += gameScore * weight
		totalWeight += weight

		// Scoring momentum (goal differential trend)
		scoringMomentum += float64(game.GoalDifferential) * weight

		// Quality momentum (opponent strength)
		if game.OpponentStrength > 0 {
			qualityMomentum += gameScore * game.OpponentStrength * weight
		}
	}

	// Normalize scores
	if totalWeight > 0 {
		momentum.PerformanceMomentum = weightedScore / totalWeight
		momentum.ScoringMomentum = scoringMomentum / (totalWeight * 3.0) // Normalize to -1 to +1
		momentum.QualityMomentum = qualityMomentum / totalWeight
	}

	// Overall momentum is weighted average of components
	momentum.Overall = (momentum.PerformanceMomentum*0.5 + 
		momentum.ScoringMomentum*0.3 + 
		momentum.QualityMomentum*0.2)

	// Clamp to -1.0 to +1.0
	momentum.Overall = math.Max(-1.0, math.Min(1.0, momentum.Overall))

	// Calculate trend (comparing recent vs older games)
	momentum.Trend = mqs.calculateTrend(recentGames)
	
	// Calculate acceleration (is trend increasing?)
	momentum.Acceleration = momentum.Trend * 0.5 // Simplified

	// Calculate consistency
	momentum.Consistency = mqs.calculateConsistency(recentGames)

	// Calculate confidence based on sample size and consistency
	momentum.Confidence = math.Min(1.0, (float64(len(recentGames))/10.0)*momentum.Consistency)

	// Calculate impact factor for predictions
	momentum.ImpactFactor = momentum.Overall * 0.12 * momentum.Confidence

	mqs.teamMomentum[teamCode] = momentum
	mqs.saveMomentumData()

	return momentum
}

// calculateTrend determines if momentum is increasing or decreasing
func (mqs *MomentumQuantificationService) calculateTrend(games []models.GameResult) float64 {
	if len(games) < 4 {
		return 0.0
	}

	// Compare last 3 games vs previous 3 games
	recentStart := len(games) - 3
	olderStart := len(games) - 6
	if olderStart < 0 {
		olderStart = 0
	}

	recentScore := 0.0
	olderScore := 0.0

	for i := recentStart; i < len(games); i++ {
		if games[i].Result == "W" {
			recentScore += 1.0
		} else if games[i].Result == "L" {
			recentScore -= 1.0
		}
	}

	for i := olderStart; i < recentStart && i < len(games); i++ {
		if games[i].Result == "W" {
			olderScore += 1.0
		} else if games[i].Result == "L" {
			olderScore -= 1.0
		}
	}

	// Normalize and compare
	recentAvg := recentScore / 3.0
	olderAvg := olderScore / float64(recentStart-olderStart)

	trend := recentAvg - olderAvg
	return math.Max(-1.0, math.Min(1.0, trend))
}

// calculateConsistency measures how consistent the momentum is
func (mqs *MomentumQuantificationService) calculateConsistency(games []models.GameResult) float64 {
	if len(games) < 3 {
		return 0.5
	}

	// Calculate variance in recent results
	var scores []float64
	for _, game := range games {
		if game.Result == "W" {
			scores = append(scores, 1.0)
		} else if game.Result == "OTL" {
			scores = append(scores, 0.1)
		} else {
			scores = append(scores, -1.0)
		}
	}

	// Calculate mean
	var sum float64
	for _, score := range scores {
		sum += score
	}
	mean := sum / float64(len(scores))

	// Calculate variance
	var variance float64
	for _, score := range scores {
		variance += math.Pow(score-mean, 2)
	}
	variance /= float64(len(scores))

	// Convert variance to consistency (lower variance = higher consistency)
	consistency := 1.0 / (1.0 + variance)
	return consistency
}

// CompareMomentum compares momentum between two teams
func (mqs *MomentumQuantificationService) CompareMomentum(homeTeam, awayTeam string) *models.MomentumComparison {
	homeMomentum := mqs.GetMomentum(homeTeam)
	awayMomentum := mqs.GetMomentum(awayTeam)

	gap := homeMomentum.Overall - awayMomentum.Overall
	impact := gap * 0.08 * math.Min(homeMomentum.Confidence, awayMomentum.Confidence)

	advantage := "Neutral"
	if gap > 0.3 {
		advantage = "Home"
	} else if gap < -0.3 {
		advantage = "Away"
	}

	trendAnalysis := fmt.Sprintf("Home momentum: %.2f (%s trend), Away momentum: %.2f (%s trend)",
		homeMomentum.Overall, getTrendDirection(homeMomentum.Trend),
		awayMomentum.Overall, getTrendDirection(awayMomentum.Trend))

	return &models.MomentumComparison{
		HomeTeam:          homeTeam,
		AwayTeam:          awayTeam,
		HomeMomentum:      *homeMomentum,
		AwayMomentum:      *awayMomentum,
		Advantage:         advantage,
		MomentumGap:       gap,
		ImpactFactor:      impact,
		Confidence:        (homeMomentum.Confidence + awayMomentum.Confidence) / 2.0,
		TrendAnalysis:     trendAnalysis,
		RecommendedWeight: math.Min(homeMomentum.Confidence, awayMomentum.Confidence),
	}
}

// GetMomentum retrieves current momentum for a team
func (mqs *MomentumQuantificationService) GetMomentum(teamCode string) *models.MomentumScore {
	mqs.mu.RLock()
	defer mqs.mu.RUnlock()

	if momentum, exists := mqs.teamMomentum[teamCode]; exists {
		return momentum
	}

	// Return neutral momentum if not found
	return &models.MomentumScore{
		TeamCode:   teamCode,
		Overall:    0.0,
		Trend:      0.0,
		Confidence: 0.3,
	}
}

func getTrendDirection(trend float64) string {
	if trend > 0.2 {
		return "rising"
	} else if trend < -0.2 {
		return "falling"
	}
	return "stable"
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (mqs *MomentumQuantificationService) getMomentumDataPath() string {
	return filepath.Join(mqs.dataDir, "team_momentum.json")
}

func (mqs *MomentumQuantificationService) saveMomentumData() error {
	data, err := json.MarshalIndent(mqs.teamMomentum, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal momentum data: %w", err)
	}
	if err := os.WriteFile(mqs.getMomentumDataPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write momentum data: %w", err)
	}
	return nil
}

func (mqs *MomentumQuantificationService) loadMomentumData() error {
	filePath := mqs.getMomentumDataPath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data yet
		}
		return fmt.Errorf("failed to read momentum data: %w", err)
	}

	if err := json.Unmarshal(data, &mqs.teamMomentum); err != nil {
		return fmt.Errorf("failed to unmarshal momentum data: %w", err)
	}

	fmt.Printf("📈 Loaded momentum data for %d teams\n", len(mqs.teamMomentum))
	return nil
}

