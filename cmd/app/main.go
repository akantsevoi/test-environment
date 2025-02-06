package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/akantsevoi/test-environment/pkg/election"
	"github.com/akantsevoi/test-environment/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	leaderKey = "/maroon/leader"
	hashesKey = "/maroon/hashes"
)

func main() {
	vars := envs()
	podName := vars.podName
	logger.Infof(logger.Application, "Starting maroon pod: %s", podName)
	logger.Infof(logger.Application, "Using etcd endpoints: %v", vars.etcdEndpoints)

	server := NewTCPServer(8080)
	go server.Start()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   vars.etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logger.Errorf(logger.Application, "failed to create etcd client: %v", err)
		os.Exit(1)
	}
	defer cli.Close()

	// watching hashes
	watchChan := cli.Watch(context.Background(), hashesKey+"/", clientv3.WithPrefix())
	go watchHashes(watchChan)

	leader := election.NewLeader(cli, leaderKey, podName)

	for {
		leaderCh, err := leader.Campaign()
		if err != nil {
			logger.Errorf(logger.Election, "failed to campaign: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		logger.Infof(logger.Election, "pod %s became leader", podName)

		// Start application logic in a separate goroutine
		stopCh := make(chan struct{})
		go runApplication(cli, podName, server, stopCh)

		// Wait for leadership loss
		<-leaderCh
		close(stopCh)
		logger.Infof(logger.Election, "lost leadership")

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
		logger.Fatalf(logger.Application, "POD_NAME environment variable is required")
	}

	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		logger.Fatalf(logger.Application, "ETCD_ENDPOINTS environment variable is required")
	}
	endpoints := strings.Split(etcdEndpoints, ",")

	return envVariables{
		podName:       podName,
		etcdEndpoints: endpoints,
	}
}
