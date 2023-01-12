package p2p

// Peer is an interface that represent the remote node.
type Peer interface {
	Close() error
}

// Transport is anything that handle the communication
// between nodes in the network.
// This can be of the form (TCP,UDP.websockets...)
type Transport interface {
	ListenAndAccept() error

	// Consume will read rpc messages.
	Consume() <-chan RPC
}
