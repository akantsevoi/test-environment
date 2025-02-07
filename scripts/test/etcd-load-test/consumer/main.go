package main

import (
	"context"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	hashesKey = "/maroon/hashes"
)

func main() {
	endpoints := []string{} //strings.Split(os.Getenv("ETCD_ENDPOINTS"), ",")
	if len(endpoints) == 0 {
		endpoints = []string{"http://localhost:2379"}
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer cli.Close()

	watchCh := cli.Watch(context.Background(), hashesKey, clientv3.WithPrefix())

	log.Printf("Starting to watch %s...", hashesKey)

	var counter int
	for resp := range watchCh {
		for _, ev := range resp.Events {

			switch ev.Type {
			case clientv3.EventTypePut:
				counter++
				log.Println("got: ", counter)
				// log.Printf("New hash: %s = %s", string(ev.Kv.Key), string(ev.Kv.Value))
			case clientv3.EventTypeDelete:
				log.Printf("Deleted hash: %s", string(ev.Kv.Key))
			}
		}
	}
}
