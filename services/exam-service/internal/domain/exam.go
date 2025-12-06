package domain

import (
	"context"
	"time"

	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	"gorm.io/gorm"
)

type TopicModel struct {
	Id          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:255;uniqueIndex;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	Sections    []SectionModel `gorm:"foreignKey:TopicID" json:"sections"`
}

type SectionModel struct {
	Id          int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string          `gorm:"size:255;not null" json:"name"`
	Description string          `gorm:"type:text" json:"description"`
	TopicID     int64           `gorm:"not null;index" json:"topic_id"`
	Topic       TopicModel      `gorm:"foreignKey:TopicID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Questions   []QuestionModel `gorm:"foreignKey:SectionID" json:"questions"`
	CreatedAt   time.Time       `json:"created_at"`
}

type QuestionDifficultyModel struct {
	Id         int64
	Difficulty string
}
type QuestionTypeModel struct {
	Id   int64
	Type string
}
type ChoiceModel struct {
	Id         int64
	QuestionID int64
	Content    string
	IsCorrect  bool
	CreatedAt  time.Time
}

type QuestionModel struct {
	Id            int64                   `gorm:"primaryKey;autoIncrement" json:"id"`
	SectionID     int64                   `gorm:"not null;index" json:"section_id"`
	Section       SectionModel            `gorm:"foreignKey:SectionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"section"`
	TopicID       int64                   `gorm:"not null;index" json:"topic_id"`
	CreatorID     int64                   `gorm:"not null" json:"creator_id"`
	Content       string                  `gorm:"type:text;not null" json:"content"`
	TypeID        int64                   `gorm:"not null" json:"type_id"`
	Type          QuestionTypeModel       `gorm:"foreignKey:TypeID" json:"type"`
	DifficultyID  int64                   `gorm:"not null" json:"difficulty_id"`
	Difficulty    QuestionDifficultyModel `gorm:"foreignKey:DifficultyID" json:"difficulty"`
	Explanation   string                  `gorm:"type:text" json:"explanation"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
	Choices       []ChoiceModel           `gorm:"foreignKey:QuestionID" json:"choices"`
	AttachmentURL string                  `gorm:"size:255" json:"attachment_url"`
}

type QuestionListItem struct {
    ID            int64
    Content       string
    QuestionType  string
    Difficulty    string
    SectionID     int64
    SectionName   string
    TopicID       int64
    TopicName     string
    AttachmentURL string
    ChoiceCount   int32
}

type ExamModel struct {
	Id              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Title           string     `gorm:"size:255;not null" json:"title"`
	Description     string     `gorm:"type:text" json:"description"`
	DurationMinutes int        `gorm:"not null" json:"duration_minutes"`
	StartTime       *time.Time `json:"start_time"`
	EndTime         *time.Time `json:"end_time"`
	MaxAttempts     int        `gorm:"default:1" json:"max_attempts"`
	Password        string     `gorm:"size:50" json:"password"`

	ShuffleQuestions      bool `gorm:"default:false" json:"shuffle_questions"`
	ShowResultImmediately bool `gorm:"default:true" json:"show_result_immediately"`
	RequiresApproval      bool `gorm:"default:false" json:"requires_approval"`

	TopicID     int64            `gorm:"not null;index" json:"topic_id"`
	CreatorID   int64            `gorm:"not null" json:"creator_id"`
	IsPublished bool             `gorm:"not null;default:false" json:"is_published"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Questions   []*QuestionModel `gorm:"many2many:exam_questions;" json:"questions"`
}

type ExamQuestionModel struct {
	ExamID     int64 `gorm:"primaryKey"`
	QuestionID int64 `gorm:"primaryKey"`
}

func (ExamQuestionModel) TableName() string {
	return "exam_questions"
}

