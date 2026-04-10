package config

import (
	"log"

	"github.com/yourikka/feed-flow/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := "root:root123@tcp(127.0.0.1:3306)/douyin?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established")
	err = DB.AutoMigrate(&model.User{}, &model.Video{}, &model.Comment{}, &model.Like{}, &model.Favorite{}, &model.Follow{})
}
