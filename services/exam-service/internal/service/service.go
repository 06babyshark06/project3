package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xuri/excelize/v2"

	database "github.com/06babyshark06/JQKStudy/services/exam-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	"github.com/06babyshark06/JQKStudy/shared/env"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	"gorm.io/gorm"
)

type examService struct {
	repo     domain.ExamRepository
	producer domain.EventProducer
}

func NewExamService(repo domain.ExamRepository, producer domain.EventProducer) domain.ExamService {
	return &examService{repo: repo, producer: producer}
}

func (s *examService) createR2Client(ctx context.Context) (*s3.PresignClient, error) {
	accountID := env.GetString("R2_ACCOUNT_ID", "")
	accessKey := env.GetString("R2_ACCESS_KEY_ID", "")
	secretKey := env.GetString("R2_SECRET_ACCESS_KEY", "")
	if accountID == "" || accessKey == "" || secretKey == "" {
		return nil, errors.New("cấu hình R2 chưa đầy đủ")
	}
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	return s3.NewPresignClient(s3Client), nil
}

func (s *examService) GetUploadURL(ctx context.Context, req *pb.GetUploadURLRequest) (*pb.GetUploadURLResponse, error) {
	bucketName := env.GetString("R2_BUCKET_NAME", "")
	presignClient, err := s.createR2Client(ctx)
	if err != nil {
		return nil, err
	}

	fileKey := fmt.Sprintf("exams/%s/%d_%s", req.Folder, time.Now().Unix(), req.FileName)

	presignedReq, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(fileKey),
		ContentType: aws.String(req.ContentType),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return nil, err
	}

	publicDomain := env.GetString("R2_PUBLIC_DOMAIN", "")
	finalURL := fmt.Sprintf("https://%s/%s", publicDomain, fileKey)

	return &pb.GetUploadURLResponse{UploadUrl: presignedReq.URL, FinalUrl: finalURL}, nil
}

