package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	apperrors "github.com/tdawe1/translation-app/internal/errors"
	"github.com/tdawe1/translation-app/internal/middleware"
)

const (
	// wsTicketPrefix is the Redis key prefix for WebSocket tickets
	wsTicketPrefix = "ws:ticket:"
	// wsTicketExpiry is the lifetime of a WebSocket ticket (30 seconds)
	wsTicketExpiry = 30 * time.Second
)

var (
	// ErrInvalidWSTicket is returned when the WebSocket ticket is invalid
	ErrInvalidWSTicket = errors.New("invalid websocket ticket")
	// ErrExpiredWSTicket is returned when the WebSocket ticket has expired
	ErrExpiredWSTicket = errors.New("expired websocket ticket")
	// ErrOriginNotAllowed is returned when the Origin header is not allowed
	ErrOriginNotAllowed = errors.New("origin not allowed")
)

// WebSocketHandler handles real-time WebSocket connections
type WebSocketHandler struct {
	redis          *redis.Client
	allowedOrigins map[string]bool
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(redisClient *redis.Client, allowedOrigins []string) *WebSocketHandler {
	originMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		originMap[origin] = true
	}
	return &WebSocketHandler{
		redis:          redisClient,
		allowedOrigins: originMap,
	}
}

// WSTicketRequest represents the request body (empty, just authenticated)
type WSTicketRequest struct{}

// WSTicketResponse represents a WebSocket ticket response
type WSTicketResponse struct {
	Ticket   string `json:"ticket"`
	ExpiresAt int64  `json:"expires_at"`
}

// GetWSTicket generates a one-time-use ticket for WebSocket authentication
func (h *WebSocketHandler) GetWSTicket(c *fiber.Ctx) error {
	// User is authenticated via JWT middleware (cookie)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return RespondWithError(c, fiber.StatusUnauthorized, apperrors.ErrNotAuthenticated, "Not authenticated")
	}

	// Generate a random ticket UUID
	ticket := uuid.New().String()

	// Store ticket in Redis with user_id and 30-second expiration
	ctx := context.Background()
	key := wsTicketPrefix + ticket
	ticketData := map[string]interface{}{
		"user_id": userID,
		"created": time.Now().Unix(),
	}

	if err := h.redis.HMSet(ctx, key, ticketData).Err(); err != nil {
		log.Printf("[WS] Failed to store ticket: %v", err)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Failed to generate ticket")
	}

	// Set expiration
	if err := h.redis.Expire(ctx, key, wsTicketExpiry).Err(); err != nil {
		log.Printf("[WS] Failed to set ticket expiration: %v", err)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Failed to generate ticket")
	}

	log.Printf("[WS] Generated ticket for user %s", userID)

	return c.JSON(WSTicketResponse{
		Ticket:    ticket,
		ExpiresAt: time.Now().Add(wsTicketExpiry).Unix(),
	})
}

// validateTicket validates a WebSocket ticket and returns the user ID
// The ticket is deleted after successful validation (one-time use)
func (h *WebSocketHandler) validateTicket(ctx context.Context, ticket string) (string, error) {
	if ticket == "" {
		return "", ErrInvalidWSTicket
	}

	key := wsTicketPrefix + ticket

	// Get ticket data from Redis
	data, err := h.redis.HGetAll(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrInvalidWSTicket
		}
		return "", fmt.Errorf("failed to validate ticket: %w", err)
	}

	if len(data) == 0 {
		return "", ErrInvalidWSTicket
	}

	userID, ok := data["user_id"]
	if !ok || userID == "" {
		return "", ErrInvalidWSTicket
	}

	// Delete ticket to prevent reuse (one-time use)
	if err := h.redis.Del(ctx, key).Err(); err != nil {
		log.Printf("[WS] Failed to delete ticket after use: %v", err)
		// Continue anyway - the important part is we validated it
	}

	return userID, nil
}

// validateOrigin checks if the Origin header is in the allowed list
func (h *WebSocketHandler) validateOrigin(origin string) error {
	if origin == "" {
		// Some clients don't send Origin (e.g., some mobile apps, non-browser clients)
		// Allow these connections but log a warning
		return nil
	}

	// Normalize origin (remove trailing slash)
	origin = strings.TrimSuffix(origin, "/")

	if h.allowedOrigins[origin] {
		return nil
	}

	return ErrOriginNotAllowed
}

// HandleWebSocket upgrades HTTP to WebSocket and streams events
func (h *WebSocketHandler) HandleWebSocket() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		// Validate Origin header before doing any expensive operations
		if err := h.validateOrigin(c.Headers("Origin")); err != nil {
			log.Printf("[WS] Rejected connection from origin %s: %v", c.Headers("Origin"), err)
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(
				websocket.ClosePolicyViolation, "Origin not allowed",
			))
			c.Close()
			return
		}

		// Extract ticket from query parameter
		ticket := c.Query("ticket")
		if ticket == "" {
			log.Printf("[WS] Rejected connection: missing ticket parameter")
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(
				websocket.ClosePolicyViolation, "Missing ticket",
			))
			c.Close()
			return
		}

		// Validate ticket and get user ID
		ticketCtx := context.Background()
		userIDStr, err := h.validateTicket(ticketCtx, ticket)
		if err != nil {
			log.Printf("[WS] Rejected connection: invalid ticket: %v", err)
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(
				websocket.ClosePolicyViolation, "Invalid ticket",
			))
			c.Close()
			return
		}

		// Parse user ID as UUID
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("[WS] Rejected connection: invalid user_id in ticket")
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(
				websocket.ClosePolicyViolation, "Invalid user",
			))
			c.Close()
			return
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
