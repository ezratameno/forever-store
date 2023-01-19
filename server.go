package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/ezratameno/forever-store/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
}

type FileServer struct {
	FileServerOpts

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
	}
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

func (s *FileServer) Start() error {

	err := s.Transport.ListenAndAccept()
	if err != nil {
		return err
	}

	s.loop()
	return nil

}
