package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/ezratameno/forever-store/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitch chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {

	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}
	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

// OnPeer will add the peer to our network.
func (s *FileServer) OnPeer(p p2p.Peer) error {

	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p
	log.Printf("Connected with remote peer: %s", p.RemoteAddr().String())
	return nil
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

// loop will read messages.
func (s *FileServer) loop() {

	defer func() {
		log.Println("file server stopped due to user quit action.")
		s.Transport.Close()
	}()

	for {
		select {
		case msg := <-s.Transport.Consume():
			fmt.Printf("msg: %s, from: %+v:\n", strings.TrimSpace(string(msg.Payload)), msg.From)
		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) BootstrapNetwork() error {

	for _, addr := range s.BootstrapNodes {

		fmt.Printf("Attempting to connect with remote: %s \n", addr)

		go func(addr string) {

			err := s.Transport.Dial(addr)
			if err != nil {
				log.Printf("dial error: %+v", err)
				return
			}

			fmt.Printf("Connected with remote: %s \n", addr)

		}(addr)
	}

	return nil
}

func (s *FileServer) Start() error {

	err := s.Transport.ListenAndAccept()
	if err != nil {
		return err
	}

	s.BootstrapNetwork()
	s.loop()
	return nil

}
