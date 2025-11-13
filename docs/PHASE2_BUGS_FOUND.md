# Phase 2 Bugs Found

## 🐛 Critical Bugs

### Bug 1: Race Condition in HeadToHeadService.GetMatchupAnalysis
**Severity**: High  
**File**: `services/head_to_head_service.go`  
**Lines**: 150-172

**Problem**:
```go
func (h2h *HeadToHeadService) GetMatchupAnalysis(homeTeam, awayTeam string) (*models.HeadToHeadRecord, error) {
    h2h.mu.RLock()  // READ lock acquired
    defer h2h.mu.RUnlock()
    
    // ...
    
    // Recalculate recency weights
    h2h.calculateRecencyWeights(record)  // BUG: Modifies record while holding READ lock!
    
    // ...
}
```

The `calculateRecencyWeights` function modifies `record.RecentGames` (sorts and updates Recency field) while only holding a READ lock. This is a race condition that can cause data corruption if multiple goroutines access the same record.

**Impact**: Potential data corruption, panics in concurrent access scenarios

**Fix**: Should use a write lock (RW mutex upgrade) or return a copy of the record

---

### Bug 2: Feature Count Mismatch in Gradient Boosting Model
**Severity**: High  
**File**: `services/gradient_boosting.go`  
**Line**: 79

**Problem**:
```go
featureNames: make([]string, 156), // 156 features (140 Phase 1 + 16 Phase 2)
```

But the Neural Network has **182 features** (148 base + 20 interactions + 8 additional + 6 Phase 2).

**Impact**: Dimension mismatch will cause prediction failures when features are passed to Gradient Boosting model

**Fix**: Update to 182 features

---

### Bug 3: Missing Initialization of OpponentFatigue
**Severity**: Medium  
**File**: `services/ensemble_predictions.go`  
**Lines**: 204-209

**Problem**:
```go
// Opponent fatigue (0-1 scale, higher = more tired)
if restAdv.AwayOnB2B {
    homeFactors.OpponentFatigue = 0.75 + (restAdv.AwayB2BPenalty * -2.0)
}
if restAdv.HomeOnB2B {
    awayFactors.OpponentFatigue = 0.75 + (restAdv.HomeB2BPenalty * -2.0)
}
```

If neither team is on a B2B, `OpponentFatigue` is never initialized and remains at its zero value.

**Impact**: Missing data for models when neither team is on B2B

**Fix**: Initialize with default value (e.g., 0.0 for no fatigue)

---

### Bug 4: Empty RecentGames Can Cause Division By Zero
**Severity**: Medium  
**File**: `services/head_to_head_service.go`  
**Lines**: 165-176

**Problem**:
```go
record, _ := h2hService.GetMatchupAnalysis(homeFactors.TeamCode, awayFactors.TeamCode)
if record != nil && len(record.RecentGames) > 0 {
    recentWins := 0
    for _, game := range record.RecentGames {
        if game.Winner == homeFactors.TeamCode {
            recentWins++
        }
    }
    homeFactors.H2HRecentForm = float64(recentWins) / float64(len(record.RecentGames))
    // ...
}
```

The check is there, but if `GetMatchupAnalysis` returns an empty record with initialized but empty `RecentGames`, the division is safe but the feature won't be set, leaving it at 0.

**Impact**: Minor - Feature will be 0 instead of uninitialized, which is probably okay

**Fix**: Could set default to 0.5 (neutral) when no data exists

---

## ⚠️ Medium Priority Bugs

### Bug 5: Potential Nil Pointer in Goalie Matchup History
**Severity**: Medium  
**File**: `services/ensemble_predictions.go`  
**Lines**: 133-150

**Problem**:
```go
// PHASE 2: Enhanced Goalie Matchup History
if goalieService != nil && comparison != nil {
    // Get home goalie ID
    if comparison.HomeGoalie != nil {
        matchupAdj := goalieService.GetGoalieMatchupAdjustment(comparison.HomeGoalie.PlayerID, awayFactors.TeamCode)
        // ...
    }
    // ...
}
```

This is properly protected, but if `comparison.HomeGoalie` is nil, the feature won't be set. This is handled gracefully but worth noting.

**Impact**: Low - Graceful degradation

**Fix**: None needed, but could set default value

---

### Bug 6: RestAdvantageDetailed Not Initialized When Service is Nil
**Severity**: Low  
**File**: `services/ensemble_predictions.go`  
**Lines**: 185-216

**Problem**:
If `restService` is nil, the Phase 2 rest features are never initialized.

**Impact**: Features remain at zero value, models will see 0 for all rest-related Phase 2 features

**Fix**: Initialize with defaults when service is unavailable

---

### Bug 7: H2HRecentForm Not Set When No Recent Games
**Severity**: Low  
**File**: `services/ensemble_predictions.go`  
**Lines**: 165-176

**Problem**:
When `len(record.RecentGames) == 0`, the feature isn't set.

