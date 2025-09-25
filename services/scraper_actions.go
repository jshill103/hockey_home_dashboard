package services

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
)

// LogAction logs changes to console and/or file
type LogAction struct {
	id       string
	name     string
	logFile  string
	logLevel string
}

// NewLogAction creates a new log action
func NewLogAction(logFile string) *LogAction {
	return &LogAction{
		id:       "log_action",
		name:     "Log Changes",
		logFile:  logFile,
		logLevel: "info",
	}
}

// GetID returns the action ID
func (la *LogAction) GetID() string {
	return la.id
}

// GetName returns the action name
func (la *LogAction) GetName() string {
	return la.name
}

// ShouldTrigger determines if this action should run for a given change
func (la *LogAction) ShouldTrigger(change models.Change) bool {
	// Log all changes
	return true
}

// Execute logs the changes
func (la *LogAction) Execute(changes []models.Change) error {
	for _, change := range changes {
		message := fmt.Sprintf("[%s] %s: %s - %s",
			change.Timestamp.Format("2006-01-02 15:04:05"),
			strings.ToUpper(change.Type.String()),
			change.Item.GetTitle(),
			change.Item.GetURL())

		// Log to console
		fmt.Println(message)

		// Log to file if specified
		if la.logFile != "" {
			if err := la.logToFile(message); err != nil {
				return fmt.Errorf("failed to log to file: %v", err)
			}
		}
	}

	return nil
}

