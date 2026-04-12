package service

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
)

type videoStats struct {
	LikeCount     int64
	CommentCount  int64
	FavoriteCount int64
}

const (
	videoStatsCacheKeyPrefix = "video:stats:"
	videoStatsLikeField      = "like_count"
	videoStatsCommentField   = "comment_count"
	videoStatsFavoriteField  = "favorite_count"
	videoStatsCacheTTL       = 10 * time.Minute
	videoStatsCacheTTLJitter = 2 * time.Minute
)

type videoStatsBatchLoadCall struct {
	done   chan struct{}
	result map[uint]videoStats
	err    error
}

var videoStatsBatchLoadGroup = struct {
	mu    sync.Mutex
	calls map[string]*videoStatsBatchLoadCall
}{
	calls: make(map[string]*videoStatsBatchLoadCall),
}

func videoStatsCacheKey(videoID uint) string {
	return fmt.Sprintf("%s%d", videoStatsCacheKeyPrefix, videoID)
}

func normalizeVideoIDs(videoIDs []uint) []uint {
	if len(videoIDs) == 0 {
		return nil
	}

	normalized := make([]uint, 0, len(videoIDs))
	seen := make(map[uint]struct{}, len(videoIDs))
	for _, videoID := range videoIDs {
		if videoID == 0 {
			continue
		}
		if _, ok := seen[videoID]; ok {
			continue
		}
		seen[videoID] = struct{}{}
		normalized = append(normalized, videoID)
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i] < normalized[j]
	})
	return normalized
}

func buildVideoStatsBatchLoadKey(videoIDs []uint) string {
	normalized := normalizeVideoIDs(videoIDs)
	if len(normalized) == 0 {
		return ""
	}

	parts := make([]string, 0, len(normalized))
	for _, videoID := range normalized {
		parts = append(parts, strconv.FormatUint(uint64(videoID), 10))
	}
	return strings.Join(parts, ",")
}

func getVideoStatsCacheTTL() time.Duration {
	if videoStatsCacheTTLJitter <= 0 {
		return videoStatsCacheTTL
	}
	return videoStatsCacheTTL + time.Duration(rand.Int63n(int64(videoStatsCacheTTLJitter)+1))
}

