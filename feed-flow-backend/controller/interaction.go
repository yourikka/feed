package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourikka/feed-flow/service"
)

// 点赞
func LikeVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	vid, _ := strconv.Atoi(c.Query("video_id"))

	liked, err := service.LikeVideo(uint(vid), uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "liked": liked})
}

// 收藏
func FavoriteVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	vid, _ := strconv.Atoi(c.Query("video_id"))

	favorited, err := service.FavoriteVideo(uint(vid), uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "favorited": favorited})
}

// 关注
func FollowUser(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	targetUserID, _ := strconv.Atoi(c.Query("to_user_id"))

	following, err := service.FollowUser(uid, uint(targetUserID))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "following": following})
}

// 发表评论
func CommentVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	vid, _ := strconv.Atoi(c.Query("video_id"))
	content := c.Query("content")

	err := service.CommentVideo(uint(vid), uid, content)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0})
}

// 删除评论
func DeleteComment(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid := userID.(uint)
	commentID, _ := strconv.Atoi(c.Query("comment_id"))

	err := service.DeleteComment(uint(commentID), uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "删除成功"})
}

// 获取评论
func GetComment(c *gin.Context) {
	vid, _ := strconv.Atoi(c.Query("video_id"))
	comments, err := service.GetComments(uint(vid))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"comments":    comments,
	})
}
