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
	"golang.org/x/net/html"
)

// RefereeScraper handles web scraping of referee data from various sources
type RefereeScraper struct {
	httpClient    *http.Client
	refereeService *RefereeService
	LastScrape     time.Time // Exported for handler access
	userAgent     string
}

// NewRefereeScraper creates a new referee scraper
func NewRefereeScraper(refereeService *RefereeService) *RefereeScraper {
	return &RefereeScraper{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		refereeService: refereeService,
		userAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
	}
}

// ============================================================================
// SCOUTING THE REFS - REFEREE STATISTICS
// ============================================================================

// ScrapeRefereeStats fetches referee statistics from Scouting The Refs
func (rs *RefereeScraper) ScrapeRefereeStats(season string) error {
	log.Printf("🔍 Scraping referee stats for %s season from Scouting The Refs...", season)
	
	url := fmt.Sprintf("https://scoutingtherefs.com/%s-nhl-referee-stats/", season)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", rs.userAgent)
	
	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// Parse HTML and extract referee data
	referees, err := rs.parseRefereeStatsHTML(string(body), season)
	if err != nil {
		return fmt.Errorf("failed to parse referee stats: %w", err)
	}
	
	log.Printf("✅ Scraped %d referees from Scouting The Refs", len(referees))
	
	// Store each referee and their stats
	for _, ref := range referees {
		// Add referee
		if err := rs.refereeService.AddReferee(&ref.Referee); err != nil {
			log.Printf("⚠️ Failed to add referee %s: %v", ref.Referee.FullName, err)
			continue
		}
		
		// Add stats
		if err := rs.refereeService.UpdateRefereeStats(&ref.Stats); err != nil {
			log.Printf("⚠️ Failed to add stats for referee %s: %v", ref.Referee.FullName, err)
		}
	}
	
	rs.LastScrape = time.Now()
	return nil
}

// RefereeWithStats combines referee and stats for parsing
type RefereeWithStats struct {
	Referee models.Referee
	Stats   models.RefereeSeasonStats
}

// parseRefereeStatsHTML parses the HTML table from Scouting The Refs
func (rs *RefereeScraper) parseRefereeStatsHTML(htmlContent, season string) ([]RefereeWithStats, error) {
	var referees []RefereeWithStats
	
	// Find the tablepress table specifically - match any ID that starts with tablepress
	// Use a more flexible regex that matches either id="tablepress-XXX" or class="tablepress..."
	tableRegex := regexp.MustCompile(`(?s)<table[^>]*(?:id|class)=["\']?[^"\']*tablepress[^"\']*["\']?[^>]*>(.*?)</table>`)
	tableMatches := tableRegex.FindStringSubmatch(htmlContent)
	
	if len(tableMatches) < 2 {
		log.Printf("⚠️ Could not find tablepress table in HTML")
		return referees, nil
	}
	
	tableContent := tableMatches[1]
	
	// Find tbody content - use (?s) flag for multiline matching
	tbodyRegex := regexp.MustCompile(`(?si)<tbody[^>]*>(.*?)</tbody>`)
	tbodyMatches := tbodyRegex.FindStringSubmatch(tableContent)
	
	if len(tbodyMatches) < 2 {
		log.Printf("⚠️ Could not find tbody in table")
		return referees, nil
	}
	
	tbodyContent := tbodyMatches[1]
	
	// Find all table rows in tbody - use (?s) for multiline matching
	rowRegex := regexp.MustCompile(`(?si)<tr[^>]*class="row-\d+"[^>]*>(.*?)</tr>`)
	rows := rowRegex.FindAllStringSubmatch(tbodyContent, -1)
	
	log.Printf("Found %d table rows", len(rows))
	
	for i, row := range rows {
		// Extract cells with column classes - use (?s) for multiline matching
		cellRegex := regexp.MustCompile(`(?si)<td[^>]*class="column-(\d+)"[^>]*>(.*?)</td>`)
		cells := cellRegex.FindAllStringSubmatch(row[1], -1)
		
		if len(cells) < 3 {
			continue
		}
		
		// Parse referee data
		ref, stats, err := rs.parseRefereeRow(cells, season)
		if err != nil {
			continue
		}
		
		referees = append(referees, RefereeWithStats{
			Referee: *ref,
			Stats:   *stats,
		})
	}
	
	return referees, nil
}

