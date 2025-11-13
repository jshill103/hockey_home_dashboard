package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jaredshillingburg/go_uhc/models"
)

// NHL Schedule API response structure for game ID lookup
type nhlScheduleResponse struct {
	GameWeek []struct {
		Date  string `json:"date"`
		Games []struct {
			ID       int    `json:"id"`
			GameDate string `json:"gameDate"`
			HomeTeam struct {
				Abbrev string `json:"abbrev"`
			} `json:"homeTeam"`
			AwayTeam struct {
				Abbrev string `json:"abbrev"`
			} `json:"awayTeam"`
		} `json:"games"`
	} `json:"gameWeek"`
}

// TwitterRefereeCollector collects referee assignments from Twitter/X
// Primary source: @ScoutingTheRefs and other NHL referee Twitter accounts
type TwitterRefereeCollector struct {
	httpClient      *http.Client
	userAgent       string
	refereeService  *RefereeService
	
	// Twitter API credentials (optional - will use web scraping if not available)
	twitterAPIKey    string
	twitterAPISecret string
	BearerToken      string // Exported for status checks
	
	// Accounts to monitor
	twitterHandles   []string
	
	// Last check time to avoid duplicate processing
	lastCheckTime    time.Time
}

// NewTwitterRefereeCollector creates a new Twitter referee collector
func NewTwitterRefereeCollector(refereeService *RefereeService) *TwitterRefereeCollector {
	return &TwitterRefereeCollector{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		refereeService: refereeService,
		twitterHandles: []string{
			"ScoutingTheRefs",  // Primary source
			"NHLOfficials",     // Official NHL referees account (if they post assignments)
			"NHLRefWatcher",    // Backup source
		},
		lastCheckTime: time.Now().Add(-24 * time.Hour), // Start with yesterday
	}
}

// SetTwitterCredentials sets Twitter API credentials for API-based collection
func (trc *TwitterRefereeCollector) SetTwitterCredentials(bearerToken string) {
	trc.BearerToken = bearerToken
}

// ============================================================================
// MAIN COLLECTION METHOD
// ============================================================================

// CollectRefereeAssignments collects referee assignments from Twitter
// Returns number of assignments found and any error
func (trc *TwitterRefereeCollector) CollectRefereeAssignments() (int, error) {
	log.Printf("🐦 Collecting referee assignments from Twitter...")
	
	totalAssignments := 0
	
	// Try different collection methods
	if trc.BearerToken != "" {
		// Method 1: Twitter API v2 (requires bearer token)
		assignments, err := trc.collectViaTwitterAPI()
		if err != nil {
			log.Printf("⚠️ Twitter API collection failed: %v", err)
		} else {
			totalAssignments += len(assignments)
			trc.storeAssignments(assignments)
		}
	}
	
	if totalAssignments == 0 {
		// Method 2: Web scraping (fallback if API unavailable)
		log.Printf("🔄 Falling back to web scraping...")
		assignments, err := trc.collectViaWebScraping()
		if err != nil {
			log.Printf("⚠️ Web scraping failed: %v", err)
			return 0, fmt.Errorf("all collection methods failed")
		}
		totalAssignments += len(assignments)
		trc.storeAssignments(assignments)
	}
	
	log.Printf("✅ Collected %d referee assignments from Twitter", totalAssignments)
	return totalAssignments, nil
}

// ============================================================================
// METHOD 1: TWITTER API V2
// ============================================================================

