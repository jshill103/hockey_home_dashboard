package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// SlackConfig holds Slack app configuration
type SlackConfig struct {
	AppID             string
	ClientID          string
	ClientSecret      string
	SigningSecret     string
	VerificationToken string
	BotToken          string // Will be set after OAuth or manually
	WebhookURL        string // Incoming webhook URL for posting messages
}

// SlackAction sends notifications to Slack when new products are detected
type SlackAction struct {
	id      string
	name    string
	enabled bool
	config  SlackConfig
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Text        string            `json:"text"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	ImageURL   string       `json:"image_url,omitempty"`
	ThumbURL   string       `json:"thumb_url,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackAction creates a new Slack action for product notifications
func NewSlackAction(config SlackConfig, webhookURL string) *SlackAction {
	config.WebhookURL = webhookURL

	return &SlackAction{
		id:      "slack_notifications",
		name:    "Slack Product Notifications",
		enabled: true,
		config:  config,
	}
}

// GetID returns the action's ID
func (sa *SlackAction) GetID() string {
	return sa.id
}

// GetName returns the action's name
func (sa *SlackAction) GetName() string {
	return sa.name
}

// IsEnabled returns whether the action is enabled
func (sa *SlackAction) IsEnabled() bool {
	return sa.enabled
}

// ShouldTrigger determines if this action should be triggered for a specific change
func (sa *SlackAction) ShouldTrigger(change models.Change) bool {
	if !sa.enabled {
		return false
	}

	// Trigger on new items and significant updates
	switch change.Type {
	case models.ChangeTypeNew:
		return true // Always notify on new products
	case models.ChangeTypeUpdated:
		// Only trigger for significant updates (could be enhanced)
		return true
	case models.ChangeTypeRemoved:
		return false // Don't spam about removed items
	default:
		return false
	}
}

// Execute processes detected changes and sends Slack notifications
func (sa *SlackAction) Execute(changes []models.Change) error {
	if !sa.enabled {
		return nil
	}

	if len(changes) == 0 {
		return nil
	}

	// Group changes by type
	newItems := []models.Change{}
	updatedItems := []models.Change{}
	removedItems := []models.Change{}

	for _, change := range changes {
		switch change.Type {
		case models.ChangeTypeNew:
			newItems = append(newItems, change)
		case models.ChangeTypeUpdated:
			updatedItems = append(updatedItems, change)
		case models.ChangeTypeRemoved:
			removedItems = append(removedItems, change)
		}
	}

	// Send notifications for new items (most important)
	if len(newItems) > 0 {
		err := sa.sendNewProductsNotification(newItems)
		if err != nil {
			return fmt.Errorf("failed to send new products notification: %w", err)
		}
	}

	// Send notifications for updated items (if significant)
	if len(updatedItems) > 0 {
		err := sa.sendUpdatedProductsNotification(updatedItems)
		if err != nil {
			return fmt.Errorf("failed to send updated products notification: %w", err)
		}
	}

	// Send notifications for removed items (if wanted)
	if len(removedItems) > 0 {
		err := sa.sendRemovedProductsNotification(removedItems)
		if err != nil {
			return fmt.Errorf("failed to send removed products notification: %w", err)
		}
	}

	return nil
}

// sendNewProductsNotification sends a notification about new products
func (sa *SlackAction) sendNewProductsNotification(newItems []models.Change) error {
	if len(newItems) == 0 {
		return nil
	}

	// Create message based on number of items
	var message SlackMessage
	if len(newItems) == 1 {
		item := newItems[0].Item
		message = sa.createSingleProductMessage(item, "🆕 New Product Alert!")
	} else {
		message = sa.createMultipleProductsMessage(newItems, "🆕 New Products Alert!")
	}

	return sa.sendSlackMessage(message)
}

// sendUpdatedProductsNotification sends a notification about updated products
func (sa *SlackAction) sendUpdatedProductsNotification(updatedItems []models.Change) error {
	if len(updatedItems) == 0 {
		return nil
	}

	// Only send updates for significant changes (price changes, availability changes)
	significantUpdates := []models.Change{}
	for _, change := range updatedItems {
		if sa.isSignificantUpdate(change) {
			significantUpdates = append(significantUpdates, change)
		}
	}

	if len(significantUpdates) == 0 {
		return nil // No significant updates
	}

	var message SlackMessage
	if len(significantUpdates) == 1 {
		item := significantUpdates[0].Item
		message = sa.createSingleProductMessage(item, "📝 Product Update!")
	} else {
		message = sa.createMultipleProductsMessage(significantUpdates, "📝 Product Updates!")
	}

	return sa.sendSlackMessage(message)
}

// sendRemovedProductsNotification sends a notification about removed products
func (sa *SlackAction) sendRemovedProductsNotification(removedItems []models.Change) error {
	if len(removedItems) == 0 {
		return nil
	}

	// Usually we don't want to spam about removed items, but could be useful for limited items
	return nil // Skip for now
}

// createSingleProductMessage creates a rich Slack message for a single product
func (sa *SlackAction) createSingleProductMessage(item models.ScrapedItem, alertText string) SlackMessage {
	if productItem, ok := item.(*models.ProductItem); ok {
		attachment := SlackAttachment{
			Color:      "#36a64f", // Green for new items
			Title:      productItem.Title,
			TitleLink:  productItem.URL,
			Text:       fmt.Sprintf("New item available from %s", productItem.Brand),
			Footer:     "Utah Mammoth Store Monitor",
			FooterIcon: "🏒",
			Timestamp:  time.Now().Unix(),
		}

		// Add price if available
		if productItem.Price > 0 {
			attachment.Fields = append(attachment.Fields, SlackField{
				Title: "Price",
				Value: fmt.Sprintf("%s%.2f", productItem.Currency, productItem.Price),
				Short: true,
			})
		}

		// Add availability
		availabilityText := "✅ Available"
		if !productItem.Available {
			availabilityText = "❌ Out of Stock"
			attachment.Color = "#ff0000" // Red for out of stock
		}
		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "Status",
			Value: availabilityText,
			Short: true,
		})

		// Add category if available
		if productItem.Category != "" {
			attachment.Fields = append(attachment.Fields, SlackField{
				Title: "Category",
				Value: productItem.Category,
				Short: true,
			})
		}

		// Add image if available
		if productItem.ImageURL != "" {
			attachment.ThumbURL = productItem.ImageURL
		}

		return SlackMessage{
			Text:        alertText,
			Username:    "Mammoth Store Monitor",
			IconEmoji:   ":shopping_bags:",
			Attachments: []SlackAttachment{attachment},
		}
	}

	// Fallback for non-product items
	return SlackMessage{
		Text:      fmt.Sprintf("%s %s - %s", alertText, item.GetTitle(), item.GetURL()),
		Username:  "Mammoth Store Monitor",
		IconEmoji: ":information_source:",
	}
}