// parseRefereeRow parses a single table row into Referee and Stats
func (rs *RefereeScraper) parseRefereeRow(cells [][]string, season string) (*models.Referee, *models.RefereeSeasonStats, error) {
	if len(cells) < 3 {
		return nil, nil, fmt.Errorf("insufficient cells: %d", len(cells))
	}
	
	// Create a map of column number to content
	cellMap := make(map[string]string)
	for _, cell := range cells {
		if len(cell) >= 3 {
			columnNum := cell[1]    // column number
			content := cell[2]      // content
			cellMap[columnNum] = content
		}
	}
	
	// Clean HTML tags from cell content
	cleanContent := func(cell string) string {
		// Remove HTML tags and line breaks
		cleaned := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(cell, "")
		cleaned = strings.ReplaceAll(cleaned, "\n", " ")
		cleaned = strings.ReplaceAll(cleaned, "\t", " ")
		// Remove extra spaces
		cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
		return strings.TrimSpace(cleaned)
	}
	
	getFloat := func(columnNum string) float64 {
		cleaned := cleanContent(cellMap[columnNum])
		val, _ := strconv.ParseFloat(cleaned, 64)
		return val
	}
	
	getInt := func(columnNum string) int {
		cleaned := cleanContent(cellMap[columnNum])
		val, _ := strconv.Atoi(cleaned)
		return val
	}
	
	// Column mapping from the table:
	// column-1: Name
	// column-2: Jersey #
	// column-3: Games
	// column-7: Penalties/gm
	
	fullName := cleanContent(cellMap["1"])
	if fullName == "" {
		return nil, nil, fmt.Errorf("no name found")
	}
	
	// Remove asterisks and special characters from names
	fullName = strings.ReplaceAll(fullName, "*", "")
	fullName = strings.TrimSpace(fullName)
	
	// Parse name into first/last (format: "LastName, FirstName")
	nameParts := strings.Split(fullName, ",")
	firstName := ""
	lastName := fullName
	if len(nameParts) >= 2 {
		lastName = strings.TrimSpace(nameParts[0])
		firstName = strings.TrimSpace(nameParts[1])
	} else {
		// Try space-separated
		parts := strings.Fields(fullName)
		if len(parts) >= 2 {
			firstName = parts[0]
			lastName = strings.Join(parts[1:], " ")
		}
	}
	
	jerseyNum := getInt("2")
	gamesOfficiated := getInt("3")
	avgPenaltiesPerGame := getFloat("7")
	
	// Generate a referee ID based on name (simple hash)
	refereeID := rs.generateRefereeID(fullName)
	
	// Parse season int properly (e.g., "2024-25" -> 20242025)
	seasonInt := rs.parseSeasonToInt(season)
	
	referee := &models.Referee{
		RefereeID:        refereeID,
		FirstName:        firstName,
		LastName:         lastName,
		FullName:         fullName,
		JerseyNumber:     jerseyNum,
		Active:           true,
		SeasonGames:      gamesOfficiated,
		CareerGames:      gamesOfficiated, // We don't have career data yet
		LastUpdated:      time.Now(),
		ExternalSourceID: fmt.Sprintf("STR_%s", strings.ReplaceAll(fullName, " ", "_")),
	}
	
	// Calculate total penalties from games * avg
	totalPenalties := int(float64(gamesOfficiated) * avgPenaltiesPerGame)
	
	stats := &models.RefereeSeasonStats{
		RefereeID:           refereeID,
		Season:              seasonInt,
		GamesOfficiated:     gamesOfficiated,
		TotalPenalties:      totalPenalties,
		AvgPenaltiesPerGame: avgPenaltiesPerGame,
		LastUpdated:         time.Now(),
	}
	
	return referee, stats, nil
}

