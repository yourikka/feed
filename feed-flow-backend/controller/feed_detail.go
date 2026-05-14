package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourikka/feed-flow/service"
	"github.com/yourikka/feed-flow/util"
)

func FeedIDs(c *gin.Context) {
	var userID *uint
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			if claims, err := util.ParseToken(parts[1]); err == nil {
				userID = &claims.UserID
			}
		}
	}

	cursor, ok := parseOptionalUintQuery(c, "cursor", 0)
	if !ok {
		return
	}
	limit, ok := parseOptionalPositiveIntQuery(c, "limit", 10)
	if !ok {
		return
	}
	sortType := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort", "latest")))
	clientID := strings.TrimSpace(c.GetHeader("X-Client-Id"))
	if clientID == "" {
		clientID = strings.TrimSpace(c.Query("client_id"))
	}

	page, err := service.GetFeedVideoIDs(service.FeedQuery{
		UserID:     userID,
		ClientID:   clientID,
		Cursor:     strings.TrimSpace(c.Query("cursor_token")),
		LegacyID:   cursor,
		Limit:      limit,
		SortType:   sortType,
		FilterSeen: true,
	})
	if err != nil {
		respondError(c, "获取视频ID失败", gin.H{
			"video_ids":   []uint{},
			"next_cursor": 0,
			"next_token":  "",
			"has_more":    false,
		})
		return
	}

	respondSuccess(c, "获取成功", gin.H{
		"video_ids":   page.VideoIDs,
		"next_cursor": page.NextCursor,
		"next_token":  page.NextToken,
		"has_more":    page.HasMore,
	})
}

func FeedDetails(c *gin.Context) {
	var userID *uint
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			if claims, err := util.ParseToken(parts[1]); err == nil {
				userID = &claims.UserID
			}
		}
	}

	rawIDs := strings.TrimSpace(c.Query("video_ids"))
	if rawIDs == "" {
		respondError(c, "video_ids 参数错误", gin.H{"video_list": []any{}})
		return
	}

	videoIDs, err := parseUintCSV(rawIDs)
	if err != nil || len(videoIDs) == 0 {
		respondError(c, "video_ids 参数错误", gin.H{"video_list": []any{}})
		return
	}

	videoList, err := service.GetFeedVideosByIDs(videoIDs, userID)
	if err != nil {
		respondError(c, "获取视频详情失败", gin.H{"video_list": []any{}})
		return
	}
	respondSuccess(c, "获取成功", gin.H{"video_list": videoList})
}
