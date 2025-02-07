package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/akantsevoi/test-environment/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type OperationType int64

const (
	PrintTimestamp OperationType = iota
)

type Operation struct {
	SequenceID int64
	OpType     OperationType
	Value      string
}

// Leader specific

var (
	leadSequenceCounter int64
	hashesToAck         map[string]struct{} = make(map[string]struct{})
)

// / Follower specific
var (
	followerSequenceCounter int64

	nextHashBatch []string

	// Storage for all operations on follower nodes
	incomingOperations map[string]Operation = make(map[string]Operation)

	// key - operationHash, value - sequenceID
	// is needed to save not acked hashes
	sequenceLog        map[string]int64    = make(map[string]int64)
	notAckedEtcdHashes map[string]struct{} = make(map[string]struct{})
)

type AckMessage struct {
	Hashes []string
}

func runApplication(cli *clientv3.Client, podName string, server *TCPServer, isLeaderCh <-chan bool, tcpInCh <-chan []byte, etcdWatchCh clientv3.WatchChan, stopCh <-chan struct{}) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var isLeader bool

	for {
		select {
		case <-stopCh:
			return
		case isL := <-isLeaderCh:
			isLeader = isL
		case newMessage := <-tcpInCh:
			if isLeader {
				var ackMess AckMessage
				if err := json.Unmarshal(newMessage, &ackMess); err != nil {
					logger.Errorf(logger.Application, "Failed to unmarshal acknowledgeMessage: %v", err)
					continue
				}

				for _, h := range ackMess.Hashes {
					delete(hashesToAck, h)
				}
				logger.Infof(logger.Application, "Hashes acked: %d. Moving next", len(ackMess.Hashes))
				continue
			}
			// here we just store all the incoming operations
			var newOp Operation
			if err := json.Unmarshal(newMessage, &newOp); err != nil {
				logger.Errorf(logger.Application, "Failed to unmarshal operation: %v", err)
				continue
			}
			hashStr := fmt.Sprintf("%x", sha256.Sum256(newMessage))
			incomingOperations[hashStr] = newOp
			logger.Infof(logger.Application, "incoming op counter: %d", len(incomingOperations))
		case newEvent := <-etcdWatchCh:
			if isLeader {
				// leader already knows about it
				continue
			}
			var incomingHashes []string
			for _, ev := range newEvent.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					hashes := string(ev.Kv.Value)
					incomingHashes = strings.Split(hashes, ",")
				case clientv3.EventTypeDelete:
					// TODO: what to do on delete hashes?
					// I'll delete old values from etcd
					// log.Printf("Hash deleted: %s", string(ev.Kv.Key))
				}
			}

			var counter int
			for _, h := range incomingHashes { // right now we have 3 in batch
				if _, ok := incomingOperations[h]; ok {
					counter++
				}
			}
			if counter == 3 {
				ack := AckMessage{
					Hashes: incomingHashes,
				}
				message, err := json.Marshal(ack)
				if err != nil {
					logger.Errorf(logger.Application, "Failed to marshal ack: %v", err)
				}
				if err := server.SendMessage("maroon-0", message); err != nil {
					logger.Errorf(logger.Application, "Failed to send ack: %v", err)
				}
			} else {
				logger.Infof(logger.Application, "Not all hashes are in operations. Expected: 3, got: %d", counter)
			}

		case <-ticker.C: // main logic

			if !isLeader {
				// follower
				continue
			}

			if len(hashesToAck) > 0 && len(nextHashBatch) == 0 {
				// means - previous batch were sent on ack but we didn't get ack yet
				// waiting
				logger.Infof(logger.Application, "Waiting for ack. SequenceID: %d. Not acked: %d", leadSequenceCounter, len(hashesToAck))
				continue
			}

			fakeNewOperation(cli, podName, server)
		}
	}
}

func fakeNewOperation(cli *clientv3.Client, podName string, server *TCPServer) {
	timestamp := time.Now().Unix()

	/// new operation
	leadSequenceCounter++
	newOp := Operation{
		SequenceID: leadSequenceCounter,
		OpType:     PrintTimestamp,
		Value:      strconv.FormatInt(timestamp, 10),
	}

	message, err := json.Marshal(newOp)
	if err != nil {
		logger.Errorf(logger.Application, "Failed to marshal operation: %v", err)
		return
	}

	hashStr := fmt.Sprintf("%x", sha256.Sum256(message))
	nextHashBatch = append(nextHashBatch, hashStr)
	hashesToAck[hashStr] = struct{}{}

	// Send TCP messages to other pods
	for i := 0; i < 3; i++ {
		targetPod := fmt.Sprintf("maroon-%d", i)
		if targetPod != podName {
			if err := server.SendMessage(targetPod, message); err != nil {
				logger.Errorf(logger.Application, "Failed to send operation: %v", err)
			}
		}
	}

	if len(nextHashBatch) >= 3 {
		key := fmt.Sprintf("%s/%d", hashesKey, leadSequenceCounter)

		// IMITATION!!!!
		merkleHash := strings.Join(nextHashBatch, ",")

		_, err := cli.Put(context.Background(), key, merkleHash)
		if err != nil {
			logger.Errorf(logger.Application, "Failed to write hash: %v", err)
		} else {
			logger.Infof(logger.Application, "Published hash: %s", merkleHash)
		}

		nextHashBatch = nil
	}
}
