# Creating the Video Analyzer Application - Next Steps

## 🎯 Decision Summary

**Architectural Decision**: Separate the video processing into a new standalone application

### Why This Is Smart:
✅ **Separation of Concerns**: Video processing is completely different from game predictions  
✅ **Independent Scaling**: Can scale video processing separately from main app  
✅ **Easier Maintenance**: Each app has a single responsibility  
✅ **Technology Freedom**: Can use different ML frameworks for video without impacting predictions  
✅ **Resource Management**: Video processing is resource-intensive, keep it isolated  

---

## 📦 Repository Structure

### **Current Repository** (This one): `go_uhc` / `hockey_home_dashboard`
**Remains as-is**: Game predictions, statistics, ML models, web UI

**Role**: Consumes video analysis data via API

### **New Repository**: `hockey_video_analyzer`
**Purpose**: HDHomeRun recording, video processing, ML video analysis

**Role**: Produces video analysis data for the dashboard

---

## 🚀 Step-by-Step Setup Guide

### Step 1: Create New GitHub Repository
```bash
# On GitHub:
# 1. Go to https://github.com/new
# 2. Repository name: hockey_video_analyzer
# 3. Description: "NHL video capture and ML analysis from HDHomeRun"
# 4. Public or Private: Your choice
# 5. Add .gitignore: Go
# 6. Add README: Yes
# 7. License: Same as main app (MIT recommended)
```

### Step 2: Clone and Initialize Locally
```bash
cd ~/Jared-Repos
git clone https://github.com/YOUR_USERNAME/hockey_video_analyzer.git
cd hockey_video_analyzer

# Initialize Go module
go mod init github.com/YOUR_USERNAME/hockey_video_analyzer

# Create directory structure
mkdir -p cmd/{recorder,processor,analyzer}
mkdir -p internal/{capture,processing,analysis,models,storage}
mkdir -p pkg/{api,client}
mkdir -p models/{shot_detector,player_tracker,event_classifier}
mkdir -p scripts
mkdir -p k8s
mkdir -p docker
mkdir -p docs
```

### Step 3: Create Initial Files

#### README.md
```markdown
# Hockey Video Analyzer

NHL game capture, processing, and ML-based video analysis from HDHomeRun.

## Overview
This application records NHL games from HDHomeRun, processes the video, and uses ML models to analyze:
- Shot detection and quality
- Player tracking and positioning
- Event classification
- Zone time and coverage analysis

## Architecture
- **Recorder**: Captures games from HDHomeRun
- **Processor**: Extracts keyframes and preprocesses video
- **Analyzer**: Runs ML models for video analysis
- **API**: Exposes analysis results to the main dashboard

## Documentation
- [Architecture](docs/ARCHITECTURE.md)
- [HDHomeRun Setup](docs/HDHOMERUN_SETUP.md)
- [Model Training](docs/MODEL_TRAINING.md)
- [API Documentation](docs/API.md)

## Quick Start
Coming soon...

## Hardware Requirements
- HDHomeRun device (for live TV capture)
- Kubernetes cluster (k3s recommended)
- GPU for ML inference (Mac Mini M4 recommended)
- Storage: 500GB+ for video recordings
```

### Step 4: Initial Commit
```bash
git add .
git commit -m "chore: Initialize hockey video analyzer project

- Set up Go module structure
- Create directory layout for microservices
- Add initial README
- Prepare for HDHomeRun integration"

git push origin main
```

---

## 📋 Development Roadmap

### Phase 1: HDHomeRun Recording (2 weeks)
**Goal**: Successfully record NHL games

**Tasks**:
- [ ] Implement HDHomeRun discovery
- [ ] Create recording service
- [ ] Set up storage (Longhorn PVC)
- [ ] Add scheduling based on NHL schedule
- [ ] Test with 5 live games

**Deliverable**: Recorded games stored in cluster

### Phase 2: Video Processing (2 weeks)
**Goal**: Extract and preprocess frames

**Tasks**:
- [ ] Implement FFmpeg Go bindings
- [ ] Create keyframe extraction
- [ ] Add scene segmentation
- [ ] Distribute processing across Raspberry Pis
- [ ] Create processing API

**Deliverable**: Keyframes ready for ML analysis

### Phase 3: Shot Detection (4 weeks)
**Goal**: Detect shots and assess quality

