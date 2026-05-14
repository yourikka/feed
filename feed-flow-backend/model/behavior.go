package model

import "gorm.io/gorm"

// VideoBehaviorEvent 记录 feed 播放链路中的关键行为事件。
type VideoBehaviorEvent struct {
	gorm.Model
	UserID     uint   `gorm:"index"`
	ViewerKey  string `gorm:"size:96;index"`
	VideoID    uint   `gorm:"index"`
	EventType  string `gorm:"size:32;index"`
	RequestID  string `gorm:"size:64;index"`
	SessionID  string `gorm:"size:64;index"`
	ProgressMs int64
	DurationMs int64
	PositionMs int64
}