// collectViaTwitterAPI uses Twitter API v2 to fetch recent tweets
func (trc *TwitterRefereeCollector) collectViaTwitterAPI() ([]models.RefereeGameAssignment, error) {
	var allAssignments []models.RefereeGameAssignment
	
	for _, handle := range trc.twitterHandles {
		log.Printf("📱 Fetching tweets from @%s via API...", handle)
		
		// First, get user ID
		userID, err := trc.getUserID(handle)
		if err != nil {
			log.Printf("⚠️ Could not get user ID for @%s: %v", handle, err)
			continue
		}
		
		// Fetch recent tweets
		tweets, err := trc.getUserTweets(userID)
		if err != nil {
			log.Printf("⚠️ Could not fetch tweets for @%s: %v", handle, err)
			continue
		}
		
		// Parse tweets for referee assignments
		for _, tweet := range tweets {
			assignments := trc.parseTweetForAssignments(tweet)
			allAssignments = append(allAssignments, assignments...)
		}
	}
	
	return allAssignments, nil
}

// getUserID fetches Twitter user ID from handle
func (trc *TwitterRefereeCollector) getUserID(handle string) (string, error) {
	url := fmt.Sprintf("https://api.twitter.com/2/users/by/username/%s", handle)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", trc.BearerToken))
	
	resp, err := trc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	return result.Data.ID, nil
}

// getUserTweets fetches recent tweets from a user
func (trc *TwitterRefereeCollector) getUserTweets(userID string) ([]string, error) {
	// Get tweets from last 24 hours
	startTime := trc.lastCheckTime.Format(time.RFC3339)
	
	url := fmt.Sprintf("https://api.twitter.com/2/users/%s/tweets?max_results=100&start_time=%s&tweet.fields=created_at,text",
		userID, startTime)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", trc.BearerToken))
	
	resp, err := trc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var result struct {
		Data []struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	var tweets []string
	for _, tweet := range result.Data {
		tweets = append(tweets, tweet.Text)
	}
	
	return tweets, nil
}

// ============================================================================
// METHOD 2: WEB SCRAPING
// ============================================================================

// collectViaWebScraping scrapes Twitter web pages for referee assignments
func (trc *TwitterRefereeCollector) collectViaWebScraping() ([]models.RefereeGameAssignment, error) {
	var allAssignments []models.RefereeGameAssignment
	
	for _, handle := range trc.twitterHandles {
		log.Printf("🌐 Scraping tweets from @%s...", handle)
		
		// Use Nitter instance (Twitter frontend alternative) or direct scraping
		assignments, err := trc.scrapeTwitterProfile(handle)
		if err != nil {
			log.Printf("⚠️ Failed to scrape @%s: %v", handle, err)
			continue
		}
		
		allAssignments = append(allAssignments, assignments...)
	}
	
	return allAssignments, nil
}

// scrapeTwitterProfile scrapes a Twitter profile for referee tweets
func (trc *TwitterRefereeCollector) scrapeTwitterProfile(handle string) ([]models.RefereeGameAssignment, error) {
	// Try multiple nitter instances (public Twitter frontends)
	nitterInstances := []string{
		"https://nitter.net",
		"https://nitter.poast.org",
		"https://nitter.privacydev.net",
	}
	
	for _, instance := range nitterInstances {
		url := fmt.Sprintf("%s/%s", instance, handle)
		
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		
		req.Header.Set("User-Agent", trc.userAgent)
		
		resp, err := trc.httpClient.Do(req)
		if err != nil {
			log.Printf("⚠️ Failed to connect to %s: %v", instance, err)
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			log.Printf("⚠️ %s returned status %d", instance, resp.StatusCode)
			continue
		}
		
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		
		// Parse HTML for tweets
		assignments := trc.parseHTMLForAssignments(string(body))
		if len(assignments) > 0 {
			log.Printf("✅ Found %d assignments from %s", len(assignments), instance)
			return assignments, nil
		}
	}
	
	return nil, fmt.Errorf("all nitter instances failed")
}

// parseHTMLForAssignments extracts referee assignments from Twitter HTML
func (trc *TwitterRefereeCollector) parseHTMLForAssignments(html string) []models.RefereeGameAssignment {
	var assignments []models.RefereeGameAssignment
	
	// Find tweet text sections (Nitter uses <div class="tweet-content">)
	tweetRegex := regexp.MustCompile(`<div class="tweet-content[^"]*">(.*?)</div>`)
	matches := tweetRegex.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		// Remove HTML tags
		tweetText := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(match[1], "")
		tweetText = strings.TrimSpace(tweetText)
		
		// Parse for assignments
		parsed := trc.parseTweetForAssignments(tweetText)
		assignments = append(assignments, parsed...)
	}
	
	return assignments
}

