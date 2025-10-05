# Phase 4 Implementation - Final Steps

## ✅ **What's Been Completed**

### **1. Complete Data Models** ✅
- `models/goalie.go` - 7 goalie structs (GoalieInfo, GoalieComparison, etc.)
- `models/betting_market.go` - 8 betting market structs (BettingOdds, MarketConsensus, etc.)
- `models/schedule_context.go` - 6 schedule structs (ScheduleContext, ScheduleComparison, etc.)

### **2. Complete Services** ✅
- `services/goalie_intelligence_service.go` - Full goalie analysis (400+ lines)
- `services/betting_market_service.go` - Market integration (500+ lines)

### **3. What's Left**
- Schedule Context Service implementation
- Integration into predictions
- Neural Network feature updates
- Testing

---

## 🔧 **Remaining Implementation Tasks**

### **Task 1: Create Schedule Context Service** (2-3 hours)

Create: `services/schedule_context_service.go`

**Core Functions Needed:**
```go
type ScheduleContextService struct {
    cityCoords  map[string]CityCoordinates
    gameHistory map[string][]GameScheduleEntry
    dataDir     string
    mutex       sync.RWMutex
}

// Main API
func GetScheduleComparison(homeTeam, awayTeam string, gameDate time.Time) (*models.ScheduleComparison, error)

// Helper functions
func calculateTravelDistance(from, to string) float64
func detectBackToBack(team string, gameDate time.Time) bool
func calculateRestDays(team string, gameDate time.Time) int
func identifyTrapGame(context *models.ScheduleContext) bool
func calculatePlayoffImportance(team string, gameDate time.Time) float64
```

**City Coordinates Map:**
```go
var nhlCityCoordinates = map[string]CityCoordinates{
    "UTA": {City: "Salt Lake City", Latitude: 40.7608, Longitude: -111.8910, TimeZone: "MST"},
    "VGK": {City: "Las Vegas", Latitude: 36.1699, Longitude: -115.1398, TimeZone: "PST"},
    "COL": {City: "Denver", Latitude: 39.7392, Longitude: -104.9903, TimeZone: "MST"},
    "ARI": {City: "Tempe", Latitude: 33.4484, Longitude: -111.9261, TimeZone: "MST"},
    // ... add all 32 teams
}
```

**Distance Calculation (Haversine Formula):**
```go
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
    const earthRadiusMiles = 3959.0
    
    lat1Rad := lat1 * math.Pi / 180
    lat2Rad := lat2 * math.Pi / 180
    deltaLat := (lat2 - lat1) * math.Pi / 180
    deltaLon := (lon2 - lon1) * math.Pi / 180
    
    a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
         math.Cos(lat1Rad)*math.Cos(lat2Rad)*
         math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
    
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    
    return earthRadiusMiles * c
}
```

---

### **Task 2: Update PredictionFactors Model** (30 min)

Update: `models/predictions.go`

```go
type PredictionFactors struct {
    // ... existing fields ...
    
    // PHASE 4: Goalie Intelligence
    GoalieAdvantage      float64 `json:"goalieAdvantage"`      // -0.15 to +0.15
    GoalieSavePctDiff    float64 `json:"goalieSavePctDiff"`    // Save % difference
    GoalieRecentFormDiff float64 `json:"goalieRecentFormDiff"` // Recent form difference
    GoalieFatigueDiff    float64 `json:"goalieFatigueDiff"`    // Workload difference
    
    // PHASE 4: Betting Markets
    MarketConsensus      float64 `json:"marketConsensus"`      // Market win %
    MarketLineMovement   float64 `json:"marketLineMovement"`   // Line movement indicator
    SharpMoneyIndicator  float64 `json:"sharpMoneyIndicator"`  // Sharp money signal
    MarketConfidence     float64 `json:"marketConfidence"`     // Market confidence
    
    // PHASE 4: Schedule Context
    TravelDistance       float64 `json:"travelDistance"`       // Miles traveled
    BackToBackIndicator  float64 `json:"backToBackIndicator"`  // 1.0 if B2B, 0.0 if not
    ScheduleDensity      float64 `json:"scheduleDensity"`      // Games in last 7 days
    TrapGameFactor       float64 `json:"trapGameFactor"`       // Trap game likelihood
    PlayoffImportance    float64 `json:"playoffImportance"`    // Playoff stakes
    RestAdvantage        float64 `json:"restAdvantage"`        // Rest days advantage
}
```

