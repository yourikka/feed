package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
)

const (
	videoDetailCacheKeyPrefix = "video:detail:"
	videoDetailCacheTTL       = 15 * time.Minute
	videoDetailCacheTTLJitter = 3 * time.Minute
)

type cachedVideoDetail struct {
	ID       uint       `json:"id"`
	Title    string     `json:"title"`
	PlayUrl  string     `json:"play_url"`
	CoverUrl string     `json:"cover_url"`
	AuthorID uint       `json:"author_id"`
	Author   FeedAuthor `json:"author"`
}

func getVideoDetailCacheKey(videoID uint) string {
	return fmt.Sprintf("%s%d", videoDetailCacheKeyPrefix, videoID)
}

func getVideoDetailCacheTTL() time.Duration {
	if videoDetailCacheTTLJitter <= 0 {
		return videoDetailCacheTTL
	}
	return videoDetailCacheTTL + time.Duration(rand.Int63n(int64(videoDetailCacheTTLJitter)+1))
}

func toCachedVideoDetail(video model.Video) cachedVideoDetail {
	return cachedVideoDetail{
		ID:       video.ID,
		Title:    video.Title,
		PlayUrl:  video.PlayUrl,
		CoverUrl: video.CoverUrl,
		AuthorID: video.AuthorID,
		Author: FeedAuthor{
			ID:       video.Author.ID,
			Username: video.Author.Username,
			Avatar:   video.Author.Avatar,
		},
	}
}

func toModelVideo(detail cachedVideoDetail) model.Video {
	video := model.Video{
		Title:    detail.Title,
		PlayUrl:  detail.PlayUrl,
		CoverUrl: detail.CoverUrl,
		AuthorID: detail.AuthorID,
		Author: model.User{
			Username: detail.Author.Username,
			Avatar:   detail.Author.Avatar,
		},
	}
	video.ID = detail.ID
	video.Author.ID = detail.Author.ID
	return video
}

func getVideoDetailFromCache(videoID uint) (model.Video, bool) {
	client := config.GetRedisClient()
	if client == nil || videoID == 0 {
		return model.Video{}, false
	}
	raw, err := client.Get(config.Ctx, getVideoDetailCacheKey(videoID)).Result()
	if err != nil || raw == "" {
		return model.Video{}, false
	}
	var detail cachedVideoDetail
	if err := json.Unmarshal([]byte(raw), &detail); err != nil {
		return model.Video{}, false
	}
	video := toModelVideo(detail)
	video.ID = detail.ID
	video.Author.ID = detail.Author.ID
	return video, true
}

func setVideoDetailCache(video model.Video) {
	client := config.GetRedisClient()
	if client == nil || video.ID == 0 {
		return
	}
	payload, err := json.Marshal(toCachedVideoDetail(video))
	if err != nil {
		return
	}
	_ = client.Set(config.Ctx, getVideoDetailCacheKey(video.ID), payload, getVideoDetailCacheTTL()).Err()
}

func invalidateVideoDetailCache(videoID uint) {
	client := config.GetRedisClient()
	if client == nil || videoID == 0 {
		return
	}
	_ = client.Del(config.Ctx, getVideoDetailCacheKey(videoID)).Err()
}

func getVideosByIDsOrderedWithCache(videoIDs []uint) ([]model.Video, error) {
	if len(videoIDs) == 0 {
		return []model.Video{}, nil
	}

	videoMap := make(map[uint]model.Video, len(videoIDs))
	missIDs := make([]uint, 0, len(videoIDs))
	client := config.GetRedisClient()
	if client != nil {
		keys := make([]string, 0, len(videoIDs))
		keyToID := make(map[string]uint, len(videoIDs))
		for _, videoID := range videoIDs {
			if videoID == 0 {
				continue
			}
			key := getVideoDetailCacheKey(videoID)
			keys = append(keys, key)
			keyToID[key] = videoID
		}
		values, err := client.MGet(config.Ctx, keys...).Result()
		if err == nil && len(values) == len(keys) {
			for i, value := range values {
				videoID := keyToID[keys[i]]
				if value == nil {
					missIDs = append(missIDs, videoID)
					continue
				}
				raw, ok := value.(string)
				if !ok || raw == "" {
					missIDs = append(missIDs, videoID)
					continue
				}
				var detail cachedVideoDetail
				if err := json.Unmarshal([]byte(raw), &detail); err != nil {
					missIDs = append(missIDs, videoID)
					continue
				}
				videoMap[videoID] = toModelVideo(detail)
			}
		} else {
			for _, videoID := range videoIDs {
				if videoID > 0 {
					missIDs = append(missIDs, videoID)
				}
			}
		}
	} else {
		for _, videoID := range videoIDs {
			if videoID > 0 {
				missIDs = append(missIDs, videoID)
			}
		}
	}

	if len(missIDs) > 0 {
		var videos []model.Video
		if err := config.DB.Preload("Author").Where("id IN ?", missIDs).Find(&videos).Error; err != nil {
			return nil, err
		}
		for _, video := range videos {
			videoMap[video.ID] = video
			setVideoDetailCache(video)
		}
	}

	ordered := make([]model.Video, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		if video, ok := videoMap[videoID]; ok {
			ordered = append(ordered, video)
		}
	}
	return ordered, nil
}
