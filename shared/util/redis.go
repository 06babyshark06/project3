package util

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/06babyshark06/JQKStudy/shared/env"
	"github.com/redis/go-redis/v9"
)

// InitRedis initializes a Redis client with configuration from environment variables
// and verifies the connection with a Ping.
func InitRedis() (*redis.Client, error) {
	addr := env.GetString("REDIS_ADDR", "redis:6379")
	password := env.GetString("REDIS_PASSWORD", "")
	db := env.GetInt("REDIS_DB", 0)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("không thể kết nối tới Redis tại %s: %v", addr, err)
	}

	log.Printf("✅ Đã kết nối thành công tới Redis tại %s", addr)
	return rdb, nil
}
