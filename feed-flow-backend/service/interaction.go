package service

import (
	"errors"
	"strings"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/ranking"
	"gorm.io/gorm"
)

func normalizeCommentLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

// 点赞/取消点赞
func LikeVideo(videoId, userId uint) (bool, error) {
	var like model.Like
	err := config.DB.Where("video_id = ? AND user_id = ?", videoId, userId).First(&like).Error
	if err == nil {
		if delErr := config.DB.Unscoped().Where("video_id = ? AND user_id = ?", videoId, userId).Delete(&model.Like{}).Error; delErr != nil {
			return false, delErr
		}
		adjustVideoStatsCache(videoId, videoStatsLikeField, -1)
		invalidateFeedVideoCacheForVideo(videoId)
		ranking.RecordHotEvent(videoId, -ranking.ScoreLike)
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	if delErr := config.DB.Unscoped().Where("video_id = ? AND user_id = ?", videoId, userId).Delete(&model.Like{}).Error; delErr != nil {
		return false, delErr
	}

	err = config.DB.Create(&model.Like{VideoID: videoId, UserID: userId}).Error
	if isDuplicateEntry(err) {
		invalidateVideoStatsCache(videoId)
		return true, nil
	}
	if err == nil {
		adjustVideoStatsCache(videoId, videoStatsLikeField, 1)
		invalidateFeedVideoCacheForVideo(videoId)
		ranking.RecordHotEvent(videoId, ranking.ScoreLike)
	}
	return true, err
}

// 收藏/取消收藏
func FavoriteVideo(videoId, userId uint) (bool, error) {
	var favorite model.Favorite
	err := config.DB.Where("video_id = ? AND user_id = ?", videoId, userId).First(&favorite).Error
	if err == nil {
		if delErr := config.DB.Unscoped().Where("video_id = ? AND user_id = ?", videoId, userId).Delete(&model.Favorite{}).Error; delErr != nil {
			return false, delErr
		}
		adjustVideoStatsCache(videoId, videoStatsFavoriteField, -1)
		invalidateFeedVideoCacheForVideo(videoId)
		ranking.RecordHotEvent(videoId, -ranking.ScoreFavorite)
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	if delErr := config.DB.Unscoped().Where("video_id = ? AND user_id = ?", videoId, userId).Delete(&model.Favorite{}).Error; delErr != nil {
		return false, delErr
	}

	err = config.DB.Create(&model.Favorite{VideoID: videoId, UserID: userId}).Error
	if isDuplicateEntry(err) {
		invalidateVideoStatsCache(videoId)
		return true, nil
	}
	if err == nil {
		adjustVideoStatsCache(videoId, videoStatsFavoriteField, 1)
		invalidateFeedVideoCacheForVideo(videoId)
		ranking.RecordHotEvent(videoId, ranking.ScoreFavorite)
	}
	return true, err
}

// 关注/取消关注
func FollowUser(userId, targetUserId uint) (bool, error) {
	if userId == targetUserId {
		return false, errors.New("不能关注自己")
	}

	var follow model.Follow
	err := config.DB.Where("user_id = ? AND target_user_id = ?", userId, targetUserId).First(&follow).Error
	if err == nil {
		if delErr := config.DB.Unscoped().Where("user_id = ? AND target_user_id = ?", userId, targetUserId).Delete(&model.Follow{}).Error; delErr != nil {
			return false, delErr
		}
		invalidateFeedVideoCacheByAuthor(targetUserId)
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	if delErr := config.DB.Unscoped().Where("user_id = ? AND target_user_id = ?", userId, targetUserId).Delete(&model.Follow{}).Error; delErr != nil {
		return false, delErr
	}

	err = config.DB.Create(&model.Follow{UserID: userId, TargetUserID: targetUserId}).Error
	if isDuplicateEntry(err) {
		return true, nil
	}
	if err == nil {
		invalidateFeedVideoCacheByAuthor(targetUserId)
	}
	return true, err
}

// 评论视频
func CommentVideo(videoId, userId uint, content string) error {
	if content == "" {
		return errors.New("评论内容不能为空")
	}
	err := config.DB.Create(&model.Comment{
		VideoID: videoId,
		UserID:  userId,
		Content: content,
	}).Error
	if err == nil {
		adjustVideoStatsCache(videoId, videoStatsCommentField, 1)
		invalidateFeedVideoCacheForVideo(videoId)
		ranking.RecordHotEvent(videoId, ranking.ScoreComment)
	}
	return err
}

// 获取视频的评论列表
func GetComments(videoId uint, cursor uint, limit int) ([]model.Comment, uint, bool, error) {
	limit = normalizeCommentLimit(limit)
	query := config.DB.Preload("User").Where("video_id = ?", videoId).Order("id desc")
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	var comments []model.Comment
	err := query.Limit(limit + 1).Find(&comments).Error
	if err != nil {
		return nil, 0, false, err
	}

	hasMore := len(comments) > limit
	if hasMore {
		comments = comments[:limit]
	}

	nextCursor := uint(0)
	if hasMore && len(comments) > 0 {
		nextCursor = comments[len(comments)-1].ID
	}

	return comments, nextCursor, hasMore, nil
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

	if err := config.DB.Delete(&model.Comment{}, commentID).Error; err != nil {
		return err
	}
	adjustVideoStatsCache(comment.VideoID, videoStatsCommentField, -1)
	invalidateFeedVideoCacheForVideo(comment.VideoID)
	ranking.RecordHotEvent(comment.VideoID, -ranking.ScoreComment)
	return nil
}
