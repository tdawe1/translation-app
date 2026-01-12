package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslationJobQueue_EnqueueJob(t *testing.T) {
	redisClient := RequireRedis(t)
	ctx := context.Background()

	queuePrefix := "translation:queue:"
	jobsKey := "translation:jobs"

	t.Run("enqueue translation job to Redis", func(t *testing.T) {
		jobID := uuid.New().String()
		job := map[string]interface{}{
			"id":          jobID,
			"state":       "pending",
			"source_path": "/watch/incoming/test.docx",
			"priority":    "normal",
			"created_at":  time.Now().UTC().Format(time.RFC3339),
			"metadata": map[string]interface{}{
				"user_id":     uuid.New().String(),
				"source_lang": "ja",
				"target_lang": "en",
			},
		}

		jobData, err := json.Marshal(job)
		require.NoError(t, err)

		err = redisClient.HSet(ctx, jobsKey, jobID, jobData).Err()
		require.NoError(t, err)

		priority := 5.0
		err = redisClient.ZAdd(ctx, queuePrefix+"normal", redis.Z{Score: priority, Member: jobID}).Err()
		require.NoError(t, err)

		stored, err := redisClient.HGet(ctx, jobsKey, jobID).Result()
		require.NoError(t, err)

		var storedJob map[string]interface{}
		err = json.Unmarshal([]byte(stored), &storedJob)
		require.NoError(t, err)

		assert.Equal(t, jobID, storedJob["id"])
		assert.Equal(t, "pending", storedJob["state"])
		assert.Equal(t, "/watch/incoming/test.docx", storedJob["source_path"])

		redisClient.HDel(ctx, jobsKey, jobID)
		redisClient.ZRem(ctx, queuePrefix+"normal", jobID)
	})

	t.Run("publish job progress event", func(t *testing.T) {
		channel := "translation:progress"
		jobID := uuid.New().String()

		pubsub := redisClient.Subscribe(ctx, channel)
		defer pubsub.Close()

		_, err := pubsub.Receive(ctx)
		require.NoError(t, err)

		progressEvent := map[string]interface{}{
			"job_id":   jobID,
			"progress": 50,
			"message":  "Translating segment 5 of 10",
		}
		eventData, _ := json.Marshal(progressEvent)

		err = redisClient.Publish(ctx, channel, eventData).Err()
		require.NoError(t, err)

		msg, err := pubsub.ReceiveMessage(ctx)
		require.NoError(t, err)

		var received map[string]interface{}
		err = json.Unmarshal([]byte(msg.Payload), &received)
		require.NoError(t, err)

		assert.Equal(t, jobID, received["job_id"])
		assert.Equal(t, float64(50), received["progress"])
	})

	t.Run("job state transitions", func(t *testing.T) {
		jobID := uuid.New().String()
		job := map[string]interface{}{
			"id":    jobID,
			"state": "pending",
		}

		jobData, _ := json.Marshal(job)
		err := redisClient.HSet(ctx, jobsKey, jobID, jobData).Err()
		require.NoError(t, err)

		states := []string{"processing", "translating", "review_pending", "completed"}
		for _, state := range states {
			job["state"] = state
			jobData, _ = json.Marshal(job)
			err = redisClient.HSet(ctx, jobsKey, jobID, jobData).Err()
			require.NoError(t, err)

			stored, err := redisClient.HGet(ctx, jobsKey, jobID).Result()
			require.NoError(t, err)

			var storedJob map[string]interface{}
			json.Unmarshal([]byte(stored), &storedJob)
			assert.Equal(t, state, storedJob["state"])
		}

		redisClient.HDel(ctx, jobsKey, jobID)
	})

	t.Run("checkpoint save and restore", func(t *testing.T) {
		jobID := uuid.New().String()
		checkpointKey := "translation:checkpoints:" + jobID

		checkpoint := map[string]interface{}{
			"job_id":        jobID,
			"checkpoint_id": uuid.New().String(),
			"progress":      35,
			"data": map[string]interface{}{
				"completed_segments": []int{1, 2, 3, 4, 5},
				"current_segment":    6,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		checkpointData, _ := json.Marshal(checkpoint)
		err := redisClient.Set(ctx, checkpointKey, checkpointData, time.Hour).Err()
		require.NoError(t, err)

		stored, err := redisClient.Get(ctx, checkpointKey).Result()
		require.NoError(t, err)

		var storedCheckpoint map[string]interface{}
		json.Unmarshal([]byte(stored), &storedCheckpoint)

		assert.Equal(t, jobID, storedCheckpoint["job_id"])
		assert.Equal(t, float64(35), storedCheckpoint["progress"])

		redisClient.Del(ctx, checkpointKey)
	})
}
