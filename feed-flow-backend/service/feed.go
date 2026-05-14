package service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/mq"
	"github.com/yourikka/feed-flow/ranking"
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

type FeedCursor struct {
	Mode     string `json:"mode"`
	LastID   uint   `json:"last_id,omitempty"`
	HotToken string `json:"hot_token,omitempty"`
}

type FeedQuery struct {
	UserID     *uint
	ClientID   string
	Cursor     string
	LegacyID   uint
	Limit      int
	SortType   string
	FilterSeen bool
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
	ranking.RecordHotEvent(video.ID, ranking.ScorePublish)
	// 发布视频到mq
	mq.PublishVideo(video.ID)
	return nil

}

func getVideosByIDsOrdered(videoIDs []uint) ([]model.Video, error) {
	return getVideosByIDsOrderedWithCache(videoIDs)
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
	viewerKey := BuildViewerKey(userID, "")
	if viewerKey != "" {
		cachedItems := make(map[uint]FeedVideo, len(videos))
		missVideos := make([]model.Video, 0, len(videos))
		for _, video := range videos {
			if item, ok := getFeedVideoFromCache(viewerKey, video.ID); ok {
				cachedItems[video.ID] = item
				continue
			}
			missVideos = append(missVideos, video)
		}
		if len(missVideos) == 0 {
			result := make([]FeedVideo, 0, len(videos))
			for _, video := range videos {
				if item, ok := cachedItems[video.ID]; ok {
					result = append(result, item)
				}
			}
			return result, nil
		}

		freshItems, err := buildFeedVideosWithoutCache(missVideos, userID)
		if err != nil {
			return nil, err
		}
		for _, item := range freshItems {
			cachedItems[item.ID] = item
			setFeedVideoCache(viewerKey, item)
		}

		result := make([]FeedVideo, 0, len(videos))
		for _, video := range videos {
			if item, ok := cachedItems[video.ID]; ok {
				result = append(result, item)
			}
		}
		return result, nil
	}

	return buildFeedVideosWithoutCache(videos, userID)
}

func buildFeedVideosWithoutCache(videos []model.Video, userID *uint) ([]FeedVideo, error) {
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

func encodeFeedCursor(cursor FeedCursor) string {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeFeedCursor(raw string) (FeedCursor, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return FeedCursor{}, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return FeedCursor{}, false
	}
	var cursor FeedCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return FeedCursor{}, false
	}
	return cursor, true
}

func getLatestVideoFeed(userID *uint, viewerKey string, cursor uint, limit int, filterSeen bool) ([]FeedVideo, string, bool, error) {
	videoIDs, nextToken, hasMore, err := getLatestVideoIDsPage(viewerKey, cursor, 0, limit, filterSeen)
	if err != nil {
		return nil, "", false, err
	}
	videos, err := getVideosByIDsOrderedWithCache(videoIDs)
	if err != nil {
		return nil, "", false, err
	}
	feedVideos, err := buildFeedVideos(videos, userID)
	if err != nil {
		return nil, "", false, err
	}
	return feedVideos, nextToken, hasMore, nil
}

func getLatestVideoIDsPage(viewerKey string, legacyCursor uint, tokenCursor uint, limit int, filterSeen bool) ([]uint, string, bool, error) {
	limit = normalizeFeedLimit(limit)

	fetchLimit := limit + 6
	lastID := legacyCursor
	if tokenCursor > 0 {
		lastID = tokenCursor
	}
	filteredVideos := make([]model.Video, 0, limit+1)
	hasMore := false

	for attempt := 0; attempt < 3 && len(filteredVideos) < limit+1; attempt++ {
		query := config.DB.Preload("Author").Order("id desc")
		if lastID > 0 {
			query = query.Where("id < ?", lastID)
		}

		var batch []model.Video
		err := query.Limit(fetchLimit).Find(&batch).Error
		if err != nil {
			return nil, "", false, err
		}
		if len(batch) == 0 {
			break
		}

		candidates := batch
		if filterSeen {
			var filterErr error
			candidates, filterErr = FilterRecentlyExposedVideos(viewerKey, batch)
			if filterErr != nil {
				return nil, "", false, filterErr
			}
		}
		filteredVideos = append(filteredVideos, candidates...)
		lastID = batch[len(batch)-1].ID
		if len(batch) < fetchLimit {
			break
		}
	}

	if len(filteredVideos) > limit {
		hasMore = true
		filteredVideos = filteredVideos[:limit]
	}

	if !hasMore && len(filteredVideos) == limit && lastID > 0 {
		hasMore = true
	}

	nextCursor := ""
	if hasMore && len(filteredVideos) > 0 {
		nextCursor = encodeFeedCursor(FeedCursor{
			Mode:   "latest",
			LastID: filteredVideos[len(filteredVideos)-1].ID,
		})
	}

	return modelVideosToIDs(filteredVideos), nextCursor, hasMore, nil
}

