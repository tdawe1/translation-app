package watcher

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/gofiber/websocket/v2"
)

// WebSocketMonitor monitors Gengo WebSocket for new jobs
type WebSocketMonitor struct {
	SessionToken string
	UserID       uuid.UUID
	Conn         *websocket.Conn
}

// NewWebSocketMonitor creates a new WebSocket monitor
func NewWebSocketMonitor(sessionToken string, userID uuid.UUID) *WebSocketMonitor {
	return &WebSocketMonitor{
		SessionToken: sessionToken,
		UserID:       userID,
	}
}

// Start begins monitoring the WebSocket connection
func (m *WebSocketMonitor) Start(ctx context.Context, jobChan chan<- Job) error {
	// TODO: Implement WebSocket connection to Gengo
	// This would:
	// 1. Connect to Gengo's WebSocket endpoint
	// 2. Authenticate with the session token
	// 3. Listen for job notifications
	// 4. Parse job data and send to jobChan

	log.Printf("WebSocket monitor started for user %s", m.UserID)

	// Keep running until context is cancelled
	<-ctx.Done()

	log.Printf("WebSocket monitor stopped for user %s", m.UserID)
	return nil
}

// Connect establishes the WebSocket connection
func (m *WebSocketMonitor) Connect() error {
	// TODO: Implement WebSocket connection logic
	// Use the session token to authenticate with Gengo
	return nil
}

// Disconnect closes the WebSocket connection
func (m *WebSocketMonitor) Disconnect() error {
	if m.Conn != nil {
		return m.Conn.Close()
	}
	return nil
}
