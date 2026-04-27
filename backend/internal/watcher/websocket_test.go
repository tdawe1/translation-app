package watcher

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWebSocketMonitorProcessesGengoCollectionPayload(t *testing.T) {
	var msg wsMessage
	err := json.Unmarshal([]byte(`{
		"type": "available_collection",
		"collection": {
			"id": 9876,
			"lc_src": "Japanese",
			"lc_tgt": "English",
			"rewards": "25.50"
		}
	}`), &msg)
	require.NoError(t, err)

	monitor := NewWebSocketMonitor(uuid.New(), "session", "", "789487", false)
	jobChan := make(chan Job, 1)

	err = monitor.processMessage([]byte(`{
		"type": "available_collection",
		"collection": {
			"id": 9876,
			"lc_src": "Japanese",
			"lc_tgt": "English",
			"rewards": "25.50"
		}
	}`), jobChan)
	require.NoError(t, err)

	select {
	case job := <-jobChan:
		require.Equal(t, "9876", job.ID)
		require.Equal(t, "Japanese > English", job.Title)
		require.Equal(t, 25.50, job.Reward)
		require.Equal(t, "https://gengo.com/t/jobs/details/9876", job.URL)
		require.Equal(t, "websocket", job.Source)
	default:
		t.Fatal("expected websocket job to be published")
	}
}

func TestWebSocketMonitorKeepsTopLevelJobIDCompatibility(t *testing.T) {
	monitor := NewWebSocketMonitor(uuid.New(), "session", "", "789487", false)
	jobChan := make(chan Job, 1)

	err := monitor.processMessage([]byte(`{"type":"available_collection","job_id":"job-123"}`), jobChan)
	require.NoError(t, err)

	select {
	case job := <-jobChan:
		require.Equal(t, "job-123", job.ID)
		require.Equal(t, "Job job-123", job.Title)
		require.Equal(t, "https://gengo.com/t/jobs/details/job-123", job.URL)
	default:
		t.Fatal("expected websocket job to be published")
	}
}
