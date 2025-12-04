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
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	"gorm.io/gorm"
)

type notificationService struct {
	repo  domain.NotificationRepository
	email domain.EmailProvider
}

func NewNotificationService(repo domain.NotificationRepository, email domain.EmailProvider) domain.NotificationService {
	return &notificationService{
		repo:  repo,
		email: email,
	}
}

func (s *notificationService) HandleUserRegisteredEvent(ctx context.Context, eventBytes []byte) error {
	var event contracts.UserRegisteredEvent
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		log.Printf("Lỗi parse sự kiện user_registered: %v", err)
		return errors.New("dữ liệu sự kiện không hợp lệ")
	}

	template, err := s.repo.GetTemplateByName(ctx, "user_registered")
	if err != nil {
		log.Printf("Không tìm thấy template 'user_registered': %v", err)
		return err
	}

	body := fmt.Sprintf(template.Body, event.FullName)
	subject := template.Subject

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

	err = s.email.SendEmail(ctx, event.Email, subject, body)

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

	errUpdate := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateLogStatus(ctx, tx, notificationLog.Id, statusToUpdate, errMsg)
	})
	
	if errUpdate != nil {
		log.Printf("LỖI CẬP NHẬT LOG: %v", errUpdate)
		return err
	}

	return err
}

func (s *notificationService) HandleExamSubmittedEvent(ctx context.Context, eventBytes []byte) error {
	var event contracts.ExamSubmittedEvent
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		log.Printf("Lỗi parse sự kiện exam_submitted: %v", err)
		return errors.New("dữ liệu sự kiện không hợp lệ")
	}

	template, err := s.repo.GetTemplateByName(ctx, "exam_submitted")
	if err != nil {
		log.Printf("Không tìm thấy template 'exam_submitted': %v", err)
		return err
	}

	body := fmt.Sprintf(template.Body, event.FullName, event.ExamTitle, event.Score)
	subject := fmt.Sprintf(template.Subject, event.ExamTitle)

	pendingStatus, _ := s.repo.GetStatusByName(ctx, "pending")
	sentStatus, _ := s.repo.GetStatusByName(ctx, "sent")
	failedStatus, _ := s.repo.GetStatusByName(ctx, "failed")
	if pendingStatus == nil || sentStatus == nil || failedStatus == nil {
		return errors.New("không thể lấy các status ID từ CSDL")
	}


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

	err = s.email.SendEmail(ctx, event.Email, subject, body)

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

	return err
}

func (s *notificationService) HandleCourseEnrolledEvent(ctx context.Context, eventBytes []byte) error {
	var event contracts.CourseEnrolledEvent
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		log.Printf("Lỗi parse sự kiện course_enrolled: %v", err)
		return errors.New("dữ liệu sự kiện không hợp lệ")
	}

	template, err := s.repo.GetTemplateByName(ctx, "course_enrolled")
	if err != nil {
		log.Printf("Không tìm thấy template 'course_enrolled': %v", err)
		return err
	}

	body := fmt.Sprintf(template.Body, event.FullName, event.CourseTitle)
	subject := fmt.Sprintf(template.Subject, event.CourseTitle)

	pendingStatus, _ := s.repo.GetStatusByName(ctx, "pending")
	sentStatus, _ := s.repo.GetStatusByName(ctx, "sent")
	failedStatus, _ := s.repo.GetStatusByName(ctx, "failed")
	if pendingStatus == nil || sentStatus == nil || failedStatus == nil {
		return errors.New("không thể lấy các status ID từ CSDL")
	}

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

	err = s.email.SendEmail(ctx, event.Email, subject, body)

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