package ranking

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/yourikka/feed-flow/config"
)

const hotAggReadTTL = 20 * time.Second
const hotAggRefreshInterval = 15 * time.Second

func EnsureHotAggKey(aggKey string) error {
	client := config.GetRedisClient()
	if client == nil {
		return redis.Nil
	}
	exists, err := client.Exists(config.Ctx, aggKey).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return client.Expire(config.Ctx, aggKey, hotAggReadTTL).Err()
	}

	keys := buildSlidingWindowBucketKeys()
	if len(keys) == 0 {
		return nil
	}
	if err := client.ZUnionStore(config.Ctx, aggKey, &redis.ZStore{Keys: keys}).Err(); err != nil {
		return err
	}
	return client.Expire(config.Ctx, aggKey, hotAggReadTTL).Err()
}

func RefreshHotAggKeyNow() error {
	return EnsureHotAggKey(buildAggKey())
}

func StartHotAggRefreshWorker(ctx context.Context) {
	if config.GetRedisClient() == nil {
		return
	}
	_ = RefreshHotAggKeyNow()
	ticker := time.NewTicker(hotAggRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = RefreshHotAggKeyNow()
		}
	}
}
