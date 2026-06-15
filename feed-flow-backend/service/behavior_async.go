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
	"gorm.io/gorm/clause"
)

const (
	behaviorEventMaxRetry      = 3
	behaviorEventBatchSize     = 64
	behaviorEventBatchFlush    = 200 * time.Millisecond
	behaviorEventInsertBatchSz = 128
)

type queuedVideoEvent struct {
	EventID    string `json:"event_id"`
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
	return enqueueVideoEventWithContext(context.Background(), event)
}

func enqueueVideoEventWithContext(ctx context.Context, event queuedVideoEvent) bool {
	payload, err := json.Marshal(event)
	if err != nil {
		return false
	}
	headers := amqp091.Table{
		"x-event-type": event.EventType,
	}
	err = mq.PublishMessage(
		ctx,
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
			_ = item.msg.Ack(false)
			continue
		}
		models = append(models, toBehaviorModel(item.event))
		accepted = append(accepted, item)
	}

	insertedEventIDs := map[string]struct{}{}
	if len(models) > 0 {
		existingBeforeInsert, err := loadBehaviorEventIDSetFromEvents(accepted)
		if err != nil {
			for _, item := range accepted {
				handleBehaviorConsumeFailure(item.msg, item.retryCount, err)
			}
			return
		}
		if err := config.DB.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(models, behaviorEventInsertBatchSz).Error; err != nil {
			for _, item := range accepted {
				handleBehaviorConsumeFailure(item.msg, item.retryCount, err)
			}
			return
		}
		for _, item := range accepted {
			eventID := buildBehaviorEventID(item.event)
			if eventID == "" {
				continue
			}
			if _, existed := existingBeforeInsert[eventID]; existed {
				continue
			}
			insertedEventIDs[eventID] = struct{}{}
		}
	}

	for _, item := range accepted {
		if !shouldScoreBehaviorEvent(item.event, insertedEventIDs) {
			_ = item.msg.Ack(false)
			continue
		}
		ranking.RecordHotEvent(
			context.Background(),
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
		EventID:    buildBehaviorEventID(event),
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
		context.Background(),
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
		context.Background(),
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

func loadBehaviorEventIDSetFromEvents(events []behaviorPendingMessage) (map[string]struct{}, error) {
	eventIDs := make([]string, 0, len(events))
	for _, item := range events {
		eventID := buildBehaviorEventID(item.event)
		if eventID == "" {
			continue
		}
		eventIDs = append(eventIDs, eventID)
	}
	result := make(map[string]struct{}, len(eventIDs))
	if len(eventIDs) == 0 {
		return result, nil
	}
	var rows []string
	if err := config.DB.Model(&model.VideoBehaviorEvent{}).
		Where("event_id IN ?", eventIDs).
		Pluck("event_id", &rows).Error; err != nil {
		return nil, err
	}
	for _, eventID := range rows {
		eventID = strings.TrimSpace(eventID)
		if eventID == "" {
			continue
		}
		result[eventID] = struct{}{}
	}
	return result, nil
}

func shouldScoreBehaviorEvent(event queuedVideoEvent, insertedEventIDs map[string]struct{}) bool {
	eventID := buildBehaviorEventID(event)
	if eventID == "" {
		return true
	}
	_, ok := insertedEventIDs[eventID]
	return ok
}

func buildBehaviorEventID(event queuedVideoEvent) string {
	if trimmed := strings.TrimSpace(event.EventID); trimmed != "" {
		return trimmed
	}
	requestID := strings.TrimSpace(event.RequestID)
	if requestID != "" {
		return "req:" + requestID
	}
	return fmt.Sprintf(
		"evt:%d:%s:%d:%s:%d:%d:%d",
		event.UserID,
		strings.TrimSpace(event.ViewerKey),
		event.VideoID,
		strings.TrimSpace(strings.ToLower(event.EventType)),
		event.ProgressMs,
		event.DurationMs,
		event.PositionMs,
	)
}
