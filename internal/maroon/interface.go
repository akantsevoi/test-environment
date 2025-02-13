package maroon

import (
	"context"

	"github.com/akantsevoi/test-environment/internal/p2p"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ETCD interface {
	Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error)
}

type DistTransport interface {
	DistributeTx(m p2p.Transaction)
}

type Application interface {
	Run(isLeaderCh <-chan bool, distributedTxCh <-chan p2p.TransactionDistributed, etcdWatchCh clientv3.WatchChan, stopCh <-chan struct{})
	AddOp(op Operation)
}

type OperationType int64

const (
	PrintTimestamp OperationType = iota
)

type Operation struct {
	OpType OperationType
	Value  string
}
