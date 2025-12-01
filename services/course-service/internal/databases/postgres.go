package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/06babyshark06/JQKStudy/services/course-service/internal/domain"
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
		&domain.LessonTypeModel{},

		&domain.CourseModel{},

		&domain.SectionModel{},
		&domain.EnrollmentModel{},

		&domain.LessonModel{},

		&domain.LessonProgressModel{},
	); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	DB = db
	log.Println("✅ Database connected and migrated")

	seedCourseData(db)
}

func seedCourseData(db *gorm.DB) {
	types := []domain.LessonTypeModel{
		{Type: "video"},
		{Type: "text"},
	}
	for _, t := range types {
		db.FirstOrCreate(&t, domain.LessonTypeModel{Type: t.Type})
	}

	log.Println("✅ Seeded default course lookup data")
}