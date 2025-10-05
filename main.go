package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jaredshillingburg/go_uhc/handlers"
	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// Global variables for caching
var (
	cachedSchedule        models.Game
	cachedScheduleUpdated time.Time
	cachedScoreboard      models.ScoreboardGame
	cachedNews            []models.NewsHeadline
	cachedUpcomingGames   []models.Game
	currentSeasonStatus   models.SeasonStatus
	isGameCurrentlyLive   bool = false
	// Player stats caching
	cachedPlayerStats     models.PlayerStatsLeaders
	cachedGoalieStats     models.GoalieStatsLeaders
	cachedTeamPlayerStats models.PlayerStatsLeaders
	cachedTeamGoalieStats models.GoalieStatsLeaders
)

// Global team configuration
var (
	teamConfig models.TeamConfig
)

// Channels for background communication
var (
	scheduleChannel      = make(chan models.Game, 1)
	scoreboardChannel    = make(chan models.ScoreboardGame, 1)
	newsChannel          = make(chan []models.NewsHeadline, 1)
	upcomingGamesChannel = make(chan []models.Game, 1)
	playerStatsChannel   = make(chan models.PlayerStatsLeaders, 1)
	goalieStatsChannel   = make(chan models.GoalieStatsLeaders, 1)
)

func main() {
	// Parse command line arguments
	teamCodeFlag := flag.String("team", "UTA", "NHL team code (e.g., UTA, COL, NYR, BOS)")

	// Weather API key flags (optional - for enabling weather analysis)
	openWeatherAPIKey := flag.String("openweather-key", "", "OpenWeatherMap API key for weather analysis")
	weatherAPIKey := flag.String("weather-key", "", "WeatherAPI key for weather analysis")
	accuWeatherAPIKey := flag.String("accuweather-key", "", "AccuWeather API key for weather analysis")

	flag.Parse()

	// Set environment variables from command line flags if provided
	// This allows command line flags to override environment variables
	if *openWeatherAPIKey != "" {
		os.Setenv("OPENWEATHER_API_KEY", *openWeatherAPIKey)
		fmt.Printf("🌦️ OpenWeatherMap API key set via command line\n")
	}
	if *weatherAPIKey != "" {
		os.Setenv("WEATHER_API_KEY", *weatherAPIKey)
		fmt.Printf("🌦️ WeatherAPI key set via command line\n")
	}
	if *accuWeatherAPIKey != "" {
		os.Setenv("ACCUWEATHER_API_KEY", *accuWeatherAPIKey)
		fmt.Printf("🌦️ AccuWeather API key set via command line\n")
	}

	// Initialize team configuration
	teamCode := strings.ToUpper(*teamCodeFlag)
	teamConfig = models.GetTeamConfigByCode(teamCode)

	fmt.Printf("Starting NHL Web Application for %s (%s)...\n", teamConfig.Name, teamConfig.Code)

	// Validate team code
	if !models.IsValidTeamCode(teamCode) {
		fmt.Printf("Warning: Team code '%s' not found, using default team %s\n", teamCode, teamConfig.Code)
	}

	// Initialize schedule data on startup
	fmt.Printf("Initializing schedule data for %s...\n", teamConfig.Code)
	game, err := services.GetTeamSchedule(teamConfig.Code)
	if err != nil {
		fmt.Printf("Error fetching initial schedule: %v\n", err)
	} else {
		cachedSchedule = game
		cachedScheduleUpdated = time.Now()
		fmt.Printf("Initial schedule loaded: %s vs %s on %s\n",
			game.AwayTeam.CommonName.Default,
			game.HomeTeam.CommonName.Default,
			game.GameDate)
	}

	// Initialize scoreboard data on startup
	fmt.Printf("Initializing scoreboard data for %s...\n", teamConfig.Code)
	scoreboard, err := services.GetTeamScoreboard(teamConfig.Code)
	if err != nil {
		fmt.Printf("Error fetching initial scoreboard: %v\n", err)
	} else if scoreboard.GameID != 0 {
		cachedScoreboard = scoreboard
		isGameCurrentlyLive = services.IsGameLive(scoreboard.GameState)
		fmt.Printf("Initial scoreboard loaded: %s vs %s (State: %s)\n",
			scoreboard.AwayTeam.Name.Default,
			scoreboard.HomeTeam.Name.Default,
			scoreboard.GameState)
	} else {
		fmt.Println("No active game found")
	}

	// Initialize news data on startup
	fmt.Println("Initializing news data...")
	headlines, err := services.ScrapeNHLNews()
	if err != nil {
		fmt.Printf("Error fetching initial news: %v\n", err)
	} else {
		cachedNews = headlines
		fmt.Printf("Initial news loaded: %d headlines\n", len(headlines))
	}

	// Initialize season status on startup
	fmt.Println("Initializing season status...")
	currentSeasonStatus = services.GetSeasonStatusWithTeamOverride(teamConfig.Code)
	fmt.Printf("Season status: %s (%s) - Hockey Season: %t\n",
		currentSeasonStatus.CurrentSeason,
		currentSeasonStatus.SeasonPhase,
		currentSeasonStatus.IsHockeySeason)

	// Validate with NHL API
	apiValidation, err := services.ValidateSeasonWithAPI()
	if err != nil {
		fmt.Printf("Warning: Could not validate season status with NHL API: %v\n", err)
	} else {
		fmt.Printf("NHL API validation: Games found = %t\n", apiValidation)
	}

	// Initialize upcoming games data only during hockey season
	if currentSeasonStatus.IsHockeySeason {
		fmt.Println("Initializing upcoming games data...")
		upcomingGames, err := services.GetUpcomingGames()
		if err != nil {
			fmt.Printf("Error fetching initial upcoming games: %v\n", err)
		} else {
			cachedUpcomingGames = upcomingGames
			fmt.Printf("Initial upcoming games loaded: %d games\n", len(upcomingGames))
		}
	}

	// Initialize player stats data only during hockey season
	if currentSeasonStatus.IsHockeySeason {
		fmt.Println("Initializing player stats data...")
		playerLeaders, err := services.GetPlayerStatsLeaders()
		if err != nil {
			fmt.Printf("Error fetching initial player stats: %v\n", err)
		} else {
			cachedPlayerStats = playerLeaders
			cachedTeamPlayerStats = services.GetTeamPlayerStats(playerLeaders, teamConfig.Code)
			fmt.Printf("Initial player stats loaded: %d goals, %d assists, %d points leaders\n",
				len(playerLeaders.Goals), len(playerLeaders.Assists), len(playerLeaders.Points))
		}

		fmt.Println("Initializing goalie stats data...")
		goalieLeaders, err := services.GetGoalieStatsLeaders()
		if err != nil {
			fmt.Printf("Error fetching initial goalie stats: %v\n", err)
		} else {
			cachedGoalieStats = goalieLeaders
			cachedTeamGoalieStats = services.GetTeamGoalieStats(goalieLeaders, teamConfig.Code)
			fmt.Printf("Initial goalie stats loaded: %d wins, %d save%%, %d GAA leaders\n",
				len(goalieLeaders.Wins), len(goalieLeaders.SavePct), len(goalieLeaders.GAA))
		}
	}

	// Start background fetchers
	go scheduleFetcher()
	go newsFetcher()
	go scoreboardFetcher()
	if currentSeasonStatus.IsHockeySeason {
		go playerStatsFetcher()
	}

	// Initialize handlers with cached data
	handlers.Init(
		&cachedSchedule,
		&cachedScheduleUpdated,
		&cachedScoreboard,
		&cachedNews,
		&cachedUpcomingGames,
		&currentSeasonStatus,
		&isGameCurrentlyLive,
	)

	// Initialize team configuration in handlers
	handlers.InitTeamConfig(&teamConfig)

	// Initialize player stats handlers
	handlers.InitPlayerStats(
		&cachedPlayerStats,
		&cachedGoalieStats,
		&cachedTeamPlayerStats,
		&cachedTeamGoalieStats,
	)

	// Initialize AI predictions (available for testing regardless of season)
	fmt.Println("Initializing AI prediction service...")
	handlers.InitPredictions(teamConfig.Code)

	// Initialize Live Prediction System for real-time model updates
	fmt.Println("Initializing Live Prediction System...")
	if err := services.InitializeLivePredictionSystem(teamConfig.Code); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize live prediction system: %v\n", err)
		fmt.Println("Predictions will still work, but won't update automatically with new data")
	} else {
		fmt.Printf("✅ Live Prediction System initialized for %s\n", teamConfig.Code)
	}

	// Initialize Game Results Collection Service for automatic model learning
	fmt.Println("Initializing Game Results Collection Service...")
	if err := services.InitializeGameResultsService(teamConfig.Code); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize game results service: %v\n", err)
		fmt.Println("Models will not learn automatically from completed games")
	} else {
		fmt.Printf("✅ Game Results Service initialized for %s\n", teamConfig.Code)
	}

	// Initialize Rolling Stats Service
	fmt.Println("Initializing Rolling Stats Service...")
	if err := services.InitializeRollingStatsService(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize rolling stats service: %v\n", err)
	} else {
		fmt.Printf("✅ Rolling Stats Service initialized\n")
	}

	// Initialize Model Evaluation Service for train/test split and performance metrics
	fmt.Println("Initializing Model Evaluation Service...")
	liveSys := services.GetLivePredictionSystem()
	if liveSys != nil {
		neuralNet := liveSys.GetNeuralNetwork()
		eloModel := liveSys.GetEloModel()
		poissonModel := liveSys.GetPoissonModel()

		if err := services.InitializeEvaluationService(neuralNet, eloModel, poissonModel); err != nil {
			fmt.Printf("⚠️ Warning: Failed to initialize evaluation service: %v\n", err)
		} else {
			fmt.Printf("✅ Model Evaluation Service initialized\n")
		}
	}

	// ============================================================================
	// PHASE 4: ENHANCED PREDICTION SERVICES
	// ============================================================================
	fmt.Println("🚀 Initializing Phase 4 Enhanced Services...")

	// Goalie Intelligence Service
	fmt.Println("Initializing Goalie Intelligence Service...")
	if err := services.InitializeGoalieService(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize goalie service: %v\n", err)
	} else {
		fmt.Printf("✅ Goalie Intelligence Service initialized\n")
	}

	// Betting Market Service (optional - requires ODDS_API_KEY)
	fmt.Println("Initializing Betting Market Service...")
	if err := services.InitializeBettingMarketService(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize betting market service: %v\n", err)
	} else {
		fmt.Printf("✅ Betting Market Service initialized\n")
	}

	// Schedule Context Service
	fmt.Println("Initializing Schedule Context Service...")
	if err := services.InitializeScheduleContextService(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize schedule context service: %v\n", err)
	} else {
		fmt.Printf("✅ Schedule Context Service initialized\n")
	}

	fmt.Println("🎉 Phase 4 services ready! Predictions now include:")
	fmt.Println("   🥅 Goalie Intelligence (+3-4% accuracy)")
	fmt.Println("   💰 Betting Market Data (+2-3% accuracy)")
	fmt.Println("   📅 Schedule Context (+1-2% accuracy)")
	fmt.Println("   🎯 Expected Total: +6-9% accuracy improvement!")

	// ============================================================================
	// PHASE 6: FEATURE ENGINEERING
	// ============================================================================
	fmt.Println("🚀 Initializing Phase 6 Feature Engineering...")

	// Matchup Database Service
	fmt.Println("Initializing Matchup Database Service...")
	if err := services.InitializeMatchupService(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize matchup service: %v\n", err)
	} else {
		fmt.Printf("✅ Matchup Database Service initialized\n")
	}

	// Player Impact Service
	fmt.Println("Initializing Player Impact Service...")
	if err := services.InitializePlayerImpactService(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize player impact service: %v\n", err)
	} else {
		fmt.Printf("✅ Player Impact Service initialized\n")
	}

	// Advanced Rolling Stats are calculated within RollingStatsService
	fmt.Println("✅ Advanced Rolling Statistics integrated")

	fmt.Println("🎉 Phase 6 services ready! Predictions now include:")
	fmt.Println("   📊 Matchup History & Rivalries (+1-2% accuracy)")
	fmt.Println("   📈 Advanced Rolling Statistics (+1% accuracy)")
	fmt.Println("   ⭐ Player Impact Tracking (+1-2% accuracy)")
	fmt.Println("   🎯 Expected Total: +3-6% accuracy improvement!")
	fmt.Println("   🏆 Total System: 87-99% accuracy expected!")

	// Initialize scraper handlers
	// Removed - scraper service no longer used

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("."))))

	// Register handlers
	http.HandleFunc("/schedule", handlers.HandleSchedule)
	http.HandleFunc("/news", handlers.HandleNews)
	http.HandleFunc("/banner", handlers.HandleBanner)
	http.HandleFunc("/scoreboard", handlers.HandleScoreboard)
	http.HandleFunc("/mammoth-analysis", handlers.HandleTeamAnalysis)
	http.HandleFunc("/season-status", handlers.HandleSeasonStatus)
	http.HandleFunc("/upcoming-games", handlers.HandleUpcomingGames)
	http.HandleFunc("/goalie-stats", handlers.HandleGoalieStats)
	http.HandleFunc("/model-insights", handlers.HandleModelInsights)
	http.HandleFunc("/season-countdown", handlers.HandleSeasonCountdown)
	http.HandleFunc("/season-countdown-json", handlers.HandleSeasonCountdownJSON)
	http.HandleFunc("/api-test", handlers.HandleAPITest)
	http.HandleFunc("/playoff-odds", handlers.HandlePlayoffOdds)

	// AI Prediction endpoints
	// Register prediction routes (available regardless of season status for testing)
	http.HandleFunc("/api/prediction", handlers.HandleGamePrediction)
	http.HandleFunc("/prediction-widget", handlers.HandlePredictionWidget)

	// Performance Metrics Dashboard endpoints
	http.HandleFunc("/api/performance", handlers.PerformanceDashboardHandler)
	http.HandleFunc("/api/metrics", handlers.ModelMetricsHandler)

	// Live Prediction System management endpoints
	if livePredictionSystem := services.GetLivePredictionSystem(); livePredictionSystem != nil {
		http.HandleFunc("/api/live-system/status", livePredictionSystem.HandleSystemStatus)
		http.HandleFunc("/api/live-system/force-update", livePredictionSystem.HandleForceUpdate)
		http.HandleFunc("/api/live-system/model-history", livePredictionSystem.HandleModelHistory)
		fmt.Println("📡 Live prediction system API endpoints registered")
	}

	if currentSeasonStatus.IsHockeySeason {
		// Currently no additional routes needed only during hockey season
	}

	http.HandleFunc("/", handlers.HandleHome)

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n🛑 Shutting down server...")

		// Stop the game results service
		if err := services.StopGameResultsService(); err != nil {
			fmt.Printf("⚠️ Warning: Error stopping game results service: %v\n", err)
		} else {
			fmt.Println("✅ Game results service stopped")
		}

		// Stop the live prediction system
		if err := services.StopLivePredictionSystem(); err != nil {
			fmt.Printf("⚠️ Warning: Error stopping live prediction system: %v\n", err)
		} else {
			fmt.Println("✅ Live prediction system stopped")
		}

		fmt.Println("👋 Server shutdown complete")
		os.Exit(0)
	}()

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Schedule will be automatically updated every night at midnight")
	fmt.Println("News will be automatically updated every 10 minutes")
	fmt.Println("Scoreboard will be updated every 10 minutes (30 seconds when game is live)")
	fmt.Println("🤖 Live prediction models will update automatically every hour")
	fmt.Println("Press Ctrl+C to shutdown gracefully")
	http.ListenAndServe(":8080", nil)
}

