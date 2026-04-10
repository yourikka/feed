package service

import (
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/mq"
)

type FeedAuthor struct {
	ID       uint   `json:"ID"`
	Username string `json:"Username"`
	Avatar   string `json:"Avatar"`
}

type FeedVideo struct {
	ID            uint       `json:"ID"`
	Title         string     `json:"Title"`
	PlayUrl       string     `json:"PlayUrl"`
	CoverUrl      string     `json:"CoverUrl"`
	Author        FeedAuthor `json:"Author"`
	LikeCount     int64      `json:"LikeCount"`
	CommentCount  int64      `json:"CommentCount"`
	FavoriteCount int64      `json:"FavoriteCount"`
	IsLiked       bool       `json:"IsLiked"`
	IsFavorited   bool       `json:"IsFavorited"`
	IsFollowing   bool       `json:"IsFollowing"`
}

// PublishVideo 发布视频
func PublishVideo(title, playUrl, coverUrl string, authorId uint) error {
	video := model.Video{
		Title:    title,
		PlayUrl:  playUrl,
		CoverUrl: coverUrl,
		AuthorID: authorId,
	}
	// 保存视频到数据库
	if err := config.DB.Create(&video).Error; err != nil {
		return err
	}
	// 发布视频到mq
	mq.PublishVideo(video.ID)
	return nil

}

func buildFeedVideos(videos []model.Video, userID *uint) []FeedVideo {
	feedVideos := make([]FeedVideo, 0, len(videos))
	for _, video := range videos {
		item := FeedVideo{
			ID:       video.ID,
			Title:    video.Title,
			PlayUrl:  video.PlayUrl,
			CoverUrl: video.CoverUrl,
			Author: FeedAuthor{
				ID:       video.Author.ID,
				Username: video.Author.Username,
				Avatar:   video.Author.Avatar,
			},
		}

		config.DB.Model(&model.Like{}).Where("video_id = ?", video.ID).Count(&item.LikeCount)
		config.DB.Model(&model.Comment{}).Where("video_id = ?", video.ID).Count(&item.CommentCount)
		config.DB.Model(&model.Favorite{}).Where("video_id = ?", video.ID).Count(&item.FavoriteCount)

		if userID != nil {
			var likedCount int64
			var favoriteCount int64
			var followCount int64
			config.DB.Model(&model.Like{}).Where("video_id = ? AND user_id = ?", video.ID, *userID).Count(&likedCount)
			config.DB.Model(&model.Favorite{}).Where("video_id = ? AND user_id = ?", video.ID, *userID).Count(&favoriteCount)
			if video.Author.ID != *userID {
				config.DB.Model(&model.Follow{}).Where("user_id = ? AND target_user_id = ?", *userID, video.Author.ID).Count(&followCount)
			}
			item.IsLiked = likedCount > 0
			item.IsFavorited = favoriteCount > 0
			item.IsFollowing = followCount > 0
		}

		feedVideos = append(feedVideos, item)
	}

	return feedVideos
}

// GetVideoFeed 获取视频流数据和交互状态
func GetVideoFeed(userID *uint) ([]FeedVideo, error) {
	var videos []model.Video
	err := config.DB.Preload("Author").Order("created_at desc").Limit(10).Find(&videos).Error
	if err != nil {
		return nil, err
	}

	return buildFeedVideos(videos, userID), nil
}

// GetUserVideoList 获取用户作品列表
func GetUserVideoList(targetUserID uint, currentUserID *uint) ([]FeedVideo, error) {
	var videos []model.Video
	err := config.DB.Preload("Author").Where("author_id = ?", targetUserID).Order("created_at desc").Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return buildFeedVideos(videos, currentUserID), nil
}
