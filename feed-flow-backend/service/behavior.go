package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/ranking"
)

const (
	EventExposure  = "exposure"
	EventPlayStart = "play_start"
	EventProgress  = "play_progress"
	EventPlayEnd   = "play_finish"
	EventPause     = "pause"
	EventSkip      = "skip"

	behaviorRequestKeyPrefix  = "feed:event:req:"
	exposureDedupeKeyPrefix   = "feed:exposure:dedupe:"
	recentExposureKeyPrefix   = "feed:user:exposed:"
	behaviorRequestKeyTTL     = 24 * time.Hour
	exposureDedupeBucketTTL   = 6 * time.Hour
	recentExposureLookback    = 72 * time.Hour
	recentExposureSetTTL      = 7 * 24 * time.Hour
	maxRecentExposureScanSize = 500
)

type TrackVideoEventInput struct {
	UserID     *uint
	ClientID   string
	VideoID    uint
	EventType  string
	RequestID  string
	SessionID  string
	ProgressMs int64
	DurationMs int64
	PositionMs int64
}

type TrackVideoEventResult struct {
	Accepted bool
	Deduped  bool
}

func BuildViewerKey(userID *uint, clientID string) string {
	if userID != nil && *userID > 0 {
		return "u:" + strconv.FormatUint(uint64(*userID), 10)
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return ""
	}
	return "c:" + clientID
}

func TrackVideoEvent(ctx context.Context, input TrackVideoEventInput) (TrackVideoEventResult, error) {
	eventType := strings.TrimSpace(strings.ToLower(input.EventType))
	if !isSupportedVideoEvent(eventType) {
		return TrackVideoEventResult{}, errors.New("不支持的事件类型")
	}
	if input.VideoID == 0 {
		return TrackVideoEventResult{}, errors.New("video_id 参数错误")
	}

	viewerKey := BuildViewerKey(input.UserID, input.ClientID)
	if viewerKey == "" {
		return TrackVideoEventResult{}, errors.New("缺少用户身份或 client_id")
	}

	requestID := strings.TrimSpace(input.RequestID)
	if requestID != "" {
		deduped, err := behaviorRequestExists(requestID)
		if err != nil {
			return TrackVideoEventResult{}, err
		}
		if deduped {
			return TrackVideoEventResult{Accepted: false, Deduped: true}, nil
		}
	}

	now := time.Now()
	if eventType == EventExposure {
		unique, err := isUniqueExposure(ctx, viewerKey, input.VideoID, now)
		if err != nil {
			return TrackVideoEventResult{}, err
		}
		if !unique {
			return TrackVideoEventResult{Accepted: false, Deduped: true}, nil
		}
	}

	event := queuedVideoEvent{
		EventID:    "",
		UserID:     derefUint(input.UserID),
		ViewerKey:  viewerKey,
		VideoID:    input.VideoID,
		EventType:  eventType,
		RequestID:  requestID,
		SessionID:  strings.TrimSpace(input.SessionID),
		ProgressMs: sanitizePositiveInt64(input.ProgressMs),
		DurationMs: sanitizePositiveInt64(input.DurationMs),
		PositionMs: sanitizePositiveInt64(input.PositionMs),
	}
	event.EventID = buildBehaviorEventID(event)

	if !enqueueVideoEventWithContext(ctx, event) {
		if err := persistAcceptedVideoEvent(ctx, TrackVideoEventInput{
			UserID:     uintPtr(event.UserID),
			VideoID:    event.VideoID,
			EventType:  event.EventType,
			RequestID:  event.RequestID,
			SessionID:  event.SessionID,
			ProgressMs: event.ProgressMs,
			DurationMs: event.DurationMs,
			PositionMs: event.PositionMs,
		}, event.ViewerKey); err != nil {
			return TrackVideoEventResult{}, err
		}
		if err := finalizeAcceptedVideoEvent(ctx, requestID, eventType, viewerKey, input.VideoID, now); err != nil {
			return TrackVideoEventResult{}, err
		}
		return TrackVideoEventResult{Accepted: true}, nil
	}
	if err := finalizeAcceptedVideoEvent(ctx, requestID, eventType, viewerKey, input.VideoID, now); err != nil {
		return TrackVideoEventResult{}, err
	}
	return TrackVideoEventResult{Accepted: true}, nil
}

