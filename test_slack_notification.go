package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

func main() {
	// Get webhook URL from environment or command line
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if len(os.Args) > 1 {
		webhookURL = os.Args[1]
	}

	if webhookURL == "" {
		fmt.Println("❌ No Slack webhook URL provided!")
		fmt.Println("")
		fmt.Println("🔗 Get your webhook URL:")
		fmt.Println("1. Go to: https://api.slack.com/apps/A09GHT50BFW")
		fmt.Println("2. Navigate to: Incoming Webhooks")
		fmt.Println("3. Activate Incoming Webhooks (if not already enabled)")
		fmt.Println("4. Click 'Add New Webhook to Workspace'")
		fmt.Println("5. Select your channel and copy the webhook URL")
		fmt.Println("")
		fmt.Println("📝 Usage:")
			fmt.Println("   export SLACK_WEBHOOK_URL=\"https://your-slack-webhook-url-here.example.com/replace-with-real-webhook\"")
		fmt.Println("   go run test_slack_notification.go")
		fmt.Println("")
		fmt.Println("   OR")
		fmt.Println("")
			fmt.Println("   go run test_slack_notification.go \"https://your-slack-webhook-url-here.example.com/replace-with-real-webhook\"")
		os.Exit(1)
	}

	fmt.Println("🧪 Testing Slack notification for Utah Mammoth Team Store...")
	fmt.Printf("📡 Webhook URL: %s...%s\n", webhookURL[:30], webhookURL[len(webhookURL)-10:])

	// Create Slack configuration with your credentials
	slackConfig := services.SlackConfig{
		AppID:             "A09GHT50BFW",
		ClientID:          "1023450607298.9561923011540",
		ClientSecret:      "e3b035bbbe7c3276849f7dc71f981dbb",
		SigningSecret:     "d52faf6d507fad36b8c2bb3b9d327908",
		VerificationToken: "M60EwLFxKj1EsRqJ7tNFM89j",
		WebhookURL:        webhookURL,
	}

	// Create Slack action
	slackAction := services.NewSlackAction(slackConfig, webhookURL)

	// Create test product items
	testProducts := []models.Change{
		{
			Type: models.ChangeTypeNew,
			Item: &models.ProductItem{
				Title:     "Utah Mammoth Home Jersey - Player Edition",
				URL:       "https://www.mammothteamstore.com/en/utah-mammoth-men/jerseys/player-home-jersey",
				Price:     129.99,
				Currency:  "$",
				ImageURL:  "https://example.com/jersey-image.jpg",
				Available: true,
				Category:  "Jerseys",
				Brand:     "Utah Mammoth Team Store",
				TeamCode:  "UTA",
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		},
		{
			Type: models.ChangeTypeNew,
			Item: &models.ProductItem{
				Title:     "Mammoth Team Logo Hoodie - Official",
				URL:       "https://www.mammothteamstore.com/en/utah-mammoth-men/apparel/team-hoodie",
				Price:     74.99,
				Currency:  "$",
				ImageURL:  "https://example.com/hoodie-image.jpg",
				Available: true,
				Category:  "Apparel",
				Brand:     "Utah Mammoth Team Store",
				TeamCode:  "UTA",
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		},
		{
			Type: models.ChangeTypeNew,
			Item: &models.ProductItem{
				Title:     "Utah Mammoth Snapback Hat - Adjustable",
				URL:       "https://www.mammothteamstore.com/en/utah-mammoth-men/accessories/snapback-hat",
				Price:     29.99,
				Currency:  "$",
				ImageURL:  "https://example.com/hat-image.jpg",
				Available: true,
				Category:  "Accessories",
				Brand:     "Utah Mammoth Team Store",
				TeamCode:  "UTA",
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		},
	}

	fmt.Println("")
	fmt.Println("📦 Test Products:")
	for i, change := range testProducts {
		if productItem, ok := change.Item.(*models.ProductItem); ok {
			fmt.Printf("   %d. %s - %s%.2f\n", i+1, productItem.Title, productItem.Currency, productItem.Price)
		}
	}

	fmt.Println("")
	fmt.Println("📱 Sending test notification to Slack...")

	// Execute the Slack action with test data
	err := slackAction.Execute(testProducts)
	if err != nil {
		fmt.Printf("❌ Error sending Slack notification: %v\n", err)
		fmt.Println("")
		fmt.Println("🔧 Troubleshooting:")
		fmt.Println("   • Verify your webhook URL is correct")
		fmt.Println("   • Check that your Slack app has permission to post messages")
		fmt.Println("   • Ensure the webhook is active in your Slack workspace")
		os.Exit(1)
	}

	fmt.Println("✅ Test notification sent successfully!")
	fmt.Println("")
	fmt.Println("📱 Check your Slack channel for the test message!")
	fmt.Println("")
	fmt.Println("🎯 Message should show:")
	fmt.Println("   • 🆕 New Products Alert!")
	fmt.Println("   • Found 3 new items")
	fmt.Println("   • Product details with prices and links")
	fmt.Println("   • Sent from 'Mammoth Store Monitor'")
	fmt.Println("")
	fmt.Println("🚀 Your Mammoth store monitoring system is ready!")

	// Test single product notification too
	fmt.Println("")
	fmt.Println("📱 Sending single product test notification...")

	singleProductChange := []models.Change{
		{
			Type: models.ChangeTypeNew,
			Item: &models.ProductItem{
				Title:     "🧪 TEST ALERT - Utah Mammoth Practice Jersey",
				URL:       "https://www.mammothteamstore.com/test-product",
				Price:     89.99,
				Currency:  "$",
				Available: true,
				Category:  "Test Category",
				Brand:     "Utah Mammoth Team Store",
				TeamCode:  "UTA",
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		},
	}

	err = slackAction.Execute(singleProductChange)
	if err != nil {
		fmt.Printf("❌ Error sending single product notification: %v\n", err)
	} else {
		fmt.Println("✅ Single product test notification sent!")
	}

	fmt.Println("")
	fmt.Println("🏒 Utah Mammoth Team Store Slack integration is now fully operational! 🏒")
}
