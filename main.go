package main

import (
	"log"

	"github.com/ezratameno/forever-store/p2p"
)

func main() {

	opts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tr := p2p.NewTCPTransport(opts)

	err := tr.ListenAndAccept()
	if err != nil {
		log.Fatal(err)
	}

	// blocking
	select {}
}
