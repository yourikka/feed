package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/mq"
)

const (
	behaviorEventMaxRetry = 3
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

		for {
			select {
			case <-ctx.Done():
				_ = consumeCh.Cancel(consumerTag, false)
				return
			case msg, ok := <-msgs:
				if !ok {
					goto RECONNECT
				}
				handleBehaviorMessage(msg)
			}
		}

	RECONNECT:
		time.Sleep(time.Second)
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

func handleBehaviorMessage(msg amqp091.Delivery) {
	retryCount := mq.GetRetryCount(msg.Headers)

	var event queuedVideoEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		handleBehaviorFinalFailure(msg, retryCount, err)
		return
	}

	if err := persistVideoEvent(event); err != nil {
		handleBehaviorConsumeFailure(msg, retryCount, err)
		return
	}
	if err := msg.Ack(false); err != nil {
		return
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