func (s *examService) CreateTopic(ctx context.Context, req *pb.CreateTopicRequest) (*pb.CreateTopicResponse, error) {
	topic := &domain.TopicModel{Name: req.Name, Description: req.Description}
	created, err := s.repo.CreateTopic(ctx, database.DB, topic)
	if err != nil {
		return nil, err
	}
	return &pb.CreateTopicResponse{Topic: &pb.Topic{Id: created.Id, Name: created.Name, Description: created.Description}}, nil
}
func (s *examService) GetTopics(ctx context.Context, req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error) {
	topics, err := s.repo.GetTopics(ctx)
	if err != nil {
		return nil, err
	}
	var pbTopics []*pb.Topic
	for _, t := range topics {
		pbTopics = append(pbTopics, &pb.Topic{Id: t.Id, Name: t.Name, Description: t.Description})
	}
	return &pb.GetTopicsResponse{Topics: pbTopics}, nil
}
func (s *examService) CreateSection(ctx context.Context, req *pb.CreateSectionRequest) (*pb.CreateSectionResponse, error) {
	sec := &domain.SectionModel{Name: req.Name, Description: req.Description, TopicID: req.TopicId}
	created, err := s.repo.CreateSection(ctx, database.DB, sec)
	if err != nil {
		return nil, err
	}
	return &pb.CreateSectionResponse{Section: &pb.Section{Id: created.Id, Name: created.Name, Description: created.Description, TopicId: created.TopicID}}, nil
}
func (s *examService) GetSections(ctx context.Context, req *pb.GetSectionsRequest) (*pb.GetSectionsResponse, error) {
	secs, err := s.repo.GetSectionsByTopic(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	var pbSecs []*pb.Section
	for _, s := range secs {
		pbSecs = append(pbSecs, &pb.Section{Id: s.Id, Name: s.Name, Description: s.Description, TopicId: s.TopicID})
	}
	return &pb.GetSectionsResponse{Sections: pbSecs}, nil
}

func (s *examService) CreateQuestion(ctx context.Context, req *pb.CreateQuestionRequest) (*pb.CreateQuestionResponse, error) {
	var qID int64
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		section, err := s.repo.GetSectionByID(ctx, req.SectionId)
		if err != nil {
			return errors.New("section not found")
		}
		diff, _ := s.repo.GetDifficulty(ctx, req.Difficulty)
		qType, _ := s.repo.GetQuestionType(ctx, req.QuestionType)

		question := &domain.QuestionModel{
			SectionID: req.SectionId, TopicID: section.TopicID, CreatorID: req.CreatorId,
			Content: req.Content, TypeID: qType.Id, DifficultyID: diff.Id,
			Explanation: req.Explanation, AttachmentURL: req.AttachmentUrl,
		}
		createdQ, err := s.repo.CreateQuestion(ctx, tx, question)
		if err != nil {
			return err
		}
		qID = createdQ.Id

		var choices []*domain.ChoiceModel
		for _, c := range req.Choices {
			choices = append(choices, &domain.ChoiceModel{QuestionID: qID, Content: c.Content, IsCorrect: c.IsCorrect, AttachmentURL: c.AttachmentUrl})
		}
		return s.repo.CreateChoices(ctx, tx, choices)
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateQuestionResponse{Id: qID, Content: req.Content}, nil
}

func (s *examService) ImportQuestions(ctx context.Context, req *pb.ImportQuestionsRequest) (*pb.ImportQuestionsResponse, error) {
	if len(req.FileContent) == 0 {
		return nil, errors.New("file excel rỗng")
	}
	reader := bytes.NewReader(req.FileContent)
	f, err := excelize.OpenReader(reader)
	if err != nil {
		log.Printf("Lỗi đọc excel: %v", err)
		return nil, errors.New("không thể đọc file excel, vui lòng kiểm tra định dạng")
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, errors.New("không tìm thấy Sheet1")
	}

	topicCache := make(map[string]int64)
	sectionCache := make(map[string]int64)
	typeCache := make(map[string]int64)
	diffCache := make(map[string]int64)

	getOrCreateTopic := func(name string) (int64, error) {
		name = strings.TrimSpace(name)
		if id, ok := topicCache[name]; ok {
			return id, nil
		}

		t, _ := s.repo.GetTopicByName(ctx, name)
		if t != nil {
			topicCache[name] = t.Id
			return t.Id, nil
		}

		newTopic := &domain.TopicModel{Name: name, Description: "Auto imported"}
		created, err := s.repo.CreateTopic(ctx, database.DB, newTopic)
		if err != nil {
			return 0, err
		}
		topicCache[name] = created.Id
		return created.Id, nil
	}

	getOrCreateSection := func(topicID int64, name string) (int64, error) {
		name = strings.TrimSpace(name)
		cacheKey := fmt.Sprintf("%d_%s", topicID, name)
		if id, ok := sectionCache[cacheKey]; ok {
			return id, nil
		}

		sections, _ := s.repo.GetSectionsByTopic(ctx, topicID)
		for _, sec := range sections {
			if strings.EqualFold(sec.Name, name) {
				sectionCache[cacheKey] = sec.Id
				return sec.Id, nil
			}
		}

		newSec := &domain.SectionModel{Name: name, Description: "Auto imported", TopicID: topicID}
		created, err := s.repo.CreateSection(ctx, database.DB, newSec)
		if err != nil {
			return 0, err
		}
		sectionCache[cacheKey] = created.Id
		return created.Id, nil
	}

	getTypeID := func(name string) int64 {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "multiple" || name == "multiple_choice" {
			name = "multiple_choice"
		} else {
			name = "single_choice"
		}
		if id, ok := typeCache[name]; ok {
			return id
		}
		t, _ := s.repo.GetQuestionType(ctx, name)
		if t != nil {
			typeCache[name] = t.Id
			return t.Id
		}
		return 1
	}
	getDiffID := func(name string) int64 {
		name = strings.TrimSpace(strings.ToLower(name))
		if id, ok := diffCache[name]; ok {
			return id
		}
		d, _ := s.repo.GetDifficulty(ctx, name)
		if d != nil {
			diffCache[name] = d.Id
			return d.Id
		}
		return 1
	}

	successCount := 0
	errorCount := 0

	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 9 {
			errorCount++
			continue
		}

		topicName := row[0]
		sectionName := row[1]
		content := strings.TrimSpace(row[2])
		if topicName == "" || sectionName == "" || content == "" {
			errorCount++
			continue
		}

		tID, err := getOrCreateTopic(topicName)
		if err != nil {
			errorCount++
			continue
		}

		sID, err := getOrCreateSection(tID, sectionName)
		if err != nil {
			errorCount++
			continue
		}
		qTypeID := getTypeID(row[3])
		diffID := getDiffID(row[4])
		explanation := row[5]
		imageURL := strings.TrimSpace(row[6])
		correctStr := strings.ToUpper(strings.TrimSpace(row[7]))

		correctMap := make(map[int]bool)
		parts := strings.Split(correctStr, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if len(p) > 0 {
				idx := int(p[0] - 'A')
				if idx >= 0 {
					correctMap[idx] = true
				}
			}
		}

		err = database.DB.Transaction(func(tx *gorm.DB) error {
			q := &domain.QuestionModel{
				SectionID: sID, TopicID: tID, CreatorID: req.CreatorId,
				Content: content, TypeID: qTypeID, DifficultyID: diffID, Explanation: explanation,
				AttachmentURL: imageURL,
			}
			createdQ, err := s.repo.CreateQuestion(ctx, tx, q)
			if err != nil {
				return err
			}

			var choices []*domain.ChoiceModel
			for cIdx := 8; cIdx < len(row); cIdx++ {
				val := strings.TrimSpace(row[cIdx])
				if val == "" {
					continue
				}
				isCorrect := correctMap[cIdx-8]
				choices = append(choices, &domain.ChoiceModel{
					QuestionID: createdQ.Id, Content: val, IsCorrect: isCorrect, AttachmentURL: "",
				})
			}
			if len(choices) < 2 {
				return errors.New("cần ít nhất 2 lựa chọn")
			}
			return s.repo.CreateChoices(ctx, tx, choices)
		})

		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	return &pb.ImportQuestionsResponse{SuccessCount: int32(successCount), ErrorCount: int32(errorCount)}, nil
}

func (s *examService) GenerateExam(ctx context.Context, req *pb.GenerateExamRequest) (*pb.CreateExamResponse, error) {
	allQuestionIDs := []int64{}
	uniqueMap := make(map[int64]bool)

	for _, cfg := range req.SectionConfigs {
		ids, err := s.repo.GetRandomQuestionsBySection(ctx, cfg.SectionId, cfg.Difficulty, int(cfg.Count))
		if err != nil {
			continue
		}
		for _, id := range ids {
			if !uniqueMap[id] {
				uniqueMap[id] = true
				allQuestionIDs = append(allQuestionIDs, id)
			}
		}
	}

	if len(allQuestionIDs) == 0 {
		return nil, errors.New("không tìm thấy câu hỏi nào phù hợp với cấu hình")
	}

	var examID int64
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		exam := &domain.ExamModel{
			Title: req.Title, Description: req.Description,
			DurationMinutes: int(req.Settings.DurationMinutes), MaxAttempts: int(req.Settings.MaxAttempts),
			Password: req.Settings.Password, ShuffleQuestions: req.Settings.ShuffleQuestions,
			ShowResultImmediately: req.Settings.ShowResultImmediately, RequiresApproval: req.Settings.RequiresApproval,
			TopicID: req.TopicId, CreatorID: req.CreatorId,
		}
		if req.Settings.StartTime != "" {
			t, _ := time.Parse(time.RFC3339, req.Settings.StartTime)
			exam.StartTime = &t
		}
		if req.Settings.EndTime != "" {
			t, _ := time.Parse(time.RFC3339, req.Settings.EndTime)
			exam.EndTime = &t
		}

		created, err := s.repo.CreateExam(ctx, tx, exam)
		if err != nil {
			return err
		}
		examID = created.Id

		return s.repo.LinkQuestionsToExam(ctx, tx, created.Id, allQuestionIDs)
	})

	if err != nil {
		return nil, err
	}
	return &pb.CreateExamResponse{Id: examID, Title: req.Title}, nil
}

