package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	database "github.com/06babyshark06/JQKStudy/services/notification-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/notification-service/external/email"
	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/infrastructure/events"
	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/infrastructure/repository"
	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/service"
)

func main() {
	database.Connect()

	emailProvider, err := email.NewMailtrapProvider()
	if err != nil {
		log.Fatalf("❌ Không thể khởi tạo Email Provider: %v", err)
	}

	repo := repository.NewNotificationRepository()
	service := service.NewNotificationService(repo, emailProvider)

	kafkaConsumer, err := events.NewKafkaConsumer(service)
	if err != nil {
		log.Fatalf("❌ Không thể khởi tạo Kafka Consumer: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	stopped := make(chan struct{})

	go func() {
		log.Println("🚀 Kafka consumer is running...")
		kafkaConsumer.StartConsuming(ctx) 
		close(stopped) 
	}()

	sig := <-sigChan
	log.Printf("Received signal: %v. Shutting down gracefully...", sig)

	cancel()

	select {
	case <-stopped:
		log.Println("✅ Consumer stopped gracefully.")
	case <-time.After(5 * time.Second):
		log.Println("⏰ Timeout reached. Force exiting.")
	}
}