**Tasks**:
- [ ] Collect training data (1000+ shots)
- [ ] Train shot detection model (YOLOv8)
- [ ] Convert to CoreML for Mac Mini
- [ ] Implement inference service
- [ ] Add shot quality scoring
- [ ] Create API endpoint

**Deliverable**: Shot detection working on recorded games

### Phase 4: Advanced Analysis (4 weeks)
**Goal**: Full video analysis pipeline

**Tasks**:
- [ ] Player tracking (DeepSORT)
- [ ] Event classification
- [ ] Zone analysis
- [ ] xG from video
- [ ] API integration with dashboard

**Deliverable**: Complete video analysis available to dashboard

### Phase 5: Real-Time (2+ weeks)
**Goal**: Process live games in real-time

**Tasks**:
- [ ] Optimize inference speed
- [ ] Add streaming processing
- [ ] WebSocket for real-time updates
- [ ] Performance tuning

**Deliverable**: Live game analysis with <30s delay

---

## 🔗 Integration Points with Main Dashboard

### Changes Needed in `hockey_home_dashboard`:

#### 1. Add Video Client Package
**File**: `pkg/videoclient/client.go`
```go
package videoclient

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    baseURL string
    client  *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

func (c *Client) GetGameAnalysis(gameID int) (*VideoAnalysis, error) {
    resp, err := c.client.Get(fmt.Sprintf("%s/api/v1/games/%d/analysis", c.baseURL, gameID))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var analysis VideoAnalysis
    if err := json.NewDecoder(resp.Body).Decode(&analysis); err != nil {
        return nil, err
    }
    
    return &analysis, nil
}
```

#### 2. Add Video Features to Predictions
**File**: `models/predictions.go`
```go
// Add to PredictionFactors struct
VideoShotQuality        float64 `json:"videoShotQuality"`        // 0-1
VideoDefensiveCoverage  float64 `json:"videoDefensiveCoverage"`  // 0-1
VideoForeCheckIntensity float64 `json:"videoForeCheckIntensity"` // 0-1
VideoZoneControl        float64 `json:"videoZoneControl"`        // -1 to 1
```

#### 3. Update Ensemble Service
**File**: `services/ensemble_predictions.go`
```go
// Add video analysis integration
func (eps *EnsemblePredictionService) enrichWithVideoAnalysis(
    homeFactors, awayFactors *models.PredictionFactors,
    gameID int,
) {
    videoClient := GetVideoAnalysisClient()
    if videoClient == nil {
        return // Video analysis not available
    }
    
    analysis, err := videoClient.GetGameAnalysis(gameID)
    if err != nil {
        log.Printf("⚠️ Could not fetch video analysis: %v", err)
        return
    }
    
    // Enrich prediction factors with video data
    homeFactors.VideoShotQuality = analysis.HomeShotQuality
    awayFactors.VideoShotQuality = analysis.AwayShotQuality
    // ... etc
}
```

#### 4. Add Configuration
**File**: `main.go`
```go
// Add video analyzer URL to configuration
var (
    videoAnalyzerURL = os.Getenv("VIDEO_ANALYZER_URL")
    // Default: http://video-analyzer.video-analyzer.svc.cluster.local:8080
)
```

---

## 🖥️ Hardware Setup

### Your Current Cluster:
```
┌──────────────────┐
│ Raspberry Pi x4  │ → k3s worker nodes
└──────────────────┘
┌──────────────────┐
│ Mini PC (GPU)    │ → k3s worker node
└──────────────────┘
```

### Add Mac Mini M4:
```bash
# 1. Install k3s agent on Mac Mini
curl -sfL https://get.k3s.io | K3S_URL=https://192.168.1.99:6443 \
  K3S_TOKEN=your-token sh -

# 2. Label for GPU workloads
kubectl label node mac-mini-m4 gpu=true
kubectl label node mac-mini-m4 gpu-type=apple-m4

# 3. Update video analyzer deployment to use Mac Mini
# See k8s/analyzer-deployment.yaml
```

### Storage Setup (Longhorn):
```bash
# 1. Create namespace
kubectl create namespace video-analyzer

# 2. Create PVC for video storage
kubectl apply -f k8s/video-storage-pvc.yaml

# PVC size: 500GB (adjust based on needs)
```

---

## 📊 Resource Allocation

### Recorder Service
- **CPU**: 0.5 cores
- **Memory**: 512 MB
- **Storage**: Write to PVC
- **Node**: Any Raspberry Pi