func (s *examService) RequestExamAccess(ctx context.Context, req *pb.RequestExamAccessRequest) (*pb.RequestExamAccessResponse, error) {
	existing, _ := s.repo.GetAccessRequest(ctx, req.ExamId, req.UserId)
	if existing != nil {
		return &pb.RequestExamAccessResponse{Success: true, Status: existing.Status}, nil
	}
	newReq := &domain.ExamAccessRequestModel{
		ExamID: req.ExamId, UserID: req.UserId, Status: "pending", CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateAccessRequest(ctx, newReq); err != nil {
		return nil, err
	}
	return &pb.RequestExamAccessResponse{Success: true, Status: "pending"}, nil
}

func (s *examService) ApproveExamAccess(ctx context.Context, req *pb.ApproveExamAccessRequest) (*pb.ApproveExamAccessResponse, error) {
	status := "rejected"
	if req.IsApproved {
		status = "approved"
	}
	err := s.repo.UpdateAccessRequestStatus(ctx, req.ExamId, req.StudentId, status)
	if err != nil {
		return nil, err
	}
	return &pb.ApproveExamAccessResponse{Success: true}, nil
}

func (s *examService) CheckExamAccess(ctx context.Context, req *pb.CheckExamAccessRequest) (*pb.CheckExamAccessResponse, error) {
	exam, err := s.repo.GetExamDetails(ctx, req.ExamId)
	if err != nil {
		return nil, errors.New("exam not found")
	}

	if exam.RequiresApproval {
		access, err := s.repo.GetAccessRequest(ctx, req.ExamId, req.UserId)
		if err != nil || access == nil {
			return &pb.CheckExamAccessResponse{CanAccess: false, Message: "none"}, nil
		}
		if access.Status == "pending" {
			return &pb.CheckExamAccessResponse{CanAccess: false, Message: "pending"}, nil
		}
		if access.Status == "rejected" {
			return &pb.CheckExamAccessResponse{CanAccess: false, Message: "rejected"}, nil
		}
	}

	now := time.Now()
	if exam.StartTime != nil && now.Before(*exam.StartTime) {
		return &pb.CheckExamAccessResponse{CanAccess: false, Message: "not_started"}, nil
	}
	if exam.EndTime != nil && now.After(*exam.EndTime) {
		return &pb.CheckExamAccessResponse{CanAccess: false, Message: "ended"}, nil
	}

	count, _ := s.repo.CountSubmissionsForExam(ctx, req.ExamId, req.UserId)
	if exam.MaxAttempts > 0 && int(count) >= exam.MaxAttempts {
		return &pb.CheckExamAccessResponse{CanAccess: false, Message: "max_attempts"}, nil
	}

	return &pb.CheckExamAccessResponse{CanAccess: true, Message: "ok"}, nil
}

func (s *examService) CreateExam(ctx context.Context, req *pb.CreateExamRequest) (*pb.CreateExamResponse, error) {
	var createdExam *domain.ExamModel
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		exam := &domain.ExamModel{
			Title: req.Title, Description: req.Description,
			DurationMinutes: int(req.Settings.DurationMinutes), MaxAttempts: int(req.Settings.MaxAttempts),
			Password: req.Settings.Password, ShuffleQuestions: req.Settings.ShuffleQuestions,
			ShowResultImmediately: req.Settings.ShowResultImmediately, RequiresApproval: req.Settings.RequiresApproval,
			TopicID: req.TopicId, CreatorID: req.CreatorId,
		}
		if req.Settings.StartTime != "" {
			t, _ := time.Parse(time.RFC3339, req.Settings.StartTime)
			exam.StartTime = &t
		}
		if req.Settings.EndTime != "" {
			t, _ := time.Parse(time.RFC3339, req.Settings.EndTime)
			exam.EndTime = &t
		}

		var err error
		createdExam, err = s.repo.CreateExam(ctx, tx, exam)
		if err != nil {
			return err
		}
		return s.repo.LinkQuestionsToExam(ctx, tx, createdExam.Id, req.QuestionIds)
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateExamResponse{Id: createdExam.Id, Title: createdExam.Title}, nil
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
		difficulty := "medium"
		if q.Difficulty.Difficulty != "" {
			difficulty = q.Difficulty.Difficulty
		}
		sectionName := ""
		var sectionID int64 = 0
		topicName := ""
		var topicID int64 = 0

		if q.Section != nil {
			sectionName = q.Section.Name
			sectionID = q.Section.Id
			if q.Section.Topic != nil {
				topicName = q.Section.Topic.Name
				topicID = q.Section.Topic.Id
			}
		}
		pbQuestions = append(pbQuestions, &pb.QuestionDetails{
			Id:            q.Id,
			Content:       q.Content,
			Choices:       pbChoices,
			QuestionType:  qType,
			AttachmentUrl: q.AttachmentURL,
			Difficulty:    difficulty,
			Explanation:   q.Explanation,
			SectionName:   sectionName,
			TopicName:     topicName,
			SectionId:     sectionID,
			TopicId:       topicID,
		})
	}

	startTime := ""
	if examModel.StartTime != nil {
		startTime = examModel.StartTime.Format(time.RFC3339)
	}

	endTime := ""
	if examModel.EndTime != nil {
		endTime = examModel.EndTime.Format(time.RFC3339)
	}

	return &pb.GetExamDetailsResponse{
		Id:    examModel.Id,
		Title: examModel.Title,
		Settings: &pb.ExamSettings{
			DurationMinutes: int32(examModel.DurationMinutes),
			MaxAttempts:     int32(examModel.MaxAttempts),
			StartTime:       startTime,
			EndTime:         endTime,
		},
		Questions:   pbQuestions,
		TopicId:     examModel.TopicID,
		IsPublished: examModel.IsPublished,
		Description: examModel.Description,
	}, nil
}

