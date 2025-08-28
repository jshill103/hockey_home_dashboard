package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jaredshillingburg/go_uhc/models"
	"github.com/jaredshillingburg/go_uhc/services"
)

// HandleNews handles news requests
func HandleNews(w http.ResponseWriter, r *http.Request) {
	// Use cached news data
	headlines := *cachedNews

	// If no cached data, try to fetch fresh data as fallback
	if len(headlines) == 0 {
		fmt.Println("No cached news data, fetching fresh data...")
		var err error
		headlines, err = services.ScrapeNHLNews()
		if err != nil {
			w.Write([]byte("<p>Error fetching news: " + err.Error() + "</p>"))
			return
		}
		// Update cache
		*cachedNews = headlines
	}

	if len(headlines) == 0 {
		w.Write([]byte("<p>No news headlines available at the moment.</p>"))
		return
	}

	html := formatNewsHTML(headlines)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func formatNewsHTML(headlines []models.NewsHeadline) string {
	var html strings.Builder

	if len(headlines) == 0 {
		html.WriteString("<p>No news headlines available at the moment.</p>")
		return html.String()
	}

	// Display each news headline as a properly formatted news item
	for _, headline := range headlines {
		html.WriteString("<div class='news-item'>")
		
		// Title with link
		if headline.URL != "" {
			html.WriteString(fmt.Sprintf("<a href='%s' target='_blank'>%s</a>", 
				headline.URL, headline.Title))
		} else {
			html.WriteString(fmt.Sprintf("<a>%s</a>", headline.Title))
		}
		
		html.WriteString("</div>")
	}

	return html.String()
} 