#!/bin/bash

echo "🔍 ML Model Verification Script"
echo "================================"
echo ""

# 1. Check persistence files
echo "📁 Checking persistence files..."
if [ -d "data/models" ]; then
    ls -lh data/models/ | grep -E "elo_ratings|poisson_rates|neural_network|gradient_boosting|lstm|random_forest|meta_learner"
    echo ""
else
    echo "⚠️ No models directory found"
    echo ""
fi

# 2. Check if server is running
echo "🌐 Checking if server is running..."
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Server is running"
else
    echo "❌ Server is not running - start with: ./web_server UTA"
fi
echo ""

# 3. Make a prediction and check all models
echo "🎯 Testing prediction (checking all 9 models)..."
if curl -s http://localhost:8080/api/prediction-widget > /dev/null 2>&1; then
    response=$(curl -s http://localhost:8080/api/prediction-widget)
    
    # Count models
    model_count=$(echo "$response" | jq -r '.modelResults | length' 2>/dev/null)
    echo "Found $model_count models in ensemble"
    echo ""
    
    # Show each model's prediction
    echo "$response" | jq -r '.modelResults[] | "  \(.modelName): \(.winProbability * 100 | round)% (Confidence: \(.confidence * 100 | round)%)"' 2>/dev/null
    echo ""
    
    # Check ensemble method
    ensemble_method=$(echo "$response" | jq -r '.ensembleMethod' 2>/dev/null)
    echo "Ensemble Method: $ensemble_method"
    
    if [[ "$ensemble_method" == *"Meta-Learner"* ]]; then
        echo "✅ Meta-Learner is active!"
    else
        echo "⚠️ Meta-Learner not active (using weighted average - this is normal if not trained)"
    fi
else
    echo "❌ Could not get prediction - is server running?"
fi
echo ""

# 4. Check model weights
echo "⚖️ Checking model weights..."
if curl -s http://localhost:8080/api/model-weights > /dev/null 2>&1; then
    curl -s http://localhost:8080/api/model-weights | jq '.' 2>/dev/null
else
    echo "❌ Could not get model weights"
fi
echo ""

# 5. Check accuracy
echo "📊 Checking accuracy..."
if [ -f "data/accuracy/accuracy_data.json" ]; then
    cat data/accuracy/accuracy_data.json | jq '.overall' 2>/dev/null || echo "⚠️ Could not parse accuracy data"
else
    echo "⚠️ No accuracy data yet (will appear after making predictions)"
fi
echo ""

# 6. Check for trained models
echo "🎓 Checking which models are trained..."
for model_file in data/models/*.json; do
    if [ -f "$model_file" ]; then
        filename=$(basename "$model_file")
        trained=$(cat "$model_file" | jq -r '.trained' 2>/dev/null)
        if [ "$trained" == "true" ]; then
            echo "  ✅ $filename: TRAINED"
        elif [ "$trained" == "false" ]; then
            echo "  ⚠️ $filename: NOT TRAINED (will improve with data)"
        else
            echo "  ℹ️ $filename: (no training status)"
        fi
    fi
done
echo ""

# 7. Summary
echo "📋 Summary"
echo "=========="
echo ""

# Count persistence files
model_files=$(ls data/models/*.json 2>/dev/null | wc -l | tr -d ' ')
echo "Persistence files: $model_files/7 (Elo, Poisson, NN, GB, LSTM, RF, Meta)"

# Check if all models are present
if [ "$model_count" == "9" ]; then
    echo "Active models: ✅ All 9 base models present"
elif [ -n "$model_count" ]; then
    echo "Active models: ⚠️ Only $model_count/9 models active"
else
    echo "Active models: ❌ Could not verify (server not running?)"
fi

echo ""
echo "✅ Verification complete!"
echo ""
echo "💡 Tips:"
echo "  - If models show 50% probability, they need training data"
echo "  - Models will improve as they learn from completed games"
echo "  - Meta-learner activates after training on historical data"
echo "  - Check ML_MODEL_VERIFICATION_GUIDE.md for detailed info"
