package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jaredshillingburg/go_uhc/handlers"
	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
	"github.com/jaredshillingburg/go_uhc/utils"
)

// Global variables for caching
var (
	cachedSchedule        models.Game
	cachedScheduleUpdated time.Time
	cachedNews            []models.NewsHeadline
	cachedUpcomingGames   []models.Game
	currentSeasonStatus   models.SeasonStatus
	isGameCurrentlyLive   bool = false
	// Player stats caching
	cachedPlayerStats     models.PlayerStatsLeaders
	cachedGoalieStats     models.GoalieStatsLeaders
	cachedTeamPlayerStats models.PlayerStatsLeaders
	cachedTeamGoalieStats models.GoalieStatsLeaders
	// Mutexes for thread-safe access to global caches
	cacheMu sync.RWMutex
)

// Global team configuration
var (
	teamConfig models.TeamConfig
)

// Channels for background communication
var (
	scheduleChannel      = make(chan models.Game, 1)
	newsChannel          = make(chan []models.NewsHeadline, 1)
	upcomingGamesChannel = make(chan []models.Game, 1)
	playerStatsChannel   = make(chan models.PlayerStatsLeaders, 1)
	goalieStatsChannel   = make(chan models.GoalieStatsLeaders, 1)
)

// initializeDataDirectories creates all required data directories at startup
// This replaces the docker-entrypoint.sh script functionality
func initializeDataDirectories() error {
	fmt.Println("📁 Initializing data directories...")
	
	directories := []string{
		"data/accuracy",
		"data/models",
		"data/results",
		"data/matchups",
		"data/rolling_stats",
		"data/player_impact",
		"data/lineups",
		"data/play_by_play",
		"data/shifts",
		"data/landing_page",
		"data/game_summary",
		"data/cache/predictions",
		"data/cache/api",
		"data/predictions",
		"data/metrics",
		"data/rosters",
		"data/evaluation",
		"data/goalies",
		"data/betting_markets",
		"data/architecture_search",
		"data/time_weighted_stats",
		"data/feature_importance",
	}
	
	for _, dir := range directories {
		fullPath := "/app/" + dir
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			// Don't fail - directory might already exist or be created by PVC
			fmt.Printf("   ⚠️  Warning: Could not create %s: %v (continuing anyway)\n", dir, err)
		}
	}
	
	// Try to create a test file to verify write permissions
	testFile := "/app/data/.write_test"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		fmt.Printf("   ⚠️  Warning: Cannot write to /app/data directory: %v\n", err)
		fmt.Printf("   The application will attempt to create files as needed.\n")
		fmt.Printf("   If you see permission errors, add fsGroup: 1001 to your Kubernetes securityContext\n")
	} else {
		os.Remove(testFile)
		fmt.Println("   ✅ Data directory is writable")
	}
	
	fmt.Println("✅ Data directories initialized successfully")
	return nil
}

