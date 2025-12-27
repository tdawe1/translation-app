package watcher

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// RSSFeed represents an RSS feed
type RSSFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		Items       []RSSItem `xml:"item"`
	} `xml:"channel"`
}

// RSSItem represents an item in the RSS feed
type RSSItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	PubDate     string  `xml:"pubDate"`
	GUID        string  `xml:"guid"`
}

// RSSMonitor monitors an RSS feed for new jobs
type RSSMonitor struct {
	FeedURL   string
	UserID    uuid.UUID
	MinReward float64
	seenIDs   map[string]bool
}

// NewRSSMonitor creates a new RSS monitor
func NewRSSMonitor(feedURL string, userID uuid.UUID, minReward float64) *RSSMonitor {
	return &RSSMonitor{
		FeedURL:   feedURL,
		UserID:    userID,
		MinReward: minReward,
		seenIDs:   make(map[string]bool),
	}
}

// Start begins monitoring the RSS feed
func (m *RSSMonitor) Start(ctx context.Context, jobChan chan<- Job) error {
	ticker := time.NewTicker(30 * time.Second) // Poll every 30 seconds
	defer ticker.Stop()

	// Initial fetch
	if err := m.fetch(jobChan); err != nil {
		log.Printf("RSS initial fetch error for user %s: %v", m.UserID, err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("RSS monitor stopped for user %s", m.UserID)
			return nil
		case <-ticker.C:
			if err := m.fetch(jobChan); err != nil {
				log.Printf("RSS fetch error for user %s: %v", m.UserID, err)
			}
		}
	}
}

// fetch fetches and parses the RSS feed
func (m *RSSMonitor) fetch(jobChan chan<- Job) error {
	// In a real implementation, you'd use http.Get to fetch the feed
	// For now, this is a stub that simulates the structure
	// TODO: Implement actual HTTP fetch

	// Simulated fetch for now
	// In production, use: resp, err := http.Get(m.FeedURL)

	return nil
}

// extractReward extracts reward from an RSS item
func (m *RSSMonitor) extractReward(item RSSItem) float64 {
	// TODO: Implement reward extraction logic
	// This would parse the title/description for reward values
	// For Gengo, it might be in the title like "Job $5.00"
	return 0.0
}

// extractJobID extracts a unique job ID from an RSS item
func (m *RSSMonitor) extractJobID(item RSSItem) string {
	// Use GUID if available, otherwise use link
	if item.GUID != "" {
		return item.GUID
	}
	return item.Link
}
