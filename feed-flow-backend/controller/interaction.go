package controller

import (
	"net/http"
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

	liked, err := service.LikeVideo(videoID, uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "点赞失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "操作成功", "liked": liked})
}

// 收藏
func FavoriteVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	videoID, ok := parsePositiveUintQuery(c, "video_id")
	if !ok {
		return
	}

	favorited, err := service.FavoriteVideo(videoID, uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "收藏失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "操作成功", "favorited": favorited})
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
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "操作成功", "following": following})
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
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "评论成功"})
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
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "删除成功"})
}

// 获取评论
func GetComment(c *gin.Context) {
	videoID, ok := parsePositiveUintQuery(c, "video_id")
	if !ok {
		return
	}

	comments, err := service.GetComments(videoID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "获取评论失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "获取成功",
		"comments":    comments,
	})
}
