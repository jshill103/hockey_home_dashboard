package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

func standing(code, conf, div string, gp, pts int) *models.TeamStanding {
	return &models.TeamStanding{
		TeamAbbrev:     models.TeamNameInfo{Default: code},
		TeamName:       models.TeamNameInfo{Default: code},
		ConferenceName: conf,
		DivisionName:   div,
		GamesPlayed:    gp,
		Points:         pts,
		Wins:           pts / 2,
		PointPctg:      safePct(pts, gp),
	}
}

func safePct(pts, gp int) float64 {
	if gp == 0 {
		return 0
	}
	return float64(pts) / float64(2*gp)
}

func TestPlayoffQualifiers_UsesDivisionFormatNotTopEight(t *testing.T) {
	// Central dominates: C1..C6 occupy conference ranks 1-6; Pacific's best are ranks 7, 9, 10.
	order := []*models.TeamStanding{
		standing("C1", "Western", "Central", 82, 120),
		standing("C2", "Western", "Central", 82, 115),
		standing("C3", "Western", "Central", 82, 110),
		standing("C4", "Western", "Central", 82, 105),
		standing("C5", "Western", "Central", 82, 104),
		standing("C6", "Western", "Central", 82, 103),
		standing("P1", "Western", "Pacific", 82, 102),
		standing("C7", "Western", "Central", 82, 101),
		standing("P2", "Western", "Pacific", 82, 95),
		standing("P3", "Western", "Pacific", 82, 90),
		standing("P4", "Western", "Pacific", 82, 85),
	}
	spots := PlayoffQualifiers(order)

	want := map[string]string{
		"C1": "division", "C2": "division", "C3": "division",
		"P1": "division", "P2": "division", "P3": "division",
		"C4": "wildcard", "C5": "wildcard",
	}
	if len(spots) != len(want) {
		t.Fatalf("got %d qualifiers %v, want %d", len(spots), spots, len(want))
	}
	for code, spot := range want {
		if spots[code] != spot {
			t.Errorf("%s: got %q, want %q", code, spots[code], spot)
		}
	}
	// C6 is 6th in the conference (top 8) but misses: only 2 wild cards
	if _, ok := spots["C6"]; ok {
		t.Error("C6 should miss the playoffs under the division/wild-card format")
	}
}

func TestSimulationRatingFromRecord_ShrinksEarlySeason(t *testing.T) {
	oneGameWinner := standing("UTA", "Western", "Central", 1, 2)
	if r := SimulationRatingFromRecord(oneGameWinner, 1500); r > 1510 {
		t.Errorf("1-0 team rating = %.1f, want close to prior 1500", r)
	}
	if r := SimulationRatingFromRecord(oneGameWinner, 1580); r < 1570 || r > 1590 {
		t.Errorf("1-0 team with 1580 prior = %.1f, want ~1580", r)
	}
	lateSeason := standing("UTA", "Western", "Central", 70, 91) // .650
	r := SimulationRatingFromRecord(lateSeason, 1500)
	if r < 1530 || r > 1560 {
		t.Errorf("70-GP .650 team rating = %.1f, want mostly record-driven (~1545)", r)
	}
	if r := SimulationRatingFromRecord(standing("X", "W", "C", 0, 0), 0); r != 1500 {
		t.Errorf("no games, no prior = %.1f, want 1500", r)
	}
}

func TestEloPredictor_PrepareForSimulationRefreshesCache(t *testing.T) {
	ep := NewEloPredictor()
	ctx := &PredictionContext{Date: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}
	home, away := standing("H", "W", "C", 40, 60), standing("A", "W", "C", 40, 40)

	ep.PrepareForSimulation([]*models.TeamStanding{home, away}, nil)
	p1, _ := ep.PredictWinProbability("H", "A", ctx)

	home.Points, away.Points = 40, 60
	ep.PrepareForSimulation([]*models.TeamStanding{home, away}, nil)
	p2, _ := ep.PredictWinProbability("H", "A", ctx)

	if !(p1 > 0.5 && p2 < p1) {
		t.Errorf("expected refreshed prediction after reseeding: before=%.3f after=%.3f", p1, p2)
	}
}

func TestSimulateSeason_CountsCrossConferenceGames(t *testing.T) {
	ps := NewPlayoffSimulationService(nil)
	uta := standing("UTA", "Western", "Central", 0, 0)
	bos := standing("BOS", "Eastern", "Atlantic", 0, 0)
	conf := []*models.TeamStanding{uta}
	all := []*models.TeamStanding{uta, bos}
	ps.prepareSimulationPredictor(all)

	games := []RemainingGame{
		{HomeTeam: "UTA", AwayTeam: "BOS", Date: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)},
		{HomeTeam: "BOS", AwayTeam: "UTA", Date: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)},
	}
	res := ps.simulateSeason(uta, conf, all, games)
	if gp := res.FinalWins + res.FinalLosses + res.FinalOTLosses; gp != 2 {
		t.Errorf("UTA played %d simulated games, want 2 (cross-conference games must count)", gp)
	}
	if uta.GamesPlayed != 0 {
		t.Error("simulateSeason must not mutate input standings")
	}
}

// One game into the season, a 1-0 team must not be treated as a lock for the playoffs.
func TestSimulation_EarlySeasonOddsAreNotRunaway(t *testing.T) {
	ps := NewPlayoffSimulationService(nil)
	var conf, all []*models.TeamStanding
	for i := 0; i < 16; i++ {
		div := []string{"Central", "Pacific"}[i%2]
		pts := 0
		if (i/2)%2 == 0 {
			pts = 2 // half of each division won its opener
		}
		team := standing(fmt.Sprintf("W%02d", i), "Western", div, 1, pts)
		conf = append(conf, team)
		all = append(all, team)
	}
	target := conf[0] // 1-0 Central team
	ps.prepareSimulationPredictor(all)

	// Round robin x4 (60 games each) spread over the season
	var games []RemainingGame
	day := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	for rep := 0; rep < 4; rep++ {
		for i := 0; i < len(conf); i++ {
			for j := i + 1; j < len(conf); j++ {
				h, a := conf[i], conf[j]
				if rep%2 == 1 {
					h, a = a, h
				}
				games = append(games, RemainingGame{HomeTeam: h.TeamAbbrev.Default, AwayTeam: a.TeamAbbrev.Default, Date: day})
				day = day.Add(3 * time.Hour)
			}
		}
	}

	made := 0
	const sims = 400
	totalGP := 0
	for i := 0; i < sims; i++ {
		r := ps.simulateSeason(target, conf, all, games)
		if r.MadePlayoffs {
			made++
		}
		totalGP += r.FinalWins + r.FinalLosses + r.FinalOTLosses
	}
	odds := float64(made) / sims * 100
	if odds < 30 || odds > 85 {
		t.Errorf("early-season playoff odds for a 1-0 team = %.1f%%, want a realistic 30-85%% (not a runaway)", odds)
	}
	if avgGP := float64(totalGP) / sims; avgGP < 60 || avgGP > 62 {
		t.Errorf("average games played = %.1f, want 61 (1 played + 60 simulated)", avgGP)
	}
}
