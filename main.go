package main

import (
	"fmt"
	"log"

	"github.com/ezratameno/forever-store/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcpTransport := p2p.NewTCPTransport(&tcpTransportOpts)
	fileServerOptions := FileServerOpts{
		StorageRoot:       fmt.Sprintf("%s_network", listenAddr),
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOptions)

	tcpTransportOpts.OnPeer = s.OnPeer

	return s
}

func main() {

	s1 := makeServer(":3001")
	s2 := makeServer(":4001", ":3001")

	go func() {
		log.Fatal(s1.Start())
	}()

	log.Fatal(s2.Start())

}
