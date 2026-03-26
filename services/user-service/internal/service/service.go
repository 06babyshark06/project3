package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	database "github.com/06babyshark06/JQKStudy/services/user-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/user-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/user"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type userService struct {
	repo     domain.UserRepository
	producer domain.EventProducer
}

func NewUserService(repo domain.UserRepository, producer domain.EventProducer) domain.UserService {
	return &userService{repo: repo, producer: producer}
}

func (s *userService) getUserVersion(ctx context.Context, userID int64) string {
	if database.RedisClient == nil {
		return "0"
	}
	v, err := database.RedisClient.Get(ctx, fmt.Sprintf("user:version:%d", userID)).Result()
	if err == redis.Nil {
		database.RedisClient.Set(ctx, fmt.Sprintf("user:version:%d", userID), "1", 24*time.Hour)
		return "1"
	}
	return v
}

func (s *userService) invalidateUserCache(ctx context.Context, userID int64) {
	if database.RedisClient == nil {
		return
	}
	database.RedisClient.Incr(ctx, fmt.Sprintf("user:version:%d", userID))
	log.Printf("♻️ Invalidated cache for user %d", userID)
}

func (s *userService) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	// Try to get from cache
	version := s.getUserVersion(ctx, req.UserId)
	cacheKey := fmt.Sprintf("user:profile:%d:v:%s", req.UserId, version)

	if database.RedisClient != nil {
		cachedData, err := database.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var resp pb.GetProfileResponse
			if err := json.Unmarshal([]byte(cachedData), &resp); err == nil {
				log.Printf("🔹 Cache Hit: %s", cacheKey)
				return &resp, nil
			}
		}
	}

	user, err := s.repo.GetUserById(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	resp := &pb.GetProfileResponse{
		Id:       user.Id,
		FullName: user.FullName,
		Email:    user.Email,
		Role:     user.Role.Name,
	}

	// Save to cache
	if database.RedisClient != nil {
		data, _ := json.Marshal(resp)
		database.RedisClient.Set(ctx, cacheKey, data, 1*time.Hour)
	}

	return resp, nil
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
	if limit <= 0 {
		limit = 10
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	users, total, err := s.repo.GetUsersWithPagination(ctx, limit, offset, req.Search, req.Role)
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
	if err != nil {
		return nil, err
	}
	return &pb.GetUserCountResponse{Count: count}, nil
}

func (s *userService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if err := s.repo.DeleteUser(ctx, req.Id); err != nil {
		return nil, err
	}
	if database.RedisClient != nil {
		database.RedisClient.Del(ctx, fmt.Sprintf("user:version:%d", req.Id))
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
	user.Role = domain.Role{} // Clear preloaded association to ensure RoleId update persists
	user.UpdatedAt = time.Now().UTC()

	if _, err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}
	s.invalidateUserCache(ctx, req.Id)
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
	s.invalidateUserCache(ctx, req.Id)
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
	s.invalidateUserCache(ctx, req.Id)
	return &pb.UpdatePasswordResponse{Success: true}, nil
}

func (s *userService) CreateClass(ctx context.Context, req *pb.CreateClassRequest) (*pb.CreateClassResponse, error) {
	class := &domain.ClassModel{
		Name: req.Name, Code: req.Code, Description: req.Description,
		TeacherID: req.TeacherId, CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateClass(ctx, class); err != nil {
		return nil, err
	}

	return &pb.CreateClassResponse{Class: mapClassToProto(class)}, nil
}

func (s *userService) UpdateClass(ctx context.Context, req *pb.UpdateClassRequest) (*pb.UpdateClassResponse, error) {
	class, err := s.repo.GetClassByID(ctx, req.Id)
	if err != nil {
		return nil, errors.New("class not found")
	}
	// Check requester permission
	requester, err := s.repo.GetUserById(ctx, req.TeacherId)
	if err != nil {
		return nil, errors.New("requester not found")
	}

	// Validate: must be Owner OR Admin
	if class.TeacherID != req.TeacherId && requester.Role.Name != "admin" {
		return nil, errors.New("unauthorized")
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	err = s.repo.UpdateClass(ctx, req.Id, updates)
	return &pb.UpdateClassResponse{Success: err == nil}, err
}

func (s *userService) DeleteClass(ctx context.Context, req *pb.DeleteClassRequest) (*pb.DeleteClassResponse, error) {
	class, err := s.repo.GetClassByID(ctx, req.Id)
	if err != nil {
		return nil, errors.New("class not found")
	}
	// Check requester permission
	requester, err := s.repo.GetUserById(ctx, req.TeacherId)
	if err != nil {
		return nil, errors.New("requester not found")
	}

	// Validate: must be Owner OR Admin
	if class.TeacherID != req.TeacherId && requester.Role.Name != "admin" {
		return nil, errors.New("unauthorized")
	}

	err = s.repo.DeleteClass(ctx, req.Id)
	return &pb.DeleteClassResponse{Success: err == nil}, err
}

func (s *userService) GetClasses(ctx context.Context, req *pb.GetClassesRequest) (*pb.GetClassesResponse, error) {
	classes, total, err := s.repo.GetClasses(ctx, req.TeacherId, req.StudentId, int(req.Limit), int(req.Page-1)*int(req.Limit))
	if err != nil {
		return nil, err
	}

	var pbClasses []*pb.Class
	for _, c := range classes {
		pbClasses = append(pbClasses, mapClassToProto(c))
	}
	return &pb.GetClassesResponse{Classes: pbClasses, Total: total}, nil
}

func (s *userService) GetClassDetails(ctx context.Context, req *pb.GetClassDetailsRequest) (*pb.GetClassDetailsResponse, error) {
	class, err := s.repo.GetClassByID(ctx, req.ClassId)
	if err != nil {
		return nil, err
	}

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
	if err != nil {
		return nil, errors.New("class not found")
	}

	// Check requester permission
	requester, err := s.repo.GetUserById(ctx, req.TeacherId)
	if err != nil {
		return nil, errors.New("requester not found")
	}

	// Validate: must be Owner OR Admin
	if class.TeacherID != req.TeacherId && requester.Role.Name != "admin" {
		return nil, errors.New("unauthorized")
	}

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
		if err == nil {
			success++
		} else {
			failed = append(failed, email)
		}
	}
	return &pb.AddMembersResponse{SuccessCount: int32(success), FailedEmails: failed}, nil
}

func (s *userService) AddMembersBulk(ctx context.Context, req *pb.AddMembersRequest) (*pb.AddMembersResponse, error) {
	class, err := s.repo.GetClassByID(ctx, req.ClassId)
	if err != nil {
		return nil, errors.New("class not found")
	}

	// Check requester permission
	requester, err := s.repo.GetUserById(ctx, req.TeacherId)
	if err != nil {
		return nil, errors.New("requester not found")
	}

	// Validate: must be Owner OR Admin
	if class.TeacherID != req.TeacherId && requester.Role.Name != "admin" {
		return nil, errors.New("unauthorized")
	}

	// 1. Lấy tất cả user có email trong danh sách (Bulk fetch)
	users, err := s.repo.GetUsersByEmails(ctx, req.Emails)
	if err != nil {
		return nil, err
	}

	foundEmails := make(map[string]int64)
	for _, u := range users {
		foundEmails[u.Email] = u.Id
	}

	failed := []string{}
	membersToAdd := []*domain.ClassMemberModel{}
	now := time.Now().UTC()

	// 2. Phân loại email tìm thấy và không tìm thấy
	for _, email := range req.Emails {
		if id, ok := foundEmails[email]; ok {
			membersToAdd = append(membersToAdd, &domain.ClassMemberModel{
				ClassID:  req.ClassId,
				UserID:   id,
				Role:     "student",
				JoinedAt: now,
			})
		} else {
			failed = append(failed, email)
		}
	}

	// 3. Thực hiện Bulk Insert vào Class Members
	if len(membersToAdd) > 0 {
		err = s.repo.AddClassMembersBulk(ctx, membersToAdd)
		if err != nil {
			return nil, err
		}
	}

	return &pb.AddMembersResponse{
		SuccessCount: int32(len(membersToAdd)),
		FailedEmails: failed,
	}, nil
}

func (s *userService) RemoveMember(ctx context.Context, req *pb.RemoveMemberRequest) (*pb.RemoveMemberResponse, error) {
	class, err := s.repo.GetClassByID(ctx, req.ClassId)
	if err != nil {
		return nil, errors.New("class not found")
	}

	// Check requester permission
	requester, err := s.repo.GetUserById(ctx, req.TeacherId)
	if err != nil {
		return nil, errors.New("requester not found")
	}

	// Validate: must be Owner OR Admin
	if class.TeacherID != req.TeacherId && requester.Role.Name != "admin" {
		return nil, errors.New("unauthorized")
	}

	err = s.repo.RemoveClassMember(ctx, req.ClassId, req.UserId)
	return &pb.RemoveMemberResponse{Success: err == nil}, err
}

func (s *userService) CheckUserInClass(ctx context.Context, req *pb.CheckUserInClassRequest) (*pb.CheckUserInClassResponse, error) {
	isMember := false
	for _, cid := range req.ClassIds {
		m, _ := s.repo.GetClassMember(ctx, cid, req.UserId)
		if m != nil {
			isMember = true
			break
		}
	}
	return &pb.CheckUserInClassResponse{IsMember: isMember}, nil
}

func (s *userService) JoinClassByCode(ctx context.Context, req *pb.JoinClassByCodeRequest) (*pb.JoinClassByCodeResponse, error) {
	class, err := s.repo.GetClassByCode(ctx, req.Code)
	if err != nil {
		return &pb.JoinClassByCodeResponse{
			Success: false,
			Message: "Mã lớp không tồn tại",
		}, nil
	}

	exists, err := s.repo.IsClassMember(ctx, class.Id, req.UserId)
	if err != nil {
		return nil, err
	}
	if exists {
		return &pb.JoinClassByCodeResponse{
			Success: false,
			Message: "Bạn đã tham gia lớp này rồi",
			ClassId: class.Id,
		}, nil
	}

	member := &domain.ClassMemberModel{
		ClassID:  class.Id,
		UserID:   req.UserId,
		Role:     "student",
		JoinedAt: time.Now().UTC(),
	}

	err = s.repo.AddClassMember(ctx, member)
	if err != nil {
		return nil, err
	}

	return &pb.JoinClassByCodeResponse{
		Success: true,
		Message: "Tham gia lớp thành công",
		ClassId: class.Id,
	}, nil
}

func mapClassToProto(c *domain.ClassModel) *pb.Class {
	tName := ""
	if c.Teacher != nil {
		tName = c.Teacher.FullName
	}
	return &pb.Class{
		Id: c.Id, Name: c.Name, Code: c.Code, Description: c.Description,
		TeacherId: c.TeacherID, TeacherName: tName,
		StudentCount: int32(len(c.Members)), CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
}
