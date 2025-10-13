package services

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database provides SQLite persistence
type Database struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

var (
	globalDB     *Database
	globalDBOnce sync.Once
)

// InitDatabase initializes the global database
func InitDatabase(dbPath string) error {
	var initErr error

	globalDBOnce.Do(func() {
		// Create directory if it doesn't exist
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create database directory: %w", err)
			return
		}

		// Open database
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			initErr = fmt.Errorf("failed to open database: %w", err)
			return
		}

		// Configure connection pool
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		globalDB = &Database{
			db:   db,
			path: dbPath,
		}

		// Initialize schema
		if err := globalDB.initSchema(); err != nil {
			initErr = fmt.Errorf("failed to initialize schema: %w", err)
			return
		}

		fmt.Printf("✅ Database initialized: %s\n", dbPath)
	})

	return initErr
}

// GetDatabase returns the global database
func GetDatabase() *Database {
	return globalDB
}

// initSchema creates the database tables
func (db *Database) initSchema() error {
	schema := `
	-- Team Stats Table
	CREATE TABLE IF NOT EXISTS team_stats (
		team_code TEXT NOT NULL,
		season INTEGER NOT NULL,
		games_played INTEGER DEFAULT 0,
		wins INTEGER DEFAULT 0,
		losses INTEGER DEFAULT 0,
		points INTEGER DEFAULT 0,
		goals_for INTEGER DEFAULT 0,
		goals_against INTEGER DEFAULT 0,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (team_code, season)
	);
	
	-- Game Results Table
	CREATE TABLE IF NOT EXISTS game_results (
		game_id INTEGER PRIMARY KEY,
		home_team TEXT NOT NULL,
		away_team TEXT NOT NULL,
		home_score INTEGER,
		away_score INTEGER,
		game_date DATE,
		season INTEGER,
		game_type TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	-- Predictions Table
	CREATE TABLE IF NOT EXISTS predictions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		game_id INTEGER NOT NULL,
		home_team TEXT NOT NULL,
		away_team TEXT NOT NULL,
		predicted_home_win_prob REAL,
		predicted_away_win_prob REAL,
		prediction_date TIMESTAMP,
		actual_winner TEXT,
		correct BOOLEAN,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (game_id) REFERENCES game_results(game_id)
	);
	
	-- Model Performance Table
	CREATE TABLE IF NOT EXISTS model_performance (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_name TEXT NOT NULL,
		total_predictions INTEGER DEFAULT 0,
		correct_predictions INTEGER DEFAULT 0,
		accuracy REAL,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	-- Player Stats Table
	CREATE TABLE IF NOT EXISTS player_stats (
		player_id INTEGER NOT NULL,
		team_code TEXT NOT NULL,
		season INTEGER NOT NULL,
		games_played INTEGER DEFAULT 0,
		goals INTEGER DEFAULT 0,
		assists INTEGER DEFAULT 0,
		points INTEGER DEFAULT 0,
		plus_minus INTEGER DEFAULT 0,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (player_id, season)
	);
	
	-- Goalie Stats Table
	CREATE TABLE IF NOT EXISTS goalie_stats (
		goalie_id INTEGER NOT NULL,
		team_code TEXT NOT NULL,
		season INTEGER NOT NULL,
		games_played INTEGER DEFAULT 0,
		wins INTEGER DEFAULT 0,
		save_percentage REAL DEFAULT 0.0,
		goals_against_avg REAL DEFAULT 0.0,
		shutouts INTEGER DEFAULT 0,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (goalie_id, season)
	);
	
	-- ML Model Weights Table (for persistence)
	CREATE TABLE IF NOT EXISTS ml_model_weights (
		model_name TEXT PRIMARY KEY,
		weights_json TEXT,
		training_count INTEGER DEFAULT 0,
		accuracy REAL DEFAULT 0.0,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	-- Cache Table (generic key-value cache)
	CREATE TABLE IF NOT EXISTS cache (
		cache_key TEXT PRIMARY KEY,
		cache_value TEXT,
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	-- System Metrics Table
	CREATE TABLE IF NOT EXISTS system_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		metric_name TEXT NOT NULL,
		metric_value REAL,
		metric_data TEXT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_game_results_date ON game_results(game_date);
	CREATE INDEX IF NOT EXISTS idx_game_results_teams ON game_results(home_team, away_team);
	CREATE INDEX IF NOT EXISTS idx_predictions_game_id ON predictions(game_id);
	CREATE INDEX IF NOT EXISTS idx_player_stats_team ON player_stats(team_code, season);
	CREATE INDEX IF NOT EXISTS idx_goalie_stats_team ON goalie_stats(team_code, season);
	CREATE INDEX IF NOT EXISTS idx_cache_expires ON cache(expires_at);
	CREATE INDEX IF NOT EXISTS idx_system_metrics_name ON system_metrics(metric_name, timestamp);
	`

	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.db.Exec(schema)
	return err
}

