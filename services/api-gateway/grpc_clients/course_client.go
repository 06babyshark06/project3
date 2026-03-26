package grpcclients

import (
	"github.com/06babyshark06/JQKStudy/shared/env"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/course"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type CourseServiceClient struct {
	Client pb.CourseServiceClient
	conn   *grpc.ClientConn
}

func NewCourseServiceClient() (*CourseServiceClient, error) {
	courseServiceURL := env.GetString("COURSE_SERVICE_URL", "course-service:9003")
	conn, err := grpc.NewClient("dns:///"+courseServiceURL, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return &CourseServiceClient{
		Client: pb.NewCourseServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *CourseServiceClient) Close() error {
	return c.conn.Close()
}
