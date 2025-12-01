package handlers

import (
	"net/http"
	"sync"

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
	var wg sync.WaitGroup
	wg.Add(3)

	var userCount, courseCount, examCount int64
	var errUser, errCourse, errExam error

	// Gọi song song (Concurrent) 3 service để tiết kiệm thời gian
	go func() {
		defer wg.Done()
		resp, err := h.userClient.GetUserCount(c.Request.Context(), &userpb.GetUserCountRequest{})
		if err == nil { userCount = resp.Count } else { errUser = err }
	}()

	go func() {
		defer wg.Done()
		resp, err := h.courseClient.GetCourseCount(c.Request.Context(), &coursepb.GetCourseCountRequest{})
		if err == nil { courseCount = resp.Count } else { errCourse = err }
	}()

	go func() {
		defer wg.Done()
		resp, err := h.examClient.GetExamCount(c.Request.Context(), &exampb.GetExamCountRequest{})
		if err == nil { examCount = resp.Count } else { errExam = err }
	}()

	wg.Wait()

	// Nếu có lỗi, log lại nhưng vẫn trả về các số liệu lấy được (partial response)
	if errUser != nil || errCourse != nil || errExam != nil {
		// log.Println("Stats partial error:", errUser, errCourse, errExam)
	}

	c.JSON(http.StatusOK, contracts.APIResponse{
		Data: gin.H{
			"total_users":   userCount,
			"total_courses": courseCount,
			"total_exams":   examCount,
		},
	})
}