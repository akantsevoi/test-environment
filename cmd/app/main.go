package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

func main() {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		log.Fatal("POD_NAME environment variable is required")
	}
	log.Printf("Starting maroon pod: %s", podName)

	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		log.Fatal("ETCD_ENDPOINTS environment variable is required")
	}
	endpoints := strings.Split(etcdEndpoints, ",")
	log.Printf("Using etcd endpoints: %v", endpoints)

	log.Printf("Connecting to etcd...")
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer cli.Close()
	log.Printf("Connected to etcd successfully")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = cli.Get(ctx, "test-key")
	if err != nil {
		log.Fatalf("failed to perform test Get operation: %v", err)
	}
	log.Printf("Test Get operation successful")

	log.Printf("Creating etcd session...")
	session, err := concurrency.NewSession(cli, concurrency.WithTTL(10))
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()
	log.Printf("Session created successfully")

	log.Printf("Creating election...")
	election := concurrency.NewElection(session, "/election")
	log.Printf("Election created")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		leader, err := election.Leader(ctx)
		cancel()
		if err == nil {
			log.Printf("Current leader: %s", string(leader.Kvs[0].Value))
			if string(leader.Kvs[0].Value) != podName {
				// We're not the leader, wait and observe
				time.Sleep(5 * time.Second)
				continue
			}
		}

		log.Printf("Starting campaign for leadership...")
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		if err := election.Campaign(ctx, podName); err != nil {
			log.Printf("Failed to campaign: %v", err)
			cancel()
			time.Sleep(5 * time.Second)
			continue
		}
		defer cancel()

		log.Printf("Pod %s became leader", podName)

	leaderLoop:
		for {
			select {
			case <-session.Done():
				log.Printf("Lost leadership due to session termination")
				break leaderLoop
			default:
				log.Printf("Leader is working. Ping")
				time.Sleep(5 * time.Second)
			}
		}

		time.Sleep(1 * time.Second)
	}
}
