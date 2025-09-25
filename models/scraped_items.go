package models

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"time"
)

// NewsItem represents a scraped news article
type NewsItem struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	URL       string                 `json:"url"`
	Date      string                 `json:"date"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// GetID returns the unique identifier for the news item
func (ni *NewsItem) GetID() string {
	if ni.ID == "" {
		// Generate ID from URL hash if not set
		hash := md5.Sum([]byte(ni.URL))
		ni.ID = fmt.Sprintf("news_%x", hash)
	}
	return ni.ID
}

// GetTitle returns the news article title
func (ni *NewsItem) GetTitle() string {
	return ni.Title
}

// GetURL returns the news article URL
func (ni *NewsItem) GetURL() string {
	return ni.URL
}

// GetTimestamp returns when the item was scraped
func (ni *NewsItem) GetTimestamp() time.Time {
	return ni.Timestamp
}

// GetMetadata returns additional metadata for the news item
func (ni *NewsItem) GetMetadata() map[string]interface{} {
	if ni.Metadata == nil {
		ni.Metadata = make(map[string]interface{})
	}
	ni.Metadata["date"] = ni.Date
	ni.Metadata["type"] = "news"
	return ni.Metadata
}

// Equals compares this news item with another for changes
func (ni *NewsItem) Equals(other ScrapedItem) bool {
	if otherNews, ok := other.(*NewsItem); ok {
		return ni.Title == otherNews.Title &&
			ni.URL == otherNews.URL &&
			ni.Date == otherNews.Date
	}
	return false
}

// ProductItem represents a scraped product/merchandise item
type ProductItem struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	URL         string                 `json:"url"`
	Price       float64                `json:"price"`
	Currency    string                 `json:"currency"`
	ImageURL    string                 `json:"image_url"`
	Available   bool                   `json:"available"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Brand       string                 `json:"brand"`
	TeamCode    string                 `json:"team_code"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GetID returns the unique identifier for the product item
func (pi *ProductItem) GetID() string {
	if pi.ID == "" {
		// Generate ID from URL hash if not set
		hash := md5.Sum([]byte(pi.URL))
		pi.ID = fmt.Sprintf("product_%x", hash)
	}
	return pi.ID
}

// GetTitle returns the product title
func (pi *ProductItem) GetTitle() string {
	return pi.Title
}

// GetURL returns the product URL
func (pi *ProductItem) GetURL() string {
	return pi.URL
}

// GetTimestamp returns when the item was scraped
func (pi *ProductItem) GetTimestamp() time.Time {
	return pi.Timestamp
}

// GetMetadata returns additional metadata for the product item
func (pi *ProductItem) GetMetadata() map[string]interface{} {
	if pi.Metadata == nil {
		pi.Metadata = make(map[string]interface{})
	}
	pi.Metadata["price"] = pi.Price
	pi.Metadata["currency"] = pi.Currency
	pi.Metadata["available"] = pi.Available
	pi.Metadata["category"] = pi.Category
	pi.Metadata["brand"] = pi.Brand
	pi.Metadata["team_code"] = pi.TeamCode
	pi.Metadata["image_url"] = pi.ImageURL
	pi.Metadata["type"] = "product"
	return pi.Metadata
}

// Equals compares this product item with another for changes
func (pi *ProductItem) Equals(other ScrapedItem) bool {
	if otherProduct, ok := other.(*ProductItem); ok {
		return pi.Title == otherProduct.Title &&
			pi.URL == otherProduct.URL &&
			pi.Price == otherProduct.Price &&
			pi.Available == otherProduct.Available &&
			pi.Description == otherProduct.Description
	}
	return false
}

// GetPriceString returns a formatted price string
func (pi *ProductItem) GetPriceString() string {
	if pi.Currency == "" {
		pi.Currency = "USD"
	}
	return fmt.Sprintf("%.2f %s", pi.Price, pi.Currency)
}

// IsNew checks if this is a newly detected item (within last 24 hours)
func (pi *ProductItem) IsNew() bool {
	return time.Since(pi.Timestamp) < 24*time.Hour
}

// GenericItem represents any other type of scraped item
type GenericItem struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	URL       string                 `json:"url"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// GetID returns the unique identifier for the generic item
func (gi *GenericItem) GetID() string {
	if gi.ID == "" {
		// Generate ID from URL and type hash if not set
		hash := md5.Sum([]byte(gi.Type + "_" + gi.URL))
		gi.ID = fmt.Sprintf("generic_%x", hash)
	}
	return gi.ID
}

// GetTitle returns the generic item title
func (gi *GenericItem) GetTitle() string {
	return gi.Title
}

// GetURL returns the generic item URL
func (gi *GenericItem) GetURL() string {
	return gi.URL
}

// GetTimestamp returns when the item was scraped
func (gi *GenericItem) GetTimestamp() time.Time {
	return gi.Timestamp
}

// GetMetadata returns additional metadata for the generic item
func (gi *GenericItem) GetMetadata() map[string]interface{} {
	if gi.Metadata == nil {
		gi.Metadata = make(map[string]interface{})
	}
	gi.Metadata["type"] = gi.Type
	return gi.Metadata
}

// Equals compares this generic item with another for changes
func (gi *GenericItem) Equals(other ScrapedItem) bool {
	if otherGeneric, ok := other.(*GenericItem); ok {
		return gi.Title == otherGeneric.Title &&
			gi.URL == otherGeneric.URL &&
			gi.Type == otherGeneric.Type
	}
	return false
}

// Helper function to parse price from string
func ParsePrice(priceStr string) (float64, error) {
	// Remove common price formatting characters
	cleaned := priceStr
	for _, char := range []string{"$", "€", "£", "¥", ",", " "} {
		cleaned = cleanString(cleaned, char)
	}

	return strconv.ParseFloat(cleaned, 64)
}

// Helper function to clean strings
func cleanString(str, remove string) string {
	result := ""
	for _, char := range str {
		if string(char) != remove {
			result += string(char)
		}
	}
	return result
}
