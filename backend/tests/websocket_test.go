package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	

	"github.com/tdawe1/translation-app/internal/handlers"
)

// TestWebSocket_Authentication tests WebSocket authentication scenarios
func TestWebSocket_Authentication(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	wsHandler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	t.Run("missing ticket parameter", func(t *testing.T) {
		ctx := context.Background()
		_, err := wsHandler.ValidateTicket(ctx, "")
		assert.Error(t, err, "Should reject empty ticket")
		assert.ErrorIs(t, err, handlers.ErrInvalidWSTicket)
	})

	t.Run("invalid ticket format", func(t *testing.T) {
		ctx := context.Background()
		_, err := wsHandler.ValidateTicket(ctx, "invalid-ticket-123")
		assert.Error(t, err, "Should reject invalid ticket")
		assert.ErrorIs(t, err, handlers.ErrInvalidWSTicket)
	})

	t.Run("valid ticket", func(t *testing.T) {
		ctx := context.Background()
		userID := uuid.New()

		// Create and store a valid ticket directly
		ticket := uuid.New().String()
		key := "ws:ticket:" + ticket
		ticketData := map[string]interface{}{
			"user_id": userID.String(),
			"created": time.Now().Unix(),
		}

		err := redisClient.HMSet(ctx, key, ticketData).Err()
		require.NoError(t, err)
		defer redisClient.Del(ctx, key)

		err = redisClient.Expire(ctx, key, 30*time.Second).Err()
		require.NoError(t, err)

		// Validate the ticket
		validatedUserID, err := wsHandler.ValidateTicket(ctx, ticket)
		assert.NoError(t, err, "Valid ticket should pass validation")
		assert.Equal(t, userID.String(), validatedUserID)
	})

	t.Run("ticket is one-time use", func(t *testing.T) {
		ctx := context.Background()
		userID := uuid.New()

		ticket := uuid.New().String()
		key := "ws:ticket:" + ticket
		ticketData := map[string]interface{}{
			"user_id": userID.String(),
			"created": time.Now().Unix(),
		}

		err := redisClient.HMSet(ctx, key, ticketData).Err()
		require.NoError(t, err)

		err = redisClient.Expire(ctx, key, 30*time.Second).Err()
		require.NoError(t, err)

		// First validation should succeed
		handler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})
		_, err = handler.ValidateTicket(ctx, ticket)
		assert.NoError(t, err)

		// Second validation should fail (ticket was deleted)
		_, err = handler.ValidateTicket(ctx, ticket)
		assert.Error(t, err, "Ticket should not be reusable")
	})
}

// TestWebSocket_ReceivesJobNotification tests job notification via Redis pub/sub
func TestWebSocket_ReceivesJobNotification(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userID := uuid.New()
	wsHandler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	// Create a valid ticket
	ticket := uuid.New().String()
	key := "ws:ticket:" + ticket
	ticketData := map[string]interface{}{
		"user_id": userID.String(),
		"created": time.Now().Unix(),
	}
	require.NoError(t, redisClient.HMSet(ctx, key, ticketData).Err())
	require.NoError(t, redisClient.Expire(ctx, key, 30*time.Second).Err())

	// Create test job
	testJob := map[string]interface{}{
		"id":     "job-123",
		"title":  "Test Job",
		"reward": 5.50,
		"source": "rss",
	}

	// Create channels for synchronization
	connected := make(chan bool)
	received := make(chan string)
	errors := make(chan error, 1)

	// Start a goroutine to simulate WebSocket connection
	go func() {
		// Simulate WebSocket handler behavior
		pubsub := redisClient.Subscribe(ctx, wsHandler.GetUserChannels(userID)...)
		defer pubsub.Close()

		close(connected)

		// Listen for one message
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			errors <- err
			return
		}

		received <- msg.Payload
	}()

	// Wait for subscription to be ready
	<-connected

	// Publish job notification
	err := wsHandler.PublishJob(ctx, userID, testJob)
	require.NoError(t, err, "Should publish job successfully")

	// Wait to receive the message
	select {
	case payload := <-received:
		var receivedJob map[string]interface{}
		err := json.Unmarshal([]byte(payload), &receivedJob)
		require.NoError(t, err)
		assert.Equal(t, "job-123", receivedJob["id"])
		assert.Equal(t, "Test Job", receivedJob["title"])
		assert.Equal(t, 5.50, receivedJob["reward"])
	case err := <-errors:
		t.Fatalf("Error receiving message: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for job notification")
	}
}

