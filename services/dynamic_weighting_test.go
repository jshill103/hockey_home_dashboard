package services

import (
	"math"
	"testing"
)

// TestGetCurrentWeightsReturnsCopy pins the bug where the live internal map
// was handed to callers. The ensemble applies a data-quality style adjustment
// to whatever it gets back, so returning the real map meant every prediction
// wrote into service state and the adjustment compounded request over
// request instead of being applied once.
func TestGetCurrentWeightsReturnsCopy(t *testing.T) {
	dws := NewDynamicWeightingService()

	first := dws.GetCurrentWeights()
	if len(first) == 0 {
		t.Skip("no weights configured in this build")
	}

	var name string
	for k := range first {
		name = k
		break
	}
	original := first[name]

	// Simulate a caller scaling the weights it was given.
	first[name] *= 10

	second := dws.GetCurrentWeights()
	if second[name] != original {
		t.Fatalf("caller mutation leaked into the service: %v became %v",
			original, second[name])
	}
}

// TestGetCurrentWeightsReturnsCopyWhenDisabled covers the other return path,
// which handed out baseWeights -- the pristine originals the service falls
// back to. Corrupting those is worse than corrupting the current weights.
func TestGetCurrentWeightsReturnsCopyWhenDisabled(t *testing.T) {
	dws := NewDynamicWeightingService()
	dws.isEnabled = false

	first := dws.GetCurrentWeights()
	if len(first) == 0 {
		t.Skip("no base weights configured in this build")
	}

	var name string
	for k := range first {
		name = k
		break
	}
	original := first[name]
	first[name] = -999

	if got := dws.GetCurrentWeights()[name]; got != original {
		t.Fatalf("base weights were corrupted: %v became %v", original, got)
	}
}

// TestEffectiveMinWeightCannotExceedBudget is the arithmetic the configured
// floor got wrong. A 0.15 floor across nine models reserves 135% of the
// weight budget, which guarantees weak models a large share regardless of
// how badly they perform.
func TestEffectiveMinWeightCannotExceedBudget(t *testing.T) {
	for _, n := range []int{1, 2, 5, 9, 15} {
		floor := effectiveMinWeight(0.15, n)
		if total := floor * float64(n); total > 1.0 {
			t.Fatalf("with %d models the floors demand %.2f of the budget", n, total)
		}
		if floor <= 0 {
			t.Fatalf("with %d models the floor collapsed to %v", n, floor)
		}
		if floor > 0.15 {
			t.Fatalf("with %d models the floor %v exceeds the configured 0.15", n, floor)
		}
	}
}

// TestEffectiveMinWeightLeavesRoomForGoodModels checks the point of the
// change: with nine models, floors must not consume so much of the budget
// that accurate models cannot be rewarded.
func TestEffectiveMinWeightLeavesRoomForGoodModels(t *testing.T) {
	const models = 9
	reserved := effectiveMinWeight(0.15, models) * float64(models)
	if reserved > 0.5 {
		t.Fatalf("floors reserve %.2f of the budget across %d models; too little is left to reward skill",
			reserved, models)
	}
}

// TestApplyWeightConstraintsRespectsFloorAndShift covers the clamp that was
// being discarded: the shift check compared against the pre-clamp value, so
// whenever it engaged it overwrote the floor using a stale number.
func TestApplyWeightConstraintsRespectsFloorAndShift(t *testing.T) {
	dws := NewDynamicWeightingService()

	current := map[string]float64{"A": 0.50, "B": 0.30, "C": 0.20}
	dws.calculator.currentWeights = current
	dws.calculator.weightConstraints = WeightConstraints{
		MinWeight:         0.15,
		MaxWeight:         0.60,
		MaxShiftPerUpdate: 0.03,
		MinSampleSize:     15,
	}

	// "C" is pushed far below any floor in a single step.
	proposed := map[string]float64{"A": 0.50, "B": 0.30, "C": 0.0}
	dws.applyWeightConstraints(proposed)

	floor := effectiveMinWeight(0.15, len(proposed))

	// The shift limit should dominate here, holding C near its current value.
	if proposed["C"] < current["C"]-0.03-1e-9 {
		t.Fatalf("C moved %v, more than the 0.03 shift limit allows (from %v to %v)",
			current["C"]-proposed["C"], current["C"], proposed["C"])
	}
	// And the result must never land under the floor.
	if proposed["C"] < floor-1e-9 {
		t.Fatalf("C=%v fell below the floor %v", proposed["C"], floor)
	}
}

// TestNormalizeWeightsSumsToOne guards the final step, since the blended
// probability is only meaningful if the weights are a true distribution.
func TestNormalizeWeightsSumsToOne(t *testing.T) {
	dws := NewDynamicWeightingService()

	w := map[string]float64{"A": 0.4, "B": 0.4, "C": 0.4}
	dws.normalizeWeights(w)

	total := 0.0
	for _, v := range w {
		total += v
	}
	if math.Abs(total-1.0) > 1e-9 {
		t.Fatalf("weights sum to %v, want 1.0", total)
	}
}