// ============================================================================
// TWEET PARSING LOGIC
// ============================================================================

// parseTweetForAssignments extracts referee assignments from tweet text
func (trc *TwitterRefereeCollector) parseTweetForAssignments(tweetText string) []models.RefereeGameAssignment {
	var assignments []models.RefereeGameAssignment
	
	// Common patterns in referee announcement tweets:
	// 1. "Game: Team A @ Team B | Refs: Name1, Name2"
	// 2. "TOR @ MTL: Referee1 & Referee2"
	// 3. "Tonight's officials: TOR/MTL - Name1, Name2"
	// 4. "Refs for UTA vs VGK: Name1 and Name2"
	
	// Pattern 1: Team @ Team format with referees
	pattern1 := regexp.MustCompile(`([A-Z]{2,3})\s*[@vs]+\s*([A-Z]{2,3}).*?(?:Refs?|Officials?)[:|\s]+([A-Z][a-z]+\s+[A-Z][a-z]+)(?:\s*[,&and]+\s*([A-Z][a-z]+\s+[A-Z][a-z]+))?`)
	
	matches := pattern1.FindAllStringSubmatch(tweetText, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		
		awayTeam := match[1]
		homeTeam := match[2]
		ref1 := strings.TrimSpace(match[3])
		ref2 := ""
		if len(match) > 4 && match[4] != "" {
			ref2 = strings.TrimSpace(match[4])
		}
		
		// Try to extract date (look for date patterns in tweet)
		gameDate := trc.extractDateFromTweet(tweetText)
		
		assignment := models.RefereeGameAssignment{
			GameID:      0, // Will be filled in when we match to actual game
			GameDate:    gameDate,
			HomeTeam:    homeTeam,
			AwayTeam:    awayTeam,
			Referee1ID:  0, // Will look up by name
			Referee2ID:  0,
			Source:      "Twitter",
			LastUpdated: time.Now(),
		}
		
		// Try to find referees by name and get their IDs
		if ref1 != "" {
			if ref := trc.refereeService.FindRefereeByName(ref1); ref != nil {
				assignment.Referee1ID = ref.RefereeID
			}
		}
		if ref2 != "" {
			if ref := trc.refereeService.FindRefereeByName(ref2); ref != nil {
				assignment.Referee2ID = ref.RefereeID
			}
		}
		
		assignments = append(assignments, assignment)
		
		log.Printf("📋 Parsed: %s @ %s | Refs: %s, %s", awayTeam, homeTeam, ref1, ref2)
	}
	
	return assignments
}

// extractDateFromTweet tries to extract game date from tweet text
func (trc *TwitterRefereeCollector) extractDateFromTweet(tweetText string) time.Time {
	// Look for common date patterns
	// "Tonight", "Today", "Tomorrow", "Nov 13", "11/13", etc.
	
	lowerText := strings.ToLower(tweetText)
	now := time.Now()
	
	if strings.Contains(lowerText, "tonight") || strings.Contains(lowerText, "today") {
		return now
	}
	
	if strings.Contains(lowerText, "tomorrow") {
		return now.Add(24 * time.Hour)
	}
	
	// Try to parse explicit dates
	// Format: "Nov 13" or "November 13"
	monthPattern := regexp.MustCompile(`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+(\d{1,2})`)
	if match := monthPattern.FindStringSubmatch(tweetText); len(match) >= 3 {
		monthMap := map[string]time.Month{
			"Jan": time.January, "Feb": time.February, "Mar": time.March,
			"Apr": time.April, "May": time.May, "Jun": time.June,
			"Jul": time.July, "Aug": time.August, "Sep": time.September,
			"Oct": time.October, "Nov": time.November, "Dec": time.December,
		}
		
		month := monthMap[match[1]]
		day, _ := strconv.Atoi(match[2])
		
		year := now.Year()
		// If the month has already passed this year, it's for next year
		if month < now.Month() {
			year++
		}
		
		return time.Date(year, month, day, 19, 0, 0, 0, time.Local) // Default to 7 PM game time
	}
	
	// Default to today if we can't parse
	return now
}

