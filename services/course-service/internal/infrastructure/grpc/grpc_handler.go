package grpc

import (
	"context"

	// THAY ĐỔI: Trỏ đến domain của course-service
	"github.com/06babyshark06/JQKStudy/services/course-service/internal/domain"
	// THAY ĐỔI: Import proto của COURSE
	pb "github.com/06babyshark06/JQKStudy/shared/proto/course"
	"google.golang.org/grpc"
)

type gRPCHandler struct {
	// THAY ĐỔI: Implement server của COURSE
	pb.UnimplementedCourseServiceServer
	service domain.CourseService
}

func NewGRPCHandler(server *grpc.Server, service domain.CourseService) *gRPCHandler {
	handler := &gRPCHandler{
		service: service,
	}

	// THAY ĐỔI: Register COURSE server
	pb.RegisterCourseServiceServer(server, handler)
	return handler
}

// 4. THAY ĐỔI: Implement tất cả các phương thức của CourseService

// === Course Management (for Instructors) ===

func (h *gRPCHandler) CreateCourse(ctx context.Context, req *pb.CreateCourseRequest) (*pb.CreateCourseResponse, error) {
	resp, err := h.service.CreateCourse(ctx, req)
	if err != nil {
		return nil, err 
	}
	return resp, nil
}

func (h *gRPCHandler) CreateSection(ctx context.Context, req *pb.CreateSectionRequest) (*pb.CreateSectionResponse, error) {
	resp, err := h.service.CreateSection(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) CreateLesson(ctx context.Context, req *pb.CreateLessonRequest) (*pb.CreateLessonResponse, error) {
	resp, err := h.service.CreateLesson(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// === Course (for Students/Public) ===

func (h *gRPCHandler) GetCourses(ctx context.Context, req *pb.GetCoursesRequest) (*pb.GetCoursesResponse, error) {
	resp, err := h.service.GetCourses(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) GetCourseDetails(ctx context.Context, req *pb.GetCourseDetailsRequest) (*pb.GetCourseDetailsResponse, error) {
	resp, err := h.service.GetCourseDetails(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// === Enrollment & Progress (for Students) ===

func (h *gRPCHandler) EnrollCourse(ctx context.Context, req *pb.EnrollCourseRequest) (*pb.EnrollCourseResponse, error) {
	resp, err := h.service.EnrollCourse(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) GetMyCourses(ctx context.Context, req *pb.GetMyCoursesRequest) (*pb.GetMyCoursesResponse, error) {
	resp, err := h.service.GetMyCourses(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) MarkLessonCompleted(ctx context.Context, req *pb.MarkLessonCompletedRequest) (*pb.MarkLessonCompletedResponse, error) {
	resp, err := h.service.MarkLessonCompleted(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// === File Upload ===

func (h *gRPCHandler) GetUploadURL(ctx context.Context, req *pb.GetUploadURLRequest) (*pb.GetUploadURLResponse, error) {
	resp, err := h.service.GetUploadURL(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) UpdateCourse(ctx context.Context, req *pb.UpdateCourseRequest) (*pb.UpdateCourseResponse, error) {
    return h.service.UpdateCourse(ctx, req)
}

func (h *gRPCHandler) UpdateSection(ctx context.Context, req *pb.UpdateSectionRequest) (*pb.UpdateSectionResponse, error) {
    return h.service.UpdateSection(ctx, req)
}

func (h *gRPCHandler) DeleteSection(ctx context.Context, req *pb.DeleteSectionRequest) (*pb.DeleteSectionResponse, error) {
    return h.service.DeleteSection(ctx, req)
}

func (h *gRPCHandler) DeleteLesson(ctx context.Context, req *pb.DeleteLessonRequest) (*pb.DeleteLessonResponse, error) {
    return h.service.DeleteLesson(ctx, req)
}

func (h *gRPCHandler) UpdateLesson(ctx context.Context, req *pb.UpdateLessonRequest) (*pb.UpdateLessonResponse, error) {
	return h.service.UpdateLesson(ctx, req)
}

func (h *gRPCHandler) PublishCourse(ctx context.Context, req *pb.PublishCourseRequest) (*pb.PublishCourseResponse, error) {
    return h.service.PublishCourse(ctx, req)
}

func (h *gRPCHandler) GetCourseCount(ctx context.Context, req *pb.GetCourseCountRequest) (*pb.GetCourseCountResponse, error) {
    return h.service.GetCourseCount(ctx, req)
}

func (h *gRPCHandler) DeleteCourse(ctx context.Context, req *pb.DeleteCourseRequest) (*pb.DeleteCourseResponse, error) {
	return h.service.DeleteCourse(ctx, req)
}