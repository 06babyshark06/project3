package main

import (
	"context" // Cần context để hủy consumer
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Cập nhật các đường dẫn cho notification-service
	database "github.com/06babyshark06/JQKStudy/services/notification-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/notification-service/external/email"
	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/infrastructure/events"
	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/infrastructure/repository"
	"github.com/06babyshark06/JQKStudy/services/notification-service/internal/service"
)

func main() {
	// (Giả sử env đã được load)
	// env.LoadEnv() 
	
	database.Connect()

	// 1. Khởi tạo tất cả dependencies
	emailProvider, err := email.NewMailtrapProvider()
	if err != nil {
		log.Fatalf("❌ Không thể khởi tạo Email Provider: %v", err)
	}

	repo := repository.NewNotificationRepository()
	service := service.NewNotificationService(repo, emailProvider)

	// 2. Khởi tạo "Server" (chính là Kafka Consumer)
	kafkaConsumer, err := events.NewKafkaConsumer(service)
	if err != nil {
		log.Fatalf("❌ Không thể khởi tạo Kafka Consumer: %v", err)
	}

	// 3. Channel nhận tín hiệu hệ thống (giống hệt template của bạn)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 4. Tạo context để có thể hủy goroutine
	ctx, cancel := context.WithCancel(context.Background())
	
	// Channel để báo consumer đã dừng
	stopped := make(chan struct{})

	// 5. Chạy Consumer trong goroutine riêng
	go func() {
		log.Println("🚀 Kafka consumer is running...")
		// Hàm này sẽ block cho đến khi context bị hủy
		kafkaConsumer.StartConsuming(ctx) 
		// Sau khi StartConsuming kết thúc, nó báo hiệu
		close(stopped) 
	}()

	// 6. Đợi tín hiệu dừng (giống hệt template của bạn)
	sig := <-sigChan
	log.Printf("Received signal: %v. Shutting down gracefully...", sig)

	// 7. Graceful stop (thay vì grpcServer.GracefulStop(), chúng ta gọi cancel())
	cancel()

	// 8. Đợi consumer dừng, với timeout (giống hệt template của bạn)
	select {
	case <-stopped:
		log.Println("✅ Consumer stopped gracefully.")
	case <-time.After(5 * time.Second):
		// Nếu consumer không dừng sau 5s, app sẽ tự thoát
		log.Println("⏰ Timeout reached. Force exiting.")
	}
}