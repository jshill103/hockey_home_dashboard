package services

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"golang.org/x/net/html"
)

// BaseScraper provides common functionality for all scrapers
type BaseScraper struct {
	config models.ScraperConfig
	client *http.Client
}

// NewBaseScraper creates a new base scraper with the given configuration
func NewBaseScraper(config models.ScraperConfig) *BaseScraper {
	// Create HTTP client with configured timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	return &BaseScraper{
		config: config,
		client: client,
	}
}

// GetID returns the scraper ID
func (bs *BaseScraper) GetID() string {
	return bs.config.ID
}

// GetConfig returns the scraper configuration
func (bs *BaseScraper) GetConfig() models.ScraperConfig {
	return bs.config
}

// SetConfig updates the scraper configuration
func (bs *BaseScraper) SetConfig(config models.ScraperConfig) {
	bs.config = config

	// Update HTTP client timeout if changed
	if config.Timeout != 0 {
		bs.client.Timeout = config.Timeout
	}
}

// Validate checks if the scraper configuration is valid
func (bs *BaseScraper) Validate() error {
	if bs.config.ID == "" {
		return fmt.Errorf("scraper ID cannot be empty")
	}
	if bs.config.URL == "" {
		return fmt.Errorf("scraper URL cannot be empty")
	}
	if bs.config.Interval <= 0 {
		return fmt.Errorf("scraper interval must be positive")
	}
	return nil
}

// FetchHTML makes an HTTP request and returns the parsed HTML document
func (bs *BaseScraper) FetchHTML() (*html.Node, error) {
	fmt.Printf("Scraper [%s]: Fetching %s\n", bs.config.ID, bs.config.URL)

	// Create HTTP request
	req, err := http.NewRequest("GET", bs.config.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set User-Agent
	userAgent := bs.config.UserAgent
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (compatible; NHL-Dashboard-Bot/1.0)"
	}
	req.Header.Set("User-Agent", userAgent)

	// Set additional headers
	for key, value := range bs.config.Headers {
		req.Header.Set(key, value)
	}

	// Make the request
	resp, err := bs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Scraper [%s]: HTTP Response Status: %d\n", bs.config.ID, resp.StatusCode)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	fmt.Printf("Scraper [%s]: Response body length: %d bytes\n", bs.config.ID, len(body))

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %v", err)
	}

	return doc, nil
}

// ProcessChanges compares old and new items to detect changes
func (bs *BaseScraper) ProcessChanges(old, new []models.ScrapedItem) []models.Change {
	var changes []models.Change
	timestamp := time.Now()

	// Create maps for efficient lookup
	oldMap := make(map[string]models.ScrapedItem)
	newMap := make(map[string]models.ScrapedItem)

	for _, item := range old {
		oldMap[item.GetID()] = item
	}

	for _, item := range new {
		newMap[item.GetID()] = item
	}

	// Find new and updated items
	for id, newItem := range newMap {
		if oldItem, exists := oldMap[id]; exists {
			// Item exists, check if it's changed
			if !newItem.Equals(oldItem) {
				changes = append(changes, models.Change{
					Type:      models.ChangeTypeUpdated,
					Item:      newItem,
					Previous:  oldItem,
					Timestamp: timestamp,
				})
			}
		} else {
			// New item
			changes = append(changes, models.Change{
				Type:      models.ChangeTypeNew,
				Item:      newItem,
				Timestamp: timestamp,
			})
		}
	}

	// Find removed items
	for id, oldItem := range oldMap {
		if _, exists := newMap[id]; !exists {
			changes = append(changes, models.Change{
				Type:      models.ChangeTypeRemoved,
				Item:      oldItem,
				Timestamp: timestamp,
			})
		}
	}

	return changes
}

// ExtractTextFromNode extracts all text content from an HTML node
func (bs *BaseScraper) ExtractTextFromNode(n *html.Node) string {
	var text strings.Builder
	var traverse func(*html.Node)

	traverse = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	traverse(n)
	return strings.TrimSpace(text.String())
}

// FindElementsByClass finds all elements with the specified CSS class
func (bs *BaseScraper) FindElementsByClass(doc *html.Node, className string) []*html.Node {
	var elements []*html.Node

	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, attr := range node.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, className) {
					elements = append(elements, node)
					break
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	traverse(doc)
	return elements
}

// FindElementsByTag finds all elements with the specified tag name
func (bs *BaseScraper) FindElementsByTag(doc *html.Node, tagName string) []*html.Node {
	var elements []*html.Node

	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tagName {
			elements = append(elements, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	traverse(doc)
	return elements
}

// GetAttributeValue gets the value of a specific attribute from an HTML node
func (bs *BaseScraper) GetAttributeValue(node *html.Node, attributeName string) string {
	for _, attr := range node.Attr {
		if attr.Key == attributeName {
			return attr.Val
		}
	}
	return ""
}

// LogDebug logs debug information for the scraper
func (bs *BaseScraper) LogDebug(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("Scraper [%s]: %s\n", bs.config.ID, message)
}

// LogError logs error information for the scraper
func (bs *BaseScraper) LogError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("Scraper [%s] ERROR: %s\n", bs.config.ID, message)
}

// MakeAbsoluteURL converts relative URLs to absolute URLs
func (bs *BaseScraper) MakeAbsoluteURL(href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}

	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}

	if strings.HasPrefix(href, "/") {
		// Extract base URL from config URL
		baseURL := bs.config.URL
		if idx := strings.Index(baseURL[8:], "/"); idx != -1 {
			baseURL = baseURL[:8+idx]
		}
		return baseURL + href
	}

	// Relative path - this is more complex and might need more sophisticated handling
	return bs.config.URL + "/" + href
}
