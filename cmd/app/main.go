package main

import (
	"context"
	"fmt"
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

	// Start TCP server in a separate goroutine
	server := NewTCPServer(8080)
	go server.Start()

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

	log.Printf("Creating etcd session...")
	session, err := concurrency.NewSession(cli, concurrency.WithTTL(3))
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	defer session.Close()

	log.Printf("Session created successfully")

	log.Printf("Creating election...")
	election := concurrency.NewElection(session, "/election")
	log.Printf("Election created")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		leader, err := election.Leader(ctx)
		cancel()
		if err == nil {
			log.Printf("Current leader: %s", string(leader.Kvs[0].Value))
			if string(leader.Kvs[0].Value) != podName {
				time.Sleep(1 * time.Second)
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
				for i := 0; i < 3; i++ {
					targetPod := fmt.Sprintf("maroon-%d", i)
					if targetPod != podName {
						err := server.SendMessage(targetPod, fmt.Sprintf("Hello from leader %s", podName))
						if err != nil {
							log.Printf("Failed to send message to %s: %v", targetPod, err)
						}
					}
				}
				log.Printf("Leader is working. Ping")
				time.Sleep(1 * time.Second)
			}
		}

		time.Sleep(1 * time.Second)
	}
}