// logToFile appends the message to the log file
func (la *LogAction) logToFile(message string) error {
	file, err := os.OpenFile(la.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	logger := log.New(file, "", 0)
	logger.Println(message)
	return nil
}

// NewProductAction logs new product discoveries
type NewProductAction struct {
	id       string
	name     string
	teamCode string
	logFile  string
}

// NewNewProductAction creates an action specifically for new product discoveries
func NewNewProductAction(teamCode string, logFile string) *NewProductAction {
	return &NewProductAction{
		id:       fmt.Sprintf("new_product_action_%s", strings.ToLower(teamCode)),
		name:     fmt.Sprintf("New %s Product Alert", teamCode),
		teamCode: teamCode,
		logFile:  logFile,
	}
}

// GetID returns the action ID
func (npa *NewProductAction) GetID() string {
	return npa.id
}

// GetName returns the action name
func (npa *NewProductAction) GetName() string {
	return npa.name
}

// ShouldTrigger determines if this action should run for a given change
func (npa *NewProductAction) ShouldTrigger(change models.Change) bool {
	// Only trigger for new products of the specified team
	if change.Type != models.ChangeTypeNew {
		return false
	}

	// Check if it's a product item
	if productItem, ok := change.Item.(*models.ProductItem); ok {
		return productItem.TeamCode == npa.teamCode
	}

	return false
}

// Execute handles new product discoveries
func (npa *NewProductAction) Execute(changes []models.Change) error {
	for _, change := range changes {
		if productItem, ok := change.Item.(*models.ProductItem); ok {
			message := fmt.Sprintf("🏒 NEW %s PRODUCT ALERT! 🏒\n"+
				"Product: %s\n"+
				"Price: %s\n"+
				"Available: %t\n"+
				"URL: %s\n"+
				"Discovered: %s\n"+
				"---",
				npa.teamCode,
				productItem.Title,
				productItem.GetPriceString(),
				productItem.Available,
				productItem.URL,
				change.Timestamp.Format("2006-01-02 15:04:05"))

			// Log to console with emoji
			fmt.Println(message)

			// Log to file if specified
			if npa.logFile != "" {
				if err := npa.logToFile(message); err != nil {
					return fmt.Errorf("failed to log new product to file: %v", err)
				}
			}
		}
	}

	return nil
}

// logToFile appends the message to the log file
func (npa *NewProductAction) logToFile(message string) error {
	file, err := os.OpenFile(npa.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	logger := log.New(file, "", 0)
	logger.Println(message)
	return nil
}

// PriceChangeAction triggers on product price changes
type PriceChangeAction struct {
	id       string
	name     string
	teamCode string
	logFile  string
}

// NewPriceChangeAction creates an action for product price changes
func NewPriceChangeAction(teamCode string, logFile string) *PriceChangeAction {
	return &PriceChangeAction{
		id:       fmt.Sprintf("price_change_action_%s", strings.ToLower(teamCode)),
		name:     fmt.Sprintf("%s Price Change Alert", teamCode),
		teamCode: teamCode,
		logFile:  logFile,
	}
}

// GetID returns the action ID
func (pca *PriceChangeAction) GetID() string {
	return pca.id
}

// GetName returns the action name
func (pca *PriceChangeAction) GetName() string {
	return pca.name
}

// ShouldTrigger determines if this action should run for a given change
func (pca *PriceChangeAction) ShouldTrigger(change models.Change) bool {
	// Only trigger for updated products with price changes
	if change.Type != models.ChangeTypeUpdated || change.Previous == nil {
		return false
	}

	// Check if both items are product items and prices are different
	currentProduct, okCurrent := change.Item.(*models.ProductItem)
	previousProduct, okPrevious := change.Previous.(*models.ProductItem)

	if okCurrent && okPrevious {
		return currentProduct.TeamCode == pca.teamCode &&
			currentProduct.Price != previousProduct.Price
	}

	return false
}

// Execute handles price change alerts
func (pca *PriceChangeAction) Execute(changes []models.Change) error {
	for _, change := range changes {
		currentProduct := change.Item.(*models.ProductItem)
		previousProduct := change.Previous.(*models.ProductItem)

		priceDirection := "📈"
		if currentProduct.Price < previousProduct.Price {
			priceDirection = "📉"
		}

		message := fmt.Sprintf("%s PRICE CHANGE ALERT - %s %s\n"+
			"Product: %s\n"+
			"Previous Price: %s\n"+
			"New Price: %s\n"+
			"Change: $%.2f\n"+
			"URL: %s\n"+
			"Updated: %s\n"+
			"---",
			priceDirection,
			pca.teamCode,
			priceDirection,
			currentProduct.Title,
			previousProduct.GetPriceString(),
			currentProduct.GetPriceString(),
			currentProduct.Price-previousProduct.Price,
			currentProduct.URL,
			change.Timestamp.Format("2006-01-02 15:04:05"))

		// Log to console
		fmt.Println(message)

		// Log to file if specified
		if pca.logFile != "" {
			if err := pca.logToFile(message); err != nil {
				return fmt.Errorf("failed to log price change to file: %v", err)
			}
		}
	}

	return nil
}

// logToFile appends the message to the log file
func (pca *PriceChangeAction) logToFile(message string) error {
	file, err := os.OpenFile(pca.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	logger := log.New(file, "", 0)
	logger.Println(message)
	return nil
}

// NewsAction handles new NHL news articles
type NewsAction struct {
	id      string
	name    string
	logFile string
}

// NewNewsAction creates an action for new news articles
func NewNewsAction(logFile string) *NewsAction {
	return &NewsAction{
		id:      "news_action",
		name:    "NHL News Alert",
		logFile: logFile,
	}
}

// GetID returns the action ID
func (na *NewsAction) GetID() string {
	return na.id
}

// GetName returns the action name
func (na *NewsAction) GetName() string {
	return na.name
}

// ShouldTrigger determines if this action should run for a given change
func (na *NewsAction) ShouldTrigger(change models.Change) bool {
	// Only trigger for new news items
	if change.Type != models.ChangeTypeNew {
		return false
	}

	// Check if it's a news item
	_, ok := change.Item.(*models.NewsItem)
	return ok
}

// Execute handles new news alerts
func (na *NewsAction) Execute(changes []models.Change) error {
	for _, change := range changes {
		if newsItem, ok := change.Item.(*models.NewsItem); ok {
			message := fmt.Sprintf("📰 NEW NHL NEWS 📰\n"+
				"Headline: %s\n"+
				"Date: %s\n"+
				"URL: %s\n"+
				"Discovered: %s\n"+
				"---",
				newsItem.Title,
				newsItem.Date,
				newsItem.URL,
				change.Timestamp.Format("2006-01-02 15:04:05"))

			// Log to console
			fmt.Println(message)

			// Log to file if specified
			if na.logFile != "" {
				if err := na.logToFile(message); err != nil {
					return fmt.Errorf("failed to log news to file: %v", err)
				}
			}
		}
	}

	return nil
}

// logToFile appends the message to the log file
func (na *NewsAction) logToFile(message string) error {
	file, err := os.OpenFile(na.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	logger := log.New(file, "", 0)
	logger.Println(message)
	return nil
}

// StockAlertAction triggers when products go out of stock
type StockAlertAction struct {
	id       string
	name     string
	teamCode string
	logFile  string
}

// NewStockAlertAction creates an action for stock availability changes
func NewStockAlertAction(teamCode string, logFile string) *StockAlertAction {
	return &StockAlertAction{
		id:       fmt.Sprintf("stock_alert_action_%s", strings.ToLower(teamCode)),
		name:     fmt.Sprintf("%s Stock Alert", teamCode),
		teamCode: teamCode,
		logFile:  logFile,
	}
}

// GetID returns the action ID
func (saa *StockAlertAction) GetID() string {
	return saa.id
}

// GetName returns the action name
func (saa *StockAlertAction) GetName() string {
	return saa.name
}

// ShouldTrigger determines if this action should run for a given change
func (saa *StockAlertAction) ShouldTrigger(change models.Change) bool {
	// Only trigger for updated products with availability changes
	if change.Type != models.ChangeTypeUpdated || change.Previous == nil {
		return false
	}

	// Check if both items are product items and availability changed
	currentProduct, okCurrent := change.Item.(*models.ProductItem)
	previousProduct, okPrevious := change.Previous.(*models.ProductItem)

	if okCurrent && okPrevious {
		return currentProduct.TeamCode == saa.teamCode &&
			currentProduct.Available != previousProduct.Available
	}

	return false
}

// Execute handles stock availability alerts
func (saa *StockAlertAction) Execute(changes []models.Change) error {
	for _, change := range changes {
		currentProduct := change.Item.(*models.ProductItem)
		_ = change.Previous.(*models.ProductItem) // Previous product for context

		statusEmoji := "❌"
		statusText := "OUT OF STOCK"
		if currentProduct.Available {
			statusEmoji = "✅"
			statusText = "BACK IN STOCK"
		}

		message := fmt.Sprintf("%s STOCK ALERT - %s %s\n"+
			"Product: %s\n"+
			"Status: %s\n"+
			"Price: %s\n"+
			"URL: %s\n"+
			"Updated: %s\n"+
			"---",
			statusEmoji,
			saa.teamCode,
			statusText,
			currentProduct.Title,
			statusText,
			currentProduct.GetPriceString(),
			currentProduct.URL,
			change.Timestamp.Format("2006-01-02 15:04:05"))

		// Log to console
		fmt.Println(message)

		// Log to file if specified
		if saa.logFile != "" {
			if err := saa.logToFile(message); err != nil {
				return fmt.Errorf("failed to log stock alert to file: %v", err)
			}
		}
	}

	return nil
}

// logToFile appends the message to the log file
func (saa *StockAlertAction) logToFile(message string) error {
	file, err := os.OpenFile(saa.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	logger := log.New(file, "", 0)
	logger.Println(message)
	return nil
}
