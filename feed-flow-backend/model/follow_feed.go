package model

import "gorm.io/gorm"

// FollowFeedInbox 存储推模式下预分发给用户的关注流视频ID。
type FollowFeedInbox struct {
	gorm.Model
	UserID   uint `gorm:"uniqueIndex:uk_follow_feed_inbox_user_video;index;index:idx_follow_feed_inbox_user_id,priority:1"`
	VideoID  uint `gorm:"uniqueIndex:uk_follow_feed_inbox_user_video;index"`
	AuthorID uint `gorm:"index"`
}