// generateRefereeID creates a simple ID from referee name

// ============================================================================
// NHL.COM - GAME ASSIGNMENTS
// ============================================================================

// ScrapeGameAssignments fetches today's referee assignments from NHL sources
func (rs *RefereeScraper) ScrapeGameAssignments(date time.Time) error {
	log.Printf("🔍 Fetching referee assignments for %s...", date.Format("2006-01-02"))
	
	// Try multiple sources
	assignments, err := rs.tryScrapingNHLAssignments(date)
	if err != nil {
		log.Printf("⚠️ NHL.com scraping failed, trying alternative sources: %v", err)
		// Could try other sources here
		return err
	}
	
	// Store assignments
	for _, assignment := range assignments {
		if err := rs.refereeService.AddGameAssignment(&assignment); err != nil {
			log.Printf("⚠️ Failed to add assignment for game %d: %v", assignment.GameID, err)
		}
	}
	
	log.Printf("✅ Stored %d referee assignments", len(assignments))
	return nil
}

// tryScrapingNHLAssignments attempts to scrape from NHL.com or similar sources
func (rs *RefereeScraper) tryScrapingNHLAssignments(date time.Time) ([]models.RefereeGameAssignment, error) {
	// Try to fetch game data from NHL API and extract referee info
	assignments, err := rs.FetchRefereesFromNHLAPI(date)
	if err != nil {
		log.Printf("⚠️ Failed to fetch from NHL API: %v", err)
		return []models.RefereeGameAssignment{}, nil
	}
	
	if len(assignments) > 0 {
		log.Printf("✅ Found %d referee assignments from NHL API", len(assignments))
		return assignments, nil
	}
	
	log.Printf("ℹ️ No referee assignments found in NHL API for %s", date.Format("2006-01-02"))
	return []models.RefereeGameAssignment{}, nil
}

// FetchRefereesFromNHLAPI fetches referee data from NHL API for completed games
func (rs *RefereeScraper) FetchRefereesFromNHLAPI(date time.Time) ([]models.RefereeGameAssignment, error) {
	// NHL API provides game details including officials for completed games
	// Format: https://api-web.nhle.com/v1/schedule/[YYYY-MM-DD]
	
	dateStr := date.Format("2006-01-02")
	url := fmt.Sprintf("https://api-web.nhle.com/v1/schedule/%s", dateStr)
	
	log.Printf("🔍 Fetching NHL schedule for %s...", dateStr)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", rs.userAgent)
	
	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NHL API: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var scheduleData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&scheduleData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return rs.extractRefereesFromSchedule(scheduleData, date)
}

// extractRefereesFromSchedule extracts referee assignments from NHL schedule data
func (rs *RefereeScraper) extractRefereesFromSchedule(data map[string]interface{}, date time.Time) ([]models.RefereeGameAssignment, error) {
	var assignments []models.RefereeGameAssignment
	
	gameWeek, ok := data["gameWeek"].([]interface{})
	if !ok {
		return assignments, nil
	}
	
	for _, dayData := range gameWeek {
		day, ok := dayData.(map[string]interface{})
		if !ok {
			continue
		}
		
		games, ok := day["games"].([]interface{})
		if !ok {
			continue
		}
		
		for _, gameData := range games {
			game, ok := gameData.(map[string]interface{})
			if !ok {
				continue
			}
			
			// For each game, we need to fetch detailed game data
			gameID, ok := game["id"].(float64)
			if !ok {
				continue
			}
			
			assignment := rs.fetchGameOfficials(int(gameID), date)
			if assignment != nil {
				assignments = append(assignments, *assignment)
			}
		}
	}
	
	return assignments, nil
}

