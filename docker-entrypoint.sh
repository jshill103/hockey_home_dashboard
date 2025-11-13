#!/bin/sh
set -e

echo "📁 Initializing data directories after PVC mount..."

# Create all required data directories if they don't exist
# This runs after PVC mount, so directories need to be created at runtime
# Use || true to continue even if mkdir fails (directories might already exist or PVC might handle it)
mkdir -p /app/data/accuracy 2>/dev/null || echo "   ✓ accuracy directory ready"
mkdir -p /app/data/models 2>/dev/null || echo "   ✓ models directory ready"
mkdir -p /app/data/results 2>/dev/null || echo "   ✓ results directory ready"
mkdir -p /app/data/matchups 2>/dev/null || echo "   ✓ matchups directory ready"
mkdir -p /app/data/rolling_stats 2>/dev/null || echo "   ✓ rolling_stats directory ready"
mkdir -p /app/data/player_impact 2>/dev/null || echo "   ✓ player_impact directory ready"
mkdir -p /app/data/lineups 2>/dev/null || echo "   ✓ lineups directory ready"
mkdir -p /app/data/play_by_play 2>/dev/null || echo "   ✓ play_by_play directory ready"
mkdir -p /app/data/shifts 2>/dev/null || echo "   ✓ shifts directory ready"
mkdir -p /app/data/landing_page 2>/dev/null || echo "   ✓ landing_page directory ready"
mkdir -p /app/data/game_summary 2>/dev/null || echo "   ✓ game_summary directory ready"
mkdir -p /app/data/cache/predictions 2>/dev/null || echo "   ✓ cache/predictions directory ready"
mkdir -p /app/data/cache/api 2>/dev/null || echo "   ✓ cache/api directory ready"
mkdir -p /app/data/predictions 2>/dev/null || echo "   ✓ predictions directory ready"
mkdir -p /app/data/metrics 2>/dev/null || echo "   ✓ metrics directory ready"
mkdir -p /app/data/rosters 2>/dev/null || echo "   ✓ rosters directory ready"
mkdir -p /app/data/evaluation 2>/dev/null || echo "   ✓ evaluation directory ready"
mkdir -p /app/data/goalies 2>/dev/null || echo "   ✓ goalies directory ready"
mkdir -p /app/data/betting_markets 2>/dev/null || echo "   ✓ betting_markets directory ready"
mkdir -p /app/data/architecture_search 2>/dev/null || echo "   ✓ architecture_search directory ready"

# Set permissions (best effort - don't fail if we can't)
# If running as root (init container scenario), chown to appuser
# Otherwise, rely on Kubernetes fsGroup or existing permissions
if [ "$(id -u)" = "0" ]; then
    echo "🔧 Running as root, setting ownership to appuser:appgroup..."
    chown -R appuser:appgroup /app/data 2>/dev/null || echo "⚠️  Warning: Could not chown /app/data (may already be correct)"
    chmod -R 755 /app/data 2>/dev/null || echo "⚠️  Warning: Could not chmod /app/data"
else
    # Running as appuser - try to ensure directories are writable (best effort)
    chmod -R 755 /app/data 2>/dev/null || echo "⚠️  Warning: Could not chmod /app/data (relying on Kubernetes fsGroup)"
    
    # Try to verify we can write to the data directory (non-fatal)
    if touch /app/data/.write_test 2>/dev/null; then
        rm -f /app/data/.write_test
        echo "✅ Data directory is writable"
    else
        echo "⚠️  Warning: Cannot create test file in /app/data"
        echo "   The application will attempt to create directories as needed."
        echo "   If you see permission errors, add fsGroup: 1001 to your Kubernetes securityContext"
    fi
fi

echo "✅ Data directories initialized successfully"

# Execute the main command
exec "$@"

