package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	database "github.com/06babyshark06/JQKStudy/services/exam-service/internal/databases" // Import global DB
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	"gorm.io/gorm"
)

type examService struct {
	repo domain.ExamRepository
	producer domain.EventProducer
}

func NewExamService(repo domain.ExamRepository, producer domain.EventProducer) domain.ExamService {
	return &examService{repo: repo, producer: producer}
}

// CreateTopic implements domain.ExamService.
func (s *examService) CreateTopic(ctx context.Context, req *pb.CreateTopicRequest) (*pb.CreateTopicResponse, error) {
	// 1. Kiểm tra logic (ví dụ: trùng tên)
	existing, _ := s.repo.GetTopicByName(ctx, req.Name)
	if existing != nil {
		return nil, errors.New("chủ đề đã tồn tại")
	}

	// 2. Map từ proto sang model
	topic := &domain.TopicModel{
		Name:        req.Name,
		Description: req.Description,
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		createdTopic, err := s.repo.CreateTopic(ctx, tx, topic)
		if err != nil {
			return err
		}
		topic = createdTopic
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 4. Map từ model sang proto response
	return &pb.CreateTopicResponse{
		Topic: &pb.Topic{
			Id:          topic.Id, // Gán trực tiếp
			Name:        topic.Name,
			Description: topic.Description,
		},
	}, nil
}

// GetTopics implements domain.ExamService.
func (s *examService) GetTopics(ctx context.Context, req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error) {
	topics, err := s.repo.GetTopics(ctx)
	if err != nil {
		return nil, err
	}

	// Map []*domain.TopicModel sang []*pb.Topic
	var pbTopics []*pb.Topic
	for _, t := range topics {
		pbTopics = append(pbTopics, &pb.Topic{
			Id:          t.Id, // Gán trực tiếp
			Name:        t.Name,
			Description: t.Description,
		})
	}

	return &pb.GetTopicsResponse{Topics: pbTopics}, nil
}

// CreateQuestion implements domain.ExamService.
func (s *examService) CreateQuestion(ctx context.Context, req *pb.CreateQuestionRequest) (*pb.CreateQuestionResponse, error) {
	var createdQuestion *domain.QuestionModel

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		// 1. Lấy ID của difficulty và type
		var diff domain.QuestionDifficultyModel
		if err = tx.WithContext(ctx).Where("difficulty = ?", req.Difficulty).First(&diff).Error; err != nil {
			return errors.New("difficulty không hợp lệ")
		}
		var qtype domain.QuestionTypeModel
		if err = tx.WithContext(ctx).Where("type = ?", req.QuestionType).First(&qtype).Error; err != nil {
			return errors.New("question type không hợp lệ")
		}

		// 2. Tạo QuestionModel
		question := &domain.QuestionModel{
			TopicID:      req.TopicId,   // Gán trực tiếp
			CreatorID:    req.CreatorId, // Gán trực tiếp
			Content:      req.Content,
			TypeID:       qtype.Id,
			DifficultyID: diff.Id,
			Explanation:  req.Explanation,
		}

		// 3. Tạo câu hỏi trong DB (SỬ DỤNG REPO)
		createdQuestion, err = s.repo.CreateQuestion(ctx, tx, question)
		if err != nil {
			return err
		}

		// 4. Tạo các Choices
		var choiceModels []*domain.ChoiceModel
		for _, c := range req.Choices {
			choiceModels = append(choiceModels, &domain.ChoiceModel{
				QuestionID: createdQuestion.Id, // Lấy ID từ câu hỏi vừa tạo
				Content:    c.Content,
				IsCorrect:  c.IsCorrect,
			})
		}

		// 5. Bulk insert các choices (SỬ DỤNG REPO)
		if err = s.repo.CreateChoices(ctx, tx, choiceModels); err != nil {
			return err
		}

		if req.ExamId > 0 {
            // Gọi hàm repo có sẵn để link
            err = s.repo.LinkQuestionsToExam(ctx, tx, req.ExamId, []int64{createdQuestion.Id})
            if err != nil {
                return err
            }
        }

		return nil // Commit transaction
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateQuestionResponse{
		Id:      createdQuestion.Id, // Gán trực tiếp
		Content: createdQuestion.Content,
	}, nil
}

// CreateExam implements domain.ExamService.
func (s *examService) CreateExam(ctx context.Context, req *pb.CreateExamRequest) (*pb.CreateExamResponse, error) {
	var createdExam *domain.ExamModel

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var err error

		// 1. Tạo Exam
		exam := &domain.ExamModel{
			Title:           req.Title,
			Description:     req.Description,
			DurationMinutes: int(req.DurationMinutes),
			TopicID:         req.TopicId,   // Gán trực tiếp
			CreatorID:       req.CreatorId, // Gán trực tiếp
		}

		// SỬ DỤNG REPO
		createdExam, err = s.repo.CreateExam(ctx, tx, exam)
		if err != nil {
			return err
		}

		// 2. Link các câu hỏi vào exam
		// req.QuestionIds đã là []int64, không cần loop
		if err = s.repo.LinkQuestionsToExam(ctx, tx, createdExam.Id, req.QuestionIds); err != nil {
			return err
		}

		return nil // Commit
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateExamResponse{
		Id:    createdExam.Id, // Gán trực tiếp
		Title: createdExam.Title,
	}, nil
}

// GetExamDetails implements domain.ExamService.
func (s *examService) GetExamDetails(ctx context.Context, req *pb.GetExamDetailsRequest) (*pb.GetExamDetailsResponse, error) {
	// Nghiệp vụ này chỉ đọc, dùng Repo
	examModel, err := s.repo.GetExamDetails(ctx, req.ExamId) // Gán trực tiếp
	if err != nil {
		return nil, err
	}

	// Map domain.ExamModel sang pb.GetExamDetailsResponse
	pbQuestions := []*pb.QuestionDetails{}
	for _, q := range examModel.Questions {
		pbChoices := []*pb.ChoiceDetails{}
		for _, c := range q.Choices {
			pbChoices = append(pbChoices, &pb.ChoiceDetails{
				Id:      c.Id, // Gán trực tiếp
				Content: c.Content,
			})
		}
		qType := "single_choice"
		if q.Type.Type != "" {
			qType = q.Type.Type
		}
		pbQuestions = append(pbQuestions, &pb.QuestionDetails{
			Id:      q.Id, // Gán trực tiếp
			Content: q.Content,
			Choices: pbChoices,
			QuestionType: qType,
		})
	}

	return &pb.GetExamDetailsResponse{
		Id:              examModel.Id, // Gán trực tiếp
		Title:           examModel.Title,
		DurationMinutes: int32(examModel.DurationMinutes),
		Questions:       pbQuestions,
		TopicId:         examModel.TopicID,
		IsPublished:     examModel.IsPublished,
	}, nil
}

// SubmitExam implements domain.ExamService.
func (s *examService) SubmitExam(ctx context.Context, req *pb.SubmitExamRequest) (*pb.SubmitExamResponse, error) {
	var correctCount int32 = 0
	var totalQuestions int32 = 0
	var finalScore float64 = 0
	var submissionID int64 = 0
	var examTitle string

	// 1. Lấy đáp án đúng (Thao tác ĐỌC, nên làm BÊN NGOÀI transaction)
	correctMap, err := s.repo.GetCorrectAnswers(ctx, req.ExamId)
	if err != nil {
		return nil, errors.New("lỗi khi lấy đáp án: " + err.Error())
	}

	userAnswerMap := make(map[int64][]int64)
	for _, ans := range req.Answers {
		if ans.ChosenChoiceId != 0 {
			userAnswerMap[ans.QuestionId] = append(userAnswerMap[ans.QuestionId], ans.ChosenChoiceId)
		}
	}
	totalQuestions = int32(len(correctMap))

	// 2. Bắt đầu transaction để GHI dữ liệu
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var exam domain.ExamModel
		if err := tx.WithContext(ctx).Select("title").First(&exam, req.ExamId).Error; err != nil {
			return errors.New("không tìm thấy bài thi")
		}
		examTitle = exam.Title
		// 2. Lấy status "in_progress"
		var inProgressStatus domain.SubmissionStatusModel
		if err := tx.WithContext(ctx).Where("status = ?", "in_progress").First(&inProgressStatus).Error; err != nil {
			return errors.New("không tìm thấy status 'in_progress'")
		}
		
		// 3. Tạo ExamSubmission (SỬ DỤNG REPO)
		submission := &domain.ExamSubmissionModel{
			ExamID:    req.ExamId, // Gán trực tiếp
			UserID:    req.UserId, // Gán trực tiếp
			StatusID:  inProgressStatus.Id,
			StartedAt: time.Now().UTC(),
		}
		createdSubmission, err := s.repo.CreateSubmission(ctx, tx, submission)
		if err != nil {
			return err
		}
		submissionID = createdSubmission.Id // Lưu ID để trả về

		// 4. Chấm điểm và chuẩn bị UserAnswers
		var userAnswerModels []*domain.UserAnswerModel
		totalQuestions = int32(len(req.Answers))

		for qID, correctChoices := range correctMap {
			userChoices := userAnswerMap[qID] // Các đáp án user chọn cho câu này

			// So sánh 2 mảng: User chọn vs Đáp án đúng
			isCorrect := compareInt64Slices(userChoices, correctChoices)

			if isCorrect {
				correctCount++
			}

			// Lưu chi tiết câu trả lời vào DB
			// Lưu ý: Với Multiple Choice, ta lưu từng lựa chọn của user thành 1 dòng trong DB
			for _, cID := range userChoices {
				// Tạo biến tạm để lấy địa chỉ con trỏ
				choiceIDVal := cID
				correctVal := isCorrect // Lưu kết quả đúng/sai của CÂU HỎI

				userAnswerModels = append(userAnswerModels, &domain.UserAnswerModel{
					SubmissionID:   createdSubmission.Id,
					QuestionID:     qID,
					ChosenChoiceID: &choiceIDVal,
					IsCorrect:      &correctVal,
				})
			}
		}

		// 5. Bulk insert UserAnswers (SỬ DỤNG REPO)
		if len(userAnswerModels) > 0 {
			if err := s.repo.CreateUserAnswers(ctx, tx, userAnswerModels); err != nil {
				return err
			}
		}

		// 6. Tính điểm và cập nhật Submission
		if totalQuestions > 0 {
			finalScore = (float64(correctCount) / float64(totalQuestions)) * 10.0
		}

		var completedStatus domain.SubmissionStatusModel
		if err := tx.WithContext(ctx).Where("status = ?", "completed").First(&completedStatus).Error; err != nil {
			return errors.New("không tìm thấy status 'completed'")
		}

		now := time.Now().UTC()
		createdSubmission.StatusID = completedStatus.Id
		createdSubmission.Score = finalScore
		createdSubmission.SubmittedAt = &now

		// SỬ DỤNG REPO
		if _, err := s.repo.UpdateSubmission(ctx, tx, createdSubmission); err != nil {
			return err
		}

		return nil // Commit transaction
	})

	if err != nil {
		return nil, err
	}

	eventPayload := contracts.ExamSubmittedEvent{
		UserID:       req.UserId,
		ExamID:       req.ExamId,
		SubmissionID: submissionID,
		ExamTitle:    examTitle,
		Score:        finalScore,
	}
	eventBytes, err := json.Marshal(eventPayload)
	if err != nil {
		log.Printf("LỖI: Không thể marshal sự kiện exam_submitted: %v", err)
	} else {
		key := []byte(strconv.FormatInt(submissionID, 10))
		err = s.producer.Produce("exam_events", key, eventBytes)
		if err != nil {
			log.Printf("LỖI: Không thể gửi sự kiện exam_submitted: %v", err)
		}
	}

	return &pb.SubmitExamResponse{
		SubmissionId:   submissionID, // Gán trực tiếp
		Score:          float32(finalScore),
		CorrectCount:   correctCount,
		TotalQuestions: totalQuestions,
	}, nil
}

