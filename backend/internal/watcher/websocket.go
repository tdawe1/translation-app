package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Gengo WebSocket URL
	gengoWebSocketURL = "wss://live-dashboard.gengo.com"

	// Default heartbeat interval for regular users
	defaultUserHeartbeatInterval = 10 * time.Second

	// Default heartbeat interval for admin users (fastest detection)
	defaultAdminHeartbeatInterval = 1 * time.Second

	// Default pong timeout for regular users (2x heartbeat)
	defaultUserPongWait = 20 * time.Second

	// Default pong timeout for admin users (2x heartbeat)
	defaultAdminPongWait = 2 * time.Second

	// Initial reconnection delay on failure
	initialReconnectDelay = 1 * time.Second
)

// WebSocketMonitor monitors Gengo's WebSocket for new jobs
type WebSocketMonitor struct {
	UserID      uuid.UUID
	UserSession string
	UserKey     string
	GengoUserID string // External Gengo user ID
	// P0-2 FIX: Use LRU cache instead of unbounded map to prevent memory leaks
	seenIDs *LRUCache
	mu      sync.RWMutex

	// Timing configuration
	HeartbeatInterval time.Duration
	PongWait          time.Duration

	// Connection state
	conn     *websocket.Conn
	status   string
	statusMu sync.RWMutex

	// Metrics
	lastPongTime time.Time
	lastPingTime time.Time
	pingLatency  time.Duration
	pingCount    int64
}

// NewWebSocketMonitor creates a new WebSocket monitor
// isAdmin determines heartbeat interval (1s for admin, 10s for regular users)
func NewWebSocketMonitor(userID uuid.UUID, userSession, userKey, gengoUserID string, isAdmin bool) *WebSocketMonitor {
	heartbeatInterval := defaultUserHeartbeatInterval
	pongWait := defaultUserPongWait

	if isAdmin {
		heartbeatInterval = defaultAdminHeartbeatInterval
		pongWait = defaultAdminPongWait
	}

	return &WebSocketMonitor{
		UserID:            userID,
		UserSession:       userSession,
		UserKey:           userKey,
		GengoUserID:       gengoUserID,
		// P0-2 FIX: Use LRU cache with 1000 item limit to prevent unbounded memory growth
		seenIDs:           NewLRUCache(1000),
		status:            "disconnected",
		HeartbeatInterval: heartbeatInterval,
		PongWait:          pongWait,
	}
}

// wsAuthPayload represents the authentication payload sent to Gengo
type wsAuthPayload struct {
	Action      string `json:"action"`
	UserSession string `json:"user_session"`
	UserKey     string `json:"user_key"`
	UserID      string `json:"user_id"`
}

// wsMessage represents a message received from Gengo
type wsMessage struct {
	Type  string                 `json:"type"`
	JobID string                 `json:"job_id,omitempty"`
	Other map[string]interface{} `json:"-"`
}

// UnmarshalJSON handles unmarshaling with unknown fields
func (m *wsMessage) UnmarshalJSON(data []byte) error {
	// First, try to unmarshal into a map to capture all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract known fields
	if typ, ok := raw["type"].(string); ok {
		m.Type = typ
	}
	if jobID, ok := raw["job_id"].(string); ok {
		m.JobID = jobID
	}

	// Store unknown fields
	m.Other = make(map[string]interface{})
	for k, v := range raw {
		if k != "type" && k != "job_id" {
			m.Other[k] = v
		}
	}
	return nil
}

// Start begins monitoring the WebSocket connection
func (m *WebSocketMonitor) Start(ctx context.Context, jobChan chan<- Job) {
	// Check if WebSocket is properly configured
	if m.UserSession == "" || m.UserKey == "" {
		log.Printf("[WS] User %s: WebSocket not configured (missing session token or user key)", m.UserID)
		log.Printf("[WS] User %s: WebSocket monitor disabled (requires Gengo credentials)", m.UserID)
		return
	}

	log.Printf("[WS] Starting monitor for user %s (gengo_id=%s, heartbeat=%v, pong_wait=%v)",
		m.UserID, m.GengoUserID, m.HeartbeatInterval, m.PongWait)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[WS] Monitor stopped for user %s", m.UserID)
			return
		default:
			if err := m.connectAndMonitor(ctx, jobChan); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[WS] Connection error for user %s: %v (reconnecting in %v)", m.UserID, err, initialReconnectDelay)
				m.setStatus("reconnecting")
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(initialReconnectDelay):
		}
	}
}

