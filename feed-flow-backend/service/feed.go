package service

import (
	"errors"

	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/mq"
	"github.com/yourikka/feed-flow/util"
	"gorm.io/gorm"
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

type videoCountRow struct {
	VideoID uint
	Count   int64
}

type videoIDRow struct {
	VideoID uint
}

type followIDRow struct {
	TargetUserID uint
}

func normalizeFeedLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 30 {
		return 30
	}
	return limit
}

func listToCountMap(rows []videoCountRow) map[uint]int64 {
	result := make(map[uint]int64, len(rows))
	for _, row := range rows {
		result[row.VideoID] = row.Count
	}
	return result
}

func idRowsToMap(rows []videoIDRow) map[uint]bool {
	result := make(map[uint]bool, len(rows))
	for _, row := range rows {
		result[row.VideoID] = true
	}
	return result
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

// DeleteVideo 删除作者自己的视频及其关联数据
func DeleteVideo(videoID, operatorID uint) error {
	if videoID == 0 {
		return errors.New("视频不存在")
	}

	var video model.Video
	if err := config.DB.Where("id = ?", videoID).First(&video).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("视频不存在")
		}
		return err
	}

	if video.AuthorID != operatorID {
		return errors.New("无权删除该视频")
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("video_id = ?", videoID).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("video_id = ?", videoID).Delete(&model.Like{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("video_id = ?", videoID).Delete(&model.Favorite{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&video).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := util.DeleteUploadedFile(video.PlayUrl); err != nil {
		return err
	}
	if err := util.DeleteUploadedFile(video.CoverUrl); err != nil {
		return err
	}
	invalidateVideoStatsCache(video.ID)

	return nil
}

func buildFeedVideos(videos []model.Video, userID *uint) ([]FeedVideo, error) {
	if len(videos) == 0 {
		return []FeedVideo{}, nil
	}

	videoIDs := make([]uint, 0, len(videos))
	authorIDs := make([]uint, 0, len(videos))
	authorSet := make(map[uint]bool, len(videos))
	for _, video := range videos {
		videoIDs = append(videoIDs, video.ID)
		if !authorSet[video.Author.ID] {
			authorSet[video.Author.ID] = true
			authorIDs = append(authorIDs, video.Author.ID)
		}
	}

	videoStatsMap, err := getVideoStatsBatch(videoIDs)
	if err != nil {
		return nil, err
	}

	likedMap := map[uint]bool{}
	favoritedMap := map[uint]bool{}
	followingMap := map[uint]bool{}

	if userID != nil {
		var likedRows []videoIDRow
		if err := config.DB.Model(&model.Like{}).
			Select("video_id").
			Where("user_id = ? AND video_id IN ?", *userID, videoIDs).
			Scan(&likedRows).Error; err != nil {
			return nil, err
		}

		var favoritedRows []videoIDRow
		if err := config.DB.Model(&model.Favorite{}).
			Select("video_id").
			Where("user_id = ? AND video_id IN ?", *userID, videoIDs).
			Scan(&favoritedRows).Error; err != nil {
			return nil, err
		}

		likedMap = idRowsToMap(likedRows)
		favoritedMap = idRowsToMap(favoritedRows)

		if len(authorIDs) > 0 {
			var followRows []followIDRow
			if err := config.DB.Model(&model.Follow{}).
				Select("target_user_id").
				Where("user_id = ? AND target_user_id IN ?", *userID, authorIDs).
				Scan(&followRows).Error; err != nil {
				return nil, err
			}
			followingMap = make(map[uint]bool, len(followRows))
			for _, row := range followRows {
				followingMap[row.TargetUserID] = true
			}
		}
	}

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
			LikeCount:     videoStatsMap[video.ID].LikeCount,
			CommentCount:  videoStatsMap[video.ID].CommentCount,
			FavoriteCount: videoStatsMap[video.ID].FavoriteCount,
		}

		if userID != nil {
			item.IsLiked = likedMap[video.ID]
			item.IsFavorited = favoritedMap[video.ID]
			item.IsFollowing = video.Author.ID != *userID && followingMap[video.Author.ID]
		}

		feedVideos = append(feedVideos, item)
	}

	return feedVideos, nil
}

// GetVideoFeed 获取视频流数据和交互状态
func GetVideoFeed(userID *uint, cursor uint, limit int) ([]FeedVideo, uint, bool, error) {
	limit = normalizeFeedLimit(limit)

	query := config.DB.Preload("Author").Order("id desc")
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	var videos []model.Video
	err := query.Limit(limit + 1).Find(&videos).Error
	if err != nil {
		return nil, 0, false, err
	}

	hasMore := len(videos) > limit
	if hasMore {
		videos = videos[:limit]
	}

	feedVideos, err := buildFeedVideos(videos, userID)
	if err != nil {
		return nil, 0, false, err
	}

	nextCursor := uint(0)
	if hasMore && len(videos) > 0 {
		nextCursor = videos[len(videos)-1].ID
	}

	return feedVideos, nextCursor, hasMore, nil
}

// GetUserVideoList 获取用户作品列表
func GetUserVideoList(targetUserID uint, currentUserID *uint, cursor uint, limit int) ([]FeedVideo, uint, bool, error) {
	limit = normalizeFeedLimit(limit)

	query := config.DB.Preload("Author").Where("author_id = ?", targetUserID).Order("id desc")
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	var videos []model.Video
	err := query.Limit(limit + 1).Find(&videos).Error
	if err != nil {
		return nil, 0, false, err
	}

	hasMore := len(videos) > limit
	if hasMore {
		videos = videos[:limit]
	}

	feedVideos, err := buildFeedVideos(videos, currentUserID)
	if err != nil {
		return nil, 0, false, err
	}

	nextCursor := uint(0)
	if hasMore && len(videos) > 0 {
		nextCursor = videos[len(videos)-1].ID
	}

	return feedVideos, nextCursor, hasMore, nil
}
