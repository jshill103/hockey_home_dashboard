# Docker Quick Start - Persistent Storage

## TL;DR

Your accuracy data now persists automatically! 🎉

```bash
# Start with persistent storage
docker-compose up -d

# Your data is automatically saved to: nhl-data-uta volume
# It survives container restarts, rebuilds, and updates!
```

## What Changed?

✅ **Added persistent volume** - `nhl-data-uta` stores all accuracy data  
✅ **Automatic directory creation** - `/app/data/accuracy` created on startup  
✅ **Proper permissions** - Non-root user (UID 1001) has write access  
✅ **Data survives** - Container restarts, rebuilds, and host reboots  

## Quick Commands

### View your data volume
```bash
docker volume ls | grep nhl-data
```

### Backup accuracy data
```bash
docker cp nhl-dashboard:/app/data/accuracy/accuracy_data.json ./backup.json
```

### Check accuracy file in container
```bash
docker exec nhl-dashboard cat /app/data/accuracy/accuracy_data.json
```

### View logs
```bash
docker-compose logs -f
```

## Where is the data stored?

**Inside Container:** `/app/data/accuracy/accuracy_data.json`

**On Host:** Docker manages it automatically  
To find exact location:
```bash
docker volume inspect nhl-data-uta -f '{{ .Mountpoint }}'
```

## How to verify it's working?

1. **Start the container:**
   ```bash
   docker-compose up -d
   ```

2. **Wait for a prediction** (or manually record one)

3. **Check the file was created:**
   ```bash
   docker exec nhl-dashboard ls -lh /app/data/accuracy/
   ```

4. **Restart the container:**
   ```bash
   docker-compose restart
   ```

5. **Verify data persists:**
   ```bash
   docker exec nhl-dashboard cat /app/data/accuracy/accuracy_data.json
   ```

## Data Will Persist Through:

✅ Container restart (`docker-compose restart`)  
✅ Container rebuild (`docker-compose up --build`)  
✅ Host reboot  
✅ Docker daemon restart  
✅ Container removal (`docker-compose down`)  

## Data Will NOT Persist If:

❌ You run `docker volume rm nhl-data-uta`  
❌ You run `docker-compose down -v` (removes volumes)  
❌ You manually delete the volume  

## Multiple Teams?

Each team gets its own volume:

```yaml
# UTA Dashboard
volumes:
  - nhl-data-uta:/app/data

# Colorado Dashboard  
volumes:
  - nhl-data-col:/app/data
```

Separate data = separate accuracy tracking!

## Need More Details?

See `DOCKER_STORAGE.md` for comprehensive documentation.
