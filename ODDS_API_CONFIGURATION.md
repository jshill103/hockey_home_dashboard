# 💰 Odds API Configuration Guide

## Overview

The NHL Dashboard now supports **The Odds API** for real-time betting market integration. This enhances predictions with live odds, line movements, and market consensus data.

---

## ✅ What Was Implemented

### 1. **Command-Line Flag Support**
```bash
./web_server -team UTA -odds-key your_api_key_here
```

The new `-odds-key` flag allows you to pass your Odds API key directly via command line, making it easy to test and configure.

### 2. **Environment Variable Support**
```bash
export ODDS_API_KEY=your_api_key_here
./web_server -team UTA
```

The application reads `ODDS_API_KEY` from environment variables, perfect for Docker and production deployments.

### 3. **Docker Support**
```bash
# Docker run
docker run -d -p 8080:8080 \
  -e TEAM_CODE=UTA \
  -e ODDS_API_KEY=your_api_key_here \
  jshillingburg/hockey_home_dashboard:latest

# Docker Compose
environment:
  - TEAM_CODE=UTA
  - ODDS_API_KEY=your_odds_api_key_here
```

### 4. **Automatic Fallback**
If no API key is provided, the application runs normally with betting market features disabled. All other ML predictions remain fully functional.

---

## 🔑 Getting Your API Key

