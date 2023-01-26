package p2p

import "net"

// Peer is an interface that represent the remote node.
type Peer interface {
	// interface embedding.
	net.Conn
	Send([]byte) error
	CloseStream()
}

// Transport is anything that handle the communication
// between nodes in the network.
// This can be of the form (TCP,UDP,websocket...)
type Transport interface {
	Addr() string
	Dial(string) error
	ListenAndAccept() error

	// Consume will read rpc messages.
	Consume() <-chan RPC

	Close() error
}
