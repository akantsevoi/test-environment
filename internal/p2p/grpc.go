package p2p

import (
	"context"

	maroonv1 "github.com/akantsevoi/test-environment/gen/proto/maroon/p2p/v1"
	"github.com/akantsevoi/test-environment/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *serv) AddTx(ctx context.Context, req *maroonv1.AddTxRequest) (*maroonv1.AddTxResponse, error) {
	logger.Infof(logger.Network, "got message %v", req.String())
	return nil, status.Errorf(codes.Unimplemented, "method AddTx not implemented")
}

func (s *serv) AckBatch(context.Context, *maroonv1.AckBatchRequest) (*maroonv1.AckBatchResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddTx not implemented")
}
