package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// Đảm bảo đường dẫn này trỏ đúng đến file domain.go của exam-service
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/env"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		env.GetString("DB_HOST", "postgres"),
		env.GetString("DB_USER", "admin"),
		env.GetString("DB_PASSWORD", "1"),
		env.GetString("DB_NAME", "jqk"),
		env.GetString("DB_PORT", "5432"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to DB: %v", err)
	}

	// Auto migrate tất cả các model của Exam Service
	// Thứ tự này quan trọng để xử lý khóa ngoại
	if err := db.AutoMigrate(
		// 1. Các bảng tra cứu (Lookup tables) - Không phụ thuộc
		&domain.TopicModel{},
		&domain.QuestionDifficultyModel{},
		&domain.QuestionTypeModel{},
		&domain.SubmissionStatusModel{},

		// 2. Các bảng chính - Phụ thuộc vào các bảng trên
		&domain.QuestionModel{},
		&domain.ExamModel{},

		// 3. Các bảng phụ - Phụ thuộc vào bảng chính
		&domain.ChoiceModel{},
		&domain.ExamQuestionModel{}, // Bảng join many-to-many
		&domain.ExamSubmissionModel{},

		// 4. Bảng chi tiết - Phụ thuộc vào bảng phụ
		&domain.UserAnswerModel{},
	); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	DB = db
	log.Println("✅ Database connected and migrated")

	// Seed dữ liệu cho các bảng tra cứu
	seedExamData(db)
}

// seedExamData dùng để điền các giá trị mặc định cho các bảng lookup
func seedExamData(db *gorm.DB) {
	// Seed Difficulties
	difficulties := []domain.QuestionDifficultyModel{
		{Difficulty: "easy"},
		{Difficulty: "medium"},
		{Difficulty: "hard"},
	}
	for _, d := range difficulties {
		db.FirstOrCreate(&d, domain.QuestionDifficultyModel{Difficulty: d.Difficulty})
	}

	// Seed Types
	types := []domain.QuestionTypeModel{
		{Type: "single_choice"},
		{Type: "multiple_choice"},
	}
	for _, t := range types {
		db.FirstOrCreate(&t, domain.QuestionTypeModel{Type: t.Type})
	}

	statuses := []domain.SubmissionStatusModel{
		{Status: "in_progress"},
		{Status: "completed"},
	}
	for _, s := range statuses {
		db.FirstOrCreate(&s, domain.SubmissionStatusModel{Status: s.Status})
	}

	log.Println("✅ Seeded default exam lookup data")
}