### Step 1: Sign Up
Visit: [https://the-odds-api.com/](https://the-odds-api.com/)

### Step 2: Get Your Key
- Free tier includes **500 requests/month**
- Covers NHL and other major sports
- Real-time odds from multiple sportsbooks

### Step 3: Copy Your Key
From the dashboard, copy your API key (looks like: `a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6`)

---

## 📖 Usage Examples

### Local Development
```bash
# Command line flag (easiest for testing)
./web_server -team UTA -odds-key your_api_key_here

# Environment variable
export ODDS_API_KEY=your_api_key_here
./web_server -team UTA
```

### Docker
```bash
# Single line
docker run -d -p 8080:8080 -e TEAM_CODE=UTA -e ODDS_API_KEY=your_key jshillingburg/hockey_home_dashboard:latest

# With persistent data
docker run -d \
  -p 8080:8080 \
  -e TEAM_CODE=UTA \
  -e ODDS_API_KEY=your_api_key_here \
  -v hockey_ml_data:/app/data \
  --name uta-dashboard \
  jshillingburg/hockey_home_dashboard:latest
```

### Docker Compose
```yaml
version: '3.8'

services:
  nhl-dashboard:
    image: jshillingburg/hockey_home_dashboard:latest
    ports:
      - "8080:8080"
    environment:
      - TEAM_CODE=UTA
      - ODDS_API_KEY=your_odds_api_key_here  # Add your key here
    volumes:
      - nhl_data:/app/data
    restart: unless-stopped

volumes:
  nhl_data:
```

### Full Featured Setup
```bash
# All optional features enabled
docker run -d \
  -p 8080:8080 \
  -e TEAM_CODE=UTA \
  -e WEATHER_API_KEY=your_weather_key \
  -e ODDS_API_KEY=your_odds_key \
  -v hockey_ml_data:/app/data \
  --name uta-dashboard \
  jshillingburg/hockey_home_dashboard:latest
```

---

## 💡 What You Get

### Betting Market Features
When enabled, the betting market service provides:

- **Real-time Odds** - Live betting lines from multiple sportsbooks (DraftKings, FanDuel, BetMGM, etc.)
- **Line Movement** - Track how odds change over time
- **Sharp Money Detection** - Identify professional betting patterns
- **Market Consensus** - See where the betting public leans
- **Enhanced AI Predictions** - Predictions enriched with market data
- **Moneyline, Spread, and Totals** - Complete betting market coverage

### Integration Points
The betting market data is integrated into:
- **AI Model Insights Widget** - Shows market consensus
- **Prediction Factors** - Includes market confidence, line movement, sharp money indicators
- **Phase 4 Intelligence** - Part of the advanced prediction system

---

## 📊 Free Tier Usage

### What You Get
- **500 requests/month**
- Approximately **16 requests/day**
- Real-time odds from 10+ sportsbooks
- Covers NHL, NBA, NFL, MLB, and more

### How We Use It
- API is called once per prediction request
- Odds are cached to minimize API calls
- Perfect for following your favorite team
- No rate limiting issues with normal usage

### Tips
1. **Cache Time**: Odds are cached for performance
2. **Usage Tracking**: Monitor your usage in The Odds API dashboard
3. **Upgrade**: Paid tiers available for higher volume

---

## 🔍 Verification

### Check if It's Working

**1. Look for startup message:**
```
💰 Odds API key set via command line
✅ Betting Market Service initialized
```

**2. Check logs for API calls:**
```
📊 Fetching betting odds for NHL game...
✅ Received odds from 8 sportsbooks
```

**3. If disabled (no key):**
```
⚠️ Betting Market Service disabled (no ODDS_API_KEY)
💡 To enable: Get free API key from https://the-odds-api.com/
   Then set: export ODDS_API_KEY=your_key_here
```

---

## 🛠️ Troubleshooting

### Issue: "Betting Market Service disabled"
**Solution**: Make sure your API key is set
```bash
# Check if it's set
echo $ODDS_API_KEY

# Set it
export ODDS_API_KEY=your_api_key_here
```

### Issue: "API rate limit exceeded"
**Solution**: You've used your 500 requests for the month
- Wait until next month (resets on signup date)
- Upgrade to paid tier
- Reduce prediction frequency

### Issue: "Invalid API key"
**Solution**: 
- Check for typos in your key
- Verify key is active in The Odds API dashboard
- Try regenerating your key

### Issue: No betting odds showing
**Possible causes**:
- Game too far in future (odds not posted yet)
- Game already completed
- NHL season not active

---

## 📝 Configuration Files Updated

The following files were updated to support Odds API:

1. **`main.go`**
   - Added `-odds-key` command-line flag
   - Environment variable handling

2. **`README.md`**
   - New "Betting Market Configuration" section
   - Usage examples and documentation

3. **`DOCKER_HUB_README.md`**
   - Docker usage examples with Odds API
   - Environment variable documentation

4. **`docker-compose.yml`**
   - Added `ODDS_API_KEY` option (commented)

---

## 🔐 Security Best Practices

### Do:
✅ Use environment variables in production
✅ Keep API keys in `.env` files (not committed to git)
✅ Use Docker secrets for sensitive deployments
✅ Monitor usage in The Odds API dashboard

### Don't:
❌ Commit API keys to git
❌ Share API keys publicly
❌ Hardcode keys in source code
❌ Use same key across multiple environments

---

## 📊 Example Output

When enabled, you'll see betting market data in predictions:

```json
{
  "marketConsensus": 0.62,
  "marketLineMovement": 0.05,
  "sharpMoneyIndicator": 0.75,
  "marketConfidenceVal": 0.68
}
```

This data is factored into the ensemble prediction for enhanced accuracy!

---

## 🚀 Next Steps

1. **Get your free API key**: [https://the-odds-api.com/](https://the-odds-api.com/)
2. **Set the environment variable** or use the command-line flag
3. **Run the application** and check for the initialization message
4. **View enhanced predictions** with betting market intelligence

---

## 📚 Additional Resources

- **The Odds API Docs**: [https://the-odds-api.com/liveapi/guides/v4/](https://the-odds-api.com/liveapi/guides/v4/)
- **GitHub Repository**: [jshill103/hockey_home_dashboard](https://github.com/jshill103/hockey_home_dashboard)
- **Docker Hub**: [jshillingburg/hockey_home_dashboard](https://hub.docker.com/r/jshillingburg/hockey_home_dashboard)

---

**Built with ❤️ for hockey fans and sports analytics enthusiasts!** 🏒💰

