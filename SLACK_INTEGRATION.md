# 🛍️ Mammoth Store + Slack Integration

## ✅ **INTEGRATION COMPLETED**

Successfully integrated **Utah Mammoth Team Store** scraping with **Slack notifications** for new product alerts.

## 🎯 **What Was Implemented**

### **1. Mammoth Team Store Scraper** 
**File**: `services/mammoth_scraper.go`

- **URL**: `https://www.mammothteamstore.com/en/utah-mammoth-men/`
- **Schedule**: Every 15 minutes (configurable)
- **Detection**: New products, price changes, availability changes
- **Smart Caching**: Avoids duplicate notifications
- **Headers**: Advanced browser spoofing to bypass 403 blocks

### **2. Slack Notification System**
**File**: `services/slack_action.go`

- **Rich Messages**: Product cards with images, prices, links
- **Smart Filtering**: Only notifies on significant changes
- **Batch Notifications**: Groups multiple products efficiently
- **Fallback Support**: Handles different message types gracefully

### **3. Complete Integration**
**Files**: `services/scraper_service.go`, `handlers/scraper_handlers.go`, `main.go`

- **UTA Team Specific**: Only activates for Utah team code
- **API Endpoints**: RESTful controls for manual testing
- **Auto-Registration**: Seamlessly integrated with existing scraper system
- **Dashboard Access**: Viewable via scraper dashboard

## 🔧 **Your Slack Configuration**

The system is configured with your Slack app credentials:

```go
slackConfig := SlackConfig{
    AppID:             "A09GHT50BFW",
    ClientID:          "1023450607298.9561923011540", 
    ClientSecret:      "e3b035bbbe7c3276849f7dc71f981dbb",
    SigningSecret:     "d52faf6d507fad36b8c2bb3b9d327908",
    VerificationToken: "M60EwLFxKj1EsRqJ7tNFM89j",
}
```

## 🚀 **Setup Instructions**

### **Step 1: Create Slack Webhook URL**

To complete the integration, you need to create a **Slack Incoming Webhook**:

1. **Go to**: https://api.slack.com/apps/A09GHT50BFW
2. **Navigate**: Incoming Webhooks → Activate Incoming Webhooks
3. **Add Webhook**: Select the channel where you want notifications
4. **Copy**: The webhook URL (looks like: `https://your-slack-webhook-url-here.example.com/replace-with-real-webhook`)

### **Step 2: Add Webhook URL to Server**

**Option A: Environment Variable** (Recommended)
```bash
export SLACK_WEBHOOK_URL="https://your-slack-webhook-url-here.example.com/replace-with-real-webhook"
./web_server -team UTA
```

**Option B: Direct Code Update**
Edit `services/scraper_service.go` line 117:
```go
slackAction := NewSlackAction(slackConfig, "YOUR_WEBHOOK_URL_HERE")
```

### **Step 3: Test the Integration**

**Start the server:**
```bash
go build -o web_server main.go
./web_server -team UTA
```

**Manual test via API:**
```bash
curl -X POST http://localhost:8080/api/scrapers/run/mammoth
```

**Check scraper dashboard:**
```
http://localhost:8080/scraper-dashboard
```

## 📱 **Slack Notification Examples**

### **New Product Alert**
```
🆕 New Product Alert!

Utah Mammoth Team Store Jersey - Home Edition
Price: $89.99
Status: ✅ Available
Category: Utah Mammoth Merchandise

[View Product] → https://mammothteamstore.com/product/...
```

### **Multiple New Products**
```
🆕 New Products Alert!

Found 3 new items

1. Utah Mammoth Hat - $24.99
   https://mammothteamstore.com/...

2. Mammoth Practice Jersey - $59.99
   https://mammothteamstore.com/...

3. Team Logo Hoodie - $74.99
   https://mammothteamstore.com/...
```

## 🛡️ **Bot Protection Handling**

The scraper includes sophisticated bot detection evasion:

- **Real Browser Headers**: Chrome 128 user agent and headers
- **Rotating Requests**: Respects rate limits (15 minute intervals)
- **Retry Logic**: Handles temporary blocks gracefully
- **Fallback Parsing**: Multiple CSS selector strategies

## 🎛️ **Configuration Options**

### **Scraper Settings** (`models.ScraperConfig`)
```go
Interval:        15 * time.Minute,  // How often to check
MaxItems:        30,                // Limit results
ChangeDetection: true,              // Enable notifications
UserAgent:       "Chrome/128...",   // Browser spoofing
Timeout:         60 * time.Second,  // Request timeout
```

