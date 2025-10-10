# Critical Fixes Implementation Plan

## Issue #1: NaN Values in Predictions (Lines 966, 971 in logs)

### Root Cause:
- **Location**: `services/situational_analysis.go:640` and `services/predictions.go:231`
- **Problem**: `WinPercentage = Wins / GamesPlayed` when `GamesPlayed = 0` → produces `NaN`
- **Impact**: Cascades through entire statistical model causing invalid predictions

### Fix Strategy:
```go
// BEFORE (Line 640):
WinPercentage: float64(teamStanding.Wins) / float64(teamStanding.GamesPlayed),

// AFTER:
WinPercentage: safeDiv(float64(teamStanding.Wins), float64(teamStanding.GamesPlayed), 0.5),

// Helper function:
func safeDiv(numerator, denominator, defaultValue float64) float64 {
    if denominator == 0 || math.IsNaN(denominator) || math.IsInf(denominator, 0) {
        return defaultValue
    }
    result := numerator / denominator
    if math.IsNaN(result) || math.IsInf(result, 0) {
        return defaultValue
    }
    return result
}
```

### Files to Modify:
1. ✅ `services/situational_analysis.go` - lines 640, 644, 645
2. ✅ `services/predictions.go` - lines 231, 235, 236  
3. ✅ Add `safeDiv()` helper function to both files

---

## Issue #2: Graceful Degradation for API Failures

### Root Cause:
- **Problem**: Any NHL API failure during prediction causes entire prediction to fail
- **Impact**: Users see "Loading..." indefinitely; no fallback to cached data

### Fix Strategy:
**Phase 1: Add Error Recovery in Ensemble Predictions**
```go
// In services/ensemble_predictions.go, wrap each data fetch:

// Example for goalie data:
comparison, err := goalieService.GetGoalieComparisonWithConfirmed(...)
if err != nil {
    log.Printf("⚠️ Goalie data unavailable, using defaults: %v", err)
    // Use neutral goalie impact instead of failing
    homeFactors.GoalieAdvantage = 0.0
    awayFactors.GoalieAdvantage = 0.0
} else {
    // Use real data
    homeFactors.GoalieAdvantage = comparison.WinProbabilityImpact
    ...
}
```

**Phase 2: Add Cached Prediction Fallback**
```go
// In services/predictions.go:
func (ps *PredictionService) PredictNextGame() (*models.GamePrediction, error) {
    prediction, err := ps.generateNewPrediction()
    if err != nil {
        log.Printf("⚠️ Prediction failed, checking cache: %v", err)
        cachedPrediction := ps.getCachedPrediction(gameID)
        if cachedPrediction != nil {
            log.Printf("✅ Using cached prediction from %v", cachedPrediction.GeneratedAt)
            return cachedPrediction, nil
        }
        // Last resort: return degraded prediction with defaults
        return ps.generateDegradedPrediction(), nil
    }
    ps.cachePrediction(prediction)
    return prediction, nil
}
```

### Files to Modify:
1. ✅ `services/predictions.go` - Add prediction caching
2. ✅ `services/ensemble_predictions.go` - Wrap all API-dependent fetches with error recovery
3. ✅ Add new file `services/prediction_cache.go` for caching logic

---

## Issue #3: Proper Health Check Implementation

### Root Cause:
- **Problem**: `/api/health` endpoint exists but has no validation logic
- **Impact**: Can't detect if services are broken, just if server responds

