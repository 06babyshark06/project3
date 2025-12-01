package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var ctx = context.Background()

func InitRedis(addr string) {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", 
		DB:       0,
	})
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
