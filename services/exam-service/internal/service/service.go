package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	database "github.com/06babyshark06/JQKStudy/services/exam-service/internal/databases"
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

func (s *examService) CreateTopic(ctx context.Context, req *pb.CreateTopicRequest) (*pb.CreateTopicResponse, error) {
	// 1. Kiểm tra logic (ví dụ: trùng tên)
	existing, _ := s.repo.GetTopicByName(ctx, req.Name)
	if existing != nil {
		return nil, errors.New("chủ đề đã tồn tại")
	}

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

	return &pb.CreateTopicResponse{
		Topic: &pb.Topic{
			Id:          topic.Id,
			Name:        topic.Name,
			Description: topic.Description,
		},
	}, nil
}

func (s *examService) GetTopics(ctx context.Context, req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error) {
	topics, err := s.repo.GetTopics(ctx)
	if err != nil {
		return nil, err
	}

	var pbTopics []*pb.Topic
	for _, t := range topics {
		pbTopics = append(pbTopics, &pb.Topic{
			Id:          t.Id,
			Name:        t.Name,
			Description: t.Description,
		})
	}

	return &pb.GetTopicsResponse{Topics: pbTopics}, nil
}

func (s *examService) CreateQuestion(ctx context.Context, req *pb.CreateQuestionRequest) (*pb.CreateQuestionResponse, error) {
	var createdQuestion *domain.QuestionModel

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		var diff domain.QuestionDifficultyModel
		if err = tx.WithContext(ctx).Where("difficulty = ?", req.Difficulty).First(&diff).Error; err != nil {
			return errors.New("difficulty không hợp lệ")
		}
		var qtype domain.QuestionTypeModel
		if err = tx.WithContext(ctx).Where("type = ?", req.QuestionType).First(&qtype).Error; err != nil {
			return errors.New("question type không hợp lệ")
		}

		question := &domain.QuestionModel{
			TopicID:      req.TopicId,
			CreatorID:    req.CreatorId,
			Content:      req.Content,
			TypeID:       qtype.Id,
			DifficultyID: diff.Id,
			Explanation:  req.Explanation,
		}

		createdQuestion, err = s.repo.CreateQuestion(ctx, tx, question)
		if err != nil {
			return err
		}

		var choiceModels []*domain.ChoiceModel
		for _, c := range req.Choices {
			choiceModels = append(choiceModels, &domain.ChoiceModel{
				QuestionID: createdQuestion.Id,
				Content:    c.Content,
				IsCorrect:  c.IsCorrect,
			})
		}

		if err = s.repo.CreateChoices(ctx, tx, choiceModels); err != nil {
			return err
		}

		if req.ExamId > 0 {
            err = s.repo.LinkQuestionsToExam(ctx, tx, req.ExamId, []int64{createdQuestion.Id})
            if err != nil {
                return err
            }
        }

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateQuestionResponse{
		Id:      createdQuestion.Id,
		Content: createdQuestion.Content,
	}, nil
}

func (s *examService) CreateExam(ctx context.Context, req *pb.CreateExamRequest) (*pb.CreateExamResponse, error) {
	var createdExam *domain.ExamModel

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var err error

		exam := &domain.ExamModel{
			Title:           req.Title,
			Description:     req.Description,
			DurationMinutes: int(req.DurationMinutes),
			TopicID:         req.TopicId,
			CreatorID:       req.CreatorId,
		}

		createdExam, err = s.repo.CreateExam(ctx, tx, exam)
		if err != nil {
			return err
		}

		if err = s.repo.LinkQuestionsToExam(ctx, tx, createdExam.Id, req.QuestionIds); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateExamResponse{
		Id:    createdExam.Id,
		Title: createdExam.Title,
	}, nil
}

