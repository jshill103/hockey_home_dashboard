#!/bin/sh
set -e

echo "📁 Initializing data directories after PVC mount..."

# Create all required data directories if they don't exist
# This runs after PVC mount, so directories need to be created at runtime
mkdir -p /app/data/accuracy
mkdir -p /app/data/models
mkdir -p /app/data/results
mkdir -p /app/data/matchups
mkdir -p /app/data/rolling_stats
mkdir -p /app/data/player_impact
mkdir -p /app/data/lineups
mkdir -p /app/data/play_by_play
mkdir -p /app/data/shifts
mkdir -p /app/data/landing_page
mkdir -p /app/data/game_summary
mkdir -p /app/data/cache/predictions
mkdir -p /app/data/cache/api
mkdir -p /app/data/predictions
mkdir -p /app/data/metrics
mkdir -p /app/data/rosters
mkdir -p /app/data/evaluation
mkdir -p /app/data/goalies
mkdir -p /app/data/betting_markets
mkdir -p /app/data/architecture_search

# Set permissions
# If running as root (init container scenario), chown to appuser
# Otherwise, rely on Kubernetes fsGroup or existing permissions
if [ "$(id -u)" = "0" ]; then
    echo "🔧 Running as root, setting ownership to appuser:appgroup..."
    chown -R appuser:appgroup /app/data 2>/dev/null || echo "⚠️  Warning: Could not chown /app/data (may already be correct)"
    chmod -R 755 /app/data 2>/dev/null || echo "⚠️  Warning: Could not chmod /app/data"
else
    # Running as appuser - ensure directories are writable
    chmod -R 755 /app/data 2>/dev/null || echo "⚠️  Warning: Could not chmod /app/data (may need fsGroup in Kubernetes)"
    
    # Verify we can write to the data directory
    if ! touch /app/data/.write_test 2>/dev/null; then
        echo "❌ ERROR: Cannot write to /app/data directory!"
        echo "   This usually means the PVC is mounted with incorrect permissions."
        echo "   Solution: Add fsGroup: 1001 to your Kubernetes deployment securityContext"
        exit 1
    fi
    rm -f /app/data/.write_test
fi

echo "✅ Data directories initialized successfully"

# Execute the main command
exec "$@"

