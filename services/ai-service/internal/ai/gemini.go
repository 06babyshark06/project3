package ai

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	database "github.com/06babyshark06/JQKStudy/services/ai-service/internal/databases"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/ai"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	apiKey string
}

func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{apiKey: apiKey}
}

// GenerateQuestions formats the prompt and calls Gemini SDK
func (c *GeminiClient) GenerateQuestions(ctx context.Context, req *pb.GenerateQuestionsFromAIRequest, text string) ([]*pb.GeneratedAIQuestion, error) {
	// Try to get from cache
	cacheKey := c.generateCacheKey(req, text)
	if database.RedisClient != nil {
		cachedData, err := database.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var questions []*pb.GeneratedAIQuestion
			if err := json.Unmarshal([]byte(cachedData), &questions); err == nil {
				log.Printf("🔹 AI Cache Hit: %s", cacheKey)
				return questions, nil
			}
		}
	}

	if c.apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is missing")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")
	model.ResponseMIMEType = "application/json"

	schema := &genai.Schema{
		Type: genai.TypeArray,
		Items: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"question_text": {Type: genai.TypeString, Description: "Nội dung câu hỏi"},
				"choices": {
					Type: genai.TypeArray,
					Items: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"content":    {Type: genai.TypeString, Description: "Nội dung đáp án"},
							"is_correct": {Type: genai.TypeBoolean, Description: "true nếu đây là đáp án đúng, false nếu sai"},
						},
						Required: []string{"content", "is_correct"},
					},
					Description: fmt.Sprintf("Danh sách CHÍNH XÁC %d lựa chọn/đáp án", req.MaxOptions),
				},
				"explanation": {Type: genai.TypeString, Description: "Giải thích ngắn gọn tại sao đáp án lại như vậy"},
				"difficulty":  {Type: genai.TypeString, Description: "Mức độ khó của câu hỏi này"},
			},
			Required: []string{"question_text", "choices", "explanation", "difficulty"},
		},
	}
	model.ResponseSchema = schema

	basePrompt := fmt.Sprintf(`
Bạn là một chuyên gia giáo dục. Từ tài liệu được cung cấp, hãy tạo ra %d câu hỏi trắc nghiệm với độ khó %s. 
Thể loại câu hỏi: %s.
Yêu cầu về ngôn ngữ: TRẢ LỜI VÀ SINH CÂU HỎI BẰNG %s.
Trọng tâm đặc biệt / Yêu cầu thêm từ giáo viên: %s.

Mỗi câu hỏi phải có CHÍNH XÁC %d lựa chọn (choices).
- Nếu "Thể loại câu hỏi" là "Một đáp án đúng", thì chỉ được có DUY NHẤT 1 lựa chọn có is_correct=true, các cái khác phải là false.
- Nếu "Thể loại câu hỏi" là "Nhiều đáp án đúng", thì phải có ÍT NHẤT 2 lựa chọn có is_correct=true.

Hãy soạn thảo cực kỳ cẩn thận, theo sát ngữ cảnh và đảm bảo tính chính xác về mặt học thuật.
	`, req.QuestionCount, req.Difficulty, req.QuestionType, req.Language, req.FocusTopic, req.MaxOptions)

	var parts []genai.Part

	if req.MimeType == "application/pdf" && len(req.FileBytes) > 0 {
		parts = append(parts, genai.Blob{
			MIMEType: req.MimeType,
			Data:     req.FileBytes,
		})
		parts = append(parts, genai.Text(basePrompt))
	} else {
		fullPrompt := basePrompt + fmt.Sprintf("\n--- TÀI LIỆU BẮT ĐẦU ---\n%s\n--- TÀI LIỆU KẾT THÚC ---\n", text)
		parts = append(parts, genai.Text(fullPrompt))
	}

	resp, err := model.GenerateContent(ctx, parts...)
	if err != nil {
		return nil, fmt.Errorf("API error: %v", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	part := resp.Candidates[0].Content.Parts[0]
	textResponse, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("expected text part but got something else")
	}

	var parsedQuestions []struct {
		QuestionText string `json:"question_text"`
		Choices      []struct {
			Content   string `json:"content"`
			IsCorrect bool   `json:"is_correct"`
		} `json:"choices"`
		Explanation string `json:"explanation"`
		Difficulty  string `json:"difficulty"`
	}

	err = json.Unmarshal([]byte(textResponse), &parsedQuestions)
	if err != nil {
		log.Printf("Failed to unmarshal JSON: \n%s\n", textResponse)
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	var grpcQuestions []*pb.GeneratedAIQuestion
	for _, q := range parsedQuestions {
		var protoChoices []*pb.GeneratedChoice
		for _, c := range q.Choices {
			protoChoices = append(protoChoices, &pb.GeneratedChoice{
				Content:   c.Content,
				IsCorrect: c.IsCorrect,
			})
		}

		grpcQuestions = append(grpcQuestions, &pb.GeneratedAIQuestion{
			QuestionText: q.QuestionText,
			Choices:      protoChoices,
			Explanation:  q.Explanation,
			Difficulty:   q.Difficulty,
		})
	}

	// Save to cache
	if database.RedisClient != nil {
		data, _ := json.Marshal(grpcQuestions)
		database.RedisClient.Set(ctx, cacheKey, data, 24*time.Hour)
	}

	return grpcQuestions, nil
}

func (c *GeminiClient) generateCacheKey(req *pb.GenerateQuestionsFromAIRequest, text string) string {
	// Key includes all parameters that affect the output
	keyData := fmt.Sprintf("%v|%v|%v|%v|%v|%v|%s",
		req.QuestionCount, req.QuestionType, req.Difficulty,
		req.Language, req.MaxOptions, req.FocusTopic, text)
	
	if req.MimeType == "application/pdf" && len(req.FileBytes) > 0 {
		fileHash := md5.Sum(req.FileBytes)
		keyData += fmt.Sprintf("|%x", fileHash)
	}
	
	hash := md5.Sum([]byte(keyData))
	return "ai:questions:" + hex.EncodeToString(hash[:])
}

// ExplainAnswer provides an explanation for why a specific choice is correct/incorrect
func (c *GeminiClient) ExplainAnswer(ctx context.Context, req *pb.ExplainAnswerRequest) (string, error) {
	cacheKey := c.generateExplainCacheKey(req)
	if database.RedisClient != nil {
		cachedData, err := database.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil && cachedData != "" {
			log.Printf("🔹 AI Cache Hit (Explain): %s", cacheKey)
			return cachedData, nil
		}
	}

	if c.apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is missing")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")
	
	// Format choices
	choicesContext := "Các lựa chọn:\n"
	for i, choice := range req.Choices {
		choicesContext += fmt.Sprintf("- Lựa chọn %d: %s\n", i+1, choice)
	}

	prompt := fmt.Sprintf(`
Bạn là một gia sư AI thân thiện, chuyên môn cao. Xin hãy nhận xét ngắn gọn và giải thích cặn kẽ câu hỏi sau để giúp sinh viên nhận ra lỗi sai:

Câu hỏi: %s

%s
Đáp án đúng là: %s

Học sinh đã chọn: %s

Yêu cầu:
1. Giải thích cụ thể tại sao "Đáp án đúng" lại đúng.
2. Chỉ ra lỗi sai logic hoặc lỗ hổng kiến thức dẫn đến việc học sinh chọn "Đáp án đã chọn".
3. Lời lẽ khích lệ, thân thiện nhưng đúng văn phong học thuật.
4. Trả về định dạng Markdown.
`, req.QuestionContent, choicesContext, req.CorrectChoice, req.UserChoice)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("API error: %v", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	part := resp.Candidates[0].Content.Parts[0]
	textResponse, ok := part.(genai.Text)
	if !ok {
		return "", fmt.Errorf("expected text part but got something else")
	}

	explanation := string(textResponse)

	if database.RedisClient != nil {
		database.RedisClient.Set(ctx, cacheKey, explanation, 7*24*time.Hour) // Cache for 7 days
	}

	return explanation, nil
}

func (c *GeminiClient) generateExplainCacheKey(req *pb.ExplainAnswerRequest) string {
	keyData := fmt.Sprintf("%v|%v|%v", req.QuestionContent, req.CorrectChoice, req.UserChoice)
	hash := md5.Sum([]byte(keyData))
	return "ai:explain:" + hex.EncodeToString(hash[:])
}

// ChatWithTutor handles subsequent questions from the student about a specific answer
func (c *GeminiClient) ChatWithTutor(ctx context.Context, req *pb.ChatWithTutorRequest) (string, error) {
	// Generate cache key for the whole conversation history + new message
	cacheKey := c.generateChatCacheKey(req)
	if database.RedisClient != nil {
		cachedData, err := database.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil && cachedData != "" {
			log.Printf("🔹 AI Cache Hit (Chat): %s", cacheKey)
			return cachedData, nil
		}
	}

	if c.apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is missing")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")
	
	// Create the chat session with history
	chat := model.StartChat()
	
	// Prepare system context / initial history if not present
	// We want the AI to remember it's a tutor for a specific question
	choicesContext := "Các lựa chọn:\n"
	for i, choice := range req.Choices {
		choicesContext += fmt.Sprintf("- Lựa chọn %d: %s\n", i+1, choice)
	}

	systemContext := fmt.Sprintf(`
BỐI CẢNH LUỒNG HỘI THOẠI:
Bạn là một gia sư AI đang giúp sinh viên hiểu một câu hỏi trắc nghiệm.
Câu hỏi: %s
%s
Đáp án đúng: %s
Học sinh đã trả lời: %s

Nhiệm vụ: Trả lời các thắc mắc tiếp theo của sinh viên về câu hỏi này. 
Hãy giữ văn phong gia sư, giải thích cặn kẽ, khích lệ. 
Nếu sinh viên hỏi lạc đề, hãy nhắc sinh viên quay lại nội dung kiến thức của câu hỏi.
Trả về định dạng Markdown.
`, req.QuestionContent, choicesContext, req.CorrectChoice, req.UserChoice)

	// In the Go SDK, we can't easily set a "System Instruction" for individual StartChat calls 
	// unless we set it on the model. To keep it per-request, we'll prepend it to the history 
	// or use the first user message. 
	// Actually, the best way for genai Go SDK is model.SystemInstruction = ... before StartChat()
	model.SystemInstruction = genai.NewUserContent(genai.Text(systemContext))

	// Convert pb.ChatMessage history to genai.Content
	var history []*genai.Content
	for _, msg := range req.History {
		history = append(history, &genai.Content{
			Role:  msg.Role,
			Parts: []genai.Part{genai.Text(msg.Content)},
		})
	}
	chat.History = history

	resp, err := chat.SendMessage(ctx, genai.Text(req.NewMessage))
	if err != nil {
		return "", fmt.Errorf("API error during chat: %v", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from model in chat")
	}

	part := resp.Candidates[0].Content.Parts[0]
	textResponse, ok := part.(genai.Text)
	if !ok {
		return "", fmt.Errorf("expected text part but got something else in chat")
	}

	reply := string(textResponse)

	if database.RedisClient != nil {
		database.RedisClient.Set(ctx, cacheKey, reply, 7*24*time.Hour)
	}

	return reply, nil
}

func (c *GeminiClient) generateChatCacheKey(req *pb.ChatWithTutorRequest) string {
	// History content + original context + new message
	historyStr := ""
	for _, h := range req.History {
		historyStr += h.Role + ":" + h.Content + "|"
	}
	keyData := fmt.Sprintf("%v|%v|%v|%v|%v|%v", 
		req.QuestionContent, req.CorrectChoice, req.UserChoice, 
		historyStr, req.NewMessage, "v1") // v1 for busting cache if needed
	
	hash := md5.Sum([]byte(keyData))
	return "ai:chat:" + hex.EncodeToString(hash[:])
}
