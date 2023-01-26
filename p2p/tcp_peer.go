package p2p

import (
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP established connection.
type TCPPeer struct {

	// conn is the underlying connection of the peer.
	// Which in this case is a tcp connection.
	net.Conn

	// if we dial and retrieve a conn => outbound == true.
	//
	// if we accept and retrieve a conn => outbound == false.
	outbound bool

	// sync the read of the peer.
	wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		// embed the conn interface.
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Conn.Write(b)
	return err
}

// CloseStream will close the stream
func (p *TCPPeer) CloseStream() {
	p.wg.Done()

}
