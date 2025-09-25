package models

import (
	"time"
)

// ScrapedItem represents any item that can be scraped from a website
type ScrapedItem interface {
	GetID() string                       // Unique identifier for the item
	GetTitle() string                    // Display title/name
	GetURL() string                      // Item URL
	GetTimestamp() time.Time             // When item was scraped
	GetMetadata() map[string]interface{} // Additional metadata
	Equals(other ScrapedItem) bool       // Compare items for changes
}

// ChangeType represents different types of changes that can occur
type ChangeType int

const (
	ChangeTypeNew ChangeType = iota
	ChangeTypeUpdated
	ChangeTypeRemoved
)

func (ct ChangeType) String() string {
	switch ct {
	case ChangeTypeNew:
		return "new"
	case ChangeTypeUpdated:
		return "updated"
	case ChangeTypeRemoved:
		return "removed"
	default:
		return "unknown"
	}
}

// Change represents a detected change in scraped items
type Change struct {
	Type      ChangeType  `json:"type"`
	Item      ScrapedItem `json:"item"`
	Previous  ScrapedItem `json:"previous,omitempty"` // Only for updates
	Timestamp time.Time   `json:"timestamp"`
}

// ScraperConfig holds configuration for a scraper
type ScraperConfig struct {
	ID              string            `json:"id"`               // Unique scraper ID
	Name            string            `json:"name"`             // Human readable name
	URL             string            `json:"url"`              // Target URL
	Interval        time.Duration     `json:"interval"`         // How often to scrape
	Enabled         bool              `json:"enabled"`          // Whether scraper is active
	UserAgent       string            `json:"user_agent"`       // HTTP User-Agent
	Timeout         time.Duration     `json:"timeout"`          // HTTP timeout
	Headers         map[string]string `json:"headers"`          // Additional HTTP headers
	MaxItems        int               `json:"max_items"`        // Maximum items to return
	ChangeDetection bool              `json:"change_detection"` // Enable change detection
	PersistData     bool              `json:"persist_data"`     // Whether to save data
	CustomSelectors map[string]string `json:"custom_selectors"` // CSS/XPath selectors
	TeamCode        string            `json:"team_code"`        // Associated team (if applicable)
}

// ScraperResult represents the result of a scraping operation
type ScraperResult struct {
	ScraperID   string        `json:"scraper_id"`
	Items       []ScrapedItem `json:"items"`
	Changes     []Change      `json:"changes"`
	Timestamp   time.Time     `json:"timestamp"`
	Duration    time.Duration `json:"duration"`
	ItemCount   int           `json:"item_count"`
	ChangeCount int           `json:"change_count"`
	Error       error         `json:"error,omitempty"`
}

// Scraper interface defines the contract for all scrapers
type Scraper interface {
	GetID() string                                  // Get scraper ID
	GetConfig() ScraperConfig                       // Get configuration
	SetConfig(config ScraperConfig)                 // Update configuration
	Scrape() (*ScraperResult, error)                // Perform scraping
	ProcessChanges(old, new []ScrapedItem) []Change // Detect changes
	Validate() error                                // Validate configuration
}

// ActionTrigger defines actions that can be taken when changes occur
type ActionTrigger interface {
	GetID() string                    // Unique action ID
	GetName() string                  // Human readable name
	ShouldTrigger(change Change) bool // Whether this action should run
	Execute(changes []Change) error   // Execute the action
}

// ScraperManager manages multiple scrapers and their execution
type ScraperManager interface {
	RegisterScraper(scraper Scraper) error               // Add a new scraper
	UnregisterScraper(scraperID string) error            // Remove a scraper
	GetScraper(scraperID string) (Scraper, error)        // Get specific scraper
	GetAllScrapers() []Scraper                           // Get all scrapers
	RunScraper(scraperID string) (*ScraperResult, error) // Run specific scraper
	RunAllScrapers() ([]*ScraperResult, error)           // Run all enabled scrapers
	Start() error                                        // Start automatic scraping
	Stop() error                                         // Stop automatic scraping
}
