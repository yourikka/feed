package mq

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	outboxBatchSize      = 32
	outboxPollInterval   = 500 * time.Millisecond
	outboxLeaseDuration  = 5 * time.Second
	outboxRetryBackoff   = 2 * time.Second
	outboxMaxErrorLength = 512
)

type OutboxMessage struct {
	Exchange   string
	RoutingKey string
	Publishing amqp091.Publishing
}

func SaveOutboxMessage(tx *gorm.DB, message OutboxMessage) error {
	if tx == nil {
		return errors.New("nil transaction")
	}

	headersJSON, err := marshalOutboxHeaders(message.Publishing.Headers)
	if err != nil {
		return err
	}

	record := model.MQOutboxMessage{
		Exchange:        message.Exchange,
		RoutingKey:      message.RoutingKey,
		ContentType:     message.Publishing.ContentType,
		ContentEncoding: message.Publishing.ContentEncoding,
		CorrelationID:   message.Publishing.CorrelationId,
		MessageID:       message.Publishing.MessageId,
		MessageType:     message.Publishing.Type,
		DeliveryMode:    uint8(message.Publishing.DeliveryMode),
		HeadersJSON:     headersJSON,
		Body:            append([]byte(nil), message.Publishing.Body...),
		AvailableAt:     time.Now(),
	}
	return tx.Create(&record).Error
}

func StartOutboxPublisher(ctx context.Context) {
	ticker := time.NewTicker(outboxPollInterval)
	defer ticker.Stop()

	for {
		if err := publishOutboxBatch(); err != nil {
			log.Printf("outbox publish batch failed: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func publishOutboxBatch() error {
	if config.DB == nil {
		return errors.New("db unavailable")
	}

	records, err := claimOutboxBatch()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	for _, record := range records {
		if err := publishClaimedOutboxRecord(record); err != nil {
			log.Printf("outbox publish failed id=%d retry=%d err=%v", record.ID, record.RetryCount, err)
		}
	}
	return nil
}

func claimOutboxBatch() ([]model.MQOutboxMessage, error) {
	now := time.Now()
	leaseExpiresAt := now.Add(outboxLeaseDuration)
	records := []model.MQOutboxMessage{}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var candidates []model.MQOutboxMessage
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("available_at <= ?", now).
			Where("lease_expires_at IS NULL OR lease_expires_at <= ?", now).
			Order("id asc").
			Limit(outboxBatchSize).
			Find(&candidates).Error
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return nil
		}

		ids := make([]uint, 0, len(candidates))
		for _, candidate := range candidates {
			ids = append(ids, candidate.ID)
		}
		if err := tx.Model(&model.MQOutboxMessage{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"lease_expires_at": leaseExpiresAt,
				"updated_at":       now,
			}).Error; err != nil {
			return err
		}

		for i := range candidates {
			candidates[i].LeaseExpiresAt = &leaseExpiresAt
		}
		records = candidates
		return nil
	})
	if err != nil {
		return nil, err
	}

	return records, nil
}

func publishClaimedOutboxRecord(record model.MQOutboxMessage) error {
	headers, err := unmarshalOutboxHeaders(record.HeadersJSON)
	if err != nil {
		releaseOutboxRecord(record.ID, record.RetryCount+1, err)
		return err
	}

	err = PublishMessage(
		record.Exchange,
		record.RoutingKey,
		amqp091.Publishing{
			ContentType:     record.ContentType,
			ContentEncoding: record.ContentEncoding,
			CorrelationId:   record.CorrelationID,
			MessageId:       record.MessageID,
			Type:            record.MessageType,
			DeliveryMode:    uint8(record.DeliveryMode),
			Headers:         headers,
			Body:            record.Body,
			Timestamp:       time.Now(),
		},
	)
	if err != nil {
		releaseOutboxRecord(record.ID, record.RetryCount+1, err)
		return err
	}

	return config.DB.Delete(&model.MQOutboxMessage{}, record.ID).Error
}

func releaseOutboxRecord(id uint, retryCount int, cause error) {
	if config.DB == nil || id == 0 {
		return
	}

	lastError := ""
	if cause != nil {
		lastError = cause.Error()
		if len(lastError) > outboxMaxErrorLength {
			lastError = lastError[:outboxMaxErrorLength]
		}
	}
	nextAvailableAt := time.Now().Add(outboxRetryBackoff)
	_ = config.DB.Model(&model.MQOutboxMessage{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"retry_count":      retryCount,
			"last_error":       lastError,
			"available_at":     nextAvailableAt,
			"lease_expires_at": nil,
			"updated_at":       time.Now(),
		}).Error
}

func marshalOutboxHeaders(headers amqp091.Table) ([]byte, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	return json.Marshal(headers)
}

func unmarshalOutboxHeaders(raw []byte) (amqp091.Table, error) {
	if len(raw) == 0 {
		return amqp091.Table{}, nil
	}
	var headers amqp091.Table
	if err := json.Unmarshal(raw, &headers); err != nil {
		return nil, err
	}
	return headers, nil
}