func compareInt64Slices(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	
	// Tạo map để đếm tần suất xuất hiện của từng ID
	countMap := make(map[int64]int)
	
	for _, x := range a {
		countMap[x]++
	}
	
	for _, x := range b {
		countMap[x]--
		if countMap[x] < 0 {
			return false
		}
	}
	
	return true
}

func (s *examService) GetSubmission(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error) {
	// 1. Lấy dữ liệu từ DB
	submission, err := s.repo.GetSubmissionByID(ctx, req.SubmissionId)
	if err != nil {
		return nil, errors.New("không tìm thấy kết quả bài thi")
	}

	// 2. Bảo mật: Kiểm tra xem người yêu cầu có phải chủ nhân bài thi không
	// (Trừ khi là admin - logic này có thể mở rộng sau)
	if submission.UserID != req.UserId {
		return nil, errors.New("bạn không có quyền xem kết quả này")
	}

	// 3. Tính toán số câu đúng/tổng số câu
	correctCount := 0
	totalQuestions := len(submission.UserAnswers)
	for _, ans := range submission.UserAnswers {
		if ans.IsCorrect != nil && *ans.IsCorrect {
			correctCount++
		}
	}

	// 4. Format thời gian
	submittedAt := ""
	if submission.SubmittedAt != nil {
		submittedAt = submission.SubmittedAt.Format(time.RFC3339)
	}

	return &pb.GetSubmissionResponse{
		Id:             submission.Id,
		ExamTitle:      submission.Exam.Title,
		Score:          float32(submission.Score),
		CorrectCount:   int32(correctCount),
		TotalQuestions: int32(totalQuestions),
		Status:         submission.Status.Status,
		SubmittedAt:    submittedAt,
	}, nil
}

