package p2p

import (
	"testing"
	"time"
)

func TestSome(t *testing.T) {
	s1 := New("maroon-1", "8081")
	s2 := New("maroon-2", "8082")

	go s1.Start()
	go s2.Start()

	s1.AddToQueue(Message{Destination: "localhost:8082", Message: "hello"})

	time.Sleep(5 * time.Second)

	t.Fail()
}