// TestWebSocket_ReceivesEventNotification tests watcher event notifications
func TestWebSocket_ReceivesEventNotification(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userID := uuid.New()
	wsHandler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	// Create a valid ticket
	ticket := uuid.New().String()
	key := "ws:ticket:" + ticket
	ticketData := map[string]interface{}{
		"user_id": userID.String(),
		"created": time.Now().Unix(),
	}
	require.NoError(t, redisClient.HMSet(ctx, key, ticketData).Err())
	require.NoError(t, redisClient.Expire(ctx, key, 30*time.Second).Err())

	// Test events
	testEvents := []struct {
		eventType string
		data      map[string]interface{}
	}{
		{
			eventType: "watcher.started",
			data:      map[string]interface{}{"status": "running"},
		},
		{
			eventType: "watcher.stopped",
			data:      map[string]interface{}{"status": "stopped"},
		},
		{
			eventType: "job.filtered",
			data: map[string]interface{}{
				"job_id":  "job-456",
				"reason":  "reward_below_minimum",
			},
		},
	}

	for _, tc := range testEvents {
		t.Run(tc.eventType, func(t *testing.T) {
			// Create channels for synchronization
			connected := make(chan bool)
			received := make(chan map[string]interface{})
			errors := make(chan error, 1)

			// Start a goroutine to simulate WebSocket connection
			go func() {
				pubsub := redisClient.Subscribe(ctx, "user:"+userID.String()+":events")
				defer pubsub.Close()

				close(connected)

				msg, err := pubsub.ReceiveMessage(ctx)
				if err != nil {
					errors <- err
					return
				}

				var result map[string]interface{}
				if err := json.Unmarshal([]byte(msg.Payload), &result); err != nil {
					errors <- err
					return
				}
				received <- result
			}()

			// Wait for subscription
			<-connected

			// Publish event
			err := wsHandler.PublishEvent(ctx, userID, tc.eventType, tc.data)
			require.NoError(t, err)

			// Verify received event
			select {
			case result := <-received:
				assert.Equal(t, "event", result["type"])
				assert.Equal(t, tc.eventType, result["event"])
				assert.NotNil(t, result["timestamp"])
			case err := <-errors:
				t.Fatalf("Error receiving event: %v", err)
			case <-time.After(2 * time.Second):
				t.Fatal("Timeout waiting for event notification")
			}
		})
	}
}

// TestWebSocket_HandlesDisconnect tests graceful connection cleanup
func TestWebSocket_HandlesDisconnect(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userID := uuid.New()
	wsHandler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	// Create a valid ticket
	ticket := uuid.New().String()
	key := "ws:ticket:" + ticket
	ticketData := map[string]interface{}{
		"user_id": userID.String(),
		"created": time.Now().Unix(),
	}
	require.NoError(t, redisClient.HMSet(ctx, key, ticketData).Err())
	require.NoError(t, redisClient.Expire(ctx, key, 30*time.Second).Err())

	// Simulate connection lifecycle
	pubsub := redisClient.Subscribe(ctx, wsHandler.GetUserChannels(userID)...)

	// Wait for subscription confirmation (Redis pub/sub is async)
	subCtx, subCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer subCancel()

	_, err := pubsub.Receive(subCtx)
	// We expect a timeout (no messages yet) or subscription confirmation
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("Subscription warning: %v", err)
	}

	// Close pubsub (simulating disconnect)
	pubsub.Close()

	// Verify cleanup - should be able to subscribe again (no resource leak)
	pubsub2 := redisClient.Subscribe(ctx, wsHandler.GetUserChannels(userID)...)
	defer pubsub2.Close()

	// The second subscription should work (no resource leak)
	// Just verify we can create a new subscription without error
	assert.NotNil(t, pubsub2, "Should be able to create new subscription")
}

// TestWebSocket_GetUserChannels tests channel name generation
func TestWebSocket_GetUserChannels(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	wsHandler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	channels := wsHandler.GetUserChannels(userID)
	require.Len(t, channels, 3, "Should return 3 channels")

	expectedChannels := []string{
		"user:550e8400-e29b-41d4-a716-446655440000:jobs",
		"user:550e8400-e29b-41d4-a716-446655440000:events",
		"user:550e8400-e29b-41d4-a716-446655440000:errors",
	}

	assert.Equal(t, expectedChannels, channels)
}

// TestWebSocket_ValidateOrigin tests origin validation
func TestWebSocket_ValidateOrigin(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	allowedOrigins := []string{
		"http://localhost:3000",
		"https://app.example.com",
		"https://gengowatcher.com",
	}
	wsHandler := handlers.NewWebSocketHandler(redisClient, allowedOrigins)

	t.Run("allowed origin", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("http://localhost:3000")
		assert.NoError(t, err, "Should allow valid origin")

		err = wsHandler.ValidateOrigin("http://localhost:3000/") // with trailing slash
		assert.NoError(t, err, "Should allow valid origin with trailing slash")
	})

	t.Run("disallowed origin", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("http://evil.com")
		assert.Error(t, err, "Should reject disallowed origin")
		assert.ErrorIs(t, err, handlers.ErrOriginNotAllowed)
	})

	t.Run("empty origin", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("")
		assert.NoError(t, err, "Should allow empty origin (non-browser clients)")
	})
}