func parseInt64FromCacheValue(v any) (int64, bool) {
	if v == nil {
		return 0, false
	}
	switch value := v.(type) {
	case int64:
		return value, true
	case int:
		return int64(value), true
	case string:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case []byte:
		parsed, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func getVideoStatsFromCache(videoID uint) (videoStats, bool) {
	if config.RDB == nil || videoID == 0 {
		return videoStats{}, false
	}

	key := videoStatsCacheKey(videoID)
	values, err := config.RDB.HMGet(config.Ctx, key, videoStatsLikeField, videoStatsCommentField, videoStatsFavoriteField).Result()
	if err != nil || len(values) != 3 {
		return videoStats{}, false
	}

	likeCount, ok1 := parseInt64FromCacheValue(values[0])
	commentCount, ok2 := parseInt64FromCacheValue(values[1])
	favoriteCount, ok3 := parseInt64FromCacheValue(values[2])
	if !ok1 || !ok2 || !ok3 {
		return videoStats{}, false
	}

	return videoStats{
		LikeCount:     likeCount,
		CommentCount:  commentCount,
		FavoriteCount: favoriteCount,
	}, true
}

func setVideoStatsCache(videoID uint, stats videoStats) {
	if config.RDB == nil || videoID == 0 {
		return
	}

	if stats.LikeCount < 0 {
		stats.LikeCount = 0
	}
	if stats.CommentCount < 0 {
		stats.CommentCount = 0
	}
	if stats.FavoriteCount < 0 {
		stats.FavoriteCount = 0
	}

	key := videoStatsCacheKey(videoID)
	_ = config.RDB.HSet(config.Ctx, key, map[string]any{
		videoStatsLikeField:     stats.LikeCount,
		videoStatsCommentField:  stats.CommentCount,
		videoStatsFavoriteField: stats.FavoriteCount,
	}).Err()
	_ = config.RDB.Expire(config.Ctx, key, getVideoStatsCacheTTL()).Err()
}

func adjustVideoStatsCache(videoID uint, field string, delta int64) {
	if config.RDB == nil || videoID == 0 || delta == 0 {
		return
	}

	key := videoStatsCacheKey(videoID)
	newVal, err := config.RDB.HIncrBy(config.Ctx, key, field, delta).Result()
	if err != nil {
		return
	}
	if newVal < 0 {
		_ = config.RDB.HSet(config.Ctx, key, field, 0).Err()
	}
	_ = config.RDB.Expire(config.Ctx, key, getVideoStatsCacheTTL()).Err()
}

func invalidateVideoStatsCache(videoID uint) {
	if config.RDB == nil || videoID == 0 {
		return
	}
	_ = config.RDB.Del(config.Ctx, videoStatsCacheKey(videoID)).Err()
}

func queryVideoStats(videoIDs []uint) (map[uint]videoStats, error) {
	videoIDs = normalizeVideoIDs(videoIDs)
	stats := make(map[uint]videoStats, len(videoIDs))
	for _, videoID := range videoIDs {
		stats[videoID] = videoStats{}
	}

	var likeRows []videoCountRow
	if err := config.DB.Model(&model.Like{}).
		Select("video_id, COUNT(*) AS count").
		Where("video_id IN ?", videoIDs).
		Group("video_id").
		Scan(&likeRows).Error; err != nil {
		return nil, err
	}

	var commentRows []videoCountRow
	if err := config.DB.Model(&model.Comment{}).
		Select("video_id, COUNT(*) AS count").
		Where("video_id IN ?", videoIDs).
		Group("video_id").
		Scan(&commentRows).Error; err != nil {
		return nil, err
	}

	var favoriteRows []videoCountRow
	if err := config.DB.Model(&model.Favorite{}).
		Select("video_id, COUNT(*) AS count").
		Where("video_id IN ?", videoIDs).
		Group("video_id").
		Scan(&favoriteRows).Error; err != nil {
		return nil, err
	}

	likeMap := listToCountMap(likeRows)
	commentMap := listToCountMap(commentRows)
	favoriteMap := listToCountMap(favoriteRows)

	for _, videoID := range videoIDs {
		stats[videoID] = videoStats{
			LikeCount:     likeMap[videoID],
			CommentCount:  commentMap[videoID],
			FavoriteCount: favoriteMap[videoID],
		}
	}

	return stats, nil
}

func loadAndCacheVideoStats(videoIDs []uint) (map[uint]videoStats, error) {
	videoIDs = normalizeVideoIDs(videoIDs)
	if len(videoIDs) == 0 {
		return map[uint]videoStats{}, nil
	}

	loadKey := buildVideoStatsBatchLoadKey(videoIDs)

	videoStatsBatchLoadGroup.mu.Lock()
	if call, ok := videoStatsBatchLoadGroup.calls[loadKey]; ok {
		videoStatsBatchLoadGroup.mu.Unlock()
		<-call.done
		return call.result, call.err
	}

	call := &videoStatsBatchLoadCall{done: make(chan struct{})}
	videoStatsBatchLoadGroup.calls[loadKey] = call
	videoStatsBatchLoadGroup.mu.Unlock()

	stats, err := queryVideoStats(videoIDs)
	if err == nil {
		for _, videoID := range videoIDs {
			setVideoStatsCache(videoID, stats[videoID])
		}
	}

	call.result = stats
	call.err = err

	videoStatsBatchLoadGroup.mu.Lock()
	delete(videoStatsBatchLoadGroup.calls, loadKey)
	videoStatsBatchLoadGroup.mu.Unlock()
	close(call.done)

	return stats, err
}

func getVideoStatsBatch(videoIDs []uint) (map[uint]videoStats, error) {
	videoIDs = normalizeVideoIDs(videoIDs)
	if len(videoIDs) == 0 {
		return map[uint]videoStats{}, nil
	}

	result := make(map[uint]videoStats, len(videoIDs))
	missIDs := make([]uint, 0, len(videoIDs))

	for _, videoID := range videoIDs {
		if stats, ok := getVideoStatsFromCache(videoID); ok {
			result[videoID] = stats
			continue
		}
		missIDs = append(missIDs, videoID)
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	dbStats, err := loadAndCacheVideoStats(missIDs)
	if err != nil {
		return nil, err
	}

	for _, videoID := range missIDs {
		result[videoID] = dbStats[videoID]
	}

	return result, nil
}
