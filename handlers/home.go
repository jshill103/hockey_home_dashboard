package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
)

// HandleHome serves the main dashboard page
func HandleHome(w http.ResponseWriter, r *http.Request) {
	// Get analysis content directly
	analysisContent := getAnalysisContent()

	// Generate dynamic CSS with team colors
	dynamicCSS := generateTeamCSS()

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>NHL Team Dashboard</title>
    <link rel="icon" type="image/png" href="` + teamConfig.FaviconPath + `">
    <link rel="apple-touch-icon" href="` + teamConfig.FaviconPath + `">
    <script src="https://unpkg.com/htmx.org@1.9.2"></script>
    <style>
        ` + dynamicCSS + `
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(rgba(0, 0, 0, 0.4), rgba(0, 0, 0, 0.4)), url('` + teamConfig.BackgroundImage + `');
            background-size: cover;
            background-position: center;
            background-attachment: fixed;
            color: white;
            min-height: 100vh;
            overflow-x: hidden;
            padding-bottom: 130px;
        }
        
        .header {
            text-align: center;
            padding: 20px;
            background: rgba(0, 0, 0, 0.4);
            margin-bottom: 20px;
        }
        
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }
        
        /* Bottom Scrolling Banner Styles */
        .bottom-banner {
            background: linear-gradient(135deg, var(--team-secondary), var(--team-primary-dark), var(--team-primary));
            height: 120px;
            width: 100%;
            position: fixed;
            bottom: 0;
            left: 0;
            z-index: 1000;
            overflow: hidden;
            box-shadow: 0 -2px 10px rgba(15, 52, 96, 0.4);
            border-top: 2px solid rgba(0, 123, 255, 0.3);
        }
        
        .banner-content {
            padding: 0;
            margin: 0;
            display: inline-block;
            white-space: nowrap;
            animation: scroll-ticker 60s linear infinite;
            line-height: 120px;
            height: 120px;
            will-change: transform;
        }
        
        /* Scrolling ticker animation - smooth continuous scroll */
        @keyframes scroll-ticker {
            0% { transform: translateX(100vw); }
            100% { transform: translateX(-100%); }
        }
        
        /* Add a subtle shine effect */
        .bottom-banner::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(90deg, transparent, rgba(0, 123, 255, 0.2), transparent);
            animation: shimmer 4s ease-in-out infinite;
        }
        
        @keyframes shimmer {
            0% { left: -100%; }
            100% { left: 100%; }
        }
        
        .main-layout {
            display: grid;
            grid-template-columns: 1fr 1fr 1fr;
            gap: 20px;
            padding: 0 20px;
            max-width: 2400px;
            margin: 0 auto;
        }
        
        .scoreboard-section {
            grid-column: 1 / -1;
            grid-row: 1;
            background: rgba(0, 0, 0, 0.3);
            padding: 30px 40px 30px 40px;
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.3);
            min-height: 160px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-bottom: 30px;
        }
        
        .news-section {
            grid-column: 1;
            grid-row: 1;
            background: rgba(0, 0, 0, 0.3);
            padding: 30px 35px 30px 35px;
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.3);
            height: calc(100vh - 180px);
            width: 100%;
            display: flex;
            flex-direction: column;
            justify-content: flex-start;
            overflow-y: hidden;
        }
        
        .news-section h2 {
            margin: 0 0 18px 0;
            color: white;
            font-size: 2.0em;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }
        
        .news-item {
            border-bottom: 1px solid rgba(255, 255, 255, 0.2);
            padding: 12px 0;
            margin-bottom: 8px;
        }
        
        .news-item:last-child {
            border-bottom: none;
        }
        
        .news-item a {
            color: var(--team-primary);
            text-decoration: none;
            font-weight: bold;
            font-size: 1.2em;
            display: block;
            margin-bottom: 8px;
            line-height: 1.4;
        }
        
        .news-item a:hover {
            color: var(--team-primary-dark);
            text-decoration: underline;
        }
        
        /* Upcoming Games Styles */
        .upcoming-games-section {
            grid-column: 1;
            grid-row: 1;
            background: rgba(0, 0, 0, 0.3);
            padding: 30px 35px 30px 35px;
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.3);
            height: calc(100vh - 180px);
            width: 100%;
            display: flex;
            flex-direction: column;
            justify-content: flex-start;
            overflow-y: hidden;
        }
        
        .upcoming-games-section h2 {
            margin: 0 0 25px 0;
            color: white;
            font-size: 2.2em;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }
        
        .game-item {
            border-bottom: 1px solid rgba(255, 255, 255, 0.2);
            padding: 12px 0;
            margin-bottom: 8px;
        }
        
        .game-item:last-child {
            border-bottom: none;
        }
        
        .game-date {
            color: var(--team-accent);
            font-weight: bold;
            font-size: 1.4em;
            margin-bottom: 8px;
        }
        
        .game-matchup {
            margin-bottom: 4px;
        }
        
        .game-teams {
            color: var(--team-primary);
            font-weight: bold;
            font-size: 1.7em;
        }
        
        .game-time {
            color: var(--team-primary);
            font-size: 1.4em;
        }
        
        /* Player Stats Section */
        .player-stats-section {
            grid-column: 2;
            grid-row: 1;
            background: rgba(0, 0, 0, 0.3);
            padding: 30px 35px 30px 35px;
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.3);
            height: calc(100vh - 180px);
            width: 100%;
            display: flex;
            flex-direction: column;
            justify-content: flex-start;
            overflow-y: hidden;
        }
        
        .player-stats-section h2 {
            margin: 0 0 25px 0;
            color: white;
            font-size: 2.2em;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }
        
        .goalie-stats-subsection {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 2px solid rgba(255, 255, 255, 0.2);
        }
        
        .goalie-stats-subsection h3 {
            margin: 0 0 15px 0;
            color: var(--team-accent);
            font-size: 1.3em;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }
        
        .player-grid, .goalie-grid {
            display: grid;
            grid-template-columns: 1fr;
            gap: 10px;
            margin-bottom: 15px;
        }
        
        .player-card, .goalie-card {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 8px;
            padding: 12px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            transition: background-color 0.3s ease;
        }
        
        .player-card:hover, .goalie-card:hover {
            background: rgba(255, 255, 255, 0.15);
        }
        
        .player-info, .goalie-info {
            display: flex;
            flex-direction: column;
        }
        
        .player-name, .goalie-name {
            font-weight: bold;
            font-size: 1.7em;
            color: white;
        }
        
        .player-position, .goalie-position {
            font-size: 1.4em;
            color: #aaa;
        }
        
        .player-stats, .goalie-stats {
            text-align: right;
        }
        
        .stat-value {
            font-size: 2.2em;
            font-weight: bold;
            color: var(--team-primary);
        }
        
        .stat-label {
            font-size: 1.3em;
            color: #ccc;
            text-transform: uppercase;
        }
        
        .stat-category {
            margin-bottom: 20px;
        }
        
        .stat-category h4 {
            margin: 0 0 15px 0;
            color: var(--team-accent);
            font-size: 1.7em;
            text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.8);
        }
        
        .no-stats {
            text-align: center;
            color: #aaa;
            font-style: italic;
            padding: 20px;
        }
        

        
        /* Season Countdown Styles */
        .season-countdown-section {
            grid-column: 2;
            grid-row: 1;
            background: rgba(0, 0, 0, 0.3);
            padding: 30px 35px 30px 35px;
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.3);
            height: calc(100vh - 180px);
            width: 100%;
            display: flex;
            flex-direction: column;
            justify-content: flex-start;
            overflow-y: hidden;
        }



        /* Countdown Components */
        .countdown-container {
            display: flex;
            flex-direction: column;
            gap: 10px;
            height: calc(100% - 20px);
            padding: 10px 0;
            overflow: hidden;
        }

        .main-countdown, .uta-countdown {
            background: linear-gradient(135deg, var(--team-secondary), var(--team-primary-dark));
            border-radius: 10px;
            padding: 12px;
            border: 2px solid rgba(0, 123, 255, 0.3);
            flex: 1;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            min-height: 0;
            max-height: 50%;
        }

        .countdown-header h3, .countdown-header h4 {
            margin: 0 0 6px 0;
            color: var(--team-accent);
            text-align: center;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
            font-size: 1.5em;
            line-height: 1.0;
        }

        .countdown-main, .uta-countdown-main {
            text-align: center;
            margin-bottom: 6px;
            display: flex;
            justify-content: center;
            align-items: center;
            flex: 1;
        }

        .days-counter, .uta-days-counter {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 8px;
        }

        .days-number, .uta-days-number {
            font-size: 3.2em;
            font-weight: bold;
            color: var(--team-primary);
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
            line-height: 0.8;
        }

        .days-label, .uta-days-label {
            font-size: 0.9em;
            color: #a8c8ec;
            text-transform: uppercase;
            letter-spacing: 1px;
            line-height: 1.0;
        }

        .first-game-details, .uta-game-details {
            background: rgba(0, 123, 255, 0.1);
            border-radius: 6px;
            padding: 8px;
            border: 1px solid rgba(0, 123, 255, 0.2);
            flex-shrink: 0;
        }

        .game-info, .uta-game-info {
            text-align: center;
            padding: 2px 0;
        }

        .game-teams, .uta-game-teams {
            font-size: 1.2em;
            font-weight: bold;
            color: #e0e8f5;
            margin-bottom: 4px;
            line-height: 1.0;
        }

        .game-date, .uta-game-date {
            font-size: 1.0em;
            color: #ffc107;
            margin-bottom: 2px;
            line-height: 1.1;
        }

        .game-venue, .uta-game-venue {
            font-size: 0.95em;
            color: #a8c8ec;
            line-height: 1.0;
            margin: 0;
        }

        .no-countdown, .countdown-error, .season-started {
            text-align: center;
            color: #aaa;
            font-style: italic;
            padding: 20px;
        }

        .season-started {
            color: var(--team-primary);
            font-weight: bold;
            font-size: 1.2em;
        }

        /* Team Analysis Styles */
        .mammoth-analysis-section {
            grid-column: 3;
            grid-row: 1;
            background: rgba(0, 0, 0, 0.3);
            padding: 30px 35px 30px 35px;
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.3);
            height: calc(100vh - 180px);
            width: 100%;
            display: block;
            overflow-y: hidden;
        }
        
        .mammoth-analysis-section h2 {
            margin: 0 0 20px 0;
            color: white;
            font-size: 2.0em;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }
        
        .last-updated {
            margin-top: auto;
            padding-top: 12px;
            font-size: 1.1em;
            color: #ccc;
            text-align: center;
            border-top: 1px solid rgba(255, 255, 255, 0.2);
        }
        
        .news-loading {
            color: var(--team-accent);
        }
        
        .news-updated {
            color: var(--team-primary);
        }
        
        .news-error {
            color: #dc3545;
        }
        
        /* Season-based visibility classes */
        .hockey-season-only {
            display: none;
        }
        .offseason-only {
            display: none;
        }
        body.hockey-season .hockey-season-only {
            display: flex;
        }
        body.hockey-season .offseason-only {
            display: none;
        }
        body.offseason .offseason-only {
            display: flex;
        }
        .mammoth-analysis-section {
            display: block;
        }
        body.offseason .mammoth-analysis-section {
            display: block;
        }
        body.offseason .hockey-season-only {
            display: none;
        }
        

        
        /* Responsive Design */
        @media (max-width: 1200px) {
            .main-layout {
                grid-template-columns: 1fr;
                grid-template-rows: auto auto auto auto auto auto;
            }
            
            .scoreboard-section {
                grid-column: 1;
                grid-row: 1;
            }
            
            .news-section {
                grid-column: 1;
                grid-row: 2;
            }
            
            .upcoming-games-section {
                grid-column: 1;
                grid-row: 3;
            }
            

            
            .season-countdown-section {
                grid-column: 1;
                grid-row: 4;
            }
            
            .mammoth-analysis-section {
                grid-column: 1;
                grid-row: 5;
            }
            
            .player-stats-section {
                grid-column: 1;
                grid-row: 6;
            }
            
            /* Hockey season responsive layout */
            body.hockey-season .main-layout {
                grid-template-columns: 1fr;
                grid-template-rows: auto auto auto auto;
            }
            
            body.hockey-season .upcoming-games-section {
                grid-row: 2;
            }
            
            body.hockey-season .player-stats-section {
                grid-row: 3;
            }
            
            body.hockey-season .playoff-odds-section {
                grid-row: 4;
            }
        }
        
        .analysis-box {
            background: rgba(0, 0, 0, 0.3);
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.3);
            padding: 15px 20px 15px 20px;
            min-height: 450px;
            width: 100%;
            color: white;
            overflow-y: hidden;
        }

        /* Enhanced Analysis Section Styles - Dark Blue Theme */
        .analysis-header-enhanced {
            margin-bottom: 25px;
            text-align: center;
            background: linear-gradient(135deg, var(--team-secondary), var(--team-primary-dark));
            padding: 20px;
            border-radius: 12px;
            border: 2px solid rgba(0, 123, 255, 0.3);
        }

        .analysis-header-enhanced h3 {
            margin: 0 0 10px 0;
            color: var(--team-accent);
            font-size: 1.5em;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }

        .season-info {
            color: #a8c8ec;
            font-size: 0.9em;
            margin-bottom: 8px;
        }

        .team-record-enhanced {
            color: #e0e8f5;
            font-size: 1.2em;
            font-weight: bold;
        }

        /* Current Form Section */
        .current-form-section {
            margin-bottom: 20px;
            background: linear-gradient(135deg, var(--team-primary), var(--team-secondary));
            padding: 18px;
            border-radius: 10px;
            border-left: 4px solid var(--team-accent);
        }

        .form-grid {
            display: grid;
            grid-template-columns: 1fr 1fr 1fr;
            gap: 15px;
        }

        .form-box {
            background: rgba(0, 123, 255, 0.1);
            padding: 15px;
            border-radius: 8px;
            text-align: center;
            border: 1px solid rgba(0, 123, 255, 0.2);
        }

        .form-label {
            font-size: 0.8em;
            color: #a8c8ec;
            margin-bottom: 5px;
        }

        .form-value {
            font-size: 1.1em;
            font-weight: bold;
            color: #e0e8f5;
        }

        .analysis-section {
            margin-bottom: 12px;
            padding-bottom: 8px;
            border-bottom: 1px solid rgba(94, 179, 245, 0.2);
        }

        .analysis-section:last-child {
            border-bottom: none;
        }

        .analysis-section h4 {
            margin: 0 0 8px 0;
            color: var(--team-accent);
            font-size: 1.0em;
            border-left: 3px solid var(--team-accent);
            padding-left: 8px;
        }

        /* Enhanced Metrics Grid */
        .metrics-grid-enhanced {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
            gap: 8px;
            margin-bottom: 10px;
        }

        .metric-box-enhanced {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-secondary));
            padding: 15px;
            border-radius: 8px;
            text-align: center;
            border: 1px solid rgba(0, 123, 255, 0.2);
            transition: all 0.3s ease;
        }

        .metric-box-enhanced.positive {
            border-color: rgba(40, 167, 69, 0.5);
            box-shadow: 0 0 10px rgba(40, 167, 69, 0.2);
        }

        .metric-box-enhanced.negative {
            border-color: rgba(220, 53, 69, 0.5);
            box-shadow: 0 0 10px rgba(220, 53, 69, 0.2);
        }

        /* Home vs Road Performance */
        .home-road-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }

        .location-box {
            background: linear-gradient(135deg, var(--team-primary), var(--team-primary-dark));
            padding: 18px;
            border-radius: 10px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .location-box h5 {
            margin: 0 0 10px 0;
            color: var(--team-accent);
            font-size: 1.1em;
        }

        .location-record {
            font-weight: bold;
            font-size: 1em;
            color: #e0e8f5;
        }

        .location-pct {
            font-size: 0.9em;
            color: #a8c8ec;
            margin-top: 5px;
        }

        /* League Standing */
        .standings-info {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-primary));
            padding: 15px;
            border-radius: 8px;
            border-left: 4px solid var(--team-accent);
        }

        .standing-item {
            margin: 8px 0;
            color: #e0e8f5;
        }

        .playoff-status {
            color: #90ee90;
            font-weight: bold;
        }

        /* Advanced Analytics */
        .analytics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
            gap: 12px;
        }

        .analytics-box {
            background: rgba(0, 123, 255, 0.1);
            padding: 15px;
            border-radius: 8px;
            text-align: center;
            border: 1px solid rgba(0, 123, 255, 0.2);
        }

        .analytics-label {
            font-size: 0.8em;
            color: #a8c8ec;
            margin-bottom: 5px;
        }

        .analytics-value {
            font-size: 1.1em;
            font-weight: bold;
            color: #e0e8f5;
        }

        /* Enhanced Insights */
        .insight-box-enhanced {
            background: linear-gradient(135deg, var(--team-primary), var(--team-secondary));
            padding: 15px;
            margin: 10px 0;
            border-radius: 8px;
            border-left: 4px solid var(--team-accent);
            color: #e0e8f5;
        }

        .insight-box-enhanced.trend-analysis {
            border-left-color: #17a2b8;
        }

        .insight-box-enhanced.playoff-analysis {
            border-left-color: #ffc107;
        }

        .insight-box-enhanced.form-analysis {
            border-left-color: #28a745;
        }

        .insight-box-enhanced.balance-analysis {
            border-left-color: #6f42c1;
        }

        /* Enhanced Strengths & Weaknesses */
        .strength-weakness-grid-enhanced {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }

        .strength-box-enhanced, .weakness-box-enhanced {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-primary));
            padding: 18px;
            border-radius: 10px;
        }

        .strength-box-enhanced {
            border-left: 4px solid #28a745;
            border-top: 1px solid rgba(40, 167, 69, 0.3);
        }

        .weakness-box-enhanced {
            border-left: 4px solid #dc3545;
            border-top: 1px solid rgba(220, 53, 69, 0.3);
        }

        .strength-box-enhanced h4, .weakness-box-enhanced h4 {
            margin: 0 0 12px 0;
            font-size: 1.1em;
            color: var(--team-accent);
        }

        .strength-item-enhanced, .weakness-item-enhanced {
            margin: 8px 0;
            font-size: 0.9em;
            padding: 5px 0;
        }

        .strength-item-enhanced {
            color: #90ee90;
        }

        .weakness-item-enhanced {
            color: #ffb3ba;
        }

        /* Performance Trends */
        .trends-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }

        .trend-item {
            background: linear-gradient(135deg, var(--team-primary), var(--team-primary-dark));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.2);
        }

        .trend-item.offensive {
            border-left: 4px solid #17a2b8;
        }

        .trend-item.defensive {
            border-left: 4px solid #6f42c1;
        }

        /* Season Projection */
        .season-projection {
            background: linear-gradient(135deg, #1a1a2e, #0f3460);
            padding: 20px;
            border-radius: 12px;
            border: 2px solid rgba(0, 123, 255, 0.3);
        }

        .projection-box {
            background: rgba(0, 123, 255, 0.1);
            padding: 15px;
            border-radius: 8px;
        }

        .projection-item {
            margin: 10px 0;
            padding: 8px 12px;
            background: rgba(255, 255, 255, 0.05);
            border-radius: 6px;
            color: #e0e8f5;
        }

        .projection-item.playoff-target {
            border-left: 4px solid #ffc107;
            background: rgba(255, 193, 7, 0.1);
        }

        .metric-label {
            font-size: 1.3em;
            color: #a8c8ec;
            margin-bottom: 8px;
        }

        .metric-value {
            font-size: 1.2em;
            font-weight: bold;
            color: #e0e8f5;
        }

        /* Rotating Analysis Sections */
        .analysis-container {
            position: relative;
            height: calc(100vh - 280px);
            min-height: calc(100vh - 280px);
            max-height: calc(100vh - 280px);
            overflow: hidden;
        }

        .analysis-nav {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
            padding: 10px 15px;
            background: linear-gradient(135deg, var(--team-primary), var(--team-secondary));
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.2);
        }

        .nav-dots {
            display: flex;
            gap: 8px;
        }

        .nav-dot {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            background: rgba(94, 179, 245, 0.3);
            cursor: pointer;
            transition: all 0.3s ease;
        }

        .nav-dot:hover {
            background: rgba(94, 179, 245, 0.6);
            transform: scale(1.1);
        }

        .nav-dot.active {
            background: var(--team-accent);
            box-shadow: 0 0 8px rgba(94, 179, 245, 0.5);
        }

        .section-indicator {
            font-size: 0.9em;
            color: #a8c8ec;
            font-weight: bold;
        }

        .analysis-section-rotating {
            display: none;
            opacity: 0;
            transition: opacity 0.5s ease;
            height: calc(100vh - 380px);
            min-height: calc(100vh - 380px);
            max-height: calc(100vh - 380px);
            overflow-y: hidden;
            overflow-x: hidden;
        }

        .analysis-section-rotating.active {
            display: block;
            opacity: 1;
        }

        .section-header {
            text-align: center;
            margin-bottom: 12px;
            padding: 10px;
            background: linear-gradient(135deg, var(--team-secondary), var(--team-primary-dark));
            border-radius: 8px;
            border: 2px solid rgba(0, 123, 255, 0.3);
        }

        .section-header h3 {
            margin: 0 0 8px 0;
            color: var(--team-accent);
            font-size: 2.2em;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }

        .section-subtitle {
            color: #a8c8ec;
            font-size: 1.4em;
            margin: 0;
        }

        /* Section 1: Overview Styles */
        .overview-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 12px;
            height: auto;
            max-height: calc(100% - 70px);
            overflow: hidden;
        }

        .overview-main {
            background: linear-gradient(135deg, var(--team-primary), var(--team-primary-dark));
            padding: 15px;
            border-radius: 8px;
            text-align: center;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .team-record-large {
            font-size: 2.8em;
            font-weight: bold;
            color: var(--team-accent);
            margin-bottom: 12px;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }

        .team-points {
            font-size: 1.5em;
            color: #e0e8f5;
            margin-bottom: 10px;
        }

        .division-info {
            font-size: 1.3em;
            color: #a8c8ec;
        }

        .current-form {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-primary));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .streak-info, .last10-info, .games-progress {
            margin: 12px 0;
            padding: 10px 15px;
            background: rgba(0, 123, 255, 0.1);
            border-radius: 6px;
            font-size: 1.3em;
            color: #e0e8f5;
        }

        /* Section 2: Metrics Styles */
        .metrics-compact-grid {
            display: grid;
            grid-template-columns: 1fr 1fr 1fr;
            gap: 15px;
            margin-bottom: 20px;
        }

        .metric-compact {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-secondary));
            padding: 15px;
            border-radius: 8px;
            text-align: center;
            border: 1px solid rgba(0, 123, 255, 0.2);
        }

        .metric-compact.positive {
            border-color: rgba(40, 167, 69, 0.5);
            box-shadow: 0 0 10px rgba(40, 167, 69, 0.2);
        }

        .metric-compact.negative {
            border-color: rgba(220, 53, 69, 0.5);
            box-shadow: 0 0 10px rgba(220, 53, 69, 0.2);
        }

        .metric-value-large {
            font-size: 2.4em;
            font-weight: bold;
            color: #e0e8f5;
            margin-top: 8px;
        }

        .analytics-compact {
            background: linear-gradient(135deg, var(--team-primary), var(--team-primary-dark));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .analytics-item {
            margin: 15px 0;
            color: #e0e8f5;
            font-size: 1.4em;
        }

        /* Section 3: Location & Standing Styles */
        .location-standings-grid {
            display: grid;
            grid-template-rows: 1fr auto;
            gap: 8px;
            height: auto;
            max-height: calc(100% - 60px);
            overflow: hidden;
        }

        .location-splits {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 8px;
        }

        .split-item {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-primary));
            padding: 8px;
            border-radius: 6px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .split-label {
            font-weight: bold;
            color: var(--team-accent);
            margin-bottom: 6px;
            font-size: 1.3em;
        }

        .split-record {
            font-size: 1.35em;
            color: #e0e8f5;
            margin-bottom: 5px;
        }

        .split-pct {
            font-size: 1.2em;
            color: #a8c8ec;
        }

        .league-position {
            background: linear-gradient(135deg, var(--team-primary), var(--team-secondary));
            padding: 8px;
            border-radius: 6px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .position-item {
            margin: 6px 0;
            color: #e0e8f5;
            font-size: 1.2em;
        }

        .playoff-indicator {
            color: #90ee90;
            font-weight: bold;
        }

        /* Section 4: Analysis & Trends Styles */
        .insights-trends-grid {
            margin-bottom: 15px;
        }

        .insights-compact {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-primary));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
            margin-bottom: 15px;
        }

        .insight-compact {
            margin: 10px 0;
            padding: 8px 12px;
            background: rgba(0, 123, 255, 0.1);
            border-radius: 6px;
            color: #e0e8f5;
            font-size: 0.9em;
        }

        .strengths-weaknesses-compact {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
            margin-bottom: 15px;
        }

        .strengths-compact, .weaknesses-compact {
            background: linear-gradient(135deg, var(--team-primary), var(--team-primary-dark));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .strengths-compact h4, .weaknesses-compact h4 {
            margin: 0 0 10px 0;
            font-size: 1em;
            color: var(--team-accent);
        }

        .strength-compact, .weakness-compact {
            margin: 6px 0;
            font-size: 0.85em;
            padding: 4px 8px;
            border-radius: 4px;
        }

        .strength-compact {
            color: #90ee90;
            background: rgba(40, 167, 69, 0.1);
        }

        .weakness-compact {
            color: #ffb3ba;
            background: rgba(220, 53, 69, 0.1);
        }

        .performance-trends-compact {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }

        .trend-compact {
            background: linear-gradient(135deg, #1a1a2e, #0f3460);
            padding: 12px;
            border-radius: 6px;
            border: 1px solid rgba(0, 123, 255, 0.2);
            font-size: 0.9em;
            color: #e0e8f5;
        }

        /* New simplified analysis styles */
        .analysis-two-col {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
            margin-bottom: 15px;
        }

        .analysis-left, .analysis-right {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-primary));
            padding: 12px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .insight-brief {
            margin: 8px 0;
            padding: 6px 8px;
            background: rgba(255, 255, 255, 0.1);
            border-radius: 4px;
            font-size: 0.8em;
            line-height: 1.3;
        }

        .trend-brief {
            margin: 6px 0;
            padding: 8px;
            background: linear-gradient(135deg, #1a1a2e, #0f3460);
            border-radius: 6px;
            border: 1px solid rgba(0, 123, 255, 0.2);
            font-size: 0.8em;
            line-height: 1.3;
        }

        /* Simplified projection styles */
        .projection-simplified {
            display: flex;
            flex-direction: column;
            gap: 15px;
        }

        .projection-compact {
            background: linear-gradient(135deg, var(--team-primary), var(--team-primary-dark));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
            text-align: center;
        }

        .proj-main {
            font-size: 1.4em;
            color: var(--team-accent);
            margin-bottom: 5px;
        }

        .proj-detail {
            font-size: 0.9em;
            color: #ccc;
            margin-bottom: 8px;
        }

        .proj-target {
            font-size: 0.85em;
            color: #ffaa44;
            font-weight: bold;
        }

        .proj-good {
            font-size: 0.85em;
            color: #4CAF50;
            font-weight: bold;
        }

        /* Team summary styles */
        .team-summary {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 12px;
        }

        .summary-col {
            background: linear-gradient(135deg, var(--team-primary), var(--team-primary-dark));
            padding: 12px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .summary-col h4 {
            margin: 0 0 8px 0;
            font-size: 0.9em;
            color: var(--team-accent);
        }

        .summary-item {
            margin: 4px 0;
            font-size: 0.75em;
            line-height: 1.3;
            padding: 3px 6px;
            background: rgba(255, 255, 255, 0.1);
            border-radius: 3px;
        }

        /* Section 5: Projection Styles */
        .projection-content {
            height: calc(100% - 80px);
            display: grid;
            grid-template-rows: auto 1fr auto;
            gap: 15px;
        }

        .projection-main {
            background: linear-gradient(135deg, var(--team-primary), var(--team-secondary));
            padding: 20px;
            border-radius: 10px;
            text-align: center;
            border: 2px solid rgba(0, 123, 255, 0.3);
        }

        .projected-points {
            font-size: 2.5em;
            font-weight: bold;
            color: var(--team-accent);
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }

        .projection-label {
            color: #a8c8ec;
            font-size: 1em;
            margin-top: 5px;
        }

        .projection-details {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-primary));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .detail-row {
            margin: 8px 0;
            padding: 6px 12px;
            background: rgba(0, 123, 255, 0.1);
            border-radius: 4px;
            color: #e0e8f5;
            font-size: 0.9em;
        }

        .detail-row.playoff-target {
            border-left: 4px solid #ffc107;
            background: rgba(255, 193, 7, 0.1);
        }

        .detail-row.playoff-good {
            border-left: 4px solid #28a745;
            background: rgba(40, 167, 69, 0.1);
        }

        .playoff-outlook {
            background: linear-gradient(135deg, var(--team-secondary), var(--team-primary-dark));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .outlook-header {
            font-weight: bold;
            color: var(--team-accent);
            margin-bottom: 10px;
            font-size: 1em;
        }

        .outlook-text {
            color: #e0e8f5;
            font-size: 0.9em;
            line-height: 1.4;
        }

        /* Playoff Odds Section Styles */
        .playoff-odds-section {
            grid-column: 3;
            grid-row: 1;
            background: linear-gradient(135deg, var(--team-secondary), var(--team-primary-dark), var(--team-primary));
            border-radius: 15px;
            padding: 35px;
            box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
            border: 2px solid rgba(0, 123, 255, 0.3);  
            height: calc(100vh - 180px);
            overflow-y: hidden;
            flex-direction: column;
        }

        .playoff-odds-section h2 {
            margin: 0 0 30px 0;
            color: var(--team-accent);
            font-size: 2.4em;
            text-align: center;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }

        .playoff-odds-container {
            height: 100%;
            display: flex;
            flex-direction: column;
            gap: 15px;
        }

        .playoff-status-header {
            text-align: center;
            background: linear-gradient(135deg, var(--team-primary), var(--team-primary-dark));
            padding: 20px;
            border-radius: 10px;
            border: 1px solid rgba(0, 123, 255, 0.3);
        }

        .playoff-odds-main {
            font-size: 3em;
            font-weight: bold;
            color: var(--team-accent);
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
            margin-bottom: 5px;
        }

        .playoff-status-text {
            font-size: 1.1em;
            color: #a8c8ec;
            font-weight: bold;
        }

        .current-position,
        .projections,
        .whats-needed,
        .odds-breakdown {
            background: linear-gradient(135deg, var(--team-primary-dark), var(--team-secondary));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.2);
        }

        .current-position h4,
        .projections h4,
        .whats-needed h4,
        .odds-breakdown h4 {
            margin: 0 0 12px 0;
            color: var(--team-accent);
            font-size: 1em;
            text-align: center;
        }

        .position-grid,
        .projection-grid,
        .needs-grid,
        .odds-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 10px;
        }

        .position-item,
        .projection-item,
        .need-item,
        .odds-item {
            background: rgba(0, 123, 255, 0.1);
            padding: 8px 12px;
            border-radius: 6px;
            text-align: center;
        }

        .position-label,
        .projection-label,
        .need-label,
        .odds-label {
            font-size: 0.8em;
            color: #a8c8ec;
            margin-bottom: 4px;
        }

        .position-value,
        .projection-value,
        .need-value,
        .odds-value {
            font-size: 0.9em;
            color: #e0e8f5;
            font-weight: bold;
        }

        .division-spot {
            border-left: 4px solid #28a745;
            background: rgba(40, 167, 69, 0.1);
        }

        .wildcard-spot {
            border-left: 4px solid #ffc107;
            background: rgba(255, 193, 7, 0.1);
        }

        .out-of-playoffs {
            border-left: 4px solid #dc3545;
            background: rgba(220, 53, 69, 0.1);
        }

        .playoff-insight {
            background: linear-gradient(135deg, var(--team-primary), var(--team-secondary));
            padding: 15px;
            border-radius: 8px;
            border: 1px solid rgba(0, 123, 255, 0.3);
            text-align: center;
        }

        .insight-text {
            color: #e0e8f5;
            font-size: 0.95em;
            margin-bottom: 8px;
            font-weight: bold;
        }

        .milestone-text {
            color: #a8c8ec;
            font-size: 0.85em;
            font-style: italic;
        }
        
        .analysis-header {
            display: flex;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 15px;
            border-bottom: 2px solid rgba(255, 255, 255, 0.2);
        }
        
        .analysis-title {
            font-size: 1.8em;
            font-weight: bold;
            color: var(--team-primary);
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
        }
        
        .analysis-section {
            margin-bottom: 20px;
        }
        
        .analysis-section h3 {
            color: var(--team-accent);
            font-size: 1.0em;
            margin-bottom: 6px;
            border-left: 3px solid var(--team-accent);
            padding-left: 8px;
        }
        
        .analysis-content {
            line-height: 1.4;
            color: #e0e0e0;
            margin-bottom: 8px;
            font-size: 0.9em;
        }
        
        .stats-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
            margin: 15px 0;
        }
        
        .stat-item {
            background: rgba(255, 255, 255, 0.1);
            padding: 10px;
            border-radius: 8px;
            text-align: center;
        }
        
        .stat-value {
            font-size: 1.4em;
            font-weight: bold;
            color: #28a745;
        }
        
        .stat-label {
            font-size: 0.9em;
            color: #ccc;
            margin-top: 5px;
        }
        
        .strength-list, .weakness-list {
            list-style: none;
            padding: 0;
        }
        
        .strength-list li {
            background: rgba(40, 167, 69, 0.2);
            margin: 5px 0;
            padding: 8px 12px;
            border-radius: 6px;
            border-left: 4px solid #28a745;
        }
        
        .weakness-list li {
            background: rgba(220, 53, 69, 0.2);
            margin: 5px 0;
            padding: 8px 12px;
            border-radius: 6px;
            border-left: 4px solid #dc3545;
        }
        
        .trend-indicator {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 0.8em;
            font-weight: bold;
            margin-left: 10px;
        }
        
        .trend-positive {
            background: rgba(40, 167, 69, 0.3);
            color: #28a745;
        }
        
        .trend-negative {
            background: rgba(220, 53, 69, 0.3);
            color: #dc3545;
        }
        
        .trend-neutral {
            background: rgba(255, 193, 7, 0.3);
            color: #ffc107;
        }
        
        /* Banner Styles */
        .away-indicator, .home-indicator {
            font-weight: bold;
            padding: 6px 18px;
            border-radius: 12px;
            font-size: 1.8em;
            margin-right: 15px;
        }
        
        .away-indicator {
            background-color: #dc3545;
            color: white;
        }
        
        .home-indicator {
            background-color: #28a745;
            color: white;
        }
        
        .tv-broadcasts {
            color: #ffc107;
            font-style: italic;
            margin-left: 10px;
        }
    </style>
