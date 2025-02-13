package p2p

import (
	"context"
	"time"

	maroonv1 "github.com/akantsevoi/test-environment/gen/proto/maroon/p2p/v1"
	"github.com/akantsevoi/test-environment/pkg/logger"
)

func (s *serv) AddTx(_ context.Context, req *maroonv1.AddTxRequest) (*maroonv1.AddTxResponse, error) {
	logger.Infof(logger.Network, "got message addtx: %v", req.String())

	// TODO: imitate that we store transaction somewhere
	// implement passing this tx to application layer
	time.Sleep(30 * time.Millisecond)

	return &maroonv1.AddTxResponse{Acced: true}, nil
}
