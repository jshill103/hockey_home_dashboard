# System Statistics Feature

## Overview

A comprehensive statistics tracking system that monitors backfilled games, prediction accuracy, and system health. Accessible via a clickable calendar emoji (📅) in the UI.

## Features Implemented

### 1. Statistics Tracking Service (`services/system_stats_service.go`)

#### Backfill Statistics
- **Total Games Processed**: Count of all games backfilled
- **Play-by-Play Games**: Games with xG and shot quality data
- **Shift Data Games**: Games with shift analysis
- **Game Summary Games**: Games with enhanced context
- **Failed Games**: Count of failed backfill attempts
- **Events Processed**: Total play events analyzed
- **Processing Time**: Average processing time per game
- **Last Backfill Time**: Timestamp of last backfill

#### Prediction Accuracy
- **Total Predictions**: All predictions made by the system
- **Correct Predictions**: Count of correct winner predictions
- **Overall Accuracy**: System-wide accuracy percentage
- **Individual Model Accuracy**: Per-model tracking with:
  - Total predictions
  - Correct predictions
  - Accuracy percentage
  - Average confidence
- **Best/Worst Models**: Automatically identifies top and bottom performers

#### System Health
- **System Uptime**: Time since server started
- **Total API Requests**: Count of NHL API calls
- **Last Updated**: Timestamp of last stat update

### 2. Models (`models/system_stats.go`)

```go
type SystemStats struct {
	BackfillStats    BackfillStats
	PredictionStats  PredictionStats
	LastUpdated      time.Time
	SystemUptime     time.Duration
	TotalAPIRequests int
}
```

### 3. Frontend Integration

#### Clickable Calendar Emoji
- Located in the "Upcoming Games" section header
- Hover effect with scale animation
- Click to open statistics popup

#### Popup Display
- **Overlay**: Full-screen with blur effect
- **Close Methods**:
  - Click X button
  - Click outside popup
  - Press Escape key
- **Responsive**: Mobile-friendly design

#### Visual Features
- Team color theming
- Progress bars for model accuracy
- Time-relative formatting ("5 minutes ago")
- Hover effects on stat items
- Gradient backgrounds

### 4. API Endpoints

#### `/system-stats` (JSON)
Returns raw statistics data for programmatic access.

#### `/system-stats-popup` (HTML)
Returns formatted HTML for the popup display.

### 5. Data Persistence

Statistics are persisted to disk in two files:

- `data/metrics/system_stats.json` - Overall system statistics
- `data/metrics/prediction_records.json` - Individual prediction records

## Usage

### Recording Backfill Events

```go
systemStatsService.RecordBackfillGame(
    "play-by-play",  // game type
    235,             // events processed
    1250*time.Millisecond, // processing time
)
```

### Recording Predictions

```go
systemStatsService.RecordPrediction(
    gameID,
    gameDate,
    "NSH",           // home team
    "UTA",           // away team
    "NSH",           // predicted winner
    "3-2",           // predicted score
    0.73,            // confidence
    modelPredictions, // individual model predictions
)
```

### Verifying Predictions

```go
systemStatsService.VerifyPrediction(
    gameID,
    "NSH",           // actual winner
    "4-2",           // actual score
)
```

### Incrementing API Requests

```go
systemStatsService.IncrementAPIRequest()
```

## Frontend Access

1. Navigate to the dashboard
2. Look for the calendar emoji (📅) next to "Upcoming Games"
3. Click the emoji to open the statistics popup
4. View comprehensive system stats including:
   - Backfill progress
   - Prediction accuracy
   - Model performance comparison
   - System health

## Statistics Categories

### 📦 Backfill Status
- Games processed breakdown by type
- Total events analyzed
- Average processing time
- Failed attempts

### 🎯 Prediction Accuracy
- Overall system accuracy
- Total predictions made
- Correct predictions count
- Best performing model
- Last prediction time

### 🤖 Model Performance
- Accuracy bars for each model
- Prediction counts (correct/total)
- Visual comparison of all 9 models

### ⚙️ System Info
- Current uptime
- API request count
- Last update time

## File Structure

```
/Users/jaredshillingburg/Jared-Repos/go_uhc/
├── models/
│   └── system_stats.go              # Data structures
├── services/
│   └── system_stats_service.go      # Tracking logic
├── handlers/
│   └── system_stats.go              # HTTP handlers
├── handlers/
│   └── home.go                      # UI integration
├── main.go                          # Service initialization
└── data/
    └── metrics/
        ├── system_stats.json        # Persisted stats
        └── prediction_records.json  # Prediction history
```

## Future Enhancements

### Potential Improvements
1. **Export to CSV**: Download stats as CSV for analysis
2. **Historical Charts**: Graph accuracy trends over time
3. **Email Alerts**: Notify on accuracy drops
4. **Model Comparison**: Side-by-side model analysis
5. **Confidence Calibration**: Track if confidence matches accuracy
6. **Score Accuracy**: Track not just winner but exact score predictions
7. **API Rate Limiting Visualization**: Show API usage patterns
8. **Backfill Progress Bar**: Visual progress during backfill operations

### Integration Points
- Automatically record stats during game result collection
- Integrate with live prediction system
- Add to health check endpoint
- Create admin dashboard

## Technical Details

### Thread Safety
- All operations use `sync.RWMutex` for concurrent access
- Safe to call from multiple goroutines

### Performance
- Stats saved every 100 API requests (batched writes)
- Immediate save for critical stats (predictions, backfill)
- In-memory caching with disk persistence

### Error Handling
- Graceful degradation if stats service unavailable
- Non-blocking operations
- Error logging for debugging

## Testing

### Manual Testing
1. Start server: `./web_server UTA`
2. Open browser: `http://localhost:8080`
3. Click calendar emoji (📅)
4. Verify popup displays stats
5. Test close methods (X, outside click, Escape)

### API Testing
```bash
# Get raw stats
curl http://localhost:8080/system-stats | jq '.'

# Get popup HTML
curl http://localhost:8080/system-stats-popup
```

## Notes

- Initial stats will be zero until games are processed
- Model accuracy requires verified predictions
- System uptime resets on server restart
- Prediction records accumulate over time
- All timestamps use server local time zone