### Processor Service
- **CPU**: 2 cores
- **Memory**: 2 GB
- **Storage**: Read/Write PVC
- **Node**: Any Raspberry Pi (distribute across nodes)

### Analyzer Service (ML Inference)
- **CPU**: 4 cores
- **Memory**: 8 GB
- **GPU**: Apple M4 (10 cores)
- **Storage**: Read from PVC
- **Node**: Mac Mini M4 (node selector)

### API Service
- **CPU**: 0.5 cores
- **Memory**: 512 MB
- **Storage**: Read from PVC
- **Node**: Any node

---

## 🔧 Development Tools Needed

### On Your Mac (Development):
```bash
# Install Go (if not already)
brew install go

# Install FFmpeg (for video processing)
brew install ffmpeg

# Install Python (for ML training)
brew install python@3.11

# Install ML libraries
pip install torch torchvision ultralytics onnx coremltools

# Install k3s kubectl context
# (Already have from main app)
```

### On Raspberry Pis:
```bash
# FFmpeg for video processing
sudo apt-get install ffmpeg

# Already have k3s agent installed
```

### On Mac Mini M4 (when added):
```bash
# Install k3s agent (shown above)
# Install FFmpeg
brew install ffmpeg

# Python for CoreML inference (optional)
brew install python@3.11
```

---

## 💡 Quick Wins to Start

### Week 1: Proof of Concept
1. **Create basic HDHomeRun client** in Go
   - Test discovery
   - Test channel selection
   - Record 30 seconds of video

2. **Set up storage**
   - Create Longhorn PVC
   - Test write performance
   - Verify playback

3. **Simple API**
   - List recordings endpoint
   - Health check endpoint

**Goal**: Prove you can record from HDHomeRun to cluster storage

### Week 2: First Full Recording
1. **Schedule-based recording**
   - Integrate with NHL schedule API
   - Auto-start/stop recording

2. **Storage management**
   - Auto-cleanup old recordings
   - Monitor disk usage

3. **Basic processing**
   - Extract 1 frame per second
   - Save as JPEG

**Goal**: Successfully record and process one full NHL game

---

## 📖 Documentation to Create

Create these in the new repo's `docs/` directory:

1. **ARCHITECTURE.md** - Copy from VIDEO_APP_ARCHITECTURE.md
2. **HDHOMERUN_SETUP.md** - How to configure HDHomeRun device
3. **DEVELOPMENT.md** - Local development setup
4. **DEPLOYMENT.md** - Deploy to k3s cluster
5. **API.md** - API documentation
6. **MODEL_TRAINING.md** - How to train ML models

---

## 🎯 Success Criteria

### Phase 1 Success:
- ✅ Record 10 NHL games without data loss
- ✅ Storage < 30 GB for 10 games
- ✅ API accessible from main dashboard
- ✅ Zero downtime during recordings

### Phase 3 Success:
- ✅ Detect 90%+ of shots correctly
- ✅ Shot quality scores are reasonable
- ✅ Processing time < 15 min per game
- ✅ Dashboard displays shot heatmap

### Phase 5 Success:
- ✅ Real-time analysis < 30s behind live
- ✅ Zero impact on recording quality
- ✅ GPU utilization 70-90%
- ✅ Dashboard shows live insights

---

## 🤔 Questions to Answer Before Starting

1. **HDHomeRun Model**: Which model do you have? (Need to know for API compatibility)
2. **Network**: Is HDHomeRun on same network as k3s cluster?
3. **Mac Mini**: When will it be added to cluster?
4. **Storage**: Current Longhorn capacity? How much free?
5. **Priority**: Which NHL team(s) to prioritize for recording?

---

## 💬 Recommended First Steps

### This Week:
1. ✅ Create GitHub repository `hockey_video_analyzer`
2. ✅ Set up basic Go project structure
3. ✅ Test HDHomeRun connectivity from cluster
4. ✅ Create 500GB PVC in Longhorn

### Next Week:
1. Implement basic HDHomeRun client
2. Record first test video
3. Verify storage and playback
4. Document what you learned

### Within Month:
1. Complete Phase 1 (recording)
2. Start Phase 2 (processing)
3. Order/setup Mac Mini M4 (if not yet)

---

## 📞 Ready to Start?

When you're ready to create the new repository, I can help you:

1. **Generate initial code** for HDHomeRun client
2. **Create Kubernetes manifests** for deployment
3. **Write setup scripts** for the cluster
4. **Build first Docker images** for the services

Just let me know when you want to start! 🚀

