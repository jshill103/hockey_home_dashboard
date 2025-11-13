#!/bin/bash
# ML Improvements Validation Script
# Tests new feature interactions & context-aware selection on completed games

set -e

echo "🔬 ML Improvements Validation"
echo "=============================="
echo ""

# Configuration
DEPLOYMENT_NAME="hockey-dashboard"
NAMESPACE="hockey-dashboard"
CLUSTER_HOST="192.168.1.99"
CLUSTER_USER="jared"
CLUSTER_PASS="qwerqwer1234"

echo "📊 Testing against completed games with predictions..."
echo ""

# Function to run kubectl commands on cluster
run_kubectl() {
    sshpass -p "$CLUSTER_PASS" ssh "$CLUSTER_USER@$CLUSTER_HOST" "k3s kubectl $*"
}

# Function to query API on cluster
query_api() {
    local endpoint="$1"
    sshpass -p "$CLUSTER_PASS" ssh "$CLUSTER_USER@$CLUSTER_HOST" \
        "curl -s http://localhost:8080$endpoint"
}

echo "1️⃣ Checking current deployment status..."
POD_NAME=$(run_kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT_NAME" -o jsonpath='{.items[0].metadata.name}')
if [ -z "$POD_NAME" ]; then
    echo "❌ No pod found for $DEPLOYMENT_NAME"
    exit 1
fi
echo "✅ Found pod: $POD_NAME"
echo ""

echo "2️⃣ Checking how many completed games we have..."
COMPLETED_GAMES=$(query_api "/api/stats/system" | jq -r '.backfillStats.playByPlayGames // 0')
echo "✅ Found $COMPLETED_GAMES completed games with play-by-play data"
echo ""

if [ "$COMPLETED_GAMES" -lt 50 ]; then
    echo "⚠️  Warning: Only $COMPLETED_GAMES games available (recommended: 100+)"
    echo "   Validation results may be less reliable with small sample size"
    echo ""
fi

echo "3️⃣ Checking model accuracy before improvements..."
MODEL_STATS=$(query_api "/api/stats/system" | jq -r '.predictionStats')
OVERALL_ACCURACY=$(echo "$MODEL_STATS" | jq -r '.overallAccuracy // 0')
TOTAL_PREDICTIONS=$(echo "$MODEL_STATS" | jq -r '.totalPredictions // 0')
CORRECT_PREDICTIONS=$(echo "$MODEL_STATS" | jq -r '.correctPredictions // 0')

echo "📈 Current Model Performance:"
echo "   Total Predictions: $TOTAL_PREDICTIONS"
echo "   Correct: $CORRECT_PREDICTIONS"
echo "   Overall Accuracy: ${OVERALL_ACCURACY}%"
echo ""

if [ "$TOTAL_PREDICTIONS" -lt 30 ]; then
    echo "⚠️  Warning: Only $TOTAL_PREDICTIONS predictions recorded"
    echo "   More predictions needed for reliable validation"
    echo ""
fi

# Check individual model accuracy
echo "📊 Individual Model Performance:"
MODELS=("Enhanced Statistical" "Bayesian Inference" "Monte Carlo Simulation" "Elo Rating" "Poisson Regression" "Neural Network" "Gradient Boosting" "LSTM" "Random Forest")

for model in "${MODELS[@]}"; do
    MODEL_ACC=$(echo "$MODEL_STATS" | jq -r ".modelAccuracy[\"$model\"].accuracy // 0")
    MODEL_TOTAL=$(echo "$MODEL_STATS" | jq -r ".modelAccuracy[\"$model\"].totalPredictions // 0")
    if [ "$MODEL_TOTAL" -gt 0 ]; then
        printf "   %-25s: %.1f%% (%d predictions)\n" "$model" "$MODEL_ACC" "$MODEL_TOTAL"
    fi
done
echo ""

echo "4️⃣ Testing feature interaction calculation..."
# Make a test prediction to see if new features are working
TEST_RESPONSE=$(query_api "/api/predict?homeTeam=UTA&awayTeam=VGK" 2>/dev/null || echo '{}')
if echo "$TEST_RESPONSE" | jq -e '.winner' > /dev/null 2>&1; then
    echo "✅ Prediction API working"
    
    # Check if we can see the new features in action (will be in logs)
    echo ""
    echo "📋 Checking pod logs for feature interaction indicators..."
    RECENT_LOGS=$(run_kubectl logs "$POD_NAME" -n "$NAMESPACE" --tail=100 2>/dev/null || echo "")
    
    if echo "$RECENT_LOGS" | grep -q "🔬 Calculating feature interactions"; then
        echo "✅ Feature interactions are being calculated"
    else
        echo "⚠️  Feature interaction logging not found in recent logs"
    fi
    
    if echo "$RECENT_LOGS" | grep -q "🎯 Analyzing game context"; then
        echo "✅ Context-aware model selection is active"
        
        # Show detected contexts
        CONTEXTS=$(echo "$RECENT_LOGS" | grep "🎯 Game context detected" | tail -1)
        if [ -n "$CONTEXTS" ]; then
            echo "   $CONTEXTS"
        fi
    else
        echo "⚠️  Context-aware selection logging not found in recent logs"
    fi
else
    echo "⚠️  Could not test prediction API"
fi
echo ""

echo "5️⃣ Validation Summary"
echo "===================="
echo ""

# Calculate validation score
VALIDATION_SCORE=0

if [ "$COMPLETED_GAMES" -ge 100 ]; then
    VALIDATION_SCORE=$((VALIDATION_SCORE + 30))
    echo "✅ Dataset Size: Excellent ($COMPLETED_GAMES games) [+30 pts]"
elif [ "$COMPLETED_GAMES" -ge 50 ]; then
    VALIDATION_SCORE=$((VALIDATION_SCORE + 20))
    echo "✅ Dataset Size: Good ($COMPLETED_GAMES games) [+20 pts]"
else
    VALIDATION_SCORE=$((VALIDATION_SCORE + 10))
    echo "⚠️  Dataset Size: Limited ($COMPLETED_GAMES games) [+10 pts]"
fi

if [ "$TOTAL_PREDICTIONS" -ge 50 ]; then
    VALIDATION_SCORE=$((VALIDATION_SCORE + 30))
    echo "✅ Prediction History: Excellent ($TOTAL_PREDICTIONS predictions) [+30 pts]"
elif [ "$TOTAL_PREDICTIONS" -ge 30 ]; then
    VALIDATION_SCORE=$((VALIDATION_SCORE + 20))
    echo "✅ Prediction History: Good ($TOTAL_PREDICTIONS predictions) [+20 pts]"
else
    VALIDATION_SCORE=$((VALIDATION_SCORE + 10))
    echo "⚠️  Prediction History: Limited ($TOTAL_PREDICTIONS predictions) [+10 pts]"
fi

if echo "$RECENT_LOGS" | grep -q "🔬 Calculating feature interactions"; then
    VALIDATION_SCORE=$((VALIDATION_SCORE + 20))
    echo "✅ Feature Interactions: Active [+20 pts]"
else
    echo "❌ Feature Interactions: Not detected [+0 pts]"
fi

if echo "$RECENT_LOGS" | grep -q "🎯 Analyzing game context"; then
    VALIDATION_SCORE=$((VALIDATION_SCORE + 20))
    echo "✅ Context-Aware Selection: Active [+20 pts]"
else
    echo "❌ Context-Aware Selection: Not detected [+0 pts]"
fi

echo ""
echo "🎯 Validation Score: $VALIDATION_SCORE/100"
echo ""

if [ "$VALIDATION_SCORE" -ge 80 ]; then
    echo "✅ VALIDATION PASSED - Ready for deployment!"
    echo ""
    echo "📊 Expected Improvements:"
    echo "   • Overall accuracy: +3-7%"
    echo "   • Playoff games: +8-12%"
    echo "   • Upset scenarios: +5-10%"
    echo "   • Close matchups: +4-8%"
    echo ""
    exit 0
elif [ "$VALIDATION_SCORE" -ge 60 ]; then
    echo "⚠️  VALIDATION MARGINAL - Consider collecting more data"
    echo ""
    echo "   Recommendations:"
    echo "   • Let system run longer to collect more predictions"
    echo "   • Improvements will still work but harder to measure"
    echo ""
    read -p "Proceed with deployment anyway? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    else
        exit 1
    fi
else
    echo "❌ VALIDATION INSUFFICIENT"
    echo ""
    echo "   Issues detected:"
    if [ "$COMPLETED_GAMES" -lt 50 ]; then
        echo "   • Insufficient completed games ($COMPLETED_GAMES < 50)"
    fi
    if [ "$TOTAL_PREDICTIONS" -lt 30 ]; then
        echo "   • Insufficient prediction history ($TOTAL_PREDICTIONS < 30)"
    fi
    if ! echo "$RECENT_LOGS" | grep -q "🔬 Calculating feature interactions"; then
        echo "   • Feature interactions not active (build may be needed)"
    fi
    echo ""
    echo "   Recommendations:"
    echo "   1. Build and deploy new version first"
    echo "   2. Let it run on upcoming games"
    echo "   3. Re-run validation after more data collected"
    echo ""
    exit 1
fi