func (s *examService) GetExamCount(ctx context.Context, req *pb.GetExamCountRequest) (*pb.GetExamCountResponse, error) {
    count, err := s.repo.CountExams(ctx)
    if err != nil { return nil, err }
    return &pb.GetExamCountResponse{Count: count}, nil
}

func (s *examService) GetExams(ctx context.Context, req *pb.GetExamsRequest) (*pb.GetExamsResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 { limit = 10 }
	offset := (int(req.Page) - 1) * limit
	if offset < 0 { offset = 0 }

	exams, total, err := s.repo.GetExams(ctx, limit, offset, req.CreatorId)
	if err != nil { return nil, err }

	var pbExams []*pb.ExamListItem
	for _, e := range exams {
		pbExams = append(pbExams, &pb.ExamListItem{
			Id:              e.Id,
			Title:           e.Title,
			DurationMinutes: int32(e.DurationMinutes),
			TopicId:         e.TopicID,
			CreatorId:       e.CreatorID,
			IsPublished:     e.IsPublished,
		})
	}
    
    // Tính total pages
    totalPages := int32((total + int64(limit) - 1) / int64(limit))

	return &pb.GetExamsResponse{
		Exams:      pbExams,
		Total:      total,
		Page:       req.Page,
		TotalPages: totalPages,
	}, nil
}

