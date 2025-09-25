package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// ScraperManagerImpl implements the ScraperManager interface
type ScraperManagerImpl struct {
	scrapers    map[string]models.Scraper
	actions     map[string]models.ActionTrigger
	dataStore   ScraperDataStore
	mu          sync.RWMutex
	isRunning   bool
	stopChannel chan bool
	wg          sync.WaitGroup
}

// ScraperDataStore interface for persisting scraper data
type ScraperDataStore interface {
	SaveResults(scraperID string, results *models.ScraperResult) error
	LoadLastResults(scraperID string) (*models.ScraperResult, error)
	SaveItems(scraperID string, items []models.ScrapedItem) error
	LoadLastItems(scraperID string) ([]models.ScrapedItem, error)
}

// FileDataStore implements ScraperDataStore using JSON files
type FileDataStore struct {
	dataDir string
}

// NewFileDataStore creates a new file-based data store
func NewFileDataStore(dataDir string) *FileDataStore {
	// Create data directory if it doesn't exist
	os.MkdirAll(dataDir, 0755)

	return &FileDataStore{
		dataDir: dataDir,
	}
}

// SaveResults saves scraper results to a JSON file
func (fds *FileDataStore) SaveResults(scraperID string, results *models.ScraperResult) error {
	filename := filepath.Join(fds.dataDir, fmt.Sprintf("%s_results.json", scraperID))

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling results: %v", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadLastResults loads the last scraper results from a JSON file
func (fds *FileDataStore) LoadLastResults(scraperID string) (*models.ScraperResult, error) {
	filename := filepath.Join(fds.dataDir, fmt.Sprintf("%s_results.json", scraperID))

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No previous results
		}
		return nil, fmt.Errorf("error reading results file: %v", err)
	}

	var results models.ScraperResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("error unmarshaling results: %v", err)
	}

	return &results, nil
}

// SaveItems saves scraped items to a JSON file
func (fds *FileDataStore) SaveItems(scraperID string, items []models.ScrapedItem) error {
	filename := filepath.Join(fds.dataDir, fmt.Sprintf("%s_items.json", scraperID))

	// Convert items to a serializable format
	var serializable []map[string]interface{}
	for _, item := range items {
		itemData := map[string]interface{}{
			"id":        item.GetID(),
			"title":     item.GetTitle(),
			"url":       item.GetURL(),
			"timestamp": item.GetTimestamp(),
			"metadata":  item.GetMetadata(),
		}
		serializable = append(serializable, itemData)
	}

	data, err := json.MarshalIndent(serializable, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling items: %v", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadLastItems loads the last scraped items from a JSON file
func (fds *FileDataStore) LoadLastItems(scraperID string) ([]models.ScrapedItem, error) {
	filename := filepath.Join(fds.dataDir, fmt.Sprintf("%s_items.json", scraperID))

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No previous items
		}
		return nil, fmt.Errorf("error reading items file: %v", err)
	}

	var serializable []map[string]interface{}
	if err := json.Unmarshal(data, &serializable); err != nil {
		return nil, fmt.Errorf("error unmarshaling items: %v", err)
	}

	// Convert back to ScrapedItem interfaces
	var items []models.ScrapedItem
	for _, itemData := range serializable {
		// Determine item type based on metadata
		metadata, _ := itemData["metadata"].(map[string]interface{})
		itemType, _ := metadata["type"].(string)

		switch itemType {
		case "news":
			newsItem := &models.NewsItem{
				Title:     itemData["title"].(string),
				URL:       itemData["url"].(string),
				Timestamp: parseTimeFromInterface(itemData["timestamp"]),
			}
			if date, ok := metadata["date"].(string); ok {
				newsItem.Date = date
			}
			items = append(items, newsItem)

		case "product":
			productItem := &models.ProductItem{
				Title:     itemData["title"].(string),
				URL:       itemData["url"].(string),
				Timestamp: parseTimeFromInterface(itemData["timestamp"]),
			}
			if price, ok := metadata["price"].(float64); ok {
				productItem.Price = price
			}
			if currency, ok := metadata["currency"].(string); ok {
				productItem.Currency = currency
			}
			if available, ok := metadata["available"].(bool); ok {
				productItem.Available = available
			}
			// Add other product-specific fields as needed
			items = append(items, productItem)

		default:
			// Generic item
			genericItem := &models.GenericItem{
				Title:     itemData["title"].(string),
				URL:       itemData["url"].(string),
				Type:      itemType,
				Timestamp: parseTimeFromInterface(itemData["timestamp"]),
			}
			items = append(items, genericItem)
		}
	}

	return items, nil
}

