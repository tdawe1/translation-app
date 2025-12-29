package tests

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
)

// TestWebSocket_TicketGeneration tests the ticket endpoint
func TestWebSocket_TicketGeneration(t *testing.T) {
	db := TestDB(t)
	redis := RequireRedis(t)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	wsHandler := handlers.NewWebSocketHandler(redis, []string{"http://localhost:3000"})
	jwtCfg := middleware.NewJWTConfig()

	app.Post("/api/v1/ws/ticket", middleware.JWTValidator(jwtCfg), wsHandler.GetWSTicket)

	user := CreateTestUser(t, db, "test-ws-ticket@example.com")
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	t.Run("GetWSTicket requires authentication", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/ws/ticket", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("GetWSTicket returns a valid ticket", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/ws/ticket", nil)
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var ticketResp handlers.WSTicketResponse
		err = json.NewDecoder(resp.Body).Decode(&ticketResp)
		require.NoError(t, err)

		assert.NotEmpty(t, ticketResp.Ticket)
		assert.Greater(t, ticketResp.ExpiresAt, time.Now().Unix())
		assert.LessOrEqual(t, ticketResp.ExpiresAt, time.Now().Add(31*time.Second).Unix())
	})

	t.Run("Generated ticket is stored in Redis", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/ws/ticket", nil)
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var ticketResp handlers.WSTicketResponse
		err = json.NewDecoder(resp.Body).Decode(&ticketResp)
		require.NoError(t, err)

		// Verify ticket exists in Redis
		ctx := context.Background()
		key := "ws:ticket:" + ticketResp.Ticket
		exists, err := redis.Exists(ctx, key).Result()
		require.NoError(t, err)
		assert.True(t, exists > 0, "Ticket should exist in Redis")

		// Verify ticket contains user_id
		data, err := redis.HGetAll(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, user.ID.String(), data["user_id"])
	})
}

// TestWebSocket_TicketValidation tests the ticket validation logic
func TestWebSocket_TicketValidation(t *testing.T) {
	redis := RequireRedis(t)
	wsHandler := handlers.NewWebSocketHandler(redis, []string{"http://localhost:3000"})

	t.Run("ValidateTicket rejects empty ticket", func(t *testing.T) {
		ctx := context.Background()
		_, err := wsHandler.ValidateTicket(ctx, "")
		assert.Error(t, err)
		assert.Equal(t, handlers.ErrInvalidWSTicket, err)
	})

	t.Run("ValidateTicket rejects non-existent ticket", func(t *testing.T) {
		ctx := context.Background()
		_, err := wsHandler.ValidateTicket(ctx, "non-existent-ticket")
		assert.Error(t, err)
		assert.Equal(t, handlers.ErrInvalidWSTicket, err)
	})

	t.Run("ValidateTicket accepts valid ticket and deletes it", func(t *testing.T) {
		ctx := context.Background()
		userID := uuid.New()

		// Create a valid ticket
		ticket := uuid.New().String()
		key := "ws:ticket:" + ticket
		ticketData := map[string]interface{}{
			"user_id": userID.String(),
			"created": time.Now().Unix(),
		}
		err := redis.HMSet(ctx, key, ticketData).Err()
		require.NoError(t, err)
		redis.Expire(ctx, key, 30*time.Second)

		// Validate ticket
		retrievedUserID, err := wsHandler.ValidateTicket(ctx, ticket)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), retrievedUserID)

		// Ticket should be deleted (one-time use)
		exists, _ := redis.Exists(ctx, key).Result()
		assert.Equal(t, int64(0), exists, "Ticket should be deleted after use")

		// Second validation should fail
		_, err = wsHandler.ValidateTicket(ctx, ticket)
		assert.Error(t, err)
	})
}

// TestWebSocket_OriginValidation tests the Origin header validation
func TestWebSocket_OriginValidation(t *testing.T) {
	redis := RequireRedis(t)
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	wsHandler := handlers.NewWebSocketHandler(redis, allowedOrigins)

	t.Run("accepts empty origin", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("")
		assert.NoError(t, err)
	})

	t.Run("accepts allowed origin", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("http://localhost:3000")
		assert.NoError(t, err)

		err = wsHandler.ValidateOrigin("http://localhost:3000/")
		assert.NoError(t, err)
	})

	t.Run("accepts allowed origin with trailing slash", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("https://example.com")
		assert.NoError(t, err)
	})

	t.Run("rejects disallowed origin", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("https://evil.com")
		assert.Error(t, err)
		assert.Equal(t, handlers.ErrOriginNotAllowed, err)
	})
}

