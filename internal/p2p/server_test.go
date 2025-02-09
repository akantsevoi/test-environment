package p2p

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTransportCommunication(t *testing.T) {
	leader, _ := New("localhost", "8081")
	f1, inf1Ch := New("localhost", "8082")
	f2, inf2Ch := New("localhost", "8083")

	leader.UpdateHosts([]string{"localhost:8082", "localhost:8083"})
	f1.UpdateHosts([]string{"localhost:8081", "localhost:8083"})
	f2.UpdateHosts([]string{"localhost:8081", "localhost:8082"})

	go leader.Start()
	go f1.Start()
	go f2.Start()

	var f1Res []string
	go func() {
		for m := range inf1Ch {
			f1Res = append(f1Res, string(m.AddTxData))
		}
	}()
	var f2Res []string
	go func() {
		for m := range inf2Ch {
			f2Res = append(f2Res, string(m.AddTxData))
		}
	}()

	leader.AddToQueue(Message{Destination: "localhost:8082", AddTxData: []byte("hello")})
	leader.AddToQueue(Message{Destination: "", AddTxData: []byte("broadcast")})

	time.Sleep(1 * time.Second)

	require.ElementsMatch(t, []string{"hello", "broadcast"}, f1Res)
	require.ElementsMatch(t, []string{"broadcast"}, f2Res)
}
