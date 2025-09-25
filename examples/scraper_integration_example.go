package examples

import (
	"fmt"
	"log"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// Example of how to integrate the new scraping system into the main application
func IntegrateScrapingSystemExample(teamCode string) {
	fmt.Printf("=== NHL Team Dashboard Scraping System Integration Example ===\n")
	fmt.Printf("Team: %s\n\n", teamCode)

	// Initialize the scraper service
	scraperService := services.NewScraperService(teamCode, "./data/scrapers")

	// Initialize with default scrapers and actions
	if err := scraperService.Initialize(); err != nil {
		log.Fatalf("Failed to initialize scraper service: %v", err)
	}

	fmt.Println("✅ Scraper service initialized with:")
	fmt.Println("   - NHL News Scraper (https://www.nhl.com/news)")
	fmt.Printf("   - Fanatics %s Products Scraper\n", teamCode)
	fmt.Println("   - Change detection and actions configured")
	fmt.Println()

	// Example 1: Run scrapers once to get immediate results
	fmt.Println("🔍 Running all scrapers once...")
	results, err := scraperService.RunAllScrapersOnce()
	if err != nil {
		log.Printf("Error running scrapers: %v", err)
	} else {
		for _, result := range results {
			fmt.Printf("   - %s: %d items, %d changes (took %v)\n",
				result.ScraperID, result.ItemCount, result.ChangeCount, result.Duration)
		}
	}
	fmt.Println()

	// Example 2: Get latest products
	fmt.Println("🛍️  Latest products from Fanatics:")
	products, err := scraperService.GetLatestProducts(5)
	if err != nil {
		log.Printf("Error getting products: %v", err)
	} else {
		for i, item := range products {
			if productItem, ok := item.(*models.ProductItem); ok {
				fmt.Printf("   %d. %s - %s\n", i+1, productItem.Title, productItem.GetPriceString())
			}
		}
	}
	fmt.Println()

	// Example 3: Get latest news
	fmt.Println("📰 Latest NHL news:")
	news, err := scraperService.GetLatestNews(3)
	if err != nil {
		log.Printf("Error getting news: %v", err)
	} else {
		for i, item := range news {
			if newsItem, ok := item.(*models.NewsItem); ok {
				fmt.Printf("   %d. %s\n", i+1, newsItem.Title)
			}
		}
	}
	fmt.Println()

	// Example 4: Get scraper status
	fmt.Println("📊 Scraper status:")
	status := scraperService.GetScraperStatus()
	for scraperID, scraperStatus := range status {
		statusMap := scraperStatus.(map[string]interface{})
		fmt.Printf("   - %s: Enabled=%v, Interval=%s\n",
			statusMap["name"], statusMap["enabled"], statusMap["interval"])
	}
	fmt.Println()

	// Example 5: Start automatic scraping (commented out for example)
	/*
		fmt.Println("🚀 Starting automatic scraping...")
		if err := scraperService.Start(); err != nil {
			log.Printf("Error starting scrapers: %v", err)
		} else {
			fmt.Println("   - Scrapers are now running automatically")
			fmt.Println("   - Check logs in ./data/scrapers/logs/ for change alerts")

			// Let it run for a bit
			time.Sleep(30 * time.Second)

			fmt.Println("🛑 Stopping automatic scraping...")
			scraperService.Stop()
		}
	*/

	fmt.Println("✨ Integration example completed!")
}

// Example of creating a custom scraper
func CreateCustomScraperExample() *services.BaseScraper {
	config := models.ScraperConfig{
		ID:              "example_custom_scraper",
		Name:            "Example Custom Scraper",
		URL:             "https://example.com",
		Interval:        15 * time.Minute,
		Enabled:         true,
		UserAgent:       "Custom-Bot/1.0",
		Timeout:         30 * time.Second,
		MaxItems:        20,
		ChangeDetection: true,
		PersistData:     true,
	}

	return services.NewBaseScraper(config)
}

// Example of creating a custom action
func CreateCustomActionExample() *CustomAction {
	return &CustomAction{
		id:   "custom_action_example",
		name: "Custom Action Example",
	}
}

// CustomAction is an example implementation of ActionTrigger
type CustomAction struct {
	id   string
	name string
}

func (ca *CustomAction) GetID() string {
	return ca.id
}

func (ca *CustomAction) GetName() string {
	return ca.name
}

func (ca *CustomAction) ShouldTrigger(change models.Change) bool {
	// Example: Only trigger for new items
	return change.Type == models.ChangeTypeNew
}

func (ca *CustomAction) Execute(changes []models.Change) error {
	// Example: Send a webhook or email notification
	for _, change := range changes {
		fmt.Printf("🔔 CUSTOM ACTION: New item detected - %s\n", change.Item.GetTitle())

		// Here you could:
		// - Send a webhook to a Discord/Slack channel
		// - Send an email notification
		// - Post to a social media account
		// - Update a database
		// - Trigger other automated processes
	}
	return nil
}

// Example of how to integrate with the existing main.go
func IntegrateWithExistingMainExample(teamCode string) {
	/*
		// This would go in your existing main.go file:

		import "github.com/jaredshillingburg/go_uhc/services"

		// In main() function, after team configuration:
		fmt.Println("Initializing scraping system...")
		scraperService := services.NewScraperService(teamCode, "./data/scrapers")

		if err := scraperService.Initialize(); err != nil {
			fmt.Printf("Warning: Failed to initialize scraper service: %v\n", err)
		} else {
			// Start automatic scraping in background
			go func() {
				if err := scraperService.Start(); err != nil {
					fmt.Printf("Warning: Failed to start scraper service: %v\n", err)
				}
			}()

			fmt.Println("Scraping system initialized and started")
		}

		// Add new HTTP handlers for scraper endpoints:
		http.HandleFunc("/scrapers/status", func(w http.ResponseWriter, r *http.Request) {
			status := scraperService.GetScraperStatus()
			// Return JSON status
		})

		http.HandleFunc("/scrapers/products", func(w http.ResponseWriter, r *http.Request) {
			products, _ := scraperService.GetLatestProducts(10)
			// Return JSON products
		})

		http.HandleFunc("/scrapers/run", func(w http.ResponseWriter, r *http.Request) {
			results, _ := scraperService.RunAllScrapersOnce()
			// Return JSON results
		})
	*/
}
