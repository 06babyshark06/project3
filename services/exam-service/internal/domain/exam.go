package domain

import (
	"context"
	"time"

	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	"gorm.io/gorm"
)

type TopicModel struct {
	Id          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:255;uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type QuestionDifficultyModel struct {
	Id         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Difficulty string `gorm:"size:50;uniqueIndex;not null" json:"difficulty"`
}

type QuestionTypeModel struct {
	Id   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Type string `gorm:"size:50;uniqueIndex;not null" json:"type"`
}

type QuestionModel struct {
	Id           int64                   `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID      int64                   `gorm:"not null;index" json:"topic_id"`
	Topic        TopicModel              `gorm:"foreignKey:TopicID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"topic"`
	CreatorID    int64                   `gorm:"not null" json:"creator_id"`
	Content      string                  `gorm:"type:text;not null" json:"content"`
	TypeID       int64                   `gorm:"not null" json:"type_id"`
	Type         QuestionTypeModel       `gorm:"foreignKey:TypeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"type"`
	DifficultyID int64                   `gorm:"not null" json:"difficulty_id"`
	Difficulty   QuestionDifficultyModel `gorm:"foreignKey:DifficultyID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"difficulty"`
	Explanation  string                  `gorm:"type:text" json:"explanation"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	Choices      []ChoiceModel           `gorm:"foreignKey:QuestionID" json:"choices"`
}

type ChoiceModel struct {
	Id         int64         `gorm:"primaryKey;autoIncrement" json:"id"`
	QuestionID int64         `gorm:"not null;index" json:"question_id"`
	Question   QuestionModel `gorm:"foreignKey:QuestionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Content    string        `gorm:"type:text;not null" json:"content"`
	IsCorrect  bool          `gorm:"not null" json:"is_correct"`
	CreatedAt  time.Time     `json:"created_at"`
}

type ExamModel struct {
	Id              int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	Title           string            `gorm:"size:255;not null" json:"title"`
	Description     string            `gorm:"type:text" json:"description"`
	DurationMinutes int               `gorm:"not null" json:"duration_minutes"`
	TopicID         int64             `gorm:"not null;index" json:"topic_id"`
	Topic           TopicModel        `gorm:"foreignKey:TopicID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"topic"`
	CreatorID       int64             `gorm:"not null" json:"creator_id"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Questions       []*QuestionModel  `gorm:"many2many:exam_questions;joinForeignKey:exam_id;joinReferences:question_id" json:"questions"`
	IsPublished bool      `gorm:"not null;default:false" json:"is_published"`
}

type ExamQuestionModel struct {
	ExamID     int64 `gorm:"primaryKey" json:"exam_id"`
	QuestionID int64 `gorm:"primaryKey" json:"question_id"`
}

func (ExamQuestionModel) TableName() string {
	return "exam_questions"
}

type SubmissionStatusModel struct {
	Id     int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Status string `gorm:"size:50;uniqueIndex;not null" json:"status"`
}

type ExamSubmissionModel struct {
	Id          int64                 `gorm:"primaryKey;autoIncrement" json:"id"`
	ExamID      int64                 `gorm:"not null;index" json:"exam_id"`
	Exam        ExamModel             `gorm:"foreignKey:ExamID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"exam"`
	UserID      int64                 `gorm:"not null;index" json:"user_id"`
	StatusID    int64                 `gorm:"not null" json:"status_id"`
	Status      SubmissionStatusModel `gorm:"foreignKey:StatusID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"status"`
	Score       float64               `json:"score"`
	StartedAt   time.Time             `json:"started_at"`
	SubmittedAt *time.Time            `json:"submitted_at"`
	UserAnswers []UserAnswerModel     `gorm:"foreignKey:SubmissionID" json:"user_answers"`
}

type UserAnswerModel struct {
	Id             int64               `gorm:"primaryKey;autoIncrement" json:"id"`
	SubmissionID   int64               `gorm:"not null;index" json:"submission_id"`
	Submission     ExamSubmissionModel `gorm:"foreignKey:SubmissionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	QuestionID     int64               `gorm:"not null" json:"question_id"`
	Question       QuestionModel       `gorm:"foreignKey:QuestionID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"question"`
	ChosenChoiceID *int64              `json:"chosen_choice_id"`
	Choice         ChoiceModel         `gorm:"foreignKey:ChosenChoiceID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"choice"`
	IsCorrect      *bool               `json:"is_correct"`
	CreatedAt      time.Time           `json:"created_at"`
}

type ExamRepository interface {
	CreateTopic(ctx context.Context, tx *gorm.DB, topic *TopicModel) (*TopicModel, error)
	GetTopicByName(ctx context.Context, name string) (*TopicModel, error)
	GetTopics(ctx context.Context) ([]*TopicModel, error)

	CreateQuestion(ctx context.Context, tx *gorm.DB, question *QuestionModel) (*QuestionModel, error)
	CreateChoices(ctx context.Context, tx *gorm.DB, choices []*ChoiceModel) error

	CreateExam(ctx context.Context, tx *gorm.DB, exam *ExamModel) (*ExamModel, error)
	LinkQuestionsToExam(ctx context.Context, tx *gorm.DB, examID int64, questionIDs []int64) error
	GetExamDetails(ctx context.Context, examID int64) (*ExamModel, error)

	GetCorrectAnswers(ctx context.Context, examID int64) (map[int64][]int64, error)
	CreateSubmission(ctx context.Context, tx *gorm.DB, submission *ExamSubmissionModel) (*ExamSubmissionModel, error)
	CreateUserAnswers(ctx context.Context, tx *gorm.DB, answers []*UserAnswerModel) error
	UpdateSubmission(ctx context.Context, tx *gorm.DB, submission *ExamSubmissionModel) (*ExamSubmissionModel, error)
	GetSubmissionByID(ctx context.Context, submissionID int64) (*ExamSubmissionModel, error)
	CountExams(ctx context.Context) (int64, error)
	GetExams(ctx context.Context, limit, offset int, creatorID int64) ([]*ExamModel, int64, error)
	UpdateExamStatus(ctx context.Context, tx *gorm.DB, examID int64, isPublished bool) error
	UpdateQuestion(ctx context.Context, tx *gorm.DB, qID int64, updates map[string]interface{}) error
	DeleteChoicesByQuestionID(ctx context.Context, tx *gorm.DB, qID int64) error
	DeleteQuestion(ctx context.Context, tx *gorm.DB, questionID int64) error
	UpdateExam(ctx context.Context, tx *gorm.DB, examID int64, updates map[string]interface{}) error
	DeleteExam(ctx context.Context, tx *gorm.DB, examID int64) error
	CountSubmissionsByUserID(ctx context.Context, userID int64) (int64, error)
}

type EventProducer interface {
	Produce(topic string, key []byte, message []byte) error
	Close()
}

type ExamService interface {
	CreateTopic(ctx context.Context, req *pb.CreateTopicRequest) (*pb.CreateTopicResponse, error)
	GetTopics(ctx context.Context, req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error)
	CreateQuestion(ctx context.Context, req *pb.CreateQuestionRequest) (*pb.CreateQuestionResponse, error)
	CreateExam(ctx context.Context, req *pb.CreateExamRequest) (*pb.CreateExamResponse, error)
	GetExamDetails(ctx context.Context, req *pb.GetExamDetailsRequest) (*pb.GetExamDetailsResponse, error)
	SubmitExam(ctx context.Context, req *pb.SubmitExamRequest) (*pb.SubmitExamResponse, error)
	GetSubmission(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error)
	GetExamCount(ctx context.Context, req *pb.GetExamCountRequest) (*pb.GetExamCountResponse, error)
	GetExams(ctx context.Context, req *pb.GetExamsRequest) (*pb.GetExamsResponse, error)
	PublishExam(ctx context.Context, req *pb.PublishExamRequest) (*pb.PublishExamResponse, error)
	UpdateQuestion(ctx context.Context, req *pb.UpdateQuestionRequest) (*pb.UpdateQuestionResponse, error)
	DeleteQuestion(ctx context.Context, req *pb.DeleteQuestionRequest) (*pb.DeleteQuestionResponse, error)
	UpdateExam(ctx context.Context, req *pb.UpdateExamRequest) (*pb.UpdateExamResponse, error)
	DeleteExam(ctx context.Context, req *pb.DeleteExamRequest) (*pb.DeleteExamResponse, error)
	GetUserExamStats(ctx context.Context, req *pb.GetUserExamStatsRequest) (*pb.GetUserExamStatsResponse, error)
}