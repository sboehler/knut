package server

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/sboehler/knut/server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Test with:
// grpcurl --plaintext -d '{"name":"foobar"}'  localhost:7777 knut.service.KnutService/Hello

// Run runs the GRPC server.
func Run() error {
	lis, err := net.Listen("tcp", "localhost:7777")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterKnutServiceServer(grpcServer, new(Server))
	reflection.Register(grpcServer)
	return grpcServer.Serve(lis)
}

type Server struct {
	pb.UnimplementedKnutServiceServer
}

var _ pb.KnutServiceServer = (*Server)(nil)

func (srv *Server) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Greeting: fmt.Sprintf("Hello, %s", req.Name)}, nil
}