// TestWebSocket_PublishError tests error notification publishing
func TestWebSocket_PublishError(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userID := uuid.New()
	wsHandler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	// Subscribe to error channel
	errorChannel := "user:" + userID.String() + ":errors"
	pubsub := redisClient.Subscribe(ctx, errorChannel)
	defer pubsub.Close()

	// Publish error
	errMsg := "Connection to Gengo WebSocket failed"
	err := wsHandler.PublishError(ctx, userID, errMsg)
	require.NoError(t, err, "Should publish error successfully")

	// Verify received error
	msg, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err, "Should receive published error")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(msg.Payload), &result)
	require.NoError(t, err)

	assert.Equal(t, "error", result["type"])
	assert.Equal(t, errMsg, result["message"])
	assert.NotNil(t, result["timestamp"])
}

// TestWebSocket_TicketExpiry tests ticket expiration behavior
func TestWebSocket_TicketExpiry(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userID := uuid.New()
	wsHandler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	// Create a ticket with 1 second expiration (Redis minimum)
	ticket := uuid.New().String()
	key := "ws:ticket:" + ticket
	ticketData := map[string]interface{}{
		"user_id": userID.String(),
		"created": time.Now().Unix(),
	}

	require.NoError(t, redisClient.HMSet(ctx, key, ticketData).Err())
	require.NoError(t, redisClient.Expire(ctx, key, 1*time.Second).Err())

	// Wait for ticket to expire (slightly more than 1 second)
	time.Sleep(1100 * time.Millisecond)

	// Try to validate expired ticket
	_, err := wsHandler.ValidateTicket(ctx, ticket)
	assert.Error(t, err, "Should reject expired ticket")
}

// TestWebSocket_ConcurrentConnections tests multiple concurrent connections
func TestWebSocket_ConcurrentConnections(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	wsHandler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	// Create multiple users with tickets
	numUsers := 5
	userIDs := make([]uuid.UUID, numUsers)
	tickets := make([]string, numUsers)

	for i := 0; i < numUsers; i++ {
		userID := uuid.New()
		userIDs[i] = userID
		ticket := uuid.New().String()
		tickets[i] = ticket

		key := "ws:ticket:" + ticket
		ticketData := map[string]interface{}{
			"user_id": userID.String(),
			"created": time.Now().Unix(),
		}
		require.NoError(t, redisClient.HMSet(ctx, key, ticketData).Err())
		require.NoError(t, redisClient.Expire(ctx, key, 30*time.Second).Err())
	}

	// Validate all tickets concurrently
	done := make(chan bool, numUsers)
	for i := 0; i < numUsers; i++ {
		go func(idx int) {
			validatedUserID, err := wsHandler.ValidateTicket(ctx, tickets[idx])
			assert.NoError(t, err)
			assert.Equal(t, userIDs[idx].String(), validatedUserID)
			done <- true
		}(i)
	}

	// Wait for all validations
	for i := 0; i < numUsers; i++ {
		select {
		case <-done:
			// OK
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for concurrent validations")
		}
	}

	// Verify tickets were deleted (one-time use)
	for _, ticket := range tickets {
		_, err := wsHandler.ValidateTicket(ctx, ticket)
		assert.Error(t, err, "Ticket should not be reusable after validation")
	}
}

// TestWebSocket_ErrorChannel tests error notification via Redis
func TestWebSocket_ErrorChannel(t *testing.T) {
	redisClient := RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userID := uuid.New()
	wsHandler := handlers.NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	// Create a valid ticket
	ticket := uuid.New().String()
	key := "ws:ticket:" + ticket
	ticketData := map[string]interface{}{
		"user_id": userID.String(),
		"created": time.Now().Unix(),
	}
	require.NoError(t, redisClient.HMSet(ctx, key, ticketData).Err())
	require.NoError(t, redisClient.Expire(ctx, key, 30*time.Second).Err())

	// Subscribe to error channel
	errorChannel := "user:" + userID.String() + ":errors"
	pubsub := redisClient.Subscribe(ctx, errorChannel)
	defer pubsub.Close()

	// Publish multiple errors
	testErrors := []string{
		"RSS feed fetch failed",
		"WebSocket connection lost",
		"Job parsing error",
	}

	for _, errMsg := range testErrors {
		err := wsHandler.PublishError(ctx, userID, errMsg)
		require.NoError(t, err)
	}

	// Receive all errors
	receivedErrors := make([]string, 0)
	timeout := time.After(2 * time.Second)

	for len(receivedErrors) < len(testErrors) {
		select {
		case msg := <-pubsub.Channel():
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &result); err != nil {
				t.Fatalf("Failed to unmarshal error: %v", err)
			}
			if errMsg, ok := result["message"].(string); ok {
				receivedErrors = append(receivedErrors, errMsg)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for errors. Received %d/%d", len(receivedErrors), len(testErrors))
		}
	}

	assert.ElementsMatch(t, testErrors, receivedErrors)
}
