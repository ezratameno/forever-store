package main

import (
	"bytes"
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
		EncKey:            newEncryptionKey(),
	}

	s := NewFileServer(fileServerOptions)

	tcpTransportOpts.OnPeer = s.OnPeer

	return s
}

func main() {

	s1 := makeServer(":3001")
	s2 := makeServer(":4001", ":3001")
	s3 := makeServer(":5001", ":4001", ":3001")

	go func() {
		log.Fatal(s1.Start())
	}()

	time.Sleep(2 * time.Second)

	go s2.Start()

	go func() {
		log.Fatal(s3.Start())
	}()

	time.Sleep(2 * time.Second)

	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("%s_%d", "myPrivateData", i)

		data := bytes.NewReader([]byte("my big data file here!"))
		err := s3.Store(key, data)
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(5 * time.Millisecond)

		// Delete the key so we can fetch it remotely.
		err = s3.store.Delete(key)
		if err != nil {
			log.Fatal(err)
		}

		r, err := s3.Get(key)
		if err != nil {
			log.Fatal(err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("got: %s\n", string(b))
	}

	select {}

}
