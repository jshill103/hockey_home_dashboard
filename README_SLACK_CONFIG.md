# 🏒 Centralized Slack Configuration System

## 📍 One Place for All Slack Settings

All Slack configuration is now centralized in **`config.go`** in your project root. This makes it super easy to set up and manage your Utah Mammoth Team Store notifications.

## ⚡ Quick Setup Options

### Option 1: Automated Setup (Easiest!)
```bash
go run setup_slack.go
```
This interactive script will:
- Guide you through getting your webhook URL
- Automatically update `config.go` 
- Validate your setup

### Option 2: Manual Setup
1. **Edit `config.go`**:
   ```go
   var SlackConfig = struct {
       WebhookURL string
       Enabled    bool
   }{
       // Replace this line:
       WebhookURL: "REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE",
       
       // With your real webhook URL:
       WebhookURL: "https://example.com/your-slack-webhook-url-goes-here",
       
       Enabled: true,
   }
   ```

2. **Get your webhook URL** from: https://api.slack.com/apps/A09GHT50BFW

### Option 3: Environment Variable (Temporary)
```bash
export SLACK_WEBHOOK_URL="https://example.com/your-webhook-url"
./web_server -team UTA
```

## 🔧 How It Works

The system checks for your Slack webhook URL in this priority order:

1. **Environment Variable** (`SLACK_WEBHOOK_URL`) - highest priority
2. **config.go file** (`SlackConfig.WebhookURL`) - permanent configuration  
3. **API request body** - for testing via API calls

## ✅ Configuration Validation

The system automatically validates your setup:

```
✅ Slack webhook configured for Utah Mammoth notifications
✅ Slack notifications enabled for new Utah Mammoth products!
```

If not configured, you'll see:
```
⚠️  Slack setup needed: Slack webhook URL not configured!
⚠️  Slack notifications disabled - no webhook URL configured
```

## 🧪 Testing Your Setup

### 1. API Test Endpoint
```bash
curl -X POST http://localhost:8080/api/slack/test
```

### 2. Direct Test Script  
```bash
go run test_slack_notification.go
```

### 3. Mock Preview (No actual sending)
```bash
go run mock_slack_demo.go
```

## 📁 File Structure

```
go_uhc/
├── config.go                 ← 🎯 MAIN CONFIGURATION FILE
├── setup_slack.go            ← Interactive setup script
├── SLACK_SETUP.md            ← Complete setup guide
├── test_slack_notification.go ← Test with real webhook
├── mock_slack_demo.go        ← Preview messages
└── services/
    ├── slack_action.go       ← Slack integration logic
    └── scraper_service.go    ← Uses config from main
```

## 🎛️ Advanced Configuration

Edit `config.go` to customize:

```go
var SlackConfig = struct {
	WebhookURL string
	Enabled    bool
}{
	WebhookURL: "https://example.com/your-webhook-url",
	Enabled:    true,  // Set to false to disable all Slack notifications
}

var SlackAppConfig = struct {
	AppID             string
	ClientID          string  
	ClientSecret      string
	SigningSecret     string
	VerificationToken string
}{
	// These are already configured for you - no need to change!
	AppID:             "A09GHT50BFW",
	ClientID:          "1023450607298.9561923011540",
	// ... other credentials
}
```

## 🔍 Troubleshooting

### Problem: "Slack webhook URL not configured"
**Solution**: Edit `config.go` and replace the placeholder with your real webhook URL

### Problem: "No Slack webhook URL configured" via API
**Solution**: The API provides helpful instructions:
```json
{
  "instructions": {
    "config_file": "Edit config.go and replace REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE",
    "quick_setup": "Run: go run setup_slack.go",
    "get_webhook": "https://api.slack.com/apps/A09GHT50BFW → Incoming Webhooks"
  }
}
```

### Problem: Messages not appearing in Slack
**Solution**: 
1. Verify your webhook URL is correct
2. Check that the Slack app has permission to post to your channel
3. Test with: `curl -X POST http://localhost:8080/api/slack/test`

## 🚀 Benefits of Centralized Config

✅ **Single source of truth** - all Slack settings in one place  
✅ **Easy to find and edit** - clear documentation in `config.go`  
✅ **Environment override** - can still use env vars for different deployments  
✅ **Validation built-in** - automatically checks if configured correctly  
✅ **Multiple setup methods** - choose what works best for you  
✅ **Type-safe** - Go compiler ensures configuration is valid  

## 🎯 What You Get

Once configured, you'll receive rich Slack notifications for new Utah Mammoth products:

- 🏒 **Product alerts** with images, prices, and direct links
- 🔄 **Smart filtering** - only new products trigger notifications  
- ⏰ **Automatic monitoring** every 15 minutes
- 📱 **Beautiful formatting** with team branding
- 🚫 **No spam** - intelligent change detection

**Your Utah Mammoth Team Store monitoring is now fully automated!** 🎉