// ============================================================================
// GAME ID LOOKUP
// ============================================================================

// lookupGameID fetches the real NHL game ID from the schedule API
// based on date, home team, and away team
func (trc *TwitterRefereeCollector) lookupGameID(gameDate time.Time, homeTeam, awayTeam string) (int, error) {
	// Format date for NHL API (YYYY-MM-DD)
	dateStr := gameDate.Format("2006-01-02")
	
	// Fetch NHL schedule for this date
	url := fmt.Sprintf("https://api-web.nhle.com/v1/schedule/%s", dateStr)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create schedule request: %w", err)
	}
	req.Header.Set("User-Agent", trc.userAgent)
	
	resp, err := trc.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch schedule: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("schedule API returned status %d", resp.StatusCode)
	}
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read schedule response: %w", err)
	}
	
	var scheduleData nhlScheduleResponse
	if err := json.Unmarshal(body, &scheduleData); err != nil {
		return 0, fmt.Errorf("failed to parse schedule: %w", err)
	}
	
	// Search for matching game (home team + away team)
	for _, week := range scheduleData.GameWeek {
		for _, game := range week.Games {
			// Match by team codes (case-insensitive)
			if strings.EqualFold(game.HomeTeam.Abbrev, homeTeam) && 
			   strings.EqualFold(game.AwayTeam.Abbrev, awayTeam) {
				log.Printf("✅ Matched game: %s @ %s = Game ID %d", awayTeam, homeTeam, game.ID)
				return game.ID, nil
			}
		}
	}
	
	return 0, fmt.Errorf("no game found for %s @ %s on %s", awayTeam, homeTeam, dateStr)
}

// ============================================================================
// STORAGE
// ============================================================================

// storeAssignments stores referee assignments in the database
func (trc *TwitterRefereeCollector) storeAssignments(assignments []models.RefereeGameAssignment) {
	for _, assignment := range assignments {
		// Try to match to actual game ID from NHL API
		if assignment.GameID == 0 {
			gameID, err := trc.lookupGameID(assignment.GameDate, assignment.HomeTeam, assignment.AwayTeam)
			if err != nil {
				log.Printf("⚠️ Failed to lookup game ID for %s @ %s: %v", 
					assignment.AwayTeam, assignment.HomeTeam, err)
				// Skip this assignment if we can't find the real game ID
				continue
			}
			assignment.GameID = gameID
		}
		
		if err := trc.refereeService.AddGameAssignment(&assignment); err != nil {
			log.Printf("⚠️ Failed to store assignment: %v", err)
		} else {
			log.Printf("💾 Stored assignment: Game %d | Refs: %d, %d", 
				assignment.GameID, assignment.Referee1ID, assignment.Referee2ID)
		}
	}
	
	// Update last check time
	trc.lastCheckTime = time.Now()
}

// ============================================================================
// AUTOMATED COLLECTION
// ============================================================================

// StartAutomatedCollection starts automated collection every N hours
func (trc *TwitterRefereeCollector) StartAutomatedCollection(intervalHours int) {
	log.Printf("🤖 Starting automated Twitter referee collection (every %d hours)", intervalHours)
	
	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	go func() {
		for range ticker.C {
			count, err := trc.CollectRefereeAssignments()
			if err != nil {
				log.Printf("❌ Automated collection failed: %v", err)
			} else {
				log.Printf("✅ Automated collection complete: %d assignments", count)
			}
		}
	}()
}

