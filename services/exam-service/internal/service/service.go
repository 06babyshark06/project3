package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	database "github.com/06babyshark06/JQKStudy/services/exam-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	"github.com/06babyshark06/JQKStudy/shared/env"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
			Points: 1.0,
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
		if len(choices) > 0 {
			return s.repo.CreateChoices(ctx, tx, choices)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateQuestionResponse{Id: qID, Content: req.Content}, nil
}

func (s *examService) CreateBulkQuestions(ctx context.Context, req *pb.CreateBulkQuestionsRequest) (*pb.CreateBulkQuestionsResponse, error) {
	if len(req.Questions) == 0 {
		return &pb.CreateBulkQuestionsResponse{SuccessCount: 0, Message: "Không có câu hỏi nào để lưu"}, nil
	}

	var successCount int32
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, qReq := range req.Questions {
			section, err := s.repo.GetSectionByID(ctx, qReq.SectionId)
			if err != nil {
				return fmt.Errorf("section not found for question: %s", qReq.Content)
			}
			diff, _ := s.repo.GetDifficulty(ctx, qReq.Difficulty)
			qType, _ := s.repo.GetQuestionType(ctx, qReq.QuestionType)

			question := &domain.QuestionModel{
				SectionID: qReq.SectionId, TopicID: section.TopicID, CreatorID: qReq.CreatorId,
				Content: qReq.Content, TypeID: qType.Id, DifficultyID: diff.Id,
				Explanation: qReq.Explanation, AttachmentURL: qReq.AttachmentUrl,
				Points: 1.0,
			}
			createdQ, err := s.repo.CreateQuestion(ctx, tx, question)
			if err != nil {
				return err
			}

			var choices []*domain.ChoiceModel
			for _, c := range qReq.Choices {
				choices = append(choices, &domain.ChoiceModel{
					QuestionID: createdQ.Id, Content: c.Content, IsCorrect: c.IsCorrect, AttachmentURL: c.AttachmentUrl,
				})
			}
			if len(choices) > 0 {
				if err := s.repo.CreateChoices(ctx, tx, choices); err != nil {
					return err
				}
			}
			successCount++
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateBulkQuestionsResponse{SuccessCount: successCount, Message: "Lưu hàng loạt câu hỏi thành công"}, nil
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

	sheetName := "Sheet1"
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("file excel không có sheet nào")
	}
	found := false
	for _, s := range sheets {
		if s == "Sheet1" {
			found = true
			break
		}
	}
	if !found {
		sheetName = sheets[0]
	}

	log.Printf("📥 Đang import từ sheet: %s", sheetName)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc dữ liệu từ sheet %s: %v", sheetName, err)
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

		newTopic := &domain.TopicModel{Name: name, Description: "Auto imported", CreatorID: req.CreatorId}
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
		} else if name == "short" || name == "short_answer" {
			name = "short_answer"
		} else if name == "essay" {
			name = "essay"
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
		if len(row) < 3 {
			errorCount++
			continue
		}

		for len(row) < 8 {
			row = append(row, "")
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
				Points: 1.0,
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

			isObjective := row[3] == "single_choice" || row[3] == "multiple_choice" || row[3] == "multiple"
			if isObjective && len(choices) < 2 {
				return errors.New("cần ít nhất 2 lựa chọn cho câu hỏi trắc nghiệm")
			}
			if len(choices) > 0 {
				return s.repo.CreateChoices(ctx, tx, choices)
			}
			return nil
		})

		if err == nil {
			successCount++
		} else {
			log.Printf("❌ Lỗi import dòng %d (%s): %v", i+1, content, err)
			errorCount++
		}
	}

	return &pb.ImportQuestionsResponse{SuccessCount: int32(successCount), ErrorCount: int32(errorCount)}, nil
}

