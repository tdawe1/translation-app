package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// WebSocketHandler handles real-time WebSocket connections
type WebSocketHandler struct {
	redis *redis.Client
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(redisClient *redis.Client) *WebSocketHandler {
	return &WebSocketHandler{
		redis: redisClient,
	}
}

// HandleWebSocket upgrades HTTP to WebSocket and streams events
func (h *WebSocketHandler) HandleWebSocket() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		// Extract user from context (set by JWT middleware)
		userIDStr := c.Locals("user_id")
		if userIDStr == nil {
			log.Printf("[WS] Rejected connection: no user_id in context")
			c.Close()
			return
		}

		userID, ok := userIDStr.(uuid.UUID)
		if !ok {
			// Try to parse as string
			if strID, ok := userIDStr.(string); ok {
				var err error
				userID, err = uuid.Parse(strID)
				if err != nil {
					log.Printf("[WS] Rejected connection: invalid user_id format")
					c.Close()
					return
				}
			} else {
				log.Printf("[WS] Rejected connection: invalid user_id type")
				c.Close()
				return
			}
		}

		log.Printf("[WS] Client connected for user %s", userID)

		// Create context for this connection
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Subscribe to user's Redis channels
		pubsub := h.redis.Subscribe(ctx, h.getUserChannels(userID)...)
		defer pubsub.Close()

		// Channel for WebSocket messages
		wsChan := make(chan []byte, 100)
		errChan := make(chan error, 1)

		// Start Redis listener goroutine
		go h.listenForEvents(ctx, pubsub, wsChan, errChan)

		// Send initial state
		initialMsg := map[string]interface{}{
			"type":      "connected",
			"user_id":   userID.String(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		if data, err := json.Marshal(initialMsg); err == nil {
			c.WriteMessage(websocket.TextMessage, data)
		}

		// Message loop
		for {
			select {
			case <-ctx.Done():
				log.Printf("[WS] Context done for user %s", userID)
				return

			case err := <-errChan:
				log.Printf("[WS] Error for user %s: %v", userID, err)
				return

			case data := <-wsChan:
				if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Printf("[WS] Write error for user %s: %v", userID, err)
					return
				}
			}
		}
	})
}

// getUserChannels returns the Redis pub/sub channels for a user
func (h *WebSocketHandler) getUserChannels(userID uuid.UUID) []string {
	return []string{
		"user:" + userID.String() + ":jobs",
		"user:" + userID.String() + ":events",
		"user:" + userID.String() + ":errors",
	}
}

// listenForEvents listens for Redis pub/sub messages and forwards to WebSocket
func (h *WebSocketHandler) listenForEvents(
	ctx context.Context,
	pubsub *redis.PubSub,
	wsChan chan<- []byte,
	errChan chan<- error,
) {
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			errChan <- err
			return
		}

		// Forward the message payload to WebSocket
		wsChan <- []byte(msg.Payload)
	}
}

// PublishJob publishes a job notification to Redis
func (h *WebSocketHandler) PublishJob(ctx context.Context, userID uuid.UUID, job interface{}) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return h.redis.Publish(ctx, "user:"+userID.String()+":jobs", data).Err()
}

// PublishEvent publishes an event to Redis
func (h *WebSocketHandler) PublishEvent(ctx context.Context, userID uuid.UUID, eventType string, data interface{}) error {
	msg := map[string]interface{}{
		"type":      "event",
		"event":     eventType,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.redis.Publish(ctx, "user:"+userID.String()+":events", payload).Err()
}

// PublishError publishes an error to Redis
func (h *WebSocketHandler) PublishError(ctx context.Context, userID uuid.UUID, errMsg string) error {
	msg := map[string]interface{}{
		"type":      "error",
		"message":   errMsg,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.redis.Publish(ctx, "user:"+userID.String()+":errors", payload).Err()
}
