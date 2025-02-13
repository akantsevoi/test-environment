package maroon

import (
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/akantsevoi/test-environment/internal/p2p"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type servMock struct {
	distr func(tx p2p.Transaction)
}

func (s *servMock) DistributeTx(tx p2p.Transaction) {
	s.distr(tx)
}

type etcdMock struct {
	put func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error)
}

func (e *etcdMock) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	return e.put(ctx, key, val, opts...)
}

func TestCheckProofSentToETCD(t *testing.T) {
	var etcdValueRequest string
	etcd := &etcdMock{
		put: func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
			log.Println("etcd", key, val)
			etcdValueRequest = val
			return &clientv3.PutResponse{}, nil
		},
	}

	opDistributedCh := make(chan p2p.TransactionDistributed)

	serv := &servMock{
		distr: func(tx p2p.Transaction) {
			log.Println("server imitation", tx)
			go func() {
				time.Sleep(10 * time.Millisecond)
				opDistributedCh <- p2p.TransactionDistributed{
					ID: tx.ID,
				}
			}()
		},
	}

	isLeaderCh := make(chan bool)
	etcdWatchCh := make(clientv3.WatchChan)
	stopCh := make(chan struct{})

	app := New(etcd, serv)
	go app.Run(isLeaderCh, opDistributedCh, etcdWatchCh, stopCh)
	isLeaderCh <- true

	op1, op2, op3 := Operation{OpType: PrintTimestamp, Value: "1"}, Operation{OpType: PrintTimestamp, Value: "2"}, Operation{OpType: PrintTimestamp, Value: "3"}

	app.AddOp(op1)
	app.AddOp(op2)
	app.AddOp(op3)

	time.Sleep(50 * time.Millisecond)
	stopCh <- struct{}{}

	time.Sleep(50 * time.Millisecond)
	require.ElementsMatch(t,
		strings.Split(etcdValueRequest, ","),
		[]string{
			op1.Hash(),
			op2.Hash(),
			op3.Hash(),
		},
	)

}
