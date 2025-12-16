package database

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/06babyshark06/JQKStudy/services/user-service/internal/domain"
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
		&domain.Role{},
		&domain.UserModel{},
		&domain.ClassModel{},
		&domain.ClassMemberModel{},
	); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	DB = db
	log.Println("✅ Database connected and migrated")

	seedRoles(db)
	seedUsers(db)
}

func seedRoles(db *gorm.DB) {
	defaultRoles := []domain.Role{
		{Name: "student"},
		{Name: "instructor"},
		{Name: "admin"},
	}

	for _, r := range defaultRoles {
		var count int64
		db.Model(&domain.Role{}).Where("name = ?", r.Name).Count(&count)
		if count == 0 {
			db.Create(&r)
		}
	}
	log.Println("✅ Seeded default roles")
}

func seedUsers(db *gorm.DB) {
	var studentRole domain.Role
	db.Where("name = ?", "student").First(&studentRole)

	var instructorRole domain.Role
	db.Where("name = ?", "instructor").First(&instructorRole)

	var adminRole domain.Role
	db.Where("name = ?", "admin").First(&adminRole)

	if studentRole.Id == 0 || instructorRole.Id == 0 || adminRole.Id == 0 {
		log.Println("⚠️ Không tìm thấy vai trò (roles), bỏ qua seed users.")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("❌ Không thể băm mật khẩu: %v", err)
		return
	}
	hashedPasswordStr := string(hashedPassword)

	defaultUsers := []domain.UserModel{
		{
			FullName:  "Admin JQK",
			Email:     "admin@jqk.com",
			Password:  hashedPasswordStr,
			RoleId:    adminRole.Id,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		{
			FullName:  "Instructor JQK",
			Email:     "instructor@jqk.com",
			Password:  hashedPasswordStr,
			RoleId:    instructorRole.Id,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		{
			FullName:  "Student JQK",
			Email:     "student@jqk.com",
			Password:  hashedPasswordStr,
			RoleId:    studentRole.Id,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}

	for _, u := range defaultUsers {
		var count int64
		db.Model(&domain.UserModel{}).Where("email = ?", u.Email).Count(&count)
		if count == 0 {
			db.Create(&u)
		}
	}

	log.Println("🌱 Seeding 30 extra students...")
	
	for i := 1; i <= 30; i++ {
		email := fmt.Sprintf("student%d@jqk.com", i)
		
		var count int64
		db.Model(&domain.UserModel{}).Where("email = ?", email).Count(&count)
		
		if count == 0 {
			newStudent := domain.UserModel{
				FullName:  fmt.Sprintf("Học Sinh Số %d", i),
				Email:     email,
				Password:  hashedPasswordStr,
				RoleId:    studentRole.Id,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			db.Create(&newStudent)
		}
	}
	
	log.Println("✅ Seeded default users & 30 extra students (pass: 123456)")
}