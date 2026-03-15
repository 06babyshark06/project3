package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/domain"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	"github.com/06babyshark06/JQKStudy/shared/env"
	pbUser "github.com/06babyshark06/JQKStudy/shared/proto/user"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type KafkaConsumer struct {
	consumer   *kafka.Consumer
	service    domain.ExamService
	userConn   *grpc.ClientConn
	userClient pbUser.UserServiceClient
}

func NewKafkaConsumer(service domain.ExamService) (*KafkaConsumer, error) {
	bootstrapServer := env.GetString("KAFKA_BOOTSTRAP_SERVER", "")
	apiKey := env.GetString("KAFKA_API_KEY", "")
	apiSecret := env.GetString("KAFKA_API_SECRET", "")
	consumerGroup := env.GetString("KAFKA_CONSUMER_GROUP", "exam-service-group")

	if bootstrapServer == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("cấu hình Kafka bị thiếu")
	}

	config := &kafka.ConfigMap{
		"bootstrap.servers": bootstrapServer,
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"sasl.username":     apiKey,
		"sasl.password":     apiSecret,
		"group.id":          consumerGroup,
		"auto.offset.reset": "earliest",
	}

	c, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, err
	}

	userSvcAddr := env.GetString("USER_GRPC_ADDR", "user-service:9001")
	conn, err := grpc.Dial(userSvcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Cảnh báo: Không thể kết nối tới user-service: %v", err)
	}

	var userClient pbUser.UserServiceClient
	if conn != nil {
		userClient = pbUser.NewUserServiceClient(conn)
	}

	kc := &KafkaConsumer{
		consumer:   c,
		service:    service,
		userConn:   conn,
		userClient: userClient,
	}

	return kc, nil
}

func (kc *KafkaConsumer) Start() {
	err := kc.consumer.SubscribeTopics([]string{"exam_assigned"}, nil)
	if err != nil {
		log.Printf("Lỗi subscribe topic: %v", err)
		return
	}

	log.Println("✅ Kafka Consumer đã khởi động (Exam Service)")

	go func() {
		for {
			msg, err := kc.consumer.ReadMessage(-1)
			if err == nil {
				if *msg.TopicPartition.Topic == "exam_assigned" {
					var event contracts.ExamAssignedEvent
					if err := json.Unmarshal(msg.Value, &event); err == nil {
						kc.handleExamAssigned(event)
					}
				}
			} else {
				log.Printf("Lỗi đọc Kafka: %v (%v)\n", err, msg)
			}
		}
	}()
}

func (kc *KafkaConsumer) handleExamAssigned(event contracts.ExamAssignedEvent) {
	if kc.userClient == nil {
		log.Println("Không thể xử lý exam_assigned do userClient chưa kết nối")
		return
	}

	ctx := context.Background()
	resp, err := kc.userClient.GetClassDetails(ctx, &pbUser.GetClassDetailsRequest{ClassId: event.ClassID})
	if err != nil {
		log.Printf("Lỗi lấy chi tiết lớp học từ user-service: %v", err)
		return
	}

	var studentIDs []int64
	for _, m := range resp.Members {
		if m.Role == "student" {
			studentIDs = append(studentIDs, m.UserId)
		}
	}

	if len(studentIDs) > 0 {
		err := kc.service.GeneratePersonalizedExamForStudents(ctx, event.ExamID, studentIDs)
		if err != nil {
			log.Printf("Lỗi tạo đề personalized: %v", err)
		} else {
			log.Printf("Đã tạo đề Personalized cho %d học sinh của lớp %d", len(studentIDs), event.ClassID)
		}
	}
}

func (kc *KafkaConsumer) Close() {
	if kc.userConn != nil {
		kc.userConn.Close()
	}
	kc.consumer.Close()
}
