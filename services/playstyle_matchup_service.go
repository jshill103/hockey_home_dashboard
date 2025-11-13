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
	playstyleMatchupInstance *PlaystyleMatchupService
	playstyleMatchupOnce     sync.Once
)

// PlaystyleMatchupService analyzes team playstyles and matchups
type PlaystyleMatchupService struct {
	mu                 sync.RWMutex
	teamPlaystyles     map[string]*models.PlaystyleProfile     // teamCode -> profile
	matchupMatrix      *models.MatchupAdvantageMatrix          // Style compatibility matrix
	compatibilityRules map[string]map[string]float64           // [Style1][Style2] -> advantage
	dataDir            string
}

// InitializePlaystyleMatchup initializes the singleton PlaystyleMatchupService
func InitializePlaystyleMatchup() error {
	var err error
	playstyleMatchupOnce.Do(func() {
		dataDir := filepath.Join("data", "playstyle")
		if err = os.MkdirAll(dataDir, 0755); err != nil {
			err = fmt.Errorf("failed to create playstyle directory: %w", err)
			return
		}

		playstyleMatchupInstance = &PlaystyleMatchupService{
			teamPlaystyles:     make(map[string]*models.PlaystyleProfile),
			compatibilityRules: make(map[string]map[string]float64),
			dataDir:            dataDir,
		}

		// Initialize compatibility rules
		playstyleMatchupInstance.initializeCompatibilityRules()

		// Load existing playstyle data
		if loadErr := playstyleMatchupInstance.loadPlaystyleData(); loadErr != nil {
			fmt.Printf("⚠️ Warning: Could not load existing playstyle data: %v\n", loadErr)
		}

		fmt.Println("✅ Playstyle Matchup Service initialized")
	})
	return err
}

// GetPlaystyleMatchupService returns the singleton instance
func GetPlaystyleMatchupService() *PlaystyleMatchupService {
	return playstyleMatchupInstance
}

// initializeCompatibilityRules sets up the matchup matrix
func (pms *PlaystyleMatchupService) initializeCompatibilityRules() {
	// Format: positive value = first style has advantage
	// Offensive vs others
	pms.setCompatibility("Offensive", "Defensive", -0.05) // Defensive trap counters offensive
	pms.setCompatibility("Offensive", "Offensive", 0.00)  // Even matchup
	pms.setCompatibility("Offensive", "Possession", 0.03) // Offensive can disrupt possession
	pms.setCompatibility("Offensive", "Speed", 0.02)      // Slight advantage
	pms.setCompatibility("Offensive", "Physical", -0.02)  // Physical slows down offense
	pms.setCompatibility("Offensive", "Skilled", 0.01)    // Slight advantage

	// Defensive vs others
	pms.setCompatibility("Defensive", "Offensive", 0.05)  // Trap works vs offense
	pms.setCompatibility("Defensive", "Defensive", 0.00)  // Low-event game
	pms.setCompatibility("Defensive", "Possession", -0.03) // Possession breaks trap
	pms.setCompatibility("Defensive", "Speed", -0.04)     // Speed beats trap
	pms.setCompatibility("Defensive", "Physical", 0.00)   // Even
	pms.setCompatibility("Defensive", "Skilled", -0.02)   // Skill finds gaps

	// Possession vs others
	pms.setCompatibility("Possession", "Offensive", -0.03) // Offensive disrupts
	pms.setCompatibility("Possession", "Defensive", 0.03)  // Beats trap
	pms.setCompatibility("Possession", "Possession", 0.00) // Even
	pms.setCompatibility("Possession", "Speed", -0.05)     // Speed counters possession
	pms.setCompatibility("Possession", "Physical", 0.02)   // Can control physical game
	pms.setCompatibility("Possession", "Skilled", 0.01)    // Slight advantage

	// Speed vs others
	pms.setCompatibility("Speed", "Offensive", -0.02)  // Slight disadvantage
	pms.setCompatibility("Speed", "Defensive", 0.04)   // Speed beats trap
	pms.setCompatibility("Speed", "Possession", 0.05)  // Speed counters possession
	pms.setCompatibility("Speed", "Speed", 0.00)       // High-event game
	pms.setCompatibility("Speed", "Physical", 0.03)    // Can avoid physicality
	pms.setCompatibility("Speed", "Skilled", 0.02)     // Slight advantage

	// Physical vs others
	pms.setCompatibility("Physical", "Offensive", 0.02)  // Slows offense
	pms.setCompatibility("Physical", "Defensive", 0.00)  // Even
	pms.setCompatibility("Physical", "Possession", -0.02) // Possession avoids hits
	pms.setCompatibility("Physical", "Speed", -0.03)     // Speed avoids hits
	pms.setCompatibility("Physical", "Physical", 0.00)   // Even
	pms.setCompatibility("Physical", "Skilled", 0.04)    // Disrupts skill

	// Skilled vs others
	pms.setCompatibility("Skilled", "Offensive", -0.01)  // Slight disadvantage
	pms.setCompatibility("Skilled", "Defensive", 0.02)   // Finds gaps
	pms.setCompatibility("Skilled", "Possession", -0.01) // Slight disadvantage
	pms.setCompatibility("Skilled", "Speed", -0.02)      // Speed advantage
	pms.setCompatibility("Skilled", "Physical", -0.04)   // Physical disrupts
	pms.setCompatibility("Skilled", "Skilled", 0.00)     // Even
}

