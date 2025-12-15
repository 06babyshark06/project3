package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/user"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

type ClassHandler struct {
	userClient pb.UserServiceClient
}

func NewClassHandler(client *grpcclients.UserServiceClient) *ClassHandler {
	return &ClassHandler{userClient: client.Client}
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

	resp, err := h.userClient.CreateClass(c.Request.Context(), &pb.CreateClassRequest{
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

	_, err := h.userClient.UpdateClass(c.Request.Context(), &pb.UpdateClassRequest{
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

	_, err := h.userClient.DeleteClass(c.Request.Context(), &pb.DeleteClassRequest{
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

	req := &pb.GetClassesRequest{
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
	resp, err := h.userClient.GetClassDetails(c.Request.Context(), &pb.GetClassDetailsRequest{ClassId: id})
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

	resp, err := h.userClient.AddMembers(c.Request.Context(), &pb.AddMembersRequest{
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

	_, err := h.userClient.RemoveMember(c.Request.Context(), &pb.RemoveMemberRequest{
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

func getUserRoleFromContext(c *gin.Context) string {
	claims := jwt.ExtractClaims(c)
	return fmt.Sprintf("%v", claims["role"]) 
}