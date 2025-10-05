# Docker Persistent Storage Guide

## Overview

The NHL Dashboard uses Docker volumes to persist prediction accuracy data across container restarts and rebuilds. This ensures that your historical accuracy tracking data is never lost.

## Volume Configuration

### Default Setup (docker-compose.yml)

The application automatically creates a named volume for persistent data storage:

```yaml
volumes:
  - nhl-data-uta:/app/data
```

This mounts the `nhl-data-uta` volume to `/app/data` inside the container, where the accuracy tracking data is stored.

## What Data is Persisted?

The following data is stored in the persistent volume:

### Prediction Accuracy Data
- **`/app/data/accuracy/accuracy_data.json`** - Historical prediction accuracy data
  - All predictions made by the system
  - Actual game results
  - Model performance statistics
  - Accuracy trends over time

### Machine Learning Model Parameters
- **`/app/data/models/elo_ratings.json`** - Elo Rating Model state
  - Team ratings (e.g., COL: 1650, UTA: 1450)
  - Rating history for each team
  - Confidence factors per team
  - **Benefit**: Model learns team strengths over time
  
- **`/app/data/models/poisson_rates.json`** - Poisson Regression Model state
  - Team offensive rates (goals scored patterns)
  - Team defensive rates (goals allowed patterns)
  - Rate history for each team
  - Confidence tracking
  - **Benefit**: Model learns scoring/defensive patterns over time

- **`/app/data/models/neural_network_weights.json`** - Neural Network Model state
  - Trained weights and biases
  - Layer configurations
  - Training history
  - **Benefit**: Model learns complex patterns over time

### Game Results Database
- **`/app/data/results/YYYY-MM.json`** - Monthly game results
  - Complete game details (scores, stats, players)
  - Used for model training and retraining
  - Historical game database
  - **Benefit**: System learns from all completed games

### Phase 6: Feature Engineering Data

- **`/app/data/matchups/matchup_index.json`** - Head-to-head matchup history
  - Win/loss records between teams
  - Venue-specific performance
  - Recent matchup trends
  - Rivalry detection
  - **Benefit**: Captures team-specific matchup dynamics (+1-2% accuracy)

- **`/app/data/rolling_stats/rolling_stats.json`** - Advanced rolling statistics
  - Form ratings (0-10 scale)
  - Momentum scores (-1 to +1)
  - Hot/cold streaks
  - Time-weighted performance
  - Quality-weighted stats
  - **Benefit**: Captures current team form and momentum (+1% accuracy)

- **`/app/data/player_impact/player_impact_index.json`** - Player talent tracking
  - Top scorer statistics
  - Star power ratings
  - Depth scoring analysis
  - Player form tracking
  - **Benefit**: Accounts for player talent differentials (+1-2% accuracy)

## Docker Commands

### View Volumes

List all Docker volumes:
```bash
docker volume ls
```

You should see `nhl-data-uta` in the list.

### Inspect Volume

View detailed information about the volume:
```bash
docker volume inspect nhl-data-uta
```

This shows the volume's mountpoint on your host system.

### Backup Accuracy Data

To backup your accuracy data:

**Option 1: Copy from running container**
```bash
docker cp nhl-dashboard:/app/data/accuracy/accuracy_data.json ./backup_accuracy_$(date +%Y%m%d).json
```

**Option 2: Access volume directly**
```bash
# Find the volume mountpoint
VOLUME_PATH=$(docker volume inspect nhl-data-uta -f '{{ .Mountpoint }}')

# Copy the file (may require sudo)
sudo cp "$VOLUME_PATH/accuracy/accuracy_data.json" ./backup_accuracy_$(date +%Y%m%d).json
```

### Restore Accuracy Data

To restore from a backup:

```bash
# Copy backup into running container
docker cp ./backup_accuracy_20251004.json nhl-dashboard:/app/data/accuracy/accuracy_data.json

# Restart container to load the data
docker-compose restart
```

### Clean Up Old Data

To remove all accuracy data and start fresh:

```bash
# Stop the container
docker-compose down

# Remove the volume
docker volume rm nhl-data-uta

# Restart (will create new empty volume)
docker-compose up -d
```

## Multiple Team Dashboards

