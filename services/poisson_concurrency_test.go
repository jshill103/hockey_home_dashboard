package services

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

// The Poisson rate maps are read on prediction goroutines and written by the
// periodic Dixon-Coles refit, so they have to be safe to touch concurrently.
// getOffensiveRate/getDefensiveRate also insert on a cache miss, which makes
// even the read path a writer. Run with -race.
func TestPoissonRateAccessIsRaceFree(t *testing.T) {
	sl := newSyntheticLeague()
	rng := rand.New(rand.NewSource(syntheticRandomSeed))
	games := sl.generateSeason(rng)

	pr := newTestPoissonModel(t)

	var wg sync.WaitGroup
	const readers = 8
	const iterations = 50

	// Writer: repeated full refits, as AddGameToBatch triggers.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_ = pr.FitFromHistory(games)
		}
	}()

	// Readers: rate lookups, including for teams never seen before, which
	// forces the insert path.
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				code := sl.teams[i%len(sl.teams)]
				_ = pr.getOffensiveRate(code)
				_ = pr.getDefensiveRate(code)
				_, _ = pr.GetTeamRates(code)

				// A code no other goroutine will have inserted yet.
				fresh := fmt.Sprintf("N%d_%d", id, i)
				_ = pr.getOffensiveRate(fresh)
				_ = pr.getDefensiveRate(fresh)
			}
		}(r)
	}

	// Concurrent online updates, as processGameResult performs.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			g := games[i%len(games)]
			pr.UpdateRates(g.HomeTeam.TeamCode, g.AwayTeam.TeamCode, g.HomeTeam.Score, g.AwayTeam.Score)
		}
	}()

	wg.Wait()
}
