package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/domain"
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
		&domain.ChannelTypeModel{},
		&domain.NotificationStatusModel{},
		&domain.NotificationTemplateModel{},
		&domain.NotificationModel{},
	); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	DB = db
	log.Println("✅ Database connected and migrated")

	seedNotificationData(db)
}

func seedNotificationData(db *gorm.DB) {
	defaultTypes := []domain.ChannelTypeModel{
		{Type: "email"},
		{Type: "sms"},
		{Type: "push"},
	}
	for _, t := range defaultTypes {
		// FirstOrCreate sẽ kiểm tra nếu tồn tại thì không tạo
		db.FirstOrCreate(&t, domain.ChannelTypeModel{Type: t.Type})
	}

	defaultStatuses := []domain.NotificationStatusModel{
		{Status: "pending"},
		{Status: "sent"},
		{Status: "failed"},
	}
	for _, s := range defaultStatuses {
		db.FirstOrCreate(&s, domain.NotificationStatusModel{Status: s.Status})
	}

	var emailChannel domain.ChannelTypeModel
	if err := db.Where("type = ?", "email").First(&emailChannel).Error; err != nil {
		log.Printf("⚠️ Không tìm thấy channel 'email' để tạo template: %v", err)
		return
	}

	const (
		containerStyle = `font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #ffffff; border-radius: 8px; border: 1px solid #e0e0e0;`
		headerStyle    = `text-align: center; padding-bottom: 20px; border-bottom: 2px solid #f0f0f0;`
		logoStyle      = `font-size: 24px; font-weight: bold; color: #E11D48; text-decoration: none;` // Màu Rose-600
		contentStyle   = `padding: 30px 0; color: #333333; line-height: 1.6; font-size: 16px;`
		buttonStyle    = `display: inline-block; padding: 12px 24px; background-color: #E11D48; color: #ffffff; text-decoration: none; border-radius: 5px; font-weight: bold; margin-top: 20px;`
		footerStyle    = `text-align: center; padding-top: 20px; border-top: 1px solid #f0f0f0; font-size: 12px; color: #888888;`
	)

	baseTemplate := fmt.Sprintf(`
        <div style="background-color: #f9fafb; padding: 40px 0;">
            <div style="%s">
                <div style="%s">
                    <a href="#" style="%s">JQK Study</a>
                </div>
                <div style="%s">
                    %%s 
                </div>
                <div style="%s">
                    <p>&copy; 2025 JQK Study. All rights reserved.</p>
                    <p>Học mọi lúc, mọi nơi, kiến tạo tương lai.</p>
                </div>
            </div>
        </div>
    `, containerStyle, headerStyle, logoStyle, contentStyle, footerStyle)

	userRegisteredBody := fmt.Sprintf(baseTemplate, `
        <h2 style="color: #111827; margin-top: 0;">Chào mừng gia nhập! 🎉</h2>
        <p>Xin chào <strong>%s</strong>,</p>
        <p>Cảm ơn bạn đã tin tưởng và lựa chọn <strong>JQK Study</strong>. Tài khoản của bạn đã được tạo thành công.</p>
        <p>Giờ đây, bạn có thể truy cập hàng ngàn khóa học chất lượng và bắt đầu hành trình chinh phục tri thức mới.</p>
        <div style="text-align: center;">
            <a href="http://localhost:3000/login" style="`+buttonStyle+`">Đăng Nhập Ngay</a>
        </div>
    `)

	examSubmittedBody := fmt.Sprintf(baseTemplate, `
        <h2 style="color: #111827; margin-top: 0;">Kết quả bài thi 📝</h2>
        <p>Chào <strong>%s</strong>,</p>
        <p>Hệ thống đã ghi nhận kết quả bài thi của bạn:</p>
        <div style="background-color: #fff1f2; border: 1px solid #fda4af; padding: 15px; border-radius: 6px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Bài thi:</strong> %s</p>
            <p style="margin: 5px 0; font-size: 18px;"><strong>Điểm số:</strong> <span style="color: #E11D48; font-weight: bold;">%.2f / 10</span></p>
        </div>
        <p>Hãy tiếp tục cố gắng nhé!</p>
        <div style="text-align: center;">
            <a href="http://localhost:3000/dashboard" style="`+buttonStyle+`">Xem Chi Tiết</a>
        </div>
    `)

	courseEnrolledBody := fmt.Sprintf(baseTemplate, `
        <h2 style="color: #111827; margin-top: 0;">Đăng ký thành công! 🎓</h2>
        <p>Xin chào <strong>%s</strong>,</p>
        <p>Chúc mừng bạn đã đăng ký thành công khóa học:</p>
        <h3 style="color: #E11D48; text-align: center; margin: 20px 0;">%s</h3>
        <p>Bạn có thể bắt đầu học ngay bây giờ. Chúc bạn có những giờ học thật hiệu quả!</p>
        <div style="text-align: center;">
            <a href="http://localhost:3000/my-courses" style="`+buttonStyle+`">Vào Học Ngay</a>
        </div>
    `)

	templates := []domain.NotificationTemplateModel{
        {
            Name:    "user_registered",
            TypeID:  emailChannel.Id,
            Subject: "Chào mừng bạn đến với JQK Study! 🚀",
            Body:    userRegisteredBody,
        },
        {
            Name:    "exam_submitted",
            TypeID:  emailChannel.Id,
            Subject: "Kết quả bài thi: %s",
            Body:    examSubmittedBody,
        },
        {
            Name:    "course_enrolled",
            TypeID:  emailChannel.Id,
            Subject: "Xác nhận đăng ký khóa học: %s",
            Body:    courseEnrolledBody,
        },
    }

	for _, t := range templates {
		var existing domain.NotificationTemplateModel
		if err := db.Where("name = ?", t.Name).First(&existing).Error; err != nil {
			db.Create(&t)
			log.Printf("✅ Đã tạo template: %s", t.Name)
		}
	}

	log.Println("✅ Seeded default notification data (Channels, Statuses, Templates)")
}
