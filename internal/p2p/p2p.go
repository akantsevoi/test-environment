package p2p

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	maroonv1 "github.com/akantsevoi/test-environment/gen/proto/maroon/p2p/v1"
	"github.com/akantsevoi/test-environment/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type serv struct {
	maroonv1.UnimplementedP2PServiceServer

	grpc *grpc.Server

	// port where to spin a service
	port string

	// dnsName that other nodes can reach this service back
	// TODO: do I really need it?
	dnsName string

	toDistributeQueueCh chan Transaction
	distributedTxCh     chan TransactionDistributed

	// key - hostname:port
	// TODO: add info about regions as well
	clients   map[string]hostInfo
	clientsMu sync.RWMutex
}

type hostInfo struct {
	client     maroonv1.P2PServiceClient
	connection *grpc.ClientConn
}

// wanted to explicitly return transactionDistributed channel here
// to highlight uniqueness of ownership.
//   - so it will be not possible to get channel in many places and consume and block it
//   - makes sense?
func New(dnsName string, port string) (Transport, chan TransactionDistributed) {
	distributedCh := make(chan TransactionDistributed)
	return &serv{
		port:                port,
		dnsName:             dnsName,
		toDistributeQueueCh: make(chan Transaction),
		distributedTxCh:     distributedCh,
	}, distributedCh
}

func (s *serv) DistributeTx(m Transaction) {
	go func() {
		s.toDistributeQueueCh <- m
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
	for tx := range s.toDistributeQueueCh {

		ctx := context.TODO()
		// TODO: some algorithm on how to distribute
		// which nodes/regions/etc

		// TODO: that lock is not right thing here
		// I need to find a better way of replacing it
		s.clientsMu.RLock()
		var counterDistributed atomic.Int32
		for _, hostI := range s.clients {
			client := hostI.client
			go func() {
				resp, err := client.AddTx(ctx, &maroonv1.AddTxRequest{
					FromNode: fmt.Sprintf("%v:%v", s.dnsName, s.port),
					Id:       tx.ID,
					Payload:  tx.TxData,
				})
				if err != nil {
					logger.Errorf(logger.Network, "failed to send addTX message: %v", err)
				} else {
					log.Println(tx.ID, "resp", resp, "err", err)

					// TODO: that's a very naive way of checking that it was distributed
					// rethink it later and take regions into consideration
					counterDistributed.Add(1)
					if counterDistributed.Load() >= 2 {
						s.distributedTxCh <- TransactionDistributed{
							ID: tx.ID,
						}
						log.Println(tx.ID, "resp after put to distributed channel")
					}
				}
			}()
		}

		s.clientsMu.RUnlock()

	}
}
