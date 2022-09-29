package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/sboehler/knut/server/proto"
)

// Test with:
// grpcurl --plaintext -d '{"name":"foobar"}'  localhost:7777 knut.service.KnutService/Hello

// Run runs the GRPC server.
func Run() error {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterKnutServiceServer(grpcServer, new(Server))
	reflection.Register(grpcServer)

	wrappedGrpc := grpcweb.WrapServer(grpcServer)
	f := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if wrappedGrpc.IsGrpcWebRequest(req) {
			wrappedGrpc.ServeHTTP(resp, req)
			return
		}
		// Fall back to other servers.
		http.DefaultServeMux.ServeHTTP(resp, req)
	})

	return http.ListenAndServe("localhost:7777", f)
}

type Server struct {
	pb.UnimplementedKnutServiceServer
}

var _ pb.KnutServiceServer = (*Server)(nil)

func (srv *Server) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Greeting: fmt.Sprintf("Hello, %s", req.Name)}, nil
}
