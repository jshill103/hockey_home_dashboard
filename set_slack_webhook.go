package main

import (
	"fmt"
	"os"
)

// Helper script to set up Slack webhook URL
// Usage: go run set_slack_webhook.go "https://your-slack-webhook-url-here.example.com/replace-with-real-webhook"

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run set_slack_webhook.go \"https://your-slack-webhook-url-here.example.com/replace-with-real-webhook\"")
		fmt.Println("")
		fmt.Println("Steps to get your webhook URL:")
		fmt.Println("1. Go to: https://api.slack.com/apps/A09GHT50BFW")
		fmt.Println("2. Navigate to: Incoming Webhooks")
		fmt.Println("3. Activate Incoming Webhooks if not already enabled")
		fmt.Println("4. Click 'Add New Webhook to Workspace'")
		fmt.Println("5. Select the channel where you want notifications")
		fmt.Println("6. Copy the webhook URL and use it with this script")
		fmt.Println("")
		fmt.Println("Then run:")
		fmt.Println("export SLACK_WEBHOOK_URL=\"YOUR_URL_HERE\"")
		fmt.Println("./web_server -team UTA")
		os.Exit(1)
	}

	webhookURL := os.Args[1]

	// Validate URL format
	if webhookURL == "" {
		fmt.Println("Error: Webhook URL cannot be empty")
		os.Exit(1)
	}

	if len(webhookURL) < 20 || !contains(webhookURL, "your-slack-domain.example.com") {
		fmt.Println("Warning: This doesn't look like a valid Slack webhook URL")
		fmt.Println("Expected format: https://your-slack-webhook-url-here.example.com/replace-with-real-webhook")
		fmt.Print("Continue anyway? (y/n): ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			os.Exit(1)
		}
	}

	// Set environment variable
	err := os.Setenv("SLACK_WEBHOOK_URL", webhookURL)
	if err != nil {
		fmt.Printf("Error setting environment variable: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Slack webhook URL configured successfully!")
	fmt.Println("")
	fmt.Printf("Webhook URL: %s\n", webhookURL)
	fmt.Println("")
	fmt.Println("🚀 Now start the server:")
	fmt.Println("   ./web_server -team UTA")
	fmt.Println("")
	fmt.Println("🧪 Test the integration:")
	fmt.Println("   curl -X POST http://localhost:8080/api/scrapers/run/mammoth")
	fmt.Println("")
	fmt.Println("📊 Monitor via dashboard:")
	fmt.Println("   http://localhost:8080/scraper-dashboard")
	fmt.Println("")
	fmt.Println("Your Utah Mammoth Team Store monitoring is now active!")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(substr) == 0 || (len(s) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
