package repository

import (
	"context"

	database "github.com/06babyshark06/JQKStudy/services/exam-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/domain"
	"gorm.io/gorm"
)

type examRepository struct{}

func NewExamRepository() domain.ExamRepository {
	return &examRepository{}
}

func (r *examRepository) CreateTopic(ctx context.Context, tx *gorm.DB, topic *domain.TopicModel) (*domain.TopicModel, error) {
	if err := tx.WithContext(ctx).Create(topic).Error; err != nil {
		return nil, err
	}
	return topic, nil
}

func (r *examRepository) GetTopicByName(ctx context.Context, name string) (*domain.TopicModel, error) {
	var topic domain.TopicModel
	if err := database.DB.WithContext(ctx).Where("name = ?", name).First(&topic).Error; err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r *examRepository) GetTopics(ctx context.Context) ([]*domain.TopicModel, error) {
	var topics []*domain.TopicModel
	if err := database.DB.WithContext(ctx).Order("created_at DESC").Find(&topics).Error; err != nil {
		return nil, err
	}
	return topics, nil
}

func (r *examRepository) CreateQuestion(ctx context.Context, tx *gorm.DB, question *domain.QuestionModel) (*domain.QuestionModel, error) {
	if err := tx.WithContext(ctx).Create(question).Error; err != nil {
		return nil, err
	}
	return question, nil
}

func (r *examRepository) CreateChoices(ctx context.Context, tx *gorm.DB, choices []*domain.ChoiceModel) error {
	if err := tx.WithContext(ctx).Create(choices).Error; err != nil {
		return err
	}
	return nil
}

func (r *examRepository) CreateExam(ctx context.Context, tx *gorm.DB, exam *domain.ExamModel) (*domain.ExamModel, error) {
	if err := tx.WithContext(ctx).Create(exam).Error; err != nil {
		return nil, err
	}
	return exam, nil
}

func (r *examRepository) LinkQuestionsToExam(ctx context.Context, tx *gorm.DB, examID int64, questionIDs []int64) error {
	if len(questionIDs) == 0 {
        return nil
    }

	var examQuestions []*domain.ExamQuestionModel
	for _, qid := range questionIDs {
		examQuestions = append(examQuestions, &domain.ExamQuestionModel{
			ExamID:     examID,
			QuestionID: qid,
		})
	}

	if err := tx.WithContext(ctx).Create(&examQuestions).Error; err != nil {
		return err
	}
	return nil
}

func (r *examRepository) GetExamDetails(ctx context.Context, examID int64) (*domain.ExamModel, error) {
	var exam domain.ExamModel

	err := database.DB.WithContext(ctx).
		Preload("Questions").
		Preload("Questions.Choices").
		Preload("Questions.Type").
		Preload("Topic").
		First(&exam, examID).Error

	if err != nil {
		return nil, err
	}
	return &exam, nil
}

func (r *examRepository) GetCorrectAnswers(ctx context.Context, examID int64) (map[int64][]int64, error) {
	type CorrectAnswer struct {
		QuestionID int64
		ChoiceID   int64
	}
	var results []CorrectAnswer

	err := database.DB.WithContext(ctx).
		Table("choice_models").
		Select("choice_models.question_id, choice_models.id as choice_id").
		Joins("JOIN exam_questions eq ON choice_models.question_id = eq.question_id").
		Joins("JOIN question_models q ON q.id = choice_models.question_id").
		Where("eq.exam_id = ? AND choice_models.is_correct = ?", examID, true).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	answerMap := make(map[int64][]int64)
	for _, res := range results {
		answerMap[res.QuestionID] = append(answerMap[res.QuestionID], res.ChoiceID)
	}

	return answerMap, nil
}

func (r *examRepository) CreateSubmission(ctx context.Context, tx *gorm.DB, submission *domain.ExamSubmissionModel) (*domain.ExamSubmissionModel, error) {
	if err := tx.WithContext(ctx).Create(submission).Error; err != nil {
		return nil, err
	}
	return submission, nil
}

func (r *examRepository) CreateUserAnswers(ctx context.Context, tx *gorm.DB, answers []*domain.UserAnswerModel) error {
	if err := tx.WithContext(ctx).Create(answers).Error; err != nil {
		return err
	}
	return nil
}

