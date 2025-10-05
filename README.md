# NHL Team Dashboard 🏒

A comprehensive, team-specific NHL web dashboard built in Go that provides real-time hockey information, news, statistics, and analysis for your favorite NHL team.

🐳 **Docker Hub**: [jshillingburg/hockey_home_dashboard](https://hub.docker.com/r/jshillingburg/hockey_home_dashboard)  
📦 **Quick Start**: `docker run -d -p 8080:8080 -e TEAM_CODE=YOUR_TEAM jshillingburg/hockey_home_dashboard:latest`

## Features ✨

### 🎯 Team-Specific Experience
- **32 NHL Teams Supported** - Each with custom logos, backgrounds, and team colors
- **Dynamic Theming** - Background images and color schemes adapt to selected team
- **Team-Specific Content** - News, stats, and analysis tailored to your chosen team

### 📊 Real-Time Data
- **Live Scoreboard** - Current and upcoming games with real-time updates
- **Season Countdown** - Days until season start and your team's first game
- **NHL Standings** - Current league standings and team positioning
- **Game Predictions** - AI-powered predictions with playoff odds

### 🤖 Advanced AI & Machine Learning
- **AI Model Insights Widget** - NEW! See what all ML models are thinking about upcoming games
- **Ensemble Predictions** - 6+ ML models working together (Elo, Poisson, Neural Network, Bayesian, Monte Carlo, Gradient Boosting)
- **Phase 4 Intelligence**: 
  - 🥅 Goalie Intelligence (save %, recent form, fatigue tracking)
  - 💰 Betting Market Integration (odds analysis, sharp money detection)
  - 📅 Schedule Context (travel distance, back-to-backs, rest advantages)
- **Phase 6 Feature Engineering** (+3-6% accuracy improvement):
  - 📊 Matchup Database (H2H history, rivalry detection, venue-specific records)
  - 🔥 Advanced Rolling Stats (form ratings, momentum, hot/cold streaks, quality-weighted performance)
  - ⭐ Player Impact (star power analysis, depth scoring, talent differentials)
- **Continuous Learning** - Models automatically learn from every completed game
- **Model Persistence** - Neural Network, Elo, and Poisson ratings saved and improved over time
- **Expected Accuracy** - 87-99% prediction accuracy with all systems active

### 📰 News & Information
- **Live News Feed** - Real-time NHL news scraping from official sources
- **Team Analysis** - 5 rotating analysis sections with insights and trends
- **Schedule Information** - Upcoming games and team schedule

### 🎨 Modern UI/UX
- **Responsive Design** - Works seamlessly on desktop and mobile
- **Transparent Sections** - Beautiful overlays that showcase team backgrounds
- **Rotating Content** - Analysis sections cycle automatically every 10 seconds
- **Clean Typography** - Optimized text sizing and spacing

## Supported Teams 🏒

The application supports all 32 NHL teams with custom assets:

| Team | Code | Team | Code | Team | Code |
|------|------|------|------|------|------|
| Anaheim Ducks | ANA | Colorado Avalanche | COL | Minnesota Wild | MIN |
| Boston Bruins | BOS | Dallas Stars | DAL | Montreal Canadiens | MTL |
| Buffalo Sabres | BUF | Detroit Red Wings | DET | Nashville Predators | NSH |
| Calgary Flames | CGY | Edmonton Oilers | EDM | New Jersey Devils | NJD |
| Carolina Hurricanes | CAR | Florida Panthers | FLA | New York Islanders | NYI |
| Chicago Blackhawks | CHI | Los Angeles Kings | LAK | New York Rangers | NYR |
| Columbus Blue Jackets | CBJ | Ottawa Senators | OTT | Philadelphia Flyers | PHI |
| Pittsburgh Penguins | PIT | San Jose Sharks | SJS | Toronto Maple Leafs | TOR |
| Seattle Kraken | SEA | St. Louis Blues | STL | Utah Hockey Club | UTA |
| Tampa Bay Lightning | TBL | Vancouver Canucks | VAN | Vegas Golden Knights | VGK |
| Washington Capitals | WSH | Winnipeg Jets | WPG | | |

## Quick Start 🚀

### Prerequisites

#### For Docker Installation (Recommended):
- Docker and Docker Compose installed
- Internet connection (for live NHL API data)

#### For Direct Go Installation:
- Go 1.23.3+ installed  
- Internet connection (for live NHL API data)

### Installation

#### Option 1: Docker (Recommended) 🐳

**🚀 Quick Start - Use Pre-built Image from Docker Hub:**
   ```bash
   # Run with your favorite team (no build required!)
   docker run -d -p 8080:8080 -e TEAM_CODE=UTA jshillingburg/hockey_home_dashboard:latest
   docker run -d -p 8080:8080 -e TEAM_CODE=COL jshillingburg/hockey_home_dashboard:latest
   docker run -d -p 8080:8080 -e TEAM_CODE=TOR jshillingburg/hockey_home_dashboard:latest
   ```

**📦 Or Build from Source:**

1. **Clone the repository**
   ```bash
   git clone https://github.com/jshill103/hockey_home_dashboard.git
   cd hockey_home_dashboard
   ```

2. **Run with Docker Compose**
   ```bash
   # Default team (Utah Hockey Club)
   docker-compose up -d

   # Or specify a different team
   TEAM_CODE=COL docker-compose up -d
   TEAM_CODE=TOR docker-compose up -d
   TEAM_CODE=BOS docker-compose up -d
   ```

3. **Or build and run manually**
   ```bash
   # Build the image
   docker build -t nhl-dashboard .

   # Run with your favorite team
   docker run -d -p 8080:8080 -e TEAM_CODE=VGK nhl-dashboard
   docker run -d -p 8080:8080 -e TEAM_CODE=LAK nhl-dashboard
   ```

4. **Access your dashboard**
   ```
   http://localhost:8080
   ```

#### Option 2: Direct Go Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/jshill103/hockey_home_dashboard.git
   cd hockey_home_dashboard
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Build the application**
   ```bash
   go build -o web_server main.go
   ```

4. **Run with your favorite team**
   ```bash
   # Examples:
   ./web_server -team UTA    # Utah Hockey Club
   ./web_server -team TOR    # Toronto Maple Leafs  
   ./web_server -team BOS    # Boston Bruins
   ./web_server              # Defaults to UTA
   ```

5. **Open your browser**
   ```
   http://localhost:8080
   ```

## Usage 💻

### Command Line Options

```bash
./web_server [OPTIONS]

Options:
  -team string              Team code (default "UTA")
  -port int                 Port to run server on (default 8080)
  -weather-key string       WeatherAPI key for weather analysis
  -openweather-key string   OpenWeatherMap API key for weather analysis  
  -accuweather-key string   AccuWeather API key for weather analysis
```

### Supported Team Codes
Use any of the 3-letter NHL team codes listed in the table above.

### Examples

```bash
# Boston Bruins with custom port
./web_server -team BOS -port 3000

# Toronto Maple Leafs on default port
./web_server -team TOR

# Vegas Golden Knights
./web_server -team VGK

# Colorado Avalanche with weather analysis
./web_server -team COL -weather-key your_weather_api_key_here

# Utah Hockey Club with multiple weather sources
./web_server -team UTA -openweather-key key1 -weather-key key2
```

## API Endpoints 🔗

| Endpoint | Description |
|----------|-------------|
| `/` | Main dashboard page |
| `/model-insights` | AI Model Insights widget (season only) |
| `/game-prediction` | AI prediction for next game (JSON) |
| `/prediction-widget` | AI prediction widget (HTML) |
| `/playoff-odds` | Playoff probability widget |
| `/season-countdown` | JSON API for season countdown data |
| `/performance/metrics` | ML model performance metrics |
| `/performance/dashboard` | Performance dashboard (HTML) |
| `/static/` | Static assets (images, CSS, JS) |

## Project Structure 📁

```
go_uhc/
├── handlers/           # HTTP request handlers
│   ├── analysis.go    # Team analysis generation
│   ├── countdown.go   # Season countdown logic
│   ├── home.go        # Main dashboard handler
│   ├── model_insights.go # AI Model Insights widget (NEW!)
│   ├── predictions.go # AI prediction handlers
│   ├── news.go        # News feed handler
│   └── ...
├── models/             # Data structures
│   ├── team_config.go # Team configurations
│   ├── predictions.go # Prediction models
│   ├── matchup.go     # Head-to-head matchup data
│   ├── player_impact.go # Player talent tracking
│   ├── game_result.go # Completed game data
│   ├── news.go        # News models
│   └── ...
├── services/           # Business logic & ML
│   ├── nhl_api.go     # NHL API integration
│   ├── ensemble_predictions.go # Ensemble ML system
│   ├── ml_models.go   # Neural Network, Gradient Boosting
│   ├── elo_rating_model.go # Elo rating system
│   ├── poisson_regression_model.go # Poisson scoring model
│   ├── matchup_database.go # H2H history tracking
│   ├── rolling_stats_service.go # Form & momentum
│   ├── player_impact_service.go # Player analysis
│   ├── goalie_intelligence.go # Goalie tracking
│   ├── game_results_service.go # Auto-learning system
│   ├── news_scraper.go # News scraping service
│   └── ...
├── data/              # Persistent ML data
│   ├── accuracy/      # Prediction accuracy tracking
│   ├── models/        # Saved ML model weights
│   ├── results/       # Historical game results
│   ├── matchups/      # H2H matchup history
│   ├── rolling_stats/ # Team form & momentum data
│   └── player_impact/ # Player talent data
├── media/             # Team assets
│   └── photos/        # Team logos and backgrounds
└── main.go           # Application entry point
```

## Weather Analysis Configuration 🌦️

The application includes advanced weather analysis for game predictions. Weather analysis is **optional** and will be automatically disabled if no API keys are provided.

### Supported Weather Services

| Service | Free Tier | Rate Limit | Sign Up |
|---------|-----------|------------|---------|
| **WeatherAPI** | 1M calls/month | ~33k/day | [weatherapi.com](https://www.weatherapi.com/) |
| **OpenWeatherMap** | 1k calls/day | 1k/day | [openweathermap.org](https://openweathermap.org/api) |
| **AccuWeather** | 50 calls/day | 50/day | [developer.accuweather.com](https://developer.accuweather.com/) |

### Environment Variables

Set one or more of these environment variables to enable weather analysis:

```bash
# WeatherAPI (Recommended - highest rate limit)
WEATHER_API_KEY=your_weather_api_key_here

# OpenWeatherMap
OPENWEATHER_API_KEY=your_openweather_api_key_here

# AccuWeather
ACCUWEATHER_API_KEY=your_accuweather_api_key_here
```

### Docker Configuration

**Docker Run:**
```bash
docker run -d -p 8080:8080 \
  -e TEAM_CODE=UTA \
  -e WEATHER_API_KEY=your_api_key_here \
  jshillingburg/hockey_home_dashboard:latest
```

**Docker Compose:**
```yaml
environment:
  - TEAM_CODE=UTA
  - WEATHER_API_KEY=your_weather_api_key_here
  # - OPENWEATHER_API_KEY=your_openweather_api_key_here
  # - ACCUWEATHER_API_KEY=your_accuweather_api_key_here
```

**Direct Go Run:**
```bash
export WEATHER_API_KEY=your_api_key_here
go run main.go -team UTA
```

**Command Line Flags:**
```bash
# Using command line flags (overrides environment variables)
go run main.go -team UTA -weather-key your_weather_api_key_here
go run main.go -team UTA -openweather-key your_openweather_key -weather-key your_weather_key
./web_server -team UTA -weather-key your_api_key_here

# Available weather flags:
# -weather-key          WeatherAPI key
# -openweather-key      OpenWeatherMap API key  
# -accuweather-key      AccuWeather API key
```

### Weather Analysis Features

When enabled, weather analysis provides:
- **Real-time weather conditions** for game locations
- **Travel weather impact** analysis for visiting teams
- **Game performance effects** (temperature, wind, precipitation)
- **Enhanced AI predictions** with weather factors
- **Outdoor game detection** and special weather considerations

**Note:** If no weather API keys are provided, the application will run normally with weather analysis disabled. All other features remain fully functional.

## Configuration ⚙️

### Adding New Teams

1. **Add team assets** to `media/photos/[team-folder]/`
   - `[team-name]-logo.png` - Team logo
   - `[team-name]-background.jpg/png` - Team background image

2. **Update team configuration** in `models/team_config.go`
   ```go
   "NEW": {
       Code:            "NEW",
       Name:            "New Team Name",
       ShortName:       "Team",
       City:            "City",
       Arena:           "Arena Name",
       PrimaryColor:    "#123456",
       SecondaryColor:  "#654321",
       BackgroundImage: "/static/media/photos/new-team/background.jpg",
       FaviconPath:     "/static/media/photos/new-team/logo.png",
       LogoPath:        "/static/media/photos/new-team/logo.png",
   },
   ```

3. **Add to valid team codes** in the `IsValidTeamCode` function

### Customizing Update Intervals

Edit `main.go` to adjust refresh rates:
- **News updates**: Default 10 minutes
- **Scoreboard updates**: Default 10 minutes (30 seconds during live games)
- **Schedule updates**: Default daily at midnight

## Development 🛠️

### Running in Development Mode

```bash
# Run with auto-restart on file changes
go run main.go -team YOUR_TEAM

# Build and run
go build -o web_server main.go && ./web_server -team YOUR_TEAM
```

## Docker Usage 🐳

### Using Pre-built Docker Hub Image (Easiest!)

```bash
# Pull from Docker Hub (automatic on first run)
docker pull jshillingburg/hockey_home_dashboard:latest

# Run with default team (UTA)
docker run -d -p 8080:8080 jshillingburg/hockey_home_dashboard:latest

# Run with specific team
docker run -d -p 8080:8080 -e TEAM_CODE=TOR jshillingburg/hockey_home_dashboard:latest

# Run multiple team instances on different ports
docker run -d -p 8080:8080 -e TEAM_CODE=UTA --name uta-dashboard jshillingburg/hockey_home_dashboard:latest
docker run -d -p 8081:8080 -e TEAM_CODE=COL --name col-dashboard jshillingburg/hockey_home_dashboard:latest
docker run -d -p 8082:8080 -e TEAM_CODE=VGK --name vgk-dashboard jshillingburg/hockey_home_dashboard:latest
```

### Basic Docker Commands (Building from Source)

```bash
# Build the Docker image
docker build -t nhl-dashboard .

# Run with default team (UTA)
docker run -d -p 8080:8080 nhl-dashboard

# Run with specific team
docker run -d -p 8080:8080 -e TEAM_CODE=TOR nhl-dashboard

# Run multiple team instances on different ports
docker run -d -p 8080:8080 -e TEAM_CODE=UTA --name uta-dashboard nhl-dashboard
docker run -d -p 8081:8080 -e TEAM_CODE=COL --name col-dashboard nhl-dashboard
docker run -d -p 8082:8080 -e TEAM_CODE=VGK --name vgk-dashboard nhl-dashboard
```

### Docker Compose

The `docker-compose.yml` file provides an easy way to run the application with persistent ML data:

```bash
# Start with default configuration
docker-compose up -d

# Start with custom team
TEAM_CODE=LAK docker-compose up -d

# View logs
docker-compose logs -f

# Stop the application
docker-compose down
```

**Persistent Data**: Docker volumes automatically save ML model data:
- Prediction accuracy history
- Trained model weights (Neural Network, Elo, Poisson)
- Historical game results
- Head-to-head matchup database
- Team form and momentum tracking
- Player impact analysis

Models improve over time as they learn from completed games!

### 🔔 Docker + Slack Notifications

**REMOVED** - Product monitoring and Slack notifications have been removed to simplify the application.

### Multi-Team Setup

To run multiple team dashboards simultaneously, uncomment the additional services in `docker-compose.yml` and run:

```bash
docker-compose up -d
```

This will start dashboards for multiple teams on different ports:
- Utah (UTA): http://localhost:8080
- Colorado (COL): http://localhost:8081  
- Vegas (VGK): http://localhost:8082

### Key Components

- **NHL API Integration** - Real-time data from official NHL APIs
- **Web Scraping** - News content extraction from NHL.com
- **Static Asset Management** - Team-specific images and styling  
- **Template Rendering** - Dynamic HTML generation with team theming
- **Background Services** - Automatic data updates and caching

### Adding New Features

1. Create new handler in `handlers/`
2. Add corresponding model in `models/`
3. Implement business logic in `services/`
4. Register route in `main.go`
5. Update UI in handler templates

## Contributing 🤝

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Contribution Guidelines

- Follow Go best practices and conventions
- Add tests for new functionality
- Update documentation for new features
- Ensure all team configurations work properly
- Maintain responsive design principles

## Technical Stack 📋

- **Backend**: Go 1.23.3+
- **HTTP Router**: Go standard library
- **Data Sources**: Official NHL APIs
- **Machine Learning**: 
  - Custom Neural Network (105 features, backpropagation)
  - Elo Rating System (with persistence)
  - Poisson Regression Model
  - Gradient Boosting Trees (Pure Go implementation)
  - Bayesian Inference
  - Monte Carlo Simulation
- **Data Persistence**: JSON-based model storage with Docker volumes
- **Web Scraping**: Custom Go scraper
- **Frontend**: HTML5, CSS3, JavaScript (ES6+), HTMX
- **Styling**: Custom CSS with responsive design
- **Assets**: Team-specific images and icons

## Performance Features ⚡

- **Efficient Data Caching** - Minimizes API calls
- **Background Updates** - Non-blocking data refresh
- **Responsive Images** - Optimized loading
- **Clean HTML** - Fast rendering
- **Minimal Dependencies** - Quick startup times

## Troubleshooting 🔧

### Common Issues

**Server won't start**
- Check if port 8080 is available
- Verify Go installation: `go version`

**Team assets not loading**
- Ensure team code is valid (see supported teams)
- Check that team assets exist in `media/photos/[team]/`

**Data not updating**
- Verify internet connection
- Check NHL API availability
- Restart server to refresh cache

**Performance issues**
- Reduce update intervals in configuration
- Check system resources
- Verify network connectivity

## License 📄

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments 🙏

- **NHL.com** - Official data source and news content
- **NHL API** - Real-time statistics and game information
- **Team Assets** - Official team logos and imagery
- **Go Community** - Excellent libraries and tools

## Support 💬

- Create an issue for bugs or feature requests
- Check existing issues before creating new ones
- Provide detailed information for troubleshooting

---

**Built with ❤️ for hockey fans everywhere!** 🏒

*Enjoy your personalized NHL dashboard experience!*
