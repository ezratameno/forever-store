package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
)

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}
type TCPTransport struct {
	*TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC
}

func NewTCPTransport(opts *TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC, 1024),
	}
}

// Consume implements the Transport interface, which will return read-only channel.
//
// For reading the incoming messages received from another peer in the network.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial implements the Transport interface.
func (t *TCPTransport) Dial(addr string) error {

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	// outbound connection.
	go t.handleConn(conn, true)

	return nil
}

// Addr implements the Transport interface.
// Returns the address of the transport that is accepting connections.
func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

func (t *TCPTransport) ListenAndAccept() error {

	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	log.Printf("TCP transport listening on port: %s\n", t.ListenAddr)
	return nil

}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()

		// if the connection is closed.
		if errors.Is(err, net.ErrClosed) {
			return
		}

		if err != nil {
			fmt.Printf("TCP accept error: %+v\n", err)
			continue
		}

		// inbound connection because we are accepting connection.
		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {

	var err error
	defer func() {
		fmt.Printf("Dropping peer connection: %+v\n", err)
		conn.Close()

	}()

	peer := NewTCPPeer(conn, outbound)

	// First shake hands before reading.
	if err := t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// Read loop.
	for {
		rpc := RPC{}

		// Read the message from the connection.
		err = t.Decoder.Decode(conn, &rpc)
		if err != nil {
			fmt.Printf("TCP error: %s\n", err)
			return
		}

		rpc.From = conn.RemoteAddr().String()

		if rpc.Stream {

			// Guard the reading operation, we can't continue to read until the stream ended.
			peer.wg.Add(1)
			fmt.Printf("[%s] incoming stream, waiting...\n", conn.RemoteAddr().String())

			peer.wg.Wait()
			fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr().String())

			// In streaming there is no need to send the rpc, because it will be empty.
			// We are sending files and not messages.
			continue
		}

		t.rpcch <- rpc

	}
}
