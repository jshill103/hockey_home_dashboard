package main

import (
	"fmt"
	"os"
)

// ========================================
// 🏒 UTAH MAMMOTH SLACK CONFIGURATION 🏒
// ========================================
//
// EASY SETUP INSTRUCTIONS:
//
// 1. Get your Slack webhook URL:
//    - Go to: https://api.slack.com/apps/A09GHT50BFW
//    - Click "Incoming Webhooks"
//    - Click "Add New Webhook to Workspace"
//    - Select your channel
//    - Copy the webhook URL
//
// 2. REPLACE the webhook URL below with your real one:
//
var SlackConfig = struct {
	WebhookURL string
	Enabled    bool
}{
	// 🚨 REPLACE THIS WITH YOUR REAL SLACK WEBHOOK URL:
	WebhookURL: "REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE",
	
	// Set to true to enable Slack notifications:
	Enabled: true,
}

// ========================================
// 📱 SLACK APP CREDENTIALS (Already configured for you!)
// ========================================
var SlackAppConfig = struct {
	AppID             string
	ClientID          string
	ClientSecret      string
	SigningSecret     string
	VerificationToken string
}{
	AppID:             "A09GHT50BFW",
	ClientID:          "1023450607298.9561923011540",
	ClientSecret:      "e3b035bbbe7c3276849f7dc71f981dbb",
	SigningSecret:     "d52faf6d507fad36b8c2bb3b9d327908",
	VerificationToken: "M60EwLFxKj1EsRqJ7tNFM89j",
}

// ========================================
// 🔧 INTERNAL FUNCTIONS (Don't modify these)
// ========================================

// GetSlackWebhookURL returns the configured webhook URL, checking environment variable first
func GetSlackWebhookURL() string {
	// Check environment variable first
	if envURL := os.Getenv("SLACK_WEBHOOK_URL"); envURL != "" {
		return envURL
	}
	
	// Return configured URL
	return SlackConfig.WebhookURL
}

// IsSlackEnabled returns whether Slack notifications are enabled
func IsSlackEnabled() bool {
	return SlackConfig.Enabled && GetSlackWebhookURL() != "REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE"
}

// ValidateSlackConfig checks if Slack is properly configured
func ValidateSlackConfig() error {
	if !SlackConfig.Enabled {
		return fmt.Errorf("Slack notifications are disabled in config.go")
	}
	
	webhookURL := GetSlackWebhookURL()
	if webhookURL == "" || webhookURL == "REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE" {
		return fmt.Errorf(`
❌ Slack webhook URL not configured!

TO SET UP SLACK NOTIFICATIONS:
1. Edit config.go in your project root
2. Replace "REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE" with your real webhook URL
3. Get your webhook URL from: https://api.slack.com/apps/A09GHT50BFW

OR use environment variable:
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
`)
	}
	
	return nil
}
