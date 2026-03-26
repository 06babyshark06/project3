package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/06babyshark06/JQKStudy/services/api-gateway/redis"
	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	coursepb "github.com/06babyshark06/JQKStudy/shared/proto/course"
	exampb "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	userpb "github.com/06babyshark06/JQKStudy/shared/proto/user"
	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	userClient   userpb.UserServiceClient
	courseClient coursepb.CourseServiceClient
	examClient   exampb.ExamServiceClient
}

func NewStatsHandler(u *grpcclients.UserServiceClient, c *grpcclients.CourseServiceClient, e *grpcclients.ExamServiceClient) *StatsHandler {
	return &StatsHandler{
		userClient:   u.Client,
		courseClient: c.Client,
		examClient:   e.Client,
	}
}

func (h *StatsHandler) GetAdminStats(c *gin.Context) {
	// 1. Kiểm tra Cache trước
	cacheKey := "admin_stats"
	if cachedData, err := redis.GetCache(cacheKey); err == nil {
		var data gin.H
		if err := json.Unmarshal([]byte(cachedData), &data); err == nil {
			c.JSON(http.StatusOK, contracts.APIResponse{Data: data})
			return
		}
	}

	var wg sync.WaitGroup
	wg.Add(3)

	var userCount, courseCount, examCount int64
	var errUser, errCourse, errExam error

	go func() {
		defer wg.Done()
		resp, err := h.userClient.GetUserCount(c.Request.Context(), &userpb.GetUserCountRequest{})
		if err == nil {
			userCount = resp.Count
		} else {
			errUser = err
		}
	}()

	go func() {
		defer wg.Done()
		resp, err := h.courseClient.GetCourseCount(c.Request.Context(), &coursepb.GetCourseCountRequest{})
		if err == nil {
			courseCount = resp.Count
		} else {
			errCourse = err
		}
	}()

	go func() {
		defer wg.Done()
		resp, err := h.examClient.GetExamCount(c.Request.Context(), &exampb.GetExamCountRequest{})
		if err == nil {
			examCount = resp.Count
		} else {
			errExam = err
		}
	}()

	wg.Wait()

	if errUser != nil || errCourse != nil || errExam != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	statsData := gin.H{
		"total_users":   userCount,
		"total_courses": courseCount,
		"total_exams":   examCount,
	}

	// 2. Lưu vào Cache (TTL 1 phút cho stats)
	if jsonData, err := json.Marshal(statsData); err == nil {
		redis.SetCache(cacheKey, string(jsonData), 1*time.Minute)
	}

	c.JSON(http.StatusOK, contracts.APIResponse{
		Data: statsData,
	})
}
