package maroon

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
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

	// all the operations that were confirmed by the followers
	// a slice because they have global order now
	confirmedOps []Operation

	//
	ackedHashes []string

	// txs that were created and sent to the followers but not ack-ed yet
	inFlyOPs map[string]Operation = make(map[string]Operation)

	// TODO: get rid of locks
	opMU *sync.Mutex = &sync.Mutex{}

	batchCounter int64 = 0
)

// / Follower specific
var ()

type AckMessage struct {
	Hashes []string
}

func RunApplication(cli ETCD, server DistTransport, isLeaderCh <-chan bool, distributedTxCh <-chan p2p.TransactionDistributed, etcdWatchCh clientv3.WatchChan, stopCh <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var isLeader bool

	for {
		select {
		case <-stopCh:
			return
		case isL := <-isLeaderCh:
			isLeader = isL
		case confirmation := <-distributedTxCh:
			if !isLeader {
				continue
			}
			logger.Infof(logger.Application, "tx %v confirmed", confirmation.ID)
			issueBlockIfCan(cli, confirmation)

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

func issueBlockIfCan(cli ETCD, confirmation p2p.TransactionDistributed) {
	opMU.Lock()
	defer opMU.Unlock()
	if _, ok := inFlyOPs[confirmation.ID]; !ok {
		// TODO: just ignore. But would be nice to clarify how you've got them
		return
	}

	ackedHashes = append(ackedHashes, confirmation.ID)

	if len(ackedHashes) >= 3 {
		// IMITATION!!!!
		merkleHash := strings.Join(ackedHashes, ",")
		_, err := cli.Put(context.TODO(), fmt.Sprintf("%s/%d", HashesKey, batchCounter), merkleHash)
		if err != nil {
			logger.Errorf(logger.Application, "failed to put merkle hash: %v", err)
			return
		}

		for _, hash := range ackedHashes {
			op, ok := inFlyOPs[hash]
			if !ok {
				continue
			}
			confirmedOps = append(confirmedOps, op)
			delete(inFlyOPs, hash)
		}
		ackedHashes = nil
	}

}

func fakeNewOperation(server DistTransport) {
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

	opMU.Lock()
	inFlyOPs[hashStr] = newOp
	opMU.Unlock()
	server.DistributeTx(p2p.Transaction{
		ID:     hashStr,
		TxData: message,
	})
}
