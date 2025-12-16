package grpc

import (
	"context"

	"github.com/06babyshark06/JQKStudy/services/user-service/internal/domain"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/user"
	"google.golang.org/grpc"
)
type gRPCHandler struct{
	pb.UnimplementedUserServiceServer
	service domain.UserService
}

func NewGRPCHandler(server *grpc.Server, service domain.UserService) *gRPCHandler {
	handler := &gRPCHandler{
		service: service,
	}

	pb.RegisterUserServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	resp, err := h.service.Register(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	resp, err := h.service.Login(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	resp, err := h.service.GetProfile(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) GetAllUsers(ctx context.Context, req *pb.GetAllUsersRequest) (*pb.GetAllUsersResponse, error) {
	resp, err := h.service.GetAllUsers(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *gRPCHandler) GetUserCount(ctx context.Context, req *pb.GetUserCountRequest) (*pb.GetUserCountResponse, error) {
    return h.service.GetUserCount(ctx, req)
}

func (h *gRPCHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	return h.service.DeleteUser(ctx, req)
}

func (h *gRPCHandler) UpdateUserRole(ctx context.Context, req *pb.UpdateUserRoleRequest) (*pb.UpdateUserRoleResponse, error) {
	return h.service.UpdateUserRole(ctx, req)
}

func (h *gRPCHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	return h.service.UpdateUser(ctx, req)
}

func (h *gRPCHandler) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordResponse, error) {
	return h.service.UpdatePassword(ctx, req)
}

func (h *gRPCHandler) CreateClass(ctx context.Context, req *pb.CreateClassRequest) (*pb.CreateClassResponse, error) {
	return h.service.CreateClass(ctx, req)
}

func (h *gRPCHandler) UpdateClass(ctx context.Context, req *pb.UpdateClassRequest) (*pb.UpdateClassResponse, error) {
	return h.service.UpdateClass(ctx, req)
}

func (h *gRPCHandler) DeleteClass(ctx context.Context, req *pb.DeleteClassRequest) (*pb.DeleteClassResponse, error) {
	return h.service.DeleteClass(ctx, req)
}

func (h *gRPCHandler) GetClasses(ctx context.Context, req *pb.GetClassesRequest) (*pb.GetClassesResponse, error) {
	return h.service.GetClasses(ctx, req)
}
func (h *gRPCHandler) GetClassDetails(ctx context.Context, req *pb.GetClassDetailsRequest) (*pb.GetClassDetailsResponse, error) {
	return h.service.GetClassDetails(ctx, req)
}
func (h *gRPCHandler) AddMembers(ctx context.Context, req *pb.AddMembersRequest) (*pb.AddMembersResponse, error) {
	return h.service.AddMembers(ctx, req)
}
func (h *gRPCHandler) RemoveMember(ctx context.Context, req *pb.RemoveMemberRequest) (*pb.RemoveMemberResponse, error) {
	return h.service.RemoveMember(ctx, req)
}
func (h *gRPCHandler) CheckUserInClass(ctx context.Context, req *pb.CheckUserInClassRequest) (*pb.CheckUserInClassResponse, error) {
	return h.service.CheckUserInClass(ctx, req)
}

func (h *gRPCHandler) JoinClassByCode(ctx context.Context, req *pb.JoinClassByCodeRequest) (*pb.JoinClassByCodeResponse, error) {
	return h.service.JoinClassByCode(ctx, req)
}