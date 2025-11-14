# Schedule API Deployment Instructions

## ✅ What's Been Done

1. ✅ **Created Schedule API** (`handlers/schedule_api.go`)
   - 5 new JSON endpoints for schedule data
   - Supports all 32 NHL teams
   - CORS enabled for external apps
   - League-wide schedule endpoint

2. ✅ **Registered API Routes** (`main.go`)
   - `/api/schedule/next`
   - `/api/schedule/upcoming`
   - `/api/schedule/season`
   - `/api/schedule/all-teams`
   - `/api/schedule/health`

3. ✅ **Created Documentation** (`docs/SCHEDULE_API.md`)
   - Complete API documentation
   - Request/response examples
   - Use cases and integration examples
   - Team code reference

4. ✅ **Built Docker Images**
   - Tagged: `schedule-api`, `latest`
   - Build tested and successful

5. ✅ **Pushed to DockerHub**
   - Repository: `jshillingburg/hockey_home_dashboard`
   - Available for deployment

6. ✅ **Committed to Git**
   - All code changes committed
   - Pushed to GitHub

---

## 🏠 When You Get Home - Deployment

### Step 1: Deploy to k3s Cluster

```bash
# SSH to home server
ssh jared@192.168.1.99

# Restart deployment to pull latest image
k3s kubectl rollout restart deployment hockey-dashboard -n hockey-dashboard

# Watch rollout progress
k3s kubectl rollout status deployment hockey-dashboard -n hockey-dashboard

# Check pod is running
k3s kubectl get pods -n hockey-dashboard

# Check logs for schedule API registration
k3s kubectl logs -f <pod-name> -n hockey-dashboard | grep "Schedule API"
```

You should see:
```
📅 Schedule API endpoints registered (for external apps like video analyzer)
```

### Step 2: Test the API Endpoints

From any machine on your home network:

```bash
# Health check
curl "http://192.168.1.99:30001/api/schedule/health"

# Next game for Utah
curl "http://192.168.1.99:30001/api/schedule/next?team=UTA"

# Upcoming games (next 7 days)
curl "http://192.168.1.99:30001/api/schedule/upcoming?team=UTA"

# All NHL games today
curl "http://192.168.1.99:30001/api/schedule/all-teams"
```

### Step 3: Test CORS (Optional)

From a browser console on any site:

```javascript
fetch('http://192.168.1.99:30001/api/schedule/next?team=UTA')
  .then(r => r.json())
  .then(data => console.log(data));
```

---

## 🧪 Expected Responses

### Health Check

```json
{
  "status": "healthy",
  "timestamp": "2025-11-14T...",
  "version": "1.0.0",
  "endpoints": [
    "/api/schedule/next",
    "/api/schedule/upcoming",
    "/api/schedule/season"
  ],
  "description": "NHL schedule API for external applications"
}
```

### Next Game

```json
{
  "status": "success",
  "teamCode": "UTA",
  "lastUpdated": "2025-11-14T...",
  "nextGame": {
    "id": 2025020123,
    "gameDate": "2025-11-15",
    "startTimeUTC": "2025-11-15T02:00:00Z",
    "formattedTime": "2025-11-14 19:00:00",
    "tvBroadcasts": [
      {
        "network": "ESPN"
      }
    ],
    "homeTeam": {
      "commonName": {
        "default": "Utah Hockey Club"
      },
      "abbrev": "UTA"
    },
    "awayTeam": {
      "commonName": {
        "default": "Colorado Avalanche"
      },
      "abbrev": "COL"
    },
    "venue": {
      "default": "Delta Center"
    }
  },
  "count": 1
}
```

### All Teams (League-Wide)

```json
{
  "status": "success",
  "date": "2025-11-14",
  "lastUpdated": "2025-11-14T...",
  "games": [
    {
      "id": 2025020115,
      "gameDate": "2025-11-14",
      "homeTeam": { ... },
      "awayTeam": { ... },
      "venue": { ... },
      "tvBroadcasts": [ ... ]
    },
    // ... more games
  ],
  "count": 8
}
```

---

## 📝 Testing Checklist

When home, verify:

- [ ] Deployment successful (pod running)
- [ ] Logs show "Schedule API endpoints registered"
- [ ] `/api/schedule/health` returns healthy status
- [ ] `/api/schedule/next?team=UTA` returns Utah's next game
- [ ] `/api/schedule/upcoming?team=UTA` returns multiple games
- [ ] `/api/schedule/all-teams` returns league-wide schedule
- [ ] CORS headers present (`Access-Control-Allow-Origin: *`)
- [ ] Can call API from browser console (different origin)

---