func (s *examService) SubmitExam(ctx context.Context, req *pb.SubmitExamRequest) (*pb.SubmitExamResponse, error) {
	check, _ := s.CheckExamAccess(ctx, &pb.CheckExamAccessRequest{ExamId: req.ExamId, UserId: req.UserId})
	if !check.CanAccess {
		return nil, fmt.Errorf("không đủ điều kiện nộp bài: %s", check.Message)
	}

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
		if q.Type.Type != "" {
			qType = q.Type.Type
		}

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
		if v {
			correctCount++
		}
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
	if err != nil {
		return nil, err
	}
	return &pb.GetExamCountResponse{Count: count}, nil
}

func (s *examService) GetExams(ctx context.Context, req *pb.GetExamsRequest) (*pb.GetExamsResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	offset := (int(req.Page) - 1) * limit
	if offset < 0 {
		offset = 0
	}

	exams, total, err := s.repo.GetExams(ctx, limit, offset, req.CreatorId)
	if err != nil {
		return nil, err
	}

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
		diff, _ := s.repo.GetDifficulty(ctx, req.Difficulty)
		qType, _ := s.repo.GetQuestionType(ctx, req.QuestionType)

		updates := map[string]interface{}{
			"content": req.Content, "explanation": req.Explanation,
			"difficulty_id": diff.Id, "type_id": qType.Id, "attachment_url": req.AttachmentUrl,
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
				choices = append(choices, &domain.ChoiceModel{QuestionID: req.QuestionId, Content: c.Content, IsCorrect: c.IsCorrect, AttachmentURL: c.AttachmentUrl})
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
	if err != nil {
		return nil, err
	}
	return &pb.DeleteQuestionResponse{Success: true}, nil
}

func (s *examService) UpdateExam(ctx context.Context, req *pb.UpdateExamRequest) (*pb.UpdateExamResponse, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		updates := make(map[string]interface{})

		if req.Title != "" {
			updates["title"] = req.Title
		}
		if req.Description != "" {
			updates["description"] = req.Description
		}
		if req.Settings.DurationMinutes > 0 {
			updates["duration_minutes"] = req.Settings.DurationMinutes
		}
		if req.TopicId > 0 {
			updates["topic_id"] = req.TopicId
		}
		if req.Settings != nil {
			if req.Settings.DurationMinutes > 0 {
				updates["duration_minutes"] = req.Settings.DurationMinutes
			}
			if req.Settings.MaxAttempts > 0 {
				updates["max_attempts"] = req.Settings.MaxAttempts
			}
			if req.Settings.Password != "" {
				updates["password"] = req.Settings.Password
			}
			updates["shuffle_questions"] = req.Settings.ShuffleQuestions
			updates["show_result_immediately"] = req.Settings.ShowResultImmediately
			updates["requires_approval"] = req.Settings.RequiresApproval

			if req.Settings.StartTime != "" {
				t, err := time.Parse(time.RFC3339, req.Settings.StartTime)
				if err == nil {
					updates["start_time"] = &t
				}
			}
			if req.Settings.EndTime != "" {
				t, err := time.Parse(time.RFC3339, req.Settings.EndTime)
				if err == nil {
					updates["end_time"] = &t
				}
			}
		}

		if len(updates) > 0 {
			if err := s.repo.UpdateExam(ctx, tx, req.ExamId, updates); err != nil {
				return err
			}
		}

		if err := s.repo.ReplaceExamQuestions(ctx, tx, req.ExamId, req.QuestionIds); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateExamResponse{Success: true}, nil
}

func (s *examService) DeleteExam(ctx context.Context, req *pb.DeleteExamRequest) (*pb.DeleteExamResponse, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.DeleteExam(ctx, tx, req.ExamId)
	})
	if err != nil {
		return nil, err
	}
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