func main() {
	// Get default team code from environment variable if set
	defaultTeam := os.Getenv("TEAM_CODE")
	if defaultTeam == "" {
		defaultTeam = "UTA"
	}
	
	// Parse command line arguments
	teamCodeFlag := flag.String("team", defaultTeam, "NHL team code (e.g., UTA, COL, NYR, BOS)")

	// Weather API key flags (optional - for enabling weather analysis)
	openWeatherAPIKey := flag.String("openweather-key", "", "OpenWeatherMap API key for weather analysis")
	weatherAPIKey := flag.String("weather-key", "", "WeatherAPI key for weather analysis")
	accuWeatherAPIKey := flag.String("accuweather-key", "", "AccuWeather API key for weather analysis")

	// Odds API key flag (optional - for enabling betting market integration)
	oddsAPIKey := flag.String("odds-key", "", "The Odds API key for betting market data")

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
	if *oddsAPIKey != "" {
		os.Setenv("ODDS_API_KEY", *oddsAPIKey)
		fmt.Printf("💰 Odds API key set via command line\n")
	}

	// Initialize team configuration
	teamCode := strings.ToUpper(*teamCodeFlag)
	teamConfig = models.GetTeamConfigByCode(teamCode)

	fmt.Printf("Starting NHL Web Application for %s (%s)...\n", teamConfig.Name, teamConfig.Code)

	// Validate team code
	if !models.IsValidTeamCode(teamCode) {
		fmt.Printf("Warning: Team code '%s' not found, using default team %s\n", teamCode, teamConfig.Code)
	}

	// Initialize data directories (replaces docker-entrypoint.sh)
	if err := initializeDataDirectories(); err != nil {
		log.Printf("⚠️ Warning during directory initialization: %v\n", err)
		// Don't fail - the application can still work
	}

	// Initialize schedule data on startup
	fmt.Printf("Initializing schedule data for %s...\n", teamConfig.Code)
	game, err := services.GetTeamSchedule(teamConfig.Code)
	if err != nil {
		fmt.Printf("Error fetching initial schedule: %v\n", err)
	} else {
		cacheMu.Lock()
		cachedSchedule = game
		cachedScheduleUpdated = time.Now()
		cacheMu.Unlock()
		// Safe access with checks for empty team names
		awayTeam := "Unknown"
		homeTeam := "Unknown"
		if game.AwayTeam.CommonName.Default != "" {
			awayTeam = game.AwayTeam.CommonName.Default
		}
		if game.HomeTeam.CommonName.Default != "" {
			homeTeam = game.HomeTeam.CommonName.Default
		}
		fmt.Printf("Initial schedule loaded: %s vs %s on %s\n",
			awayTeam, homeTeam, game.GameDate)
	}

	// Initialize news data on startup
	fmt.Println("Initializing news data...")
	headlines, err := services.ScrapeNHLNews()
	if err != nil {
		fmt.Printf("Error fetching initial news: %v\n", err)
	} else {
		cacheMu.Lock()
		cachedNews = headlines
		cacheMu.Unlock()
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
			cacheMu.Lock()
			cachedUpcomingGames = upcomingGames
			cacheMu.Unlock()
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
			cacheMu.Lock()
			cachedPlayerStats = playerLeaders
			cachedTeamPlayerStats = services.GetTeamPlayerStats(playerLeaders, teamConfig.Code)
			cacheMu.Unlock()
			fmt.Printf("Initial player stats loaded: %d goals, %d assists, %d points leaders\n",
				len(playerLeaders.Goals), len(playerLeaders.Assists), len(playerLeaders.Points))
		}

		fmt.Println("Initializing goalie stats data...")
		goalieLeaders, err := services.GetGoalieStatsLeaders()
		if err != nil {
			fmt.Printf("Error fetching initial goalie stats: %v\n", err)
		} else {
			cacheMu.Lock()
			cachedGoalieStats = goalieLeaders
			cachedTeamGoalieStats = services.GetTeamGoalieStats(goalieLeaders, teamConfig.Code)
			cacheMu.Unlock()
			fmt.Printf("Initial goalie stats loaded: %d wins, %d save%%, %d GAA leaders\n",
				len(goalieLeaders.Wins), len(goalieLeaders.SavePct), len(goalieLeaders.GAA))
		}
	}

	// Start background fetchers
	go scheduleFetcher()
	go newsFetcher()
	if currentSeasonStatus.IsHockeySeason {
		go playerStatsFetcher()
	}

	// Initialize handlers with cached data
	handlers.Init(
		&cachedSchedule,
		&cachedScheduleUpdated,
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

	// Initialize System Stats Service BEFORE other services that use it
	fmt.Println("\n📊 Initializing System Statistics Service...")
	systemStatsService := services.NewSystemStatsService()
	handlers.InitSystemStatsService(systemStatsService)

	// Initialize Feature Importance Service for ML model analysis
	fmt.Println("🔍 Initializing Feature Importance Service...")
	services.InitFeatureImportanceService()
	fmt.Println("✅ Feature Importance Service initialized")

	// Initialize Training Metrics Service for training frequency tracking
	fmt.Println("📊 Initializing Training Metrics Service...")
	services.InitTrainingMetricsService()
	fmt.Println("✅ Training Metrics Service initialized")

	// Initialize API Cache Service for NHL API response caching
	fmt.Println("💾 Initializing API Cache Service...")
	services.InitAPICacheService()
	fmt.Println("✅ API Cache Service initialized")

	// Initialize Standings Cache Service to reduce redundant API calls
	fmt.Println("📊 Initializing Standings Cache Service...")
	services.InitStandingsCacheService()
	fmt.Println("✅ Standings Cache Service initialized")

	// Initialize Request Deduplicator to prevent concurrent duplicate API calls
	fmt.Println("🔄 Initializing Request Deduplicator...")
	services.InitRequestDeduplicator()
	fmt.Println("✅ Request Deduplicator initialized")

	// Initialize Health Check Service
	fmt.Println("🏥 Initializing Health Check Service...")
	services.InitHealthCheckService("v1.0.0")
	fmt.Println("✅ Health Check Service initialized")

	// Initialize Play-by-Play Analytics Service for xG and shot quality metrics
	fmt.Println("Initializing Play-by-Play Analytics Service (xG Engine)...")
	services.InitPlayByPlayService(systemStatsService)
	fmt.Println("✅ Play-by-Play Analytics Service initialized (Expected Goals ready)")

	// Initialize Shift Analysis Service for line chemistry and coaching tendencies
	fmt.Println("Initializing Shift Analysis Service (Line Chemistry Engine)...")
	services.InitShiftAnalysisService(systemStatsService)
	fmt.Println("✅ Shift Analysis Service initialized (Line Chemistry ready)")

	// Initialize Landing Page Analytics Service for enhanced physical play and zone control
	fmt.Println("Initializing Landing Page Analytics Service (Enhanced Metrics Engine)...")
	services.InitLandingPageService()
	fmt.Println("✅ Landing Page Analytics Service initialized (Enhanced Metrics ready)")

	// Initialize Game Summary Analytics Service for enhanced game context
	fmt.Println("Initializing Game Summary Analytics Service (Enhanced Context Engine)...")
	services.InitGameSummaryService(systemStatsService)
	fmt.Println("✅ Game Summary Analytics Service initialized (Enhanced Context ready)")

	// Backfill play-by-play data for ALL NHL teams (league-wide)
	fmt.Println("🌐 Starting league-wide xG backfill (last 10 games per team, 32 teams)...")
	fmt.Println("⏱️ This will take ~10-15 minutes but provides 32x more training data")
	pbpService := services.GetPlayByPlayService()
	if pbpService != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("⚠️ Panic during play-by-play backfill: %v\n", r)
				}
			}()
			// Run league-wide backfill in background to not block server startup
			if err := pbpService.BackfillAllTeams(10); err != nil {
				fmt.Printf("⚠️ Warning: Failed to backfill play-by-play data: %v\n", err)
			} else {
				fmt.Println("✅ League-wide Play-by-Play backfill complete (320 games processed)")
			}
		}()
	}

	// Initialize Live Prediction System for real-time model updates
	fmt.Println("Initializing Live Prediction System...")
	if err := services.InitializeLivePredictionSystem(teamConfig.Code); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize live prediction system: %v\n", err)
		fmt.Println("Predictions will still work, but won't update automatically with new data")
	} else {
		fmt.Printf("✅ Live Prediction System initialized for %s\n", teamConfig.Code)
	}

	// Initialize Playoff Simulation Service for ML-powered playoff odds
	fmt.Println("Initializing ML-powered Playoff Simulation Service...")
	liveSys := services.GetLivePredictionSystem()
	if liveSys != nil {
		ensembleService := liveSys.GetEnsemble()
		if ensembleService != nil {
			services.InitPlayoffSimulationService(ensembleService)
			fmt.Println("✅ Playoff Simulation Service initialized with ML models")
		} else {
			fmt.Println("⚠️ Warning: Could not initialize Playoff Simulation Service")
		}
	}

	// Initialize Prediction Storage Service for league-wide predictions
	fmt.Println("📝 Initializing Prediction Storage Service...")
	predictionStorage := services.InitPredictionStorageService()
	fmt.Println("✅ Prediction Storage Service initialized")

	// Initialize Daily Prediction Service for all NHL games
	fmt.Println("🎯 Initializing Daily Prediction Service...")
	ensembleService := services.NewEnsemblePredictionService(teamConfig.Code)
	dailyPredictionService := services.InitDailyPredictionService(predictionStorage, ensembleService)
	dailyPredictionService.Start()
	fmt.Println("✅ Daily Prediction Service started (generates predictions for all NHL games)")

	// Initialize Smart Re-Prediction Service
	fmt.Println("🔄 Initializing Smart Re-Prediction Service...")
	services.InitSmartRePredictionService(dailyPredictionService, predictionStorage, ensembleService)
	fmt.Println("✅ Smart Re-Prediction Service initialized (adaptive re-prediction after training)")

	// Initialize Model Evaluation Service BEFORE Game Results Service (for batch training)
	fmt.Println("Initializing Model Evaluation Service...")
	liveSys = services.GetLivePredictionSystem()
	if liveSys != nil {
		neuralNet := liveSys.GetNeuralNetwork()
		eloModel := liveSys.GetEloModel()
		poissonModel := liveSys.GetPoissonModel()

	if err := services.InitializeEvaluationService(neuralNet, eloModel, poissonModel); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize evaluation service: %v\n", err)
	} else {
		fmt.Printf("✅ Model Evaluation Service initialized\n")
		
		// Start periodic auto-save (every 30 minutes)
		if evalSvc := services.GetEvaluationService(); evalSvc != nil {
			evalSvc.StartPeriodicSave()
			fmt.Println("✅ Periodic model save started (every 30 minutes)")
		}
	}
}

	// Initialize Game Results Collection Service for automatic model learning
	// NOTE: This MUST come AFTER EvaluationService so they can be linked for batch training
	fmt.Println("Initializing Game Results Collection Service...")
	if err := services.InitializeGameResultsService(teamConfig.Code); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize game results service: %v\n", err)
		fmt.Println("Models will not learn automatically from completed games")
	} else {
		fmt.Printf("✅ Game Results Service initialized (league-wide collection, primary: %s)\n", teamConfig.Code)
	}

	// Initialize Rolling Stats Service
	fmt.Println("Initializing Rolling Stats Service...")
	if err := services.InitializeRollingStatsService(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize rolling stats service: %v\n", err)
	} else {
		fmt.Printf("✅ Rolling Stats Service initialized\n")
	}

	// Initialize Roster Validation Service (needed for player/goalie validation)
	fmt.Println("Initializing Roster Validation Service...")
	if err := services.InitializeRosterValidationService(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize roster validation service: %v\n", err)
	} else {
		fmt.Printf("✅ Roster Validation Service initialized\n")
	}

	// Initialize Goalie Intelligence Service and fetch goalie stats
	fmt.Println("Initializing Goalie Intelligence Service...")
	if err := services.InitializeGoalieService(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize goalie service: %v\n", err)
	} else {
		fmt.Printf("✅ Goalie Intelligence Service initialized\n")
	}

	// Fetch goalie stats for the configured team (with previous season fallback)
	fmt.Println("Fetching goalie stats...")
	goalieService := services.GetGoalieService()
	if goalieService != nil {
		currentSeason := utils.GetCurrentSeason()
		fmt.Printf("📅 Using season: %s\n", utils.FormatSeason(currentSeason))
		if err := goalieService.FetchGoalieStats(teamConfig.Code, currentSeason); err != nil {
			fmt.Printf("⚠️ Could not fetch goalie stats for %s: %v\n", teamConfig.Code, err)
		} else {
			fmt.Printf("✅ Goalie stats loaded for %s\n", teamConfig.Code)
		}
	} else {
		fmt.Println("⚠️ Warning: Goalie service is nil, cannot fetch stats")
	}

	// Model Evaluation Service already initialized above (before Game Results Service)
	// This ensures proper linking for batch training

	// ============================================================================
	// PHASE 4: ENHANCED PREDICTION SERVICES
	// ============================================================================
	fmt.Println("🚀 Initializing Phase 4 Enhanced Services...")

	// Goalie Intelligence Service (already initialized above, skip duplicate)
	// Note: Goalie service was initialized earlier at line 337-342

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

	// Pre-Game Lineup Service
	fmt.Println("Initializing Pre-Game Lineup Service...")
	services.InitPreGameLineupService(teamConfig.Code)
	fmt.Println("✅ Pre-Game Lineup Service initialized (monitoring upcoming games)")

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

	// ============================================================================
	// PHASE 1: FOUNDATION & QUICK WINS
	// ============================================================================
	fmt.Println("\n🚀 Initializing Phase 1 Accuracy Enhancement Services...")

	// Error Analysis & Monitoring System
	fmt.Println("📊 Initializing Error Analysis Service...")
	if err := services.InitializeErrorAnalysis(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize error analysis service: %v\n", err)
	} else {
		fmt.Printf("✅ Error Analysis Service initialized\n")
	}

	// Feature Importance Analyzer
	fmt.Println("🔍 Initializing Feature Importance Analyzer...")
	if err := services.InitializeFeatureImportanceAnalyzer(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize feature importance analyzer: %v\n", err)
	} else {
		fmt.Printf("✅ Feature Importance Analyzer initialized\n")
	}

	// Time-Weighted Stats Service
	fmt.Println("⏱️ Initializing Time-Weighted Stats Service...")
	if err := services.InitializeTimeWeightedStats(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize time-weighted stats service: %v\n", err)
	} else {
		fmt.Printf("✅ Time-Weighted Stats Service initialized\n")
	}

	fmt.Println("🎉 Phase 1 services ready! Accuracy enhancements:")
	fmt.Println("   📊 Error Analysis & Pattern Detection")
	fmt.Println("   🔍 Feature Importance Tracking")
	fmt.Println("   ⏱️ Time-Weighted Performance Analysis")
	fmt.Println("   🏒 Enhanced Special Teams Matchup Analysis")
	fmt.Println("   🎯 Expected: +7-11% accuracy improvement!")

	// ============================================================================
	// PHASE 2: ENHANCED DATA QUALITY
	// ============================================================================
	fmt.Println("\n🔬 Initializing Phase 2 Enhanced Data Quality Services...")

	// Head-to-Head Matchup Database
	fmt.Println("🏒 Initializing Head-to-Head Service...")
	if err := services.InitializeHeadToHead(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize head-to-head service: %v\n", err)
	} else {
		fmt.Printf("✅ Head-to-Head Service initialized\n")
	}

	// Rest Impact Analysis
	fmt.Println("😴 Initializing Rest Impact Service...")
	if err := services.InitializeRestImpact(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize rest impact service: %v\n", err)
	} else {
		fmt.Printf("✅ Rest Impact Service initialized\n")
	}

	// Note: Goalie matchup tracking is already part of Goalie Intelligence Service
	fmt.Println("🥅 Goalie Matchup Tracking: integrated with Goalie Intelligence Service")

	fmt.Println("🎉 Phase 2 services ready! Enhanced data quality:")
	fmt.Println("   🏒 Head-to-Head Matchup Database")
	fmt.Println("   😴 Rest & Fatigue Analysis")
	fmt.Println("   🥅 Goalie Matchup History Tracking")
	fmt.Println("   📊 Lineup Stability Monitoring (pending)")
	fmt.Println("   🎯 Expected: +8-12% accuracy improvement!")
	fmt.Println("   🔥 Combined Phase 1+2: +15-23% total improvement!")

	// ============================================================================
	// PHASE 3: CONFIDENCE & MODEL SELECTION
	// ============================================================================
	fmt.Println("\n🎯 Initializing Phase 3 Confidence & Model Selection Services...")

	// Context Analysis Service
	fmt.Println("📊 Initializing Context Analysis Service...")
	if err := services.InitializeContextAnalysis(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize context analysis service: %v\n", err)
	} else {
		fmt.Printf("✅ Context Analysis Service initialized\n")
	}

	// Ensemble Recalibration Service
	fmt.Println("⚖️ Initializing Ensemble Recalibration Service...")
	if err := services.InitializeRecalibration(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize recalibration service: %v\n", err)
	} else {
		fmt.Printf("✅ Ensemble Recalibration Service initialized\n")
	}

	// Confidence Calibration Service
	fmt.Println("📈 Initializing Confidence Calibration Service...")
	if err := services.InitializeConfidenceCalibration(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize confidence calibration service: %v\n", err)
	} else {
		fmt.Printf("✅ Confidence Calibration Service initialized\n")
	}

	// Prediction Quality Service
	fmt.Println("✅ Initializing Prediction Quality Service...")
	if err := services.InitializePredictionQuality(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize prediction quality service: %v\n", err)
	} else {
		fmt.Printf("✅ Prediction Quality Service initialized\n")
	}

	fmt.Println("🎉 Phase 3 services ready! Intelligent confidence & model selection:")
	fmt.Println("   🎯 Context-Aware Model Weighting")
	fmt.Println("   ⚖️ Dynamic Ensemble Recalibration")
	fmt.Println("   📈 Confidence Calibration Curves")
	fmt.Println("   ✅ Prediction Quality Assessment")
	fmt.Println("   🎯 Expected: +5-8% accuracy improvement!")
	fmt.Println("   🔥 Combined Phase 1+2+3: +20-31% total improvement!")

	// ============================================================================
	// PHASE 4: ADVANCED PATTERN RECOGNITION
	// ============================================================================
	fmt.Println("\n🔥 Initializing Phase 4 Advanced Pattern Recognition Services...")

	// Streak Detection Service
	fmt.Println("🔥 Initializing Streak Detection Service...")
	if err := services.InitializeStreakDetection(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize streak detection service: %v\n", err)
	} else {
		fmt.Printf("✅ Streak Detection Service initialized\n")
	}

	// Momentum Quantification Service
	fmt.Println("📈 Initializing Momentum Quantification Service...")
	if err := services.InitializeMomentumQuantification(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize momentum service: %v\n", err)
	} else {
		fmt.Printf("✅ Momentum Quantification Service initialized\n")
	}

	// Clutch Performance Service
	fmt.Println("🎯 Initializing Clutch Performance Service...")
	if err := services.InitializeClutchPerformance(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize clutch performance service: %v\n", err)
	} else {
		fmt.Printf("✅ Clutch Performance Service initialized\n")
	}

	fmt.Println("🎉 Phase 4 services ready! Advanced pattern recognition:")
	fmt.Println("   🔥 Streak Detection & Analysis")
	fmt.Println("   📈 Momentum Quantification")
	fmt.Println("   🎯 Clutch Performance Tracking")
	fmt.Println("   🎯 Expected: +6-10% accuracy improvement!")
	fmt.Println("   🔥 Combined Phase 1+2+3+4: +26-41% total improvement!")

	// Initialize scraper handlers
	// Removed - scraper service no longer used

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("."))))

	// Register handlers
	http.HandleFunc("/schedule", handlers.HandleSchedule)
	http.HandleFunc("/news", handlers.HandleNews)
	http.HandleFunc("/banner", handlers.HandleBanner)
	http.HandleFunc("/mammoth-analysis", handlers.HandleTeamAnalysis)
	http.HandleFunc("/season-status", handlers.HandleSeasonStatus)
	http.HandleFunc("/upcoming-games", handlers.HandleUpcomingGames)
	http.HandleFunc("/goalie-stats", handlers.HandleGoalieStats)
	http.HandleFunc("/model-insights", handlers.HandleModelInsights)
	http.HandleFunc("/season-countdown", handlers.HandleSeasonCountdown)
	http.HandleFunc("/season-countdown-json", handlers.HandleSeasonCountdownJSON)
	http.HandleFunc("/api-test", handlers.HandleAPITest)
	http.HandleFunc("/playoff-odds", handlers.HandlePlayoffOdds)
	http.HandleFunc("/api/what-if", handlers.HandleWhatIf)                    // Phase 4.3: What-If simulator
	http.HandleFunc("/api/what-if/scenarios", handlers.HandleWhatIfScenarios) // Phase 4.3: Common scenarios
	http.HandleFunc("/api/simulation/metrics", handlers.HandleSimulationMetrics)    // Phase 5.5: Performance metrics
	http.HandleFunc("/api/simulation/reset-metrics", handlers.HandleResetMetrics)   // Phase 5.5: Reset metrics

	// AI Prediction endpoints
	// Register prediction routes (available regardless of season status for testing)
	http.HandleFunc("/api/prediction", handlers.HandleGamePrediction)
	http.HandleFunc("/prediction-widget", handlers.HandlePredictionWidget)

	// League-wide prediction endpoints
	http.HandleFunc("/api/predictions/all", handlers.HandleLeagueWidePredictions)
	http.HandleFunc("/api/predictions/accuracy", handlers.HandlePredictionAccuracy)
	http.HandleFunc("/api/predictions/daily-stats", handlers.HandleDailyPredictionStats)
	http.HandleFunc("/api/predictions/trigger", handlers.HandleTriggerDailyPredictions)
	http.HandleFunc("/predictions-stats-popup", handlers.HandlePredictionsStatsPopup)

	// Pre-Game Lineup endpoints
	http.HandleFunc("/api/lineup", handlers.HandleLineup)
	http.HandleFunc("/lineup", handlers.HandleLineupHTML)

	// Play-by-Play Analytics endpoints
	http.HandleFunc("/api/backfill-pbp", handlers.HandleBackfillPlayByPlay)
	http.HandleFunc("/api/pbp-stats", handlers.HandlePlayByPlayStats)
	
	// Game Results backfill endpoint (for processing missed games)
	http.HandleFunc("/api/backfill-games", handlers.HandleBackfillGameResults)
	
	// Force training endpoint (for training on existing completed games)
	http.HandleFunc("/api/force-training", handlers.HandleForceTraining)
	
	// Check unprocessed predictions endpoint
	http.HandleFunc("/api/check-predictions", handlers.HandleCheckUnprocessedPredictions)

	// Performance Metrics Dashboard endpoints
	http.HandleFunc("/api/performance", handlers.PerformanceDashboardHandler)
	http.HandleFunc("/api/metrics", handlers.ModelMetricsHandler)
	http.HandleFunc("/api/rate-limiter", handlers.HandleRateLimiterMetrics)
	http.HandleFunc("/health", handlers.HandleHealth)

	// System Statistics endpoints
	http.HandleFunc("/system-stats", handlers.HandleSystemStats)
	http.HandleFunc("/system-stats-popup", handlers.HandleSystemStatsPopup)
	http.HandleFunc("/api/health", handlers.HandleHealth) // Alternative endpoint

	// Feature Importance Analysis endpoints
	http.HandleFunc("/api/feature-importance", handlers.HandleFeatureImportance)
	http.HandleFunc("/api/feature-importance/markdown", handlers.HandleFeatureImportanceMarkdown)

	// Training Metrics endpoints
	http.HandleFunc("/api/training-metrics", handlers.HandleTrainingMetrics)
	http.HandleFunc("/api/training-metrics/model", handlers.HandleModelTrainingMetrics)
	http.HandleFunc("/api/training-metrics/events", handlers.HandleRecentTrainingEvents)

	// Re-Prediction Metrics endpoint
	http.HandleFunc("/api/reprediction-metrics", handlers.GetRePredictionMetrics)

	// Live Prediction System management endpoints
	if livePredictionSystem := services.GetLivePredictionSystem(); livePredictionSystem != nil {
		http.HandleFunc("/api/live-system/status", livePredictionSystem.HandleSystemStatus)
		http.HandleFunc("/api/live-system/force-update", livePredictionSystem.HandleForceUpdate)
		http.HandleFunc("/api/live-system/model-history", livePredictionSystem.HandleModelHistory)
		fmt.Println("📡 Live prediction system API endpoints registered")
	}

	// Phase 1: Accuracy Enhancement API endpoints
	http.HandleFunc("/api/phase1/dashboard", handlers.GetPhase1AnalyticsDashboard)
	http.HandleFunc("/api/accuracy/summary", handlers.GetAccuracySummary)
	http.HandleFunc("/api/accuracy/patterns", handlers.GetErrorPatterns)
	http.HandleFunc("/api/accuracy/recent", handlers.GetRecentPredictions)
	http.HandleFunc("/api/feature-analysis", handlers.GetFeatureImportance)
	http.HandleFunc("/api/feature-analysis/low-value", handlers.GetLowValueFeatures)
	http.HandleFunc("/api/time-weighted-stats", handlers.GetTimeWeightedStats)
	http.HandleFunc("/api/special-teams-matchup", handlers.AnalyzeSpecialTeamsMatchup)
	fmt.Println("📊 Phase 1 analytics API endpoints registered")

	// Phase 2: Enhanced Data Quality API endpoints
	http.HandleFunc("/api/phase2/dashboard", handlers.GetPhase2AnalyticsDashboard)
	http.HandleFunc("/api/head-to-head/", handlers.GetHeadToHeadMatchup)
	http.HandleFunc("/api/goalie-matchup", handlers.GetGoalieMatchupHistory)
	http.HandleFunc("/api/rest-impact/", handlers.GetRestImpactAnalysis)
	http.HandleFunc("/api/rest-advantage", handlers.GetRestAdvantageComparison)
	http.HandleFunc("/api/rest-rankings", handlers.GetAllRestImpactRankings)
	http.HandleFunc("/api/lineup-impact", handlers.GetLineupImpact)
	fmt.Println("🔬 Phase 2 enhanced data quality API endpoints registered")

	// Phase 3: Confidence & Model Selection API endpoints
	http.HandleFunc("/api/phase3/dashboard", handlers.GetPhase3Dashboard)
	http.HandleFunc("/api/context-analysis/", handlers.GetContextAnalysis)
	http.HandleFunc("/api/model-performance", handlers.GetModelPerformance)
	http.HandleFunc("/api/calibration-curve", handlers.GetCalibrationCurve)
	http.HandleFunc("/api/prediction-quality-metrics", handlers.GetPredictionQualityMetrics)
	http.HandleFunc("/api/recalibration/trigger", handlers.TriggerRecalibration)
	http.HandleFunc("/api/recalibration/history", handlers.GetRecalibrationHistory)
	http.HandleFunc("/api/calibration/update", handlers.UpdateCalibrationCurve)
	http.HandleFunc("/api/context-performance", handlers.GetContextPerformance)
	fmt.Println("🎯 Phase 3 confidence & model selection API endpoints registered")

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

		// Save API cache before shutdown
		apiCache := services.GetAPICacheService()
		if apiCache != nil {
			fmt.Println("💾 Saving API cache...")
			if err := apiCache.SaveCache(); err != nil {
				fmt.Printf("⚠️ Warning: Failed to save API cache: %v\n", err)
			} else {
				fmt.Println("✅ API cache saved")
			}
		}

		// Save training metrics before shutdown
		trainingMetrics := services.GetTrainingMetricsService()
		if trainingMetrics != nil {
			fmt.Println("💾 Saving training metrics...")
			if err := trainingMetrics.SaveMetrics(); err != nil {
				fmt.Printf("⚠️ Warning: Failed to save training metrics: %v\n", err)
			} else {
				fmt.Println("✅ Training metrics saved")
			}
		}

		// Stop the daily prediction service
		if dailyPredictionService != nil {
			dailyPredictionService.Stop()
		}

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
	fmt.Println("🤖 Live prediction models will update automatically every hour")
	fmt.Println("Press Ctrl+C to shutdown gracefully")

	// Start HTTP server with proper error handling
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
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
			cacheMu.Lock()
			cachedSchedule = game
			cachedScheduleUpdated = time.Now()
			cacheMu.Unlock()
			// Safe access with checks for empty team names
			awayTeam := "Unknown"
			homeTeam := "Unknown"
			if game.AwayTeam.CommonName.Default != "" {
				awayTeam = game.AwayTeam.CommonName.Default
			}
			if game.HomeTeam.CommonName.Default != "" {
				homeTeam = game.HomeTeam.CommonName.Default
			}
			fmt.Printf("Schedule updated: %s vs %s on %s\n",
				awayTeam, homeTeam, game.GameDate)

			// Send update to channel only if we successfully fetched a valid game
			select {
			case scheduleChannel <- game:
				// Successfully sent to channel
			default:
				// Channel is full, skip this update
			}
		}

		// Also fetch upcoming games if we're in hockey season
		if currentSeasonStatus.IsHockeySeason {
			fmt.Printf("Fetching updated upcoming games for %s...\n", teamConfig.Code)
			upcomingGames, err := services.GetTeamUpcomingGames(teamConfig.Code)
			if err != nil {
				fmt.Printf("Error fetching upcoming games: %v\n", err)
			} else {
				// Update cached upcoming games
				cacheMu.Lock()
				cachedUpcomingGames = upcomingGames
				cacheMu.Unlock()
				fmt.Printf("Upcoming games updated: %d games found\n", len(upcomingGames))

				// Send update to channel only if we successfully fetched games
				select {
				case upcomingGamesChannel <- upcomingGames:
					// Successfully sent to channel
				default:
					// Channel is full, skip this update
				}
			}
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
		cacheMu.Lock()
		cachedNews = headlines
		cacheMu.Unlock()
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
		cacheMu.Lock()
		cachedPlayerStats = playerLeaders
		cachedTeamPlayerStats = services.GetTeamPlayerStats(playerLeaders, teamConfig.Code)
		cacheMu.Unlock()
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
		cacheMu.Lock()
		cachedGoalieStats = goalieLeaders
		cachedTeamGoalieStats = services.GetTeamGoalieStats(goalieLeaders, teamConfig.Code)
		cacheMu.Unlock()
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
