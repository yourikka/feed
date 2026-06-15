package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/yourikka/feed-flow/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN is required")
	}
	dbLogLevel := gormLogger.Warn
	if getEnvAsBool("DB_LOG_INFO", false) {
		dbLogLevel = gormLogger.Info
	}
	slowThresholdMs := getDBEnvAsInt("DB_SLOW_THRESHOLD_MS", 200)
	gormLog := gormLogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormLogger.Config{
			SlowThreshold:             time.Duration(slowThresholdMs) * time.Millisecond,
			LogLevel:                  dbLogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
	gormCfg := &gorm.Config{
		SkipDefaultTransaction: getEnvAsBool("DB_SKIP_DEFAULT_TX", true),
		PrepareStmt:            getEnvAsBool("DB_PREPARE_STMT", true),
		Logger:                 gormLog,
	}
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), gormCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to extract sql DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(getDBEnvAsInt("DB_MAX_OPEN_CONNS", 200))
	sqlDB.SetMaxIdleConns(getDBEnvAsInt("DB_MAX_IDLE_CONNS", 80))
	sqlDB.SetConnMaxLifetime(time.Duration(getDBEnvAsInt("DB_CONN_MAX_LIFETIME_SECONDS", 180)) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(getDBEnvAsInt("DB_CONN_MAX_IDLE_SECONDS", 60)) * time.Second)

	log.Println("Database connection established")
	if getEnvAsBool("DB_AUTO_MIGRATE", false) {
		err = DB.AutoMigrate(
			&model.User{},
			&model.Video{},
			&model.Comment{},
			&model.Like{},
			&model.Favorite{},
			&model.Follow{},
			&model.FollowFeedInbox{},
			&model.VideoBehaviorEvent{},
			&model.MQOutboxMessage{},
		)
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

func getDBEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
