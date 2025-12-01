package domain

import (
	"context"
	"time"

	// THAY ĐỔI: Import proto của Course Service
	pb "github.com/06babyshark06/JQKStudy/shared/proto/course" 
	"gorm.io/gorm"
)

// =================================================================
// GORM MODELS (Dựa trên cấu trúc database bạn cung cấp)
// =================================================================

// CourseModel 🎓
type CourseModel struct {
	Id           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Title        string    `gorm:"size:255;not null" json:"title"`
	Description  string    `gorm:"type:text" json:"description"`
	ThumbnailURL string    `gorm:"size:255" json:"thumbnail_url"`
	InstructorID int64     `gorm:"not null" json:"instructor_id"` // User ID từ User Service
	Price        float64   `gorm:"not null;default:0" json:"price"`
	IsPublished  bool      `gorm:"not null;default:false" json:"is_published"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Sections     []SectionModel `gorm:"foreignKey:CourseID" json:"sections"` // Quan hệ một-nhiều
}

// SectionModel 📂
type SectionModel struct {
	Id         int64         `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseID   int64         `gorm:"not null;index" json:"course_id"`
	Course     CourseModel   `gorm:"foreignKey:CourseID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Title      string        `gorm:"size:255;not null" json:"title"`
	OrderIndex int           `gorm:"not null;default:0" json:"order_index"`
	CreatedAt  time.Time     `json:"created_at"`
	Lessons    []LessonModel `gorm:"foreignKey:SectionID" json:"lessons"` // Quan hệ một-nhiều
}

// LessonTypeModel 📋
type LessonTypeModel struct {
	Id   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Type string `gorm:"size:50;uniqueIndex;not null" json:"type"` // "video", "text"
}

// LessonModel 📖
type LessonModel struct {
	Id              int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	SectionID       int64           `gorm:"not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"section_id"`
	Section         SectionModel    `gorm:"foreignKey:SectionID" json:"-"`
	Title           string          `gorm:"size:255;not null" json:"title"`
	TypeID          int64           `gorm:"not null" json:"type_id"`
	Type            LessonTypeModel `gorm:"foreignKey:TypeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"type"`
	ContentURL      string          `gorm:"size:255" json:"content_url"` // Link R2/S3
	DurationSeconds int             `gorm:"not null;default:0" json:"duration_seconds"`
	OrderIndex      int             `gorm:"not null;default:0" json:"order_index"`
	CreatedAt       time.Time       `json:"created_at"`
}

// EnrollmentModel 🧑‍🎓
type EnrollmentModel struct {
	UserID     int64     `gorm:"primaryKey" json:"user_id"`  // Khóa chính kết hợp
	CourseID   int64     `gorm:"primaryKey" json:"course_id"` // Khóa chính kết hợp
	Course     CourseModel `gorm:"foreignKey:CourseID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"course"`
	EnrolledAt time.Time `json:"enrolled_at"`
}

// LessonProgressModel 📊
type LessonProgressModel struct {
	UserID      int64       `gorm:"primaryKey" json:"user_id"`  // Khóa chính kết hợp
	LessonID    int64       `gorm:"primaryKey" json:"lesson_id"` // Khóa chính kết hợp
	Lesson      LessonModel `gorm:"foreignKey:LessonID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"lesson"`
	CompletedAt time.Time   `json:"completed_at"`
}


// =================================================================
// INTERFACES (Định nghĩa các "Hợp đồng")
// =================================================================

