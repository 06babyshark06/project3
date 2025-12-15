package repository

import (
	"context"
	"sort"
	"time"

	database "github.com/06babyshark06/JQKStudy/services/exam-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/domain"
	"gorm.io/gorm"
)

type examRepository struct{}

func NewExamRepository() domain.ExamRepository {
	return &examRepository{}
}

func (r *examRepository) CreateTopic(ctx context.Context, tx *gorm.DB, topic *domain.TopicModel) (*domain.TopicModel, error) {
	if err := tx.WithContext(ctx).Create(topic).Error; err != nil { return nil, err }
	return topic, nil
}
func (r *examRepository) GetTopics(ctx context.Context) ([]*domain.TopicModel, error) {
	var topics []*domain.TopicModel
	if err := database.DB.WithContext(ctx).Order("created_at DESC").Find(&topics).Error; err != nil { return nil, err }
	return topics, nil
}
func (r *examRepository) GetTopicByName(ctx context.Context, name string) (*domain.TopicModel, error) {
	var topic domain.TopicModel
	if err := database.DB.WithContext(ctx).Where("name = ?", name).First(&topic).Error; err != nil { return nil, err }
	return &topic, nil
}
func (r *examRepository) CreateSection(ctx context.Context, tx *gorm.DB, section *domain.SectionModel) (*domain.SectionModel, error) {
	if err := tx.WithContext(ctx).Create(section).Error; err != nil { return nil, err }
	return section, nil
}
func (r *examRepository) GetSectionsByTopic(ctx context.Context, topicID int64) ([]*domain.SectionModel, error) {
	var sections []*domain.SectionModel
	if err := database.DB.WithContext(ctx).Where("topic_id = ?", topicID).Order("id ASC").Find(&sections).Error; err != nil { return nil, err }
	return sections, nil
}
func (r *examRepository) GetSectionByID(ctx context.Context, id int64) (*domain.SectionModel, error) {
	var section domain.SectionModel
	if err := database.DB.WithContext(ctx).First(&section, id).Error; err != nil { return nil, err }
	return &section, nil
}

func (r *examRepository) CreateQuestion(ctx context.Context, tx *gorm.DB, question *domain.QuestionModel) (*domain.QuestionModel, error) {
	if err := tx.WithContext(ctx).Create(question).Error; err != nil { return nil, err }
	return question, nil
}
func (r *examRepository) CreateChoices(ctx context.Context, tx *gorm.DB, choices []*domain.ChoiceModel) error {
	return tx.WithContext(ctx).Create(choices).Error
}
func (r *examRepository) GetQuestionType(ctx context.Context, typeName string) (*domain.QuestionTypeModel, error) {
	var m domain.QuestionTypeModel
	err := database.DB.WithContext(ctx).Where("type = ?", typeName).First(&m).Error
	return &m, err
}
func (r *examRepository) GetDifficulty(ctx context.Context, level string) (*domain.QuestionDifficultyModel, error) {
	var m domain.QuestionDifficultyModel
	err := database.DB.WithContext(ctx).Where("difficulty = ?", level).First(&m).Error
	return &m, err
}
func (r *examRepository) UpdateQuestion(ctx context.Context, tx *gorm.DB, qID int64, updates map[string]interface{}) error {
	return tx.WithContext(ctx).Model(&domain.QuestionModel{}).Where("id = ?", qID).Updates(updates).Error
}
func (r *examRepository) DeleteChoicesByQuestionID(ctx context.Context, tx *gorm.DB, qID int64) error {
	return tx.WithContext(ctx).Where("question_id = ?", qID).Delete(&domain.ChoiceModel{}).Error
}
func (r *examRepository) DeleteQuestion(ctx context.Context, tx *gorm.DB, questionID int64) error {
	tx.WithContext(ctx).Where("question_id = ?", questionID).Delete(&domain.ChoiceModel{})
	tx.WithContext(ctx).Where("question_id = ?", questionID).Delete(&domain.ExamQuestionModel{})
	return tx.WithContext(ctx).Delete(&domain.QuestionModel{}, questionID).Error
}

func (r *examRepository) GetRandomQuestionsBySection(ctx context.Context, sectionID int64, difficulty string, limit int) ([]int64, error) {
	var questionIDs []int64
	query := database.DB.WithContext(ctx).Table("question_models").Select("question_models.id")
	query = query.Where("section_id = ?", sectionID)
	if difficulty != "" {
		query = query.Joins("JOIN question_difficulty_models d ON question_models.difficulty_id = d.id").Where("d.difficulty = ?", difficulty)
	}

	err := query.Order("RANDOM()").Limit(limit).Pluck("id", &questionIDs).Error
	return questionIDs, err
}

