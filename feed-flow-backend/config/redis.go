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

const (
	redisReconnectInterval = 5 * time.Second
	redisOpTimeout         = 2 * time.Second
)

func InitRedis() {
	if err := connectRedis(); err != nil {
		log.Printf("Redis unavailable, enter degraded mode: %v", err)
	}
	go keepRedisAlive()
}

func connectRedis() error {
	client := redis.NewClient(&redis.Options{
		Addr:         getEnv("REDIS_ADDR", "redis:6379"),
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           0,
		DialTimeout:  redisOpTimeout,
		ReadTimeout:  redisOpTimeout,
		WriteTimeout: redisOpTimeout,
		PoolTimeout:  redisOpTimeout,
	})
	ctx, cancel := WithRedisTimeout(context.Background())
	defer cancel()

	_, err := client.Ping(ctx).Result()
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
	oldClient := RDB
	RDB = client
	redisMu.Unlock()

	if oldClient != nil && oldClient != client {
		_ = oldClient.Close()
	}
}

func keepRedisAlive() {
	ticker := time.NewTicker(redisReconnectInterval)
	defer ticker.Stop()

	for range ticker.C {
		client := GetRedisClient()
		if client != nil {
			ctx, cancel := WithRedisTimeout(context.Background())
			_, err := client.Ping(ctx).Result()
			cancel()
			if err == nil {
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

func WithRedisTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, redisOpTimeout)
}
