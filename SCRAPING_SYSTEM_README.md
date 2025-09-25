# 🕷️ Multi-Behavior Web Scraping System

A flexible, extensible web scraping framework designed for the NHL Team Dashboard that supports multiple scraping behaviors, change detection, and automated actions.

## 🎯 Overview

This system extends the existing NHL Team Dashboard with a modular scraping framework that can:

- **Handle Multiple Content Types**: News, products, statistics, or any web content
- **Detect Changes**: Automatically identify new, updated, or removed items
- **Trigger Actions**: Execute custom behaviors when changes are detected
- **Persist Data**: Store scraping results for historical analysis
- **Scale Automatically**: Add new scrapers and actions without code changes

## 🏗️ Architecture

### Core Components

```
📁 models/
├── scraper.go          # Core interfaces and data structures
└── scraped_items.go    # Specific item implementations (News, Products, Generic)

📁 services/
├── scraper_base.go     # Common scraper functionality
├── scraper_manager.go  # Orchestrates multiple scrapers
├── scraper_service.go  # High-level service interface
├── scraper_actions.go  # Built-in action implementations
├── news_scraper_v2.go  # NHL news scraper (refactored)
└── fanatics_scraper.go # Fanatics merchandise scraper

📁 examples/
└── scraper_integration_example.go  # Integration examples
```

### Key Interfaces

- **`Scraper`**: Defines how to scrape a specific type of content
- **`ScrapedItem`**: Represents any item that can be scraped
- **`ActionTrigger`**: Defines what happens when changes are detected
- **`ScraperManager`**: Coordinates multiple scrapers and their execution

## 🚀 Quick Start

### 1. Basic Integration

```go
import "github.com/jaredshillingburg/go_uhc/services"

// Initialize for your team
scraperService := services.NewScraperService("UTA", "./scraper_data")

// Set up default scrapers (NHL news + Fanatics products)
if err := scraperService.Initialize(); err != nil {
    log.Fatalf("Failed to initialize: %v", err)
}

// Start automatic scraping
scraperService.Start()
defer scraperService.Stop()
```

### 2. Run Scrapers Manually

```go
// Run all scrapers once
results, err := scraperService.RunAllScrapersOnce()

// Run specific scraper
result, err := scraperService.RunNewsScraperOnce()
result, err := scraperService.RunFanaticsScraperOnce()
```

### 3. Access Scraped Data

```go
// Get latest products
products, _ := scraperService.GetLatestProducts(10)

// Get latest news
news, _ := scraperService.GetLatestNews(5)

// Get recent changes
changes, _ := scraperService.GetRecentChanges(20)
```

## 🛠️ Built-in Scrapers

### 1. NHL News Scraper
- **Target**: https://www.nhl.com/news
- **Content**: News headlines, URLs, publication dates
- **Frequency**: Every 10 minutes
- **Change Detection**: New articles, updated headlines

### 2. Fanatics Merchandise Scraper  
- **Target**: Team-specific Fanatics URLs
- **Content**: Products, prices, availability, images
- **Frequency**: Every 30 minutes
- **Change Detection**: New products, price changes, stock updates

## ⚡ Built-in Actions

### 1. Log Action
Logs all changes to console and/or file
```go
logAction := services.NewLogAction("./logs/changes.log")
```

### 2. New Product Action
Alerts when new team merchandise is discovered
```go
productAction := services.NewNewProductAction("UTA", "./logs/new_products.log")
```

### 3. Price Change Action
Monitors product price changes
```go
priceAction := services.NewPriceChangeAction("UTA", "./logs/price_changes.log")
```

### 4. Stock Alert Action
Tracks product availability changes
```go
stockAction := services.NewStockAlertAction("UTA", "./logs/stock_alerts.log")
```

### 5. News Action
Handles new NHL news discoveries
```go
newsAction := services.NewNewsAction("./logs/nhl_news.log")
```

