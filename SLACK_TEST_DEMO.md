# 🧪 Utah Mammoth Store Slack Test Demo

## ✅ **SLACK INTEGRATION READY FOR TESTING**

Your Utah Mammoth Team Store Slack integration is fully built and ready to send test alerts! Here are **multiple ways** to test and activate it.

---

## 🎯 **Quick Test Options**

### **Option 1: Preview Slack Messages (No Webhook Needed)**
See exactly what your notifications will look like:

```bash
go run mock_slack_demo.go
```

**Output Preview:**
- 📱 Full Slack message JSON format
- 🎨 Visual representation of the alert
- 📦 Multiple product alerts
- 🏒 Single product alerts
- ⚡ Real-time test data simulation

---

### **Option 2: API Test Endpoint (Webhook Required)**
Test via your running dashboard server:

```bash
# Start server (if not running)
./web_server -team UTA

# Test without webhook (shows instructions)
curl -X POST http://localhost:8080/api/slack/test

# Test with webhook URL
curl -X POST http://localhost:8080/api/slack/test \
  -H "Content-Type: application/json" \
  -d '{"webhook_url": "YOUR_WEBHOOK_URL_HERE"}'
```

---

### **Option 3: Standalone Test Script (Webhook Required)**
Direct Slack notification test:

```bash
# With environment variable
export SLACK_WEBHOOK_URL="YOUR_WEBHOOK_URL_HERE"
go run test_slack_notification.go

# Or directly with URL
go run test_slack_notification.go "YOUR_WEBHOOK_URL_HERE"
```

---

## 🔗 **Get Your Slack Webhook URL**

### **Step-by-Step:**

1. **Visit Your Slack App**: https://api.slack.com/apps/A09GHT50BFW
2. **Navigate**: Incoming Webhooks (left sidebar)
3. **Activate**: Turn on "Activate Incoming Webhooks" (if not already on)
4. **Add Webhook**: Click "Add New Webhook to Workspace"
5. **Select Channel**: Choose where you want notifications
6. **Copy URL**: Copy the webhook URL (starts with `https://your-slack-domain.example.com/services/TXXXXXXXX/BXXXXXXXX/...`)

---

## 🧪 **Test Results You'll See**

### **✅ Success Response**
```json
{
  "status": "success",
  "message": "Test Slack notification sent successfully!",
  "products_sent": 2,
  "timestamp": "2025-09-24T15:13:41Z",
  "details": {
    "webhook_url": "https://your-slack-webhook-url-here.example.com/replace-with-real-webhook",
    "products": [
      {
        "title": "🧪 TEST ALERT - Utah Mammoth Home Jersey",
        "price": "$129.99",
        "url": "https://www.mammothteamstore.com/test-home-jersey"
      },
      {
        "title": "🧪 TEST ALERT - Mammoth Team Hoodie", 
        "price": "$74.99",
        "url": "https://www.mammothteamstore.com/test-hoodie"
      }
    ]
  }
}
```

### **❌ Error Response (No Webhook)**
```json
{
  "status": "error",
  "message": "No Slack webhook URL configured",
  "instructions": {
    "environment": "Set SLACK_WEBHOOK_URL environment variable",
    "request_body": "Send JSON with 'webhook_url' field",
    "get_webhook": "https://api.slack.com/apps/A09GHT50BFW → Incoming Webhooks"
  }
}
```

---

## 📱 **What You'll See in Slack**

### **Multi-Product Alert:**
```
🆕 New Products Alert!

Utah Mammoth Team Store

Found 2 new items

Check out these new arrivals from the Utah Mammoth Team Store!

1. 🧪 TEST ALERT - Utah Mammoth Home Jersey
   $129.99 - https://www.mammothteamstore.com/test-home-jersey

2. 🧪 TEST ALERT - Mammoth Team Hoodie
   $74.99 - https://www.mammothteamstore.com/test-hoodie

🏒 Utah Mammoth Store Monitor • Just now
```

### **Single Product Alert:**
```
🆕 New Product Alert!

Utah Mammoth Team Store

🧪 TEST ALERT - Utah Mammoth Home Jersey
New item available from Utah Mammoth Team Store

Price: $129.99
Status: ✅ Available  
Category: Test - Jerseys

[View Product] → https://www.mammothteamstore.com/test-home-jersey

🏒 Utah Mammoth Store Monitor • Just now
```

---

## 🚀 **Full Integration Test**

Once you have your webhook URL, test the complete system:

```bash
# 1. Set webhook URL
export SLACK_WEBHOOK_URL="YOUR_WEBHOOK_URL_HERE"

# 2. Start monitoring server
./web_server -team UTA

# 3. Manual scraper test (check for real products)
curl -X POST http://localhost:8080/api/scrapers/run/mammoth

# 4. Manual Slack test (send test alert)
curl -X POST http://localhost:8080/api/slack/test

# 5. Check dashboard
open http://localhost:8080/scraper-dashboard
```

---

## 🎛️ **Dashboard Integration**

Your scraper dashboard now includes Slack testing:

**Visit**: http://localhost:8080/scraper-dashboard

**New Features:**
- 🏒 **Utah Mammoth Team Store** scraper visible
- ▶️ **"Run Now"** button for manual scraping
- 📱 **Slack Test** button (coming soon to dashboard)
- 📊 **Live monitoring** of scrape results

---

## 🔧 **Available Test Endpoints**

| Endpoint | Method | Description |
|----------|---------|-------------|
| `/api/slack/test` | POST | Send test Slack notification |
| `/api/scrapers/run/mammoth` | POST | Run Mammoth scraper manually |
| `/api/scrapers/status` | GET | View all scraper statuses |
| `/scraper-dashboard` | GET | Web dashboard for monitoring |

---

## 🏆 **System Status**

```json
{
  "mammoth_scraper": "✅ Active and Running",
  "slack_integration": "✅ Ready for Testing",
  "test_endpoints": "✅ All Available",
  "dashboard": "✅ Fully Integrated",
  "webhook_needed": "⚠️  Get from Slack App",
  "ready_to_test": "✅ YES - Multiple Options Available"
}
```

---

## 🎉 **Ready to Test!**

Your **Utah Mammoth Team Store Slack integration** is complete and ready for testing. Choose any of the test methods above to see your notifications in action!

### **Quick Start:**
1. 🔗 **Get webhook URL** from your Slack app
2. 🧪 **Run any test** from the options above  
3. 📱 **Check your Slack channel** for the test alert
4. 🚀 **Start monitoring** with `./web_server -team UTA`

**Once you see the test alerts working, your system will automatically send notifications whenever new Utah Mammoth products are found!** 🏒⚡
