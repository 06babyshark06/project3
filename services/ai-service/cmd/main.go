package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/06babyshark06/JQKStudy/services/ai-service/internal/databases"
	"github.com/06babyshark06/JQKStudy/services/ai-service/internal/server"
	"github.com/06babyshark06/JQKStudy/shared/env"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/ai"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := env.GetString("AI_GRPC_ADDR", ":9005")
	geminiApiKey := env.GetString("GEMINI_API_KEY", "")

	database.InitRedis()

	if geminiApiKey == "" {
		log.Println("WARNING: GEMINI_API_KEY is not set.")
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", port, err)
	}

	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(50*1024*1024),
		grpc.MaxSendMsgSize(50*1024*1024),
	)
	aiServer := server.NewAIServiceServer(geminiApiKey)
	pb.RegisterAIServiceServer(s, aiServer)

	// Register reflection service on gRPC server for evans / postman testing
	reflection.Register(s)

	go func() {
		fmt.Printf("🚀 AI gRPC Server is listening on %s\n", port)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gRPC server...")

	// Graceful stop
	s.GracefulStop()
	log.Println("Server exiting")
}