// CourseRepository định nghĩa các phương thức tương tác với DB cho CourseService
type CourseRepository interface {
	// Course
	CreateCourse(ctx context.Context, tx *gorm.DB, course *CourseModel) (*CourseModel, error)
	GetCourseByID(ctx context.Context, courseID int64) (*CourseModel, error)
	GetCourses(ctx context.Context, search string, minPrice, maxPrice float64, sortBy string, page, limit int, instructorID int64) ([]*CourseModel, int64, error)
	GetCourseDetails(ctx context.Context, courseID int64) (*CourseModel, error) // Preload Sections.Lessons
	GetEnrolledCourses(ctx context.Context, userID int64) ([]*CourseModel, error) // Dùng Joins

	// Section
	CreateSection(ctx context.Context, tx *gorm.DB, section *SectionModel) (*SectionModel, error)

	// Lesson
	CreateLesson(ctx context.Context, tx *gorm.DB, lesson *LessonModel) (*LessonModel, error)
	GetLessonType(ctx context.Context, typeName string) (*LessonTypeModel, error)

	// Enrollment
	CreateEnrollment(ctx context.Context, tx *gorm.DB, enrollment *EnrollmentModel) error
	GetEnrollment(ctx context.Context, userID int64, courseID int64) (*EnrollmentModel, error)

	// Progress
	CreateLessonProgress(ctx context.Context, tx *gorm.DB, progress *LessonProgressModel) error
	GetLessonProgress(ctx context.Context, userID int64, lessonID int64) (*LessonProgressModel, error)
	GetCompletedLessonIDs(ctx context.Context, userID int64, courseID int64) (map[int64]bool, error)

	UpdateCourse(ctx context.Context, tx *gorm.DB, courseID int64, updates map[string]interface{}) error
    UpdateSection(ctx context.Context, tx *gorm.DB, sectionID int64, title string) error
    DeleteSection(ctx context.Context, tx *gorm.DB, sectionID int64) error
    DeleteLesson(ctx context.Context, tx *gorm.DB, lessonID int64) error
	UpdateLesson(ctx context.Context, tx *gorm.DB, lessonID int64, updates map[string]interface{}) error
	UpdateCourseStatus(ctx context.Context, tx *gorm.DB, courseID int64, isPublished bool) error
	CountCourses(ctx context.Context) (int64, error)
	DeleteCourse(ctx context.Context, tx *gorm.DB, courseID int64) error
}

type EventProducer interface {
	Produce(topic string, key []byte, message []byte) error
	Close()
}

// CourseService định nghĩa các logic nghiệp vụ (business logic)
// Nó hoạt động trên các Protobuf (pb) structs
type CourseService interface {
	CreateCourse(ctx context.Context, req *pb.CreateCourseRequest) (*pb.CreateCourseResponse, error)
	CreateSection(ctx context.Context, req *pb.CreateSectionRequest) (*pb.CreateSectionResponse, error)
	CreateLesson(ctx context.Context, req *pb.CreateLessonRequest) (*pb.CreateLessonResponse, error)
	GetCourses(ctx context.Context, req *pb.GetCoursesRequest) (*pb.GetCoursesResponse, error)
	GetCourseDetails(ctx context.Context, req *pb.GetCourseDetailsRequest) (*pb.GetCourseDetailsResponse, error)
	EnrollCourse(ctx context.Context, req *pb.EnrollCourseRequest) (*pb.EnrollCourseResponse, error)
	GetMyCourses(ctx context.Context, req *pb.GetMyCoursesRequest) (*pb.GetMyCoursesResponse, error)
	MarkLessonCompleted(ctx context.Context, req *pb.MarkLessonCompletedRequest) (*pb.MarkLessonCompletedResponse, error)
	GetUploadURL(ctx context.Context, req *pb.GetUploadURLRequest) (*pb.GetUploadURLResponse, error)
	UpdateCourse(ctx context.Context, req *pb.UpdateCourseRequest) (*pb.UpdateCourseResponse, error)
	UpdateSection(ctx context.Context, req *pb.UpdateSectionRequest) (*pb.UpdateSectionResponse, error)
	DeleteSection(ctx context.Context, req *pb.DeleteSectionRequest) (*pb.DeleteSectionResponse, error)
	DeleteLesson(ctx context.Context, req *pb.DeleteLessonRequest) (*pb.DeleteLessonResponse, error)
	UpdateLesson(ctx context.Context, req *pb.UpdateLessonRequest) (*pb.UpdateLessonResponse, error)
	PublishCourse(ctx context.Context, req *pb.PublishCourseRequest) (*pb.PublishCourseResponse, error)
	GetCourseCount(ctx context.Context, req *pb.GetCourseCountRequest) (*pb.GetCourseCountResponse, error)
	DeleteCourse(ctx context.Context, req *pb.DeleteCourseRequest) (*pb.DeleteCourseResponse, error)
}