func (pms *PlaystyleMatchupService) setCompatibility(style1, style2 string, value float64) {
	if pms.compatibilityRules[style1] == nil {
		pms.compatibilityRules[style1] = make(map[string]float64)
	}
	pms.compatibilityRules[style1][style2] = value
}

// AnalyzePlaystyle determines a team's playstyle
func (pms *PlaystyleMatchupService) AnalyzePlaystyle(teamCode string, stats models.TeamStats) *models.PlaystyleProfile {
	pms.mu.Lock()
	defer pms.mu.Unlock()

	profile := &models.PlaystyleProfile{
		TeamCode:    teamCode,
		Season:      getCurrentSeason(),
		LastUpdated: time.Now(),
	}

	// Calculate style ratings based on stats
	profile.OffensiveRating = pms.calculateOffensiveRating(stats)
	profile.DefensiveRating = pms.calculateDefensiveRating(stats)
	profile.PossessionRating = pms.calculatePossessionRating(stats)
	profile.SpeedRating = pms.calculateSpeedRating(stats)
	profile.PhysicalRating = pms.calculatePhysicalRating(stats)
	profile.SkillRating = pms.calculateSkillRating(stats)

	// Determine primary style
	profile.PrimaryStyle = pms.determinePrimaryStyle(profile)
	profile.StyleConfidence = 0.75 // Base confidence

	// Store supporting stats
	profile.AvgGoalsFor = stats.GoalsFor / float64(stats.GamesPlayed)
	profile.AvgGoalsAgainst = stats.GoalsAgainst / float64(stats.GamesPlayed)
	profile.AvgShots = stats.ShotsFor / float64(stats.GamesPlayed)

	pms.teamPlaystyles[teamCode] = profile
	pms.savePlaystyleData()

	return profile
}

func (pms *PlaystyleMatchupService) calculateOffensiveRating(stats models.TeamStats) float64 {
	// Based on goals scored and shot volume
	gfPerGame := stats.GoalsFor / float64(stats.GamesPlayed)
	shotsPerGame := stats.ShotsFor / float64(stats.GamesPlayed)
	
	// Normalize: 3.5 goals/game = 0.7, 33 shots/game = 0.7
	goalRating := math.Min(gfPerGame/5.0, 1.0)
	shotRating := math.Min(shotsPerGame/40.0, 1.0)
	
	return (goalRating*0.6 + shotRating*0.4)
}

func (pms *PlaystyleMatchupService) calculateDefensiveRating(stats models.TeamStats) float64 {
	// Based on goals against (lower is better)
	gaPerGame := stats.GoalsAgainst / float64(stats.GamesPlayed)
	
	// Normalize: 2.0 GA/game = 1.0 (excellent), 4.0 GA/game = 0.0 (poor)
	rating := 1.0 - ((gaPerGame - 2.0) / 2.0)
	return math.Max(0.0, math.Min(1.0, rating))
}

func (pms *PlaystyleMatchupService) calculatePossessionRating(stats models.TeamStats) float64 {
	// Based on faceoff win % (proxy for possession)
	return stats.FaceoffWinPct
}

func (pms *PlaystyleMatchupService) calculateSpeedRating(stats models.TeamStats) float64 {
	// Estimated based on shot differential and goals
	// Teams with high shot differential relative to goals are fast/transition-focused
	if stats.GamesPlayed == 0 {
		return 0.5
	}
	shotDiff := (stats.ShotsFor - stats.ShotsAgainst) / float64(stats.GamesPlayed)
	normalized := (shotDiff + 10.0) / 20.0 // -10 to +10 range
	return math.Max(0.0, math.Min(1.0, normalized))
}

func (pms *PlaystyleMatchupService) calculatePhysicalRating(stats models.TeamStats) float64 {
	// Estimated - would use hits data if available
	// For now, use penalty minutes as proxy
	pimPerGame := float64(stats.PenaltyMinutes) / float64(stats.GamesPlayed)
	// Normalize: 10 PIM/game = 0.7
	rating := pimPerGame / 15.0
	return math.Min(rating, 1.0)
}

func (pms *PlaystyleMatchupService) calculateSkillRating(stats models.TeamStats) float64 {
	// Based on power play % and shooting %
	ppRating := stats.PowerPlayPct
	// Combine with offensive rating
	return (ppRating + pms.calculateOffensiveRating(stats)) / 2.0
}