// connectAndMonitor establishes a connection and monitors for jobs
func (m *WebSocketMonitor) connectAndMonitor(ctx context.Context, jobChan chan<- Job) error {
	m.setStatus("connecting")
	log.Printf("[WS] Connecting to %s for user %s", gengoWebSocketURL, m.UserID)

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, gengoWebSocketURL, nil)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	m.conn = conn
	log.Printf("[WS] Connection established for user %s", m.UserID)

	conn.SetPongHandler(func(appData string) error {
		m.mu.Lock()
		defer m.mu.Unlock()
		now := time.Now()
		// Calculate round-trip time from last ping
		if !m.lastPingTime.IsZero() {
			rtt := now.Sub(m.lastPingTime)
			m.pingLatency = rtt
			// Log pong with round-trip time
			log.Printf("[WS] Pong received for user %s (RTT: %v)", m.UserID, rtt)
		}
		m.lastPongTime = now
		return nil
	})

	if err := m.authenticate(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	m.setStatus("live")
	log.Printf("[WS] Connected and authenticated for user %s", m.UserID)

	heartbeatTicker := time.NewTicker(m.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	conn.SetReadDeadline(time.Now().Add(m.PongWait))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-heartbeatTicker.C:
			if err := m.sendPing(); err != nil {
				return fmt.Errorf("ping failed: %w", err)
			}
			conn.SetReadDeadline(time.Now().Add(m.PongWait))

		default:
			conn.SetReadDeadline(time.Now().Add(m.HeartbeatInterval + 5*time.Second))
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					return fmt.Errorf("connection closed: %w", err)
				}
				// Check for timeout (net.Error with Timeout() == true)
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					continue
				}
				return fmt.Errorf("read failed: %w", err)
			}

			if err := m.processMessage(message, jobChan); err != nil {
				log.Printf("[WS] Error processing message for user %s: %v", m.UserID, err)
			}
		}
	}
}

// authenticate sends the authentication payload
func (m *WebSocketMonitor) authenticate(ctx context.Context) error {
	m.setStatus("authenticating")

	authPayload := wsAuthPayload{
		Action:      "authenticate",
		UserSession: m.UserSession,
		UserKey:     m.UserKey,
		UserID:      m.GengoUserID,
	}

	data, err := json.Marshal(authPayload)
	if err != nil {
		return fmt.Errorf("marshal auth payload: %w", err)
	}

	if err := m.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("send auth: %w", err)
	}

	log.Printf("[WS] Sent authentication for user %s (gengo_id=%s)", m.UserID, m.GengoUserID)
	return nil
}

// sendPing sends a heartbeat ping
func (m *WebSocketMonitor) sendPing() error {
	start := time.Now()
	if err := m.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return err
	}

	m.mu.Lock()
	m.lastPingTime = start
	m.pingCount++
	count := m.pingCount
	m.mu.Unlock()

	// Log every 10th ping to reduce verbosity, or if latency is high
	if count%10 == 0 {
		log.Printf("[WS] Ping #%d sent for user %s", count, m.UserID)
	}
	return nil
}

// processMessage handles an incoming WebSocket message
func (m *WebSocketMonitor) processMessage(data []byte, jobChan chan<- Job) error {
	var msg wsMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil
	}

	log.Printf("[WS] Received message for user %s: type=%s", m.UserID, msg.Type)

	switch msg.Type {
	case "available_collection":
		return m.handleJobAvailable(msg, jobChan)

	default:
		log.Printf("[WS] Ignoring message type '%s' for user %s", msg.Type, m.UserID)
	}

	return nil
}

// handleJobAvailable processes a new job notification
func (m *WebSocketMonitor) handleJobAvailable(msg wsMessage, jobChan chan<- Job) error {
	jobID := msg.JobID
	if jobID == "" {
		return fmt.Errorf("job_id missing in available_collection message")
	}

	// P0-2 FIX: Use LRU cache.Add which returns true if job was already seen
	if m.seenIDs.Add(jobID) {
		log.Printf("[WS] User %s: Job %s already seen, skipping", m.UserID, jobID)
		return nil
	}

	job := Job{
		ID:     jobID,
		Title:  fmt.Sprintf("Job %s", jobID),
		Reward: 0,
		URL:    fmt.Sprintf("https://gengo.com/dashboard/jobs/%s", jobID),
		Source: "websocket",
		UserID: m.UserID,
	}

	select {
	case jobChan <- job:
		log.Printf("[WS] User %s: New job from WebSocket - %s", m.UserID, jobID)
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending to job channel")
	}

	return nil
}

// GetStatus returns the current connection status
func (m *WebSocketMonitor) GetStatus() string {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()
	return m.status
}

// setStatus updates the connection status
func (m *WebSocketMonitor) setStatus(status string) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.status = status
}

// GetPingLatency returns the last measured ping latency
func (m *WebSocketMonitor) GetPingLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pingLatency
}

// Stop closes the WebSocket connection
func (m *WebSocketMonitor) Stop() {
	m.setStatus("stopped")
	if m.conn != nil {
		m.conn.Close()
	}
}
