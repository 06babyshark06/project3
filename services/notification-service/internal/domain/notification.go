package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// =================================================================
// GORM MODELS (Dựa trên cấu trúc database bạn cung cấp)
// =================================================================

// ChannelTypeModel ✉️
type ChannelTypeModel struct {
	Id   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Type string `gorm:"size:50;uniqueIndex;not null" json:"type"` // "email", "sms", "push"
}

// NotificationStatusModel 📊
type NotificationStatusModel struct {
	Id     int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Status string `gorm:"size:50;uniqueIndex;not null" json:"status"` // "pending", "sent", "failed"
}

// NotificationTemplateModel 📋
type NotificationTemplateModel struct {
	Id        int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string           `gorm:"size:100;uniqueIndex;not null" json:"name"` // "user_registered", "exam_submitted"
	TypeID    int64            `gorm:"not null" json:"type_id"`
	Type      ChannelTypeModel `gorm:"foreignKey:TypeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"type"`
	Subject   string           `gorm:"size:255" json:"subject"` // Mẫu tiêu đề
	Body      string           `gorm:"type:text" json:"body"`   // Mẫu nội dung (HTML/Text)
	CreatedAt time.Time        `json:"created_at"`
}

// NotificationModel 📬
type NotificationModel struct {
	Id              int64                   `gorm:"primaryKey;autoIncrement" json:"id"`
	RecipientID     int64                   `gorm:"not null;index" json:"recipient_id"` // User ID
	TypeID          int64                   `gorm:"not null" json:"type_id"`
	Type            ChannelTypeModel        `gorm:"foreignKey:TypeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"type"`
	StatusID        int64                   `gorm:"not null" json:"status_id"`
	Status          NotificationStatusModel `gorm:"foreignKey:StatusID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"status"`
	RenderedContent string                  `gorm:"type:text" json:"rendered_content"` // Nội dung cuối cùng đã được render
	ErrorMessage    string                  `gorm:"type:text" json:"error_message"`
	ScheduledAt     time.Time               `json:"scheduled_at"`
	SentAt          *time.Time              `json:"sent_at"` // Sửa lỗi chính tả từ Sented_at
	CreatedAt       time.Time               `json:"created_at"`
}


// =================================================================
// INTERFACES (Định nghĩa các "Hợp đồng")
// =================================================================

// NotificationRepository định nghĩa các phương thức tương tác với DB
type NotificationRepository interface {
	// Dùng để lấy template khi xử lý sự kiện
	GetTemplateByName(ctx context.Context, name string) (*NotificationTemplateModel, error)

	// Dùng để lấy ID của status (ví dụ: "pending")
	GetStatusByName(ctx context.Context, status string) (*NotificationStatusModel, error)

	// Dùng để tạo log thông báo
	CreateNotificationLog(ctx context.Context, tx *gorm.DB, log *NotificationModel) (*NotificationModel, error)
	
	// Dùng để cập nhật log (thành công / thất bại)
	UpdateLogStatus(ctx context.Context, tx *gorm.DB, logID int64, statusID int64, errorMessage string) error
}

// NotificationService định nghĩa các logic nghiệp vụ
// Các hàm này sẽ được gọi bởi Kafka Consumer
type NotificationService interface {
	// Xử lý sự kiện đăng ký người dùng mới
	HandleUserRegisteredEvent(ctx context.Context, eventBytes []byte) error

	// Xử lý sự kiện nộp bài thi
	HandleExamSubmittedEvent(ctx context.Context, eventBytes []byte) error

	// (Có thể thêm các hàm xử lý sự kiện khác ở đây)
	HandleCourseEnrolledEvent(ctx context.Context, eventBytes []byte) error
}

// EmailProvider là interface cho một dịch vụ bên thứ 3 (SendGrid, Mailgun, etc.)
type EmailProvider interface {
	SendEmail(ctx context.Context, toEmail string, subject string, htmlBody string) error
}