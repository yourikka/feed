package service

import "github.com/yourikka/feed-flow/model"

type FeedIDPage struct {
	VideoIDs   []uint
	NextToken  string
	HasMore    bool
	NextCursor uint
}

func GetFeedVideoIDs(query FeedQuery) (FeedIDPage, error) {
	viewerKey := BuildViewerKey(query.UserID, query.ClientID)
	limit := normalizeFeedLimit(query.Limit)
	cursor := FeedCursor{}
	if decoded, ok := decodeFeedCursor(query.Cursor); ok {
		cursor = decoded
	}

	sortType := normalizeSortType(query.SortType)
	switch sortType {
	case "hot":
		videoIDs, nextToken, hasMore, err := getHotVideoIDsPage(viewerKey, cursor.HotToken, query.LegacyID, limit, query.FilterSeen)
		if err != nil {
			return FeedIDPage{}, err
		}
		return FeedIDPage{
			VideoIDs:   videoIDs,
			NextToken:  nextToken,
			HasMore:    hasMore,
			NextCursor: 0,
		}, nil
	case "follow":
		videoIDs, nextToken, hasMore, err := getFollowVideoIDsPage(query.UserID, viewerKey, query.LegacyID, cursor, limit, query.FilterSeen)
		if err != nil {
			return FeedIDPage{}, err
		}
		return FeedIDPage{
			VideoIDs:   videoIDs,
			NextToken:  nextToken,
			HasMore:    hasMore,
			NextCursor: 0,
		}, nil
	default:
		videoIDs, nextToken, hasMore, err := getLatestVideoIDsPage(viewerKey, query.LegacyID, cursor.LastID, limit, query.FilterSeen)
		if err != nil {
			return FeedIDPage{}, err
		}
		return FeedIDPage{
			VideoIDs:   videoIDs,
			NextToken:  nextToken,
			HasMore:    hasMore,
			NextCursor: 0,
		}, nil
	}
}

func GetFeedVideosByIDs(videoIDs []uint, userID *uint, clientID string) ([]FeedVideo, error) {
	viewerKey := BuildViewerKey(userID, clientID)
	if viewerKey != "" && len(videoIDs) > 0 {
		cachedItems, missIDs := getFeedVideosFromCache(viewerKey, videoIDs)
		if len(missIDs) == 0 {
			return arrangeFeedVideosByIDs(videoIDs, cachedItems), nil
		}
		videos, err := getVideosByIDsOrderedWithCache(missIDs)
		if err != nil {
			return nil, err
		}
		freshItems, err := buildFeedVideos(videos, userID)
		if err != nil {
			return nil, err
		}
		for _, item := range freshItems {
			cachedItems[item.ID] = item
		}
		return arrangeFeedVideosByIDs(videoIDs, cachedItems), nil
	}
	videos, err := getVideosByIDsOrderedWithCache(videoIDs)
	if err != nil {
		return nil, err
	}
	return buildFeedVideos(videos, userID)
}

func normalizeSortType(sortType string) string {
	switch sortType {
	case "hot":
		return "hot"
	case "follow":
		return "follow"
	default:
		return "latest"
	}
}

func modelVideosToIDs(videos []model.Video) []uint {
	result := make([]uint, 0, len(videos))
	for _, video := range videos {
		result = append(result, video.ID)
	}
	return result
}

func arrangeFeedVideosByIDs(videoIDs []uint, items map[uint]FeedVideo) []FeedVideo {
	result := make([]FeedVideo, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		item, ok := items[videoID]
		if !ok {
			continue
		}
		result = append(result, item)
	}
	return result
}