// createMultipleProductsMessage creates a Slack message for multiple products
func (sa *SlackAction) createMultipleProductsMessage(changes []models.Change, alertText string) SlackMessage {
	attachment := SlackAttachment{
		Color:      "#36a64f", // Green
		Title:      fmt.Sprintf("Found %d new items", len(changes)),
		Text:       "Check out these new arrivals from the Utah Mammoth Team Store!",
		Footer:     "Utah Mammoth Store Monitor",
		FooterIcon: "🏒",
		Timestamp:  time.Now().Unix(),
	}

	// Add a field for each product (limit to first 10 to avoid message size limits)
	maxItems := len(changes)
	if maxItems > 10 {
		maxItems = 10
	}

	for i, change := range changes[:maxItems] {
		if productItem, ok := change.Item.(*models.ProductItem); ok {
			fieldValue := productItem.URL
			if productItem.Price > 0 {
				fieldValue = fmt.Sprintf("%s%.2f - %s", productItem.Currency, productItem.Price, productItem.URL)
			}

			attachment.Fields = append(attachment.Fields, SlackField{
				Title: fmt.Sprintf("%d. %s", i+1, productItem.Title),
				Value: fieldValue,
				Short: false,
			})
		}
	}

	if len(changes) > maxItems {
		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "...",
			Value: fmt.Sprintf("And %d more items! Check the store for the complete list.", len(changes)-maxItems),
			Short: false,
		})
	}

	return SlackMessage{
		Text:        alertText,
		Username:    "Mammoth Store Monitor",
		IconEmoji:   ":shopping_bags:",
		Attachments: []SlackAttachment{attachment},
	}
}

// isSignificantUpdate determines if a product update is worth notifying about
func (sa *SlackAction) isSignificantUpdate(change models.Change) bool {
	// For now, consider any update significant
	// Could be enhanced to only notify on price changes, availability changes, etc.
	return true
}

// sendSlackMessage sends a message to Slack using the webhook URL
func (sa *SlackAction) sendSlackMessage(message SlackMessage) error {
	if sa.config.WebhookURL == "" {
		return fmt.Errorf("no Slack webhook URL configured")
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(sa.config.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned status %d", resp.StatusCode)
	}

	return nil
}

// SetWebhookURL allows updating the webhook URL after creation
func (sa *SlackAction) SetWebhookURL(webhookURL string) {
	sa.config.WebhookURL = webhookURL
}

// SetEnabled allows enabling/disabling the action
func (sa *SlackAction) SetEnabled(enabled bool) {
	sa.enabled = enabled
}
