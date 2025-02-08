package p2p

import (
	maroonv1 "github.com/akantsevoi/test-environment/gen/proto/maroon/p2p/v1"
	"google.golang.org/grpc"
)

type serv struct {
	maroonv1.UnimplementedP2PServiceServer

	// TODO: looks a bit like a hack. don't like it
	grpc *grpc.Server

	port    string
	dnsName string

	// messages that are coming from the network
	inCh chan Message

	// messages that we're going to send to some node
	outCh chan Message
}

func New(dnsName string, port string) Transport {
	return &serv{
		port:    port,
		dnsName: dnsName,
		inCh:    make(chan Message),
		outCh:   make(chan Message),
	}
}

func (s *serv) AddToQueue(m Message) {
	go func() {
		s.outCh <- m
	}()
}

func (s *serv) UpdateHosts(newHosts []string) {}
