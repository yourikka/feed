package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/mq"
	"github.com/yourikka/feed-flow/ranking"
)

const (
	behaviorEventMaxRetry      = 3
	behaviorEventBatchSize     = 64
	behaviorEventBatchFlush    = 200 * time.Millisecond
	behaviorEventInsertBatchSz = 128
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
	payload, err := json.Marshal(event)
	if err != nil {
		return false
	}
	headers := amqp091.Table{
		"x-event-type": event.EventType,
	}
	err = mq.PublishMessage(
		"",
		config.BehaviorEventQueue,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			Headers:      headers,
			DeliveryMode: amqp091.Persistent,
			Timestamp:    time.Now(),
		},
	)
	return err == nil
}

func StartBehaviorEventWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		consumeCh := config.GetRabbitConsumeChannel()
		if consumeCh == nil {
			time.Sleep(time.Second)
			continue
		}

		consumerTag := fmt.Sprintf("behavior_event_consumer_%d", time.Now().UnixNano())
		msgs, err := consumeCh.Consume(
			config.BehaviorEventQueue,
			consumerTag,
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		ticker := time.NewTicker(behaviorEventBatchFlush)
		batch := make([]behaviorPendingMessage, 0, behaviorEventBatchSize)
		flush := func() {
			if len(batch) == 0 {
				return
			}
			flushBehaviorBatch(batch)
			batch = batch[:0]
		}

		for {
			select {
			case <-ctx.Done():
				flush()
				ticker.Stop()
				_ = consumeCh.Cancel(consumerTag, false)
				return
			case <-ticker.C:
				flush()
			case msg, ok := <-msgs:
				if !ok {
					flush()
					ticker.Stop()
					goto RECONNECT
				}
				retryCount := mq.GetRetryCount(msg.Headers)
				var event queuedVideoEvent
				if err := json.Unmarshal(msg.Body, &event); err != nil {
					handleBehaviorFinalFailure(msg, retryCount, err)
					continue
				}
				batch = append(batch, behaviorPendingMessage{
					msg:        msg,
					retryCount: retryCount,
					event:      event,
				})
				if len(batch) >= behaviorEventBatchSize {
					flush()
				}
			}
		}

	RECONNECT:
		time.Sleep(time.Second)
	}
}

type behaviorPendingMessage struct {
	msg        amqp091.Delivery
	retryCount int
	event      queuedVideoEvent
}

func flushBehaviorBatch(batch []behaviorPendingMessage) {
	if len(batch) == 0 {
		return
	}
	videoIDs := collectBehaviorVideoIDs(batch)
	existingVideoIDs, err := loadExistingVideoIDSet(videoIDs)
	if err != nil {
		for _, item := range batch {
			handleBehaviorConsumeFailure(item.msg, item.retryCount, err)
		}
		return
	}

	models := make([]model.VideoBehaviorEvent, 0, len(batch))
	accepted := make([]behaviorPendingMessage, 0, len(batch))
	for _, item := range batch {
		if item.event.VideoID == 0 {
			handleBehaviorFinalFailure(item.msg, item.retryCount, fmt.Errorf("invalid video id"))
			continue
		}
		if _, ok := existingVideoIDs[item.event.VideoID]; !ok {
			// 非法或已删除视频事件直接丢弃并 ack。
			_ = item.msg.Ack(false)
			continue
		}
		models = append(models, toBehaviorModel(item.event))
		accepted = append(accepted, item)
	}

	if len(models) > 0 {
		if err := config.DB.CreateInBatches(models, behaviorEventInsertBatchSz).Error; err != nil {
			for _, item := range accepted {
				handleBehaviorConsumeFailure(item.msg, item.retryCount, err)
			}
			return
		}
	}

	for _, item := range accepted {
		ranking.RecordHotEvent(
			item.event.VideoID,
			scoreDeltaForEvent(item.event.EventType, item.event.ProgressMs, item.event.DurationMs, item.event.PositionMs),
		)
		_ = item.msg.Ack(false)
	}
}

