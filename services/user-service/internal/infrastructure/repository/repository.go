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

func (r *userRepository) CreateClass(ctx context.Context, class *domain.ClassModel) error {
	return database.DB.WithContext(ctx).Create(class).Error
}

func (r *userRepository) UpdateClass(ctx context.Context, id int64, updates map[string]interface{}) error {
	return database.DB.WithContext(ctx).Model(&domain.ClassModel{}).Where("id = ?", id).Updates(updates).Error
}

func (r *userRepository) DeleteClass(ctx context.Context, id int64) error {
	return database.DB.WithContext(ctx).Delete(&domain.ClassModel{}, id).Error
}

func (r *userRepository) GetClassByID(ctx context.Context, id int64) (*domain.ClassModel, error) {
	var class domain.ClassModel
	err := database.DB.WithContext(ctx).
		Preload("Teacher").
		Preload("Members").Preload("Members.User").
		First(&class, id).Error
	return &class, err
}

func (r *userRepository) GetClasses(ctx context.Context, teacherID, studentID int64, limit, offset int) ([]*domain.ClassModel, int64, error) {
	var classes []*domain.ClassModel
	var total int64
	db := database.DB.WithContext(ctx).Model(&domain.ClassModel{})

	if teacherID > 0 {
		db = db.Where("teacher_id = ?", teacherID)
	}
	if studentID > 0 {
		db = db.Joins("JOIN class_members cm ON classes.id = cm.class_id").
			Where("cm.user_id = ?", studentID)
	}

	db.Count(&total)
	err := db.Preload("Teacher").
        Preload("Members").
        Order("created_at DESC").Limit(limit).Offset(offset).Find(&classes).Error
	return classes, total, err
}

func (r *userRepository) AddClassMember(ctx context.Context, member *domain.ClassMemberModel) error {
	return database.DB.WithContext(ctx).FirstOrCreate(member, domain.ClassMemberModel{ClassID: member.ClassID, UserID: member.UserID}).Error
}

func (r *userRepository) RemoveClassMember(ctx context.Context, classID, userID int64) error {
	return database.DB.WithContext(ctx).Where("class_id = ? AND user_id = ?", classID, userID).Delete(&domain.ClassMemberModel{}).Error
}

func (r *userRepository) GetClassMember(ctx context.Context, classID, userID int64) (*domain.ClassMemberModel, error) {
	var m domain.ClassMemberModel
	err := database.DB.WithContext(ctx).Where("class_id = ? AND user_id = ?", classID, userID).First(&m).Error
	return &m, err
}

func (r *userRepository) GetClassByCode(ctx context.Context, code string) (*domain.ClassModel, error) {
	var class domain.ClassModel
	if err := database.DB.WithContext(ctx).Where("code = ?", code).First(&class).Error; err != nil {
		return nil, err
	}
	return &class, nil
}

func (r *userRepository) IsClassMember(ctx context.Context, classID, userID int64) (bool, error) {
	var count int64
	err := database.DB.WithContext(ctx).Model(&domain.ClassMemberModel{}).
		Where("class_id = ? AND user_id = ?", classID, userID).
		Count(&count).Error
	return count > 0, err
}