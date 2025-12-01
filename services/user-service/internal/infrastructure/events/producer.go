package events

import (
	"log"

	"github.com/06babyshark06/JQKStudy/shared/env"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/06babyshark06/JQKStudy/services/user-service/internal/domain"
)

type kafkaProducer struct {
	producer *kafka.Producer
}

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
		"acks": "all",
	}

	p, err := kafka.NewProducer(config)
	if err != nil {
		return nil, err
	}

	log.Println("✅ Kafka Producer đã kết nối (User Service)")

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

// Close dọn dẹp producer
func (kp *kafkaProducer) Close() {
	kp.producer.Flush(5 * 1000)
	kp.producer.Close()
}