## 🔌 Integration with Video Analyzer (Future)

When you create the video analyzer app, it can use these endpoints:

### Example: Get Today's Games for Recording

```go
// In video analyzer app
type Game struct {
    ID           int       `json:"id"`
    GameDate     string    `json:"gameDate"`
    StartTimeUTC string    `json:"startTimeUTC"`
    HomeTeam     TeamInfo  `json:"homeTeam"`
    AwayTeam     TeamInfo  `json:"awayTeam"`
    Broadcasts   []Broadcast `json:"tvBroadcasts"`
}

func fetchTodaysGames() ([]Game, error) {
    url := "http://hockey-dashboard.hockey-dashboard.svc.cluster.local:8080/api/schedule/all-teams"
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Status string `json:"status"`
        Games  []Game `json:"games"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Games, nil
}

// Use to schedule recordings
games, err := fetchTodaysGames()
for _, game := range games {
    // Find TV network
    for _, broadcast := range game.Broadcasts {
        // Schedule HDHomeRun recording
        scheduleRecording(game, broadcast.Network)
    }
}
```

### Example: Monitor Upcoming Games

```python
# Python script to monitor upcoming games
import requests
from datetime import datetime, timedelta

def check_upcoming_games(team="UTA"):
    url = f"http://192.168.1.99:30001/api/schedule/upcoming?team={team}"
    response = requests.get(url)
    data = response.json()
    
    if data['status'] == 'success':
        games = data['upcomingGames']
        print(f"📅 {len(games)} games in next 7 days for {team}:")
        
        for game in games:
            print(f"  {game['awayTeam']['abbrev']} @ {game['homeTeam']['abbrev']}")
            print(f"  Time: {game['formattedTime']}")
            print(f"  TV: {', '.join([b['network'] for b in game['tvBroadcasts']])}")
            print()

# Run daily
check_upcoming_games()
```

---

## 🐛 Troubleshooting

### API Returns Empty Results

**Issue**: `nextGame: null` or `upcomingGames: []`

**Possible Causes**:
1. No games scheduled (off-season or bye week)
2. Invalid team code
3. NHL API down

**Check**:
```bash
# Verify NHL API is working
curl "https://api-web.nhle.com/v1/club-schedule/UTA/week/now"
```

### CORS Not Working

**Issue**: Browser blocks cross-origin request

**Check**:
```bash
# Verify CORS header is present
curl -I "http://192.168.1.99:30001/api/schedule/health"
# Should see: Access-Control-Allow-Origin: *
```

### Deployment Fails

**Issue**: Pod stuck in `ImagePullBackOff` or `CrashLoopBackOff`

**Check**:
```bash
# Check pod status
k3s kubectl describe pod <pod-name> -n hockey-dashboard

# Check logs
k3s kubectl logs <pod-name> -n hockey-dashboard
```

**Fix**:
```bash
# Force pull latest image
k3s kubectl delete pod <pod-name> -n hockey-dashboard
# Let k8s recreate it
```

---

## 📊 Success Metrics

Once deployed, the Schedule API will:

✅ Provide NHL schedule data to external apps  
✅ Enable video analyzer to schedule recordings  
✅ Support all 32 NHL teams  
✅ Update automatically (1-hour cache)  
✅ Handle errors gracefully  
✅ Work cross-origin (CORS enabled)  

---

## 🚀 Next Steps

After deployment and testing:

1. **Create Video Analyzer App**
   - New repository: `hockey_video_analyzer`
   - Use Schedule API to get game times
   - Schedule HDHomeRun recordings

2. **Monitor Usage**
   - Add logging to track API calls
   - Monitor response times
   - Check for errors

3. **Enhancements** (if needed)
   - Add filtering (by TV network, venue, etc.)
   - Add pagination for large results
   - Add WebSocket for real-time updates
   - Add caching headers for better performance

---

## 📚 Documentation

Full documentation available at:
- **API Reference**: `docs/SCHEDULE_API.md`
- **Video App Architecture**: `docs/VIDEO_APP_ARCHITECTURE.md`
- **Video App Next Steps**: `docs/VIDEO_APP_NEXT_STEPS.md`

---

## ✅ Summary

**Status**: Ready to deploy when home

**What to do**:
1. SSH to home server
2. Run `k3s kubectl rollout restart deployment hockey-dashboard -n hockey-dashboard`
3. Test endpoints with curl
4. Start planning video analyzer app

**API Base URL**: `http://192.168.1.99:30001/api/schedule/`

**Docker Image**: `jshillingburg/hockey_home_dashboard:latest`

**Git Commit**: `829a4a9` - feat: Add JSON API endpoints for NHL schedule data

