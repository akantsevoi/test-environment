package p2p

import (
	"context"
	"fmt"
	"log"
	"net"

	maroonv1 "github.com/akantsevoi/test-environment/gen/proto/maroon/p2p/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	go func() {
		for m := range s.outCh {
			conn, err := grpc.NewClient(m.Destination, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				panic("")
			}

			client := maroonv1.NewP2PServiceClient(conn)
			ctx := context.Background()

			resp, err := client.AddTx(ctx, &maroonv1.AddTxRequest{
				FromNode: fmt.Sprintf("%v:%v", s.dnsName, s.port),
				Payload:  []byte("hello hardcoded"),
			})
			log.Println("resp", resp, "err", err)

			conn.Close()
		}
	}()

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