func getHotVideoFeed(userID *uint, viewerKey string, cursorToken string, legacyCursor uint, limit int, filterSeen bool) ([]FeedVideo, string, bool, error) {
	videoIDs, nextCursor, hasMore, err := getHotVideoIDsPage(viewerKey, cursorToken, legacyCursor, limit, filterSeen)
	if err != nil {
		return nil, "", false, err
	}
	videos, err := getVideosByIDsOrderedWithCache(videoIDs)
	if err != nil {
		return nil, "", false, err
	}
	feedVideos, err := buildFeedVideos(videos, userID)
	if err != nil {
		return nil, "", false, err
	}
	return feedVideos, nextCursor, hasMore, nil
}

func getHotVideoIDsPage(viewerKey string, cursorToken string, legacyCursor uint, limit int, filterSeen bool) ([]uint, string, bool, error) {
	limit = normalizeFeedLimit(limit)

	offset := int(legacyCursor)
	snapshotToken := strings.TrimSpace(cursorToken)
	if snapshotToken == "" {
		snapshotToken = ranking.BuildHotSnapshotCursor(offset)
	}

	snapshot, ok := ranking.ParseHotSnapshotCursor(snapshotToken)
	if !ok || ranking.IsHotSnapshotCursorExpired(snapshot) {
		snapshotToken = ranking.BuildHotSnapshotCursor(offset)
		snapshot, ok = ranking.ParseHotSnapshotCursor(snapshotToken)
		if !ok {
			return nil, "", false, errors.New("热榜游标解析失败")
		}
	}
	if snapshot.Offset > 0 {
		offset = snapshot.Offset
	}

	videoIDs, total, err := ranking.GetHotVideoIDsByAggKey(snapshot.AggKey, offset, limit+12)
	if err != nil {
		return nil, "", false, err
	}
	if len(videoIDs) == 0 {
		return []uint{}, "", false, nil
	}

	if filterSeen {
		videoIDs, err = FilterRecentlyExposedVideoIDs(viewerKey, videoIDs)
		if err != nil {
			return nil, "", false, err
		}
	}
	if len(videoIDs) > limit {
		videoIDs = videoIDs[:limit]
	}
	if len(videoIDs) == 0 {
		return []uint{}, "", false, nil
	}

	videos, err := getVideosByIDsOrdered(videoIDs)
	if err != nil {
		return nil, "", false, err
	}
	if len(videos) == 0 {
		return []uint{}, "", false, nil
	}

	next := offset + len(videoIDs)
	hasMore := int64(next) < total
	nextCursor := ""
	if hasMore {
		nextCursor = ranking.EncodeHotSnapshotCursor(ranking.HotSnapshotCursor{
			AggKey:  snapshot.AggKey,
			Offset:  next,
			Window:  snapshot.Window,
			Created: snapshot.Created,
		})
	}
	return videoIDs, nextCursor, hasMore, nil
}

// GetVideoFeed 获取视频流数据和交互状态
func GetVideoFeed(userID *uint, cursor uint, limit int) ([]FeedVideo, uint, bool, error) {
	videos, _, hasMore, err := GetVideoFeedByQuery(FeedQuery{
		UserID:   userID,
		LegacyID: cursor,
		Limit:    limit,
		SortType: "latest",
	})
	return videos, cursor, hasMore, err
}

func GetVideoFeedBySort(userID *uint, cursor uint, limit int, sortType string) ([]FeedVideo, uint, bool, error) {
	videos, _, hasMore, err := GetVideoFeedByQuery(FeedQuery{
		UserID:   userID,
		LegacyID: cursor,
		Limit:    limit,
		SortType: sortType,
	})
	return videos, cursor, hasMore, err
}

func GetVideoFeedByQuery(query FeedQuery) ([]FeedVideo, string, bool, error) {
	viewerKey := BuildViewerKey(query.UserID, query.ClientID)
	limit := normalizeFeedLimit(query.Limit)
	cursor := FeedCursor{}
	if decoded, ok := decodeFeedCursor(query.Cursor); ok {
		cursor = decoded
	}

	sortType := strings.ToLower(strings.TrimSpace(query.SortType))
	switch sortType {
	case "", "latest":
		currentCursor := query.LegacyID
		if cursor.LastID > 0 {
			currentCursor = cursor.LastID
		}
		return getLatestVideoFeed(query.UserID, viewerKey, currentCursor, limit, query.FilterSeen)
	case "hot":
		hotToken := cursor.HotToken
		if hotToken == "" {
			hotToken = query.Cursor
		}
		videos, next, more, err := getHotVideoFeed(query.UserID, viewerKey, hotToken, query.LegacyID, limit, query.FilterSeen)
		if err != nil {
			// 热榜依赖 Redis，异常时自动降级回最新流。
			return getLatestVideoFeed(query.UserID, viewerKey, query.LegacyID, limit, query.FilterSeen)
		}
		return videos, next, more, nil
	default:
		return getLatestVideoFeed(query.UserID, viewerKey, query.LegacyID, limit, query.FilterSeen)
	}
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
