package database

import (
	"log"

	"github.com/06babyshark06/JQKStudy/shared/util"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	var err error
	RedisClient, err = util.InitRedis()
	if err != nil {
		// Log error but don't fail if Redis is just for caching
		log.Printf("⚠️ CẢNH BÁO: Không thể kết nối tới Redis trong course-service: %v. Caching sẽ bị vô hiệu hóa.", err)
	}
}
