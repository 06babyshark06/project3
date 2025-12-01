package grpcclients

import (
	"time"

	"github.com/06babyshark06/JQKStudy/shared/env"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserServiceClient struct {
	Client pb.UserServiceClient
	conn   *grpc.ClientConn
}

func NewUserServiceClient() (*UserServiceClient, error) {
	userServiceURL := env.GetString("USER_SERVICE_URL", "user-service:9001")
	conn, err := grpc.NewClient(userServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}

	return &UserServiceClient{
		Client: pb.NewUserServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *UserServiceClient) Close() error {
	return c.conn.Close()
}
