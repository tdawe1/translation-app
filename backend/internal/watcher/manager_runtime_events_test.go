package watcher

import (
	"testing"
	"time"
)

func TestRuntimeEventsFromUpdatesIncludesFeedTransportEvents(t *testing.T) {
	now := time.Date(2026, 4, 26, 10, 30, 0, 0, time.UTC)

	events := runtimeEventsFromUpdates(map[string]interface{}{
		"last_rss_poll_started_at": now,
		"last_rss_poll_ok_at":      now.Add(time.Second),
		"rss_consecutive_failures": 0,
		"last_ws_connect_at":       now.Add(2 * time.Second),
		"last_ws_message_at":       now.Add(3 * time.Second),
		"last_ws_pong_at":          now.Add(4 * time.Second),
		"last_ws_close_reason":     "read timeout",
	})

	eventTypes := make(map[string]bool)
	for _, event := range events {
		eventTypes[event.Type] = true
		if event.Message == "" {
			t.Fatalf("event %s should include a message", event.Type)
		}
		if event.Source != "rss" && event.Source != "websocket" {
			t.Fatalf("event %s has unexpected source %q", event.Type, event.Source)
		}
	}

	for _, eventType := range []string{
		EventTypeRSSPollStarted,
		EventTypeRSSPollOK,
		EventTypeWebSocketConnected,
		EventTypeWebSocketMessage,
		EventTypeWebSocketPong,
		EventTypeWebSocketClosed,
	} {
		if !eventTypes[eventType] {
			t.Fatalf("expected runtime event %s", eventType)
		}
	}
}

func TestRuntimeEventsFromUpdatesSkipsClearWebSocketCloseReason(t *testing.T) {
	events := runtimeEventsFromUpdates(map[string]interface{}{
		"last_ws_close_reason": "",
	})

	if len(events) != 0 {
		t.Fatalf("expected no event for clear close reason, got %d", len(events))
	}
}