func persistAcceptedVideoEvent(ctx context.Context, input TrackVideoEventInput, viewerKey string) error {
	eventType := strings.TrimSpace(strings.ToLower(input.EventType))
	event := queuedVideoEvent{
		UserID:     derefUint(input.UserID),
		ViewerKey:  viewerKey,
		VideoID:    input.VideoID,
		EventType:  eventType,
		RequestID:  strings.TrimSpace(input.RequestID),
		SessionID:  strings.TrimSpace(input.SessionID),
		ProgressMs: sanitizePositiveInt64(input.ProgressMs),
		DurationMs: sanitizePositiveInt64(input.DurationMs),
		PositionMs: sanitizePositiveInt64(input.PositionMs),
	}
	event.EventID = buildBehaviorEventID(event)
	if err := config.DB.Create(&model.VideoBehaviorEvent{
		EventID:    event.EventID,
		UserID:     event.UserID,
		ViewerKey:  event.ViewerKey,
		VideoID:    event.VideoID,
		EventType:  event.EventType,
		RequestID:  event.RequestID,
		SessionID:  event.SessionID,
		ProgressMs: event.ProgressMs,
		DurationMs: event.DurationMs,
		PositionMs: event.PositionMs,
	}).Error; err != nil {
		return err
	}
	ranking.RecordHotEvent(ctx, input.VideoID, scoreDeltaForEvent(eventType, input.ProgressMs, input.DurationMs, input.PositionMs))
	return nil
}

func FilterRecentlyExposedVideoIDs(viewerKey string, videoIDs []uint) ([]uint, error) {
	if viewerKey == "" || len(videoIDs) == 0 {
		return videoIDs, nil
	}
	exposedSet, err := getRecentExposureSet(viewerKey, max(maxRecentExposureScanSize, len(videoIDs)*5))
	if err != nil {
		return nil, err
	}
	if len(exposedSet) == 0 {
		return videoIDs, nil
	}

	filtered := make([]uint, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		if _, ok := exposedSet[videoID]; ok {
			continue
		}
		filtered = append(filtered, videoID)
	}
	return filtered, nil
}

func FilterRecentlyExposedVideos(viewerKey string, videos []model.Video) ([]model.Video, error) {
	if viewerKey == "" || len(videos) == 0 {
		return videos, nil
	}
	videoIDs := make([]uint, 0, len(videos))
	for _, video := range videos {
		videoIDs = append(videoIDs, video.ID)
	}
	filteredIDs, err := FilterRecentlyExposedVideoIDs(viewerKey, videoIDs)
	if err != nil {
		return nil, err
	}
	if len(filteredIDs) == len(videoIDs) {
		return videos, nil
	}

	allowed := make(map[uint]struct{}, len(filteredIDs))
	for _, videoID := range filteredIDs {
		allowed[videoID] = struct{}{}
	}

	filtered := make([]model.Video, 0, len(filteredIDs))
	for _, video := range videos {
		if _, ok := allowed[video.ID]; ok {
			filtered = append(filtered, video)
		}
	}
	return filtered, nil
}

func isSupportedVideoEvent(eventType string) bool {
	switch eventType {
	case EventExposure, EventPlayStart, EventProgress, EventPlayEnd, EventPause, EventSkip:
		return true
	default:
		return false
	}
}