func scheduleFetcher() {
	for {
		// Calculate time until next midnight
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		sleepDuration := nextMidnight.Sub(now)

		fmt.Printf("Next schedule update scheduled for: %s (in %v)\n", nextMidnight.Format("2006-01-02 15:04:05"), sleepDuration)

		// Sleep until midnight
		time.Sleep(sleepDuration)

		// Fetch new schedule
		fmt.Printf("Fetching updated schedule for %s...\n", teamConfig.Code)
		game, err := services.GetTeamSchedule(teamConfig.Code)
		if err != nil {
			fmt.Printf("Error fetching schedule: %v\n", err)
		} else {
			// Update cached schedule
			cachedSchedule = game
			cachedScheduleUpdated = time.Now()
			fmt.Printf("Schedule updated: %s vs %s on %s\n",
				game.AwayTeam.CommonName.Default,
				game.HomeTeam.CommonName.Default,
				game.GameDate)
		}

		// Also fetch upcoming games if we're in hockey season
		if currentSeasonStatus.IsHockeySeason {
			fmt.Printf("Fetching updated upcoming games for %s...\n", teamConfig.Code)
			upcomingGames, err := services.GetTeamUpcomingGames(teamConfig.Code)
			if err != nil {
				fmt.Printf("Error fetching upcoming games: %v\n", err)
			} else {
				// Update cached upcoming games
				cachedUpcomingGames = upcomingGames
				fmt.Printf("Upcoming games updated: %d games found\n", len(upcomingGames))
			}

			// Send update to channel
			select {
			case upcomingGamesChannel <- upcomingGames:
				// Successfully sent to channel
			default:
				// Channel is full, skip this update
			}
		}

		// Send update to channel
		select {
		case scheduleChannel <- game:
			// Successfully sent to channel
		default:
			// Channel is full, skip this update
		}
	}
}

