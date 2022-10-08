package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/sboehler/knut/web"

	pb "github.com/sboehler/knut/server/proto"
)

// Test with:
// grpcurl --plaintext -d '{"name":"foobar"}'  localhost:7777 knut.service.KnutService/Hello

// NewServer runs the GRPC server.
func NewServer(address string) error {
	srv := new(Server)
	grpcServer := grpc.NewServer()
	pb.RegisterKnutServiceServer(grpcServer, srv)
	reflection.Register(grpcServer)
	grpcWebServer := grpcweb.WrapServer(grpcServer)
	assets, err := web.Files()
	if err != nil {
		return fmt.Errorf("web.Files(): %w", err)
	}
	f := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if grpcWebServer.IsGrpcWebRequest(req) {
			grpcWebServer.ServeHTTP(resp, req)
		} else {
			assets.ServeHTTP(resp, req)
		}
	})
	return http.ListenAndServe(address, f)
}

type Server struct {
	pb.UnimplementedKnutServiceServer
}

var _ pb.KnutServiceServer = (*Server)(nil)

func (srv *Server) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Greeting: fmt.Sprintf("Hello, %s", req.Name)}, nil
}