</head>
<body>
    
    <div class="scoreboard-section hockey-season-only" style="display: none;">
        <div id="scoreboard-content">
        </div>
    </div>
    
    <div class="main-layout">
        <div class="news-section offseason-only">
            <h2>📰 NHL News</h2>
            <div id="news-content">
                <p>Loading NHL news headlines...</p>
            </div>
            <div class="last-updated">
                News automatically updates every 10 minutes
            </div>
        </div>
        
        <div class="upcoming-games-section hockey-season-only">
            <h2>🗓️ Upcoming Games</h2>
            <div id="upcoming-games-content">
                <p>Loading upcoming games...</p>
            </div>
            <div class="last-updated">
                Schedule automatically updates daily at midnight
            </div>
        </div>
        
        <div class="player-stats-section hockey-season-only">
            <h2>⭐ Player Stats</h2>
            <div id="player-stats-content">
                <p>Loading player statistics...</p>
            </div>
            <div class="goalie-stats-subsection">
                <h3>🥅 Goalie Stats</h3>
                <div id="goalie-stats-content">
                    <p>Loading goalie statistics...</p>
                </div>
            </div>
            <div class="last-updated">
                Player stats automatically update every 30 minutes
            </div>
        </div>
        
        <div class="playoff-odds-section hockey-season-only">
            <h2>🏆 Playoff Odds</h2>
            <div id="playoff-odds-content">
                <p>Loading playoff odds...</p>
            </div>
            <div class="last-updated">
                Playoff odds automatically update with standings
            </div>
        </div>
        

        
        <div class="season-countdown-section offseason-only">
            <div id="season-countdown-content">
                <p>Loading season countdown...</p>
            </div>
            <div class="last-updated">
                Countdown updates automatically every hour
            </div>
        </div>

        <div class="mammoth-analysis-section offseason-only">
            <h2>📊 Team Analysis</h2>
            <div id="mammoth-analysis-content">
                ` + analysisContent + `
            </div>
            <div class="last-updated">
                Analysis automatically updates every 30 minutes
            </div>
        </div>
    </div>
    
    <!-- Enhanced Game Banner at Bottom -->
    <div class="bottom-banner">
        <div class="banner-content" id="banner-content">
            <p>Loading next game information...</p>
        </div>
    </div>
    
    <script>
        let currentSeasonStatus = null;
        
        function loadBanner() {
            console.log('Loading banner content...'); // Debug log
            htmx.ajax('GET', '/banner', '#banner-content', {
                afterRequest: function(xhr) {
                    if (xhr.status === 200) {
                        console.log('Banner updated successfully');
                    } else {
                        console.error('Error updating banner:', xhr.status);
                    }
                }
            });
        }

        function loadUpcomingGames() {
            const upcomingGamesContent = document.getElementById('upcoming-games-content');
            const lastUpdated = document.querySelector('.upcoming-games-section .last-updated');
            
            // Show loading state
            if (lastUpdated) {
                lastUpdated.innerHTML = '<span class="news-loading">Updating schedule...</span>';
            }
            
            htmx.ajax('GET', '/upcoming-games', '#upcoming-games-content', {
                afterRequest: function(xhr) {
                    if (lastUpdated) {
                        if (xhr.status === 200) {
                            const now = new Date();
                            const timeString = now.toLocaleTimeString();
                            lastUpdated.innerHTML = '<span class="news-updated">Schedule updated at ' + timeString + ' (auto-updates daily at midnight)</span>';
                        } else {
                            lastUpdated.innerHTML = '<span class="news-error">Error updating schedule</span>';
                        }
                    }
                }
            });
        }

        // Season status management
        function loadSeasonStatus() {
            return fetch('/season-status')
                .then(response => response.json())
                .then(data => {
                    currentSeasonStatus = data.seasonStatus;
                    updateSectionVisibility();
                    return currentSeasonStatus;
                })
                .catch(error => {
                    console.error('Error loading season status:', error);
                    // Default to showing all sections if we can't determine season status
                    currentSeasonStatus = { isHockeySeason: true, seasonPhase: 'unknown' };
                    updateSectionVisibility();
                });
        }

        function updateSectionVisibility() {
            if (!currentSeasonStatus) return;

            const body = document.body;
            
            // Remove existing season classes
            body.classList.remove('hockey-season', 'offseason', 'regular-season');
            
            if (currentSeasonStatus.isHockeySeason) {
                // Hockey season: Add hockey-season class
                body.classList.add('hockey-season');
                
                // Check if it's specifically the regular season
                if (currentSeasonStatus.seasonPhase === 'regular') {
                    body.classList.add('regular-season');
                    console.log('Regular season active - showing news, scoreboard, upcoming games, and playoff odds');
                } else {
                    console.log('Hockey season active (' + currentSeasonStatus.seasonPhase + ') - showing news, scoreboard, upcoming games, and playoff odds');
                }
            } else {
                // Off-season: Add offseason class
                body.classList.add('offseason');
                console.log('Off-season active - showing news, countdown, and analysis');
            }

            // Update grid layout based on visible sections
            updateGridLayout();
        }

        function updateGridLayout() {
            const mainLayout = document.querySelector('.main-layout');
            if (!mainLayout || !currentSeasonStatus) return;

            if (currentSeasonStatus.isHockeySeason) {
                // Hockey season: upcoming games in column 1, player stats in column 2, playoff odds in column 3 (news hidden)
                mainLayout.style.gridTemplateColumns = '1fr 1fr 1fr';
            } else {
                // Off-season: Three columns for news, countdown, and analysis
                mainLayout.style.gridTemplateColumns = '1fr 1fr 1fr';
            }
        }

        function loadPlayerStats() {
            const playerStatsContent = document.getElementById('player-stats-content');
            const lastUpdated = document.querySelector('.player-stats-section .last-updated');
            
            // Show loading state
            if (lastUpdated) {
                lastUpdated.innerHTML = '<span class="news-loading">Updating player stats...</span>';
            }
            
            htmx.ajax('GET', '/player-stats', '#player-stats-content', {
                afterRequest: function(xhr) {
                    if (lastUpdated) {
                        if (xhr.status === 200) {
                            const now = new Date();
                            const timeString = now.toLocaleTimeString();
                            lastUpdated.innerHTML = '<span class="news-updated">Player stats updated at ' + timeString + ' (auto-updates every 30 minutes)</span>';
                        } else {
                            lastUpdated.innerHTML = '<span class="news-error">Error updating player stats</span>';
                        }
                    }
                }
            });
        }

        function loadGoalieStats() {
            const goalieStatsContent = document.getElementById('goalie-stats-content');
            
            htmx.ajax('GET', '/goalie-stats', '#goalie-stats-content');
        }

        function toggleSeasonStatus() {
            if (!currentSeasonStatus) {
                console.log('No season status available to toggle');
                return;
            }
            
            // Toggle the hockey season status
            currentSeasonStatus.isHockeySeason = !currentSeasonStatus.isHockeySeason;
            
            // Update season phase based on new status
            if (currentSeasonStatus.isHockeySeason) {
                currentSeasonStatus.seasonPhase = 'regular'; // Default to regular season
                console.log('🎥 Toggled to HOCKEY SEASON (regular)');
            } else {
                currentSeasonStatus.seasonPhase = 'offseason';
                console.log('🎥 Toggled to OFF-SEASON');
            }
            
            // Update the visibility classes and grid layout
            updateSectionVisibility();
            
            // Visual feedback - briefly highlight the clicked icon
            const cameraIcon = document.getElementById('camera-icon');
            const clockIcon = document.getElementById('clock-icon');
            const clickedIcon = cameraIcon || clockIcon;
            if (clickedIcon) {
                clickedIcon.style.transform = 'scale(1.2)';
                clickedIcon.style.transition = 'transform 0.2s ease';
                setTimeout(() => {
                    clickedIcon.style.transform = 'scale(1)';
                }, 200);
            }
        }

        function loadPlayoffOdds() {
            const playoffOddsContent = document.getElementById('playoff-odds-content');
            const lastUpdated = document.querySelector('.playoff-odds-section .last-updated');
            
            // Show loading state
            if (lastUpdated) {
                lastUpdated.innerHTML = '<span class="news-loading">Calculating playoff odds...</span>';
            }
            
            htmx.ajax('GET', '/playoff-odds', '#playoff-odds-content', {
                afterRequest: function(xhr) {
                    if (lastUpdated) {
                        if (xhr.status === 200) {
                            const now = new Date();
                            const timeString = now.toLocaleTimeString();
                            lastUpdated.innerHTML = '<span class="news-updated">Playoff odds updated at ' + timeString + '</span>';
                        } else {
                            lastUpdated.innerHTML = '<span class="news-error">Error calculating playoff odds</span>';
                        }
                    }
                }
            });
        }

        function loadNews() {
            const newsContent = document.getElementById('news-content');
            const lastUpdated = document.querySelector('.news-section .last-updated');
            
            // Show loading state
            if (lastUpdated) {
                lastUpdated.innerHTML = '<span class="news-loading">Updating news...</span>';
            }
            
            htmx.ajax('GET', '/news', '#news-content', {
                afterRequest: function(xhr) {
                    if (lastUpdated) {
                        if (xhr.status === 200) {
                            const now = new Date();
                            const timeString = now.toLocaleTimeString();
                            lastUpdated.innerHTML = '<span class="news-updated">News updated at ' + timeString + ' (auto-updates every 10 minutes)</span>';
                        } else {
                            lastUpdated.innerHTML = '<span class="news-error">Error updating news</span>';
                        }
                    }
                }
            });
        }
        
        function loadScoreboard() {
            const scoreboardSection = document.querySelector('.scoreboard-section');
            const scoreboardContent = document.getElementById('scoreboard-content');
            
            if (!scoreboardSection || !scoreboardContent) return;
            
            // Show loading state
            scoreboardContent.innerHTML = '<p>Loading live scoreboard...</p>';
            scoreboardSection.style.display = 'block';
            
            // Fetch scoreboard data
            fetch('/scoreboard')
                .then(response => response.text())
                .then(html => {
                    // Check if there's no active game OR if game is not live (upcoming/future games)
                    if (html.includes('No active game at the moment') || 
                        html.includes('No active game') ||
                        html.includes('⏱️ UPCOMING') ||
                        html.includes('✅ FINAL')) {
                        // Hide the entire scoreboard section for non-live games
                        scoreboardSection.style.display = 'none';
                        console.log('No live game - hiding scoreboard section');
                    } else if (html.includes('🔴 LIVE') || html.includes('⏸️ INTERMISSION')) {
                        // Show the section only for live games and intermissions
                        scoreboardSection.style.display = 'block';
                        scoreboardContent.innerHTML = html;
                        console.log('Live game found - showing scoreboard');
                    } else {
                        // Default case - hide if unsure
                        scoreboardSection.style.display = 'none';
                        console.log('Unknown game state - hiding scoreboard section');
                    }
                })
                .catch(error => {
                    console.error('Error loading scoreboard:', error);
                    // Hide section on error
                    scoreboardSection.style.display = 'none';
                });
        }
        
        function loadSeasonCountdown() {
            const countdownContent = document.getElementById('season-countdown-content');
            const lastUpdated = document.querySelector('.season-countdown-section .last-updated');
            
            // Show loading state
            if (lastUpdated) {
                lastUpdated.innerHTML = '<span class="news-loading">Updating countdown...</span>';
            }
            
            fetch('/season-countdown')
                .then(response => {
                    if (response.ok) {
                        return response.text();
                    }
                    throw new Error('Network response was not ok');
                })
                .then(html => {
                    if (countdownContent) {
                        countdownContent.innerHTML = html;
                        // Start live countdown updates
                        updateLiveCountdowns();
                    }
                    if (lastUpdated) {
                        const now = new Date();
                        const timeString = now.toLocaleTimeString();
                        lastUpdated.innerHTML = '<span class="news-updated">Countdown updated at ' + timeString + ' (auto-updates every hour)</span>';
                    }
                })
                .catch(error => {
                    console.error('Error loading countdown:', error);
                    if (lastUpdated) {
                        lastUpdated.innerHTML = '<span class="news-error">Error updating countdown</span>';
                    }
                });
        }

        function updateLiveCountdowns() {
            const counters = document.querySelectorAll('.days-counter, .uta-days-counter');
            
            counters.forEach(counter => {
                const targetDate = counter.getAttribute('data-target-date');
                if (targetDate) {
                    const target = new Date(targetDate);
                    const now = new Date();
                    const timeDiff = target - now;
                    
                    if (timeDiff > 0) {
                        const days = Math.floor(timeDiff / (1000 * 60 * 60 * 24));
                        const hoursLeft = Math.floor((timeDiff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
                        
                        const numberElement = counter.querySelector('.days-number, .uta-days-number');
                        const labelElement = counter.querySelector('.days-label, .uta-days-label');
                        
                        if (numberElement) {
                            numberElement.textContent = days;
                        }
                        
                        if (labelElement && days === 0) {
                            if (hoursLeft > 0) {
                                labelElement.textContent = hoursLeft + ' Hours Until Game';
                            } else {
                                labelElement.textContent = 'Game Day!';
                            }
                        }
                    }
                }
            });
        }

        function loadTeamAnalysis() {
            const analysisContent = document.getElementById('mammoth-analysis-content');
            const lastUpdated = document.querySelector('.mammoth-analysis-section .last-updated');
            
            // Show loading state
            if (lastUpdated) {
                lastUpdated.innerHTML = '<span class="news-loading">Updating analysis...</span>';
            }
            
            fetch('/mammoth-analysis')
                .then(response => {
                    if (response.ok) {
                        return response.text();
                    }
                    throw new Error('Network response was not ok');
                })
                .then(html => {
                    if (analysisContent) {
                        analysisContent.innerHTML = html;
                    }
                    if (lastUpdated) {
                        const now = new Date();
                        const timeString = now.toLocaleTimeString();
                        lastUpdated.innerHTML = '<span class="news-updated">Analysis updated at ' + timeString + ' (auto-updates every 30 minutes)</span>';
                    }
                })
                .catch(error => {
                    console.error('Error loading analysis:', error);
                    if (lastUpdated) {
                        lastUpdated.innerHTML = '<span class="news-error">Error updating analysis</span>';
                    }
                });
        }
        
        // Initialize the page
        document.addEventListener('DOMContentLoaded', function() {
            // Load news immediately (always available)
            loadNews();
            
            // Then load season status to determine what else to show
            loadSeasonStatus().then(() => {
                // Load banner content (always shown)
                loadBanner();
                
                // Set up banner auto-update (every hour)
                setInterval(loadBanner, 3600000); // 60 minutes
                
                // Conditionally load content based on season status
                if (currentSeasonStatus && currentSeasonStatus.isHockeySeason) {
                    // Hockey season: Load scoreboard, upcoming games, player stats, and playoff odds
                    loadScoreboard();
                    loadUpcomingGames();
                    loadPlayerStats();
                    loadGoalieStats();
                    loadPlayoffOdds();
                    
                    // Set up automatic updates
                    setInterval(loadScoreboard, 30000); // Check every 30 seconds
                    setInterval(loadUpcomingGames, 3600000); // Update upcoming games every hour
                    setInterval(loadPlayerStats, 1800000); // Update player stats every 30 minutes
                    setInterval(loadGoalieStats, 1800000); // Update goalie stats every 30 minutes
                    setInterval(loadPlayoffOdds, 1800000); // Update playoff odds every 30 minutes
                } else {
                    // Offseason: Load news, countdown, and mammoth analysis
                    loadSeasonCountdown();
                    loadTeamAnalysis();
                    setInterval(loadNews, 600000); // 10 minutes
                    setInterval(loadSeasonCountdown, 3600000); // 60 minutes
                    setInterval(loadTeamAnalysis, 1800000); // 30 minutes
                    
                    // Update live countdown every minute
                    setInterval(updateLiveCountdowns, 60000);
                }
            });
        });
        
        // Schedule a reload of the analysis section at the next local midnight (off-season only)
        function scheduleMidnightAnalysisReload() {
            const now = new Date();
            const nextMidnight = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1, 0, 0, 5, 0); // 5 seconds after midnight
            const msUntilMidnight = nextMidnight - now;
            setTimeout(function() {
                // Only reload analysis if it's off-season
                if (currentSeasonStatus && !currentSeasonStatus.isHockeySeason) {
                    loadTeamAnalysis();
                }
                // Schedule the next midnight reload
                scheduleMidnightAnalysisReload();
            }, msUntilMidnight);
        }
        
        // Start the midnight reload scheduler after season status is loaded
        loadSeasonStatus().then(() => {
            // Only start midnight scheduler during off-season
            if (currentSeasonStatus && !currentSeasonStatus.isHockeySeason) {
                scheduleMidnightAnalysisReload();
            }
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// getAnalysisContent calls the analysis handler internally and returns the HTML content
func getAnalysisContent() string {
	// Create a mock request for the analysis handler
	req := httptest.NewRequest("GET", "/mammoth-analysis", nil)
	w := httptest.NewRecorder()

	// Call the analysis handler
	HandleTeamAnalysis(w, req)

	// Return the response body
	result := w.Result()
	if result.StatusCode == 200 {
		body := w.Body.String()
		return body
	}

	// Return fallback if there's an error
	return "<p>Unable to load analysis data at this time.</p>"
}

// generateTeamCSS creates dynamic CSS variables based on team colors
func generateTeamCSS() string {
	if teamConfig == nil {
		return ""
	}

	return fmt.Sprintf(`
		:root {
			--team-primary: %s;
			--team-secondary: %s;
			--team-primary-light: %s80;
			--team-primary-dark: %s;
			--team-accent: %s;
		}
	`, teamConfig.PrimaryColor, teamConfig.SecondaryColor,
		teamConfig.PrimaryColor, darkenColor(teamConfig.PrimaryColor),
		lightenColor(teamConfig.PrimaryColor))
}

// darkenColor takes a hex color and returns a darker version
func darkenColor(hex string) string {
	// Simple darkening by reducing the hex values
	if len(hex) != 7 || hex[0] != '#' {
		return hex
	}

	// Convert hex to RGB, darken by 20%, convert back
	r, g, b := hexToRGB(hex)
	r = int(float64(r) * 0.8)
	g = int(float64(g) * 0.8)
	b = int(float64(b) * 0.8)

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// lightenColor takes a hex color and returns a lighter version
func lightenColor(hex string) string {
	if len(hex) != 7 || hex[0] != '#' {
		return hex
	}

	r, g, b := hexToRGB(hex)
	r = int(float64(r) + (255-float64(r))*0.3)
	g = int(float64(g) + (255-float64(g))*0.3)
	b = int(float64(b) + (255-float64(b))*0.3)

	if r > 255 {
		r = 255
	}
	if g > 255 {
		g = 255
	}
	if b > 255 {
		b = 255
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// hexToRGB converts hex color to RGB values
func hexToRGB(hex string) (int, int, int) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0
	}

	r, _ := strconv.ParseInt(hex[1:3], 16, 0)
	g, _ := strconv.ParseInt(hex[3:5], 16, 0)
	b, _ := strconv.ParseInt(hex[5:7], 16, 0)

	return int(r), int(g), int(b)
}
