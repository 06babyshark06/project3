package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	database "github.com/06babyshark06/JQKStudy/services/notification-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/domain"
	"gorm.io/gorm"
)

// Struct cho các sự kiện (events) từ Kafka
type UserRegisteredEvent struct {
	UserID   int64  `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type ExamSubmittedEvent struct {
	UserID    int64   `json:"user_id"`
	Email     string  `json:"email"`
	FullName  string  `json:"full_name"`
	ExamTitle string  `json:"exam_title"`
	Score     float64 `json:"score"`
}

type CourseEnrolledEvent struct {
	UserID      int64  `json:"user_id"`
	CourseID    int64  `json:"course_id"`
	CourseTitle string `json:"course_title"`
	Email       string `json:"email"`     // Giả định producer sẽ gửi kèm
	FullName    string `json:"full_name"` // Giả định producer sẽ gửi kèm
}

type notificationService struct {
	repo  domain.NotificationRepository
	email domain.EmailProvider
}

// THAY ĐỔI: NewNotificationService cần cả repo và email provider
func NewNotificationService(repo domain.NotificationRepository, email domain.EmailProvider) domain.NotificationService {
	return &notificationService{
		repo:  repo,
		email: email,
	}
}

// HandleUserRegisteredEvent implements domain.NotificationService.
// Đây là logic xử lý sự kiện từ Kafka
func (s *notificationService) HandleUserRegisteredEvent(ctx context.Context, eventBytes []byte) error {
	// 1. Parse sự kiện
	var event UserRegisteredEvent
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		log.Printf("Lỗi parse sự kiện user_registered: %v", err)
		return errors.New("dữ liệu sự kiện không hợp lệ")
	}

	// 2. Lấy template
	template, err := s.repo.GetTemplateByName(ctx, "user_registered")
	if err != nil {
		log.Printf("Không tìm thấy template 'user_registered': %v", err)
		return err
	}

	// 3. Render nội dung
	// (Trong dự án thực tế, bạn sẽ dùng html/template để thay thế các biến)
	body := fmt.Sprintf(template.Body, event.FullName)
	subject := template.Subject

	// 4. Lấy các Status ID (đọc)
	pendingStatus, err := s.repo.GetStatusByName(ctx, "pending")
	if err != nil {
		return err
	}
	sentStatus, err := s.repo.GetStatusByName(ctx, "sent")
	if err != nil {
		return err
	}
	failedStatus, err := s.repo.GetStatusByName(ctx, "failed")
	if err != nil {
		return err
	}

	// 5. GHI LOG (trong transaction)
	var notificationLog *domain.NotificationModel
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		logEntry := &domain.NotificationModel{
			RecipientID:     event.UserID,
			TypeID:          template.TypeID,
			StatusID:        pendingStatus.Id,
			RenderedContent: body, // (Trong thực tế nên lưu cả subject, v.v. vào JSON)
			ScheduledAt:     time.Now().UTC(),
		}
		var createErr error
		notificationLog, createErr = s.repo.CreateNotificationLog(ctx, tx, logEntry)
		return createErr
	})

	if err != nil {
		log.Printf("Lỗi khi tạo log thông báo: %v", err)
		return err
	}

	// 6. GỬI EMAIL (bên ngoài transaction)
	err = s.email.SendEmail(ctx, event.Email, subject, body)

	// 7. CẬP NHẬT LOG (trong transaction mới)
	var statusToUpdate int64
	var errMsg string
	if err != nil {
		log.Printf("LỖI GỬI EMAIL: %v", err)
		statusToUpdate = failedStatus.Id
		errMsg = err.Error()
	} else {
		log.Printf("Gửi email 'user_registered' cho %s thành công", event.Email)
		statusToUpdate = sentStatus.Id
		errMsg = ""
	}

	// Cập nhật log
	errUpdate := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateLogStatus(ctx, tx, notificationLog.Id, statusToUpdate, errMsg)
	})
	
	if errUpdate != nil {
		log.Printf("LỖI CẬP NHẬT LOG: %v", errUpdate)
		// Dù lỗi cập nhật, email cũng đã gửi (hoặc thất bại)
		// Trả về lỗi gốc (err) của việc gửi email
		return err
	}

	return err // Trả về lỗi của việc gửi email (nếu có)
}

// HandleExamSubmittedEvent implements domain.NotificationService.
func (s *notificationService) HandleExamSubmittedEvent(ctx context.Context, eventBytes []byte) error {
	// 1. Parse sự kiện
	var event ExamSubmittedEvent
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		log.Printf("Lỗi parse sự kiện exam_submitted: %v", err)
		return errors.New("dữ liệu sự kiện không hợp lệ")
	}

	// 2. Lấy template
	template, err := s.repo.GetTemplateByName(ctx, "exam_submitted")
	if err != nil {
		log.Printf("Không tìm thấy template 'exam_submitted': %v", err)
		return err
	}

	// 3. Render nội dung
	body := fmt.Sprintf(template.Body, event.FullName, event.ExamTitle, event.Score)
	subject := fmt.Sprintf(template.Subject, event.ExamTitle)

	// 4. Lấy các Status ID (đọc)
	pendingStatus, _ := s.repo.GetStatusByName(ctx, "pending")
	sentStatus, _ := s.repo.GetStatusByName(ctx, "sent")
	failedStatus, _ := s.repo.GetStatusByName(ctx, "failed")
	if pendingStatus == nil || sentStatus == nil || failedStatus == nil {
		return errors.New("không thể lấy các status ID từ CSDL")
	}


	// 5. GHI LOG (trong transaction)
	var notificationLog *domain.NotificationModel
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		logEntry := &domain.NotificationModel{
			RecipientID:     event.UserID,
			TypeID:          template.TypeID,
			StatusID:        pendingStatus.Id,
			RenderedContent: body,
			ScheduledAt:     time.Now().UTC(),
		}
		var createErr error
		notificationLog, createErr = s.repo.CreateNotificationLog(ctx, tx, logEntry)
		return createErr
	})

	if err != nil {
		log.Printf("Lỗi khi tạo log thông báo: %v", err)
		return err
	}

	// 6. GỬI EMAIL (bên ngoài transaction)
	err = s.email.SendEmail(ctx, event.Email, subject, body)

	// 7. CẬP NHẬT LOG (trong transaction mới)
	var statusToUpdate int64
	var errMsg string
	if err != nil {
		log.Printf("LỖI GỬI EMAIL: %v", err)
		statusToUpdate = failedStatus.Id
		errMsg = err.Error()
	} else {
		log.Printf("Gửi email 'exam_submitted' cho %s thành công", event.Email)
		statusToUpdate = sentStatus.Id
		errMsg = ""
	}

	errUpdate := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateLogStatus(ctx, tx, notificationLog.Id, statusToUpdate, errMsg)
	})
	
	if errUpdate != nil {
		log.Printf("LỖI CẬP NHẬT LOG: %v", errUpdate)
		return err
	}

	return err // Trả về lỗi của việc gửi email (nếu có)
}

func (s *notificationService) HandleCourseEnrolledEvent(ctx context.Context, eventBytes []byte) error {
	// 1. Parse sự kiện
	var event CourseEnrolledEvent
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		log.Printf("Lỗi parse sự kiện course_enrolled: %v", err)
		return errors.New("dữ liệu sự kiện không hợp lệ")
	}

	// 2. Lấy template
	template, err := s.repo.GetTemplateByName(ctx, "course_enrolled")
	if err != nil {
		log.Printf("Không tìm thấy template 'course_enrolled': %v", err)
		return err
	}

	// 3. Render nội dung
	body := fmt.Sprintf(template.Body, event.FullName, event.CourseTitle)
	subject := fmt.Sprintf(template.Subject, event.CourseTitle)

	// 4. Lấy các Status ID (đọc)
	pendingStatus, _ := s.repo.GetStatusByName(ctx, "pending")
	sentStatus, _ := s.repo.GetStatusByName(ctx, "sent")
	failedStatus, _ := s.repo.GetStatusByName(ctx, "failed")
	if pendingStatus == nil || sentStatus == nil || failedStatus == nil {
		return errors.New("không thể lấy các status ID từ CSDL")
	}

	// 5. GHI LOG (trong transaction)
	var notificationLog *domain.NotificationModel
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		logEntry := &domain.NotificationModel{
			RecipientID:     event.UserID,
			TypeID:          template.TypeID,
			StatusID:        pendingStatus.Id,
			RenderedContent: body,
			ScheduledAt:     time.Now().UTC(),
		}
		var createErr error
		notificationLog, createErr = s.repo.CreateNotificationLog(ctx, tx, logEntry)
		return createErr
	})

	if err != nil {
		log.Printf("Lỗi khi tạo log thông báo: %v", err)
		return err
	}

	// 6. GỬI EMAIL (bên ngoài transaction)
	// QUAN TRỌNG: Giả định 'event.Email' được gửi kèm trong payload
	err = s.email.SendEmail(ctx, event.Email, subject, body)

	// 7. CẬP NHẬT LOG (trong transaction mới)
	var statusToUpdate int64
	var errMsg string
	if err != nil {
		log.Printf("LỖI GỬI EMAIL: %v", err)
		statusToUpdate = failedStatus.Id
		errMsg = err.Error()
	} else {
		log.Printf("Gửi email 'course_enrolled' cho %s thành công", event.Email)
		statusToUpdate = sentStatus.Id
		errMsg = ""
	}

	errUpdate := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateLogStatus(ctx, tx, notificationLog.Id, statusToUpdate, errMsg)
	})
	
	if errUpdate != nil {
		log.Printf("LỖI CẬP NHẬT LOG: %v", errUpdate)
		return err
	}

	return err
}