func (pms *PlaystyleMatchupService) determinePrimaryStyle(profile *models.PlaystyleProfile) string {
	styles := map[string]float64{
		"Offensive":  profile.OffensiveRating,
		"Defensive":  profile.DefensiveRating,
		"Possession": profile.PossessionRating,
		"Speed":      profile.SpeedRating,
		"Physical":   profile.PhysicalRating,
		"Skilled":    profile.SkillRating,
	}

	maxStyle := "Offensive"
	maxRating := 0.0
	for style, rating := range styles {
		if rating > maxRating {
			maxRating = rating
			maxStyle = style
		}
	}

	return maxStyle
}

// ComparePlaystyles analyzes matchup between two team styles
func (pms *PlaystyleMatchupService) ComparePlaystyles(homeTeam, awayTeam string) *models.PlaystyleMatchup {
	pms.mu.RLock()
	defer pms.mu.RUnlock()

	homeProfile := pms.getPlaystyleProfile(homeTeam)
	awayProfile := pms.getPlaystyleProfile(awayTeam)

	compatibility := pms.GetStyleCompatibility(homeProfile.PrimaryStyle, awayProfile.PrimaryStyle)

	matchup := &models.PlaystyleMatchup{
		HomeTeam:      homeTeam,
		AwayTeam:      awayTeam,
		HomeStyle:     homeProfile.PrimaryStyle,
		AwayStyle:     awayProfile.PrimaryStyle,
		Compatibility: compatibility,
		ImpactFactor:  compatibility * 0.10, // Scale to -0.10 to +0.10
		Confidence:    (homeProfile.StyleConfidence + awayProfile.StyleConfidence) / 2.0,
		LastUpdated:   time.Now(),
	}

	// Determine advantage
	if matchup.ImpactFactor > 0.02 {
		matchup.Advantage = "Home"
	} else if matchup.ImpactFactor < -0.02 {
		matchup.Advantage = "Away"
	} else {
		matchup.Advantage = "Neutral"
	}

	// Generate explanation
	matchup.Explanation = pms.generateMatchupExplanation(matchup)

	return matchup
}

// GetStyleCompatibility returns compatibility between two styles
func (pms *PlaystyleMatchupService) GetStyleCompatibility(homeStyle, awayStyle string) float64 {
	if compat, exists := pms.compatibilityRules[homeStyle][awayStyle]; exists {
		return compat
	}
	return 0.0 // Neutral if not defined
}

func (pms *PlaystyleMatchupService) getPlaystyleProfile(teamCode string) *models.PlaystyleProfile {
	if profile, exists := pms.teamPlaystyles[teamCode]; exists {
		return profile
	}

	// Return neutral profile if not found
	return &models.PlaystyleProfile{
		TeamCode:         teamCode,
		PrimaryStyle:     "Offensive",
		StyleConfidence:  0.5,
		OffensiveRating:  0.5,
		DefensiveRating:  0.5,
		PossessionRating: 0.5,
		SpeedRating:      0.5,
		PhysicalRating:   0.5,
		SkillRating:      0.5,
	}
}

func (pms *PlaystyleMatchupService) generateMatchupExplanation(matchup *models.PlaystyleMatchup) string {
	if matchup.Advantage == "Neutral" {
		return fmt.Sprintf("%s vs %s is an even stylistic matchup", matchup.HomeStyle, matchup.AwayStyle)
	}

	advantage := matchup.Advantage
	favoredStyle := matchup.HomeStyle
	if advantage == "Away" {
		favoredStyle = matchup.AwayStyle
	}

	return fmt.Sprintf("%s style has advantage over %s style (%.1f%% impact)",
		favoredStyle, 
		getOpponentStyle(matchup, advantage),
		math.Abs(matchup.ImpactFactor)*100)
}

func getOpponentStyle(matchup *models.PlaystyleMatchup, advantage string) string {
	if advantage == "Home" {
		return matchup.AwayStyle
	}
	return matchup.HomeStyle
}

// ============================================================================
// PERSISTENCE
// ============================================================================

func (pms *PlaystyleMatchupService) getPlaystyleDataPath() string {
	return filepath.Join(pms.dataDir, "team_playstyles.json")
}

func (pms *PlaystyleMatchupService) savePlaystyleData() error {
	data, err := json.MarshalIndent(pms.teamPlaystyles, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal playstyle data: %w", err)
	}
	if err := os.WriteFile(pms.getPlaystyleDataPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write playstyle data: %w", err)
	}
	return nil
}

func (pms *PlaystyleMatchupService) loadPlaystyleData() error {
	filePath := pms.getPlaystyleDataPath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data yet
		}
		return fmt.Errorf("failed to read playstyle data: %w", err)
	}

	if err := json.Unmarshal(data, &pms.teamPlaystyles); err != nil {
		return fmt.Errorf("failed to unmarshal playstyle data: %w", err)
	}

	fmt.Printf("🎨 Loaded playstyle data for %d teams\n", len(pms.teamPlaystyles))
	return nil
}

