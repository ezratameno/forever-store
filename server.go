package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

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

	// We need to specifiy the size of the message.
	// because we are streaming we won't get EOF, so we need to know how many bytes to read.
	Size int64
}

type MessageGetFile struct {
	Key string
}

// stream will send the Message to all the known peers in the network.
func (s *FileServer) stream(msg *Message) error {

	// peer implements the io.writer interface because the net.Conn implements it.
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)

	// transmit the data to all the peers.
	return gob.NewEncoder(mw).Encode(msg)
}

// broadcast will send the message to all the known peers in the network.
func (s *FileServer) broadcast(msg *Message) error {

	buf := new(bytes.Buffer)

	// Encode the msg.
	err := gob.NewEncoder(buf).Encode(msg)
	if err != nil {
		return err
	}

	// Send a message to all the peers telling what we want to do.
	for _, peer := range s.peers {
		err := peer.Send(buf.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *FileServer) Get(key string) (io.Reader, error) {

	// we have the key locally.
	if s.store.Has(key) {
		return s.store.Read(key)
	}

	fmt.Printf("don't have file (%s) locally, fetching from network\n", key)

	// ======================================================
	// Send a message to the known peers to get the key.
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	err := s.broadcast(&msg)
	if err != nil {
		return nil, err
	}

	// ==========================================
	// Read from every peer to get us the file.

	for _, peer := range s.peers {
		fmt.Printf("receiving stream from peer:%s \n", peer.RemoteAddr().String())
		fileBuffer := new(bytes.Buffer)
		n, err := io.CopyN(fileBuffer, peer, 22)
		if err != nil {
			return nil, err
		}

		fmt.Printf("received (%d) bytes over the wire\n", n)
	}

	select {}

	return nil, nil
}

// Store will store the data into disk and broadcast to all the known peers.
func (s *FileServer) Store(key string, r io.Reader) error {
	// 1. store this file to disk.
	// 2. broadcast this file to all known peers in the network.

	fileBuffer := new(bytes.Buffer)

	// once we read from the r the data will no longer be available.
	// tee helps us to make the data to be available also on the buffer.
	tee := io.TeeReader(r, fileBuffer)

	// ======================================
	// Store this file to our own disk.
	size, err := s.store.Write(key, tee)
	if err != nil {
		return err
	}

	// ========================================
	// Broadcast the file to all the network to store.

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	err = s.broadcast(&msg)
	if err != nil {
		return err
	}

	// Stream the file we want to store.

	// Give the server some time to process the msg.
	time.Sleep(1 * time.Second)

	// TODO: use a multiwriter here.
	for _, peer := range s.peers {
		n, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}

		fmt.Printf("received and written (%d) bytes to disk \n", n)
	}
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
		log.Println("file server stopped due error or user quit action.")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():

			var msg Message

			// decode into a Message.
			err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg)
			if err != nil {
				log.Printf("decoding error: %+v\n", err)
				continue
			}

			err = s.handleMessage(rpc.From, &msg)
			if err != nil {
				log.Printf("handle message error: %+v\n", err)
				continue
			}

		case <-s.quitch:
			return
		}
	}
}

// handleMessage will handle the message according to the underlying type.
func (s *FileServer) handleMessage(from string, msg *Message) error {

	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)

	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}
	return nil
}

// handleMessageGetFile will send over the wire the desired file.
func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {

	if !s.store.Has(msg.Key) {
		return fmt.Errorf("need to serve file (%s), but it does not exist on disk", msg.Key)
	}

	fmt.Printf("serving file (%s) over the network \n", msg.Key)

	// Fetch the file.
	r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	// Send the reader over the wire to the one who sent the message.
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer map", from)
	}

	// Stream the file over the wire.
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("wrote (%d) bytes over the network to %s\n", n, from)

	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer map", from)
	}

	// Tell how many bytes we want to read from the peer.
	r := io.LimitReader(peer, msg.Size)

	// Store the file into the disk from the peer who sent the message.
	n, err := s.store.Write(msg.Key, r)
	if err != nil {
		return err
	}

	fmt.Printf("written (%d) bytes to disk: %s\n", n, from)

	// Enable to continue reading from the peer.
	peer.(*p2p.TCPPeer).Wg.Done()

	return nil
}

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
	gob.Register(MessageGetFile{})

}
