package controller

import (
	"log"
	"strconv"
	"strings"
	"unicode/utf8"

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
		respondError(c, "cursor 参数错误", gin.H{
			"video_list":  []any{},
			"next_cursor": 0,
			"has_more":    false,
		})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit <= 0 {
		respondError(c, "limit 参数错误", gin.H{
			"video_list":  []any{},
			"next_cursor": 0,
			"has_more":    false,
		})
		return
	}

	videos, nextCursor, hasMore, err := service.GetVideoFeed(userID, uint(cursor), limit)
	if err != nil {
		respondError(c, "获取视频流失败", gin.H{
			"video_list":  []any{},
			"next_cursor": 0,
			"has_more":    false,
		})
		return
	}

	respondSuccess(c, "获取成功", gin.H{
		"video_list":  videos,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func PublishVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)

	title := c.PostForm("title")
	title = strings.TrimSpace(title)
	if title == "" {
		respondError(c, "标题不能为空", nil)
		return
	}
	if utf8.RuneCountInString(title) > 50 {
		respondError(c, "标题不能超过 50 个字符", nil)
		return
	}

	// 上传视频
	playUrl, err := util.SaveUploadedFile(c, "video", "video", util.AllowVideoType, util.MaxVideoSize)
	if err != nil {
		respondError(c, err.Error(), nil)
		return
	}

	// 上传封面
	coverUrl, err := util.SaveUploadedFile(c, "cover", "cover", util.AllowImageType, util.MaxImageSize)
	if err != nil {
		if cleanupErr := util.DeleteUploadedFile(playUrl); cleanupErr != nil {
			log.Printf("cleanup uploaded video failed: %v", cleanupErr)
		}
		respondError(c, err.Error(), nil)
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
		respondError(c, "发布失败", nil)
		return
	}

	respondSuccess(c, "发布成功", nil)
}

func DeleteVideo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)
	videoID, ok := parsePositiveUintQuery(c, "video_id")
	if !ok {
		return
	}

	if err := service.DeleteVideo(videoID, uid); err != nil {
		respondError(c, err.Error(), nil)
		return
	}

	respondSuccess(c, "删除成功", nil)
}
