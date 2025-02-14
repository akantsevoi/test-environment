package main

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/akantsevoi/test-environment/internal/maroon"
	"github.com/akantsevoi/test-environment/internal/p2p"
	"github.com/akantsevoi/test-environment/pkg/election"
	"github.com/akantsevoi/test-environment/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	vars := envs()
	podName := vars.podName
	logger.Infof(logger.Application, "Starting maroon pod: %s", podName)
	logger.Infof(logger.Application, "Using etcd endpoints: %v", vars.etcdEndpoints)

	// start TCP p2p distributor
	p2pDistr, confirmedTXsCh := p2p.New(podName, "8080")
	p2pDistr.UpdateHosts([]string{
		"maroon-1:8080",
		"maroon-2:8080",
	})
	go p2pDistr.Start()

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
	watchChan := cli.Watch(context.Background(), maroon.HashesKey+"/", clientv3.WithPrefix())

	// Start application logic in a separate goroutine
	stopCh := make(chan struct{})
	isLeaderCh := make(chan bool)
	app := maroon.New(cli, p2pDistr)
	go app.Run(isLeaderCh, confirmedTXsCh, watchChan, stopCh)
	isLeaderCh <- false

	leader := election.NewLeader(cli, maroon.LeaderKey, podName)

	// imitation of incoming requests
	go func() {
		tickerCh := time.Tick(10 * time.Second)
		for tick := range tickerCh {
			timestamp := tick.Unix()

			app.AddOp(maroon.Operation{
				OpType: maroon.PrintTimestamp,
				Value:  strconv.FormatInt(timestamp, 10),
			})
		}
	}()

	for {
		const timeBetweenAttempts = 3 * time.Second
		leaderCh, err := leader.Campaign()
		if err != nil {
			isLeaderCh <- false
			logger.Errorf(logger.Election, "failed to campaign: %v", err)
			time.Sleep(timeBetweenAttempts)
			continue
		}

		logger.Infof(logger.Election, "pod %s became leader", podName)
		isLeaderCh <- true

		// Wait for leadership loss
		<-leaderCh
		isLeaderCh <- false
		logger.Infof(logger.Election, "lost leadership")

		// this wait is for followers or for the leader who lost leadership to wait and start campaign again
		time.Sleep(timeBetweenAttempts)
	}

	// Unreachable
	// TODO: add graceful shutdown
	close(stopCh)
	close(isLeaderCh)
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
