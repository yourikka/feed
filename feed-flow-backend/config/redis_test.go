package config

import (
	"testing"

	"github.com/go-redis/redis/v8"
)

func TestSetAndGetRedisClient(t *testing.T) {
	original := GetRedisClient()
	t.Cleanup(func() {
		setRedisClient(original)
	})

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() {
		_ = client.Close()
	})

	setRedisClient(client)
	if got := GetRedisClient(); got != client {
		t.Fatalf("GetRedisClient() returned %p, want %p", got, client)
	}

	setRedisClient(nil)
	if got := GetRedisClient(); got != nil {
		t.Fatalf("GetRedisClient() returned %p, want nil", got)
	}
}
