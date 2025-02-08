package p2p

type Transport interface {
	Start()
	Stop()

	AddToQueue(m Message)
	UpdateHosts([]string)
}

type Message struct {
	// if len(destination) == 0 -> broadcast
	Destination string

	Message any
}