// parseTimeFromInterface parses time from various interface types
func parseTimeFromInterface(timeInterface interface{}) time.Time {
	switch t := timeInterface.(type) {
	case string:
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed
		}
	case time.Time:
		return t
	}
	return time.Now()
}

// NewScraperManager creates a new scraper manager
func NewScraperManager(dataDir string) models.ScraperManager {
	return &ScraperManagerImpl{
		scrapers:    make(map[string]models.Scraper),
		actions:     make(map[string]models.ActionTrigger),
		dataStore:   NewFileDataStore(dataDir),
		stopChannel: make(chan bool),
	}
}

// RegisterScraper adds a new scraper to the manager
func (sm *ScraperManagerImpl) RegisterScraper(scraper models.Scraper) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := scraper.Validate(); err != nil {
		return fmt.Errorf("scraper validation failed: %v", err)
	}

	sm.scrapers[scraper.GetID()] = scraper
	fmt.Printf("Registered scraper: %s\n", scraper.GetID())
	return nil
}

// UnregisterScraper removes a scraper from the manager
func (sm *ScraperManagerImpl) UnregisterScraper(scraperID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.scrapers[scraperID]; !exists {
		return fmt.Errorf("scraper %s not found", scraperID)
	}

	delete(sm.scrapers, scraperID)
	fmt.Printf("Unregistered scraper: %s\n", scraperID)
	return nil
}

// GetScraper returns a specific scraper
func (sm *ScraperManagerImpl) GetScraper(scraperID string) (models.Scraper, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	scraper, exists := sm.scrapers[scraperID]
	if !exists {
		return nil, fmt.Errorf("scraper %s not found", scraperID)
	}

	return scraper, nil
}

// GetAllScrapers returns all registered scrapers
func (sm *ScraperManagerImpl) GetAllScrapers() []models.Scraper {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	scrapers := make([]models.Scraper, 0, len(sm.scrapers))
	for _, scraper := range sm.scrapers {
		scrapers = append(scrapers, scraper)
	}

	return scrapers
}

// RunScraper runs a specific scraper and handles change detection
func (sm *ScraperManagerImpl) RunScraper(scraperID string) (*models.ScraperResult, error) {
	sm.mu.RLock()
	scraper, exists := sm.scrapers[scraperID]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("scraper %s not found", scraperID)
	}

	fmt.Printf("Running scraper: %s\n", scraperID)

	// Run the scraper
	result, err := scraper.Scrape()
	if err != nil {
		return result, fmt.Errorf("scraper %s failed: %v", scraperID, err)
	}

	// Handle change detection if enabled
	config := scraper.GetConfig()
	if config.ChangeDetection {
		lastItems, _ := sm.dataStore.LoadLastItems(scraperID)
		if lastItems != nil {
			changes := scraper.ProcessChanges(lastItems, result.Items)
			result.Changes = changes
			result.ChangeCount = len(changes)

			// Trigger actions for changes
			if len(changes) > 0 {
				fmt.Printf("Scraper %s: Found %d changes\n", scraperID, len(changes))
				sm.triggerActions(changes)
			}
		}
	}

	// Persist data if enabled
	if config.PersistData {
		if err := sm.dataStore.SaveResults(scraperID, result); err != nil {
			fmt.Printf("Warning: Failed to save results for %s: %v\n", scraperID, err)
		}
		if err := sm.dataStore.SaveItems(scraperID, result.Items); err != nil {
			fmt.Printf("Warning: Failed to save items for %s: %v\n", scraperID, err)
		}
	}

	fmt.Printf("Scraper %s completed: %d items, %d changes\n",
		scraperID, result.ItemCount, result.ChangeCount)

	return result, nil
}

