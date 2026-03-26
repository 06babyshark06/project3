package handlers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/06babyshark06/JQKStudy/services/api-gateway/redis"
	coursepb "github.com/06babyshark06/JQKStudy/shared/proto/course"
	exampb "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	userpb "github.com/06babyshark06/JQKStudy/shared/proto/user"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	userClient   userpb.UserServiceClient
	courseClient coursepb.CourseServiceClient
	examClient   exampb.ExamServiceClient
}

func NewHealthHandler(u userpb.UserServiceClient, c coursepb.CourseServiceClient, e exampb.ExamServiceClient) *HealthHandler {
	return &HealthHandler{u, c, e}
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(4)

	statuses := make(map[string]string)
	var mu sync.Mutex

	// 1. Check User Service
	go func() {
		defer wg.Done()
		_, err := h.userClient.GetUserCount(ctx, &userpb.GetUserCountRequest{})
		mu.Lock()
		if err == nil { statuses["user_service"] = "UP" } else { statuses["user_service"] = "DOWN" }
		mu.Unlock()
	}()

	// 2. Check Course Service
	go func() {
		defer wg.Done()
		_, err := h.courseClient.GetCourseCount(ctx, &coursepb.GetCourseCountRequest{})
		mu.Lock()
		if err == nil { statuses["course_service"] = "UP" } else { statuses["course_service"] = "DOWN" }
		mu.Unlock()
	}()

	// 3. Check Exam Service
	go func() {
		defer wg.Done()
		_, err := h.examClient.GetExamCount(ctx, &exampb.GetExamCountRequest{})
		mu.Lock()
		if err == nil { statuses["exam_service"] = "UP" } else { statuses["exam_service"] = "DOWN" }
		mu.Unlock()
	}()

	// 4. Check Redis
	go func() {
		defer wg.Done()
		err := redis.Rdb.Ping(ctx).Err()
		mu.Lock()
		if err == nil { statuses["redis"] = "UP" } else { statuses["redis"] = "DOWN" }
		mu.Unlock()
	}()

	wg.Wait()

	overall := "UP"
	for _, s := range statuses {
		if s == "DOWN" {
			overall = "DEGRADED"
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   overall,
		"services": statuses,
		"time":     time.Now().Format(time.RFC3339),
	})
}
