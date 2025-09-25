package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"golang.org/x/net/html"
)

// FanaticsScraper handles scraping products from Fanatics.com
type FanaticsScraper struct {
	*BaseScraper
	teamCode string
}

// NewFanaticsScraper creates a new Fanatics scraper for a specific team
func NewFanaticsScraper(teamCode string) *FanaticsScraper {
	// Build the Fanatics URL for the specific team
	// The URL pattern appears to be: /nhl/{team-name}/o-{league_id}+t-{team_id}+z-{category_id}
	teamURL := buildFanaticsURL(teamCode)

	config := models.ScraperConfig{
		ID:              fmt.Sprintf("fanatics_%s", strings.ToLower(teamCode)),
		Name:            fmt.Sprintf("Fanatics NHL %s Products", teamCode),
		URL:             teamURL,
		Interval:        30 * time.Minute, // Check every 30 minutes
		Enabled:         true,
		UserAgent:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		Timeout:         45 * time.Second, // Longer timeout for e-commerce sites
		MaxItems:        50,               // Limit to prevent excessive data
		ChangeDetection: true,
		PersistData:     true,
		TeamCode:        teamCode,
		Headers: map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
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
		},
		CustomSelectors: map[string]string{
			"product_container": ".product-card, .product-item, [data-testid*='product']",
			"product_title":     ".product-title, .product-name, h3, h4",
			"product_price":     ".price, .product-price, [data-testid*='price']",
			"product_link":      "a[href*='/p-']",
			"product_image":     "img[src*='product'], img[alt*='product']",
			"product_category":  ".category, .product-category",
		},
	}

	baseScraper := NewBaseScraper(config)

	return &FanaticsScraper{
		BaseScraper: baseScraper,
		teamCode:    teamCode,
	}
}

// Scrape performs the actual scraping of Fanatics products
func (fs *FanaticsScraper) Scrape() (*models.ScraperResult, error) {
	start := time.Now()
	result := &models.ScraperResult{
		ScraperID: fs.GetID(),
		Timestamp: start,
	}

	fs.LogDebug("Starting Fanatics product scrape")

	// Fetch HTML document
	doc, err := fs.FetchHTML()
	if err != nil {
		result.Error = err
		return result, err
	}

	// Extract products from the HTML
	items, err := fs.extractProducts(doc)
	if err != nil {
		result.Error = err
		return result, err
	}

	// Apply max items limit
	if fs.config.MaxItems > 0 && len(items) > fs.config.MaxItems {
		items = items[:fs.config.MaxItems]
	}

	result.Items = items
	result.ItemCount = len(items)
	result.Duration = time.Since(start)

	fs.LogDebug("Found %d products in %v", result.ItemCount, result.Duration)

	return result, nil
}

// extractProducts extracts product information from the HTML document
func (fs *FanaticsScraper) extractProducts(doc *html.Node) ([]models.ScrapedItem, error) {
	var products []models.ScrapedItem
	timestamp := time.Now()

	// Look for product containers using multiple selectors
	productContainers := fs.findProductContainers(doc)

	fs.LogDebug("Found %d potential product containers", len(productContainers))

	for _, container := range productContainers {
		product := fs.extractProductFromContainer(container, timestamp)
		if product != nil {
			products = append(products, product)
		}
	}

	// Remove duplicates
	products = fs.removeDuplicateProducts(products)

	return products, nil
}

// findProductContainers finds elements that likely contain product information
func (fs *FanaticsScraper) findProductContainers(doc *html.Node) []*html.Node {
	var containers []*html.Node

	// Try multiple strategies to find product containers
	selectors := []string{
		"product-card",
		"product-item",
		"ProductCard",
		"ProductItem",
		"product",
		"item",
	}

	for _, selector := range selectors {
		found := fs.FindElementsByClass(doc, selector)
		containers = append(containers, found...)

		if len(containers) > 0 {
			fs.LogDebug("Found %d containers with class containing '%s'", len(found), selector)
		}
	}

	// If we didn't find specific product containers, look for divs with data attributes
	if len(containers) == 0 {
		fs.LogDebug("No specific product containers found, looking for data attributes")
		containers = fs.findElementsWithProductData(doc)
	}

	// If still no luck, look for anchor tags with product URLs
	if len(containers) == 0 {
		fs.LogDebug("No containers with data attributes, looking for product links")
		containers = fs.findProductLinks(doc)
	}

	return containers
}