// SaveTeamStats saves team statistics
func (db *Database) SaveTeamStats(teamCode string, season int, stats map[string]interface{}) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
		INSERT INTO team_stats (team_code, season, games_played, wins, losses, points, goals_for, goals_against)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(team_code, season) DO UPDATE SET
			games_played = excluded.games_played,
			wins = excluded.wins,
			losses = excluded.losses,
			points = excluded.points,
			goals_for = excluded.goals_for,
			goals_against = excluded.goals_against,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.db.Exec(query,
		teamCode,
		season,
		getInt(stats, "games_played"),
		getInt(stats, "wins"),
		getInt(stats, "losses"),
		getInt(stats, "points"),
		getInt(stats, "goals_for"),
		getInt(stats, "goals_against"),
	)

	return err
}

// SaveGameResult saves a game result
func (db *Database) SaveGameResult(gameID int, homeTeam, awayTeam string, homeScore, awayScore int, gameDate time.Time, season int) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
		INSERT INTO game_results (game_id, home_team, away_team, home_score, away_score, game_date, season)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(game_id) DO UPDATE SET
			home_score = excluded.home_score,
			away_score = excluded.away_score
	`

	_, err := db.db.Exec(query, gameID, homeTeam, awayTeam, homeScore, awayScore, gameDate, season)
	return err
}

// SavePrediction saves a prediction
func (db *Database) SavePrediction(gameID int, homeTeam, awayTeam string, homeWinProb, awayWinProb float64) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
		INSERT INTO predictions (game_id, home_team, away_team, predicted_home_win_prob, predicted_away_win_prob, prediction_date)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.db.Exec(query, gameID, homeTeam, awayTeam, homeWinProb, awayWinProb, time.Now())
	return err
}

// UpdatePredictionResult updates a prediction with actual result
func (db *Database) UpdatePredictionResult(gameID int, actualWinner string, correct bool) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	query := `UPDATE predictions SET actual_winner = ?, correct = ? WHERE game_id = ?`
	_, err := db.db.Exec(query, actualWinner, correct, gameID)
	return err
}

// GetModelAccuracy retrieves model accuracy from database
func (db *Database) GetModelAccuracy(modelName string) (float64, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	db.mu.RLock()
	defer db.mu.RUnlock()

	var accuracy sql.NullFloat64
	query := `SELECT accuracy FROM model_performance WHERE model_name = ?`
	err := db.db.QueryRow(query, modelName).Scan(&accuracy)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	if accuracy.Valid {
		return accuracy.Float64, nil
	}

	return 0, nil
}

// SaveModelPerformance saves model performance metrics
func (db *Database) SaveModelPerformance(modelName string, totalPredictions, correctPredictions int, accuracy float64) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
		INSERT INTO model_performance (model_name, total_predictions, correct_predictions, accuracy)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(model_name) DO UPDATE SET
			total_predictions = excluded.total_predictions,
			correct_predictions = excluded.correct_predictions,
			accuracy = excluded.accuracy,
			last_updated = CURRENT_TIMESTAMP
	`

	_, err := db.db.Exec(query, modelName, totalPredictions, correctPredictions, accuracy)
	return err
}

// SaveCache saves a key-value pair to cache
func (db *Database) SaveCache(key, value string, ttl time.Duration) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	expiresAt := time.Now().Add(ttl)
	query := `
		INSERT INTO cache (cache_key, cache_value, expires_at)
		VALUES (?, ?, ?)
		ON CONFLICT(cache_key) DO UPDATE SET
			cache_value = excluded.cache_value,
			expires_at = excluded.expires_at,
			created_at = CURRENT_TIMESTAMP
	`

	_, err := db.db.Exec(query, key, value, expiresAt)
	return err
}

// GetCache retrieves a value from cache
func (db *Database) GetCache(key string) (string, bool, error) {
	if db == nil {
		return "", false, fmt.Errorf("database not initialized")
	}

	db.mu.RLock()
	defer db.mu.RUnlock()

	var value string
	var expiresAt time.Time
	query := `SELECT cache_value, expires_at FROM cache WHERE cache_key = ?`
	err := db.db.QueryRow(query, key).Scan(&value, &expiresAt)

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		return "", false, nil
	}

	return value, true, nil
}

// CleanExpiredCache removes expired cache entries
func (db *Database) CleanExpiredCache() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	query := `DELETE FROM cache WHERE expires_at < ?`
	result, err := db.db.Exec(query, time.Now())
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		fmt.Printf("🗑️ Cleaned %d expired cache entries\n", rows)
	}

	return nil
}

// Close closes the database connection
func (db *Database) Close() error {
	if db == nil {
		return nil
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	return db.db.Close()
}

// Helper functions

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.0
}
