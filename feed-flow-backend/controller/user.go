package controller

import (
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
		respondError(c, "参数错误", nil)
		return
	}

	user, err := service.Register(req.Username, req.Password)
	if err != nil {
		respondError(c, err.Error(), nil)
		return
	}

	token, err := util.GenerateToken(user.ID)
	if err != nil {
		respondError(c, "生成token失败", nil)
		return
	}
	respondSuccess(c, "注册成功", gin.H{
		"user_id": user.ID,
		"token":   token,
	})
}

func Login(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, "参数错误", nil)
		return
	}

	user, err := service.Login(req.Username, req.Password)
	if err != nil {
		respondError(c, err.Error(), nil)
		return
	}

	token, err := util.GenerateToken(user.ID)
	if err != nil {
		respondError(c, "生成token失败", nil)
		return
	}
	respondSuccess(c, "登录成功", gin.H{
		"user_id": user.ID,
		"token":   token,
	})
}

func GetUserInfo(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)

	user, err := service.GetUserByID(uid)
	if err != nil {
		respondError(c, "获取用户信息失败", nil)
		return
	}

	stats, err := service.GetUserStats(uid)
	if err != nil {
		respondError(c, "获取用户信息失败", nil)
		return
	}

	respondSuccess(c, "获取成功", gin.H{
		"user":  user,
		"stats": stats,
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

	cursor, ok := parseOptionalUintQuery(c, "cursor", 0)
	if !ok {
		return
	}
	limit, ok := parseOptionalPositiveIntQuery(c, "limit", 10)
	if !ok {
		return
	}

	videoList, nextCursor, hasMore, err := service.GetUserVideoList(targetUID, &currentUID, cursor, limit)
	if err != nil {
		respondError(c, "获取作品失败", gin.H{
			"video_list":  []any{},
			"next_cursor": 0,
			"has_more":    false,
		})
		return
	}

	respondSuccess(c, "获取成功", gin.H{
		"video_list":  videoList,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func UpdateAvatar(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)

	avatarUrl, err := util.SaveUploadedFile(c, "avatar", "avatar", util.AllowImageType, util.MaxImageSize)
	if err != nil {
		respondError(c, err.Error(), nil)
		return
	}

	if err := config.DB.Model(&model.User{}).Where("id = ?", uid).Update("avatar", avatarUrl).Error; err != nil {
		respondError(c, "更新头像失败", nil)
		return
	}

	respondSuccess(c, "头像更新成功", gin.H{
		"avatar_url": avatarUrl,
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
		respondError(c, "获取关注列表失败", gin.H{"user_list": []any{}})
		return
	}

	respondSuccess(c, "获取成功", gin.H{
		"user_list": list,
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
		respondError(c, "获取粉丝列表失败", gin.H{"user_list": []any{}})
		return
	}

	respondSuccess(c, "获取成功", gin.H{
		"user_list": list,
	})
}

func UpdatePassword(c *gin.Context) {
	userID, _ := c.Get("userId")
	uid, _ := userID.(uint)

	var req PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, "参数错误", nil)
		return
	}

	if err := service.ChangePassword(uid, req.OldPassword, req.NewPassword); err != nil {
		respondError(c, err.Error(), nil)
		return
	}

	respondSuccess(c, "修改密码成功", nil)
}