// findElementsWithProductData finds elements with product-related data attributes
func (fs *FanaticsScraper) findElementsWithProductData(doc *html.Node) []*html.Node {
	var elements []*html.Node

	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, attr := range node.Attr {
				if (attr.Key == "data-testid" || attr.Key == "data-product" || attr.Key == "data-item") &&
					strings.Contains(strings.ToLower(attr.Val), "product") {
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

// findProductLinks finds anchor tags that link to product pages
func (fs *FanaticsScraper) findProductLinks(doc *html.Node) []*html.Node {
	var links []*html.Node

	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			href := fs.GetAttributeValue(node, "href")
			// Look for common product URL patterns
			if strings.Contains(href, "/p-") || strings.Contains(href, "/product") ||
				strings.Contains(href, "-item-") {
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

// extractProductFromContainer extracts product information from a container element
func (fs *FanaticsScraper) extractProductFromContainer(container *html.Node, timestamp time.Time) models.ScrapedItem {
	// Extract product title
	title := fs.extractProductTitle(container)
	if title == "" {
		return nil // Skip if we can't find a title
	}

	// Extract product URL
	url := fs.extractProductURL(container)
	if url == "" {
		return nil // Skip if we can't find a URL
	}

	// Extract price
	price, currency := fs.extractProductPrice(container)

	// Extract image URL
	imageURL := fs.extractProductImage(container)

	// Extract category
	category := fs.extractProductCategory(container)

	// Check availability (assume available unless explicitly stated otherwise)
	available := fs.checkProductAvailability(container)

	product := &models.ProductItem{
		Title:     title,
		URL:       fs.MakeAbsoluteURL(url),
		Price:     price,
		Currency:  currency,
		ImageURL:  fs.MakeAbsoluteURL(imageURL),
		Available: available,
		Category:  category,
		Brand:     "Fanatics",
		TeamCode:  fs.teamCode,
		Timestamp: timestamp,
	}

	fs.LogDebug("Extracted product: %s - $%.2f", product.Title, product.Price)
	return product
}

// extractProductTitle extracts the product title from a container
func (fs *FanaticsScraper) extractProductTitle(container *html.Node) string {
	// Try multiple strategies to find the title
	selectors := []string{"product-title", "product-name", "title", "name"}

	for _, selector := range selectors {
		elements := fs.FindElementsByClass(container, selector)
		for _, elem := range elements {
			title := strings.TrimSpace(fs.ExtractTextFromNode(elem))
			if title != "" && len(title) > 5 { // Must be at least 5 characters
				return title
			}
		}
	}

	// Try header tags
	for _, tag := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		headers := fs.FindElementsByTag(container, tag)
		for _, header := range headers {
			title := strings.TrimSpace(fs.ExtractTextFromNode(header))
			if title != "" && len(title) > 5 {
				return title
			}
		}
	}

	// Try alt text from images
	images := fs.FindElementsByTag(container, "img")
	for _, img := range images {
		alt := fs.GetAttributeValue(img, "alt")
		if alt != "" && len(alt) > 10 && !strings.Contains(strings.ToLower(alt), "logo") {
			return alt
		}
	}

	return ""
}

// extractProductURL extracts the product URL from a container
func (fs *FanaticsScraper) extractProductURL(container *html.Node) string {
	// Look for anchor tags
	links := fs.FindElementsByTag(container, "a")
	for _, link := range links {
		href := fs.GetAttributeValue(link, "href")
		if href != "" && (strings.Contains(href, "/p-") || strings.Contains(href, "/product")) {
			return href
		}
	}

	// If container itself is an anchor
	if container.Data == "a" {
		return fs.GetAttributeValue(container, "href")
	}

	return ""
}

// extractProductPrice extracts the price and currency from a container
func (fs *FanaticsScraper) extractProductPrice(container *html.Node) (float64, string) {
	priceSelectors := []string{"price", "product-price", "cost", "amount"}

	for _, selector := range priceSelectors {
		elements := fs.FindElementsByClass(container, selector)
		for _, elem := range elements {
			priceText := strings.TrimSpace(fs.ExtractTextFromNode(elem))
			if price, currency := fs.parsePrice(priceText); price > 0 {
				return price, currency
			}
		}
	}

	// Look for price patterns in any text content
	allText := fs.ExtractTextFromNode(container)
	if price, currency := fs.findPriceInText(allText); price > 0 {
		return price, currency
	}

	return 0, "USD"
}

// parsePrice parses price from text
func (fs *FanaticsScraper) parsePrice(priceText string) (float64, string) {
	// Common price patterns
	patterns := []string{
		`\$(\d+\.?\d*)`,          // $19.99
		`(\d+\.?\d*)\s*USD`,      // 19.99 USD
		`(\d+\.?\d*)\s*dollars?`, // 19.99 dollars
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(priceText)
		if len(matches) > 1 {
			if price, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return price, "USD"
			}
		}
	}

	return 0, ""
}

// findPriceInText finds price patterns in any text
func (fs *FanaticsScraper) findPriceInText(text string) (float64, string) {
	// Look for dollar amounts
	re := regexp.MustCompile(`\$(\d+(?:\.\d{2})?)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if price, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return price, "USD"
		}
	}
	return 0, ""
}

// extractProductImage extracts the product image URL
func (fs *FanaticsScraper) extractProductImage(container *html.Node) string {
	images := fs.FindElementsByTag(container, "img")
	for _, img := range images {
		src := fs.GetAttributeValue(img, "src")
		if src != "" && !strings.Contains(strings.ToLower(src), "logo") {
			return src
		}
	}
	return ""
}

// extractProductCategory extracts the product category
func (fs *FanaticsScraper) extractProductCategory(container *html.Node) string {
	categorySelectors := []string{"category", "product-category", "breadcrumb"}

	for _, selector := range categorySelectors {
		elements := fs.FindElementsByClass(container, selector)
		for _, elem := range elements {
			category := strings.TrimSpace(fs.ExtractTextFromNode(elem))
			if category != "" {
				return category
			}
		}
	}

	return "NHL Merchandise"
}

// checkProductAvailability checks if a product is available
func (fs *FanaticsScraper) checkProductAvailability(container *html.Node) bool {
	text := strings.ToLower(fs.ExtractTextFromNode(container))

	// Look for unavailable indicators
	unavailableKeywords := []string{
		"out of stock", "sold out", "unavailable", "discontinued",
		"back order", "backorder", "pre-order", "coming soon",
	}

	for _, keyword := range unavailableKeywords {
		if strings.Contains(text, keyword) {
			return false
		}
	}

	return true // Assume available if no unavailable indicators
}

// removeDuplicateProducts removes duplicate products based on URL
func (fs *FanaticsScraper) removeDuplicateProducts(products []models.ScrapedItem) []models.ScrapedItem {
	seen := make(map[string]bool)
	var unique []models.ScrapedItem

	for _, product := range products {
		url := product.GetURL()
		if !seen[url] {
			seen[url] = true
			unique = append(unique, product)
		}
	}

	return unique
}

// buildFanaticsURL builds the appropriate Fanatics URL for a team
func buildFanaticsURL(teamCode string) string {
	// Map team codes to Fanatics URLs
	// This would need to be expanded for all teams
	teamURLMap := map[string]string{
		"UTA": "https://www.fanatics.com/nhl/utah-hockey-club/o-3528+t-8357454616+z-9818-3541420422?_ref=m-TOPNAV&sortOption=TopSellers&pageSize=72&filters=%7B%22d%22%3A%5B%228011%22%5D%7D",
		"COL": "https://www.fanatics.com/nhl/colorado-avalanche/o-3528+t-70383449+z-9818-3541420422?_ref=m-TOPNAV&sortOption=TopSellers&pageSize=72",
		"TOR": "https://www.fanatics.com/nhl/toronto-maple-leafs/o-3528+t-81041493+z-9818-3541420422?_ref=m-TOPNAV&sortOption=TopSellers&pageSize=72",
		"BOS": "https://www.fanatics.com/nhl/boston-bruins/o-3528+t-92487538+z-9818-3541420422?_ref=m-TOPNAV&sortOption=TopSellers&pageSize=72",
		// Add more teams as needed
	}

	if url, exists := teamURLMap[teamCode]; exists {
		return url
	}

	// Default fallback URL
	return fmt.Sprintf("https://www.fanatics.com/nhl/o-3528+z-9818-3541420422?query=%s", teamCode)
}
