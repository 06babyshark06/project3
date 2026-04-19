package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/06babyshark06/JQKStudy/services/api-gateway/redis"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

func (h *NotificationHandler) StreamNotifications(c *gin.Context) {
	// Dùng jwt.ExtractClaims giống như các handler khác để lấy user_id xịn
	claims := jwt.ExtractClaims(c)
	var userIDStr string
	
	if val, ok := claims["user_id"]; ok {
		switch v := val.(type) {
		case float64:
			userIDStr = fmt.Sprintf("%d", int(v))
		case string:
			userIDStr = v
		default:
			userIDStr = fmt.Sprintf("%v", v)
		}
	} else {
		c.JSON(401, gin.H{"error": "Unauthorized: No user_id in claims"})
		return
	}

	channel := fmt.Sprintf("notifications:%s", userIDStr)
	log.Printf("🔌 SSE: Client connected. Subscribing to Redis channel: [%s] and [notifications:all]", channel)

	// Set headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	
	c.Writer.Flush()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	subscriber := redis.Rdb.Subscribe(ctx, channel)
	defer subscriber.Close()

	ch := subscriber.Channel()

	// 🛠 Test if SSE works immediately upon connection
	welcomeMsg := fmt.Sprintf(`{"type":"INFO", "message":"Đã kết nối luồng SSE! (Đang nghe kênh %s)", "timestamp":"%s"}`, channel, time.Now().Format(time.RFC3339))
	c.SSEvent("message", welcomeMsg)
	c.Writer.Flush()

	// Bỏ c.Stream của Gin để dùng vòng lặp for native, sửa lỗi miss sự kiện
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				log.Printf("Redis Channel closed for user %s", userIDStr)
				return
			}
			log.Printf("📥 API-Gateway Received from Redis: %s", msg.Payload)
			c.SSEvent("message", msg.Payload)
			c.Writer.Flush()
		case <-ctx.Done():
			log.Printf("Client disconnected from SSE stream. User: %s", userIDStr)
			return
		}
	}
}