func (s *examService) SaveAnswer(ctx context.Context, req *pb.SaveAnswerRequest) (*pb.SaveAnswerResponse, error) {
	var sub domain.ExamSubmissionModel
	err := database.DB.Where("exam_id = ? AND user_id = ? AND status_id = (SELECT id FROM submission_status_models WHERE status = 'in_progress')", req.ExamId, req.UserId).First(&sub).Error
	if err != nil {
		return nil, errors.New("không tìm thấy bài làm đang diễn ra")
	}

	correctMap, _ := s.repo.GetCorrectAnswers(ctx, req.ExamId)
	correctChoices := correctMap[req.QuestionId]

	isCorrect := false
	for _, cID := range correctChoices {
		if cID == req.ChosenChoiceId {
			isCorrect = true
			break
		}
	}

	ans := &domain.UserAnswerModel{
		SubmissionID:   sub.Id,
		QuestionID:     req.QuestionId,
		ChosenChoiceID: &req.ChosenChoiceId,
		IsCorrect:      &isCorrect,
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.SaveUserAnswer(ctx, tx, ans)
	})

	return &pb.SaveAnswerResponse{Success: err == nil}, err
}

func (s *examService) LogViolation(ctx context.Context, req *pb.LogViolationRequest) (*pb.LogViolationResponse, error) {
	v := &domain.ExamViolationModel{
		ExamID:        req.ExamId,
		UserID:        req.UserId,
		ViolationType: req.ViolationType,
		ViolationTime: time.Now().UTC(),
	}
	err := s.repo.LogViolation(ctx, v)
	return &pb.LogViolationResponse{Success: err == nil}, err
}

