package events

import (
	"log"

	"github.com/06babyshark06/JQKStudy/shared/env"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/06babyshark06/JQKStudy/services/user-service/internal/domain"
)

// kafkaProducer triển khai interface
type kafkaProducer struct {
	producer *kafka.Producer
}

// NewKafkaProducer tạo một producer mới và kết nối tới Confluent
func NewKafkaProducer() (domain.EventProducer, error) {
	bootstrapServer := env.GetString("KAFKA_BOOTSTRAP_SERVER", "pkc-921jm.us-east-2.aws.confluent.cloud:9092")
	apiKey := env.GetString("KAFKA_API_KEY", "4Q7GDTN7GVOXGSWJ")
	apiSecret := env.GetString("KAFKA_API_SECRET", "cfltONrMheJciXzdki9Qo1oq7R89eMgR4mM2/nCbc5jdw/iIg2PIwYi34X2ZLBDg")

	if bootstrapServer == "" || apiKey == "" || apiSecret == "" {
		log.Fatal("❌ Cấu hình Kafka (Confluent) bị thiếu trong .env của User Service")
	}

	config := &kafka.ConfigMap{
		"bootstrap.servers": bootstrapServer,
		"security.protocol": "SASL_SSL",
		"sasl.mechanisms":   "PLAIN",
		"sasl.username":     apiKey,
		"sasl.password":     apiSecret,
		
		// Cấu hình quan trọng cho producer
		"acks": "all", // Chờ tất cả broker xác nhận (an toàn nhất)
	}

	p, err := kafka.NewProducer(config)
	if err != nil {
		return nil, err
	}

	log.Println("✅ Kafka Producer đã kết nối (User Service)")

	// QUAN TRỌNG: Chạy một goroutine để lắng nghe Delivery Reports
	// Kafka produce là bất đồng bộ, đây là cách duy nhất để biết message đã GỬI THÀNH CÔNG
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("LỖI: Gửi message thất bại: %v\n", ev.TopicPartition.Error)
				} else {
					log.Printf("Đã gửi message thành công đến topic %s [%d] at offset %v\n",
						*ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
				}
			}
		}
	}()

	return &kafkaProducer{producer: p}, nil
}

// Produce gửi message (bất đồng bộ)
func (kp *kafkaProducer) Produce(topic string, key []byte, message []byte) error {
	// Gửi message vào hàng đợi nội bộ của producer
	// Goroutine ở trên sẽ xử lý kết quả
	return kp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            key,
		Value:          message,
	}, nil) // nil deliveryChan để dùng chung
}

// Close dọn dẹp producer
func (kp *kafkaProducer) Close() {
	// Chờ (flush) tất cả các message đang chờ gửi
	kp.producer.Flush(5 * 1000) // 5 giây timeout
	kp.producer.Close()
}