func (s *examService) GenerateExam(ctx context.Context, req *pb.GenerateExamRequest) (*pb.CreateExamResponse, error) {
	var examID int64

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		isDynamic := false
		dynamicConfig := "{}"
		var examQuestions []*domain.ExamQuestionModel

		if req.Settings != nil && req.Settings.IsDynamic {
			isDynamic = true
			if len(req.SectionConfigs) > 0 {
				b, _ := json.Marshal(req.SectionConfigs)
				dynamicConfig = string(b)
			}

			for i, q := range req.FixedQuestions {
				examQuestions = append(examQuestions, &domain.ExamQuestionModel{
					QuestionID: q.QuestionId,
					Points:     float64(q.Points),
					Sequence:   i + 1,
				})
			}
		} else {

			allQuestionIDs := []int64{}
			uniqueMap := make(map[int64]bool)

			for _, cfg := range req.SectionConfigs {
				ids, err := s.repo.GetRandomQuestionsBySection(ctx, cfg.SectionId, cfg.Difficulty, int(cfg.Count), req.TopicId)
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

			if len(allQuestionIDs) == 0 && len(req.FixedQuestions) == 0 {
				return errors.New("không tìm thấy câu hỏi nào phù hợp với cấu hình")
			}

			for i, qID := range allQuestionIDs {
				examQuestions = append(examQuestions, &domain.ExamQuestionModel{
					QuestionID: qID,
					Sequence:   i + 1,
					Points:     1.0,
				})
			}

			offset := len(allQuestionIDs)
			for i, q := range req.FixedQuestions {
				examQuestions = append(examQuestions, &domain.ExamQuestionModel{
					QuestionID: q.QuestionId,
					Points:     float64(q.Points),
					Sequence:   offset + i + 1,
				})
			}
		}

		exam := &domain.ExamModel{
			Title: req.Title, Description: req.Description,
			DurationMinutes: int(req.Settings.DurationMinutes), MaxAttempts: int(req.Settings.MaxAttempts),
			Password: req.Settings.Password, ShuffleQuestions: req.Settings.ShuffleQuestions,
			ShowResultImmediately: req.Settings.ShowResultImmediately, RequiresApproval: req.Settings.RequiresApproval,
			TopicID: req.TopicId, CreatorID: req.CreatorId,
			IsDynamic: isDynamic, DynamicConfig: dynamicConfig,
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

		if len(examQuestions) > 0 {
			return s.repo.LinkQuestionsToExam(ctx, tx, created.Id, examQuestions)
		}
		return nil
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
		ExamID: req.ExamId, UserID: req.UserId, StudentName: req.StudentName, Status: "pending", CreatedAt: time.Now().UTC(),
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
	log.Printf("RECEIVING CreateExam request: Title=%s, IsDynamic=%v", req.Title, req.Settings.IsDynamic)
	var createdExam *domain.ExamModel
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		exam := &domain.ExamModel{
			Title: req.Title, Description: req.Description,
			DurationMinutes: int(req.Settings.DurationMinutes), MaxAttempts: int(req.Settings.MaxAttempts),
			Password: req.Settings.Password, ShuffleQuestions: req.Settings.ShuffleQuestions,
			ShowResultImmediately: req.Settings.ShowResultImmediately, RequiresApproval: req.Settings.RequiresApproval,
			IsDynamic: req.Settings.IsDynamic,
			TopicID: req.TopicId, CreatorID: req.CreatorId, Status: req.Status,
		}
		if req.Settings.DynamicConfig == "" {
			exam.DynamicConfig = "{}"
		} else {
			exam.DynamicConfig = req.Settings.DynamicConfig
		}

		if req.Settings.IsDynamic {
			var configs []struct {
				SectionId  int64   `json:"section_id"`
				Count      int     `json:"count"`
				Difficulty string  `json:"difficulty"`
				Points     float64 `json:"points"`
			}
			if req.Settings.DynamicConfig != "" && req.Settings.DynamicConfig != "{}" {
				if err := json.Unmarshal([]byte(req.Settings.DynamicConfig), &configs); err != nil {
					return fmt.Errorf("cấu hình sinh đề không hợp lệ: %v", err)
				}
				for i, cfg := range configs {
					ids, err := s.repo.GetQuestionIDsForSection(ctx, cfg.SectionId, cfg.Difficulty, req.TopicId)
					if err != nil {
						return fmt.Errorf("lỗi khi kiểm tra ngân hàng câu hỏi: %v", err)
					}
					if len(ids) < cfg.Count {
						sectionName := "Tất cả chương"
						if cfg.SectionId > 0 {
							sec, errSec := s.repo.GetSectionByID(ctx, cfg.SectionId)
							if errSec == nil && sec != nil {
								sectionName = fmt.Sprintf("chương '%s'", sec.Name)
							}
						}
						diffLabel := cfg.Difficulty
						if diffLabel == "all" || diffLabel == "" {
							diffLabel = "tất cả độ khó"
						}
						return fmt.Errorf("quy tắc %d: Không đủ câu hỏi trong ngân hàng. Yêu cầu %d câu (%s, %s), nhưng chỉ có %d câu.", i+1, cfg.Count, sectionName, diffLabel, len(ids))
					}
				}
			}
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
		if !createdExam.IsDynamic {
			var examQuestions []*domain.ExamQuestionModel
			for i, q := range req.Questions {
				examQuestions = append(examQuestions, &domain.ExamQuestionModel{
					QuestionID: q.QuestionId,
					Points:     float64(q.Points),
					Sequence:   i,
				})
			}
			return s.repo.LinkQuestionsToExam(ctx, tx, createdExam.Id, examQuestions)
		}
		return nil
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
		pbQuestions = append(pbQuestions, s.convertToPBQuestion(q))
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
		Id:          examModel.Id,
		Title:       examModel.Title,
		Description: examModel.Description,
		Settings: &pb.ExamSettings{
			DurationMinutes:       int32(examModel.DurationMinutes),
			MaxAttempts:           int32(examModel.MaxAttempts),
			Password:              examModel.Password,
			StartTime:             startTime,
			EndTime:               endTime,
			ShuffleQuestions:      examModel.ShuffleQuestions,
			ShowResultImmediately: examModel.ShowResultImmediately,
			RequiresApproval:      examModel.RequiresApproval,
			IsDynamic:             examModel.IsDynamic,
			DynamicConfig:         examModel.DynamicConfig,
		},
		Questions: pbQuestions,
		TopicId:   examModel.TopicID,
		Status:    examModel.Status,
	}, nil
}

func (s *examService) GetExamPreview(ctx context.Context, req *pb.GetExamPreviewRequest) (*pb.GetExamDetailsResponse, error) {
	examModel, err := s.repo.GetExamDetails(ctx, req.ExamId)
	if err != nil {
		return nil, err
	}

	pbQuestions := []*pb.QuestionDetails{}

	if examModel.IsDynamic {
		uniqueMap := make(map[int64]bool)

		for _, q := range examModel.Questions {
			if !uniqueMap[q.Id] {
				uniqueMap[q.Id] = true
				pbQuestions = append(pbQuestions, s.convertToPBQuestion(q))
			}
		}

		var configs []struct {
			SectionId  int64  `json:"section_id"`
			Count      int    `json:"count"`
			Difficulty string `json:"difficulty"`
		}
		if examModel.DynamicConfig != "" {
			if err := json.Unmarshal([]byte(examModel.DynamicConfig), &configs); err != nil {
				log.Printf("Lỗi parse DynamicConfig: %v", err)
			}
		}

		for _, cfg := range configs {
			questionIDs, err := s.repo.GetRandomQuestionsBySection(ctx, cfg.SectionId, cfg.Difficulty, cfg.Count, examModel.TopicID)
			if err != nil {
				continue
			}
			for _, qID := range questionIDs {
				if uniqueMap[qID] {
					continue
				}
				q, err := s.repo.GetQuestionByID(ctx, qID)
				if err == nil {
					uniqueMap[qID] = true
					pbQuestions = append(pbQuestions, s.convertToPBQuestion(q))
				}
			}
		}
	} else {

		for _, q := range examModel.Questions {
			pbQuestions = append(pbQuestions, s.convertToPBQuestion(q))
		}
	}

	return &pb.GetExamDetailsResponse{
		Id:    examModel.Id,
		Title: examModel.Title,
		Settings: &pb.ExamSettings{
			DurationMinutes:       int32(examModel.DurationMinutes),
			IsDynamic:             examModel.IsDynamic,
			DynamicConfig:         examModel.DynamicConfig,
		},
		Questions:   pbQuestions,
		TopicId:     examModel.TopicID,
		Status:      examModel.Status,
		Description: examModel.Description,
	}, nil
}

func (s *examService) convertToPBQuestion(q *domain.QuestionModel) *pb.QuestionDetails {
	pbChoices := []*pb.ChoiceDetails{}
	for _, c := range q.Choices {
		pbChoices = append(pbChoices, &pb.ChoiceDetails{
			Id:            c.Id,
			Content:       c.Content,
			AttachmentUrl: c.AttachmentURL,
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
	if q.Section != nil {
		sectionName = q.Section.Name
		sectionID = q.Section.Id
	}

	return &pb.QuestionDetails{
		Id:            q.Id,
		Content:       q.Content,
		Choices:       pbChoices,
		QuestionType:  qType,
		AttachmentUrl: q.AttachmentURL,
		Difficulty:    difficulty,
		Explanation:   q.Explanation,
		SectionName:   sectionName,
		SectionId:     sectionID,
		Points:        float32(q.Points),
	}
}

func (s *examService) SubmitExam(ctx context.Context, req *pb.SubmitExamRequest) (*pb.SubmitExamResponse, error) {
	if err := s.validateSessionLock(ctx, req.ExamId, req.UserId, req.IpAddress, req.UserAgent); err != nil {
		return nil, err
	}

	var submission domain.ExamSubmissionModel
	err := database.DB.Where("exam_id = ? AND user_id = ? AND status_id = (SELECT id FROM submission_status_models WHERE status = 'in_progress')", req.ExamId, req.UserId).
		First(&submission).Error

	if err != nil {
		return nil, errors.New("không tìm thấy bài làm đang diễn ra (hoặc đã nộp rồi)")
	}

	var correctCount int32 = 0
	var finalScore float64 = 0

	examModel, _ := s.repo.GetExamDetails(ctx, req.ExamId)
	var questions []*domain.QuestionModel
	qPointsMap := make(map[int64]float64)

	if examModel != nil && examModel.IsDynamic {
		sExam, err := s.repo.GetStudentExam(ctx, req.ExamId, req.UserId)
		if err != nil {
			return nil, errors.New("lỗi lấy đề thi cá nhân hóa")
		}

		dynamicQs := parseStudentExamQuestions(sExam.QuestionIDs)
		for _, dq := range dynamicQs {
			q, err := s.repo.GetQuestionByID(ctx, dq.ID)
			if err == nil {
				qPointsMap[dq.ID] = dq.Points
				questions = append(questions, q)
			}
		}
	} else if examModel != nil {
		questions = examModel.Questions
		for _, eq := range examModel.Questions {
			qPointsMap[eq.Id] = eq.Points
		}
	}

	totalQuestions := int32(len(questions))

	userAnswerMap := make(map[int64][]int64)
	userTextAnswerMap := make(map[int64]string)
	for _, ans := range req.Answers {
		if ans.ChosenChoiceId != 0 {
			userAnswerMap[ans.QuestionId] = append(userAnswerMap[ans.QuestionId], ans.ChosenChoiceId)
		}
		if ans.TextAnswer != "" {
			userTextAnswerMap[ans.QuestionId] = ans.TextAnswer
		}
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {

		var userAnswerModels []*domain.UserAnswerModel
		var totalMaxPoints float64 = 0
		var totalEarnedPoints float64 = 0

		for _, q := range questions {
			userChoices := userAnswerMap[q.Id]
			textAnswer := userTextAnswerMap[q.Id]

			qPoints := 1.0
			if pts, ok := qPointsMap[q.Id]; ok {
				qPoints = pts
			}

			totalMaxPoints += qPoints

			isCorrect := false

			qType := ""
			if q.Type.Type != "" {
				qType = q.Type.Type
			}

			if qType == "short_answer" {

				for _, c := range q.Choices {
					if c.IsCorrect && strings.EqualFold(strings.TrimSpace(c.Content), strings.TrimSpace(textAnswer)) {
						isCorrect = true
						break
					}
				}

				if !isCorrect {
					for _, cID := range userChoices {
						for _, c := range q.Choices {
							if c.Id == cID && c.IsCorrect {
								isCorrect = true
								break
							}
						}
						if isCorrect { break }
					}
				}

				if textAnswer != "" {
					correctVal := isCorrect
					val := textAnswer
					userAnswerModels = append(userAnswerModels, &domain.UserAnswerModel{
						SubmissionID:   submission.Id,
						QuestionID:     q.Id,
						TextAnswer:     &val,
						IsCorrect:      &correctVal,
					})
				}

				for _, cID := range userChoices {
					choiceIDVal := cID
					correctVal := isCorrect
					userAnswerModels = append(userAnswerModels, &domain.UserAnswerModel{
						SubmissionID:   submission.Id,
						QuestionID:     q.Id,
						ChosenChoiceID: &choiceIDVal,
						IsCorrect:      &correctVal,
					})
				}
			} else if qType == "essay" {
				if textAnswer != "" {
					val := textAnswer
					userAnswerModels = append(userAnswerModels, &domain.UserAnswerModel{
						SubmissionID:   submission.Id,
						QuestionID:     q.Id,
						TextAnswer:     &val,
					})
				}
			} else {
				var correctChoices []int64
				for _, c := range q.Choices {
					if c.IsCorrect {
						correctChoices = append(correctChoices, c.Id)
					}
				}
				isCorrect = compareInt64Slices(userChoices, correctChoices)

				for _, cID := range userChoices {
					choiceIDVal := cID
					correctVal := isCorrect

					userAnswerModels = append(userAnswerModels, &domain.UserAnswerModel{
						SubmissionID:   submission.Id,
						QuestionID:     q.Id,
						ChosenChoiceID: &choiceIDVal,
						IsCorrect:      &correctVal,
					})
				}
			}

			if isCorrect {
				correctCount++
				totalEarnedPoints += qPoints
			}
		}

		tx.Where("submission_id = ?", submission.Id).Delete(&domain.UserAnswerModel{})

		if len(userAnswerModels) > 0 {
			if err := s.repo.CreateUserAnswers(ctx, tx, userAnswerModels); err != nil {
				return err
			}
		}

		if totalMaxPoints > 0 {
			finalScore = (totalEarnedPoints / totalMaxPoints) * 10.0
		} else if totalQuestions > 0 {
			finalScore = (float64(correctCount) / float64(totalQuestions)) * 10.0
		}

		var completedStatus domain.SubmissionStatusModel
		if err := tx.WithContext(ctx).Where("status = ?", "completed").First(&completedStatus).Error; err != nil {
			return errors.New("không tìm thấy status 'completed' trong DB")
		}

		submission.StatusID = completedStatus.Id
		submission.Score = finalScore
		now := time.Now().UTC()
		submission.SubmittedAt = &now

		if _, err := s.repo.UpdateSubmission(ctx, tx, &submission); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	var examTitle string
	if examModel != nil {
		examTitle = examModel.Title
	}

	eventPayload := contracts.ExamSubmittedEvent{
		UserID:       req.UserId,
		ExamID:       req.ExamId,
		SubmissionID: submission.Id,
		ExamTitle:    examTitle,
		Score:        finalScore,
		FullName:     req.FullName,
		Email:        req.Email,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	key := []byte(strconv.FormatInt(submission.Id, 10))
	s.producer.Produce("exam_events", key, eventBytes)

	respScore := float32(finalScore)
	respCorrectCount := correctCount
	respTotalQuestions := totalQuestions

	return &pb.SubmitExamResponse{
		SubmissionId:   submission.Id,
		Score:          respScore,
		CorrectCount:   respCorrectCount,
		TotalQuestions: respTotalQuestions,
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

	isInstructor := false
	if submission.UserID != req.UserId {

		exam, err := s.repo.GetExamDetails(ctx, submission.ExamID)
		if err != nil {
			return nil, errors.New("không thể kiểm tra quyền truy cập")
		}
		if exam.CreatorID == req.UserId {
			isInstructor = true
		} else {
			return nil, errors.New("bạn không có quyền xem kết quả này")
		}
	}

	examFull, err := s.repo.GetExamDetails(ctx, submission.ExamID)
	if err != nil {
		return nil, errors.New("không thể tải nội dung đề thi gốc")
	}
	submittedAt := ""
	if submission.SubmittedAt != nil {
		submittedAt = submission.SubmittedAt.Format(time.RFC3339)
	}

	userSelections := make(map[int64]map[int64]bool)
	userTextAnswers := make(map[int64]string)
	uaMap := make(map[int64]*domain.UserAnswerModel)

	for _, ua := range submission.UserAnswers {
		if _, exists := userSelections[ua.QuestionID]; !exists {
			userSelections[ua.QuestionID] = make(map[int64]bool)
		}
		if ua.ChosenChoiceID != nil {
			userSelections[ua.QuestionID][*ua.ChosenChoiceID] = true
		}

		if ua.TextAnswer != nil {
			userTextAnswers[ua.QuestionID] = *ua.TextAnswer
		}

		if _, exists := uaMap[ua.QuestionID]; !exists {
			uaMap[ua.QuestionID] = &ua
		}
	}

	var questions []*domain.QuestionModel
	qPointsMap := make(map[int64]float64)

	if examFull.IsDynamic {
		sExam, err := s.repo.GetStudentExam(ctx, submission.ExamID, submission.UserID)
		if err == nil {
			dynamicQs := parseStudentExamQuestions(sExam.QuestionIDs)
			for _, dq := range dynamicQs {
				q, err := s.repo.GetQuestionByID(ctx, dq.ID)
				if err == nil {
					questions = append(questions, q)
					qPointsMap[dq.ID] = dq.Points
				}
			}
		}
	} else {
		questions = examFull.Questions
		for _, q := range questions {
			qPointsMap[q.Id] = q.Points
		}
	}

	var pbDetails []*pb.SubmissionDetail

	for _, q := range questions {
		var pbChoices []*pb.ChoiceReview

		for _, c := range q.Choices {
			pbChoices = append(pbChoices, &pb.ChoiceReview{
				Id:            c.Id,
				Content:       c.Content,
				IsCorrect:     c.IsCorrect,
				UserSelected:  userSelections[q.Id][c.Id],
				AttachmentUrl: c.AttachmentURL,
			})
		}

		qType := "single_choice"
		if q.Type.Type != "" {
			qType = q.Type.Type
		}

		qPoints := qPointsMap[q.Id]
		if qPoints <= 0 {
			qPoints = 1.0
		}

		var awardedPoints float32 = 0
		isCorrect := false

		if ua, ok := uaMap[q.Id]; ok {
			if ua.AwardedPoints != nil {
				awardedPoints = float32(*ua.AwardedPoints)
			} else if ua.IsCorrect != nil && *ua.IsCorrect {
				awardedPoints = float32(qPoints)
			}

			if ua.IsCorrect != nil {
				isCorrect = *ua.IsCorrect
			}
		}

		isGraded := true
		if qType == "essay" {
			if ua, ok := uaMap[q.Id]; ok && ua.TextAnswer != nil && *ua.TextAnswer != "" {
				isGraded = ua.IsCorrect != nil
			}
		}

		pbDetails = append(pbDetails, &pb.SubmissionDetail{
			QuestionId:      q.Id,
			QuestionContent: q.Content,
			Explanation:     q.Explanation,
			QuestionType:    qType,
			IsCorrect:       isCorrect,
			Choices:         pbChoices,
			AttachmentUrl:   q.AttachmentURL,
			TextAnswer:      userTextAnswers[q.Id],
			AwardedPoints:   awardedPoints,
			Points:          float32(qPoints),
			IsGraded:        isGraded,
		})
	}

	correctCount := 0
	for _, d := range pbDetails {
		if d.IsCorrect {
			correctCount++
		}
	}

	var finalDetails []*pb.SubmissionDetail = pbDetails
	if !examFull.ShowResultImmediately && !isInstructor {
		finalDetails = nil
	}

	return &pb.GetSubmissionResponse{
		Id:             submission.Id,
		ExamTitle:      submission.Exam.Title,
		Score:          float32(submission.Score),
		CorrectCount:   int32(correctCount),
		TotalQuestions: int32(len(questions)),
		Status:         submission.Status.Status,
		SubmittedAt:    submittedAt,
		Details:        finalDetails,
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

	exams, total, err := s.repo.GetExams(ctx, limit, offset, req.CreatorId, req.Status)
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
			Status:          e.Status,
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
	if req.ExamId == 0 {
		return nil, status.Error(codes.InvalidArgument, "exam_id is required")
	}

	if err := s.repo.UpdateExamStatus(ctx, database.DB, req.ExamId, req.Status); err != nil {
		return nil, status.Error(codes.Internal, "failed to update exam status")
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

func (s *examService) DeleteBulkQuestions(ctx context.Context, req *pb.DeleteBulkQuestionsRequest) (*pb.DeleteBulkQuestionsResponse, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.DeleteQuestions(ctx, tx, req.QuestionIds)
	})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteBulkQuestionsResponse{Success: true}, nil
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
		if req.Status != "" {
			updates["status"] = req.Status
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
			updates["is_dynamic"] = req.Settings.IsDynamic
			if req.Settings.DynamicConfig != "" {
				updates["dynamic_config"] = req.Settings.DynamicConfig
			} else {
				updates["dynamic_config"] = "{}"
			}

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

		if req.Settings != nil && req.Settings.IsDynamic {
			var topicID int64
			if req.TopicId > 0 {
				topicID = req.TopicId
			} else {
				existing, err := s.repo.GetExamDetails(ctx, req.ExamId)
				if err != nil {
					return err
				}
				topicID = existing.TopicID
			}

			var configs []struct {
				SectionId  int64   `json:"section_id"`
				Count      int     `json:"count"`
				Difficulty string  `json:"difficulty"`
				Points     float64 `json:"points"`
			}
			if req.Settings.DynamicConfig != "" && req.Settings.DynamicConfig != "{}" {
				if err := json.Unmarshal([]byte(req.Settings.DynamicConfig), &configs); err != nil {
					return fmt.Errorf("cấu hình sinh đề không hợp lệ: %v", err)
				}
				for i, cfg := range configs {
					ids, err := s.repo.GetQuestionIDsForSection(ctx, cfg.SectionId, cfg.Difficulty, topicID)
					if err != nil {
						return fmt.Errorf("lỗi khi kiểm tra ngân hàng câu hỏi: %v", err)
					}
					if len(ids) < cfg.Count {
						sectionName := "Tất cả chương"
						if cfg.SectionId > 0 {
							sec, errSec := s.repo.GetSectionByID(ctx, cfg.SectionId)
							if errSec == nil && sec != nil {
								sectionName = fmt.Sprintf("chương '%s'", sec.Name)
							}
						}
						diffLabel := cfg.Difficulty
						if diffLabel == "all" || diffLabel == "" {
							diffLabel = "tất cả độ khó"
						}
						return fmt.Errorf("quy tắc %d: Không đủ câu hỏi trong ngân hàng. Yêu cầu %d câu (%s, %s), nhưng chỉ có %d câu.", i+1, cfg.Count, sectionName, diffLabel, len(ids))
					}
				}
			}
		}

		if len(updates) > 0 {
			if err := s.repo.UpdateExam(ctx, tx, req.ExamId, updates); err != nil {
				return err
			}
		}

		var examQuestions []*domain.ExamQuestionModel
		for i, q := range req.Questions {
			examQuestions = append(examQuestions, &domain.ExamQuestionModel{
				QuestionID: q.QuestionId,
				Points:     float64(q.Points),
				Sequence:   i,
			})
		}
		if err := s.repo.ReplaceExamQuestions(ctx, tx, req.ExamId, examQuestions); err != nil {
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

	if err := s.validateSessionLock(ctx, req.ExamId, req.UserId, req.IpAddress, req.UserAgent); err != nil {
		return nil, err
	}

	var sub domain.ExamSubmissionModel
	err := database.DB.Where("exam_id = ? AND user_id = ? AND status_id = (SELECT id FROM submission_status_models WHERE status = 'in_progress')", req.ExamId, req.UserId).First(&sub).Error
	if err != nil {
		return nil, errors.New("không tìm thấy bài làm đang diễn ra")
	}

	ans := &domain.UserAnswerModel{
		SubmissionID: sub.Id,
		QuestionID:   req.QuestionId,
	}

	if req.TextAnswer != "" {
		ans.TextAnswer = &req.TextAnswer

		ans.IsCorrect = nil
	} else if req.ChosenChoiceId != 0 {
		correctMap, _ := s.repo.GetCorrectAnswers(ctx, req.ExamId)
		correctChoices := correctMap[req.QuestionId]

		isCorrect := false
		for _, cID := range correctChoices {
			if cID == req.ChosenChoiceId {
				isCorrect = true
				break
			}
		}
		ans.ChosenChoiceID = &req.ChosenChoiceId
		ans.IsCorrect = &isCorrect
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

	if err == nil {
		exam, _ := s.repo.GetExamDetails(ctx, req.ExamId)
		if exam != nil {
			if database.RedisClient != nil {
				msg := map[string]interface{}{
					"type":           "VIOLATION",
					"exam_id":        req.ExamId,
					"user_id":        req.UserId,
					"violation_type": req.ViolationType,
					"message":        fmt.Sprintf("Hệ thống phát hiện hành vi: %s", req.ViolationType),
					"timestamp":      v.ViolationTime.Format(time.RFC3339),
				}
				jsonMsg, _ := json.Marshal(msg)
				channel := fmt.Sprintf("notifications:%d", exam.CreatorID)
				log.Printf("📢 Redis Publish: Sending violation to [%s]", channel)
				database.RedisClient.Publish(ctx, channel, string(jsonMsg))

				monitorChannel := fmt.Sprintf("exam_monitor:%d", req.ExamId)
				log.Printf("📢 Redis Publish: Sending violation to [%s]", monitorChannel)
				database.RedisClient.Publish(ctx, monitorChannel, string(jsonMsg))
			}
		}
	}

	return &pb.LogViolationResponse{Success: err == nil}, err
}

func (s *examService) GetExamStatsDetailed(ctx context.Context, req *pb.GetExamStatsDetailedRequest) (*pb.GetExamStatsDetailedResponse, error) {
	submissions, err := s.repo.GetExamSubmissions(ctx, req.ExamId)
	if err != nil {
		return nil, err
	}

	totalParticipants, err := s.repo.CountUniqueParticipants(ctx, req.ExamId)
	if err != nil {
		return nil, err
	}

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
		if score > highest {
			highest = score
		}
		if score < lowest {
			lowest = score
		}

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

	search := req.Search
	var creatorID int64
	if strings.HasPrefix(search, "creator:") {
		parts := strings.SplitN(search, "|", 2)
		if len(parts) > 0 {
			idStr := strings.TrimPrefix(parts[0], "creator:")
			creatorID, _ = strconv.ParseInt(idStr, 10, 64)
		}
		if len(parts) > 1 {
			search = parts[1]
		} else {
			search = ""
		}
	}

	questions, total, err := s.repo.GetQuestions(
		ctx,
		req.SectionId,
		req.TopicId,
		req.Difficulty,
		search,
		page,
		limit,
		creatorID,
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
			TopicName:     q.TopicName + "|cid:" + strconv.FormatInt(q.CreatorID, 10),
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

	q, err := s.repo.GetQuestionByID(ctx, req.QuestionId)
	if err != nil {
		return nil, err
	}

	var pbChoices []*pb.ChoiceDetails
	for _, c := range q.Choices {
		pbChoices = append(pbChoices, &pb.ChoiceDetails{
			Id:            c.Id,
			Content:       c.Content,
			IsCorrect:     c.IsCorrect,
			AttachmentUrl: c.AttachmentURL,
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

	if req.TopicId > 0 {
		db = db.Where("topic_id = ?", req.TopicId)
	}
	if req.SectionId > 0 {
		db = db.Where("section_id = ?", req.SectionId)
	}
	if req.Difficulty != "" && req.Difficulty != "all" {
		db = db.Joins("JOIN question_difficulty_models ON question_models.difficulty_id = question_difficulty_models.id").
			Where("question_difficulty_models.difficulty = ?", req.Difficulty)
	}
	if req.Search != "" {
		db = db.Where("content ILIKE ?", "%"+req.Search+"%")
	}

	if err := db.Find(&questions).Error; err != nil {
		return nil, err
	}

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
	if err != nil {
		return nil, err
	}

	fileKey := fmt.Sprintf("exports/backup_%d_%d.xlsx", req.CreatorId, time.Now().Unix())
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

	return &pb.ExportQuestionsResponse{FileUrl: finalURL}, nil
}

func generateSessionHash(ipAddress, userAgent string) string {
	raw := fmt.Sprintf("%s|%s", ipAddress, userAgent)
	hash := md5.Sum([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func (s *examService) validateSessionLock(ctx context.Context, examID, userID int64, ipAddress, userAgent string) error {
	sessionKey := fmt.Sprintf("exam:%d:user:%d:session_lock", examID, userID)
	expectedHash, err := database.RedisClient.Get(ctx, sessionKey).Result()
	if err == redis.Nil {

		return nil
	} else if err != nil {
		return fmt.Errorf("lỗi kiểm tra phiên làm bài: %v", err)
	}

	currentHash := generateSessionHash(ipAddress, userAgent)
	if currentHash != expectedHash {
		return fmt.Errorf("tài khoản đang mở bài thi trên một thiết bị hoặc trình duyệt khác (IP/Browser không khớp). Phát hiện gian lận thi hộ!")
	}
	return nil
}

func (s *examService) StartExam(ctx context.Context, req *pb.StartExamRequest) (*pb.StartExamResponse, error) {
	examDetails, err := s.repo.GetExamDetails(ctx, req.ExamId)
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy bài thi: %v", err)
	}

	check, _ := s.CheckExamAccess(ctx, &pb.CheckExamAccessRequest{ExamId: req.ExamId, UserId: req.UserId})

	if database.RedisClient != nil {
		sessionKey := fmt.Sprintf("exam:%d:user:%d:session_lock", req.ExamId, req.UserId)
		sessionHash := generateSessionHash(req.IpAddress, req.UserAgent)
		ttl := time.Duration(examDetails.DurationMinutes) * time.Minute
		if ttl <= 0 {
			ttl = 120 * time.Minute
		}

		if err := database.RedisClient.Set(ctx, sessionKey, sessionHash, ttl).Err(); err != nil {
			fmt.Printf("Warning: Failed to set session lock for exam %d user %d: %v\n", req.ExamId, req.UserId, err)
		}
	}

	var submission domain.ExamSubmissionModel
	err = database.DB.Where("exam_id = ? AND user_id = ? AND status_id = (SELECT id FROM submission_status_models WHERE status = 'in_progress')", req.ExamId, req.UserId).
		Preload("UserAnswers").
		First(&submission).Error

	var submissionID int64
	var startTime time.Time

	if err == nil {
		submissionID = submission.Id
		startTime = submission.StartedAt
	} else {
		if !check.CanAccess {
			return nil, fmt.Errorf("bạn không thể bắt đầu bài thi: %s", check.Message)
		}

		var inProgressStatus domain.SubmissionStatusModel
		if err := database.DB.Where("status = ?", "in_progress").First(&inProgressStatus).Error; err != nil {
			return nil, errors.New("lỗi hệ thống: chưa cấu hình status in_progress")
		}

		newSub := &domain.ExamSubmissionModel{
			ExamID:    req.ExamId,
			UserID:    req.UserId,
			StatusID:  inProgressStatus.Id,
			StartedAt: time.Now().UTC(),
			Score:     0,
		}
		created, err := s.repo.CreateSubmission(ctx, database.DB, newSub)
		if err != nil {
			return nil, err
		}

		submissionID = created.Id
		startTime = created.StartedAt
	}

	durationSeconds := float64(examDetails.DurationMinutes * 60)
	now := time.Now().UTC()
	elapsed := now.Sub(startTime).Seconds()
	remaining := int32(durationSeconds - elapsed)

	if examDetails.EndTime != nil {
		timeUntilClose := examDetails.EndTime.Sub(now).Seconds()
		if timeUntilClose < float64(remaining) {
			remaining = int32(timeUntilClose)
		}
	}

	if remaining < 0 {
		remaining = 0
	}

	var pbQuestions []*pb.QuestionDetails

	var qIDsToFetch []int64
	var orderMap map[int64]int
	qPointsMap := make(map[int64]float64)

	if examDetails.IsDynamic {
		sExam, err := s.repo.GetStudentExam(ctx, req.ExamId, req.UserId)
		if err != nil {

			log.Printf("Bắt đầu sinh đề tự động cho User %d tại Exam %d", req.UserId, req.ExamId)
			errGen := s.GeneratePersonalizedExamForStudents(ctx, req.ExamId, []int64{req.UserId})
			if errGen != nil {
				return nil, fmt.Errorf("không thể tự động sinh đề thi: %v", errGen)
			}

			sExam, err = s.repo.GetStudentExam(ctx, req.ExamId, req.UserId)
			if err != nil {
				return nil, fmt.Errorf("đề thi đang được sinh, vui lòng quay lại sau giây lát")
			}
		}
		dynamicQs := parseStudentExamQuestions(sExam.QuestionIDs)
		orderMap = make(map[int64]int)
		for i, dq := range dynamicQs {
			qIDsToFetch = append(qIDsToFetch, dq.ID)
			orderMap[dq.ID] = i
			qPointsMap[dq.ID] = dq.Points
		}
	} else {
		for _, q := range examDetails.Questions {
			qIDsToFetch = append(qIDsToFetch, q.Id)
		}
		orderMap = make(map[int64]int)
		for i, id := range qIDsToFetch {
			orderMap[id] = i
		}
	}

	if len(qIDsToFetch) == 0 {
		return nil, errors.New("đề thi chưa có câu hỏi nào (vui lòng liên hệ giáo viên)")
	}

	if len(qIDsToFetch) > 0 {
		pbQuestions = make([]*pb.QuestionDetails, len(qIDsToFetch))
		var missingIDs []int64

		if database.RedisClient != nil {
			keys := make([]string, len(qIDsToFetch))
			for i, id := range qIDsToFetch {
				keys[i] = fmt.Sprintf("question:%d", id)
			}

			cachedVals, err := database.RedisClient.MGet(ctx, keys...).Result()
			if err == nil {
				for i, val := range cachedVals {
					if val != nil {
						var pbQ pb.QuestionDetails
						if err := json.Unmarshal([]byte(val.(string)), &pbQ); err == nil {

							if pts, ok := qPointsMap[pbQ.Id]; ok {
								pbQ.Points = float32(pts)
							} else if !examDetails.IsDynamic {
								for _, eq := range examDetails.Questions {
									if eq.Id == pbQ.Id {
										pbQ.Points = float32(eq.Points)
										break
									}
								}
							}
							pbQuestions[i] = &pbQ
						} else {
							missingIDs = append(missingIDs, qIDsToFetch[i])
						}
					} else {
						missingIDs = append(missingIDs, qIDsToFetch[i])
					}
				}
			} else {
				log.Printf("Lỗi lấy Cache Redis: %v", err)
				missingIDs = qIDsToFetch
			}
		} else {
			missingIDs = qIDsToFetch
		}

		if len(missingIDs) > 0 {
			var missingQuestions []*domain.QuestionModel
			database.DB.WithContext(ctx).Where("id IN ?", missingIDs).Preload("Choices").Preload("Type").Preload("Difficulty").Preload("Section").Preload("Section.Topic").Find(&missingQuestions)

			var pipe redis.Pipeliner
			if database.RedisClient != nil {
				pipe = database.RedisClient.Pipeline()
			}

			for _, q := range missingQuestions {
				var pbChoices []*pb.ChoiceDetails
				for _, c := range q.Choices {
					pbChoices = append(pbChoices, &pb.ChoiceDetails{
						Id:            c.Id,
						Content:       c.Content,
						AttachmentUrl: c.AttachmentURL,
					})
				}
				qType, diff := "single_choice", "medium"
				if q.Type.Type != "" {
					qType = q.Type.Type
				}
				if q.Difficulty.Difficulty != "" {
					diff = q.Difficulty.Difficulty
				}

				secName, topicName := "", ""
				var secID, topicID int64
				if q.Section != nil {
					secName = q.Section.Name
					secID = q.SectionID
					if q.Section.Topic != nil {
						topicName = q.Section.Topic.Name
						topicID = q.Section.TopicID
					}
				}

				qPoints := float64(1.0)
				if !examDetails.IsDynamic {
					for _, eq := range examDetails.Questions {
						if eq.Id == q.Id {
							qPoints = eq.Points
							break
						}
					}
				} else {
					if pts, ok := qPointsMap[q.Id]; ok {
						qPoints = pts
					}
				}

				pbQ := &pb.QuestionDetails{
					Id:            q.Id,
					Content:       q.Content,
					Choices:       pbChoices,
					QuestionType:  qType,
					AttachmentUrl: q.AttachmentURL,
					Difficulty:    diff,
					Explanation:   q.Explanation,
					SectionName:   secName,
					TopicName:     topicName,
					SectionId:     secID,
					TopicId:       topicID,
					Points:        float32(qPoints),
				}

				idx := orderMap[q.Id]
				pbQuestions[idx] = pbQ

				if pipe != nil {
					b, _ := json.Marshal(pbQ)
					pipe.Set(ctx, fmt.Sprintf("question:%d", q.Id), b, 24*time.Hour)
				}
			}

			if pipe != nil {
				_, _ = pipe.Exec(ctx)
			}
		}

		var finalPbQs []*pb.QuestionDetails
		for _, q := range pbQuestions {
			if q != nil {

				if examDetails.ShuffleQuestions && len(q.Choices) > 0 {
					rand.Shuffle(len(q.Choices), func(i, j int) {
						q.Choices[i], q.Choices[j] = q.Choices[j], q.Choices[i]
					})
				}

				finalPbQs = append(finalPbQs, q)
			}
		}
		pbQuestions = finalPbQs
	}

	var pbCurrentAnswers []*pb.AnswerDetail
	userAnswerMap := make(map[int64]*pb.AnswerDetail)

	for _, ua := range submission.UserAnswers {
		if _, exists := userAnswerMap[ua.QuestionID]; !exists {
			userAnswerMap[ua.QuestionID] = &pb.AnswerDetail{
				QuestionId: ua.QuestionID,
			}
		}
		if ua.ChosenChoiceID != nil {
			userAnswerMap[ua.QuestionID].ChoiceIds = append(userAnswerMap[ua.QuestionID].ChoiceIds, *ua.ChosenChoiceID)
		}
		if ua.TextAnswer != nil {
			userAnswerMap[ua.QuestionID].TextAnswer = *ua.TextAnswer
		}
	}
	for _, ad := range userAnswerMap {
		pbCurrentAnswers = append(pbCurrentAnswers, ad)
	}

	return &pb.StartExamResponse{
		SubmissionId:     submissionID,
		RemainingSeconds: remaining,
		Questions:        pbQuestions,
		CurrentAnswers:   pbCurrentAnswers,
	}, nil
}

func (s *examService) GetAccessRequests(ctx context.Context, req *pb.GetAccessRequestsRequest) (*pb.GetAccessRequestsResponse, error) {
	exam, err := s.repo.GetExamDetails(ctx, req.ExamId)
	if err != nil {
		return nil, err
	}
	if exam.CreatorID != req.CreatorId {
		return nil, errors.New("không có quyền xem yêu cầu của bài thi này")
	}

	requests, err := s.repo.GetAccessRequestsByExam(ctx, req.ExamId)
	if err != nil {
		return nil, err
	}

	var pbRequests []*pb.AccessRequestItem
	for _, r := range requests {
		pbRequests = append(pbRequests, &pb.AccessRequestItem{
			Id:        r.Id,
			UserId:    r.UserID,
			Status:    r.Status,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
			FullName:  r.StudentName,
		})
	}

	return &pb.GetAccessRequestsResponse{Requests: pbRequests}, nil
}

func (s *examService) UpdateTopic(ctx context.Context, req *pb.UpdateTopicRequest) (*pb.UpdateTopicResponse, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	err := s.repo.UpdateTopic(ctx, req.Id, updates)
	return &pb.UpdateTopicResponse{Success: err == nil}, err
}

func (s *examService) DeleteTopic(ctx context.Context, req *pb.DeleteTopicRequest) (*pb.DeleteTopicResponse, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.DeleteTopic(ctx, tx, req.Id)
	})
	return &pb.DeleteTopicResponse{Success: err == nil}, err
}

func (s *examService) UpdateSection(ctx context.Context, req *pb.UpdateSectionRequest) (*pb.UpdateSectionResponse, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	err := s.repo.UpdateSection(ctx, req.Id, updates)
	return &pb.UpdateSectionResponse{Success: err == nil}, err
}

func (s *examService) DeleteSection(ctx context.Context, req *pb.DeleteSectionRequest) (*pb.DeleteSectionResponse, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.repo.DeleteSection(ctx, tx, req.Id)
	})
	return &pb.DeleteSectionResponse{Success: err == nil}, err
}

func (s *examService) AssignExamToClass(ctx context.Context, req *pb.AssignExamToClassRequest) (*pb.AssignExamToClassResponse, error) {
	err := s.repo.AssignExamToClass(ctx, req.ExamId, req.ClassId)
	if err != nil {
		return nil, err
	}

	exam, _ := s.repo.GetExamDetails(ctx, req.ExamId)
	if exam != nil && exam.IsDynamic {
		event := contracts.ExamAssignedEvent{ExamID: req.ExamId, ClassID: req.ClassId}
		eventBytes, _ := json.Marshal(event)
		s.producer.Produce("exam_assigned", []byte(fmt.Sprintf("%d-%d", req.ExamId, req.ClassId)), eventBytes)
	}

	return &pb.AssignExamToClassResponse{Success: true}, nil
}

func (s *examService) GetExamsByClass(ctx context.Context, req *pb.GetExamsByClassRequest) (*pb.GetExamsByClassResponse, error) {
	exams, err := s.repo.GetExamsByClass(ctx, req.ClassId)
	if err != nil {
		return nil, err
	}

	var pbExams []*pb.Exam
	for _, e := range exams {
		pbE := mapDomainExamToProto(e)
		if req.StudentId > 0 {
			count, _ := s.repo.CountSubmissionsForExam(ctx, e.Id, req.StudentId)
			pbE.AttemptsUsed = int32(count)
		}
		pbExams = append(pbExams, pbE)
	}

	return &pb.GetExamsByClassResponse{Exams: pbExams}, nil
}

func (s *examService) UnassignExamFromClass(ctx context.Context, req *pb.AssignExamToClassRequest) (*pb.AssignExamToClassResponse, error) {
	err := s.repo.UnassignExamFromClass(ctx, req.ExamId, req.ClassId)
	if err != nil {
		return nil, err
	}
	return &pb.AssignExamToClassResponse{Success: true}, nil
}

func (s *examService) GetInstructorExams(ctx context.Context, req *pb.GetInstructorExamsRequest) (*pb.GetInstructorExamsResponse, error) {
	exams, err := s.repo.GetExamsByTeacher(ctx, req.TeacherId)
	if err != nil {
		return nil, err
	}

	var pbExams []*pb.Exam
	for _, e := range exams {
		pbExams = append(pbExams, mapDomainExamToProto(e))
	}

	return &pb.GetInstructorExamsResponse{Exams: pbExams}, nil
}

func (s *examService) GetExamSubmissions(ctx context.Context, req *pb.GetExamSubmissionsRequest) (*pb.GetExamSubmissionsResponse, error) {

	if req.InstructorId > 0 {
		exam, err := s.repo.GetExamDetails(ctx, req.ExamId)
		if err != nil {
			return nil, err
		}
		if exam.CreatorID != req.InstructorId {
			return nil, errors.New("bạn không có quyền xem danh sách bài thi này")
		}
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	limit := int(req.Limit)
	if limit < 1 {
		limit = 10
	}

	subs, total, err := s.repo.GetExamSubmissionsByExamID(ctx, req.ExamId, page, limit, req.Search)
	if err != nil {
		return nil, err
	}

	var pbSubs []*pb.SubmissionSummary
	for _, sub := range subs {
		submittedAt := ""
		if sub.SubmittedAt != nil {
			submittedAt = sub.SubmittedAt.Format(time.RFC3339)
		}

		status := "unknown"
		if sub.Status.Status != "" {
			status = sub.Status.Status
		}

		pbSubs = append(pbSubs, &pb.SubmissionSummary{
			SubmissionId:   sub.Id,
			UserId:         sub.UserID,
			StudentName:    "",
			Score:          float32(sub.Score),
			SubmittedAt:    submittedAt,
			Status:         status,
			CorrectCount:   0,
			TotalQuestions: 0,
		})
	}

	totalPages := int32((total + int64(limit) - 1) / int64(limit))

	return &pb.GetExamSubmissionsResponse{
		Submissions: pbSubs,
		Total:       total,
		Page:        int32(page),
		TotalPages:  totalPages,
	}, nil
}

func (s *examService) GetRecentSubmissions(ctx context.Context, req *pb.GetRecentSubmissionsRequest) (*pb.GetRecentSubmissionsResponse, error) {
	limit := int(req.Limit)
	if limit < 1 {
		limit = 5
	}

	subs, err := s.repo.GetRecentSubmissions(ctx, req.InstructorId, limit)
	if err != nil {
		return nil, err
	}

	var pbItems []*pb.RecentSubmissionItem
	for _, sub := range subs {
		submittedAt := ""
		if sub.SubmittedAt != nil {
			submittedAt = sub.SubmittedAt.Format(time.RFC3339)
		}

		pbItems = append(pbItems, &pb.RecentSubmissionItem{
			SubmissionId: sub.Id,
			ExamId:       sub.ExamID,
			ExamTitle:    sub.Exam.Title,
			StudentId:    sub.UserID,
			StudentName:  "",
			Score:        float32(sub.Score),
			SubmittedAt:  submittedAt,
			Status:       sub.Status.Status,
		})
	}

	return &pb.GetRecentSubmissionsResponse{
		Submissions: pbItems,
	}, nil
}

type DynamicQuestion struct {
	ID     int64   `json:"id"`
	Points float64 `json:"points"`
}

func parseStudentExamQuestions(jsonStr string) []DynamicQuestion {
	var rawData []json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &rawData); err != nil {
		return nil
	}

	var result []DynamicQuestion
	for _, raw := range rawData {
		var id int64
		if err := json.Unmarshal(raw, &id); err == nil {
			result = append(result, DynamicQuestion{ID: id, Points: 1.0})
		} else {
			var dq DynamicQuestion
			if err := json.Unmarshal(raw, &dq); err == nil {
				result = append(result, dq)
			}
		}
	}
	return result
}

func mapDomainExamToProto(e *domain.ExamModel) *pb.Exam {
	if e == nil {
		return nil
	}

	topicID := e.TopicID

	createdAt := ""
	if !e.CreatedAt.IsZero() {
		createdAt = e.CreatedAt.Format(time.RFC3339)
	}

	startTime := ""
	if e.StartTime != nil {
		startTime = e.StartTime.Format(time.RFC3339)
	}
	endTime := ""
	if e.EndTime != nil {
		endTime = e.EndTime.Format(time.RFC3339)
	}

	return &pb.Exam{
		Id:              e.Id,
		Title:           e.Title,
		Description:     e.Description,
		DurationMinutes: int32(e.DurationMinutes),
		TopicId:         topicID,
		CreatorId:       e.CreatorID,
		Status:          e.Status,
		CreatedAt:       createdAt,
		QuestionCount:   int32(len(e.Questions)),
		StartTime:       startTime,
		EndTime:         endTime,
		MaxAttempts:     int32(e.MaxAttempts),
	}
}

func (s *examService) GeneratePersonalizedExamForStudents(ctx context.Context, examID int64, studentIDs []int64) error {
	examDetails, err := s.repo.GetExamDetails(ctx, examID)
	if err != nil {
		return err
	}
	if !examDetails.IsDynamic {
		return errors.New("đề thi không phải là dạng sinh động")
	}
	dynamicConfig := examDetails.DynamicConfig
	if dynamicConfig == "" {
		dynamicConfig = "{}"
	}

	var configs []struct {
		SectionId  int64   `json:"section_id"`
		Count      int     `json:"count"`
		Difficulty string  `json:"difficulty"`
		Points     float64 `json:"points"`
	}
	err = json.Unmarshal([]byte(dynamicConfig), &configs)
	if err != nil {
		return errors.New("cấu hình sinh đề không hợp lệ")
	}

	configPools := make(map[int][]int64)
	for i, cfg := range configs {
		ids, err := s.repo.GetQuestionIDsForSection(ctx, cfg.SectionId, cfg.Difficulty, examDetails.TopicID)
		if err != nil {
			log.Printf("Lỗi lấy danh sách câu hỏi cho section %d: %v", cfg.SectionId, err)
			ids = []int64{}
		}
		log.Printf("Rule %d: Section=%d, Diff=%s, Topic=%d => Found %d questions", i, cfg.SectionId, cfg.Difficulty, examDetails.TopicID, len(ids))
		configPools[i] = ids
	}

	var batchInserts []*domain.StudentExamModel

	for _, studentID := range studentIDs {
		allQuestions := []DynamicQuestion{}
		uniqueMap := make(map[int64]bool)

		for _, q := range examDetails.Questions {
			if !uniqueMap[q.Id] {
				uniqueMap[q.Id] = true
				allQuestions = append(allQuestions, DynamicQuestion{ID: q.Id, Points: q.Points})
			}
		}

		for i, cfg := range configs {
			pool := configPools[i]
			if len(pool) == 0 {
				continue
			}

			shuffledPool := make([]int64, len(pool))
			copy(shuffledPool, pool)

			for idx := len(shuffledPool) - 1; idx > 0; idx-- {
				j := rand.Intn(idx + 1)
				shuffledPool[idx], shuffledPool[j] = shuffledPool[j], shuffledPool[idx]
			}

			collectedCount := 0
			pts := cfg.Points
			if pts <= 0 {
				pts = 1.0
			}
			for _, id := range shuffledPool {
				if !uniqueMap[id] {
					uniqueMap[id] = true
					allQuestions = append(allQuestions, DynamicQuestion{ID: id, Points: pts})
					collectedCount++
					if collectedCount >= cfg.Count {
						break
					}
				}
			}
		}

		if len(allQuestions) > 0 {
			qBytes, _ := json.Marshal(allQuestions)
			sExam := &domain.StudentExamModel{
				ExamID:      examID,
				UserID:      studentID,
				QuestionIDs: string(qBytes),
			}
			batchInserts = append(batchInserts, sExam)
		}
	}

	if len(batchInserts) > 0 {
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			return tx.WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "exam_id"}, {Name: "user_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"question_ids", "created_at"}),
			}).Create(&batchInserts).Error
		})
		if err != nil {
			log.Printf("Lỗi Upsert StudentExams cho exam %d: %v", examID, err)
			return err
		}
	} else {
		return errors.New("không tìm thấy đủ câu hỏi phù hợp để sinh đề")
	}

	return nil
}
func (s *examService) GetMySubmissions(ctx context.Context, req *pb.GetMySubmissionsRequest) (*pb.GetMySubmissionsResponse, error) {
	subs, err := s.repo.GetSubmissionsByUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	var pbSubs []*pb.SubmissionSummary
	for _, sub := range subs {
		submittedAt := ""
		if sub.SubmittedAt != nil {
			submittedAt = sub.SubmittedAt.Format(time.RFC3339)
		}

		status := "unknown"
		if sub.Status.Status != "" {
			status = sub.Status.Status
		}

		pbSubs = append(pbSubs, &pb.SubmissionSummary{
			SubmissionId: sub.Id,
			UserId:       sub.UserID,
			Score:        float32(sub.Score),
			SubmittedAt:  submittedAt,
			Status:       status,
			ExamTitle:    sub.Exam.Title,
		})
	}

	return &pb.GetMySubmissionsResponse{
		Submissions: pbSubs,
	}, nil
}

func (s *examService) GradeEssay(ctx context.Context, req *pb.GradeEssayRequest) (*pb.GradeEssayResponse, error) {

	updates := map[string]interface{}{
		"is_correct": req.IsCorrect,
		"awarded_points": req.ScoreRatio,
	}
	err := s.repo.UpdateUserAnswer(ctx, nil, req.SubmissionId, req.QuestionId, updates)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Lỗi cập nhật câu trả lời: %v", err)
	}

	submission, err := s.repo.GetSubmissionByID(ctx, req.SubmissionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Lỗi lấy thông tin bài nộp: %v", err)
	}

	qPointsMap := make(map[int64]float64)
	if submission.Exam.Id != 0 {
		if submission.Exam.IsDynamic {
			sExam, err := s.repo.GetStudentExam(ctx, submission.ExamID, submission.UserID)
			if err == nil {
				dynamicQs := parseStudentExamQuestions(sExam.QuestionIDs)
				for _, dq := range dynamicQs {
					qPointsMap[dq.ID] = dq.Points
				}
			}
		} else {
			for _, eq := range submission.Exam.Questions {
				qPointsMap[eq.Id] = eq.Points
			}
		}
	}

	var totalMaxPoints float64 = 0
	var totalEarnedPoints float64 = 0

	for _, ans := range submission.UserAnswers {
		qPts := 1.0
		if pts, ok := qPointsMap[ans.QuestionID]; ok {
			qPts = pts
		} else if ans.Question.Points > 0 {
			qPts = ans.Question.Points
		}

		totalMaxPoints += qPts

		if ans.AwardedPoints != nil {
			totalEarnedPoints += (*ans.AwardedPoints) * qPts
		} else if ans.IsCorrect != nil && *ans.IsCorrect {
			totalEarnedPoints += qPts
		}
	}

	newScore := 0.0
	if totalMaxPoints > 0 {
		newScore = (totalEarnedPoints / totalMaxPoints) * 10.0
	} else {
		totalQuestions := len(submission.UserAnswers)
		if totalQuestions > 0 {
			newScore = (totalEarnedPoints / float64(totalQuestions)) * 10.0
		}
	}

	submission.Score = newScore
	_, err = s.repo.UpdateSubmission(ctx, nil, submission)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Lỗi cập nhật điểm bài nộp: %v", err)
	}

	return &pb.GradeEssayResponse{Success: true}, nil
}

func (s *examService) GetClassGradebook(ctx context.Context, req *pb.GetClassGradebookRequest) (*pb.GetClassGradebookResponse, error) {
	if req.ClassId == 0 {
		return nil, status.Error(codes.InvalidArgument, "Mã lớp không hợp lệ")
	}

	exams, err := s.repo.GetExamsByClass(ctx, req.ClassId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Lỗi lấy danh sách bài thi của lớp: %v", err)
	}

	examIDs := make([]int64, len(exams))
	pbExams := make([]*pb.Exam, len(exams))
	for i, e := range exams {
		examIDs[i] = e.Id
		pbExams[i] = &pb.Exam{
			Id:              e.Id,
			Title:           e.Title,
			DurationMinutes: int32(e.DurationMinutes),
			QuestionCount:   int32(len(e.Questions)),
		}
	}

	if len(exams) == 0 || len(req.StudentIds) == 0 {
		return &pb.GetClassGradebookResponse{
			Exams:  pbExams,
			Grades: []*pb.StudentGrade{},
		}, nil
	}

	scoreMap, err := s.repo.GetBestScoresForGradebook(ctx, examIDs, req.StudentIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Lỗi lấy bảng điểm: %v", err)
	}

	var grades []*pb.StudentGrade
	for _, studentID := range req.StudentIds {
		var studentScores []*pb.ExamScore
		for _, examID := range examIDs {
			score, found := scoreMap[studentID][examID]
			studentScores = append(studentScores, &pb.ExamScore{
				ExamId:    examID,
				Score:     float32(score),
				Completed: found,
			})
		}
		grades = append(grades, &pb.StudentGrade{
			StudentId: studentID,
			Scores:    studentScores,
		})
	}

	return &pb.GetClassGradebookResponse{
		Exams:  pbExams,
		Grades: grades,
	}, nil
}

