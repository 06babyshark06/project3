package handlers

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/06babyshark06/JQKStudy/services/api-gateway/redis"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

func (h *NotificationHandler) StreamNotifications(c *gin.Context) {
	// Require Auth
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	
	// Convert userId to string or int as needed, usually in this middleware it's int64
	var userIDStr string
	switch v := userIDVal.(type) {
	case string:
		userIDStr = v
	case int64:
		userIDStr = fmt.Sprintf("%d", v)
	case int:
		userIDStr = fmt.Sprintf("%d", v)
	case float64:
		userIDStr = fmt.Sprintf("%d", int(v))
	default:
		userIDStr = fmt.Sprintf("%v", v)
	}

	channel := fmt.Sprintf("notifications:%s", userIDStr)

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

	c.Stream(func(w io.Writer) bool {
		select {
		case msg := <-ch:
			// Write the SSE event
			c.SSEvent("message", msg.Payload)
			// Returning true means keep the connection open
			return true
		case <-ctx.Done():
			log.Printf("Client disconnected from SSE stream. User: %s", userIDStr)
			// Returning false closes the stream
			return false
		}
	})
}
