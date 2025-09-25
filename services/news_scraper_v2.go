package services

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
	"golang.org/x/net/html"
)

// NewsScraperV2 handles scraping NHL news using the new scraper interface
type NewsScraperV2 struct {
	*BaseScraper
}

// NewNewsScraperV2 creates a new NHL news scraper
func NewNewsScraperV2() *NewsScraperV2 {
	config := models.ScraperConfig{
		ID:              "nhl_news",
		Name:            "NHL Official News",
		URL:             "https://www.nhl.com/news",
		Interval:        10 * time.Minute,
		Enabled:         true,
		UserAgent:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		Timeout:         30 * time.Second,
		MaxItems:        10,
		ChangeDetection: true,
		PersistData:     true,
	}

	baseScraper := NewBaseScraper(config)

	return &NewsScraperV2{
		BaseScraper: baseScraper,
	}
}

// Scrape performs the actual scraping of NHL news
func (ns *NewsScraperV2) Scrape() (*models.ScraperResult, error) {
	start := time.Now()
	result := &models.ScraperResult{
		ScraperID: ns.GetID(),
		Timestamp: start,
	}

	ns.LogDebug("Starting NHL news scrape")

	// Fetch HTML document
	doc, err := ns.FetchHTML()
	if err != nil {
		result.Error = err
		return result, err
	}

	// Search for interesting classes (for debugging)
	ns.searchForInterestingClasses(doc)

	// Extract headlines
	items, err := ns.extractHeadlines(doc)
	if err != nil {
		result.Error = err
		return result, err
	}

	// Apply max items limit
	if ns.config.MaxItems > 0 && len(items) > ns.config.MaxItems {
		items = items[:ns.config.MaxItems]
	}

	result.Items = items
	result.ItemCount = len(items)
	result.Duration = time.Since(start)

	ns.LogDebug("Found %d headlines in %v", result.ItemCount, result.Duration)

	return result, nil
}

// searchForInterestingClasses helps debug the HTML structure
func (ns *NewsScraperV2) searchForInterestingClasses(doc *html.Node) {
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "div" {
			for _, attr := range node.Attr {
				if attr.Key == "class" {
					classVal := attr.Val
					if strings.Contains(classVal, "story") ||
						strings.Contains(classVal, "title") ||
						strings.Contains(classVal, "news") ||
						strings.Contains(classVal, "article") ||
						strings.Contains(classVal, "card") {
						ns.LogDebug("Found interesting div with classes: %s", classVal)
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	traverse(doc)
}

// extractHeadlines extracts news headlines from the HTML document
func (ns *NewsScraperV2) extractHeadlines(doc *html.Node) ([]models.ScrapedItem, error) {
	var headlines []models.ScrapedItem
	timestamp := time.Now()

	// Look for the specific structure we found in the NHL.com page
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		// Look for anchor tags with href containing "/news/"
		if node.Type == html.ElementNode && node.Data == "a" {
			href := ns.GetAttributeValue(node, "href")

			// Check if this is a news article link
			if strings.Contains(href, "/news/") {
				title := ns.ExtractTextFromNode(node)
				if title != "" && len(title) > 10 && !ns.isTopicPage(title) {
					// Clean up the title
					title = strings.TrimSpace(title)
					title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

					// Make sure URL is absolute
					if strings.HasPrefix(href, "/") {
						href = "https://www.nhl.com" + href
					}

					newsItem := &models.NewsItem{
						Title:     title,
						URL:       href,
						Date:      ns.extractDateFromTitle(title),
						Timestamp: timestamp,
					}

					// Check for duplicates
					isDuplicate := false
					for _, existing := range headlines {
						if existing.GetTitle() == newsItem.GetTitle() || existing.GetURL() == newsItem.GetURL() {
							isDuplicate = true
							break
						}
					}

					if !isDuplicate {
						headlines = append(headlines, newsItem)
					}
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	traverse(doc)

	// Sort headlines by date (most recent first), fallback to title for same dates
	sort.Slice(headlines, func(i, j int) bool {
		newsI := headlines[i].(*models.NewsItem)
		newsJ := headlines[j].(*models.NewsItem)

		dateI, errI := time.Parse("2006-01-02", newsI.Date)
		dateJ, errJ := time.Parse("2006-01-02", newsJ.Date)

		// If both dates parse successfully, sort by date (newest first)
		if errI == nil && errJ == nil {
			return dateI.After(dateJ)
		}

		// If one date fails to parse, put it at the end
		if errI != nil && errJ == nil {
			return false
		}
		if errI == nil && errJ != nil {
			return true
		}

		// If both fail to parse, sort by title
		return newsI.Title < newsJ.Title
	})

	return headlines, nil
}

// isTopicPage checks if a headline is a category/topic page rather than a news article
func (ns *NewsScraperV2) isTopicPage(title string) bool {
	topicPageTitles := []string{
		"Situation Room",
		"NHL Insider",
		"Short Shifts",
		"Free Agency",
		"Trade Tracker",
		"Injury Report",
		"Statistics",
		"Standings",
		"Schedule",
		"Scores",
		"News",
		"Videos",
	}

	titleLower := strings.ToLower(title)
	for _, topic := range topicPageTitles {
		if strings.ToLower(topic) == titleLower {
			return true
		}
	}

	// Also filter out very generic/short titles without dates
	if !regexp.MustCompile(`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},\s+\d{4}\b`).MatchString(title) {
		if len(title) < 20 || strings.Count(title, " ") < 3 {
			return true
		}
	}

	return false
}

// extractDateFromTitle extracts date from headline title text in format "Month DD, YYYY"
func (ns *NewsScraperV2) extractDateFromTitle(title string) string {
	// Try to find date patterns in the title
	patterns := []string{
		`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2}),\s+(\d{4})`,
		`(\d{1,2})/(\d{1,2})/(\d{4})`,
		`(\d{4})-(\d{2})-(\d{2})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(title)
		if len(matches) > 0 {
			// Try to parse and standardize the date
			if standardDate := ns.standardizeDate(matches[0]); standardDate != "" {
				return standardDate
			}
		}
	}

	// If no date found in title, use current date
	return time.Now().Format("2006-01-02")
}

// standardizeDate converts various date formats to YYYY-MM-DD
func (ns *NewsScraperV2) standardizeDate(dateStr string) string {
	// Month name to number mapping
	monthMap := map[string]string{
		"Jan": "01", "Feb": "02", "Mar": "03", "Apr": "04",
		"May": "05", "Jun": "06", "Jul": "07", "Aug": "08",
		"Sep": "09", "Oct": "10", "Nov": "11", "Dec": "12",
	}

	// Try parsing "Month DD, YYYY" format
	monthPattern := regexp.MustCompile(`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2}),\s+(\d{4})`)
	if matches := monthPattern.FindStringSubmatch(dateStr); len(matches) == 4 {
		month := monthMap[matches[1]]
		day := matches[2]
		if len(day) == 1 {
			day = "0" + day
		}
		year := matches[3]
		return year + "-" + month + "-" + day
	}

	// Try parsing "MM/DD/YYYY" format
	slashPattern := regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4})`)
	if matches := slashPattern.FindStringSubmatch(dateStr); len(matches) == 4 {
		month := matches[1]
		if len(month) == 1 {
			month = "0" + month
		}
		day := matches[2]
		if len(day) == 1 {
			day = "0" + day
		}
		year := matches[3]
		return year + "-" + month + "-" + day
	}

	// If already in YYYY-MM-DD format, validate and return
	dashPattern := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	if matches := dashPattern.FindStringSubmatch(dateStr); len(matches) == 4 {
		return dateStr
	}

	return ""
}