If running multiple team dashboards, each gets its own volume:

```yaml
services:
  nhl-dashboard:
    volumes:
      - nhl-data-uta:/app/data
  
  nhl-dashboard-avalanche:
    volumes:
      - nhl-data-col:/app/data
  
  nhl-dashboard-knights:
    volumes:
      - nhl-data-vgk:/app/data
```

Each volume maintains separate accuracy tracking data for that team's predictions.

## Volume Driver Options

### Local Driver (Default)

The default `local` driver stores data on the host filesystem:

```yaml
volumes:
  nhl-data-uta:
    driver: local
```

### Alternative Storage Options

For production deployments, you can use different volume drivers:

**Network Storage (NFS)**
```yaml
volumes:
  nhl-data-uta:
    driver: local
    driver_opts:
      type: nfs
      o: addr=your-nfs-server.com,rw
      device: ":/path/to/shared/storage"
```

**Cloud Storage (AWS EFS, Azure Files, etc.)**
```yaml
volumes:
  nhl-data-uta:
    driver: cloudstor:aws
    driver_opts:
      backing: shared
```

## Bind Mounts (Alternative)

Instead of named volumes, you can use bind mounts to map to a specific directory on your host:

```yaml
volumes:
  - ./local_data:/app/data  # Maps to ./local_data on host
```

**Pros:**
- Easy to access and backup
- Files are in a known location on host

**Cons:**
- Path-dependent (not portable)
- May have permission issues
- Less efficient on non-Linux hosts (Windows/Mac)

## Monitoring Data Size

Check the size of your accuracy data:

```bash
# From outside container
docker exec nhl-dashboard du -sh /app/data

# From inside container
docker exec nhl-dashboard ls -lh /app/data/accuracy/
```

## Automated Backups

You can create a cron job to automatically backup accuracy data:

```bash
# Add to crontab (daily backup at 3 AM)
0 3 * * * docker cp nhl-dashboard:/app/data/accuracy/accuracy_data.json /backups/nhl/accuracy_$(date +\%Y\%m\%d).json
```

## Troubleshooting

### Volume Not Persisting Data

1. Check volume exists:
   ```bash
   docker volume inspect nhl-data-uta
   ```

2. Check file exists in container:
   ```bash
   docker exec nhl-dashboard ls -la /app/data/accuracy/
   ```

3. Check permissions:
   ```bash
   docker exec nhl-dashboard ls -la /app/data/
   ```
   - Should be owned by `appuser:appgroup` (UID 1001)

### Permission Denied Errors

If you see permission errors:

```bash
# Fix permissions in container
docker exec nhl-dashboard chown -R appuser:appgroup /app/data
```

### Data Not Loading After Restore

Restart the container to reload data:
```bash
docker-compose restart
```

Check logs for errors:
```bash
docker-compose logs -f nhl-dashboard
```

## Best Practices

1. **Regular Backups**: Backup accuracy data weekly during the season
2. **Monitor Size**: Large accuracy files (>10MB) may need optimization
3. **Named Volumes**: Use named volumes instead of bind mounts for better portability
4. **Separate Volumes**: Use separate volumes for each team dashboard
5. **Version Control**: Don't commit `data/` directory to git (already in `.gitignore`)

## Data Location

- **In Container**: `/app/data/accuracy/accuracy_data.json`
- **On Host** (named volume): Use `docker volume inspect` to find mountpoint
- **On Host** (bind mount): The path you specified in `docker-compose.yml`

## Migration

To migrate data between hosts:

1. **Backup on old host:**
   ```bash
   docker cp nhl-dashboard:/app/data/accuracy/accuracy_data.json ./accuracy_backup.json
   ```

2. **Copy to new host:**
   ```bash
   scp accuracy_backup.json newhost:/path/to/backup/
   ```

3. **Restore on new host:**
   ```bash
   docker cp ./accuracy_backup.json nhl-dashboard:/app/data/accuracy/accuracy_data.json
   docker-compose restart
   ```

## Security Considerations

- Accuracy data is stored inside the container (not exposed to host network)
- Volume data persists with proper permissions (non-root user: UID 1001)
- Backups should be stored securely (contains prediction history)
- Consider encrypting backup files if storing sensitive analysis data

