package controller

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourikka/feed-flow/service"
	"github.com/yourikka/feed-flow/util"
)

func Feed(c *gin.Context) {
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

	cursor, err := strconv.ParseUint(c.DefaultQuery("cursor", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "cursor 参数错误", "video_list": []any{}, "next_cursor": 0, "has_more": false})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit <= 0 {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "limit 参数错误", "video_list": []any{}, "next_cursor": 0, "has_more": false})
		return
	}

	videos, nextCursor, hasMore, err := service.GetVideoFeed(userID, uint(cursor), limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status_code": 1,
			"status_msg":  "获取视频流失败",
			"video_list":  []any{},
			"next_cursor": 0,
			"has_more":    false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "获取成功",
		"video_list":  videos,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func PublishVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)

	title := c.PostForm("title")
	if title == "" {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "标题不能为空"})
		return
	}

	// 上传视频
	playUrl, err := util.SaveUploadedFile(c, "video", "video", util.AllowVideoType, util.MaxVideoSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}

	// 上传封面
	coverUrl, err := util.SaveUploadedFile(c, "cover", "cover", util.AllowImageType, util.MaxImageSize)
	if err != nil {
		if cleanupErr := util.DeleteUploadedFile(playUrl); cleanupErr != nil {
			log.Printf("cleanup uploaded video failed: %v", cleanupErr)
		}
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}

	// 保存到数据库
	err = service.PublishVideo(title, playUrl, coverUrl, uid)
	if err != nil {
		if cleanupErr := util.DeleteUploadedFile(playUrl); cleanupErr != nil {
			log.Printf("cleanup uploaded video failed: %v", cleanupErr)
		}
		if cleanupErr := util.DeleteUploadedFile(coverUrl); cleanupErr != nil {
			log.Printf("cleanup uploaded cover failed: %v", cleanupErr)
		}
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "发布失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "发布成功"})
}

func DeleteVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)
	videoID, ok := parsePositiveUintQuery(c, "video_id")
	if !ok {
		return
	}

	if err := service.DeleteVideo(videoID, uid); err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "删除成功"})
}