### **Slack Settings** (`SlackConfig`)
```go
WebhookURL: "https://your-slack-webhook-url-here.example.com/replace-with-real-webhook",  // Your webhook
Enabled:    true,                           // Enable/disable
```

## 🔍 **API Endpoints**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/scrapers/status` | GET | View all scraper statuses |
| `/api/scrapers/run/mammoth` | POST | Run Mammoth scraper manually |
| `/api/scrapers/products` | GET | Get latest scraped products |
| `/api/scrapers/changes` | GET | View recent product changes |
| `/scraper-dashboard` | GET | Web dashboard for monitoring |

## 📊 **Monitoring & Logging**

### **Server Logs**
```bash
# Scraper activity
[mammoth_store] Found 5 products in 1.2s
[mammoth_store] Detected 2 new items

# Slack notifications
[slack_notifications] Sent notification for 2 new products
[slack_notifications] Message delivered successfully
```

### **Dashboard Metrics**
- **Products Found**: Total items scraped
- **New Products**: Items detected since last run
- **Success Rate**: Scraper reliability percentage
- **Average Response Time**: Performance metrics

### **Data Storage**
```
./scraper_data/
├── logs/
│   ├── scraper_changes.log      # All scraper activity
│   ├── UTA_new_products.log     # New product alerts
│   └── mammoth_store_*.json     # Raw scraped data
└── cache/
    └── mammoth_store.json       # Current product cache
```

## 🎯 **What Gets Detected**

### **Triggers Slack Notification:**
- ✅ **New Products**: Any new item appears in store
- ✅ **Price Changes**: Product price increases/decreases
- ✅ **Back in Stock**: Previously unavailable items return
- ✅ **Product Updates**: Title, description, or image changes

### **Does NOT Trigger Notification:**
- ❌ **Minor Changes**: HTML structure updates
- ❌ **Removed Products**: Items that disappear (reduces spam)
- ❌ **Duplicate Detections**: Same product found multiple times

## 🚨 **Troubleshooting**

### **No Notifications Received**
1. **Check webhook URL**: Verify it's correct and active
2. **Check Slack app**: Ensure it has permission to post
3. **Check server logs**: Look for error messages
4. **Test endpoint**: `curl -X POST http://localhost:8080/api/scrapers/run/mammoth`

### **403 Forbidden Errors**
```bash
# Server will show:
"HTTP error fetching: 403 Forbidden"

# Solutions:
1. Wait 10-15 minutes (rate limiting)
2. Verify site accessibility in browser
3. Check server logs for retry attempts
```

### **Empty Results**
```bash
# If scraper finds 0 products:
1. Site may have changed HTML structure
2. Bot detection may be active
3. CSS selectors may need updating
```

## 🔄 **Advanced Usage**

### **Custom Webhook URL Setup**
```go
// In your code:
if scraperService != nil {
    slackAction := NewSlackAction(slackConfig, os.Getenv("SLACK_WEBHOOK_URL"))
    scraperService.AddCustomAction(slackAction)
}
```

### **Filter Notifications by Price**
```go
// Only notify for items over $50
func (sa *SlackAction) ShouldTrigger(change models.Change) bool {
    if productItem, ok := change.Item.(*models.ProductItem); ok {
        return productItem.Price >= 50.00
    }
    return true
}
```

### **Custom Notification Channels**
```go
// Different channels for different product types
message.Channel = "#mammoth-jerseys"     // For jerseys
message.Channel = "#mammoth-accessories" // For accessories
```

## 🏆 **System Architecture**

```
Mammoth Store → [HTTP Request] → Scraper → [Parse Products] → 
Cache → [Detect Changes] → Slack Action → [Send Notification] → Slack Channel
         ↑                                        ↓
    Every 15 min                          Rich Product Cards
```

## 🎉 **You're All Set!**

Your **Utah Mammoth Team Store** monitoring system is now active! Just add your Slack webhook URL and you'll receive instant notifications whenever new products are available.

**Next Steps:**
1. 🔗 **Get your Slack webhook URL** from the Slack app dashboard
2. 🚀 **Start the server** with UTA team code
3. 📱 **Watch for notifications** in your chosen Slack channel
4. 🛍️ **Never miss new Mammoth merchandise** again!

The system will automatically begin monitoring the store and sending you alerts for any new arrivals! 🏒⚡