**Impact**: Feature defaults to 0, which might be interpreted as "all losses" rather than "no data"

**Fix**: Set to 0.5 (neutral) when no data exists

---

## 📝 Minor Issues

### Issue 1: Inconsistent Season Format in Record Storage
**File**: `services/head_to_head_service.go`  
**Line**: 385

Records are saved with season suffix: `{key}_{season}.json`

This is fine but means records won't automatically carry over between seasons. Worth documenting.

---

### Issue 2: calculateWeightedAdvantage Can Produce NaN
**File**: `services/head_to_head_service.go`  
**Lines**: 275-322

If `record.TotalGames` is 0, we could potentially divide by zero in calculations.

**Impact**: Very low - should be caught by the check at line 277

**Fix**: Already has safeguard, but could add explicit NaN check

---

### Issue 3: Sort.Slice Modifies Data During Read
**File**: `services/head_to_head_service.go`  
**Lines**: 325-327

This is part of Bug #1 but worth noting separately - sorting modifies the slice order which affects all concurrent readers.

---

## 🔧 Recommended Fixes (Priority Order)

### 1. Fix Race Condition (Bug #1) - CRITICAL
```go
func (h2h *HeadToHeadService) GetMatchupAnalysis(homeTeam, awayTeam string) (*models.HeadToHeadRecord, error) {
    h2h.mu.RLock()
    key := h2h.getMatchupKey(homeTeam, awayTeam)
    record, exists := h2h.records[key]
    h2h.mu.RUnlock()
    
    if !exists {
        return &models.HeadToHeadRecord{
            HomeTeam:          homeTeam,
            AwayTeam:          awayTeam,
            Season:            h2h.currentSeason,
            WeightedAdvantage: 0.0,
        }, nil
    }
    
    // Create a copy to avoid race conditions
    recordCopy := *record
    if record.RecentGames != nil {
        recordCopy.RecentGames = make([]models.H2HGame, len(record.RecentGames))
        copy(recordCopy.RecentGames, record.RecentGames)
    }
    
    // Now safe to modify the copy
    h2h.calculateRecencyWeights(&recordCopy)
    recordCopy.DaysSinceLastMeet = int(time.Since(recordCopy.LastMeetingDate).Hours() / 24)
    
    return &recordCopy, nil
}
```

### 2. Fix Feature Count Mismatch (Bug #2) - CRITICAL
```go
// In gradient_boosting.go line 79:
featureNames: make([]string, 182), // 182 features to match Neural Network
```

### 3. Initialize OpponentFatigue (Bug #3) - HIGH
```go
// In ensemble_predictions.go, add after line 209:
// Initialize defaults if not set
if homeFactors.OpponentFatigue == 0 && !restAdv.AwayOnB2B {
    homeFactors.OpponentFatigue = 0.0 // No opponent fatigue
}
if awayFactors.OpponentFatigue == 0 && !restAdv.HomeOnB2B {
    awayFactors.OpponentFatigue = 0.0 // No opponent fatigue
}
```

### 4. Set Default for H2HRecentForm (Bug #7) - MEDIUM
```go
// In ensemble_predictions.go, modify lines 165-176:
record, _ := h2hService.GetMatchupAnalysis(homeFactors.TeamCode, awayFactors.TeamCode)
if record != nil && len(record.RecentGames) > 0 {
    recentWins := 0
    for _, game := range record.RecentGames {
        if game.Winner == homeFactors.TeamCode {
            recentWins++
        }
    }
    homeFactors.H2HRecentForm = float64(recentWins) / float64(len(record.RecentGames))
    awayFactors.H2HRecentForm = 1.0 - homeFactors.H2HRecentForm
} else {
    // No H2H data, use neutral values
    homeFactors.H2HRecentForm = 0.5
    awayFactors.H2HRecentForm = 0.5
}
```

---

## 📊 Bug Summary

| Severity | Count | Fixed |
|----------|-------|-------|
| Critical | 2 | ❌ |
| High | 1 | ❌ |
| Medium | 4 | ❌ |
| Low | 3 | ❌ |

**Total Bugs**: 10  
**Bugs Fixed**: 0  
**Bugs Remaining**: 10

---

## 🎯 Testing Recommendations

After fixes are applied:

1. **Race Condition Test**: Run concurrent predictions with same team matchups
2. **Feature Dimension Test**: Verify all models receive correct feature count
3. **Default Value Test**: Verify all Phase 2 features have sensible defaults
4. **Integration Test**: Run end-to-end predictions with Phase 2 enabled
5. **Load Test**: Stress test with high concurrency

---

## 📝 Notes

- Most bugs are low severity and represent graceful degradation scenarios
- The race condition (Bug #1) is the most serious and should be fixed immediately
- Feature count mismatch (Bug #2) will cause immediate failures and must be fixed
- Other bugs are defensive improvements that will make the system more robust

---

**Report Generated**: 2025-11-13  
**Phase**: 2 (Enhanced Data Quality)  
**Status**: Bugs identified, fixes pending

