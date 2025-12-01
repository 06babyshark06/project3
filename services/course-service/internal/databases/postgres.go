package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// THAY ĐỔI: Trỏ đến domain của course-service
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
		env.GetString("DB_NAME", "jqk"), // THAY ĐỔI: Tên DB mặc định
		env.GetString("DB_PORT", "5432"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to DB: %v", err)
	}

	// THAY ĐỔI: Auto migrate tất cả các model của Course Service
	if err := db.AutoMigrate(
		// 1. Các bảng tra cứu (Lookup tables)
		&domain.LessonTypeModel{},

		// 2. Bảng chính
		&domain.CourseModel{},

		// 3. Các bảng phụ (phụ thuộc bảng chính)
		&domain.SectionModel{},
		&domain.EnrollmentModel{},

		// 4. Bảng chi tiết (phụ thuộc bảng phụ)
		&domain.LessonModel{},

		// 5. Bảng tiến độ (phụ thuộc bảng chi tiết & user)
		&domain.LessonProgressModel{},
	); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	DB = db
	log.Println("✅ Database connected and migrated")

	// THAY ĐỔI: Seed dữ liệu cho Course Service
	seedCourseData(db)
}

// seedCourseData dùng để điền các giá trị mặc định cho các bảng lookup
func seedCourseData(db *gorm.DB) {
	// Seed Lesson Types
	types := []domain.LessonTypeModel{
		{Type: "video"},
		{Type: "text"},
	}
	for _, t := range types {
		// FirstOrCreate sẽ kiểm tra nếu tồn tại thì không tạo
		db.FirstOrCreate(&t, domain.LessonTypeModel{Type: t.Type})
	}

	log.Println("✅ Seeded default course lookup data")
}