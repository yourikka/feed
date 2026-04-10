package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/service"
	"github.com/yourikka/feed-flow/util"
)

type UserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type PasswordChangeRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func Register(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	user, err := service.Register(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}

	token, err := util.GenerateToken(user.ID)
	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "注册成功",
		"user_id":     user.ID,
		"token":       token,
	})
}

func Login(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	user, err := service.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}

	token, err := util.GenerateToken(user.ID)
	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "登录成功",
		"user_id":     user.ID,
		"token":       token,
	})
}

func GetUserInfo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)

	user, err := service.GetUserByID(uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "获取用户信息失败"})
		return
	}

	stats, err := service.GetUserStats(uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "获取用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "获取成功",
		"user":        user,
		"stats":       stats,
	})
}

func GetUserVideoList(c *gin.Context) {
	userID, _ := c.Get("userId")
	currentUID, _ := userID.(uint)

	targetUID := currentUID
	if rawUserID := c.Query("user_id"); rawUserID != "" {
		if parsedID, err := strconv.Atoi(rawUserID); err == nil && parsedID > 0 {
			targetUID = uint(parsedID)
		}
	}

	videoList, err := service.GetUserVideoList(targetUID, &currentUID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "获取作品失败", "video_list": []any{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "获取成功",
		"video_list":  videoList,
	})
}

func UpdateAvatar(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)

	avatarUrl, err := util.SaveUploadedFile(c, "avatar", "avatar", util.AllowImageType, util.MaxImageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}

	if err := config.DB.Model(&model.User{}).Where("id = ?", uid).Update("avatar", avatarUrl).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "更新头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "头像更新成功",
		"avatar_url":  avatarUrl,
	})
}

func GetFollowList(c *gin.Context) {
	userID, _ := c.Get("userId")
	currentUID, _ := userID.(uint)

	targetUID := currentUID
	if rawUserID := c.Query("user_id"); rawUserID != "" {
		if parsedID, err := strconv.Atoi(rawUserID); err == nil && parsedID > 0 {
			targetUID = uint(parsedID)
		}
	}

	list, err := service.GetFollowingList(targetUID, &currentUID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "获取关注列表失败", "user_list": []any{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "获取成功",
		"user_list":   list,
	})
}

func GetFollowerList(c *gin.Context) {
	userID, _ := c.Get("userId")
	currentUID, _ := userID.(uint)

	targetUID := currentUID
	if rawUserID := c.Query("user_id"); rawUserID != "" {
		if parsedID, err := strconv.Atoi(rawUserID); err == nil && parsedID > 0 {
			targetUID = uint(parsedID)
		}
	}

	list, err := service.GetFollowerList(targetUID, &currentUID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "获取粉丝列表失败", "user_list": []any{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "获取成功",
		"user_list":   list,
	})
}

func UpdatePassword(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)

	var req PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status_code": 1, "status_msg": "参数错误"})
		return
	}

	if err := service.ChangePassword(uid, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "修改密码成功",
	})
}
