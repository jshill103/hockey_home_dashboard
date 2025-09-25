package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("🏒 Utah Mammoth Slack Notifications Setup")
	fmt.Println("=========================================")
	fmt.Println()

	// Check if already configured
	if isAlreadyConfigured() {
		fmt.Println("✅ Slack webhook is already configured!")
		fmt.Println()
		fmt.Print("Do you want to update it? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Setup cancelled.")
			return
		}
	}

	fmt.Println("📱 STEP 1: Get your Slack webhook URL")
	fmt.Println("   1. Go to: https://api.slack.com/apps/A09GHT50BFW")
	fmt.Println("   2. Click: 'Incoming Webhooks'")
	fmt.Println("   3. Click: 'Add New Webhook to Workspace'")
	fmt.Println("   4. Select your channel")
	fmt.Println("   5. Copy the webhook URL")
	fmt.Println()

	fmt.Print("🔗 Paste your webhook URL here: ")
	reader := bufio.NewReader(os.Stdin)
	webhookURL, _ := reader.ReadString('\n')
	webhookURL = strings.TrimSpace(webhookURL)

	if webhookURL == "" {
		fmt.Println("❌ No URL provided. Setup cancelled.")
		return
	}

	if !strings.Contains(webhookURL, "hooks.slack.com") {
		fmt.Println("⚠️  Warning: This doesn't look like a Slack webhook URL")
		fmt.Print("Continue anyway? (y/n): ")
		response, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Setup cancelled.")
			return
		}
	}

	// Update config.go
	if err := updateConfigFile(webhookURL); err != nil {
		fmt.Printf("❌ Error updating config.go: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("✅ SUCCESS! Slack webhook configured in config.go")
	fmt.Println()
	fmt.Println("🚀 Next steps:")
	fmt.Println("   1. Build: go build -o web_server main.go")
	fmt.Println("   2. Start: ./web_server -team UTA")
	fmt.Println("   3. Test:  curl -X POST http://localhost:8080/api/slack/test")
	fmt.Println()
	fmt.Println("🏒 You'll now receive notifications for new Utah Mammoth products!")
}

func isAlreadyConfigured() bool {
	content, err := os.ReadFile("config.go")
	if err != nil {
		return false
	}

	return !strings.Contains(string(content), "REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE")
}

func updateConfigFile(webhookURL string) error {
	// Read the current config.go file
	content, err := os.ReadFile("config.go")
	if err != nil {
		return fmt.Errorf("failed to read config.go: %v", err)
	}

	// Replace the placeholder with the real URL
	oldLine := `	WebhookURL: "REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE",`
	newLine := fmt.Sprintf(`	WebhookURL: "%s",`, webhookURL)

	updatedContent := strings.Replace(string(content), oldLine, newLine, 1)

	if updatedContent == string(content) {
		return fmt.Errorf("could not find placeholder in config.go - it may already be configured")
	}

	// Write the updated content back to the file
	err = os.WriteFile("config.go", []byte(updatedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write config.go: %v", err)
	}

	return nil
}