func (s *examService) GetExamDetails(ctx context.Context, req *pb.GetExamDetailsRequest) (*pb.GetExamDetailsResponse, error) {
	examModel, err := s.repo.GetExamDetails(ctx, req.ExamId)
	if err != nil {
		return nil, err
	}

	pbQuestions := []*pb.QuestionDetails{}
	for _, q := range examModel.Questions {
		pbChoices := []*pb.ChoiceDetails{}
		for _, c := range q.Choices {
			pbChoices = append(pbChoices, &pb.ChoiceDetails{
				Id:      c.Id,
				Content: c.Content,
			})
		}
		qType := "single_choice"
		if q.Type.Type != "" {
			qType = q.Type.Type
		}
		pbQuestions = append(pbQuestions, &pb.QuestionDetails{
			Id:      q.Id,
			Content: q.Content,
			Choices: pbChoices,
			QuestionType: qType,
		})
	}

	return &pb.GetExamDetailsResponse{
		Id:              examModel.Id,
		Title:           examModel.Title,
		DurationMinutes: int32(examModel.DurationMinutes),
		Questions:       pbQuestions,
		TopicId:         examModel.TopicID,
		IsPublished:     examModel.IsPublished,
		Description:     examModel.Description,
	}, nil
}

func (s *examService) SubmitExam(ctx context.Context, req *pb.SubmitExamRequest) (*pb.SubmitExamResponse, error) {
	var correctCount int32 = 0
	var totalQuestions int32 = 0
	var finalScore float64 = 0
	var submissionID int64 = 0
	var examTitle string

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

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var exam domain.ExamModel
		if err := tx.WithContext(ctx).Select("title").First(&exam, req.ExamId).Error; err != nil {
			return errors.New("không tìm thấy bài thi")
		}
		examTitle = exam.Title
		var inProgressStatus domain.SubmissionStatusModel
		if err := tx.WithContext(ctx).Where("status = ?", "in_progress").First(&inProgressStatus).Error; err != nil {
			return errors.New("không tìm thấy status 'in_progress'")
		}
		
		submission := &domain.ExamSubmissionModel{
			ExamID:    req.ExamId,
			UserID:    req.UserId, 
			StatusID:  inProgressStatus.Id,
			StartedAt: time.Now().UTC(),
		}
		createdSubmission, err := s.repo.CreateSubmission(ctx, tx, submission)
		if err != nil {
			return err
		}
		submissionID = createdSubmission.Id

		var userAnswerModels []*domain.UserAnswerModel
		totalQuestions = int32(len(correctMap))

		for qID, correctChoices := range correctMap {
			userChoices := userAnswerMap[qID]

			isCorrect := compareInt64Slices(userChoices, correctChoices)

			if isCorrect {
				correctCount++
			}

			for _, cID := range userChoices {
				choiceIDVal := cID
				correctVal := isCorrect

				userAnswerModels = append(userAnswerModels, &domain.UserAnswerModel{
					SubmissionID:   createdSubmission.Id,
					QuestionID:     qID,
					ChosenChoiceID: &choiceIDVal,
					IsCorrect:      &correctVal,
				})
			}
		}

		if len(userAnswerModels) > 0 {
			if err := s.repo.CreateUserAnswers(ctx, tx, userAnswerModels); err != nil {
				return err
			}
		}

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

		if _, err := s.repo.UpdateSubmission(ctx, tx, createdSubmission); err != nil {
			return err
		}

		return nil
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
		Email:        req.Email,    
		FullName:     req.FullName,
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
		SubmissionId:   submissionID,
		Score:          float32(finalScore),
		CorrectCount:   correctCount,
		TotalQuestions: totalQuestions,
	}, nil
}

