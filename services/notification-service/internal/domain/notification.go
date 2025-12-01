package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ChannelTypeModel struct {
	Id   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Type string `gorm:"size:50;uniqueIndex;not null" json:"type"`
}

type NotificationStatusModel struct {
	Id     int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Status string `gorm:"size:50;uniqueIndex;not null" json:"status"`
}

type NotificationTemplateModel struct {
	Id        int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string           `gorm:"size:100;uniqueIndex;not null" json:"name"`
	TypeID    int64            `gorm:"not null" json:"type_id"`
	Type      ChannelTypeModel `gorm:"foreignKey:TypeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"type"`
	Subject   string           `gorm:"size:255" json:"subject"`
	Body      string           `gorm:"type:text" json:"body"`
	CreatedAt time.Time        `json:"created_at"`
}

type NotificationModel struct {
	Id              int64                   `gorm:"primaryKey;autoIncrement" json:"id"`
	RecipientID     int64                   `gorm:"not null;index" json:"recipient_id"`
	TypeID          int64                   `gorm:"not null" json:"type_id"`
	Type            ChannelTypeModel        `gorm:"foreignKey:TypeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"type"`
	StatusID        int64                   `gorm:"not null" json:"status_id"`
	Status          NotificationStatusModel `gorm:"foreignKey:StatusID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"status"`
	RenderedContent string                  `gorm:"type:text" json:"rendered_content"`
	ErrorMessage    string                  `gorm:"type:text" json:"error_message"`
	ScheduledAt     time.Time               `json:"scheduled_at"`
	SentAt          *time.Time              `json:"sent_at"`
	CreatedAt       time.Time               `json:"created_at"`
}

type NotificationRepository interface {
	GetTemplateByName(ctx context.Context, name string) (*NotificationTemplateModel, error)

	GetStatusByName(ctx context.Context, status string) (*NotificationStatusModel, error)

	CreateNotificationLog(ctx context.Context, tx *gorm.DB, log *NotificationModel) (*NotificationModel, error)
	
	UpdateLogStatus(ctx context.Context, tx *gorm.DB, logID int64, statusID int64, errorMessage string) error
}

type NotificationService interface {
	HandleUserRegisteredEvent(ctx context.Context, eventBytes []byte) error

	HandleExamSubmittedEvent(ctx context.Context, eventBytes []byte) error

	HandleCourseEnrolledEvent(ctx context.Context, eventBytes []byte) error
}

type EmailProvider interface {
	SendEmail(ctx context.Context, toEmail string, subject string, htmlBody string) error
}