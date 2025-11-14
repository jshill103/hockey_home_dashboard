# Schedule API Documentation

## Overview

The Schedule API provides JSON endpoints for accessing NHL game schedule data. These endpoints are designed for external applications (like the video analyzer) to programmatically access game schedules.

All endpoints return JSON responses and support CORS for cross-origin requests.

---

## Base URL

```
http://your-dashboard-url
```

For local development: `http://localhost:8080`  
For k3s cluster: `http://192.168.1.99:30001`

---

## Endpoints

### 1. Get Next Upcoming Game

Returns the next scheduled game for a specific team.

**Endpoint**: `GET /api/schedule/next`

**Query Parameters**:
- `team` (optional) - Team code (default: `UTA`)

**Example Request**:
```bash
curl "http://localhost:8080/api/schedule/next?team=UTA"
```

**Example Response**:
```json
{
  "status": "success",
  "teamCode": "UTA",
  "lastUpdated": "2025-11-14T10:30:00Z",
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

**No Game Available**:
```json
{
  "status": "success",
  "teamCode": "UTA",
  "lastUpdated": "2025-11-14T10:30:00Z",
  "nextGame": null,
  "count": 0
}
```

---

### 2. Get Upcoming Games (Next 7 Days)

Returns all games scheduled in the next 7 days for a specific team.

**Endpoint**: `GET /api/schedule/upcoming`

**Query Parameters**:
- `team` (optional) - Team code (default: `UTA`)

**Example Request**:
```bash
curl "http://localhost:8080/api/schedule/upcoming?team=UTA"
```

**Example Response**:
```json
{
  "status": "success",
  "teamCode": "UTA",
  "lastUpdated": "2025-11-14T10:30:00Z",
  "upcomingGames": [
    {
      "id": 2025020123,
      "gameDate": "2025-11-15",
      "startTimeUTC": "2025-11-15T02:00:00Z",
      "formattedTime": "2025-11-14 19:00:00",
      "tvBroadcasts": [...],
      "homeTeam": {...},
      "awayTeam": {...},
      "venue": {...}
    },
    {
      "id": 2025020124,
      "gameDate": "2025-11-17",
      "startTimeUTC": "2025-11-17T02:30:00Z",
      "formattedTime": "2025-11-16 19:30:00",
      "tvBroadcasts": [...],
      "homeTeam": {...},
      "awayTeam": {...},
      "venue": {...}
    }
  ],
  "count": 2
}
```

---

### 3. Get Full Season Schedule

Returns the complete schedule for a team's entire season.

**Endpoint**: `GET /api/schedule/season`

**Query Parameters**:
- `team` (optional) - Team code (default: `UTA`)
- `season` (optional) - Season year in format YYYYYYYY (default: `20242025`)

**Example Request**:
```bash
curl "http://localhost:8080/api/schedule/season?team=UTA&season=20242025"
```

**Example Response**:
```json
{
  "status": "success",
  "teamCode": "UTA",
  "lastUpdated": "2025-11-14T10:30:00Z",
  "seasonGames": [
    {
      "id": 2025020001,
      "gameDate": "2024-10-08",
      "startTimeUTC": "2024-10-08T23:00:00Z",
      "homeTeam": {...},
      "awayTeam": {...},
      "venue": {...}
    },
    // ... all 82+ games
  ],
  "count": 82
}
```

---

### 4. Get All Teams Schedule (League-Wide)

Returns all NHL games for a specific date. Useful for recording all games happening on a given day.

**Endpoint**: `GET /api/schedule/all-teams`

**Query Parameters**:
- `date` (optional) - Date in format `YYYY-MM-DD` (default: today)

**Example Request**:
```bash
curl "http://localhost:8080/api/schedule/all-teams?date=2025-11-14"
```

**Example Response**:
```json
{
  "status": "success",
  "date": "2025-11-14",
  "lastUpdated": "2025-11-14T10:30:00Z",
  "games": [
    {
      "id": 2025020115,
      "gameDate": "2025-11-14",
      "startTimeUTC": "2025-11-14T23:00:00Z",
      "homeTeam": {
        "commonName": {
          "default": "Toronto Maple Leafs"
        },
        "abbrev": "TOR"
      },
      "awayTeam": {
        "commonName": {
          "default": "Boston Bruins"
        },
        "abbrev": "BOS"
      },
      "venue": {
        "default": "Scotiabank Arena"
      },
      "tvBroadcasts": [
        {
          "network": "ESPN"
        }
      ]
    },
    // ... all games for the date
  ],
  "count": 8
}
```

---

### 5. API Health Check

Returns health status and metadata about the Schedule API.

**Endpoint**: `GET /api/schedule/health`

**Example Request**:
```bash
curl "http://localhost:8080/api/schedule/health"
```

**Example Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-11-14T10:30:00Z",
  "version": "1.0.0",
  "endpoints": [
    "/api/schedule/next",
    "/api/schedule/upcoming",
    "/api/schedule/season"
  ],
  "description": "NHL schedule API for external applications",
  "usage": {
    "next": "GET /api/schedule/next?team=UTA - Returns next upcoming game",
    "upcoming": "GET /api/schedule/upcoming?team=UTA - Returns games in next 7 days",
    "season": "GET /api/schedule/season?team=UTA&season=20242025 - Returns full season schedule"
  }
}
```

---

## Team Codes

All NHL teams use 3-letter abbreviations:

