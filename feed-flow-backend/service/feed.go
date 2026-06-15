package service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/mq"
	"github.com/yourikka/feed-flow/ranking"
	"github.com/yourikka/feed-flow/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	Mode         string `json:"mode"`
	LastID       uint   `json:"last_id,omitempty"`
	HotToken     string `json:"hot_token,omitempty"`
	FollowInbox  uint   `json:"follow_inbox,omitempty"`
	FollowOutbox uint   `json:"follow_outbox,omitempty"`
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
	var video model.Video
	return config.DB.Transaction(func(tx *gorm.DB) error {
		video = model.Video{
			Title:    title,
			PlayUrl:  playUrl,
			CoverUrl: coverUrl,
			AuthorID: authorId,
		}
		if err := tx.Create(&video).Error; err != nil {
			return err
		}
		if err := fanoutVideoToFollowersTx(tx, video.ID, authorId); err != nil {
			return err
		}
		return mq.SaveOutboxMessage(tx, mq.BuildVideoPublishOutboxMessage(video.ID))
	})
}

func getFollowPushThreshold() int64 {
	raw := strings.TrimSpace(os.Getenv("FEED_FOLLOW_PUSH_MAX_FANS"))
	if raw == "" {
		return 2000
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		return 2000
	}
	return parsed
}

func fanoutVideoToFollowers(videoID, authorID uint) {
	_ = fanoutVideoToFollowersTx(config.DB, videoID, authorID)
}

