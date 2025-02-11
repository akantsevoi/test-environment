package p2p

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTransportCommunication(t *testing.T) {
	leader, distributedCh := New("localhost", "8081")
	f1, _ := New("localhost", "8082")
	f2, _ := New("localhost", "8083")

	leader.UpdateHosts([]string{"localhost:8082", "localhost:8083"})
	f1.UpdateHosts([]string{"localhost:8081", "localhost:8083"})
	f2.UpdateHosts([]string{"localhost:8081", "localhost:8082"})

	go leader.Start()
	go f1.Start()
	go f2.Start()

	var distributed []string
	go func() {
		for m := range distributedCh {
			distributed = append(distributed, string(m.ID))
		}
	}()

	leader.DistributeTx(Transaction{ID: "tx-1", TxData: []byte("hello-1")})
	leader.DistributeTx(Transaction{ID: "tx-2", TxData: []byte("hello-2")})

	time.Sleep(1 * time.Second)

	require.ElementsMatch(t, []string{"tx-1", "tx-2"}, distributed)
}