func compareInt64Slices(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	
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
	submission, err := s.repo.GetSubmissionByID(ctx, req.SubmissionId)
	if err != nil {
		return nil, errors.New("không tìm thấy kết quả bài thi")
	}

	if submission.UserID != req.UserId {
		return nil, errors.New("bạn không có quyền xem kết quả này")
	}
	examFull, err := s.repo.GetExamDetails(ctx, submission.ExamID)
	if err != nil {
		return nil, errors.New("không thể tải nội dung đề thi gốc")
	}

	userSelections := make(map[int64]map[int64]bool)
	
	questionIsCorrectMap := make(map[int64]bool)

	for _, ua := range submission.UserAnswers {
		if _, exists := userSelections[ua.QuestionID]; !exists {
			userSelections[ua.QuestionID] = make(map[int64]bool)
		}
		if ua.ChosenChoiceID != nil {
			userSelections[ua.QuestionID][*ua.ChosenChoiceID] = true
		}
		
		if ua.IsCorrect != nil && *ua.IsCorrect {
			questionIsCorrectMap[ua.QuestionID] = true
		}
	}

	var pbDetails []*pb.SubmissionDetail

	for _, q := range examFull.Questions {
		var pbChoices []*pb.ChoiceReview
		
		for _, c := range q.Choices {
			pbChoices = append(pbChoices, &pb.ChoiceReview{
				Id:           c.Id,
				Content:      c.Content,
				IsCorrect:    c.IsCorrect,
				UserSelected: userSelections[q.Id][c.Id],
			})
		}

        qType := "single_choice"
        if q.Type.Type != "" { qType = q.Type.Type }

		pbDetails = append(pbDetails, &pb.SubmissionDetail{
			QuestionId:      q.Id,
			QuestionContent: q.Content,
			Explanation:     q.Explanation,
			QuestionType:    qType,
			IsCorrect:       questionIsCorrectMap[q.Id],
			Choices:         pbChoices,
		})
	}

	correctCount := 0
	for _, ans := range submission.UserAnswers {
		if ans.IsCorrect != nil && *ans.IsCorrect {
			correctCount++
		}
	}
    correctCount = 0
    for _, v := range questionIsCorrectMap {
        if v { correctCount++ }
    }

	submittedAt := ""
	if submission.SubmittedAt != nil {
		submittedAt = submission.SubmittedAt.Format(time.RFC3339)
	}

	return &pb.GetSubmissionResponse{
		Id:             submission.Id,
		ExamTitle:      submission.Exam.Title,
		Score:          float32(submission.Score),
		CorrectCount:   int32(correctCount),
		TotalQuestions: int32(len(examFull.Questions)),
		Status:         submission.Status.Status,
		SubmittedAt:    submittedAt,
		Details:        pbDetails,
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
		
		var diff domain.QuestionDifficultyModel
		if err := tx.WithContext(ctx).Where("difficulty = ?", req.Difficulty).First(&diff).Error; err != nil {
			return errors.New("difficulty không hợp lệ: " + req.Difficulty)
		}

		var qtype domain.QuestionTypeModel
		if err := tx.WithContext(ctx).Where("type = ?", req.QuestionType).First(&qtype).Error; err != nil {
			return errors.New("question type không hợp lệ: " + req.QuestionType)
		}

		updates := map[string]interface{}{
			"content":       req.Content,
			"explanation":   req.Explanation,
			"difficulty_id": diff.Id,
			"type_id":       qtype.Id,
		}
		
		if err := s.repo.UpdateQuestion(ctx, tx, req.QuestionId, updates); err != nil {
			return err
		}

		if err := s.repo.DeleteChoicesByQuestionID(ctx, tx, req.QuestionId); err != nil {
			return err
		}
		
		if len(req.Choices) > 0 {
			var choices []*domain.ChoiceModel
			for _, c := range req.Choices {
				choices = append(choices, &domain.ChoiceModel{
					QuestionID: req.QuestionId,
					Content:    c.Content,
					IsCorrect:  c.IsCorrect,
				})
			}

			if err := s.repo.CreateChoices(ctx, tx, choices); err != nil {
				return err
			}
		}

		return nil
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

func (s *examService) GetUserExamStats(ctx context.Context, req *pb.GetUserExamStatsRequest) (*pb.GetUserExamStatsResponse, error) {
    count, err := s.repo.CountSubmissionsByUserID(ctx, req.UserId)
    if err != nil {
        return nil, err
    }
    return &pb.GetUserExamStatsResponse{
        TotalExamsTaken: count,
    }, nil
}