// RunAllScrapers runs all enabled scrapers
func (sm *ScraperManagerImpl) RunAllScrapers() ([]*models.ScraperResult, error) {
	sm.mu.RLock()
	scrapers := make([]models.Scraper, 0)
	for _, scraper := range sm.scrapers {
		if scraper.GetConfig().Enabled {
			scrapers = append(scrapers, scraper)
		}
	}
	sm.mu.RUnlock()

	var results []*models.ScraperResult
	var errors []error

	// Run scrapers concurrently
	resultChan := make(chan *models.ScraperResult, len(scrapers))
	errorChan := make(chan error, len(scrapers))

	for _, scraper := range scrapers {
		go func(s models.Scraper) {
			result, err := sm.RunScraper(s.GetID())
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}(scraper)
	}

	// Collect results
	for i := 0; i < len(scrapers); i++ {
		select {
		case result := <-resultChan:
			results = append(results, result)
		case err := <-errorChan:
			errors = append(errors, err)
		case <-time.After(5 * time.Minute): // Timeout
			errors = append(errors, fmt.Errorf("scraper timeout"))
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("some scrapers failed: %v", errors)
	}

	return results, nil
}

// Start begins automatic scraping based on scraper intervals
func (sm *ScraperManagerImpl) Start() error {
	sm.mu.Lock()
	if sm.isRunning {
		sm.mu.Unlock()
		return fmt.Errorf("scraper manager is already running")
	}
	sm.isRunning = true
	sm.mu.Unlock()

	fmt.Println("Starting scraper manager...")

	// Start a goroutine for each enabled scraper
	for _, scraper := range sm.scrapers {
		if scraper.GetConfig().Enabled {
			sm.wg.Add(1)
			go sm.runScraperLoop(scraper)
		}
	}

	return nil
}

// Stop stops automatic scraping
func (sm *ScraperManagerImpl) Stop() error {
	sm.mu.Lock()
	if !sm.isRunning {
		sm.mu.Unlock()
		return fmt.Errorf("scraper manager is not running")
	}
	sm.isRunning = false
	sm.mu.Unlock()

	fmt.Println("Stopping scraper manager...")

	// Signal all goroutines to stop
	close(sm.stopChannel)

	// Wait for all goroutines to finish
	sm.wg.Wait()

	// Recreate the stop channel for future use
	sm.stopChannel = make(chan bool)

	fmt.Println("Scraper manager stopped")
	return nil
}

// runScraperLoop runs a scraper in a loop based on its interval
func (sm *ScraperManagerImpl) runScraperLoop(scraper models.Scraper) {
	defer sm.wg.Done()

	config := scraper.GetConfig()
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	// Run immediately first time
	sm.RunScraper(scraper.GetID())

	for {
		select {
		case <-ticker.C:
			sm.RunScraper(scraper.GetID())
		case <-sm.stopChannel:
			return
		}
	}
}

// RegisterAction adds an action trigger to the manager
func (sm *ScraperManagerImpl) RegisterAction(action models.ActionTrigger) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.actions[action.GetID()] = action
	fmt.Printf("Registered action: %s\n", action.GetID())
	return nil
}

// triggerActions executes appropriate actions for the given changes
func (sm *ScraperManagerImpl) triggerActions(changes []models.Change) {
	sm.mu.RLock()
	actions := make([]models.ActionTrigger, 0, len(sm.actions))
	for _, action := range sm.actions {
		actions = append(actions, action)
	}
	sm.mu.RUnlock()

	for _, action := range actions {
		// Check which changes should trigger this action
		var triggeredChanges []models.Change
		for _, change := range changes {
			if action.ShouldTrigger(change) {
				triggeredChanges = append(triggeredChanges, change)
			}
		}

		// Execute action if there are triggered changes
		if len(triggeredChanges) > 0 {
			go func(a models.ActionTrigger, changes []models.Change) {
				if err := a.Execute(changes); err != nil {
					fmt.Printf("Action %s failed: %v\n", a.GetID(), err)
				}
			}(action, triggeredChanges)
		}
	}
}
