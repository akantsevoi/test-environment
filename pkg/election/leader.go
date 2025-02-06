package election

import (
	"context"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Leader struct {
	cli       *clientv3.Client
	leaderKey string
	nodeID    string
	lease     clientv3.LeaseID
}

func NewLeader(cli *clientv3.Client, leaderKey, nodeID string) *Leader {
	return &Leader{
		cli:       cli,
		leaderKey: leaderKey,
		nodeID:    nodeID,
	}
}

// returns channel to notify about leadership loss
func (l *Leader) Campaign() (<-chan struct{}, error) {
	lease, err := l.cli.Grant(context.Background(), 10)
	if err != nil {
		return nil, fmt.Errorf("failed to create lease: %v", err)
	}
	l.lease = lease.ID

	resp, err := l.cli.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(l.leaderKey), "=", 0)).
		Then(clientv3.OpPut(l.leaderKey, l.nodeID, clientv3.WithLease(lease.ID))).
		Else(clientv3.OpGet(l.leaderKey)).
		Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to execute leader transaction: %v", err)
	}

	if !resp.Succeeded {
		currentLeader := string(resp.Responses[0].GetResponseRange().Kvs[0].Value)
		return nil, fmt.Errorf("failed to become leader, current leader is: %s", currentLeader)
	}

	keepAliveCh, err := l.cli.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to keep lease alive: %v", err)
	}

	leaderCh := make(chan struct{})
	go func() {
		for {
			if _, ok := <-keepAliveCh; !ok {
				close(leaderCh)
				return
			}
		}
	}()

	return leaderCh, nil
}

func (l *Leader) IsLeader() bool {
	resp, err := l.cli.Get(context.Background(), l.leaderKey)
	if err != nil {
		return false
	}
	if len(resp.Kvs) == 0 {
		return false
	}
	return string(resp.Kvs[0].Value) == l.nodeID
}