func behaviorRequestExists(requestID string) (bool, error) {
	client := config.GetRedisClient()
	if client == nil || requestID == "" {
		return false, nil
	}
	redisCtx, cancel := config.WithRedisTimeout(context.Background())
	defer cancel()
	exists, err := client.Exists(redisCtx, behaviorRequestKeyPrefix+requestID).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func isUniqueExposure(ctx context.Context, viewerKey string, videoID uint, now time.Time) (bool, error) {
	client := config.GetRedisClient()
	if client == nil {
		return true, nil
	}
	bucket := now.UTC().Format("2006010215")
	key := fmt.Sprintf("%s%s:%d:%s", exposureDedupeKeyPrefix, viewerKey, videoID, bucket)
	redisCtx, cancel := config.WithRedisTimeout(ctx)
	defer cancel()
	exists, err := client.Exists(redisCtx, key).Result()
	if err != nil {
		return false, err
	}
	return exists == 0, nil
}

func recordBehaviorRequest(ctx context.Context, requestID string) error {
	client := config.GetRedisClient()
	if client == nil || requestID == "" {
		return nil
	}
	redisCtx, cancel := config.WithRedisTimeout(ctx)
	defer cancel()
	return client.Set(redisCtx, behaviorRequestKeyPrefix+requestID, "1", behaviorRequestKeyTTL).Err()
}

func recordUniqueExposure(ctx context.Context, viewerKey string, videoID uint, now time.Time) error {
	client := config.GetRedisClient()
	if client == nil || viewerKey == "" || videoID == 0 {
		return nil
	}
	bucket := now.UTC().Format("2006010215")
	key := fmt.Sprintf("%s%s:%d:%s", exposureDedupeKeyPrefix, viewerKey, videoID, bucket)
	redisCtx, cancel := config.WithRedisTimeout(ctx)
	defer cancel()
	return client.Set(redisCtx, key, "1", exposureDedupeBucketTTL).Err()
}

func recordRecentExposure(ctx context.Context, viewerKey string, videoID uint, now time.Time) {
	client := config.GetRedisClient()
	if client == nil || viewerKey == "" || videoID == 0 {
		return
	}
	key := recentExposureKeyPrefix + viewerKey
	member := strconv.FormatUint(uint64(videoID), 10)
	score := float64(now.Unix())
	redisCtx, cancel := config.WithRedisTimeout(ctx)
	defer cancel()
	pipe := client.Pipeline()
	pipe.ZAdd(redisCtx, key, &redis.Z{Score: score, Member: member})
	pipe.ZRemRangeByScore(redisCtx, key, "-inf", strconv.FormatInt(now.Add(-recentExposureLookback).Unix(), 10))
	pipe.Expire(redisCtx, key, recentExposureSetTTL)
	_, _ = pipe.Exec(redisCtx)
}

func getRecentExposureSet(viewerKey string, limit int) (map[uint]struct{}, error) {
	client := config.GetRedisClient()
	if client == nil || viewerKey == "" {
		return map[uint]struct{}{}, nil
	}
	if limit <= 0 {
		limit = maxRecentExposureScanSize
	}
	key := recentExposureKeyPrefix + viewerKey
	minScore := strconv.FormatInt(time.Now().Add(-recentExposureLookback).Unix(), 10)
	redisCtx, cancel := config.WithRedisTimeout(context.Background())
	defer cancel()
	rows, err := client.ZRevRangeByScore(redisCtx, key, &redis.ZRangeBy{
		Max:   "+inf",
		Min:   minScore,
		Count: int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		videoID, parseErr := strconv.ParseUint(row, 10, 64)
		if parseErr != nil || videoID == 0 {
			continue
		}
		result[uint(videoID)] = struct{}{}
	}
	return result, nil
}

func scoreDeltaForEvent(eventType string, progressMs, durationMs, positionMs int64) float64 {
	switch eventType {
	case EventExposure:
		return 0.2
	case EventPlayStart:
		return 0.8
	case EventPlayEnd:
		return 2.6
	case EventProgress:
		if durationMs > 0 {
			ratio := float64(progressMs) / float64(durationMs)
			switch {
			case ratio >= 0.85:
				return 1.6
			case ratio >= 0.5:
				return 0.8
			}
		}
		if progressMs >= 5000 {
			return 0.4
		}
	case EventSkip:
		if positionMs > 0 && positionMs < 3000 {
			return -0.6
		}
		return -0.2
	}
	return 0
}

func sanitizePositiveInt64(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

func derefUint(v *uint) uint {
	if v == nil {
		return 0
	}
	return *v
}

func finalizeAcceptedVideoEvent(ctx context.Context, requestID, eventType, viewerKey string, videoID uint, now time.Time) error {
	if err := recordBehaviorRequest(ctx, requestID); err != nil {
		return err
	}
	if eventType != EventExposure {
		return nil
	}
	if err := recordUniqueExposure(ctx, viewerKey, videoID, now); err != nil {
		return err
	}
	recordRecentExposure(ctx, viewerKey, videoID, now)
	return nil
}
