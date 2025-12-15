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

func (s *userService) CreateClass(ctx context.Context, req *pb.CreateClassRequest) (*pb.CreateClassResponse, error) {
	class := &domain.ClassModel{
		Name: req.Name, Code: req.Code, Description: req.Description,
		TeacherID: req.TeacherId, CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateClass(ctx, class); err != nil { return nil, err }
	
    return &pb.CreateClassResponse{Class: mapClassToProto(class)}, nil
}

func (s *userService) UpdateClass(ctx context.Context, req *pb.UpdateClassRequest) (*pb.UpdateClassResponse, error) {
    class, err := s.repo.GetClassByID(ctx, req.Id)
    if err != nil { return nil, errors.New("class not found") }
    if class.TeacherID != req.TeacherId { return nil, errors.New("unauthorized") }

	updates := map[string]interface{}{}
	if req.Name != "" { updates["name"] = req.Name }
	if req.Description != "" { updates["description"] = req.Description }
	
	err = s.repo.UpdateClass(ctx, req.Id, updates)
	return &pb.UpdateClassResponse{Success: err == nil}, err
}

func (s *userService) DeleteClass(ctx context.Context, req *pb.DeleteClassRequest) (*pb.DeleteClassResponse, error) {
    class, err := s.repo.GetClassByID(ctx, req.Id)
    if err != nil { return nil, errors.New("class not found") }
    if class.TeacherID != req.TeacherId { return nil, errors.New("unauthorized") }

	err = s.repo.DeleteClass(ctx, req.Id)
	return &pb.DeleteClassResponse{Success: err == nil}, err
}

func (s *userService) GetClasses(ctx context.Context, req *pb.GetClassesRequest) (*pb.GetClassesResponse, error) {
	classes, total, err := s.repo.GetClasses(ctx, req.TeacherId, req.StudentId, int(req.Limit), int(req.Page-1)*int(req.Limit))
	if err != nil { return nil, err }

	var pbClasses []*pb.Class
	for _, c := range classes {
		pbClasses = append(pbClasses, mapClassToProto(c))
	}
	return &pb.GetClassesResponse{Classes: pbClasses, Total: total}, nil
}

func (s *userService) GetClassDetails(ctx context.Context, req *pb.GetClassDetailsRequest) (*pb.GetClassDetailsResponse, error) {
	class, err := s.repo.GetClassByID(ctx, req.ClassId)
	if err != nil { return nil, err }

	var members []*pb.ClassMember
	for _, m := range class.Members {
        if m.User != nil {
            members = append(members, &pb.ClassMember{
                UserId: m.UserID, FullName: m.User.FullName, Email: m.User.Email,
                Role: m.Role, JoinedAt: m.JoinedAt.Format(time.RFC3339),
            })
        }
	}
	return &pb.GetClassDetailsResponse{Class: mapClassToProto(class), Members: members}, nil
}

func (s *userService) AddMembers(ctx context.Context, req *pb.AddMembersRequest) (*pb.AddMembersResponse, error) {
    class, err := s.repo.GetClassByID(ctx, req.ClassId)
    if err != nil || class.TeacherID != req.TeacherId { return nil, errors.New("unauthorized or class not found") }

    success := 0
    failed := []string{}

    for _, email := range req.Emails {
        user, err := s.repo.GetUserByEmail(ctx, email)
        if err != nil {
            failed = append(failed, email)
            continue
        }
        err = s.repo.AddClassMember(ctx, &domain.ClassMemberModel{
            ClassID: req.ClassId, UserID: user.Id, Role: "student", JoinedAt: time.Now().UTC(),
        })
        if err == nil { success++ } else { failed = append(failed, email) }
    }
    return &pb.AddMembersResponse{SuccessCount: int32(success), FailedEmails: failed}, nil
}

func (s *userService) RemoveMember(ctx context.Context, req *pb.RemoveMemberRequest) (*pb.RemoveMemberResponse, error) {
    class, err := s.repo.GetClassByID(ctx, req.ClassId)
    if err != nil || class.TeacherID != req.TeacherId { return nil, errors.New("unauthorized") }

    err = s.repo.RemoveClassMember(ctx, req.ClassId, req.UserId)
    return &pb.RemoveMemberResponse{Success: err == nil}, err
}

func (s *userService) CheckUserInClass(ctx context.Context, req *pb.CheckUserInClassRequest) (*pb.CheckUserInClassResponse, error) {
    isMember := false
    for _, cid := range req.ClassIds {
        m, _ := s.repo.GetClassMember(ctx, cid, req.UserId)
        if m != nil {
            isMember = true; break
        }
    }
    return &pb.CheckUserInClassResponse{IsMember: isMember}, nil
}

func mapClassToProto(c *domain.ClassModel) *pb.Class {
    tName := ""
    if c.Teacher != nil { tName = c.Teacher.FullName }
    return &pb.Class{
        Id: c.Id, Name: c.Name, Code: c.Code, Description: c.Description,
        TeacherId: c.TeacherID, TeacherName: tName,
        StudentCount: int32(len(c.Members)), CreatedAt: c.CreatedAt.Format(time.RFC3339),
    }
}