package main

import (
	"fmt"
	"io"
	"log"
	"time"

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

	time.Sleep(2 * time.Second)

	go s2.Start()

	time.Sleep(2 * time.Second)

	// for i := 0; i < 3; i++ {
	// data := bytes.NewReader([]byte("my big data file here!"))
	// err := s2.Store("myPrivateData", data)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// time.Sleep(5 * time.Millisecond)
	// }

	r, err := s2.Get("myPrivateData")
	if err != nil {
		log.Fatal(err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("got: %s\n", string(b))

	select {}

}