## 🔧 Creating Custom Scrapers

### Step 1: Implement the Scraper Interface

```go
type MyCustomScraper struct {
    *services.BaseScraper
}

func NewMyCustomScraper() *MyCustomScraper {
    config := models.ScraperConfig{
        ID:       "my_custom_scraper",
        Name:     "My Custom Scraper",
        URL:      "https://example.com",
        Interval: 15 * time.Minute,
        Enabled:  true,
        // ... other config
    }
    
    return &MyCustomScraper{
        BaseScraper: services.NewBaseScraper(config),
    }
}

func (mcs *MyCustomScraper) Scrape() (*models.ScraperResult, error) {
    start := time.Now()
    result := &models.ScraperResult{
        ScraperID: mcs.GetID(),
        Timestamp: start,
    }

    // Fetch HTML
    doc, err := mcs.FetchHTML()
    if err != nil {
        result.Error = err
        return result, err
    }

    // Extract your items
    items := mcs.extractMyItems(doc)
    
    result.Items = items
    result.ItemCount = len(items)
    result.Duration = time.Since(start)
    
    return result, nil
}
```

### Step 2: Register Your Scraper

```go
scraperService.AddCustomScraper(NewMyCustomScraper())
```

## 🎬 Creating Custom Actions

### Step 1: Implement ActionTrigger Interface

```go
type MyCustomAction struct {
    id   string
    name string
}

func (mca *MyCustomAction) GetID() string { return mca.id }
func (mca *MyCustomAction) GetName() string { return mca.name }

func (mca *MyCustomAction) ShouldTrigger(change models.Change) bool {
    // Define when this action should run
    return change.Type == models.ChangeTypeNew
}

func (mca *MyCustomAction) Execute(changes []models.Change) error {
    // Execute your custom behavior
    for _, change := range changes {
        // Send webhook, email, database update, etc.
        fmt.Printf("New item: %s\n", change.Item.GetTitle())
    }
    return nil
}
```

### Step 2: Register Your Action

```go
scraperService.AddCustomAction(&MyCustomAction{
    id:   "my_custom_action",
    name: "My Custom Action",
})
```

## 📊 Real-world Example: Fanatics Integration

The Fanatics scraper demonstrates a complete implementation:

```go
// Target URL (automatically built for each team)
url := "https://www.fanatics.com/nhl/utah-hockey-club/..."

// Extracted data includes:
type ProductItem struct {
    Title       string    // "Utah Hockey Club Jersey"
    URL         string    // Direct product link
    Price       float64   // 89.99
    Currency    string    // "USD"
    ImageURL    string    // Product image
    Available   bool      // In stock?
    Category    string    // "NHL Merchandise"
    Brand       string    // "Fanatics"
    TeamCode    string    // "UTA"
    Timestamp   time.Time // When scraped
}
```

### Change Detection Examples

When the scraper runs, it automatically detects:

1. **New Product**: `🏒 NEW UTA PRODUCT ALERT! Product: Utah Hockey Club Winter Classic Jersey - Price: $199.99 USD`

2. **Price Change**: `📉 PRICE CHANGE ALERT - UTA Jersey was $199.99, now $149.99 (saved $50.00)`

3. **Stock Alert**: `✅ STOCK ALERT - UTA Hockey Puck BACK IN STOCK - $24.99`

## 🗂️ Data Persistence

All scraped data is automatically persisted:

```
./scraper_data/
├── logs/                    # Action logs
│   ├── scraper_changes.log  # All changes
│   ├── UTA_new_products.log # New products for team
│   ├── UTA_price_changes.log# Price changes
│   └── nhl_news.log         # NHL news
├── nhl_news_items.json      # Latest news items
├── nhl_news_results.json    # News scraper results
├── fanatics_uta_items.json  # Latest products  
└── fanatics_uta_results.json# Product scraper results
```

## ⚙️ Configuration

### Scraper Configuration Options

