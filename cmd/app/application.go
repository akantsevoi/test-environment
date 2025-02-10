package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/akantsevoi/test-environment/internal/p2p"
	"github.com/akantsevoi/test-environment/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type OperationType int64

const (
	PrintTimestamp OperationType = iota
)

type Operation struct {
	OpType OperationType
	Value  string
}

// Leader specific
var (
	totallyOrderedTxs []string

	unconfirmedTxs []string
)

// / Follower specific
var ()

type AckMessage struct {
	Hashes []string
}

func runApplication(server p2p.Transport, isLeaderCh <-chan bool, distributedTxCh <-chan p2p.TransactionDistributed, etcdWatchCh clientv3.WatchChan, stopCh <-chan struct{}) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var isLeader bool

	for {
		select {
		case <-stopCh:
			return
		case isL := <-isLeaderCh:
			isLeader = isL
		case confirmation := <-distributedTxCh:
			if isLeader {
				_ = confirmation
				// batch.add(confirmation)
				// if batch.isFull -> send to etcd

				logger.Infof(logger.Application, "tx %v confirmed", confirmation.ID)
				continue
			}
		case newEvent := <-etcdWatchCh:
			if isLeader {
				// leader already knows about it
				continue
			}

			_ = newEvent
			// TODO: check that all the transactions with that hash are here
			// and if there are not - start requesting them

		case <-ticker.C: // main logic
			if !isLeader {
				// follower
				continue
			}

			fakeNewOperation(server)
		}
	}
}

func fakeNewOperation(server p2p.Transport) {
	timestamp := time.Now().Unix()

	/// new operation
	newOp := Operation{
		OpType: PrintTimestamp,
		Value:  strconv.FormatInt(timestamp, 10),
	}

	message, err := json.Marshal(newOp)
	if err != nil {
		logger.Errorf(logger.Application, "Failed to marshal operation: %v", err)
		return
	}

	hashStr := fmt.Sprintf("%x", sha256.Sum256(message))
	server.DistributeTx(p2p.Transaction{
		ID:     hashStr,
		TxData: message,
	})
}
