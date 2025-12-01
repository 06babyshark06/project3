package domain

import (
	"context"
	"time"

	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam" // Sử dụng import path từ ví dụ của bạn
	"gorm.io/gorm"
)

// =================================================================
// GORM MODELS (Dựa trên cấu trúc database bạn cung cấp)
// =================================================================

// TopicModel 📚
type TopicModel struct {
	Id          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:255;uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// QuestionDifficultyModel 📈
type QuestionDifficultyModel struct {
	Id         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Difficulty string `gorm:"size:50;uniqueIndex;not null" json:"difficulty"` // Ví dụ: "Dễ", "Trung bình", "Khó"
}

// QuestionTypeModel 📋
type QuestionTypeModel struct {
	Id   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Type string `gorm:"size:50;uniqueIndex;not null" json:"type"` // Ví dụ: "single_choice", "multiple_choice"
}

// QuestionModel ❓
type QuestionModel struct {
	Id           int64                   `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID      int64                   `gorm:"not null;index" json:"topic_id"`
	Topic        TopicModel              `gorm:"foreignKey:TopicID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"topic"`
	CreatorID    int64                   `gorm:"not null" json:"creator_id"` // User ID từ User Service
	Content      string                  `gorm:"type:text;not null" json:"content"`
	TypeID       int64                   `gorm:"not null" json:"type_id"`
	Type         QuestionTypeModel       `gorm:"foreignKey:TypeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"type"`
	DifficultyID int64                   `gorm:"not null" json:"difficulty_id"`
	Difficulty   QuestionDifficultyModel `gorm:"foreignKey:DifficultyID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"difficulty"`
	Explanation  string                  `gorm:"type:text" json:"explanation"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	Choices      []ChoiceModel           `gorm:"foreignKey:QuestionID" json:"choices"` // Quan hệ một-nhiều
}

// ChoiceModel ✅
type ChoiceModel struct {
	Id         int64         `gorm:"primaryKey;autoIncrement" json:"id"`
	QuestionID int64         `gorm:"not null;index" json:"question_id"`
	Question   QuestionModel `gorm:"foreignKey:QuestionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"` // Xóa choice nếu question bị xóa
	Content    string        `gorm:"type:text;not null" json:"content"`
	IsCorrect  bool          `gorm:"not null" json:"is_correct"`
	CreatedAt  time.Time     `json:"created_at"`
}

// ExamModel 📝
type ExamModel struct {
	Id              int64             `gorm:"primaryKey;autoIncrement" json:"id"`
	Title           string            `gorm:"size:255;not null" json:"title"`
	Description     string            `gorm:"type:text" json:"description"`
	DurationMinutes int               `gorm:"not null" json:"duration_minutes"`
	TopicID         int64             `gorm:"not null;index" json:"topic_id"`
	Topic           TopicModel        `gorm:"foreignKey:TopicID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"topic"`
	CreatorID       int64             `gorm:"not null" json:"creator_id"` // User ID từ User Service
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Questions       []*QuestionModel  `gorm:"many2many:exam_questions;joinForeignKey:exam_id;joinReferences:question_id" json:"questions"`
	IsPublished bool      `gorm:"not null;default:false" json:"is_published"`
}

// ExamQuestionModel (Bảng trung gian cho GORM, nếu không dùng `many2many` tự động)
type ExamQuestionModel struct {
	ExamID     int64 `gorm:"primaryKey" json:"exam_id"`
	QuestionID int64 `gorm:"primaryKey" json:"question_id"`
}

// TableName chỉ định tên bảng nếu GORM không tự động đoán đúng `exam_questions`
func (ExamQuestionModel) TableName() string {
	return "exam_questions"
}

// SubmissionStatusModel 📊
type SubmissionStatusModel struct {
	Id     int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Status string `gorm:"size:50;uniqueIndex;not null" json:"status"` // Ví dụ: "in_progress", "completed"
}

// ExamSubmissionModel 🚀
type ExamSubmissionModel struct {
	Id          int64                 `gorm:"primaryKey;autoIncrement" json:"id"`
	ExamID      int64                 `gorm:"not null;index" json:"exam_id"`
	Exam        ExamModel             `gorm:"foreignKey:ExamID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"exam"`
	UserID      int64                 `gorm:"not null;index" json:"user_id"` // User ID từ User Service
	StatusID    int64                 `gorm:"not null" json:"status_id"`
	Status      SubmissionStatusModel `gorm:"foreignKey:StatusID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"status"`
	Score       float64               `json:"score"`
	StartedAt   time.Time             `json:"started_at"`
	SubmittedAt *time.Time            `json:"submitted_at"` // Dùng con trỏ để có thể là NULL
	UserAnswers []UserAnswerModel     `gorm:"foreignKey:SubmissionID" json:"user_answers"` // Quan hệ một-nhiều
}

// UserAnswerModel 🖋️
type UserAnswerModel struct {
	Id             int64               `gorm:"primaryKey;autoIncrement" json:"id"`
	SubmissionID   int64               `gorm:"not null;index" json:"submission_id"`
	Submission     ExamSubmissionModel `gorm:"foreignKey:SubmissionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	QuestionID     int64               `gorm:"not null" json:"question_id"`
	Question       QuestionModel       `gorm:"foreignKey:QuestionID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"question"`
	ChosenChoiceID *int64              `json:"chosen_choice_id"` // Con trỏ để có thể là NULL (nếu không trả lời)
	Choice         ChoiceModel         `gorm:"foreignKey:ChosenChoiceID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"choice"`
	IsCorrect      *bool               `json:"is_correct"` // Con trỏ để có thể là NULL (chưa chấm)
	CreatedAt      time.Time           `json:"created_at"`
}


// =================================================================
// INTERFACES (Định nghĩa các "Hợp đồng")
// =================================================================

// ExamRepository định nghĩa các phương thức tương tác với DB cho ExamService
// Nó hoạt động trên các GORM models
type ExamRepository interface {
	// Topic
	CreateTopic(ctx context.Context, tx *gorm.DB, topic *TopicModel) (*TopicModel, error)
	GetTopicByName(ctx context.Context, name string) (*TopicModel, error)
	GetTopics(ctx context.Context) ([]*TopicModel, error)

	// Question & Choice
	CreateQuestion(ctx context.Context, tx *gorm.DB, question *QuestionModel) (*QuestionModel, error)
	CreateChoices(ctx context.Context, tx *gorm.DB, choices []*ChoiceModel) error
    // (Lưu ý: Bạn có thể gộp CreateQuestion và CreateChoices trong một transaction ở tầng service)

	// Exam
	CreateExam(ctx context.Context, tx *gorm.DB, exam *ExamModel) (*ExamModel, error)
	LinkQuestionsToExam(ctx context.Context, tx *gorm.DB, examID int64, questionIDs []int64) error
	GetExamDetails(ctx context.Context, examID int64) (*ExamModel, error) // Repo này sẽ Preload Questions và Choices (không có is_correct)

	// Submission
	GetCorrectAnswers(ctx context.Context, examID int64) (map[int64][]int64, error) // map[question_id] -> []correct_choice_id
	CreateSubmission(ctx context.Context, tx *gorm.DB, submission *ExamSubmissionModel) (*ExamSubmissionModel, error)
	CreateUserAnswers(ctx context.Context, tx *gorm.DB, answers []*UserAnswerModel) error // Bulk insert
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
}