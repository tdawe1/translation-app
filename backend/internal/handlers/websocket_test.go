package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/tests"
)

func TestWebSocketHandler_RequiresAuth(t *testing.T) {
	redisClient := tests.RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	wsHandler := NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	t.Run("ValidateTicket returns error for empty ticket", func(t *testing.T) {
		_, err := wsHandler.ValidateTicket(ctx, "")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidWSTicket)
	})

	t.Run("ValidateTicket returns error for non-existent ticket", func(t *testing.T) {
		_, err := wsHandler.ValidateTicket(ctx, "non-existent-ticket")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidWSTicket)
	})
}

func TestWebSocketHandler_ConnectsWithValidToken(t *testing.T) {
	redisClient := tests.RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userID := uuid.New()
	wsHandler := NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	t.Run("valid ticket allows connection", func(t *testing.T) {
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

		validatedUserID, err := wsHandler.ValidateTicket(ctx, ticket)
		assert.NoError(t, err, "Valid ticket should pass validation")
		assert.Equal(t, userID.String(), validatedUserID)
	})

	t.Run("ticket is one-time use", func(t *testing.T) {
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

		_, err = wsHandler.ValidateTicket(ctx, ticket)
		assert.NoError(t, err, "First validation should succeed")

		_, err = wsHandler.ValidateTicket(ctx, ticket)
		assert.Error(t, err, "Second validation should fail (ticket was deleted)")
	})
}

func TestWebSocketHandler_ReceivesJobUpdates(t *testing.T) {
	redisClient := tests.RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userID := uuid.New()
	wsHandler := NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	t.Run("job update is received via pub/sub", func(t *testing.T) {
		testJob := map[string]interface{}{
			"id":     "job-123",
			"title":  "Test Job",
			"reward": 5.50,
			"status": "new",
		}

		connected := make(chan bool)
		received := make(chan map[string]interface{})
		errors := make(chan error, 1)

		go func() {
			pubsub := redisClient.Subscribe(ctx, wsHandler.GetUserChannels(userID)...)
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

		<-connected

		err := wsHandler.PublishJob(ctx, userID, testJob)
		require.NoError(t, err, "Should publish job successfully")

		select {
		case result := <-received:
			assert.Equal(t, "job-123", result["id"])
			assert.Equal(t, "Test Job", result["title"])
			assert.Equal(t, 5.50, result["reward"])
			assert.Equal(t, "new", result["status"])
		case err := <-errors:
			t.Fatalf("Error receiving job update: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for job update")
		}
	})

	t.Run("multiple job updates are received", func(t *testing.T) {
		testJobs := []map[string]interface{}{
			{"id": "job-1", "status": "new"},
			{"id": "job-2", "status": "in_progress"},
			{"id": "job-3", "status": "completed"},
		}

		connected := make(chan bool)
		received := make(chan map[string]interface{}, len(testJobs))
		errors := make(chan error, 1)

		go func() {
			pubsub := redisClient.Subscribe(ctx, wsHandler.GetUserChannels(userID)...)
			defer pubsub.Close()

			close(connected)

			for i := 0; i < len(testJobs); i++ {
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
			}
		}()

		<-connected

		for _, job := range testJobs {
			err := wsHandler.PublishJob(ctx, userID, job)
			require.NoError(t, err)
		}

		receivedJobs := make([]map[string]interface{}, 0, len(testJobs))
		timeout := time.After(2 * time.Second)

		for len(receivedJobs) < len(testJobs) {
			select {
			case job := <-received:
				receivedJobs = append(receivedJobs, job)
			case err := <-errors:
				t.Fatalf("Error receiving job update: %v", err)
			case <-timeout:
				t.Fatalf("Timeout waiting for job updates. Received %d/%d", len(receivedJobs), len(testJobs))
			}
		}

		assert.Len(t, receivedJobs, len(testJobs))
	})
}

func TestWebSocketHandler_MultiTenantIsolation(t *testing.T) {
	redisClient := tests.RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()
	wsHandler := NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	t.Run("User A does not receive User B's job updates", func(t *testing.T) {
		jobForUserB := map[string]interface{}{
			"id":     "job-b-only",
			"title":  "Job for User B",
			"reward": 10.00,
		}

		connectedA := make(chan bool)
		receivedA := make(chan map[string]interface{}, 1)
		timeoutA := make(chan bool)

		go func() {
			pubsub := redisClient.Subscribe(ctx, wsHandler.GetUserChannels(userA)...)
			defer pubsub.Close()

			close(connectedA)

			select {
			case msg := <-pubsub.Channel():
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(msg.Payload), &result); err == nil {
					receivedA <- result
				}
			case <-time.After(500 * time.Millisecond):
				timeoutA <- true
			}
		}()

		<-connectedA

		err := wsHandler.PublishJob(ctx, userB, jobForUserB)
		require.NoError(t, err)

		select {
		case job := <-receivedA:
			t.Fatalf("User A should not receive User B's job, but got: %v", job)
		case <-timeoutA:
			assert.True(t, true, "User A correctly did not receive User B's job")
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for isolation confirmation")
		}
	})

	t.Run("User B receives their own job updates", func(t *testing.T) {
		jobForUserB := map[string]interface{}{
			"id":     "job-b-only-2",
			"title":  "Another Job for User B",
			"reward": 15.00,
		}

		connectedB := make(chan bool)
		receivedB := make(chan map[string]interface{})
		errorsB := make(chan error, 1)

		go func() {
			pubsub := redisClient.Subscribe(ctx, wsHandler.GetUserChannels(userB)...)
			defer pubsub.Close()

			close(connectedB)

			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				errorsB <- err
				return
			}

			var result map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &result); err != nil {
				errorsB <- err
				return
			}
			receivedB <- result
		}()

		<-connectedB

		err := wsHandler.PublishJob(ctx, userB, jobForUserB)
		require.NoError(t, err)

		select {
		case result := <-receivedB:
			assert.Equal(t, "job-b-only-2", result["id"])
			assert.Equal(t, "Another Job for User B", result["title"])
		case err := <-errorsB:
			t.Fatalf("Error receiving job: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for User B's job update")
		}
	})

	t.Run("separate user channels are isolated", func(t *testing.T) {
		channelsA := wsHandler.GetUserChannels(userA)
		channelsB := wsHandler.GetUserChannels(userB)

		for _, channelA := range channelsA {
			assert.NotContains(t, channelsB, channelA, "User A's channels should not overlap with User B's")
		}

		for _, channelB := range channelsB {
			assert.NotContains(t, channelsA, channelB, "User B's channels should not overlap with User A's")
		}
	})

	t.Run("events are isolated between users", func(t *testing.T) {
		eventForUserA := map[string]interface{}{
			"event_type": "user_a_event",
			"message":    "This is for User A only",
		}

		connectedB := make(chan bool)
		receivedB := make(chan map[string]interface{}, 1)
		timeoutB := make(chan bool)

		go func() {
			pubsub := redisClient.Subscribe(ctx, wsHandler.GetUserChannels(userB)...)
			defer pubsub.Close()

			close(connectedB)

			select {
			case msg := <-pubsub.Channel():
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(msg.Payload), &result); err == nil {
					receivedB <- result
				}
			case <-time.After(500 * time.Millisecond):
				timeoutB <- true
			}
		}()

		<-connectedB

		err := wsHandler.PublishEvent(ctx, userA, "custom_event", eventForUserA)
		require.NoError(t, err)

		select {
		case event := <-receivedB:
			t.Fatalf("User B should not receive User A's event, but got: %v", event)
		case <-timeoutB:
			assert.True(t, true, "User B correctly did not receive User A's event")
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for isolation confirmation")
		}
	})
}

