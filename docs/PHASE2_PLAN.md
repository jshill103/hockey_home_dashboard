# Phase 2: Enhanced Data Quality

## 🎯 Objectives

Improve prediction accuracy by implementing better historical matchup tracking and contextual analysis.

**Expected Accuracy Gain**: +8-12%  
**Timeline**: 3 weeks  
**Current Status**: Phase 1 Complete ✅ → Starting Phase 2

## 📋 Components

### 2.1 Head-to-Head Matchup Database ⭐ HIGH IMPACT
**Goal**: Track historical performance between specific team pairs

**Implementation**:
- Store recent H2H results (last 10 games between teams)
- Calculate win/loss differential
- Track scoring patterns (average goals for/against)
- Consider home/away splits
- Weight recent games more heavily (exponential decay)

**Data Model**:
```go
type HeadToHeadRecord struct {
    HomeTeam      string
    AwayTeam      string
    TotalGames    int
    HomeWins      int
    AwayWins      int
    RecentGames   []GameResult  // Last 10 H2H games
    AvgHomeGoals  float64
    AvgAwayGoals  float64
    LastMeetingDate time.Time
    WeightedAdvantage float64  // Composite score favoring recent performance
}
```

**Files to Create**:
- `services/head_to_head_service.go`
- `models/head_to_head.go`
- `handlers/head_to_head_handler.go`

---

### 2.2 Goalie Matchup History ⭐ HIGH IMPACT
**Goal**: Track how specific goalies perform against specific teams

**Implementation**:
- Store goalie performance vs each team
- Track save percentage by team
- Identify "nemesis teams" (unusually good/bad matchups)
- Weight recent starts more heavily
- Minimum games threshold for statistical significance

**Data Model**:
```go
type GoalieMatchupHistory struct {
    GoalieID      int
    GoalieName    string
    OpponentTeam  string
    GamesPlayed   int
    Wins          int
    Losses        int
    SavePct       float64
    GAA           float64
    RecentForm    []GameResult  // Last 5 starts vs this team
    PerformanceRating float64    // Normalized rating vs this opponent
    LastFaceOff   time.Time
}
```

**Integration**:
- Enhance existing `GoalieIntelligenceService`
- Add matchup-specific adjustments to goalie advantage calculations

**Files to Modify**:
- `services/goalie_intelligence_service.go`
- `models/goalie.go`

---

### 2.3 Lineup-Based Predictions ⭐ MEDIUM-HIGH IMPACT
**Goal**: Adjust predictions based on confirmed lineups

**Implementation**:
- Track performance with specific line combinations
- Identify "chemistry" between linemates
- Detect lineup changes and assess impact
- Weight predictions when star players are out
- Track special teams unit effectiveness

**Data Model**:
```go
type LineupImpact struct {
    TeamCode      string
    GameDate      time.Time
    LineChanges   int              // Number of changes from previous game
    StarPlayersOut []string        // Key players missing
    NewPairings   []PlayerPair     // New line combinations
    ImpactScore   float64          // Estimated impact on performance (-1 to +1)
    SpecialTeamsChange float64     // Change in PP/PK effectiveness
}

type PlayerPair struct {
    Player1       string
    Player2       string
    GamesPlayed   int
    PointsPerGame float64
    PlusMinusAvg  float64
}
```

**Integration**:
- Enhance `PreGameLineupService`
- Add lineup stability factor to predictions

**Files to Modify**:
- `services/pre_game_lineup_service.go`
- `models/lineup.go`

---

### 2.4 Rest Days Impact Analysis ⭐ MEDIUM IMPACT
**Goal**: Better quantify the impact of rest/fatigue

**Implementation**:
- Track team performance by rest days (0, 1, 2, 3+)
- Identify teams that perform well/poorly on back-to-backs
- Account for travel distance + rest days
- Consider opponent's rest advantage/disadvantage
- Track "tired legs" indicators (shot volume decline, etc.)

**Data Model**:
```go
type RestImpactAnalysis struct {
    TeamCode           string
    Season             string
    
    // Performance by rest days
    BackToBackRecord   TeamRecord     // 0 days rest
    OneDayRestRecord   TeamRecord     // 1 day rest
    TwoDayRestRecord   TeamRecord     // 2 days rest
    ThreePlusRestRecord TeamRecord    // 3+ days rest
    
    // Comparative metrics
    B2BPenalty         float64        // Win% drop on B2B
    OptimalRest        int            // Days of rest for peak performance
    FatigueResistance  float64        // How well team handles fatigue (0-1)
    
    // Advanced metrics
    ShotsDeclineB2B    float64        // Shot volume drop on B2B
    SavePctDeclineB2B  float64        // Goalie performance drop on B2B
}

type TeamRecord struct {
    Games   int
    Wins    int
    Losses  int
    WinPct  float64
    AvgGoalsFor float64
    AvgGoalsAgainst float64
}
```

**Files to Create**:
- `services/rest_impact_service.go`
- `models/rest_analysis.go`

**Files to Modify**:
- `services/game_predictor.go` (add rest impact to PredictionContext)

---

### 2.5 Enhanced Feature Engineering
**Goal**: Create new composite features from Phase 2 data

