package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	leaderKey = "/maroon/leader"
	hashesKey = "/maroon/hashes"
)

func main() {
	endpoints := []string{} //strings.Split(os.Getenv("ETCD_ENDPOINTS"), ",")
	if len(endpoints) == 0 {
		endpoints = []string{"http://localhost:2379"}
	}

	log.Printf("Connecting to etcd endpoints: %v", endpoints)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer cli.Close()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = cli.Status(ctx, endpoints[0])
	if err != nil {
		log.Fatalf("failed to connect to etcd: %v", err)
	}
	log.Println("Successfully connected to etcd")

	log.Println("Attempting to grant lease...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lease, err := cli.Grant(ctx, 10)
	if err != nil {
		log.Fatalf("failed to create lease: %v", err)
	}
	log.Printf("Successfully granted lease with ID: %d", lease.ID)

	log.Println("try to become leader")
	leaderID := "node-1"

	getResp, err := cli.Get(context.Background(), leaderKey)
	if err != nil {
		log.Fatalf("failed to check leader key: %v", err)
	}
	log.Printf("%#v", getResp)

	resp, err := cli.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(leaderKey), "=", 0)).
		Then(clientv3.OpPut(leaderKey, leaderID, clientv3.WithLease(lease.ID))).
		Else(clientv3.OpGet(leaderKey)).
		Commit()
	if err != nil {
		log.Fatalf("failed to try become leader: %v", err)
	}

	if !resp.Succeeded {
		currentLeader := string(resp.Responses[0].GetResponseRange().Kvs[0].Value)
		log.Fatalf("failed to become leader, current leader is: %s", currentLeader)
	}

	log.Println("Successfully became leader")

	log.Println("keep lease alive")
	keepAliveCh, err := cli.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		log.Fatalf("failed to keep lease alive: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()
	var counter int
	for {
		select {
		case <-ticker.C:
			timestamp := time.Now().Unix()
			hash := sha256.Sum256([]byte(strconv.FormatInt(timestamp, 10)))
			hashStr := fmt.Sprintf("%x", hash)

			// Write hash with transaction
			key := fmt.Sprintf("%s/%d", hashesKey, timestamp)
			_, err = cli.Txn(context.Background()).
				If(clientv3.Compare(clientv3.Value(leaderKey), "=", leaderID)).
				Then(clientv3.OpPut(key, hashStr)).
				Commit()
			if err != nil {
				log.Printf("failed to write hash: %v", err)
				continue
			}
			counter++
			// log.Printf("Written hash for timestamp %d: %s", timestamp, hashStr)
			log.Println("counter: ", counter)

		case _, ok := <-keepAliveCh:
			if !ok {
				log.Fatal("lost lease")
			}

		case <-sigCh:
			log.Println("Shutting down...")
			return
		}
	}
}
