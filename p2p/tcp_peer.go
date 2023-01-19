package p2p

import "net"

// TCPPeer represents the remote node over a TCP established connection.
type TCPPeer struct {

	// conn is the underlying connection of the peer.
	conn net.Conn

	// if we dial and retrieve a conn => outbound == true.
	//
	// if we accept and retrieve a conn => outbound == false.
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.conn.Write(b)
	return err
}

// RemoteAddr implements the Peer interface and will return the remote address of it's underlying connection.
func (p *TCPPeer) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}

// Close implements the Peer interface.
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}
