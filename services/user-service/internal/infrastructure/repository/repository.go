package repository

import (
	"context"

	database "github.com/06babyshark06/JQKStudy/services/user-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/user-service/internal/domain"
)

type userRepository struct{}

func NewUserRepository() domain.UserRepository {
	return &userRepository{}
}

func (r *userRepository) CreateUser(ctx context.Context, user *domain.UserModel) (*domain.UserModel, error) {
	if err := database.DB.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*domain.UserModel, error) {
	var user domain.UserModel
	if err := database.DB.WithContext(ctx).Preload("Role").Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserById(ctx context.Context, id int64) (*domain.UserModel, error) {
	var user domain.UserModel
	if err := database.DB.WithContext(ctx).Preload("Role").Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *domain.UserModel) (*domain.UserModel, error) {
	if err := database.DB.WithContext(ctx).Save(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id int64) error {
	if err := database.DB.WithContext(ctx).Where("id = ?", id).Delete(&domain.UserModel{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *userRepository) GetUsersWithPagination(ctx context.Context, limit, offset int) ([]*domain.UserModel, int64, error) {
	var users []*domain.UserModel
	var total int64

	if err := database.DB.WithContext(ctx).Model(&domain.UserModel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := database.DB.WithContext(ctx).
		Preload("Role").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) CountUsers(ctx context.Context) (int64, error) {
    var count int64
    if err := database.DB.WithContext(ctx).Model(&domain.UserModel{}).Count(&count).Error; err != nil {
        return 0, err
    }
    return count, nil
}

func (r *userRepository) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	var role domain.Role
	if err := database.DB.WithContext(ctx).Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}