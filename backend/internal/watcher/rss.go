package watcher

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
)

// Pre-compiled regex patterns for reward extraction (compiled once at package load)
var (
	// Pattern 1: $XX.XX or $XXX
	rewardDollarRegex = regexp.MustCompile(`\$(\d+\.?\d*)`)
	// Pattern 2: XX.XX USD or XX.XX dollars (case insensitive)
	rewardUSDRegex = regexp.MustCompile(`(?i)(\d+\.?\d*)\s*(?:USD|dollars?)`)
	// Pattern 3: USD XX.XX or Reward: XX.XX
	rewardPrefixRegex = regexp.MustCompile(`(?i)(?:USD|Reward|price)\s*[:=]?\s*\$?(\d+\.?\d*)`)
	// Pattern 4: Just a number followed by currency symbol
	rewardSymbolRegex = regexp.MustCompile(`(\d+\.?\d*)\s*[$€£¥]`)
	// Language pair pattern: ISO code patterns like "EN to JP", "EN→JP", "EN-JP"
	langPairRegex = regexp.MustCompile(`(?i)\b([a-z]{2})\s*(?:to|→|-)\s*([a-z]{2})\b`)
)

// RSSMonitor monitors an RSS feed for new jobs
type RSSMonitor struct {
	feedParser *gofeed.Parser
	FeedURL    string
	UserID     uuid.UUID
	MinReward  float64
	MaxReward  float64
	seenIDs    map[string]bool
}

// NewRSSMonitor creates a new RSS monitor
func NewRSSMonitor(feedURL string, userID uuid.UUID, minReward float64) *RSSMonitor {
	return &RSSMonitor{
		feedParser: &gofeed.Parser{},
		FeedURL:    feedURL,
		UserID:     userID,
		MinReward:  minReward,
		MaxReward:  999999, // Default max
		seenIDs:    make(map[string]bool),
	}
}

// SetMaxReward sets the maximum reward filter
func (m *RSSMonitor) SetMaxReward(max float64) {
	m.MaxReward = max
}

// GetFeedURL returns the RSS feed URL
func (m *RSSMonitor) GetFeedURL() string {
	return m.FeedURL
}

// GetUserID returns the user ID
func (m *RSSMonitor) GetUserID() uuid.UUID {
	return m.UserID
}

// GetMinReward returns the minimum reward filter
func (m *RSSMonitor) GetMinReward() float64 {
	return m.MinReward
}

// Start begins monitoring the RSS feed
func (m *RSSMonitor) Start(ctx context.Context, jobChan chan<- Job) error {
	ticker := time.NewTicker(30 * time.Second) // Poll every 30 seconds
	defer ticker.Stop()

	// Initial fetch
	if err := m.Fetch(ctx, jobChan); err != nil {
		log.Printf("[RSS] Initial fetch error for user %s: %v", m.UserID, err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("[RSS] Monitor stopped for user %s", m.UserID)
			return nil
		case <-ticker.C:
			if err := m.Fetch(ctx, jobChan); err != nil {
				log.Printf("[RSS] Fetch error for user %s: %v", m.UserID, err)
			}
		}
	}
}

