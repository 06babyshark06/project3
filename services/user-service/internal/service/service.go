package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/06babyshark06/JQKStudy/services/user-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/user"
	"golang.org/x/crypto/bcrypt"
)

type userService struct {
	repo domain.UserRepository
	producer domain.EventProducer
}

func NewUserService(repo domain.UserRepository, producer domain.EventProducer) domain.UserService {
	return &userService{repo: repo, producer: producer}
}

func (s *userService) GetProfile(ctx context.Context, userId *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	user, err := s.repo.GetUserById(ctx, userId.UserId)
	if err != nil {
		return nil, err
	}

	return &pb.GetProfileResponse{
		Id:       user.Id,
		FullName: user.FullName,
		Email:    user.Email,
		Role:     user.Role.Name,
	}, nil
}

func (s *userService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, err
	}

	return &pb.LoginResponse{
		Id:       user.Id,
		FullName: user.FullName,
		Email:    user.Email,
		Role:     user.Role.Name,
	}, nil
}

func (s *userService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	existing, _ := s.repo.GetUserByEmail(ctx, req.Email)
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	user := &domain.UserModel{
		FullName:  req.FullName,
		Email:     req.Email,
		Password:  string(hash),
		RoleId:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	eventPayload := contracts.UserRegisteredEvent{
		UserID:   createdUser.Id,
		Email:    createdUser.Email,
		FullName: createdUser.FullName,
	}
	eventBytes, err := json.Marshal(eventPayload)

	if err != nil {
		log.Printf("LỖI: Không thể marshal sự kiện user_registered: %v", err)
	} else {
		key := []byte(strconv.FormatInt(createdUser.Id, 10))
		err = s.producer.Produce("user_events", key, eventBytes)
		if err != nil {
			log.Printf("LỖI: Không thể gửi sự kiện user_registered: %v", err)
		}
	}

	return &pb.RegisterResponse{
		Id:       createdUser.Id,
		FullName: createdUser.FullName,
		Email:    createdUser.Email,
	}, nil
}

func (s *userService) GetAllUsers(ctx context.Context, req *pb.GetAllUsersRequest) (*pb.GetAllUsersResponse, error) {
	limit := int(req.PageSize)
    if limit <= 0 { limit = 10 }
    page := int(req.Page)
    if page <= 0 { page = 1 }
    offset := (page - 1) * limit 
    users, total, err := s.repo.GetUsersWithPagination(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	var pbUsers []*pb.GetProfileResponse
	for _, u := range users {
		pbUsers = append(pbUsers, &pb.GetProfileResponse{
			Id:       u.Id,
			Email:    u.Email,
			FullName: u.FullName,
			Role:     u.Role.Name,
		})
	}

	totalPages := int32((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &pb.GetAllUsersResponse{
		Users:      pbUsers,
		Page:       req.Page,
		PageSize:   req.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *userService) GetUserCount(ctx context.Context, req *pb.GetUserCountRequest) (*pb.GetUserCountResponse, error) {
    count, err := s.repo.CountUsers(ctx)
    if err != nil { return nil, err }
    return &pb.GetUserCountResponse{Count: count}, nil
}

func (s *userService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	
	if err := s.repo.DeleteUser(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.DeleteUserResponse{Success: true}, nil
}

func (s *userService) UpdateUserRole(ctx context.Context, req *pb.UpdateUserRoleRequest) (*pb.UpdateUserRoleResponse, error) {
	user, err := s.repo.GetUserById(ctx, req.Id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	role, err := s.repo.GetRoleByName(ctx, req.Role)
	if err != nil {
		return nil, errors.New("role invalid")
	}

	user.RoleId = role.Id
	user.UpdatedAt = time.Now().UTC()

	if _, err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return &pb.UpdateUserRoleResponse{Success: true}, nil
}

func (s *userService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	user, err := s.repo.GetUserById(ctx, req.Id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	user.UpdatedAt = time.Now().UTC()

	updatedUser, err := s.repo.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateUserResponse{
		Id:       updatedUser.Id,
		FullName: updatedUser.FullName,
		Email:    updatedUser.Email,
		Role:     updatedUser.Role.Name,
	}, nil
}

func (s *userService) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordResponse, error) {
	user, err := s.repo.GetUserById(ctx, req.Id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return nil, errors.New("mật khẩu cũ không đúng")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user.Password = string(newHash)
	user.UpdatedAt = time.Now().UTC()

	if _, err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return &pb.UpdatePasswordResponse{Success: true}, nil
}