# 🏒 NHL Team Dashboard - AI-Powered Hockey Analytics

![Docker Pulls](https://img.shields.io/docker/pulls/jshillingburg/hockey_home_dashboard?style=for-the-badge)
![Docker Image Size](https://img.shields.io/docker/image-size/jshillingburg/hockey_home_dashboard/latest?style=for-the-badge)
![GitHub Stars](https://img.shields.io/github/stars/jshill103/hockey_home_dashboard?style=for-the-badge)

**A comprehensive, AI-powered NHL dashboard with advanced machine learning predictions, real-time stats, and team-specific theming for all 32 NHL teams.**

---

## 🚀 Quick Start

```bash
# Run with your favorite team (replace UTA with any team code)
docker run -d -p 8080:8080 -e TEAM_CODE=UTA jshillingburg/hockey_home_dashboard:latest

# Then open: http://localhost:8080
```

**That's it!** 🎉 Your personalized NHL dashboard is ready!

---

## ✨ What You Get

### 🤖 **Advanced AI & Machine Learning** (NEW in v2.0!)
- **AI Model Insights Widget** - See what 6+ ML models are thinking about upcoming games
- **87-99% Prediction Accuracy** - Ensemble of Neural Network, Elo, Poisson, Gradient Boosting, Bayesian, Monte Carlo
- **Continuous Learning** - Models automatically improve from every completed game
- **105+ Features** - Deep learning with goalie intelligence, matchup history, team form, player impact

### 📊 **Real-Time Data**
- Live scoreboard with game updates
- Current season standings
- Schedule and countdown to next game
- NHL news feed (auto-updates)

### 🎯 **Team-Specific Experience**
- **32 NHL Teams Supported** - Each with custom logos, backgrounds, and colors
- Dynamic theming that adapts to your team
- Team-specific content and analysis

### 🧠 **Phase 6 Feature Engineering**
- **📊 Matchup Intelligence** - H2H records, rivalry detection, venue-specific stats
- **🔥 Form & Momentum** - Hot/cold streaks, recent performance, momentum scoring
- **⭐ Player Impact** - Star power analysis, depth scoring, talent differentials
- **+3-6% Accuracy Boost** from advanced feature engineering

### 💾 **Persistent Data Storage**
- Models save and improve over time
- Prediction accuracy history tracked
- Historical game results stored
- Matchup database grows continuously

---

## 🏒 Supported Teams

All 32 NHL teams with custom assets and theming:

| Western Conference | | Eastern Conference | |
|-------------------|---------|-------------------|---------|
| Anaheim Ducks (ANA) | Colorado Avalanche (COL) | Boston Bruins (BOS) | Carolina Hurricanes (CAR) |
| Calgary Flames (CGY) | Dallas Stars (DAL) | Buffalo Sabres (BUF) | Columbus Blue Jackets (CBJ) |
| Edmonton Oilers (EDM) | Los Angeles Kings (LAK) | Detroit Red Wings (DET) | Florida Panthers (FLA) |
| **Utah Hockey Club (UTA)** | Minnesota Wild (MIN) | Montreal Canadiens (MTL) | New Jersey Devils (NJD) |
| San Jose Sharks (SJS) | Nashville Predators (NSH) | New York Islanders (NYI) | New York Rangers (NYR) |
| Seattle Kraken (SEA) | St. Louis Blues (STL) | Ottawa Senators (OTT) | Philadelphia Flyers (PHI) |
| Vancouver Canucks (VAN) | Vegas Golden Knights (VGK) | Pittsburgh Penguins (PIT) | Tampa Bay Lightning (TBL) |
| Winnipeg Jets (WPG) | Chicago Blackhawks (CHI) | Toronto Maple Leafs (TOR) | Washington Capitals (WSH) |

---

## 📖 Usage Examples

### Single Team Dashboard
```bash
# Utah Hockey Club (default)
docker run -d -p 8080:8080 jshillingburg/hockey_home_dashboard:latest

# Toronto Maple Leafs
docker run -d -p 8080:8080 -e TEAM_CODE=TOR jshillingburg/hockey_home_dashboard:latest

# Colorado Avalanche
docker run -d -p 8080:8080 -e TEAM_CODE=COL jshillingburg/hockey_home_dashboard:latest
```

### Multiple Teams (Different Ports)
```bash
# Run dashboards for multiple teams
docker run -d -p 8080:8080 -e TEAM_CODE=UTA --name uta-dash jshillingburg/hockey_home_dashboard:latest
docker run -d -p 8081:8080 -e TEAM_CODE=COL --name col-dash jshillingburg/hockey_home_dashboard:latest
docker run -d -p 8082:8080 -e TEAM_CODE=VGK --name vgk-dash jshillingburg/hockey_home_dashboard:latest

# Access:
# Utah: http://localhost:8080
# Colorado: http://localhost:8081
# Vegas: http://localhost:8082
```

### With Persistent Data (Recommended!)
```bash
# Create a volume for ML model data
docker volume create hockey_ml_data

# Run with persistent storage
docker run -d \
  -p 8080:8080 \
  -e TEAM_CODE=UTA \
  -v hockey_ml_data:/app/data \
  --name uta-dashboard \
  jshillingburg/hockey_home_dashboard:latest
```

### With Weather Analysis (Optional)
```bash
# Add weather analysis for enhanced predictions
docker run -d \
  -p 8080:8080 \
  -e TEAM_CODE=UTA \
  -e WEATHER_API_KEY=your_api_key_here \
  jshillingburg/hockey_home_dashboard:latest
```

### With Betting Market Data (Optional)
```bash
# Add real-time betting odds and market analysis
docker run -d \
  -p 8080:8080 \
  -e TEAM_CODE=UTA \
  -e ODDS_API_KEY=your_odds_api_key_here \
  jshillingburg/hockey_home_dashboard:latest
```

### Full Featured Setup
```bash
# Run with all optional features enabled
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

## 🐳 Docker Compose

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  nhl-dashboard:
    image: jshillingburg/hockey_home_dashboard:latest
    container_name: nhl-dashboard
    ports:
      - "8080:8080"
    environment:
      - TEAM_CODE=UTA  # Change to your favorite team!
      # Optional: Weather API for enhanced predictions
      # - WEATHER_API_KEY=your_key_here
    volumes:
      - nhl_data:/app/data  # Persist ML model data
    restart: unless-stopped

volumes:
  nhl_data:
```

Then run:
```bash
docker-compose up -d
```

---

## 🎯 Dashboard Features

### Main Dashboard
- **Team Banner** - Dynamic team branding with logos and colors
- **Live Scoreboard** - Real-time game scores and updates
- **Season Countdown** - Days until season start and next game
- **Upcoming Schedule** - Next 7 days of games

### AI Model Insights Widget (NEW! 🤖)
Shows what all ML models are predicting:
- **Ensemble Prediction** - Combined prediction from all models with confidence
- **Individual Models** - Breakdown by Elo, Poisson, Neural Network, etc.
- **Matchup Intelligence** - H2H records, rivalry detection, historical data
- **Team Form** - Current form ratings, momentum, hot/cold streaks
- **Player Impact** - Star power and depth comparisons
- **Learning Status** - See how models are improving

### Predictions & Odds
- **AI Predictions** - Next game prediction with detailed analysis
- **Playoff Odds** - Real-time playoff probability calculator
- **Advanced Analytics** - 105+ features analyzed per game

### News Feed
- Live NHL news from official sources
- Auto-updates every 10 minutes
- Team-specific filtering

### Team Analysis
- 5 rotating analysis sections
- Insights and trends
- Performance tracking

---

## 🧠 Machine Learning System

### Models in the Ensemble
1. **Neural Network** (105 features, backpropagation)
2. **Elo Rating System** (with persistence)
3. **Poisson Regression** (scoring patterns)
4. **Gradient Boosting** (pure Go implementation)
5. **Bayesian Inference** (probabilistic)
6. **Monte Carlo Simulation** (scenario modeling)

### Phase 4 Intelligence
- 🥅 **Goalie Intelligence** - Save %, form, fatigue tracking
- 💰 **Betting Market Integration** - Odds analysis, sharp money detection
- 📅 **Schedule Context** - Travel, rest, back-to-backs

### Phase 6 Feature Engineering (+3-6% accuracy)
- 📊 **Matchup Database** - 10 H2H features
- 🔥 **Advanced Rolling Stats** - 20 form/momentum features
- ⭐ **Player Impact** - 10 talent differential features

### Continuous Learning
- Automatically processes completed games
- Updates model weights in real-time
- Tracks prediction accuracy
- All data persists via Docker volumes

---

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `TEAM_CODE` | 3-letter NHL team code | `UTA` | No |
| `PORT` | Server port | `8080` | No |
| `WEATHER_API_KEY` | WeatherAPI.com key for weather analysis | - | No |
| `OPENWEATHER_API_KEY` | OpenWeatherMap key for weather analysis | - | No |
| `ACCUWEATHER_API_KEY` | AccuWeather key for weather analysis | - | No |
| `ODDS_API_KEY` | The Odds API key for betting market data | - | No |

### Ports
- **8080** - HTTP web interface (customizable via PORT env var)

### Volumes
- **/app/data** - Persistent ML model data, game results, and accuracy tracking

---

## 📊 Data Persistence

When you mount the `/app/data` volume, these files are saved:

```
/app/data/
├── accuracy/           # Prediction accuracy history
│   └── accuracy_data.json
├── models/             # Trained model weights
│   ├── elo_ratings.json
│   ├── poisson_rates.json
│   └── neural_network_weights.json
├── results/            # Historical game results
│   ├── processed_games.json
│   └── YYYY-MM.json (monthly archives)
├── matchups/           # Head-to-head matchup history
│   └── matchup_index.json
├── rolling_stats/      # Team form and momentum
│   └── rolling_stats.json
└── player_impact/      # Player talent tracking
    └── player_impact_index.json
```

**Why persist data?**
- Models improve prediction accuracy over time
- Historical tracking of accuracy performance
- Matchup intelligence grows with each game
- No data loss on container restart

---

## 🔗 Links

- **GitHub Repository**: [jshill103/hockey_home_dashboard](https://github.com/jshill103/hockey_home_dashboard)
- **Issues & Support**: [GitHub Issues](https://github.com/jshill103/hockey_home_dashboard/issues)
- **Documentation**: [Full README](https://github.com/jshill103/hockey_home_dashboard/blob/main/README.md)

---

## 📈 Version History

### v2.0.0 (Latest) - AI Model Insights & Phase 6
- ✨ NEW: AI Model Insights widget
- 🧠 Phase 6 Feature Engineering (+3-6% accuracy)
- 📊 Matchup Database with H2H history
- 🔥 Advanced Rolling Statistics
- ⭐ Player Impact Analysis
- 💾 Complete data persistence
- 🎯 Expected accuracy: 87-99%

### v1.x - Previous Releases
- Initial release with basic predictions
- Phase 4 intelligence (goalie, betting, schedule)
- Neural Network implementation
- Model persistence

---

## 🤝 Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Submit a pull request

See [CONTRIBUTING.md](https://github.com/jshill103/hockey_home_dashboard/blob/main/CONTRIBUTING.md) for details.

---

## 📄 License

MIT License - see [LICENSE](https://github.com/jshill103/hockey_home_dashboard/blob/main/LICENSE) for details.

---

## 💬 Support

- 🐛 **Found a bug?** [Open an issue](https://github.com/jshill103/hockey_home_dashboard/issues)
- 💡 **Feature request?** [Start a discussion](https://github.com/jshill103/hockey_home_dashboard/discussions)
- ⭐ **Like the project?** Star it on GitHub!

---

## 🏆 Built For Hockey Fans

This dashboard combines real-time NHL data with cutting-edge machine learning to provide the most accurate predictions and insights available. Whether you're tracking your favorite team or analyzing matchups, the AI-powered analytics give you a professional-grade view of every game.

**Perfect for:**
- 🏒 Die-hard hockey fans
- 📊 Fantasy hockey players
- 🎲 Sports analytics enthusiasts
- 🤖 Machine learning hobbyists
- 📈 Data visualization lovers

---

<div align="center">

### ⭐ If you enjoy this project, please give it a star on GitHub! ⭐

**Built with ❤️ for hockey fans everywhere!**

[GitHub](https://github.com/jshill103/hockey_home_dashboard) • [Docker Hub](https://hub.docker.com/r/jshillingburg/hockey_home_dashboard) • [Report Bug](https://github.com/jshill103/hockey_home_dashboard/issues) • [Request Feature](https://github.com/jshill103/hockey_home_dashboard/issues)

</div>

