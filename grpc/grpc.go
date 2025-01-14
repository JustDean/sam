package grpc

import (
	context "context"
	"log"
	"net"

	"github.com/JustDean/sam/pkg/auth"
	grpc_base "google.golang.org/grpc"
)

func SetServer(c Config, am *auth.AuthManager) (*Server, error) {
	lis, err := net.Listen("tcp", c.url())
	if err != nil {
		return nil, err
	}
	s := grpc_base.NewServer()
	server := &Server{
		l:  lis,
		s:  s,
		am: am,
	}
	RegisterSamServer(s, server)
	return server, nil
}

type Server struct {
	UnimplementedSamServer
	l  net.Listener
	s  *grpc_base.Server
	am *auth.AuthManager
}

func (s *Server) Run(ctx context.Context) {
	log.Printf("Starting gRPC Server on %s", s.l.Addr())
	go func() {
		s.s.Serve(s.l)
	}()
	<-ctx.Done()
	log.Println("Stopping gRPC Server")
	s.s.GracefulStop()
	log.Println("gRPC Server is stopped")
}