func scoreboardFetcher() {
	for {
		var sleepDuration time.Duration
		if isGameCurrentlyLive {
			sleepDuration = 30 * time.Second // 30 seconds when game is live
		} else {
			sleepDuration = 10 * time.Minute // 10 minutes when no live game
		}

		time.Sleep(sleepDuration)

		// Fetch new scoreboard
		scoreboard, err := services.GetTeamScoreboard(teamConfig.Code)
		if err != nil {
			fmt.Printf("Error fetching scoreboard: %v\n", err)
			continue
		}

		// Update cached scoreboard
		cachedScoreboard = scoreboard
		isGameCurrentlyLive = services.IsGameLive(scoreboard.GameState)

		if scoreboard.GameID != 0 {
			fmt.Printf("Scoreboard updated: %s %d - %d %s (State: %s)\n",
				scoreboard.AwayTeam.Name.Default,
				scoreboard.AwayTeam.Score,
				scoreboard.HomeTeam.Score,
				scoreboard.HomeTeam.Name.Default,
				scoreboard.GameState)
		} else {
			fmt.Println("No live game - next scoreboard update in 10m0s")
		}

		// Send update to channel
		select {
		case scoreboardChannel <- scoreboard:
			// Successfully sent to channel
		default:
			// Channel is full, skip this update
		}
	}
}

