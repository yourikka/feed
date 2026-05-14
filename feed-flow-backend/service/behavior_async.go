package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yourikka/feed-flow/config"
)

const (
	behaviorEventQueueKey   = "feed:behavior:queue"
	behaviorWorkerBlockTime = 3 * time.Second
)

type queuedVideoEvent struct {
	UserID     uint   `json:"user_id"`
	ViewerKey  string `json:"viewer_key"`
	VideoID    uint   `json:"video_id"`
	EventType  string `json:"event_type"`
	RequestID  string `json:"request_id"`
	SessionID  string `json:"session_id"`
	ProgressMs int64  `json:"progress_ms"`
	DurationMs int64  `json:"duration_ms"`
	PositionMs int64  `json:"position_ms"`
}

func enqueueVideoEvent(event queuedVideoEvent) bool {
	if config.RDB == nil {
		return false
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return false
	}
	return config.RDB.RPush(config.Ctx, behaviorEventQueueKey, payload).Err() == nil
}

func StartBehaviorEventWorker(ctx context.Context) {
	if config.RDB == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		rows, err := config.RDB.BLPop(ctx, behaviorWorkerBlockTime, behaviorEventQueueKey).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if len(rows) != 2 {
			continue
		}

		var event queuedVideoEvent
		if err := json.Unmarshal([]byte(rows[1]), &event); err != nil {
			continue
		}
		_ = persistVideoEvent(event)
	}
}

func persistVideoEvent(event queuedVideoEvent) error {
	return persistAcceptedVideoEvent(TrackVideoEventInput{
		UserID:     uintPtr(event.UserID),
		VideoID:    event.VideoID,
		EventType:  event.EventType,
		RequestID:  event.RequestID,
		SessionID:  event.SessionID,
		ProgressMs: event.ProgressMs,
		DurationMs: event.DurationMs,
		PositionMs: event.PositionMs,
	}, event.ViewerKey)
}

func uintPtr(v uint) *uint {
	if v == 0 {
		return nil
	}
	value := v
	return &value
}
