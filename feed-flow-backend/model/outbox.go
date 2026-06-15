package model

import "time"

// MQOutboxMessage 持久化待投递的 MQ 消息，成功发布后会被删除。
type MQOutboxMessage struct {
	ID              uint `gorm:"primaryKey"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Exchange        string     `gorm:"size:128;not null"`
	RoutingKey      string     `gorm:"size:128;not null"`
	ContentType     string     `gorm:"size:64"`
	ContentEncoding string     `gorm:"size:64"`
	CorrelationID   string     `gorm:"column:correlation_id;size:128"`
	MessageID       string     `gorm:"column:message_id;size:128;index"`
	MessageType     string     `gorm:"column:message_type;size:64"`
	DeliveryMode    uint8      `gorm:"not null;default:2"`
	HeadersJSON     []byte     `gorm:"column:headers_json;type:json"`
	Body            []byte     `gorm:"type:longblob;not null"`
	AvailableAt     time.Time  `gorm:"index:idx_mq_outbox_available,priority:1"`
	LeaseExpiresAt  *time.Time `gorm:"index:idx_mq_outbox_lease,priority:1"`
	RetryCount      int        `gorm:"not null;default:0"`
	LastError       string     `gorm:"size:512"`
}
