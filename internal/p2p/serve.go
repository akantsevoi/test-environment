package p2p

import (
	"fmt"
	"net"

	maroonv1 "github.com/akantsevoi/test-environment/gen/proto/maroon/p2p/v1"
	"google.golang.org/grpc"
)

// Blocking function
func (s *serv) Start() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", s.port))
	if err != nil {
		panic(err)
	}
	grpcServ := grpc.NewServer()
	s.grpc = grpcServ

	maroonv1.RegisterP2PServiceServer(grpcServ, s)

	go s.serveOutboundMessageQueue()

	if err := grpcServ.Serve(lis); err != nil {
		panic(err)
	}
}

// Graceful stop
func (s *serv) Stop() {
	if s.grpc == nil {
		return
	}
	s.grpc.GracefulStop()
	s.grpc = nil
}