// fetchGameOfficials fetches official data for a specific game
func (rs *RefereeScraper) fetchGameOfficials(gameID int, date time.Time) *models.RefereeGameAssignment {
	// NHL game detail endpoint includes officials
	// Format: https://api-web.nhle.com/v1/gamecenter/[GAMEID]/boxscore
	
	url := fmt.Sprintf("https://api-web.nhle.com/v1/gamecenter/%d/boxscore", gameID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", rs.userAgent)
	
	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	
	var gameData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&gameData); err != nil {
		return nil
	}
	
	// Extract home/away teams
	homeTeam := ""
	awayTeam := ""
	if home, ok := gameData["homeTeam"].(map[string]interface{}); ok {
		if abbrev, ok := home["abbrev"].(string); ok {
			homeTeam = abbrev
		}
	}
	if away, ok := gameData["awayTeam"].(map[string]interface{}); ok {
		if abbrev, ok := away["abbrev"].(string); ok {
			awayTeam = abbrev
		}
	}
	
	// Extract officials
	officials, ok := gameData["gameInfo"].(map[string]interface{})
	if !ok {
		return nil
	}
	
	refs, ok := officials["referees"].([]interface{})
	if !ok || len(refs) == 0 {
		return nil
	}
	
	assignment := &models.RefereeGameAssignment{
		GameID:      gameID,
		GameDate:    date,
		HomeTeam:    homeTeam,
		AwayTeam:    awayTeam,
		Source:      "NHL_API",
		LastUpdated: time.Now(),
	}
	
	// Parse referee data
	if len(refs) > 0 {
		if ref1, ok := refs[0].(map[string]interface{}); ok {
			if name, ok := ref1["default"].(string); ok {
				assignment.Referee1Name = name
				assignment.Referee1ID = rs.findRefereeID(name)
			}
		}
	}
	if len(refs) > 1 {
		if ref2, ok := refs[1].(map[string]interface{}); ok {
			if name, ok := ref2["default"].(string); ok {
				assignment.Referee2Name = name
				assignment.Referee2ID = rs.findRefereeID(name)
			}
		}
	}
	
	log.Printf("✅ Found officials for game %d: %s @ %s (Refs: %s, %s)",
		gameID, awayTeam, homeTeam, assignment.Referee1Name, assignment.Referee2Name)
	
	return assignment
}

