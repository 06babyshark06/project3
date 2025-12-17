package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	pbExam "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	pbUser "github.com/06babyshark06/JQKStudy/shared/proto/user"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

type ClassHandler struct {
	userClient pbUser.UserServiceClient
	examClient pbExam.ExamServiceClient
}

func NewClassHandler(userClient *grpcclients.UserServiceClient, examClient *grpcclients.ExamServiceClient) *ClassHandler {
	return &ClassHandler{userClient: userClient.Client, examClient: examClient.Client}
}

// --- QUẢN LÝ LỚP HỌC ---

func (h *ClassHandler) CreateClass(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.userClient.CreateClass(c.Request.Context(), &pbUser.CreateClassRequest{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		TeacherId:   userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp.Class})
}

func (h *ClassHandler) UpdateClass(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, _ := getUserIDFromContext(c)

	_, err := h.userClient.UpdateClass(c.Request.Context(), &pbUser.UpdateClassRequest{
		Id:          id,
		Name:        req.Name,
		Description: req.Description,
		TeacherId:   userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: gin.H{"success": true}})
}

func (h *ClassHandler) DeleteClass(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID, _ := getUserIDFromContext(c)

	_, err := h.userClient.DeleteClass(c.Request.Context(), &pbUser.DeleteClassRequest{
		Id:        id,
		TeacherId: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: gin.H{"success": true}})
}

func (h *ClassHandler) GetClasses(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	userID, _ := getUserIDFromContext(c)
	role := getUserRoleFromContext(c)

	req := &pbUser.GetClassesRequest{
		Page:  int32(page),
		Limit: int32(limit),
	}

	if role == "student" {
		req.StudentId = userID
	} else {
		req.TeacherId = userID
	}

	resp, err := h.userClient.GetClasses(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ClassHandler) GetClassDetails(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	resp, err := h.userClient.GetClassDetails(c.Request.Context(), &pbUser.GetClassDetailsRequest{ClassId: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ClassHandler) AddMembers(c *gin.Context) {
	var req struct {
		ClassId int64    `json:"class_id"`
		Emails  []string `json:"emails"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, _ := getUserIDFromContext(c)

	resp, err := h.userClient.AddMembers(c.Request.Context(), &pbUser.AddMembersRequest{
		ClassId:   req.ClassId,
		Emails:    req.Emails,
		TeacherId: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ClassHandler) RemoveMember(c *gin.Context) {
	classID, _ := strconv.ParseInt(c.Query("class_id"), 10, 64)
	studentID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	teacherID, _ := getUserIDFromContext(c)

	_, err := h.userClient.RemoveMember(c.Request.Context(), &pbUser.RemoveMemberRequest{
		ClassId:   classID,
		UserId:    studentID,
		TeacherId: teacherID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: gin.H{"success": true}})
}

func (h *ClassHandler) JoinClassByCode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập mã lớp"})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	resp, err := h.userClient.JoinClassByCode(c.Request.Context(), &pbUser.JoinClassByCodeRequest{
		UserId: userID,
		Code:   req.Code,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": resp.Message})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": resp.Message,
		"classId": resp.ClassId,
	})
}

func (h *ClassHandler) AssignExamToClass(c *gin.Context) {
	classID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Class ID"})
		return
	}

	var req struct {
		ExamID int64 `json:"exam_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Gọi sang Exam Service
	_, err = h.examClient.AssignExamToClass(c.Request.Context(), &pbExam.AssignExamToClassRequest{
		ClassId: classID,
		ExamId:  req.ExamID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã gán bài thi vào lớp thành công"})
}

func (h *ClassHandler) RemoveExamFromClass(c *gin.Context) {
	classID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Class ID"})
		return
	}
	examID, err := strconv.ParseInt(c.Param("exam_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Exam ID"})
		return
	}

	_, err = h.examClient.UnassignExamFromClass(c.Request.Context(), &pbExam.AssignExamToClassRequest{
		ClassId: classID,
		ExamId:  examID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã xóa bài thi khỏi lớp"})
}

// GET /classes/:id/exams
func (h *ClassHandler) GetClassExams(c *gin.Context) {
	classID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Class ID"})
		return
	}
	userID, _ := getUserIDFromContext(c)

	// Gọi sang Exam Service
	resp, err := h.examClient.GetExamsByClass(c.Request.Context(), &pbExam.GetExamsByClassRequest{
		ClassId:   classID,
		Status:    "published",
		StudentId: userID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp.Exams})
}

func getUserRoleFromContext(c *gin.Context) string {
	claims := jwt.ExtractClaims(c)
	return fmt.Sprintf("%v", claims["role"])
}