func (s *examService) GetExamStatsDetailed(ctx context.Context, req *pb.GetExamStatsDetailedRequest) (*pb.GetExamStatsDetailedResponse, error) {
	submissions, err := s.repo.GetExamSubmissions(ctx, req.ExamId)
	if err != nil {
		return nil, err
	}

	totalParticipants, err := s.repo.CountUniqueParticipants(ctx, req.ExamId)
    if err != nil { return nil, err }

	submittedCount := int64(len(submissions))

	if totalParticipants == 0 {
        return &pb.GetExamStatsDetailedResponse{
            TotalStudents:     0,
            SubmittedCount:    0,
            ScoreDistribution: make(map[string]int32),
        }, nil
    }

	var sum, highest, lowest float64
    lowest = 10.0
    dist := make(map[string]int32)

	for i := 0; i < 10; i++ {
        key := fmt.Sprintf("%d-%d", i, i+1)
        dist[key] = 0
    }

	for _, sub := range submissions {
		score := sub.Score
		sum += score
		if score > highest { highest = score }
		if score < lowest { lowest = score }

		bucketIndex := int(score)
		if bucketIndex >= 10 {
			bucketIndex = 9
		}
		if bucketIndex < 0 {
			bucketIndex = 0
		}

		key := fmt.Sprintf("%d-%d", bucketIndex, bucketIndex+1)
		dist[key]++
	}

	averageScore := 0.0
    if submittedCount > 0 {
        averageScore = sum / float64(submittedCount)
    } else {
        lowest = 0
        highest = 0
    }

	return &pb.GetExamStatsDetailedResponse{
		TotalStudents:     totalParticipants,
		SubmittedCount:    submittedCount,
		AverageScore:      averageScore,
		HighestScore:      highest,
		LowestScore:       lowest,
		ScoreDistribution: dist,
	}, nil
}

func (s *examService) ExportExamResults(ctx context.Context, req *pb.ExportExamResultsRequest) (*pb.ExportExamResultsResponse, error) {
	submissions, err := s.repo.GetExamSubmissions(ctx, req.ExamId)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheetName := "Results"
	f.NewSheet(sheetName)
	f.SetCellValue(sheetName, "A1", "User ID")
	f.SetCellValue(sheetName, "B1", "Score")
	f.SetCellValue(sheetName, "C1", "Submitted At")
	f.SetCellValue(sheetName, "D1", "Violations Count")

	violations, _ := s.repo.GetViolationsByExam(ctx, req.ExamId)
	violationMap := make(map[int64]int)
	for _, v := range violations {
		violationMap[v.UserID]++
	}

	for i, sub := range submissions {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), sub.UserID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), sub.Score)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), sub.SubmittedAt.Format(time.RFC3339))
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), violationMap[sub.UserID])
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	fileKey := fmt.Sprintf("exports/exam_%d_%d.xlsx", req.ExamId, time.Now().Unix())
	bucketName := env.GetString("R2_BUCKET_NAME", "")

	client, err := s.createR2ClientForUpload(ctx)
	if err != nil {
		return nil, err
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(fileKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"),
	})
	if err != nil {
		return nil, err
	}

	publicDomain := env.GetString("R2_PUBLIC_DOMAIN", "")
	finalURL := fmt.Sprintf("https://%s/%s", publicDomain, fileKey)

	return &pb.ExportExamResultsResponse{FileUrl: finalURL}, nil
}

func (s *examService) createR2ClientForUpload(ctx context.Context) (*s3.Client, error) {
	accountID := env.GetString("R2_ACCOUNT_ID", "")
	accessKey := env.GetString("R2_ACCESS_KEY_ID", "")
	secretKey := env.GetString("R2_SECRET_ACCESS_KEY", "")
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	}), nil
}

