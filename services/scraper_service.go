package services

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ScraperService provides a high-level interface to the scraping system
type ScraperService struct {
	manager         models.ScraperManager
	teamCode        string
	dataDir         string
	slackWebhookURL string
}

// NewScraperService creates a new scraper service
func NewScraperService(teamCode string, dataDir string) *ScraperService {
	if dataDir == "" {
		dataDir = "./scraper_data"
	}

	manager := NewScraperManager(dataDir)

	return &ScraperService{
		manager:  manager,
		teamCode: teamCode,
		dataDir:  dataDir,
	}
}

// NewScraperServiceWithSlack creates a new scraper service with Slack webhook URL
func NewScraperServiceWithSlack(teamCode string, dataDir string, slackWebhookURL string) *ScraperService {
	if dataDir == "" {
		dataDir = "./scraper_data"
	}

	manager := NewScraperManager(dataDir)

	return &ScraperService{
		manager:         manager,
		teamCode:        teamCode,
		dataDir:         dataDir,
		slackWebhookURL: slackWebhookURL,
	}
}

// Initialize sets up the default scrapers and actions for the application
func (ss *ScraperService) Initialize() error {
	// Create NHL News scraper
	newsScraperV2 := NewNewsScraperV2()
	if err := ss.manager.RegisterScraper(newsScraperV2); err != nil {
		return fmt.Errorf("failed to register news scraper: %v", err)
	}

	// Create Fanatics scraper for the current team
	fanaticsScraper := NewFanaticsScraper(ss.teamCode)
	if err := ss.manager.RegisterScraper(fanaticsScraper); err != nil {
		return fmt.Errorf("failed to register fanatics scraper: %v", err)
	}

	// Create Mammoth team store scraper (specifically for UTA team)
	if strings.ToUpper(ss.teamCode) == "UTA" {
		mammothScraper := NewMammothScraper()
		if err := ss.manager.RegisterScraper(mammothScraper); err != nil {
			return fmt.Errorf("failed to register mammoth scraper: %v", err)
		}
	}

	// Setup actions
	if err := ss.setupActions(); err != nil {
		return fmt.Errorf("failed to setup actions: %v", err)
	}

	return nil
}

// setupActions configures the default actions for change detection
func (ss *ScraperService) setupActions() error {
	logDir := filepath.Join(ss.dataDir, "logs")

	// General log action
	logAction := NewLogAction(filepath.Join(logDir, "scraper_changes.log"))
	if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
		if err := scraperManager.RegisterAction(logAction); err != nil {
			return fmt.Errorf("failed to register log action: %v", err)
		}
	}

	// News action
	newsAction := NewNewsAction(filepath.Join(logDir, "nhl_news.log"))
	if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
		if err := scraperManager.RegisterAction(newsAction); err != nil {
			return fmt.Errorf("failed to register news action: %v", err)
		}
	}

	// Team-specific product actions
	newProductAction := NewNewProductAction(ss.teamCode, filepath.Join(logDir, fmt.Sprintf("%s_new_products.log", ss.teamCode)))
	if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
		if err := scraperManager.RegisterAction(newProductAction); err != nil {
			return fmt.Errorf("failed to register new product action: %v", err)
		}
	}

	priceChangeAction := NewPriceChangeAction(ss.teamCode, filepath.Join(logDir, fmt.Sprintf("%s_price_changes.log", ss.teamCode)))
	if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
		if err := scraperManager.RegisterAction(priceChangeAction); err != nil {
			return fmt.Errorf("failed to register price change action: %v", err)
		}
	}

	stockAlertAction := NewStockAlertAction(ss.teamCode, filepath.Join(logDir, fmt.Sprintf("%s_stock_alerts.log", ss.teamCode)))
	if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
		if err := scraperManager.RegisterAction(stockAlertAction); err != nil {
			return fmt.Errorf("failed to register stock alert action: %v", err)
		}
	}

	// Slack notification action for UTA team (Mammoth store monitoring)
	if strings.ToUpper(ss.teamCode) == "UTA" && ss.slackWebhookURL != "" {
		slackConfig := SlackConfig{
			AppID:             "A09GHT50BFW",
			ClientID:          "1023450607298.9561923011540",
			ClientSecret:      "e3b035bbbe7c3276849f7dc71f981dbb",
			SigningSecret:     "d52faf6d507fad36b8c2bb3b9d327908",
			VerificationToken: "M60EwLFxKj1EsRqJ7tNFM89j",
		}

		// Create Slack action with configured webhook URL
		slackAction := NewSlackAction(slackConfig, ss.slackWebhookURL)
		if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
			if err := scraperManager.RegisterAction(slackAction); err != nil {
				return fmt.Errorf("failed to register slack action: %v", err)
			}
		}
		fmt.Println("✅ Slack notifications enabled for new Utah Mammoth products!")
	} else if strings.ToUpper(ss.teamCode) == "UTA" {
		fmt.Println("⚠️  Slack notifications disabled - no webhook URL configured")
		fmt.Println("   Edit config.go to set up Slack notifications")
	}

	return nil
}

// Start begins automatic scraping
func (ss *ScraperService) Start() error {
	fmt.Printf("Starting scraper service for team: %s\n", ss.teamCode)
	return ss.manager.Start()
}

// Stop stops automatic scraping
func (ss *ScraperService) Stop() error {
	fmt.Printf("Stopping scraper service for team: %s\n", ss.teamCode)
	return ss.manager.Stop()
}