```go
models.ScraperConfig{
    ID:                "unique_id",
    Name:              "Human Readable Name", 
    URL:               "https://target-site.com",
    Interval:          30 * time.Minute,    // How often to scrape
    Enabled:           true,                // Active?
    UserAgent:         "Custom-Bot/1.0",    // HTTP User-Agent
    Timeout:           30 * time.Second,    // Request timeout
    Headers:           map[string]string{}, // Additional HTTP headers
    MaxItems:          100,                 // Item limit per scrape
    ChangeDetection:   true,                // Enable change detection
    PersistData:       true,                // Save results to disk
    CustomSelectors:   map[string]string{}, // CSS selectors
    TeamCode:          "UTA",               // Associated team
}
```

### Runtime Management

```go
// Check scraper status
status := scraperService.GetScraperStatus()

// Enable/disable scrapers
scraperService.EnableScraper("nhl_news")
scraperService.DisableScraper("fanatics_uta")

// Update configuration
newConfig := scraper.GetConfig()
newConfig.Interval = 5 * time.Minute
scraperService.UpdateScraperConfig("nhl_news", newConfig)
```

## 🔌 Integration with Existing App

Add to your existing `main.go`:

```go
// Initialize scraping system
scraperService := services.NewScraperService(teamCode, "./data/scrapers")
scraperService.Initialize()

// Start in background
go scraperService.Start()

// Add HTTP endpoints
http.HandleFunc("/api/scrapers/status", func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(scraperService.GetScraperStatus())
})

http.HandleFunc("/api/products/latest", func(w http.ResponseWriter, r *http.Request) {
    products, _ := scraperService.GetLatestProducts(20)
    json.NewEncoder(w).Encode(products)
})

http.HandleFunc("/api/news/latest", func(w http.ResponseWriter, r *http.Request) {
    news, _ := scraperService.GetLatestNews(10) 
    json.NewEncoder(w).Encode(news)
})
```

## 🎯 Use Cases

### E-commerce Monitoring
- Track product availability and prices
- Alert on new merchandise releases
- Monitor competitor pricing

### Content Aggregation  
- Collect news from multiple sources
- Track blog posts or social media
- Monitor schedule/roster changes

### Performance Monitoring
- Scrape statistics from various sites
- Track player performance metrics
- Monitor team rankings

### Custom Alerts
- Discord/Slack notifications
- Email alerts for specific changes
- Database updates for analytics

## 🚀 Performance & Scalability

- **Concurrent Execution**: Scrapers run in parallel
- **Efficient Change Detection**: Only processes actual changes
- **Configurable Intervals**: Balance freshness vs. load
- **Error Handling**: Graceful degradation on failures
- **Resource Management**: Timeouts and limits prevent runaway processes

## 🔧 Troubleshooting

### Common Issues

1. **Scraper Not Finding Items**:
   - Check CSS selectors in configuration
   - Verify target site HTML structure hasn't changed
   - Enable debug logging to see what's being found

2. **Too Many/Few Changes Detected**:
   - Adjust comparison logic in `Equals()` method
   - Check data normalization (whitespace, encoding)
   - Review change detection sensitivity

3. **Performance Issues**:
   - Increase scraping intervals
   - Reduce `MaxItems` limits  
   - Optimize CSS selectors
   - Add more specific filtering

4. **Actions Not Triggering**:
   - Verify `ShouldTrigger()` logic
   - Check action registration
   - Enable debug logging for actions

## 🎉 Conclusion

This flexible scraping system transforms your NHL Team Dashboard into a comprehensive monitoring platform that can track any web content, detect changes, and automate responses. The modular design makes it easy to add new scrapers and actions without touching existing code.

**Next Steps:**
- Add scrapers for additional content sources
- Create custom actions for your specific needs  
- Integrate with external services (webhooks, APIs)
- Build dashboards to visualize scraping data

Happy scraping! 🕷️🏒
