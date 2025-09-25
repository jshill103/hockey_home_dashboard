# 🚀 NHL Team Dashboard Scraping System - Execution Guide

This guide shows you exactly how to execute and test the new multi-behavior web scraping system that has been integrated into your NHL Team Dashboard.

## ✅ Prerequisites

Ensure you have:
- Go 1.23.3+ installed
- Your existing NHL Team Dashboard project
- Internet connection for scraping external websites

## 🏗️ What Has Been Added

The scraping system has been **automatically integrated** into your existing application:

### New Files Created:
```
📁 models/
├── scraper.go           # Core scraper interfaces
└── scraped_items.go     # Item implementations (News, Products, Generic)

📁 services/
├── scraper_base.go      # Common scraper functionality
├── scraper_manager.go   # Multi-scraper orchestration
├── scraper_service.go   # High-level service interface
├── scraper_actions.go   # Built-in action implementations
├── news_scraper_v2.go   # NHL news scraper (refactored)
└── fanatics_scraper.go  # Fanatics merchandise scraper

📁 handlers/
└── scraper_handlers.go  # HTTP endpoints for scraper management

📁 examples/
└── scraper_integration_example.go

📁 Root/
├── test_scraper_system.go        # Testing script
├── SCRAPING_SYSTEM_README.md     # Detailed documentation
└── EXECUTION_GUIDE.md            # This file
```

### Modified Files:
- **`main.go`**: Integrated scraper service initialization and HTTP endpoints

### New Endpoints Added:
- `/scraper-dashboard` - Web-based management dashboard
- `/api/scrapers/status` - Scraper status information
- `/api/scrapers/products` - Latest Fanatics products
- `/api/scrapers/news` - Latest NHL news (v2 scraper)
- `/api/scrapers/run` - Manually run all scrapers
- `/api/scrapers/run/news` - Run news scraper only
- `/api/scrapers/run/fanatics` - Run Fanatics scraper only
- `/api/scrapers/changes` - Recent change detections

## 🚀 Step-by-Step Execution

### Step 1: Build the Application

```bash
# Build the web server with new scraping system
go build -o web_server main.go
```

### Step 2: Start the NHL Team Dashboard

```bash
# Start with Utah Hockey Club (or your preferred team)
./web_server -team UTA
```

**Expected output:**
```
Starting NHL Web Application for Utah Hockey Club (UTA)...
Initializing scraping system...
✅ Scraping system initialized for UTA
   - NHL News Scraper: https://www.nhl.com/news
   - Fanatics UTA Products Scraper
   - Change detection and automated actions enabled
🕷️  Automatic scraping started for UTA
Server starting on http://localhost:8080
🕷️  Scraping system active - Dashboard: http://localhost:8080/scraper-dashboard
   - NHL News: Every 10 minutes
   - UTA Products: Every 30 minutes
   - Change logs: ./scraper_data/logs/
```

### Step 3: Access the Scraper Dashboard

Open your browser and navigate to:
**http://localhost:8080/scraper-dashboard**

You'll see a management interface showing:
- 📊 Scraper status and configuration
- ▶️ Manual scraper execution buttons
- 📦 Quick data viewing options
- 📝 Log file locations

### Step 4: Test the Scraping System

In a **new terminal window**, run the test script:

```bash
# Run the automated test
go run test_scraper_system.go
```

This will:
1. ✅ Check scraper status
2. 🚀 Run all scrapers manually
3. 📦 Fetch latest products from Fanatics
4. 📰 Fetch latest NHL news
5. 🔄 Check for recent changes
6. 🎯 Test individual scraper runs

### Step 5: Monitor Real-Time Activity

**While the system is running**, you can:

#### Check API Endpoints:
```bash
# Check scraper status
curl http://localhost:8080/api/scrapers/status

# Get latest products
curl http://localhost:8080/api/scrapers/products?limit=5

# Get latest news
curl http://localhost:8080/api/scrapers/news?limit=5

# Manually run scrapers
curl -X POST http://localhost:8080/api/scrapers/run
```

#### Monitor Log Files:
```bash
# Create log directory (if not exists)
mkdir -p ./scraper_data/logs

# Watch all changes
tail -f ./scraper_data/logs/scraper_changes.log

# Watch new product alerts
tail -f ./scraper_data/logs/UTA_new_products.log

# Watch price changes
tail -f ./scraper_data/logs/UTA_price_changes.log

# Watch NHL news
tail -f ./scraper_data/logs/nhl_news.log
```

