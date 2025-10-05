# Game Results Collection Service - Implementation Plan

## 🎯 **Objective**

Automatically detect completed games, fetch comprehensive game data from the NHL API, store it persistently, and feed it to ML models for continuous learning.

---

## 📋 **Overview**

### What We Need:
1. **Detect** when games finish
2. **Fetch** comprehensive game data from NHL API
3. **Store** game results persistently
4. **Feed** data to Elo & Poisson models for learning
5. **Track** what games have been processed

---

## 🏗️ **Architecture**

```
┌─────────────────────────────────────────────────────────┐
│              Game Results Service                        │
│                                                          │
│  ┌──────────────────────────────────────────┐          │
│  │  1. Game Monitor (runs every 5 minutes)  │          │
│  │     - Check team schedule                 │          │
│  │     - Detect completed games              │          │
│  │     - Filter out already-processed games  │          │
│  └──────────────────────────────────────────┘          │
│                      ↓                                   │
│  ┌──────────────────────────────────────────┐          │
│  │  2. Data Fetcher                         │          │
│  │     - GET game details (boxscore)        │          │
│  │     - GET team stats                     │          │
│  │     - GET player stats                   │          │
│  │     - Combine into GameResult struct     │          │
│  └──────────────────────────────────────────┘          │
│                      ↓                                   │
│  ┌──────────────────────────────────────────┐          │
│  │  3. Data Storage                         │          │
│  │     - Save to: data/results/YYYY-MM.json │          │
│  │     - Track processed games              │          │
│  │     - Append-only design                 │          │
│  └──────────────────────────────────────────┘          │
│                      ↓                                   │
│  ┌──────────────────────────────────────────┐          │
│  │  4. Model Updater                        │          │
│  │     - Feed to Elo model                  │          │
│  │     - Feed to Poisson model              │          │
│  │     - Feed to Accuracy tracker           │          │
│  │     - Auto-save models                   │          │
│  └──────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────┘
```

---

## 📊 **Data Structures**

### 1. Game Result Storage (`models/game_result.go`)

```go
// CompletedGame represents a finished game with all relevant data
type CompletedGame struct {
    // Game Identification
    GameID       int       `json:"gameId"`
    GameDate     time.Time `json:"gameDate"`
    Season       int       `json:"season"`
    GameType     int       `json:"gameType"` // 1=preseason, 2=regular, 3=playoffs
    ProcessedAt  time.Time `json:"processedAt"`
    
    // Teams
    HomeTeam     TeamGameResult `json:"homeTeam"`
    AwayTeam     TeamGameResult `json:"awayTeam"`
    
    // Game Outcome
    Winner       string `json:"winner"` // Team code
    WinType      string `json:"winType"` // "REG", "OT", "SO"
    
    // Venue & Conditions
    Venue        string  `json:"venue"`
    Attendance   int     `json:"attendance"`
    
    // API Source
    DataSource   string  `json:"dataSource"` // "NHLE_API_v1"
    DataVersion  string  `json:"dataVersion"` // "1.0"
}

// TeamGameResult represents one team's performance in a game
type TeamGameResult struct {
    TeamCode     string `json:"teamCode"`
    TeamName     string `json:"teamName"`
    Score        int    `json:"score"`
    
    // Shooting Stats
    Shots        int     `json:"shots"`
    ShotPct      float64 `json:"shotPct"`
    
    // Special Teams
    PowerPlayGoals      int     `json:"ppGoals"`
    PowerPlayOpps       int     `json:"ppOpps"`
    PowerPlayPct        float64 `json:"ppPct"`
    PenaltyKillSaves    int     `json:"pkSaves"`
    PenaltyKillOpps     int     `json:"pkOpps"`
    PenaltyKillPct      float64 `json:"pkPct"`
    
    // Discipline
    PenaltyMinutes int `json:"pim"`
    
    // Possession
    FaceoffWins    int     `json:"faceoffWins"`
    FaceoffTotal   int     `json:"faceoffTotal"`
    FaceoffPct     float64 `json:"faceoffPct"`
    
    // Physical
    Hits           int `json:"hits"`
    Blocks         int `json:"blocks"`
    Giveaways      int `json:"giveaways"`
    Takeaways      int `json:"takeaways"`
    
    // Goalie Performance
    GoalieName     string  `json:"goalieName"`
    Saves          int     `json:"saves"`
    SavePct        float64 `json:"savePct"`
    GoalsAgainst   int     `json:"goalsAgainst"`
    
    // Top Performers (optional, if we want to track)
    TopScorer      string `json:"topScorer,omitempty"`
    TopScorerPts   int    `json:"topScorerPts,omitempty"`
}

// ProcessedGamesIndex tracks which games have been processed
type ProcessedGamesIndex struct {
    LastUpdated    time.Time       `json:"lastUpdated"`
    ProcessedGames map[int]bool    `json:"processedGames"` // gameID -> processed
    TotalProcessed int             `json:"totalProcessed"`
    Version        string          `json:"version"`
}
```

