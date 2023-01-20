package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
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

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key string
}

// broadcast will send the Message to all the known peers in the network.
func (s *FileServer) broadcast(msg *Message) error {

	// peer implements the io.writer interface because the net.Conn implements it.
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)

	// transmit the data to all the peers.
	return gob.NewEncoder(mw).Encode(msg)
}

func (s *FileServer) StoreData(key string, r io.Reader) error {

	// buf := new(bytes.Buffer)

	// // once we read from the r the data will no longer be available.
	// // tee helps us to make the data to be available also on the buffer.
	// tee := io.TeeReader(r, buf)

	// // 1. store this file to disk.
	// err := s.store.Write(key, tee)
	// if err != nil {
	// 	return err
	// }

	// // 2. broadcast this file to all known peers in the network.
	// p := &DataMessage{
	// 	Key:  key,
	// 	Data: buf.Bytes(),
	// }

	// return s.broadcast(&Message{
	// 	From:    "todo",
	// 	Payload: p,
	// })

	buf := new(bytes.Buffer)
	msg := Message{
		Payload: MessageStoreFile{
			Key: key,
		},
	}

	err := gob.NewEncoder(buf).Encode(msg)
	if err != nil {
		return err
	}

	// send the message telling what we want to do to all the peers.
	for _, peer := range s.peers {
		err := peer.Send(buf.Bytes())
		if err != nil {
			return err
		}
	}

	// time.Sleep(3 * time.Second)

	// payload := []byte("THIS LARGE FILE")

	// // stream the file.
	// for _, peer := range s.peers {
	// 	err := peer.Send(payload)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	return nil
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
		case rpc := <-s.Transport.Consume():

			// decode into a Message.
			var msg Message

			err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg)
			if err != nil {
				log.Println(err)
				continue
			}

			fmt.Printf("%+v\n", msg.Payload)

			// Get the peer that sent the message.
			peer, ok := s.peers[rpc.From]
			if !ok {
				panic("peer not found in peer map")
			}

			fmt.Printf("%+v\n", peer)

			// Read the next incoming (should be the stream that's incoming).
			b := make([]byte, 1000)
			_, err = peer.Read(b)
			if err != nil {
				panic(err)
			}

			fmt.Println(string(b))

			// enable to continue reading.
			peer.(*p2p.TCPPeer).Wg.Done()

			// err = s.handleMessage(&m)
			// if err != nil {
			// 	log.Println(err)
			// 	continue
			// }

		case <-s.quitch:
			return
		}
	}
}

// func (s *FileServer) handleMessage(msg *Message) error {

// 	switch v := msg.Payload.(type) {
// 	case *DataMessage:
// 		fmt.Println("recived data %+v\n", v)
// 	}
// 	return nil
// }

// BootstrapNetwork will add default nodes to our network.
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

func init() {

	// we need to register all the types we want to send in the message.payload as any.
	gob.Register(MessageStoreFile{})
}