| Team | Code |
|------|------|
| Anaheim Ducks | ANA |
| Boston Bruins | BOS |
| Buffalo Sabres | BUF |
| Calgary Flames | CGY |
| Carolina Hurricanes | CAR |
| Chicago Blackhawks | CHI |
| Colorado Avalanche | COL |
| Columbus Blue Jackets | CBJ |
| Dallas Stars | DAL |
| Detroit Red Wings | DET |
| Edmonton Oilers | EDM |
| Florida Panthers | FLA |
| Los Angeles Kings | LAK |
| Minnesota Wild | MIN |
| Montreal Canadiens | MTL |
| Nashville Predators | NSH |
| New Jersey Devils | NJD |
| New York Islanders | NYI |
| New York Rangers | NYR |
| Ottawa Senators | OTT |
| Philadelphia Flyers | PHI |
| Pittsburgh Penguins | PIT |
| San Jose Sharks | SJS |
| Seattle Kraken | SEA |
| St. Louis Blues | STL |
| Tampa Bay Lightning | TBL |
| Toronto Maple Leafs | TOR |
| Utah Hockey Club | UTA |
| Vancouver Canucks | VAN |
| Vegas Golden Knights | VGK |
| Washington Capitals | WSH |
| Winnipeg Jets | WPG |

---

## Error Responses

All endpoints return consistent error responses:

**Error Response Format**:
```json
{
  "status": "error",
  "error": "Error message describing what went wrong",
  "teamCode": "UTA",
  "games": [],
  "count": 0
}
```

**Common Error Scenarios**:
1. **Invalid Team Code**: Returns empty results
2. **NHL API Down**: Status "error" with descriptive message
3. **Invalid Season**: Returns empty season schedule
4. **Network Issues**: Returns error status with timeout message

---

## Rate Limiting

The Schedule API uses the underlying NHL API which has built-in caching and rate limiting:

- **Cache Duration**: 1 hour for schedule data
- **Rate Limit**: No explicit limit (uses NHL's public API)
- **Recommended**: Cache responses on client side for at least 15-30 minutes

---

## CORS Support

All endpoints include CORS headers to allow cross-origin requests:

```
Access-Control-Allow-Origin: *
```

This enables external applications (like the video analyzer) to call the API from different domains.

---

## Use Cases

### Video Analyzer Application

The video analyzer can use these endpoints to:

1. **Schedule Recordings**:
   ```bash
   # Get all games happening today
   curl "http://dashboard:8080/api/schedule/all-teams?date=2025-11-14"
   
   # Parse response and schedule HDHomeRun recordings
   ```

2. **Team-Specific Monitoring**:
   ```bash
   # Get Utah's upcoming games
   curl "http://dashboard:8080/api/schedule/upcoming?team=UTA"
   
   # Set up automated recording for next 7 days
   ```

3. **Full Season Planning**:
   ```bash
   # Get entire season schedule
   curl "http://dashboard:8080/api/schedule/season?team=UTA&season=20242025"
   
   # Pre-allocate storage and plan recording schedule
   ```

### External Dashboards

```javascript
// Fetch and display upcoming games
fetch('http://dashboard:8080/api/schedule/upcoming?team=UTA')
  .then(response => response.json())
  .then(data => {
    if (data.status === 'success') {
      data.upcomingGames.forEach(game => {
        console.log(`${game.awayTeam.commonName.default} @ ${game.homeTeam.commonName.default}`);
        console.log(`Time: ${game.formattedTime}`);
      });
    }
  });
```

### Monitoring & Alerts

```python
import requests
import datetime

# Check for games today
response = requests.get('http://dashboard:8080/api/schedule/next?team=UTA')
data = response.json()

if data['nextGame']:
    game = data['nextGame']
    game_time = datetime.datetime.fromisoformat(game['startTimeUTC'].replace('Z', '+00:00'))
    
    # Send alert if game is within 2 hours
    if (game_time - datetime.datetime.now(datetime.timezone.utc)).total_seconds() < 7200:
        print(f"🚨 Game starting soon: {game['awayTeam']['abbrev']} @ {game['homeTeam']['abbrev']}")
```

---

## Testing

### Local Testing

```bash
# Start the dashboard
go run main.go

# Test endpoints
curl "http://localhost:8080/api/schedule/health"
curl "http://localhost:8080/api/schedule/next?team=UTA"
curl "http://localhost:8080/api/schedule/upcoming?team=UTA"
```

### k3s Cluster Testing

```bash
# From any machine on the network
curl "http://192.168.1.99:30001/api/schedule/health"
curl "http://192.168.1.99:30001/api/schedule/next?team=UTA"
```

---

## Future Enhancements

Potential additions to the Schedule API:

1. **WebSocket Support**: Real-time schedule updates
2. **Filtering**: Filter by TV network, venue, date range
3. **Caching Control**: Client-controlled cache headers
4. **Pagination**: For large season schedules
5. **Game Status**: Include live game status and scores
6. **Notifications**: Webhook support for game reminders

---

## Support

For issues or questions:
- Check logs: `kubectl logs <pod-name> -n hockey-dashboard`
- API health: `GET /api/schedule/health`
- Main dashboard health: `GET /health`

---

## Version History

- **v1.0.0** (2025-11-14): Initial release
  - Next game endpoint
  - Upcoming games endpoint (7 days)
  - Full season schedule endpoint
  - League-wide schedule endpoint
  - Health check endpoint
  - CORS support

