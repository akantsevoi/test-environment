package p2p

import (
	"context"

	maroonv1 "github.com/akantsevoi/test-environment/gen/proto/maroon/p2p/v1"
	"github.com/akantsevoi/test-environment/pkg/logger"
)

func (s *serv) AddTx(_ context.Context, req *maroonv1.AddTxRequest) (*maroonv1.AddTxResponse, error) {
	logger.Infof(logger.Network, "got message addtx: %v", req.String())

	go func() {
		s.inCh <- Message{
			Destination: "incoming message", // ?req.FromNode,
			AddTxData:   req.Payload,
		}
	}()

	// TODO: think more on protocol. What does success-true mean in that case?
	// just message accepted, but do we need success then?
	return &maroonv1.AddTxResponse{Success: true}, nil
}

func (s *serv) AckBatch(_ context.Context, req *maroonv1.AckBatchRequest) (*maroonv1.AckBatchResponse, error) {
	logger.Infof(logger.Network, "got message ackbatch: %v", req.String())

	go func() {
		s.inCh <- Message{
			Destination: "incoming message", // ?req.FromNode,
			AckData:     req.Hash,
		}
	}()

	// TODO: think more on protocol. What does success-true mean in that case?
	// just message accepted, but do we need success then?
	return &maroonv1.AckBatchResponse{Success: true}, nil
}
