package handlers

import (
	"log"
	"net/http"
	"strconv"

	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/ai"
	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	aiClient *grpcclients.AIServiceClient
}

func NewAIHandler(aiClient *grpcclients.AIServiceClient) *AIHandler {
	return &AIHandler{aiClient: aiClient}
}

// GenerateQuestions handles multipart/form-data upload, extracts file, and calls AI service
func (h *AIHandler) GenerateQuestions(c *gin.Context) {
	// Parse multipart form
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	title := c.PostForm("title")
	difficulty := c.PostForm("difficulty")
	qType := c.PostForm("question_type")
	questionCountStr := c.PostForm("question_count")

	focusTopic := c.PostForm("focus_topic")
	language := c.PostForm("language")
	maxOptionsStr := c.PostForm("max_options")

	if difficulty == "" {
		difficulty = "medium"
	}
	if qType == "" {
		qType = "multiple_choice"
	}
	if language == "" {
		language = "Tiếng Việt"
	}

	questionCount, err := strconv.Atoi(questionCountStr)
	if err != nil || questionCount <= 0 {
		questionCount = 5 // default
	}

	maxOptions, err := strconv.Atoi(maxOptionsStr)
	if err != nil || maxOptions < 2 {
		maxOptions = 4 // default
	}

	file, header, err := c.Request.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy file tài liệu đính kèm (field: document)"})
		return
	}
	defer file.Close()

	// Read file into bytes
	fileBytes := make([]byte, header.Size)
	_, err = file.Read(fileBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	req := &pb.GenerateQuestionsFromAIRequest{
		Title:         title,
		Difficulty:    difficulty,
		QuestionType:  qType,
		QuestionCount: int32(questionCount),
		FileBytes:     fileBytes,
		FileName:      header.Filename,
		MimeType:      header.Header.Get("Content-Type"),
		FocusTopic:    focusTopic,
		Language:      language,
		MaxOptions:    int32(maxOptions),
	}

	// Call AI gRPC Service
	resp, err := h.aiClient.Client.GenerateQuestionsFromAI(c.Request.Context(), req)
	if err != nil {
		log.Printf("AI Service Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi gọi AI Service: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