// generateRefereeID creates a new referee ID and adds basic info to database
func (rs *RefereeScraper) generateRefereeID(name string) int {
	if name == "" {
		return 0
	}
	
	// Check if referee already exists (thread-safe check)
	rs.refereeService.mutex.RLock()
	for id, ref := range rs.refereeService.referees {
		if ref.FullName == name {
			rs.refereeService.mutex.RUnlock()
			return id
		}
	}
	rs.refereeService.mutex.RUnlock()
	
	// Generate new ID based on name hash
	hash := 0
	for _, char := range name {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	newID := 90000 + (hash % 10000) // Use range 90000-99999 for auto-generated IDs
	
	// Create basic referee record
	nameParts := strings.Split(name, " ")
	referee := &models.Referee{
		RefereeID:        newID,
		FullName:         name,
		FirstName:        nameParts[0],
		LastName:         nameParts[len(nameParts)-1],
		JerseyNumber:     0,
		Active:           true,
		CareerGames:      0,
		SeasonGames:      0,
		LastUpdated:      time.Now(),
		ExternalSourceID: fmt.Sprintf("AUTO_%s", strings.ReplaceAll(name, " ", "_")),
	}
	
	// Add to database
	rs.refereeService.mutex.Lock()
	rs.refereeService.referees[newID] = referee
	rs.refereeService.mutex.Unlock()
	
	log.Printf("🆕 Created new referee record: %s (ID: %d)", name, newID)
	
	return newID
}

// ============================================================================
// ALTERNATIVE SOURCES
// ============================================================================

// ScrapeFromDailyFaceoff attempts to get referee assignments from Daily Faceoff
func (rs *RefereeScraper) ScrapeFromDailyFaceoff(date time.Time) ([]models.RefereeGameAssignment, error) {
	log.Printf("🔍 Attempting to scrape Daily Faceoff for %s...", date.Format("2006-01-02"))
	
	url := "https://www.dailyfaceoff.com/nhl-referees-and-linesmen-schedule/"
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", rs.userAgent)
	
	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Parse HTML to extract assignments
	assignments, err := rs.parseDailyFaceoffHTML(string(body), date)
	if err != nil {
		return nil, fmt.Errorf("failed to parse assignments: %w", err)
	}
	
	return assignments, nil
}

// parseDailyFaceoffHTML parses referee assignments from Daily Faceoff HTML
func (rs *RefereeScraper) parseDailyFaceoffHTML(htmlContent string, date time.Time) ([]models.RefereeGameAssignment, error) {
	var assignments []models.RefereeGameAssignment
	
	// Daily Faceoff uses a table structure with game matchups and referee assignments
	// Format: Team A @ Team B | Referee 1, Referee 2 | Linesman 1, Linesman 2
	
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Find the referee schedule table
	var findTable func(*html.Node) *html.Node
	findTable = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && n.Data == "table" {
			// Check if this is the referee table (look for class or id)
			for _, attr := range n.Attr {
				if (attr.Key == "class" || attr.Key == "id") && 
				   (strings.Contains(attr.Val, "referee") || strings.Contains(attr.Val, "official")) {
					return n
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if result := findTable(c); result != nil {
				return result
			}
		}
		return nil
	}
	
	table := findTable(doc)
	if table == nil {
		log.Printf("⚠️ Could not find referee table in Daily Faceoff HTML")
		return assignments, nil
	}
	
	// Parse table rows
	var parseRows func(*html.Node)
	parseRows = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			assignment := rs.parseAssignmentRow(n, date)
			if assignment != nil {
				assignments = append(assignments, *assignment)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseRows(c)
		}
	}
	
	parseRows(table)
	
	log.Printf("✅ Parsed %d referee assignments from Daily Faceoff", len(assignments))
	return assignments, nil
}

// parseAssignmentRow parses a single row from Daily Faceoff table
func (rs *RefereeScraper) parseAssignmentRow(row *html.Node, date time.Time) *models.RefereeGameAssignment {
	var cells []string
	
	// Extract all cell texts
	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "td" || n.Data == "th") {
			text := rs.getNodeText(n)
			cells = append(cells, strings.TrimSpace(text))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}
	extractText(row)
	
	if len(cells) < 3 {
		return nil
	}
	
	// Try to parse matchup (e.g., "BUF @ UTA")
	matchup := cells[0]
	teams := strings.Split(matchup, "@")
	if len(teams) != 2 {
		teams = strings.Split(matchup, "at")
	}
	if len(teams) != 2 {
		return nil
	}
	
	awayTeam := strings.TrimSpace(teams[0])
	homeTeam := strings.TrimSpace(teams[1])
	
	// Parse referees from next cell
	refCell := cells[1]
	refs := strings.Split(refCell, ",")
	
	ref1Name := ""
	ref2Name := ""
	if len(refs) > 0 {
		ref1Name = strings.TrimSpace(refs[0])
	}
	if len(refs) > 1 {
		ref2Name = strings.TrimSpace(refs[1])
	}
	
	// Generate game ID (we don't have real one yet)
	gameID := rs.generateGameID(date, homeTeam, awayTeam)
	
	return &models.RefereeGameAssignment{
		GameID:        gameID,
		GameDate:      date,
		HomeTeam:      homeTeam,
		AwayTeam:      awayTeam,
		Referee1ID:    rs.findRefereeID(ref1Name),
		Referee1Name:  ref1Name,
		Referee2ID:    rs.findRefereeID(ref2Name),
		Referee2Name:  ref2Name,
		Source:        "DailyFaceoff",
		LastUpdated:   time.Now(),
	}
}