func (r *examRepository) UpdateSubmission(ctx context.Context, tx *gorm.DB, submission *domain.ExamSubmissionModel) (*domain.ExamSubmissionModel, error) {
	if err := tx.WithContext(ctx).Save(submission).Error; err != nil {
		return nil, err
	}
	return submission, nil
}

func (r *examRepository) GetSubmissionByID(ctx context.Context, submissionID int64) (*domain.ExamSubmissionModel, error) {
	var submission domain.ExamSubmissionModel

	err := database.DB.WithContext(ctx).
		Preload("Exam").
		Preload("Status").
		Preload("UserAnswers").
		Preload("UserAnswers.Question").
		Preload("UserAnswers.Choice").
		First(&submission, submissionID).Error

	if err != nil {
		return nil, err
	}
	return &submission, nil
}

func (r *examRepository) CountExams(ctx context.Context) (int64, error) {
    var count int64
    if err := database.DB.WithContext(ctx).Model(&domain.ExamModel{}).Count(&count).Error; err != nil {
        return 0, err
    }
    return count, nil
}

func (r *examRepository) GetExams(ctx context.Context, limit, offset int, creatorID int64) ([]*domain.ExamModel, int64, error) {
	var exams []*domain.ExamModel
	var total int64

	query := database.DB.WithContext(ctx).Model(&domain.ExamModel{})

	if creatorID > 0 {
		query = query.Where("creator_id = ?", creatorID)
	} else {
		query = query.Where("is_published = ?", true)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&exams).Error; err != nil {
		return nil, 0, err
	}
	return exams, total, nil
}

func (r *examRepository) UpdateExamStatus(ctx context.Context, tx *gorm.DB, examID int64, isPublished bool) error {
    return tx.WithContext(ctx).Model(&domain.ExamModel{}).Where("id = ?", examID).Update("is_published", isPublished).Error
}

func (r *examRepository) UpdateQuestion(ctx context.Context, tx *gorm.DB, qID int64, updates map[string]interface{}) error {
    return tx.WithContext(ctx).Model(&domain.QuestionModel{}).Where("id = ?", qID).Updates(updates).Error
}

func (r *examRepository) DeleteChoicesByQuestionID(ctx context.Context, tx *gorm.DB, qID int64) error {
    return tx.WithContext(ctx).Where("question_id = ?", qID).Delete(&domain.ChoiceModel{}).Error
}

func (r *examRepository) DeleteQuestion(ctx context.Context, tx *gorm.DB, questionID int64) error {
    if err := tx.WithContext(ctx).Where("question_id = ?", questionID).Delete(&domain.ChoiceModel{}).Error; err != nil {
        return err
    }

    if err := tx.WithContext(ctx).Where("question_id = ?", questionID).Delete(&domain.ExamQuestionModel{}).Error; err != nil {
        return err
    }

    // 3. Xóa Câu hỏi
    if err := tx.WithContext(ctx).Delete(&domain.QuestionModel{}, questionID).Error; err != nil {
        return err
    }
    return nil
}

func (r *examRepository) UpdateExam(ctx context.Context, tx *gorm.DB, examID int64, updates map[string]interface{}) error {
    return tx.WithContext(ctx).Model(&domain.ExamModel{}).Where("id = ?", examID).Updates(updates).Error
}

func (r *examRepository) DeleteExam(ctx context.Context, tx *gorm.DB, examID int64) error {
    
    if err := tx.WithContext(ctx).Where("exam_id = ?", examID).Delete(&domain.ExamQuestionModel{}).Error; err != nil {
        return err
    }
    if err := tx.WithContext(ctx).Where("exam_id = ?", examID).Delete(&domain.ExamSubmissionModel{}).Error; err != nil {
        return err
    }

    if err := tx.WithContext(ctx).Delete(&domain.ExamModel{}, examID).Error; err != nil {
        return err
    }
    return nil
}

func (r *examRepository) CountSubmissionsByUserID(ctx context.Context, userID int64) (int64, error) {
	var count int64
	if err := database.DB.WithContext(ctx).
		Model(&domain.ExamSubmissionModel{}).
		Where("user_id = ?", userID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}