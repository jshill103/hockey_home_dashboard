package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleSlackTest sends a test Slack notification for the Mammoth store
func HandleSlackTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get webhook URL from environment or request body
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")

	// Check if webhook URL provided in request body
	if r.Header.Get("Content-Type") == "application/json" {
		var requestBody struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err == nil && requestBody.WebhookURL != "" {
			webhookURL = requestBody.WebhookURL
		}
	}

	if webhookURL == "" {
		response := map[string]interface{}{
			"status":  "error",
			"message": "No Slack webhook URL configured",
			"instructions": map[string]string{
				"config_file":  "Edit config.go and replace REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL_HERE with your webhook URL",
				"environment":  "Set SLACK_WEBHOOK_URL environment variable",
				"request_body": "Send JSON with 'webhook_url' field",
				"get_webhook":  "https://api.slack.com/apps/A09GHT50BFW → Incoming Webhooks",
				"quick_setup":  "Run: go run setup_slack.go",
			},
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

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
				Title:     "🧪 TEST ALERT - Utah Mammoth Home Jersey",
				URL:       "https://www.mammothteamstore.com/test-home-jersey",
				Price:     129.99,
				Currency:  "$",
				ImageURL:  "https://example.com/jersey.jpg",
				Available: true,
				Category:  "Test - Jerseys",
				Brand:     "Utah Mammoth Team Store",
				TeamCode:  "UTA",
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		},
		{
			Type: models.ChangeTypeNew,
			Item: &models.ProductItem{
				Title:     "🧪 TEST ALERT - Mammoth Team Hoodie",
				URL:       "https://www.mammothteamstore.com/test-hoodie",
				Price:     74.99,
				Currency:  "$",
				Available: true,
				Category:  "Test - Apparel",
				Brand:     "Utah Mammoth Team Store",
				TeamCode:  "UTA",
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		},
	}

	// Send test notification
	err := slackAction.Execute(testProducts)
	if err != nil {
		response := map[string]interface{}{
			"status":             "error",
			"message":            fmt.Sprintf("Failed to send Slack notification: %v", err),
			"webhook_url_length": len(webhookURL),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success response
	response := map[string]interface{}{
		"status":        "success",
		"message":       "Test Slack notification sent successfully!",
		"products_sent": len(testProducts),
		"timestamp":     time.Now().Format(time.RFC3339),
		"details": map[string]interface{}{
			"webhook_url": webhookURL[:30] + "...", // Only show first 30 chars for security
			"products": []map[string]interface{}{
				{
					"title": "🧪 TEST ALERT - Utah Mammoth Home Jersey",
					"price": "$129.99",
					"url":   "https://www.mammothteamstore.com/test-home-jersey",
				},
				{
					"title": "🧪 TEST ALERT - Mammoth Team Hoodie",
					"price": "$74.99",
					"url":   "https://www.mammothteamstore.com/test-hoodie",
				},
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}
