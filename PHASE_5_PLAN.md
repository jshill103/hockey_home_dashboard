# 🚀 **Phase 5: Model Improvements**

## **Goal: +5-9% Accuracy Through Advanced ML**

**Current Status:** Phase 4 Complete (81-94% accuracy)  
**Target:** 86-97% accuracy after Phase 5  
**Timeline:** 3-4 weeks  
**Focus:** Advanced models and ensemble optimization

---

## 📊 **Current Model Status**

### **Active Models (6):**
1. ✅ **Enhanced Statistical** (35% weight)
2. ✅ **Bayesian Inference** (15% weight)
3. ✅ **Monte Carlo Simulation** (10% weight)
4. ✅ **Elo Rating Model** (20% weight) - Learning
5. ✅ **Poisson Regression** (15% weight) - Learning
6. ✅ **Neural Network** (5% weight) - Learning [65 features]

### **Implemented But Unused (4):**
1. ⚠️ **XGBoost** - Created but not trained/integrated
2. ⚠️ **LSTM** - Created but not trained/integrated
3. ⚠️ **Random Forest** - Created but not trained/integrated
4. ⚠️ **Gradient Boosting** - Planned but not created

---

## 🎯 **Phase 5 Components**

### **5.1: Neural Network Architecture Search** ⭐⭐⭐⭐⭐
**Priority:** HIGH  
**Expected Impact:** +2-3% accuracy  
**Timeline:** 1 week

**Current Architecture:**
```
[65, 32, 16, 3] - Input, Hidden, Hidden, Output
```

**Goals:**
1. **Test multiple architectures** to find optimal
2. **Hyperparameter tuning** (learning rate, layers, neurons)
3. **Regularization** (dropout, L2) to prevent overfitting
4. **Activation functions** (ReLU, LeakyReLU, ELU, Swish)
5. **Batch normalization** for training stability

**Architecture Candidates:**
```
Shallow & Wide:
- [65, 128, 64, 3]
- [65, 256, 3]

Deep & Narrow:
- [65, 32, 32, 16, 16, 3]
- [65, 48, 48, 24, 12, 3]

Balanced:
- [65, 64, 32, 16, 3]
- [65, 96, 48, 24, 3]

With Dropout:
- [65, 128(dropout), 64(dropout), 3]
- [65, 96(dropout), 48(dropout), 24, 3]
```

**Tuning Strategy:**
1. Use train/test split for evaluation
2. Grid search over hyperparameters
3. Track validation loss, not just training loss
4. Select best performing architecture
5. Retrain on full dataset

**Implementation Steps:**
1. Create architecture search framework
2. Test 20-30 architectures
3. Evaluate on validation set
4. Select top 3 architectures
5. Ensemble the top 3 (if beneficial)

---

### **5.2: XGBoost Integration** ⭐⭐⭐⭐⭐
**Priority:** HIGH  
**Expected Impact:** +1-2% accuracy  
**Timeline:** 1 week

**Current Status:** Skeleton exists but not trained

**Goals:**
1. **Implement proper XGBoost training**
2. **Feature importance analysis**
3. **Hyperparameter tuning**
4. **Add to ensemble**

**Why XGBoost:**
- Excellent for tabular data (we have 65 features)
- Handles non-linear relationships well
- Built-in feature importance
- Fast inference
- Robust to overfitting

**Implementation:**
```go
type XGBoostModel struct {
    model      *xgb.Booster
    params     map[string]interface{}
    features   []string
    weight     float64
    trained    bool
    dataDir    string
}

// Training parameters
params := map[string]interface{}{
    "max_depth":        6,
    "eta":              0.1,
    "objective":        "multi:softprob",
    "num_class":        3,
    "subsample":        0.8,
    "colsample_bytree": 0.8,
    "min_child_weight": 1,
    "gamma":            0.1,
}
```

**Training Strategy:**
1. Use game results from `GameResultsService`
2. Train on same data as Neural Network
3. Use cross-validation for hyperparameters
4. Track feature importance
5. Save model with persistence