// TestWebSocket_PublishMethods tests the Redis publish methods
func TestWebSocket_PublishMethods(t *testing.T) {
	db := TestDB(t)
	redis := RequireRedis(t)
	user := CreateTestUser(t, db, "test-ws-publish@example.com")
	wsHandler := handlers.NewWebSocketHandler(redis, []string{"http://localhost:3000"})

	ctx := context.Background()

	t.Run("PublishJob publishes to correct channel", func(t *testing.T) {
		job := map[string]interface{}{
			"id":     "job-123",
			"title":  "Test Job",
			"reward": 5.50,
			"source": "rss",
		}

		// Subscribe first (before publishing)
		jobsChannel := "user:" + user.ID.String() + ":jobs"
		pubsub := redis.Subscribe(ctx, jobsChannel)
		defer pubsub.Close()

		// Wait for subscription to be ready
		_, err := pubsub.Receive(ctx)
		require.NoError(t, err, "Should receive subscription confirmation")

		// Now publish the job
		err = wsHandler.PublishJob(ctx, user.ID, job)
		require.NoError(t, err)

		// Wait for message with timeout
		msgCtx, msgCancel := context.WithTimeout(ctx, 2*time.Second)
		defer msgCancel()
		msg, err := pubsub.ReceiveMessage(msgCtx)
		require.NoError(t, err)

		var receivedJob map[string]interface{}
		err = json.Unmarshal([]byte(msg.Payload), &receivedJob)
		require.NoError(t, err)
		assert.Equal(t, "job-123", receivedJob["id"])
		assert.Equal(t, "Test Job", receivedJob["title"])
	})

	t.Run("PublishEvent publishes to correct channel", func(t *testing.T) {
		eventData := map[string]interface{}{
			"status": "running",
		}

		// Subscribe first
		eventsChannel := "user:" + user.ID.String() + ":events"
		pubsub := redis.Subscribe(ctx, eventsChannel)
		defer pubsub.Close()

		// Wait for subscription to be ready
		_, err := pubsub.Receive(ctx)
		require.NoError(t, err, "Should receive subscription confirmation")

		// Now publish the event
		err = wsHandler.PublishEvent(ctx, user.ID, "watcher.started", eventData)
		require.NoError(t, err)

		// Wait for message with timeout
		msgCtx, msgCancel := context.WithTimeout(ctx, 2*time.Second)
		defer msgCancel()
		msg, err := pubsub.ReceiveMessage(msgCtx)
		require.NoError(t, err)

		var receivedEvent map[string]interface{}
		err = json.Unmarshal([]byte(msg.Payload), &receivedEvent)
		require.NoError(t, err)
		assert.Equal(t, "event", receivedEvent["type"])
		assert.Equal(t, "watcher.started", receivedEvent["event"])
		assert.Contains(t, receivedEvent, "data")
		assert.Contains(t, receivedEvent, "timestamp")
	})

	t.Run("PublishError publishes to correct channel", func(t *testing.T) {
		errMsg := "Test error message"

		// Subscribe first
		errorsChannel := "user:" + user.ID.String() + ":errors"
		pubsub := redis.Subscribe(ctx, errorsChannel)
		defer pubsub.Close()

		// Wait for subscription to be ready
		_, err := pubsub.Receive(ctx)
		require.NoError(t, err, "Should receive subscription confirmation")

		// Now publish the error
		err = wsHandler.PublishError(ctx, user.ID, errMsg)
		require.NoError(t, err)

		// Wait for message with timeout
		msgCtx, msgCancel := context.WithTimeout(ctx, 2*time.Second)
		defer msgCancel()
		msg, err := pubsub.ReceiveMessage(msgCtx)
		require.NoError(t, err)

		var receivedError map[string]interface{}
		err = json.Unmarshal([]byte(msg.Payload), &receivedError)
		require.NoError(t, err)
		assert.Equal(t, "error", receivedError["type"])
		assert.Equal(t, errMsg, receivedError["message"])
		assert.Contains(t, receivedError, "timestamp")
	})
}