**New Features to Add to PredictionFactors**:
```go
// Add to models/predictions.go
type PredictionFactors struct {
    // ... existing fields ...
    
    // PHASE 2: Enhanced Data Quality
    HeadToHeadAdvantage    float64 `json:"headToHeadAdvantage"`    // -0.30 to +0.30
    H2HRecentForm          float64 `json:"h2hRecentForm"`          // Last 3 H2H games
    GoalieVsTeamRating     float64 `json:"goalieVsTeamRating"`     // Goalie matchup history
    LineupStabilityFactor  float64 `json:"lineupStabilityFactor"`  // 0.0 to 1.0
    RestAdvantageDetailed  float64 `json:"restAdvantageDetailed"`  // Enhanced rest analysis
    OpponentFatigue        float64 `json:"opponentFatigue"`        // Opponent's fatigue level
}
```

---

## 📊 Implementation Order

### Week 1: Foundation
1. ✅ Create data models (`models/head_to_head.go`, `models/rest_analysis.go`)
2. ✅ Implement Head-to-Head Service with persistence
3. ✅ Build H2H data collection from historical games
4. ✅ Create H2H API endpoints

### Week 2: Matchup Intelligence
1. ✅ Enhance Goalie Intelligence with matchup history
2. ✅ Implement Rest Impact Analysis service
3. ✅ Build lineup impact analyzer
4. ✅ Create API endpoints for new services

### Week 3: Integration & Testing
1. ✅ Add Phase 2 features to PredictionFactors
2. ✅ Update Neural Network architecture (176 → 182 features)
3. ✅ Integrate services into prediction pipeline
4. ✅ Build Phase 2 analytics dashboard
5. ✅ Test and validate

---

## 🎯 Success Metrics

### Data Quality Indicators
- H2H database populated with 500+ team pairs
- Goalie matchup data for 60+ goalies
- Rest impact profiles for all 32 teams
- Lineup stability tracking active

### Accuracy Improvements
- **Target**: +8-12% overall accuracy
- **Baseline**: Current Phase 1 accuracy (measure after 20 games)
- **Validation**: Compare predictions with/without Phase 2 features

### API Performance
- H2H lookup: < 10ms
- Goalie matchup query: < 15ms
- Rest analysis: < 5ms
- All endpoints respond within 50ms

---

## 🔧 Technical Details

### Data Persistence
- **Location**: `/app/data/head_to_head/`, `/app/data/rest_analysis/`
- **Format**: JSON files per team/matchup
- **Updates**: After each completed game
- **Backfill**: Historical data from existing game records

### Integration Points
1. **Prediction Pipeline**: `services/ensemble_predictions.go`
   - Call H2H service before prediction
   - Enrich with goalie matchup data
   - Apply rest impact adjustments

2. **Daily Predictions**: `services/daily_prediction_service.go`
   - Pre-cache H2H data for scheduled games
   - Update lineup impact as lineups confirmed

3. **Model Training**: `services/ml_models.go`
   - Expand feature array to accommodate new features
   - Retrain with Phase 2 data

### API Endpoints
```
GET  /api/phase2/dashboard              - Phase 2 overview
GET  /api/head-to-head/:home/:away      - H2H matchup details
GET  /api/goalie-matchup/:id/:opponent  - Goalie vs team history
GET  /api/rest-impact/:team             - Team's rest performance
GET  /api/lineup-impact/:team/:date     - Lineup change impact
```

---

## 🚀 Deployment Strategy

### Phase 2.1 (Week 1)
- Deploy H2H service
- Begin data collection
- No prediction changes yet (monitoring only)

### Phase 2.2 (Week 2)
- Deploy rest impact and goalie matchup
- Begin feature integration (soft rollout)
- A/B test with/without Phase 2 features

### Phase 2.3 (Week 3)
- Full integration into predictions
- Update model architecture
- Retrain with Phase 2 features
- Deploy to production

---

## 📈 Expected Results

### Immediate (After Deployment)
- New data collection starts
- API endpoints available
- Historical data backfilled

### Short Term (10-20 Games)
- H2H patterns emerge
- Goalie matchups identified
- Rest impact profiles complete

### Medium Term (30-50 Games)
- **+8-12% accuracy improvement**
- Clear identification of:
  - Favorable/unfavorable matchups
  - Goalie "nemesis" teams
  - Rest-sensitive teams
  - Lineup chemistry effects

### Long Term (100+ Games)
- Refined matchup predictions
- Lineup chemistry database mature
- Optimal rest patterns identified
- Significant competitive advantage

---

## 🎉 Phase 2 Success Criteria

- ✅ All services implemented and tested
- ✅ Data collection active for 30+ games
- ✅ API endpoints functional
- ✅ Features integrated into predictions
- ✅ Measurable accuracy improvement (+8-12%)
- ✅ No performance degradation (< 100ms prediction time)
- ✅ Comprehensive documentation
- ✅ Successfully deployed to cluster

---

**Phase 1 Status**: ✅ Complete (+7-11% accuracy)  
**Phase 2 Status**: 🚀 Starting Implementation  
**Combined Expected Gain**: +15-23% accuracy improvement

Let's build Phase 2! 🏒

