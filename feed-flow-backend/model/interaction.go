package model

import (
	"gorm.io/gorm"
)

// 评论表
type Comment struct {
	gorm.Model
	VideoID uint   `gorm:"index"` //视频ID
	UserID  uint   `gorm:"index"` //用户ID
	Content string //评论内容
	User    User   `gorm:"foreignKey:UserID"` //评论用户
}

// 点赞表
type Like struct {
	gorm.Model
	VideoID uint `gorm:"uniqueIndex:uk_likes_video_user;index;index:idx_likes_user_video,priority:2"` //视频ID
	UserID  uint `gorm:"uniqueIndex:uk_likes_video_user;index;index:idx_likes_user_video,priority:1"` //用户ID
}

// 收藏表
type Favorite struct {
	gorm.Model
	VideoID uint `gorm:"uniqueIndex:uk_favorites_video_user;index;index:idx_favorites_user_video,priority:2"` //视频ID
	UserID  uint `gorm:"uniqueIndex:uk_favorites_video_user;index;index:idx_favorites_user_video,priority:1"` //用户ID
}

// 关注表
type Follow struct {
	gorm.Model
	UserID       uint `gorm:"uniqueIndex:uk_follows_user_target;index"` // 发起关注的用户
	TargetUserID uint `gorm:"uniqueIndex:uk_follows_user_target;index"` // 被关注的用户
}
