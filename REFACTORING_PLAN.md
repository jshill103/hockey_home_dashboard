# Service-to-Model Refactoring Plan

## Overview
Move struct definitions from service files to appropriate model files to follow proper architecture patterns.

## Current State
- **21 service files** contain struct definitions
- **100+ structs** that should be in models
- Many structs are tightly coupled to service logic
- Some structs are already properly placed in models

## Refactoring Strategy

### Phase 1: Create New Model Files (Priority: HIGH)

#### 1. `models/model_tracking.go` - Model Performance & Accuracy
**Move from:**
- `services/accuracy_tracking.go`: AccuracyTrackingService
- `services/cross_validation.go`: HistoricalPrediction, ValidationResult, ModelValidationResult, ConfidenceInterval, CalibrationCurve, ConfidenceBin, CrossValidationSettings, ValidationSummary, PredictionFactorSnapshot
- `services/dynamic_weighting.go`: ModelPerformanceTracker, AccuracyRecord, GameContext, ContextualPerformance, RecentPerformanceWindow, WindowStats, ModelWeightCalculator, WeightConstraints, WeightSnapshot, DynamicWeightingSettings

**Estimated Impact:** Medium - Used across prediction system

#### 2. `models/data_quality.go` - Data Quality & Validation  
**Move from:**
- `services/data_quality.go`: DataQualityScore, DataQualityFactor, SourceQuality, DataQualityIssue

**Estimated Impact:** Low - Isolated usage
**Status:** ✅ Some structs already exist in models/accuracy_tracking.go

#### 3. `models/ml_training.go` - ML Model Internal Structures
**Move from:**
- `services/ml_models.go`: NeuralNetworkModel, XGBoostModel, DecisionTree, LSTMModel, RandomForestModel
- `services/elo_rating_model.go`: RatingRecord  
- `services/poisson_regression_model.go`: RateRecord

**Estimated Impact:** Low - Internal to ML training, not exposed

#### 4. `models/uncertainty.go` - Model Uncertainty & Calibration
**Move from:**
- `services/model_uncertainty_service.go`: UncertaintyDataPoint, CalibrationPoint, UncertaintyMetrics, BucketMetrics, UncertaintyQuantification, UncertaintySource

**Estimated Impact:** Medium - Used in prediction confidence calculation

#### 5. `models/market_data.go` - Betting Market Data
**Move from:**
- `services/market_data_service.go`: MarketData, MoneyLineOdds, SpreadOdds, TotalOdds, PropBet, OddsMovement, BettingVolume, SharpMoneyIndicator, PublicBettingData

**Estimated Impact:** Low - Feature not fully implemented
**Status:** ✅ MarketAdjustment already in models/predictions.go

#### 6. `models/line_combinations.go` - Line Combination Analysis
**Move from:**
- `services/line_combination_analysis.go`: LineCombinationAnalysis, LineVsLineMatchup, PredictionImpact

**Estimated Impact:** Medium - Used in player analysis

#### 7. `models/weather_data.go` - Weather Analysis
**Move from:**
- `services/weather_analysis.go`: CityCoordinates, OpenWeatherMapResponse, etc.

**Estimated Impact:** Low - Optional feature
**Status:** ✅ WeatherAnalysis already in models/predictions.go

#### 8. `models/live_system.go` - Live Data & Prediction System
**Move from:**
- `services/live_data_service.go`: UpdateTask, UpdateStatistics, DataCache, CachedData, RateLimiter
- `services/live_prediction_system.go`: SystemStatus
- `services/model_update_scheduler.go`: ModelUpdateTask, UpdateRecord, ModelUpdateStats, LiveGameData

**Estimated Impact:** HIGH - Core system functionality
**Status:** ✅ STARTED - Created models/live_system.go

### Phase 2: Service-Specific Structs (Priority: LOW)

These structs are tightly coupled to service implementation and can stay in services:

#### Keep in Services:
- `*Service` structs (e.g., AccuracyTrackingService, AdvancedAnalyticsService, etc.)
- `*Analyzer` structs (e.g., SituationalAnalyzer)
- Interfaces and abstract types specific to service contracts
- Data source implementations (NHLOfficialSource, ESPNSource, TSNSource)

### Phase 3: Update Imports & References

For each moved struct:
1. Add to appropriate model file
2. Update service file to remove struct definition
3. Add `models.` prefix to all usages in service files
4. Update import statements
5. Run `go build` to catch errors
6. Fix any circular dependency issues

## Implementation Order

### Week 1: Low-Risk Moves
1. ✅ Create `models/live_system.go`
2. Create `models/ml_training.go`
3. Create `models/uncertainty.go`
4. Test build after each move

### Week 2: Medium-Risk Moves
1. Create `models/model_tracking.go`
2. Create `models/line_combinations.go`
3. Update all references
4. Test full system

### Week 3: Cleanup & Polish
1. Update remaining structs
2. Fix any circular dependencies
3. Update documentation
4. Final integration testing

## Risk Mitigation

### Breaking Changes:
- **Circular Dependencies:** models importing services is NOT ALLOWED
- **Interface Violations:** Ensure interfaces stay compatible
- **JSON Serialization:** Verify struct tags remain correct

### Testing Strategy:
1. Build after each file move
2. Run server and verify endpoints
3. Test prediction generation
4. Test live data updates
5. Verify JSON API responses

## Circular Dependency Prevention

**Rule:** Models NEVER import services

If a struct in models needs service functionality:
- Use interfaces in models
- Implement interfaces in services
- Pass dependencies via constructors

## Quick Start

To begin refactoring:

```bash
# 1. Create backup branch
git checkout -b refactor/models-cleanup

# 2. Start with low-risk move
# Create models/ml_training.go
# Move DecisionTree, etc.

# 3. Update imports in services/ml_models.go
# Change: type DecisionTree struct
# To: models.DecisionTree

# 4. Test build
go build

# 5. Test server
go run main.go -team UTA

# 6. Commit if successful
git add .
git commit -m "refactor: move ML model structs to models package"
```

## Status Tracking

- [ ] Phase 1: Create new model files (2/8 complete)
- [ ] Phase 2: Update service references
- [ ] Phase 3: Remove old definitions
- [ ] Phase 4: Integration testing
- [ ] Phase 5: Documentation update

## Notes

- This is a **breaking change** for any external code importing these types
- Consider doing this incrementally to avoid breaking the running system
- Each move should be a separate commit for easy rollback
- Update this document as you progress

## Estimated Time
- Full refactoring: **8-12 hours**
- Per-file refactoring: **30-60 minutes each**
- Critical files only: **3-4 hours**

## Questions/Decisions Needed

1. Should ALL structs move, or only shared/API structs?
2. How to handle service-specific internal structs?
3. Timing - do this all at once or incrementally?
4. Need regression testing before/after?