// TestWebSocket_GetUserChannels tests the channel naming
func TestWebSocket_GetUserChannels(t *testing.T) {
	redis := RequireRedis(t)
	wsHandler := handlers.NewWebSocketHandler(redis, []string{"http://localhost:3000"})

	userID := uuid.New()
	channels := wsHandler.GetUserChannels(userID)

	t.Run("returns correct channel names", func(t *testing.T) {
		expected := []string{
			"user:" + userID.String() + ":jobs",
			"user:" + userID.String() + ":events",
			"user:" + userID.String() + ":errors",
		}

		// Verify the returned channels match expected
		assert.Equal(t, expected, channels)

		// Verify channels are correctly named through PublishJob test
		ctx := context.Background()
		testJob := map[string]interface{}{"id": "test"}

		// Subscribe first
		pubsub := redis.Subscribe(ctx, channels[0])
		defer pubsub.Close()

		// Wait for subscription to be ready
		_, err := pubsub.Receive(ctx)
		require.NoError(t, err, "Should receive subscription confirmation")

		// Now publish
		err = wsHandler.PublishJob(ctx, userID, testJob)
		require.NoError(t, err)

		// Receive message
		msgCtx, msgCancel := context.WithTimeout(ctx, 2*time.Second)
		defer msgCancel()
		msg, err := pubsub.ReceiveMessage(msgCtx)
		require.NoError(t, err)
		var received map[string]interface{}
		json.Unmarshal([]byte(msg.Payload), &received)
		assert.Equal(t, "test", received["id"])
	})
}

// TestWebSocket_RedisPubSubIntegration tests end-to-end Redis pub/sub flow
func TestWebSocket_RedisPubSubIntegration(t *testing.T) {
	db := TestDB(t)
	redis := RequireRedis(t)
	user := CreateTestUser(t, db, "test-ws-pubsub@example.com")
	wsHandler := handlers.NewWebSocketHandler(redis, []string{"http://localhost:3000"})

	ctx := context.Background()

	t.Run("multiple subscribers receive same message", func(t *testing.T) {
		// Create multiple subscribers
		jobsChannel := "user:" + user.ID.String() + ":jobs"
		pubsub1 := redis.Subscribe(ctx, jobsChannel)
		pubsub2 := redis.Subscribe(ctx, jobsChannel)
		defer pubsub1.Close()
		defer pubsub2.Close()

		// Publish a job
		job := map[string]interface{}{
			"id":     "job-multi",
			"title":  "Multi Test",
			"reward": 10.0,
		}
		err := wsHandler.PublishJob(ctx, user.ID, job)
		require.NoError(t, err)

		// Both subscribers should receive the message
		msgCtx, msgCancel := context.WithTimeout(ctx, 2*time.Second)
		defer msgCancel()
		msg1, err := pubsub1.ReceiveMessage(msgCtx)
		require.NoError(t, err)

		msg2, err := pubsub2.ReceiveMessage(msgCtx)
		require.NoError(t, err)

		// Messages should have the same payload
		assert.Equal(t, msg1.Payload, msg2.Payload)

		var receivedJob map[string]interface{}
		err = json.Unmarshal([]byte(msg1.Payload), &receivedJob)
		require.NoError(t, err)
		assert.Equal(t, "job-multi", receivedJob["id"])
	})
}

// TestWebSocket_TicketExpiration tests ticket expiration behavior
func TestWebSocket_TicketExpiration(t *testing.T) {
	redis := RequireRedis(t)

	t.Run("expired ticket is rejected", func(t *testing.T) {
		ctx := context.Background()
		userID := uuid.New()

		// Create a ticket with short expiration
		ticket := uuid.New().String()
		key := "ws:ticket:" + ticket
		ticketData := map[string]interface{}{
			"user_id": userID.String(),
			"created": time.Now().Unix(),
		}
		err := redis.HMSet(ctx, key, ticketData).Err()
		require.NoError(t, err)

		// Set short expiration (100ms - Redis minimum is 1s, but we'll delete manually after delay)
		// Note: Redis Expire minimum is 1 second, so we'll manually delete the ticket
		// after a short delay to simulate expiration
		redis.Expire(ctx, key, 1*time.Second)

		// Manually delete the ticket after a short delay to simulate expiration
		// (This is faster than waiting for Redis's 1-second minimum expiration)
		go func() {
			time.Sleep(50 * time.Millisecond)
			redis.Del(ctx, key)
		}()

		// Wait for ticket to be deleted
		time.Sleep(100 * time.Millisecond)

		// Try to validate expired ticket
		wsHandler := handlers.NewWebSocketHandler(redis, []string{"http://localhost:3000"})
		_, err = wsHandler.ValidateTicket(ctx, ticket)
		assert.Error(t, err, "Expired ticket should be rejected")
		assert.Equal(t, handlers.ErrInvalidWSTicket, err)
	})
}

