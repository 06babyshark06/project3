package database

import (
	"context"
	"log"

	"github.com/06babyshark06/JQKStudy/shared/env"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func ConnectRedis() {
	redisAddr := env.GetString("REDIS_ADDR", "localhost:6379")
	redisPassword := env.GetString("REDIS_PASSWORD", "")
	redisDB := env.GetInt("REDIS_DB", 0)

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx := context.Background()
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("⚠️ Cảnh báo: Không thể kết nối tới Redis tại %s: %v", redisAddr, err)
	} else {
		log.Printf("✅ Đã kết nối thành công tới Redis tại %s", redisAddr)
	}
}