// Fetch fetches and parses the RSS feed (exported for testing)
func (m *RSSMonitor) Fetch(ctx context.Context, jobChan chan<- Job) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// Fetch the feed
	resp, err := client.Get(m.FeedURL)
	if err != nil {
		return fmt.Errorf("HTTP fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// Parse the feed using gofeed
	feed, err := m.feedParser.Parse(resp.Body)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	// Process each item
	newJobs := 0
	for _, item := range feed.Items {
		jobID := m.extractJobID(item)

		// Skip if already seen
		if m.seenIDs[jobID] {
			continue
		}

		// Extract reward from the item
		reward := m.extractReward(item)

		// Filter by reward range
		if reward < m.MinReward || reward > m.MaxReward {
			log.Printf("[RSS] User %s: Job %s filtered by reward ($%.2f vs $%.2f-$%.2f)",
				m.UserID, jobID, reward, m.MinReward, m.MaxReward)
			m.seenIDs[jobID] = true
			continue
		}

		// Create job and send to channel
		job := Job{
			ID:     jobID,
			Title:  item.Title,
			Reward: reward,
			URL:    item.Link,
			Source: "rss",
			UserID: m.UserID,
		}

		select {
		case jobChan <- job:
			newJobs++
			log.Printf("[RSS] User %s: New job found - %s ($%.2f)", m.UserID, item.Title, reward)
		case <-ctx.Done():
			return ctx.Err()
		}

		m.seenIDs[jobID] = true
	}

	if newJobs > 0 {
		log.Printf("[RSS] User %s: %d new jobs found this poll", m.UserID, newJobs)
	}

	return nil
}

// extractReward extracts reward from an RSS item
// Gengo RSS feeds typically have reward info in the title or description
// Format examples: "Job $5.00 - English to Japanese", "Reward: 3.50 USD"
func (m *RSSMonitor) extractReward(item *gofeed.Item) float64 {
	// Try title first
	if reward := m.extractRewardFromString(item.Title); reward > 0 {
		return reward
	}

	// Try description
	if reward := m.extractRewardFromString(item.Description); reward > 0 {
		return reward
	}

	// Try content (some feeds use content instead of description)
	if item.Content != "" {
		if reward := m.extractRewardFromString(item.Content); reward > 0 {
			return reward
		}
	}

	// Try categories (sometimes reward is stored here)
	for _, category := range item.Categories {
		if reward := m.extractRewardFromString(category); reward > 0 {
			return reward
		}
	}

	return 0.0
}

// extractRewardFromString extracts a reward value from a string
// Supports formats: $5.00, 5.00 USD, USD 5.00, Reward: $5.00
// Uses pre-compiled regex patterns for better performance
func (m *RSSMonitor) extractRewardFromString(s string) float64 {
	// Pattern 1: $XX.XX or $XXX
	if matches := rewardDollarRegex.FindStringSubmatch(s); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}

	// Pattern 2: XX.XX USD or XX.XX dollars (case insensitive)
	if matches := rewardUSDRegex.FindStringSubmatch(s); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}

	// Pattern 3: USD XX.XX or Reward: XX.XX
	if matches := rewardPrefixRegex.FindStringSubmatch(s); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}

	// Pattern 4: Just a number followed by currency symbol
	if matches := rewardSymbolRegex.FindStringSubmatch(s); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}

	return 0.0
}

// extractJobID extracts a unique job ID from an RSS item
func (m *RSSMonitor) extractJobID(item *gofeed.Item) string {
	// Use GUID if available
	if item.GUID != "" {
		return item.GUID
	}

	// Use link if available
	if item.Link != "" {
		return item.Link
	}

	// Fall back to title + pubdate combination
	if !item.PublishedParsed.IsZero() {
		return fmt.Sprintf("%s-%d", item.Title, item.PublishedParsed.Unix())
	}

	// Last resort: use title
	return item.Title
}

// GengoJobInfo represents parsed Gengo job information
type GengoJobInfo struct {
	ID       string
	Title    string
	Reward   float64
	Source   string
	Link     string
	PubDate  time.Time
	LanguagePair string // Source -> Target
}

// ParseGengoJob parses a Gengo job from an RSS item
func (m *RSSMonitor) ParseGengoJob(item *gofeed.Item) (*GengoJobInfo, error) {
	jobID := m.extractJobID(item)
	reward := m.extractReward(item)

	pubDate := time.Now()
	if item.PublishedParsed != nil {
		pubDate = *item.PublishedParsed
	}

	// Try to extract language pair from title/description
	// Common format: "English to Japanese translation job"
	langPair := m.extractLanguagePair(item.Title, item.Description)

	return &GengoJobInfo{
		ID:        jobID,
		Title:     item.Title,
		Reward:    reward,
		Source:    "rss",
		Link:      item.Link,
		PubDate:   pubDate,
		LanguagePair: langPair,
	}, nil
}

// extractLanguagePair attempts to extract language pair from text
// Looks for patterns like "English to Japanese", "EN → JP", etc.
func (m *RSSMonitor) extractLanguagePair(title, description string) string {
	text := title + " " + description

	// Common language name patterns
	langPatterns := map[string]string{
		"English to Japanese":       "en → ja",
		"Japanese to English":      "ja → en",
		"English to Spanish":        "en → es",
		"Spanish to English":        "es → en",
		"English to French":         "en → fr",
		"French to English":         "fr → en",
		"English to German":         "en → de",
		"German to English":         "de → en",
		"English to Chinese":        "en → zh",
		"Chinese to English":        "zh → en",
		"English to Korean":         "en → ko",
		"Korean to English":         "ko → en",
		"English to Portuguese":     "en → pt",
		"Portuguese to English":     "pt → en",
		"English to Italian":        "en → it",
		"Italian to English":        "it → en",
		"English to Russian":        "en → ru",
		"Russian to English":        "ru → en",
	}

	// Check for full language names
	for pattern, code := range langPatterns {
		if strings.Contains(text, pattern) {
			return code
		}
	}

	// Check for ISO code patterns like "EN to JP", "EN→JP", "EN-JP"
	// Uses pre-compiled regex for better performance
	if matches := langPairRegex.FindStringSubmatch(text); len(matches) > 2 {
		return fmt.Sprintf("%s → %s", strings.ToLower(matches[1]), strings.ToLower(matches[2]))
	}

	return "unknown"
}
