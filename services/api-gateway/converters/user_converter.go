package converters

import (
	pb "github.com/06babyshark06/JQKStudy/shared/proto/user"
)


type RegisterRequest struct {
	FullName string `json:"full_name" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type UpdateUserInfoRequest struct {
	FullName string `json:"full_name" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=6"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

type ChangeUserRoleRequest struct {
	RoleName string `json:"role_name" validate:"required,min=3"`
}
func ConvertRegisterJSONToProto(req *RegisterRequest) *pb.RegisterRequest {
	return &pb.RegisterRequest{
		FullName: req.FullName,
		Email:    req.Email,
		Password: req.Password,
	}
}

func ConvertLoginJSONToProto(req *LoginRequest) *pb.LoginRequest {
	return &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}
}