#### View Stored Data:
```bash
# View latest scraped products
cat ./scraper_data/fanatics_uta_items.json | head -20

# View latest news items
cat ./scraper_data/nhl_news_items.json | head -20

# View scraper execution results
cat ./scraper_data/fanatics_uta_results.json
cat ./scraper_data/nhl_news_results.json
```

## 🎯 What to Expect

### First Run:
When you first start the system:

1. **NHL News Scraper** will immediately fetch ~10 latest headlines from NHL.com
2. **Fanatics Scraper** will fetch products from your team's Fanatics page
3. **Change Detection** will be empty (no previous data to compare)
4. **Action Logs** will show initial scraping activity

### Ongoing Operation:
As the system runs automatically:

1. **Every 10 minutes**: NHL news scraper runs, detects new articles
2. **Every 30 minutes**: Fanatics scraper runs, detects product changes
3. **Change Alerts** appear in logs when:
   - 📰 New NHL news articles are published
   - 🛍️ New team products are added to Fanatics
   - 💰 Product prices change
   - 📦 Products go in/out of stock

### Example Change Alerts:
```
🏒 NEW UTA PRODUCT ALERT! 🏒
Product: Utah Hockey Club Authentic Jersey
Price: 199.99 USD
Available: true
URL: https://www.fanatics.com/nhl/utah-hockey-club/...
Discovered: 2025-09-01 15:30:45
---

📉 PRICE CHANGE ALERT - UTA 📉
Product: Utah Hockey Club Hat
Previous Price: 29.99 USD
New Price: 24.99 USD
Change: -5.00
URL: https://www.fanatics.com/nhl/utah-hockey-club/...
Updated: 2025-09-01 16:15:23
---

📰 NEW NHL NEWS 📰
Headline: Utah Hockey Club Signs New Prospect
Date: 2025-09-01
URL: https://www.nhl.com/news/utah-signs-prospect
Discovered: 2025-09-01 14:45:12
---
```

## 🔧 Customization

### Change Team:
```bash
# Scrape different team's products
./web_server -team COL  # Colorado Avalanche
./web_server -team TOR  # Toronto Maple Leafs
./web_server -team BOS  # Boston Bruins
```

### Modify Intervals:
Edit scraper configurations in the initialization code to change how often scrapers run.

### Add Custom Scrapers:
Use the framework to add scrapers for:
- Player statistics sites
- Social media monitoring
- Trade rumor aggregation
- Injury report tracking

### Add Custom Actions:
Create actions for:
- Discord/Slack notifications
- Email alerts
- Database updates
- Social media posting

## 🎮 Interactive Testing

### Dashboard Features:
1. **🚀 Run All Scrapers**: Execute all scrapers immediately
2. **📦 View Products**: See latest product data in popup
3. **📰 View News**: See latest news data in popup
4. **🔄 Refresh Status**: Update scraper status display

### API Testing:
```bash
# Get comprehensive status
curl -s http://localhost:8080/api/scrapers/status | jq

# Run specific scraper
curl -X POST http://localhost:8080/api/scrapers/run/fanatics

# Get recent changes
curl -s http://localhost:8080/api/scrapers/changes | jq
```

## 🚨 Troubleshooting

### Common Issues:

1. **Scraper service not initialized**:
   - Check console output for error messages
   - Ensure internet connectivity
   - Verify Go dependencies are installed

2. **No products found**:
   - Fanatics website structure may have changed
   - Check error logs for HTTP errors
   - Try running scraper manually via dashboard

3. **Change detection not working**:
   - First run will have no changes (baseline)
   - Check that `ChangeDetection: true` in config
   - Verify data persistence is working

4. **Port conflicts**:
   - Ensure port 8080 is not in use by other applications
   - Kill existing processes: `pkill -f web_server`

### Debug Mode:
Enable detailed logging by checking the console output and log files for specific error messages.

## ✨ Success Indicators

Your scraping system is working correctly when you see:

1. ✅ **Console Output**: Scraper initialization messages
2. 🌐 **Dashboard Access**: http://localhost:8080/scraper-dashboard loads
3. 📊 **API Responses**: All endpoints return JSON data
4. 📁 **Log Files**: Created in `./scraper_data/logs/`
5. 🔄 **Change Detection**: Alerts appear on subsequent runs
6. 📦 **Data Files**: JSON files created in `./scraper_data/`

## 🎉 Next Steps

Once the system is running:

1. **Monitor Performance**: Watch logs for scraping efficiency
2. **Customize Actions**: Add Discord/Slack notifications
3. **Extend Scrapers**: Add new content sources
4. **Analyze Data**: Build reports from scraped data
5. **Scale Up**: Add more teams or scraper types

Congratulations! Your NHL Team Dashboard is now powered by an advanced, automated web scraping system! 🏒🕷️
