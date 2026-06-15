package config

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	redisMu sync.RWMutex
	RDB     *redis.Client
)
var Ctx = context.Background()

const redisReconnectInterval = 5 * time.Second

func InitRedis() {
	if err := connectRedis(); err != nil {
		log.Printf("Redis unavailable, enter degraded mode: %v", err)
	}
	go keepRedisAlive()
}

func connectRedis() error {
	client := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "redis:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})
	_, err := client.Ping(Ctx).Result()
	if err != nil {
		_ = client.Close()
		return err
	}
	setRedisClient(client)
	log.Println("Connected to Redis successfully")
	return nil
}

func GetRedisClient() *redis.Client {
	redisMu.RLock()
	defer redisMu.RUnlock()
	return RDB
}

func setRedisClient(client *redis.Client) {
	redisMu.Lock()
	RDB = client
	redisMu.Unlock()
}

func keepRedisAlive() {
	ticker := time.NewTicker(redisReconnectInterval)
	defer ticker.Stop()

	for range ticker.C {
		client := GetRedisClient()
		if client != nil {
			if _, err := client.Ping(Ctx).Result(); err == nil {
				continue
			}
			log.Println("Redis ping failed, try reconnect")
		}

		if err := connectRedis(); err != nil {
			log.Printf("Redis reconnect failed: %v", err)
			continue
		}
		log.Println("Redis self-heal succeeded")
	}
}
