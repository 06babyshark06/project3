package events

import (
	"context"
	"log"
	"time"

	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/env"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaConsumer lắng nghe các sự kiện và gọi service
type KafkaConsumer struct {
	consumer *kafka.Consumer
	service  domain.NotificationService
}

// NewKafkaConsumer khởi tạo và kết nối tới Confluent Cloud
func NewKafkaConsumer(service domain.NotificationService) (*KafkaConsumer, error) {
	// Lấy thông tin cấu hình từ .env
	bootstrapServer := env.GetString("KAFKA_BOOTSTRAP_SERVER", "")
	apiKey := env.GetString("KAFKA_API_KEY", "")
	apiSecret := env.GetString("KAFKA_API_SECRET", "")

	if bootstrapServer == "" || apiKey == "" || apiSecret == "" {
		log.Fatal("❌ Cấu hình Kafka (Confluent) bị thiếu trong .env")
	}

	// 1. Cấu hình consumer
	config := &kafka.ConfigMap{
		"bootstrap.servers": bootstrapServer,
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"sasl.username":     apiKey,
		"sasl.password":     apiSecret,
		
		// Quan trọng: Tên consumer group
		// Kafka sẽ nhớ message cuối cùng mà group này đã đọc
		"group.id":          "notification_service_group_v1", 
		
		// Bắt đầu đọc từ message cũ nhất nếu đây là group mới
		"auto.offset.reset": "earliest", 
	}

	// 2. Tạo consumer
	c, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, err
	}

	// 3. Đăng ký (Subscribe) các topics mà service này quan tâm
	topics := []string{
		"user_events",     // Topic cho User Service
		"exam_events",     // Topic cho Exam Service
		"course_events",   // Topic cho Course Service
	}
	if err := c.SubscribeTopics(topics, nil); err != nil {
		return nil, err
	}

	log.Println("✅ Kafka Consumer đã kết nối và đăng ký topics:", topics)

	return &KafkaConsumer{
		consumer: c,
		service:  service,
	}, nil
}

// StartConsuming bắt đầu vòng lặp lắng nghe sự kiện
// Hàm này nên được chạy trong một goroutine
func (kc *KafkaConsumer) StartConsuming(ctx context.Context) {
	log.Println("... Kafka Consumer đang lắng nghe ...")

	for {
		select {
		case <-ctx.Done(): // Nếu context bị hủy (app tắt)
			log.Println("Dừng Kafka consumer...")
			kc.consumer.Close()
			return
		default:
			// 1. Đọc message, chờ tối đa 1 giây
			msg, err := kc.consumer.ReadMessage(1 * time.Second)
			
			// 2. Nếu không có lỗi (có message)
			if err == nil {
				// 3. Lấy tên topic và xử lý
				topic := *msg.TopicPartition.Topic
				log.Printf("Nhận được message từ topic: %s\n", topic)

				// 4. Định tuyến (Route) sự kiện đến service
				kc.routeEvent(ctx, topic, msg.Value)

				// (Quan trọng) Tự động commit offset nếu không có lỗi
				// confluent-kafka-go mặc định là auto-commit
				
			} else if !err.(kafka.Error).IsTimeout() {
				// Bỏ qua lỗi timeout (không có message mới)
				log.Printf("Lỗi Kafka Consumer: %v (%v)\n", err, msg)
			}
		}
	}
}

// routeEvent điều hướng message đến đúng hàm xử lý
func (kc *KafkaConsumer) routeEvent(ctx context.Context, topic string, value []byte) {
	var err error

	// Lấy tên sự kiện từ (ví dụ: "user_registered")
	// (Trong thực tế, bạn có thể lấy event_type từ message header)
	// Hiện tại, chúng ta giả định 1 topic = 1 loại sự kiện chính
	
	switch topic {
	case "user_events":
		// Giả sử topic này chỉ chứa sự kiện đăng ký
		err = kc.service.HandleUserRegisteredEvent(ctx, value)
	
	case "exam_events":
		// Giả sử topic này chỉ chứa sự kiện nộp bài
		err = kc.service.HandleExamSubmittedEvent(ctx, value)
		
	case "course_events":
		err = kc.service.HandleCourseEnrolledEvent(ctx, value)

	default:
		log.Printf("Không tìm thấy hàm xử lý cho topic: %s", topic)
	}

	if err != nil {
		log.Printf("Lỗi khi xử lý sự kiện từ topic %s: %v", topic, err)
		// (Trong thực tế, bạn sẽ implement retry hoặc gửi vào Dead Letter Queue)
	}
}