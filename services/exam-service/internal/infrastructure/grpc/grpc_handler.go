package grpc

import (
	"context"

	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/domain"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam" 
	"google.golang.org/grpc"
)

type gRPCHandler struct {
	pb.UnimplementedExamServiceServer
	service domain.ExamService
}

func NewGRPCHandler(server *grpc.Server, service domain.ExamService) *gRPCHandler {
	handler := &gRPCHandler{
		service: service,
	}

	pb.RegisterExamServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) GetUploadURL(ctx context.Context, req *pb.GetUploadURLRequest) (*pb.GetUploadURLResponse, error) {
	return h.service.GetUploadURL(ctx, req)
}

func (h *gRPCHandler) CreateTopic(ctx context.Context, req *pb.CreateTopicRequest) (*pb.CreateTopicResponse, error) { return h.service.CreateTopic(ctx, req) }
func (h *gRPCHandler) GetTopics(ctx context.Context, req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error) { return h.service.GetTopics(ctx, req) }
func (h *gRPCHandler) CreateSection(ctx context.Context, req *pb.CreateSectionRequest) (*pb.CreateSectionResponse, error) { return h.service.CreateSection(ctx, req) }
func (h *gRPCHandler) GetSections(ctx context.Context, req *pb.GetSectionsRequest) (*pb.GetSectionsResponse, error) { return h.service.GetSections(ctx, req) }

func (h *gRPCHandler) CreateQuestion(ctx context.Context, req *pb.CreateQuestionRequest) (*pb.CreateQuestionResponse, error) { return h.service.CreateQuestion(ctx, req) }
func (h *gRPCHandler) ImportQuestions(ctx context.Context, req *pb.ImportQuestionsRequest) (*pb.ImportQuestionsResponse, error) { return h.service.ImportQuestions(ctx, req) }
func (h *gRPCHandler) UpdateQuestion(ctx context.Context, req *pb.UpdateQuestionRequest) (*pb.UpdateQuestionResponse, error) { return h.service.UpdateQuestion(ctx, req) }
func (h *gRPCHandler) DeleteQuestion(ctx context.Context, req *pb.DeleteQuestionRequest) (*pb.DeleteQuestionResponse, error) { return h.service.DeleteQuestion(ctx, req) }

func (h *gRPCHandler) CreateExam(ctx context.Context, req *pb.CreateExamRequest) (*pb.CreateExamResponse, error) { return h.service.CreateExam(ctx, req) }
func (h *gRPCHandler) GenerateExam(ctx context.Context, req *pb.GenerateExamRequest) (*pb.CreateExamResponse, error) { return h.service.GenerateExam(ctx, req) }
func (h *gRPCHandler) GetExamDetails(ctx context.Context, req *pb.GetExamDetailsRequest) (*pb.GetExamDetailsResponse, error) { return h.service.GetExamDetails(ctx, req) }
func (h *gRPCHandler) GetExams(ctx context.Context, req *pb.GetExamsRequest) (*pb.GetExamsResponse, error) { return h.service.GetExams(ctx, req) }
func (h *gRPCHandler) UpdateExam(ctx context.Context, req *pb.UpdateExamRequest) (*pb.UpdateExamResponse, error) { return h.service.UpdateExam(ctx, req) }
func (h *gRPCHandler) DeleteExam(ctx context.Context, req *pb.DeleteExamRequest) (*pb.DeleteExamResponse, error) { return h.service.DeleteExam(ctx, req) }
func (h *gRPCHandler) PublishExam(ctx context.Context, req *pb.PublishExamRequest) (*pb.PublishExamResponse, error) { return h.service.PublishExam(ctx, req) }

func (h *gRPCHandler) RequestExamAccess(ctx context.Context, req *pb.RequestExamAccessRequest) (*pb.RequestExamAccessResponse, error) { return h.service.RequestExamAccess(ctx, req) }
func (h *gRPCHandler) ApproveExamAccess(ctx context.Context, req *pb.ApproveExamAccessRequest) (*pb.ApproveExamAccessResponse, error) { return h.service.ApproveExamAccess(ctx, req) }
func (h *gRPCHandler) CheckExamAccess(ctx context.Context, req *pb.CheckExamAccessRequest) (*pb.CheckExamAccessResponse, error) { return h.service.CheckExamAccess(ctx, req) }

func (h *gRPCHandler) SubmitExam(ctx context.Context, req *pb.SubmitExamRequest) (*pb.SubmitExamResponse, error) { return h.service.SubmitExam(ctx, req) }
func (h *gRPCHandler) GetSubmission(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error) { return h.service.GetSubmission(ctx, req) }
func (h *gRPCHandler) GetExamCount(ctx context.Context, req *pb.GetExamCountRequest) (*pb.GetExamCountResponse, error) { return h.service.GetExamCount(ctx, req) }
func (h *gRPCHandler) GetUserExamStats(ctx context.Context, req *pb.GetUserExamStatsRequest) (*pb.GetUserExamStatsResponse, error) { return h.service.GetUserExamStats(ctx, req) }