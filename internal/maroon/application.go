package maroon

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/akantsevoi/test-environment/internal/p2p"
	"github.com/akantsevoi/test-environment/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type application struct {
	data
	deps
}

type data struct {
	// all the operations that were confirmed by the followers
	// a slice because they have global order now
	confirmedOps []Operation

	//
	ackedHashes []string

	// txs that were created and sent to the followers but not ack-ed yet
	inFlyOPs map[string]Operation

	// TODO: get rid of locks
	opMU *sync.Mutex

	batchCounter int64

	isLeader bool
}

type deps struct {
	cli      ETCD
	p2pDistr DistTransport
}

func New(cli ETCD, p2pDistr DistTransport) *application {
	return &application{
		data: data{
			inFlyOPs: make(map[string]Operation),
			opMU:     &sync.Mutex{},
		},
		deps: deps{
			cli:      cli,
			p2pDistr: p2pDistr,
		},
	}
}

func (a *application) Run(isLeaderCh <-chan bool, distributedTxCh <-chan p2p.TransactionDistributed, etcdWatchCh clientv3.WatchChan, stopCh <-chan struct{}) {

	for {
		select {
		case <-stopCh:
			return
		case isL := <-isLeaderCh:
			a.isLeader = isL
		case confirmation := <-distributedTxCh:
			if !a.isLeader {
				continue
			}
			logger.Infof(logger.Application, "tx %v confirmed", confirmation.ID)
			a.issueBlockIfCan(a.cli, confirmation)

		case newEvent := <-etcdWatchCh:
			if a.isLeader {
				// leader already knows about it
				continue
			}

			_ = newEvent
			// TODO: check that all the transactions with that hash are here
			// and if there are not - start requesting them
		}
	}
}

func (a *application) issueBlockIfCan(cli ETCD, confirmation p2p.TransactionDistributed) {
	a.opMU.Lock()
	defer a.opMU.Unlock()
	if _, ok := a.inFlyOPs[confirmation.ID]; !ok {
		// TODO: just ignore. But would be nice to clarify how you've got them
		return
	}

	a.ackedHashes = append(a.ackedHashes, confirmation.ID)

	if len(a.ackedHashes) >= 3 {
		// IMITATION!!!!
		merkleHash := strings.Join(a.ackedHashes, ",")
		_, err := cli.Put(context.TODO(), fmt.Sprintf("%s/%d", HashesKey, a.batchCounter), merkleHash)
		if err != nil {
			logger.Errorf(logger.Application, "failed to put merkle hash: %v", err)
			return
		}

		for _, hash := range a.ackedHashes {
			op, ok := a.inFlyOPs[hash]
			if !ok {
				continue
			}
			a.confirmedOps = append(a.confirmedOps, op)
			delete(a.inFlyOPs, hash)
		}
		a.ackedHashes = nil
	}

}

func (a *application) AddOp(op Operation) {
	if !a.isLeader {
		return
	}

	hashStr, message := op.HashBin()

	a.opMU.Lock()
	a.inFlyOPs[hashStr] = op
	a.opMU.Unlock()
	a.p2pDistr.DistributeTx(p2p.Transaction{
		ID:     hashStr,
		TxData: message,
	})
}
