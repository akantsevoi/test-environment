package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	leaderKey = "/maroon/leader"
	hashesKey = "/maroon/hashes"
)

func main() {
	vars := envs()
	podName := vars.podName
	log.Printf("Starting maroon pod: %s", podName)
	log.Printf("Using etcd endpoints: %v", vars.etcdEndpoints)

	server := NewTCPServer(8080)
	go server.Start()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   vars.etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer cli.Close()

	// watching hashes
	watchChan := cli.Watch(context.Background(), hashesKey+"/", clientv3.WithPrefix())
	go watchHashes(watchChan)

	for {
		// Create lease
		lease, err := cli.Grant(context.Background(), 10)
		if err != nil {
			log.Printf("Failed to create lease: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Try to become leader using transaction
		resp, err := cli.Txn(context.Background()).
			If(clientv3.Compare(clientv3.Version(leaderKey), "=", 0)).
			Then(clientv3.OpPut(leaderKey, podName, clientv3.WithLease(lease.ID))).
			Else(clientv3.OpGet(leaderKey)).
			Commit()
		if err != nil {
			log.Printf("Failed to execute leader transaction: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if !resp.Succeeded {
			currentLeader := string(resp.Responses[0].GetResponseRange().Kvs[0].Value)
			log.Printf("Current leader is: %s", currentLeader)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("Pod %s became leader", podName)

		// Keep lease alive
		keepAliveCh, err := cli.KeepAlive(context.Background(), lease.ID)
		if err != nil {
			log.Printf("Failed to keep lease alive: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

	leaderLoop:
		for {
			select {
			case _, ok := <-keepAliveCh:
				if !ok {
					log.Printf("Lost lease")
					break leaderLoop
				}
			default:
				timestamp := time.Now().Unix()
				hash := sha256.Sum256([]byte(strconv.FormatInt(timestamp, 10)))
				hashStr := fmt.Sprintf("%x", hash)

				key := fmt.Sprintf("%s/%d", hashesKey, timestamp)
				_, err = cli.Txn(context.Background()).
					If(clientv3.Compare(clientv3.Value(leaderKey), "=", podName)).
					Then(clientv3.OpPut(key, hashStr)).
					Commit()
				if err != nil {
					log.Printf("Failed to write hash: %v", err)
				} else {
					log.Printf("Published hash: %s", hashStr)
				}

				// Send TCP messages to other pods
				// for i := 0; i < 3; i++ {
				// 	targetPod := fmt.Sprintf("maroon-%d", i)
				// 	if targetPod != podName {
				// 		err := server.SendMessage(targetPod, fmt.Sprintf("Hash: %s", hashStr))
				// 		if err != nil {
				// 			log.Printf("Failed to send message to %s: %v", targetPod, err)
				// 		}
				// 	}
				// }

				time.Sleep(1 * time.Second)
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func watchHashes(watchChan clientv3.WatchChan) {
	for resp := range watchChan {
		for _, ev := range resp.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				log.Printf("Received new hash: %s", string(ev.Kv.Value))
			case clientv3.EventTypeDelete:
				log.Printf("Hash deleted: %s", string(ev.Kv.Key))
			}
		}
	}
}

type envVariables struct {
	podName       string
	etcdEndpoints []string
}

func envs() envVariables {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		log.Fatal("POD_NAME environment variable is required")
	}

	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		log.Fatal("ETCD_ENDPOINTS environment variable is required")
	}
	endpoints := strings.Split(etcdEndpoints, ",")

	return envVariables{
		podName:       podName,
		etcdEndpoints: endpoints,
	}
}