### 2. Service Configuration

```go
// GameResultsService configuration
type GameResultsService struct {
    teamCode           string
    dataDir            string
    checkInterval      time.Duration
    processedGames     map[int]bool
    eloModel           *EloRatingModel
    poissonModel       *PoissonRegressionModel
    accuracyTracker    *AccuracyTrackingService
    mutex              sync.RWMutex
    stopChan           chan bool
    isRunning          bool
}
```

---

## 🔌 **NHL API Endpoints**

### 1. Get Game Details (Boxscore)
```
GET https://api-web.nhle.com/v1/gamecenter/{gameId}/boxscore

Response includes:
- Final score
- Team stats (shots, PP, PK, faceoffs, hits, blocks)
- Player stats (goals, assists, saves)
- Goalie stats
- Game outcome (REG/OT/SO)
```

### 2. Get Team Schedule (Already using)
```
GET https://api-web.nhle.com/v1/club-schedule/{teamCode}/week/now

Use to find:
- Recently completed games (gameState: "FINAL" or "OFF")
- Game IDs to fetch details for
```

### 3. Get Game Landing (Alternative)
```
GET https://api-web.nhle.com/v1/gamecenter/{gameId}/landing

More detailed game information if needed
```

---

## 📁 **File Structure**

```
data/
├── accuracy/
│   └── accuracy_data.json              # Existing
├── models/
│   ├── elo_ratings.json                # Existing
│   └── poisson_rates.json              # Existing
└── results/                             # ✨ NEW
    ├── processed_games.json            # Index of processed games
    ├── 2025-10.json                    # October 2025 games
    ├── 2025-11.json                    # November 2025 games
    └── ...                             # One file per month
```

**Rationale for monthly files:**
- Easier to manage than one large file
- Natural organization by time period
- Easier to backup/archive
- Can delete old months if needed

---

## 🔄 **Service Workflow**

### Initialization:
```go
1. Load processed games index
2. Load last 30 days of game results
3. Start background monitor goroutine
4. Log: "📊 Game Results Service started (X games in history)"
```

### Every 5 Minutes (Monitor Loop):
```go
1. Fetch team schedule for current week
2. Filter games where gameState == "FINAL" or "OFF"
3. Filter out already-processed games
4. For each new completed game:
   a. Fetch boxscore data
   b. Parse into CompletedGame struct
   c. Save to monthly file
   d. Mark as processed
   e. Feed to models
   f. Log: "✅ Processed game: {TeamA} vs {TeamB} ({score})"
5. If any games processed:
   - Save updated index
   - Trigger model saves
```

