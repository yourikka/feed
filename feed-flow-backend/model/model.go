package model

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null;size:20"` //用户名，唯一
	Password string `gorm:"not null"`                     //密码
	Avatar   string //头像
}

type Video struct {
	gorm.Model
	Title    string `gorm:"size:50"` //视频标题
	PlayUrl  string //视频播放地址
	CoverUrl string //视频封面地址
	AuthorID uint   `gorm:"index:idx_videos_author_id,priority:1"` //视频作者ID
	Author   User   `gorm:"foreignKey:AuthorID"`                   //视频作者
}