func fanoutVideoToFollowersTx(tx *gorm.DB, videoID, authorID uint) error {
	if tx == nil {
		return errors.New("nil transaction")
	}

	var followerCount int64
	if err := tx.Model(&model.Follow{}).
		Where("target_user_id = ?", authorID).
		Count(&followerCount).Error; err != nil {
		return err
	}
	if followerCount == 0 {
		return nil
	}
	if followerCount > getFollowPushThreshold() {
		// 大V走拉模式，不做写扩散。
		return nil
	}

	var followers []followIDRow
	if err := tx.Model(&model.Follow{}).
		Select("user_id as target_user_id").
		Where("target_user_id = ?", authorID).
		Scan(&followers).Error; err != nil {
		return err
	}
	if len(followers) == 0 {
		return nil
	}

	inboxes := make([]model.FollowFeedInbox, 0, len(followers))
	for _, follower := range followers {
		if follower.TargetUserID == 0 {
			continue
		}
		inboxes = append(inboxes, model.FollowFeedInbox{
			UserID:   follower.TargetUserID,
			VideoID:  videoID,
			AuthorID: authorID,
		})
	}
	if len(inboxes) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "video_id"}},
		DoNothing: true,
	}).Create(&inboxes).Error
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
		videoIDs := make([]uint, 0, len(videos))
		videoByID := make(map[uint]model.Video, len(videos))
		for _, video := range videos {
			videoIDs = append(videoIDs, video.ID)
			videoByID[video.ID] = video
		}
		cachedItems, missIDs := getFeedVideosFromCache(viewerKey, videoIDs)
		missVideos := make([]model.Video, 0, len(missIDs))
		for _, videoID := range missIDs {
			video, ok := videoByID[videoID]
			if !ok {
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
		var err error
		likedMap, err = getInteractionStatesBatch(interactionKindLike, *userID, videoIDs)
		if err != nil {
			return nil, err
		}
		favoritedMap, err = getInteractionStatesBatch(interactionKindFavorite, *userID, videoIDs)
		if err != nil {
			return nil, err
		}

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

func resolveHotCursorToken(raw string, cursor FeedCursor) string {
	if cursor.HotToken != "" {
		return cursor.HotToken
	}
	return strings.TrimSpace(raw)
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
	filteredIDs := make([]uint, 0, limit+1)
	hasMore := false

	for attempt := 0; attempt < 3 && len(filteredIDs) < limit+1; attempt++ {
		query := config.DB.Model(&model.Video{}).Select("id as video_id").Order("id desc")
		if lastID > 0 {
			query = query.Where("id < ?", lastID)
		}

		var batch []videoIDRow
		err := query.Limit(fetchLimit).Scan(&batch).Error
		if err != nil {
			return nil, "", false, err
		}
		if len(batch) == 0 {
			break
		}

		candidates := make([]uint, 0, len(batch))
		for _, row := range batch {
			if row.VideoID > 0 {
				candidates = append(candidates, row.VideoID)
			}
		}
		if filterSeen {
			filteredBatch, filterErr := FilterRecentlyExposedVideoIDs(viewerKey, candidates)
			if filterErr != nil {
				return nil, "", false, filterErr
			}
			candidates = filteredBatch
		}
		filteredIDs = append(filteredIDs, candidates...)
		lastID = batch[len(batch)-1].VideoID
		if len(batch) < fetchLimit {
			break
		}
	}

	if len(filteredIDs) > limit {
		hasMore = true
		filteredIDs = filteredIDs[:limit]
	}

	if !hasMore && len(filteredIDs) == limit && lastID > 0 {
		hasMore = true
	}

	nextCursor := ""
	if hasMore && len(filteredIDs) > 0 {
		nextCursor = encodeFeedCursor(FeedCursor{
			Mode:   "latest",
			LastID: filteredIDs[len(filteredIDs)-1],
		})
	}

	return filteredIDs, nextCursor, hasMore, nil
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

func getFollowVideoFeed(userID *uint, viewerKey string, legacyCursor uint, tokenCursor FeedCursor, limit int, filterSeen bool) ([]FeedVideo, string, bool, error) {
	videoIDs, nextCursor, hasMore, err := getFollowVideoIDsPage(userID, viewerKey, legacyCursor, tokenCursor, limit, filterSeen)
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

func getFollowVideoIDsPage(userID *uint, viewerKey string, legacyCursor uint, tokenCursor FeedCursor, limit int, filterSeen bool) ([]uint, string, bool, error) {
	if userID == nil {
		return []uint{}, "", false, nil
	}
	limit = normalizeFeedLimit(limit)

	inboxCursor := legacyCursor
	if tokenCursor.FollowInbox > 0 {
		inboxCursor = tokenCursor.FollowInbox
	}
	outboxCursor := tokenCursor.FollowOutbox

	inboxIDs, nextInbox, inboxHasMore, err := getFollowInboxVideoIDs(*userID, inboxCursor, limit+6)
	if err != nil {
		return nil, "", false, err
	}

	need := limit
	if len(inboxIDs) >= need {
		videoIDs := inboxIDs[:need]
		if filterSeen {
			videoIDs, err = FilterRecentlyExposedVideoIDs(viewerKey, videoIDs)
			if err != nil {
				return nil, "", false, err
			}
		}
		hasMore := inboxHasMore || len(inboxIDs) > need
		nextToken := ""
		if hasMore {
			nextToken = encodeFeedCursor(FeedCursor{
				Mode:         "follow",
				FollowInbox:  nextInbox,
				FollowOutbox: outboxCursor,
			})
		}
		return videoIDs, nextToken, hasMore, nil
	}

	merged := make([]uint, 0, limit+8)
	merged = append(merged, inboxIDs...)
	need -= len(inboxIDs)
	outboxIDs, nextOutbox, outboxHasMore, err := getFollowOutboxVideoIDs(*userID, outboxCursor, need+8)
	if err != nil {
		return nil, "", false, err
	}
	merged = append(merged, outboxIDs...)
	merged = dedupeVideoIDs(merged)

	if filterSeen {
		merged, err = FilterRecentlyExposedVideoIDs(viewerKey, merged)
		if err != nil {
			return nil, "", false, err
		}
	}
	if len(merged) > limit {
		merged = merged[:limit]
	}
	if len(merged) == 0 {
		return []uint{}, "", false, nil
	}

	hasMore := inboxHasMore || outboxHasMore || len(outboxIDs) > need
	nextToken := ""
	if hasMore {
		nextToken = encodeFeedCursor(FeedCursor{
			Mode:         "follow",
			FollowInbox:  nextInbox,
			FollowOutbox: nextOutbox,
		})
	}
	return merged, nextToken, hasMore, nil
}

func dedupeVideoIDs(videoIDs []uint) []uint {
	if len(videoIDs) <= 1 {
		return videoIDs
	}
	seen := make(map[uint]struct{}, len(videoIDs))
	result := make([]uint, 0, len(videoIDs))
	for _, id := range videoIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func getFollowInboxVideoIDs(userID uint, cursor uint, limit int) ([]uint, uint, bool, error) {
	query := config.DB.Model(&model.FollowFeedInbox{}).
		Select("video_id, id").
		Where("user_id = ?", userID).
		Order("id desc")
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	type inboxRow struct {
		ID      uint
		VideoID uint
	}
	var rows []inboxRow
	if err := query.Limit(limit + 1).Scan(&rows).Error; err != nil {
		return nil, 0, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	videoIDs := make([]uint, 0, len(rows))
	var nextCursor uint
	for _, row := range rows {
		videoIDs = append(videoIDs, row.VideoID)
		nextCursor = row.ID
	}
	return videoIDs, nextCursor, hasMore, nil
}

func getFollowOutboxVideoIDs(userID uint, cursor uint, limit int) ([]uint, uint, bool, error) {
	var followRows []followIDRow
	if err := config.DB.Model(&model.Follow{}).
		Select("target_user_id").
		Where("user_id = ?", userID).
		Scan(&followRows).Error; err != nil {
		return nil, 0, false, err
	}
	if len(followRows) == 0 {
		return []uint{}, 0, false, nil
	}
	authorIDs := make([]uint, 0, len(followRows))
	for _, row := range followRows {
		if row.TargetUserID > 0 {
			authorIDs = append(authorIDs, row.TargetUserID)
		}
	}
	if len(authorIDs) == 0 {
		return []uint{}, 0, false, nil
	}

	threshold := getFollowPushThreshold()
	if threshold > 0 {
		var largeAuthorIDs []uint
		if err := config.DB.Raw(`
			SELECT f.target_user_id
			FROM follows f
			JOIN follows ff ON ff.target_user_id = f.target_user_id
			WHERE f.user_id = ?
			GROUP BY f.target_user_id
			HAVING COUNT(ff.id) > ?
		`, userID, threshold).Scan(&largeAuthorIDs).Error; err == nil && len(largeAuthorIDs) > 0 {
			authorIDs = largeAuthorIDs
		} else if err == nil && len(largeAuthorIDs) == 0 {
			return []uint{}, 0, false, nil
		}
	}

	query := config.DB.Model(&model.Video{}).
		Select("id").
		Where("author_id IN ?", authorIDs).
		Order("id desc")
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	var rows []videoIDRow
	if err := query.Limit(limit + 1).Scan(&rows).Error; err != nil {
		return nil, 0, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	videoIDs := make([]uint, 0, len(rows))
	var nextCursor uint
	for _, row := range rows {
		videoIDs = append(videoIDs, row.VideoID)
		nextCursor = row.VideoID
	}
	return videoIDs, nextCursor, hasMore, nil
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

	candidateIDs := videoIDs
	if filterSeen {
		videoIDs, err = FilterRecentlyExposedVideoIDs(viewerKey, videoIDs)
		if err != nil {
			return nil, "", false, err
		}
	}
	selectedIDs, nextOffset, hasMore := selectHotVideoIDsPage(candidateIDs, videoIDs, offset, limit, total)
	if len(selectedIDs) == 0 {
		nextCursor := encodeNextHotCursor(snapshot, nextOffset, hasMore)
		return []uint{}, nextCursor, hasMore, nil
	}

	videos, err := getVideosByIDsOrdered(selectedIDs)
	if err != nil {
		return nil, "", false, err
	}
	if len(videos) == 0 {
		return []uint{}, "", false, nil
	}

	nextCursor := encodeNextHotCursor(snapshot, nextOffset, hasMore)
	return modelVideosToIDs(videos), nextCursor, hasMore, nil
}

func selectHotVideoIDsPage(candidateIDs, visibleIDs []uint, offset, limit int, total int64) ([]uint, int, bool) {
	if limit <= 0 || len(candidateIDs) == 0 {
		return []uint{}, offset, false
	}

	selectedIDs := visibleIDs
	if len(selectedIDs) > limit {
		selectedIDs = selectedIDs[:limit]
	}

	consumedCount := consumedHotCandidateCount(candidateIDs, selectedIDs)
	nextOffset := offset + consumedCount
	hasMore := int64(nextOffset) < total
	return selectedIDs, nextOffset, hasMore
}

func consumedHotCandidateCount(candidateIDs, selectedIDs []uint) int {
	if len(candidateIDs) == 0 {
		return 0
	}
	if len(selectedIDs) == 0 {
		return len(candidateIDs)
	}

	lastSelectedID := selectedIDs[len(selectedIDs)-1]
	for i, videoID := range candidateIDs {
		if videoID == lastSelectedID {
			return i + 1
		}
	}
	return len(candidateIDs)
}

func encodeNextHotCursor(snapshot ranking.HotSnapshotCursor, nextOffset int, hasMore bool) string {
	if !hasMore {
		return ""
	}
	return ranking.EncodeHotSnapshotCursor(ranking.HotSnapshotCursor{
		AggKey:  snapshot.AggKey,
		Offset:  nextOffset,
		Window:  snapshot.Window,
		Created: snapshot.Created,
	})
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
		hotToken := resolveHotCursorToken(query.Cursor, cursor)
		videos, next, more, err := getHotVideoFeed(query.UserID, viewerKey, hotToken, query.LegacyID, limit, query.FilterSeen)
		if err != nil {
			// 热榜依赖 Redis，异常时自动降级回最新流。
			return getLatestVideoFeed(query.UserID, viewerKey, query.LegacyID, limit, query.FilterSeen)
		}
		return videos, next, more, nil
	case "follow":
		return getFollowVideoFeed(query.UserID, viewerKey, query.LegacyID, cursor, limit, query.FilterSeen)
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