func (r *examRepository) CreateExam(ctx context.Context, tx *gorm.DB, exam *domain.ExamModel) (*domain.ExamModel, error) {
	if err := tx.WithContext(ctx).Create(exam).Error; err != nil { return nil, err }
	return exam, nil
}
func (r *examRepository) LinkQuestionsToExam(ctx context.Context, tx *gorm.DB, examID int64, questionIDs []int64) error {
	if len(questionIDs) == 0 { return nil }
	var examQuestions []*domain.ExamQuestionModel
	for i, qid := range questionIDs {
		examQuestions = append(examQuestions, &domain.ExamQuestionModel{ExamID: examID, QuestionID: qid, Sequence: i})
	}
	return tx.WithContext(ctx).Create(&examQuestions).Error
}
func (r *examRepository) GetExamDetails(ctx context.Context, examID int64) (*domain.ExamModel, error) {
	var exam domain.ExamModel
	err := database.DB.WithContext(ctx).
		Preload("Questions").Preload("Questions.Choices").Preload("Questions.Type").Preload("Questions.Difficulty").
		Preload("Questions.Section").
		Preload("Questions.Section.Topic").Preload("Topic").
		First(&exam, examID).Error
	
	if err != nil { return nil, err }
	type ExamQuestionOrder struct {
		QuestionID int64
		Sequence   int
	}
	var orders []ExamQuestionOrder
	database.DB.WithContext(ctx).
        Table("exam_questions").
		Select("question_id, sequence").
		Where("exam_id = ?", examID).
		Scan(&orders)

	if len(orders) > 0 {
		orderMap := make(map[int64]int)
		for _, o := range orders {
			orderMap[o.QuestionID] = o.Sequence
		}

		sort.Slice(exam.Questions, func(i, j int) bool {
			return orderMap[exam.Questions[i].Id] < orderMap[exam.Questions[j].Id]
		})
	}
	return &exam, err
}
func (r *examRepository) UpdateExam(ctx context.Context, tx *gorm.DB, examID int64, updates map[string]interface{}) error {
	return tx.WithContext(ctx).Model(&domain.ExamModel{}).Where("id = ?", examID).Updates(updates).Error
}
func (r *examRepository) DeleteExam(ctx context.Context, tx *gorm.DB, examID int64) error {
	tx.WithContext(ctx).Where("exam_id = ?", examID).Delete(&domain.ExamQuestionModel{})
	tx.WithContext(ctx).Where("exam_id = ?", examID).Delete(&domain.ExamSubmissionModel{})
	tx.WithContext(ctx).Where("exam_id = ?", examID).Delete(&domain.ExamAccessRequestModel{})
	return tx.WithContext(ctx).Delete(&domain.ExamModel{}, examID).Error
}
func (r *examRepository) UpdateExamStatus(ctx context.Context, tx *gorm.DB, examID int64, isPublished bool) error {
	return tx.WithContext(ctx).Model(&domain.ExamModel{}).Where("id = ?", examID).Update("is_published", isPublished).Error
}
func (r *examRepository) GetExams(ctx context.Context, limit, offset int, creatorID int64) ([]*domain.ExamModel, int64, error) {
	var exams []*domain.ExamModel
	var total int64
	query := database.DB.WithContext(ctx).Model(&domain.ExamModel{})
	if creatorID > 0 { query = query.Where("creator_id = ?", creatorID) } else { query = query.Where("is_published = ?", true) }
	query.Count(&total)
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&exams).Error
	return exams, total, err
}
func (r *examRepository) CountExams(ctx context.Context) (int64, error) {
	var count int64
	err := database.DB.WithContext(ctx).Model(&domain.ExamModel{}).Count(&count).Error
	return count, err
}

func (r *examRepository) CreateAccessRequest(ctx context.Context, req *domain.ExamAccessRequestModel) error {
	return database.DB.WithContext(ctx).Create(req).Error
}
func (r *examRepository) GetAccessRequest(ctx context.Context, examID, userID int64) (*domain.ExamAccessRequestModel, error) {
	var req domain.ExamAccessRequestModel
	err := database.DB.WithContext(ctx).Where("exam_id = ? AND user_id = ?", examID, userID).First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}
func (r *examRepository) UpdateAccessRequestStatus(ctx context.Context, examID, userID int64, status string) error {
	return database.DB.WithContext(ctx).Model(&domain.ExamAccessRequestModel{}).
		Where("exam_id = ? AND user_id = ?", examID, userID).
		Updates(map[string]interface{}{"status": status, "updated_at": time.Now().UTC()}).Error
}

