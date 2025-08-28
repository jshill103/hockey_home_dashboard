package models

// Performance Analysis Objects
type TeamPerformance struct {
	TeamStanding
	TrendAnalysis         string
	PlayoffChances        string
	KeyStrengths          []string
	KeyWeaknesses         []string
	RecentForm            string
	HomeRoadBalance       string
	GoalScoringTrend      string
	DefensiveTrend        string
} 