// getNodeText extracts text content from an HTML node
func (rs *RefereeScraper) getNodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += rs.getNodeText(c)
	}
	return text
}

// findRefereeID looks up referee ID by name
func (rs *RefereeScraper) findRefereeID(name string) int {
	if name == "" {
		return 0
	}
	
	// Look through our referee database
	for id, ref := range rs.refereeService.referees {
		if ref.FullName == name {
			return id
		}
	}
	
	// If not found, generate new ID
	return rs.generateRefereeID(name)
}

// generateGameID creates a temporary game ID
func (rs *RefereeScraper) generateGameID(date time.Time, home, away string) int {
	// Format: YYYYMMDDHHMMSS + hash of teams
	dateStr := date.Format("20060102")
	teamStr := home + away
	hash := 0
	for _, char := range teamStr {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	id, _ := strconv.Atoi(dateStr + fmt.Sprintf("%04d", hash%10000))
	return id
}

// ============================================================================
// JSON API ENDPOINTS (if available)
// ============================================================================

// FetchFromJSONAPI attempts to fetch from a JSON API if available
func (rs *RefereeScraper) FetchFromJSONAPI(url string) ([]models.RefereeGameAssignment, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", rs.userAgent)
	
	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}
	
	var assignments []models.RefereeGameAssignment
	if err := json.NewDecoder(resp.Body).Decode(&assignments); err != nil {
		return nil, err
	}
	
	return assignments, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// convertSeasonToString converts season int (20242025) to string format ("2024-25")
func (rs *RefereeScraper) convertSeasonToString(seasonInt int) string {
	startYear := seasonInt / 10000
	endYear := seasonInt % 10000
	return fmt.Sprintf("%d-%02d", startYear, endYear%100)
}

// parseSeasonToInt converts season string to int
// Handles formats: "2024-25" -> 20242025, "2024-2025" -> 20242025
func (rs *RefereeScraper) parseSeasonToInt(season string) int {
	parts := strings.Split(season, "-")
	if len(parts) != 2 {
		// Try parsing as single int
		if val, err := strconv.Atoi(season); err == nil {
			return val
		}
		return 0
	}
	
	startYear, err1 := strconv.Atoi(parts[0])
	endYear, err2 := strconv.Atoi(parts[1])
	
	if err1 != nil || err2 != nil {
		log.Printf("⚠️ Invalid season format: %s", season)
		return 0
	}
	
	// Handle 2-digit end year (e.g., "2024-25")
	if endYear < 100 {
		// Assume it's the last 2 digits
		endYear = (startYear / 100 * 100) + endYear
		// Handle century rollover
		if endYear < startYear {
			endYear += 100
		}
	}
	
	// Validate that end year is exactly 1 more than start year
	if endYear != startYear+1 {
		log.Printf("⚠️ Invalid season range: %d to %d", startYear, endYear)
		// Correct it
		endYear = startYear + 1
	}
	
	return startYear*10000 + endYear
}


// cleanHTMLString removes HTML tags and trims whitespace
func cleanHTMLString(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]+>`)
	cleaned := re.ReplaceAllString(s, "")
	
	// Decode HTML entities
	cleaned = html.UnescapeString(cleaned)
	
	// Trim whitespace
	cleaned = strings.TrimSpace(cleaned)
	
	return cleaned
}

// parseHTMLTable parses an HTML table into a 2D string slice
func parseHTMLTable(tableHTML string) ([][]string, error) {
	var rows [][]string
	
	rowRegex := regexp.MustCompile(`<tr[^>]*>(.*?)</tr>`)
	rowMatches := rowRegex.FindAllStringSubmatch(tableHTML, -1)
	
	for _, rowMatch := range rowMatches {
		cellRegex := regexp.MustCompile(`<t[dh][^>]*>(.*?)</t[dh]>`)
		cellMatches := cellRegex.FindAllStringSubmatch(rowMatch[1], -1)
		
		var row []string
		for _, cellMatch := range cellMatches {
			row = append(row, cleanHTMLString(cellMatch[1]))
		}
		
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	
	return rows, nil
}

// ============================================================================
// AUTOMATED UPDATE FUNCTIONS
// ============================================================================

// RunDailyUpdate performs a full daily update of referee data
func (rs *RefereeScraper) RunDailyUpdate() error {
	log.Printf("🔄 Starting daily referee data update...")
	
	// 1. Update referee statistics (use dynamic season)
	currentSeason := rs.convertSeasonToString(rs.refereeService.currentSeason)
	log.Printf("📅 Fetching data for season: %s", currentSeason)
	if err := rs.ScrapeRefereeStats(currentSeason); err != nil {
		log.Printf("⚠️ Failed to scrape referee stats: %v", err)
		// Continue with assignments even if stats fail
	}
	
	// Small delay to be respectful to servers
	time.Sleep(2 * time.Second)
	
	// 2. Try to fetch today's assignments
	today := time.Now()
	if err := rs.ScrapeGameAssignments(today); err != nil {
		log.Printf("⚠️ Failed to scrape game assignments: %v", err)
	}
	
	// 3. Try tomorrow's assignments (often posted in advance)
	tomorrow := today.Add(24 * time.Hour)
	if err := rs.ScrapeGameAssignments(tomorrow); err != nil {
		log.Printf("⚠️ Failed to scrape tomorrow's assignments: %v", err)
	}
	
	log.Printf("✅ Daily referee update complete")
	return nil
}

// BackfillRefereeAssignments backfills referee data from completed games
func (rs *RefereeScraper) BackfillRefereeAssignments(startDate, endDate time.Time) error {
	log.Printf("🔄 Starting referee data backfill from %s to %s...",
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	totalDays := int(endDate.Sub(startDate).Hours() / 24)
	collected := 0
	failed := 0
	
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		log.Printf("📅 Backfilling %s...", date.Format("2006-01-02"))
		
		// Try NHL API first (more reliable for completed games)
		assignments, err := rs.FetchRefereesFromNHLAPI(date)
		if err != nil {
			log.Printf("⚠️ Failed to fetch from NHL API for %s: %v", date.Format("2006-01-02"), err)
			failed++
		} else if len(assignments) > 0 {
			// Store assignments
			for _, assignment := range assignments {
				if err := rs.refereeService.AddGameAssignment(&assignment); err != nil {
					log.Printf("⚠️ Failed to store assignment: %v", err)
				} else {
					collected++
				}
			}
		}
		
		// Rate limiting - don't hammer the API
		time.Sleep(500 * time.Millisecond)
	}
	
	log.Printf("✅ Backfill complete! Collected %d assignments across %d days (%d failed)",
		collected, totalDays, failed)
	
	return nil
}

// BackfillFromSeasonStart backfills from the start of the current season
func (rs *RefereeScraper) BackfillFromSeasonStart() error {
	// NHL season typically starts in October
	currentYear := time.Now().Year()
	seasonStart := time.Date(currentYear, time.October, 1, 0, 0, 0, 0, time.UTC)
	
	// If we're before October, use last year's season
	if time.Now().Month() < time.October {
		seasonStart = time.Date(currentYear-1, time.October, 1, 0, 0, 0, 0, time.UTC)
	}
	
	today := time.Now()
	
	log.Printf("📅 Backfilling from season start (%s) to today...", seasonStart.Format("2006-01-02"))
	
	return rs.BackfillRefereeAssignments(seasonStart, today)
}

// ScheduleAutomaticUpdates starts a goroutine that runs daily updates
func (rs *RefereeScraper) ScheduleAutomaticUpdates() {
	go func() {
		// Run immediately on startup
		rs.RunDailyUpdate()
		
		// Then run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
			rs.RunDailyUpdate()
		}
	}()
	
	log.Printf("📅 Scheduled automatic referee data updates (every 24 hours)")
}

