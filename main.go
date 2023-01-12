package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/ezratameno/forever-store/p2p"
)

func OnPeer(peer p2p.Peer) error {

	fmt.Println("doing some logic with the peer outside of TPC Transport")
	return nil
}
func main() {

	opts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(opts)

	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("message: %+v, from: %s\n", strings.TrimSpace(string(msg.Payload)), msg.From)

		}
	}()
	err := tr.ListenAndAccept()
	if err != nil {
		log.Fatal(err)
	}

	// blocking
	select {}
}