func (s *examService) PublishExam(ctx context.Context, req *pb.PublishExamRequest) (*pb.PublishExamResponse, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateExamStatus(ctx, tx, req.ExamId, req.IsPublished)
	})
	if err != nil {
		return nil, err
	}
	return &pb.PublishExamResponse{Success: true}, nil
}

func (s *examService) UpdateQuestion(ctx context.Context, req *pb.UpdateQuestionRequest) (*pb.UpdateQuestionResponse, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		
		// === 1. Lấy ID của Difficulty và QuestionType ===
		// (Phần này quan trọng để chuyển đổi từ string sang ID trong DB)
		
		var diff domain.QuestionDifficultyModel
		if err := tx.WithContext(ctx).Where("difficulty = ?", req.Difficulty).First(&diff).Error; err != nil {
			return errors.New("difficulty không hợp lệ: " + req.Difficulty)
		}

		var qtype domain.QuestionTypeModel
		if err := tx.WithContext(ctx).Where("type = ?", req.QuestionType).First(&qtype).Error; err != nil {
			return errors.New("question type không hợp lệ: " + req.QuestionType)
		}

		// === 2. Cập nhật bảng QuestionModel ===
		updates := map[string]interface{}{
			"content":       req.Content,
			"explanation":   req.Explanation,
			"difficulty_id": diff.Id,  // Cập nhật ID mới lấy được
			"type_id":       qtype.Id, // Cập nhật ID mới lấy được
			// "updated_at" sẽ được GORM tự động cập nhật
		}
		
		if err := s.repo.UpdateQuestion(ctx, tx, req.QuestionId, updates); err != nil {
			return err
		}

		// === 3. Cập nhật Đáp án (Chiến lược: Xóa hết cũ -> Tạo mới) ===
		
		// 3a. Xóa các đáp án cũ
		if err := s.repo.DeleteChoicesByQuestionID(ctx, tx, req.QuestionId); err != nil {
			return err
		}
		
		// 3b. Tạo danh sách đáp án mới từ request
		if len(req.Choices) > 0 {
			var choices []*domain.ChoiceModel
			for _, c := range req.Choices {
				choices = append(choices, &domain.ChoiceModel{
					QuestionID: req.QuestionId, // Link với câu hỏi đang sửa
					Content:    c.Content,
					IsCorrect:  c.IsCorrect,
				})
			}
			// 3c. Bulk insert đáp án mới
			if err := s.repo.CreateChoices(ctx, tx, choices); err != nil {
				return err
			}
		}

		return nil // Commit transaction
	})

	if err != nil {
		return nil, err
	}

	return &pb.UpdateQuestionResponse{Success: true}, nil
}

