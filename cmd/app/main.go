package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/akantsevoi/test-environment/pkg/election"
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

	leader := election.NewLeader(cli, leaderKey, podName)

	for {
		leaderCh, err := leader.Campaign()
		if err != nil {
			log.Printf("Failed to campaign: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("Pod %s became leader", podName)

		// Start application logic in a separate goroutine
		stopCh := make(chan struct{})
		go runApplication(cli, podName, server, stopCh)

		// Wait for leadership loss
		<-leaderCh
		close(stopCh)
		log.Printf("Lost leadership")

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
