package services

import (
	"math"
	"testing"

	"github.com/jaredshillingburg/go_uhc/models"
)

// TestPossessionSharesSumToOne pins the invariant the old code violated.
// Corsi-for and Corsi-against are the two halves of the same set of shot
// attempts, so the two teams' shares must add up to exactly 1. The previous
// implementation never populated CorsiAgainst, so every team scored
// CF/(CF+0) = 1.0 and the pair summed to 2.
func TestPossessionSharesSumToOne(t *testing.T) {
	pbp := &PlayByPlayService{}

	home := &models.TeamPlayAnalytics{
		TeamCode:     "NYR",
		ShotAttempts: 60,
		ShotsOnGoal:  32,
		MissedShots:  14,
	}
	away := &models.TeamPlayAnalytics{
		TeamCode:     "UTA",
		ShotAttempts: 40,
		ShotsOnGoal:  22,
		MissedShots:  9,
	}

	pbp.calculateDerivedMetrics(home, away)
	pbp.calculateDerivedMetrics(away, home)

	if got := home.CorsiForPct + away.CorsiForPct; math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("Corsi shares sum to %v, want 1.0 (home=%v away=%v)",
			got, home.CorsiForPct, away.CorsiForPct)
	}
	if got := home.FenwickForPct + away.FenwickForPct; math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("Fenwick shares sum to %v, want 1.0 (home=%v away=%v)",
			got, home.FenwickForPct, away.FenwickForPct)
	}

	// 60 of 100 attempts is 60%, the number a box score would report.
	if math.Abs(home.CorsiForPct-0.60) > 1e-9 {
		t.Fatalf("home Corsi share = %v, want 0.60", home.CorsiForPct)
	}
	if home.CorsiAgainst != away.ShotAttempts {
		t.Fatalf("home CorsiAgainst = %d, want the opponent's %d attempts",
			home.CorsiAgainst, away.ShotAttempts)
	}
}

// TestPossessionShareIsNotAlwaysOne is the direct regression guard: the
// dominant team must not read 100%.
func TestPossessionShareIsNotAlwaysOne(t *testing.T) {
	pbp := &PlayByPlayService{}

	home := &models.TeamPlayAnalytics{ShotAttempts: 70, ShotsOnGoal: 40, MissedShots: 18}
	away := &models.TeamPlayAnalytics{ShotAttempts: 30, ShotsOnGoal: 15, MissedShots: 8}

	pbp.calculateDerivedMetrics(home, away)

	if home.CorsiForPct >= degeneratePossessionThreshold {
		t.Fatalf("home Corsi share = %v; the opponent's attempts were ignored again",
			home.CorsiForPct)
	}
	// The team with more attempts must hold the larger share.
	if home.CorsiForPct <= neutralPossessionShare {
		t.Fatalf("team with 70 of 100 attempts should hold over half, got %v",
			home.CorsiForPct)
	}
}

// TestRepairDegeneratePossessionShares covers the migration for values
// already on disk from before the fix.
func TestRepairDegeneratePossessionShares(t *testing.T) {
	stats := map[string]*models.TeamPlayByPlayStats{
		"NYR": {TeamCode: "NYR", AvgCorsiForPct: 1.0, AvgFenwickForPct: 1.0},
		"UTA": {TeamCode: "UTA", AvgCorsiForPct: 0.54, AvgFenwickForPct: 0.51},
		"BOS": nil,
	}

	if n := repairDegeneratePossessionShares(stats); n != 1 {
		t.Fatalf("repaired %d teams, want 1", n)
	}

	if stats["NYR"].AvgCorsiForPct != neutralPossessionShare {
		t.Fatalf("NYR Corsi = %v, want reset to %v",
			stats["NYR"].AvgCorsiForPct, neutralPossessionShare)
	}
	if stats["UTA"].AvgCorsiForPct != 0.54 {
		t.Fatalf("UTA Corsi = %v, a plausible value must be left alone",
			stats["UTA"].AvgCorsiForPct)
	}
}
