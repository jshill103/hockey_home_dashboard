package services

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
	"github.com/jaredshillingburg/go_uhc/models"
)

// ScrapeNHLNews scrapes NHL news headlines from NHL.com
func ScrapeNHLNews() ([]models.NewsHeadline, error) {
	url := "https://www.nhl.com/news"
	
	fmt.Printf("Starting to scrape NHL news from: %s\n", url)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Create request with user agent to avoid being blocked
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	
	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()
	
	fmt.Printf("HTTP Response Status: %d\n", resp.StatusCode)
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}
	
	fmt.Printf("Response body length: %d bytes\n", len(body))
	
	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %v", err)
	}
	
	// Debug: Search for any div elements with interesting classes
	searchForInterestingClasses(doc)
	
	// Extract headlines
	headlines := extractHeadlines(doc)
	
	fmt.Printf("Found %d headlines\n", len(headlines))
	
	return headlines, nil
}

func searchForInterestingClasses(n *html.Node) {
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
						fmt.Printf("Found interesting div with classes: %s\n", classVal)
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	traverse(n)
}

func extractHeadlines(n *html.Node) []models.NewsHeadline {
	var headlines []models.NewsHeadline
	
	// Look for the specific structure we found in the NHL.com page
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		// Look for anchor tags with href containing "/news/"
		if node.Type == html.ElementNode && node.Data == "a" {
			var href string
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					href = attr.Val
					break
				}
			}
			
			// Check if this is a news article link
			if strings.Contains(href, "/news/") {
				title := extractTextFromNode(node)
				if title != "" && len(title) > 10 && !isTopicPage(title) { // Filter out very short titles and topic pages
					// Clean up the title
					title = strings.TrimSpace(title)
					title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
					
					// Make sure URL is absolute
					if strings.HasPrefix(href, "/") {
						href = "https://www.nhl.com" + href
					}
					
					headline := models.NewsHeadline{
						Title: title,
						URL:   href,
						Date:  "", // Will be extracted from title later
					}
					
					// Check for duplicates
					isDuplicate := false
					for _, existing := range headlines {
						if existing.Title == headline.Title || existing.URL == headline.URL {
							isDuplicate = true
							break
						}
					}
					
					if !isDuplicate {
						headlines = append(headlines, headline)
					}
				}
			}
		}
		
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	
	// Extract dates from headline text and sort by date (most recent first)
	for i := range headlines {
		headlines[i].Date = extractDateFromTitle(headlines[i].Title)
	}
	
	// Sort headlines by date (most recent first), fallback to title for same dates
	sort.Slice(headlines, func(i, j int) bool {
		dateI, errI := time.Parse("2006-01-02", headlines[i].Date)
		dateJ, errJ := time.Parse("2006-01-02", headlines[j].Date)
		
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
		return headlines[i].Title < headlines[j].Title
	})
	
	// Limit to 10 headlines
	if len(headlines) > 10 {
		headlines = headlines[:10]
	}
	
	return headlines
}

func extractTextFromNode(n *html.Node) string {
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

// isTopicPage checks if a headline is a category/topic page rather than a news article
func isTopicPage(title string) bool {
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
func extractDateFromTitle(title string) string {
	// Common month abbreviations
	monthMap := map[string]string{
		"Jan": "01", "Feb": "02", "Mar": "03", "Apr": "04",
		"May": "05", "Jun": "06", "Jul": "07", "Aug": "08",
		"Sep": "09", "Oct": "10", "Nov": "11", "Dec": "12",
	}
	
	// Regex to match date pattern at end of title (Month DD, YYYY)
	dateRegex := regexp.MustCompile(`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2}),\s+(\d{4})\b`)
	
	matches := dateRegex.FindStringSubmatch(title)
	if len(matches) == 4 {
		month := monthMap[matches[1]]
		day := matches[2]
		year := matches[3]
		
		// Pad day with leading zero if needed
		if len(day) == 1 {
			day = "0" + day
		}
		
		return fmt.Sprintf("%s-%s-%s", year, month, day)
	}
	
	// If no date found, return today's date as fallback
	return time.Now().Format("2006-01-02")
} 