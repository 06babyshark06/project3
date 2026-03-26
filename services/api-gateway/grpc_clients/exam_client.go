package grpcclients

import (
	"github.com/06babyshark06/JQKStudy/shared/env"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/exam"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ExamServiceClient struct {
	Client pb.ExamServiceClient
	conn   *grpc.ClientConn
}

func NewExamServiceClient() (*ExamServiceClient, error) {
	examServiceURL := env.GetString("EXAM_SERVICE_URL", "exam-service:9002")
	conn, err := grpc.NewClient("dns:///"+examServiceURL, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return &ExamServiceClient{
		Client: pb.NewExamServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *ExamServiceClient) Close() error {
	return c.conn.Close()
}
