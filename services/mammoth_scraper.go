package services

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"golang.org/x/net/html"
)

// MammothScraper scrapes product information from mammothteamstore.com
type MammothScraper struct {
	*BaseScraper
}

// NewMammothScraper creates a new MammothScraper instance
func NewMammothScraper() *MammothScraper {
	config := models.ScraperConfig{
		ID:              "mammoth_store",
		Name:            "Utah Mammoth Team Store",
		URL:             "https://www.mammothteamstore.com/en/utah-mammoth-men/t-1657129183+ga-45+z-96181-308137989?sortOption=NewestArrivals",
		Interval:        15 * time.Minute, // Check every 15 minutes for new arrivals
		Enabled:         true,
		UserAgent:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		Timeout:         60 * time.Second, // Longer timeout for e-commerce sites
		MaxItems:        30,               // Limit to prevent excessive data
		ChangeDetection: true,
		PersistData:     true,
		TeamCode:        "UTA",
		Headers: map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
			"Accept-Language":           "en-US,en;q=0.9",
			"Accept-Encoding":           "gzip, deflate, br, zstd",
			"Cache-Control":             "max-age=0",
			"DNT":                       "1",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
			"Sec-Fetch-User":            "?1",
			"Sec-Ch-Ua":                 `"Chromium";v="128", "Not;A=Brand";v="24", "Google Chrome";v="128"`,
			"Sec-Ch-Ua-Mobile":          "?0",
			"Sec-Ch-Ua-Platform":        `"macOS"`,
		},
	}

	baseScraper := NewBaseScraper(config)

	return &MammothScraper{
		BaseScraper: baseScraper,
	}
}

