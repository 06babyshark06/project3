package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

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

	if err := db.AutoMigrate(
		&domain.TopicModel{},
		&domain.QuestionDifficultyModel{},
		&domain.QuestionTypeModel{},
		&domain.SubmissionStatusModel{},

		&domain.QuestionModel{},
		&domain.ExamModel{},

		&domain.ChoiceModel{},
		&domain.ExamQuestionModel{},
		&domain.ExamSubmissionModel{},

		&domain.UserAnswerModel{},
		&domain.ExamViolationModel{},
		&domain.ExamAccessRequestModel{},
	); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	DB = db
	log.Println("✅ Database connected and migrated")

	seedExamData(db)
}

func seedExamData(db *gorm.DB) {
	difficulties := []domain.QuestionDifficultyModel{
		{Difficulty: "easy"},
		{Difficulty: "medium"},
		{Difficulty: "hard"},
	}
	for _, d := range difficulties {
		db.FirstOrCreate(&d, domain.QuestionDifficultyModel{Difficulty: d.Difficulty})
	}

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