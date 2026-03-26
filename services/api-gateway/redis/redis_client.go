package redis

import (
	"context"
	"log"
	"time"

	"github.com/06babyshark06/JQKStudy/shared/util"
	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var ctx = context.Background()

func InitRedis() {
	var err error
	Rdb, err = util.InitRedis()
	if err != nil {
		log.Fatalf("LỖI: %v", err)
	}
}

func SetRefreshToken(userID string, token string, ttl time.Duration) error {
	return Rdb.Set(ctx, "refresh:"+userID, token, ttl).Err()
}

func GetRefreshToken(userID string) (string, error) {
	return Rdb.Get(ctx, "refresh:"+userID).Result()
}

func DeleteRefreshToken(userID string) error {
	return Rdb.Del(ctx, "refresh:"+userID).Err()
}

// Caching Helpers
func SetCache(key string, value interface{}, ttl time.Duration) error {
	return Rdb.Set(ctx, "cache:"+key, value, ttl).Err()
}

func GetCache(key string) (string, error) {
	return Rdb.Get(ctx, "cache:"+key).Result()
}

func DeleteCache(key string) error {
	return Rdb.Del(ctx, "cache:"+key).Err()
}

// Rate Limiting
func CheckRateLimit(key string, limit int, window time.Duration) (int, bool, error) {
	fullKey := "ratelimit:" + key
	count, err := Rdb.Incr(ctx, fullKey).Result()
	if err != nil {
		return 0, false, err
	}
	if count == 1 {
		Rdb.Expire(ctx, fullKey, window)
	}
	return int(count), int(count) <= limit, nil
}