### Model Integration:
```go
// After processing each game
func (grs *GameResultsService) feedToModels(game *CompletedGame) {
    // Create game result for model updates
    gameResult := &GameResult{
        GameID:       game.GameID,
        HomeTeam:     game.HomeTeam.TeamCode,
        AwayTeam:     game.AwayTeam.TeamCode,
        HomeGoals:    game.HomeTeam.Score,
        AwayGoals:    game.AwayTeam.Score,
        GameDate:     game.GameDate,
        WinType:      game.WinType,
        // ... other fields
    }
    
    // Update Elo ratings
    grs.eloModel.processGameResult(gameResult)
    
    // Update Poisson rates
    grs.poissonModel.processGameResult(gameResult)
    
    // Track accuracy if we had a prediction
    grs.accuracyTracker.UpdateGameResult(
        game.HomeTeam.TeamCode,
        game.AwayTeam.TeamCode,
        game.GameDate,
        &ActualGameFactors{
            HomeGoals: game.HomeTeam.Score,
            AwayGoals: game.AwayTeam.Score,
            // ... other actual factors
        },
    )
}
```

---

## 🎯 **Implementation Steps**

### Phase 1: Core Infrastructure (High Priority)
- [ ] Create `models/game_result.go` with data structures
- [ ] Create `services/game_results_service.go`
- [ ] Implement processed games index (load/save)
- [ ] Implement game detection (check schedule)
- [ ] Implement boxscore fetching

### Phase 2: Data Storage (High Priority)
- [ ] Implement monthly file storage
- [ ] Implement append-to-file logic
- [ ] Implement duplicate detection
- [ ] Add data validation

### Phase 3: Model Integration (Critical)
- [ ] Create GameResult → LiveGameData converter
- [ ] Integrate with Elo model's processGameResult
- [ ] Integrate with Poisson model's processGameResult
- [ ] Integrate with AccuracyTrackingService
- [ ] Test model learning with real data

### Phase 4: Service Lifecycle (High Priority)
- [ ] Implement Start() method
- [ ] Implement Stop() method
- [ ] Implement background monitor goroutine
- [ ] Add to main.go initialization
- [ ] Add graceful shutdown

### Phase 5: Robustness (Medium Priority)
- [ ] Error handling & retry logic
- [ ] Rate limiting for API calls
- [ ] Logging & debugging
- [ ] Handle edge cases (postponed games, etc.)

### Phase 6: Enhancements (Low Priority)
- [ ] Backfill historical games
- [ ] Export/import functionality
- [ ] Statistics dashboard
- [ ] Manual game entry (for testing)

---

## 🔍 **Key Implementation Details**

### 1. Detecting Completed Games

```go
func (grs *GameResultsService) checkForCompletedGames() error {
    // Get team schedule
    schedule, err := GetTeamSchedule(grs.teamCode, "week", "now")
    if err != nil {
        return err
    }
    
    var newGames []int
    for _, game := range schedule.Games {
        // Check if game is completed and not yet processed
        if (game.GameState == "FINAL" || game.GameState == "OFF") &&
           !grs.isProcessed(game.ID) {
            newGames = append(newGames, game.ID)
        }
    }
    
    log.Printf("📊 Found %d new completed games", len(newGames))
    
    // Process each new game
    for _, gameID := range newGames {
        if err := grs.processGame(gameID); err != nil {
            log.Printf("❌ Failed to process game %d: %v", gameID, err)
        }
    }
    
    return nil
}
```

### 2. Fetching Boxscore Data

```go
func (grs *GameResultsService) fetchGameData(gameID int) (*CompletedGame, error) {
    url := fmt.Sprintf("https://api-web.nhle.com/v1/gamecenter/%d/boxscore", gameID)
    
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var boxscore BoxscoreResponse
    if err := json.NewDecoder(resp.Body).Decode(&boxscore); err != nil {
        return nil, err
    }
    
    // Transform API response to our CompletedGame struct
    game := grs.transformBoxscore(&boxscore)
    
    return game, nil
}
```

### 3. Persistent Storage

```go
func (grs *GameResultsService) saveGame(game *CompletedGame) error {
    // Determine monthly file
    monthKey := game.GameDate.Format("2006-01")
    filePath := filepath.Join(grs.dataDir, "results", monthKey+".json")
    
    // Load existing games for this month
    var games []CompletedGame
    if data, err := ioutil.ReadFile(filePath); err == nil {
        json.Unmarshal(data, &games)
    }
    
    // Append new game
    games = append(games, *game)
    
    // Save back
    data, _ := json.MarshalIndent(games, "", "  ")
    os.MkdirAll(filepath.Dir(filePath), 0755)
    return ioutil.WriteFile(filePath, data, 0644)
}
```