func (r *examRepository) GetCorrectAnswers(ctx context.Context, examID int64) (map[int64][]int64, error) {
	type CorrectAnswer struct { QuestionID int64; ChoiceID int64 }
	var results []CorrectAnswer
	err := database.DB.WithContext(ctx).Table("choice_models").
		Select("choice_models.question_id, choice_models.id as choice_id").
		Joins("JOIN exam_questions eq ON choice_models.question_id = eq.question_id").
		Where("eq.exam_id = ? AND choice_models.is_correct = ?", examID, true).Scan(&results).Error
	if err != nil { return nil, err }
	answerMap := make(map[int64][]int64)
	for _, res := range results { answerMap[res.QuestionID] = append(answerMap[res.QuestionID], res.ChoiceID) }
	return answerMap, nil
}
func (r *examRepository) CreateSubmission(ctx context.Context, tx *gorm.DB, sub *domain.ExamSubmissionModel) (*domain.ExamSubmissionModel, error) {
	if err := tx.WithContext(ctx).Create(sub).Error; err != nil { return nil, err }
	return sub, nil
}
func (r *examRepository) CreateUserAnswers(ctx context.Context, tx *gorm.DB, ans []*domain.UserAnswerModel) error {
	return tx.WithContext(ctx).Create(ans).Error
}
func (r *examRepository) UpdateSubmission(ctx context.Context, tx *gorm.DB, sub *domain.ExamSubmissionModel) (*domain.ExamSubmissionModel, error) {
	if err := tx.WithContext(ctx).Save(sub).Error; err != nil { return nil, err }
	return sub, nil
}
func (r *examRepository) GetSubmissionByID(ctx context.Context, id int64) (*domain.ExamSubmissionModel, error) {
	var sub domain.ExamSubmissionModel
	err := database.DB.WithContext(ctx).Preload("Exam").Preload("Status").
		Preload("UserAnswers").Preload("UserAnswers.Question").Preload("UserAnswers.Choice").First(&sub, id).Error
	return &sub, err
}
func (r *examRepository) CountSubmissionsByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := database.DB.WithContext(ctx).Model(&domain.ExamSubmissionModel{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
func (r *examRepository) CountSubmissionsForExam(ctx context.Context, examID, userID int64) (int64, error) {
	var count int64
	err := database.DB.WithContext(ctx).Model(&domain.ExamSubmissionModel{}).Where("exam_id = ? AND user_id = ?", examID, userID).Count(&count).Error
	return count, err
}

func (r *examRepository) SaveUserAnswer(ctx context.Context, tx *gorm.DB, ans *domain.UserAnswerModel) error {
	var existing domain.UserAnswerModel
	err := tx.WithContext(ctx).
		Where("submission_id = ? AND question_id = ?", ans.SubmissionID, ans.QuestionID).
		First(&existing).Error

	if err == nil {
		existing.ChosenChoiceID = ans.ChosenChoiceID
        existing.IsCorrect = ans.IsCorrect
		return tx.WithContext(ctx).Save(&existing).Error
	}
	return tx.WithContext(ctx).Create(ans).Error
}

func (r *examRepository) LogViolation(ctx context.Context, v *domain.ExamViolationModel) error {
    return database.DB.WithContext(ctx).Create(v).Error
}

func (r *examRepository) GetExamSubmissions(ctx context.Context, examID int64) ([]*domain.ExamSubmissionModel, error) {
    var subs []*domain.ExamSubmissionModel
    err := database.DB.WithContext(ctx).
        Preload("Status").
        Where("exam_id = ? AND status_id = (SELECT id FROM submission_status_models WHERE status = 'completed')", examID).
        Find(&subs).Error
    return subs, err
}

func (r *examRepository) GetViolationsByExam(ctx context.Context, examID int64) ([]*domain.ExamViolationModel, error) {
    var vs []*domain.ExamViolationModel
    err := database.DB.WithContext(ctx).Where("exam_id = ?", examID).Find(&vs).Error
    return vs, err
}

func (r *examRepository) GetQuestions(ctx context.Context, sectionID int64, topicID int64, difficulty, search string, page, limit int) ([]*domain.QuestionListItem, int64, error) {
    var questions []*domain.QuestionListItem
    var total int64

    query := database.DB.WithContext(ctx).
        Table("question_models q").
        Select(`
            q.id, q.content, qt.type as question_type, qd.difficulty,
            q.section_id, s.name as section_name, 
            s.topic_id, t.name as topic_name,
            q.attachment_url,
            (SELECT COUNT(*) FROM choice_models WHERE question_id = q.id) as choice_count
        `).
        Joins("LEFT JOIN question_type_models qt ON q.type_id = qt.id").
        Joins("LEFT JOIN question_difficulty_models qd ON q.difficulty_id = qd.id").
        Joins("LEFT JOIN exam_sections s ON q.section_id = s.id").
        Joins("LEFT JOIN topic_models t ON s.topic_id = t.id")

    if sectionID > 0 {
        query = query.Where("q.section_id = ?", sectionID)
    }

	if topicID > 0 && sectionID == 0 {
        query = query.Where("q.topic_id = ?", topicID)
    }

    if difficulty != "" {
        query = query.Where("qd.difficulty = ?", difficulty)
    }

    if search != "" {
        query = query.Where("q.content ILIKE ?", "%"+search+"%")
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    offset := (page - 1) * limit
    err := query.Order("q.created_at DESC").Limit(limit).Offset(offset).Scan(&questions).Error

    return questions, total, err
}

func (r *examRepository) GetQuestionByID(ctx context.Context, id int64) (*domain.QuestionModel, error) {
	var q domain.QuestionModel
	err := database.DB.WithContext(ctx).
		Preload("Choices").
		Preload("Type").
		Preload("Difficulty").
		Preload("Section").Preload("Section.Topic").
		First(&q, id).Error
	return &q, err
}

func (r *examRepository) CountUniqueParticipants(ctx context.Context, examID int64) (int64, error) {
    var count int64
    err := database.DB.WithContext(ctx).
        Model(&domain.ExamSubmissionModel{}).
        Where("exam_id = ?", examID).
        Distinct("user_id").
        Count(&count).Error
    return count, err
}

func (r *examRepository) ReplaceExamQuestions(ctx context.Context, tx *gorm.DB, examID int64, questionIDs []int64) error {
	if err := tx.WithContext(ctx).Where("exam_id = ?", examID).Delete(&domain.ExamQuestionModel{}).Error; err != nil {
		return err
	}

	if len(questionIDs) > 0 {
		return r.LinkQuestionsToExam(ctx, tx, examID, questionIDs)
	}
	
	return nil
}

func (r *examRepository) GetAccessRequestsByExam(ctx context.Context, examID int64) ([]*domain.ExamAccessRequestModel, error) {
	var reqs []*domain.ExamAccessRequestModel
	err := database.DB.WithContext(ctx).
        Where("exam_id = ?", examID).
        Order("created_at DESC").
        Find(&reqs).Error
	return reqs, err
}

func (r *examRepository) UpdateTopic(ctx context.Context, id int64, updates map[string]interface{}) error {
	return database.DB.WithContext(ctx).Model(&domain.TopicModel{}).Where("id = ?", id).Updates(updates).Error
}

func (r *examRepository) DeleteTopic(ctx context.Context, tx *gorm.DB, topicID int64) error {
	var sectionIDs []int64
	if err := tx.WithContext(ctx).Model(&domain.SectionModel{}).Where("topic_id = ?", topicID).Pluck("id", &sectionIDs).Error; err != nil {
		return err
	}

	for _, sid := range sectionIDs {
		if err := r.DeleteSection(ctx, tx, sid); err != nil {
			return err
		}
	}

	return tx.WithContext(ctx).Delete(&domain.TopicModel{}, topicID).Error
}

func (r *examRepository) UpdateSection(ctx context.Context, id int64, updates map[string]interface{}) error {
	return database.DB.WithContext(ctx).Model(&domain.SectionModel{}).Where("id = ?", id).Updates(updates).Error
}

func (r *examRepository) DeleteSection(ctx context.Context, tx *gorm.DB, sectionID int64) error {
	var questionIDs []int64
	if err := tx.WithContext(ctx).Model(&domain.QuestionModel{}).Where("section_id = ?", sectionID).Pluck("id", &questionIDs).Error; err != nil {
		return err
	}

	if len(questionIDs) > 0 {
		if err := tx.WithContext(ctx).Where("question_id IN ?", questionIDs).Delete(&domain.ChoiceModel{}).Error; err != nil { return err }
		
		if err := tx.WithContext(ctx).Where("question_id IN ?", questionIDs).Delete(&domain.ExamQuestionModel{}).Error; err != nil { return err }
		
		if err := tx.WithContext(ctx).Where("question_id IN ?", questionIDs).Delete(&domain.UserAnswerModel{}).Error; err != nil { return err }

		if err := tx.WithContext(ctx).Where("id IN ?", questionIDs).Delete(&domain.QuestionModel{}).Error; err != nil { return err }
	}

	return tx.WithContext(ctx).Delete(&domain.SectionModel{}, sectionID).Error
}