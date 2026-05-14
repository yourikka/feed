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
	feedVideoCacheTTL       = 5 * time.Minute
	feedVideoCacheTTLJitter = 90 * time.Second
)

func getFeedVideoCacheKey(viewerKey string, videoID uint) string {
	return fmt.Sprintf("%s%s:%d", feedVideoCacheKeyPrefix, viewerKey, videoID)
}

func getFeedVideoCacheTTL() time.Duration {
	if feedVideoCacheTTLJitter <= 0 {
		return feedVideoCacheTTL
	}
	return feedVideoCacheTTL + time.Duration(rand.Int63n(int64(feedVideoCacheTTLJitter)+1))
}

func getFeedVideoFromCache(viewerKey string, videoID uint) (FeedVideo, bool) {
	if config.RDB == nil || viewerKey == "" || videoID == 0 {
		return FeedVideo{}, false
	}
	raw, err := config.RDB.Get(config.Ctx, getFeedVideoCacheKey(viewerKey, videoID)).Result()
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
	if config.RDB == nil || viewerKey == "" || item.ID == 0 {
		return
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return
	}
	_ = config.RDB.Set(config.Ctx, getFeedVideoCacheKey(viewerKey, item.ID), payload, getFeedVideoCacheTTL()).Err()
}

func invalidateFeedVideoCache(viewerKey string, videoID uint) {
	if config.RDB == nil || viewerKey == "" || videoID == 0 {
		return
	}
	_ = config.RDB.Del(config.Ctx, getFeedVideoCacheKey(viewerKey, videoID)).Err()
}

func invalidateFeedVideoCacheForVideo(videoID uint) {
	if config.RDB == nil || videoID == 0 {
		return
	}
	pattern := fmt.Sprintf("%s*:%d", feedVideoCacheKeyPrefix, videoID)
	keys, err := config.RDB.Keys(config.Ctx, pattern).Result()
	if err != nil || len(keys) == 0 {
		return
	}
	_ = config.RDB.Del(config.Ctx, keys...).Err()
}

func invalidateFeedVideoCacheByAuthor(authorID uint) {
	if config.RDB == nil || authorID == 0 {
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
