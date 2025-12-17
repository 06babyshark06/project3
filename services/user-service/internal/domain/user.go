package domain

import (
	"context"
	"time"

	pb "github.com/06babyshark06/JQKStudy/shared/proto/user"
)

type Role struct {
	Id   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"size:50;uniqueIndex;not null" json:"name"`
}

type UserModel struct {
	Id        int64     `gorm:"primary_key,autoIncrement" json:"id"`
	FullName  string    `gorm:"size:255;not null" json:"full_name"`
	Email     string    `gorm:"size:255;not null;uniqueIndex" json:"email"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	RoleId    int64     `gorm:"not null" json:"role_id"`
	Role      Role      `gorm:"foreignKey:RoleId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserPayload struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	RoleId   int64  `json:"role_id"`
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *UserModel) (*UserModel, error)
	GetUserByEmail(ctx context.Context, email string) (*UserModel, error)
	GetUserById(ctx context.Context, id int64) (*UserModel, error)
	UpdateUser(ctx context.Context, user *UserModel) (*UserModel, error)
	DeleteUser(ctx context.Context, id int64) error
	GetUsersWithPagination(ctx context.Context, limit, offset int, search, role string) ([]*UserModel, int64, error)
	CountUsers(ctx context.Context) (int64, error)
	GetRoleByName(ctx context.Context, name string) (*Role, error)

	CreateClass(ctx context.Context, class *ClassModel) error
	UpdateClass(ctx context.Context, id int64, updates map[string]interface{}) error
	DeleteClass(ctx context.Context, id int64) error
	GetClassByID(ctx context.Context, id int64) (*ClassModel, error)
	GetClasses(ctx context.Context, teacherID, studentID int64, limit, offset int) ([]*ClassModel, int64, error)

	AddClassMember(ctx context.Context, member *ClassMemberModel) error
	RemoveClassMember(ctx context.Context, classID, userID int64) error
	GetClassMember(ctx context.Context, classID, userID int64) (*ClassMemberModel, error)
	GetClassByCode(ctx context.Context, code string) (*ClassModel, error)
	IsClassMember(ctx context.Context, classID, userID int64) (bool, error)
}

type EventProducer interface {
	Produce(topic string, key []byte, message []byte) error
	Close()
}

type UserService interface {
	Register(ctx context.Context, user *pb.RegisterRequest) (*pb.RegisterResponse, error)
	Login(ctx context.Context, user *pb.LoginRequest) (*pb.LoginResponse, error)
	GetProfile(ctx context.Context, userId *pb.GetProfileRequest) (*pb.GetProfileResponse, error)
	GetAllUsers(ctx context.Context, req *pb.GetAllUsersRequest) (*pb.GetAllUsersResponse, error)
	GetUserCount(ctx context.Context, req *pb.GetUserCountRequest) (*pb.GetUserCountResponse, error)
	DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error)
	UpdateUserRole(ctx context.Context, req *pb.UpdateUserRoleRequest) (*pb.UpdateUserRoleResponse, error)
	UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error)
	UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordResponse, error)

	CreateClass(ctx context.Context, req *pb.CreateClassRequest) (*pb.CreateClassResponse, error)
	UpdateClass(ctx context.Context, req *pb.UpdateClassRequest) (*pb.UpdateClassResponse, error)
	DeleteClass(ctx context.Context, req *pb.DeleteClassRequest) (*pb.DeleteClassResponse, error)
	GetClasses(ctx context.Context, req *pb.GetClassesRequest) (*pb.GetClassesResponse, error)
	GetClassDetails(ctx context.Context, req *pb.GetClassDetailsRequest) (*pb.GetClassDetailsResponse, error)
	AddMembers(ctx context.Context, req *pb.AddMembersRequest) (*pb.AddMembersResponse, error)
	RemoveMember(ctx context.Context, req *pb.RemoveMemberRequest) (*pb.RemoveMemberResponse, error)
	CheckUserInClass(ctx context.Context, req *pb.CheckUserInClassRequest) (*pb.CheckUserInClassResponse, error)
	JoinClassByCode(ctx context.Context, req *pb.JoinClassByCodeRequest) (*pb.JoinClassByCodeResponse, error)
}