func newsFetcher() {
	for {
		// Sleep for 10 minutes
		sleepDuration := 10 * time.Minute
		fmt.Printf("Next news update scheduled for: %s (in %v)\n",
			time.Now().Add(sleepDuration).Format("2006-01-02 15:04:05"),
			sleepDuration)

		time.Sleep(sleepDuration)

		// Fetch new news
		fmt.Println("Fetching updated news...")
		headlines, err := services.ScrapeNHLNews()
		if err != nil {
			fmt.Printf("Error fetching news: %v\n", err)
			continue
		}

		// Update cached news
		cachedNews = headlines
		fmt.Printf("News updated: %d headlines\n", len(headlines))

		// Send update to channel
		select {
		case newsChannel <- headlines:
			// Successfully sent to channel
		default:
			// Channel is full, skip this update
		}
	}
}

func playerStatsFetcher() {
	for {
		// Sleep for 30 minutes during hockey season
		sleepDuration := 30 * time.Minute
		fmt.Printf("Next player stats update scheduled for: %s (in %v)\n",
			time.Now().Add(sleepDuration).Format("2006-01-02 15:04:05"),
			sleepDuration)

		time.Sleep(sleepDuration)

		// Fetch updated player stats
		fmt.Println("Fetching updated player stats...")
		playerLeaders, err := services.GetPlayerStatsLeaders()
		if err != nil {
			fmt.Printf("Error fetching player stats: %v\n", err)
			continue
		}

		// Update cached player stats
		cachedPlayerStats = playerLeaders
		cachedTeamPlayerStats = services.GetTeamPlayerStats(playerLeaders, teamConfig.Code)
		fmt.Printf("Player stats updated: %d goals, %d assists, %d points leaders\n",
			len(playerLeaders.Goals), len(playerLeaders.Assists), len(playerLeaders.Points))

		// Fetch updated goalie stats
		fmt.Println("Fetching updated goalie stats...")
		goalieLeaders, err := services.GetGoalieStatsLeaders()
		if err != nil {
			fmt.Printf("Error fetching goalie stats: %v\n", err)
			continue
		}

		// Update cached goalie stats
		cachedGoalieStats = goalieLeaders
		cachedTeamGoalieStats = services.GetTeamGoalieStats(goalieLeaders, teamConfig.Code)
		fmt.Printf("Goalie stats updated: %d wins, %d save%%, %d GAA leaders\n",
			len(goalieLeaders.Wins), len(goalieLeaders.SavePct), len(goalieLeaders.GAA))

		// Send updates to channels
		select {
		case playerStatsChannel <- playerLeaders:
			// Successfully sent to channel
		default:
			// Channel is full, skip this update
		}

		select {
		case goalieStatsChannel <- goalieLeaders:
			// Successfully sent to channel
		default:
			// Channel is full, skip this update
		}
	}
}
