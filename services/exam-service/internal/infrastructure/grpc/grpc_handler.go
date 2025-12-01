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

func (h *gRPCHandler) CreateTopic(ctx context.Context, req *pb.CreateTopicRequest) (*pb.CreateTopicResponse, error) {
	resp, err := h.service.CreateTopic(ctx, req)
	if err != nil {
		return nil, err // Tương lai: Chuyển đổi sang gRPC error codes
	}
	return resp, nil
}

func (h *gRPCHandler) GetTopics(ctx context.Context, req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error) {
	resp, err := h.service.GetTopics(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) CreateQuestion(ctx context.Context, req *pb.CreateQuestionRequest) (*pb.CreateQuestionResponse, error) {
	resp, err := h.service.CreateQuestion(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) CreateExam(ctx context.Context, req *pb.CreateExamRequest) (*pb.CreateExamResponse, error) {
	resp, err := h.service.CreateExam(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) GetExamDetails(ctx context.Context, req *pb.GetExamDetailsRequest) (*pb.GetExamDetailsResponse, error) {
	resp, err := h.service.GetExamDetails(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) SubmitExam(ctx context.Context, req *pb.SubmitExamRequest) (*pb.SubmitExamResponse, error) {
	resp, err := h.service.SubmitExam(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) GetSubmission(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error) {
	resp, err := h.service.GetSubmission(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) GetExamCount(ctx context.Context, req *pb.GetExamCountRequest) (*pb.GetExamCountResponse, error) {
    return h.service.GetExamCount(ctx, req)
}

func (h *gRPCHandler) GetExams(ctx context.Context, req *pb.GetExamsRequest) (*pb.GetExamsResponse, error) {
	return h.service.GetExams(ctx, req)
}

func (h *gRPCHandler) PublishExam(ctx context.Context, req *pb.PublishExamRequest) (*pb.PublishExamResponse, error) {
	return h.service.PublishExam(ctx, req)
}

func (h *gRPCHandler) UpdateQuestion(ctx context.Context, req *pb.UpdateQuestionRequest) (*pb.UpdateQuestionResponse, error) {
	return h.service.UpdateQuestion(ctx, req)
}

func (h *gRPCHandler) DeleteQuestion(ctx context.Context, req *pb.DeleteQuestionRequest) (*pb.DeleteQuestionResponse, error) {
    return h.service.DeleteQuestion(ctx, req)
}

func (h *gRPCHandler) UpdateExam(ctx context.Context, req *pb.UpdateExamRequest) (*pb.UpdateExamResponse, error) {
    return h.service.UpdateExam(ctx, req)
}

func (h *gRPCHandler) DeleteExam(ctx context.Context, req *pb.DeleteExamRequest) (*pb.DeleteExamResponse, error) {
	return h.service.DeleteExam(ctx, req)
}

func (h *gRPCHandler) GetUserExamStats(ctx context.Context, req *pb.GetUserExamStatsRequest) (*pb.GetUserExamStatsResponse, error) {
    return h.service.GetUserExamStats(ctx, req)
}