package events

import (
	"log"

	"github.com/06babyshark06/JQKStudy/shared/env"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/domain"
)

type kafkaProducer struct {
	producer *kafka.Producer
}

func NewKafkaProducer() (domain.EventProducer, error) {
	bootstrapServer := env.GetString("KAFKA_BOOTSTRAP_SERVER", "localhost:9092")
	apiKey := env.GetString("KAFKA_API_KEY", "")
	apiSecret := env.GetString("KAFKA_API_SECRET", "")

	if bootstrapServer == "" {
		log.Fatal("❌ Cấu hình KAFKA_BOOTSTRAP_SERVER bị thiếu")
	}

	configMap := kafka.ConfigMap{
		"bootstrap.servers": bootstrapServer,
		"acks":              "all",
	}

	if apiKey != "" && apiSecret != "" {
		configMap["security.protocol"] = "SASL_SSL"
		configMap["sasl.mechanisms"] = "PLAIN"
		configMap["sasl.username"] = apiKey
		configMap["sasl.password"] = apiSecret
	}

	p, err := kafka.NewProducer(&configMap)
	if err != nil {
		return nil, err
	}

	log.Println("✅ Kafka Producer đã kết nối (Exam Service)")

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

func (kp *kafkaProducer) Produce(topic string, key []byte, message []byte) error {
	return kp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            key,
		Value:          message,
	}, nil)
}

func (kp *kafkaProducer) Close() {
	kp.producer.Flush(5 * 1000)
	kp.producer.Close()
}
