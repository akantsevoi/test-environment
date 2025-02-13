package etcdmock

import (
	"context"
	"strings"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

///
/// In-memory implementation of the ETCD's subset client interface
/// Keep in mind that it's no accurate and reliable implementation but rather test helper
///

func New() ETCDMock {
	return &etcd{
		prefixWatch: make(map[string]chan clientv3.WatchResponse),
		store:       make(map[string]string),
	}
}

type ETCDMock interface {
	Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error)
	Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan
}

type etcd struct {
	prefixWatch map[string](chan clientv3.WatchResponse)
	store       map[string]string
}

func (e *etcd) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	e.store[key] = val

	// Notify all watchers
	for prefix, watchCh := range e.prefixWatch {
		if strings.HasPrefix(key, prefix) {
			watchCh <- clientv3.WatchResponse{
				Events: []*clientv3.Event{
					{
						Type: clientv3.EventTypePut,
						Kv: &mvccpb.KeyValue{
							Key:   []byte(key),
							Value: []byte(val),
						},
					},
				},
			}
		}
	}

	// TODO: implement to mimick better?
	return &clientv3.PutResponse{}, nil
}

// TODO: only support by prefix right now
func (e *etcd) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	newWatchCh := make(clientv3.WatchChan)
	e.prefixWatch[key] = make(chan clientv3.WatchResponse)
	return newWatchCh
}
