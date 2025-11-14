# Hockey Video Analyzer - Architecture Plan

## Overview
Separate microservice for HDHomeRun video capture, processing, and ML-based video analysis of NHL games.

---

## 🎯 Purpose

### What It Does:
- Captures live/recorded NHL games from HDHomeRun
- Processes video to extract hockey-specific events
- Analyzes player movements, shot quality, defensive coverage
- Provides video-based insights to the main dashboard

### What It Doesn't Do:
- Game predictions (that's the main app)
- Statistics tracking (that's the main app)
- Web UI for users (that's the main app)

---

## 🏗️ Repository Structure

```
hockey_video_analyzer/
├── cmd/
│   ├── recorder/           # HDHomeRun recording service
│   ├── processor/          # Video processing service
│   └── analyzer/           # ML analysis service
├── internal/
│   ├── capture/           # HDHomeRun integration
│   │   ├── hdhomerun.go   # HDHomeRun API client
│   │   ├── stream.go      # Video stream handling
│   │   └── recorder.go    # Recording logic
│   ├── processing/        # Video processing
│   │   ├── keyframes.go   # Keyframe extraction
│   │   ├── preprocessing.go # Video preprocessing
│   │   └── segmentation.go  # Scene segmentation
│   ├── analysis/          # ML analysis
│   │   ├── shot_detector.go     # Shot detection & quality
│   │   ├── player_tracker.go    # Player tracking
│   │   ├── event_classifier.go  # Event classification
│   │   └── zone_analyzer.go     # Ice zone analysis
│   ├── models/            # ML model management
│   │   ├── yolo.go        # Object detection (YOLO)
│   │   ├── action.go      # Action recognition
│   │   └── loader.go      # Model loading/inference
│   └── storage/           # Video & data storage
│       ├── video_store.go # Video file management
│       └── results_store.go # Analysis results
├── pkg/
│   ├── api/               # REST API for results
│   └── client/            # Client library for dashboard
├── models/                # Trained ML models
│   ├── shot_detector/
│   ├── player_tracker/
│   └── event_classifier/
├── scripts/
│   ├── setup_hdhomerun.sh
│   └── train_models.sh
├── k8s/
│   ├── recorder-deployment.yaml
│   ├── processor-deployment.yaml
│   ├── analyzer-deployment.yaml
│   └── storage-pvc.yaml
├── docker/
│   ├── Dockerfile.recorder
│   ├── Dockerfile.processor
│   └── Dockerfile.analyzer
├── go.mod
├── go.sum
├── README.md
└── docs/
    ├── ARCHITECTURE.md
    ├── HDHOMERUN_SETUP.md
    ├── MODEL_TRAINING.md
    └── API.md
```

---

## 🔄 Data Flow

```
HDHomeRun → Recorder → Storage → Processor → Analyzer → API → Dashboard
```

### 1. Recording Service (cmd/recorder)
- Monitors HDHomeRun for NHL games
- Streams video to storage (Longhorn PVC)
- Handles channel selection & scheduling
- **Output**: Raw video files (MPEG-TS)

### 2. Processing Service (cmd/processor)
- Extracts keyframes from recorded video
- Preprocesses for ML analysis
- Segments into events/scenes
- **Output**: Processed frames & metadata

### 3. Analysis Service (cmd/analyzer)
- Runs ML models on processed frames
- Detects shots, passes, hits, etc.
- Tracks player positions & movements
- Analyzes zone time & quality chances
- **Output**: Video analysis results (JSON)

### 4. API Service
- Exposes analysis results via REST API
- Consumed by main hockey dashboard
- WebSocket for real-time updates (optional)

---

## 🧠 ML Models & Analysis

### Phase 1: Shot Detection & Quality
**Model**: Custom CNN or YOLOv8
**Input**: Video frames
**Output**: 
- Shot detected (yes/no)
- Shot location (x, y coordinates)
- Shot type (wrist, slap, snapshot, tip, etc.)
- Shot quality score (0-1)
- Goalie positioning quality

**Training Data**: 
- NHL condensed games (shots highlighted)
- Manual annotation of ~1000 shots
- Transfer learning from existing hockey models

### Phase 2: Player Tracking
**Model**: DeepSORT + Custom tracker
**Input**: Video frames
**Output**:
- Player positions over time
- Player speed & acceleration
- Formation analysis
- Zone entries/exits

### Phase 3: Event Classification
**Model**: Action recognition (3D CNN or LSTM)
**Input**: Video sequences
**Output**:
- Event type (shot, pass, hit, faceoff, etc.)
- Event quality/effectiveness
- Contextual analysis

### Phase 4: Advanced Analysis
**Models**: Custom ensemble
**Output**:
- Defensive coverage quality
- Forecheck effectiveness
- Power play formation analysis
- Shot assist quality (how good was the pass?)
- Expected goals (xG) from video

---

## 🖥️ Hardware Requirements

### Current Home Cluster:
- **Raspberry Pis**: Recording & preprocessing
- **Mini PC with GPU**: ML inference
- **Mac Mini M4 (10 GPU cores, 32GB RAM)**: Primary ML processing

### Distributed Architecture:
```
┌─────────────────┐
│ Raspberry Pi #1 │ → HDHomeRun Recording
└─────────────────┘
         ↓
┌─────────────────┐
│ Longhorn Storage│ → Shared video storage
└─────────────────┘
         ↓
┌─────────────────┐
│ Raspberry Pi #2 │ → Keyframe extraction
└─────────────────┘
         ↓
┌─────────────────┐
│ Mac Mini M4     │ → ML inference (CoreML)
│ (Primary GPU)   │
└─────────────────┘
         ↓
┌─────────────────┐
│ Mini PC GPU     │ → Backup ML inference
│ (Secondary)     │
└─────────────────┘
```

---

## 📡 Communication with Main Dashboard

### Option 1: REST API (Recommended)
**Video Analyzer → Dashboard**

```go
// Dashboard queries video analyzer
GET /api/v1/games/{gameId}/video-analysis
Response:
{
  "gameId": 2025020123,
  "shots": [
    {
      "time": "12:34",
      "period": 2,
      "team": "UTA",
      "player": "#9 Smith",
      "quality": 0.78,
      "type": "wrist",
      "result": "goal",
      "expectedGoal": 0.65
    }
  ],
  "zoneTime": {
    "offensiveZone": 312.5,
    "defensiveZone": 289.3,
    "neutralZone": 156.2
  },
  "playerTracking": {...}
}
```

### Option 2: Message Queue (Advanced)
- Video analyzer publishes events to NATS/RabbitMQ
- Dashboard subscribes to relevant events
- Real-time updates during live games

### Option 3: Shared Database
- Video analyzer writes to PostgreSQL
- Dashboard reads from same database
- Simpler but tighter coupling

**Recommendation**: Start with Option 1 (REST API), migrate to Option 2 if real-time becomes critical.

---

## 🎬 HDHomeRun Integration

### Discovery & Connection
```go
// internal/capture/hdhomerun.go
type HDHomeRunClient struct {
    deviceIP   string
    tuner      int
    channel    string
}

func (h *HDHomeRunClient) Discover() ([]Device, error)
func (h *HDHomeRunClient) GetChannel(channel string) (*Stream, error)
func (h *HDHomeRunClient) StartRecording(channel, output string) error
```

### Channel Mapping
```go
// Map NHL games to HDHomeRun channels
var ChannelMap = map[string]string{
    "ESPN":  "706",  // Example channel numbers
    "TNT":   "245",
    "ABC":   "7",
    "LOCAL": "5",    // Local broadcast
}
```

### Recording Strategy
1. **Pre-Game**: Start recording 15 min before game
2. **Live**: Record full game + 30 min overtime buffer
3. **Post-Game**: Stop recording 30 min after scheduled end
4. **Storage**: Keep for 7 days, then delete

---

## 🗄️ Storage Strategy

### Video Files
- **Location**: Longhorn PVC (replicated across cluster)
- **Format**: MPEG-TS (from HDHomeRun)
- **Retention**: 7 days for full video
- **Size**: ~2-3 GB per game (HD quality)

### Processed Frames
- **Location**: Same PVC, separate directory
- **Format**: JPEG keyframes
- **Retention**: 30 days
- **Size**: ~100-200 MB per game

### Analysis Results
- **Location**: JSON files or PostgreSQL
- **Format**: JSON
- **Retention**: Permanent
- **Size**: ~1-5 MB per game

### Total Storage Needs
- **Per Game**: ~3 GB
- **Active Games** (7 days): ~150 GB
- **Processed Data**: ~50 GB
- **Recommended PVC Size**: 500 GB (with buffer)

---

## 🚀 Development Phases

### Phase 1: Basic Recording (Week 1-2)
- [ ] HDHomeRun integration
- [ ] Schedule-based recording
- [ ] Storage management
- [ ] Basic API for recorded games

**Deliverable**: Can record and store NHL games from HDHomeRun

### Phase 2: Video Processing (Week 3-4)
- [ ] Keyframe extraction
- [ ] Scene segmentation
- [ ] Preprocessing pipeline
- [ ] Distributed processing (Raspberry Pis)

**Deliverable**: Raw video → Processed frames ready for ML

### Phase 3: Shot Detection (Week 5-8)
- [ ] Train shot detection model
- [ ] Implement inference on Mac Mini
- [ ] Shot quality scoring
- [ ] API endpoint for shot data

**Deliverable**: Detect shots and quality from video

### Phase 4: Advanced Analysis (Week 9-12)
- [ ] Player tracking
- [ ] Event classification
- [ ] Zone analysis
- [ ] xG from video

**Deliverable**: Full video analysis integrated with dashboard

### Phase 5: Real-Time Processing (Week 13+)
- [ ] Live game processing
- [ ] Real-time event detection
- [ ] WebSocket updates to dashboard
- [ ] Performance optimization

**Deliverable**: Real-time video analysis during live games

---

## 🔧 Technology Stack

### Backend
- **Language**: Go (same as main app)
- **Video Processing**: FFmpeg (via Go bindings)
- **ML Framework**: 
  - CoreML (Mac Mini M4)
  - ONNX Runtime (cross-platform)
  - TensorFlow Lite (Raspberry Pi inference if needed)

### ML Development
- **Training**: Python (PyTorch/TensorFlow)
- **Deployment**: ONNX → CoreML/TFLite
- **Models**: 
  - YOLOv8 (object detection)
  - DeepSORT (tracking)
  - Custom CNNs (action recognition)

### Storage
- **Video**: Longhorn (Kubernetes PVC)
- **Metadata**: PostgreSQL or JSON files
- **Cache**: Redis (optional, for real-time)

### Deployment
- **Orchestration**: Kubernetes (k3s)
- **Services**: 3 separate deployments (recorder, processor, analyzer)
- **Load Balancing**: CoreDNS + Service mesh

---

## 📊 Performance Targets

### Recording
- **Latency**: < 10 seconds behind live
- **Reliability**: 99.9% uptime during games
- **Storage I/O**: Sustained 50 MB/s writes

### Processing
- **Throughput**: 1 game processed in < 10 minutes
- **Keyframe Rate**: 1 frame per second
- **CPU Usage**: < 50% on Raspberry Pi

### Analysis (ML Inference)
- **Shot Detection**: < 100ms per frame
- **Player Tracking**: < 200ms per frame
- **Full Game Analysis**: < 30 minutes for 2.5 hour game
- **GPU Utilization**: 70-90% on Mac Mini

### API Response
- **Get Analysis**: < 500ms
- **List Games**: < 100ms
- **Concurrent Requests**: 100 req/s

---

## 🔌 API Design

### Endpoints

#### Games
```
GET  /api/v1/games                    # List recorded games
GET  /api/v1/games/{id}               # Get game info
POST /api/v1/games/{id}/analyze       # Trigger analysis
GET  /api/v1/games/{id}/status        # Analysis status
```

#### Video Analysis
```
GET /api/v1/games/{id}/shots          # Shot data
GET /api/v1/games/{id}/players        # Player tracking
GET /api/v1/games/{id}/events         # All events
GET /api/v1/games/{id}/zones          # Zone analysis
GET /api/v1/games/{id}/xg             # Expected goals from video
```

#### Recording
```
GET  /api/v1/recordings               # List recordings
POST /api/v1/recordings/schedule      # Schedule a recording
GET  /api/v1/recordings/{id}/stream   # Stream video (optional)
```

#### Health & Metrics
```
GET /health                           # Service health
GET /metrics                          # Prometheus metrics
```

---

## 🔐 Security Considerations

1. **Network Isolation**: Video analyzer in separate k8s namespace
2. **API Authentication**: JWT tokens for dashboard access
3. **Storage Encryption**: Encrypt video at rest (if needed)
4. **Rate Limiting**: Prevent API abuse
5. **RBAC**: Kubernetes RBAC for service accounts

---

## 📈 Monitoring & Observability

### Metrics to Track
- Recording uptime & failures
- Processing queue length
- ML inference latency
- API request rate & latency
- Storage usage & growth
- GPU utilization

### Tools
- **Prometheus**: Metrics collection
- **Grafana**: Dashboards
- **Loki**: Log aggregation
- **Jaeger**: Distributed tracing (if using microservices)

---

## 🧪 Testing Strategy

### Unit Tests
- HDHomeRun API client
- Video processing functions
- ML model inference
- API handlers

### Integration Tests
- End-to-end recording pipeline
- Storage operations
- Model loading & inference
- API endpoints

### Load Tests
- Concurrent recordings
- Processing throughput
- API performance under load

---

## 💰 Cost Considerations

### Hardware (Already Owned)
- ✅ HDHomeRun device
- ✅ Raspberry Pis
- ✅ Mini PC with GPU
- ✅ Mac Mini M4 (to be added)

### Storage
- Longhorn PVC: Using existing cluster storage
- Video retention: 7 days to manage costs

### Compute
- Local cluster: No cloud costs
- Power consumption: Estimate ~$20/month

### Development
- ML model training: Use existing hardware
- No cloud GPU costs (using Mac Mini M4)

**Total Monthly Cost**: ~$20 (electricity only)

---

## 🎯 Success Metrics

### Phase 1 Success
- ✅ Successfully record 10 games
- ✅ Store without data loss
- ✅ API accessible from dashboard

### Phase 3 Success
- ✅ Detect 90%+ of shots accurately
- ✅ Shot quality scores correlate with xG
- ✅ Dashboard displays video-based insights

### Phase 5 Success
- ✅ Real-time analysis during live games
- ✅ < 30 second delay from broadcast
- ✅ Enhanced predictions using video data

---

## 🔗 Integration with Main Dashboard

### Dashboard Updates Needed

#### 1. Add Video Analysis Client
```go
// pkg/videoclient/client.go in main dashboard
type VideoAnalysisClient struct {
    baseURL string
    client  *http.Client
}

func (v *VideoAnalysisClient) GetGameAnalysis(gameID int) (*VideoAnalysis, error)
```

#### 2. Enhance Predictions
```go
// services/ensemble_predictions.go
// Add video-based features to prediction factors
type PredictionFactors struct {
    // ... existing fields ...
    
    // New: Video-based features
    VideoShotQuality    float64
    VideoDefensiveCoverage float64
    VideoForeCheckIntensity float64
    VideoPlayerMovement  float64
}
```

#### 3. New API Endpoints in Dashboard
```go
// handlers/video_analysis.go
GET /api/video/games/{id}/analysis    # Proxy to video analyzer
GET /api/video/games/{id}/highlights  # Video highlights
```

#### 4. UI Updates
- Add "Video Analysis" tab to game pages
- Display shot quality heatmap
- Show player tracking visualizations
- Highlight key moments from video

---

## 📚 Documentation Needs

1. **HDHOMERUN_SETUP.md** - How to configure HDHomeRun
2. **MODEL_TRAINING.md** - Training ML models
3. **DEPLOYMENT.md** - Deploying to k3s cluster
4. **API.md** - API documentation
5. **ARCHITECTURE.md** - Detailed architecture
6. **CONTRIBUTING.md** - Development guidelines

---

## 🎉 Summary

This architecture provides:
- ✅ Separation of concerns (video processing separate from predictions)
- ✅ Scalable (distributed processing across cluster)
- ✅ Cost-effective (local hardware, no cloud costs)
- ✅ Flexible (can add more ML models later)
- ✅ Maintainable (clear boundaries between services)

**Next Steps**:
1. Create new repository: `hockey_video_analyzer`
2. Set up basic Go project structure
3. Start with Phase 1: HDHomeRun recording
4. Iterate through phases

**Estimated Timeline**: 12-16 weeks to full production

