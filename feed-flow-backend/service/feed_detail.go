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

func GetFeedVideosByIDs(videoIDs []uint, userID *uint) ([]FeedVideo, error) {
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
