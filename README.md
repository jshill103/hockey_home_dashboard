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
- **Player Statistics** - Key player performance metrics

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
  -team string    Team code (default "UTA")
  -port int       Port to run server on (default 8080)
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
```

## API Endpoints 🔗

| Endpoint | Description |
|----------|-------------|
| `/` | Main dashboard page |
| `/season-countdown` | JSON API for season countdown data |
| `/static/` | Static assets (images, CSS, JS) |

## Project Structure 📁

```
go_uhc/
├── handlers/           # HTTP request handlers
│   ├── analysis.go    # Team analysis generation
│   ├── countdown.go   # Season countdown logic
│   ├── home.go        # Main dashboard handler
│   ├── news.go        # News feed handler
│   └── ...
├── models/             # Data structures
│   ├── team_config.go # Team configurations
│   ├── news.go        # News models
│   └── ...
├── services/           # Business logic
│   ├── nhl_api.go     # NHL API integration
│   ├── news_scraper.go # News scraping service
│   └── ...
├── media/             # Team assets
│   └── photos/        # Team logos and backgrounds
└── main.go           # Application entry point
```

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

The `docker-compose.yml` file provides an easy way to run the application:

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

- **Backend**: Go 1.19+
- **HTTP Router**: Go standard library
- **Data Sources**: Official NHL APIs
- **Web Scraping**: Custom Go scraper
- **Frontend**: HTML5, CSS3, JavaScript (ES6+)
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
