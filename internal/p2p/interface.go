package p2p

type Transport interface {
	Start()
	Stop()

	// nonblocking
	// will distribute it to some amount of hosts according to the settings I'll introduce later
	// it's an async channel
	// confirmation
	DistributeTx(m Transaction)

	// blocking
	UpdateHosts([]string)
}

// this message comes from the server when the transaction is confirmed by other nodes
// how many and which nodes TBD
type TransactionDistributed struct {
	// ID of a transaction
	ID string
}

type Transaction struct {
	// unique transaction id
	// it's important that this thing is globally unique!!!!!
	ID string

	// TODO: I'll do smth smarter later
	TxData []byte
}
