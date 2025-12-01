package repository

import (
	"context"

	// THAY ĐỔI: Trỏ đến database của notification-service
	database "github.com/06babyshark06/JQKStudy/services/notification-service/internal/databases"
	// THAY ĐỔI: Trỏ đến domain của notification-service
	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/domain"
	"gorm.io/gorm"
)

// THAY ĐỔI: Tên struct
type notificationRepository struct{}

// THAY ĐỔI: Tên hàm
func NewNotificationRepository() domain.NotificationRepository {
	return &notificationRepository{}
}

// GetTemplateByName (Đọc - Dùng global DB)
func (r *notificationRepository) GetTemplateByName(ctx context.Context, name string) (*domain.NotificationTemplateModel, error) {
	var template domain.NotificationTemplateModel
	// Preload "Type" để lấy tên của ChannelType
	if err := database.DB.WithContext(ctx).Preload("Type").Where("name = ?", name).First(&template).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// GetStatusByName (Đọc - Dùng global DB)
func (r *notificationRepository) GetStatusByName(ctx context.Context, status string) (*domain.NotificationStatusModel, error) {
	var statusModel domain.NotificationStatusModel
	if err := database.DB.WithContext(ctx).Where("status = ?", status).First(&statusModel).Error; err != nil {
		return nil, err
	}
	return &statusModel, nil
}

// CreateNotificationLog (Ghi - Dùng tx)
func (r *notificationRepository) CreateNotificationLog(ctx context.Context, tx *gorm.DB, log *domain.NotificationModel) (*domain.NotificationModel, error) {
	if err := tx.WithContext(ctx).Create(log).Error; err != nil {
		return nil, err
	}
	return log, nil
}

// UpdateLogStatus (Ghi - Dùng tx)
func (r *notificationRepository) UpdateLogStatus(ctx context.Context, tx *gorm.DB, logID int64, statusID int64, errorMessage string) error {
	// Chỉ cập nhật các trường cần thiết
	updates := map[string]interface{}{
		"status_id":     statusID,
		"error_message": errorMessage,
		"sent_at":       gorm.Expr("NOW()"), // Dùng NOW() nếu status là "sent" hoặc "failed"
	}

	if err := tx.WithContext(ctx).Model(&domain.NotificationModel{}).Where("id = ?", logID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}