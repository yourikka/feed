package config

import (
	"log"
	"os"
	"strconv"

	"github.com/yourikka/feed-flow/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN is required")
	}
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established")
	if getEnvAsBool("DB_AUTO_MIGRATE", false) {
		err = DB.AutoMigrate(&model.User{}, &model.Video{}, &model.Comment{}, &model.Like{}, &model.Favorite{}, &model.Follow{})
		if err != nil {
			log.Fatalf("Failed to auto migrate database: %v", err)
		}
		log.Println("Auto migrate finished")
	} else {
		log.Println("Auto migrate skipped (set DB_AUTO_MIGRATE=true to enable)")
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
