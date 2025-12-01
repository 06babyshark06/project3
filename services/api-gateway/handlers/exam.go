package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam" // THAY ĐỔI
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

type ExamHandler struct {
	examClient pb.ExamServiceClient // THAY ĐỔI
}

// NewExamHandler tạo handler với kết nối gRPC
func NewExamHandler(client *grpcclients.ExamServiceClient) *ExamHandler { // THAY ĐỔI
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

// --- Topic Handlers ---

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

	// Lấy creator_id từ JWT
	// userID, err := getUserIDFromContext(c)
	// if err != nil {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	// 	return
	// }
	// req.CreatorId = userID

	resp, err := h.examClient.CreateTopic(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

// --- Question Handlers ---

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

// --- Exam Handlers ---

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
	req.UserId = userID // Gán UserID vào request gRPC

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
	// 1. Lấy Question ID từ URL
	idStr := c.Param("id")
	questionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question ID"})
		return
	}

	// 2. Xác thực người dùng (Chỉ instructor/admin mới được sửa)
	// (Middleware Authorize đã lo phần role, ở đây ta lấy ID để log hoặc logic sau này)
	_, err = getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 3. Bind JSON Body
	// Định nghĩa struct tạm để hứng JSON
	type ChoiceReq struct {
		Content   string `json:"content"`
		IsCorrect bool   `json:"is_correct"`
	}
	var req struct {
		Content      string      `json:"content" binding:"required"`
		QuestionType string      `json:"question_type" binding:"required"` // "single_choice", "multiple_choice"
		Difficulty   string      `json:"difficulty" binding:"required"`    // "easy", "medium", "hard"
		Explanation  string      `json:"explanation"`
		Choices      []ChoiceReq `json:"choices" binding:"required,min=2"` // Ít nhất 2 lựa chọn
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Map sang Proto Request
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

	// Gọi gRPC (Giả sử proto đã có DeleteQuestionRequest)
	// Nếu chưa có trong proto, bạn cần thêm vào proto exam và build lại
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

	var req struct {
		Title           string `json:"title"`
		Description     string `json:"description"`
		DurationMinutes int    `json:"duration_minutes"`
		TopicId         int    `json:"topic_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Gọi gRPC UpdateExam (Cần đảm bảo proto đã có RPC này)
	// Chúng ta đã dùng UpdateExamRequest trong service ở các bước trước
	_, err = h.examClient.UpdateExam(c.Request.Context(), &pb.UpdateExamRequest{
		ExamId:          examID,
		Title:           req.Title,
		Description:     req.Description,
		DurationMinutes: int32(req.DurationMinutes),
		TopicId:         int64(req.TopicId),
		// CreatorId: lấy từ token nếu cần check quyền
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