package server

import (
	"context"
	"log"

	"github.com/06babyshark06/JQKStudy/services/ai-service/internal/ai"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/ai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AIServiceServer struct {
	pb.UnimplementedAIServiceServer
	geminiClient *ai.GeminiClient
}

func NewAIServiceServer(apiKey string) *AIServiceServer {
	return &AIServiceServer{
		geminiClient: ai.NewGeminiClient(apiKey),
	}
}

func (s *AIServiceServer) GenerateQuestionsFromAI(ctx context.Context, req *pb.GenerateQuestionsFromAIRequest) (*pb.GenerateQuestionsFromAIResponse, error) {
	log.Printf("Received request to generate %d %s questions", req.QuestionCount, req.Difficulty)

	var textContext string

	// For now, assume content_text is provided by API Gateway (so we can choose where to do file extraction)
	if req.ContentText != "" {
		textContext = req.ContentText
	} else if len(req.FileBytes) > 0 {
		if req.MimeType == "application/pdf" {
			log.Printf("Received PDF file, will pass natively to Gemini.")
			// textContext is empty, let geminiClient handle genai.Blob
		} else {
			// Attempt to extract text based on extension
			extracted, err := ai.ExtractTextFromFile(req.FileBytes, req.FileName)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Failed to extract text from file: %v", err)
			}
			textContext = extracted
		}
	} else {
		return nil, status.Error(codes.InvalidArgument, "Either content_text or file_bytes is required")
	}

	questions, err := s.geminiClient.GenerateQuestions(ctx, req, textContext)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "AI Generation failed: %v", err)
	}

	return &pb.GenerateQuestionsFromAIResponse{
		Success:   true,
		Message:   "Questions generated successfully",
		Questions: questions,
	}, nil
}
