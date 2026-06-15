package model

import "gorm.io/gorm"

// VideoBehaviorEvent 记录 feed 播放链路中的关键行为事件。
type VideoBehaviorEvent struct {
	gorm.Model
	EventID    string `gorm:"size:96;not null;uniqueIndex:uk_behavior_event_id"`
	UserID     uint   `gorm:"index"`
	ViewerKey  string `gorm:"size:96;index"`
	VideoID    uint   `gorm:"index:idx_behavior_video_created,priority:1"`
	EventType  string `gorm:"size:32;index"`
	RequestID  string `gorm:"size:64;index"`
	SessionID  string `gorm:"size:64;index"`
	ProgressMs int64
	DurationMs int64
	PositionMs int64
}
