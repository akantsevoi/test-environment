package p2p

type Transport interface {
	Start()
	Stop()

	// nonblocking
	AddToQueue(m Message)
	// blocking
	UpdateHosts([]string)
}

type Message struct {
	// if len(destination) == 0 -> broadcast
	Destination string

	// TODO: I'll do smth smarter later
	AddTxData []byte
	AckData   string
}
