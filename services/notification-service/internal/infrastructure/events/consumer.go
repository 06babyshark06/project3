package events

import (
	"context"
	"log"
	"time"

	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/env"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaConsumer struct {
	consumer *kafka.Consumer
	service  domain.NotificationService
}

func NewKafkaConsumer(service domain.NotificationService) (*KafkaConsumer, error) {
	bootstrapServer := env.GetString("KAFKA_BOOTSTRAP_SERVER", "")
	apiKey := env.GetString("KAFKA_API_KEY", "")
	apiSecret := env.GetString("KAFKA_API_SECRET", "")

	if bootstrapServer == "" || apiKey == "" || apiSecret == "" {
		log.Fatal("❌ Cấu hình Kafka (Confluent) bị thiếu trong .env")
	}

	config := &kafka.ConfigMap{
		"bootstrap.servers": bootstrapServer,
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"sasl.username":     apiKey,
		"sasl.password":     apiSecret,
		"group.id":          "notification_service_group_v1", 
		"auto.offset.reset": "earliest", 
	}

	c, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, err
	}

	topics := []string{
		"user_events",
		"exam_events",
		"course_events",
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

func (kc *KafkaConsumer) StartConsuming(ctx context.Context) {
	log.Println("... Kafka Consumer đang lắng nghe ...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Dừng Kafka consumer...")
			kc.consumer.Close()
			return
		default:
			msg, err := kc.consumer.ReadMessage(1 * time.Second)
		
			if err == nil {
				topic := *msg.TopicPartition.Topic
				log.Printf("Nhận được message từ topic: %s\n", topic)

				kc.routeEvent(ctx, topic, msg.Value)
				
			} else if !err.(kafka.Error).IsTimeout() {
				log.Printf("Lỗi Kafka Consumer: %v (%v)\n", err, msg)
			}
		}
	}
}

func (kc *KafkaConsumer) routeEvent(ctx context.Context, topic string, value []byte) {
	var err error
	
	switch topic {
	case "user_events":
		err = kc.service.HandleUserRegisteredEvent(ctx, value)
	
	case "exam_events":
		err = kc.service.HandleExamSubmittedEvent(ctx, value)
		
	case "course_events":
		err = kc.service.HandleCourseEnrolledEvent(ctx, value)

	default:
		log.Printf("Không tìm thấy hàm xử lý cho topic: %s", topic)
	}

	if err != nil {
		log.Printf("Lỗi khi xử lý sự kiện từ topic %s: %v", topic, err)
	}
}