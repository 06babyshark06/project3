package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

type ExamHandler struct {
	examClient pb.ExamServiceClient
}

func NewExamHandler(client *grpcclients.ExamServiceClient) *ExamHandler {
	return &ExamHandler{examClient: client.Client}
}

func getUserIDFromContext(c *gin.Context) (int64, error) {
	claims := jwt.ExtractClaims(c)
	userIDStr := fmt.Sprint(claims["user_id"])

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return 0, errors.New("invalid user_id in token")
	}
	return userID, nil
}

func (h *ExamHandler) GetTopics(c *gin.Context) {
	resp, err := h.examClient.GetTopics(c.Request.Context(), &pb.GetTopicsRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) CreateTopic(c *gin.Context) {
	var req pb.CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.examClient.CreateTopic(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) CreateSection(c *gin.Context) {
	var req pb.CreateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.examClient.CreateSection(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) GetSections(c *gin.Context) {
	topicID, _ := strconv.ParseInt(c.Query("topic_id"), 10, 64)
	resp, err := h.examClient.GetSections(c.Request.Context(), &pb.GetSectionsRequest{TopicId: topicID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) ImportQuestions(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	sectionID, _ := strconv.ParseInt(c.PostForm("section_id"), 10, 64)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.examClient.ImportQuestions(c.Request.Context(), &pb.ImportQuestionsRequest{
		SectionId:   sectionID,
		CreatorId:   userID,
		FileContent: fileBytes,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) GetUploadURL(c *gin.Context) {
	var req pb.GetUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Folder == "" {
		req.Folder = "questions"
	}

	resp, err := h.examClient.GetUploadURL(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) GenerateExam(c *gin.Context) {
	var req pb.GenerateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	req.CreatorId = userID

	resp, err := h.examClient.GenerateExam(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) RequestAccess(c *gin.Context) {
	var req pb.RequestExamAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	req.UserId = userID

	resp, err := h.examClient.RequestExamAccess(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) ApproveAccess(c *gin.Context) {
	var req pb.ApproveExamAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.examClient.ApproveExamAccess(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) CheckAccess(c *gin.Context) {
	examID, _ := strconv.ParseInt(c.Query("exam_id"), 10, 64)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.examClient.CheckExamAccess(c.Request.Context(), &pb.CheckExamAccessRequest{
		ExamId: examID,
		UserId: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) CreateQuestion(c *gin.Context) {
	var req pb.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	req.CreatorId = userID

	resp, err := h.examClient.CreateQuestion(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) CreateExam(c *gin.Context) {
	var req pb.CreateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	req.CreatorId = userID

	resp, err := h.examClient.CreateExam(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) GetExamDetails(c *gin.Context) {
	examID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exam id"})
		return
	}

	resp, err := h.examClient.GetExamDetails(c.Request.Context(), &pb.GetExamDetailsRequest{ExamId: examID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) SubmitExam(c *gin.Context) {
	var req pb.SubmitExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	req.UserId = userID

	resp, err := h.examClient.SubmitExam(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) GetSubmission(c *gin.Context) {
	submissionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submission ID"})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.examClient.GetSubmission(c.Request.Context(), &pb.GetSubmissionRequest{
		SubmissionId: submissionID,
		UserId:       userID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) GetExams(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	resp, err := h.examClient.GetExams(c.Request.Context(), &pb.GetExamsRequest{
		Page:  int32(page),
		Limit: int32(limit),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) PublishExam(c *gin.Context) {
    idStr := c.Param("id")
    examID, _ := strconv.ParseInt(idStr, 10, 64)
    
    var req struct { IsPublished bool `json:"is_published"` }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    resp, err := h.examClient.PublishExam(c.Request.Context(), &pb.PublishExamRequest{
        ExamId:      examID,
        IsPublished: req.IsPublished,
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) GetInstructorExams(c *gin.Context) {
    userID, err := getUserIDFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }

    resp, err := h.examClient.GetExams(c.Request.Context(), &pb.GetExamsRequest{
        Page:      1,
        Limit:     100,
        CreatorId: userID, 
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) UpdateQuestion(c *gin.Context) {
	idStr := c.Param("id")
	questionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question ID"})
		return
	}

	_, err = getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	type ChoiceReq struct {
		Content   string `json:"content"`
		IsCorrect bool   `json:"is_correct"`
	}
	var req struct {
		Content      string      `json:"content" binding:"required"`
		QuestionType string      `json:"question_type" binding:"required"`
		Difficulty   string      `json:"difficulty" binding:"required"` 
		Explanation  string      `json:"explanation"`
		Choices      []ChoiceReq `json:"choices" binding:"required,min=2"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pbChoices []*pb.ChoiceInput
	for _, ch := range req.Choices {
		pbChoices = append(pbChoices, &pb.ChoiceInput{
			Content:   ch.Content,
			IsCorrect: ch.IsCorrect,
		})
	}

	resp, err := h.examClient.UpdateQuestion(c.Request.Context(), &pb.UpdateQuestionRequest{
		QuestionId:   questionID,
		Content:      req.Content,
		QuestionType: req.QuestionType,
		Difficulty:   req.Difficulty,
		Explanation:  req.Explanation,
		Choices:      pbChoices,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}

func (h *ExamHandler) DeleteQuestion(c *gin.Context) {
	idStr := c.Param("id")
	questionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question ID"})
		return
	}

	_, err = h.examClient.DeleteQuestion(c.Request.Context(), &pb.DeleteQuestionRequest{QuestionId: questionID})
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contracts.APIResponse{Data: gin.H{"message": "Question deleted"}})
}

func (h *ExamHandler) UpdateExam(c *gin.Context) {
	idStr := c.Param("id")
	examID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid exam ID"})
		return
	}

	// Cập nhật struct hứng dữ liệu JSON để bao gồm Settings
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		TopicId     int64  `json:"topic_id"`
		Settings    struct {
			DurationMinutes       int    `json:"duration_minutes"`
			MaxAttempts           int    `json:"max_attempts"`
			Password              string `json:"password"`
			StartTime             string `json:"start_time"` // Format: RFC3339 (e.g., "2023-10-01T08:00:00Z")
			EndTime               string `json:"end_time"`
			ShuffleQuestions      bool   `json:"shuffle_questions"`
			ShowResultImmediately bool   `json:"show_result_immediately"`
			RequiresApproval      bool   `json:"requires_approval"`
		} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.examClient.UpdateExam(c.Request.Context(), &pb.UpdateExamRequest{
		ExamId:      examID,
		Title:       req.Title,
		Description: req.Description,
		TopicId:     req.TopicId,
		Settings: &pb.ExamSettings{
			DurationMinutes:       int32(req.Settings.DurationMinutes),
			MaxAttempts:           int32(req.Settings.MaxAttempts),
			Password:              req.Settings.Password,
			StartTime:             req.Settings.StartTime,
			EndTime:               req.Settings.EndTime,
			ShuffleQuestions:      req.Settings.ShuffleQuestions,
			ShowResultImmediately: req.Settings.ShowResultImmediately,
			RequiresApproval:      req.Settings.RequiresApproval,
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contracts.APIResponse{Data: gin.H{"message": "Exam updated"}})
}

func (h *ExamHandler) DeleteExam(c *gin.Context) {
    idStr := c.Param("id")
    examID, _ := strconv.ParseInt(idStr, 10, 64)
    
    _, err := h.examClient.DeleteExam(c.Request.Context(), &pb.DeleteExamRequest{ExamId: examID})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, contracts.APIResponse{Data: gin.H{"message": "Exam deleted"}})
}

func (h *ExamHandler) GetMyExamStats(c *gin.Context) {
    userID, err := getUserIDFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }

    resp, err := h.examClient.GetUserExamStats(c.Request.Context(), &pb.GetUserExamStatsRequest{
        UserId: userID,
    })
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, contracts.APIResponse{Data: resp})
}