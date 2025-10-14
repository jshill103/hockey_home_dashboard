# Docker Hub Information

**Repository**: `jshillingburg/hockey_home_dashboard`  
**URL**: https://hub.docker.com/r/jshillingburg/hockey_home_dashboard

## Common Tags

- `latest` - Most recent stable build
- `league-wide` - League-wide data collection enabled
- Feature-specific tags (e.g., `prediction-fix`, `cleanup`)

## Build & Push Commands

```bash
# Build
docker build -t jshillingburg/hockey_home_dashboard:latest .

# Tag additional versions
docker tag jshillingburg/hockey_home_dashboard:latest jshillingburg/hockey_home_dashboard:league-wide
docker tag jshillingburg/hockey_home_dashboard:latest jshillingburg/hockey_home_dashboard:YOUR_TAG

# Push
docker push jshillingburg/hockey_home_dashboard:latest
docker push jshillingburg/hockey_home_dashboard:league-wide
docker push jshillingburg/hockey_home_dashboard:YOUR_TAG
```

## ⚠️ Important Note

The repository name is **`hockey_home_dashboard`** (NOT `go_uhc`).  
Always use the correct repository name when building and pushing images.

