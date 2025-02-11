package maroon

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/akantsevoi/test-environment/internal/p2p"
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

func TestApp(t *testing.T) {

	etcd := &etcdMock{
		put: func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
			log.Println("etcd", key, val)
			return &clientv3.PutResponse{}, nil
		},
	}

	opDistributedCh := make(chan p2p.TransactionDistributed)

	serv := &servMock{
		distr: func(tx p2p.Transaction) {
			log.Println(tx)
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

	go RunApplication(etcd, serv, isLeaderCh, opDistributedCh, etcdWatchCh, stopCh)
	isLeaderCh <- true

	time.Sleep(8 * time.Second)
	stopCh <- struct{}{}

	time.Sleep(4 * time.Second)
	t.Fail()
}