type ExamAccessRequestModel struct {
	Id        int64     `gorm:"primaryKey;autoIncrement"`
	ExamID    int64     `gorm:"not null;index"`
	UserID    int64     `gorm:"not null;index"`
	Status    string    `gorm:"size:20;default:'pending'"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ExamSubmissionModel struct {
	Id          int64                 `gorm:"primaryKey;autoIncrement"`
	ExamID      int64                 `gorm:"not null;index"`
	Exam        ExamModel             `gorm:"foreignKey:ExamID"`
	UserID      int64                 `gorm:"not null;index"`
	StatusID    int64                 `gorm:"not null"`
	Status      SubmissionStatusModel `gorm:"foreignKey:StatusID"`
	Score       float64
	StartedAt   time.Time
	SubmittedAt *time.Time
	UserAnswers []UserAnswerModel `gorm:"foreignKey:SubmissionID"`
}

type SubmissionStatusModel struct {
	Id     int64
	Status string
}
type UserAnswerModel struct {
	Id             int64       `gorm:"primaryKey;autoIncrement"`
	SubmissionID   int64       `gorm:"not null;index"`
	QuestionID     int64       `gorm:"not null"`
	ChosenChoiceID *int64      
	Choice         ChoiceModel `gorm:"foreignKey:ChosenChoiceID"` 
	Question       QuestionModel `gorm:"foreignKey:QuestionID"`
	IsCorrect      *bool
	CreatedAt      time.Time
}

type ExamViolationModel struct {
	Id            int64     `gorm:"primaryKey;autoIncrement"`
	ExamID        int64     `gorm:"not null;index"`
	UserID        int64     `gorm:"not null;index"`
	ViolationType string    `gorm:"size:50"`
	ViolationTime time.Time
	CreatedAt     time.Time
}

type ExamRepository interface {
	CreateTopic(ctx context.Context, tx *gorm.DB, topic *TopicModel) (*TopicModel, error)
	GetTopicByName(ctx context.Context, name string) (*TopicModel, error)
	GetTopics(ctx context.Context) ([]*TopicModel, error)
	CreateSection(ctx context.Context, tx *gorm.DB, section *SectionModel) (*SectionModel, error)
	GetSectionsByTopic(ctx context.Context, topicID int64) ([]*SectionModel, error)
	GetSectionByID(ctx context.Context, id int64) (*SectionModel, error)

	CreateQuestion(ctx context.Context, tx *gorm.DB, question *QuestionModel) (*QuestionModel, error)
	CreateChoices(ctx context.Context, tx *gorm.DB, choices []*ChoiceModel) error
	GetQuestionType(ctx context.Context, typeName string) (*QuestionTypeModel, error)
	GetDifficulty(ctx context.Context, level string) (*QuestionDifficultyModel, error)
	GetRandomQuestionsBySection(ctx context.Context, sectionID int64, difficulty string, limit int) ([]int64, error)
	UpdateQuestion(ctx context.Context, tx *gorm.DB, qID int64, updates map[string]interface{}) error
	DeleteChoicesByQuestionID(ctx context.Context, tx *gorm.DB, qID int64) error
	DeleteQuestion(ctx context.Context, tx *gorm.DB, questionID int64) error

	CreateExam(ctx context.Context, tx *gorm.DB, exam *ExamModel) (*ExamModel, error)
	LinkQuestionsToExam(ctx context.Context, tx *gorm.DB, examID int64, questionIDs []int64) error
	GetExamDetails(ctx context.Context, examID int64) (*ExamModel, error)
	UpdateExam(ctx context.Context, tx *gorm.DB, examID int64, updates map[string]interface{}) error
	DeleteExam(ctx context.Context, tx *gorm.DB, examID int64) error
	UpdateExamStatus(ctx context.Context, tx *gorm.DB, examID int64, isPublished bool) error
	GetExams(ctx context.Context, limit, offset int, creatorID int64) ([]*ExamModel, int64, error)
	CountExams(ctx context.Context) (int64, error)

	CreateAccessRequest(ctx context.Context, req *ExamAccessRequestModel) error
	GetAccessRequest(ctx context.Context, examID, userID int64) (*ExamAccessRequestModel, error)
	UpdateAccessRequestStatus(ctx context.Context, examID, userID int64, status string) error

	GetCorrectAnswers(ctx context.Context, examID int64) (map[int64][]int64, error)
	CreateSubmission(ctx context.Context, tx *gorm.DB, submission *ExamSubmissionModel) (*ExamSubmissionModel, error)
	CreateUserAnswers(ctx context.Context, tx *gorm.DB, answers []*UserAnswerModel) error
	UpdateSubmission(ctx context.Context, tx *gorm.DB, submission *ExamSubmissionModel) (*ExamSubmissionModel, error)
	GetSubmissionByID(ctx context.Context, submissionID int64) (*ExamSubmissionModel, error)
	CountSubmissionsByUserID(ctx context.Context, userID int64) (int64, error)
	CountSubmissionsForExam(ctx context.Context, examID, userID int64) (int64, error)

	SaveUserAnswer(ctx context.Context, tx *gorm.DB, ans *UserAnswerModel) error
    LogViolation(ctx context.Context, violation *ExamViolationModel) error
    GetExamSubmissions(ctx context.Context, examID int64) ([]*ExamSubmissionModel, error)
    GetViolationsByExam(ctx context.Context, examID int64) ([]*ExamViolationModel, error)
	GetQuestions(ctx context.Context, sectionID int64, topicID int64, difficulty, search string, page, limit int) ([]*QuestionListItem, int64, error)
}

type EventProducer interface {
	Produce(topic string, key []byte, message []byte) error
	Close()
}

type ExamService interface {
	CreateTopic(ctx context.Context, req *pb.CreateTopicRequest) (*pb.CreateTopicResponse, error)
	GetTopics(ctx context.Context, req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error)
	CreateSection(ctx context.Context, req *pb.CreateSectionRequest) (*pb.CreateSectionResponse, error)
	GetSections(ctx context.Context, req *pb.GetSectionsRequest) (*pb.GetSectionsResponse, error)

	GetQuestions(ctx context.Context, req *pb.GetQuestionsRequest) (*pb.GetQuestionsResponse, error)
	CreateQuestion(ctx context.Context, req *pb.CreateQuestionRequest) (*pb.CreateQuestionResponse, error)
	ImportQuestions(ctx context.Context, req *pb.ImportQuestionsRequest) (*pb.ImportQuestionsResponse, error)
	UpdateQuestion(ctx context.Context, req *pb.UpdateQuestionRequest) (*pb.UpdateQuestionResponse, error)
	DeleteQuestion(ctx context.Context, req *pb.DeleteQuestionRequest) (*pb.DeleteQuestionResponse, error)
	GetUploadURL(ctx context.Context, req *pb.GetUploadURLRequest) (*pb.GetUploadURLResponse, error)

	CreateExam(ctx context.Context, req *pb.CreateExamRequest) (*pb.CreateExamResponse, error)
	GenerateExam(ctx context.Context, req *pb.GenerateExamRequest) (*pb.CreateExamResponse, error)
	GetExamDetails(ctx context.Context, req *pb.GetExamDetailsRequest) (*pb.GetExamDetailsResponse, error)
	UpdateExam(ctx context.Context, req *pb.UpdateExamRequest) (*pb.UpdateExamResponse, error)
	DeleteExam(ctx context.Context, req *pb.DeleteExamRequest) (*pb.DeleteExamResponse, error)
	PublishExam(ctx context.Context, req *pb.PublishExamRequest) (*pb.PublishExamResponse, error)
	GetExams(ctx context.Context, req *pb.GetExamsRequest) (*pb.GetExamsResponse, error)
	GetExamCount(ctx context.Context, req *pb.GetExamCountRequest) (*pb.GetExamCountResponse, error)

	RequestExamAccess(ctx context.Context, req *pb.RequestExamAccessRequest) (*pb.RequestExamAccessResponse, error)
	ApproveExamAccess(ctx context.Context, req *pb.ApproveExamAccessRequest) (*pb.ApproveExamAccessResponse, error)
	CheckExamAccess(ctx context.Context, req *pb.CheckExamAccessRequest) (*pb.CheckExamAccessResponse, error)

	SubmitExam(ctx context.Context, req *pb.SubmitExamRequest) (*pb.SubmitExamResponse, error)
	GetSubmission(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error)
	GetUserExamStats(ctx context.Context, req *pb.GetUserExamStatsRequest) (*pb.GetUserExamStatsResponse, error)

	SaveAnswer(ctx context.Context, req *pb.SaveAnswerRequest) (*pb.SaveAnswerResponse, error)
	LogViolation(ctx context.Context, req *pb.LogViolationRequest) (*pb.LogViolationResponse, error)
	GetExamStatsDetailed(ctx context.Context, req *pb.GetExamStatsDetailedRequest) (*pb.GetExamStatsDetailedResponse, error)
	ExportExamResults(ctx context.Context, req *pb.ExportExamResultsRequest) (*pb.ExportExamResultsResponse, error)
}
