package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	database "github.com/06babyshark06/JQKStudy/services/exam-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/infrastructure/events"
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/infrastructure/grpc"
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/infrastructure/repository"
	"github.com/06babyshark06/JQKStudy/services/exam-service/internal/service"
	"github.com/06babyshark06/JQKStudy/shared/env"
	grpcserver "google.golang.org/grpc"
)

func main() {
	addr := env.GetString("EXAM_GRPC_ADDR", ":9002")

	database.Connect()
	database.InitRedis()
	if database.RedisClient != nil {
		defer database.RedisClient.Close()
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	repo := repository.NewExamRepository()
	kafkaProducer, err := events.NewKafkaProducer()
	if err != nil {
		log.Fatalf("Không thể khởi tạo Kafka Producer: %v", err)
	}
	defer kafkaProducer.Close()

	service := service.NewExamService(repo, kafkaProducer)

	kafkaConsumer, err := events.NewKafkaConsumer(service)
	if err != nil {
		log.Printf("Cảnh báo: Không thể khởi tạo Kafka Consumer: %v", err)
	} else {
		kafkaConsumer.Start()
		defer kafkaConsumer.Close()
	}

	grpcServer := grpcserver.NewServer()
	grpc.NewGRPCHandler(grpcServer, service)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 gRPC server is running on %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	sig := <-sigChan
	log.Printf("Received signal: %v. Shutting down gracefully...", sig)

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("✅ Server stopped gracefully.")
	case <-time.After(5 * time.Second):
		log.Println("⏰ Timeout reached. Force stopping server.")
		grpcServer.Stop()
	}
}
