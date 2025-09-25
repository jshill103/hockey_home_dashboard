# 🏒 Utah Mammoth Slack Notifications - Easy Setup Guide

## 🚀 Quick Start (2-Minute Setup!)

### Step 1: Get Your Slack Webhook URL

1. **Go to the Slack App**: https://api.slack.com/apps/A09GHT50BFW
2. **Click**: "Incoming Webhooks" in the left sidebar
3. **Toggle**: "Activate Incoming Webhooks" if it's OFF
4. **Click**: "Add New Webhook to Workspace"
5. **Select**: Choose the channel where you want notifications (e.g., #general, #hockey, etc.)
6. **Copy**: The webhook URL that appears (starts with `https://hooks.slack...`)

### Step 2: Add URL to Your Dashboard

**Option A: Edit config.go (Permanent Setup)**
1. Open `config.go` in your project root
2. Find this line:
   ```go
   WebhookURL: "REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE",
   ```
3. Replace it with your real webhook URL:
   ```go
   WebhookURL: "https://example.com/your-slack-webhook-url-goes-here",
   ```
4. Save the file

**Option B: Environment Variable (Temporary)**
```bash
export SLACK_WEBHOOK_URL="https://example.com/your-webhook-url"
./web_server -team UTA
```

### Step 3: Start Your Server

```bash
go build -o web_server main.go
./web_server -team UTA
```

You should see: `✅ Slack notifications enabled for new Utah Mammoth products!`

## 🧪 Test Your Setup

### Quick Test via API
```bash
curl -X POST http://localhost:8080/api/slack/test
```

### Manual Test via Script
```bash
go run test_slack_notification.go
```

## 🎯 What You'll Get

When new Utah Mammoth products are found, you'll receive Slack notifications like:

```
🆕 New Products Alert!

Utah Mammoth Team Store

Found 2 new items
Check out these new arrivals from the Utah Mammoth Team Store!

1. Utah Mammoth Home Jersey - Player Edition
   $129.99 - https://www.mammothteamstore.com/...

2. Mammoth Team Logo Hoodie - Official  
   $74.99 - https://www.mammothteamstore.com/...

🏒 Utah Mammoth Store Monitor • Just now
```

## ⚙️ Configuration Options

Edit `config.go` to customize:

```go
var SlackConfig = struct {
	WebhookURL string
	Enabled    bool
}{
	WebhookURL: "YOUR_WEBHOOK_URL_HERE",
	Enabled:    true,  // Set to false to disable notifications
}
```

## 🛠️ Troubleshooting

### ❌ "Slack webhook URL not configured"
- Edit `config.go` and replace the placeholder URL with your real webhook URL

### ❌ "No Slack webhook URL provided"
- Make sure you copied the complete webhook URL from Slack
- The URL should start with `https://hooks.slack...`

### ❌ "Failed to send Slack message"
- Check that your webhook URL is correct
- Make sure the Slack app has permission to post to your selected channel

### ⚠️ "Slack notifications disabled - no webhook URL configured"
- This means the system detected UTA team but no webhook URL was found
- Follow Step 2 above to configure your webhook URL

## 🏒 Ready to Go!

Once configured, your system will:
- ✅ Monitor the Utah Mammoth Team Store every 15 minutes
- ✅ Detect new products automatically  
- ✅ Send rich Slack notifications with prices, images, and links
- ✅ Never spam you - only new items trigger notifications

**Your Utah Mammoth product alerts are now fully automated!** 🎉