### 4. Integration with Main

```go
// In main.go
func main() {
    // ... existing initialization ...
    
    // Initialize game results service
    gameResultsService := services.NewGameResultsService(
        teamConfig.Code,
        eloModel,
        poissonModel,
        accuracyTracker,
    )
    
    // Start monitoring for completed games
    gameResultsService.Start()
    
    // ... rest of server setup ...
    
    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan
    
    gameResultsService.Stop()
    // ... other cleanup ...
}
```

---

## 📈 **Expected Benefits**

### Automatic Learning:
```
Before: Manual model updates, no learning
After:  Automatic learning from every completed game

Timeline:
- Day 1:  Service starts, processes recent games
- Week 1: 10-15 games processed, models adjusting
- Month 1: 50-60 games processed, patterns emerging
- Season: 82+ games, models highly tuned
```

### Data Accumulation:
```
Month 1:  ~20 games   (~100 KB)
Month 2:  ~40 games   (~200 KB)
Season:   ~100 games  (~500 KB)
3 Years:  ~300 games  (~1.5 MB)
```

### Model Improvements:
```
Without auto-collection: Models static, 60% accuracy
With auto-collection:    Models improve to 75%+ accuracy
```

---

## 🎛️ **Configuration Options**

```go
// GameResultsServiceConfig allows customization
type GameResultsServiceConfig struct {
    CheckInterval      time.Duration // Default: 5 minutes
    EnableBackfill     bool          // Fetch historical games
    BackfillDays       int           // How far back to look
    MaxRetries         int           // API retry attempts
    EnablePlayerStats  bool          // Store player-level data
    DataRetentionDays  int           // Delete old data (0=keep all)
}
```

---

## 🚨 **Edge Cases to Handle**

1. **Postponed Games**: gameState might change from "FINAL" to "PPD"
2. **Game Corrections**: NHL might update stats after initial "FINAL"
3. **Duplicate Processing**: Ensure we don't process same game twice
4. **API Failures**: Retry logic with exponential backoff
5. **Data Corruption**: Validate all data before storing
6. **Time Zones**: Handle game times correctly across zones
7. **Preseason vs Regular**: Different handling for game types

---

## 📝 **Logging Strategy**

```
Startup:
📊 Game Results Service initialized
📊 Loaded 42 processed games from index
📊 Service started, checking every 5 minutes

During Operation:
🔍 Checking for completed games...
📊 Found 2 new completed games
📥 Fetching data for game 2025010123 (UTA vs COL)
✅ Game processed: UTA 4 - COL 3 (OT)
💾 Saved to data/results/2025-10.json
🏆 Elo ratings updated: UTA 1485 → 1510 (+25)
🎯 Poisson rates updated
📊 Accuracy tracking updated
✅ All games processed, next check in 5 minutes

Errors:
❌ Failed to fetch game 2025010124: API timeout
⚠️ Retrying in 30 seconds...
```

---

## 🎉 **Success Criteria**

✅ **Automatically detects** completed games within 5 minutes  
✅ **Fetches comprehensive** game data from NHL API  
✅ **Stores data persistently** in monthly JSON files  
✅ **Feeds data** to Elo & Poisson models automatically  
✅ **Tracks processed games** to avoid duplicates  
✅ **Survives restarts** with no data loss  
✅ **Improves model accuracy** continuously  

---

## 🚀 **Next Steps**

1. **Review this plan** - Does this approach make sense?
2. **Approve implementation** - Ready to start coding?
3. **Prioritize features** - Which phases are most important?
4. **Set timeline** - How quickly do you want this?

**Estimated Implementation Time:**
- Phase 1-3 (Core + Models): ~2-3 hours
- Phase 4 (Lifecycle): ~30 minutes
- Phase 5 (Robustness): ~1 hour
- **Total: 4-5 hours of development**

Would you like me to proceed with implementing this service?