func TestWebSocketHandler_GetUserChannels(t *testing.T) {
	redisClient := tests.RequireRedis(t)
	if redisClient == nil {
		return
	}

	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	wsHandler := NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	channels := wsHandler.GetUserChannels(userID)
	require.Len(t, channels, 3, "Should return 3 channels")

	expectedChannels := []string{
		"user:550e8400-e29b-41d4-a716-446655440000:jobs",
		"user:550e8400-e29b-41d4-a716-446655440000:events",
		"user:550e8400-e29b-41d4-a716-446655440000:errors",
	}

	assert.Equal(t, expectedChannels, channels)
}

func TestWebSocketHandler_ValidateOrigin(t *testing.T) {
	redisClient := tests.RequireRedis(t)
	if redisClient == nil {
		return
	}

	allowedOrigins := []string{
		"http://localhost:3000",
		"https://app.example.com",
		"https://gengowatcher.com",
	}
	wsHandler := NewWebSocketHandler(redisClient, allowedOrigins)

	t.Run("allowed origin", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("http://localhost:3000")
		assert.NoError(t, err)

		err = wsHandler.ValidateOrigin("http://localhost:3000/")
		assert.NoError(t, err)
	})

	t.Run("disallowed origin", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("http://evil.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrOriginNotAllowed)
	})

	t.Run("empty origin", func(t *testing.T) {
		err := wsHandler.ValidateOrigin("")
		assert.NoError(t, err, "Should allow empty origin (non-browser clients)")
	})
}

func TestWebSocketHandler_PublishError(t *testing.T) {
	redisClient := tests.RequireRedis(t)
	if redisClient == nil {
		return
	}

	ctx := context.Background()
	userID := uuid.New()
	wsHandler := NewWebSocketHandler(redisClient, []string{"http://localhost:3000"})

	errorChannel := "user:" + userID.String() + ":errors"
	pubsub := redisClient.Subscribe(ctx, errorChannel)
	defer pubsub.Close()

	errMsg := "Connection to Gengo WebSocket failed"
	err := wsHandler.PublishError(ctx, userID, errMsg)
	require.NoError(t, err)

	msg, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(msg.Payload), &result)
	require.NoError(t, err)

	assert.Equal(t, "error", result["type"])
	assert.Equal(t, errMsg, result["message"])
	assert.NotNil(t, result["timestamp"])
}
