package ranking

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/yourikka/feed-flow/config"
)

const (
	ScorePublish  = 5.0
	ScoreLike     = 1.0
	ScoreFavorite = 3.0
	ScoreComment  = 2.0

	hotBucketPrefix = "feed:hot:bucket:"
	hotAggPrefix    = "feed:hot:agg:"
	hotAggTTL       = 30 * time.Second
)

func RecordHotEvent(videoID uint, delta float64) {
	client := config.GetRedisClient()
	if videoID == 0 || delta == 0 || client == nil {
		return
	}

	member := strconv.FormatUint(uint64(videoID), 10)
	bucketKey := currentBucketKey()

	pipe := client.Pipeline()
	pipe.ZIncrBy(config.Ctx, bucketKey, delta, member)
	pipe.Expire(config.Ctx, bucketKey, getBucketTTL())
	if _, err := pipe.Exec(config.Ctx); err != nil {
		log.Printf("record hot event failed: %v", err)
	}
}

func GetHotVideoIDs(offset, limit int) ([]uint, int64, error) {
	return GetHotVideoIDsByAggKey(buildAggKey(), offset, limit)
}

func GetHotVideoIDsByAggKey(aggKey string, offset, limit int) ([]uint, int64, error) {
	if limit <= 0 {
		return []uint{}, 0, nil
	}
	client := config.GetRedisClient()
	if client == nil {
		return nil, 0, fmt.Errorf("redis unavailable")
	}

	keys := buildSlidingWindowBucketKeys()
	if len(keys) == 0 {
		return []uint{}, 0, nil
	}

	if aggKey == "" {
		aggKey = buildAggKey()
	}
	if err := EnsureHotAggKey(aggKey); err != nil {
		return nil, 0, err
	}

	total, err := client.ZCard(config.Ctx, aggKey).Result()
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []uint{}, 0, nil
	}

	start := int64(offset)
	end := int64(offset + limit - 1)
	rows, err := client.ZRevRange(config.Ctx, aggKey, start, end).Result()
	if err != nil {
		return nil, 0, err
	}

	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		parsed, parseErr := strconv.ParseUint(row, 10, 64)
		if parseErr != nil || parsed == 0 {
			continue
		}
		ids = append(ids, uint(parsed))
	}
	return ids, total, nil
}

func currentBucketKey() string {
	return fmt.Sprintf("%s%d", hotBucketPrefix, time.Now().Unix()/3600)
}

func buildAggKey() string {
	nowHour := time.Now().Unix() / 3600
	return fmt.Sprintf("%s%d:%d", hotAggPrefix, nowHour, getHotWindowHours())
}

func buildSlidingWindowBucketKeys() []string {
	window := getHotWindowHours()
	nowHour := time.Now().Unix() / 3600
	keys := make([]string, 0, window)
	for i := 0; i < window; i++ {
		keys = append(keys, fmt.Sprintf("%s%d", hotBucketPrefix, nowHour-int64(i)))
	}
	return keys
}

func getHotWindowHours() int {
	const fallback = 24
	raw := os.Getenv("FEED_HOT_WINDOW_HOURS")
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 || parsed > 168 {
		return fallback
	}
	return parsed
}

func getBucketTTL() time.Duration {
	return time.Duration(getHotWindowHours()+2) * time.Hour
}