---

### **Task 3: Update Neural Network Features** (1 hour)

Update: `services/ml_models.go`

**Change Architecture:**
```go
// OLD
layers := []int{50, 32, 16, 3}

// NEW
layers := []int{65, 32, 16, 3} // 15 new features
```

**Add New Feature Extraction:**
```go
func (nn *NeuralNetworkModel) extractFeatures(home, away *models.PredictionFactors) []float64 {
    features := make([]float64, 65) // Increased from 50
    
    // Existing features 0-49 (keep as is)
    // ... existing code ...
    
    // PHASE 4: New features 50-64
    
    // Goalie features (50-53)
    features[50] = home.GoalieAdvantage
    features[51] = away.GoalieAdvantage
    features[52] = home.GoalieSavePctDiff
    features[53] = home.GoalieRecentFormDiff
    
    // Market features (54-57)
    features[54] = home.MarketConsensus
    features[55] = away.MarketConsensus
    features[56] = home.SharpMoneyIndicator
    features[57] = home.MarketLineMovement
    
    // Schedule features (58-64)
    features[58] = home.TravelDistance / 3000.0  // Normalize
    features[59] = away.TravelDistance / 3000.0
    features[60] = home.BackToBackIndicator
    features[61] = away.BackToBackIndicator
    features[62] = home.ScheduleDensity / 7.0
    features[63] = away.ScheduleDensity / 7.0
    features[64] = home.RestAdvantage / 5.0
    
    return features
}
```

---

### **Task 4: Integrate Phase 4 into Predictions** (2 hours)

Update: `services/ensemble_predictions.go`

**Add to PredictGame function:**
```go
func (eps *EnsemblePredictionService) PredictGame(homeTeam, awayTeam string, gameDate time.Time, homeFactors, awayFactors *models.PredictionFactors) (*models.Prediction, error) {
    
    // ... existing code ...
    
    // ============================================================================
    // PHASE 4: ENHANCED DATA ENRICHMENT
    // ============================================================================
    
    // 1. Goalie Intelligence
    goalieService := GetGoalieService()
    if goalieService != nil {
        goalieComparison, err := goalieService.GetGoalieComparison(homeTeam, awayTeam, gameDate)
        if err == nil {
            // Apply goalie impact to factors
            impact := goalieComparison.WinProbabilityImpact
            homeFactors.GoalieAdvantage = impact
            awayFactors.GoalieAdvantage = -impact
            
            homeFactors.GoalieSavePctDiff = goalieComparison.SeasonPerformance
            homeFactors.GoalieRecentFormDiff = goalieComparison.RecentForm
            homeFactors.GoalieFatigueDiff = goalieComparison.WorkloadFatigue
            
            log.Printf("🥅 Goalie Impact: %s advantage (%.1f%% swing)", 
                goalieComparison.OverallAdvantage, 
                math.Abs(impact)*100)
        }
    }
    
    // 2. Betting Market Intelligence
    marketService := GetBettingMarketService()
    if marketService != nil && marketService.isEnabled {
        marketAdjustment, err := marketService.GetMarketAdjustment(homeTeam, awayTeam, gameDate)
        if err == nil {
            homeFactors.MarketConsensus = marketAdjustment.MarketPrediction
            awayFactors.MarketConsensus = 1.0 - marketAdjustment.MarketPrediction
            homeFactors.MarketConfidence = marketAdjustment.MarketEfficiency
            
            log.Printf("💰 Market Says: %.1f%% home win (confidence: %.1f%%)",
                marketAdjustment.MarketPrediction*100,
                marketAdjustment.MarketEfficiency*100)
        }
    }
    
    // 3. Schedule Context Analysis
    scheduleService := GetScheduleContextService()
    if scheduleService != nil {
        scheduleComp, err := scheduleService.GetScheduleComparison(homeTeam, awayTeam, gameDate)
        if err == nil {
            homeContext := scheduleComp.HomeContext
            awayContext := scheduleComp.AwayContext
            
            homeFactors.TravelDistance = homeContext.TravelDistance
            awayFactors.TravelDistance = awayContext.TravelDistance
            
            homeFactors.BackToBackIndicator = 0.0
            if homeContext.IsBackToBack {
                homeFactors.BackToBackIndicator = 1.0
            }
            awayFactors.BackToBackIndicator = 0.0
            if awayContext.IsBackToBack {
                awayFactors.BackToBackIndicator = 1.0
            }
            
            homeFactors.ScheduleDensity = float64(homeContext.GamesInLast7Days)
            awayFactors.ScheduleDensity = float64(awayContext.GamesInLast7Days)
            
            homeFactors.TrapGameFactor = homeContext.TrapGameScore
            awayFactors.TrapGameFactor = awayContext.TrapGameScore
            
            homeFactors.PlayoffImportance = homeContext.PlayoffImportance
            awayFactors.PlayoffImportance = awayContext.PlayoffImportance
            
            homeFactors.RestAdvantage = float64(homeContext.RestAdvantage)
            awayFactors.RestAdvantage = float64(-homeContext.RestAdvantage)
            
            log.Printf("📅 Schedule Impact: %s advantage (%.1f%% swing)",
                scheduleComp.OverallAdvantage,
                math.Abs(scheduleComp.TotalImpact)*100)
        }
    }
    
    // ... continue with existing prediction logic ...
}
```

