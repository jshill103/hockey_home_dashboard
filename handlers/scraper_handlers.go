package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jaredshillingburg/go_uhc/services"
)

// Global reference to scraper service (will be set by main.go)
var globalScraperService *services.ScraperService

// InitScraperHandlers initializes the scraper handlers with the scraper service
func InitScraperHandlers(service *services.ScraperService) {
	globalScraperService = service
}

// HandleScraperStatus returns the status of all scrapers
func HandleScraperStatus(w http.ResponseWriter, r *http.Request) {
	if globalScraperService == nil {
		http.Error(w, "Scraper service not initialized", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	status := globalScraperService.GetScraperStatus()

	response := map[string]interface{}{
		"status":   "success",
		"scrapers": status,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleLatestProducts returns the latest products from Fanatics
func HandleLatestProducts(w http.ResponseWriter, r *http.Request) {
	if globalScraperService == nil {
		http.Error(w, "Scraper service not initialized", http.StatusServiceUnavailable)
		return
	}

	// Get limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	products, err := globalScraperService.GetLatestProducts(limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching products: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":   "success",
		"count":    len(products),
		"products": products,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleLatestNews returns the latest NHL news (v2 scraper)
func HandleLatestNewsV2(w http.ResponseWriter, r *http.Request) {
	if globalScraperService == nil {
		http.Error(w, "Scraper service not initialized", http.StatusServiceUnavailable)
		return
	}

	// Get limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	news, err := globalScraperService.GetLatestNews(limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching news: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status": "success",
		"count":  len(news),
		"news":   news,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleRunScrapers manually runs all scrapers once
func HandleRunScrapers(w http.ResponseWriter, r *http.Request) {
	if globalScraperService == nil {
		http.Error(w, "Scraper service not initialized", http.StatusServiceUnavailable)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	results, err := globalScraperService.RunAllScrapersOnce()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error running scrapers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":  "success",
		"message": "All scrapers executed successfully",
		"results": results,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleRunNewsScraperOnce manually runs the news scraper once
func HandleRunNewsScraperOnce(w http.ResponseWriter, r *http.Request) {
	if globalScraperService == nil {
		http.Error(w, "Scraper service not initialized", http.StatusServiceUnavailable)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := globalScraperService.RunNewsScraperOnce()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error running news scraper: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":  "success",
		"message": "News scraper executed successfully",
		"result":  result,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleRunFanaticsScraperOnce manually runs the Fanatics scraper once
func HandleRunFanaticsScraperOnce(w http.ResponseWriter, r *http.Request) {
	if globalScraperService == nil {
		http.Error(w, "Scraper service not initialized", http.StatusServiceUnavailable)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := globalScraperService.RunFanaticsScraperOnce()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error running Fanatics scraper: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":  "success",
		"message": "Fanatics scraper executed successfully",
		"result":  result,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleRunMammothScraperOnce manually runs the Mammoth team store scraper once
func HandleRunMammothScraperOnce(w http.ResponseWriter, r *http.Request) {
	if globalScraperService == nil {
		http.Error(w, "Scraper service not initialized", http.StatusServiceUnavailable)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := globalScraperService.RunMammothScraperOnce()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error running Mammoth scraper: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":  "success",
		"message": "Mammoth team store scraper executed successfully",
		"result":  result,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleRecentChanges returns recent changes detected by scrapers
func HandleRecentChanges(w http.ResponseWriter, r *http.Request) {
	if globalScraperService == nil {
		http.Error(w, "Scraper service not initialized", http.StatusServiceUnavailable)
		return
	}

	// Get limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	changes, err := globalScraperService.GetRecentChanges(limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching recent changes: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":  "success",
		"count":   len(changes),
		"changes": changes,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleScraperDashboard returns an HTML dashboard for scraper management
func HandleScraperDashboard(w http.ResponseWriter, r *http.Request) {
	if globalScraperService == nil {
		http.Error(w, "Scraper service not initialized", http.StatusServiceUnavailable)
		return
	}

	status := globalScraperService.GetScraperStatus()

	html := `
<!DOCTYPE html>
<html>
<head>
    <title>NHL Scraper Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #1a1a1a; color: white; }
        .container { max-width: 1200px; margin: 0 auto; }
        .scraper { background: #2d2d2d; padding: 20px; margin: 10px 0; border-radius: 8px; }
        .scraper h3 { margin-top: 0; color: #4CAF50; }
        .status { display: inline-block; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
        .enabled { background: #4CAF50; }
        .disabled { background: #f44336; }
        button { background: #2196F3; color: white; border: none; padding: 8px 16px; cursor: pointer; border-radius: 4px; margin: 4px; }
        button:hover { background: #1976D2; }
        .logs { background: #333; padding: 10px; border-radius: 4px; font-family: monospace; font-size: 12px; margin: 10px 0; }
    </style>
    <script>
        function runScraper(type) {
            let url = '/api/scrapers/run';
            if (type === 'news') url = '/api/scrapers/run/news';
            if (type === 'fanatics') url = '/api/scrapers/run/fanatics';
            
            fetch(url, { method: 'POST' })
                .then(response => response.json())
                .then(data => {
                    alert('Scraper executed: ' + JSON.stringify(data.message || data.status));
                    location.reload();
                })
                .catch(error => alert('Error: ' + error));
        }

        function viewData(type) {
            let url = '/api/scrapers/products?limit=10';
            if (type === 'news') url = '/api/scrapers/news?limit=10';
            
            fetch(url)
                .then(response => response.json())
                .then(data => {
                    const popup = window.open('', '_blank', 'width=800,height=600');
                    popup.document.write('<pre>' + JSON.stringify(data, null, 2) + '</pre>');
                })
                .catch(error => alert('Error: ' + error));
        }
    </script>
</head>
<body>
    <div class="container">
        <h1>🕷️ NHL Scraper Dashboard</h1>
        <p>Manage and monitor your web scrapers</p>
        
        <div style="margin: 20px 0;">
            <button onclick="runScraper('all')">🚀 Run All Scrapers</button>
            <button onclick="location.reload()">🔄 Refresh Status</button>
            <button onclick="viewData('products')">📦 View Products</button>
            <button onclick="viewData('news')">📰 View News</button>
        </div>
`

	// Add scraper status information
	for scraperID, scraperStatus := range status {
		statusMap := scraperStatus.(map[string]interface{})
		enabled := statusMap["enabled"].(bool)

		statusClass := "disabled"
		statusText := "Disabled"
		if enabled {
			statusClass = "enabled"
			statusText = "Enabled"
		}

		html += fmt.Sprintf(`
        <div class="scraper">
            <h3>%s <span class="status %s">%s</span></h3>
            <p><strong>ID:</strong> %s</p>
            <p><strong>URL:</strong> %s</p>
            <p><strong>Interval:</strong> %s</p>
            <p><strong>Team:</strong> %s</p>
            <div>
                <button onclick="runScraper('%s')">▶️ Run Now</button>
            </div>
        </div>
		`, statusMap["name"], statusClass, statusText, statusMap["id"], statusMap["url"], statusMap["interval"], statusMap["team_code"], scraperID)
	}

	html += `
        <div class="logs">
            <h3>📝 Recent Activity</h3>
            <p>Check ./scraper_data/logs/ for detailed scraper logs and change alerts.</p>
            <p>Log files include:</p>
            <ul>
                <li>scraper_changes.log - All detected changes</li>
                <li>nhl_news.log - NHL news discoveries</li>
                <li>` + fmt.Sprintf("%s_new_products.log", globalScraperService.GetAllScrapers()[0].GetConfig().TeamCode) + ` - New product alerts</li>
                <li>` + fmt.Sprintf("%s_price_changes.log", globalScraperService.GetAllScrapers()[0].GetConfig().TeamCode) + ` - Price change alerts</li>
            </ul>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
