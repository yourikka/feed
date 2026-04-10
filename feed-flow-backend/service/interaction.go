package service

import (
	"errors"

	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"gorm.io/gorm"
)

// 点赞/取消点赞
func LikeVideo(videoId, userId uint) (bool, error) {
	var like model.Like
	err := config.DB.Where("video_id = ? AND user_id = ?", videoId, userId).First(&like).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, config.DB.Create(&model.Like{VideoID: videoId, UserID: userId}).Error
	}
	if err != nil {
		return false, err
	}
	return false, config.DB.Where("video_id = ? AND user_id = ?", videoId, userId).Delete(&model.Like{}).Error
}

// 收藏/取消收藏
func FavoriteVideo(videoId, userId uint) (bool, error) {
	var favorite model.Favorite
	err := config.DB.Where("video_id = ? AND user_id = ?", videoId, userId).First(&favorite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, config.DB.Create(&model.Favorite{VideoID: videoId, UserID: userId}).Error
	}
	if err != nil {
		return false, err
	}
	return false, config.DB.Where("video_id = ? AND user_id = ?", videoId, userId).Delete(&model.Favorite{}).Error
}

// 关注/取消关注
func FollowUser(userId, targetUserId uint) (bool, error) {
	if userId == targetUserId {
		return false, errors.New("不能关注自己")
	}

	var follow model.Follow
	err := config.DB.Where("user_id = ? AND target_user_id = ?", userId, targetUserId).First(&follow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, config.DB.Create(&model.Follow{UserID: userId, TargetUserID: targetUserId}).Error
	}
	if err != nil {
		return false, err
	}
	return false, config.DB.Where("user_id = ? AND target_user_id = ?", userId, targetUserId).Delete(&model.Follow{}).Error
}

// 评论视频
func CommentVideo(videoId, userId uint, content string) error {
	if content == "" {
		return errors.New("评论内容不能为空")
	}
	return config.DB.Create(&model.Comment{
		VideoID: videoId,
		UserID:  userId,
		Content: content,
	}).Error
}

// 获取视频的评论列表
func GetComments(videoId uint) ([]model.Comment, error) {
	var comments []model.Comment
	err := config.DB.Preload("User").Where("video_id = ?", videoId).Order("created_at desc").Find(&comments).Error
	return comments, err
}

// 删除评论
func DeleteComment(commentID, operatorID uint) error {
	if commentID == 0 {
		return errors.New("评论不存在")
	}

	var comment model.Comment
	if err := config.DB.Where("id = ?", commentID).First(&comment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("评论不存在")
		}
		return err
	}

	if comment.UserID != operatorID {
		var video model.Video
		if err := config.DB.Where("id = ?", comment.VideoID).First(&video).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("视频不存在")
			}
			return err
		}
		if video.AuthorID != operatorID {
			return errors.New("无权删除该评论")
		}
	}

	return config.DB.Delete(&model.Comment{}, commentID).Error
}