---

### **Task 5: Initialize Phase 4 Services in Main** (30 min)

Update: `main.go`

```go
// After existing service initialization...

// Initialize Phase 4 Services
fmt.Println("Initializing Phase 4 Enhanced Services...")

// Goalie Intelligence
if err := services.InitializeGoalieService(); err != nil {
    fmt.Printf("⚠️ Warning: Failed to initialize goalie service: %v\n", err)
} else {
    fmt.Printf("✅ Goalie Intelligence Service initialized\n")
}

// Betting Markets (optional - requires API key)
if err := services.InitializeBettingMarketService(); err != nil {
    fmt.Printf("⚠️ Warning: Failed to initialize betting market service: %v\n", err)
} else {
    fmt.Printf("✅ Betting Market Service initialized\n")
}

// Schedule Context
if err := services.InitializeScheduleContextService(); err != nil {
    fmt.Printf("⚠️ Warning: Failed to initialize schedule context service: %v\n", err)
} else {
    fmt.Printf("✅ Schedule Context Service initialized\n")
}

fmt.Println("🚀 Phase 4 services ready!")
```

---

### **Task 6: Testing & Validation** (2-3 hours)

**Test Plan:**

1. **Unit Tests for Each Service:**
```bash
# Test goalie comparison
go test ./services -run TestGoalieComparison

# Test betting market parsing
go test ./services -run TestBettingMarketAPI

# Test schedule context
go test ./services -run TestScheduleContext
```

2. **Integration Test:**
```bash
# Make a prediction with all Phase 4 features
curl "http://localhost:8080/api/prediction?homeTeam=UTA&awayTeam=VGK"

# Check logs for Phase 4 indicators:
# - 🥅 Goalie Impact
# - 💰 Market Says
# - 📅 Schedule Impact
```

3. **Accuracy Test on Historical Games:**
```go
// Run on test set (last 100 games)
// Compare accuracy before/after Phase 4
// Target: +6-9% improvement
```

---

## 📋 **Quick Setup Checklist**

### **Environment Variables:**
```bash
# Optional: Betting market data (free API)
export ODDS_API_KEY="your_key_from_the-odds-api.com"

# Restart server
./web_server --team UTA
```

