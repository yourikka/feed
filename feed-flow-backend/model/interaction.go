package model

import (
	"gorm.io/gorm"
)

// 评论表
type Comment struct {
	gorm.Model
	VideoID uint   //视频ID
	UserID  uint   //用户ID
	Content string //评论内容
	User    User   `gorm:"foreignKey:UserID"` //评论用户
}

// 点赞表
type Like struct {
	gorm.Model
	VideoID uint //视频ID
	UserID  uint //用户ID
}

// 收藏表
type Favorite struct {
	gorm.Model
	VideoID uint //视频ID
	UserID  uint //用户ID
}

// 关注表
type Follow struct {
	gorm.Model
	UserID       uint // 发起关注的用户
	TargetUserID uint // 被关注的用户
}
