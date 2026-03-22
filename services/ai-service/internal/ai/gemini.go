package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/ai"
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

	prompt := fmt.Sprintf(`
Bạn là một chuyên gia giáo dục. Từ tài liệu được cung cấp dưới đây, hãy tạo ra %d câu hỏi trắc nghiệm với độ khó %s. 
Thể loại câu hỏi: %s.
Yêu cầu về ngôn ngữ: TRẢ LỜI VÀ SINH CÂU HỎI BẰNG %s.
Trọng tâm đặc biệt / Yêu cầu thêm từ giáo viên: %s.

Mỗi câu hỏi phải có CHÍNH XÁC %d lựa chọn (choices).
- Nếu "Thể loại câu hỏi" là "Một đáp án đúng", thì chỉ được có DUY NHẤT 1 lựa chọn có is_correct=true, các cái khác phải là false.
- Nếu "Thể loại câu hỏi" là "Nhiều đáp án đúng", thì phải có ÍT NHẤT 2 lựa chọn có is_correct=true.

Hãy soạn thảo cực kỳ cẩn thận, theo sát ngữ cảnh và đảm bảo tính chính xác về mặt học thuật.

--- TÀI LIỆU BẮT ĐẦU ---
%s
--- TÀI LIỆU KẾT THÚC ---
	`, req.QuestionCount, req.Difficulty, req.QuestionType, req.Language, req.FocusTopic, req.MaxOptions, text)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
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

	return grpcQuestions, nil
}