func collectBehaviorVideoIDs(batch []behaviorPendingMessage) []uint {
	seen := make(map[uint]struct{}, len(batch))
	ids := make([]uint, 0, len(batch))
	for _, item := range batch {
		videoID := item.event.VideoID
		if videoID == 0 {
			continue
		}
		if _, ok := seen[videoID]; ok {
			continue
		}
		seen[videoID] = struct{}{}
		ids = append(ids, videoID)
	}
	return ids
}

func loadExistingVideoIDSet(videoIDs []uint) (map[uint]struct{}, error) {
	result := make(map[uint]struct{}, len(videoIDs))
	if len(videoIDs) == 0 {
		return result, nil
	}

	var rows []videoIDRow
	if err := config.DB.Model(&model.Video{}).
		Select("id as video_id").
		Where("id IN ?", videoIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row.VideoID == 0 {
			continue
		}
		result[row.VideoID] = struct{}{}
	}
	return result, nil
}

func toBehaviorModel(event queuedVideoEvent) model.VideoBehaviorEvent {
	return model.VideoBehaviorEvent{
		UserID:     event.UserID,
		ViewerKey:  strings.TrimSpace(event.ViewerKey),
		VideoID:    event.VideoID,
		EventType:  strings.TrimSpace(strings.ToLower(event.EventType)),
		RequestID:  strings.TrimSpace(event.RequestID),
		SessionID:  strings.TrimSpace(event.SessionID),
		ProgressMs: sanitizePositiveInt64(event.ProgressMs),
		DurationMs: sanitizePositiveInt64(event.DurationMs),
		PositionMs: sanitizePositiveInt64(event.PositionMs),
	}
}

func handleBehaviorConsumeFailure(msg amqp091.Delivery, retryCount int, cause error) {
	if retryCount >= behaviorEventMaxRetry {
		handleBehaviorFinalFailure(msg, retryCount, cause)
		return
	}
	if err := publishBehaviorRetryMessage(msg, retryCount+1, cause); err != nil {
		_ = msg.Nack(false, true)
		return
	}
	_ = msg.Ack(false)
}

func handleBehaviorFinalFailure(msg amqp091.Delivery, retryCount int, cause error) {
	if err := publishBehaviorDLQMessage(msg, retryCount, cause); err != nil {
		_ = msg.Nack(false, true)
		return
	}
	_ = msg.Ack(false)
}

func publishBehaviorRetryMessage(msg amqp091.Delivery, retryCount int, cause error) error {
	headers := mq.CloneHeaders(msg.Headers)
	headers["x-retry-count"] = retryCount
	headers["x-last-error"] = cause.Error()
	return mq.PublishMessage(
		"",
		config.BehaviorEventRetryQueue,
		amqp091.Publishing{
			ContentType:     msg.ContentType,
			Body:            msg.Body,
			Headers:         headers,
			DeliveryMode:    amqp091.Persistent,
			CorrelationId:   msg.CorrelationId,
			ContentEncoding: msg.ContentEncoding,
			MessageId:       msg.MessageId,
			Type:            msg.Type,
			Timestamp:       time.Now(),
		},
	)
}

func publishBehaviorDLQMessage(msg amqp091.Delivery, retryCount int, cause error) error {
	headers := mq.CloneHeaders(msg.Headers)
	headers["x-retry-count"] = retryCount
	headers["x-final-error"] = cause.Error()
	return mq.PublishMessage(
		config.BehaviorEventDLX,
		config.BehaviorEventDLQRoutingKey,
		amqp091.Publishing{
			ContentType:     msg.ContentType,
			Body:            msg.Body,
			Headers:         headers,
			DeliveryMode:    amqp091.Persistent,
			CorrelationId:   msg.CorrelationId,
			ContentEncoding: msg.ContentEncoding,
			MessageId:       msg.MessageId,
			Type:            msg.Type,
			Timestamp:       time.Now(),
		},
	)
}

func uintPtr(v uint) *uint {
	if v == 0 {
		return nil
	}
	value := v
	return &value
}
