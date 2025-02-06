package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"strconv"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func runApplication(cli *clientv3.Client, podName string, server *TCPServer, stopCh <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			timestamp := time.Now().Unix()
			hash := sha256.Sum256([]byte(strconv.FormatInt(timestamp, 10)))
			hashStr := fmt.Sprintf("%x", hash)

			key := fmt.Sprintf("%s/%d", hashesKey, timestamp)
			_, err := cli.Put(context.Background(), key, hashStr)
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
		}
	}
}