func (s *examService) GetQuestions(ctx context.Context, req *pb.GetQuestionsRequest) (*pb.GetQuestionsResponse, error) {
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	limit := int(req.Limit)
	if limit < 1 {
		limit = 10
	}

	questions, total, err := s.repo.GetQuestions(
		ctx,
		req.SectionId,
		req.TopicId,
		req.Difficulty,
		req.Search,
		page,
		limit,
	)
	if err != nil {
		return nil, err
	}

	var pbQuestions []*pb.QuestionListItem
	for _, q := range questions {
		pbQuestions = append(pbQuestions, &pb.QuestionListItem{
			Id:            q.ID,
			Content:       q.Content,
			QuestionType:  q.QuestionType,
			Difficulty:    q.Difficulty,
			SectionId:     q.SectionID,
			SectionName:   q.SectionName,
			TopicId:       q.TopicID,
			TopicName:     q.TopicName,
			AttachmentUrl: q.AttachmentURL,
			ChoiceCount:   q.ChoiceCount,
		})
	}

	totalPages := int32((total + int64(limit) - 1) / int64(limit))

	return &pb.GetQuestionsResponse{
		Questions:  pbQuestions,
		Total:      total,
		Page:       int32(page),
		TotalPages: totalPages,
	}, nil
}

func (s *examService) GetQuestion(ctx context.Context, req *pb.GetQuestionRequest) (*pb.GetQuestionResponse, error) {
	// 1. Gọi Repository để lấy câu hỏi kèm các quan hệ (Preload)
	q, err := s.repo.GetQuestionByID(ctx, req.QuestionId)
	if err != nil {
		return nil, err
	}

	// 2. Map danh sách đáp án (Choices) sang Proto
	var pbChoices []*pb.ChoiceDetails
	for _, c := range q.Choices {
		pbChoices = append(pbChoices, &pb.ChoiceDetails{
			Id:            c.Id,
			Content:       c.Content,
			IsCorrect:     c.IsCorrect,
			AttachmentUrl: c.AttachmentURL,
		})
	}

	// 3. Xử lý các giá trị mặc định để tránh null/empty
	qType := "single_choice"
	if q.Type.Type != "" {
		qType = q.Type.Type
	}

	difficulty := "medium"
	if q.Difficulty.Difficulty != "" {
		difficulty = q.Difficulty.Difficulty
	}

	// 4. Trả về response đầy đủ
	return &pb.GetQuestionResponse{
		Question: &pb.QuestionDetails{
			Id:            q.Id,
			Content:       q.Content,
			QuestionType:  qType,
			Difficulty:    difficulty,
			Explanation:   q.Explanation,
			AttachmentUrl: q.AttachmentURL,
			SectionName:   q.Section.Name,
			TopicName:     q.Section.Topic.Name,
			SectionId:     q.SectionID,
			TopicId:       q.Section.TopicID,
			Choices:       pbChoices,
		},
	}, nil
}

func (s *examService) GetExamViolations(ctx context.Context, req *pb.GetExamViolationsRequest) (*pb.GetExamViolationsResponse, error) {
	violations, err := s.repo.GetViolationsByExam(ctx, req.ExamId)
	if err != nil {
		return nil, err
	}

	var pbViolations []*pb.ExamViolation
	for _, v := range violations {
		pbViolations = append(pbViolations, &pb.ExamViolation{
			Id:            v.Id,
			UserId:        v.UserID,
			ViolationType: v.ViolationType,
			ViolationTime: v.ViolationTime.Format(time.RFC3339),
		})
	}

	return &pb.GetExamViolationsResponse{Violations: pbViolations}, nil
}