// Scrape performs the scraping operation for Mammoth team store products
func (ms *MammothScraper) Scrape() (*models.ScraperResult, error) {
	startTime := time.Now()

	result := &models.ScraperResult{
		ScraperID: ms.GetID(),
		Timestamp: time.Now(),
	}

	doc, err := ms.FetchHTML()
	if err != nil {
		ms.LogError("Failed to fetch HTML: %v", err)
		result.Error = err
		result.Duration = time.Since(startTime)
		return result, err
	}

	products, err := ms.extractProducts(doc)
	if err != nil {
		ms.LogError("Failed to extract products: %v", err)
		result.Error = err
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Filter out duplicates
	products = ms.filterUniqueProducts(products)

	// Convert to ScrapedItem interface
	scrapedItems := make([]models.ScrapedItem, len(products))
	for i, p := range products {
		scrapedItems[i] = p
	}

	// Apply max items limit if configured
	if ms.GetConfig().MaxItems > 0 && len(scrapedItems) > ms.GetConfig().MaxItems {
		scrapedItems = scrapedItems[:ms.GetConfig().MaxItems]
	}

	result.Items = scrapedItems
	result.ItemCount = len(scrapedItems)
	result.Duration = time.Since(startTime)

	ms.LogDebug("Found %d products in %v", result.ItemCount, result.Duration)

	return result, nil
}

// ProcessChanges detects changes between old and new product lists
func (ms *MammothScraper) ProcessChanges(oldItems, newItems []models.ScrapedItem) []models.Change {
	changes := []models.Change{}
	oldMap := make(map[string]models.ScrapedItem)
	newMap := make(map[string]models.ScrapedItem)

	for _, item := range oldItems {
		oldMap[item.GetID()] = item
	}
	for _, item := range newItems {
		newMap[item.GetID()] = item
	}

	// Check for new and updated items
	for newID, newItem := range newMap {
		if oldItem, exists := oldMap[newID]; exists {
			// Item exists, check for updates
			if !newItem.Equals(oldItem) {
				changes = append(changes, models.Change{
					Type:      models.ChangeTypeUpdated,
					Item:      newItem,
					Previous:  oldItem,
					Timestamp: time.Now(),
				})
			}
		} else {
			// New item
			changes = append(changes, models.Change{
				Type:      models.ChangeTypeNew,
				Item:      newItem,
				Timestamp: time.Now(),
			})
		}
	}

	// Check for removed items
	for oldID, oldItem := range oldMap {
		if _, exists := newMap[oldID]; !exists {
			changes = append(changes, models.Change{
				Type:      models.ChangeTypeRemoved,
				Item:      oldItem,
				Timestamp: time.Now(),
			})
		}
	}

	return changes
}

// SetConfig updates the scraper configuration
func (ms *MammothScraper) SetConfig(config models.ScraperConfig) {
	ms.BaseScraper.SetConfig(config)
}

// Validate checks if the scraper configuration is valid
func (ms *MammothScraper) Validate() error {
	config := ms.GetConfig()
	if config.URL == "" {
		return fmt.Errorf("scraper URL cannot be empty")
	}
	if config.ID == "" {
		return fmt.Errorf("scraper ID cannot be empty")
	}
	return nil
}

// extractProducts extracts product details from the HTML document
func (ms *MammothScraper) extractProducts(doc *html.Node) ([]*models.ProductItem, error) {
	var products []*models.ProductItem
	timestamp := time.Now()

	// Look for product containers using multiple selectors
	productContainers := ms.findProductContainers(doc)

	ms.LogDebug("Found %d potential product containers", len(productContainers))

	if len(productContainers) == 0 {
		// Fallback: try to find elements with product data attributes
		productContainers = ms.findElementsWithProductData(doc)
		ms.LogDebug("Fallback: Found %d elements with product data attributes", len(productContainers))
	}

	if len(productContainers) == 0 {
		// Another fallback: look for common product link patterns
		productLinks := ms.findProductLinks(doc)
		ms.LogDebug("Fallback: Found %d product links", len(productLinks))
		for _, link := range productLinks {
			title := ms.ExtractTextFromNode(link)
			url := ms.GetAttributeValue(link, "href")
			if title != "" && url != "" {
				products = append(products, &models.ProductItem{
					Title:     strings.TrimSpace(title),
					URL:       ms.MakeAbsoluteURL(url),
					Available: true, // Assume available if found
					Timestamp: timestamp,
					TeamCode:  "UTA",
					Brand:     "Mammoth Team Store",
				})
			}
		}
		if len(products) > 0 {
			return products, nil
		}
	}

	for _, container := range productContainers {
		product, err := ms.parseProductContainer(container, timestamp)
		if err != nil {
			ms.LogError("Error parsing product container: %v", err)
			continue
		}
		if product != nil {
			products = append(products, product)
		}
	}

	return products, nil
}

// findProductContainers tries to locate HTML nodes that represent individual product listings
func (ms *MammothScraper) findProductContainers(doc *html.Node) []*html.Node {
	var containers []*html.Node
	// Common class names for product listings on e-commerce sites
	selectors := []string{
		"product-item", "product-card", "product", "item", "grid-item",
		"product-container", "product-tile", "card-product", "item-card",
		"product-summary", "product-listing", "merchandise-item",
	}

	for _, selector := range selectors {
		found := ms.FindElementsByClass(doc, selector)
		containers = append(containers, found...)

		if len(found) > 0 {
			ms.LogDebug("Found %d containers with class containing '%s'", len(found), selector)
		}
	}

	// Also try data attributes
	dataElements := ms.findElementsWithProductData(doc)
	containers = append(containers, dataElements...)
	if len(dataElements) > 0 {
		ms.LogDebug("Found %d containers with product data attributes", len(dataElements))
	}

	return containers
}

// findElementsWithProductData looks for elements that have data attributes indicating product info
func (ms *MammothScraper) findElementsWithProductData(doc *html.Node) []*html.Node {
	var elements []*html.Node

	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, attr := range node.Attr {
				if (attr.Key == "data-testid" || attr.Key == "data-product" ||
					attr.Key == "data-item" || attr.Key == "data-sku") &&
					(strings.Contains(strings.ToLower(attr.Val), "product") ||
						strings.Contains(strings.ToLower(attr.Val), "item")) {
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

// findProductLinks looks for anchor tags that are likely product links
func (ms *MammothScraper) findProductLinks(doc *html.Node) []*html.Node {
	var links []*html.Node

	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			href := ms.GetAttributeValue(node, "href")
			// Look for product URL patterns specific to mammoth store
			if strings.Contains(href, "/p-") || strings.Contains(href, "/product") ||
				strings.Contains(href, "-item-") || strings.Contains(href, "mammoth") {
				links = append(links, node)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	traverse(doc)
	return links
}

// parseProductContainer extracts details from a single product container node
func (ms *MammothScraper) parseProductContainer(container *html.Node, timestamp time.Time) (*models.ProductItem, error) {
	// Extract URL (usually from an <a> tag within the container)
	var url string
	links := ms.FindElementsByTag(container, "a")
	if len(links) > 0 {
		url = ms.GetAttributeValue(links[0], "href")
	}
	if url == "" {
		// Fallback: sometimes the container itself has a data-url or similar
		url = ms.GetAttributeValue(container, "data-url")
	}
	if url == "" {
		ms.LogDebug("Could not find URL for a product container, skipping.")
		return nil, nil // Skip if no URL
	}

	// Extract title
	title := ms.extractProductTitle(container)
	if title == "" {
		ms.LogDebug("Could not find title for product at %s, skipping.", url)
		return nil, nil // Skip if no title
	}

	// Extract price
	price, currency := ms.extractProductPrice(container)

	// Extract image URL
	imageURL := ms.extractProductImage(container)

	// Extract category
	category := ms.extractProductCategory(container)

	// Check availability
	available := ms.checkProductAvailability(container)

	product := &models.ProductItem{
		Title:     title,
		URL:       ms.MakeAbsoluteURL(url),
		Price:     price,
		Currency:  currency,
		ImageURL:  ms.MakeAbsoluteURL(imageURL),
		Available: available,
		Category:  category,
		Brand:     "Mammoth Team Store",
		TeamCode:  "UTA",
		Timestamp: timestamp,
	}

	return product, nil
}

// extractProductTitle tries to find the product title within a container
func (ms *MammothScraper) extractProductTitle(container *html.Node) string {
	// Try multiple strategies to find the title
	selectors := []string{"product-title", "product-name", "title", "name", "item-title"}

	for _, selector := range selectors {
		elements := ms.FindElementsByClass(container, selector)
		if len(elements) > 0 {
			text := ms.ExtractTextFromNode(elements[0])
			if text != "" {
				return strings.TrimSpace(text)
			}
		}
	}

	// Fallback: look for h1, h2, h3, h4 tags
	for _, tag := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		elements := ms.FindElementsByTag(container, tag)
		if len(elements) > 0 {
			text := ms.ExtractTextFromNode(elements[0])
			if text != "" && len(text) > 3 { // Basic length check
				return strings.TrimSpace(text)
			}
		}
	}

	// Last resort: check the alt text of an image or title attribute of a link
	imgElements := ms.FindElementsByTag(container, "img")
	if len(imgElements) > 0 {
		altText := ms.GetAttributeValue(imgElements[0], "alt")
		if altText != "" && len(altText) > 3 {
			return strings.TrimSpace(altText)
		}
	}

	linkElements := ms.FindElementsByTag(container, "a")
	if len(linkElements) > 0 {
		titleText := ms.GetAttributeValue(linkElements[0], "title")
		if titleText != "" && len(titleText) > 3 {
			return strings.TrimSpace(titleText)
		}
	}

	return ""
}

// extractProductPrice tries to find the product price within a container
func (ms *MammothScraper) extractProductPrice(container *html.Node) (float64, string) {
	priceSelectors := []string{"price", "product-price", "cost", "amount", "currency"}

	for _, selector := range priceSelectors {
		elements := ms.FindElementsByClass(container, selector)
		if len(elements) > 0 {
			priceText := ms.ExtractTextFromNode(elements[0])
			if priceText != "" {
				price, err := models.ParsePrice(priceText)
				if err == nil {
					currency := "$" // Default to USD
					if strings.Contains(priceText, "€") {
						currency = "€"
					} else if strings.Contains(priceText, "£") {
						currency = "£"
					} else if strings.Contains(priceText, "CAD") {
						currency = "CAD"
					}
					return price, currency
				}
			}
		}
	}

	// Fallback: regex search for price patterns in the entire container text
	fullText := ms.ExtractTextFromNode(container)
	patterns := []string{
		`\$(\d+\.?\d*)`,          // $19.99
		`(\d+\.?\d*)\s*USD`,      // 19.99 USD
		`(\d+\.?\d*)\s*dollars?`, // 19.99 dollars
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(fullText); len(matches) > 1 {
			price, err := models.ParsePrice(matches[1])
			if err == nil {
				return price, "$" // Assume USD for now
			}
		}
	}

	return 0.0, "USD" // Default if not found
}

// extractProductImage tries to find the product image URL within a container
func (ms *MammothScraper) extractProductImage(container *html.Node) string {
	imgElements := ms.FindElementsByTag(container, "img")
	if len(imgElements) > 0 {
		src := ms.GetAttributeValue(imgElements[0], "src")
		if src == "" {
			src = ms.GetAttributeValue(imgElements[0], "data-src") // Common for lazy loading
		}
		if src == "" {
			src = ms.GetAttributeValue(imgElements[0], "data-lazy-src") // Another lazy loading pattern
		}
		return src
	}
	return ""
}

// extractProductCategory tries to find the product category within a container
func (ms *MammothScraper) extractProductCategory(container *html.Node) string {
	categorySelectors := []string{"category", "product-category", "breadcrumb", "type"}

	for _, selector := range categorySelectors {
		elements := ms.FindElementsByClass(container, selector)
		if len(elements) > 0 {
			text := ms.ExtractTextFromNode(elements[0])
			if text != "" {
				return strings.TrimSpace(text)
			}
		}
	}

	return "Utah Mammoth Merchandise"
}

// checkProductAvailability checks if a product is available based on text indicators
func (ms *MammothScraper) checkProductAvailability(container *html.Node) bool {
	text := strings.ToLower(ms.ExtractTextFromNode(container))

	// Look for unavailable indicators
	unavailableKeywords := []string{
		"out of stock", "sold out", "unavailable", "discontinued",
		"back order", "backorder", "pre-order", "coming soon",
		"not available", "temporarily unavailable",
	}

	for _, keyword := range unavailableKeywords {
		if strings.Contains(text, keyword) {
			return false
		}
	}

	return true // Assume available if no unavailable indicators
}

// filterUniqueProducts removes duplicate products based on URL and Title
func (ms *MammothScraper) filterUniqueProducts(products []*models.ProductItem) []*models.ProductItem {
	seen := make(map[string]bool)
	var unique []*models.ProductItem

	for _, product := range products {
		id := product.GetID() // Use the ID for uniqueness
		if _, ok := seen[id]; !ok {
			seen[id] = true
			unique = append(unique, product)
		}
	}

	return unique
}
