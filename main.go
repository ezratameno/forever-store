package main

import (
	"log"
	"time"

	"github.com/ezratameno/forever-store/p2p"
)

func main() {

	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO: onPeer func.
	}

	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
	fileServerOptions := FileServerOpts{
		StorageRoot:       "3000_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
	}
	s := NewFileServer(fileServerOptions)

	go func() {
		time.Sleep(time.Second * 3)
		s.Stop()
	}()

	err := s.Start()
	if err != nil {
		log.Fatal(err)
	}

}