func (s *examService) DeleteQuestion(ctx context.Context, req *pb.DeleteQuestionRequest) (*pb.DeleteQuestionResponse, error) {
    err := database.DB.Transaction(func(tx *gorm.DB) error {
        return s.repo.DeleteQuestion(ctx, tx, req.QuestionId)
    })
    if err != nil { return nil, err }
    return &pb.DeleteQuestionResponse{Success: true}, nil
}

func (s *examService) UpdateExam(ctx context.Context, req *pb.UpdateExamRequest) (*pb.UpdateExamResponse, error) {
    err := database.DB.Transaction(func(tx *gorm.DB) error {
        updates := make(map[string]interface{})
        
        if req.Title != "" { updates["title"] = req.Title }
        if req.Description != "" { updates["description"] = req.Description }
        if req.DurationMinutes > 0 { updates["duration_minutes"] = req.DurationMinutes }
        if req.TopicId > 0 { updates["topic_id"] = req.TopicId }

        return s.repo.UpdateExam(ctx, tx, req.ExamId, updates)
    })
    if err != nil { return nil, err }
    return &pb.UpdateExamResponse{Success: true}, nil
}

func (s *examService) DeleteExam(ctx context.Context, req *pb.DeleteExamRequest) (*pb.DeleteExamResponse, error) {
    err := database.DB.Transaction(func(tx *gorm.DB) error {
        return s.repo.DeleteExam(ctx, tx, req.ExamId)
    })
    if err != nil { return nil, err }
    return &pb.DeleteExamResponse{Success: true}, nil
}