// RunNewsScraperOnce runs the news scraper once and returns results
func (ss *ScraperService) RunNewsScraperOnce() (*models.ScraperResult, error) {
	return ss.manager.RunScraper("nhl_news")
}

// RunFanaticsScraperOnce runs the Fanatics scraper once and returns results
func (ss *ScraperService) RunFanaticsScraperOnce() (*models.ScraperResult, error) {
	return ss.manager.RunScraper(fmt.Sprintf("fanatics_%s", strings.ToLower(ss.teamCode)))
}

// RunMammothScraperOnce runs the Mammoth team store scraper once and returns results
func (ss *ScraperService) RunMammothScraperOnce() (*models.ScraperResult, error) {
	return ss.manager.RunScraper("mammoth_store")
}

// RunAllScrapersOnce runs all scrapers once
func (ss *ScraperService) RunAllScrapersOnce() ([]*models.ScraperResult, error) {
	return ss.manager.RunAllScrapers()
}

// AddCustomScraper allows adding additional scrapers
func (ss *ScraperService) AddCustomScraper(scraper models.Scraper) error {
	return ss.manager.RegisterScraper(scraper)
}

// AddCustomAction allows adding additional actions
func (ss *ScraperService) AddCustomAction(action models.ActionTrigger) error {
	if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
		return scraperManager.RegisterAction(action)
	}
	return fmt.Errorf("manager does not support custom actions")
}

// GetScraperResults returns the results of a specific scraper
func (ss *ScraperService) GetScraperResults(scraperID string) (*models.ScraperResult, error) {
	// This would need to load from the data store
	if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
		return scraperManager.dataStore.LoadLastResults(scraperID)
	}
	return nil, fmt.Errorf("unable to load scraper results")
}

// GetAllScrapers returns all registered scrapers
func (ss *ScraperService) GetAllScrapers() []models.Scraper {
	return ss.manager.GetAllScrapers()
}

// GetScraperStatus returns the status of all scrapers
func (ss *ScraperService) GetScraperStatus() map[string]interface{} {
	status := make(map[string]interface{})

	scrapers := ss.manager.GetAllScrapers()
	for _, scraper := range scrapers {
		config := scraper.GetConfig()
		scraperStatus := map[string]interface{}{
			"id":               config.ID,
			"name":             config.Name,
			"url":              config.URL,
			"interval":         config.Interval.String(),
			"enabled":          config.Enabled,
			"change_detection": config.ChangeDetection,
			"persist_data":     config.PersistData,
			"team_code":        config.TeamCode,
		}

		// Try to get last results
		if lastResult, err := ss.GetScraperResults(config.ID); err == nil && lastResult != nil {
			scraperStatus["last_run"] = lastResult.Timestamp
			scraperStatus["last_item_count"] = lastResult.ItemCount
			scraperStatus["last_change_count"] = lastResult.ChangeCount
			scraperStatus["last_duration"] = lastResult.Duration.String()
			if lastResult.Error != nil {
				scraperStatus["last_error"] = lastResult.Error.Error()
			}
		}

		status[config.ID] = scraperStatus
	}

	return status
}

// GetLatestProducts returns the latest products from Fanatics
func (ss *ScraperService) GetLatestProducts(limit int) ([]models.ScrapedItem, error) {
	scraperID := fmt.Sprintf("fanatics_%s", strings.ToLower(ss.teamCode))

	if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
		items, err := scraperManager.dataStore.LoadLastItems(scraperID)
		if err != nil {
			return nil, err
		}

		// Apply limit
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}

		return items, nil
	}

	return nil, fmt.Errorf("unable to load latest products")
}

// GetLatestNews returns the latest news items
func (ss *ScraperService) GetLatestNews(limit int) ([]models.ScrapedItem, error) {
	if scraperManager, ok := ss.manager.(*ScraperManagerImpl); ok {
		items, err := scraperManager.dataStore.LoadLastItems("nhl_news")
		if err != nil {
			return nil, err
		}

		// Apply limit
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}

		return items, nil
	}

	return nil, fmt.Errorf("unable to load latest news")
}

// GetRecentChanges returns recent changes detected by scrapers
func (ss *ScraperService) GetRecentChanges(limit int) ([]*models.ScraperResult, error) {
	var results []*models.ScraperResult

	scrapers := ss.manager.GetAllScrapers()
	for _, scraper := range scrapers {
		if result, err := ss.GetScraperResults(scraper.GetID()); err == nil && result != nil {
			if len(result.Changes) > 0 {
				results = append(results, result)
			}
		}
	}

	// Sort by timestamp (newest first) and apply limit
	// This is a simplified implementation - in practice you'd want more sophisticated sorting
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// UpdateScraperConfig updates the configuration for a specific scraper
func (ss *ScraperService) UpdateScraperConfig(scraperID string, config models.ScraperConfig) error {
	scraper, err := ss.manager.GetScraper(scraperID)
	if err != nil {
		return err
	}

	scraper.SetConfig(config)
	return scraper.Validate()
}

// EnableScraper enables a specific scraper
func (ss *ScraperService) EnableScraper(scraperID string) error {
	scraper, err := ss.manager.GetScraper(scraperID)
	if err != nil {
		return err
	}

	config := scraper.GetConfig()
	config.Enabled = true
	scraper.SetConfig(config)
	return nil
}

// DisableScraper disables a specific scraper
func (ss *ScraperService) DisableScraper(scraperID string) error {
	scraper, err := ss.manager.GetScraper(scraperID)
	if err != nil {
		return err
	}

	config := scraper.GetConfig()
	config.Enabled = false
	scraper.SetConfig(config)
	return nil
}
