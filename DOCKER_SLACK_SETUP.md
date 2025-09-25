# 🐳 Docker + Slack Configuration Guide

## 🚀 Quick Start with Slack Notifications

The Docker setup now fully supports the **centralized Slack configuration system**! You can enable Utah Mammoth Team Store notifications using environment variables.

### **Method 1: Docker Compose with Environment Variables**

1. **Get your Slack webhook URL**:
   - Go to: https://api.slack.com/apps/A09GHT50BFW
   - Click "Incoming Webhooks" → "Add New Webhook to Workspace"
   - Select your channel → Copy the webhook URL

2. **Create/Edit `docker-compose.yml`**:
   ```yaml
   version: '3.8'
   
   services:
     nhl-dashboard:
       image: jshillingburg/hockey_home_dashboard:latest
       ports:
         - "8080:8080"
       environment:
         - TEAM_CODE=UTA
         - SLACK_WEBHOOK_URL=https://example.com/your-slack-webhook-url
       restart: unless-stopped
   ```

3. **Start the container**:
   ```bash
   docker-compose up -d
   ```

### **Method 2: Direct Docker Run**

```bash
docker run -d \
  --name utah-mammoth-dashboard \
  -p 8080:8080 \
  -e TEAM_CODE=UTA \
   -e SLACK_WEBHOOK_URL="https://example.com/your-slack-webhook-url" \
  --restart unless-stopped \
  jshillingburg/hockey_home_dashboard:latest
```

### **Method 3: Environment File (.env)**

1. **Create `.env` file**:
   ```bash
   TEAM_CODE=UTA
   SLACK_WEBHOOK_URL=https://example.com/your-slack-webhook-url
   ```

2. **Update `docker-compose.yml`**:
   ```yaml
   version: '3.8'
   
   services:
     nhl-dashboard:
       image: jshillingburg/hockey_home_dashboard:latest
       ports:
         - "8080:8080"
       env_file: .env
       restart: unless-stopped
   ```

3. **Start with compose**:
   ```bash
   docker-compose up -d
   ```

## 🔧 Configuration Options

### **Environment Variables**

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `TEAM_CODE` | NHL team code (UTA, COL, VGK, etc.) | No | `UTA` |
| `SLACK_WEBHOOK_URL` | Slack webhook URL for notifications | No | None |

### **Slack Notifications**

- **Only available for UTA team** (Utah Mammoth)
- **Monitors**: Mammoth Team Store for new products
- **Frequency**: Every 15 minutes
- **Alerts**: New products with prices, images, and links

## 🧪 Testing Your Setup

### **1. Check Container Logs**
```bash
docker logs utah-mammoth-dashboard
```

**Look for these messages:**
```
✅ Slack webhook configured for Utah Mammoth notifications
✅ Slack notifications enabled for new Utah Mammoth products!
```

**Or if not configured:**
```
⚠️ Slack notifications disabled - no webhook URL configured
```

### **2. Test Slack Notifications**
```bash
curl -X POST http://localhost:8080/api/slack/test
```

**Success response:**
```json
{
  "status": "success",
  "message": "Test Slack notification sent successfully!"
}
```

### **3. Access Scraper Dashboard**
Visit: http://localhost:8080/scraper-dashboard

## 🏗️ Building Your Own Image

If you want to build the image locally with your Slack configuration:

### **1. Build with Build Args**
```bash
docker build \
  --build-arg SLACK_WEBHOOK_URL="https://example.com/your-slack-webhook-url" \
  -t my-hockey-dashboard .
```

### **2. Create Custom Dockerfile**
```dockerfile
FROM jshillingburg/hockey_home_dashboard:latest

# Set your Slack webhook URL
ENV SLACK_WEBHOOK_URL=https://example.com/your-slack-webhook-url
ENV TEAM_CODE=UTA
```

## 🔍 Troubleshooting

### **❌ "Slack notifications disabled"**
**Problem**: Container logs show Slack is disabled
**Solution**: 
- Check your `SLACK_WEBHOOK_URL` environment variable
- Ensure URL starts with `https://hooks.slack...` (your actual Slack domain)
- Verify the webhook URL is correct in Slack

### **❌ "No Slack webhook URL configured"** 
**Problem**: API test returns error
**Solution**: 
```bash
# Check environment variables in container
docker exec utah-mammoth-dashboard env | grep SLACK

# Restart container with correct URL
docker-compose down
# Edit docker-compose.yml with correct SLACK_WEBHOOK_URL
docker-compose up -d
```

### **❌ "Failed to send Slack message"**
**Problem**: Webhook URL is set but messages fail to send
**Solutions**:
- Verify the webhook URL is correct and active
- Check Slack app permissions in your workspace
- Test the webhook URL manually with curl

### **❌ Container Won't Start**
**Problem**: Build or runtime errors
**Solutions**:
```bash
# Check logs
docker logs utah-mammoth-dashboard

# Rebuild image with latest code
docker-compose build --no-cache
docker-compose up -d
```

## 📊 Multi-Team Setup with Slack

You can run multiple team dashboards, but **Slack notifications only work for UTA**:

```yaml
version: '3.8'

services:
  # Utah with Slack notifications
  utah-dashboard:
    image: jshillingburg/hockey_home_dashboard:latest
    ports:
      - "8080:8080"
    environment:
      - TEAM_CODE=UTA
      - SLACK_WEBHOOK_URL=https://example.com/your-slack-webhook-url
    restart: unless-stopped

  # Colorado (no Slack notifications available)
  colorado-dashboard:
    image: jshillingburg/hockey_home_dashboard:latest
    ports:
      - "8081:8080"
    environment:
      - TEAM_CODE=COL
    restart: unless-stopped

  # Vegas (no Slack notifications available)
  vegas-dashboard:
    image: jshillingburg/hockey_home_dashboard:latest
    ports:
      - "8082:8080"
    environment:
      - TEAM_CODE=VGK
    restart: unless-stopped
```

## 🎯 What You'll Get

Once configured correctly, your Utah Mammoth dashboard will:

- ✅ **Automatically monitor** the Mammoth Team Store every 15 minutes
- ✅ **Detect new products** and price changes
- ✅ **Send rich Slack notifications** with product images, prices, and direct purchase links
- ✅ **Never spam** - only new items trigger alerts
- ✅ **Run 24/7** in Docker with automatic restart on failure

**Your containerized Utah Mammoth Team Store monitoring is now ready!** 🏒🐳

## 📝 Security Best Practices

1. **Use environment files** (`.env`) instead of hardcoding webhook URLs
2. **Restrict webhook permissions** to specific channels in Slack
3. **Regularly rotate webhook URLs** for security
4. **Monitor container logs** for any security issues
5. **Keep Docker images updated** with `docker-compose pull`

Your webhook URLs contain sensitive tokens - treat them like passwords!
