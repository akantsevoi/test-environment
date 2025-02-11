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
