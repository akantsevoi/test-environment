package p2p

import (
	"context"
	"fmt"
	"log"
	"sync"

	maroonv1 "github.com/akantsevoi/test-environment/gen/proto/maroon/p2p/v1"
	"github.com/akantsevoi/test-environment/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	// key - hostname:port
	clients   map[string]hostInfo
	clientsMu sync.RWMutex
}

type hostInfo struct {
	client     maroonv1.P2PServiceClient
	connection *grpc.ClientConn
}

func New(dnsName string, port string) (Transport, chan Message) {
	inCh := make(chan Message)
	return &serv{
		port:    port,
		dnsName: dnsName,
		inCh:    inCh,
		outCh:   make(chan Message),
	}, inCh
}

func (s *serv) AddToQueue(m Message) {
	go func() {
		s.outCh <- m
	}()
}

// for new hosts - will establish a new connection
// for removed hosts - will close the connection
// for unchanged - will do nothing
func (s *serv) UpdateHosts(newHosts []string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if s.clients == nil {
		s.clients = make(map[string]hostInfo)
	}

	currentHosts := make(map[string]bool)
	for host := range s.clients {
		currentHosts[host] = true
	}

	for _, host := range newHosts {
		if _, exists := s.clients[host]; !exists {
			conn, err := grpc.NewClient(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				// TODO: proper error handling
				continue
			}
			s.clients[host] = hostInfo{
				maroonv1.NewP2PServiceClient(conn),
				conn,
			}
		}
		delete(currentHosts, host)
	}

	// Remove clients that are no longer in the new hosts list
	for host := range currentHosts {
		if err := s.clients[host].connection.Close(); err != nil {
			logger.Errorf(logger.Network, "failed to close peer connection: %v: %v", host, err)
		}
		delete(s.clients, host)
	}
}

func (s *serv) serveOutboundMessageQueue() {
	for m := range s.outCh {

		ctx := context.TODO()

		for _, client := range s.clientsForDestination(m.Destination) {
			// TODO: herak-herak and run unit tests to confirm
			// that it works as expected
			// I'll do smth later to wrap it nicer
			if len(m.AddTxData) > 0 {
				resp, err := client.AddTx(ctx, &maroonv1.AddTxRequest{
					FromNode: fmt.Sprintf("%v:%v", s.dnsName, s.port),
					Payload:  m.AddTxData,
				})
				if err != nil {
					logger.Errorf(logger.Network, "failed to send addTX message: %v", err)
				}
				log.Println("resp", resp, "err", err)
			} else {
				resp, err := client.AckBatch(ctx, &maroonv1.AckBatchRequest{
					FromNode: fmt.Sprintf("%v:%v", s.dnsName, s.port),
					Hash:     m.AckData,
				})
				if err != nil {
					logger.Errorf(logger.Network, "failed to send ackBatch message: %v", err)
				}
				log.Println("resp", resp, "err", err)
			}
		}
	}
}

func (s *serv) clientsForDestination(dest string) []maroonv1.P2PServiceClient {
	// TODO: I don't like that lock
	// should replace it with smth else later
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	if len(dest) == 0 {
		var clients []maroonv1.P2PServiceClient
		for _, h := range s.clients {
			clients = append(clients, h.client)
		}
		return clients
	} else {
		host, ok := s.clients[dest]
		if !ok {
			logger.Errorf(logger.Network, "no such peer %v", dest)
			return nil
		}
		return []maroonv1.P2PServiceClient{
			host.client,
		}
	}
}
