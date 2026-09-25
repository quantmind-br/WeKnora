package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CleanupStaleRunningTasks cleans up potentially leftover running task keys
// This is a debugging and maintenance tool that can be used to clean up leftover running keys caused by abnormal situations
func CleanupStaleRunningTasks(ctx context.Context, redisClient *redis.Client, keyPrefix string, maxAge time.Duration) (int, error) {
	// Get all matching keys
	keys, err := redisClient.Keys(ctx, keyPrefix+"*").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return 0, nil
	}

	// Check the TTL of each key
	var staleTasks []string
	for _, key := range keys {
		ttl, err := redisClient.TTL(ctx, key).Result()
		if err != nil {
			continue // Skip erroneous keys
		}

		// If TTL is less than 0 (never expires) or the remaining time is too long (possibly leftover), mark as stale
		if ttl < 0 || ttl > maxAge {
			staleTasks = append(staleTasks, key)
		}
	}

	if len(staleTasks) == 0 {
		return 0, nil
	}

	// Delete stale keys
	deleted, err := redisClient.Del(ctx, staleTasks...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to delete stale keys: %w", err)
	}

	return int(deleted), nil
}

// CheckRunningTaskStatus checks the status of the specified running task
func CheckRunningTaskStatus(ctx context.Context, redisClient *redis.Client, runningKey, progressKey string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Check the running key
	runningTaskID, err := redisClient.Get(ctx, runningKey).Result()
	if err != nil {
		if err == redis.Nil {
			result["running_task_exists"] = false
		} else {
			return nil, fmt.Errorf("failed to get running task: %w", err)
		}
	} else {
		result["running_task_exists"] = true
		result["running_task_id"] = runningTaskID

		// Get the TTL of the running key
		ttl, _ := redisClient.TTL(ctx, runningKey).Result()
		result["running_task_ttl"] = ttl.String()
	}

	// Check the progress key
	progressData, err := redisClient.Get(ctx, progressKey).Result()
	if err != nil {
		if err == redis.Nil {
			result["progress_exists"] = false
		} else {
			return nil, fmt.Errorf("failed to get progress: %w", err)
		}
	} else {
		result["progress_exists"] = true
		result["progress_data"] = progressData

		// Get the TTL of the progress key
		ttl, _ := redisClient.TTL(ctx, progressKey).Result()
		result["progress_ttl"] = ttl.String()
	}

	return result, nil
}