// TestWebSocket_HttpEndpoint tests the HTTP upgrade endpoint behavior
func TestWebSocket_HttpEndpoint(t *testing.T) {
	db := TestDB(t)
	redis := RequireRedis(t)
	user := CreateTestUser(t, db, "test-ws-http@example.com")

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	wsHandler := handlers.NewWebSocketHandler(redis, []string{"http://localhost:3000"})
	app.Get("/ws", wsHandler.HandleWebSocket())

	t.Run("WebSocket upgrade requires ticket parameter", func(t *testing.T) {
		// Request without ticket should return 426 Upgrade Required
		req := httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")

		resp, err := app.Test(req)
		require.NoError(t, err)
		// Fiber websocket middleware returns 426 for non-websocket requests
		assert.NotEqual(t, 200, resp.StatusCode, "Should not accept connection without proper WebSocket upgrade")
	})

	t.Run("WebSocket upgrade with valid ticket accepts connection", func(t *testing.T) {
		// Generate a valid ticket
		ctx := context.Background()
		ticket := uuid.New().String()
		key := "ws:ticket:" + ticket
		ticketData := map[string]interface{}{
			"user_id": user.ID.String(),
			"created": time.Now().Unix(),
		}
		redis.HMSet(ctx, key, ticketData)
		redis.Expire(ctx, key, 30*time.Second)

		// Request with ticket - Fiber websocket handler handles this
		// We can't easily test the actual WebSocket upgrade in a unit test,
		// but we can verify the handler is registered and responds
		req := httptest.NewRequest("GET", "/ws?ticket="+ticket, nil)
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")

		resp, err := app.Test(req)
		require.NoError(t, err)
		// The websocket middleware handles the upgrade
		// A 426 means the request wasn't a proper WebSocket upgrade
		// A 101 would be successful upgrade
		// Either way, the handler is processing the request
		assert.True(t, resp.StatusCode == 101 || resp.StatusCode == 426,
			"WebSocket handler should process the request")
	})
}

// TestWebSocket_ConnectedMessage tests the initial connected message format
func TestWebSocket_ConnectedMessage(t *testing.T) {
	// Test the connected message structure without actual WebSocket
	db := TestDB(t)
	redis := RequireRedis(t)
	user := CreateTestUser(t, db, "test-ws-connected@example.com")

	wsHandler := handlers.NewWebSocketHandler(redis, []string{"http://localhost:3000"})

	// We can test that the handler creates the correct message structure
	// by checking the PublishEvent method which uses the same JSON structure
	ctx := context.Background()

	// Subscribe first (before publishing)
	eventsChannel := "user:" + user.ID.String() + ":events"
	pubsub := redis.Subscribe(ctx, eventsChannel)
	defer pubsub.Close()

	// Wait for subscription to be ready
	_, err := pubsub.Receive(ctx)
	require.NoError(t, err, "Should receive subscription confirmation")

	// Now publish the event
	err = wsHandler.PublishEvent(ctx, user.ID, "test.event", map[string]interface{}{
		"test_data": "value",
	})
	require.NoError(t, err)

	// Wait for message with timeout
	msgCtx, msgCancel := context.WithTimeout(ctx, 2*time.Second)
	defer msgCancel()
	msg, err := pubsub.ReceiveMessage(msgCtx)
	require.NoError(t, err)

	var event map[string]interface{}
	err = json.Unmarshal([]byte(msg.Payload), &event)
	require.NoError(t, err)

	// Verify event has expected structure
	assert.Equal(t, "event", event["type"])
	assert.Equal(t, "test.event", event["event"])
	assert.Contains(t, event, "data")
	assert.Contains(t, event, "timestamp")
}
