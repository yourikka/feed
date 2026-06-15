package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
)

const (
	feedVideoCacheKeyPrefix = "feed:video:viewer:"
	feedVideoIndexKeyPrefix = "feed:video:idx:"
	feedVideoCacheTTL       = 5 * time.Minute
	feedVideoCacheTTLJitter = 90 * time.Second
	feedVideoIndexTTL       = 20 * time.Minute
)

func getFeedVideoCacheKey(viewerKey string, videoID uint) string {
	return fmt.Sprintf("%s%s:%d", feedVideoCacheKeyPrefix, viewerKey, videoID)
}

func getFeedVideoIndexKey(videoID uint) string {
	return fmt.Sprintf("%s%d", feedVideoIndexKeyPrefix, videoID)
}

func getFeedVideoCacheTTL() time.Duration {
	if feedVideoCacheTTLJitter <= 0 {
		return feedVideoCacheTTL
	}
	return feedVideoCacheTTL + time.Duration(rand.Int63n(int64(feedVideoCacheTTLJitter)+1))
}

func getFeedVideoFromCache(viewerKey string, videoID uint) (FeedVideo, bool) {
	client := config.GetRedisClient()
	if client == nil || viewerKey == "" || videoID == 0 {
		return FeedVideo{}, false
	}
	raw, err := client.Get(config.Ctx, getFeedVideoCacheKey(viewerKey, videoID)).Result()
	if err != nil || raw == "" {
		return FeedVideo{}, false
	}
	var item FeedVideo
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return FeedVideo{}, false
	}
	return item, true
}

func setFeedVideoCache(viewerKey string, item FeedVideo) {
	client := config.GetRedisClient()
	if client == nil || viewerKey == "" || item.ID == 0 {
		return
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return
	}
	cacheKey := getFeedVideoCacheKey(viewerKey, item.ID)
	indexKey := getFeedVideoIndexKey(item.ID)
	pipe := client.Pipeline()
	pipe.Set(config.Ctx, cacheKey, payload, getFeedVideoCacheTTL())
	pipe.SAdd(config.Ctx, indexKey, cacheKey)
	pipe.Expire(config.Ctx, indexKey, feedVideoIndexTTL)
	_, _ = pipe.Exec(config.Ctx)
}

func getFeedVideosFromCache(viewerKey string, videoIDs []uint) (map[uint]FeedVideo, []uint) {
	cached := make(map[uint]FeedVideo, len(videoIDs))
	client := config.GetRedisClient()
	if client == nil || viewerKey == "" || len(videoIDs) == 0 {
		return cached, videoIDs
	}

	keys := make([]string, 0, len(videoIDs))
	keyToVideoID := make(map[string]uint, len(videoIDs))
	for _, videoID := range videoIDs {
		if videoID == 0 {
			continue
		}
		key := getFeedVideoCacheKey(viewerKey, videoID)
		keys = append(keys, key)
		keyToVideoID[key] = videoID
	}
	if len(keys) == 0 {
		return cached, videoIDs
	}

	values, err := client.MGet(config.Ctx, keys...).Result()
	if err != nil || len(values) != len(keys) {
		return cached, videoIDs
	}

	missIDs := make([]uint, 0, len(videoIDs))
	seenMiss := make(map[uint]struct{}, len(videoIDs))
	for i, value := range values {
		videoID := keyToVideoID[keys[i]]
		if value == nil {
			if _, ok := seenMiss[videoID]; !ok {
				missIDs = append(missIDs, videoID)
				seenMiss[videoID] = struct{}{}
			}
			continue
		}
		raw, ok := value.(string)
		if !ok || raw == "" {
			if _, ok := seenMiss[videoID]; !ok {
				missIDs = append(missIDs, videoID)
				seenMiss[videoID] = struct{}{}
			}
			continue
		}
		var item FeedVideo
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			if _, ok := seenMiss[videoID]; !ok {
				missIDs = append(missIDs, videoID)
				seenMiss[videoID] = struct{}{}
			}
			continue
		}
		cached[videoID] = item
	}
	return cached, missIDs
}

func invalidateFeedVideoCache(viewerKey string, videoID uint) {
	client := config.GetRedisClient()
	if client == nil || viewerKey == "" || videoID == 0 {
		return
	}
	_ = client.Del(config.Ctx, getFeedVideoCacheKey(viewerKey, videoID)).Err()
}

func invalidateFeedVideoCacheForVideo(videoID uint) {
	client := config.GetRedisClient()
	if client == nil || videoID == 0 {
		return
	}
	indexKey := getFeedVideoIndexKey(videoID)
	keys, err := client.SMembers(config.Ctx, indexKey).Result()
	if err != nil || len(keys) == 0 {
		return
	}
	pipe := client.Pipeline()
	pipe.Unlink(config.Ctx, keys...)
	pipe.Del(config.Ctx, indexKey)
	_, _ = pipe.Exec(config.Ctx)
}

func invalidateFeedVideoCacheByAuthor(authorID uint) {
	if config.GetRedisClient() == nil || authorID == 0 {
		return
	}
	var videoIDs []uint
	if err := config.DB.Model(&model.Video{}).Where("author_id = ?", authorID).Pluck("id", &videoIDs).Error; err != nil {
		return
	}
	for _, videoID := range videoIDs {
		invalidateFeedVideoCacheForVideo(videoID)
	}
}