func (s *examService) ExportQuestions(ctx context.Context, req *pb.ExportQuestionsRequest) (*pb.ExportQuestionsResponse, error) {
	var questions []*domain.QuestionModel
	
	db := database.DB.Model(&domain.QuestionModel{}).
		Preload("Section").Preload("Section.Topic").Preload("Choices").
		Preload("Type").Preload("Difficulty").
		Where("creator_id = ?", req.CreatorId)

	if req.TopicId > 0 { db = db.Where("topic_id = ?", req.TopicId) }
	if req.SectionId > 0 { db = db.Where("section_id = ?", req.SectionId) }
	if req.Difficulty != "" && req.Difficulty != "all" {
		db = db.Joins("JOIN question_difficulty_models ON question_models.difficulty_id = question_difficulty_models.id").
			Where("question_difficulty_models.difficulty = ?", req.Difficulty)
	}
	if req.Search != "" { db = db.Where("content ILIKE ?", "%"+req.Search+"%") }

	if err := db.Find(&questions).Error; err != nil { return nil, err }

	f := excelize.NewFile()
	sheetName := "Sheet1"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{
		"Topic Name", "Section Name", "Content", "Type", "Difficulty", 
		"Explanation", "Image URL", "Correct Answer", 
		"Option A", "Option B", "Option C", "Option D", "Option E", "Option F",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}
	style, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	f.SetRowStyle(sheetName, 1, 1, style)

	for i, q := range questions {
		row := i + 2
		
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), q.Section.Topic.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), q.Section.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), q.Content)
		
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), q.Type.Type)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), q.Difficulty.Difficulty)
		
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), q.Explanation)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), q.AttachmentURL)
		
		correctAnswers := []string{}
		
		optionStartCol := 9 
		
		for j, c := range q.Choices {
			cell, _ := excelize.CoordinatesToCellName(optionStartCol+j, row)
			f.SetCellValue(sheetName, cell, c.Content)

			if c.IsCorrect {
				char := string(rune('A' + j))
				correctAnswers = append(correctAnswers, char)
			}
		}
		
		correctStr := strings.Join(correctAnswers, ",")
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), correctStr)
	}

	buf, err := f.WriteToBuffer()
	if err != nil { return nil, err }

	fileKey := fmt.Sprintf("exports/backup_%d_%d.xlsx", req.CreatorId, time.Now().Unix())
	bucketName := env.GetString("R2_BUCKET_NAME", "")
	
	client, err := s.createR2ClientForUpload(ctx)
	if err != nil { return nil, err }

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(fileKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"),
	})
	if err != nil { return nil, err }

	publicDomain := env.GetString("R2_PUBLIC_DOMAIN", "")
	finalURL := fmt.Sprintf("https://%s/%s", publicDomain, fileKey)

	return &pb.ExportQuestionsResponse{FileUrl: finalURL}, nil
}

func (s *examService) StartExam(ctx context.Context, req *pb.StartExamRequest) (*pb.StartExamResponse, error) {
	check, _ := s.CheckExamAccess(ctx, &pb.CheckExamAccessRequest{ExamId: req.ExamId, UserId: req.UserId})

	var submission domain.ExamSubmissionModel
	err := database.DB.Where("exam_id = ? AND user_id = ? AND status_id = (SELECT id FROM submission_status_models WHERE status = 'in_progress')", req.ExamId, req.UserId).
        Preload("UserAnswers").
        First(&submission).Error

	var submissionID int64
	var startTime time.Time
    var currentAnswers = make(map[int64]*pb.Int64List)

	if err == nil {
		submissionID = submission.Id
		startTime = submission.StartedAt
        
        for _, ans := range submission.UserAnswers {
            if ans.ChosenChoiceID != nil {
                if _, exists := currentAnswers[ans.QuestionID]; !exists {
                    currentAnswers[ans.QuestionID] = &pb.Int64List{Values: []int64{}}
                }
                currentAnswers[ans.QuestionID].Values = append(currentAnswers[ans.QuestionID].Values, *ans.ChosenChoiceID)
            }
        }

	} else {
        if !check.CanAccess {
            return nil, fmt.Errorf("bạn không thể bắt đầu bài thi: %s", check.Message)
        }

		_, err := s.repo.GetExamDetails(ctx, req.ExamId)
		if err != nil { return nil, err }

		var inProgressStatus domain.SubmissionStatusModel
		database.DB.Where("status = ?", "in_progress").First(&inProgressStatus)

		newSub := &domain.ExamSubmissionModel{
			ExamID:    req.ExamId,
			UserID:    req.UserId,
			StatusID:  inProgressStatus.Id,
			StartedAt: time.Now().UTC(),
			Score:     0,
		}
		created, err := s.repo.CreateSubmission(ctx, database.DB, newSub)
		if err != nil { return nil, err }
		
		submissionID = created.Id
		startTime = created.StartedAt
	}

	examDetails, _ := s.repo.GetExamDetails(ctx, req.ExamId)
	durationSeconds := examDetails.DurationMinutes * 60
	
	elapsed := time.Since(startTime).Seconds()
	remaining := int32(float64(durationSeconds) - elapsed)

	if remaining < 0 { remaining = 0 }

	return &pb.StartExamResponse{
		SubmissionId:     submissionID,
		RemainingSeconds: remaining,
        CurrentAnswers:   currentAnswers,
	}, nil
}