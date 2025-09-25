package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

func main() {
	fmt.Println("🧪 Utah Mammoth Store Slack Integration - Mock Test")
	fmt.Println("============================================================")

	// Create test product items (simulating new products found)
	testProducts := []models.Change{
		{
			Type: models.ChangeTypeNew,
			Item: &models.ProductItem{
				Title:     "Utah Mammoth Home Jersey - Player Edition",
				URL:       "https://www.mammothteamstore.com/utah-mammoth-home-jersey",
				Price:     129.99,
				Currency:  "$",
				ImageURL:  "https://mammothteamstore.com/images/home-jersey.jpg",
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
				URL:       "https://www.mammothteamstore.com/utah-mammoth-hoodie",
				Price:     74.99,
				Currency:  "$",
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
				URL:       "https://www.mammothteamstore.com/utah-mammoth-hat",
				Price:     29.99,
				Currency:  "$",
				Available: true,
				Category:  "Accessories",
				Brand:     "Utah Mammoth Team Store",
				TeamCode:  "UTA",
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		},
	}

	fmt.Println("📦 Simulated New Products Detected:")
	for i, change := range testProducts {
		if productItem, ok := change.Item.(*models.ProductItem); ok {
			fmt.Printf("   %d. %s\n", i+1, productItem.Title)
			fmt.Printf("      💰 Price: %s%.2f\n", productItem.Currency, productItem.Price)
			fmt.Printf("      🔗 URL: %s\n", productItem.URL)
			fmt.Printf("      📂 Category: %s\n", productItem.Category)
			fmt.Println()
		}
	}

	// Create mock Slack action (without webhook URL)
	slackConfig := services.SlackConfig{
		AppID:             "A09GHT50BFW",
		ClientID:          "1023450607298.9561923011540",
		ClientSecret:      "e3b035bbbe7c3276849f7dc71f981dbb",
		SigningSecret:     "d52faf6d507fad36b8c2bb3b9d327908",
		VerificationToken: "M60EwLFxKj1EsRqJ7tNFM89j",
	}

	slackAction := services.NewSlackAction(slackConfig, "")

	fmt.Println("📱 Generating Slack Message Preview...")
	fmt.Println("============================================================")

	// Create the message that would be sent
	message := createSlackMessagePreview(testProducts)

	// Display the message preview
	jsonBytes, _ := json.MarshalIndent(message, "", "  ")
	fmt.Println("🔍 Raw Slack Message JSON:")
	fmt.Println(string(jsonBytes))

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("📱 How this will appear in Slack:")
	fmt.Println("============================================================")

	// Show user-friendly preview
	fmt.Printf("👤 From: %s %s\n", message.Username, message.IconEmoji)
	fmt.Printf("📢 Main Text: %s\n", message.Text)
	fmt.Println()

	if len(message.Attachments) > 0 {
		attachment := message.Attachments[0]
		fmt.Printf("🎨 Color: %s\n", attachment.Color)
		fmt.Printf("📋 Title: %s\n", attachment.Title)
		fmt.Printf("📝 Description: %s\n", attachment.Text)
		fmt.Println()

		fmt.Println("📊 Product Details:")
		for i, field := range attachment.Fields {
			fmt.Printf("   %d. %s: %s\n", i+1, field.Title, field.Value)
		}
		fmt.Println()

		fmt.Printf("👣 Footer: %s %s\n", attachment.Footer, attachment.FooterIcon)
		fmt.Printf("⏰ Timestamp: %s\n", time.Unix(attachment.Timestamp, 0).Format("2006-01-02 15:04:05"))
	}

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("🚀 Next Steps to Activate:")
	fmt.Println("============================================================")
	fmt.Println("1. 🔗 Get your Slack webhook URL:")
	fmt.Println("   https://api.slack.com/apps/A09GHT50BFW")
	fmt.Println()
	fmt.Println("2. 🧪 Test with real webhook:")
	fmt.Println("   go run test_slack_notification.go \"YOUR_WEBHOOK_URL\"")
	fmt.Println()
	fmt.Println("3. 🚀 Start monitoring:")
	fmt.Println("   export SLACK_WEBHOOK_URL=\"YOUR_WEBHOOK_URL\"")
	fmt.Println("   ./web_server -team UTA")
	fmt.Println()
	fmt.Println("🏒 Your Utah Mammoth Team Store alerts will look exactly like this!")

	// Test single product message
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("📱 Single Product Alert Preview:")
	fmt.Println("============================================================")

	singleProductChange := []models.Change{testProducts[0]}
	singleMessage := createSlackMessagePreview(singleProductChange)

	fmt.Printf("👤 From: %s %s\n", singleMessage.Username, singleMessage.IconEmoji)
	fmt.Printf("📢 Main Text: %s\n", singleMessage.Text)
	if len(singleMessage.Attachments) > 0 {
		att := singleMessage.Attachments[0]
		fmt.Printf("🏒 Product: %s\n", att.Title)
		fmt.Printf("🔗 Link: %s\n", att.TitleLink)
		fmt.Printf("📝 Description: %s\n", att.Text)
		for _, field := range att.Fields {
			fmt.Printf("   • %s: %s\n", field.Title, field.Value)
		}
	}
}

func createSlackMessagePreview(changes []models.Change) services.SlackMessage {
	if len(changes) == 1 {
		// Single product message
		item := changes[0].Item
		if productItem, ok := item.(*models.ProductItem); ok {
			attachment := services.SlackAttachment{
				Color:      "#36a64f",
				Title:      productItem.Title,
				TitleLink:  productItem.URL,
				Text:       fmt.Sprintf("New item available from %s", productItem.Brand),
				Footer:     "Utah Mammoth Store Monitor",
				FooterIcon: "🏒",
				Timestamp:  time.Now().Unix(),
			}

			if productItem.Price > 0 {
				attachment.Fields = append(attachment.Fields, services.SlackField{
					Title: "Price",
					Value: fmt.Sprintf("%s%.2f", productItem.Currency, productItem.Price),
					Short: true,
				})
			}

			attachment.Fields = append(attachment.Fields, services.SlackField{
				Title: "Status",
				Value: "✅ Available",
				Short: true,
			})

			if productItem.Category != "" {
				attachment.Fields = append(attachment.Fields, services.SlackField{
					Title: "Category",
					Value: productItem.Category,
					Short: true,
				})
			}

			return services.SlackMessage{
				Text:        "🆕 New Product Alert!",
				Username:    "Mammoth Store Monitor",
				IconEmoji:   ":shopping_bags:",
				Attachments: []services.SlackAttachment{attachment},
			}
		}
	}

	// Multiple products message
	attachment := services.SlackAttachment{
		Color:      "#36a64f",
		Title:      fmt.Sprintf("Found %d new items", len(changes)),
		Text:       "Check out these new arrivals from the Utah Mammoth Team Store!",
		Footer:     "Utah Mammoth Store Monitor",
		FooterIcon: "🏒",
		Timestamp:  time.Now().Unix(),
	}

	for i, change := range changes {
		if productItem, ok := change.Item.(*models.ProductItem); ok {
			fieldValue := productItem.URL
			if productItem.Price > 0 {
				fieldValue = fmt.Sprintf("%s%.2f - %s", productItem.Currency, productItem.Price, productItem.URL)
			}

			attachment.Fields = append(attachment.Fields, services.SlackField{
				Title: fmt.Sprintf("%d. %s", i+1, productItem.Title),
				Value: fieldValue,
				Short: false,
			})
		}
	}

	return services.SlackMessage{
		Text:        "🆕 New Products Alert!",
		Username:    "Mammoth Store Monitor",
		IconEmoji:   ":shopping_bags:",
		Attachments: []services.SlackAttachment{attachment},
	}
}
