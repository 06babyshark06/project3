package grpcclients

import (
	"log"

	"github.com/06babyshark06/JQKStudy/shared/env"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/ai"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AIServiceClient struct {
	Client pb.AIServiceClient
	conn   *grpc.ClientConn
}

func NewAIServiceClient() (*AIServiceClient, error) {
	aiAddr := env.GetString("AI_GRPC_ADDR", "ai-service:9005")
	conn, err := grpc.NewClient("dns:///"+aiAddr, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("Failed to connect to ai service at %s: %v", aiAddr, err)
		return nil, err
	}

	client := pb.NewAIServiceClient(conn)

	return &AIServiceClient{
		Client: client,
		conn:   conn,
	}, nil
}

func (c *AIServiceClient) Close() error {
	return c.conn.Close()
}