### **Expected Logs on Startup:**
```
🥅 Goalie Intelligence Service initialized
💰 Betting Market Service initialized (or disabled if no API key)
📅 Schedule Context Service initialized
🚀 Phase 4 services ready!
```

### **Expected Logs on Prediction:**
```
Making prediction for UTA vs VGK...
🥅 Goalie Impact: home advantage (4.2% swing)
💰 Market Says: 68.5% home win (confidence: 82.1%)
📅 Schedule Impact: away advantage (2.1% swing)
⚖️ Final Prediction: UTA 71.3% (ensemble of 6 models)
```

---

## 📊 **Expected Accuracy Improvement**

### **Test Methodology:**
1. Run predictions on last 100 completed games
2. Compare to actual results
3. Calculate accuracy before/after Phase 4

### **Expected Results:**

| Metric | Before Phase 4 | After Phase 4 | Improvement |
|--------|----------------|---------------|-------------|
| **Overall Accuracy** | 75-85% | 81-94% | +6-9% |
| **Close Games (<1 goal)** | 60-70% | 70-80% | +10% |
| **Upset Detection** | 30-40% | 45-55% | +15% |
| **Goalie-Dependent Games** | 65-75% | 78-88% | +13% |

### **Where Improvements Come From:**

1. **Goalie Intelligence (+3-4%)**
   - Better predictions when goalie quality differs
   - Starter vs backup detection
   - Hot/cold goalie streaks

2. **Betting Markets (+2-3%)**
   - Incorporate "smart money" insights
   - Detect insider information
   - Market efficiency helps calibration

3. **Schedule Context (+1-2%)**
   - Back-to-back games more predictable
   - Travel fatigue accurately modeled
   - Trap game detection

---

## 🎯 **Success Criteria**

### **Phase 4 is successful if:**

1. ✅ All 3 services initialize without errors
2. ✅ Prediction logs show Phase 4 factors
3. ✅ Neural Network uses 65 features (not 50)
4. ✅ Accuracy improves by at least +5% on test set
5. ✅ Goalie-dependent games improve by +10%
6. ✅ No performance degradation (<1s per prediction)

---

## 🚀 **Next Steps After Phase 4**

Once Phase 4 is complete and validated:

### **Phase 5: Model Optimization**
- Neural Network architecture search
- Ensemble weight optimization
- XGBoost integration
- LSTM for sequences

### **Phase 6: Advanced Features**
- Player impact (simplified)
- Matchup database
- Coaching factors
- Weather integration improvements

### **Phase 7: User Experience**
- Prediction explanations
- Confidence visualization
- What-if scenarios
- Live game updates

---

## 💡 **Quick Start Guide**

### **To Complete Phase 4 Today:**

**Hour 1:** Create schedule context service
- Copy structure from goalie/market services
- Implement travel distance calculation
- Implement back-to-back detection

**Hour 2:** Integration
- Update PredictionFactors model
- Update Neural Network features
- Wire services into ensemble

**Hour 3:** Testing
- Build and run server
- Test on a few predictions
- Check logs for Phase 4 indicators

**Hour 4:** Validation
- Run on historical test set
- Calculate accuracy improvement
- Document results

---

## 📝 **File Checklist**

### **Created (Complete):** ✅
- [x] `models/goalie.go`
- [x] `models/betting_market.go`
- [x] `models/schedule_context.go`
- [x] `services/goalie_intelligence_service.go`
- [x] `services/betting_market_service.go`

### **Need to Create:** ⬜
- [ ] `services/schedule_context_service.go` (main remaining task)

### **Need to Update:** ⬜
- [ ] `models/predictions.go` (add Phase 4 fields)
- [ ] `services/ml_models.go` (update features 50→65)
- [ ] `services/ensemble_predictions.go` (integrate Phase 4)
- [ ] `main.go` (initialize Phase 4 services)

---

**Phase 4 is 60% complete! The hardest parts (models and complex services) are done. The remaining work is straightforward integration.** 🚀