### Fix Strategy:
```go
// services/health_check_service.go (NEW FILE)
type HealthCheckService struct {
    checks map[string]HealthCheck
}

type HealthCheck struct {
    Name        string
    Status      string // "healthy", "degraded", "unhealthy"
    LastChecked time.Time
    ErrorMsg    string
    ResponseTime time.Duration
}

func (hcs *HealthCheckService) RunHealthChecks() map[string]HealthCheck {
    results := make(map[string]HealthCheck)
    
    // 1. NHL API Check
    results["nhl_api"] = hcs.checkNHLAPI()
    
    // 2. Data Persistence Check  
    results["data_persistence"] = hcs.checkDataPersistence()
    
    // 3. ML Models Check
    results["ml_models"] = hcs.checkMLModels()
    
    // 4. Memory Usage Check
    results["memory"] = hcs.checkMemoryUsage()
    
    // 5. Cache Health
    results["cache"] = hcs.checkCacheHealth()
    
    return results
}

func (hcs *HealthCheckService) checkNHLAPI() HealthCheck {
    start := time.Now()
    _, err := MakeAPICall("https://api-web.nhle.com/v1/standings/now")
    responseTime := time.Since(start)
    
    if err != nil {
        return HealthCheck{
            Name: "NHL API",
            Status: "unhealthy",
            ErrorMsg: fmt.Sprintf("API unreachable: %v", err),
            ResponseTime: responseTime,
        }
    }
    
    if responseTime > 5*time.Second {
        return HealthCheck{
            Name: "NHL API",
            Status: "degraded",
            ErrorMsg: "Slow response time",
            ResponseTime: responseTime,
        }
    }
    
    return HealthCheck{
        Name: "NHL API",
        Status: "healthy",
        ResponseTime: responseTime,
    }
}
```

### API Response Format:
```json
{
  "status": "healthy",  // or "degraded" or "unhealthy"
  "timestamp": "2025-10-09T20:15:00Z",
  "checks": {
    "nhl_api": {
      "status": "healthy",
      "responseTime": "234ms"
    },
    "ml_models": {
      "status": "healthy",
      "loadedModels": 9
    },
    "data_persistence": {
      "status": "healthy",
      "writeable": true
    },
    "memory": {
      "status": "healthy",
      "usage": "245MB / 2GB"
    }
  }
}
```

### Files to Create:
1. ✅ `services/health_check_service.go` (NEW)
2. ✅ `handlers/health.go` - Update to use real health checks
3. ✅ `models/health.go` (NEW) - Health check data structures

---

## Implementation Order:

### Priority 1: NaN Fix (30 minutes)
- ✅ Add `safeDiv()` helper
- ✅ Fix all division operations in factor calculations
- ✅ Test with UTA preseason (0 games played)

### Priority 2: Graceful Degradation (1-2 hours)
- ✅ Add prediction caching
- ✅ Wrap API calls with error recovery
- ✅ Add degraded prediction fallback
- ✅ Test with simulated API failures

### Priority 3: Health Checks (1 hour)
- ✅ Create health check service
- ✅ Implement 5 key health checks
- ✅ Update handlers
- ✅ Test all check scenarios

---

## Testing Checklist:

### NaN Fix:
- [ ] Start server with UTA (0 games played)
- [ ] Verify no "Base: NaN" in logs
- [ ] Verify predictions still generate
- [ ] Check all models produce valid numbers

### Graceful Degradation:
- [ ] Block NHL API (hosts file)
- [ ] Verify predictions still work with cache
- [ ] Verify degraded predictions as last resort
- [ ] Verify user sees predictions, not errors

### Health Checks:
- [ ] Call `/api/health` endpoint
- [ ] Verify all 5 checks run
- [ ] Simulate each failure scenario
- [ ] Verify status codes (200=healthy, 503=unhealthy)

---

## Expected Results:

### Before Fixes:
```
📊 NSH Advanced Score Breakdown:
   Base: NaN, Travel: -0.0, Altitude: 0.0, ...
   🏒 Advanced: xG 0.0, Talent 0.0, ...
```

### After Fixes:
```
📊 NSH Advanced Score Breakdown:
   Base: 50.0, Travel: -0.0, Altitude: 0.0, ...
   🏒 Advanced: xG 0.0, Talent 0.0, ...
```

### Health Check Response:
```bash
$ curl localhost:8080/api/health
{
  "status": "healthy",
  "checks": {
    "nhl_api": { "status": "healthy", "responseTime": "234ms" },
    "ml_models": { "status": "healthy", "loadedModels": 9 },
    ...
  }
}
```

---

## Rollback Plan:

If any fix causes issues:
1. Keep changes in a separate Git branch
2. Each fix is independent - can rollback individually
3. NaN fix is safest (only adds safety checks)
4. Degradation fix needs testing (could mask real errors)
5. Health checks are additive (won't break existing)

---

## Future Enhancements:

After these 3 critical fixes:
- Circuit breaker pattern (fail fast after N failures)
- Cache eviction (LRU for memory management)
- Structured logging (debug/info/warn/error levels)
- Database migration (PostgreSQL for historical data)
- Authentication (API keys for production)

