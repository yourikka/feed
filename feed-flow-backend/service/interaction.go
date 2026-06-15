package service

import (
	"context"
	"errors"
	"strings"
	"time"

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
func LikeVideo(ctx context.Context, videoId, userId uint) (bool, error) {
	return toggleVideoInteraction(ctx, interactionKindLike, videoId, userId)
}

// 收藏/取消收藏
func FavoriteVideo(ctx context.Context, videoId, userId uint) (bool, error) {
	return toggleVideoInteraction(ctx, interactionKindFavorite, videoId, userId)
}

// 关注/取消关注
func FollowUser(userId, targetUserId uint) (bool, error) {
	if userId == targetUserId {
		return false, errors.New("不能关注自己")
	}

	res := config.DB.Unscoped().Where("user_id = ? AND target_user_id = ?", userId, targetUserId).Delete(&model.Follow{})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected > 0 {
		invalidateFeedVideoCacheByAuthor(targetUserId)
		return false, nil
	}

	err := config.DB.Create(&model.Follow{UserID: userId, TargetUserID: targetUserId}).Error
	if isDuplicateEntry(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	invalidateFeedVideoCacheByAuthor(targetUserId)
	return true, nil
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
		adjustVideoStatsCache(context.Background(), videoId, videoStatsCommentField, 1)
		invalidateFeedVideoCacheForVideo(videoId)
		ranking.RecordHotEvent(context.Background(), videoId, ranking.ScoreComment)
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
	adjustVideoStatsCache(context.Background(), comment.VideoID, videoStatsCommentField, -1)
	invalidateFeedVideoCacheForVideo(comment.VideoID)
	ranking.RecordHotEvent(context.Background(), comment.VideoID, -ranking.ScoreComment)
	return nil
}

func toggleVideoInteraction(ctx context.Context, kind interactionKind, videoID, userID uint) (bool, error) {
	currentState, currentVersion, err := getInteractionState(kind, userID, videoID)
	if err != nil {
		return false, err
	}

	nextState := !currentState
	nextVersion := currentVersion + time.Now().UnixNano()

	cmd := interactionCommand{
		Kind:         kind,
		UserID:       userID,
		VideoID:      videoID,
		DesiredState: nextState,
		Version:      nextVersion,
	}
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		return publishInteractionCommandTx(tx, cmd)
	}); err != nil {
		if syncErr := applyInteractionCommand(cmd); syncErr != nil {
			return false, syncErr
		}
	}

	setInteractionStateCache(ctx, kind, userID, videoID, nextState, nextVersion)

	switch kind {
	case interactionKindLike:
		applyInteractionSideEffects(ctx, videoID, nextState, videoStatsLikeField, ranking.ScoreLike)
	case interactionKindFavorite:
		applyInteractionSideEffects(ctx, videoID, nextState, videoStatsFavoriteField, ranking.ScoreFavorite)
	default:
		return false, errors.New("不支持的交互类型")
	}

	return nextState, nil
}

func applyInteractionSideEffects(ctx context.Context, videoID uint, active bool, statsField string, scoreDelta float64) {
	delta := int64(-1)
	score := -scoreDelta
	if active {
		delta = 1
		score = scoreDelta
	}
	adjustVideoStatsCache(ctx, videoID, statsField, delta)
	invalidateFeedVideoCacheForVideo(videoID)
	ranking.RecordHotEvent(ctx, videoID, score)
}
