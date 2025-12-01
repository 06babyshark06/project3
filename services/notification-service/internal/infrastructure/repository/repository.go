package repository

import (
	"context"

	database "github.com/06babyshark06/JQKStudy/services/notification-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/domain"
	"gorm.io/gorm"
)

type notificationRepository struct{}

func NewNotificationRepository() domain.NotificationRepository {
	return &notificationRepository{}
}

func (r *notificationRepository) GetTemplateByName(ctx context.Context, name string) (*domain.NotificationTemplateModel, error) {
	var template domain.NotificationTemplateModel
	if err := database.DB.WithContext(ctx).Preload("Type").Where("name = ?", name).First(&template).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *notificationRepository) GetStatusByName(ctx context.Context, status string) (*domain.NotificationStatusModel, error) {
	var statusModel domain.NotificationStatusModel
	if err := database.DB.WithContext(ctx).Where("status = ?", status).First(&statusModel).Error; err != nil {
		return nil, err
	}
	return &statusModel, nil
}

func (r *notificationRepository) CreateNotificationLog(ctx context.Context, tx *gorm.DB, log *domain.NotificationModel) (*domain.NotificationModel, error) {
	if err := tx.WithContext(ctx).Create(log).Error; err != nil {
		return nil, err
	}
	return log, nil
}

func (r *notificationRepository) UpdateLogStatus(ctx context.Context, tx *gorm.DB, logID int64, statusID int64, errorMessage string) error {
	updates := map[string]interface{}{
		"status_id":     statusID,
		"error_message": errorMessage,
		"sent_at":       gorm.Expr("NOW()"),
	}

	if err := tx.WithContext(ctx).Model(&domain.NotificationModel{}).Where("id = ?", logID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}