**Integration:**
1. Add to `EnsemblePredictionService`
2. Start with 10% weight
3. Adjust weight based on performance
4. Compare to other models

---

### **5.3: LSTM for Sequence Learning** ⭐⭐⭐⭐
**Priority:** MEDIUM  
**Expected Impact:** +1-2% accuracy  
**Timeline:** 1 week

**Current Status:** Skeleton exists but not trained

**Goals:**
1. **Learn from game sequences** (last 10 games)
2. **Capture momentum** and trends
3. **Model hot/cold streaks**
4. **Predict performance trajectories**

**Why LSTM:**
- Designed for sequences (team's last N games)
- Can capture long-term dependencies
- Good for momentum/streak modeling
- Complementary to tabular models

**Architecture:**
```
Input: [batch, sequence_length=10, features=65]
    ↓
LSTM Layer 1: 128 units
    ↓
Dropout: 0.2
    ↓
LSTM Layer 2: 64 units
    ↓
Dropout: 0.2
    ↓
Dense: 32 units (ReLU)
    ↓
Output: 3 units (Softmax) [win/loss/OT]
```

**Data Preparation:**
```go
// For each game, use last 10 games as sequence
type GameSequence struct {
    TeamSequence     [][]float64 // [10 games][65 features]
    OpponentSequence [][]float64 // [10 games][65 features]
    Result           int         // 0=loss, 1=win, 2=OT
}
```

**Training:**
1. Prepare sequences from game results
2. Train with backpropagation through time
3. Use teacher forcing for stability
4. Validate on hold-out sequences

**Integration:**
1. Add to ensemble with 5% weight
2. Use for games with clear momentum
3. Weight higher when team on streak

---

### **5.4: Ensemble Optimization** ⭐⭐⭐⭐⭐
**Priority:** HIGH  
**Expected Impact:** +1-2% accuracy  
**Timeline:** 1 week

**Current Status:** Static weights, manual tuning

**Goals:**
1. **Learn optimal ensemble weights**
2. **Context-aware weighting** (home/away, B2B, etc.)
3. **Meta-learning** approach
4. **Automatic weight adjustment**

**Current Weights (Static):**
```
Statistical: 35%
Bayesian: 15%
Monte Carlo: 10%
Elo: 20%
Poisson: 15%
Neural Net: 5%
```

**Proposed: Meta-Learner Approach**

```go
type MetaLearner struct {
    // Learn which model to trust in which situation
    contextFeatures map[string]float64
    modelWeights    map[string]map[string]float64
}

// Context features
type PredictionContext struct {
    IsHomeTeam       bool
    IsBackToBack     bool
    IsPlayoffs       bool
    GoalieMismatch   float64 // Large advantage?
    MarketDisagrees  bool    // We differ from market?
    HistoricalRivalry bool
    TeamMomentum     float64
    ConfidenceSpread float64 // Model agreement?
}

// Context-aware weights
func (ml *MetaLearner) GetWeights(context PredictionContext) map[string]float64 {
    // Example logic:
    // - If goalie mismatch: weight Goalie Service higher
    // - If market disagrees: weight Market Service higher
    // - If back-to-back: weight Schedule Context higher
    // - If high momentum: weight LSTM higher
    // - If models agree: weight ensemble confidence higher
    // - If models disagree: weight XGBoost higher
}
```

**Training Strategy:**
1. For each historical prediction, record:
   - Context features
   - Each model's prediction
   - Actual result
2. Train meta-learner to predict:
   - Which model was most accurate
   - Optimal weight for each model
3. Use logistic regression or small NN
4. Validate on held-out games

**Expected Improvements:**
- Better handling of edge cases
- Automatic adaptation to new data
- Reduced overfitting to any single model
- Improved calibration

---

### **5.5: Random Forest (Optional)** ⭐⭐⭐
**Priority:** LOW  
**Expected Impact:** +0.5-1% accuracy  
**Timeline:** 3 days

**Current Status:** Skeleton exists but not trained

**Why Random Forest:**
- Excellent ensemble method
- Handles non-linear relationships
- Feature importance built-in
- Robust to overfitting
- No hyperparameter sensitivity

**Implementation:**
- Similar to XGBoost
- Train on tabular features
- Use for feature importance analysis
- Add to ensemble if performance warrants

---

## 📋 **Implementation Plan**

### **Week 1: Neural Network Architecture Search**
**Days 1-2:** Framework setup
- Create architecture testing framework
- Implement grid search
- Set up validation pipeline

**Days 3-5:** Testing
- Test 20-30 architectures
- Track performance metrics
- Analyze results

**Days 6-7:** Selection & Retraining
- Select best architecture
- Retrain on full dataset
- Update production model
- Validate improvement

---

### **Week 2: XGBoost Integration**
**Days 1-2:** Implementation
- Integrate XGBoost library
- Create training pipeline
- Implement persistence

**Days 3-4:** Training
- Train on game results
- Hyperparameter tuning
- Feature importance analysis

**Days 5-7:** Integration & Testing
- Add to ensemble
- Tune weight
- Validate improvement
- Performance analysis

---

### **Week 3: LSTM Sequence Learning**
**Days 1-2:** Data Preparation
- Create sequence data structures
- Prepare training sequences
- Implement sequence generator

**Days 3-4:** Model Implementation
- Build LSTM architecture
- Implement training loop
- Add persistence

**Days 5-7:** Training & Integration
- Train on sequences
- Add to ensemble
- Validate performance
- Analyze momentum capture

---

### **Week 4: Ensemble Optimization**
**Days 1-3:** Meta-Learner Development
- Design meta-learner
- Implement context features
- Build training pipeline

**Days 4-5:** Training
- Train meta-learner
- Validate weight selection
- A/B test vs static weights

**Days 6-7:** Integration & Monitoring
- Deploy meta-learner
- Monitor performance
- Fine-tune as needed

---

## 📊 **Expected Outcomes**

### **Model Performance:**

| Model | Current | After Phase 5 | Improvement |
|-------|---------|---------------|-------------|
| Neural Network | 70-75% | 75-80% | +5% |
| XGBoost | Not Active | 75-80% | NEW |
| LSTM | Not Active | 72-77% | NEW |
| Ensemble (Static) | 81-94% | - | - |
| Ensemble (Meta) | - | 86-97% | +5-3% |

### **Accuracy Progression:**
```
Phase 4 Complete: 81-94%
    ↓
After NN Architecture Search: 83-95% (+2%)
    ↓
After XGBoost Integration: 84-96% (+1%)
    ↓
After LSTM Integration: 85-96% (+1%)
    ↓
After Ensemble Optimization: 86-97% (+1%)
    ↓
PHASE 5 COMPLETE: 86-97% accuracy
```

**Total Phase 5 Improvement: +5-9%**

---

## 🎯 **Success Metrics**

Track these to validate improvements:

1. **Overall Accuracy**
   - Target: 86-97% (up from 81-94%)
   - Measure on validation set

2. **Model-Specific Accuracy**
   - NN: 75-80%
   - XGBoost: 75-80%
   - LSTM: 72-77%

3. **Ensemble Performance**
   - Better than best individual model
   - Improved calibration
   - Lower Brier score

4. **Computational Efficiency**
   - Prediction time: <100ms (all models)
   - Training time: <30min (batch)
   - Memory: <500MB

---

## 🔧 **Technical Requirements**

### **Libraries Needed:**
```go
// For XGBoost
go get github.com/dmitryikh/leaves

// For LSTM (via Python bridge or pure Go)
// Option 1: Gorgonia (pure Go)
go get gorgonia.org/gorgonia
go get gorgonia.org/tensor

// Option 2: Python bridge (TensorFlow/PyTorch)
// Use os/exec to call Python scripts
```

### **Data Requirements:**
- At least 100 completed games for training
- Validation set: 20% of games
- Test set: 20% of games
- Sequences: Last 10 games per team

### **Persistence:**
- Save models to `data/models/`
- Version models (v1, v2, etc.)
- Track performance history
- Enable rollback if needed

---

## ⚠️ **Risks & Mitigation**

### **Risk 1: Overfitting**
**Mitigation:**
- Use train/test/validation split
- Implement dropout and regularization
- Monitor validation loss
- Early stopping

### **Risk 2: Computational Complexity**
**Mitigation:**
- Optimize inference paths
- Cache predictions
- Parallel model execution
- Limit ensemble size if needed

### **Risk 3: Model Disagreement**
**Mitigation:**
- Meta-learner handles disagreement
- Weight based on historical accuracy
- Track model correlation
- Ensemble uncertainty quantification

### **Risk 4: Data Insufficiency**
**Mitigation:**
- Wait until enough games collected
- Use transfer learning from other seasons
- Start with simpler architectures
- Augment data if needed

---

## 🎓 **Key Decisions**

### **Decision 1: Pure Go vs Python Bridge**
**Recommendation:** Start with pure Go (Gorgonia)
- Easier deployment
- No dependencies
- Better performance
- Can switch to Python later if needed

### **Decision 2: All Models or Select Few**
**Recommendation:** Implement all, use top performers
- Test everything
- Keep top 3-4 in production
- Disable underperformers
- Re-evaluate monthly

### **Decision 3: Static vs Dynamic Weights**
**Recommendation:** Dynamic (Meta-Learner)
- More accurate
- Adapts to context
- Better uncertainty
- Worth the complexity

### **Decision 4: Feature Engineering**
**Recommendation:** Keep Phase 5 focused on models
- Feature engineering is Phase 6
- Don't mix concerns
- Current 65 features are good
- Add features later

---

## 🚀 **Getting Started**

### **Immediate Next Steps:**

1. **Create Phase 5 Branch**
   ```bash
   git checkout -b phase-5-model-improvements
   ```

2. **Set Up Architecture Search**
   - Create `services/nn_architecture_search.go`
   - Implement testing framework
   - Prepare validation data

3. **Verify XGBoost Library**
   ```bash
   go get github.com/dmitryikh/leaves
   go test ./services -v -run TestXGBoost
   ```

4. **Plan LSTM Data Pipeline**
   - Design sequence structures
   - Create data generator
   - Test on sample data

5. **Design Meta-Learner**
   - Define context features
   - Plan weight learning
   - Create evaluation framework

---

## 📚 **Resources & References**

### **Neural Network Architecture:**
- [Architecture Search Best Practices](https://arxiv.org/abs/1902.09635)
- [Neural Architecture Search Survey](https://arxiv.org/abs/1808.05377)

### **XGBoost:**
- [XGBoost Paper](https://arxiv.org/abs/1603.02754)
- [Go Library: Leaves](https://github.com/dmitryikh/leaves)

### **LSTM:**
- [Understanding LSTM Networks](http://colah.github.io/posts/2015-08-Understanding-LSTMs/)
- [Gorgonia Tutorial](https://github.com/gorgonia/gorgonia)

### **Ensemble Learning:**
- [Ensemble Methods Survey](https://arxiv.org/abs/1404.4088)
- [Meta-Learning](https://arxiv.org/abs/1810.03548)

---

## 🎉 **Success Criteria**

Phase 5 is complete when:

- [x] Neural Network architecture optimized (+2-3%)
- [x] XGBoost trained and integrated (+1-2%)
- [x] LSTM trained and integrated (+1-2%)
- [x] Meta-learner implemented (+1-2%)
- [x] All models persisting properly
- [x] Ensemble accuracy 86-97%
- [x] Prediction time <100ms
- [x] Full test coverage
- [x] Documentation complete

---

**Ready to build world-class ML models! Let's start with Neural Network Architecture Search!** 🚀🧠


