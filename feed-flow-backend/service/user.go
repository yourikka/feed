package service

import (
	"errors"

	"github.com/yourikka/feed-flow/config"
	"github.com/yourikka/feed-flow/model"
	"github.com/yourikka/feed-flow/util"
)

type UserStats struct {
	WorkCount         int64 `json:"work_count"`
	LikeReceivedCount int64 `json:"like_received_count"`
	FollowCount       int64 `json:"follow_count"`
	FollowerCount     int64 `json:"follower_count"`
}

type RelationUser struct {
	ID          uint   `json:"ID"`
	Username    string `json:"Username"`
	Avatar      string `json:"Avatar"`
	IsFollowing bool   `json:"IsFollowing"`
}

// Register 注册用户
func Register(username, password string) (user *model.User, err error) {
	//检查用户名是否存在
	var existingUser model.User
	if err = config.DB.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return nil, errors.New("用户名已存在")
	}
	//加密密码
	hashedPwd, _ := util.HashPassword(password)

	//创建用户
	user = &model.User{
		Username: username,
		Password: hashedPwd,
		Avatar:   "https://via.placeholder.com/150", // 默认头像
	}
	if err = config.DB.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// Login 登录用户
func Login(username, password string) (user *model.User, err error) {
	//检查用户名是否存在

	if err = config.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	//校验密码
	if !util.CheckPassword(password, user.Password) {
		return nil, errors.New("密码错误")
	}
	return user, nil
}

func GetUserByID(userID uint) (*model.User, error) {
	var user model.User
	if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	user.Password = ""
	return &user, nil
}

func GetUserStats(userID uint) (UserStats, error) {
	stats := UserStats{}

	if err := config.DB.Model(&model.Video{}).Where("author_id = ?", userID).Count(&stats.WorkCount).Error; err != nil {
		return stats, err
	}
	if err := config.DB.Model(&model.Follow{}).Where("user_id = ?", userID).Count(&stats.FollowCount).Error; err != nil {
		return stats, err
	}
	if err := config.DB.Model(&model.Follow{}).Where("target_user_id = ?", userID).Count(&stats.FollowerCount).Error; err != nil {
		return stats, err
	}

	if err := config.DB.Model(&model.Like{}).
		Joins("JOIN videos ON videos.id = likes.video_id").
		Where("videos.author_id = ?", userID).
		Count(&stats.LikeReceivedCount).Error; err != nil {
		return stats, err
	}

	return stats, nil
}

func GetFollowingList(targetUserID uint, currentUserID *uint) ([]RelationUser, error) {
	return getRelationUsers("user_id", targetUserID, currentUserID)
}

func GetFollowerList(targetUserID uint, currentUserID *uint) ([]RelationUser, error) {
	return getRelationUsers("target_user_id", targetUserID, currentUserID)
}

func getRelationUsers(field string, targetUserID uint, currentUserID *uint) ([]RelationUser, error) {
	var follows []model.Follow
	if err := config.DB.Where(field+" = ?", targetUserID).Order("created_at desc").Find(&follows).Error; err != nil {
		return nil, err
	}
	if len(follows) == 0 {
		return []RelationUser{}, nil
	}

	userIDs := make([]uint, 0, len(follows))
	for _, follow := range follows {
		if field == "user_id" {
			userIDs = append(userIDs, follow.TargetUserID)
			continue
		}
		userIDs = append(userIDs, follow.UserID)
	}

	var users []model.User
	if err := config.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}

	userMap := make(map[uint]model.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	currentFollowMap := map[uint]bool{}
	if currentUserID != nil {
		var currentFollows []model.Follow
		if err := config.DB.Where("user_id = ?", *currentUserID).Find(&currentFollows).Error; err != nil {
			return nil, err
		}
		for _, follow := range currentFollows {
			currentFollowMap[follow.TargetUserID] = true
		}
	}

	list := make([]RelationUser, 0, len(follows))
	for _, follow := range follows {
		relatedUserID := follow.UserID
		if field == "user_id" {
			relatedUserID = follow.TargetUserID
		}
		user, ok := userMap[relatedUserID]
		if !ok {
			continue
		}

		item := RelationUser{
			ID:          user.ID,
			Username:    user.Username,
			Avatar:      user.Avatar,
			IsFollowing: currentFollowMap[user.ID],
		}
		if currentUserID != nil && user.ID == *currentUserID {
			item.IsFollowing = false
		}
		list = append(list, item)
	}

	return list, nil
}

func ChangePassword(userID uint, oldPassword, newPassword string) error {
	if oldPassword == "" || newPassword == "" {
		return errors.New("原密码和新密码不能为空")
	}
	if oldPassword == newPassword {
		return errors.New("新密码不能和原密码一致")
	}

	var user model.User
	if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if !util.CheckPassword(oldPassword, user.Password) {
		return errors.New("原密码错误")
	}

	hashedPwd, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return config.DB.Model(&model.User{}).Where("id = ?", userID).Update("password", hashedPwd).Error
}
