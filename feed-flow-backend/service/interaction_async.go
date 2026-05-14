package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rabbitmq/amqp091-go"
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/mq"
	"gorm.io/gorm/clause"
)

const (
	interactionStateTTL = 24 * time.Hour
	interactionMaxRetry = 3
)

type interactionKind string

const (
	interactionKindLike     interactionKind = "like"
	interactionKindFavorite interactionKind = "favorite"
)

type interactionCommand struct {
	Kind         interactionKind `json:"kind"`
	UserID       uint            `json:"user_id"`
	VideoID      uint            `json:"video_id"`
	DesiredState bool            `json:"desired_state"`
	Version      int64           `json:"version"`
}

func StartInteractionWorker(ctx context.Context) {
	log.Println("interaction worker started")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		consumeCh := config.GetRabbitConsumeChannel()
		if consumeCh == nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		consumerTag := fmt.Sprintf("interaction_consumer_%d", time.Now().UnixNano())
		msgs, err := consumeCh.Consume(
			config.InteractionEventQueue,
			consumerTag,
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Printf("interaction worker consume failed: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		channelClosed := false
		for !channelClosed {
			select {
			case <-ctx.Done():
				log.Println("interaction worker received stop signal")
				if err := consumeCh.Cancel(consumerTag, false); err != nil {
					log.Printf("interaction worker cancel failed: %v", err)
				}
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Println("interaction worker message channel closed")
					channelClosed = true
					continue
				}
				handleInteractionMessage(msg)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func publishInteractionCommand(cmd interactionCommand) error {
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	return mq.PublishMessage(
		"",
		config.InteractionEventQueue,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp091.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

func handleInteractionMessage(msg amqp091.Delivery) {
	retryCount := mq.GetRetryCount(msg.Headers)

	var cmd interactionCommand
	if err := json.Unmarshal(msg.Body, &cmd); err != nil {
		log.Printf("interaction message decode failed: %v", err)
		handleInteractionFinalFailure(msg, retryCount, err)
		return
	}

	if err := validateInteractionCommand(cmd); err != nil {
		handleInteractionFinalFailure(msg, retryCount, err)
		return
	}

	stale, err := isInteractionCommandStale(cmd)
	if err != nil {
		handleInteractionConsumeFailure(msg, retryCount, err)
		return
	}
	if stale {
		if err := msg.Ack(false); err != nil {
			log.Printf("interaction ack stale command failed: %v", err)
		}
		return
	}

	if err := applyInteractionCommand(cmd); err != nil {
		handleInteractionConsumeFailure(msg, retryCount, err)
		return
	}

	if err := msg.Ack(false); err != nil {
		log.Printf("interaction ack failed: %v", err)
	}
}

func handleInteractionConsumeFailure(msg amqp091.Delivery, retryCount int, cause error) {
	if retryCount >= interactionMaxRetry {
		handleInteractionFinalFailure(msg, retryCount, cause)
		return
	}

	if err := publishInteractionRetry(msg, retryCount+1, cause); err != nil {
		log.Printf("interaction retry publish failed: %v", err)
		if nackErr := msg.Nack(false, true); nackErr != nil {
			log.Printf("interaction nack failed: %v", nackErr)
		}
		return
	}

	if err := msg.Ack(false); err != nil {
		log.Printf("interaction ack failed: %v", err)
	}
}

func handleInteractionFinalFailure(msg amqp091.Delivery, retryCount int, cause error) {
	if err := publishInteractionDLQ(msg, retryCount, cause); err != nil {
		log.Printf("interaction dlq publish failed: %v", err)
		if nackErr := msg.Nack(false, true); nackErr != nil {
			log.Printf("interaction nack failed: %v", nackErr)
		}
		return
	}

	if err := msg.Ack(false); err != nil {
		log.Printf("interaction ack failed: %v", err)
	}
}

func publishInteractionRetry(msg amqp091.Delivery, retryCount int, cause error) error {
	headers := mq.CloneHeaders(msg.Headers)
	headers["x-retry-count"] = retryCount
	headers["x-last-error"] = cause.Error()

	return mq.PublishMessage(
		"",
		config.InteractionEventRetryQueue,
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

func publishInteractionDLQ(msg amqp091.Delivery, retryCount int, cause error) error {
	headers := mq.CloneHeaders(msg.Headers)
	headers["x-retry-count"] = retryCount
	headers["x-final-error"] = cause.Error()

	return mq.PublishMessage(
		config.InteractionEventDLX,
		config.InteractionEventDLQRoutingKey,
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

func validateInteractionCommand(cmd interactionCommand) error {
	switch cmd.Kind {
	case interactionKindLike, interactionKindFavorite:
	default:
		return errors.New("unsupported interaction kind")
	}
	if cmd.UserID == 0 || cmd.VideoID == 0 {
		return errors.New("invalid interaction command")
	}
	return nil
}

func applyInteractionCommand(cmd interactionCommand) error {
	switch cmd.Kind {
	case interactionKindLike:
		return persistLikeCommand(cmd)
	case interactionKindFavorite:
		return persistFavoriteCommand(cmd)
	default:
		return errors.New("unsupported interaction kind")
	}
}

func persistLikeCommand(cmd interactionCommand) error {
	if cmd.DesiredState {
		return config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.Like{
			VideoID: cmd.VideoID,
			UserID:  cmd.UserID,
		}).Error
	}
	return config.DB.Unscoped().
		Where("video_id = ? AND user_id = ?", cmd.VideoID, cmd.UserID).
		Delete(&model.Like{}).Error
}

func persistFavoriteCommand(cmd interactionCommand) error {
	if cmd.DesiredState {
		return config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.Favorite{
			VideoID: cmd.VideoID,
			UserID:  cmd.UserID,
		}).Error
	}
	return config.DB.Unscoped().
		Where("video_id = ? AND user_id = ?", cmd.VideoID, cmd.UserID).
		Delete(&model.Favorite{}).Error
}

func interactionCacheKey(kind interactionKind, userID, videoID uint) string {
	return fmt.Sprintf("interaction:%s:%d:%d", kind, userID, videoID)
}

func parseInteractionCacheValue(raw string) (bool, int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, 0, false
	}

	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return false, 0, false
	}

	state := parts[0] == "1"
	version, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false, 0, false
	}
	return state, version, true
}

func formatInteractionCacheValue(active bool, version int64) string {
	state := "0"
	if active {
		state = "1"
	}
	return fmt.Sprintf("%s:%d", state, version)
}

func setInteractionStateCache(kind interactionKind, userID, videoID uint, active bool, version int64) {
	if config.RDB == nil || userID == 0 || videoID == 0 {
		return
	}
	_ = config.RDB.Set(
		config.Ctx,
		interactionCacheKey(kind, userID, videoID),
		formatInteractionCacheValue(active, version),
		interactionStateTTL,
	).Err()
}

func getInteractionStateFromCache(kind interactionKind, userID, videoID uint) (bool, int64, bool) {
	if config.RDB == nil || userID == 0 || videoID == 0 {
		return false, 0, false
	}
	raw, err := config.RDB.Get(config.Ctx, interactionCacheKey(kind, userID, videoID)).Result()
	if err != nil || raw == "" {
		return false, 0, false
	}
	return parseInteractionCacheValue(raw)
}

func isInteractionCommandStale(cmd interactionCommand) (bool, error) {
	if config.RDB == nil {
		return false, nil
	}
	raw, err := config.RDB.Get(config.Ctx, interactionCacheKey(cmd.Kind, cmd.UserID, cmd.VideoID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	_, currentVersion, ok := parseInteractionCacheValue(raw)
	if !ok || currentVersion <= 0 || cmd.Version <= 0 {
		return false, nil
	}
	return currentVersion > cmd.Version, nil
}

func queryInteractionState(kind interactionKind, userID, videoID uint) (bool, error) {
	if userID == 0 || videoID == 0 {
		return false, nil
	}

	var count int64
	switch kind {
	case interactionKindLike:
		if err := config.DB.Model(&model.Like{}).
			Where("user_id = ? AND video_id = ?", userID, videoID).
			Count(&count).Error; err != nil {
			return false, err
		}
	case interactionKindFavorite:
		if err := config.DB.Model(&model.Favorite{}).
			Where("user_id = ? AND video_id = ?", userID, videoID).
			Count(&count).Error; err != nil {
			return false, err
		}
	default:
		return false, errors.New("unsupported interaction kind")
	}

	return count > 0, nil
}

func getInteractionState(kind interactionKind, userID, videoID uint) (bool, int64, error) {
	if state, version, ok := getInteractionStateFromCache(kind, userID, videoID); ok {
		return state, version, nil
	}

	state, err := queryInteractionState(kind, userID, videoID)
	if err != nil {
		return false, 0, err
	}
	setInteractionStateCache(kind, userID, videoID, state, 0)
	return state, 0, nil
}

func getInteractionStatesBatch(kind interactionKind, userID uint, videoIDs []uint) (map[uint]bool, error) {
	result := make(map[uint]bool, len(videoIDs))
	if userID == 0 || len(videoIDs) == 0 {
		return result, nil
	}

	missIDs := make([]uint, 0, len(videoIDs))
	if config.RDB != nil {
		keys := make([]string, 0, len(videoIDs))
		keyToVideoID := make(map[string]uint, len(videoIDs))
		for _, videoID := range videoIDs {
			if videoID == 0 {
				continue
			}
			key := interactionCacheKey(kind, userID, videoID)
			keys = append(keys, key)
			keyToVideoID[key] = videoID
		}

		values, err := config.RDB.MGet(config.Ctx, keys...).Result()
		if err == nil && len(values) == len(keys) {
			for i, value := range values {
				videoID := keyToVideoID[keys[i]]
				if value == nil {
					missIDs = append(missIDs, videoID)
					continue
				}
				raw, ok := value.(string)
				if !ok {
					missIDs = append(missIDs, videoID)
					continue
				}
				state, _, ok := parseInteractionCacheValue(raw)
				if !ok {
					missIDs = append(missIDs, videoID)
					continue
				}
				result[videoID] = state
			}
		} else {
			missIDs = append(missIDs, videoIDs...)
		}
	} else {
		missIDs = append(missIDs, videoIDs...)
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	var rows []videoIDRow
	switch kind {
	case interactionKindLike:
		if err := config.DB.Model(&model.Like{}).
			Select("video_id").
			Where("user_id = ? AND video_id IN ?", userID, missIDs).
			Scan(&rows).Error; err != nil {
			return nil, err
		}
	case interactionKindFavorite:
		if err := config.DB.Model(&model.Favorite{}).
			Select("video_id").
			Where("user_id = ? AND video_id IN ?", userID, missIDs).
			Scan(&rows).Error; err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported interaction kind")
	}

	stateMap := idRowsToMap(rows)
	for _, videoID := range missIDs {
		active := stateMap[videoID]
		result[videoID] = active
		setInteractionStateCache(kind, userID, videoID, active, 0)
	}

	return result, nil
}
