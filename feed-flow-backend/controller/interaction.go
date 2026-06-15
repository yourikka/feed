package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourikka/feed-flow/service"
)

// 点赞
func LikeVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	videoID, ok := parsePositiveUintQuery(c, "video_id")
	if !ok {
		return
	}

	liked, err := service.LikeVideo(c.Request.Context(), videoID, uid)
	if err != nil {
		respondError(c, "点赞失败", nil)
		return
	}
	respondSuccess(c, "操作成功", gin.H{"liked": liked})
}

// 收藏
func FavoriteVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	videoID, ok := parsePositiveUintQuery(c, "video_id")
	if !ok {
		return
	}

	favorited, err := service.FavoriteVideo(c.Request.Context(), videoID, uid)
	if err != nil {
		respondError(c, "收藏失败", nil)
		return
	}
	respondSuccess(c, "操作成功", gin.H{"favorited": favorited})
}

// 关注
func FollowUser(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	targetUserID, ok := parsePositiveUintQuery(c, "to_user_id")
	if !ok {
		return
	}

	following, err := service.FollowUser(uid, targetUserID)
	if err != nil {
		respondError(c, err.Error(), nil)
		return
	}
	respondSuccess(c, "操作成功", gin.H{"following": following})
}

// 发表评论
func CommentVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	videoID, ok := parsePositiveUintQuery(c, "video_id")
	if !ok {
		return
	}
	content := strings.TrimSpace(c.Query("content"))

	err := service.CommentVideo(videoID, uid, content)
	if err != nil {
		respondError(c, err.Error(), nil)
		return
	}
	respondSuccess(c, "评论成功", nil)
}

// 删除评论
func DeleteComment(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	commentID, ok := parsePositiveUintQuery(c, "comment_id")
	if !ok {
		return
	}

	err := service.DeleteComment(commentID, uid)
	if err != nil {
		respondError(c, err.Error(), nil)
		return
	}
	respondSuccess(c, "删除成功", nil)
}

// 获取评论
func GetComment(c *gin.Context) {
	videoID, ok := parsePositiveUintQuery(c, "video_id")
	if !ok {
		return
	}

	cursor, err := strconv.ParseUint(c.DefaultQuery("cursor", "0"), 10, 64)
	if err != nil {
		respondError(c, "cursor 参数错误", gin.H{
			"comments":    []any{},
			"next_cursor": 0,
			"has_more":    false,
		})
		return
	}
	limit, ok := parseOptionalPositiveIntQuery(c, "limit", 20)
	if !ok {
		return
	}

	comments, nextCursor, hasMore, err := service.GetComments(videoID, uint(cursor), limit)
	if err != nil {
		respondError(c, "获取评论失败", gin.H{
			"comments":    []any{},
			"next_cursor": 0,
			"has_more":    false,
		})
		return
	}
	respondSuccess(c, "获取成功", gin.H{
		"comments":    comments,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}
